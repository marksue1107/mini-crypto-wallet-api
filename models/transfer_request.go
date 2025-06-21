package models

type TransferRequest struct {
	FromUserID uint    `json:"from_user_id" example:"1"`
	ToUserID   uint    `json:"to_user_id" example:"2"`
	Amount     float64 `json:"amount" example:"150.0"`
}
