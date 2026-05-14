package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ParkingPermitHandler struct {
	db      *pgxpool.Pool
	publish func(topic, payload string) error
}

func NewParkingPermitHandler(db *pgxpool.Pool, publish func(topic, payload string) error) *ParkingPermitHandler {
	return &ParkingPermitHandler{db: db, publish: publish}
}

type ParkingPermit struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	FullName     string     `json:"full_name,omitempty"`
	VehicleID    string     `json:"vehicle_id"`
	PlateNumber  string     `json:"plate_number,omitempty"`
	SpotID       *string    `json:"spot_id"`
	SpotNumber   *string    `json:"spot_number,omitempty"`
	Status       string     `json:"status"`
	DocumentURL  *string    `json:"document_url"`
	AdminComment *string    `json:"admin_comment"`
	CreatedAt    time.Time  `json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
}

// GET /parking/permit — список пропусков жильца
func (h *ParkingPermitHandler) MyPermits(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT pp.id, pp.user_id, pp.vehicle_id, COALESCE(v.plate_number,''),
		       pp.spot_id, ps.spot_number,
		       pp.status, pp.document_url, pp.admin_comment, pp.created_at, pp.reviewed_at
		FROM parking_permits pp
		JOIN vehicles v ON v.id = pp.vehicle_id
		LEFT JOIN parking_spots ps ON ps.id = pp.spot_id
		WHERE pp.user_id = $1
		ORDER BY pp.created_at DESC`, userID)
	if err != nil {
		internalError(c, "Permit.MyPermits/query", err)
		return
	}
	defer rows.Close()
	out := []ParkingPermit{}
	for rows.Next() {
		var p ParkingPermit
		if err := rows.Scan(&p.ID, &p.UserID, &p.VehicleID, &p.PlateNumber,
			&p.SpotID, &p.SpotNumber,
			&p.Status, &p.DocumentURL, &p.AdminComment, &p.CreatedAt, &p.ReviewedAt); err != nil {
			internalError(c, "Permit.MyPermits/scan", err)
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

// POST /parking/permit — подать заявку на пропуск (по vehicle_id)
func (h *ParkingPermitHandler) Submit(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		VehicleID string `json:"vehicle_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()

	// Проверяем что автомобиль принадлежит этому жильцу
	var count int
	if err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM vehicles WHERE id = $1 AND user_id = $2 AND is_active = true`,
		req.VehicleID, userID,
	).Scan(&count); err != nil {
		internalError(c, "Permit.Submit/checkVehicle", err)
		return
	}
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle not found in your profile"})
		return
	}

	// Нет pending/approved заявки на эту машину
	var existStatus string
	err := h.db.QueryRow(ctx,
		`SELECT status FROM parking_permits WHERE vehicle_id = $1 AND status IN ('pending','approved')`,
		req.VehicleID,
	).Scan(&existStatus)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "permit already exists for this vehicle", "status": existStatus})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		internalError(c, "Permit.Submit/checkExisting", err)
		return
	}

	var p ParkingPermit
	if err := h.db.QueryRow(ctx, `
		INSERT INTO parking_permits (user_id, vehicle_id)
		VALUES ($1, $2)
		RETURNING id, user_id, vehicle_id, spot_id, status, document_url, admin_comment, created_at, reviewed_at`,
		userID, req.VehicleID,
	).Scan(&p.ID, &p.UserID, &p.VehicleID, &p.SpotID, &p.Status,
		&p.DocumentURL, &p.AdminComment, &p.CreatedAt, &p.ReviewedAt); err != nil {
		internalError(c, "Permit.Submit/insert", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

// POST /parking/permit/:id/document — прикрепить документ к заявке
func (h *ParkingPermitHandler) UploadDocument(c *gin.Context) {
	permitID := c.Param("id")
	userID := c.GetString("user_id")
	ctx := c.Request.Context()

	var count int
	if err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM parking_permits WHERE id = $1 AND user_id = $2`,
		permitID, userID,
	).Scan(&count); err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "permit not found"})
		return
	}

	fh, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document file required"})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	dir := filepath.Join("uploads", "parking-permits", userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		internalError(c, "Permit.UploadDocument/mkdir", err)
		return
	}

	ext := filepath.Ext(fh.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), uuid.New().String()[:8], ext)
	savePath := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(fh, savePath); err != nil {
		internalError(c, "Permit.UploadDocument/save", err)
		return
	}

	docURL := fmt.Sprintf("%s/uploads/parking-permits/%s/%s", baseURL, userID, filename)
	if _, err := h.db.Exec(ctx,
		`UPDATE parking_permits SET document_url = $1 WHERE id = $2`, docURL, permitID,
	); err != nil {
		internalError(c, "Permit.UploadDocument/update", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"document_url": docURL})
}

// GET /admin/parking/permits — все заявки с данными жильца и авто
func (h *ParkingPermitHandler) AdminList(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		forbiddenAccess(c, "admin only")
		return
	}
	statusFilter := c.Query("status")
	q := `
		SELECT pp.id, pp.user_id, COALESCE(p.full_name,''),
		       pp.vehicle_id, COALESCE(v.plate_number,''),
		       pp.spot_id, ps.spot_number,
		       pp.status, pp.document_url, pp.admin_comment, pp.created_at, pp.reviewed_at
		FROM parking_permits pp
		JOIN profiles p  ON p.id  = pp.user_id
		JOIN vehicles v  ON v.id  = pp.vehicle_id
		LEFT JOIN parking_spots ps ON ps.id = pp.spot_id
		WHERE 1=1`
	args := []any{}
	if statusFilter != "" {
		args = append(args, statusFilter)
		q += ` AND pp.status = $1`
	}
	q += ` ORDER BY pp.created_at DESC`

	rows, err := h.db.Query(c.Request.Context(), q, args...)
	if err != nil {
		internalError(c, "Permit.AdminList/query", err)
		return
	}
	defer rows.Close()
	out := []ParkingPermit{}
	for rows.Next() {
		var p ParkingPermit
		if err := rows.Scan(&p.ID, &p.UserID, &p.FullName,
			&p.VehicleID, &p.PlateNumber,
			&p.SpotID, &p.SpotNumber,
			&p.Status, &p.DocumentURL, &p.AdminComment, &p.CreatedAt, &p.ReviewedAt); err != nil {
			internalError(c, "Permit.AdminList/scan", err)
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

// PUT /admin/parking/permits/:id/status — одобрить/отклонить, опционально назначить место
func (h *ParkingPermitHandler) AdminReview(c *gin.Context) {
	if c.GetString("user_role") != "admin" {
		forbiddenAccess(c, "admin only")
		return
	}
	id := c.Param("id")
	var req struct {
		Status       string  `json:"status"        binding:"required"`
		AdminComment string  `json:"admin_comment"`
		SpotID       *string `json:"spot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
		return
	}

	ctx := c.Request.Context()
	var commentPtr *string
	if req.AdminComment != "" {
		commentPtr = &req.AdminComment
	}

	var userID, vehicleID string
	err := h.db.QueryRow(ctx, `
		UPDATE parking_permits
		SET status        = $2,
		    admin_comment = COALESCE($3, admin_comment),
		    spot_id       = COALESCE($4, spot_id),
		    reviewed_at   = NOW()
		WHERE id = $1
		RETURNING user_id, vehicle_id`, id, req.Status, commentPtr, req.SpotID,
	).Scan(&userID, &vehicleID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "permit not found"})
		return
	}
	if err != nil {
		internalError(c, "Permit.AdminReview/update", err)
		return
	}

	// Синхронизируем parking_permit_status в профиле для UI-совместимости
	_, _ = h.db.Exec(ctx,
		`UPDATE profiles SET parking_permit_status = $1 WHERE id = $2`, req.Status, userID)

	c.JSON(http.StatusOK, gin.H{"ok": true, "user_id": userID, "vehicle_id": vehicleID, "status": req.Status})
}

