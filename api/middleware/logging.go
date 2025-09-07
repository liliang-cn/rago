package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RequestID adds a unique request ID to the context
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID exists in header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// Logger creates a logging middleware with structured logging
func Logger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get request ID
		requestID, _ := c.Get("request_id")

		// Log request start
		logger.Debug().
			Str("request_id", requestID.(string)).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", raw).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("Request started")

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get response status
		status := c.Writer.Status()

		// Determine log level based on status code
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = logger.Error()
		case status >= 400:
			event = logger.Warn()
		default:
			event = logger.Info()
		}

		// Log request completion
		event.
			Str("request_id", requestID.(string)).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", raw).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int("status", status).
			Dur("latency", latency).
			Int("body_size", c.Writer.Size()).
			Msg("Request completed")

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.Error().
					Str("request_id", requestID.(string)).
					Err(e.Err).
					Str("type", e.Type.String()).
					Any("meta", e.Meta).
					Msg("Request error")
			}
		}
	}
}

// Recovery recovers from panics and logs them
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID
				requestID, _ := c.Get("request_id")

				// Log the panic
				logger := zerolog.New(nil).With().
					Str("request_id", requestID.(string)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Str("ip", c.ClientIP()).
					Logger()

				logger.Error().
					Interface("error", err).
					Msg("Panic recovered")

				// Return 500 error
				c.JSON(500, gin.H{
					"error":      "internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// bodyLogWriter captures response body for logging
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// BodyLogger logs request and response bodies (use with caution in production)
func BodyLogger(logger zerolog.Logger, maxSize int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for large requests
		if c.Request.ContentLength > int64(maxSize) {
			c.Next()
			return
		}

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response body
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Get request ID
		requestID, _ := c.Get("request_id")

		// Log bodies (be careful with sensitive data)
		if len(requestBody) > 0 || blw.body.Len() > 0 {
			logger.Debug().
				Str("request_id", requestID.(string)).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Bytes("request_body", requestBody).
				Str("response_body", blw.body.String()).
				Msg("Request/Response bodies")
		}
	}
}

// Metrics middleware for Prometheus metrics
func Metrics() gin.HandlerFunc {
	// This would integrate with Prometheus metrics
	// Implementation depends on metrics setup
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()

		// In production, you would record these to Prometheus
		_ = duration
		_ = status
		_ = method
		_ = path
	}
}