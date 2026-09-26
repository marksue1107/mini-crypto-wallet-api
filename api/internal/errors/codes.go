package errors

// 錯誤代碼定義
const (
	// 通用錯誤
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// 用戶相關錯誤
	ErrCodeUserNotFound       = "USER_NOT_FOUND"
	ErrCodeUserAlreadyExists  = "USER_ALREADY_EXISTS"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// 錢包相關錯誤
	ErrCodeWalletNotFound      = "WALLET_NOT_FOUND"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"

	// ErrCodeInvalidAmount is deprecated: docs/FRONTEND_SPEC.md §7-3 asked
	// for this to be split into distinct codes per validation failure
	// (below) so the frontend can branch without parsing message text. Kept
	// here, unused, only so any external caller that already matches on the
	// literal string "INVALID_AMOUNT" isn't surprised by the identifier
	// disappearing; do not start returning it again.
	ErrCodeInvalidAmount = "INVALID_AMOUNT"

	ErrCodeAmountNotPositive  = "AMOUNT_NOT_POSITIVE"
	ErrCodeInvalidDecimals    = "INVALID_DECIMALS"
	ErrCodeAmountExceedsLimit = "AMOUNT_EXCEEDS_LIMIT"

	// SENDER_WALLET_NOT_FOUND / RECIPIENT_NOT_FOUND only apply to
	// POST /wallet/transfer, where WALLET_NOT_FOUND used to mean two
	// different things (the sender's own wallet vs. the recipient's) under
	// the same code. GET /wallet/{user_id} keeps plain WALLET_NOT_FOUND -
	// there's only one wallet in play there, so it isn't ambiguous.
	ErrCodeSenderWalletNotFound = "SENDER_WALLET_NOT_FOUND"
	ErrCodeRecipientNotFound    = "RECIPIENT_NOT_FOUND"

	// 交易相關錯誤
	ErrCodeTransactionNotFound = "TRANSACTION_NOT_FOUND"
	ErrCodeSameAccountTransfer = "SAME_ACCOUNT_TRANSFER"
	ErrCodeTransactionFailed   = "TRANSACTION_FAILED"
)
