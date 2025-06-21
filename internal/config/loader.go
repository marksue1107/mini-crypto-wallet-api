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

	viper.AutomaticEnv()                                   // 支援環境變數覆蓋
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 支援 APP_ENV ➜ app.env

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("⚠️  Failed to read config file: %v", err)
	}

	var appConfig AppConfig
	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatalf("❌ Failed to unmarshal config: %v", err)
	}

	Config = &appConfig
	log.Println("✅ Config loaded:", Config.AppEnv)
}
