// Package repository provides data access implementations
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/playconomy/wallet-service/internal/model"
	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/observability/metrics"
	"github.com/playconomy/wallet-service/internal/observability/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// PostgresRepository implements WalletRepository interface for PostgreSQL
type PostgresRepository struct {
	db      *sql.DB
	logger  *zap.Logger
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
}

// PostgresTransaction represents a PostgreSQL transaction
type PostgresTransaction struct {
	tx *sql.Tx
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB, obs *observability.Observability) *PostgresRepository {
	return &PostgresRepository{
		db:      db,
		logger:  obs.Logger.Logger,
		metrics: obs.Metrics,
		tracer:  obs.Tracer,
	}
}

// BeginTx starts a new transaction
func (r *PostgresRepository) BeginTx(ctx context.Context) (Transaction, error) {
	ctx, span := r.tracer.StartSpan(ctx, "Repository.BeginTx")
	defer span.End()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	return &PostgresTransaction{tx: tx}, nil
}

// Commit commits the transaction
func (t *PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// GetWalletByUserID retrieves a wallet by user ID
func (r *PostgresRepository) GetWalletByUserID(ctx context.Context, userID int) (*model.Wallet, error) {
	ctx, span := r.tracer.StartSpan(ctx, "Repository.GetWalletByUserID",
		trace.WithAttributes(attribute.Int("user_id", userID)))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Getting wallet for user", zap.Int("user_id", userID))

	var wallet model.Wallet
	err := r.db.QueryRowContext(ctx, QueryGetWalletByUserID, userID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Debug("Wallet not found for user", zap.Int("user_id", userID))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Error retrieving wallet",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("get wallet: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("select", "wallets", duration)
	r.metrics.SetWalletBalance(fmt.Sprintf("%d", userID), "platform", wallet.Balance)

	return &wallet, nil
}

// GetWalletByUserIDForUpdate retrieves a wallet by user ID with a lock for update
func (r *PostgresRepository) GetWalletByUserIDForUpdate(
	ctx context.Context, userID int, tx Transaction) (*model.Wallet, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.GetWalletByUserIDForUpdate",
		trace.WithAttributes(attribute.Int("user_id", userID)))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Getting wallet for update", zap.Int("user_id", userID))

	pTx, ok := tx.(*PostgresTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type")
	}

	var wallet model.Wallet
	err := pTx.tx.QueryRowContext(ctx, QueryGetWalletByUserIDForUpdate, userID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Debug("Wallet not found for update", zap.Int("user_id", userID))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Error retrieving wallet for update",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("get wallet for update: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("select", "wallets", duration)

	return &wallet, nil
}

// CreateWallet creates a new wallet
func (r *PostgresRepository) CreateWallet(
	ctx context.Context, userID int, initialBalance float64, tx Transaction) (*model.Wallet, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.CreateWallet",
		trace.WithAttributes(
			attribute.Int("user_id", userID),
			attribute.Float64("initial_balance", initialBalance),
		))
	defer span.End()

	startTime := time.Now()
	r.logger.Info("Creating new wallet",
		zap.Int("user_id", userID),
		zap.Float64("initial_balance", initialBalance))

	pTx, ok := tx.(*PostgresTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type")
	}

	var wallet model.Wallet
	err := pTx.tx.QueryRowContext(ctx, QueryCreateWallet, userID, initialBalance).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err != nil {
		r.logger.Error("Failed to create wallet",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("create wallet: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("insert", "wallets", duration)
	r.metrics.SetWalletBalance(fmt.Sprintf("%d", userID), "platform", wallet.Balance)

	return &wallet, nil
}

// UpdateWalletBalance updates a wallet's balance
func (r *PostgresRepository) UpdateWalletBalance(
	ctx context.Context, userID int, newBalance float64, tx Transaction) (*model.Wallet, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.UpdateWalletBalance",
		trace.WithAttributes(
			attribute.Int("user_id", userID),
			attribute.Float64("new_balance", newBalance),
		))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Updating wallet balance",
		zap.Int("user_id", userID),
		zap.Float64("new_balance", newBalance))

	pTx, ok := tx.(*PostgresTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type")
	}

	var wallet model.Wallet
	err := pTx.tx.QueryRowContext(ctx, QueryUpdateWalletBalance, userID, newBalance).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Warn("Wallet not found for update",
			zap.Int("user_id", userID))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Failed to update wallet balance",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("update", "wallets", duration)
	r.metrics.SetWalletBalance(fmt.Sprintf("%d", userID), "platform", wallet.Balance)

	return &wallet, nil
}

// SpendFromWallet spends tokens from a wallet
func (r *PostgresRepository) SpendFromWallet(
	ctx context.Context, userID int, amount float64, tx Transaction) (*model.Wallet, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.SpendFromWallet", 
		trace.WithAttributes(
			attribute.Int("user_id", userID),
			attribute.Float64("amount", amount),
		))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Spending from wallet",
		zap.Int("user_id", userID),
		zap.Float64("amount", amount))

	pTx, ok := tx.(*PostgresTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type")
	}

	var wallet model.Wallet
	err := pTx.tx.QueryRowContext(ctx, QuerySpendFromWallet, userID, amount).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Warn("Insufficient funds or wallet not found",
			zap.Int("user_id", userID),
			zap.Float64("amount", amount))
		return nil, fmt.Errorf("insufficient funds")
	}

	if err != nil {
		r.logger.Error("Failed to spend from wallet",
			zap.Int("user_id", userID),
			zap.Float64("amount", amount),
			zap.Error(err))
		return nil, fmt.Errorf("spend from wallet: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("update", "wallets", duration)
	r.metrics.SetWalletBalance(fmt.Sprintf("%d", userID), "platform", wallet.Balance)

	return &wallet, nil
}

// GetExchangeRate retrieves an exchange rate
func (r *PostgresRepository) GetExchangeRate(ctx context.Context, gameID, tokenType string) (*model.ExchangeRate, error) {
	ctx, span := r.tracer.StartSpan(ctx, "Repository.GetExchangeRate",
		trace.WithAttributes(
			attribute.String("game_id", gameID),
			attribute.String("token_type", tokenType),
		))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Getting exchange rate",
		zap.String("game_id", gameID),
		zap.String("token_type", tokenType))

	var rate model.ExchangeRate
	err := r.db.QueryRowContext(ctx, QueryGetExchangeRate, gameID, tokenType).Scan(
		&rate.ID, &rate.GameID, &rate.TokenType, &rate.ToPlatformRatio, &rate.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Warn("Exchange rate not found",
			zap.String("game_id", gameID),
			zap.String("token_type", tokenType))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Failed to get exchange rate",
			zap.String("game_id", gameID),
			zap.String("token_type", tokenType),
			zap.Error(err))
		return nil, fmt.Errorf("get exchange rate: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("select", "exchange_rates", duration)

	return &rate, nil
}

// GetExchangeRateByID retrieves an exchange rate by ID
func (r *PostgresRepository) GetExchangeRateByID(ctx context.Context, id int64) (*model.ExchangeRate, error) {
	ctx, span := r.tracer.StartSpan(ctx, "Repository.GetExchangeRateByID",
		trace.WithAttributes(attribute.Int64("id", id)))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Getting exchange rate by ID", zap.Int64("id", id))

	var rate model.ExchangeRate
	err := r.db.QueryRowContext(ctx, QueryGetExchangeRateByID, id).Scan(
		&rate.ID, &rate.GameID, &rate.TokenType, &rate.ToPlatformRatio, &rate.CreatedAt)

	if err == sql.ErrNoRows {
		r.logger.Warn("Exchange rate not found", zap.Int64("id", id))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Failed to get exchange rate by ID",
			zap.Int64("id", id),
			zap.Error(err))
		return nil, fmt.Errorf("get exchange rate by ID: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("select", "exchange_rates", duration)

	return &rate, nil
}

// CreateWalletLog creates a wallet transaction log
func (r *PostgresRepository) CreateWalletLog(
	ctx context.Context, log *model.WalletLog, tx Transaction) (*model.WalletLog, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.CreateWalletLog")
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Creating wallet log entry",
		zap.Int64("wallet_id", log.WalletID),
		zap.Int("user_id", log.UserID),
		zap.Float64("amount", log.Amount),
		zap.Float64("platform_amount", log.PlatformAmount),
		zap.String("source", log.Source))

	pTx, ok := tx.(*PostgresTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type")
	}

	var newLog model.WalletLog
	err := pTx.tx.QueryRowContext(ctx, QueryCreateWalletLog,
		log.WalletID, log.UserID, log.GameID, log.TokenType,
		log.Amount, log.PlatformAmount, log.Source, log.ReferenceID).Scan(
		&newLog.ID, &newLog.WalletID, &newLog.UserID, &newLog.GameID, &newLog.TokenType,
		&newLog.Amount, &newLog.PlatformAmount, &newLog.Source, &newLog.ReferenceID, &newLog.CreatedAt)

	if err != nil {
		r.logger.Error("Failed to create wallet log",
			zap.Int("user_id", log.UserID),
			zap.Error(err))
		return nil, fmt.Errorf("create wallet log: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("insert", "wallet_logs", duration)

	return &newLog, nil
}

// GetWalletLogs retrieves wallet logs for a user
func (r *PostgresRepository) GetWalletLogs(
	ctx context.Context, userID int, limit, offset int) ([]*model.WalletLog, error) {
	
	ctx, span := r.tracer.StartSpan(ctx, "Repository.GetWalletLogs",
		trace.WithAttributes(
			attribute.Int("user_id", userID),
			attribute.Int("limit", limit),
			attribute.Int("offset", offset),
		))
	defer span.End()

	startTime := time.Now()
	r.logger.Debug("Getting wallet logs",
		zap.Int("user_id", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 50 // Default limit
	}

	rows, err := r.db.QueryContext(ctx, QueryGetWalletLogs, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get wallet logs",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("get wallet logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.WalletLog
	for rows.Next() {
		var log model.WalletLog
		if err := rows.Scan(
			&log.ID, &log.WalletID, &log.UserID, &log.GameID, &log.TokenType,
			&log.Amount, &log.PlatformAmount, &log.Source, &log.ReferenceID, &log.CreatedAt); err != nil {
			r.logger.Error("Error scanning wallet log row",
				zap.Int("user_id", userID),
				zap.Error(err))
			return nil, fmt.Errorf("scan wallet log: %w", err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating wallet logs",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("iterate wallet logs: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	r.metrics.ObserveDBQueryDuration("select", "wallet_logs", duration)

	return logs, nil
}
