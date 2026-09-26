package services

import (
	"errors"
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/kafka_client"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/utils"
	"time"

	"github.com/shopspring/decimal"
)

// DefaultMaxTransferAmount is used when config.Config.MaxTransferAmount is
// empty or fails to parse. See docs/AUDIT.md M6.
const DefaultMaxTransferAmount = "1000000"

// Sentinel errors for Transfer(), so handlers can map each one to a stable
// error code for API consumers (docs/AUDIT.md S8) via errors.Is, instead of
// pattern-matching on error message text.
var (
	ErrSameAccountTransfer  = errors.New("cannot transfer to the same account")
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrAmountExceedsLimit   = errors.New("amount exceeds maximum transfer limit")
	ErrCurrencyNotFound     = errors.New("currency not found")
	ErrTooManyDecimalPlaces = errors.New("amount has more decimal places than this currency supports")
	ErrFromWalletNotFound   = errors.New("from_user wallet not found for this currency")
	ErrToWalletNotFound     = errors.New("to_user wallet not found for this currency")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrTransferFailed       = errors.New("transfer failed, please try again")
)

type TransactionService struct {
	walletRepo         repositories.IWallet
	transactionRepo    repositories.ITransaction
	currencyRepo       repositories.ICurrency
	balanceHistoryRepo repositories.IBalanceHistory
	kafkaProducer      *kafka_client.KafkaProducer
	maxTransferAmount  decimal.Decimal
}

func NewTransactionService(walletRepo repositories.IWallet, txRepo repositories.ITransaction, currencyRepo repositories.ICurrency, producer *kafka_client.KafkaProducer) *TransactionService {
	maxAmount := decimal.RequireFromString(DefaultMaxTransferAmount)
	if config.Config != nil && config.Config.MaxTransferAmount != "" {
		if parsed, err := decimal.NewFromString(config.Config.MaxTransferAmount); err == nil {
			maxAmount = parsed
		}
	}

	return &TransactionService{
		walletRepo:         walletRepo,
		transactionRepo:    txRepo,
		currencyRepo:       currencyRepo,
		balanceHistoryRepo: repositories.NewBalanceHistoryRepository(),
		kafkaProducer:      producer,
		maxTransferAmount:  maxAmount,
	}
}

func (s *TransactionService) Transfer(fromID, toID uint, currencyID uint, amount decimal.Decimal) error {
	if fromID == toID {
		return ErrSameAccountTransfer
	}

	// 驗證金額
	if !utils.ValidatePositiveAmount(amount) {
		return ErrAmountMustBePositive
	}
	if amount.GreaterThan(s.maxTransferAmount) {
		return ErrAmountExceedsLimit
	}

	currency, err := s.currencyRepo.GetCurrencyByID(currencyID)
	if err != nil {
		return ErrCurrencyNotFound
	}
	if !amount.Equal(amount.Round(int32(currency.Decimals))) {
		return ErrTooManyDecimalPlaces
	}

	tx := db_conn.Conn_DB.MasterDB.Begin()
	// committed tracks whether we reached tx.Commit() successfully. Every
	// return path below - including ordinary `return err` on validation
	// failures like insufficient balance, not just panics - must release
	// this transaction's connection back to the pool. utils.RollbackIfPanic
	// alone doesn't do that: it only rolls back on an actual Go panic, so a
	// plain early return left the connection open and idle-in-transaction
	// forever (docs/AUDIT_REMEDIATION_PLAN.md finding N2).
	committed := false
	defer func() {
		if committed {
			return
		}
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		tx.Rollback()
	}()

	// Lock both wallets in a fixed order (ascending user_id), regardless of
	// which one is "from" and which is "to". Locking in caller-argument
	// order would let a concurrent A->B transfer and B->A transfer each grab
	// one lock and then wait forever for the other (classic deadlock). See
	// docs/AUDIT.md M1.
	//
	// GetWalletByUserIDAndCurrencyWithTx also locks based on (user_id,
	// currency_id) together, not user_id alone: a user can hold more than
	// one wallet (one per currency), so filtering by currency_id in the
	// locked query itself - rather than fetching by user_id alone and
	// checking CurrencyID afterwards - matters once multi-currency wallets
	// exist. See docs/AUDIT.md M4.
	lowID, highID := fromID, toID
	if highID < lowID {
		lowID, highID = highID, lowID
	}

	walletNotFoundErr := func(userID uint) error {
		if userID == fromID {
			return ErrFromWalletNotFound
		}
		return ErrToWalletNotFound
	}

	lowWallet, err := s.walletRepo.GetWalletByUserIDAndCurrencyWithTx(lowID, currencyID, tx)
	if err != nil {
		return walletNotFoundErr(lowID)
	}
	highWallet, err := s.walletRepo.GetWalletByUserIDAndCurrencyWithTx(highID, currencyID, tx)
	if err != nil {
		return walletNotFoundErr(highID)
	}

	var fromWallet, toWallet *models.Wallet
	if fromID == lowID {
		fromWallet, toWallet = lowWallet, highWallet
	} else {
		fromWallet, toWallet = highWallet, lowWallet
	}

	// 使用 decimal 比較
	if fromWallet.Balance.LessThan(amount) {
		return ErrInsufficientBalance
	}

	// 記錄變動前的餘額
	fromBalanceBefore := fromWallet.Balance
	toBalanceBefore := toWallet.Balance

	fromWallet.Balance = fromWallet.Balance.Sub(amount)
	toWallet.Balance = toWallet.Balance.Add(amount)

	// From here on, any DB error is unexpected (not a validation failure)
	// so it's logged with detail and replaced with a generic message before
	// returning - see docs/AUDIT.md S9.
	if err := s.walletRepo.UpdateWallet(fromWallet, tx); err != nil {
		log.Println("⚠️ failed to update from_wallet during transfer:", err)
		return ErrTransferFailed
	}
	if err := s.walletRepo.UpdateWallet(toWallet, tx); err != nil {
		log.Println("⚠️ failed to update to_wallet during transfer:", err)
		return ErrTransferFailed
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
		log.Println("⚠️ failed to create transaction record during transfer:", err)
		return ErrTransferFailed
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
		log.Println("⚠️ failed to record from_user balance history during transfer:", err)
		return ErrTransferFailed
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
		log.Println("⚠️ failed to record to_user balance history during transfer:", err)
		return ErrTransferFailed
	}

	if commitDB := tx.Commit(); commitDB.Error != nil {
		// Log the real DB error for operators; don't return it to the
		// caller, since it can contain internal details (table/constraint
		// names, driver-specific text). See docs/AUDIT.md S9.
		log.Println("⚠️ failed to commit transfer transaction:", commitDB.Error)
		return ErrTransferFailed
	}
	committed = true

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
