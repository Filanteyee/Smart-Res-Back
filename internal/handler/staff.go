package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StaffHandler struct{ db *pgxpool.Pool }

func NewStaffHandler(db *pgxpool.Pool) *StaffHandler {
	return &StaffHandler{db: db}
}

type staffMember struct {
	ID          string `json:"id"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Specialty   string `json:"specialty"`
	InProgress  int    `json:"in_progress_count"`
	DoneToday   int    `json:"done_today_count"`
	LastTakenAt *time.Time `json:"last_taken_at,omitempty"`
}

// GET /admin/staff — list all staff members with their current workload.
func (h *StaffHandler) List(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		forbiddenAccess(c, "admin only")
		return
	}
	ctx := context.Background()

	rows, err := h.db.Query(ctx, `
		SELECT
			p.id,
			p.full_name,
			p.email,
			p.phone,
			COALESCE(p.specialty, '') AS specialty,
			COUNT(sr.id) FILTER (WHERE sr.status = 'in_progress')             AS in_progress,
			COUNT(sr.id) FILTER (WHERE sr.status = 'done'
			                       AND sr.updated_at >= NOW() - INTERVAL '24h') AS done_today,
			MAX(sr.taken_at)                                                    AS last_taken_at
		FROM profiles p
		LEFT JOIN service_requests sr ON sr.assigned_to = p.id
		WHERE p.role = 'staff'
		GROUP BY p.id
		ORDER BY p.full_name`)
	if err != nil {
		internalError(c, "Staff.List/query", err)
		return
	}
	defer rows.Close()

	out := []staffMember{}
	for rows.Next() {
		var m staffMember
		if err := rows.Scan(
			&m.ID, &m.FullName, &m.Email, &m.Phone, &m.Specialty,
			&m.InProgress, &m.DoneToday, &m.LastTakenAt,
		); err != nil {
			internalError(c, "Staff.List/scan", err)
			return
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
}

// GET /admin/staff/:id/requests — all requests assigned to a specific staff member.
func (h *StaffHandler) Requests(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		forbiddenAccess(c, "admin only")
		return
	}
	staffID := c.Param("id")
	rows, err := h.db.Query(context.Background(), `
		SELECT id, user_id, category, description, status,
		       assigned_to, NULL::text, taken_at, created_at, updated_at
		FROM service_requests
		WHERE assigned_to = $1
		ORDER BY created_at DESC`, staffID)
	if err != nil {
		internalError(c, "Staff.Requests/query", err)
		return
	}
	requests, err := scanSRRows(rows)
	if err != nil {
		internalError(c, "Staff.Requests/scan", err)
		return
	}
	if requests == nil {
		requests = []serviceRequest{}
	}
	c.JSON(http.StatusOK, requests)
}
