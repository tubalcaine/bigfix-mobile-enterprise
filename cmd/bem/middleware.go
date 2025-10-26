package main

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLoggingMiddleware logs detailed information about each HTTP request using slog.
// This provides structured logging separate from Gin's colorized [GIN] console logs.
//
// Log levels used:
//   - DEBUG: Request start (only when log_level=DEBUG)
//   - INFO:  Request completion with 2xx/3xx status (always logged when log_level=INFO or lower)
//   - WARN:  Request completion with 4xx status
//   - ERROR: Request completion with 5xx status
//
// This middleware respects log_to_file and log_to_console settings.
func RequestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Get client information
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Log request start (DEBUG level only)
		logger.Debug("HTTP request started",
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"user_agent", c.Request.UserAgent(),
			"content_type", c.Request.Header.Get("Content-Type"),
			"has_auth", c.Request.Header.Get("Authorization") != "",
		)

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// Determine log level based on status code
		// 2xx/3xx = INFO, 4xx = WARN, 5xx = ERROR
		logFunc := logger.Info
		if statusCode >= 500 {
			logFunc = logger.Error
		} else if statusCode >= 400 {
			logFunc = logger.Warn
		}

		// Log request completion with appropriate level
		logFunc("HTTP request completed",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"client_ip", clientIP,
			"bytes_written", c.Writer.Size(),
		)
	}
}

// ErrorLoggingMiddleware logs any errors that occur during request processing
func ErrorLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error("Request error",
					"error", err.Error(),
					"type", err.Type,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
				)
			}
		}
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					"error", err,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
				)

				c.JSON(500, gin.H{
					"error": "Internal server error",
				})
			}
		}()

		c.Next()
	}
}
