package base

import (
	"fmt"
	"os"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/evm"

	"github.com/spf13/cobra"

	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/libs/cli"
)

const (
	FlagHome          = cli.HomeFlag
	FlagEVMPassphrase = "evm.passphrase"
)

// Config contains the base config that all commands should have.
// Logger related configuration will be added later.
type Config struct {
	Home          string
	EVMPassphrase string
}

func AddEVMPassphraseFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
}

func GetEVMPassphraseFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMPassphrase)
	val, err := cmd.Flags().GetString(FlagEVMPassphrase)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

// DefaultServicePath constructs the default Blobstream store path for
// the provided service.
// It tries to get the home directory from an environment variable
// called `<service_name_in_upper_case>_HOME`. If not set, then reverts to using
// the default user home directory and returning `~/.<service_name_in_lower_case>`.
func DefaultServicePath(serviceName string) (string, error) {
	home := os.Getenv(strings.ToUpper(serviceName) + "_HOME")

	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "error getting user home directory. please either setup a home directory for the user correctly or set the '<SERVICE_NAME>_HOME' environment variable")
		}
	}
	return fmt.Sprintf("%s/.%s", home, strings.ToLower(serviceName)), nil
}

const (
	FlagBootstrappers    = "p2p.bootstrappers"
	FlagP2PListenAddress = "p2p.listen-addr"
	FlagP2PNickname      = "p2p.nickname"
	FlagGRPCInsecure     = "grpc.insecure"

	FlagEVMAccAddress      = "evm.account"
	FlagEVMChainID         = "evm.chain-id"
	FlagEVMRPC             = "evm.rpc"
	FlagEVMGasLimit        = "evm.gas-limit"
	FlagEVMContractAddress = "evm.contract-address"

	FlagCoreGRPC = "core.grpc"
	FlagCoreRPC  = "core.rpc"

	FlagStartingNonce = "starting-nonce"
)

func AddStartingNonceFlag(cmd *cobra.Command) {
	cmd.Flags().String(
		FlagStartingNonce,
		"latest",
		"Specify the nonce to start the Blobstream contract from. "+
			"\"earliest\": for genesis, "+
			"\"latest\": for latest nonce, "+
			"\"nonce\": for a specific nonce.",
	)
}

func GetStartingNonceFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagStartingNonce)
	val, err := cmd.Flags().GetString(FlagStartingNonce)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddP2PNicknameFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagP2PNickname, "", "Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)")
}

func GetP2PNicknameFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagP2PNickname)
	val, err := cmd.Flags().GetString(FlagP2PNickname)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddP2PListenAddressFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagP2PListenAddress, "/ip4/0.0.0.0/tcp/30000", "MultiAddr for the p2p peer to listen on")
}

func GetP2PListenAddressFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagP2PListenAddress)
	val, err := cmd.Flags().GetString(FlagP2PListenAddress)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddBootstrappersFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagBootstrappers, "", "Comma-separated multiaddresses of p2p peers to connect to")
}

func GetBootstrappersFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagBootstrappers)
	val, err := cmd.Flags().GetString(FlagBootstrappers)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddGRPCInsecureFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(FlagGRPCInsecure, false, "allow gRPC over insecure channels, if not TLS the server must use TLS")
}

func GetGRPCInsecureFlag(cmd *cobra.Command) (bool, bool, error) {
	changed := cmd.Flags().Changed(FlagGRPCInsecure)
	val, err := cmd.Flags().GetBool(FlagGRPCInsecure)
	if err != nil {
		return false, changed, err
	}
	return val, changed, err
}

func AddCoreGRPCFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagCoreGRPC, "localhost:9090", "Specify the celestia app grpc address")
}

func GetCoreGRPCFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagCoreGRPC)
	val, err := cmd.Flags().GetString(FlagCoreGRPC)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddEVMChainIDFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64(FlagEVMChainID, 5, "Specify the evm chain id")
}

func GetEVMChainIDFlag(cmd *cobra.Command) (uint64, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMChainID)
	val, err := cmd.Flags().GetUint64(FlagEVMChainID)
	if err != nil {
		return 0, changed, err
	}
	return val, changed, err
}

func AddCoreRPCFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagCoreRPC, "tcp://localhost:26657", "Specify the celestia app rest rpc address")
}

func GetCoreRPCFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagCoreRPC)
	val, err := cmd.Flags().GetString(FlagCoreRPC)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddEVMRPCFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagEVMRPC, "http://localhost:8545", "Specify the ethereum rpc address")
}

func GetEVMRPCFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMRPC)
	val, err := cmd.Flags().GetString(FlagEVMRPC)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddEVMContractAddressFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagEVMContractAddress, "", "Specify the contract at which the Blobstream is deployed")
}

func GetEVMContractAddressFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMContractAddress)
	val, err := cmd.Flags().GetString(FlagEVMContractAddress)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func AddEVMGasLimitFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64(FlagEVMGasLimit, evm.DefaultEVMGasLimit, "Specify the evm gas limit")
}

func GetEVMGasLimitFlag(cmd *cobra.Command) (uint64, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMGasLimit)
	val, err := cmd.Flags().GetUint64(FlagEVMGasLimit)
	if err != nil {
		return 0, changed, err
	}
	return val, changed, err
}

func AddHomeFlag(cmd *cobra.Command, serviceName string, defaultHomeDir string) {
	cmd.Flags().String(FlagHome, defaultHomeDir, fmt.Sprintf("The Blobstream %s home directory", serviceName))
}

func GetHomeFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagHome)
	val, err := cmd.Flags().GetString(FlagHome)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}

func GetHomeDirectory(cmd *cobra.Command, service string) (string, error) {
	homeDir, changed, err := GetHomeFlag(cmd)
	if err != nil {
		return "", err
	}
	if changed && homeDir != "" {
		return homeDir, nil
	}
	return DefaultServicePath(service)
}

func AddEVMAccAddressFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagEVMAccAddress, "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
}

func GetEVMAccAddressFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagEVMAccAddress)
	val, err := cmd.Flags().GetString(FlagEVMAccAddress)
	if err != nil {
		return "", changed, err
	}
	return val, changed, err
}
