package models

import "time"

// User represents the database model for user data
// Pure GORM model - no JSON/binding tags for HTTP layer separation
type User struct {
	ID        uint   `gorm:"primarykey"`
	Username  string `gorm:"uniqueIndex;size:50;not null"`
	Email     string `gorm:"uniqueIndex;size:255;not null"`
	Password  string `gorm:"column:password;size:255;not null"` // bcrypt hash
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// LoginRequest represents the HTTP request body for user login
// Pure DTO - no GORM tags for database layer separation
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse represents the HTTP response for successful login
// Pure DTO - no GORM tags for database layer separation
type LoginResponse struct {
	Token     string `json:"token"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	ExpiresIn int    `json:"expires_in"` // Seconds until token expiration
}
