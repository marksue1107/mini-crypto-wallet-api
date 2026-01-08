package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"
)

type IBalanceHistory interface {
	CreateHistory(history *models.BalanceHistory, tx ...*gorm.DB) error
	GetHistoryByUserID(userID uint) ([]models.BalanceHistory, error)
	GetHistoryByWalletID(walletID uint) ([]models.BalanceHistory, error)
}
