package config

import (
	"github.com/spf13/viper"
	"log"
	"strings"
)

func LoadConfig() {
	viper.SetConfigName("config") // config.yaml
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")      // 專案根目錄
	viper.AddConfigPath("../../") // config 資料夾

	// Register every key with an empty default so viper.AutomaticEnv can
	// resolve it purely from environment variables even when no config.yaml
	// exists at all. Without this, viper's AutomaticEnv only kicks in for
	// keys it already knows about (from the config file, a flag, or a
	// prior SetDefault/BindEnv call) - it can't discover arbitrary struct
	// fields on its own. This matters now that config.yaml is gitignored
	// and not deployed anywhere (see docs/AUDIT.md S1): production must be
	// able to run from JWT_SECRET / POSTGRES_DSN / etc. env vars alone.
	viper.SetDefault("app_env", "development")
	viper.SetDefault("db_driver", "sqlite")
	viper.SetDefault("postgres_dsn", "")
	viper.SetDefault("kafka_broker", "")
	viper.SetDefault("jwt_secret", "")
	viper.SetDefault("redis_addr", "")
	viper.SetDefault("max_transfer_amount", "")
	viper.SetDefault("trusted_proxies", "")
	viper.SetDefault("cors_allowed_origins", "")

	viper.AutomaticEnv()                                   // 支援環境變數覆蓋
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 支援 APP_ENV ➜ app.env

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("⚠️  Failed to read config file (this is expected if configuring via environment variables only): %v", err)
	}

	var appConfig AppConfig
	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatalf("❌ Failed to unmarshal config: %v", err)
	}

	Config = &appConfig
	log.Println("✅ Config loaded:", Config.AppEnv)
}
