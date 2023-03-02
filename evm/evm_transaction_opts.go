package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TODO: make gas price configurable.
type transactOpsBuilder func(ctx context.Context, client *ethclient.Client, gasLim uint64) (*bind.TransactOpts, error)

func newTransactOptsBuilder(privKey *ecdsa.PrivateKey) transactOpsBuilder {
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		panic(fmt.Errorf("invalid public key; expected: %T, got: %T", &ecdsa.PublicKey{}, publicKey))
	}

	evmAddress := ethcrypto.PubkeyToAddress(*publicKeyECDSA)
	return func(ctx context.Context, client *ethclient.Client, gasLim uint64) (*bind.TransactOpts, error) {
		nonce, err := client.PendingNonceAt(ctx, evmAddress)
		if err != nil {
			return nil, err
		}

		ethChainID, err := client.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get Ethereum chain ID: %w", err)
		}

		auth, err := bind.NewKeyedTransactorWithChainID(privKey, ethChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ethereum transactor: %w", err)
		}

		bigGasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get Ethereum gas estimate: %w", err)
		}

		auth.Nonce = new(big.Int).SetUint64(nonce)
		auth.Value = big.NewInt(0) // in wei
		auth.GasLimit = gasLim     // in units
		auth.GasPrice = bigGasPrice

		return auth, nil
	}
}

const (
	MalleabilityThreshold = "0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0"
	ZeroSValue            = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

// SigToVRS breaks apart a signature into its components to make it compatible with the contracts
func SigToVRS(sigHex string) (v uint8, r, s ethcmn.Hash, err error) {
	signatureBytes := ethcmn.FromHex(strings.ToLower(sigHex))

	// signature length should be 65: 32 bytes + vParam
	if len(signatureBytes) != 65 {
		err = errors.Wrap(ErrInvalid, "signature length")
		return
	}

	// vParam should be 0, 1, 27 or 28
	vParam := signatureBytes[64]
	switch vParam {
	case byte(0):
		vParam = byte(27)
	case byte(1):
		vParam = byte(28)
	case byte(27):
	case byte(28):
	default:
		err = errors.Wrap(ErrInvalid, "signature vParam. Should be 0, 1, 27 or 28")
		return
	}

	v = vParam
	r = ethcmn.BytesToHash(signatureBytes[0:32])
	s = ethcmn.BytesToHash(signatureBytes[32:64])

	// sValue shouldn't be malleable
	if MalleabilityThreshold <= s.String() || s.String() == ZeroSValue {
		err = errors.Wrap(ErrInvalid, "signature s. Should be 0 < s < secp256k1n ÷ 2 + 1")
		return
	}

	return
}
