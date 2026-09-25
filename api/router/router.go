package router

import (
	"log"
	"mini-crypto-wallet-api/handlers"
	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func SetupRouter(producer *kafka_client.KafkaProducer) *gin.Engine {
	r := gin.Default()

	// Trust no proxy by default: without this, Gin trusts every proxy for
	// X-Forwarded-For/X-Real-IP parsing, so ClientIP() (and therefore the
	// rate limiter below, which keys on it) can be spoofed by any client
	// simply sending its own X-Forwarded-For header, unless this service
	// sits behind a reverse proxy that overwrites that header. Only trust
	// specific proxies you control by setting TRUSTED_PROXIES. See
	// docs/AUDIT.md S4.
	var trustedProxies []string
	if config.Config.TrustedProxies != "" {
		trustedProxies = strings.Split(config.Config.TrustedProxies, ",")
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("❌ invalid TRUSTED_PROXIES: %v", err)
	}

	// CORS. Disabled (no middleware at all) unless CORS_ALLOWED_ORIGINS is
	// explicitly set - the safe default is that browsers block cross-origin
	// requests until a frontend origin is configured. Never combine a
	// wildcard origin with AllowCredentials: true (browsers forbid it, and
	// gin-contrib/cors will reject the config). See docs/AUDIT.md S5.
	if config.Config.CORSAllowedOrigins != "" {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     strings.Split(config.Config.CORSAllowedOrigins, ","),
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	// 添加追蹤中間件
	r.Use(middleware.TraceMiddleware())

	// Init JWT Manager. Fail fast rather than silently signing tokens with
	// a secret hardcoded in this repo's source - see docs/AUDIT.md S2.
	jwtSecret := config.Config.JWTSecret
	if err := auth.ValidateSecretStrength(jwtSecret); err != nil {
		log.Fatalf("❌ invalid JWT secret: %v (set the JWT_SECRET environment variable to a random string of at least %d characters)", err, auth.MinSecretLength)
	}
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	// Rate limiters. See docs/AUDIT.md S3: these middleware existed but
	// were never wired into any route. Login gets a much stricter limit
	// since it's the most common brute-force target.
	generalLimiter := middleware.DefaultRateLimit()                                                          // 60 req/min
	loginLimiter := middleware.RateLimitMiddleware(middleware.NewRateLimiter(rate.Every(time.Minute/10), 5)) // 10 req/min, small burst

	// Init repository
	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	currencyRepo := repositories.NewCurrencyRepository()

	// Init service
	userService := services.NewUserService(userRepo, walletRepo, currencyRepo)
	walletService := services.NewWalletService(walletRepo)
	txService := services.NewTransactionService(walletRepo, txRepo, currencyRepo, producer)
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
	r.POST("/users", generalLimiter, userHandler.CreateUser)
	r.POST("/auth/login", loginLimiter, userHandler.Login)
	r.GET("/currencies", currencyHandler.GetCurrencies)
	r.GET("/currencies/:id", currencyHandler.GetCurrency)
	r.GET("/tx/:hash", generalLimiter, txHandler.GetTxByHash)

	// Protected routes - require authentication
	authMiddleware := middleware.AuthMiddleware(jwtManager)
	protected := r.Group("/")
	protected.Use(authMiddleware)
	{
		protected.GET("/wallet/:user_id", walletHandler.GetWallet)
		protected.POST("/wallet/transfer", generalLimiter, txHandler.Transfer)
		protected.GET("/transactions/:user_id", txHandler.GetTransactions)
	}

	return r
}
