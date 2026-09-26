package services

import (
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type CurrencyService struct {
	currencyRepo repositories.ICurrency
}

func NewCurrencyService(currencyRepo repositories.ICurrency) *CurrencyService {
	return &CurrencyService{
		currencyRepo: currencyRepo,
	}
}

func (s *CurrencyService) GetAllCurrencies() ([]models.Currency, error) {
	return s.currencyRepo.GetAllCurrencies()
}

func (s *CurrencyService) GetCurrencyByCode(code string) (*models.Currency, error) {
	return s.currencyRepo.GetCurrencyByCode(code)
}

func (s *CurrencyService) GetCurrencyByID(id uint) (*models.Currency, error) {
	return s.currencyRepo.GetCurrencyByID(id)
}
