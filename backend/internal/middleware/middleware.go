package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" || len(requestID) > 80 {
			raw := make([]byte, 12)
			if _, err := rand.Read(raw); err != nil {
				requestID = time.Now().UTC().Format("20060102150405.000000")
			} else {
				requestID = hex.EncodeToString(raw)
			}
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func AccessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("http_request", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "bytes", c.Writer.Size(), "latency_ms", time.Since(started).Milliseconds(), "client_ip", c.ClientIP(), "request_id", c.GetString("request_id"))
	}
}

func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic_recovered", "request_id", c.GetString("request_id"), "panic", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": service.CodeInternal, "message": "unexpected server error", "request_id": c.GetString("request_id")}})
	})
}

func AuthMiddleware(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			unauthorized(c, "bearer token is required")
			return
		}
		claims, err := auth.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			unauthorized(c, "access token is invalid or expired")
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RBACMiddleware(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if !allowed[role.(string)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": service.CodeForbidden, "message": "role is not permitted for this operation", "request_id": c.GetString("request_id")}})
			return
		}
		c.Next()
	}
}

type visitor struct {
	count     int
	expiresAt time.Time
}
type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	visitors  map[string]visitor
	lastSweep time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, visitors: make(map[string]visitor), lastSweep: time.Now()}
}

func (l *RateLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > l.window {
		for k, value := range l.visitors {
			if now.After(value.expiresAt) {
				delete(l.visitors, k)
			}
		}
		l.lastSweep = now
	}
	current := l.visitors[key]
	if current.expiresAt.IsZero() || now.After(current.expiresAt) {
		current = visitor{0, now.Add(l.window)}
	}
	if current.count >= l.limit {
		return false, time.Until(current.expiresAt)
	}
	current.count++
	l.visitors[key] = current
	return true, time.Until(current.expiresAt)
}

func RateLimitMiddleware(limiter *RateLimiter, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retry := limiter.Allow(scope+":"+c.ClientIP(), time.Now())
		if !allowed {
			c.Header("Retry-After", time.Duration(retry.Seconds()).String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "RATE_LIMITED", "message": "request rate limit exceeded", "request_id": c.GetString("request_id")}})
			return
		}
		c.Next()
	}
}

func CORSMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": service.CodeUnauthorized, "message": message, "request_id": c.GetString("request_id")}})
}
