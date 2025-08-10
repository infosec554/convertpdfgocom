package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	visitors  = make(map[string]*rate.Limiter)
	mu        sync.Mutex
	rateLimit = rate.Every(1 * time.Second) // 1 request per second
	burst     = 100
)

func getVisitor(key string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[key]
	if !exists {
		limiter = rate.NewLimiter(rateLimit, burst)
		visitors[key] = limiter
	}
	return limiter
}

func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Avval user_id borligini tekshiramiz
		var key string
		if userID, exists := c.Get("user_id"); exists {
			key = "user:" + userID.(string)
		} else {
			// Agar login qilmagan bo‘lsa: IP + User-Agent orqali soft ajratish
			ua := c.GetHeader("User-Agent")
			key = "ip:" + c.ClientIP() + "_" + ua
		}

		limiter := getVisitor(key)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}
