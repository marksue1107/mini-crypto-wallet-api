package main

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"mini-crypto-wallet-api/database"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/router"
)

func main() {
	database.InitDatabase()
	r := router.SetupRouter()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run("localhost:8080")
}
