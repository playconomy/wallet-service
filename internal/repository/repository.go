// Package repository provides data access layer for the wallet service
package repository

import (
	"context"

	"github.com/playconomy/wallet-service/internal/model"
)

// WalletRepository defines the interface for wallet data access
type WalletRepository interface {
	// Wallet operations
	GetWalletByUserID(ctx context.Context, userID int) (*model.Wallet, error)
	GetWalletByUserIDForUpdate(ctx context.Context, userID int, tx Transaction) (*model.Wallet, error)
	CreateWallet(ctx context.Context, userID int, initialBalance float64, tx Transaction) (*model.Wallet, error)
	UpdateWalletBalance(ctx context.Context, userID int, newBalance float64, tx Transaction) (*model.Wallet, error)
	SpendFromWallet(ctx context.Context, userID int, amount float64, tx Transaction) (*model.Wallet, error)

	// Exchange rate operations
	GetExchangeRate(ctx context.Context, gameID, tokenType string) (*model.ExchangeRate, error)
	GetExchangeRateByID(ctx context.Context, id int64) (*model.ExchangeRate, error)

	// Log operations
	CreateWalletLog(ctx context.Context, log *model.WalletLog, tx Transaction) (*model.WalletLog, error)
	GetWalletLogs(ctx context.Context, userID int, limit, offset int) ([]*model.WalletLog, error)

	// Transaction management
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
}
