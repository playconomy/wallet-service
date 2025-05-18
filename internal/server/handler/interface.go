// Package handler provides HTTP request handlers for wallet operations
package handler

import (
	"github.com/gofiber/fiber/v2"
)

// WalletHandlerInterface defines the interface for wallet HTTP handlers
type WalletHandlerInterface interface {
	// GetWallet retrieves wallet information for a user
	GetWallet(c *fiber.Ctx) error
	
	// Exchange converts game tokens to platform tokens
	Exchange(c *fiber.Ctx) error
	
	// Spend deducts tokens from user's wallet
	Spend(c *fiber.Ctx) error
	
	// GetWalletLogs retrieves transaction logs for a user's wallet
	GetWalletLogs(c *fiber.Ctx) error
}
