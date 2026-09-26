package repositories

import (
	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"
)

type IWallet interface {
	GetWalletByUserID(userID uint) (*models.Wallet, error)
	GetWalletByUserIDAndCurrency(userID uint, currencyID uint) (*models.Wallet, error)
	// GetWalletByUserIDAndCurrencyWithTx locks (SELECT ... FOR UPDATE, where
	// supported - see docs/AUDIT.md M2 for SQLite's lack of row locking) and
	// returns the wallet matching both userID AND currencyID. Filtering by
	// currency here, rather than fetching by userID alone and checking
	// CurrencyID afterwards, matters once a user can hold more than one
	// wallet: see docs/AUDIT.md M4.
	GetWalletByUserIDAndCurrencyWithTx(userID uint, currencyID uint, tx ...*gorm.DB) (*models.Wallet, error)
	CreateWallet(wallet *models.Wallet, tx ...*gorm.DB) error
	UpdateWallet(wallet *models.Wallet, tx ...*gorm.DB) error
}
