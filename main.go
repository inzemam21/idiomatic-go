package main

import (
	"context"
	"os"
	"time"

	"idiomatic-go/database"
	custom_errors "idiomatic-go/errors"
	"idiomatic-go/handlers"
	"idiomatic-go/middleware"
	"idiomatic-go/routes"
	"idiomatic-go/services"

	_ "idiomatic-go/docs" // Import generated docs

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Config struct {
	Port      string
	DBConn    string
	LogLevel  string
	JWTSecret string
}

// @title User Management API
// @version 1.0
// @description This is a sample user management API
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /api/v1
func main() {
	config := Config{
		Port:      getEnv("PORT", "8080"),
		DBConn:    getEnv("DATABASE_URL", "postgres://user:password@localhost:5434/dbname?sslmode=disable"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),
	}

	logger := logrus.New()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(level)

	// Initialize database connection
	dbConfig := database.Config{
		DBConn:          config.DBConn,
		MaxConns:        20,
		MinConns:        2,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	}
	db, err := database.NewDB(context.Background(), dbConfig, logger)
	if err != nil {
		logger.Fatal("failed to initialize database: ", err)
	}
	defer db.Close()

	userService := services.NewUserService(db.Queries, logger)
	userHandler := handlers.NewUserHandler(userService, logger, config.JWTSecret)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(ErrorLoggingMiddleware(logger))

	api := router.Group("/api/v1")
	routes.RegisterUserRoutes(api, userHandler, config.JWTSecret)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Infof("Starting server on port %s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		logger.Fatal("failed to start server: ", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func ErrorLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				if apiErr, ok := custom_errors.IsAPIError(err.Err); ok {
					logger.WithFields(logrus.Fields{
						"status": apiErr.StatusCode,
						"code":   apiErr.Code,
					}).Error(apiErr.Message)
				} else {
					logger.WithError(err.Err).Error("unhandled error")
				}
			}
		}
	}
}
