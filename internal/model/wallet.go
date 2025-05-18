// Package model contains domain models for the wallet service
package model

import (
	"time"
)

// Wallet represents a user's wallet with platform tokens
type Wallet struct {
	ID        int64
	UserID    int
	Balance   float64
	CreatedAt time.Time
}

// ExchangeRate represents the exchange rate for a game token
type ExchangeRate struct {
	ID              int64
	GameID          string
	TokenType       string
	ToPlatformRatio float64
	CreatedAt       time.Time
}

// WalletLog represents a log of wallet transactions
type WalletLog struct {
	ID             int64
	WalletID       int64
	UserID         int
	GameID         *string
	TokenType      *string
	Amount         float64
	PlatformAmount float64
	Source         string
	ReferenceID    *string
	CreatedAt      time.Time
}

// Transaction types
const (
	TransactionExchange = "exchange"
	TransactionSpend    = "spend"
	TransactionBonus    = "bonus"
)

// Operation status
const (
	StatusSuccess           = "success"
	StatusFailure           = "failure"
	StatusInsufficientFunds = "insufficient_funds"
	StatusRateNotFound      = "rate_not_found"
	StatusValidationFailed  = "validation_failed"
)
