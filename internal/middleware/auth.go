package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := ""
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			raw = strings.TrimPrefix(h, "Bearer ")
		} else if q := c.Query("token"); q != "" {
			// EventSource (SSE) cannot set headers — accept token in query.
			raw = q
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token, err := jwt.ParseWithClaims(raw, &Claims{},
			func(t *jwt.Token) (any, error) { return []byte(secret), nil },
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(*Claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}
