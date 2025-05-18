// Package observability provides tools for logging, metrics, and tracing
package observability

import (
	"github.com/playconomy/wallet-service/internal/observability/logger"
	"github.com/playconomy/wallet-service/internal/observability/metrics"
	"github.com/playconomy/wallet-service/internal/observability/tracing"
	"go.uber.org/zap"
)

// NewTestObservability creates a minimal Observability instance for testing
func NewTestObservability() *Observability {
	// Create a no-op logger
	zapLogger, _ := zap.NewDevelopment()
	testLogger := &logger.Logger{Logger: zapLogger}
	
	// Create no-op metrics and tracer
	testMetrics := &metrics.Metrics{}
	testTracer := &tracing.Tracer{}
	
	return &Observability{
		Logger:  testLogger,
		Metrics: testMetrics,
		Tracer:  testTracer,
	}
}
