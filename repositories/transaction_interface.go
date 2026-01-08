package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"
)

type ITransaction interface {
	CreateTransaction(transaction *models.Transaction, tx ...*gorm.DB) error
	GetTransactionsByUserID(userID uint) ([]models.Transaction, error)
	GetTransactionsByUserIDWithPagination(userID uint, offset, limit int) ([]models.Transaction, int64, error)
	FindByHash(hash string) (*models.Transaction, error)
}
