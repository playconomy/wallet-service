// Package observability provides tools for logging, metrics, and tracing
package observability

import "go.uber.org/zap"

// Metrics is a simple no-op implementation for testing
type Metrics struct{}

// RecordWalletOperation records a wallet operation (no-op for tests)
func (m *Metrics) RecordWalletOperation(operation, result string) {
	// No-op implementation for testing
}

// Tracer is a simple no-op implementation for testing
type Tracer struct{}

// StartSpan creates a new span (no-op for tests)
func (t *Tracer) StartSpan(ctx interface{}, name string, options ...interface{}) (interface{}, interface{}) {
	// Return the context and a no-op span
	return ctx, noopSpan{}
}

// noopSpan is a no-op span for testing
type noopSpan struct{}

// End ends the span (no-op)
func (s noopSpan) End() {}

// NewNoopLogger creates a no-op logger for testing
func NewNoopLogger() (*Logger, error) {
	noopLogger, _ := zap.NewDevelopment()
	return &Logger{Logger: noopLogger}, nil
}
