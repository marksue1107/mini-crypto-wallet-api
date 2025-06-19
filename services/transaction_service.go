package services

import (
	"errors"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type TransactionService struct {
	walletRepo      repositories.WalletRepository
	transactionRepo repositories.TransactionRepository
}

func NewTransactionService(walletRepo repositories.WalletRepository, txRepo repositories.TransactionRepository) *TransactionService {
	return &TransactionService{
		walletRepo:      walletRepo,
		transactionRepo: txRepo,
	}
}

func (s *TransactionService) Transfer(fromID, toID uint, amount float64) error {
	fromWallet, err := s.walletRepo.GetWalletByUserID(fromID)
	if err != nil {
		return errors.New("from_user wallet not found")
	}
	toWallet, err := s.walletRepo.GetWalletByUserID(toID)
	if err != nil {
		return errors.New("to_user wallet not found")
	}
	if fromWallet.Balance < amount {
		return errors.New("insufficient balance")
	}
	fromWallet.Balance -= amount
	toWallet.Balance += amount
	if err := s.walletRepo.UpdateWallet(fromWallet); err != nil {
		return err
	}
	if err := s.walletRepo.UpdateWallet(toWallet); err != nil {
		return err
	}
	tx := &models.Transaction{
		FromUserID: fromID,
		ToUserID:   toID,
		Amount:     amount,
	}
	return s.transactionRepo.CreateTransaction(tx)
}

func (s *TransactionService) GetTransactions(userID uint) ([]models.Transaction, error) {
	return s.transactionRepo.GetTransactionsByUserID(userID)
}
