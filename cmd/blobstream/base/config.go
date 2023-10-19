package base

import (
	"fmt"
	"os"
	"strings"

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
)

func AddP2PNicknameFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagP2PNickname, "", "Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)")
}

func AddP2PListenAddressFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagP2PListenAddress, "/ip4/0.0.0.0/tcp/30000", "MultiAddr for the p2p peer to listen on")
}

func AddBootstrappersFlag(cmd *cobra.Command) {
	cmd.Flags().String(FlagBootstrappers, "", "Comma-separated multiaddresses of p2p peers to connect to")
}
