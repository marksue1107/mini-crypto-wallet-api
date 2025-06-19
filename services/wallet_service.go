package services

import (
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type WalletService struct {
	walletRepo repositories.WalletRepository
}

func NewWalletService(walletRepo repositories.WalletRepository) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
	}
}

func (s *WalletService) GetWallet(userID uint) (*models.Wallet, error) {
	return s.walletRepo.GetWalletByUserID(userID)
}
