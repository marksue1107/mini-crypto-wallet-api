package models

import "time"

type Wallet struct {
	ID        uint      `json:"id" example:"1" gorm:"primarykey"`
	UserID    uint      `json:"user_id" example:"1"`
	Balance   float64   `json:"balance" example:"1000.0"`
	CreatedAt time.Time `json:"created_at"`
}
