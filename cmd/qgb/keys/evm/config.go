package evm

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	FlagPassphrase    = "passphrase"
	FlagNewPassphrase = "new-passphrase"
)

func keysConfigFlags(cmd *cobra.Command) *cobra.Command {
	// TODO default value should be given
	cmd.Flags().String(base.FlagHome, "", "The qgb evm keys home directory")
	cmd.Flags().String(FlagPassphrase, "", "the account passphrase (if not specified as a flag, it will be asked interactively)")
	return cmd
}

type KeysConfig struct {
	*base.Config
	passphrase string
}

func parseKeysConfigFlags(cmd *cobra.Command, serviceName string) (KeysConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return KeysConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(serviceName)
		if err != nil {
			return KeysConfig{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(FlagPassphrase)
	if err != nil {
		return KeysConfig{}, err
	}

	return KeysConfig{
		Config:     &base.Config{Home: homeDir},
		passphrase: passphrase,
	}, nil
}

func keysNewPassphraseConfigFlags(cmd *cobra.Command) *cobra.Command {
	// TODO default value should be given
	cmd.Flags().String(base.FlagHome, "", "The qgb evm keys home directory")
	cmd.Flags().String(FlagPassphrase, "", "the account passphrase (if not specified as a flag, it will be asked interactively)")
	cmd.Flags().String(FlagNewPassphrase, "", "the account new passphrase (if not specified as a flag, it will be asked interactively)")
	return cmd
}

type KeysNewPassphraseConfig struct {
	*base.Config
	passphrase    string
	newPassphrase string
}

func parseKeysNewPassphraseConfigFlags(cmd *cobra.Command, serviceName string) (KeysNewPassphraseConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return KeysNewPassphraseConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(serviceName)
		if err != nil {
			return KeysNewPassphraseConfig{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(FlagPassphrase)
	if err != nil {
		return KeysNewPassphraseConfig{}, err
	}

	newPassphrase, err := cmd.Flags().GetString(FlagNewPassphrase)
	if err != nil {
		return KeysNewPassphraseConfig{}, err
	}

	return KeysNewPassphraseConfig{
		Config:        &base.Config{Home: homeDir},
		passphrase:    passphrase,
		newPassphrase: newPassphrase,
	}, nil
}
