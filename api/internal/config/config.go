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
}

var Config *AppConfig
