package handlers

import (
	"log"
	"net/http"
	"time"

	db "idiomatic-go/database"
	"idiomatic-go/middleware"
	"idiomatic-go/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	userService *services.UserService
	logger      *logrus.Logger
	jwtSecret   string
}

func NewUserHandler(userService *services.UserService, logger *logrus.Logger, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
		jwtSecret:   jwtSecret,
	}
}

type createUserRequest struct {
	Username string `json:"username" binding:"required" example:"johndoe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type UserResponse struct {
	ID        int64  `json:"id" example:"1"`
	Username  string `json:"username" example:"johndoe"`
	Email     string `json:"email" example:"john@example.com"`
	CreatedAt string `json:"created_at" example:"2025-03-23T15:04:05Z"` // Using string instead of pgtype.Timestamptz
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user with the provided details
// @Tags users
// @Accept json
// @Produce json
// @Param user body createUserRequest true "User details"
// @Success 201 {object} UserResponse
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := db.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.Password, // Should be hashed in production
	}

	user, err := h.userService.CreateUser(c.Request.Context(), params)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body loginRequest true "User credentials"
// @Success 200 {object} loginResponse
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Router /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	type loginRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	type loginResponse struct {
		Token string `json:"token"`
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	claims := middleware.Claims{
		UserID: int64(user.ID),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, loginResponse{Token: tokenString})
}
