package models

import "time"

type User struct {
	ID        uint      `json:"id" example:"1" gorm:"primarykey"`
	Username  string    `json:"username" example:"alice" binding:"required,min=3,max=50"`
	Email     string    `json:"email" example:"alice@example.com" binding:"required,email"`
	Password  string    `json:"-" gorm:"column:password" binding:"required,min=6"` // 不在 JSON 中返回
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	ExpiresIn int    `json:"expires_in"` // 秒數
}
