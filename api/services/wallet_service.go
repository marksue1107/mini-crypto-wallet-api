package services

import (
	"time"

	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
)

type WalletService struct {
	walletRepo repositories.IWallet
	txRepo     repositories.ITransaction
}

func NewWalletService(walletRepo repositories.IWallet, txRepo repositories.ITransaction) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
		txRepo:     txRepo,
	}
}

func (s *WalletService) GetWallet(userID uint) (*models.Wallet, error) {
	return s.walletRepo.GetWalletByUserID(userID)
}

// StatsWindow is the only value currently accepted by GetStats's window
// parameter. See docs/BACKEND_PREP_PLAN.md 2.1 - anything else is rejected
// by the handler before this is called.
const StatsWindow24h = "24h"

// GetStats computes transaction_count/total_sent/total_received/
// balance_change for userID over the trailing 24 hours, via a single
// database aggregate per repositories.ITransaction.GetStatsSince (not by
// loading every matching transaction into the application and summing
// them here).
func (s *WalletService) GetStats(userID uint) (*models.WalletStatsResponse, error) {
	since := time.Now().Add(-24 * time.Hour)
	count, totalSent, totalReceived, err := s.txRepo.GetStatsSince(userID, since)
	if err != nil {
		return nil, err
	}

	return &models.WalletStatsResponse{
		Window:           StatsWindow24h,
		TransactionCount: count,
		TotalSent:        totalSent,
		TotalReceived:    totalReceived,
		BalanceChange:    totalReceived.Sub(totalSent),
	}, nil
}
