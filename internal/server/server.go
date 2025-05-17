package server

import (
	"context"
	"fmt"
	"log"

	"github.com/playconomy/wallet-service/internal/config"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Server struct {
	app    *fiber.App
	config *config.Config
}

func NewServer(lc fx.Lifecycle, app *fiber.App, cfg *config.Config) *Server {
	server := &Server{
		app:    app,
		config: cfg,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
				log.Printf("Starting server on %s", addr)
				if err := server.app.Listen(addr); err != nil {
					log.Printf("Error starting server: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Shutting down server...")
			return server.app.Shutdown()
		},
	})

	return server
}
