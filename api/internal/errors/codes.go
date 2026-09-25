package errors

// 錯誤代碼定義
const (
	// 通用錯誤
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternalError  = "INTERNAL_ERROR"

	// 用戶相關錯誤
	ErrCodeUserNotFound       = "USER_NOT_FOUND"
	ErrCodeUserAlreadyExists  = "USER_ALREADY_EXISTS"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// 錢包相關錯誤
	ErrCodeWalletNotFound      = "WALLET_NOT_FOUND"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodeInvalidAmount       = "INVALID_AMOUNT"

	// 交易相關錯誤
	ErrCodeTransactionNotFound = "TRANSACTION_NOT_FOUND"
	ErrCodeSameAccountTransfer = "SAME_ACCOUNT_TRANSFER"
	ErrCodeTransactionFailed   = "TRANSACTION_FAILED"
)
