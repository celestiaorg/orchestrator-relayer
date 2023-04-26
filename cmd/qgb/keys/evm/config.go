package evm

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	FlagNewPassphrase = "new-passphrase"
)

func keysConfigFlags(cmd *cobra.Command, service string) *cobra.Command {
	homeDir, err := base.DefaultServicePath(service)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb evm keys home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
	return cmd
}

type KeysConfig struct {
	*base.Config
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
	passphrase, err := cmd.Flags().GetString(base.FlagEVMPassphrase)
	if err != nil {
		return KeysConfig{}, err
	}

	return KeysConfig{
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
	}, nil
}

func keysNewPassphraseConfigFlags(cmd *cobra.Command, service string) *cobra.Command {
	homeDir, err := base.DefaultServicePath(service)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb evm keys home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
	cmd.Flags().String(FlagNewPassphrase, "", "the evm account new passphrase (if not specified as a flag, it will be asked interactively)")
	return cmd
}

type KeysNewPassphraseConfig struct {
	*base.Config
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
	passphrase, err := cmd.Flags().GetString(base.FlagEVMPassphrase)
	if err != nil {
		return KeysNewPassphraseConfig{}, err
	}

	newPassphrase, err := cmd.Flags().GetString(FlagNewPassphrase)
	if err != nil {
		return KeysNewPassphraseConfig{}, err
	}

	return KeysNewPassphraseConfig{
		Config:        &base.Config{Home: homeDir, EVMPassphrase: passphrase},
		newPassphrase: newPassphrase,
	}, nil
}
