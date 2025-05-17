package router

import (
	"github.com/playconomy/wallet-service/internal/server/handler"
	"github.com/playconomy/wallet-service/internal/server/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.uber.org/fx"
)

// Module provides router dependencies
var Module = fx.Options(
	fx.Provide(NewRouter),
	fx.Invoke(SetupRoutes),
)

type Router struct {
	app           *fiber.App
	walletHandler *handler.WalletHandler
}

func NewRouter(app *fiber.App, walletHandler *handler.WalletHandler) *Router {
	return &Router{
		app:           app,
		walletHandler: walletHandler,
	}
}

func SetupRoutes(router *Router) {
	// Swagger routes
	router.app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check (unprotected)
	//	@Summary		Health check
	//	@Description	Check if the service is running
	//	@Tags			health
	//	@Produce		json
	//	@Success		200	{object}	map[string]string	"Health status"
	//	@Router			/health [get]
	router.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Create a group with auth middleware
	api := router.app.Group("/", middleware.AuthMiddleware())

	// Protected routes
	api.Get("/:user_id", router.walletHandler.GetWallet)
	api.Get("/:user_id/logs", router.walletHandler.GetWalletLogs)
	api.Post("/exchange", router.walletHandler.Exchange)
	api.Post("/spend", router.walletHandler.Spend)
}
