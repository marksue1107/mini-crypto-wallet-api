package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"
)

type transactionRepository struct {
	entity.DBClient
}

func NewTransactionRepository() ITransaction {
	r := new(transactionRepository)
	r.DBClient.MasterDB = db_conn.Conn_DB.MasterDB

	return r
}

func (r *transactionRepository) CreateTransaction(transaction *models.Transaction, tx ...*gorm.DB) error {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}

	return db.Create(transaction).Error
}

func (r *transactionRepository) GetTransactionsByUserID(userID uint) ([]models.Transaction, error) {
	var txs []models.Transaction
	err := r.DBClient.MasterDB.
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Order("created_at desc").
		Find(&txs).Error
	return txs, err
}

func (r *transactionRepository) FindByHash(hash string) (*models.Transaction, error) {
	var tx models.Transaction
	if err := r.DBClient.MasterDB.Where("hash = ?", hash).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}
