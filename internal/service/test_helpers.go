package service

import (
	"github.com/playconomy/wallet-service/internal/observability"
)

// GetTestObservability creates a test observability instance
// This is used for integration tests to avoid the need for actual observability services
func GetTestObservability() *observability.Observability {
	return observability.NewTestObservability()
}
