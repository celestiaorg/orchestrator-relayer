package p2p

import "errors"

var (
	ErrPeersTimeout                    = errors.New("timeout while waiting for peers")
	ErrPeersThresholdCannotBeNegative  = errors.New("peers threshold cannot be negative")
	ErrNilPrivateKey                   = errors.New("private key cannot be nil")
	ErrNotEnoughValsetConfirms         = errors.New("couldn't find enough valset confirms")
	ErrNotEnoughDataCommitmentConfirms = errors.New("couldn't find enough data commitment confirms")
	ErrInvalidConfirmNamespace         = errors.New("invalid confirm namespace")
	ErrInvalidEVMAddress               = errors.New("invalid evm address")
	ErrNotTheSameEVMAddress            = errors.New("not the same evm address")
	ErrInvalidConfirmKey               = errors.New("invalid confirm key")
	ErrNoValues                        = errors.New("can't select from no values")
	ErrNoValidValueFound               = errors.New("no valid dht confirm value found")
	ErrEmptyNamespace                  = errors.New("empty namespace")
	ErrEmptyEVMAddr                    = errors.New("empty evm address")
)
