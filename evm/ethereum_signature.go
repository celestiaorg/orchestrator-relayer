package evm

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/pkg/errors"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	signaturePrefix = "\x19Ethereum Signed Message:\n32"
)

// NewEthereumSignature creates a new eip-191 signature over a given byte array.
// hash: digest to be signed over.
// ks: the keystore to use for the signature
// acc: the account in the keystore to use for the signature
func NewEthereumSignature(hash []byte, ks *keystore.KeyStore, acc accounts.Account) ([]byte, error) {
	if ks == nil {
		return nil, errors.Wrap(celestiatypes.ErrEmpty, "nil keystore")
	}
	protectedHash := crypto.Keccak256Hash([]uint8(signaturePrefix), hash)
	return ks.SignHash(acc, protectedHash.Bytes())
}

func EthAddressFromSignature(hash []byte, signature []byte) (common.Address, error) {
	if len(signature) < 65 {
		return common.Address{}, errors.Wrap(ErrInvalid, "signature too short")
	}
	protectedHash := crypto.Keccak256Hash([]uint8(signaturePrefix), hash)
	sigPublicKey, err := crypto.Ecrecover(protectedHash.Bytes(), signature)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "ec recover failed")
	}
	pubKey, err := crypto.UnmarshalPubkey(sigPublicKey)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "unmarshalling signature public key failed")
	}
	addr := crypto.PubkeyToAddress(*pubKey)
	return addr, nil
}

// ValidateEthereumSignature takes a message, an associated signature and public key and
// returns an error if the signature isn't valid.
func ValidateEthereumSignature(hash []byte, signature []byte, ethAddress common.Address) error {
	addr, err := EthAddressFromSignature(hash, signature)
	if err != nil {
		return errors.Wrap(err, "unable to get address from signature")
	}

	if addr.Hex() != ethAddress.Hex() {
		return errors.Wrap(ErrInvalid, "signature not matching")
	}

	return nil
}
