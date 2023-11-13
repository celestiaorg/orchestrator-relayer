package p2p

import (
	"encoding/hex"
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/ipfs/boxo/keystore"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/common"
	"github.com/celestiaorg/orchestrator-relayer/store"
	util "github.com/ipfs/boxo/util"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Root(serviceName string) *cobra.Command {
	p2pCmd := &cobra.Command{
		Use:          "p2p",
		Short:        "Blobstream p2p keys manager",
		SilenceUsage: true,
	}

	p2pCmd.SetHelpCommand(&cobra.Command{})
	p2pCmd.AddCommand(
		Add(serviceName),
		List(serviceName),
		Import(serviceName),
		Delete(serviceName),
	)

	return p2pCmd
}

func Add(serviceName string) *cobra.Command {
	cmd := cobra.Command{
		Use:   "add <nickname>",
		Short: "create a new Ed25519 P2P address",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger, err := base.GetLogger(config.logLevel, config.logFormat)
			if err != nil {
				return err
			}

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
				nickname = fmt.Sprintf("%d", len(k))
			} else {
				nickname = args[0]
			}

			logger.Info("generating a new Ed25519 private key", "nickname", nickname)

			priv, err := GenerateNewEd25519()
			if err != nil {
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
	return keysConfigFlags(&cmd, serviceName)
}

func GenerateNewEd25519() (crypto.PrivKey, error) {
	sr := util.NewTimeSeededRand()
	priv, _, err := crypto.GenerateEd25519Key(sr)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func List(serviceName string) *cobra.Command {
	cmd := cobra.Command{
		Use:   "list",
		Short: "list existing p2p addresses",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger, err := base.GetLogger(config.logLevel, config.logFormat)
			if err != nil {
				return err
			}

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
	return keysConfigFlags(&cmd, serviceName)
}

func Import(serviceName string) *cobra.Command {
	cmd := cobra.Command{
		Use:   "import <nickname> <private_key_in_hex_without_0x>",
		Short: "import an existing p2p private key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger, err := base.GetLogger(config.logLevel, config.logFormat)
			if err != nil {
				return err
			}

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
	return keysConfigFlags(&cmd, serviceName)
}

func Delete(serviceName string) *cobra.Command {
	cmd := cobra.Command{
		Use:   "delete <nickname>",
		Short: "delete an Ed25519 P2P private key from store",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseKeysConfigFlags(cmd, serviceName)
			if err != nil {
				return err
			}

			logger, err := base.GetLogger(config.logLevel, config.logFormat)
			if err != nil {
				return err
			}

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

			confirm := common.ConfirmDeletePrivateKey(logger)
			if !confirm {
				logger.Info("deletion of private key has been cancelled", "nickname", args[0])
				return nil
			}

			err = s.P2PKeyStore.Delete(args[0])
			if err != nil {
				return err
			}

			logger.Info("key deleted successfully", "nickname", args[0])
			return nil
		},
	}
	return keysConfigFlags(&cmd, serviceName)
}

// GetP2PKeyOrGenerateNewOne takes a nickname and either returns its corresponding private key if it
// doesn't exist, return the first key in the store if it doesn't exist, create a new key, store it in the
// keystore, then return it.
func GetP2PKeyOrGenerateNewOne(ks *keystore.FSKeystore, nickname string) (crypto.PrivKey, error) {
	// if the key name is not empty, then we try to get its corresponding key
	if nickname != "" {
		// return the corresponding key or return an error
		return ks.Get(nickname)
	}
	// if not, check if the keystore has any other keys
	nicknames, err := ks.List()
	if err != nil {
		return nil, err
	}
	// if so, get the first key
	if len(nicknames) != 0 {
		return ks.Get(nicknames[0])
	}
	// if not, generate a new key
	priv, err := GenerateNewEd25519()
	if err != nil {
		return nil, err
	}
	// store it under the name "0"
	newKeyNickname := "0"
	err = ks.Put(newKeyNickname, priv)
	if err != nil {
		return nil, err
	}
	// return the newly generated key
	return ks.Get(newKeyNickname)
}
