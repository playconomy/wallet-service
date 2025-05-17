package service

import (
	"database/sql"
	"fmt"

	"github.com/playconomy/wallet-service/internal/server/dto"
	
	"go.uber.org/fx"
)

// Module provides service dependencies
var Module = fx.Options(
	fx.Provide(NewWalletService),
)

type WalletService struct {
	db *sql.DB
}

// Constructors for fx dependency injection
func NewWalletService(db *sql.DB) *WalletService {
	return &WalletService{db: db}
}

func (s *WalletService) GetWalletByUserID(userID int) (*dto.Wallet, error) {
	wallet := &dto.Wallet{}

	err := s.db.QueryRow(
		"SELECT id, user_id, balance, created_at FROM wallets WHERE user_id = $1",
		userID,
	).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *WalletService) Exchange(req *dto.ExchangeRequest) (float64, error) {
	// Get exchange rate
	var ratio float64
	err := s.db.QueryRow(
		"SELECT to_platform_ratio FROM exchange_rates WHERE game_id = $1 AND token_type = $2",
		req.GameID, req.TokenType,
	).Scan(&ratio)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("exchange rate not found for game_id=%s and token_type=%s", req.GameID, req.TokenType)
	}
	if err != nil {
		return 0, err
	}

	// Calculate platform amount
	platformAmount := req.Amount * ratio

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Get or create wallet
	var walletID int
	var currentBalance float64
	err = tx.QueryRow(
		"INSERT INTO wallets (user_id, balance) VALUES ($1, 0) "+
			"ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id "+
			"RETURNING id, balance",
		req.UserID,
	).Scan(&walletID, &currentBalance)
	if err != nil {
		return 0, err
	}

	// Update wallet balance
	newBalance := currentBalance + platformAmount
	_, err = tx.Exec(
		"UPDATE wallets SET balance = $1 WHERE id = $2",
		newBalance, walletID,
	)
	if err != nil {
		return 0, err
	}

	// Log transaction
	_, err = tx.Exec(
		"INSERT INTO wallet_logs (wallet_id, user_id, game_id, token_type, amount, platform_amount, source) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7)",
		walletID, req.UserID, req.GameID, req.TokenType, req.Amount, platformAmount, req.Source,
	)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (s *WalletService) Spend(req *dto.SpendRequest) (float64, error) {
	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Get wallet with lock
	var walletID int
	var currentBalance float64
	err = tx.QueryRow(
		"SELECT id, balance FROM wallets WHERE user_id = $1 FOR UPDATE",
		req.UserID,
	).Scan(&walletID, &currentBalance)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("wallet not found for user_id=%d", req.UserID)
	}
	if err != nil {
		return 0, err
	}

	// Check if balance is sufficient
	if currentBalance < req.Amount {
		return 0, fmt.Errorf("insufficient funds: current balance %.2f, required %.2f", currentBalance, req.Amount)
	}

	// Update wallet balance
	newBalance := currentBalance - req.Amount
	_, err = tx.Exec(
		"UPDATE wallets SET balance = $1 WHERE id = $2",
		newBalance, walletID,
	)
	if err != nil {
		return 0, err
	}

	// Log transaction
	_, err = tx.Exec(
		"INSERT INTO wallet_logs (wallet_id, user_id, amount, platform_amount, source, reference_id) "+
			"VALUES ($1, $2, $3, $4, $5, $6)",
		walletID, req.UserID, -req.Amount, -req.Amount, req.Reason, req.ReferenceID,
	)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (s *WalletService) GetWalletLogs(userID int) ([]dto.WalletLogEntry, error) {
	rows, err := s.db.Query(`
		SELECT 
			NULLIF(game_id, '') as game_id,
			NULLIF(token_type, '') as token_type,
			NULLIF(source, '') as source,
			ABS(amount) as original_amount,
			platform_amount as converted_amount,
			CASE 
				WHEN game_id IS NOT NULL THEN 'exchange'
				ELSE 'spend'
			END as operation,
			NULLIF(reference_id, '') as reference_id,
			created_at
		FROM wallet_logs
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []dto.WalletLogEntry
	for rows.Next() {
		var log dto.WalletLogEntry
		err := rows.Scan(
			&log.GameID,
			&log.TokenType,
			&log.Source,
			&log.OriginalAmount,
			&log.ConvertedAmount,
			&log.Operation,
			&log.ReferenceID,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
