package router

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/handlers"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 初始化 repository
	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()

	// 初始化 service
	userService := services.NewUserService(userRepo, walletRepo)
	walletService := services.NewWalletService(walletRepo)
	txService := services.NewTransactionService(walletRepo, txRepo)

	// 初始化 handler
	userHandler := handlers.NewUserHandler(userService)
	walletHandler := handlers.NewWalletHandler(walletService)
	txHandler := handlers.NewTransactionHandler(txService)

	r.POST("/users", userHandler.CreateUser)
	r.GET("/wallet/:user_id", walletHandler.GetWallet)
	r.POST("/wallet/transfer", txHandler.Transfer)
	r.GET("/transactions/:user_id", txHandler.GetTransactions)

	return r
}
