package handler

import (
	"strconv"
	"strings"

	"github.com/playconomy/wallet-service/internal/server/dto"
	"github.com/playconomy/wallet-service/internal/service"
	"github.com/playconomy/wallet-service/internal/utils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// Module provides handler dependencies
var Module = fx.Options(
	fx.Provide(NewWalletHandler),
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
	}
}

// GetWallet retrieves wallet information for a user
//
//	@Summary		Get wallet information
//	@Description	Returns wallet information for a specific user
//	@Tags			wallet
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		int					true	"User ID"
//	@Success		200		{object}	dto.WalletResponse	"Wallet information"
//	@Failure		400		{object}	dto.WalletResponse	"Invalid user ID"
//	@Failure		401		{object}	dto.GenericResponse	"Unauthorized"
//	@Failure		403		{object}	dto.WalletResponse	"Forbidden"
//	@Failure		404		{object}	dto.WalletResponse	"Wallet not found"
//	@Failure		500		{object}	dto.WalletResponse	"Server error"
//	@Security		ApiKeyAuth
//	@Security		ApiEmailAuth
//	@Security		ApiRoleAuth
//	@Router			/{user_id} [get]
func (h *WalletHandler) GetWallet(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	authenticatedUserID := c.Locals("user_id").(int)

	// Get requested user ID from path parameter
	userID, err := strconv.Atoi(c.Params("user_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.WalletResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
	}

	// Validate user ID
	if err := utils.ValidateStruct(&dto.Wallet{UserID: userID}); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.WalletResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Security check: users can only view their own wallet
	// Unless they have admin role
	userRole := c.Locals("user_role").(string)
	if authenticatedUserID != userID && userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.WalletResponse{
			Success: false,
			Error:   "You can only access your own wallet",
		})
	}

	wallet, err := h.walletService.GetWalletByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.WalletResponse{
			Success: false,
			Error:   "Internal server error",
		})
	}

	if wallet == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.WalletResponse{
			Success: false,
			Error:   "Wallet not found",
		})
	}

	return c.JSON(dto.WalletResponse{
		Success: true,
		Data:    wallet,
	})
}

// Exchange converts game tokens to platform tokens
//
//	@Summary		Exchange game tokens
//	@Description	Converts game tokens to platform tokens and adds to wallet
//	@Tags			wallet,exchange
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ExchangeRequest		true	"Exchange request"
//	@Success		200		{object}	dto.ExchangeResponse	"Exchange result"
//	@Failure		400		{object}	dto.ExchangeResponse	"Invalid request or exchange rate not found"
//	@Failure		401		{object}	dto.GenericResponse		"Unauthorized"
//	@Failure		403		{object}	dto.ExchangeResponse	"Forbidden"
//	@Failure		500		{object}	dto.ExchangeResponse	"Server error"
//	@Security		ApiKeyAuth
//	@Security		ApiEmailAuth
//	@Security		ApiRoleAuth
//	@Router			/exchange [post]
func (h *WalletHandler) Exchange(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	authenticatedUserID := c.Locals("user_id").(int)

	var req dto.ExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ExchangeResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ExchangeResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Security check: users can only exchange to their own wallet
	// Unless they have admin role
	userRole := c.Locals("user_role").(string)
	if authenticatedUserID != req.UserID && userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ExchangeResponse{
			Success: false,
			Error:   "You can only exchange to your own wallet",
		})
	}

	newBalance, err := h.walletService.Exchange(&req)
	if err != nil {
		if strings.Contains(err.Error(), "exchange rate not found") {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ExchangeResponse{
				Success: false,
				Error:   err.Error(),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ExchangeResponse{
			Success: false,
			Error:   "Internal server error",
		})
	}

	return c.JSON(dto.ExchangeResponse{
		Success:    true,
		NewBalance: newBalance,
	})
}

// Spend deducts tokens from a user's wallet
//
//	@Summary		Spend tokens
//	@Description	Deducts tokens from a user's wallet for purchases or entries
//	@Tags			wallet,spend
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.SpendRequest	true	"Spend request"
//	@Success		200		{object}	dto.SpendResponse	"Spend result"
//	@Failure		400		{object}	dto.SpendResponse	"Invalid request, insufficient funds, or wallet not found"
//	@Failure		401		{object}	dto.GenericResponse	"Unauthorized"
//	@Failure		403		{object}	dto.SpendResponse	"Forbidden"
//	@Failure		500		{object}	dto.SpendResponse	"Server error"
//	@Security		ApiKeyAuth
//	@Security		ApiEmailAuth
//	@Security		ApiRoleAuth
//	@Router			/spend [post]
func (h *WalletHandler) Spend(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	authenticatedUserID := c.Locals("user_id").(int)

	var req dto.SpendRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.SpendResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.SpendResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Security check: users can only spend from their own wallet
	// Unless they have admin role
	userRole := c.Locals("user_role").(string)
	if authenticatedUserID != req.UserID && userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.SpendResponse{
			Success: false,
			Error:   "You can only spend from your own wallet",
		})
	}

	newBalance, err := h.walletService.Spend(&req)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient funds") ||
			strings.Contains(err.Error(), "wallet not found") {
			return c.Status(fiber.StatusBadRequest).JSON(dto.SpendResponse{
				Success: false,
				Error:   err.Error(),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.SpendResponse{
			Success: false,
			Error:   "Internal server error",
		})
	}

	return c.JSON(dto.SpendResponse{
		Success:    true,
		NewBalance: newBalance,
	})
}

// GetWalletLogs retrieves transaction logs for a user wallet
//
//	@Summary		Get wallet transaction logs
//	@Description	Returns transaction logs for a specific user wallet
//	@Tags			wallet,logs
//	@Accept			json
//	@Produce		json
//	@Param			user_id	path		int						true	"User ID"
//	@Success		200		{object}	dto.WalletLogsResponse	"Wallet logs"
//	@Failure		400		{object}	dto.WalletLogsResponse	"Invalid user ID"
//	@Failure		401		{object}	dto.GenericResponse		"Unauthorized"
//	@Failure		403		{object}	dto.WalletLogsResponse	"Forbidden"
//	@Failure		500		{object}	dto.WalletLogsResponse	"Server error"
//	@Security		ApiKeyAuth
//	@Security		ApiEmailAuth
//	@Security		ApiRoleAuth
//	@Router			/{user_id}/logs [get]
func (h *WalletHandler) GetWalletLogs(c *fiber.Ctx) error {
	// Get authenticated user ID from context
	authenticatedUserID := c.Locals("user_id").(int)

	// Get requested user ID from path parameter
	userID, err := strconv.Atoi(c.Params("user_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.WalletLogsResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
	}

	// Validate user ID
	if err := utils.ValidateStruct(&dto.Wallet{UserID: userID}); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.WalletLogsResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Security check: users can only view their own logs
	// Unless they have admin role
	userRole := c.Locals("user_role").(string)
	if authenticatedUserID != userID && userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.WalletLogsResponse{
			Success: false,
			Error:   "You can only access your own wallet logs",
		})
	}

	logs, err := h.walletService.GetWalletLogs(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.WalletLogsResponse{
			Success: false,
			Error:   "Internal server error",
		})
	}

	return c.JSON(dto.WalletLogsResponse{
		Success: true,
		Data:    logs,
	})
}
