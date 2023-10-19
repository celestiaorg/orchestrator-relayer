package e2e

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/celestiaorg/celestia-app/pkg/user"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	blobstreamwrapper "github.com/celestiaorg/blobstream-contracts/v3/wrappers/Blobstream.sol"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	blobstreamtypes "github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client/http"
	testcontainers "github.com/testcontainers/testcontainers-go/modules/compose"
)

type BlobstreamNetwork struct {
	ComposePaths  []string
	Identifier    string
	Instance      *testcontainers.LocalDockerCompose
	EVMRPC        string
	TendermintRPC string
	CelestiaGRPC  string
	P2PAddr       string
	EncCfg        encoding.Config
	Logger        tmlog.Logger

	// used by the moderator to notify all the workers.
	stopChan <-chan struct{}
	// used by the workers to notify the moderator.
	toStopChan chan<- struct{}
}

func NewBlobstreamNetwork() (*BlobstreamNetwork, error) {
	id := strings.ToLower(uuid.New().String())
	paths := []string{"./docker-compose.yml"}
	instance := testcontainers.NewLocalDockerCompose(paths, id) //nolint:staticcheck
	stopChan := make(chan struct{})
	// given an initial capacity to avoid blocking in case multiple services failed
	// and wanted to notify the moderator.
	toStopChan := make(chan struct{}, 10)
	network := &BlobstreamNetwork{
		Identifier:    id,
		ComposePaths:  paths,
		Instance:      instance,
		EVMRPC:        "http://localhost:8545",
		TendermintRPC: "tcp://localhost:26657",
		CelestiaGRPC:  "localhost:9090",
		P2PAddr:       "localhost:30000",
		EncCfg:        encoding.MakeConfig(app.ModuleEncodingRegisters...),
		stopChan:      stopChan,
		toStopChan:    toStopChan,
	}

	// moderate stop notifications from waiters.
	registerModerator(stopChan, toStopChan)

	// trap SIGINT
	// helps release the docker resources without having to do it manually.
	registerGracefulExit(network)

	return network, nil
}

// registerModerator handles stop signals from a worker and notifies the others to stop.
func registerModerator(stopChan chan<- struct{}, toStopChan <-chan struct{}) {
	go func() {
		<-toStopChan
		stopChan <- struct{}{}
	}()
}

// registerGracefulExit traps SIGINT or waits for ctx.Done() to release the docker resources before exiting
// it is not calling `DeleteAll()` here as it is being called inside the tests. No need to call it two times.
// this comes from the fact that we're sticking with unit tests style tests to be able to run individual tests
// https://github.com/celestiaorg/celestia-app/issues/428
func registerGracefulExit(network *BlobstreamNetwork) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		if <-c; true {
			network.toStopChan <- struct{}{}
			forceExitIfNeeded(1)
		}
	}()
}

// forceExitIfNeeded forces stopping the network is SIGINT is sent a second time.
func forceExitIfNeeded(exitCode int) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	if <-c; true {
		fmt.Println("forcing exit. some resources might not have been cleaned.")
		os.Exit(exitCode)
	}
}

// StartAll starts the whole Blobstream cluster with multiple validators, orchestrators and a relayer
// Make sure to release the resources after finishing by calling the `StopAll()` method.
func (network BlobstreamNetwork) StartAll() error {
	// the reason for building before executing `up` is to avoid rebuilding all the images
	// if some container accidentally changed some files when running.
	// This to speed up a bit the execution.
	fmt.Println("building images...")
	err := network.Instance.
		WithCommand([]string{"build", "--quiet"}).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"up", "--no-build", "-d"}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// StopAll stops the network and leaves the containers created. This allows to resume
// execution from the point where they stopped.
func (network BlobstreamNetwork) StopAll() error {
	err := network.Instance.
		WithCommand([]string{"stop"}).
		Invoke()
	if err.Error != nil {
		return err.Error
	}
	return nil
}

// DeleteAll deletes the containers, network and everything related to the cluster.
func (network BlobstreamNetwork) DeleteAll() error {
	err := network.Instance.
		WithCommand([]string{"down"}).
		Invoke()
	if err.Error != nil {
		return err.Error
	}
	return nil
}

