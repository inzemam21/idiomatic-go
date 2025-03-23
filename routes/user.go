package routes

import (
	"idiomatic-go/handlers"
	"idiomatic-go/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RegisterUserRoutes(r *gin.RouterGroup, h *handlers.UserHandler, jwtSecret string) {
	r.POST("/login", h.Login) // Public endpoint

	users := r.Group("/users")
	users.Use(middleware.AuthMiddleware(logrus.New(), jwtSecret))
	{
		users.POST("", h.CreateUser)
		// Add other protected routes here
		// users.GET("", h.ListUsers)
		// users.GET("/:id", h.GetUser)
		// users.PUT("/:id", h.UpdateUser)
		// users.DELETE("/:id", h.DeleteUser)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
