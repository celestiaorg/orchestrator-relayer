package orchestrator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	ServiceNameOrchestrator = "orchestrator"
)

const DefaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

###############################################################################
###                           RPC Configuration                             ###
###############################################################################

# Specify the celestia app rest rpc address.
core.rpc = "{{ .CoreRPC }}"

# Specify the celestia app grpc address.
core.grpc = "{{ .CoreGRPC }}"

# allow gRPC over insecure channels, if not TLS the server must use TLS.
grpc.insecure = {{ .GRPCInsecure }}

###############################################################################
###                         P2P Configuration                               ###
###############################################################################

# Comma-separated multiaddresses of p2p peers to connect to.
# Example: "/ip4/127.0.0.1/tcp/30001/p2p/12D3K...,/ip4/127.0.0.1/tcp/30000/p2p/12D3K..."
p2p.bootstrappers = "{{ .Bootstrappers }}"

# MultiAddr for the p2p peer to listen on.
p2p.listen-addr = "{{ .P2PListenAddr }}"
`

func addOrchestratorFlags(cmd *cobra.Command) *cobra.Command {
	base.AddCoreRPCFlag(cmd)
	base.AddCoreGRPCFlag(cmd)
	base.AddEVMAccAddressFlag(cmd)
	base.AddEVMPassphraseFlag(cmd)
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameOrchestrator, homeDir)
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	base.AddGRPCInsecureFlag(cmd)
	return cmd
}

type StartConfig struct {
	base.Config
	CoreGRPC      string `mapstructure:"core.grpc" json:"core.grpc"`
	CoreRPC       string `mapstructure:"core.rpc" json:"core.rpc"`
	EvmAccAddress string `mapstructure:"evm.acccount-address" json:"evm.acccount-address"`
	Bootstrappers string `mapstructure:"p2p.bootstrappers" json:"p2p.bootstrappers"`
	P2PListenAddr string `mapstructure:"p2p.listen-addr" json:"p2p.listen-addr"`
	P2pNickname   string `mapstructure:"p2p.nickname" json:"p2p.nickname"`
	GRPCInsecure  bool   `mapstructure:"grpc.insecure" json:"grpc.insecure"`
}

func DefaultStartConfig() *StartConfig {
	return &StartConfig{
		CoreRPC:       "tcp://localhost:26657",
		CoreGRPC:      "localhost:9090",
		Bootstrappers: "",
		P2PListenAddr: "/ip4/0.0.0.0/tcp/30000",
		GRPCInsecure:  true,
	}
}

func parseOrchestratorFlags(cmd *cobra.Command, startConf *StartConfig) (StartConfig, error) {
	evmAccAddr, _, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if evmAccAddr == "" {
		return StartConfig{}, errors.New("the evm account address should be specified")
	}
	startConf.EvmAccAddress = evmAccAddr

	coreRPC, changed, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		if !strings.HasPrefix(coreRPC, "tcp://") {
			coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
		}
		startConf.CoreRPC = coreRPC
	}

	coreGRPC, changed, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.CoreGRPC = coreGRPC
	}

	bootstrappers, changed, err := base.GetBootstrappersFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.Bootstrappers = bootstrappers
	}

	p2pListenAddress, changed, err := base.GetP2PListenAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.P2PListenAddr = p2pListenAddress
	}

	p2pNickname, changed, err := base.GetP2PNicknameFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.P2pNickname = p2pNickname
	}

	homeDir, changed, err := base.GetHomeFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	startConf.Home = homeDir

	passphrase, _, err := base.GetEVMPassphraseFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	startConf.EVMPassphrase = passphrase

	grpcInsecure, changed, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.GRPCInsecure = grpcInsecure
	}

	return *startConf, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameOrchestrator, homeDir)
	return cmd
}

type InitConfig struct {
	home string
}

func parseInitFlags(cmd *cobra.Command) (InitConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return InitConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameOrchestrator)
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
