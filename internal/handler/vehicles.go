package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VehicleHandler struct{ db *pgxpool.Pool }

func NewVehicleHandler(db *pgxpool.Pool) *VehicleHandler { return &VehicleHandler{db: db} }

type vehicle struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PlateNumber string    `json:"plate_number"`
	Brand       string    `json:"brand"`
	Color       string    `json:"color"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *VehicleHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := h.db.Query(context.Background(), `
		SELECT id, user_id, plate_number, brand, color, is_active, created_at
		FROM vehicles
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		internalError(c, "Vehicle", err)
		return
	}
	defer rows.Close()

	list := []vehicle{}
	for rows.Next() {
		var v vehicle
		if err := rows.Scan(&v.ID, &v.UserID, &v.PlateNumber, &v.Brand, &v.Color, &v.IsActive, &v.CreatedAt); err != nil {
			internalError(c, "Vehicle", err)
			return
		}
		list = append(list, v)
	}
	c.JSON(http.StatusOK, list)
}

type createVehicleReq struct {
	PlateNumber string `json:"plate_number" binding:"required"`
	Brand       string `json:"brand"`
	Color       string `json:"color"`
}

func (h *VehicleHandler) Create(c *gin.Context) {
	var req createVehicleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	id := uuid.New().String()
	plate := strings.ToUpper(req.PlateNumber)

	_, err := h.db.Exec(context.Background(), `
		INSERT INTO vehicles (id, user_id, plate_number, brand, color)
		VALUES ($1, $2, $3, $4, $5)`,
		id, userID, plate, req.Brand, req.Color,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "plate already registered"})
			return
		}
		internalError(c, "Vehicle", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "plate_number": plate})
}

func (h *VehicleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	tag, err := h.db.Exec(context.Background(),
		`DELETE FROM vehicles WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		internalError(c, "Vehicle", err)
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
