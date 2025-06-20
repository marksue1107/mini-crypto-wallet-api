package services

import (
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type UserService struct {
	userRepo   repositories.IUser
	walletRepo repositories.IWallet
}

func NewUserService(userRepo repositories.IUser, walletRepo repositories.IWallet) *UserService {
	return &UserService{userRepo, walletRepo}
}

func (s *UserService) CreateUser(user *models.User) error {
	if err := s.userRepo.CreateUser(user); err != nil {
		return err
	}
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}

	return s.walletRepo.CreateWallet(wallet)
}
