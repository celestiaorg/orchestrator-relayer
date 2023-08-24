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

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	qgbtypes "github.com/celestiaorg/orchestrator-relayer/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client/http"
	testcontainers "github.com/testcontainers/testcontainers-go/modules/compose"
)

type QGBNetwork struct {
	ComposePaths         []string
	Identifier           string
	Instance             *testcontainers.LocalDockerCompose
	EVMRPC               string
	TendermintRPC        string
	CelestiaGRPC         string
	P2PAddr              string
	EncCfg               encoding.Config
	DataCommitmentWindow uint64
	Logger               tmlog.Logger

	// used by the moderator to notify all the workers.
	stopChan <-chan struct{}
	// used by the workers to notify the moderator.
	toStopChan chan<- struct{}
}

func NewQGBNetwork() (*QGBNetwork, error) {
	id := strings.ToLower(uuid.New().String())
	paths := []string{"./docker-compose.yml"}
	instance := testcontainers.NewLocalDockerCompose(paths, id) //nolint:staticcheck
	stopChan := make(chan struct{})
	// given an initial capacity to avoid blocking in case multiple services failed
	// and wanted to notify the moderator.
	toStopChan := make(chan struct{}, 10)
	network := &QGBNetwork{
		Identifier:           id,
		ComposePaths:         paths,
		Instance:             instance,
		EVMRPC:               "http://localhost:8545",
		TendermintRPC:        "tcp://localhost:26657",
		CelestiaGRPC:         "localhost:9090",
		P2PAddr:              "localhost:30000",
		EncCfg:               encoding.MakeConfig(app.ModuleEncodingRegisters...),
		DataCommitmentWindow: 101, // If this one is changed, make sure to change also the genesis file
		stopChan:             stopChan,
		toStopChan:           toStopChan,
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
func registerGracefulExit(network *QGBNetwork) {
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

// StartAll starts the whole QGB cluster with multiple validators, orchestrators and a relayer
// Make sure to release the resources after finishing by calling the `StopAll()` method.
func (network QGBNetwork) StartAll() error {
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
func (network QGBNetwork) StopAll() error {
	err := network.Instance.
		WithCommand([]string{"stop"}).
		Invoke()
	if err.Error != nil {
		return err.Error
	}
	return nil
}

// DeleteAll deletes the containers, network and everything related to the cluster.
func (network QGBNetwork) DeleteAll() error {
	err := network.Instance.
		WithCommand([]string{"down"}).
		Invoke()
	if err.Error != nil {
		return err.Error
	}
	return nil
}

// KillAll kills all the containers.
func (network QGBNetwork) KillAll() error {
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
func (network QGBNetwork) Start(service Service) error {
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

// DeployQGBContract uses the Deployer service to deploy a new QGB contract
// based on the existing running network. If no Celestia-app nor ganache is
// started, it creates them automatically.
func (network QGBNetwork) DeployQGBContract() error {
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
func (network QGBNetwork) StartMultiple(services ...Service) error {
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

func (network QGBNetwork) Stop(service Service) error {
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
func (network QGBNetwork) StopMultiple(services ...Service) error {
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

func (network QGBNetwork) ExecCommand(service Service, command []string) error {
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
func (network QGBNetwork) StartMinimal() error {
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
func (network QGBNetwork) StartBase() error {
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

func (network QGBNetwork) WaitForNodeToStart(_ctx context.Context, rpcAddr string) error {
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

func (network QGBNetwork) WaitForBlock(_ctx context.Context, height int64) error {
	return network.WaitForBlockWithCustomTimeout(_ctx, height, 5*time.Minute)
}

func (network QGBNetwork) WaitForBlockWithCustomTimeout(
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
func (network QGBNetwork) WaitForOrchestratorToStart(_ctx context.Context, dht *p2p.QgbDHT, evmAddr string) (uint64, uint64, error) {
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
					dataRootTupleRoot := qgbtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
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
func (network QGBNetwork) GetValsetContainingVals(_ctx context.Context, number int) (*types.Valset, error) {
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
func (network QGBNetwork) GetValsetConfirm(
	_ctx context.Context,
	dht *p2p.QgbDHT,
	nonce uint64,
	evmAddr string,
) (*qgbtypes.ValsetConfirm, error) {
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
func (network QGBNetwork) GetDataCommitmentConfirm(
	_ctx context.Context,
	dht *p2p.QgbDHT,
	nonce uint64,
	evmAddr string,
) (*qgbtypes.DataCommitmentConfirm, error) {
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
			dataRootTupleRoot := qgbtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), commitment)
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
func (network QGBNetwork) GetDataCommitmentConfirmByHeight(
	_ctx context.Context,
	dht *p2p.QgbDHT,
	height uint64,
	evmAddr string,
) (*qgbtypes.DataCommitmentConfirm, error) {
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
func (network QGBNetwork) GetLatestAttestationNonce(_ctx context.Context) (uint64, error) {
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
func (network QGBNetwork) WasAttestationSigned(
	_ctx context.Context,
	dht *p2p.QgbDHT,
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
				dataRootTupleRoot := qgbtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
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

func (network QGBNetwork) GetLatestDeployedQGBContract(_ctx context.Context) (*wrapper.QuantumGravityBridge, error) {
	return network.GetLatestDeployedQGBContractWithCustomTimeout(_ctx, 5*time.Minute)
}

func (network QGBNetwork) GetLatestDeployedQGBContractWithCustomTimeout(
	_ctx context.Context,
	timeout time.Duration,
) (*wrapper.QuantumGravityBridge, error) {
	client, err := ethclient.Dial(network.EVMRPC)
	if err != nil {
		return nil, err
	}
	height := 0
	ctx, cancel := context.WithTimeout(_ctx, timeout)
	for {
		select {
		case <-network.stopChan:
			cancel()
			return nil, ErrNetworkStopped
		case <-ctx.Done():
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("timeout. couldn't find deployed qgb contract")
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
				// If the bridge is loaded, then it's the latest deployed QGB contracct
				bridge, err := wrapper.NewQuantumGravityBridge(receipt.ContractAddress, client)
				if err != nil {
					continue
				}
				cancel()
				return bridge, nil
			}
		}
	}
}

func (network QGBNetwork) WaitForRelayerToStart(_ctx context.Context, bridge *wrapper.QuantumGravityBridge) error {
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

func (network QGBNetwork) WaitForEventNonce(ctx context.Context, bridge *wrapper.QuantumGravityBridge, n uint64) error {
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

func (network QGBNetwork) PrintLogs() {
	_ = network.Instance.
		WithCommand([]string{"logs"}).
		Invoke()
}

func (network QGBNetwork) GetLatestValset(ctx context.Context) (*types.Valset, error) {
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
