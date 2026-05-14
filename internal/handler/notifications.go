package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationsHandler struct{ db *pgxpool.Pool }

func NewNotificationsHandler(db *pgxpool.Pool) *NotificationsHandler {
	return &NotificationsHandler{db: db}
}

type NotificationItem struct {
	ID           string         `json:"id"`
	Kind         string         `json:"kind"`
	Title        string         `json:"title"`
	Body         string         `json:"body"`
	Data         map[string]any `json:"data"`
	ReadAt       *time.Time     `json:"read_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

// GET /notifications
func (h *NotificationsHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, kind, title, body, data, read_at, created_at
		FROM notifications
		WHERE target_user_id = $1 OR target_role = $2
		ORDER BY created_at DESC
		LIMIT 50`, userID, role)
	if err != nil {
		internalError(c, "Notifications.List/query", err)
		return
	}
	defer rows.Close()

	out := []NotificationItem{}
	for rows.Next() {
		var n NotificationItem
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body,
			&n.Data, &n.ReadAt, &n.CreatedAt); err != nil {
			internalError(c, "Notifications.List/scan", err)
			return
		}
		out = append(out, n)
	}
	c.JSON(http.StatusOK, out)
}

// GET /notifications/unread-count
func (h *NotificationsHandler) UnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")

	var count int
	if err := h.db.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM notifications
		WHERE (target_user_id = $1 OR target_role = $2)
		  AND read_at IS NULL`, userID, role,
	).Scan(&count); err != nil {
		internalError(c, "Notifications.UnreadCount/query", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// PUT /notifications/:id/read
func (h *NotificationsHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	role := c.GetString("user_role")

	tag, err := h.db.Exec(c.Request.Context(), `
		UPDATE notifications SET read_at = NOW()
		WHERE id = $1
		  AND (target_user_id = $2 OR target_role = $3)
		  AND read_at IS NULL`, id, userID, role)
	if err != nil {
		internalError(c, "Notifications.MarkRead/exec", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": tag.RowsAffected()})
}

// PUT /notifications/read-all
func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")

	tag, err := h.db.Exec(c.Request.Context(), `
		UPDATE notifications SET read_at = NOW()
		WHERE (target_user_id = $1 OR target_role = $2)
		  AND read_at IS NULL`, userID, role)
	if err != nil {
		internalError(c, "Notifications.MarkAllRead/exec", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": tag.RowsAffected()})
}
