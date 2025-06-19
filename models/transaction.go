package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Transaction represents a money transfer between users
// @Description Transaction record containing hash and signature
type Transaction struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	FromUserID uint      `json:"from_user_id"`
	ToUserID   uint      `json:"to_user_id"`
	Amount     float64   `json:"amount"`
	Hash       string    `json:"hash"`      // 🔹 交易唯一 Hash
	Signature  string    `json:"signature"` // 🔹 模擬簽章
	CreatedAt  time.Time `json:"created_at"`
}

func (t *Transaction) GenerateHash() string {
	data := fmt.Sprintf("%d|%d|%f|%d", t.FromUserID, t.ToUserID, t.Amount, t.CreatedAt.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (t *Transaction) GenerateSignature() string {
	// 假簽章，只是 userID + Amount + timestamp 再轉大寫
	return fmt.Sprintf("SIG-%d-%f-%d", t.FromUserID, t.Amount, t.CreatedAt.UnixNano())
}
