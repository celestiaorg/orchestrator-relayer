package evm

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	common2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/common"
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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

			passphrase := config.EVMPassphrase
			// if the new passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = GetNewPassphrase()
				if err != nil {
					return err
				}
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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

			logger.Info("deleting account", "address", args[0])

			acc, err := GetAccountFromStore(s.EVMKeyStore, args[0])
			if err != nil {
				return err
			}

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = GetPassphrase()
				if err != nil {
					return err
				}
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = GetPassphrase()
				if err != nil {
					return err
				}
			}

			newPassphrase := config.newPassphrase
			// if the new passphrase is not specified as a flag, ask for it.
			if newPassphrase == "" {
				newPassphrase, err = GetNewPassphrase()
				if err != nil {
					return err
				}
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = GetNewPassphrase()
				if err != nil {
					return err
				}
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
			serviceName, err := common2.CommandToServiceName(grandParentName)
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

			logger.Info("updating account", "address", args[0])

			acc, err := GetAccountFromStore(s.EVMKeyStore, args[0])
			if err != nil {
				return err
			}

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = GetPassphrase()
				if err != nil {
					return err
				}
			}

			newPassphrase := config.newPassphrase
			// if the new passphrase is not specified as a flag, ask for it.
			if newPassphrase == "" {
				newPassphrase, err = GetNewPassphrase()
				if err != nil {
					return err
				}
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

// GetAccountFromStore takes an EVM store and an EVM address and loads the corresponding account from it.
func GetAccountFromStore(ks *keystore.KeyStore, evmAddr string) (accounts.Account, error) {
	if !common.IsHexAddress(evmAddr) {
		return accounts.Account{}, fmt.Errorf("provided address is not a correct EVM address %s", evmAddr)
	}

	addr := common.HexToAddress(evmAddr)
	if !ks.HasAddress(addr) {
		return accounts.Account{}, fmt.Errorf("account not found in keystore %s", evmAddr)
	}

	var acc accounts.Account
	for _, storeAcc := range ks.Accounts() {
		if storeAcc.Address.String() == addr.String() {
			acc = storeAcc
		}
	}

	return acc, nil
}

func GetPassphrase() (string, error) {
	fmt.Print("please provide the account passphrase: ")
	bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(bzPassphrase), nil
}

func GetNewPassphrase() (string, error) {
	var err error
	var bzPassphrase []byte
	for {
		fmt.Print("please provide the account new passphrase: ")
		bzPassphrase, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Print("enter the same passphrase again: ")
		bzPassphraseConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		if bytes.Compare(bzPassphrase, bzPassphraseConfirm) == 0 {
			break
		}
	}
	return string(bzPassphrase), nil
}
