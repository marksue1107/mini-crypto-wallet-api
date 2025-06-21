package config

type AppConfig struct {
	AppEnv      string `mapstructure:"app_env"`
	DBDriver    string `mapstructure:"db_driver"`
	PostgresDSN string `mapstructure:"postgres_dsn"`
	KafkaBroker string `mapstructure:"kafka_broker"`
}

var Config *AppConfig
