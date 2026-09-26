package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeAuth sets user_id in the gin context to simulate an authenticated
// request. middleware.RequireUserID only reads this context value, so
// these handler tests don't need a real JWT/AuthMiddleware in front of them.
func fakeAuth(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// newTestTransactionHandler wires a real, sqlite-backed TransactionService
// (same pattern as newTestUserHandler) so response bodies can be checked
// against what actually landed in the database, not a mock's return value.
func newTestTransactionHandler(t *testing.T) (*TransactionHandler, *gorm.DB, repositories.ITransaction) {
	t.Helper()
	db := test.SetupTestDB()
	t.Cleanup(func() { test.CleanupTestDB(db) })

	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	currencyRepo := repositories.NewCurrencyRepository()
	txService := services.NewTransactionService(walletRepo, txRepo, currencyRepo, nil)

	return NewTransactionHandler(txService), db, txRepo
}

func doTransfer(r *gin.Engine, fromUserID, toUserID, currencyID uint, amount string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{
		"from_user_id": fromUserID,
		"to_user_id":   toUserID,
		"currency_id":  currencyID,
		"amount":       amount,
	})
	req := httptest.NewRequest(http.MethodPost, "/wallet/transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func transferRouter(h *TransactionHandler, authedAsUserID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/wallet/transfer", fakeAuth(authedAsUserID), h.Transfer)
	return r
}

type transactionResponseBody struct {
	ID         uint   `json:"id"`
	FromUserID uint   `json:"from_user_id"`
	ToUserID   uint   `json:"to_user_id"`
	CurrencyID uint   `json:"currency_id"`
	Amount     string `json:"amount"`
	Hash       string `json:"hash"`
	Signature  string `json:"signature"`
	Status     string `json:"status"`
}

type errorResponseBody struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TestTransfer_Success_ResponseMatchesCreatedTransaction is the regression
// test for docs/BACKEND_PREP_PLAN.md 1.1: the frontend needs the created
// transaction's hash (to link to the Explorer page) directly in the
// transfer response, instead of having to separately query for it.
func TestTransfer_Success_ResponseMatchesCreatedTransaction(t *testing.T) {
	h, db, txRepo := newTestTransactionHandler(t)
	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	r := transferRouter(h, alice.ID)
	w := doTransfer(r, alice.ID, bob.ID, currency.ID, "100")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp transactionResponseBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.Hash)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, alice.ID, resp.FromUserID)
	assert.Equal(t, bob.ID, resp.ToUserID)
	assert.Equal(t, currency.ID, resp.CurrencyID)
	assert.Equal(t, "100", resp.Amount)

	dbTx, err := txRepo.FindByHash(resp.Hash)
	require.NoError(t, err)
	assert.Equal(t, dbTx.ID, resp.ID)
	assert.Equal(t, dbTx.FromUserID, resp.FromUserID)
	assert.Equal(t, dbTx.ToUserID, resp.ToUserID)
	assert.Equal(t, dbTx.CurrencyID, resp.CurrencyID)
	assert.Equal(t, dbTx.Signature, resp.Signature)
}

// TestGetTransactions_And_GetTxByHash_IncludeCurrencyID is the regression
// test for docs/BACKEND_PREP_PLAN.md 1.4: TransactionResponse must carry
// currency_id on every endpoint that returns it, not just the transfer
// response covered above.
func TestGetTransactions_And_GetTxByHash_IncludeCurrencyID(t *testing.T) {
	h, db, _ := newTestTransactionHandler(t)
	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	transferR := transferRouter(h, alice.ID)
	transferW := doTransfer(transferR, alice.ID, bob.ID, currency.ID, "42")
	require.Equal(t, http.StatusOK, transferW.Code, transferW.Body.String())
	var created transactionResponseBody
	require.NoError(t, json.Unmarshal(transferW.Body.Bytes(), &created))

	gin.SetMode(gin.TestMode)
	queryR := gin.New()
	queryR.GET("/tx/:hash", h.GetTxByHash)
	queryR.GET("/transactions/:user_id", fakeAuth(alice.ID), h.GetTransactions)

	t.Run("GET /tx/:hash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tx/"+created.Hash, nil)
		w := httptest.NewRecorder()
		queryR.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp models.TransactionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, currency.ID, resp.CurrencyID)
	})

	t.Run("GET /transactions/:user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d?page=1&page_size=10", alice.ID), nil)
		w := httptest.NewRecorder()
		queryR.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp models.TransactionListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotEmpty(t, resp.Data)
		assert.Equal(t, currency.ID, resp.Data[0].CurrencyID)
	})
}
