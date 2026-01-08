package db_conn

import (
	"gorm.io/gorm"
	"log"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/models"
)

var Conn struct {
	Master *gorm.DB
}

var Conn_DB struct {
	MasterDB *gorm.DB
}

func InitDatabase() {
	switch config.Config.DBDriver {
	case "postgres":
		initPostgres()
	default:
		initSQLite()
	}

	autoMigrate()
}

func autoMigrate() {
	err := Conn_DB.MasterDB.AutoMigrate(
		&models.User{},
		&models.Currency{},
		&models.Wallet{},
		&models.Transaction{},
		&models.BalanceHistory{},
	)
	if err != nil {
		log.Fatal("❌ Failed to migrate database:", err)
	}
	log.Println("✅ Database migrated")
}
