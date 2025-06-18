package models

import "time"

type Transaction struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	FromUserID uint      `json:"from_user_id"`
	ToUserID   uint      `json:"to_user_id"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
}
