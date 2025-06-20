package services

import (
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type WalletService struct {
	walletRepo repositories.IWallet
}

func NewWalletService(walletRepo repositories.IWallet) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
	}
}

func (s *WalletService) GetWallet(userID uint) (*models.Wallet, error) {
	return s.walletRepo.GetWalletByUserID(userID)
}
