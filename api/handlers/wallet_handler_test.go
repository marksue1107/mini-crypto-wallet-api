package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newTestWalletHandler wires a real, sqlite-backed WalletService (same
// hermetic-DB pattern as newTestTransactionHandler) so stats reflect what
// actually landed in the database.
func newTestWalletHandler(t *testing.T) (*WalletHandler, *gorm.DB, repositories.ITransaction) {
	t.Helper()
	db := test.SetupTestDB()
	t.Cleanup(func() { test.CleanupTestDB(db) })

	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	walletService := services.NewWalletService(walletRepo, txRepo)

	return NewWalletHandler(walletService), db, txRepo
}

func statsRouter(h *WalletHandler, authedAsUserID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/wallet/:user_id/stats", fakeAuth(authedAsUserID), h.GetWalletStats)
	return r
}

func doGetStats(r *gin.Engine, userID uint, query string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/wallet/%d/stats", userID)
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestGetWalletStats_NoTransactions_AllZero is the regression test for
// docs/BACKEND_PREP_PLAN.md 2.1's "無交易時全為零" case.
func TestGetWalletStats_NoTransactions_AllZero(t *testing.T) {
	h, db, _ := newTestWalletHandler(t)
	alice := test.CreateTestUser(db, "alice")

	r := statsRouter(h, alice.ID)
	w := doGetStats(r, alice.ID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp models.WalletStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "24h", resp.Window)
	assert.Equal(t, int64(0), resp.TransactionCount)
	assert.True(t, resp.TotalSent.IsZero(), "total_sent=%s", resp.TotalSent)
	assert.True(t, resp.TotalReceived.IsZero(), "total_received=%s", resp.TotalReceived)
	assert.True(t, resp.BalanceChange.IsZero(), "balance_change=%s", resp.BalanceChange)
}

// TestGetWalletStats_MixedSentReceived_CorrectValues is the regression test
// for docs/BACKEND_PREP_PLAN.md 2.1's "有收有支時數值正確" case.
func TestGetWalletStats_MixedSentReceived_CorrectValues(t *testing.T) {
	h, db, txRepo := newTestWalletHandler(t)
	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 1000)

	walletRepo := repositories.NewWalletRepository()
	txService := services.NewTransactionService(walletRepo, txRepo, repositories.NewCurrencyRepository(), nil)

	_, err := txService.Transfer(alice.ID, bob.ID, currency.ID, decimal.NewFromInt(100)) // alice sends 100
	require.NoError(t, err)
	_, err = txService.Transfer(bob.ID, alice.ID, currency.ID, decimal.NewFromInt(30)) // alice receives 30
	require.NoError(t, err)

	r := statsRouter(h, alice.ID)
	w := doGetStats(r, alice.ID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp models.WalletStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.TransactionCount)
	assert.True(t, resp.TotalSent.Equal(decimal.NewFromInt(100)), "total_sent=%s", resp.TotalSent)
	assert.True(t, resp.TotalReceived.Equal(decimal.NewFromInt(30)), "total_received=%s", resp.TotalReceived)
	assert.True(t, resp.BalanceChange.Equal(decimal.NewFromInt(-70)), "balance_change=%s", resp.BalanceChange)
}

// TestGetWalletStats_ExcludesTransactionsOlderThan24h is the regression
// test for docs/BACKEND_PREP_PLAN.md 2.1's "超過 24 小時的交易不納入" case.
func TestGetWalletStats_ExcludesTransactionsOlderThan24h(t *testing.T) {
	h, db, _ := newTestWalletHandler(t)
	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")

	// Insert directly (bypassing TransactionService.Transfer, which always
	// stamps CreatedAt as "now") so created_at can be backdated.
	oldTx := &models.Transaction{
		FromUserID: alice.ID,
		ToUserID:   bob.ID,
		CurrencyID: currency.ID,
		Amount:     decimal.NewFromInt(500),
		Status:     "completed",
	}
	oldTx.Hash = oldTx.GenerateHash()
	oldTx.Signature = oldTx.GenerateSignature()
	require.NoError(t, db.Create(oldTx).Error)
	require.NoError(t, db.Model(oldTx).UpdateColumn("created_at", time.Now().Add(-48*time.Hour)).Error)

	r := statsRouter(h, alice.ID)
	w := doGetStats(r, alice.ID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp models.WalletStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(0), resp.TransactionCount)
	assert.True(t, resp.TotalSent.IsZero(), "total_sent=%s", resp.TotalSent)
}

// TestGetWalletStats_Fail_OtherUsersID_Returns403 is the regression test
// for docs/BACKEND_PREP_PLAN.md 2.1's "別人的 user_id 回 403" case.
func TestGetWalletStats_Fail_OtherUsersID_Returns403(t *testing.T) {
	h, db, _ := newTestWalletHandler(t)
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")

	r := statsRouter(h, alice.ID) // authed as alice
	w := doGetStats(r, bob.ID, "")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestGetWalletStats_Fail_UnsupportedWindow_Returns400(t *testing.T) {
	h, db, _ := newTestWalletHandler(t)
	alice := test.CreateTestUser(db, "alice")

	r := statsRouter(h, alice.ID)
	w := doGetStats(r, alice.ID, "window=7d")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var errResp errorResponseBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "INVALID_REQUEST", errResp.Code)
}

func TestGetWalletStats_Fail_NoToken_Returns401(t *testing.T) {
	h, _, _ := newTestWalletHandler(t)

	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("test-secret-at-least-32-characters-long", time.Hour)
	r := gin.New()
	r.GET("/wallet/:user_id/stats", middleware.AuthMiddleware(jwtManager), h.GetWalletStats)

	req := httptest.NewRequest(http.MethodGet, "/wallet/1/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
