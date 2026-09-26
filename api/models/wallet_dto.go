package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// WalletResponse represents the HTTP response for wallet data
// Separated from the Wallet database model to:
// 1. Control exactly what fields are exposed via API
// 2. Prevent database relationships from leaking into HTTP layer
// 3. Allow API contract evolution without database changes
type WalletResponse struct {
	ID         uint            `json:"id" example:"1"`
	UserID     uint            `json:"user_id" example:"1"`
	CurrencyID uint            `json:"currency_id" example:"1"`
	Balance    decimal.Decimal `json:"balance" swaggertype:"number" example:"1000.0"`
	CreatedAt  time.Time       `json:"created_at"`
	// Currency can be added optionally if needed, but not by default
	// to avoid exposing unnecessary database relationships
}

// WalletStatsResponse is the response for GET /wallet/{user_id}/stats. All
// three amount fields are computed by a database aggregate (SUM/COUNT),
// not by loading every matching transaction into the application - see
// TransactionRepository.GetStatsSince.
type WalletStatsResponse struct {
	Window           string          `json:"window" example:"24h"`
	TransactionCount int64           `json:"transaction_count" example:"3"`
	TotalSent        decimal.Decimal `json:"total_sent" swaggertype:"number" example:"150.0"`
	TotalReceived    decimal.Decimal `json:"total_received" swaggertype:"number" example:"50.0"`
	BalanceChange    decimal.Decimal `json:"balance_change" swaggertype:"number" example:"-100.0"`
}

// WalletWithCurrencyResponse includes currency details
// Used when the API caller explicitly needs currency information
type WalletWithCurrencyResponse struct {
	ID         uint            `json:"id" example:"1"`
	UserID     uint            `json:"user_id" example:"1"`
	CurrencyID uint            `json:"currency_id" example:"1"`
	Balance    decimal.Decimal `json:"balance" swaggertype:"number" example:"1000.0"`
	CreatedAt  time.Time       `json:"created_at"`
	Currency   *CurrencyResponse `json:"currency,omitempty"`
}

// ToWalletResponse converts a Wallet model to WalletResponse DTO
func ToWalletResponse(wallet *Wallet) *WalletResponse {
	return &WalletResponse{
		ID:         wallet.ID,
		UserID:     wallet.UserID,
		CurrencyID: wallet.CurrencyID,
		Balance:    wallet.Balance,
		CreatedAt:  wallet.CreatedAt,
	}
}

// ToWalletWithCurrencyResponse converts a Wallet model with Currency to DTO
func ToWalletWithCurrencyResponse(wallet *Wallet) *WalletWithCurrencyResponse {
	response := &WalletWithCurrencyResponse{
		ID:         wallet.ID,
		UserID:     wallet.UserID,
		CurrencyID: wallet.CurrencyID,
		Balance:    wallet.Balance,
		CreatedAt:  wallet.CreatedAt,
	}

	// Only include currency if it's loaded
	if wallet.Currency.ID != 0 {
		response.Currency = ToCurrencyResponse(&wallet.Currency)
	}

	return response
}
