package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// CurrencyResponse represents the HTTP response for currency data
// Separated from the Currency database model to control API contract
type CurrencyResponse struct {
	ID        uint      `json:"id" example:"1"`
	Code      string    `json:"code" example:"USDT"`
	Name      string    `json:"name" example:"Tether"`
	Symbol    string    `json:"symbol" example:"$"`
	Decimals  int       `json:"decimals" example:"6"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// MaxTransferAmount is the effective per-transfer limit enforced by
	// POST /wallet/transfer (MAX_TRANSFER_AMOUNT, or its default - see
	// services.MaxTransferAmount). It isn't a per-currency database column;
	// every currency currently reports the same, single global limit. See
	// docs/BACKEND_PREP_PLAN.md 2.2.
	MaxTransferAmount decimal.Decimal `json:"max_transfer_amount" swaggertype:"number" example:"1000000"`
}

// ToCurrencyResponse converts a Currency model to CurrencyResponse DTO.
// maxTransferAmount is threaded in by the caller (see
// services.CurrencyService.MaxTransferAmount) rather than looked up here,
// since it isn't a property of the currency itself.
func ToCurrencyResponse(currency *Currency, maxTransferAmount decimal.Decimal) *CurrencyResponse {
	return &CurrencyResponse{
		ID:                currency.ID,
		Code:              currency.Code,
		Name:              currency.Name,
		Symbol:            currency.Symbol,
		Decimals:          currency.Decimals,
		IsActive:          currency.IsActive,
		CreatedAt:         currency.CreatedAt,
		UpdatedAt:         currency.UpdatedAt,
		MaxTransferAmount: maxTransferAmount,
	}
}

// ToCurrencyResponses converts a slice of Currency models to DTOs
func ToCurrencyResponses(currencies []Currency, maxTransferAmount decimal.Decimal) []CurrencyResponse {
	responses := make([]CurrencyResponse, len(currencies))
	for i, currency := range currencies {
		responses[i] = *ToCurrencyResponse(&currency, maxTransferAmount)
	}
	return responses
}
