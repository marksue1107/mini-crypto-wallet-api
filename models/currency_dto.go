package models

import "time"

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
}

// ToCurrencyResponse converts a Currency model to CurrencyResponse DTO
func ToCurrencyResponse(currency *Currency) *CurrencyResponse {
	return &CurrencyResponse{
		ID:        currency.ID,
		Code:      currency.Code,
		Name:      currency.Name,
		Symbol:    currency.Symbol,
		Decimals:  currency.Decimals,
		IsActive:  currency.IsActive,
		CreatedAt: currency.CreatedAt,
		UpdatedAt: currency.UpdatedAt,
	}
}

// ToCurrencyResponses converts a slice of Currency models to DTOs
func ToCurrencyResponses(currencies []Currency) []CurrencyResponse {
	responses := make([]CurrencyResponse, len(currencies))
	for i, currency := range currencies {
		responses[i] = *ToCurrencyResponse(&currency)
	}
	return responses
}
