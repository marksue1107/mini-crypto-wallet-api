package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Wallet represents the database model for user wallet data
// Pure GORM model - no JSON/binding tags for HTTP layer separation
type Wallet struct {
	ID         uint            `gorm:"primarykey"`
	UserID     uint            `gorm:"index;not null"`
	CurrencyID uint            `gorm:"index;not null"`
	Balance    decimal.Decimal `gorm:"type:decimal(20,8);not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Relationships - only for GORM, not exposed directly via HTTP
	Currency Currency `gorm:"foreignKey:CurrencyID"`
	User     User     `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (Wallet) TableName() string {
	return "wallets"
}
