package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventNotifier dispatches an FCM push for a sensor event. Implemented by
// internal/fcm. Kept as an interface so handlers don't depend on FCM directly
// and the server still boots when FCM is disabled (notifier may be nil).
type EventNotifier interface {
	NotifyEvent(ctx context.Context, eventID string) (sent int, err error)
}

type SensorHandler struct {
	db       *pgxpool.Pool
	notifier EventNotifier
}

func NewSensorHandler(db *pgxpool.Pool, notifier EventNotifier) *SensorHandler {
	return &SensorHandler{db: db, notifier: notifier}
}

type Sensor struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	EntranceNum int       `json:"entrance_num"`
	Floor       int       `json:"floor"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
}

type SensorEvent struct {
	ID           string     `json:"id"`
	SensorID     string     `json:"sensor_id"`
	Type         string     `json:"type"`
	EntranceNum  int        `json:"entrance_num"`
	Floor        int        `json:"floor"`
	Status       string     `json:"status"`
	ThreatType   *string    `json:"threat_type"`
	AdminComment *string    `json:"admin_comment"`
	CreatedAt    time.Time  `json:"created_at"`
	ConfirmedAt  *time.Time `json:"confirmed_at"`
}

const sensorCols = `id, type, entrance_num, floor, status, last_updated`
const eventCols = `id, sensor_id, type, entrance_num, floor, status, threat_type, admin_comment, created_at, confirmed_at`

func scanSensors(rows pgx.Rows) ([]Sensor, error) {
	defer rows.Close()
	out := []Sensor{}
	for rows.Next() {
		var s Sensor
		if err := rows.Scan(&s.ID, &s.Type, &s.EntranceNum, &s.Floor, &s.Status, &s.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanEvents(rows pgx.Rows) ([]SensorEvent, error) {
	defer rows.Close()
	out := []SensorEvent{}
	for rows.Next() {
		var e SensorEvent
		if err := rows.Scan(&e.ID, &e.SensorID, &e.Type, &e.EntranceNum, &e.Floor,
			&e.Status, &e.ThreatType, &e.AdminComment, &e.CreatedAt, &e.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GET /api/v1/sensors?entrance=N
func (h *SensorHandler) ListByEntrance(c *gin.Context) {
	e, err := strconv.Atoi(c.Query("entrance"))
	if err != nil || e < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entrance query param required"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT `+sensorCols+` FROM sensors WHERE entrance_num=$1
		 ORDER BY floor ASC, type ASC`, e)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	out, err := scanSensors(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /api/v1/admin/sensors
func (h *SensorHandler) ListAll(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT `+sensorCols+` FROM sensors
		 ORDER BY entrance_num ASC, floor ASC, type ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	out, err := scanSensors(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /api/v1/sensors/events?entrance=&status=
// Resident: entrance forced from profile. Admin: optional filters.
func (h *SensorHandler) ListEvents(c *gin.Context) {
	role := c.GetString("user_role")
	userID := c.GetString("user_id")
	ctx := c.Request.Context()

	var entranceFilter *int
	if role == "admin" {
		if v := c.Query("entrance"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				entranceFilter = &n
			}
		}
	} else {
		var ent *int
		if err := h.db.QueryRow(ctx,
			`SELECT entrance FROM profiles WHERE id = $1`, userID).Scan(&ent); err != nil || ent == nil {
			c.JSON(http.StatusOK, []SensorEvent{})
			return
		}
		entranceFilter = ent
	}

	statusFilter := c.Query("status")

	q := `SELECT ` + eventCols + ` FROM sensor_events WHERE 1=1`
	args := []any{}
	if entranceFilter != nil {
		args = append(args, *entranceFilter)
		q += fmt.Sprintf(" AND entrance_num = $%d", len(args))
	}
	if statusFilter != "" {
		args = append(args, statusFilter)
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	q += " ORDER BY created_at DESC"

	rows, err := h.db.Query(ctx, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	out, err := scanEvents(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, out)
}

type updateEventStatusReq struct {
	Status       string `json:"status" binding:"required"`
	ThreatType   string `json:"threat_type"`
	AdminComment string `json:"admin_comment"`
}

// PATCH /api/v1/admin/sensors/events/:id/status
func (h *SensorHandler) UpdateEventStatus(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	id := c.Param("id")
	var req updateEventStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowedStatus := map[string]bool{
		"DETECTED": true, "CHECKING": true, "CONFIRMED": true, "FALSE_ALARM": true,
	}
	if !allowedStatus[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	allowedThreat := map[string]bool{"WATER_LEAK": true, "FIRE": true}
	if req.Status == "CONFIRMED" && !allowedThreat[req.ThreatType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "threat_type (WATER_LEAK|FIRE) required for CONFIRMED"})
		return
	}

	var threatPtr, commentPtr *string
	if req.ThreatType != "" {
		threatPtr = &req.ThreatType
	}
	if req.AdminComment != "" {
		commentPtr = &req.AdminComment
	}

	ctx := c.Request.Context()

	var e SensorEvent
	err := h.db.QueryRow(ctx, `
		UPDATE sensor_events
		SET status        = $2,
		    threat_type   = COALESCE($3, threat_type),
		    admin_comment = COALESCE($4, admin_comment),
		    confirmed_at  = CASE WHEN $2 = 'CONFIRMED' THEN NOW() ELSE confirmed_at END
		WHERE id = $1
		RETURNING `+eventCols,
		id, req.Status, threatPtr, commentPtr,
	).Scan(&e.ID, &e.SensorID, &e.Type, &e.EntranceNum, &e.Floor,
		&e.Status, &e.ThreatType, &e.AdminComment, &e.CreatedAt, &e.ConfirmedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	if req.Status == "FALSE_ALARM" {
		_, _ = h.db.Exec(ctx,
			`UPDATE sensors SET status='NORMAL', last_updated=NOW() WHERE id=$1`, e.SensorID)
	}

	c.JSON(http.StatusOK, e)
}

// POST /api/v1/admin/sensors/events/:id/notify
func (h *SensorHandler) NotifyEvent(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	id := c.Param("id")

	var entrance int
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT entrance_num FROM sensor_events WHERE id = $1`, id).Scan(&entrance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	if h.notifier == nil {
		c.JSON(http.StatusOK, gin.H{"sent": 0, "warning": "fcm not configured"})
		return
	}

	sent, err := h.notifier.NotifyEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": sent})
}
