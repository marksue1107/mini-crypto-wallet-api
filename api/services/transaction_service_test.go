package services

import (
	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransfer_Success_ValidTransfer tests the happy path for a valid transfer
func TestTransfer_Success_ValidTransfer(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(100))

	// Assert - require, not assert: a failed transfer means the wallet
	// lookups below would dereference nil pointers.
	require.NoError(t, err)

	// Verify balances
	aliceWallet, err := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	require.NoError(t, err)
	bobWallet, err := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	require.NoError(t, err)
	assert.Equal(t, "900", aliceWallet.Balance.String())
	assert.Equal(t, "100", bobWallet.Balance.String())

	// Verify transaction created
	txs, _ := txRepo.GetTransactionsByUserID(alice.ID)
	assert.Len(t, txs, 1)
	assert.Equal(t, "completed", txs[0].Status)
	assert.NotEmpty(t, txs[0].Hash)
	assert.NotEmpty(t, txs[0].Signature)
}

// TestTransfer_Success_BalanceHistoryRecorded verifies audit trail is created
func TestTransfer_Success_BalanceHistoryRecorded(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 500)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	balanceHistoryRepo := repositories.NewBalanceHistoryRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(200))
	require.NoError(t, err)

	// Verify balance history for Alice (debit)
	var aliceHistory []models.BalanceHistory
	db.Where("user_id = ?", alice.ID).Find(&aliceHistory)
	require.Len(t, aliceHistory, 1)
	assert.Equal(t, "debit", aliceHistory[0].ChangeType)
	assert.Equal(t, "200", aliceHistory[0].Amount.String())
	assert.Equal(t, "1000", aliceHistory[0].BalanceBefore.String())
	assert.Equal(t, "800", aliceHistory[0].BalanceAfter.String())

	// Verify balance history for Bob (credit)
	var bobHistory []models.BalanceHistory
	db.Where("user_id = ?", bob.ID).Find(&bobHistory)
	require.Len(t, bobHistory, 1)
	assert.Equal(t, "credit", bobHistory[0].ChangeType)
	assert.Equal(t, "200", bobHistory[0].Amount.String())
	assert.Equal(t, "500", bobHistory[0].BalanceBefore.String())
	assert.Equal(t, "700", bobHistory[0].BalanceAfter.String())

	// Verify both histories link to same transaction
	assert.Equal(t, aliceHistory[0].TransactionID, bobHistory[0].TransactionID)

	// Ensure no unused variable warning
	_ = balanceHistoryRepo
}

// TestTransfer_Success_TransactionHashGenerated verifies hash and signature generation
func TestTransfer_Success_TransactionHashGenerated(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(50))
	require.NoError(t, err)

	// Verify transaction has hash and signature
	_, err = txRepo.FindByHash("")
	assert.Error(t, err) // Should not find empty hash

	txs, err := txRepo.GetTransactionsByUserID(alice.ID)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.NotEmpty(t, txs[0].Hash)
	assert.NotEmpty(t, txs[0].Signature)
	assert.Len(t, txs[0].Hash, 64) // SHA256 produces 64 hex characters

	// Verify we can query by hash
	foundTx, err := txRepo.FindByHash(txs[0].Hash)
	require.NoError(t, err)
	assert.Equal(t, txs[0].ID, foundTx.ID)
}

// TestTransfer_Fail_DoesNotLeakConnection is a regression test for
// docs/AUDIT_REMEDIATION_PLAN.md finding N2: every early `return err` inside
// Transfer() between tx.Begin() and tx.Commit() used to skip tx.Rollback(),
// leaving the connection checked out of the pool ("idle in transaction")
// forever. Insufficient balance is the most common, entirely non-malicious
// way to hit that code path, so repeating it must not leak connections.
func TestTransfer_Fail_DoesNotLeakConnection(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 100)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(200)) // always exceeds the 100 balance
		require.Error(t, err)
		require.Equal(t, 0, sqlDB.Stats().InUse,
			"Transfer() left a connection checked out of the pool after a failed transfer (leaked, un-rolled-back transaction)")
	}
}

