package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceRequestHandler struct{ db *pgxpool.Pool }

func NewServiceRequestHandler(db *pgxpool.Pool) *ServiceRequestHandler {
	return &ServiceRequestHandler{db: db}
}

type serviceRequest struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Photos      []string  `json:"photos"`
}

func (h *ServiceRequestHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")
	ctx := context.Background()

	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}
	var err error

	if role == "admin" {
		rows, err = h.db.Query(ctx,
			`SELECT id, user_id, category, description, status, created_at, updated_at
			 FROM service_requests ORDER BY created_at DESC`)
	} else {
		rows, err = h.db.Query(ctx,
			`SELECT id, user_id, category, description, status, created_at, updated_at
			 FROM service_requests WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	defer rows.Close()

	requests := []serviceRequest{}
	idIndex := map[string]int{}

	for rows.Next() {
		var r serviceRequest
		if err := rows.Scan(&r.ID, &r.UserID, &r.Category, &r.Description,
			&r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		r.Photos = []string{}
		idIndex[r.ID] = len(requests)
		requests = append(requests, r)
	}
	if rows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	if len(requests) > 0 {
		ids := make([]string, len(requests))
		for i, r := range requests {
			ids[i] = r.ID
		}
		photoRows, err := h.db.Query(ctx,
			`SELECT request_id, file_path FROM request_photos WHERE request_id = ANY($1)`, ids)
		if err == nil {
			defer photoRows.Close()
			baseURL := os.Getenv("BASE_URL")
			for photoRows.Next() {
				var reqID, path string
				if err := photoRows.Scan(&reqID, &path); err == nil {
					if i, ok := idIndex[reqID]; ok {
						requests[i].Photos = append(requests[i].Photos,
							fmt.Sprintf("%s/uploads/request-photos/%s", baseURL, filepath.Base(path)))
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, requests)
}

type createRequestReq struct {
	Category    string `json:"category"    binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (h *ServiceRequestHandler) Create(c *gin.Context) {
	var req createRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	id := uuid.New().String()
	ctx := context.Background()

	_, err := h.db.Exec(ctx,
		`INSERT INTO service_requests (id, user_id, category, description, status)
		 VALUES ($1, $2, $3, $4, 'new')`,
		id, userID, req.Category, req.Description,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

type updateStatusReq struct {
	Status string `json:"status" binding:"required"`
}

func (h *ServiceRequestHandler) UpdateStatus(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := c.Param("id")
	var req updateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowed := map[string]bool{"new": true, "in_progress": true, "done": true}
	if !allowed[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	_, err := h.db.Exec(context.Background(),
		`UPDATE service_requests SET status = $2, updated_at = NOW() WHERE id = $1`, id, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

func (h *ServiceRequestHandler) UploadPhoto(c *gin.Context) {
	requestID := c.Param("id")

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "photo file required"})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext
	savePath := filepath.Join("uploads", "request-photos", filename)

	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	_, err = h.db.Exec(context.Background(),
		`INSERT INTO request_photos (id, request_id, file_path) VALUES ($1, $2, $3)`,
		uuid.New().String(), requestID, savePath,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	c.JSON(http.StatusCreated, gin.H{
		"url": fmt.Sprintf("%s/uploads/request-photos/%s", baseURL, filename),
	})
}
