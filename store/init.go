package store

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/celestiaorg/orchestrator-relayer/store/fslock"
	"github.com/mitchellh/go-homedir"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

const (
	// DataPath the subdir for the data folder containing the p2p data relative to the path.
	DataPath = "data"
	// SignaturePath the subdir for the signatures folder containing all the signatures that the relayer query.
	SignaturePath = "signatures"
	// EVMKeyStorePath the subdir for the path containing the EVM keystore.
	EVMKeyStorePath = "keystore/evm"
	// P2PKeyStorePath the subdir for the path containing the p2p keystore.
	P2PKeyStorePath = "keystore/p2p"
)

// storePath clean up the store path.
func storePath(path string) (string, error) {
	return homedir.Expand(filepath.Clean(path))
}

// InitOptions contains the options used to init a path or check if a path
// is already initiated.
type InitOptions struct {
	NeedDataStore      bool
	NeedSignatureStore bool
	NeedEVMKeyStore    bool
	NeedP2PKeyStore    bool
}

// Init initializes the Blobstream file system in the directory under
// 'path'.
// It also creates a lock under that directory, so it can't be used
// by multiple processes.
func Init(log tmlog.Logger, path string, options InitOptions) error {
	path, err := storePath(path)
	if err != nil {
		return err
	}
	log.Info("initializing qgb store", "path", path)

	err = initRoot(path)
	if err != nil {
		return err
	}

	flock, err := fslock.Lock(lockPath(path))
	if err != nil {
		if errors.Is(err, fslock.ErrLocked) {
			return ErrOpened
		}
		return err
	}

	if options.NeedDataStore {
		err = initDir(dataPath(path))
		if err != nil {
			return err
		}

		log.Info("data dir initialized", "path", dataPath(path))
	}

	if options.NeedSignatureStore {
		err = initDir(signaturePath(path))
		if err != nil {
			return err
		}

		log.Info("signature dir initialized", "path", signaturePath(path))
	}

	if options.NeedP2PKeyStore {
		err = initDir(p2pKeyStorePath(path))
		if err != nil {
			return err
		}

		log.Info("p2p keystore dir initialized", "path", p2pKeyStorePath(path))
	}

	if options.NeedEVMKeyStore {
		err = initDir(evmKeyStorePath(path))
		if err != nil {
			return err
		}

		log.Info("evm keystore dir initialized", "path", evmKeyStorePath(path))
	}

	err = flock.Unlock()
	if err != nil {
		return err
	}

	log.Info("qgb store initialized", "path", path)

	return nil
}

// IsInit checks whether FileSystem Store was set up under given 'path'.
// If the paths of the provided options don't exist, then it returns false.
func IsInit(logger tmlog.Logger, path string, options InitOptions) bool {
	path, err := storePath(path)
	if err != nil {
		logger.Error("parsing store path", "path", path, "err", err)
		return false
	}

	// check if the root path exists
	if !Exists(path) {
		return false
	}

	// check if the data store exists if it's needed
	if options.NeedDataStore && !Exists(dataPath(path)) {
		logger.Info("data path not initialized", "path", path)
		return false
	}

	// check if the signature store exists if it's needed
	if options.NeedSignatureStore && !Exists(signaturePath(path)) {
		logger.Info("signature path not initialized", "path", path)
		return false
	}

	// check if the p2p key store path exists if it's needed
	if options.NeedP2PKeyStore && !Exists(p2pKeyStorePath(path)) {
		logger.Info("p2p keystore not initialized", "path", path)
		return false
	}

	// check if the EVM key store path exists if it's needed
	if options.NeedEVMKeyStore && !Exists(evmKeyStorePath(path)) {
		logger.Info("evm keystore not initialized", "path", path)
		return false
	}

	return true
}

const perms = 0o755

// initRoot initializes(creates) directory if not created and check if it is writable
func initRoot(path string) error {
	err := initDir(path)
	if err != nil {
		return err
	}

	// check for writing permissions
	f, err := os.Create(filepath.Join(path, ".check"))
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return os.Remove(f.Name())
}

// initDir creates a dir if not exist
func initDir(path string) error {
	if Exists(path) {
		return nil
	}
	return os.MkdirAll(path, perms)
}

// Exists checks whether file or directory exists under the given 'path' on the system.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
