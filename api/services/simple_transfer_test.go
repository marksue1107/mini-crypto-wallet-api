package services

import (
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestSimpleTransfer is a basic transfer test that follows the existing pattern
func TestSimpleTransfer(t *testing.T) {
	// Initialize config and database like the existing test
	config.LoadConfig()
	db_conn.InitDatabase()

	// Initialize repositories and service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	currencyRepo := repositories.NewCurrencyRepository()
	service := NewTransactionService(walletRepo, txRepo, nil)

	// Create currency
	currency := &models.Currency{
		Code:     "USDT",
		Name:     "Tether",
		Symbol:   "$",
		Decimals: 8,
		IsActive: true,
	}
	db_conn.Conn_DB.MasterDB.FirstOrCreate(currency, models.Currency{Code: "USDT"})

	// Create users
	alice := &models.User{Username: "alice_test", Email: "alice_test@example.com", Password: "hash"}
	bob := &models.User{Username: "bob_test", Email: "bob_test@example.com", Password: "hash"}
	db_conn.Conn_DB.MasterDB.FirstOrCreate(alice, models.User{Username: "alice_test"})
	db_conn.Conn_DB.MasterDB.FirstOrCreate(bob, models.User{Username: "bob_test"})

	// Clean up existing wallets for these users
	db_conn.Conn_DB.MasterDB.Where("user_id IN ?", []uint{alice.ID, bob.ID}).Delete(&models.Wallet{})

	// Create wallets using repository
	aliceWallet := &models.Wallet{UserID: alice.ID, CurrencyID: currency.ID, Balance: decimal.NewFromInt(1000)}
	bobWallet := &models.Wallet{UserID: bob.ID, CurrencyID: currency.ID, Balance: decimal.Zero}
	walletRepo.CreateWallet(aliceWallet)
	walletRepo.CreateWallet(bobWallet)

	// Execute transfer
	err := service.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(100))

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

	// Cleanup
	db_conn.Conn_DB.MasterDB.Where("user_id IN ?", []uint{alice.ID, bob.ID}).Delete(&models.Wallet{})
	db_conn.Conn_DB.MasterDB.Delete(alice)
	db_conn.Conn_DB.MasterDB.Delete(bob)

	// Ensure no unused variable warning
	_ = currencyRepo
}
