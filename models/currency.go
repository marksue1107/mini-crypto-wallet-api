package models

import "time"

// Currency represents the database model for currency/cryptocurrency data
// Pure GORM model - no JSON/binding tags for HTTP layer separation
type Currency struct {
	ID        uint   `gorm:"primarykey"`
	Code      string `gorm:"uniqueIndex;size:10;not null"` // USDT, BTC, ETH, etc.
	Name      string `gorm:"size:100;not null"`            // Full name (e.g., "Tether")
	Symbol    string `gorm:"size:10;not null"`             // Symbol (e.g., "$")
	Decimals  int    `gorm:"not null;default:8"`           // Decimal places for precision
	IsActive  bool   `gorm:"not null;default:true"`        // Whether currency is active
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName specifies the table name for GORM
func (Currency) TableName() string {
	return "currencies"
}
