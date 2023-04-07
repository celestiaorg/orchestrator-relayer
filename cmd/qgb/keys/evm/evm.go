package evm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"golang.org/x/term"
)

func Root() *cobra.Command {
	evmCmd := &cobra.Command{
		Use:          "evm",
		Short:        "QGB EVM keys manager",
		SilenceUsage: true,
	}

	evmCmd.AddCommand(
		Add(),
		List(),
		Delete(),
		Import(),
		Update(),
	)

	evmCmd.SetHelpCommand(&cobra.Command{})

	return evmCmd
}

func Add() *cobra.Command {
	cmd := cobra.Command{
		Use:   "add",
		Short: "create a new EVM address",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedEVMKeyStore: true}
			isInit := store.IsInit(logger, config.Home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				err := store.Init(logger, config.Home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide a passphrase for your account")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			account, err := s.EVMKeyStore.NewAccount(passphrase)
			if err != nil {
				return err
			}
			logger.Info("account created successfully", "address", account.Address.String())
			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func List() *cobra.Command {
	cmd := cobra.Command{
		Use:   "list",
		Short: "list EVM addresses in key store",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			isInit := store.IsInit(logger, config.Home, store.InitOptions{NeedEVMKeyStore: true})

			// initialize the store if not initialized
			if !isInit {
				return store.ErrNotInited
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			logger.Info("listing accounts available in store")

			for _, acc := range s.EVMKeyStore.Accounts() {
				logger.Info(acc.Address.String())
			}

			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func Delete() *cobra.Command {
	cmd := cobra.Command{
		Use:   "delete <account address in hex>",
		Args:  cobra.ExactArgs(1),
		Short: "delete an EVM addresses from the key store",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			isInit := store.IsInit(logger, config.Home, store.InitOptions{NeedEVMKeyStore: true})

			// initialize the store if not initialized
			if !isInit {
				return store.ErrNotInited
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			if !common.IsHexAddress(args[0]) {
				logger.Error("provided address is not a correct EVM address", "address", args[0])
				return nil // should we return errors in these cases?
			}

			addr := common.HexToAddress(args[0])
			if !s.EVMKeyStore.HasAddress(addr) {
				logger.Info("account not found in keystore", "address", args[0])
				return nil
			}

			logger.Info("deleting account", "address", args[0])

			var acc accounts.Account
			for _, storeAcc := range s.EVMKeyStore.Accounts() {
				if storeAcc.Address.String() == addr.String() {
					acc = storeAcc
				}
			}

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the address passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			err = s.EVMKeyStore.Delete(acc, passphrase)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return keysConfigFlags(&cmd)
}

func Import() *cobra.Command {
	importCmd := &cobra.Command{
		Use:          "import",
		Short:        "import evm keys to the keystore",
		SilenceUsage: true,
	}

	importCmd.AddCommand(
		ImportFile(),
		ImportECDSA(),
	)

	importCmd.SetHelpCommand(&cobra.Command{})

	return importCmd
}

func ImportFile() *cobra.Command {
	cmd := cobra.Command{
		Use:   "file <path to key file>",
		Args:  cobra.ExactArgs(1),
		Short: "import an EVM address from a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysNewPassphraseConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedEVMKeyStore: true}
			isInit := store.IsInit(logger, config.Home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				err := store.Init(logger, config.Home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			logger.Info("importing account")

			keyFile, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer func(file *os.File) {
				err := file.Close()
				if err != nil {
					logger.Error("error closing key file", "err", err.Error())
				}
			}(keyFile)

			// Read the key keyFile contents into a byte slice
			fileBz, err := io.ReadAll(keyFile)
			if err != nil {
				return err
			}

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the address passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			newPassphrase := config.newPassphrase
			// if the new passphrase is not specified as a flag, ask for it.
			if newPassphrase == "" {
				logger.Info("please provide the address new passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				newPassphrase = string(bzPassphrase)
			}

			account, err := s.EVMKeyStore.Import(fileBz, passphrase, newPassphrase)
			if err != nil {
				return err
			}

			logger.Info("successfully imported file", "address", account.Address.String())
			return nil
		},
	}
	return keysNewPassphraseConfigFlags(&cmd)
}

func ImportECDSA() *cobra.Command {
	cmd := cobra.Command{
		Use:   "ecdsa <private key in hex format>",
		Args:  cobra.ExactArgs(1),
		Short: "import an EVM address from an ECDSA private key",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedEVMKeyStore: true}
			isInit := store.IsInit(logger, config.Home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				err := store.Init(logger, config.Home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			logger.Info("importing account")

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the address passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			ethPrivKey, err := ethcrypto.HexToECDSA(args[0])
			if err != nil {
				return err
			}

			account, err := s.EVMKeyStore.ImportECDSA(ethPrivKey, passphrase)
			if err != nil {
				return err
			}

			logger.Info("successfully imported file", "address", account.Address.String())
			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func Update() *cobra.Command {
	cmd := cobra.Command{
		Use:   "update <account address in hex>",
		Args:  cobra.ExactArgs(1),
		Short: "update an EVM account passphrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := commandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysNewPassphraseConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedEVMKeyStore: true}
			isInit := store.IsInit(logger, config.Home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				return store.ErrNotInited
			}

			// open store
			openOptions := store.OpenOptions{HasEVMKeyStore: true}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			if !common.IsHexAddress(args[0]) {
				logger.Error("provided address is not a correct EVM address", "address", args[0])
				return nil // should we return errors in these cases?
			}

			addr := common.HexToAddress(args[0])
			if !s.EVMKeyStore.HasAddress(addr) {
				logger.Info("account not found in keystore", "address", args[0])
				return nil
			}

			logger.Info("updating account", "address", addr.String())

			var acc accounts.Account
			for _, storeAcc := range s.EVMKeyStore.Accounts() {
				if storeAcc.Address.String() == addr.String() {
					acc = storeAcc
				}
			}

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the address passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			newPassphrase := config.newPassphrase
			// if the new passphrase is not specified as a flag, ask for it.
			if newPassphrase == "" {
				logger.Info("please provide the address new passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				newPassphrase = string(bzPassphrase)
			}

			err = s.EVMKeyStore.Update(acc, passphrase, newPassphrase)
			if err != nil {
				return err
			}

			logger.Info("successfully updated the passphrase", "address", acc.Address.String())
			return nil
		},
	}
	return keysNewPassphraseConfigFlags(&cmd)
}

func commandToServiceName(commandUsage string) (string, error) {
	if strings.Contains(commandUsage, "relayer") {
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
