package models

import "github.com/shopspring/decimal"

type TransferRequest struct {
	FromUserID uint            `json:"from_user_id" binding:"required" example:"1"`
	ToUserID   uint            `json:"to_user_id" binding:"required" example:"2"`
	CurrencyID uint            `json:"currency_id" binding:"required" example:"1"` // 幣種 ID
	Amount     decimal.Decimal `json:"amount" binding:"required" swaggertype:"number" example:"150.0"`
}
