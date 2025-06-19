package repositories

import "mini-crypto-wallet-api/models"

type TransactionRepository interface {
	CreateTransaction(tx *models.Transaction) error
	GetTransactionsByUserID(userID uint) ([]models.Transaction, error)
	FindByHash(hash string) (*models.Transaction, error)
}