// KillAll kills all the containers.
func (network BlobstreamNetwork) KillAll() error {
	err := network.Instance.
		WithCommand([]string{"kill"}).
		Invoke()
	if err.Error != nil {
		return err.Error
	}
	return nil
}

// Start starts a service from the `Service` enum. Make sure to call `Stop`, in the
// end, to release the resources.
func (network BlobstreamNetwork) Start(service Service) error {
	serviceName, err := service.toString()
	if err != nil {
		return err
	}
	fmt.Println("building images...")
	err = network.Instance.
		WithCommand([]string{"build", "--quiet", serviceName}).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"up", "--no-build", "-d", serviceName}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// DeployBlobstreamContract uses the Deployer service to deploy a new Blobstream contract
// based on the existing running network. If no Celestia-app nor ganache is
// started, it creates them automatically.
func (network BlobstreamNetwork) DeployBlobstreamContract() error {
	fmt.Println("building images...")
	err := network.Instance.
		WithCommand([]string{"build", "--quiet", DEPLOYER}).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"run", "-e", "DEPLOY_NEW_CONTRACT=true", DEPLOYER}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// StartMultiple start multiple services. Make sure to call `Stop`, in the
// end, to release the resources.
func (network BlobstreamNetwork) StartMultiple(services ...Service) error {
	if len(services) == 0 {
		return fmt.Errorf("empty list of services provided")
	}
	serviceNames := make([]string, 0)
	for _, s := range services {
		name, err := s.toString()
		if err != nil {
			return err
		}
		serviceNames = append(serviceNames, name)
	}
	fmt.Println("building images...")
	err := network.Instance.
		WithCommand(append([]string{"build", "--quiet"}, serviceNames...)).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand(append([]string{"up", "--no-build", "-d"}, serviceNames...)).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

func (network BlobstreamNetwork) Stop(service Service) error {
	serviceName, err := service.toString()
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"stop", serviceName}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// StopMultiple start multiple services. Make sure to call `Stop` or `StopMultiple`, in the
// end, to release the resources.
func (network BlobstreamNetwork) StopMultiple(services ...Service) error {
	if len(services) == 0 {
		return fmt.Errorf("empty list of services provided")
	}
	serviceNames := make([]string, 0)
	for _, s := range services {
		name, err := s.toString()
		if err != nil {
			return err
		}
		serviceNames = append(serviceNames, name)
	}
	err := network.Instance.
		WithCommand(append([]string{"up", "-d"}, serviceNames...)).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

func (network BlobstreamNetwork) ExecCommand(service Service, command []string) error {
	serviceName, err := service.toString()
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand(append([]string{"exec", serviceName}, command...)).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// StartMinimal starts a network containing: 1 validator, 1 orchestrator, 1 relayer
// and a ganache instance.
func (network BlobstreamNetwork) StartMinimal() error {
	fmt.Println("building images...")
	err := network.Instance.
		WithCommand([]string{"build", "--quiet", "core0", "core0-orch", "relayer", "ganache"}).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"up", "--no-build", "-d", "core0", "core0-orch", "relayer", "ganache"}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

// StartBase starts the very minimal component to have a network.
// It consists of starting `core0` as it is the genesis validator, and the docker network
// will be created along with it, allowing more containers to join it.
func (network BlobstreamNetwork) StartBase() error {
	fmt.Println("building images...")
	err := network.Instance.
		WithCommand([]string{"build", "--quiet", "core0"}).
		Invoke().Error
	if err != nil {
		return err
	}
	err = network.Instance.
		WithCommand([]string{"up", "-d", "--no-build", "core0"}).
		Invoke().Error
	if err != nil {
		return err
	}
	return nil
}

func (network BlobstreamNetwork) WaitForNodeToStart(_ctx context.Context, rpcAddr string) error {
	ctx, cancel := context.WithTimeout(_ctx, 5*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("node %s not initialized in time", rpcAddr)
			}
			return ctx.Err()
		default:
			trpc, err := http.New(rpcAddr, "/websocket")
			if err != nil || trpc.Start() != nil {
				fmt.Println("waiting for node to start...")
				time.Sleep(5 * time.Second)
				continue
			}
			cancel()
			return nil
		}
	}
}

func (network BlobstreamNetwork) WaitForBlock(_ctx context.Context, height int64) error {
	return network.WaitForBlockWithCustomTimeout(_ctx, height, 5*time.Minute)
}

func (network BlobstreamNetwork) WaitForBlockWithCustomTimeout(
	_ctx context.Context,
	height int64,
	timeout time.Duration,
) error {
	err := network.WaitForNodeToStart(_ctx, network.TendermintRPC)
	if err != nil {
		return err
	}
	trpc, err := http.New(network.TendermintRPC, "/websocket")
	if err != nil {
		return err
	}
	err = trpc.Start()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(_ctx, timeout)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf(" chain didn't reach height in time")
			}
			return ctx.Err()
		default:
			status, err := trpc.Status(ctx)
			if err != nil {
				continue
			}
			if status.SyncInfo.LatestBlockHeight >= height {
				cancel()
				return nil
			}
			fmt.Printf("current height: %d\n", status.SyncInfo.LatestBlockHeight)
			time.Sleep(5 * time.Second)
		}
	}
}

