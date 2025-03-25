package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	custom_errors "idiomatic-go/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	Rate   int           // Requests allowed per period
	Period time.Duration // Time period (e.g., time.Minute)
}

// RateLimitMiddleware creates a rate limiter middleware
func RateLimitMiddleware(logger *logrus.Logger, rdb *redis.Client, config RateLimiterConfig) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(rdb)

	return func(c *gin.Context) {
		key := c.ClientIP()

		res, err := limiter.Allow(context.Background(), key, redis_rate.Limit{
			Rate:   config.Rate,
			Burst:  config.Rate,
			Period: config.Period,
		})
		if err != nil {
			logger.WithError(err).Error("failed to check rate limit")
			c.JSON(http.StatusInternalServerError, custom_errors.ErrInternalServerError)
			c.Abort()
			return
		}

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

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		c.Header("X-RateLimit-Reset", time.Now().Add(res.ResetAfter).Format(time.RFC1123))

		c.Next()
	}
}
