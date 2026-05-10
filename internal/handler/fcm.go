package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FCMTokenHandler struct{ db *pgxpool.Pool }

func NewFCMTokenHandler(db *pgxpool.Pool) *FCMTokenHandler {
	return &FCMTokenHandler{db: db}
}

type fcmTokenReq struct {
	Token    string `json:"token"    binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

type fcmTokenDeleteReq struct {
	Token string `json:"token" binding:"required"`
}

// POST /api/v1/users/me/fcm-token
func (h *FCMTokenHandler) Register(c *gin.Context) {
	var req fcmTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("user_id")
	_, err := h.db.Exec(c.Request.Context(), `
		INSERT INTO fcm_tokens (user_id, token, platform, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (token) DO UPDATE
		SET user_id    = EXCLUDED.user_id,
		    platform   = EXCLUDED.platform,
		    updated_at = NOW()`,
		userID, req.Token, req.Platform,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/v1/users/me/fcm-token/delete
// Called by Flutter on signOut. Removes only the caller's own token —
// user_id check prevents one user from deleting another's token.
func (h *FCMTokenHandler) Delete(c *gin.Context) {
	var req fcmTokenDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("user_id")
	_, err := h.db.Exec(c.Request.Context(),
		`DELETE FROM fcm_tokens WHERE token = $1 AND user_id = $2`,
		req.Token, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
