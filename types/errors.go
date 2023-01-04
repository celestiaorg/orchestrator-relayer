package types

import (
	"errors"
)

var (
	ErrAttestationNotDataCommitmentRequest = errors.New("attestation is not a data commitment request")
	ErrUnknownAttestationType              = errors.New("unknown attestation type")
	ErrInvalidCommitmentInConfirm          = errors.New("confirm not carrying the right commitment for expected range")
	ErrInvalid                             = errors.New("invalid")
)
