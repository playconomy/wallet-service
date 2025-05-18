package service

import (
	"context"
	"fmt"

	"github.com/playconomy/wallet-service/internal/model"
	"github.com/playconomy/wallet-service/internal/observability"
	"github.com/playconomy/wallet-service/internal/observability/metrics"
	"github.com/playconomy/wallet-service/internal/observability/tracing"
	"github.com/playconomy/wallet-service/internal/repository"
	"github.com/playconomy/wallet-service/internal/server/dto"
	
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides service dependencies
var Module = fx.Options(
	fx.Provide(NewWalletService),
	// Provide interface implementation for dependency injection
	fx.Provide(func(s *WalletService) WalletServiceInterface { return s }),
)

type WalletService struct {
	repo    repository.WalletRepository
	logger  *zap.Logger
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
}

// Compile-time verification that WalletService implements WalletServiceInterface
var _ WalletServiceInterface = (*WalletService)(nil)

// Constructors for fx dependency injection
func NewWalletService(repo repository.WalletRepository, obs *observability.Observability) *WalletService {
	return &WalletService{
		repo:    repo,
		logger:  obs.Logger.Logger,
		metrics: obs.Metrics,
		tracer:  obs.Tracer,
	}
}

func (s *WalletService) GetWalletByUserID(ctx context.Context, userID int) (*dto.Wallet, error) {
	ctx, span := s.tracer.StartSpan(ctx, "WalletService.GetWalletByUserID",
		trace.WithAttributes(attribute.Int("user_id", userID)))
	defer span.End()

	s.logger.Info("Getting wallet for user", zap.Int("user_id", userID))
	
	// Get wallet from repository
	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	
	if err != nil {
		s.logger.Error("Error retrieving wallet", 
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	if wallet == nil {
		s.logger.Info("Wallet not found for user", zap.Int("user_id", userID))
		return nil, nil
	}

	s.logger.Debug("Retrieved wallet successfully", 
		zap.Int("user_id", userID),
		zap.Float64("balance", wallet.Balance))

	// Convert model to DTO
	return &dto.Wallet{
		ID:        int(wallet.ID),
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		CreatedAt: wallet.CreatedAt,
	}, nil
}

func (s *WalletService) Exchange(ctx context.Context, req *dto.ExchangeRequest) (float64, error) {
	ctx, span := s.tracer.StartSpan(ctx, "WalletService.Exchange", 
		trace.WithAttributes(
			attribute.String("game_id", req.GameID),
			attribute.String("token_type", req.TokenType),
			attribute.Float64("amount", req.Amount),
			attribute.Int("user_id", req.UserID),
		))
	defer span.End()

	s.logger.Info("Processing exchange request", 
		zap.String("game_id", req.GameID),
		zap.String("token_type", req.TokenType),
		zap.Float64("amount", req.Amount),
		zap.Int("user_id", req.UserID))

	// Get exchange rate
	exchangeRate, err := s.repo.GetExchangeRate(ctx, req.GameID, req.TokenType)
	if err != nil {
		s.logger.Error("Error retrieving exchange rate", 
			zap.String("game_id", req.GameID),
			zap.String("token_type", req.TokenType),
			zap.Error(err))
		s.metrics.RecordWalletOperation("exchange", "error_db")
		return 0, err
	}

	if exchangeRate == nil {
		s.logger.Error("Exchange rate not found", 
			zap.String("game_id", req.GameID),
			zap.String("token_type", req.TokenType))
		s.metrics.RecordWalletOperation("exchange", "error_rate_not_found")
		return 0, fmt.Errorf("exchange rate not found for game_id=%s and token_type=%s", req.GameID, req.TokenType)
	}

	// Calculate platform amount
	platformAmount := req.Amount * exchangeRate.ToPlatformRatio
	s.logger.Debug("Calculated platform amount", 
		zap.Float64("game_amount", req.Amount),
		zap.Float64("exchange_rate", exchangeRate.ToPlatformRatio),
		zap.Float64("platform_amount", platformAmount))

	// Start a transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		s.metrics.RecordWalletOperation("exchange", "error_transaction")
		return 0, err
	}
	defer tx.Rollback()

	// Try to get the wallet
	wallet, err := s.repo.GetWalletByUserIDForUpdate(ctx, req.UserID, tx)
	if err != nil {
		s.logger.Error("Error getting wallet for update", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("exchange", "error_wallet")
		return 0, err
	}

	var newWallet *model.Wallet
	
	// If wallet doesn't exist, create a new one
	if wallet == nil {
		s.logger.Info("Creating new wallet for user", 
			zap.Int("user_id", req.UserID),
			zap.Float64("initial_balance", platformAmount))
			
		newWallet, err = s.repo.CreateWallet(ctx, req.UserID, platformAmount, tx)
		if err != nil {
			s.logger.Error("Failed to create wallet", 
				zap.Int("user_id", req.UserID),
				zap.Error(err))
			s.metrics.RecordWalletOperation("exchange", "error_create_wallet")
			return 0, err
		}
	} else {
		// Update existing wallet
		newBalance := wallet.Balance + platformAmount
		s.logger.Debug("Updating wallet balance", 
			zap.Int("user_id", req.UserID),
			zap.Float64("old_balance", wallet.Balance),
			zap.Float64("platform_amount", platformAmount),
			zap.Float64("new_balance", newBalance))
			
		newWallet, err = s.repo.UpdateWalletBalance(ctx, req.UserID, newBalance, tx)
		if err != nil {
			s.logger.Error("Failed to update wallet balance", 
				zap.Int("user_id", req.UserID),
				zap.Error(err))
			s.metrics.RecordWalletOperation("exchange", "error_update_wallet")
			return 0, err
		}
	}

	// Create wallet log
	walletLog := &model.WalletLog{
		WalletID:       newWallet.ID,
		UserID:         req.UserID,
		GameID:         &req.GameID,
		TokenType:      &req.TokenType,
		Amount:         req.Amount,
		PlatformAmount: platformAmount,
		Source:         model.TransactionExchange,
	}
	
	_, err = s.repo.CreateWalletLog(ctx, walletLog, tx)
	if err != nil {
		s.logger.Error("Failed to create wallet log", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("exchange", "error_log")
		return 0, err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("exchange", "error_commit")
		return 0, err
	}

	s.logger.Info("Exchange completed successfully", 
		zap.Int("user_id", req.UserID),
		zap.Float64("game_amount", req.Amount),
		zap.Float64("platform_amount", platformAmount),
		zap.Float64("new_balance", newWallet.Balance))
	s.metrics.RecordWalletOperation("exchange", "success")

	return newWallet.Balance, nil
}

func (s *WalletService) Spend(ctx context.Context, req *dto.SpendRequest) (float64, error) {
	ctx, span := s.tracer.StartSpan(ctx, "WalletService.Spend", 
		trace.WithAttributes(
			attribute.Int("user_id", req.UserID),
			attribute.Float64("amount", req.Amount),
			attribute.String("reason", req.Reason),
		))
	defer span.End()

	s.logger.Info("Processing spend request", 
		zap.Int("user_id", req.UserID),
		zap.Float64("amount", req.Amount),
		zap.String("reason", req.Reason))

	// Start a transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		s.metrics.RecordWalletOperation("spend", "error_transaction")
		return 0, err
	}
	defer tx.Rollback()

	// Get wallet with lock
	wallet, err := s.repo.GetWalletByUserIDForUpdate(ctx, req.UserID, tx)
	if err != nil {
		s.logger.Error("Error getting wallet for update", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("spend", "error_wallet_fetch")
		return 0, err
	}

	if wallet == nil {
		s.logger.Error("Wallet not found for user", zap.Int("user_id", req.UserID))
		s.metrics.RecordWalletOperation("spend", "error_wallet_not_found")
		return 0, fmt.Errorf("wallet not found for user_id=%d", req.UserID)
	}
	
	// Check if balance is sufficient
	if wallet.Balance < req.Amount {
		s.logger.Error("Insufficient funds", 
			zap.Int("user_id", req.UserID),
			zap.Float64("current_balance", wallet.Balance),
			zap.Float64("required_amount", req.Amount))
		s.metrics.RecordWalletOperation("spend", "error_insufficient_funds")
		return 0, fmt.Errorf("insufficient funds: current balance %.2f, required %.2f", wallet.Balance, req.Amount)
	}

	// Update wallet balance
	updatedWallet, err := s.repo.SpendFromWallet(ctx, req.UserID, req.Amount, tx)
	if err != nil {
		s.logger.Error("Failed to spend from wallet", 
			zap.Int("user_id", req.UserID),
			zap.Float64("amount", req.Amount),
			zap.Error(err))
		s.metrics.RecordWalletOperation("spend", "error_update_wallet")
		return 0, err
	}

	// Create wallet log
	walletLog := &model.WalletLog{
		WalletID:       wallet.ID,
		UserID:         req.UserID,
		Amount:         -req.Amount,
		PlatformAmount: -req.Amount,
		Source:         req.Reason,
		ReferenceID:    &req.ReferenceID,
	}
	
	_, err = s.repo.CreateWalletLog(ctx, walletLog, tx)
	if err != nil {
		s.logger.Error("Failed to create wallet log", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("spend", "error_log")
		return 0, err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", 
			zap.Int("user_id", req.UserID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("spend", "error_commit")
		return 0, err
	}

	s.logger.Info("Spend completed successfully", 
		zap.Int("user_id", req.UserID),
		zap.Float64("amount", req.Amount),
		zap.Float64("new_balance", updatedWallet.Balance))
	s.metrics.RecordWalletOperation("spend", "success")

	return updatedWallet.Balance, nil
}

func (s *WalletService) GetWalletLogs(ctx context.Context, userID int) ([]dto.WalletLogEntry, error) {
	ctx, span := s.tracer.StartSpan(ctx, "WalletService.GetWalletLogs",
		trace.WithAttributes(attribute.Int("user_id", userID)))
	defer span.End()

	s.logger.Info("Getting wallet logs for user", zap.Int("user_id", userID))
	
	// Default limits
	limit := 50
	offset := 0
	
	// Get logs from repository
	logs, err := s.repo.GetWalletLogs(ctx, userID, limit, offset)
	
	if err != nil {
		s.logger.Error("Error retrieving wallet logs", 
			zap.Int("user_id", userID),
			zap.Error(err))
		s.metrics.RecordWalletOperation("get_logs", "error")
		return nil, err
	}

	s.logger.Debug("Retrieved wallet logs successfully", 
		zap.Int("user_id", userID),
		zap.Int("log_count", len(logs)))
	s.metrics.RecordWalletOperation("get_logs", "success")

	// Convert model to DTO
	result := make([]dto.WalletLogEntry, len(logs))
	for i, log := range logs {
		operation := model.TransactionExchange
		if log.Amount < 0 {
			operation = model.TransactionSpend
		}
		
		source := log.Source
		
		entry := dto.WalletLogEntry{
			OriginalAmount:  log.Amount,
			ConvertedAmount: log.PlatformAmount,
			CreatedAt:       log.CreatedAt,
			Operation:       operation,
			Source:          &source,
		}
		
		if log.GameID != nil {
			entry.GameID = log.GameID
		}
		
		if log.TokenType != nil {
			entry.TokenType = log.TokenType
		}
		
		if log.ReferenceID != nil {
			entry.ReferenceID = log.ReferenceID
		}
		
		result[i] = entry
	}

	return result, nil
}
