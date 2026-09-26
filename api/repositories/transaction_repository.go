package repositories

import (
	"time"

	"gorm.io/gorm"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"

	"github.com/shopspring/decimal"
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

func (r *transactionRepository) GetTransactionsByUserIDWithPagination(userID uint, offset, limit int) ([]models.Transaction, int64, error) {
	var txs []models.Transaction
	var total int64

	// 計算總數
	err := r.DBClient.MasterDB.Model(&models.Transaction{}).
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 獲取分頁數據
	err = r.DBClient.MasterDB.
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&txs).Error

	return txs, total, err
}

func (r *transactionRepository) FindByHash(hash string) (*models.Transaction, error) {
	var tx models.Transaction
	if err := r.DBClient.MasterDB.Where("hash = ?", hash).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

// sumRow scans a single `COALESCE(SUM(amount), 0) AS total` aggregate
// column. decimal.Decimal implements sql.Scanner, so this works regardless
// of whether the driver hands back a string, []byte, int64, or float64 for
// that column.
type sumRow struct {
	Total decimal.Decimal
}

func (r *transactionRepository) GetStatsSince(userID uint, since time.Time) (count int64, totalSent, totalReceived decimal.Decimal, err error) {
	if err = r.DBClient.MasterDB.Model(&models.Transaction{}).
		Where("(from_user_id = ? OR to_user_id = ?) AND created_at >= ?", userID, userID, since).
		Count(&count).Error; err != nil {
		return
	}

	var sent, received sumRow
	if err = r.DBClient.MasterDB.Model(&models.Transaction{}).
		Where("from_user_id = ? AND created_at >= ?", userID, since).
		Select("COALESCE(SUM(amount), 0) AS total").
		Scan(&sent).Error; err != nil {
		return
	}
	if err = r.DBClient.MasterDB.Model(&models.Transaction{}).
		Where("to_user_id = ? AND created_at >= ?", userID, since).
		Select("COALESCE(SUM(amount), 0) AS total").
		Scan(&received).Error; err != nil {
		return
	}

	return count, sent.Total, received.Total, nil
}