// WaitForOrchestratorToStart waits for the orchestrator having the evm address `evmAddr`
// to sign the first data commitment (could be upgraded to get any signature, either valset or data commitment,
// and for any nonce, but would require adding a new method to the querier. Don't think it is worth it now as
// the number of valsets that will be signed is trivial and reaching 0 would be in no time).
// Returns the height and the nonce of some attestation that the orchestrator signed.
func (network BlobstreamNetwork) WaitForOrchestratorToStart(_ctx context.Context, dht *p2p.BlobstreamDHT, evmAddr string) (uint64, uint64, error) {
	// create p2p querier
	p2pQuerier := p2p.NewQuerier(dht, network.Logger)

	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return 0, 0, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	tmQuerier := rpc.NewTmQuerier(network.TendermintRPC, network.Logger)
	err = tmQuerier.Start()
	if err != nil {
		return 0, 0, err
	}
	defer tmQuerier.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(_ctx, 5*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return 0, 0, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return 0, 0, fmt.Errorf("orchestrator didn't start correctly")
			}
			return 0, 0, ctx.Err()
		default:
			fmt.Println("waiting for orchestrator to start ...")
			lastNonce, err := appQuerier.QueryLatestAttestationNonce(ctx)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for i := uint64(0); i < lastNonce; i++ {
				att, err := appQuerier.QueryAttestationByNonce(ctx, lastNonce-i)
				if err != nil {
					continue
				}
				switch castedAtt := att.(type) {
				case *types.Valset:
					signBytes, err := castedAtt.SignBytes()
					if err != nil {
						continue
					}
					vsConfirm, err := p2pQuerier.QueryValsetConfirmByEVMAddress(ctx, lastNonce-i, evmAddr, signBytes.Hex())
					if err == nil && vsConfirm != nil {
						cancel()
						return castedAtt.Height, castedAtt.Nonce, nil
					}
				case *types.DataCommitment:
					commitment, err := tmQuerier.QueryCommitment(ctx, castedAtt.BeginBlock, castedAtt.EndBlock)
					if err != nil {
						continue
					}
					dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
					dcConfirm, err := p2pQuerier.QueryDataCommitmentConfirmByEVMAddress(ctx, lastNonce-i, evmAddr, dataRootTupleRoot.Hex())
					if err == nil && dcConfirm != nil {
						cancel()
						return castedAtt.EndBlock, castedAtt.Nonce, nil
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// GetValsetContainingVals Gets the last valset that contains a certain number of validator.
// This is used after enabling orchestrators not to sign unless they belong to some valset.
// Thus, any nonce after the returned valset should be signed by all orchestrators.
func (network BlobstreamNetwork) GetValsetContainingVals(_ctx context.Context, number int) (*types.Valset, error) {
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(_ctx, 5*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return nil, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("couldn't find any valset containing %d validators", number)
			}
			return nil, ctx.Err()
		default:
			fmt.Printf("searching for valset with %d validator...\n", number)
			lastNonce, err := appQuerier.QueryLatestAttestationNonce(ctx)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for i := uint64(0); i < lastNonce; i++ {
				vs, err := appQuerier.QueryValsetByNonce(ctx, lastNonce-i)
				if err == nil && vs != nil && len(vs.Members) == number {
					cancel()
					return vs, nil
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// GetValsetConfirm Returns the valset confirm for nonce `nonce`
// signed by orchestrator whose EVM address is `evmAddr`.
func (network BlobstreamNetwork) GetValsetConfirm(
	_ctx context.Context,
	dht *p2p.BlobstreamDHT,
	nonce uint64,
	evmAddr string,
) (*blobstreamtypes.ValsetConfirm, error) {
	p2pQuerier := p2p.NewQuerier(dht, network.Logger)
	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(_ctx, 2*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return nil, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("couldn't find confirm for nonce=%d", nonce)
			}
			return nil, ctx.Err()
		default:
			vs, err := appQuerier.QueryValsetByNonce(ctx, nonce)
			if err != nil {
				fmt.Printf("waiting for confirm for nonce=%d\n", nonce)
				time.Sleep(5 * time.Second)
				continue
			}
			signBytes, err := vs.SignBytes()
			if err != nil {
				fmt.Printf("waiting for confirm for nonce=%d\n", nonce)
				time.Sleep(5 * time.Second)
				continue
			}
			resp, err := p2pQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, evmAddr, signBytes.Hex())
			if err == nil && resp != nil {
				cancel()
				return resp, nil
			}
			fmt.Printf("waiting for confirm for nonce=%d\n", nonce)
			time.Sleep(5 * time.Second)
		}
	}
}

// GetDataCommitmentConfirm Returns the data commitment confirm for nonce `nonce`
// signed by orchestrator whose EVM address is `evmAddr`.
func (network BlobstreamNetwork) GetDataCommitmentConfirm(
	_ctx context.Context,
	dht *p2p.BlobstreamDHT,
	nonce uint64,
	evmAddr string,
) (*blobstreamtypes.DataCommitmentConfirm, error) {
	// create p2p querier
	p2pQuerier := p2p.NewQuerier(dht, network.Logger)

	// creating an RPC connection to tendermint
	tmQuerier := rpc.NewTmQuerier(network.TendermintRPC, network.Logger)
	err := tmQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer tmQuerier.Stop() //nolint:errcheck

	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err = appQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(_ctx, 2*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return nil, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("couldn't find confirm for nonce=%d", nonce)
			}
			return nil, ctx.Err()
		default:
			dc, err := appQuerier.QueryDataCommitmentByNonce(ctx, nonce)
			if err != nil {
				continue
			}
			commitment, err := tmQuerier.QueryCommitment(ctx, dc.BeginBlock, dc.EndBlock)
			if err != nil {
				continue
			}
			dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), commitment)
			resp, err := p2pQuerier.QueryDataCommitmentConfirmByEVMAddress(ctx, nonce, evmAddr, dataRootTupleRoot.Hex())
			if err == nil && resp != nil {
				cancel()
				return resp, nil
			}
			fmt.Printf("waiting for confirm for nonce=%d\n", nonce)
			time.Sleep(5 * time.Second)
		}
	}
}

