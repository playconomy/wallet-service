package module

import (
	"github.com/playconomy/wallet-service/database"
	_ "github.com/playconomy/wallet-service/docs" // Import for swagger
	"github.com/playconomy/wallet-service/internal/config"
	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/observability/middleware"
	"github.com/playconomy/wallet-service/internal/server"
	"github.com/playconomy/wallet-service/internal/server/handler"
	"github.com/playconomy/wallet-service/internal/server/router"
	"github.com/playconomy/wallet-service/internal/service"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module combines all application modules
var Module = fx.Options(
	// Include the observability module
	observability.Module,

	fx.Provide(
		// Config
		config.LoadConfig,

		// Server
		func(obs *observability.Observability) *fiber.App {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(ctx *fiber.Ctx, err error) error {
					code := fiber.StatusInternalServerError
					if e, ok := err.(*fiber.Error); ok {
						code = e.Code
					}
					
					// Log the error
					obs.Logger.With(
						zap.Error(err),
						zap.Int("status", code),
						zap.String("path", ctx.Path()),
						zap.String("method", ctx.Method()),
					).Error("HTTP request error")
					
					return ctx.Status(code).JSON(fiber.Map{
						"success": false,
						"error":   err.Error(),
					})
				},
			})

			// Add middlewares
			app.Use(middleware.RequestIDMiddleware())
			app.Use(middleware.LoggingMiddleware())
			
			// Add metrics middleware if enabled
			if obs.Metrics != nil {
				app.Use(middleware.MetricsMiddleware(obs.Metrics))
			}
			
			// Add tracing middleware if enabled
			if obs.Tracer != nil {
				app.Use(otelfiber.Middleware())
			}

			return app
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
		observability.SetupMetricsEndpoint,
	),
)
