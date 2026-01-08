package repositories

import (
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"
)

type currencyRepository struct {
	entity.DBClient
}

func NewCurrencyRepository() ICurrency {
	r := new(currencyRepository)
	r.DBClient.MasterDB = db_conn.Conn_DB.MasterDB
	return r
}

func (r *currencyRepository) CreateCurrency(currency *models.Currency) error {
	return r.DBClient.MasterDB.Create(currency).Error
}

func (r *currencyRepository) GetCurrencyByCode(code string) (*models.Currency, error) {
	var currency models.Currency
	if err := r.DBClient.MasterDB.Where("code = ? AND is_active = ?", code, true).First(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (r *currencyRepository) GetCurrencyByID(id uint) (*models.Currency, error) {
	var currency models.Currency
	if err := r.DBClient.MasterDB.Where("id = ? AND is_active = ?", id, true).First(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

func (r *currencyRepository) GetAllCurrencies() ([]models.Currency, error) {
	var currencies []models.Currency
	err := r.DBClient.MasterDB.Where("is_active = ?", true).Find(&currencies).Error
	return currencies, err
}
