package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionResponse represents the HTTP response for transaction data
// Separated from the Transaction database model to:
// 1. Control API contract independently from database schema
// 2. Prevent business logic methods from leaking into HTTP layer
// 3. Allow future API versioning without breaking database layer
type TransactionResponse struct {
	ID         uint            `json:"id" example:"1"`
	FromUserID uint            `json:"from_user_id" example:"1"`
	ToUserID   uint            `json:"to_user_id" example:"2"`
	Amount     decimal.Decimal `json:"amount" swaggertype:"number" example:"100.0"`
	Hash       string          `json:"hash" example:"abc123..."`
	Signature  string          `json:"signature" example:"SIG-1-100-123456"`
	Status     string          `json:"status" example:"completed"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ToTransactionResponse converts a Transaction model to TransactionResponse DTO
func ToTransactionResponse(tx *Transaction) *TransactionResponse {
	return &TransactionResponse{
		ID:         tx.ID,
		FromUserID: tx.FromUserID,
		ToUserID:   tx.ToUserID,
		Amount:     tx.Amount,
		Hash:       tx.Hash,
		Signature:  tx.Signature,
		Status:     tx.Status,
		CreatedAt:  tx.CreatedAt,
	}
}

// ToTransactionResponses converts a slice of Transaction models to DTOs
func ToTransactionResponses(txs []Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, len(txs))
	for i, tx := range txs {
		responses[i] = *ToTransactionResponse(&tx)
	}
	return responses
}
