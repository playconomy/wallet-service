package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/playconomy/wallet-service/internal/server/dto"
	"github.com/playconomy/wallet-service/internal/server/handler"
	"github.com/playconomy/wallet-service/internal/server/middleware"
	"github.com/playconomy/wallet-service/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) *fiber.App {
	testRepo := GetTestRepository(t)
	app := fiber.New()

	// Create observability for test
	obs := service.GetTestObservability()
	
	// Create service and handler with interfaces
	var walletService service.WalletServiceInterface = service.NewWalletService(testRepo, obs)
	var walletHandler handler.WalletHandlerInterface = handler.NewWalletHandler(walletService, obs)

	// Setup test routes similar to actual app
	api := app.Group("/", middleware.AuthMiddleware())
	api.Get("/:user_id", walletHandler.GetWallet)
	api.Get("/:user_id/logs", walletHandler.GetWalletLogs)
	api.Post("/exchange", walletHandler.Exchange)
	api.Post("/spend", walletHandler.Spend)

	return app
}

func TestAPIIntegration(t *testing.T) {
	// Skip if short mode is enabled
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	app := setupTestApp(t)

	t.Run("GetWallet", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet
		userID := 123
		CreateTestWallet(t, userID, 100.0)

		// Prepare request
		req, err := http.NewRequest("GET", fmt.Sprintf("/%d", userID), nil)
		require.NoError(t, err)

		// Set auth headers (normally done by middleware)
		req.Header.Set("X-User-Id", fmt.Sprintf("%d", userID))
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Execute request
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response dto.WalletResponse
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Verify response
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
		assert.Equal(t, userID, response.Data.UserID)
		assert.Equal(t, 100.0, response.Data.Balance)
	})

	t.Run("Exchange", func(t *testing.T) {
		ClearTestData(t)

		// Prepare exchange request
		exchangeReq := dto.ExchangeRequest{
			UserID:    456,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100.0,
			Source:    "won",
		}

		// Convert to JSON
		reqBody, err := json.Marshal(exchangeReq)
		require.NoError(t, err)

		// Prepare HTTP request
		req, err := http.NewRequest("POST", "/exchange", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Set auth headers
		req.Header.Set("X-User-Id", fmt.Sprintf("%d", exchangeReq.UserID))
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Execute request
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response dto.ExchangeResponse
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Verify response
		assert.True(t, response.Success)
		assert.Equal(t, 250.0, response.NewBalance) // 100 * 2.5
	})

	t.Run("Spend", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet
		userID := 789
		CreateTestWallet(t, userID, 500.0)

		// Prepare spend request
		spendReq := dto.SpendRequest{
			UserID:      userID,
			Amount:      200.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		// Convert to JSON
		reqBody, err := json.Marshal(spendReq)
		require.NoError(t, err)

		// Prepare HTTP request
		req, err := http.NewRequest("POST", "/spend", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Set auth headers
		req.Header.Set("X-User-Id", fmt.Sprintf("%d", spendReq.UserID))
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Execute request
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response dto.SpendResponse
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Verify response
		assert.True(t, response.Success)
		assert.Equal(t, 300.0, response.NewBalance) // 500 - 200

		// Test insufficient funds
		spendReq.Amount = 500.0 // More than current balance

		reqBody, err = json.Marshal(spendReq)
		require.NoError(t, err)

		req, err = http.NewRequest("POST", "/spend", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", fmt.Sprintf("%d", spendReq.UserID))
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("GetWalletLogs", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet and logs
		userID := 123
		walletID := CreateTestWallet(t, userID, 100.0)

		// Insert some test logs directly into the database
		db := GetTestDB()
		_, err := db.Exec(`
			INSERT INTO wallet_logs (wallet_id, user_id, game_id, token_type, amount, platform_amount, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, walletID, userID, "game1", "gold", 100.0, 250.0, "won")
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO wallet_logs (wallet_id, user_id, amount, platform_amount, source, reference_id)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, walletID, userID, -50.0, -50.0, "market_purchase", "ORDER-123")
		require.NoError(t, err)

		// Prepare request
		req, err := http.NewRequest("GET", fmt.Sprintf("/%d/logs", userID), nil)
		require.NoError(t, err)

		// Set auth headers
		req.Header.Set("X-User-Id", fmt.Sprintf("%d", userID))
		req.Header.Set("X-User-Email", "user@example.com")
		req.Header.Set("X-User-Role", "user")

		// Execute request
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response dto.WalletLogsResponse
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		// Verify response
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data)
		assert.Len(t, response.Data, 2)
	})
}
