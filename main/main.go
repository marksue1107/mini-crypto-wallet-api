package main

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"

	"mini-crypto-wallet-api/db_conn"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/router"
)

func main() {
	config.LoadConfig()
	db_conn.InitDatabase()

	// 初始化 Kafka Producer（連到 localhost:9092）
	producer := kafka_client.NewKafkaProducer("localhost:9092", "tx.created")

	r := router.SetupRouter(producer)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run("localhost:8080")
}
