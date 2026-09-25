package config

type AppConfig struct {
	AppEnv            string `mapstructure:"app_env"`
	DBDriver          string `mapstructure:"db_driver"`
	PostgresDSN       string `mapstructure:"postgres_dsn"`
	KafkaBroker       string `mapstructure:"kafka_broker"`
	JWTSecret         string `mapstructure:"jwt_secret"`
	RedisAddr         string `mapstructure:"redis_addr"`
	MaxTransferAmount string `mapstructure:"max_transfer_amount"`
}

var Config *AppConfig
