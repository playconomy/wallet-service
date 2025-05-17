package module

import (
	"github.com/playconomy/wallet-service/database"
	_ "github.com/playconomy/wallet-service/docs" // Import for swagger
	"github.com/playconomy/wallet-service/internal/config"
	"github.com/playconomy/wallet-service/internal/server"
	"github.com/playconomy/wallet-service/internal/server/handler"
	"github.com/playconomy/wallet-service/internal/server/router"
	"github.com/playconomy/wallet-service/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// Module combines all application modules
var Module = fx.Options(
	fx.Provide(
		// Config
		config.LoadConfig,

		// Server
		func() *fiber.App {
			return fiber.New(fiber.Config{
				ErrorHandler: func(ctx *fiber.Ctx, err error) error {
					code := fiber.StatusInternalServerError
					if e, ok := err.(*fiber.Error); ok {
						code = e.Code
					}
					return ctx.Status(code).JSON(fiber.Map{
						"success": false,
						"error":   err.Error(),
					})
				},
			})
		},
		server.NewServer,

		// Database
		database.NewConnection,

		// Services
		service.NewWalletService,

		// Handlers
		handler.NewWalletHandler,

		// Router
		router.NewRouter,
	),

	// Invocations
	fx.Invoke(
		router.SetupRoutes,
	),
)
