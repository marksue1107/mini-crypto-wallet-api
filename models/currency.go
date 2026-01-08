package models

import "time"

// Currency 幣種模型
type Currency struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Code        string    `json:"code" gorm:"uniqueIndex;size:10" example:"USDT"` // USDT, BTC, ETH
	Name        string    `json:"name" example:"Tether"`
	Symbol      string    `json:"symbol" example:"$"`
	Decimals    int       `json:"decimals" example:"6"` // 小數位數
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
