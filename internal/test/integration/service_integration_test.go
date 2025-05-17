package integration

import (
	"testing"
	"time"

	"github.com/playconomy/wallet-service/internal/server/dto"
	"github.com/playconomy/wallet-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletServiceIntegration(t *testing.T) {
	// Skip if short mode is enabled (for quick test runs that skip integration tests)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get test DB
	db := GetTestDB()

	// Create wallet service with actual DB connection
	walletService := service.NewWalletService(db)

	// Clear test data before each test
	t.Run("GetWalletByUserID", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet
		userID := 123
		CreateTestWallet(t, userID, 100.0)

		// Test getting the wallet
		wallet, err := walletService.GetWalletByUserID(userID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, wallet)
		assert.Equal(t, userID, wallet.UserID)
		assert.Equal(t, 100.0, wallet.Balance)

		// Test non-existent wallet
		nonExistentID := 999
		wallet, err = walletService.GetWalletByUserID(nonExistentID)

		require.NoError(t, err)
		assert.Nil(t, wallet)
	})

	t.Run("Exchange", func(t *testing.T) {
		ClearTestData(t)

		// Create exchange request
		req := &dto.ExchangeRequest{
			UserID:    456,
			GameID:    "game1", // Our test data has a 2.5x ratio for game1/gold
			TokenType: "gold",
			Amount:    100.0,
			Source:    "won",
		}

		// Execute exchange (should create wallet if it doesn't exist)
		newBalance, err := walletService.Exchange(req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 250.0, newBalance) // 100 * 2.5

		// Verify wallet was created with correct balance
		wallet, err := walletService.GetWalletByUserID(req.UserID)
		require.NoError(t, err)
		assert.NotNil(t, wallet)
		assert.Equal(t, 250.0, wallet.Balance)

		// Test invalid game/token combination
		invalidReq := &dto.ExchangeRequest{
			UserID:    456,
			GameID:    "invalid_game",
			TokenType: "invalid_token",
			Amount:    100.0,
			Source:    "won",
		}

		_, err = walletService.Exchange(invalidReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exchange rate not found")
	})

	t.Run("Spend", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet with balance
		userID := 789
		CreateTestWallet(t, userID, 500.0)

		// Create spend request
		req := &dto.SpendRequest{
			UserID:      userID,
			Amount:      200.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		// Execute spend
		newBalance, err := walletService.Spend(req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 300.0, newBalance) // 500 - 200

		// Verify wallet was updated
		wallet, err := walletService.GetWalletByUserID(userID)
		require.NoError(t, err)
		assert.NotNil(t, wallet)
		assert.Equal(t, 300.0, wallet.Balance)

		// Test insufficient funds
		insufficientReq := &dto.SpendRequest{
			UserID:      userID,
			Amount:      500.0, // More than current balance
			Reason:      "market_purchase",
			ReferenceID: "ORDER-456",
		}

		_, err = walletService.Spend(insufficientReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")

		// Test non-existent wallet
		nonExistentReq := &dto.SpendRequest{
			UserID:      999,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-789",
		}

		_, err = walletService.Spend(nonExistentReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wallet not found")
	})

	t.Run("GetWalletLogs", func(t *testing.T) {
		ClearTestData(t)

		// Create test wallet
		userID := 123
		walletID := CreateTestWallet(t, userID, 100.0)

		// Insert some test logs
		_, err := db.Exec(`
			INSERT INTO wallet_logs (wallet_id, user_id, game_id, token_type, amount, platform_amount, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, walletID, userID, "game1", "gold", 100.0, 250.0, "won", time.Now())
		require.NoError(t, err)

		// Insert a spend log
		_, err = db.Exec(`
			INSERT INTO wallet_logs (wallet_id, user_id, amount, platform_amount, source, reference_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, walletID, userID, -50.0, -50.0, "market_purchase", "ORDER-123", time.Now())
		require.NoError(t, err)

		// Get logs
		logs, err := walletService.GetWalletLogs(userID)

		// Assert
		require.NoError(t, err)
		assert.Len(t, logs, 2)

		// Check log entries (order may vary due to timestamp, so we don't check specific order)
		hasExchange := false
		hasSpend := false

		for _, log := range logs {
			if log.Operation == "exchange" {
				hasExchange = true
				assert.NotNil(t, log.GameID)
				assert.Equal(t, "game1", *log.GameID)
				assert.NotNil(t, log.TokenType)
				assert.Equal(t, "gold", *log.TokenType)
				assert.Equal(t, 100.0, log.OriginalAmount)
				assert.Equal(t, 250.0, log.ConvertedAmount)
			}

			if log.Operation == "spend" {
				hasSpend = true
				assert.Nil(t, log.GameID)
				assert.Nil(t, log.TokenType)
				assert.Equal(t, 50.0, log.OriginalAmount)
				assert.Equal(t, -50.0, log.ConvertedAmount)
				assert.NotNil(t, log.ReferenceID)
				assert.Equal(t, "ORDER-123", *log.ReferenceID)
			}
		}

		assert.True(t, hasExchange, "Should have an exchange log entry")
		assert.True(t, hasSpend, "Should have a spend log entry")

		// Test empty logs
		logs, err = walletService.GetWalletLogs(999)
		require.NoError(t, err)
		assert.Empty(t, logs)
	})
}
