package handlers

import (
	"github.com/gin-gonic/gin"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/services"
	"net/http"
	"strconv"
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
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} models.Wallet
// @Failure 404 {object} map[string]string
// @Router /wallet/{user_id} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	wallet, err := h.service.GetWallet(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}
