package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID         uint            `json:"id" example:"1" gorm:"primarykey"`
	UserID     uint            `json:"user_id" example:"1" gorm:"index"`
	CurrencyID uint            `json:"currency_id" example:"1" gorm:"index"`
	Balance    decimal.Decimal `json:"balance" gorm:"type:decimal(20,8)" swaggertype:"number" example:"1000.0"`
	CreatedAt  time.Time       `json:"created_at"`
	Currency   Currency        `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
}
