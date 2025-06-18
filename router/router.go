package router

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/handlers"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/users", handlers.CreateUser)
	r.GET("/wallet/:user_id", handlers.GetWallet)
	r.POST("/wallet/transfer", handlers.Transfer)
	r.GET("/transactions/:user_id", handlers.GetTransactions)

	return r
}
