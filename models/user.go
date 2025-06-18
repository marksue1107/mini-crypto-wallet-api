package models

import "time"

type User struct {
	ID        uint      `json:"id" example:"1" gorm:"primarykey"`
	Username  string    `json:"username" example:"alice"`
	Email     string    `json:"email" example:"alice@example.com"`
	CreatedAt time.Time `json:"created_at"`
}
