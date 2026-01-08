package handlers

import (
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
// @Security BearerAuth
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

	// 檢查用戶只能從自己的帳戶轉帳
	if !middleware.RequireUserID(c, req.FromUserID) {
		return
	}

	if err := h.service.Transfer(req.FromUserID, req.ToUserID, req.CurrencyID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer successful"})
}

// GetTransactions 根據使用者 ID 取得交易紀錄清單
//
// @Summary Get user transactions
// @Description Get all transactions related to a specific user with pagination
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param user_id path int true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /transactions/{user_id} [get]
func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 檢查用戶只能查看自己的交易記錄
	if !middleware.RequireUserID(c, uint(userID)) {
		return
	}

	// 解析分頁參數
	var pagination models.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		pagination.Page = 1
		pagination.PageSize = 20
	}

	offset := pagination.GetOffset()
	limit := pagination.GetLimit()

	txs, total, err := h.service.GetTransactionsWithPagination(uint(userID), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data": txs,
		"pagination": models.PaginationResponse{
			Page:       pagination.Page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetTxByHash 根據交易 Hash 查詢交易資訊
//
// @Summary Get transaction by hash
// @Description Get a transaction detail by its unique hash
// @Tags Transactions
// @Produce json
// @Param hash path string true "Transaction Hash"
// @Success 200 {object} models.Transaction
// @Failure 404 {object} map[string]string
// @Router /tx/{hash} [get]
func (h *TransactionHandler) GetTxByHash(c *gin.Context) {
	hash := c.Param("hash")
	tx, err := h.service.GetTransactionByHash(hash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, tx)
}