// TestTransfer_Fail_InsufficientBalance verifies transfer fails when balance is insufficient
func TestTransfer_Fail_InsufficientBalance(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 100) // Only 100 balance
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try to transfer 200 (more than balance)
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(200))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance")

	// Verify balances unchanged
	aliceWallet, _ := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	bobWallet, _ := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	assert.Equal(t, "100", aliceWallet.Balance.String())
	assert.Equal(t, "0", bobWallet.Balance.String())

	// Verify no transaction created
	txs, _ := txRepo.GetTransactionsByUserID(alice.ID)
	assert.Len(t, txs, 0)
}

// TestTransfer_Fail_SameAccountTransfer verifies transfer to same account is rejected
func TestTransfer_Fail_SameAccountTransfer(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try to transfer to same account
	err := service.Transfer(alice.ID, alice.ID, currency.ID, decimal.NewFromInt(100))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transfer to the same account")

	// Verify balance unchanged
	aliceWallet, _ := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	assert.Equal(t, "1000", aliceWallet.Balance.String())
}

// TestTransfer_Fail_NegativeAmount verifies negative amount is rejected
func TestTransfer_Fail_NegativeAmount(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try negative amount
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(-100))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

// TestTransfer_Fail_ZeroAmount verifies zero amount is rejected
func TestTransfer_Fail_ZeroAmount(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try zero amount
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.Zero)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
}

// TestTransfer_Fail_TooManyDecimalPlaces verifies an amount with more decimal
// places than the currency supports is rejected. See docs/AUDIT.md M6.
func TestTransfer_Fail_TooManyDecimalPlaces(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT") // Decimals: 8
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - 9 decimal places, currency only supports 8
	amount, err := decimal.NewFromString("1.123456789")
	require.NoError(t, err)
	err = service.Transfer(alice.ID, bob.ID, currency.ID, amount)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decimal places")

	// Verify balance unchanged
	aliceWallet, err := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	require.NoError(t, err)
	assert.Equal(t, "1000", aliceWallet.Balance.String())
}

// TestTransfer_Fail_ExceedsMaxAmount verifies an amount over the configured
// maximum transfer limit is rejected. See docs/AUDIT.md M6.
func TestTransfer_Fail_ExceedsMaxAmount(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	// Give alice enough balance that, if the limit check were missing,
	// the transfer would otherwise succeed.
	test.CreateTestWalletWithDecimal(db, alice.ID, currency.ID, decimal.RequireFromString("100000000"))
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - exceeds DefaultMaxTransferAmount (1,000,000)
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(2_000_000))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum transfer limit")
}

// TestTransfer_Fail_FromWalletNotFound verifies error when source wallet doesn't exist
func TestTransfer_Fail_FromWalletNotFound(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try to transfer from non-existent user ID 999
	err := service.Transfer(999, bob.ID, currency.ID, decimal.NewFromInt(100))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}

// TestTransfer_Fail_ToWalletNotFound verifies error when destination wallet doesn't exist
func TestTransfer_Fail_ToWalletNotFound(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try to transfer to non-existent user ID 999
	err := service.Transfer(alice.ID, 999, currency.ID, decimal.NewFromInt(100))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}

// TestTransfer_Fail_InvalidCurrency verifies error when currency doesn't match
func TestTransfer_Fail_InvalidCurrency(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	usdtCurrency := test.CreateTestCurrency(db, "USDT")
	btcCurrency := test.CreateTestCurrency(db, "BTC")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, usdtCurrency.ID, 1000) // Alice has USDT wallet
	test.CreateTestWallet(db, bob.ID, btcCurrency.ID, 0)       // Bob has BTC wallet

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - try to transfer with mismatched currency (Alice USDT → Bob BTC)
	err := service.Transfer(alice.ID, bob.ID, usdtCurrency.ID, decimal.NewFromInt(100))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}

