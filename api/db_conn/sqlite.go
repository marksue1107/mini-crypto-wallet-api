package db_conn

import (
	"fmt"
	"gorm.io/gorm/logger"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// sqliteBusyTimeoutMillis controls how long a connection will wait for a
// write lock held by another connection before giving up with
// SQLITE_BUSY, instead of failing immediately. SQLite only ever has one
// writer at a time; without this, two callers writing at close to the same
// moment get a hard "database is locked" error rather than one of them
// just waiting briefly for the other to finish.
const sqliteBusyTimeoutMillis = 5000

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

	if err := Conn_DB.MasterDB.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", sqliteBusyTimeoutMillis)).Error; err != nil {
		log.Fatal("❌ Failed to set SQLite busy_timeout:", err)
	}

	log.Println("✅ SQLite connected")
	// SQLite mode is a dev/demo convenience, not a production-safe
	// configuration for this service: GORM's sqlite driver silently drops
	// `FOR UPDATE` (SQLite has no row-level locking), so the pessimistic
	// locking Transfer() relies on to prevent lost updates under concurrent
	// transfers does nothing here. busy_timeout above only prevents hard
	// "database is locked" errors during brief write contention - it does
	// not make concurrent transfers correct. See docs/AUDIT.md M2.
	log.Println("⚠️  SQLite mode does not provide real concurrency safety for wallet transfers (no row-level locking). Use db_driver: postgres for anything beyond single-developer local testing.")
}
