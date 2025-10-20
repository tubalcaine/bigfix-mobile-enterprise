package main

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLoggingMiddleware logs detailed information about each HTTP request
func RequestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Get client information
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Log request start
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
		logFunc := logger.Info
		if statusCode >= 500 {
			logFunc = logger.Error
		} else if statusCode >= 400 {
			logFunc = logger.Warn
		}

		// Log request completion
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
