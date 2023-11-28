package query

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/relayer"
	"github.com/spf13/viper"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	common2 "github.com/ethereum/go-ethereum/common"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/common"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:          "query",
		Aliases:      []string{"q"},
		Short:        "Query relevant information from a running Blobstream",
		SilenceUsage: true,
	}

	queryCmd.AddCommand(
		Signers(),
		Signature(),
	)

	queryCmd.SetHelpCommand(&cobra.Command{})

	return queryCmd
}

func Signers() *cobra.Command {
	command := &cobra.Command{
		Use:   "signers <nonce>",
		Args:  cobra.ExactArgs(1),
		Short: "Queries the Blobstream for attestations signers",
		Long: "Queries the Blobstream for attestations signers. The nonce is the attestation nonce that the command" +
			" will query signatures for. It should be either a specific nonce starting from 2 and on." +
			" Or, use 'latest' as argument to check the latest attestation nonce",
		RunE: func(cmd *cobra.Command, args []string) error {
			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			fileConfig, err := tryToGetExistingConfig(cmd, logger)
			if err != nil {
				return err
			}
			config, err := parseFlags(cmd, &fileConfig)
			if err != nil {
				return err
			}

			logger.Debug("initializing queriers")

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			stopFuncs := make([]func() error, 0, 1)
			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()

			// create tm querier and app querier
			tmQuerier, appQuerier, stops, err := common.NewTmAndAppQuerier(
				logger,
				config.coreRPC,
				config.coreGRPC,
				config.grpcInsecure,
			)
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
				return err
			}

			// creating the host
			h, err := libp2p.New()
			if err != nil {
				return err
			}
			addrInfo, err := peer.AddrInfoFromString(config.targetNode)
			if err != nil {
				return err
			}
			for i := 0; i < 5; i++ {
				logger.Debug("connecting to target node...")
				err := h.Connect(ctx, *addrInfo)
				if err != nil {
					logger.Error("couldn't connect to target node", "err", err.Error())
				}
				if err == nil {
					logger.Debug("connected to target node")
					break
				}
				time.Sleep(5 * time.Second)
			}

			// creating the data store
			dataStore := dssync.MutexWrap(ds.NewMapDatastore())

			// creating the dht
			dht, err := p2p.NewBlobstreamDHT(cmd.Context(), h, dataStore, []peer.AddrInfo{}, logger)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			nonce, err := parseNonce(ctx, appQuerier, args[0])
			if err != nil {
				return err
			}
			if nonce == 1 {
				return fmt.Errorf("nonce 1 doesn't need to be signed. signatures start from nonce 2")
			}

			err = getSignaturesAndPrintThem(ctx, logger, appQuerier, tmQuerier, p2pQuerier, nonce, config.outputFile)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addFlags(command)
}

type signature struct {
	EvmAddress   string `json:"evmAddress"`
	Moniker      string `json:"moniker"`
	Signature    string `json:"signature"`
	Signed       bool   `json:"signed"`
	ValopAddress string `json:"valopAddress"`
}

type validatorInfo struct {
	EvmAddress   string `json:"evmAddress"`
	Moniker      string `json:"moniker"`
	ValopAddress string `json:"valopAddress"`
}

type queryOutput struct {
	Signatures        []signature `json:"signatures"`
	Nonce             uint64      `json:"nonce"`
	MajorityThreshold uint64      `json:"majority_threshold"`
	CurrentThreshold  uint64      `json:"current_threshold"`
	CanRelay          bool        `json:"can_relay"`
}

func getSignaturesAndPrintThem(
	ctx context.Context,
	logger tmlog.Logger,
	appQuerier *rpc.AppQuerier,
	tmQuerier *rpc.TmQuerier,
	p2pQuerier *p2p.Querier,
	nonce uint64,
	outputFile string,
) error {
	logger.Info("getting signatures for nonce", "nonce", nonce)

	lastValset, err := appQuerier.QueryLastValsetBeforeNonce(ctx, nonce)
	if err != nil {
		return err
	}

	validatorSet, err := appQuerier.QueryStakingValidatorSet(ctx)
	if err != nil {
		return err
	}

	validatorsInfo, err := toValidatorsInfo(ctx, appQuerier, validatorSet)
	if err != nil {
		return err
	}

	att, err := appQuerier.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if att == nil {
		return celestiatypes.ErrAttestationNotFound
	}

	switch castedAtt := att.(type) {
	case *celestiatypes.Valset:
		signBytes, err := castedAtt.SignBytes()
		if err != nil {
			return err
		}
		confirms, err := p2pQuerier.QueryValsetConfirms(ctx, nonce, *lastValset, signBytes.Hex())
		if err != nil {
			return err
		}
		qOutput := toQueryOutput(toValsetConfirmsMap(confirms), validatorsInfo, nonce, *lastValset)
		if outputFile == "" {
			printConfirms(logger, qOutput)
		} else {
			err := writeConfirmsToJSONFile(logger, qOutput, outputFile)
			if err != nil {
				return err
			}
		}
	case *celestiatypes.DataCommitment:
		commitment, err := tmQuerier.QueryCommitment(
			ctx,
			castedAtt.BeginBlock,
			castedAtt.EndBlock,
		)
		if err != nil {
			return err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
		confirms, err := p2pQuerier.QueryDataCommitmentConfirms(ctx, *lastValset, nonce, dataRootHash.Hex())
		if err != nil {
			return err
		}
		qOutput := toQueryOutput(toDataCommitmentConfirmsMap(confirms), validatorsInfo, nonce, *lastValset)
		if outputFile == "" {
			printConfirms(logger, qOutput)
		} else {
			err := writeConfirmsToJSONFile(logger, qOutput, outputFile)
			if err != nil {
				return err
			}
		}
	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
	return nil
}

func toValidatorsInfo(ctx context.Context, appQuerier *rpc.AppQuerier, validatorSet []stakingtypes.Validator) (map[string]validatorInfo, error) {
	validatorsInfo := make(map[string]validatorInfo)
	for _, val := range validatorSet {
		evmAddr, err := appQuerier.QueryEVMAddress(ctx, val.OperatorAddress)
		if err != nil {
			return nil, err
		}
		if evmAddr != "" {
			validatorsInfo[evmAddr] = validatorInfo{
				EvmAddress:   evmAddr,
				Moniker:      val.GetMoniker(),
				ValopAddress: val.OperatorAddress,
			}
		}
	}
	return validatorsInfo, nil
}

func parseNonce(ctx context.Context, querier *rpc.AppQuerier, nonce string) (uint64, error) {
	switch nonce {
	case "latest":
		return querier.QueryLatestAttestationNonce(ctx)
	default:
		return strconv.ParseUint(nonce, 10, 0)
	}
}

func toValsetConfirmsMap(confirms []types.ValsetConfirm) map[string]string {
	// create a map for the signatures to get them easily
	confirmsMap := make(map[string]string)
	for _, confirm := range confirms {
		confirmsMap[confirm.EthAddress] = confirm.Signature
	}
	return confirmsMap
}

func toDataCommitmentConfirmsMap(confirms []types.DataCommitmentConfirm) map[string]string {
	// create a map for the signatures to get them easily
	confirmsMap := make(map[string]string)
	for _, confirm := range confirms {
		confirmsMap[confirm.EthAddress] = confirm.Signature
	}
	return confirmsMap
}

func toQueryOutput(confirmsMap map[string]string, validatorsInfo map[string]validatorInfo, nonce uint64, lastValset celestiatypes.Valset) queryOutput {
	currThreshold := uint64(0)
	signatures := make([]signature, len(lastValset.Members))
	// create the signature slice to be used for outputting the data
	for key, val := range lastValset.Members {
		sig, found := confirmsMap[val.EvmAddress]
		if found {
			signatures[key] = signature{
				EvmAddress:   val.EvmAddress,
				Signature:    sig,
				Signed:       true,
				Moniker:      validatorsInfo[val.EvmAddress].Moniker,
				ValopAddress: validatorsInfo[val.EvmAddress].ValopAddress,
			}
			currThreshold += val.Power
		} else {
			signatures[key] = signature{
				EvmAddress:   val.EvmAddress,
				Signature:    "",
				Signed:       false,
				Moniker:      validatorsInfo[val.EvmAddress].Moniker,
				ValopAddress: validatorsInfo[val.EvmAddress].ValopAddress,
			}
		}
	}
	return queryOutput{
		Signatures:        signatures,
		Nonce:             nonce,
		MajorityThreshold: lastValset.TwoThirdsThreshold(),
		CurrentThreshold:  currThreshold,
		CanRelay:          lastValset.TwoThirdsThreshold() <= currThreshold,
	}
}

func printConfirms(logger tmlog.Logger, qOutput queryOutput) {
	logger.Info(
		"query output",
		"nonce",
		qOutput.Nonce,
		"majority_threshold",
		qOutput.MajorityThreshold,
		"current_threshold",
		qOutput.CurrentThreshold,
		"can_relay",
		qOutput.CanRelay,
	)
	logger.Info("orchestrators that signed the attestation")
	for _, sig := range qOutput.Signatures {
		if sig.Signed {
			logger.Info(sig.Moniker, "signed", sig.Signed, "evm_address", sig.EvmAddress, "valop_address", sig.ValopAddress, "signature", sig.Signature)
		}
	}
	logger.Info("orchestrators that missed signing the attestation")
	for _, sig := range qOutput.Signatures {
		if !sig.Signed {
			logger.Info(sig.Moniker, "signed", sig.Signed, "evm_address", sig.EvmAddress, "valop_address", sig.ValopAddress)
		}
	}
	logger.Info("done")
}

func writeConfirmsToJSONFile(logger tmlog.Logger, qOutput queryOutput, outputFile string) error {
	logger.Info("writing confirms json file", "path", outputFile)

	file, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Error("failed to close file", "err", err.Error())
		}
	}(file)

	encoder := json.NewEncoder(file)
	err = encoder.Encode(qOutput)
	if err != nil {
		return err
	}

	logger.Info("output written to file successfully", "path", outputFile)
	return nil
}

func Signature() *cobra.Command {
	command := &cobra.Command{
		Use:   "signature <nonce> <evm_account>",
		Args:  cobra.ExactArgs(2),
		Short: "Queries a specific signature referenced by an EVM account address and a nonce",
		Long: "Queries a specific signature referenced by an EVM account address and a nonce. The nonce is the attestation" +
			" nonce that the command will query signatures for. The EVM address is the address registered by the validator " +
			"in the staking module.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			fileConfig, err := tryToGetExistingConfig(cmd, logger)
			if err != nil {
				return err
			}
			config, err := parseFlags(cmd, &fileConfig)
			if err != nil {
				return err
			}

			logger.Debug("initializing queriers")

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			stopFuncs := make([]func() error, 0, 1)
			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()

			// create tm querier and app querier
			tmQuerier, appQuerier, stops, err := common.NewTmAndAppQuerier(logger, config.coreRPC, config.coreGRPC, config.grpcInsecure)
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
				return err
			}

			// creating the host
			h, err := libp2p.New()
			if err != nil {
				return err
			}
			addrInfo, err := peer.AddrInfoFromString(config.targetNode)
			if err != nil {
				return err
			}
			for i := 0; i < 5; i++ {
				logger.Debug("connecting to target node...")
				err := h.Connect(ctx, *addrInfo)
				if err != nil {
					logger.Error("couldn't connect to target node", "err", err.Error())
				}
				if err == nil {
					logger.Debug("connected to target node")
					break
				}
				time.Sleep(5 * time.Second)
			}

			// creating the data store
			dataStore := dssync.MutexWrap(ds.NewMapDatastore())

			// creating the dht
			dht, err := p2p.NewBlobstreamDHT(cmd.Context(), h, dataStore, []peer.AddrInfo{}, logger)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			nonce, err := parseNonce(ctx, appQuerier, args[0])
			if err != nil {
				return err
			}
			if nonce == 1 {
				return fmt.Errorf("nonce 1 doesn't need to be signed. signatures start from nonce 2")
			}

			if !common2.IsHexAddress(args[1]) {
				return fmt.Errorf("invalid EVM address provided")
			}

			err = getSignatureAndPrintIt(ctx, logger, appQuerier, tmQuerier, p2pQuerier, args[1], nonce)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addFlags(command)
}

func getSignatureAndPrintIt(
	ctx context.Context,
	logger tmlog.Logger,
	appQuerier *rpc.AppQuerier,
	tmQuerier *rpc.TmQuerier,
	p2pQuerier *p2p.Querier,
	evmAddress string,
	nonce uint64,
) error {
	logger.Info("getting signature for address and nonce", "nonce", nonce, "evm_account", evmAddress)

	att, err := appQuerier.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if att == nil {
		return celestiatypes.ErrAttestationNotFound
	}

	switch castedAtt := att.(type) {
	case *celestiatypes.Valset:
		signBytes, err := castedAtt.SignBytes()
		if err != nil {
			return err
		}
		confirm, err := p2pQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, evmAddress, signBytes.Hex())
		if err != nil {
			return err
		}
		if confirm == nil {
			logger.Info("couldn't find orchestrator signature", "nonce", nonce, "evm_account", evmAddress)
		} else {
			logger.Info("found orchestrator signature", "nonce", nonce, "evm_account", evmAddress, "signature", confirm.Signature)
		}
	case *celestiatypes.DataCommitment:
		commitment, err := tmQuerier.QueryCommitment(
			ctx,
			castedAtt.BeginBlock,
			castedAtt.EndBlock,
		)
		if err != nil {
			return err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
		confirm, err := p2pQuerier.QueryDataCommitmentConfirmByEVMAddress(ctx, nonce, evmAddress, dataRootHash.Hex())
		if err != nil {
			return err
		}
		if confirm == nil {
			logger.Info("couldn't find orchestrator signature", "nonce", nonce, "evm_account", evmAddress)
		} else {
			logger.Info("found orchestrator signature", "nonce", nonce, "evm_account", evmAddress, "signature", confirm.Signature)
		}
	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
	return nil
}

// tryToGetExistingConfig tries to get the query config from existing
// orchestrator/relayer homes. It first checks whether the `--home` flag was
// changed. If so, it gets the config from there. If not, then it tries the
// orchestrator default home directory, then the relayer default home directory.
func tryToGetExistingConfig(cmd *cobra.Command, logger tmlog.Logger) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	homeDir, changed, err := base.GetHomeFlag(cmd)
	if err != nil {
		return Config{}, err
	}
	// the --home flag was set to some directory
	if changed && homeDir != "" {
		logger.Debug("using home", "home", homeDir)
		configPath := filepath.Join(homeDir, "config")

		// assume this home is an orchestrator home directory
		orchConf, err := orchestrator.GetStartConfig(v, configPath)
		if err == nil {
			// it is an orchestrator, so we get the config from it
			return *NewPartialConfig(
				orchConf.CoreGRPC,
				orchConf.CoreRPC,
				orchConf.Bootstrappers,
				orchConf.GRPCInsecure,
			), nil
		}

		// assume this home is a relayer home directory
		relConf, err := relayer.GetStartConfig(v, configPath)
		if err == nil {
			// it is a relayer, so we get the config from it
			return *NewPartialConfig(
				relConf.CoreGRPC,
				relConf.CoreRPC,
				relConf.Bootstrappers,
				relConf.GrpcInsecure,
			), nil
		}
		return Config{}, fmt.Errorf("the provided home directory is neither an orchestrator nor a relayer home directory")
	}
	// try to get the config from the orchestrator home directory
	orchHome, err := base.GetHomeDirectory(cmd, orchestrator.ServiceNameOrchestrator)
	if err != nil {
		return Config{}, err
	}
	orchConf, err := orchestrator.GetStartConfig(v, filepath.Join(orchHome, "config"))
	if err == nil {
		// found orchestrator home, get the config from it
		logger.Debug("using home", "home", orchHome)
		return *NewPartialConfig(
			orchConf.CoreGRPC,
			orchConf.CoreRPC,
			orchConf.Bootstrappers,
			orchConf.GRPCInsecure,
		), nil
	}

	// try to get the config from the relayer home directory
	relHome, err := base.GetHomeDirectory(cmd, relayer.ServiceNameRelayer)
	if err != nil {
		return Config{}, err
	}
	relConf, err := relayer.GetStartConfig(v, filepath.Join(relHome, "config"))
	if err == nil {
		// found relayer home, so we get the config from it
		logger.Debug("using home", "home", relHome)
		return *NewPartialConfig(
			relConf.CoreGRPC,
			relConf.CoreRPC,
			relConf.Bootstrappers,
			relConf.GrpcInsecure,
		), nil
	}

	return *DefaultConfig(), nil
}
