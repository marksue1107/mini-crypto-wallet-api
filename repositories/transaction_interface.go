package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"
)

type ITransaction interface {
	CreateTransaction(transaction *models.Transaction, tx ...*gorm.DB) error
	GetTransactionsByUserID(userID uint) ([]models.Transaction, error)
	FindByHash(hash string) (*models.Transaction, error)
}
