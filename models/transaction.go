package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a money transfer between users
// @Description Transaction record containing hash and signature
type Transaction struct {
	ID         uint            `json:"id" gorm:"primarykey"`
	FromUserID uint            `json:"from_user_id"`
	ToUserID   uint            `json:"to_user_id"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:decimal(20,8)" swaggertype:"number"`
	Hash       string          `json:"hash"`                            // 🔹 交易唯一 Hash
	Signature  string          `json:"signature"`                       // 🔹 模擬簽章
	Status     string          `json:"status" gorm:"default:'pending'"` // pending, processing, completed, failed, cancelled
	CreatedAt  time.Time       `json:"created_at"`
}

func (t *Transaction) GenerateHash() string {
	data := fmt.Sprintf("%d|%d|%s|%d", t.FromUserID, t.ToUserID, t.Amount.String(), t.CreatedAt.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (t *Transaction) GenerateSignature() string {
	// 假簽章，只是 userID + Amount + timestamp 再轉大寫
	return fmt.Sprintf("SIG-%d-%s-%d", t.FromUserID, t.Amount.String(), t.CreatedAt.UnixNano())
}