// GetDataCommitmentConfirmByHeight Returns the data commitment confirm that commits
// to height `height` signed by orchestrator whose EVM address is `evmAddr`.
func (network BlobstreamNetwork) GetDataCommitmentConfirmByHeight(
	_ctx context.Context,
	dht *p2p.BlobstreamDHT,
	height uint64,
	evmAddr string,
) (*blobstreamtypes.DataCommitmentConfirm, error) {
	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	attestation, err := appQuerier.QueryDataCommitmentForHeight(_ctx, height)
	if err != nil {
		return nil, err
	}
	dcConfirm, err := network.GetDataCommitmentConfirm(_ctx, dht, attestation.Nonce, evmAddr)
	if err != nil {
		return nil, err
	}
	return dcConfirm, nil
}

// GetLatestAttestationNonce Returns the latest attestation nonce.
func (network BlobstreamNetwork) GetLatestAttestationNonce(_ctx context.Context) (uint64, error) {
	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return 0, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	nonce, err := appQuerier.QueryLatestAttestationNonce(_ctx)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

// WasAttestationSigned Returns true if the attestation confirm exist.
func (network BlobstreamNetwork) WasAttestationSigned(
	_ctx context.Context,
	dht *p2p.BlobstreamDHT,
	nonce uint64,
	evmAddress string,
) (bool, error) {
	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return false, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	// create p2p querier
	p2pQuerier := p2p.NewQuerier(dht, network.Logger)

	// creating an RPC connection to tendermint
	tmQuerier := rpc.NewTmQuerier(network.TendermintRPC, network.Logger)
	err = tmQuerier.Start()
	if err != nil {
		return false, err
	}
	defer tmQuerier.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(_ctx, 2*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return false, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return false, fmt.Errorf("couldn't find confirm for nonce=%d", nonce)
			}
			return false, ctx.Err()
		default:
			att, err := appQuerier.QueryAttestationByNonce(ctx, nonce)
			if err != nil || att == nil {
				continue
			}
			switch castedAtt := att.(type) {
			case *types.Valset:
				signBytes, err := castedAtt.SignBytes()
				if err != nil {
					continue
				}
				resp, err := p2pQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, evmAddress, signBytes.Hex())
				if err == nil && resp != nil {
					cancel()
					return true, nil
				}

			case *types.DataCommitment:
				commitment, err := tmQuerier.QueryCommitment(ctx, castedAtt.BeginBlock, castedAtt.EndBlock)
				if err != nil {
					continue
				}
				dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
				resp, err := p2pQuerier.QueryDataCommitmentConfirmByEVMAddress(
					ctx,
					castedAtt.Nonce,
					evmAddress,
					dataRootTupleRoot.Hex(),
				)
				if err == nil && resp != nil {
					cancel()
					return true, nil
				}
			}
			fmt.Printf("waiting for confirm for nonce=%d\n", nonce)
			time.Sleep(5 * time.Second)
		}
	}
}

