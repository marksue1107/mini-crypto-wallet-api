package services

import (
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"

	"github.com/shopspring/decimal"
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

// MaxTransferAmount exposes the same effective per-transfer limit that
// TransactionService.Transfer enforces, so GET /currencies can surface it
// to the frontend for client-side validation before submitting. See
// docs/BACKEND_PREP_PLAN.md 2.2.
func (s *CurrencyService) MaxTransferAmount() decimal.Decimal {
	return MaxTransferAmount()
}
