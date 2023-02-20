package p2p

import "errors"

var (
	ErrPeersTimeout                    = errors.New("timeout while waiting for peers")
	ErrPeersThresholdCannotBeNegative  = errors.New("peers threshold cannot be negative")
	ErrNilPrivateKey                   = errors.New("private key cannot be nil")
	ErrNotEnoughValsetConfirms         = errors.New("couldn't find enough valset confirms")
	ErrNotEnoughDataCommitmentConfirms = errors.New("couldn't find enough data commitment confirms")
)
