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
	seedDefaultCurrency()
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

// seedDefaultCurrency ensures at least one currency exists. Without this, a
// brand-new database (e.g. a fresh `docker compose up`) has zero rows in
// the currencies table, and there is no admin endpoint to create one via
// the API - so UserService.CreateUser's "get the default USDT currency, or
// fall back to the first one available" logic has nothing to fall back to,
// and every single POST /users fails with "no currency available". This
// was only discovered by actually running the full docker-compose stack
// end to end (see docs/AUDIT_REMEDIATION_PLAN.md Batch 5 notes) - every
// automated test up to this point used test helpers that create a
// currency explicitly.
func seedDefaultCurrency() {
	var count int64
	if err := Conn_DB.MasterDB.Model(&models.Currency{}).Count(&count).Error; err != nil {
		log.Fatal("❌ Failed to check currency count:", err)
	}
	if count > 0 {
		return
	}

	usdt := &models.Currency{
		Code:     "USDT",
		Name:     "Tether",
		Symbol:   "$",
		Decimals: 8,
		IsActive: true,
	}
	if err := Conn_DB.MasterDB.Create(usdt).Error; err != nil {
		log.Fatal("❌ Failed to seed default currency:", err)
	}
	log.Println("✅ Seeded default currency: USDT")
}
