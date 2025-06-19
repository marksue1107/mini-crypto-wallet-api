package handlers

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/services"
	"net/http"
	"strconv"
)

type TransactionHandler struct {
	service *services.TransactionService
}

func NewTransactionHandler(service *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{service}
}

// Transfer 執行兩個使用者之間的轉帳動作
//
// @Summary Transfer funds
// @Description Transfer funds between two users
// @Tags Wallet
// @Accept json
// @Produce json
// @Param transfer body models.TransferRequest true "Transfer info"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /wallet/transfer [post]
func (h *TransactionHandler) Transfer(c *gin.Context) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Transfer(req.FromUserID, req.ToUserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer successful"})
}

// GetTransactions 根據使用者 ID 取得交易紀錄清單
//
// @Summary Get user transactions
// @Description Get all transactions related to a specific user
// @Tags Transactions
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {array} models.Transaction
// @Failure 404 {object} map[string]string
// @Router /transactions/{user_id} [get]
func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	txs, err := h.service.GetTransactions(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}
	c.JSON(http.StatusOK, txs)
}
