package database

import (
	"gorm.io/gorm/logger"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"

	"mini-crypto-wallet-api/models"
)

var DB struct {
	MasterDB *gorm.DB
}

func InitDatabase() {
	var err error

	// 使用 modernc 的 sqlite driver
	dialector := sqlite.Dialector{
		DSN:        "data.db",
		DriverName: "sqlite", // 必須加這行才能明確用 modernc driver
		Conn:       nil,
	}

	DB.MasterDB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	log.Println("✅ Database connected")

	err = DB.MasterDB.AutoMigrate(&models.User{}, &models.Wallet{}, &models.Transaction{})
	if err != nil {
		log.Fatal("❌ Failed to migrate database:", err)
	}

	log.Println("✅ Database migrated")
}
