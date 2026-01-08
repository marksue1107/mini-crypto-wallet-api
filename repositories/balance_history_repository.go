package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"
)

type balanceHistoryRepository struct {
	entity.DBClient
}

func NewBalanceHistoryRepository() IBalanceHistory {
	r := new(balanceHistoryRepository)
	r.DBClient.MasterDB = db_conn.Conn_DB.MasterDB
	return r
}

func (r *balanceHistoryRepository) CreateHistory(history *models.BalanceHistory, tx ...*gorm.DB) error {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.Create(history).Error
}

func (r *balanceHistoryRepository) GetHistoryByUserID(userID uint) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	err := r.DBClient.MasterDB.
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&histories).Error
	return histories, err
}

func (r *balanceHistoryRepository) GetHistoryByWalletID(walletID uint) ([]models.BalanceHistory, error) {
	var histories []models.BalanceHistory
	err := r.DBClient.MasterDB.
		Where("wallet_id = ?", walletID).
		Order("created_at desc").
		Find(&histories).Error
	return histories, err
}
