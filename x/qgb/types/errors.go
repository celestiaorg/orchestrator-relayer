package types

import (
	"errors"
)

var (
	ErrInvalid                               = errors.New("invalid")
	ErrDuplicate                             = errors.New("duplicate")
	ErrUnknown                               = errors.New("unknown")
	ErrEmpty                                 = errors.New("empty")
	ErrResetDelegateKeys                     = errors.New("can not set orchestrator addresses more than once")
	ErrNoValidators                          = errors.New("no bonded validators in active set")
	ErrInvalidValAddress                     = errors.New("invalid validator address in current valset %v")
	ErrInvalidEVMAddress                     = errors.New("discovered invalid EVM address stored for validator %v")
	ErrInvalidValset                         = errors.New("generated invalid valset")
	ErrAttestationNotValsetRequest           = errors.New("attestation is not a valset request")
	ErrAttestationNotDataCommitmentRequest   = errors.New("attestation is not a data commitment request")
	ErrAttestationNotFound                   = errors.New("attestation not found")
	ErrNilDataCommitmentRequest              = errors.New("data commitment cannot be nil when setting attestation") //nolint:lll
	ErrNilValsetRequest                      = errors.New("valset cannot be nil when setting attestation")
	ErrValidatorNotInValset                  = errors.New("validator signing is not in the valset")
	ErrNilAttestation                        = errors.New("nil attestation")
	ErrAttestationNotCastToDataCommitment    = errors.New("couldn't cast attestation to data commitment")
	ErrDataCommitmentConfirmWrongRange       = errors.New("the confirm range is different than the requested one")
	ErrWindowNotFound                        = errors.New("couldn't find data commitment window in store")
	ErrUnmarshalllAttestation                = errors.New("couldn't unmarshall attestation from store")
	ErrNonceHigherThanLatestAttestationNonce = errors.New("the provided nonce is higher than the latest attestation nonce")
	ErrNoValsetBeforeNonceOne                = errors.New("there is no valset before attestation nonce 1")
)
