package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCurrencies_IncludesMaxTransferAmount and
// TestGetCurrency_IncludesMaxTransferAmount are the regression tests for
// docs/BACKEND_PREP_PLAN.md 2.2: the frontend needs the effective
// per-transfer limit to validate client-side before submitting, since
// MAX_TRANSFER_AMOUNT is otherwise only known to the backend. Neither test
// calls config.LoadConfig(), so config.Config stays nil and
// services.MaxTransferAmount() falls back to DefaultMaxTransferAmount
// ("1000000") - that's the value asserted against below.
func TestGetCurrencies_IncludesMaxTransferAmount(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)
	test.CreateTestCurrency(db, "USDT")

	h := NewCurrencyHandler(services.NewCurrencyService(repositories.NewCurrencyRepository()))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/currencies", h.GetCurrencies)

	req := httptest.NewRequest(http.MethodGet, "/currencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp []models.CurrencyResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp)
	assert.True(t, resp[0].MaxTransferAmount.Equal(decimal.RequireFromString(services.DefaultMaxTransferAmount)),
		"max_transfer_amount=%s", resp[0].MaxTransferAmount)
}

func TestGetCurrency_IncludesMaxTransferAmount(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)
	currency := test.CreateTestCurrency(db, "USDT")

	h := NewCurrencyHandler(services.NewCurrencyService(repositories.NewCurrencyRepository()))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/currencies/:id", h.GetCurrency)

	req := httptest.NewRequest(http.MethodGet, "/currencies/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp models.CurrencyResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, currency.ID, resp.ID)
	assert.True(t, resp.MaxTransferAmount.Equal(decimal.RequireFromString(services.DefaultMaxTransferAmount)),
		"max_transfer_amount=%s", resp.MaxTransferAmount)
}
