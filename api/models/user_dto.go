package models

import "time"

// UserCreateRequest represents the HTTP request body for creating a new user
// Separated from the User database model to enforce proper validation
// and prevent database concerns from leaking into the API layer
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"alice"`
	Email    string `json:"email" binding:"required,email" example:"alice@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
}

// UserResponse represents the HTTP response for user data
// Excludes sensitive fields like password hash
type UserResponse struct {
	ID        uint      `json:"id" example:"1"`
	Username  string    `json:"username" example:"alice"`
	Email     string    `json:"email" example:"alice@example.com"`
	CreatedAt time.Time `json:"created_at"`
}

// ToUserResponse converts a User model to UserResponse DTO
// Ensures password and other sensitive fields are never exposed
func ToUserResponse(user *User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

// UserLookupResponse is the response for GET /users/lookup. Deliberately
// carries only id and username - no email, no created_at - so a caller who
// knows (or is guessing) a username can't use this endpoint to harvest
// other account details. See docs/BACKEND_PREP_PLAN.md 2.3.
type UserLookupResponse struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"alice"`
}

// ToUserLookupResponse converts a User model to UserLookupResponse DTO.
func ToUserLookupResponse(user *User) *UserLookupResponse {
	return &UserLookupResponse{
		ID:       user.ID,
		Username: user.Username,
	}
}
