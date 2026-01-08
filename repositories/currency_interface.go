package repositories

import "mini-crypto-wallet-api/models"

type ICurrency interface {
	CreateCurrency(currency *models.Currency) error
	GetCurrencyByCode(code string) (*models.Currency, error)
	GetCurrencyByID(id uint) (*models.Currency, error)
	GetAllCurrencies() ([]models.Currency, error)
}
