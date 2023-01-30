package relayer

import (
	"errors"
)

var (
	ErrAttestationNotValsetRequest         = errors.New("attestation is not a valset request")
	ErrAttestationNotDataCommitmentRequest = errors.New("attestation is not a data commitment request")
	ErrAttestationNotFound                 = errors.New("attestation not found")
)
