package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"
)

type IWallet interface {
	GetWalletByUserID(userID uint) (*models.Wallet, error)
	GetWalletByUserIDWithTx(userID uint, tx ...*gorm.DB) (*models.Wallet, error)
	CreateWallet(wallet *models.Wallet, tx ...*gorm.DB) error
	UpdateWallet(wallet *models.Wallet, tx ...*gorm.DB) error
}
