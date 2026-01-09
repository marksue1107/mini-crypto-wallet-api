package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents the database model for money transfers between users
// Pure GORM model - no JSON/binding tags for HTTP layer separation
// Retains domain logic methods (GenerateHash, GenerateSignature) as they are
// part of the business logic, not HTTP serialization
type Transaction struct {
	ID         uint            `gorm:"primarykey"`
	FromUserID uint            `gorm:"index;not null"`
	ToUserID   uint            `gorm:"index;not null"`
	Amount     decimal.Decimal `gorm:"type:decimal(20,8);not null"`
	Hash       string          `gorm:"uniqueIndex;size:64;not null"` // SHA256 hash (64 hex chars)
	Signature  string          `gorm:"size:255;not null"`            // Transaction signature
	Status     string          `gorm:"size:50;not null;default:'pending'"` // pending, processing, completed, failed, cancelled
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Relationships - only for GORM, not exposed directly via HTTP
	FromUser User `gorm:"foreignKey:FromUserID"`
	ToUser   User `gorm:"foreignKey:ToUserID"`
}

// TableName specifies the table name for GORM
func (Transaction) TableName() string {
	return "transactions"
}

// GenerateHash creates a unique SHA256 hash for the transaction
// Domain logic method - belongs with the model
func (t *Transaction) GenerateHash() string {
	data := fmt.Sprintf("%d|%d|%s|%d", t.FromUserID, t.ToUserID, t.Amount.String(), t.CreatedAt.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateSignature creates a signature for the transaction
// Domain logic method - simulated signature for demonstration
func (t *Transaction) GenerateSignature() string {
	return fmt.Sprintf("SIG-%d-%s-%d", t.FromUserID, t.Amount.String(), t.CreatedAt.UnixNano())
}
