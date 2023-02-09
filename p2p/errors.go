package p2p

import "errors"

var (
	ErrNilPrivateKey                   = errors.New("private key cannot be nil")
	ErrNotEnoughValsetConfirms         = errors.New("couldn't find enough valset confirms")
	ErrNotEnoughDataCommitmentConfirms = errors.New("couldn't find enough data commitment confirms")
)
