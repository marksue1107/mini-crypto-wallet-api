package repositories

import (
	"time"

	"gorm.io/gorm"
	"mini-crypto-wallet-api/models"

	"github.com/shopspring/decimal"
)

type ITransaction interface {
	CreateTransaction(transaction *models.Transaction, tx ...*gorm.DB) error
	GetTransactionsByUserID(userID uint) ([]models.Transaction, error)
	GetTransactionsByUserIDWithPagination(userID uint, offset, limit int) ([]models.Transaction, int64, error)
	FindByHash(hash string) (*models.Transaction, error)
	// GetStatsSince aggregates, in the database, every transaction
	// involving userID (as sender or recipient) with created_at >= since:
	// how many there are, how much was sent, and how much was received.
	// Callers must not instead fetch matching rows and sum them in Go -
	// see docs/BACKEND_PREP_PLAN.md 2.1.
	GetStatsSince(userID uint, since time.Time) (count int64, totalSent, totalReceived decimal.Decimal, err error)
}
