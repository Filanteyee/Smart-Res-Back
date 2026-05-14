package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileHandler struct{ db *pgxpool.Pool }

func NewProfileHandler(db *pgxpool.Pool) *ProfileHandler { return &ProfileHandler{db: db} }

func (h *ProfileHandler) Get(c *gin.Context) {
	id := c.Param("id")
	callerID := c.GetString("user_id")
	callerRole := c.GetString("user_role")

	if id != callerID && callerRole != "admin" {
		forbiddenAccess(c, "forbidden")
		return
	}

	var p profileRow
	row := h.db.QueryRow(context.Background(),
		`SELECT`+profileCols+` FROM profiles WHERE id = $1`, id)
	if err := p.scanFrom(row); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type updateProfileReq struct {
	FullName       *string `json:"full_name"`
	Phone          *string `json:"phone"`
	City           *string `json:"city"`
	Street         *string `json:"street"`
	PropertyType   *string `json:"property_type"`
	PropertyNumber *string `json:"property_number"`
	FullAddress    *string `json:"full_address"`
	Entrance       *int    `json:"entrance"`
	Floor          *int    `json:"floor"`
	Apartment      *string `json:"apartment"`
}

func (h *ProfileHandler) Update(c *gin.Context) {
	id := c.Param("id")
	callerID := c.GetString("user_id")
	callerRole := c.GetString("user_role")

	if id != callerID && callerRole != "admin" {
		forbiddenAccess(c, "forbidden")
		return
	}

	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	_, err := h.db.Exec(ctx, `
		UPDATE profiles SET
			full_name       = COALESCE($2, full_name),
			phone           = COALESCE($3, phone),
			city            = COALESCE($4, city),
			street          = COALESCE($5, street),
			property_type   = COALESCE($6, property_type),
			property_number = COALESCE($7, property_number),
			full_address    = COALESCE($8, full_address),
			entrance        = COALESCE($9, entrance),
			floor           = COALESCE($10, floor),
			apartment       = COALESCE($11, apartment),
			updated_at      = NOW()
		WHERE id = $1`,
		id,
		req.FullName, req.Phone, req.City, req.Street,
		req.PropertyType, req.PropertyNumber, req.FullAddress,
		req.Entrance, req.Floor, req.Apartment,
	)
	if err != nil {
		internalError(c, "Profile.Update/exec", err)
		return
	}

	var p profileRow
	row := h.db.QueryRow(ctx,
		`SELECT`+profileCols+` FROM profiles WHERE id = $1`, id)
	if err := p.scanFrom(row); err != nil {
		internalError(c, "Profile.Update/fetch", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
