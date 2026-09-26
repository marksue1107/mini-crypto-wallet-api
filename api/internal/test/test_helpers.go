package test

import (
	"fmt"
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"os"
	"sync"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Use the same pure-Go sqlite driver as production (db_conn/sqlite.go),
	// registered under the driver name "sqlite" below. This keeps tests
	// exercising the exact driver behavior (e.g. FOR UPDATE handling) that
	// production relies on. See docs/AUDIT.md A7.
	_ "modernc.org/sqlite"
)

// testDBFiles tracks the temp file backing each test DB so CleanupTestDB can
// remove it. Keyed by *gorm.DB pointer.
var (
	testDBFilesMu sync.Mutex
	testDBFiles   = map[*gorm.DB]string{}
)

// sqliteBusyTimeoutMillis mirrors db_conn.sqliteBusyTimeoutMillis (kept as a
// separate constant since that one is unexported).
const sqliteBusyTimeoutMillis = 5000

// SetupTestDB initializes a SQLite database for testing, backed by a unique
// temp file (not ":memory:").
//
// SQLite's ":memory:" DSN gives each *distinct connection* its own private,
// empty database. GORM's connection pool can and does hand out more than one
// connection per *gorm.DB (e.g. TransactionService.Transfer opens a tx via
// Begin() on one connection while its repositories run plain, non-tx queries
// that borrow a second connection from the same pool) — with ":memory:" that
// second connection sees an empty database and queries spuriously fail with
// "record not found". A real file on disk doesn't have this problem: every
// connection in the pool opens the same file and sees the same data, exactly
// like the production SQLite path in db_conn/sqlite.go. See docs/AUDIT.md M3.
//
// (Capping the pool at a single connection was considered and rejected: the
// mixed tx/non-tx query pattern above needs at least two connections
// available, or it self-deadlocks waiting for a connection its own
// transaction is holding.)
func SetupTestDB() *gorm.DB {
	f, err := os.CreateTemp("", "mini_wallet_test_*.db")
	if err != nil {
		log.Fatal("❌ Failed to create temp file for test database:", err)
	}
	dbPath := f.Name()
	f.Close()

	dialector := sqlite.Dialector{
		DSN:        dbPath,
		DriverName: "sqlite",
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to test database:", err)
	}

	// Match production's db_conn/sqlite.go: without this, concurrent
	// transfer tests (multiple goroutines writing at close to the same
	// moment) hit a hard SQLITE_BUSY "database is locked" error instead of
	// one writer briefly waiting for the other.
	if err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", sqliteBusyTimeoutMillis)).Error; err != nil {
		log.Fatal("❌ Failed to set SQLite busy_timeout:", err)
	}

	testDBFilesMu.Lock()
	testDBFiles[db] = dbPath
	testDBFilesMu.Unlock()

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

// CleanupTestDB tears down the test database and removes its backing temp
// file (including any SQLite -journal/-wal/-shm sidecar files).
func CleanupTestDB(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	testDBFilesMu.Lock()
	dbPath, ok := testDBFiles[db]
	if ok {
		delete(testDBFiles, db)
	}
	testDBFilesMu.Unlock()

	if ok {
		for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
			os.Remove(dbPath + suffix)
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
