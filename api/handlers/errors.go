package handlers

import "errors"

// Shared sentinel errors for handler-level failures that aren't already
// backed by a service-layer error. Centralized here so the message text
// passed to apierrors.RespondError is consistent across handlers.
// See docs/AUDIT.md S8.
var (
	errInvalidUserID             = errors.New("invalid user_id")
	errInvalidCurrencyID         = errors.New("invalid currency id")
	errWalletNotFound            = errors.New("wallet not found")
	errCurrencyNotFound          = errors.New("currency not found")
	errTransactionNotFound       = errors.New("transaction not found")
	errFailedToFetchTransactions = errors.New("failed to fetch transactions")
	errFailedToFetchCurrencies   = errors.New("failed to fetch currencies")
	errFailedToCreateUser        = errors.New("failed to create user")
	errFailedToGenerateToken     = errors.New("failed to generate token")
	errUnsupportedStatsWindow    = errors.New("unsupported stats window: only 24h is supported")
	errFailedToFetchStats        = errors.New("failed to fetch wallet stats")
	errUsernameRequired          = errors.New("username query parameter is required")
	errUserNotFound              = errors.New("user not found")
)