func (network BlobstreamNetwork) GetLatestDeployedBlobstreamContract(_ctx context.Context) (*blobstreamwrapper.Wrappers, error) {
	return network.GetLatestDeployedBlobstreamContractWithCustomTimeout(_ctx, 5*time.Minute)
}

func (network BlobstreamNetwork) GetLatestDeployedBlobstreamContractWithCustomTimeout(
	_ctx context.Context,
	timeout time.Duration,
) (*blobstreamwrapper.Wrappers, error) {
	client, err := ethclient.Dial(network.EVMRPC)
	if err != nil {
		return nil, err
	}
	height := 0
	ctx, cancel := context.WithTimeout(_ctx, timeout)
	implementationFound := false
	for {
		select {
		case <-network.stopChan:
			cancel()
			return nil, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("timeout. couldn't find deployed blobstream contract")
			}
			return nil, ctx.Err()
		default:
			currentHeight, err := client.BlockNumber(ctx)
			if err != nil {
				cancel()
				return nil, err
			}
			if currentHeight < uint64(height) {
				time.Sleep(2 * time.Second)
				continue
			}
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(height)))
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}
			height++
			for _, tx := range block.Transactions() {
				// If the tx.To is not nil, then it's not a contract creation transaction
				if tx.To() != nil {
					continue
				}
				receipt, err := client.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					cancel()
					return nil, err
				}
				// TODO check if this check is actually checking if it's
				// If the contract address is 0s or empty, then it's not a contract creation transaction
				if receipt.ContractAddress == (ethcommon.Address{}) {
					continue
				}
				// If the bridge is loaded, then it's the latest-deployed proxy Blobstream contract
				bridge, err := blobstreamwrapper.NewWrappers(receipt.ContractAddress, client)
				if err != nil {
					continue
				}
				if !implementationFound {
					// at this level, we found the implementation. Now, we will look for the proxy.
					implementationFound = true
					continue
				}
				cancel()
				return bridge, nil
			}
		}
	}
}

func (network BlobstreamNetwork) WaitForRelayerToStart(_ctx context.Context, bridge *blobstreamwrapper.Wrappers) error {
	ctx, cancel := context.WithTimeout(_ctx, 2*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("relayer didn't start correctly")
			}
			return ctx.Err()
		default:
			nonce, err := bridge.StateEventNonce(&bind.CallOpts{Context: ctx})
			if err == nil && nonce != nil && nonce.Int64() >= 1 {
				cancel()
				return nil
			}
			fmt.Println("waiting for relayer to start ...")
			time.Sleep(5 * time.Second)
		}
	}
}

