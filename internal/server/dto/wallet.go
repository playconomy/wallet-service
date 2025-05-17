package dto

import "time"

// Wallet represents user wallet information
// @Description User wallet information
type Wallet struct {
	ID        int       `json:"id" validate:"required,gt=0" example:"1"`
	UserID    int       `json:"user_id" validate:"required,gt=0" example:"123"`
	Balance   float64   `json:"balance" validate:"gte=0" example:"150.50"`
	CreatedAt time.Time `json:"created_at" example:"2025-05-16T20:00:00Z"`
}

// WalletResponse is the response for wallet endpoints
// @Description Response for wallet information
type WalletResponse struct {
	Success bool    `json:"success" example:"true"`
	Data    *Wallet `json:"data,omitempty"`
	Error   string  `json:"error,omitempty" example:""`
}

// ExchangeRequest represents a token exchange request
// @Description Request for token exchange
type ExchangeRequest struct {
	UserID    int     `json:"user_id" validate:"required,gt=0" example:"123"`
	GameID    string  `json:"game_id" validate:"required,min=1" example:"game-abc"`
	TokenType string  `json:"token_type" validate:"required,min=1" example:"gold"`
	Amount    float64 `json:"amount" validate:"required,gt=0" example:"150"`
	Source    string  `json:"source" validate:"required,oneof=won purchased" example:"won"`
}

// ExchangeResponse is the response for exchange endpoint
// @Description Response for exchange operations
type ExchangeResponse struct {
	Success    bool    `json:"success" example:"true"`
	NewBalance float64 `json:"new_balance,omitempty" example:"165.50"`
	Error      string  `json:"error,omitempty" example:""`
}

// SpendRequest represents a token spend request
// @Description Request for token spending
type SpendRequest struct {
	UserID      int     `json:"user_id" validate:"required,gt=0" example:"123"`
	Amount      float64 `json:"amount" validate:"required,gt=0" example:"50.00"`
	Reason      string  `json:"reason" validate:"required,oneof=market_purchase competition_entry" example:"market_purchase"`
	ReferenceID string  `json:"reference_id" validate:"required,min=1" example:"ORDER-99887"`
}

// SpendResponse is the response for spend endpoint
// @Description Response for spend operations
type SpendResponse struct {
	Success    bool    `json:"success" example:"true"`
	NewBalance float64 `json:"new_balance,omitempty" example:"100.50"`
	Error      string  `json:"error,omitempty" example:""`
}

// WalletLogEntry represents a single wallet transaction log
// @Description Wallet transaction log entry
type WalletLogEntry struct {
	GameID          *string   `json:"game_id" example:"game-abc"`
	TokenType       *string   `json:"token_type" example:"gold"`
	Source          *string   `json:"source" example:"won"`
	OriginalAmount  float64   `json:"original_amount" validate:"gte=0" example:"150"`
	ConvertedAmount float64   `json:"converted_amount" example:"15"`
	Operation       string    `json:"operation" validate:"required,oneof=exchange spend" example:"exchange"`
	ReferenceID     *string   `json:"reference_id" example:"ORDER-99887"`
	CreatedAt       time.Time `json:"created_at" example:"2025-05-16T20:00:00Z"`
}

// WalletLogsResponse is the response for logs endpoint
// @Description Response for wallet logs
type WalletLogsResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    []WalletLogEntry `json:"data,omitempty"`
	Error   string           `json:"error,omitempty" example:""`
}

// GenericResponse is a general purpose response
// @Description Generic API response
type GenericResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error,omitempty" example:"Authentication required"`
}
