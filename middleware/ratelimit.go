package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter 創建新的速率限制器
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

// getLimiter 獲取或創建限制器
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
		rl.mu.Unlock()
	}

	return limiter
}

// RateLimitMiddleware 速率限制中間件
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", strconv.FormatFloat(float64(rl.rate), 'f', -1, 64))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", time.Now().Add(time.Second).Format(time.RFC1123))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DefaultRateLimit 預設速率限制（每分鐘 60 次請求）
func DefaultRateLimit() gin.HandlerFunc {
	rl := NewRateLimiter(rate.Every(time.Minute/60), 60)
	return RateLimitMiddleware(rl)
}
