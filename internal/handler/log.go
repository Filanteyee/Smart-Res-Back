package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// internalError logs the full error server-side and returns a generic 500 to the client.
func internalError(c *gin.Context, op string, err error) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "-"
	}
	log.Printf("[500] %s | user=%s | path=%s | %v", op, userID, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

// forbiddenAccess logs the attempt and returns 403 to the client.
func forbiddenAccess(c *gin.Context, msg string) {
	log.Printf("[403] %s | user=%s | path=%s", msg, c.GetString("user_id"), c.Request.URL.Path)
	c.JSON(http.StatusForbidden, gin.H{"error": msg})
}
