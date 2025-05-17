package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/playconomy/wallet-service/internal/server/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware() (*fiber.App, func() string) {
	app := fiber.New()
	var responseData string

	// Apply auth middleware to route
	app.Use(middleware.AuthMiddleware())

	// Handler that records whether middleware passed
	app.Get("/test", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		userEmail := c.Locals("user_email")
		userRole := c.Locals("user_role")

		responseData = "Success"

		// Check if all locals are set properly
		if userID == nil || userEmail == nil || userRole == nil {
			responseData = "Missing context values"
		}

		return c.SendString("OK")
	})

	return app, func() string { return responseData }
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("Valid Headers", func(t *testing.T) {
		app, getResponse := setupTestMiddleware()

		// Create request with valid headers
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-Id", "123")
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Success", getResponse())
	})

	t.Run("Missing User ID", func(t *testing.T) {
		app, _ := setupTestMiddleware()

		// Create request with missing user ID
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Missing User Email", func(t *testing.T) {
		app, _ := setupTestMiddleware()

		// Create request with missing email
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-Id", "123")
		req.Header.Set("X-User-Role", "user")

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Missing User Role", func(t *testing.T) {
		app, _ := setupTestMiddleware()

		// Create request with missing role
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-Id", "123")
		req.Header.Set("X-User-Email", "user@example.com")

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid User ID Format", func(t *testing.T) {
		app, _ := setupTestMiddleware()

		// Create request with invalid user ID
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-Id", "not-a-number")
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
