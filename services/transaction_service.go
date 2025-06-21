package services

import (
	"errors"
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/kafka_client"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/utils"
	"testing"
	"time"
)

type TransactionService struct {
	walletRepo      repositories.IWallet
	transactionRepo repositories.ITransaction
	kafkaProducer   *kafka_client.KafkaProducer
}

func NewTransactionService(walletRepo repositories.IWallet, txRepo repositories.ITransaction, producer *kafka_client.KafkaProducer) *TransactionService {
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

	tx := db_conn.Conn_DB.MasterDB.Begin()
	defer utils.RollbackIfPanic(tx)

	if err := s.walletRepo.UpdateWallet(fromWallet, tx); err != nil {
		return err
	}
	if err := s.walletRepo.UpdateWallet(toWallet, tx); err != nil {
		return err
	}
	transaction := &models.Transaction{
		FromUserID: fromID,
		ToUserID:   toID,
		Amount:     amount,
	}
	transaction.Hash = transaction.GenerateHash()
	transaction.Signature = transaction.GenerateSignature()

	if err := s.transactionRepo.CreateTransaction(transaction, tx); err != nil {
		return err
	}

	if commitDB := tx.Commit(); commitDB.Error != nil {
		return commitDB.Error
	}

	// Send Kafka message
	if s.kafkaProducer != nil {
		msg := kafka_client.TxCreatedMessage{
			Hash:       transaction.Hash,
			FromUserID: transaction.FromUserID,
			ToUserID:   transaction.ToUserID,
			Amount:     transaction.Amount,
			Timestamp:  transaction.CreatedAt.Format(time.RFC3339),
		}
		if err := s.kafkaProducer.SendTxCreated(msg); err != nil {
			log.Println("⚠️ Kafka tx.created 發送失敗:", err)
		}
	}
	return nil
}

func (s *TransactionService) GetTransactions(userID uint) ([]models.Transaction, error) {
	return s.transactionRepo.GetTransactionsByUserID(userID)
}

func (s *TransactionService) GetTransactionByHash(hash string) (*models.Transaction, error) {
	return s.transactionRepo.FindByHash(hash)
}

/*

tester

*/

func (s *TransactionService) TransferWithLockOption(t *testing.T, fromID, toID uint, amount float64, useLock bool) error {
	if useLock {
		return s.Transfer(fromID, toID, amount) // 使用加鎖版本
	}

	// 模擬未加鎖（不安全寫法）
	fromWallet, err := s.walletRepo.GetWalletByUserID(fromID)
	if err != nil {
		return err
	}
	toWallet, err := s.walletRepo.GetWalletByUserID(toID)
	if err != nil {
		return err
	}
	if fromWallet.Balance < amount {
		return errors.New("insufficient balance")
	}

	fromWallet.Balance -= amount
	toWallet.Balance += amount

	s.walletRepo.UpdateWallet(fromWallet)
	s.walletRepo.UpdateWallet(toWallet)
	return nil
}
