package p2p

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/common"
	"github.com/celestiaorg/orchestrator-relayer/store"
	util "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Root() *cobra.Command {
	p2pCmd := &cobra.Command{
		Use:          "p2p",
		Short:        "QGB p2p keys manager",
		SilenceUsage: true,
	}

	p2pCmd.SetHelpCommand(&cobra.Command{})
	p2pCmd.AddCommand(
		Add(),
		List(),
		Import(),
		Delete(),
	)

	return p2pCmd
}

func Add() *cobra.Command {
	cmd := cobra.Command{
		Use:   "add <nickname>",
		Short: "create a new Ed25519 P2P address",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := common.CommandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedP2PKeyStore: true}
			isInit := store.IsInit(logger, config.home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				err := store.Init(logger, config.home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasP2PKeyStore: true}
			s, err := store.OpenStore(logger, config.home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			var nickname string
			if len(args) == 0 {
				k, err := s.P2PKeyStore.List()
				if err != nil {
					return err
				}
				nickname = string(rune(len(k)))
			} else {
				nickname = args[0]
			}

			fmt.Println("generating a new Ed25519 private key", "nickname", nickname)

			sr := util.NewTimeSeededRand()
			priv, _, err := crypto.GenerateEd25519Key(sr)
			if err != nil {
				fmt.Println(err.Error())
				return err
			}

			err = s.P2PKeyStore.Put(nickname, priv)
			if err != nil {
				return err
			}

			logger.Info("key created successfully", "nickname", nickname)
			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func List() *cobra.Command {
	cmd := cobra.Command{
		Use:   "list",
		Short: "list existing p2p addresses",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := common.CommandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedP2PKeyStore: true}
			isInit := store.IsInit(logger, config.home, initOptions)

			// check if not initialized
			if !isInit {
				logger.Info("p2p store not initialized", "path", config.home)
				return store.ErrNotInited
			}

			// open store
			openOptions := store.OpenOptions{HasP2PKeyStore: true}
			s, err := store.OpenStore(logger, config.home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			logger.Info("the p2p keys nicknames list:")

			l, err := s.P2PKeyStore.List()
			if err != nil {
				return err
			}

			for _, k := range l {
				logger.Info(k)
			}

			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func Import() *cobra.Command {
	cmd := cobra.Command{
		Use:   "import <nickname> <private_key_in_hex_without_0x>",
		Short: "import an existing p2p private key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := common.CommandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedP2PKeyStore: true}
			isInit := store.IsInit(logger, config.home, initOptions)

			// initialize if not initialized
			if !isInit {
				err := store.Init(logger, config.home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasP2PKeyStore: true}
			s, err := store.OpenStore(logger, config.home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			bIdentity, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}
			pKey, err := crypto.UnmarshalEd25519PrivateKey(bIdentity)
			if err != nil {
				return err
			}

			err = s.P2PKeyStore.Put(args[0], pKey)
			if err != nil {
				return err
			}

			logger.Info("p2p key added successfully", "nickname", args[0])

			return nil
		},
	}
	return keysConfigFlags(&cmd)
}

func Delete() *cobra.Command {
	cmd := cobra.Command{
		Use:   "delete <nickname>",
		Short: "delete an Ed25519 P2P private key from store",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grandParentName := cmd.Parent().Parent().Parent().Use
			serviceName, err := common.CommandToServiceName(grandParentName)
			if err != nil {
				return err
			}
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{NeedP2PKeyStore: true}
			isInit := store.IsInit(logger, config.home, initOptions)

			// initialize the store if not initialized
			if !isInit {
				err := store.Init(logger, config.home, initOptions)
				if err != nil {
					return err
				}
			}

			// open store
			openOptions := store.OpenOptions{HasP2PKeyStore: true}
			s, err := store.OpenStore(logger, config.home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)

			logger.Info("deleting Ed25519 private key", "nickname", args[0])

			err = s.P2PKeyStore.Delete(args[0])
			if err != nil {
				return err
			}

			logger.Info("key deleted successfully", "nickname", args[0])
			return nil
		},
	}
	return keysConfigFlags(&cmd)
}