// TestTransfer_ConcurrentTransfers_NoRaceCondition verifies row locking prevents race conditions
func TestTransfer_ConcurrentTransfers_NoRaceCondition(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	charlie := test.CreateTestUser(db, "charlie")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000) // Alice starts with 1000
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)
	test.CreateTestWallet(db, charlie.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute - 5 concurrent transfers of 100 each from Alice
	var wg sync.WaitGroup
	var errors []error
	errorsChan := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			var err error
			if index%2 == 0 {
				err = service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(100))
			} else {
				err = service.Transfer(alice.ID, charlie.ID, currency.ID, decimal.NewFromInt(100))
			}
			if err != nil {
				errorsChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	for err := range errorsChan {
		errors = append(errors, err)
	}

	// Assert - At most 5 transfers should succeed (1000 balance / 100 per transfer)
	// Some should fail with insufficient balance
	successfulTransfers := 5 - len(errors)
	assert.LessOrEqual(t, successfulTransfers, 10)

	// Verify final balance is correct (no lost updates)
	aliceWallet, err := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	require.NoError(t, err)
	expectedBalance := 1000 - (successfulTransfers * 100)
	assert.Equal(t, decimal.NewFromInt(int64(expectedBalance)).String(), aliceWallet.Balance.String())

	// Verify sum of all balances equals initial total (conservation of money)
	bobWallet, err := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	require.NoError(t, err)
	charlieWallet, err := walletRepo.GetWalletByUserIDAndCurrency(charlie.ID, currency.ID)
	require.NoError(t, err)
	totalBalance := aliceWallet.Balance.Add(bobWallet.Balance).Add(charlieWallet.Balance)
	assert.Equal(t, "1000", totalBalance.String(), "Total balance should be conserved")
}

// TestTransfer_MultipleSequential verifies multiple sequential transfers work correctly
func TestTransfer_MultipleSequential(t *testing.T) {
	// Setup
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	charlie := test.CreateTestUser(db, "charlie")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)
	test.CreateTestWallet(db, charlie.ID, currency.ID, 0)

	// Create service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	service := NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	// Execute multiple transfers
	err1 := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(300))
	require.NoError(t, err1)

	err2 := service.Transfer(alice.ID, charlie.ID, currency.ID, decimal.NewFromInt(200))
	require.NoError(t, err2)

	err3 := service.Transfer(bob.ID, charlie.ID, currency.ID, decimal.NewFromInt(100))
	require.NoError(t, err3)

	// Verify final balances
	aliceWallet, err := walletRepo.GetWalletByUserIDAndCurrency(alice.ID, currency.ID)
	require.NoError(t, err)
	bobWallet, err := walletRepo.GetWalletByUserIDAndCurrency(bob.ID, currency.ID)
	require.NoError(t, err)
	charlieWallet, err := walletRepo.GetWalletByUserIDAndCurrency(charlie.ID, currency.ID)
	require.NoError(t, err)

	assert.Equal(t, "500", aliceWallet.Balance.String())   // 1000 - 300 - 200
	assert.Equal(t, "200", bobWallet.Balance.String())     // 0 + 300 - 100
	assert.Equal(t, "300", charlieWallet.Balance.String()) // 0 + 200 + 100

	// Verify transaction count
	aliceTxs, err := txRepo.GetTransactionsByUserID(alice.ID)
	require.NoError(t, err)
	assert.Len(t, aliceTxs, 2) // Alice involved in 2 transactions

	bobTxs, err := txRepo.GetTransactionsByUserID(bob.ID)
	require.NoError(t, err)
	assert.Len(t, bobTxs, 2) // Bob involved in 2 transactions

	charlieTxs, err := txRepo.GetTransactionsByUserID(charlie.ID)
	require.NoError(t, err)
	assert.Len(t, charlieTxs, 2) // Charlie involved in 2 transactions
}
