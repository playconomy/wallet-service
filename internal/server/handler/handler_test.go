package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/server/dto"
	"github.com/playconomy/wallet-service/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockWalletService is a mock implementation of WalletServiceInterface for testing
type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) GetWalletByUserID(ctx context.Context, userID int) (*dto.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.Wallet), args.Error(1)
}

func (m *MockWalletService) Exchange(ctx context.Context, req *dto.ExchangeRequest) (float64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockWalletService) Spend(ctx context.Context, req *dto.SpendRequest) (float64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockWalletService) GetWalletLogs(ctx context.Context, userID int) ([]dto.WalletLogEntry, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.WalletLogEntry), args.Error(1)
}

// Compile-time verification that MockWalletService implements WalletServiceInterface
var _ service.WalletServiceInterface = (*MockWalletService)(nil)

// Setup test app with mocked service
func setupTestApp(t *testing.T) (*fiber.App, *MockWalletService, WalletHandlerInterface) {
	app := fiber.New()
	mockService := new(MockWalletService)
	
	// Create test logger
	logger, _ := zap.NewDevelopment()
	
	// Create observability
	obs := &observability.Observability{
		Logger: &observability.Logger{Logger: logger},
	}
	
	// Create handler interface
	handler := NewWalletHandler(mockService, obs)

	// Setup routes
	app.Get("/:user_id", handler.GetWallet)
	app.Post("/exchange", handler.Exchange)
	app.Post("/spend", handler.Spend)
	app.Get("/:user_id/logs", handler.GetWalletLogs)

	return app, mockService, handler
}

	return app, mockService
}

func TestGetWallet(t *testing.T) {
	app, mockService, _ := setupTestApp(t)

	t.Run("Success", func(t *testing.T) {
		// Setup
		wallet := &dto.Wallet{
			ID:      1,
			UserID:  123,
			Balance: 100.0,
		}
		mockService.On("GetWalletByUserID", mock.Anything, 123).Return(wallet, nil).Once()

		// Create request
		req := httptest.NewRequest("GET", "/123", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values that would be set by middleware
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("requestid", "test-request-id")
		ctx.Locals("user_id", 123)
		ctx.Locals("user_role", "user")
		ctx.Params().Set("user_id", "123")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Parse response body
		var response dto.WalletResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
		assert.Equal(t, wallet.ID, response.Data.ID)
		assert.Equal(t, wallet.UserID, response.Data.UserID)
		assert.Equal(t, wallet.Balance, response.Data.Balance)
		
		// Verify that all expected calls were made
		mockService.AssertExpectations(t)
	})
	})

	t.Run("Not Found", func(t *testing.T) {
		// Setup
		mockService.On("GetWalletByUserID", 456).Return(nil, nil).Once()

		// Create request
		req := httptest.NewRequest("GET", "/456", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 456)
		ctx.Locals("user_role", "user")
		ctx.Params().Set("user_id", "456")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		// Parse response body
		var response dto.WalletResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "not found")
	})

	t.Run("Forbidden Access", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/789", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values - user trying to access another user's wallet
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 123) // Authenticated as user 123
		ctx.Locals("user_role", "user")
		ctx.Params().Set("user_id", "789") // But trying to access user 789's wallet

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

		// Parse response body
		var response dto.WalletResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "own wallet")
	})

	t.Run("Server Error", func(t *testing.T) {
		// Setup
		mockService.On("GetWalletByUserID", 999).Return(nil, errors.New("database error")).Once()

		// Create request
		req := httptest.NewRequest("GET", "/999", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 999)
		ctx.Locals("user_role", "user")
		ctx.Params().Set("user_id", "999")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestExchange(t *testing.T) {
	app, mockService := setupTestApp(t)

	t.Run("Success", func(t *testing.T) {
		// Setup
		exchangeReq := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100.0,
			Source:    "won",
		}
		mockService.On("Exchange", mock.AnythingOfType("*dto.ExchangeRequest")).
			Return(250.0, nil).Once()

		// Create request
		reqBody, _ := json.Marshal(exchangeReq)
		req := httptest.NewRequest("POST", "/exchange", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 123)
		ctx.Locals("user_role", "user")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Parse response body
		var response dto.ExchangeResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, 250.0, response.NewBalance)
	})
}

func TestSpend(t *testing.T) {
	app, mockService := setupTestApp(t)

	t.Run("Success", func(t *testing.T) {
		// Setup
		spendReq := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}
		mockService.On("Spend", mock.AnythingOfType("*dto.SpendRequest")).
			Return(150.0, nil).Once()

		// Create request
		reqBody, _ := json.Marshal(spendReq)
		req := httptest.NewRequest("POST", "/spend", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 123)
		ctx.Locals("user_role", "user")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Parse response body
		var response dto.SpendResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, 150.0, response.NewBalance)
	})

	t.Run("Insufficient Funds", func(t *testing.T) {
		// Setup
		spendReq := &dto.SpendRequest{
			UserID:      123,
			Amount:      500.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}
		mockService.On("Spend", mock.AnythingOfType("*dto.SpendRequest")).
			Return(0.0, errors.New("insufficient funds")).Once()

		// Create request
		reqBody, _ := json.Marshal(spendReq)
		req := httptest.NewRequest("POST", "/spend", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 123)
		ctx.Locals("user_role", "user")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetWalletLogs(t *testing.T) {
	app, mockService := setupTestApp(t)

	t.Run("Success", func(t *testing.T) {
		// Setup
		logs := []dto.WalletLogEntry{
			{
				OriginalAmount:  100.0,
				ConvertedAmount: 250.0,
				Operation:       "exchange",
			},
			{
				OriginalAmount:  50.0,
				ConvertedAmount: -50.0,
				Operation:       "spend",
			},
		}
		mockService.On("GetWalletLogs", 123).Return(logs, nil).Once()

		// Create request
		req := httptest.NewRequest("GET", "/123/logs", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set auth context values
		ctx := fiber.New().AcquireCtx(req)
		ctx.Locals("user_id", 123)
		ctx.Locals("user_role", "user")
		ctx.Params().Set("user_id", "123")

		// Execute
		resp, err := app.Test(req)

		// Check response
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Parse response body
		var response dto.WalletLogsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)

		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Len(t, response.Data, 2)
	})
}
