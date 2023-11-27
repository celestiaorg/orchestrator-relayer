package relayer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/spf13/cobra"
)

const (
	ServiceNameRelayer = "relayer"
)

const DefaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

###############################################################################
###                           RPC Configuration                             ###
###############################################################################

# Celestia app rest rpc address.
core-rpc = "{{ .CoreRPC }}"

# Celestia app grpc address.
core-grpc = "{{ .CoreGRPC }}"

# Allow gRPC over insecure channels, if not TLS the server must use TLS.
grpc-insecure = {{ .GrpcInsecure }}

###############################################################################
###                         P2P Configuration                               ###
###############################################################################

# Comma-separated multiaddresses of p2p peers to connect to.
# Example: "/ip4/127.0.0.1/tcp/30001/p2p/12D3K...,/ip4/127.0.0.1/tcp/30000/p2p/12D3K..."
bootstrappers = "{{ .Bootstrappers }}"

# MultiAddr for the p2p peer to listen on.
listen-addr = "{{ .P2PListenAddr }}"

###############################################################################
###                         EVM Configuration                               ###
###############################################################################

# Ethereum rpc address.
evm-rpc = "{{ .EvmRPC }}"

# Evm chain id.
evm-chain-id = "{{ .EvmChainID }}"

# Contract address at which Blobstream is deployed.
contract-address = "{{ .ContractAddr }}"

# Evm gas limit.
gas-limit = "{{ .EvmGasLimit }}"

