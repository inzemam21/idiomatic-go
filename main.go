package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"idiomatic-go/database"
	custom_errors "idiomatic-go/errors"
	"idiomatic-go/handlers"
	"idiomatic-go/middleware"
	"idiomatic-go/routes"
	"idiomatic-go/services"

	_ "idiomatic-go/docs"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type Config struct {
	Port       string
	DBConn     string
	LogLevel   string
	JWTSecret  string
	RedisAddr  string
	RedisPass  string
	RateLimit  int
	RatePeriod string
}

// Metrics (unchanged)
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func main() {
	config := Config{
		Port:       getEnv("PORT", "8080"),
		DBConn:     getEnv("DATABASE_URL", "postgres://user:password@localhost:5434/dbname?sslmode=disable"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		JWTSecret:  getEnv("JWT_SECRET", "your-secret-key"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:  getEnv("REDIS_PASS", ""),
		RateLimit:  getEnvInt("RATE_LIMIT", 100),
		RatePeriod: getEnv("RATE_PERIOD", "1m"),
	}

	logger := logrus.New()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(level)

	// Initialize OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		logger.Fatal("failed to initialize tracer: ", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	ratePeriod, err := time.ParseDuration(config.RatePeriod)
	if err != nil {
		logger.Fatal("invalid rate period: ", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPass,
		DB:       0,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("failed to connect to Redis: ", err)
	}
	logger.Info("Connected to Redis successfully")

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

	userService := services.NewUserService(db, logger)
	userHandler := handlers.NewUserHandler(userService, logger, config.JWTSecret)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(otelgin.Middleware("idiomatic-go")) // Instrument Gin for HTTP tracing
	router.Use(middleware.RateLimitMiddleware(logger, rdb, middleware.RateLimiterConfig{
		Rate:   config.RateLimit,
		Period: ratePeriod,
	}))
	router.Use(PrometheusMiddleware())
	router.Use(ErrorLoggingMiddleware(logger))

	api := router.Group("/api/v1")
	routes.RegisterUserRoutes(api, userHandler, config.JWTSecret)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.HandlerFunc(func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}))

	logger.Infof("Starting server on port %s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		logger.Fatal("failed to start server: ", err)
	}
}

// initTracer sets up OpenTelemetry with a Jaeger exporter
func initTracer() (*sdktrace.TracerProvider, error) {
	// Configure the Jaeger exporter to send traces to Jaeger's HTTP endpoint
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint("http://localhost:14268/api/traces"),
	))
	if err != nil {
		return nil, err
	}

	// Define the service name for the traces
	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceNameKey.String("idiomatic-go")),
	)
	if err != nil {
		return nil, err
	}

	// Create the tracer provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	return tp, nil
}

// ... PrometheusMiddleware, getEnv, getEnvInt, ErrorLoggingMiddleware unchanged ...
// PrometheusMiddleware instruments HTTP requests
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
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
