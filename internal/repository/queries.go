// Package repository provides data access layer and database operations
package repository

// SQL queries for wallet operations
const (
	// Wallet queries
	QueryGetWalletByUserID = `
		SELECT id, user_id, balance, created_at 
		FROM wallets 
		WHERE user_id = $1`

	QueryGetWalletByUserIDForUpdate = `
		SELECT id, user_id, balance, created_at 
		FROM wallets 
		WHERE user_id = $1 
		FOR UPDATE`

	QueryCreateWallet = `
		INSERT INTO wallets (user_id, balance) 
		VALUES ($1, $2) 
		RETURNING id, user_id, balance, created_at`

	QueryUpdateWalletBalance = `
		UPDATE wallets 
		SET balance = $2 
		WHERE user_id = $1 
		RETURNING id, user_id, balance, created_at`

	// Exchange rate queries
	QueryGetExchangeRate = `
		SELECT id, game_id, token_type, to_platform_ratio, created_at 
		FROM exchange_rates 
		WHERE game_id = $1 AND token_type = $2`

	QueryGetExchangeRateByID = `
		SELECT id, game_id, token_type, to_platform_ratio, created_at 
		FROM exchange_rates 
		WHERE id = $1`

	// Wallet logs queries
	QueryCreateWalletLog = `
		INSERT INTO wallet_logs (wallet_id, user_id, game_id, token_type, amount, platform_amount, source, reference_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, wallet_id, user_id, game_id, token_type, amount, platform_amount, source, reference_id, created_at`

	QueryGetWalletLogs = `
		SELECT id, wallet_id, user_id, game_id, token_type, amount, platform_amount, source, reference_id, created_at 
		FROM wallet_logs 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	// Spend queries
	QuerySpendFromWallet = `
		UPDATE wallets 
		SET balance = balance - $2 
		WHERE user_id = $1 AND balance >= $2 
		RETURNING id, user_id, balance, created_at`
)
