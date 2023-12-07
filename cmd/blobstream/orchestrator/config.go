package orchestrator

import (
	"bytes"
	"fmt"
	"github.com/celestiaorg/orchestrator-relayer/telemetry"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/viper"

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
core-rpc = "{{ .CoreRPC }}"

# Specify the celestia app grpc address.
core-grpc = "{{ .CoreGRPC }}"

# allow gRPC over insecure channels, if not TLS the server must use TLS.
grpc-insecure = {{ .GRPCInsecure }}

###############################################################################
###                         P2P Configuration                               ###
###############################################################################

# Comma-separated multiaddresses of p2p peers to connect to.
# Example: "/ip4/127.0.0.1/tcp/30001/p2p/12D3K...,/ip4/127.0.0.1/tcp/30000/p2p/12D3K..."
bootstrappers = "{{ .Bootstrappers }}"

# MultiAddr for the p2p peer to listen on.
listen-addr = "{{ .P2PListenAddr }}"

###############################################################################
###                         Telemetry Configuration                         ###
###############################################################################

# Enables OTLP metrics with HTTP exporter.
metrics = "{{ .MetricsConfig.Metrics }}"

# Sets HTTP endpoint for OTLP metrics to be exported to.
endpoint = "{{ .MetricsConfig.Endpoint }}"

# Enable TLS connection to OTLP metric backend.
tls = "{{ .MetricsConfig.TLS }}"
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
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	base.AddMetricsFlag(cmd)
	base.AddMetricsEndpointFlag(cmd)
	base.AddMetricsTLSFlag(cmd)

	return cmd
}

type StartConfig struct {
	base.Config
	CoreGRPC      string `mapstructure:"core-grpc" json:"core-grpc"`
	CoreRPC       string `mapstructure:"core-rpc" json:"core-rpc"`
	EvmAccAddress string
	Bootstrappers string `mapstructure:"bootstrappers" json:"bootstrappers"`
	P2PListenAddr string `mapstructure:"listen-addr" json:"listen-addr"`
	P2pNickname   string
	GRPCInsecure  bool `mapstructure:"grpc-insecure" json:"grpc-insecure"`
	LogLevel      string
	LogFormat     string
	MetricsConfig telemetry.Config `mapstructure:"metrics-config" json:"metrics-config"`
}

func DefaultStartConfig() *StartConfig {
	return &StartConfig{
		CoreRPC:       "tcp://localhost:26657",
		CoreGRPC:      "localhost:9090",
		Bootstrappers: "",
		P2PListenAddr: "/ip4/0.0.0.0/tcp/30000",
		GRPCInsecure:  true,
		MetricsConfig: telemetry.Config{
			Metrics:  false,
			Endpoint: "localhost:4318",
			TLS:      false,
		},
	}
}

func (cfg StartConfig) ValidateBasics() error {
	if err := base.ValidateEVMAddress(cfg.EvmAccAddress); err != nil {
		return fmt.Errorf("%s: flag --%s", err.Error(), base.FlagEVMAccAddress)
	}
	return nil
}

func parseOrchestratorFlags(cmd *cobra.Command, startConf *StartConfig) (StartConfig, error) {
	evmAccAddr, _, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
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

	homeDir, _, err := base.GetHomeFlag(cmd)
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

	metrics, changed, err := base.GetMetricsFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.MetricsConfig.Metrics = metrics
	}

	endpoint, changed, err := base.GetMetricsEndpointFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.MetricsConfig.Endpoint = endpoint
	}

	tls, changed, err := base.GetMetricsTLSFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		startConf.MetricsConfig.TLS = tls
	}

	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	startConf.LogLevel = logLevel

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	startConf.LogFormat = logFormat

	return *startConf, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameOrchestrator, homeDir)
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
		homeDir, err = base.DefaultServicePath(ServiceNameOrchestrator)
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

	conf, err := GetStartConfig(v, configPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get client config: %v", err)
	}
	return conf, nil
}

func initializeConfigFile(configFilePath string, configPath string, conf *StartConfig) error {
	if err := base.EnsureConfigPath(configPath); err != nil {
		return fmt.Errorf("couldn't make orchestrator config: %v", err)
	}

	if err := writeConfigToFile(configFilePath, conf); err != nil {
		return fmt.Errorf("could not write orchestrator config to the file: %v", err)
	}
	return nil
}

// writeConfigToFile parses DefaultConfigTemplate, renders config using the template and writes it to
// configFilePath.
func writeConfigToFile(configFilePath string, config *StartConfig) error {
	var buffer bytes.Buffer

	tmpl := template.New("orchestratorConfigFileTemplate")
	configTemplate, err := tmpl.Parse(DefaultConfigTemplate)
	if err != nil {
		return err
	}

	if err := configTemplate.Execute(&buffer, config); err != nil {
		return err
	}

	return os.WriteFile(configFilePath, buffer.Bytes(), 0o600)
}

// GetStartConfig reads values from config.toml file and unmarshalls them into StartConfig
func GetStartConfig(v *viper.Viper, configPath string) (*StartConfig, error) {
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
