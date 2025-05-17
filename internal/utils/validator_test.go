package utils_test

import (
	"testing"

	"github.com/playconomy/wallet-service/internal/server/dto"
	"github.com/playconomy/wallet-service/internal/utils"
	
	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	t.Run("Valid Wallet", func(t *testing.T) {
		wallet := &dto.Wallet{
			ID:      1,
			UserID:  123,
			Balance: 100.0,
		}

		err := utils.ValidateStruct(wallet)
		assert.NoError(t, err)
	})

	t.Run("Invalid Wallet - Negative Balance", func(t *testing.T) {
		wallet := &dto.Wallet{
			ID:      1,
			UserID:  123,
			Balance: -50.0, // Negative balance should fail validation
		}

		err := utils.ValidateStruct(wallet)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Balance")
	})

	t.Run("Invalid Wallet - Missing UserID", func(t *testing.T) {
		wallet := &dto.Wallet{
			ID:      1,
			UserID:  0, // Zero UserID should fail validation
			Balance: 100.0,
		}

		err := utils.ValidateStruct(wallet)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UserID")
	})

	t.Run("Valid Exchange Request", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100.0,
			Source:    "won",
		}

		err := utils.ValidateStruct(req)
		assert.NoError(t, err)
	})

	t.Run("Invalid Exchange Request - Invalid Source", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    100.0,
			Source:    "invalid", // Not in the oneof values
		}

		err := utils.ValidateStruct(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Source")
	})

	t.Run("Invalid Exchange Request - Zero Amount", func(t *testing.T) {
		req := &dto.ExchangeRequest{
			UserID:    123,
			GameID:    "game1",
			TokenType: "gold",
			Amount:    0.0, // Zero amount should fail validation
			Source:    "won",
		}

		err := utils.ValidateStruct(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Amount")
	})

	t.Run("Valid Spend Request", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "ORDER-123",
		}

		err := utils.ValidateStruct(req)
		assert.NoError(t, err)
	})

	t.Run("Invalid Spend Request - Invalid Reason", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "invalid", // Not in the oneof values
			ReferenceID: "ORDER-123",
		}

		err := utils.ValidateStruct(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Reason")
	})

	t.Run("Invalid Spend Request - Empty Reference ID", func(t *testing.T) {
		req := &dto.SpendRequest{
			UserID:      123,
			Amount:      50.0,
			Reason:      "market_purchase",
			ReferenceID: "", // Empty reference ID should fail validation
		}

		err := utils.ValidateStruct(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ReferenceID")
	})
}
