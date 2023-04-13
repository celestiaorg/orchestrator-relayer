package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

func CommandToServiceName(commandUsage string) (string, error) {
	if strings.Contains(commandUsage, "rel") {
		return "relayer", nil
	}
	if strings.Contains(commandUsage, "orch") {
		return "orchestrator", nil
	}
	if strings.Contains(commandUsage, "deploy") {
		return "deployer", nil
	}
	return "", fmt.Errorf("unknown service %s", commandUsage)
}

// ConfirmDeletePrivateKey is used to get a confirmation before deleting a private key
func ConfirmDeletePrivateKey(logger tmlog.Logger) bool {
	logger.Info("Are you sure you want to delete your private key? This action cannot be undone and may result in permanent loss of access to your account.")
	fmt.Print("Please enter 'yes' or 'no' to confirm your decision: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	return input == "yes"
}
