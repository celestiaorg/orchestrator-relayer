package network

import (
	"context"
	"math/big"
	"time"

	wrappers "github.com/celestiaorg/blobstream-contracts/v4/wrappers/Blobstream.sol"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Network struct {
	TendermintRPC     *http.HTTP
	CelestiaGRPC      *grpc.ClientConn
	BlobstreamWrapper *wrappers.Wrappers
}

func NewNetwork(ctx context.Context, evmRPC string, tendermintRPC string, celestiaGRPC string) (*Network, error) {
	trpc, err := http.New(tendermintRPC, "/websocket")
	if err != nil {
		return nil, err
	}
	if err := trpc.Start(); err != nil {
		return nil, err
	}

	grpcConn, err := grpc.Dial(celestiaGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	wrapper, err := GetLatestDeployedBlobstreamContract(ctx, time.Minute, evmRPC)
	if err != nil {
		return nil, err
	}

	return &Network{
		TendermintRPC:     trpc,
		CelestiaGRPC:      grpcConn,
		BlobstreamWrapper: wrapper,
	}, nil
}

func GetLatestDeployedBlobstreamContract(
	_ctx context.Context,
	timeout time.Duration,
	rpc string,
) (*wrappers.Wrappers, error) {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	height := 0
	ctx, cancel := context.WithTimeout(_ctx, timeout)
	implementationFound := false
	for {
		select {
		case <-ctx.Done():
			cancel()
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
				bridge, err := wrappers.NewWrappers(receipt.ContractAddress, client)
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
