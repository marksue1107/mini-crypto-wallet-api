package repositories

import "mini-crypto-wallet-api/models"

type WalletRepository interface {
	GetWalletByUserID(userID uint) (*models.Wallet, error)
	UpdateWallet(wallet *models.Wallet) error
	CreateTransaction(tx *models.Transaction) error
}
