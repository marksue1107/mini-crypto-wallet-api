package repositories

import (
	"mini-crypto-wallet-api/database"
	"mini-crypto-wallet-api/models"
)

type transactionRepositoryImpl struct{}

func NewTransactionRepository() TransactionRepository {
	return &transactionRepositoryImpl{}
}

func (r *transactionRepositoryImpl) CreateTransaction(tx *models.Transaction) error {
	return database.DB.Create(tx).Error
}

func (r *transactionRepositoryImpl) GetTransactionsByUserID(userID uint) ([]models.Transaction, error) {
	var txs []models.Transaction
	err := database.DB.
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Order("created_at desc").
		Find(&txs).Error
	return txs, err
}
