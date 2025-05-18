// Package observability combines logging, metrics, and tracing components
package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/playconomy/wallet-service/internal/config"
	"github.com/playconomy/wallet-service/internal/observability/logger"
	"github.com/playconomy/wallet-service/internal/observability/metrics"
	"github.com/playconomy/wallet-service/internal/observability/tracing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides observability dependencies
var Module = fx.Options(
	fx.Provide(
		NewObservability,
	),
)

// Observability combines all observability components
type Observability struct {
	Logger  *logger.Logger
	Metrics *metrics.Metrics
	Tracer  *tracing.Tracer
}

// NewObservability creates and initializes all observability components
func NewObservability(lc fx.Lifecycle, cfg *config.Config) (*Observability, error) {
	// Initialize structured logger
	log, err := logger.NewLogger(cfg.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize metrics collector
	metricsCollector := metrics.NewMetrics()

	// Initialize tracer
	var tracer *tracing.Tracer
	if cfg.Observability.Tracing.Enabled {
		tracer, err = tracing.NewTracer(
			cfg.App.Name,
			cfg.App.Version,
			cfg.Observability.Tracing.Endpoint,
			cfg.Observability.Tracing.SamplingRatio,
		)
		if err != nil {
			log.Error("Failed to initialize tracer", zap.Error(err))
		}
	}

	// Create observability object
	obs := &Observability{
		Logger:  log,
		Metrics: metricsCollector,
		Tracer:  tracer,
	}

	// Setup lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting observability components",
				zap.String("app", cfg.App.Name),
				zap.String("version", cfg.App.Version),
				zap.String("environment", cfg.App.Env),
				zap.String("log_level", cfg.App.LogLevel),
				zap.Bool("tracing_enabled", cfg.Observability.Tracing.Enabled),
				zap.Bool("metrics_enabled", cfg.Observability.Metrics.Enabled),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Shutting down observability components")
			
			var shutdownErrors []error
			
			// Create a context with timeout for graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			// Properly shutdown tracer
			if tracer != nil {
				if err := tracer.Shutdown(shutdownCtx); err != nil {
					shutdownErrors = append(shutdownErrors, fmt.Errorf("tracer shutdown error: %w", err))
					log.Error("Failed to shutdown tracer", zap.Error(err))
				} else {
					log.Info("Tracer shutdown successfully")
				}
			}
			
			// Properly shutdown metrics
			if metricsCollector != nil {
				if err := metricsCollector.Shutdown(shutdownCtx); err != nil {
					shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics shutdown error: %w", err))
					log.Error("Failed to shutdown metrics", zap.Error(err))
				} else {
					log.Info("Metrics shutdown successfully")
				}
			}
			
			// Properly shutdown logger (should be last)
			if err := log.Shutdown(); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("logger shutdown error: %w", err))
				// Can't log this error since the logger is being shut down
			}
			
			// Return combined errors if any
			if len(shutdownErrors) > 0 {
				return fmt.Errorf("errors during observability shutdown: %v", shutdownErrors)
			}
			
			return nil
		},
	})

	return obs, nil
}

// SetupMetricsEndpoint sets up a metrics endpoint for Prometheus
func SetupMetricsEndpoint(app *fiber.App, obs *Observability) {
	if obs.Metrics != nil {
		metrics.SetupMetricsEndpoint(app, obs.Metrics)
		obs.Logger.Info("Metrics endpoint set up at /metrics")
	}
}
