package utils

import (
	"github.com/shopspring/decimal"
)

// ValidatePositiveAmount 驗證金額是否為正數
func ValidatePositiveAmount(amount decimal.Decimal) bool {
	return amount.IsPositive()
}
