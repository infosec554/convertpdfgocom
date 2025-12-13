package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client   *redis.Client
	limit    int
	window   time.Duration
}

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	pipe := rl.client.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count, err := incr.Result()
	if err != nil {
		return false, err
	}

	return count <= int64(rl.limit), nil
}

func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func RateLimiterWithRedis(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	rl := NewRateLimiter(client, limit, window)

	return func(c *gin.Context) {
		var key string
		if userID, exists := c.Get("user_id"); exists {
			key = "ratelimit:user:" + userID.(string)
		} else {
			key = "ratelimit:ip:" + c.ClientIP()
		}

		allowed, err := rl.Allow(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}
