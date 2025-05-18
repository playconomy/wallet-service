package router

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/server/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockWalletHandler is a mock implementation of WalletHandlerInterface for testing
type MockWalletHandler struct {
	mock.Mock
}

func (m *MockWalletHandler) GetWallet(c *fiber.Ctx) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockWalletHandler) Exchange(c *fiber.Ctx) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockWalletHandler) Spend(c *fiber.Ctx) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockWalletHandler) GetWalletLogs(c *fiber.Ctx) error {
	args := m.Called(c)
	return args.Error(0)
}

// Compile-time verification that MockWalletHandler implements WalletHandlerInterface
var _ handler.WalletHandlerInterface = (*MockWalletHandler)(nil)

// Setup test router
func setupTestRouter(t *testing.T) (*fiber.App, *MockWalletHandler, RouterInterface) {
	app := fiber.New()
	mockHandler := new(MockWalletHandler)
	router := NewRouter(app, mockHandler)
	
	return app, mockHandler, router
}

func TestSetup(t *testing.T) {
	app, _, router := setupTestRouter(t)
	
	// Test setup method
	router.Setup(app)
	
	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	
	// Read and validate response
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "ok")
}

func TestRegisterRoutes(t *testing.T) {
	app, mockHandler, router := setupTestRouter(t)
	
	// Call the method to test
	router.RegisterRoutes(app)
	
	// Verify that routes are registered correctly by checking the app's routes
	routes := app.GetRoutes()
	
	// Check that we have the expected routes
	assert.NotEmpty(t, routes)
	
	// Check for health route
	hasHealthRoute := false
	for _, route := range routes {
		if route.Path == "/health" && route.Method == "GET" {
			hasHealthRoute = true
			break
		}
	}
	assert.True(t, hasHealthRoute, "Health route should be registered")
	
	// Check for wallet routes (these are pattern matched, so just check one)
	hasWalletRoute := false
	for _, route := range routes {
		if route.Path == "/:user_id" && route.Method == "GET" {
			hasWalletRoute = true
			break
		}
	}
	assert.True(t, hasWalletRoute, "Wallet routes should be registered")
}
