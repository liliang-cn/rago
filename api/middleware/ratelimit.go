package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for each client
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     int
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rateLimit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rateLimit,
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for a specific client
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter for this client
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	// Rate is per minute, convert to per second
	limiter = rate.NewLimiter(rate.Limit(float64(rl.rate)/60.0), rl.burst)
	rl.limiters[key] = limiter

	// Clean up old limiters periodically
	if len(rl.limiters) > 10000 {
		go rl.cleanup()
	}

	return limiter
}

// cleanup removes inactive rate limiters
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Keep only the most recent 5000 limiters
	if len(rl.limiters) > 5000 {
		// In production, you'd want a more sophisticated cleanup strategy
		// based on last access time
		rl.limiters = make(map[string]*rate.Limiter)
	}
}

// RateLimit creates a rate limiting middleware
func RateLimit(rateLimit, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rateLimit, burst)

	return func(c *gin.Context) {
		// Use IP address as the key for rate limiting
		key := c.ClientIP()

		// For authenticated users, use user ID instead
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok && uid != "" {
				key = "user:" + uid
			}
		}

		// Get the rate limiter for this client
		l := limiter.getLimiter(key)

		// Check if request is allowed
		if !l.Allow() {
			// Calculate retry-after
			reservation := l.Reserve()
			retryAfter := reservation.Delay()
			reservation.Cancel()

			c.Header("X-RateLimit-Limit", string(rateLimit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", retryAfter.String())

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.Seconds(),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", string(rateLimit))
		c.Header("X-RateLimit-Remaining", string(int(l.Tokens())))

		c.Next()
	}
}

// IPRateLimit creates a simple IP-based rate limiter
func IPRateLimit(requests int, window time.Duration) gin.HandlerFunc {
	type client struct {
		count    int
		lastSeen time.Time
	}

	var (
		clients = make(map[string]*client)
		mu      sync.Mutex
	)

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.Sub(c.lastSeen) > window {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		if client, exists := clients[ip]; exists {
			if now.Sub(client.lastSeen) > window {
				// Reset count for new window
				client.count = 1
				client.lastSeen = now
			} else {
				client.count++
				if client.count > requests {
					c.JSON(http.StatusTooManyRequests, gin.H{
						"error": "rate limit exceeded",
					})
					c.Abort()
					return
				}
			}
		} else {
			clients[ip] = &client{
				count:    1,
				lastSeen: now,
			}
		}

		c.Next()
	}
}

// AdaptiveRateLimit implements an adaptive rate limiter that adjusts based on system load
func AdaptiveRateLimit(baseRate, burst int, loadFactor func() float64) gin.HandlerFunc {
	limiter := NewRateLimiter(baseRate, burst)

	return func(c *gin.Context) {
		// Adjust rate based on system load
		load := loadFactor()
		adjustedRate := int(float64(baseRate) * (2.0 - load))
		if adjustedRate < 1 {
			adjustedRate = 1
		}

		// Update rate if changed significantly
		if adjustedRate != limiter.rate {
			limiter.rate = adjustedRate
		}

		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok && uid != "" {
				key = "user:" + uid
			}
		}

		l := limiter.getLimiter(key)
		if !l.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        "rate limit exceeded",
				"current_load": load,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}