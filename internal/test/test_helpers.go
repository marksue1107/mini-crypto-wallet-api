package test

import (
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB initializes an in-memory SQLite database for testing
func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to test database:", err)
	}

	// Set the global DB connection for repositories
	db_conn.Conn_DB.MasterDB = db

	// Auto-migrate all tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Currency{},
		&models.Wallet{},
		&models.Transaction{},
		&models.BalanceHistory{},
	)
	if err != nil {
		log.Fatal("❌ Failed to migrate test database:", err)
	}

	return db
}

// CleanupTestDB tears down the test database
func CleanupTestDB(db *gorm.DB) {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// CreateTestUser creates a test user with the given username
func CreateTestUser(db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username: username,
		Email:    username + "@example.com",
		Password: "hashed_password_for_testing",
	}
	result := db_conn.Conn_DB.MasterDB.Create(user)
	if result.Error != nil {
		log.Fatal("❌ Failed to create test user:", result.Error)
	}
	return user
}

// CreateTestCurrency creates a test currency with the given code
func CreateTestCurrency(db *gorm.DB, code string) *models.Currency {
	currency := &models.Currency{
		Code:     code,
		Name:     "Test " + code,
		Symbol:   "$",
		Decimals: 8,
		IsActive: true,
	}
	result := db_conn.Conn_DB.MasterDB.Create(currency)
	if result.Error != nil {
		log.Fatal("❌ Failed to create test currency:", result.Error)
	}
	return currency
}

// CreateTestWallet creates a test wallet with the given parameters
func CreateTestWallet(db *gorm.DB, userID uint, currencyID uint, balance int64) *models.Wallet {
	wallet := &models.Wallet{
		UserID:     userID,
		CurrencyID: currencyID,
		Balance:    decimal.NewFromInt(balance),
	}
	result := db_conn.Conn_DB.MasterDB.Create(wallet)
	if result.Error != nil {
		log.Fatal("❌ Failed to create test wallet:", result.Error)
	}
	return wallet
}

// CreateTestWalletWithDecimal creates a test wallet with a decimal balance
func CreateTestWalletWithDecimal(db *gorm.DB, userID uint, currencyID uint, balance decimal.Decimal) *models.Wallet {
	wallet := &models.Wallet{
		UserID:     userID,
		CurrencyID: currencyID,
		Balance:    balance,
	}
	result := db_conn.Conn_DB.MasterDB.Create(wallet)
	if result.Error != nil {
		log.Fatal("❌ Failed to create test wallet:", result.Error)
	}
	return wallet
}
