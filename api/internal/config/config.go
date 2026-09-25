package config

type AppConfig struct {
	AppEnv            string `mapstructure:"app_env"`
	DBDriver          string `mapstructure:"db_driver"`
	PostgresDSN       string `mapstructure:"postgres_dsn"`
	KafkaBroker       string `mapstructure:"kafka_broker"`
	JWTSecret         string `mapstructure:"jwt_secret"`
	RedisAddr         string `mapstructure:"redis_addr"`
	MaxTransferAmount string `mapstructure:"max_transfer_amount"`
	// TrustedProxies is a comma-separated list of IPs/CIDRs whose
	// X-Forwarded-For/X-Real-IP headers Gin should trust when determining
	// the client IP. Empty (the default) means trust none - Gin will use
	// the direct connection's remote address instead. See docs/AUDIT.md S4.
	TrustedProxies string `mapstructure:"trusted_proxies"`
	// CORSAllowedOrigins is a comma-separated list of origins (e.g.
	// https://app.example.com) allowed to make cross-origin requests to
	// this API. Empty (the default) means CORS is not enabled at all -
	// browsers will block cross-origin requests, which is the safe
	// default until a frontend origin is explicitly configured. Never set
	// this to "*" together with credentialed requests. See docs/AUDIT.md S5.
	CORSAllowedOrigins string `mapstructure:"cors_allowed_origins"`
}

var Config *AppConfig
