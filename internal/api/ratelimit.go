package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	limit    int
}

func newIPLimiter(window time.Duration, limit int) *ipLimiter {
	lim := &ipLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		limit:    limit,
	}
	
	// Start background cleanup goroutine
	go lim.cleanup()
	return lim
}

func (l *ipLimiter) limitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		l.mu.Lock()
		times := l.attempts[ip]
		
		// Filter out attempts outside the sliding window
		var validTimes []time.Time
		cutoff := now.Add(-l.window)
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}

		if len(validTimes) >= l.limit {
			l.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		validTimes = append(validTimes, now)
		l.attempts[ip] = validTimes
		l.mu.Unlock()

		c.Next()
	}
}

func (l *ipLimiter) cleanup() {
	ticker := time.NewTicker(l.window)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-l.window)
		for ip, times := range l.attempts {
			var validTimes []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(l.attempts, ip)
			} else {
				l.attempts[ip] = validTimes
			}
		}
		l.mu.Unlock()
	}
}
