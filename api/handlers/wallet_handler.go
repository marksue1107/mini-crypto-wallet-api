package handlers

import (
	_ "mini-crypto-wallet-api/docs"
	apierrors "mini-crypto-wallet-api/internal/errors"
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	service *services.WalletService
}

func NewWalletHandler(service *services.WalletService) *WalletHandler {
	return &WalletHandler{service}
}

// GetWallet 根據使用者 ID 查詢錢包餘額
//
// @Summary Get wallet balance
// @Description Retrieve wallet balance by user ID
// @Tags Wallet
// @Security BearerAuth
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} models.WalletResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /wallet/{user_id} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		apierrors.RespondError(c, http.StatusBadRequest, apierrors.ErrCodeInvalidRequest, errInvalidUserID)
		return
	}

	// 檢查用戶只能訪問自己的錢包
	if !middleware.RequireUserID(c, uint(userID)) {
		return
	}

	wallet, err := h.service.GetWallet(uint(userID))
	if err != nil {
		apierrors.RespondError(c, http.StatusNotFound, apierrors.ErrCodeWalletNotFound, errWalletNotFound)
		return
	}

	// Convert model to DTO (excludes database relationships)
	response := models.ToWalletResponse(wallet)
	c.JSON(http.StatusOK, response)
}
