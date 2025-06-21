package db_conn

import (
	"gorm.io/gorm/logger"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func initSQLite() {
	var err error
	directory := sqlite.Dialector{
		DSN:        "mini_wallet.db",
		DriverName: "sqlite",
	}

	Conn_DB.MasterDB, err = gorm.Open(directory, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to SQLite:", err)
	}

	log.Println("✅ SQLite connected")
}
