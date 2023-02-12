package deploy

import (
	"context"
	"os"
	"strconv"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "deploy <flags>",
		Short: "Deploys the QGB contract and initializes it using the provided Celestia chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseDeployFlags(cmd)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			qgbGRPC, err := grpc.Dial(config.celesGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer func(qgbGRPC *grpc.ClientConn) {
				err := qgbGRPC.Close()
				if err != nil {
					logger.Error(err.Error())
				}
			}(qgbGRPC)

			querier := rpc.NewAppQuerier(logger, qgbGRPC, encCfg)

			vs, err := getStartingValset(cmd.Context(), querier, config.startingNonce)
			if err != nil {
				return errors.Wrap(
					err,
					"cannot initialize the QGB contract without having a valset request: %s",
				)
			}

			evmClient := evm.NewClient(
				tmlog.NewTMLogger(os.Stdout),
				nil,
				config.privateKey,
				config.evmRPC,
				config.evmGasLimit,
			)

			txOpts, err := evmClient.NewTransactionOpts(cmd.Context())
			if err != nil {
				return err
			}

			backend, err := evmClient.NewEthClient()
			if err != nil {
				return err
			}
			defer backend.Close()

			address, tx, _, err := evmClient.DeployQGBContract(
				txOpts,
				backend,
				*vs,
				vs.Nonce,
				false,
			)
			if err != nil {
				logger.Error("failed to delpoy QGB contract", "hash", tx.Hash().String())
				return err
			}

			receipt, err := evmClient.WaitForTransaction(cmd.Context(), backend, tx)
			if err == nil && receipt != nil && receipt.Status == 1 {
				logger.Info("deployed QGB contract", "address", address.Hex(), "hash", tx.Hash().String())
			}

			return nil
		},
	}
	return addDeployFlags(command)
}

// getStartingValset get the valset that will be used to init the bridge contract.
func getStartingValset(ctx context.Context, querier *rpc.AppQuerier, startingNonce string) (*types.Valset, error) {
	switch startingNonce {
	case "latest":
		return querier.QueryLatestValset(ctx)
	case "earliest":
		// TODO make the first nonce 1 a const
		att, err := querier.QueryAttestationByNonce(ctx, 1)
		if err != nil {
			return nil, err
		}
		vs, ok := att.(*types.Valset)
		if !ok {
			return nil, ErrUnmarshallValset
		}
		return vs, nil
	default:
		nonce, err := strconv.ParseUint(startingNonce, 10, 0)
		if err != nil {
			return nil, err
		}
		attestation, err := querier.QueryAttestationByNonce(ctx, nonce)
		if err != nil {
			return nil, err
		}
		if attestation == nil {
			return nil, types.ErrNilAttestation
		}
		if attestation.Type() == types.ValsetRequestType {
			value, ok := attestation.(*types.Valset)
			if !ok {
				return nil, ErrUnmarshallValset
			}
			return value, nil
		}
		return querier.QueryLastValsetBeforeNonce(ctx, nonce)
	}
}
