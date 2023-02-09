package types

import (
	"errors"
)

var (
	ErrAttestationNotDataCommitmentRequest = errors.New("attestation is not a data commitment request")
	ErrUnknownAttestationType              = errors.New("unknown attestation type")
	ErrInvalidCommitmentInConfirm          = errors.New("confirm not carrying the right commitment for expected range")
	ErrInvalid                             = errors.New("invalid")
	ErrAttestationNotFound                 = errors.New("attestation not found")
	ErrUnmarshalValset                     = errors.New("couldn't unmarshal valset")
	ErrAttestationNotValsetRequest         = errors.New("attestation is not a valset request")
)
