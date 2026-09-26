package services

import (
	"mini-crypto-wallet-api/internal/config"

	"github.com/shopspring/decimal"
)

// DefaultMaxTransferAmount is used when config.Config.MaxTransferAmount is
// empty or fails to parse. See docs/AUDIT.md M6.
const DefaultMaxTransferAmount = "1000000"

// MaxTransferAmount returns the effective per-transfer maximum, falling
// back to DefaultMaxTransferAmount if MAX_TRANSFER_AMOUNT is unset or
// invalid. Shared by TransactionService (which enforces it) and
// CurrencyService (which exposes it via GET /currencies so the frontend
// can validate client-side before submitting - see
// docs/BACKEND_PREP_PLAN.md 2.2).
func MaxTransferAmount() decimal.Decimal {
	if config.Config != nil && config.Config.MaxTransferAmount != "" {
		if parsed, err := decimal.NewFromString(config.Config.MaxTransferAmount); err == nil {
			return parsed
		}
	}
	return decimal.RequireFromString(DefaultMaxTransferAmount)
}
