package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GuestHandler struct{ db *pgxpool.Pool }

func NewGuestHandler(db *pgxpool.Pool) *GuestHandler { return &GuestHandler{db: db} }

type guestPass struct {
	ID          string    `json:"id"`
	ResidentID  string    `json:"resident_id"`
	GuestName   string    `json:"guest_name"`
	GuestPhone  *string   `json:"guest_phone"`
	CarNumber   *string   `json:"car_number"`
	AccessType  string    `json:"access_type"`
	AccessCode  string    `json:"access_code"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidUntil  time.Time `json:"valid_until"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *GuestHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	ctx := context.Background()

	rows, err := h.db.Query(ctx, `
		SELECT id, resident_id, guest_name, guest_phone, car_number,
		       access_type, access_code, valid_from, valid_until, status, created_at
		FROM guest_access
		WHERE resident_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	defer rows.Close()

	passes := []guestPass{}
	for rows.Next() {
		var p guestPass
		if err := rows.Scan(
			&p.ID, &p.ResidentID, &p.GuestName, &p.GuestPhone, &p.CarNumber,
			&p.AccessType, &p.AccessCode, &p.ValidFrom, &p.ValidUntil,
			&p.Status, &p.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		passes = append(passes, p)
	}
	c.JSON(http.StatusOK, passes)
}

type createGuestReq struct {
	GuestName  string `json:"guest_name"  binding:"required"`
	GuestPhone string `json:"guest_phone"`
	CarNumber  string `json:"car_number"`
	AccessType string `json:"access_type" binding:"required"`
	ValidFrom  string `json:"valid_from"  binding:"required"`
	ValidUntil string `json:"valid_until" binding:"required"`
}

func (h *GuestHandler) Create(c *gin.Context) {
	var req createGuestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_from format, use RFC3339"})
		return
	}
	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_until format, use RFC3339"})
		return
	}
	if !validUntil.After(validFrom) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_until must be after valid_from"})
		return
	}

	userID := c.GetString("user_id")
	id := uuid.New().String()
	ms := time.Now().UnixMilli()
	code := fmt.Sprintf("G-%06d", ms%1000000)

	var phone, car *string
	if req.GuestPhone != "" {
		phone = &req.GuestPhone
	}
	if req.CarNumber != "" {
		car = &req.CarNumber
	}

	_, err = h.db.Exec(context.Background(), `
		INSERT INTO guest_access
			(id, resident_id, guest_name, guest_phone, car_number,
			 access_type, access_code, valid_from, valid_until, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active')`,
		id, userID, req.GuestName, phone, car,
		req.AccessType, code, validFrom, validUntil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "access_code": code})
}

func (h *GuestHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	_, err := h.db.Exec(context.Background(),
		`UPDATE guest_access SET status = 'cancelled'
		 WHERE id = $1 AND resident_id = $2`, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
