package common

import (
	"bufio"
	"fmt"
	"os"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

// ConfirmDeletePrivateKey is used to get a confirmation before deleting a private key
func ConfirmDeletePrivateKey(logger tmlog.Logger) bool {
	logger.Info("Are you sure you want to delete your private key? This action cannot be undone and may result in permanent loss of access to your account.")
	fmt.Print("Please enter 'yes' or 'no' to confirm your decision: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	return input == "yes"
}
