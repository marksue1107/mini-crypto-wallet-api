package handlers

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/database"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/models"
	"net/http"
	"strconv"
)

// CreateUser 建立使用者，並初始化 Wallet（預設 1000 USDC）
//
// @Summary Create user
// @Description create a new wallet user
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "User info"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /users [post]
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 建立 User
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// 建立 Wallet（預設金額）
	wallet := models.Wallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	database.DB.Create(&wallet)

	c.JSON(http.StatusOK, gin.H{"message": "user created", "user_id": user.ID})
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
func GetWallet(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var wallet models.Wallet
	if err := database.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID, "balance": wallet.Balance})
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
func Transfer(c *gin.Context) {
	var req models.TransferRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var fromWallet, toWallet models.Wallet
	if err := database.DB.Where("user_id = ?", req.FromUserID).First(&fromWallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "from_user wallet not found"})
		return
	}
	if err := database.DB.Where("user_id = ?", req.ToUserID).First(&toWallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "to_user wallet not found"})
		return
	}

	if fromWallet.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// 執行轉帳（可加入 transaction block，簡化略）
	fromWallet.Balance -= req.Amount
	toWallet.Balance += req.Amount

	database.DB.Save(&fromWallet)
	database.DB.Save(&toWallet)

	// 建立交易紀錄
	tx := models.Transaction{
		FromUserID: req.FromUserID,
		ToUserID:   req.ToUserID,
		Amount:     req.Amount,
	}
	database.DB.Create(&tx)

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
func GetTransactions(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var txs []models.Transaction
	database.DB.
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Order("created_at desc").
		Find(&txs)

	c.JSON(http.StatusOK, txs)
}