func (network BlobstreamNetwork) WaitForEventNonce(ctx context.Context, bridge *blobstreamwrapper.Wrappers, n uint64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("couldn't reach wanted nonce")
			}
			return ctx.Err()
		default:
			nonce, err := bridge.StateEventNonce(&bind.CallOpts{Context: ctx})
			if err == nil {
				if nonce != nil && nonce.Int64() >= int64(n) {
					cancel()
					return nil
				}
				fmt.Printf("waiting for nonce %d current nonce %d\n", n, nonce)
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func (network BlobstreamNetwork) UpdateDataCommitmentWindow(ctx context.Context, newWindow uint64) error {
	fmt.Printf("updating data commitment window %d\n", newWindow)
	kr, err := keyring.New(
		"blobstream-tests",
		"test",
		"celestia-app/core0",
		nil,
		encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec,
	)
	if err != nil {
		return err
	}
	blobStreamGRPC, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer blobStreamGRPC.Close()

	signer, err := user.SetupSingleSigner(ctx, kr, blobStreamGRPC, encoding.MakeConfig(app.ModuleEncodingRegisters...))
	if err != nil {
		return err
	}

	// create and submit a new param change proposal for the data commitment window
	change := proposal.NewParamChange(
		types.ModuleName,
		string(types.ParamsStoreKeyDataCommitmentWindow),
		fmt.Sprintf("\"%d\"", newWindow),
	)
	content := proposal.NewParameterChangeProposal(
		"data commitment window update",
		"description",
		[]proposal.ParamChange{change},
	)

	msg, err := v1beta1.NewMsgSubmitProposal(
		content,
		sdk.NewCoins(
			sdk.NewCoin(app.BondDenom, sdk.NewInt(5000000))),
		signer.Address(),
	)
	if err != nil {
		return err
	}

	_, err = signer.SubmitTx(ctx, []sdk.Msg{msg}, user.SetGasLimitAndFee(3000000, 300000))
	if err != nil {
		return err
	}

	// query the proposal to get the id
	gqc := v1.NewQueryClient(blobStreamGRPC)
	gresp, err := gqc.Proposals(
		ctx,
		&v1.QueryProposalsRequest{
			ProposalStatus: v1.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD,
		},
	)
	if err != nil {
		return err
	}
	if len(gresp.Proposals) != 1 {
		return fmt.Errorf("expected to have only one proposal in voting period")
	}

	// create and submit a new vote
	vote := v1.NewMsgVote(
		signer.Address(),
		gresp.Proposals[0].Id,
		v1.VoteOption_VOTE_OPTION_YES,
		"",
	)

	_, err = signer.SubmitTx(ctx, []sdk.Msg{vote}, user.SetGasLimitAndFee(3000000, 300000))
	if err != nil {
		return err
	}

	// wait for the voting period to finish
	time.Sleep(25 * time.Second)

	// check that the parameters got updated as expected
	currentWindow, err := network.GetCurrentDataCommitmentWindow(ctx)
	if err != nil {
		return err
	}
	if currentWindow != newWindow {
		return fmt.Errorf("data commitment window was not updated successfully. %d vs %d", currentWindow, newWindow)
	}

	fmt.Println("updated data commitment window successfully")
	return nil
}

func (network BlobstreamNetwork) PrintLogs() {
	_ = network.Instance.
		WithCommand([]string{"logs"}).
		Invoke()
}

func (network BlobstreamNetwork) GetLatestValset(ctx context.Context) (*types.Valset, error) {
	// create app querier
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	err := appQuerier.Start()
	if err != nil {
		return nil, err
	}
	defer appQuerier.Stop() //nolint:errcheck

	valset, err := appQuerier.QueryLatestValset(ctx)
	if err != nil {
		return nil, err
	}
	return valset, nil
}

func (network BlobstreamNetwork) GetCurrentDataCommitmentWindow(ctx context.Context) (uint64, error) {
	var window uint64
	queryFun := func() error {
		blobStreamGRPC, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		defer blobStreamGRPC.Close()
		bqc := types.NewQueryClient(blobStreamGRPC)
		presp, err := bqc.Params(ctx, &types.QueryParamsRequest{})
		if err != nil {
			return err
		}
		window = presp.Params.DataCommitmentWindow
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return 0, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return 0, fmt.Errorf("couldn't query data commitment window")
			}
			return 0, ctx.Err()
		default:
			err := queryFun()
			if err == nil {
				cancel()
				return window, nil
			}
			time.Sleep(2 * time.Second)
		}
	}
}
