package base

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/libs/cli"
)

const (
	FlagHome          = cli.HomeFlag
	FlagEVMPassphrase = "evm-passphrase"
)

// Config contains the base config that all commands should have.
// Logger related configuration will be added later.
type Config struct {
	Home          string
	EVMPassphrase string
}

// DefaultServicePath constructs the default qgb store path for
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
