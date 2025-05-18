package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/playconomy/wallet-service/internal/model"
	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/observability/metrics"
	"github.com/playconomy/wallet-service/internal/observability/tracing"
	"github.com/playconomy/wallet-service/internal/repository"
	"github.com/playconomy/wallet-service/internal/server/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock observability components for testing
type mockObservability struct {
	logger  *zap.Logger
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
}

func setupTestService(t *testing.T) (*repository.MockRepository, WalletServiceInterface) {
	// Create mock repository
	mockRepo := new(repository.MockRepository)

	// Create test observability
	obs := observability.NewTestObservability()

	// Create service with mock repository
	service := NewWalletService(mockRepo, obs)
	
	// Return the service as an interface to ensure we're testing the interface not the implementation
	return mockRepo, service
}

func TestGetWalletByUserID(t *testing.T) {
	mockRepo, service := setupTestService(t)

	// Create standard context for tests
	ctx := context.Background()

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
				mockWallet := &model.Wallet{
					ID:        1,
					UserID:    123,
					Balance:   500.50,
					CreatedAt: time.Now(),
				}
				mockRepo.On("GetWalletByUserID", mock.Anything, 123).Return(mockWallet, nil).Once()
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
				mockRepo.On("GetWalletByUserID", mock.Anything, 456).Return(nil, nil).Once()
			},
			expectedWallet: nil,
			expectError:    false,
		},
		{
			name:   "Database Error",
			userID: 789,
			mockSetup: func() {
				mockRepo.On("GetWalletByUserID", mock.Anything, 789).Return(nil, fmt.Errorf("database connection lost")).Once()
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
			wallet, err := service.GetWalletByUserID(ctx, tc.userID)

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
			
			// Verify that all expected calls were made
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestExchange(t *testing.T) {
	mockRepo, service := setupTestService(t)
	ctx := context.Background()

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

		// Mock repository responses
		exchangeRate := &model.ExchangeRate{
			ID:              1,
			GameID:          "game1",
			TokenType:       "gold",
			ToPlatformRatio: 2.5,
			CreatedAt:       time.Now(),
		}
		
		wallet := &model.Wallet{
			ID:        1,
			UserID:    123,
			Balance:   200.0, // Current balance
			CreatedAt: time.Now(),
		}
		
		updatedWallet := &model.Wallet{
			ID:        1,
			UserID:    123,
			Balance:   450.0, // Balance after exchange (200 + 100*2.5)
			CreatedAt: time.Now(),
		}
		
		mockTx := new(repository.MockTransaction)
		
		// Create expected wallet log
		expectedLog := &model.WalletLog{
			WalletID:       1,
			UserID:         123,
			GameID:         &req.GameID,
			TokenType:      &req.TokenType,
			Amount:         100,
			PlatformAmount: 250.0, // 100 * 2.5
			Source:         model.TransactionExchange,
		}
		
		returnedLog := &model.WalletLog{
			ID:             1,
			WalletID:       1,
			UserID:         123,
			GameID:         &req.GameID,
			TokenType:      &req.TokenType,
			Amount:         100,
			PlatformAmount: 250.0,
			Source:         model.TransactionExchange,
			CreatedAt:      time.Now(),
		}

		// Set up expectations
		mockRepo.On("GetExchangeRate", mock.Anything, req.GameID, req.TokenType).Return(exchangeRate, nil).Once()
		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(wallet, nil).Once()
		mockRepo.On("UpdateWalletBalance", mock.Anything, req.UserID, 450.0, mockTx).Return(updatedWallet, nil).Once()
		mockRepo.On("CreateWalletLog", mock.Anything, mock.MatchedBy(func(log *model.WalletLog) bool {
			return log.UserID == expectedLog.UserID && 
				   log.PlatformAmount == expectedLog.PlatformAmount
		}), mockTx).Return(returnedLog, nil).Once()
		mockTx.On("Commit").Return(nil).Once()

		// Call the service method
		platformAmount, err := service.Exchange(ctx, req)

		// Check results
		assert.NoError(t, err)
		assert.Equal(t, 250.0, platformAmount)
		
		// Verify that all expected calls were made
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
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

		mockRepo.On("GetExchangeRate", mock.Anything, req.GameID, req.TokenType).Return(nil, nil).Once()

		platformAmount, err := service.Exchange(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exchange rate not found")
		assert.Equal(t, 0.0, platformAmount)
		
		mockRepo.AssertExpectations(t)
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

		mockRepo.On("GetExchangeRate", mock.Anything, req.GameID, req.TokenType).Return(nil, fmt.Errorf("database error")).Once()

		platformAmount, err := service.Exchange(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, platformAmount)
		
		mockRepo.AssertExpectations(t)
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

		exchangeRate := &model.ExchangeRate{
			ID:              1,
			GameID:          "game1",
			TokenType:       "gold",
			ToPlatformRatio: 2.5,
			CreatedAt:       time.Now(),
		}
		
		mockTx := new(repository.MockTransaction)

		mockRepo.On("GetExchangeRate", mock.Anything, req.GameID, req.TokenType).Return(exchangeRate, nil).Once()
		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(nil, fmt.Errorf("database error")).Once()
		mockTx.On("Rollback").Return(nil).Once()

		platformAmount, err := service.Exchange(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, platformAmount)
		
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})
}

func TestSpend(t *testing.T) {
	mockRepo, service := setupTestService(t)
	ctx := context.Background()

	// Test case: successful spend
	t.Run("Successful Spend", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		wallet := &model.Wallet{
			ID:        1,
			UserID:    123,
			Balance:   200.0, // Current balance
			CreatedAt: time.Now(),
		}
		
		updatedWallet := &model.Wallet{
			ID:        1,
			UserID:    123,
			Balance:   150.0, // Balance after spend (200 - 50)
			CreatedAt: time.Now(),
		}
		
		mockTx := new(repository.MockTransaction)
		
		// Create expected wallet log
		expectedLog := &model.WalletLog{
			WalletID:       1,
			UserID:         123,
			Amount:         -req.Amount,
			PlatformAmount: -req.Amount,
			Source:         req.Reason,
			ReferenceID:    &req.ReferenceID,
		}
		
		returnedLog := &model.WalletLog{
			ID:             1,
			WalletID:       1,
			UserID:         123,
			Amount:         -req.Amount,
			PlatformAmount: -req.Amount,
			Source:         req.Reason,
			ReferenceID:    &req.ReferenceID,
			CreatedAt:      time.Now(),
		}

		// Set up expectations
		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(wallet, nil).Once()
		mockRepo.On("SpendFromWallet", mock.Anything, req.UserID, req.Amount, mockTx).Return(updatedWallet, nil).Once()
		mockRepo.On("CreateWalletLog", mock.Anything, mock.MatchedBy(func(log *model.WalletLog) bool {
			return log.UserID == expectedLog.UserID && 
				   log.PlatformAmount == expectedLog.PlatformAmount &&
				   log.Source == expectedLog.Source
		}), mockTx).Return(returnedLog, nil).Once()
		mockTx.On("Commit").Return(nil).Once()

		// Call the service method
		newBalance, err := service.Spend(ctx, req)

		// Check results
		assert.NoError(t, err)
		assert.Equal(t, 150.0, newBalance)
		
		// Verify that all expected calls were made
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	// Test case: wallet not found
	t.Run("Wallet Not Found", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      456,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-456",
		}

		mockTx := new(repository.MockTransaction)

		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(nil, nil).Once()
		mockTx.On("Rollback").Return(nil).Once()

		newBalance, err := service.Spend(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wallet not found")
		assert.Equal(t, 0.0, newBalance)
		
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	// Test case: insufficient funds
	t.Run("Insufficient Funds", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      789,
			Amount:      300.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-789",
		}

		wallet := &model.Wallet{
			ID:        2,
			UserID:    789,
			Balance:   100.0, // Not enough balance for 300.0 spend
			CreatedAt: time.Now(),
		}
		
		mockTx := new(repository.MockTransaction)

		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(wallet, nil).Once()
		mockTx.On("Rollback").Return(nil).Once()

		newBalance, err := service.Spend(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")
		assert.Equal(t, 0.0, newBalance)
		
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	// Test case: database error
	t.Run("Database Error", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		mockTx := new(repository.MockTransaction)

		mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil).Once()
		mockRepo.On("GetWalletByUserIDForUpdate", mock.Anything, req.UserID, mockTx).Return(nil, fmt.Errorf("database error")).Once()
		mockTx.On("Rollback").Return(nil).Once()

		newBalance, err := service.Spend(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, 0.0, newBalance)
		
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})
}

func TestGetWalletLogs(t *testing.T) {
	mockRepo, service := setupTestService(t)
	ctx := context.Background()

	// Test case: successful retrieval of logs
	t.Run("Successful Logs Retrieval", func(t *testing.T) {
		userID := 123
		
		// Create test date
		now := time.Now()
		
		// Create mock wallet logs
		gameID := "game1"
		tokenType := "gold"
		source := "won"
		refID := "ORDER-123"
		mockLogs := []*model.WalletLog{
			{
				ID:             1,
				WalletID:       1,
				UserID:         userID,
				GameID:         &gameID,
				TokenType:      &tokenType,
				Source:         source,
				Amount:         100.0,
				PlatformAmount: 250.0,
				CreatedAt:      now,
			},
			{
				ID:             2,
				WalletID:       1,
				UserID:         userID,
				GameID:         nil,
				TokenType:      nil,
				Source:         "market_purchase",
				Amount:         -50.0,
				PlatformAmount: -50.0,
				ReferenceID:    &refID,
				CreatedAt:      now,
			},
		}

		// Set up expectation for default limits
		mockRepo.On("GetWalletLogs", mock.Anything, userID, 50, 0).Return(mockLogs, nil).Once()

		// Call the service method
		logs, err := service.GetWalletLogs(ctx, userID)

		// Check results
		assert.NoError(t, err)
		assert.Len(t, logs, 2)

		// Check first log (exchange)
		assert.Equal(t, "game1", *logs[0].GameID)
		assert.Equal(t, "gold", *logs[0].TokenType)
		assert.Equal(t, "won", *logs[0].Source)
		assert.Equal(t, 100.0, logs[0].OriginalAmount)
		assert.Equal(t, 250.0, logs[0].ConvertedAmount)
		assert.Equal(t, model.TransactionExchange, logs[0].Operation)
		assert.Nil(t, logs[0].ReferenceID)

		// Check second log (spend)
		assert.Nil(t, logs[1].GameID)
		assert.Nil(t, logs[1].TokenType)
		assert.Equal(t, "market_purchase", *logs[1].Source)
		assert.Equal(t, -50.0, logs[1].OriginalAmount)
		assert.Equal(t, -50.0, logs[1].ConvertedAmount)
		assert.Equal(t, model.TransactionSpend, logs[1].Operation)
		assert.Equal(t, "ORDER-123", *logs[1].ReferenceID)
		
		mockRepo.AssertExpectations(t)
	})

	// Test case: empty logs
	t.Run("No Logs Found", func(t *testing.T) {
		userID := 456
		
		// Empty logs list
		var emptyLogs []*model.WalletLog
		
		mockRepo.On("GetWalletLogs", mock.Anything, userID, 50, 0).Return(emptyLogs, nil).Once()

		logs, err := service.GetWalletLogs(ctx, userID)

		assert.NoError(t, err)
		assert.Empty(t, logs)
		
		mockRepo.AssertExpectations(t)
	})

	// Test case: database error
	t.Run("Database Error", func(t *testing.T) {
		userID := 789

		mockRepo.On("GetWalletLogs", mock.Anything, userID, 50, 0).Return(nil, fmt.Errorf("database error")).Once()

		logs, err := service.GetWalletLogs(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, logs)
		
		mockRepo.AssertExpectations(t)
	})
}
