package router

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/handlers"
	"mini-crypto-wallet-api/kafka_client"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"
)

func SetupRouter(producer *kafka_client.KafkaProducer) *gin.Engine {
	r := gin.Default()

	// Init repository
	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()

	// Init service
	userService := services.NewUserService(userRepo, walletRepo)
	walletService := services.NewWalletService(walletRepo)
	txService := services.NewTransactionService(walletRepo, txRepo, producer)

	// Init handlers
	userHandler := handlers.NewUserHandler(userService)
	walletHandler := handlers.NewWalletHandler(walletService)
	txHandler := handlers.NewTransactionHandler(txService)

	r.POST("/users", userHandler.CreateUser)
	r.GET("/wallet/:user_id", walletHandler.GetWallet)
	r.POST("/wallet/transfer", txHandler.Transfer)
	r.GET("/transactions/:user_id", txHandler.GetTransactions)
	r.GET("/tx/:hash", txHandler.GetTxByHash)

	return r
}
