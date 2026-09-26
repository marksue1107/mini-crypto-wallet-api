package test

import (
	"context"
	"strings"
	"sync"
	"testing"

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

// TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers is a regression
// test for docs/AUDIT.md M1: Transfer() used to lock the "from" wallet and
// then the "to" wallet, in caller-argument order. Two concurrent transfers
// in opposite directions (Alice->Bob and Bob->Alice) would then each hold
// one row lock while waiting for the other - a textbook deadlock. This only
// manifests with real row-level locking, which SQLite doesn't have (see
// docs/AUDIT.md M2), so it needs a real Postgres - hence testcontainers.
func TestTransfer_NoDeadlock_BidirectionalConcurrentTransfers(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker daemon not reachable; skipping testcontainers-based deadlock regression test")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("mini_wallet_test_deadlock"),
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
	currencyRepo := repositories.NewCurrencyRepository()
	txService := services.NewTransactionService(walletRepo, txRepo, currencyRepo, nil)

	currency := &models.Currency{Code: "USDT", Name: "Tether", Symbol: "$", Decimals: 8, IsActive: true}
	require.NoError(t, db.Create(currency).Error)
	alice := &models.User{Username: "alice_deadlock", Email: "alice_deadlock@example.com", Password: "hash"}
	require.NoError(t, db.Create(alice).Error)
	bob := &models.User{Username: "bob_deadlock", Email: "bob_deadlock@example.com", Password: "hash"}
	require.NoError(t, db.Create(bob).Error)

	require.NoError(t, walletRepo.CreateWallet(&models.Wallet{UserID: alice.ID, CurrencyID: currency.ID, Balance: decimal.NewFromInt(1000)}))
	require.NoError(t, walletRepo.CreateWallet(&models.Wallet{UserID: bob.ID, CurrencyID: currency.ID, Balance: decimal.NewFromInt(1000)}))

	const rounds = 20
	var wg sync.WaitGroup
	errsCh := make(chan error, rounds*2)

	for i := 0; i < rounds; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := txService.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(10))
			errsCh <- err
		}()
		go func() {
			defer wg.Done()
			_, err := txService.Transfer(bob.ID, alice.ID, currency.ID, decimal.NewFromInt(10))
			errsCh <- err
		}()
	}

	wg.Wait()
	close(errsCh)

	for err := range errsCh {
		if err == nil {
			continue
		}
		// Any error other than "insufficient balance" (expected once a
		// wallet's balance runs low from repeated debits) is unexpected -
		// most importantly, no deadlock errors from Postgres.
		lower := strings.ToLower(err.Error())
		require.False(t, strings.Contains(lower, "deadlock"),
			"got a deadlock error - lock ordering in Transfer() is not fixed: %v", err)
		require.Contains(t, lower, "insufficient balance",
			"unexpected transfer error: %v", err)
	}

	aliceWallet, err := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	require.NoError(t, err)
	bobWallet, err := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	require.NoError(t, err)

	require.True(t, aliceWallet.Balance.GreaterThanOrEqual(decimal.Zero), "alice balance went negative: %s", aliceWallet.Balance.String())
	require.True(t, bobWallet.Balance.GreaterThanOrEqual(decimal.Zero), "bob balance went negative: %s", bobWallet.Balance.String())

	total := aliceWallet.Balance.Add(bobWallet.Balance)
	require.True(t, total.Equal(decimal.NewFromInt(2000)), "money not conserved: total=%s", total.String())
}
