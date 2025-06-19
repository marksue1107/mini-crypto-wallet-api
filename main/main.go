package main

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"mini-crypto-wallet-api/kafkaclient"

	"mini-crypto-wallet-api/database"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/router"
)

func main() {
	database.InitDatabase()

	// 初始化 Kafka Producer（連到 localhost:9092）
	producer := kafkaclient.NewKafkaProducer("localhost:9092", "tx.created")

	r := router.SetupRouter(producer)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run("localhost:8080")
}
