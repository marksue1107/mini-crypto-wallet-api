package db_conn

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"mini-crypto-wallet-api/internal/config"
)

func initPostgres() {
	dsn := config.Config.PostgresDSN
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to PostgreSQL:", err)
	}

	Conn_DB.MasterDB = db

	log.Println("✅ PostgreSQL connected")
}
