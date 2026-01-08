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

	"github.com/shopspring/decimal"
)

type TransactionService struct {
	walletRepo         repositories.IWallet
	transactionRepo    repositories.ITransaction
	balanceHistoryRepo repositories.IBalanceHistory
	kafkaProducer      *kafka_client.KafkaProducer
}

func NewTransactionService(walletRepo repositories.IWallet, txRepo repositories.ITransaction, producer *kafka_client.KafkaProducer) *TransactionService {
	return &TransactionService{
		walletRepo:         walletRepo,
		transactionRepo:    txRepo,
		balanceHistoryRepo: repositories.NewBalanceHistoryRepository(),
		kafkaProducer:      producer,
	}
}

func (s *TransactionService) Transfer(fromID, toID uint, currencyID uint, amount decimal.Decimal) error {
	if fromID == toID {
		return errors.New("cannot transfer to the same account")
	}

	// 驗證金額
	if !utils.ValidatePositiveAmount(amount) {
		return errors.New("amount must be positive")
	}

	tx := db_conn.Conn_DB.MasterDB.Begin()
	defer utils.RollbackIfPanic(tx)

	// 使用幣種查詢錢包
	fromWallet, err := s.walletRepo.GetWalletByUserIDAndCurrency(fromID, currencyID)
	if err != nil {
		return errors.New("from_user wallet not found for this currency")
	}
	toWallet, err := s.walletRepo.GetWalletByUserIDAndCurrency(toID, currencyID)
	if err != nil {
		return errors.New("to_user wallet not found for this currency")
	}

	// 使用行鎖更新錢包
	fromWalletLocked, err := s.walletRepo.GetWalletByUserIDWithTx(fromID, tx)
	if err != nil || fromWalletLocked.CurrencyID != currencyID {
		return errors.New("from_user wallet not found for this currency")
	}
	toWalletLocked, err := s.walletRepo.GetWalletByUserIDWithTx(toID, tx)
	if err != nil || toWalletLocked.CurrencyID != currencyID {
		return errors.New("to_user wallet not found for this currency")
	}

	fromWallet = fromWalletLocked
	toWallet = toWalletLocked

	// 使用 decimal 比較
	if fromWallet.Balance.LessThan(amount) {
		return errors.New("insufficient balance")
	}

	// 記錄變動前的餘額
	fromBalanceBefore := fromWallet.Balance
	toBalanceBefore := toWallet.Balance

	fromWallet.Balance = fromWallet.Balance.Sub(amount)
	toWallet.Balance = toWallet.Balance.Add(amount)

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
		Status:     "completed",
	}
	transaction.Hash = transaction.GenerateHash()
	transaction.Signature = transaction.GenerateSignature()

	if err := s.transactionRepo.CreateTransaction(transaction, tx); err != nil {
		return err
	}

	// 記錄餘額變動歷史
	fromHistory := &models.BalanceHistory{
		UserID:        fromID,
		WalletID:      fromWallet.ID,
		TransactionID: transaction.ID,
		ChangeType:    "debit",
		Amount:        amount,
		BalanceBefore: fromBalanceBefore,
		BalanceAfter:  fromWallet.Balance,
	}
	if err := s.balanceHistoryRepo.CreateHistory(fromHistory, tx); err != nil {
		return err
	}

	toHistory := &models.BalanceHistory{
		UserID:        toID,
		WalletID:      toWallet.ID,
		TransactionID: transaction.ID,
		ChangeType:    "credit",
		Amount:        amount,
		BalanceBefore: toBalanceBefore,
		BalanceAfter:  toWallet.Balance,
	}
	if err := s.balanceHistoryRepo.CreateHistory(toHistory, tx); err != nil {
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

func (s *TransactionService) GetTransactionsWithPagination(userID uint, offset, limit int) ([]models.Transaction, int64, error) {
	return s.transactionRepo.GetTransactionsByUserIDWithPagination(userID, offset, limit)
}

func (s *TransactionService) GetTransactionByHash(hash string) (*models.Transaction, error) {
	return s.transactionRepo.FindByHash(hash)
}

/*

tester

*/

func (s *TransactionService) TransferWithLockOption(t *testing.T, fromID, toID uint, currencyID uint, amount decimal.Decimal, useLock bool) error {
	if useLock {
		return s.Transfer(fromID, toID, currencyID, amount) // 使用加鎖版本
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
	if fromWallet.Balance.LessThan(amount) {
		return errors.New("insufficient balance")
	}

	fromWallet.Balance = fromWallet.Balance.Sub(amount)
	toWallet.Balance = toWallet.Balance.Add(amount)

	s.walletRepo.UpdateWallet(fromWallet)
	s.walletRepo.UpdateWallet(toWallet)
	return nil
}
