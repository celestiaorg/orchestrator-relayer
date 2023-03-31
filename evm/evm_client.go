package evm

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	coregethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

const DefaultGasLimit = uint64(25000000)

type Client struct {
	logger     tmlog.Logger
	Wrapper    *wrapper.QuantumGravityBridge
	PrivateKey *ecdsa.PrivateKey
	EvmRPC     string
	GasLimit   uint64
}

// NewClient Creates a new EVM Client that can be used to deploy the QGB contract and
// interact with it.
// The wrapper parameter can be nil when creating the client for contract deployment.
func NewClient(
	logger tmlog.Logger,
	wrapper *wrapper.QuantumGravityBridge,
	privateKey *ecdsa.PrivateKey,
	evmRPC string,
	gasLimit uint64,
) *Client {
	return &Client{
		logger:     logger,
		Wrapper:    wrapper,
		PrivateKey: privateKey,
		EvmRPC:     evmRPC,
		GasLimit:   gasLimit,
	}
}

// NewEthClient creates a new Eth client using the existing EVM RPC address.
// Should be closed after usage.
func (ec *Client) NewEthClient() (*ethclient.Client, error) {
	ethClient, err := ethclient.Dial(ec.EvmRPC)
	if err != nil {
		return nil, err
	}
	return ethClient, nil
}

// DeployQGBContract Deploys the QGB contract and initializes it with the provided valset.
// The waitToBeMined, when set to true, will wait for the transaction to be included in a block,
// and log relevant information.
// The initBridge, when set to true, will assign the newly deployed bridge to the wrapper. This
// later can be used for further interactions with the new contract.
// Multiple calls to DeployQGBContract with the initBridge flag set to true will overwrite everytime
// the bridge contract.
func (ec *Client) DeployQGBContract(
	opts *bind.TransactOpts,
	backend bind.ContractBackend,
	contractInitValset types.Valset,
	contractInitNonce uint64,
	initBridge bool,
) (gethcommon.Address, *coregethtypes.Transaction, *wrapper.QuantumGravityBridge, error) {
	ethVsHash, err := contractInitValset.Hash()
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	// deploy the QGB contract using the chain parameters
	addr, tx, bridge, err := wrapper.DeployQuantumGravityBridge(
		opts,
		backend,
		big.NewInt(int64(contractInitNonce)),
		big.NewInt(int64(contractInitValset.TwoThirdsThreshold())),
		ethVsHash,
	)
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	if initBridge {
		// initializing the bridge
		ec.Wrapper = bridge
	}

	return addr, tx, bridge, nil
}

func (ec *Client) UpdateValidatorSet(
	opts *bind.TransactOpts,
	newNonce, newThreshHold uint64,
	currentValset, newValset types.Valset,
	sigs []wrapper.Signature,
) (*coregethtypes.Transaction, error) {
	// TODO in addition to the nonce, log more interesting information
	ec.logger.Info("relaying valset", "nonce", newNonce)

	ethVals, err := ethValset(currentValset)
	if err != nil {
		return nil, err
	}

	ethVsHash, err := newValset.Hash()
	if err != nil {
		return nil, err
	}

	var currentNonce uint64
	if newValset.Nonce == 1 {
		currentNonce = 0
	} else {
		currentNonce = currentValset.Nonce
	}

	tx, err := ec.Wrapper.UpdateValidatorSet(
		opts,
		big.NewInt(int64(newNonce)),
		big.NewInt(int64(currentNonce)),
		big.NewInt(int64(newThreshHold)),
		ethVsHash,
		ethVals,
		sigs,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (ec *Client) SubmitDataRootTupleRoot(
	opts *bind.TransactOpts,
	tupleRoot gethcommon.Hash,
	newNonce uint64,
	currentValset types.Valset,
	sigs []wrapper.Signature,
) (*coregethtypes.Transaction, error) {
	ethVals, err := ethValset(currentValset)
	if err != nil {
		return nil, err
	}

	tx, err := ec.Wrapper.SubmitDataRootTupleRoot(
		opts,
		big.NewInt(int64(newNonce)),
		big.NewInt(int64(currentValset.Nonce)),
		tupleRoot,
		ethVals,
		sigs,
	)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// NewTransactionOpts creates a new transaction Opts to be used when submitting transactions.
func (ec *Client) NewTransactionOpts(ctx context.Context) (*bind.TransactOpts, error) {
	builder := newTransactOptsBuilder(ec.PrivateKey)

	ethClient, err := ethclient.Dial(ec.EvmRPC)
	if err != nil {
		return nil, err
	}

	opts, err := builder(ctx, ethClient, ec.GasLimit)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (ec *Client) StateLastEventNonce(opts *bind.CallOpts) (uint64, error) {
	nonce, err := ec.Wrapper.StateEventNonce(opts)
	if err != nil {
		return 0, err
	}
	return nonce.Uint64(), nil
}

func (ec *Client) WaitForTransaction(
	ctx context.Context,
	backend bind.DeployBackend,
	tx *coregethtypes.Transaction,
) (*coregethtypes.Receipt, error) {
	ec.logger.Debug("waiting for transaction to be confirmed", "hash", tx.Hash().String())

	receipt, err := bind.WaitMined(ctx, backend, tx)
	if err == nil && receipt != nil && receipt.Status == 1 {
		ec.logger.Info("transaction confirmed", "hash", tx.Hash().String(), "block", receipt.BlockNumber.Uint64())
		return receipt, nil
	}
	ec.logger.Error("transaction failed", "hash", tx.Hash().String())

	return receipt, err
}

func ethValset(valset types.Valset) ([]wrapper.Validator, error) {
	ethVals := make([]wrapper.Validator, len(valset.Members))
	for i, v := range valset.Members {
		if ok := gethcommon.IsHexAddress(v.EvmAddress); !ok {
			return nil, errors.New("invalid ethereum address found in validator set")
		}
		addr := gethcommon.HexToAddress(v.EvmAddress)
		ethVals[i] = wrapper.Validator{
			Addr:  addr,
			Power: big.NewInt(int64(v.Power)),
		}
	}
	return ethVals, nil
}
