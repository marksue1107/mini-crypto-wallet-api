package test

import (
	"fmt"
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"
	"sync"
	"testing"

	"gorm.io/gorm"

	"github.com/shopspring/decimal"
)

// 測試主流程
func TestConcurrentTransfers(t *testing.T) {
	config.LoadConfig()
	// 初始化資料庫
	db_conn.InitDatabase()

	// 初始化 repository 和 service
	walletRepo := repositories.NewWalletRepository()
	txRepo := repositories.NewTransactionRepository()
	txService := services.NewTransactionService(walletRepo, txRepo, nil) // Kafka 可用 nil

	// 重置 A/B 錢包
	resetWallets(db_conn.Conn_DB.MasterDB, walletRepo)

	fmt.Println("=== 測試未加鎖交易 ===")
	simulateConcurrentTransfers(t, txService, walletRepo, false)

	resetWallets(db_conn.Conn_DB.MasterDB, walletRepo)

	fmt.Println("=== 測試加鎖交易 ===")
	simulateConcurrentTransfers(t, txService, walletRepo, true)
}

// 模擬兩個 goroutine 同時轉帳
func simulateConcurrentTransfers(t *testing.T, service *services.TransactionService, walletRepo repositories.IWallet, useLock bool) {
	var wg sync.WaitGroup
	wg.Add(2)

	fromID := uint(1)
	toID := uint(2)
	currencyID := uint(1) // 預設幣種
	amount := decimal.NewFromInt(800)

	for i := 1; i <= 2; i++ {
		go func(id int) {
			defer wg.Done()
			fmt.Println("🔄 正在執行 TransferWithLockOption(..., useLock =", useLock, ")")
			if err := service.TransferWithLockOption(t, fromID, toID, currencyID, amount, useLock); err != nil {
				log.Printf("🔴 Transfer %d failed: %v", id, err)
			} else {
				log.Printf("🟢 Transfer %d success\n", id)
			}
		}(i)
	}

	wg.Wait()

	// 顯示最終餘額
	from, fromerr := walletRepo.GetWalletByUserID(fromID)
	to, toerr := walletRepo.GetWalletByUserID(toID)

	if fromerr != nil || from == nil {
		fmt.Printf("❌ 查詢 A 錢包失敗: %v\n", fromerr)
	} else {
		fmt.Printf("📊 A 最終餘額: %s\n", from.Balance.String())
	}

	if toerr != nil || to == nil {
		fmt.Printf("❌ 查詢 B 錢包失敗: %v\n", toerr)
	} else {
		fmt.Printf("📊 B 最終餘額: %s\n", to.Balance.String())
	}
}

// 重置 A/B 錢包初始金額
func resetWallets(db *gorm.DB, repo repositories.IWallet) {
	db.Exec("DELETE FROM wallets")

	repo.CreateWallet(&models.Wallet{UserID: 1, CurrencyID: 1, Balance: decimal.NewFromInt(1000)})
	repo.CreateWallet(&models.Wallet{UserID: 2, CurrencyID: 1, Balance: decimal.Zero})
}
