// Package middleware provides HTTP middlewares for the application
package middleware

import (
	"time"

	"github.com/playconomy/wallet-service/internal/observability/logger"
	"github.com/playconomy/wallet-service/internal/observability/metrics"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// LoggingMiddleware logs information about each request
func LoggingMiddleware() fiber.Handler {
	log := logger.GetLogger()

	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()
		ip := c.IP()
		userAgent := c.Get("User-Agent")

		// Add requestID to context if it exists
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = c.Locals("requestid").(string)
		}

		// Continue to next handler
		err := c.Next()

		// Record the time needed to process the request
		latency := time.Since(start)
		status := c.Response().StatusCode()

		// Log details about the request
		log.Info("HTTP Request",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.String("ip", ip),
			zap.String("user_agent", userAgent),
			zap.Duration("latency", latency),
			zap.String("error", err.Error()),
		)

		return err
	}
}

// MetricsMiddleware records metrics about each request
func MetricsMiddleware(metrics *metrics.Metrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Process request
		err := c.Next()

		// Record metrics
		status := c.Response().StatusCode()
		duration := time.Since(start).Seconds()

		metrics.RecordRequest(method, path, status)
		metrics.ObserveRequestDuration(method, path, duration)

		return err
	}
}

// RequestIDMiddleware adds a unique ID to each request if not present
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if X-Request-ID header exists
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			// Generate a new ID and set it
			requestID = generateRequestID()
			c.Set("X-Request-ID", requestID)
		}

		// Store in locals for other handlers
		c.Locals("requestid", requestID)
		
		return c.Next()
	}
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	// Simple UUID v4 implementation - in production, use a proper UUID library
	return fiber.UUID()
}
