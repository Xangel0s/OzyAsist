package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]int
	limit    int
	window   time.Duration
}

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		clients: make(map[string]int),
		limit:   limit,
		window:  window,
	}

	go func() {
		for {
			time.Sleep(window)
			rl.mu.Lock()
			rl.clients = make(map[string]int)
			rl.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		rl.mu.Lock()
		rl.clients[ip]++
		count := rl.clients[ip]
		rl.mu.Unlock()

		if count > rl.limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
