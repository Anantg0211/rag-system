package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		started := time.Now()
		c.Next()
		logger.Info("http request",
			"request_id", requestID, "method", c.Request.Method, "path", c.Request.URL.Path,
			"status", c.Writer.Status(), "latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(), "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get(RequestIDKey)
		logger.Error("panic recovered", "request_id", requestID, "panic", recovered, "stack", string(debug.Stack()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "internal_error", "message": "an internal error occurred"},
		})
	})
}