# The time, in minutes, to wait for transactions to be mined
# on the target EVM chain before recreating them with a different gas price.
retry-timeout = "{{ .EVMRetryTimeout }}"
`

func addRelayerStartFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameRelayer, homeDir)
	base.AddEVMAccAddressFlag(cmd)
	base.AddEVMChainIDFlag(cmd)
	base.AddCoreGRPCFlag(cmd)
	base.AddCoreRPCFlag(cmd)
	base.AddEVMRPCFlag(cmd)
	base.AddEVMContractAddressFlag(cmd)
	base.AddEVMGasLimitFlag(cmd)
	base.AddEVMPassphraseFlag(cmd)
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	base.AddGRPCInsecureFlag(cmd)
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	base.AddEVMRetryTimeoutFlag(cmd)

	return cmd
}

type StartConfig struct {
	base.Config
	EvmChainID      uint64 `mapstructure:"evm-chain-id" json:"evm-chain-id"`
	EvmRPC          string `mapstructure:"evm-rpc" json:"evm-rpc"`
	CoreGRPC        string `mapstructure:"core-grpc" json:"core-grpc"`
	CoreRPC         string `mapstructure:"core-rpc" json:"core-rpc"`
	evmAccAddress   string
	ContractAddr    string `mapstructure:"contract-address" json:"contract-address"`
	EvmGasLimit     uint64 `mapstructure:"gas-limit" json:"gas-limit"`
	Bootstrappers   string `mapstructure:"bootstrappers" json:"bootstrappers"`
	P2PListenAddr   string `mapstructure:"listen-addr" json:"listen-addr"`
	p2pNickname     string
	GrpcInsecure    bool `mapstructure:"grpc-insecure" json:"grpc-insecure"`
	LogLevel        string
	LogFormat       string
	EVMRetryTimeout uint64 `mapstructure:"retry-timeout" json:"retry-timeout"`
}

func DefaultStartConfig() *StartConfig {
	return &StartConfig{
		CoreRPC:         "tcp://localhost:26657",
		CoreGRPC:        "localhost:9090",
		Bootstrappers:   "",
		P2PListenAddr:   "/ip4/0.0.0.0/tcp/30000",
		GrpcInsecure:    true,
		EvmChainID:      5,
		EvmRPC:          "http://localhost:8545",
		EvmGasLimit:     2500000,
		EVMRetryTimeout: 15,
	}
}

func (cfg StartConfig) ValidateBasics() error {
	if err := base.ValidateEVMAddress(cfg.evmAccAddress); err != nil {
		return fmt.Errorf("%s: flag --%s", err.Error(), base.FlagEVMAccAddress)
	}
	if err := base.ValidateEVMAddress(cfg.ContractAddr); err != nil {
		return fmt.Errorf("%s: flag --%s", err.Error(), base.FlagEVMContractAddress)
	}
	return nil
}

func parseRelayerStartFlags(cmd *cobra.Command, fileConfig *StartConfig) (StartConfig, error) {
	evmAccAddr, _, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.evmAccAddress = evmAccAddr

	evmChainID, changed, err := base.GetEVMChainIDFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.EvmChainID = evmChainID
	}

	coreRPC, changed, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		if !strings.HasPrefix(coreRPC, "tcp://") {
			coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
		}
		fileConfig.CoreRPC = coreRPC
	}

	coreGRPC, changed, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.CoreGRPC = coreGRPC
	}

	contractAddr, changed, err := base.GetEVMContractAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.ContractAddr = contractAddr
	}

	evmRPC, changed, err := base.GetEVMRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.EvmRPC = evmRPC
	}

	evmGasLimit, changed, err := base.GetEVMGasLimitFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.EvmGasLimit = evmGasLimit
	}

	bootstrappers, changed, err := base.GetBootstrappersFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.Bootstrappers = bootstrappers
	}

	p2pListenAddress, changed, err := base.GetP2PListenAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.P2PListenAddr = p2pListenAddress
	}

	p2pNickname, _, err := base.GetP2PNicknameFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.p2pNickname = p2pNickname

	homeDir, _, err := base.GetHomeFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.Home = homeDir

	passphrase, _, err := base.GetEVMPassphraseFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.EVMPassphrase = passphrase

	grpcInsecure, changed, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.GrpcInsecure = grpcInsecure
	}

	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.LogLevel = logLevel

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	fileConfig.LogFormat = logFormat

	retryTimeout, changed, err := base.GetEVMRetryTimeoutFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.EVMRetryTimeout = retryTimeout
	}

	return *fileConfig, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameRelayer, homeDir)
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	return cmd
}

type InitConfig struct {
	home      string
	logLevel  string
	logFormat string
}

func parseInitFlags(cmd *cobra.Command) (InitConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return InitConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameRelayer)
		if err != nil {
			return InitConfig{}, err
		}
	}

	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return InitConfig{}, err
	}

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return InitConfig{}, err
	}

	return InitConfig{
		home:      homeDir,
		logFormat: logFormat,
		logLevel:  logLevel,
	}, nil
}

func LoadFileConfiguration(homeDir string) (*StartConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	configPath := filepath.Join(homeDir, "config")
	configFilePath := filepath.Join(configPath, "config.toml")
	conf := DefaultStartConfig()

	// if config.toml file does not exist, we create it and write default ClientConfig values into it.
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := initializeConfigFile(configFilePath, configPath, conf); err != nil {
			return nil, err
		}
	}

	conf, err := getStartConfig(v, configPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get client config: %v", err)
	}
	return conf, nil
}

func initializeConfigFile(configFilePath string, configPath string, conf *StartConfig) error {
	if err := base.EnsureConfigPath(configPath); err != nil {
		return fmt.Errorf("couldn't make relayer config: %v", err)
	}

	if err := writeConfigToFile(configFilePath, conf); err != nil {
		return fmt.Errorf("could not write relayer config to the file: %v", err)
	}
	return nil
}

// writeConfigToFile parses DefaultConfigTemplate, renders config using the template and writes it to
// configFilePath.
func writeConfigToFile(configFilePath string, config *StartConfig) error {
	var buffer bytes.Buffer

	tmpl := template.New("relayerConfigFileTemplate")
	configTemplate, err := tmpl.Parse(DefaultConfigTemplate)
	if err != nil {
		return err
	}

	if err := configTemplate.Execute(&buffer, config); err != nil {
		return err
	}

	return os.WriteFile(configFilePath, buffer.Bytes(), 0o600)
}

// getStartConfig reads values from config.toml file and unmarshalls them into StartConfig
func getStartConfig(v *viper.Viper, configPath string) (*StartConfig, error) {
	v.AddConfigPath(configPath)
	v.SetConfigName("config")
	v.SetConfigType("toml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := new(StartConfig)
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
