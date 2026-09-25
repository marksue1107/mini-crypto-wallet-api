package utils

import (
	"github.com/shopspring/decimal"
)

// DecimalFromFloat 將 float64 轉換為 decimal.Decimal
func DecimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// DecimalFromString 從字串創建 decimal.Decimal
func DecimalFromString(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// ValidatePositiveAmount 驗證金額是否為正數
func ValidatePositiveAmount(amount decimal.Decimal) bool {
	return amount.IsPositive()
}

// ValidateNonNegativeAmount 驗證金額是否為非負數
func ValidateNonNegativeAmount(amount decimal.Decimal) bool {
	return amount.IsPositive() || amount.IsZero()
}
