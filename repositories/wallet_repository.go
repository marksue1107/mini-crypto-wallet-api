package repositories

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mini-crypto-wallet-api/database"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"
)

type walletRepository struct {
	entity.DBClient
}

func NewWalletRepository() IWallet {
	r := new(walletRepository)
	r.DBClient.MasterDB = database.DB.MasterDB

	return r
}

func (r *walletRepository) GetWalletByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.DBClient.MasterDB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) GetWalletByUserIDWithTx(userID uint, tx ...*gorm.DB) (*models.Wallet, error) {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}

	var wallet models.Wallet

	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (r *walletRepository) CreateWallet(wallet *models.Wallet, tx ...*gorm.DB) error {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}

	return db.Create(wallet).Error
}

func (r *walletRepository) UpdateWallet(wallet *models.Wallet, tx ...*gorm.DB) error {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}

	return db.Save(wallet).Error
}
