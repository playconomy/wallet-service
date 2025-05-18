// Package service provides business logic for wallet operations
package service

import (
	"context"
	
	"github.com/playconomy/wallet-service/internal/server/dto"
)

// WalletServiceInterface defines the interface for wallet service operations
type WalletServiceInterface interface {
	// GetWalletByUserID retrieves wallet information for a specific user
	GetWalletByUserID(ctx context.Context, userID int) (*dto.Wallet, error)
	
	// Exchange converts game tokens to platform tokens
	Exchange(ctx context.Context, req *dto.ExchangeRequest) (float64, error)
	
	// Spend deducts tokens from user's wallet
	Spend(ctx context.Context, req *dto.SpendRequest) (float64, error)
	
	// GetWalletLogs retrieves transaction logs for a user's wallet
	GetWalletLogs(ctx context.Context, userID int) ([]dto.WalletLogEntry, error)
}
