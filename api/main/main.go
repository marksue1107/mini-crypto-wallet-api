// @title Mini Crypto Wallet API
// @version 1.0
// @description Mini crypto wallet backend API.
// @termsOfService http://swagger.io/terms/

// @contact.name Mark
// @contact.email dev@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
package main

import (
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"mini-crypto-wallet-api/db_conn"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/router"
)

func main() {
	config.LoadConfig()
	db_conn.InitDatabase()

	// 初始化 Kafka Producer
	kafkaBroker := config.Config.KafkaBroker
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}
	producer := kafka_client.NewKafkaProducer(kafkaBroker, "tx.created")

	r := router.SetupRouter(producer)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run("localhost:8080")
}
