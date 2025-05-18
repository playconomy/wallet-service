package server

import (
	"context"
	"fmt"

	"github.com/playconomy/wallet-service/internal/config"
	"github.com/playconomy/wallet-service/internal/observability"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Server struct {
	app    *fiber.App
	config *config.Config
	logger *zap.Logger
}

func NewServer(lc fx.Lifecycle, app *fiber.App, cfg *config.Config, obs *observability.Observability) *Server {
	server := &Server{
		app:    app,
		config: cfg,
		logger: obs.Logger.With(zap.String("component", "server")),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
				server.logger.Info("Starting server", 
					zap.String("address", addr),
					zap.String("environment", cfg.App.Env),
				)
				if err := server.app.Listen(addr); err != nil {
					server.logger.Error("Server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.logger.Info("Shutting down server...")
			return server.app.Shutdown()
		},
	})

	return server
}
