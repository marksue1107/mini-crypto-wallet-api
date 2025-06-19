package repositories

import (
	"mini-crypto-wallet-api/database"
	"mini-crypto-wallet-api/models"
)

type walletRepositoryImpl struct{}

func NewWalletRepository() WalletRepository {
	return &walletRepositoryImpl{}
}

func (r *walletRepositoryImpl) GetWalletByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := database.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepositoryImpl) UpdateWallet(wallet *models.Wallet) error {
	return database.DB.Save(wallet).Error
}

func (r *walletRepositoryImpl) CreateTransaction(tx *models.Transaction) error {
	return database.DB.Create(tx).Error
}
