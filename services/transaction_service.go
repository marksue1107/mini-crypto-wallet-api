package services

import (
	"errors"
	"log"
	"mini-crypto-wallet-api/kafkaclient"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"time"
)

type TransactionService struct {
	walletRepo      repositories.WalletRepository
	transactionRepo repositories.TransactionRepository
	kafkaProducer   *kafkaclient.KafkaProducer
}

func NewTransactionService(walletRepo repositories.WalletRepository, txRepo repositories.TransactionRepository, producer *kafkaclient.KafkaProducer) *TransactionService {
	return &TransactionService{
		walletRepo:      walletRepo,
		transactionRepo: txRepo,
		kafkaProducer:   producer,
	}
}

func (s *TransactionService) Transfer(fromID, toID uint, amount float64) error {
	if fromID == toID {
		return errors.New("cannot transfer to the same account")
	}

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
	tx.Hash = tx.GenerateHash()
	tx.Signature = tx.GenerateSignature()

	if err := s.transactionRepo.CreateTransaction(tx); err != nil {
		return err
	}

	// Send Kafka message
	msg := kafkaclient.TxCreatedMessage{
		Hash:       tx.Hash,
		FromUserID: tx.FromUserID,
		ToUserID:   tx.ToUserID,
		Amount:     tx.Amount,
		Timestamp:  tx.CreatedAt.Format(time.RFC3339),
	}
	if err := s.kafkaProducer.SendTxCreated(msg); err != nil {
		log.Println("⚠️ Kafka tx.created 發送失敗:", err)
	}

	return nil
}

func (s *TransactionService) GetTransactions(userID uint) ([]models.Transaction, error) {
	return s.transactionRepo.GetTransactionsByUserID(userID)
}

func (s *TransactionService) GetTransactionByHash(hash string) (*models.Transaction, error) {
	return s.transactionRepo.FindByHash(hash)
}
