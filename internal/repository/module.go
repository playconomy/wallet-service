// Package repository provides module configuration for data access layer
package repository

import (
	"database/sql"

	"github.com/playconomy/wallet-service/internal/observability"

	"go.uber.org/fx"
)

// Module provides repository dependencies for the application
var Module = fx.Options(
	fx.Provide(NewWalletRepository),
)

// NewWalletRepository creates a new wallet repository implementation
func NewWalletRepository(db *sql.DB, obs *observability.Observability) WalletRepository {
	return NewPostgresRepository(db, obs)
}
