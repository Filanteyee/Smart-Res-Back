package handler

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"smartresidency/internal/middleware"
)

type AuthHandler struct{ db *pgxpool.Pool }

func NewAuthHandler(db *pgxpool.Pool) *AuthHandler { return &AuthHandler{db: db} }

type registerReq struct {
	Email          string `json:"email"    binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	FullName       string `json:"full_name" binding:"required"`
	Phone          string `json:"phone"`
	IIN            string `json:"iin"`
	PersonType     string `json:"person_type"`
	City           string `json:"city"`
	Street         string `json:"street"`
	PropertyType   string `json:"property_type"`
	PropertyNumber string `json:"property_number"`
	FullAddress    string `json:"full_address"`
	Entrance       *int   `json:"entrance"`
	Floor          *int   `json:"floor"`
	Apartment      string `json:"apartment"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	userID := uuid.New().String()
	ctx := context.Background()

	_, err = h.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		userID, strings.ToLower(strings.TrimSpace(req.Email)), string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	_, err = h.db.Exec(ctx, `
		INSERT INTO profiles (
			id, full_name, email, phone, iin, person_type,
			city, street, property_type, property_number, full_address,
			entrance, floor, apartment,
			role, verification_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'resident','not_submitted')`,
		userID, req.FullName, req.Email, req.Phone, req.IIN, req.PersonType,
		req.City, req.Street, req.PropertyType, req.PropertyNumber, req.FullAddress,
		req.Entrance, req.Floor, req.Apartment,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	token, err := makeToken(userID, req.Email, "resident")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token, "user_id": userID})
}

type loginReq struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	var userID, hash, role string
	err := h.db.QueryRow(ctx, `
		SELECT u.id, u.password_hash, COALESCE(p.role, 'resident')
		FROM users u
		LEFT JOIN profiles p ON p.id = u.id
		WHERE u.email = $1`,
		strings.ToLower(strings.TrimSpace(req.Email)),
	).Scan(&userID, &hash, &role)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := makeToken(userID, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": userID, "role": role})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	userID := c.GetString("user_id")
	ctx := context.Background()

	var email, role string
	err := h.db.QueryRow(ctx,
		`SELECT u.email, COALESCE(p.role, 'resident')
		 FROM users u LEFT JOIN profiles p ON p.id = u.id
		 WHERE u.id = $1`, userID,
	).Scan(&email, &role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	token, err := makeToken(userID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": userID, "role": role})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	ctx := context.Background()

	var p profileRow
	row := h.db.QueryRow(ctx,
		`SELECT`+profileCols+` FROM profiles WHERE id = $1`, userID)
	if err := p.scanFrom(row); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func makeToken(userID, email, role string) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
