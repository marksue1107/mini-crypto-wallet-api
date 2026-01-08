package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// BalanceHistory 餘額變動歷史記錄
type BalanceHistory struct {
	ID            uint            `json:"id" gorm:"primarykey"`
	UserID        uint            `json:"user_id" gorm:"index"`
	WalletID      uint            `json:"wallet_id" gorm:"index"`
	TransactionID uint            `json:"transaction_id" gorm:"index"`
	ChangeType   string           `json:"change_type"` // credit, debit
	Amount        decimal.Decimal `json:"amount" gorm:"type:decimal(20,8)"`
	BalanceBefore decimal.Decimal `json:"balance_before" gorm:"type:decimal(20,8)"`
	BalanceAfter  decimal.Decimal `json:"balance_after" gorm:"type:decimal(20,8)"`
	CreatedAt     time.Time       `json:"created_at"`
}
