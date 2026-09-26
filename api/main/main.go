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
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"mini-crypto-wallet-api/db_conn"
	_ "mini-crypto-wallet-api/docs"
	"mini-crypto-wallet-api/router"
)

// shutdownTimeout bounds how long we wait for in-flight requests to finish
// during a graceful shutdown before giving up.
const shutdownTimeout = 10 * time.Second

func main() {
	config.LoadConfig()
	db_conn.InitDatabase()

	// 初始化 Kafka Producer
	kafkaBroker := config.Config.KafkaBroker
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}
	producer := kafka_client.NewKafkaProducer(kafkaBroker, "tx.created")
	defer producer.Close()

	r := router.SetupRouter(producer)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Bind to all interfaces, not just localhost: inside a container,
	// "localhost:8080" only accepts connections from within the container
	// itself, so any port mapping from the host (or another container)
	// would never reach it. See docs/AUDIT.md Q7.
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Run the server in the background so main() can wait for a shutdown
	// signal below, instead of blocking here forever (docs/AUDIT.md Q8).
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ server failed: %v", err)
		}
	}()
	log.Println("✅ server listening on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 shutdown signal received, draining in-flight requests...")

	// Stop accepting new connections and wait for in-flight requests to
	// finish (up to shutdownTimeout) before letting main() return - which
	// runs the deferred producer.Close() above, so we don't tear down the
	// Kafka connection out from under a request that's still using it.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("⚠️ graceful shutdown did not complete cleanly: %v", err)
	} else {
		log.Println("✅ server stopped cleanly")
	}
}
