package handlers

import (
	apierrors "mini-crypto-wallet-api/internal/errors"
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

// transferErrorResponses maps Transfer()'s sentinel errors to a stable HTTP
// status + error code, so API consumers can branch on `code` instead of
// parsing message text. Anything not in this map is treated as an
// unexpected server-side failure. See docs/AUDIT.md S8.
var transferErrorResponses = map[error]struct {
	Status int
	Code   string
}{
	services.ErrSameAccountTransfer:  {http.StatusBadRequest, apierrors.ErrCodeSameAccountTransfer},
	services.ErrAmountMustBePositive: {http.StatusBadRequest, apierrors.ErrCodeInvalidAmount},
	services.ErrAmountExceedsLimit:   {http.StatusBadRequest, apierrors.ErrCodeInvalidAmount},
	services.ErrTooManyDecimalPlaces: {http.StatusBadRequest, apierrors.ErrCodeInvalidAmount},
	services.ErrCurrencyNotFound:     {http.StatusNotFound, apierrors.ErrCodeNotFound},
	services.ErrFromWalletNotFound:   {http.StatusNotFound, apierrors.ErrCodeWalletNotFound},
	services.ErrToWalletNotFound:     {http.StatusNotFound, apierrors.ErrCodeWalletNotFound},
	services.ErrInsufficientBalance:  {http.StatusBadRequest, apierrors.ErrCodeInsufficientBalance},
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
// @Success 200 {object} models.TransactionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Router /wallet/transfer [post]
func (h *TransactionHandler) Transfer(c *gin.Context) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleValidationError(c, err)
		return
	}

	// 檢查用戶只能從自己的帳戶轉帳
	if !middleware.RequireUserID(c, req.FromUserID) {
		return
	}

	tx, err := h.service.Transfer(req.FromUserID, req.ToUserID, req.CurrencyID, req.Amount)
	if err != nil {
		if resp, ok := transferErrorResponses[err]; ok {
			apierrors.RespondError(c, resp.Status, resp.Code, err)
			return
		}
		apierrors.RespondError(c, http.StatusInternalServerError, apierrors.ErrCodeTransactionFailed, err)
		return
	}
	c.JSON(http.StatusOK, models.ToTransactionResponse(tx))
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
// @Success 200 {object} models.TransactionListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /transactions/{user_id} [get]
func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		apierrors.RespondError(c, http.StatusBadRequest, apierrors.ErrCodeInvalidRequest, errInvalidUserID)
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
		apierrors.RespondError(c, http.StatusInternalServerError, apierrors.ErrCodeInternalError, errFailedToFetchTransactions)
		return
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Convert models to DTOs (excludes database relationships)
	txResponses := models.ToTransactionResponses(txs)

	c.JSON(http.StatusOK, models.TransactionListResponse{
		Data: txResponses,
		Pagination: models.PaginationResponse{
			Page:       pagination.Page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetTxByHash 根據交易 Hash 查詢交易資訊
//
// GetTxByHash is intentionally public/unauthenticated, like a blockchain
// explorer's "look up by transaction hash" - anyone who already knows (or
// was given) the hash can look up its details. The hash is a 256-bit
// SHA-256 digest that includes a nanosecond timestamp, so it isn't
// practically guessable; the endpoint is still rate limited (see
// docs/AUDIT.md S3/S6) to bound abuse. If this design changes, update
// router.go to require auth + an ownership check like the other endpoints.
//
// @Summary Get transaction by hash
// @Description Get a transaction detail by its unique hash. Public endpoint (no auth required) - see docs/AUDIT.md S6.
// @Tags Transactions
// @Produce json
// @Param hash path string true "Transaction Hash"
// @Success 200 {object} models.TransactionResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Router /tx/{hash} [get]
func (h *TransactionHandler) GetTxByHash(c *gin.Context) {
	hash := c.Param("hash")
	tx, err := h.service.GetTransactionByHash(hash)
	if err != nil {
		apierrors.RespondError(c, http.StatusNotFound, apierrors.ErrCodeTransactionNotFound, errTransactionNotFound)
		return
	}

	// Convert model to DTO (excludes database relationships)
	response := models.ToTransactionResponse(tx)
	c.JSON(http.StatusOK, response)
}
