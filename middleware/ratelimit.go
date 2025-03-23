package middleware

import (
	"context"
	"net/http"
	"time"

	custom_errors "idiomatic-go/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	RedisAddr string        // Redis address (e.g., "localhost:6379")
	RedisPass string        // Redis password (optional)
	Rate      int           // Requests allowed per period
	Period    time.Duration // Time period (e.g., time.Minute)
}

// RateLimitMiddleware creates a rate limiter middleware
func RateLimitMiddleware(logger *logrus.Logger, config RateLimiterConfig) gin.HandlerFunc {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPass,
		DB:       0, // Default DB
	})
	defer rdb.Close() // In production, manage this connection better

	// Initialize rate limiter
	limiter := redis_rate.NewLimiter(rdb)

	return func(c *gin.Context) {
		// Use client IP as the key (could use user ID from JWT for authenticated routes)
		key := c.ClientIP()

		// Check rate limit
		res, err := limiter.Allow(context.Background(), key, redis_rate.Limit{
			Rate:   config.Rate,
			Burst:  config.Rate, // Allow bursts up to the rate
			Period: config.Period,
		})
		if err != nil {
			logger.WithError(err).Error("failed to check rate limit")
			c.JSON(http.StatusInternalServerError, custom_errors.ErrInternalServerError)
			c.Abort()
			return
		}

		// Check if rate limit is exceeded (Allowed <= 0 means no requests left)
		if res.Allowed <= 0 {
			logger.WithFields(logrus.Fields{
				"ip":          key,
				"retry_after": res.RetryAfter.Seconds(),
			}).Warn("rate limit exceeded")
			c.Header("Retry-After", res.RetryAfter.String())
			c.JSON(http.StatusTooManyRequests, custom_errors.NewAPIError(
				http.StatusTooManyRequests,
				"rate_limit_exceeded",
				"Too many requests",
			))
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(config.Rate)))
		c.Header("X-RateLimit-Remaining", string(rune(res.Remaining)))
		c.Header("X-RateLimit-Reset", time.Now().Add(res.ResetAfter).Format(time.RFC1123))

		c.Next()
	}
}