// POST /parking/gate/scan-plate — шлагбаум паркинга проверяет номер
// Логика: vehicle зарегистрирован И есть approved parking_permit для этого vehicle_id
func (h *ParkingPermitHandler) ScanParkingGate(c *gin.Context) {
	var req struct {
		PlateNumber string `json:"plate_number" binding:"required"`
		Direction   string `json:"direction"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction == "" {
		req.Direction = "IN"
	}
	ctx := c.Request.Context()

	var vehicleID, userID string
	err := h.db.QueryRow(ctx,
		`SELECT id, user_id FROM vehicles WHERE plate_number = $1 AND is_active = true`,
		req.PlateNumber,
	).Scan(&vehicleID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusForbidden, gin.H{"action": "REJECTED", "reason": "vehicle not registered"})
		return
	}
	if err != nil {
		internalError(c, "ParkingGate.ScanPlate/vehicleQuery", err)
		return
	}

	var permitID string
	err = h.db.QueryRow(ctx,
		`SELECT id FROM parking_permits WHERE vehicle_id = $1 AND status = 'approved'`,
		vehicleID,
	).Scan(&permitID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusForbidden, gin.H{
			"action": "REJECTED",
			"reason": "no approved parking permit for this vehicle",
		})
		return
	}
	if err != nil {
		internalError(c, "ParkingGate.ScanPlate/permitQuery", err)
		return
	}

	if h.publish != nil {
		payload := `{"action":"OPEN","direction":"` + req.Direction + `"}`
		_ = h.publish("smartresidency/parking/gate/command", payload)
	}

	var eventID string
	_ = h.db.QueryRow(ctx, `
		INSERT INTO barrier_events (event_type, direction, plate_number, vehicle_id, status, gate_id)
		VALUES ('AUTO_RECOGNIZED', $1, $2, $3, 'OPENED', 'parking-gate')
		RETURNING id`, req.Direction, req.PlateNumber, vehicleID,
	).Scan(&eventID)

	c.JSON(http.StatusOK, gin.H{
		"action":    "OPEN",
		"direction": req.Direction,
		"user_id":   userID,
		"permit_id": permitID,
		"event_id":  eventID,
	})
}
