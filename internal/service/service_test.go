package service

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/playconomy/wallet-service/internal/server/dto"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *WalletService) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	service := NewWalletService(db)
	return db, mock, service
}

func TestGetWalletByUserID(t *testing.T) {
	db, mock, service := setupTestDB(t)
	defer db.Close()

	// Define test cases
	testCases := []struct {
		name           string
		userID         int
		mockSetup      func()
		expectedWallet *dto.Wallet
		expectError    bool
	}{
		{
			name:   "Success - Wallet Found",
			userID: 123,
			mockSetup: func() {
				rows := sqlmock.NewRows([]string{"id", "user_id", "balance", "created_at"}).
					AddRow(1, 123, 500.50, time.Now())
				mock.ExpectQuery("SELECT id, user_id, balance, created_at FROM wallets").
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedWallet: &dto.Wallet{
				ID:      1,
				UserID:  123,
				Balance: 500.50,
			},
			expectError: false,
		},
		{
			name:   "Wallet Not Found",
			userID: 456,
			mockSetup: func() {
				mock.ExpectQuery("SELECT id, user_id, balance, created_at FROM wallets").
					WithArgs(456).
					WillReturnError(sql.ErrNoRows)
			},
			expectedWallet: nil,
			expectError:    false,
		},
		{
			name:   "Database Error",
			userID: 789,
			mockSetup: func() {
				mock.ExpectQuery("SELECT id, user_id, balance, created_at FROM wallets").
					WithArgs(789).
					WillReturnError(fmt.Errorf("database connection lost"))
			},
			expectedWallet: nil,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations
			tc.mockSetup()

			// Call the service method
			wallet, err := service.GetWalletByUserID(tc.userID)

			// Check results
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectedWallet == nil {
				assert.Nil(t, wallet)
			} else {
				assert.NotNil(t, wallet)
				assert.Equal(t, tc.expectedWallet.ID, wallet.ID)
				assert.Equal(t, tc.expectedWallet.UserID, wallet.UserID)
				assert.Equal(t, tc.expectedWallet.Balance, wallet.Balance)
			}
		})
	}
}

func TestExchange(t *testing.T) {
	db, mock, service := setupTestDB(t)
	defer db.Close()

	// Test case: successful exchange
	t.Run("Successful Exchange", func(t *testing.T) {
		// Request data
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100,
			Source:    "game_reward",
		}

		// Exchange rate setup
		mock.ExpectQuery("SELECT to_platform_ratio FROM exchange_rates").
			WithArgs(req.GameID, req.TokenType).
			WillReturnRows(sqlmock.NewRows([]string{"to_platform_ratio"}).AddRow(2.5))

		// Transaction begin
		mock.ExpectBegin()

		// Get/create wallet
		mock.ExpectQuery("INSERT INTO wallets").
			WithArgs(req.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "balance"}).AddRow(1, 200.0))

		// Update wallet balance
		mock.ExpectExec("UPDATE wallets SET balance").
			WithArgs(450.0, 1). // 200 (current) + 250 (100 * 2.5)
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Log transaction
		mock.ExpectExec("INSERT INTO wallet_logs").
			WithArgs(1, req.UserID, req.GameID, req.TokenType, req.Amount, 250.0, req.Source).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Transaction commit
		mock.ExpectCommit()

		// Call the service method
		platformAmount, err := service.Exchange(req)

		// Check results
		assert.NoError(t, err)
		assert.Equal(t, 250.0, platformAmount)
	})

	// Test case: exchange rate not found
	t.Run("Exchange Rate Not Found", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "unknown_game",
			TokenType: "unknown_token",
			Amount:    100,
			Source:    "game_reward",
		}

		mock.ExpectQuery("SELECT to_platform_ratio FROM exchange_rates").
			WithArgs(req.GameID, req.TokenType).
			WillReturnError(sql.ErrNoRows)

		platformAmount, err := service.Exchange(req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exchange rate not found")
		assert.Equal(t, 0.0, platformAmount)
	})

	// Test case: database error
	t.Run("Database Error", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100,
			Source:    "game_reward",
		}

		mock.ExpectQuery("SELECT to_platform_ratio FROM exchange_rates").
			WithArgs(req.GameID, req.TokenType).
			WillReturnError(fmt.Errorf("database error"))

		platformAmount, err := service.Exchange(req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, platformAmount)
	})

	// Test case: transaction error
	t.Run("Transaction Error", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100,
			Source:    "game_reward",
		}

		mock.ExpectQuery("SELECT to_platform_ratio FROM exchange_rates").
			WithArgs(req.GameID, req.TokenType).
			WillReturnRows(sqlmock.NewRows([]string{"to_platform_ratio"}).AddRow(2.5))

		mock.ExpectBegin()

		mock.ExpectQuery("INSERT INTO wallets").
			WithArgs(req.UserID).
			WillReturnError(fmt.Errorf("database error"))

		mock.ExpectRollback()

		platformAmount, err := service.Exchange(req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, platformAmount)
	})
}

