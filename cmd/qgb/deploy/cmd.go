package deploy

import (
	"context"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/term"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
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

			// checking if the provided home is already initiated
			isInit := store.IsInit(logger, config.Home, store.InitOptions{NeedEVMKeyStore: true})
			if !isInit {
				logger.Info("please initialize the EVM keystore using the `deploy keys add/import` command")
				return store.ErrNotInited
			}

			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			querier := rpc.NewAppQuerier(logger, config.celesGRPC, encCfg)
			err = querier.Start()
			if err != nil {
				return err
			}
			defer func() {
				err := querier.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}()

			vs, err := getStartingValset(cmd.Context(), querier, config.startingNonce)
			if err != nil {
				return errors.Wrap(
					err,
					"cannot initialize the QGB contract without having a valset request: %s",
				)
			}

			// creating the data store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			if !common.IsHexAddress(config.evmAccAddress) {
				logger.Error("provided address is not a correct EVM address", "address", config.evmAccAddress)
				return nil // should we return errors in these cases?
			}

			addr := common.HexToAddress(config.evmAccAddress)
			if !s.EVMKeyStore.HasAddress(addr) {
				logger.Info("account not found in keystore", "address", config.evmAccAddress)
				return nil
			}

			logger.Info("loading EVM account", "address", addr.String())

			var acc accounts.Account
			for _, storeAcc := range s.EVMKeyStore.Accounts() {
				if storeAcc.Address.String() == addr.String() {
					acc = storeAcc
				}
			}

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the account passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			err = s.EVMKeyStore.Unlock(acc, passphrase)
			if err != nil {
				logger.Error("unable to load the EVM private key")
				return err
			}
			defer func(EVMKeyStore *keystore.KeyStore, addr common.Address) {
				err := EVMKeyStore.Lock(addr)
				if err != nil {
					panic(err)
				}
			}(s.EVMKeyStore, acc.Address)

			evmClient := evm.NewClient(
				tmlog.NewTMLogger(os.Stdout),
				nil,
				s.EVMKeyStore,
				&acc,
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
				logger.Error("failed to deploy QGB contract")
				return err
			}

			receipt, err := evmClient.WaitForTransaction(cmd.Context(), backend, tx)
			if err == nil && receipt != nil && receipt.Status == 1 {
				logger.Info("deployed QGB contract", "address", address.Hex(), "hash", tx.Hash().String())
			}

			return nil
		},
	}
	command.AddCommand(keys.Command())
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
