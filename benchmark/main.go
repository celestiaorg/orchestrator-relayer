package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"syscall"
	"time"

	wrappers "github.com/celestiaorg/blobstream-contracts/v4/wrappers/Blobstream.sol"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/store"
	orchtypes "github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

const (
	NumberOfValidators       = 100
	NumberOfNecessarySigners = 30
	TotalValidatorPower      = int64(100000)
	FundedPrivateKeyInHex    = "funded_evm_key"
	EVMRPC                   = "evm_network_rpc"
)

type WrappedValidator struct {
	Validator  wrappers.Validator
	PrivateKey ecdsa.PrivateKey
	Account    accounts.Account
}

func main() {
	err := DeployContractAndSubmitDataCommitment()
	if err != nil {
		fmt.Println(err)
		syscall.Exit(1)
	}
}

func DeployContractAndSubmitDataCommitment() error {
	logger := tmlog.NewTMLogger(os.Stdout)

	path, err := os.MkdirTemp(os.TempDir(), "qgb_bench")
	if err != nil {
		return nil
	}

	ks := keystore.NewKeyStore(filepath.Join(path, store.EVMKeyStorePath), keystore.LightScryptN, keystore.LightScryptP)

	valsetValidators, err := generateRandomValidators(ks, NumberOfValidators, NumberOfNecessarySigners)
	if err != nil {
		return nil
	}
	bridgeValidators := ToBridgeValidators(valsetValidators)

	ethPrivKey, err := ethcrypto.HexToECDSA(FundedPrivateKeyInHex)
	if err != nil {
		return err
	}

	acc, err := ks.ImportECDSA(ethPrivKey, "1234")
	if err != nil {
		return err
	}
	err = ks.Unlock(acc, "1234")
	if err != nil {
		return err
	}

	logger.Info("loading EVM account", "address", acc.Address.Hex())

	ctx := context.Background()

	evmClient := evm.NewClient(
		tmlog.NewTMLogger(os.Stdout),
		nil,
		ks,
		&acc,
		EVMRPC,
		2500000,
	)

	txOpts, err := evmClient.NewTransactionOpts(ctx)
	if err != nil {
		return err
	}

	backend, err := evmClient.NewEthClient()
	if err != nil {
		return err
	}
	defer backend.Close()

	vs := types.Valset{
		Nonce:   1,
		Members: bridgeValidators,
		Height:  1,
		Time:    time.Now(),
	}

	address, tx, bridge, err := evmClient.DeployBlobstreamContract(txOpts, backend, vs, vs.Nonce, true)
	if err != nil {
		logger.Error("failed to deploy QGB contract")
		return err
	}

	receipt, err := evmClient.WaitForTransaction(ctx, backend, tx, time.Minute)
	if err == nil && receipt != nil && receipt.Status == 1 {
		logger.Info("deployed QGB contract", "proxy_address", address.Hex(), "tx_hash", tx.Hash().String())
	}

	txOpts.Nonce.Add(txOpts.Nonce, big.NewInt(1))

	commitment := []byte{0x12}
	dataRootHash := orchtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(2)), commitment)
	signatures := make([]wrappers.Signature, NumberOfValidators)
	cumulatedPower := int64(0)
	for i, val := range valsetValidators {
		if cumulatedPower > 2*TotalValidatorPower/3 {
			break
		}
		dcSig, err := evm.NewEthereumSignature(dataRootHash.Bytes(), ks, val.Account)
		if err != nil {
			return err
		}
		v, r, s, err := evm.SigToVRS(ethcmn.Bytes2Hex(dcSig))
		if err != nil {
			return err
		}
		signatures[i] = wrappers.Signature{
			V: v,
			R: r,
			S: s,
		}
		cumulatedPower += val.Validator.Power.Int64()
	}
	wrapperValidators := ToWrapperValidators(valsetValidators)
	submitTx, err := bridge.SubmitDataRootTupleRoot(txOpts, big.NewInt(2), big.NewInt(1), [32]byte{0x12}, wrapperValidators, signatures)
	if err != nil {
		return err
	}
	logger.Info("submitted data root tuple root", "tx_hash", submitTx.Hash().String())
	return nil
}

func generateRandomValidators(s *keystore.KeyStore, numberOfValidators, numberOfNecessarySigners int64) ([]WrappedValidator, error) {
	validators := make([]WrappedValidator, numberOfValidators)
	threshold := 2 * TotalValidatorPower / 3
	primaryPowers := threshold / (numberOfNecessarySigners - 1)
	secondaryPowers := (TotalValidatorPower - threshold) / (numberOfValidators - numberOfNecessarySigners + 1)
	for i := int64(0); i < numberOfValidators; i++ {
		pKey, err := ethcrypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		address := ethcrypto.PubkeyToAddress(pKey.PublicKey)
		if i < numberOfNecessarySigners {
			validators[i] = WrappedValidator{
				Validator: wrappers.Validator{
					Addr:  address,
					Power: big.NewInt(primaryPowers),
				},
				PrivateKey: *pKey,
			}
		} else {
			validators[i] = WrappedValidator{
				Validator: wrappers.Validator{
					Addr:  address,
					Power: big.NewInt(secondaryPowers),
				},
				PrivateKey: *pKey,
			}
		}
		account, err := s.ImportECDSA(pKey, "1234")
		if err != nil {
			return nil, err
		}
		err = s.Unlock(account, "1234")
		if err != nil {
			return nil, err
		}
		validators[i].Account = account
	}
	return validators, nil
}

func ToBridgeValidators(validators []WrappedValidator) types.BridgeValidators {
	bridgedValidators := make(types.BridgeValidators, len(validators))
	for i, val := range validators {
		bridgedValidators[i] = types.BridgeValidator{
			Power:      val.Validator.Power.Uint64(),
			EvmAddress: val.Validator.Addr.Hex(),
		}
	}
	return bridgedValidators
}

func ToWrapperValidators(validators []WrappedValidator) []wrappers.Validator {
	wrapperValidators := make([]wrappers.Validator, len(validators))
	for i, val := range validators {
		wrapperValidators[i] = wrappers.Validator{
			Power: val.Validator.Power,
			Addr:  val.Validator.Addr,
		}
	}
	return wrapperValidators
}
