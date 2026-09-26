package services

import (
	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/repositories"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestSimpleTransfer is a basic transfer test that follows the existing
// pattern used by transaction_service_test.go: a hermetic, temp-file-backed
// SQLite database per test (via test.SetupTestDB/CleanupTestDB), not the
// same fixed-path "real" database db_conn.InitDatabase() opens. The
// previous version of this test called db_conn.InitDatabase() directly
// against the shared relative-path mini_wallet.db file and never cleaned it
// up, so schema changes made by a later run's AutoMigrate (e.g. adding a
// NOT NULL column) could fail against a stale copy of that file left over
// on disk from a previous run - a real repro of that is what motivated this
// rewrite (see docs/BACKEND_PREP_PLAN.md batch 1 verification notes).
func TestSimpleTransfer(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice_test")
	bob := test.CreateTestUser(db, "bob_test")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute transfer
	_, err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(100))

	// Assert
	assert.NoError(t, err, "Transfer should succeed")

	// Verify balances
	aliceUpdated, _ := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	bobUpdated, _ := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	assert.Equal(t, "900", aliceUpdated.Balance.String(), "Alice balance should be 900")
	assert.Equal(t, "100", bobUpdated.Balance.String(), "Bob balance should be 100")

	// Verify transaction created
	txs, _ := txRepo.GetTransactionsByUserID(alice.ID)
	assert.GreaterOrEqual(t, len(txs), 1, "At least one transaction should exist")
}
