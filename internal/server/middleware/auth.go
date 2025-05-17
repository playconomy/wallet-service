package middleware

import (
	"strconv"

	"github.com/playconomy/wallet-service/internal/server/dto"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware checks if auth headers are present and valid
// @Description Middleware to authenticate requests using X-User-Id, X-User-Email, and X-User-Role headers
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check required headers
		userID := c.Get("X-User-Id")
		userEmail := c.Get("X-User-Email")
		userRole := c.Get("X-User-Role")

		// Validate headers existence
		if userID == "" || userEmail == "" || userRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.GenericResponse{
				Success: false,
				Error:   "Authentication required",
			})
		}

		// Parse and validate user ID
		id, err := strconv.Atoi(userID)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.GenericResponse{
				Success: false,
				Error:   "Invalid user ID",
			})
		}

		// Store user information in context
		c.Locals("user_id", id)
		c.Locals("user_email", userEmail)
		c.Locals("user_role", userRole)

		// Continue with next handler
		return c.Next()
	}
}
