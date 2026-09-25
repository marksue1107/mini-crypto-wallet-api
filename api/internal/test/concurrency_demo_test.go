package test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"testing"
	"time"

	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// dockerAvailable does a cheap, fast check for a reachable Docker daemon so
// this test can skip cleanly in environments without Docker instead of
// hanging in testcontainers' retry loop or (as the previous version of this
// test did) calling log.Fatal via db_conn.InitDatabase() when no local
// Postgres was running. See docs/AUDIT.md Q10.
func dockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "docker", "info").Run() == nil
}

// TestConcurrentTransfers demonstrates *why* Transfer() takes a row lock by
// running the same concurrent transfer twice against a disposable, real
// PostgreSQL instance: once through the naive/unsafe path (no lock) and once
// through the real, locked TransactionService.Transfer(). It spins up
// Postgres via testcontainers-go so it doesn't depend on a Postgres instance
// already running on the developer's machine, and needs real row-level
// locking semantics (SQLite doesn't support FOR UPDATE - see docs/AUDIT.md
// M2), which is why this test can't just use the SQLite test helpers in
// test_helpers.go.
func TestConcurrentTransfers(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker daemon not reachable; skipping testcontainers-based concurrency demo")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("mini_wallet_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	}()

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Currency{},
		&models.Wallet{},
		&models.Transaction{},
		&models.BalanceHistory{},
	))

	db_conn.Conn_DB.MasterDB = db

	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	txService := services.NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	currency := &models.Currency{Code: "USDT", Name: "Tether", Symbol: "$", Decimals: 8, IsActive: true}
	require.NoError(t, db.Create(currency).Error)
	alice := &models.User{Username: "alice_demo", Email: "alice_demo@example.com", Password: "hash"}
	require.NoError(t, db.Create(alice).Error)
	bob := &models.User{Username: "bob_demo", Email: "bob_demo@example.com", Password: "hash"}
	require.NoError(t, db.Create(bob).Error)

	fmt.Println("=== 測試未加鎖交易 ===")
	resetWallets(t, db, walletRepo, alice.ID, bob.ID, currency.ID)
	simulateConcurrentTransfers(t, txService, walletRepo, alice.ID, bob.ID, currency.ID, false)

	fmt.Println("=== 測試加鎖交易 ===")
	resetWallets(t, db, walletRepo, alice.ID, bob.ID, currency.ID)
	simulateConcurrentTransfers(t, txService, walletRepo, alice.ID, bob.ID, currency.ID, true)
}

// simulateConcurrentTransfers fires two concurrent transfers of the same
// amount from fromID to toID, with a starting balance exactly equal to one
// transfer's worth. Under the naive/unlocked path both goroutines can read
// the same starting balance before either writes, so both "succeed" and the
// balance goes negative (money created from nothing). Under the real,
// row-locked Transfer(), one goroutine wins the row lock, the other is
// blocked until it commits, then correctly fails with insufficient balance.
func simulateConcurrentTransfers(t *testing.T, service *services.TransactionService, walletRepo repositories.IWallet, fromID, toID, currencyID uint, useLock bool) {
	var wg sync.WaitGroup
	wg.Add(2)

	amount := decimal.NewFromInt(800) // starting balance is 800; only one transfer of 800 can legitimately succeed

	for i := 1; i <= 2; i++ {
		go func(id int) {
			defer wg.Done()
			fmt.Println("🔄 正在執行轉帳 (useLock =", useLock, ")")
			var err error
			if useLock {
				err = service.Transfer(fromID, toID, currencyID, amount)
			} else {
				err = simulateUnsafeTransfer(walletRepo, fromID, toID, currencyID, amount)
			}
			if err != nil {
				log.Printf("🔴 Transfer %d failed: %v", id, err)
			} else {
				log.Printf("🟢 Transfer %d success\n", id)
			}
		}(i)
	}

	wg.Wait()

	from, fromErr := walletRepo.GetWalletByUserIDAndCurrency(fromID, currencyID)
	require.NoError(t, fromErr)
	to, toErr := walletRepo.GetWalletByUserIDAndCurrency(toID, currencyID)
	require.NoError(t, toErr)

	fmt.Printf("📊 A 最終餘額: %s\n", from.Balance.String())
	fmt.Printf("📊 B 最終餘額: %s\n", to.Balance.String())

	if useLock {
		// Money must be conserved and the sender's balance must never go
		// negative when the real, locked Transfer() is used.
		require.True(t, from.Balance.GreaterThanOrEqual(decimal.Zero),
			"locked transfer allowed balance to go negative: %s", from.Balance.String())
		total := from.Balance.Add(to.Balance)
		require.True(t, total.Equal(decimal.NewFromInt(800)),
			"locked transfer did not conserve money: total=%s", total.String())
	}
	// The unlocked path is only run to demonstrate the race for anyone
	// reading test output; it is expected to be able to lose money (that's
	// the whole point), so it makes no correctness assertions.
}

// simulateUnsafeTransfer reproduces the classic read-modify-write race
// (read balance, compute new balance, write it back, all without a
// transaction or row lock) that TransactionService.Transfer() avoids by
// using GetWalletByUserIDAndCurrencyWithTx's row lock. This is deliberately unsafe and
// exists only to demonstrate the race in TestConcurrentTransfers above — it
// must never be used outside of this test file. See docs/AUDIT.md A6 (this
// used to live in production code as TransactionService.TransferWithLockOption).
func simulateUnsafeTransfer(walletRepo repositories.IWallet, fromID, toID, currencyID uint, amount decimal.Decimal) error {
	fromWallet, err := walletRepo.GetWalletByUserIDAndCurrency(fromID, currencyID)
	if err != nil {
		return err
	}
	toWallet, err := walletRepo.GetWalletByUserIDAndCurrency(toID, currencyID)
	if err != nil {
		return err
	}
	if fromWallet.Balance.LessThan(amount) {
		return errors.New("insufficient balance")
	}

	fromWallet.Balance = fromWallet.Balance.Sub(amount)
	toWallet.Balance = toWallet.Balance.Add(amount)

	if err := walletRepo.UpdateWallet(fromWallet); err != nil {
		return err
	}
	return walletRepo.UpdateWallet(toWallet)
}

// resetWallets clears the wallets table and gives fromID/toID a clean
// starting balance for the next simulateConcurrentTransfers run.
func resetWallets(t *testing.T, db *gorm.DB, repo repositories.IWallet, fromID, toID, currencyID uint) {
	require.NoError(t, db.Exec("DELETE FROM wallets").Error)
	require.NoError(t, repo.CreateWallet(&models.Wallet{UserID: fromID, CurrencyID: currencyID, Balance: decimal.NewFromInt(800)}))
	require.NoError(t, repo.CreateWallet(&models.Wallet{UserID: toID, CurrencyID: currencyID, Balance: decimal.Zero}))
}
