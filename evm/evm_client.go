package evm

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	gethcommon "github.com/ethereum/go-ethereum/common"
	coregethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	tmlog "github.com/tendermint/tendermint/libs/log"

	blobstreamwrapper "github.com/celestiaorg/blobstream-contracts/v4/wrappers/Blobstream.sol"
	proxywrapper "github.com/celestiaorg/blobstream-contracts/v4/wrappers/ERC1967Proxy.sol"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// DefaultEVMGasLimit the default gas limit to use when sending transactions to the EVM chain.
const DefaultEVMGasLimit = uint64(2500000)

type Client struct {
	logger   tmlog.Logger
	Wrapper  *blobstreamwrapper.Wrappers
	Ks       *keystore.KeyStore
	Acc      *accounts.Account
	EvmRPC   string
	GasLimit uint64
}

// NewClient Creates a new EVM Client that can be used to deploy the Blobstream contract and
// interact with it.
// The wrapper parameter can be nil when creating the client for contract deployment.
func NewClient(
	logger tmlog.Logger,
	wrapper *blobstreamwrapper.Wrappers,
	ks *keystore.KeyStore,
	acc *accounts.Account,
	evmRPC string,
	gasLimit uint64,
) *Client {
	return &Client{
		logger:   logger,
		Wrapper:  wrapper,
		Ks:       ks,
		Acc:      acc,
		EvmRPC:   evmRPC,
		GasLimit: gasLimit,
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

// DeployBlobstreamContract Deploys the Blobstream contract and initializes it with the provided valset.
// The waitToBeMined, when set to true, will wait for the transaction to be included in a block,
// and log relevant information.
// The initBridge, when set to true, will assign the newly deployed bridge to the wrapper. This
// can be used later for further interactions with the new contract.
// Multiple calls to DeployBlobstreamContract with the initBridge flag set to true will overwrite everytime
// the bridge contract.
func (ec *Client) DeployBlobstreamContract(
	opts *bind.TransactOpts,
	contractBackend bind.ContractBackend,
	contractInitValset types.Valset,
	contractInitNonce uint64,
	initBridge bool,
) (gethcommon.Address, *coregethtypes.Transaction, *blobstreamwrapper.Wrappers, error) {
	// deploy the Blobstream implementation contract
	impAddr, impTx, _, err := ec.DeployImplementation(opts, contractBackend)
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	ec.logger.Info("deploying Blobstream implementation contract...", "address", impAddr.Hex(), "tx_hash", impTx.Hash().Hex())

	// encode the Blobstream contract initialization data using the chain parameters
	ethVsCheckpoint, err := contractInitValset.SignBytes()
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}
	blobStreamABI, err := blobstreamwrapper.WrappersMetaData.GetAbi()
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}
	initData, err := blobStreamABI.Pack("initialize", big.NewInt(int64(contractInitNonce)), big.NewInt(int64(contractInitValset.TwoThirdsThreshold())), ethVsCheckpoint)
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	// bump the nonce
	if opts.Nonce != nil {
		opts.Nonce.Add(opts.Nonce, big.NewInt(1))
	}

	// deploy the ERC1967 proxy, link it to the Blobstream implementation contract, and initialize it
	proxyAddr, tx, _, err := ec.DeployERC1867Proxy(opts, contractBackend, impAddr, initData)
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	ec.logger.Info("deploying Blobstream proxy contract...", "address", proxyAddr, "tx_hash", tx.Hash().Hex())

	bridge, err := blobstreamwrapper.NewWrappers(proxyAddr, contractBackend)
	if err != nil {
		return gethcommon.Address{}, nil, nil, err
	}

	if initBridge {
		// initializing the bridge
		ec.Wrapper = bridge
	}

	return proxyAddr, tx, bridge, nil
}

func (ec *Client) UpdateValidatorSet(
	opts *bind.TransactOpts,
	newNonce, newThreshHold uint64,
	currentValset, newValset types.Valset,
	sigs []blobstreamwrapper.Signature,
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
	sigs []blobstreamwrapper.Signature,
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
	builder := newTransactOptsBuilder(ec.Ks, ec.Acc)

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

func (ec *Client) StateLastValidatorSetCheckpoint(opts *bind.CallOpts) ([32]byte, error) {
	checkpoint, err := ec.Wrapper.StateLastValidatorSetCheckpoint(opts)
	if err != nil {
		return [32]byte{}, err
	}
	return checkpoint, nil
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

func (ec *Client) DeployImplementation(opts *bind.TransactOpts, backend bind.ContractBackend) (
	gethcommon.Address,
	*coregethtypes.Transaction,
	*blobstreamwrapper.Wrappers,
	error,
) {
	return blobstreamwrapper.DeployWrappers(
		opts,
		backend,
	)
}

func (ec *Client) DeployERC1867Proxy(
	opts *bind.TransactOpts,
	backend bind.ContractBackend,
	implementationAddress gethcommon.Address,
	data []byte,
) (gethcommon.Address, *coregethtypes.Transaction, *proxywrapper.Wrappers, error) {
	return proxywrapper.DeployWrappers(
		opts,
		backend,
		implementationAddress,
		data,
	)
}

func ethValset(valset types.Valset) ([]blobstreamwrapper.Validator, error) {
	ethVals := make([]blobstreamwrapper.Validator, len(valset.Members))
	for i, v := range valset.Members {
		if ok := gethcommon.IsHexAddress(v.EvmAddress); !ok {
			return nil, errors.New("invalid ethereum address found in validator set")
		}
		addr := gethcommon.HexToAddress(v.EvmAddress)
		ethVals[i] = blobstreamwrapper.Validator{
			Addr:  addr,
			Power: big.NewInt(int64(v.Power)),
		}
	}
	return ethVals, nil
}
