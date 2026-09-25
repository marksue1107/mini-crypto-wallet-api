package services

import (
	"testing"

	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateUser_Fail_NoCurrency_DoesNotOrphanUser is a regression test for
// docs/AUDIT_REMEDIATION_PLAN.md finding N3: CreateUser used to insert the
// User row through the repository's non-transactional (autocommit)
// connection, outside of the transaction wrapping wallet creation. If
// wallet creation then failed, tx.Rollback() had nothing to undo on the
// User side - leaving a login-able account with no wallet forever. Here we
// force wallet creation to fail by leaving zero currencies in the database.
func TestCreateUser_Fail_NoCurrency_DoesNotOrphanUser(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)
	// Deliberately create zero currencies so CreateUser hits the
	// "no currency available" branch.

	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	currencyRepo := repositories.NewCurrencyRepository()
	service := NewUserService(userRepo, walletRepo, currencyRepo)

	req := &models.UserCreateRequest{Username: "orphan", Email: "orphan@example.com", Password: "password123"}
	user, err := service.CreateUser(req)
	require.Error(t, err)
	require.Nil(t, user)

	_, err = userRepo.GetUserByUsername("orphan")
	assert.Error(t, err, "CreateUser failed but left a User row behind (orphaned account with no wallet)")
}

// TestCreateUser_Success creates a user end-to-end and confirms a matching
// wallet was created atomically alongside it.
func TestCreateUser_Success(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	currency := test.CreateTestCurrency(db, "USDT")

	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	currencyRepo := repositories.NewCurrencyRepository()
	service := NewUserService(userRepo, walletRepo, currencyRepo)

	req := &models.UserCreateRequest{Username: "alice", Email: "alice@example.com", Password: "password123"}
	user, err := service.CreateUser(req)
	require.NoError(t, err)
	require.NotNil(t, user)

	wallet, err := walletRepo.GetWalletByUserIDAndCurrency(user.ID, currency.ID)
	require.NoError(t, err)
	assert.Equal(t, "1000", wallet.Balance.String())
}
