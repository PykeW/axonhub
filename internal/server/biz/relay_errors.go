package biz

import "errors"

// Sentinel errors for relay subkey operations.
// Use these with fmt.Errorf("...: %w", ErrRelayXxx) so API handlers
// can identify error types via errors.Is instead of brittle string matching.
var (
	ErrRelayProductNotFound      = errors.New("relay product not found")
	ErrRelayKeyNotFound          = errors.New("relay key not found")
	ErrRelayBindingNotFound      = errors.New("relay product channel binding not found")
	ErrRelayWalletNotFound       = errors.New("relay wallet not found")
	ErrRelayLedgerNotFound       = errors.New("relay wallet ledger entry not found")
	ErrRelayProductCodeExists    = errors.New("relay product code already exists")
	ErrRelayProductAlreadyBound  = errors.New("relay product is already bound to channel")
	ErrRelayInvalidInput         = errors.New("invalid relay input")
	ErrRelayRequiredField        = errors.New("required field missing")
	ErrRelayUnsupportedValue     = errors.New("unsupported value")
)