func TestSpend(t *testing.T) {
	db, mock, service := setupTestDB(t)
	defer db.Close()

	// Test case: successful spend
	t.Run("Successful Spend", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		// Transaction begin
		mock.ExpectBegin()

		// Get wallet with lock
		mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id = (.+) FOR UPDATE").
			WithArgs(req.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "balance"}).AddRow(1, 200.0))

		// Update wallet balance
		mock.ExpectExec("UPDATE wallets SET balance = (.+) WHERE id = (.+)").
			WithArgs(150.0, 1). // 200 (current) - 50
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Log transaction
		mock.ExpectExec("INSERT INTO wallet_logs").
			WithArgs(1, req.UserID, -req.Amount, -req.Amount, req.Reason, req.ReferenceID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Transaction commit
		mock.ExpectCommit()

		// Call the service method
		newBalance, err := service.Spend(req)

		// Check results
		assert.NoError(t, err)
		assert.Equal(t, 150.0, newBalance)
	})

	// Test case: wallet not found
	t.Run("Wallet Not Found", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      456,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-456",
		}

		mock.ExpectBegin()

		mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id = (.+) FOR UPDATE").
			WithArgs(req.UserID).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()

		newBalance, err := service.Spend(req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wallet not found")
		assert.Equal(t, 0.0, newBalance)
	})

	// Test case: insufficient funds
	t.Run("Insufficient Funds", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      789,
			Amount:      300.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-789",
		}

		mock.ExpectBegin()

		mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id = (.+) FOR UPDATE").
			WithArgs(req.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "balance"}).AddRow(2, 100.0))

		mock.ExpectRollback()

		newBalance, err := service.Spend(req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")
		assert.Equal(t, 0.0, newBalance)
	})

	// Test case: database error
	t.Run("Database Error", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		mock.ExpectBegin()

		mock.ExpectQuery("SELECT id, balance FROM wallets WHERE user_id = (.+) FOR UPDATE").
			WithArgs(req.UserID).
			WillReturnError(fmt.Errorf("database error"))

		mock.ExpectRollback()

		newBalance, err := service.Spend(req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, newBalance)
	})
}

func TestGetWalletLogs(t *testing.T) {
	db, mock, service := setupTestDB(t)
	defer db.Close()

	// Test case: successful retrieval of logs
	t.Run("Successful Logs Retrieval", func(t *testing.T) {
		userID := 123

		// Create mock rows
		rows := sqlmock.NewRows([]string{
			"game_id", "token_type", "source",
			"original_amount", "converted_amount",
			"operation", "reference_id", "created_at",
		}).
			AddRow("game1", "gold", "won", 100.0, 250.0, "exchange", nil, time.Now()).
			AddRow(nil, nil, "market_purchase", 50.0, -50.0, "spend", "ORDER-123", time.Now())

		mock.ExpectQuery("SELECT (.+) FROM wallet_logs").
			WithArgs(userID).
			WillReturnRows(rows)

		// Call the service method
		logs, err := service.GetWalletLogs(userID)

		// Check results
		assert.NoError(t, err)
		assert.Len(t, logs, 2)

		// Check first log (exchange)
		assert.Equal(t, "game1", *logs[0].GameID)
		assert.Equal(t, "gold", *logs[0].TokenType)
		assert.Equal(t, "won", *logs[0].Source)
		assert.Equal(t, 100.0, logs[0].OriginalAmount)
		assert.Equal(t, 250.0, logs[0].ConvertedAmount)
		assert.Equal(t, "exchange", logs[0].Operation)
		assert.Nil(t, logs[0].ReferenceID)

		// Check second log (spend)
		assert.Nil(t, logs[1].GameID)
		assert.Nil(t, logs[1].TokenType)
		assert.Equal(t, "market_purchase", *logs[1].Source)
		assert.Equal(t, 50.0, logs[1].OriginalAmount)
		assert.Equal(t, -50.0, logs[1].ConvertedAmount)
		assert.Equal(t, "spend", logs[1].Operation)
		assert.Equal(t, "ORDER-123", *logs[1].ReferenceID)
	})

	// Test case: empty logs
	t.Run("No Logs Found", func(t *testing.T) {
		userID := 456

		rows := sqlmock.NewRows([]string{
			"game_id", "token_type", "source",
			"original_amount", "converted_amount",
			"operation", "reference_id", "created_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM wallet_logs").
			WithArgs(userID).
			WillReturnRows(rows)

		logs, err := service.GetWalletLogs(userID)

		assert.NoError(t, err)
		assert.Empty(t, logs)
	})

	// Test case: database error
	t.Run("Database Error", func(t *testing.T) {
		userID := 789

		mock.ExpectQuery("SELECT (.+) FROM wallet_logs").
			WithArgs(userID).
			WillReturnError(fmt.Errorf("database error"))

		logs, err := service.GetWalletLogs(userID)

		assert.Error(t, err)
		assert.Nil(t, logs)
	})
}
