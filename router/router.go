package router

import (
	"mini-crypto-wallet-api/handlers"
	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter(producer *kafka_client.KafkaProducer) *gin.Engine {
	r := gin.Default()

	// 添加追蹤中間件
	r.Use(middleware.TraceMiddleware())

	// Init JWT Manager
	jwtSecret := config.Config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production-min-32-chars"
	}
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	// Init repository
	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	currencyRepo := repositories.NewCurrencyRepository()

	// Init service
	userService := services.NewUserService(userRepo, walletRepo, currencyRepo)
	walletService := services.NewWalletService(walletRepo)
	txService := services.NewTransactionService(walletRepo, txRepo, producer)
	currencyService := services.NewCurrencyService(currencyRepo)

	// Init handlers
	userHandler := handlers.NewUserHandler(userService, jwtManager)
	walletHandler := handlers.NewWalletHandler(walletService)
	txHandler := handlers.NewTransactionHandler(txService)
	currencyHandler := handlers.NewCurrencyHandler(currencyService)

	// Health check routes
	healthHandler := handlers.NewHealthHandler()
	r.GET("/health", healthHandler.HealthCheck)
	r.GET("/ready", healthHandler.ReadinessCheck)

	// Public routes
	r.POST("/users", userHandler.CreateUser)
	r.POST("/auth/login", userHandler.Login)
	r.GET("/currencies", currencyHandler.GetCurrencies)
	r.GET("/currencies/:id", currencyHandler.GetCurrency)

	// Protected routes - require authentication
	authMiddleware := middleware.AuthMiddleware(jwtManager)
	protected := r.Group("/")
	protected.Use(authMiddleware)
	{
		protected.GET("/wallet/:user_id", walletHandler.GetWallet)
		protected.POST("/wallet/transfer", txHandler.Transfer)
		protected.GET("/transactions/:user_id", txHandler.GetTransactions)
		protected.GET("/tx/:hash", txHandler.GetTxByHash)
	}

	return r
}
