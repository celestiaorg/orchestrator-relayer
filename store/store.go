package store

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/celestiaorg/orchestrator-relayer/store/fslock"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// Store contains relevant information about the QGB store.
type Store struct {
	// Datastore provides a Datastore - a KV store for dht p2p data to be stored on disk.
	DataStore datastore.Batching

	// EVMKeyStore provides a keystore for EVM addresses
	EVMKeyStore *keystore.KeyStore

	// Path the path to the qgb storage root
	Path string

	// protects directory
	dirLock *fslock.Locker
}

// OpenOptions contains the options used to create the store
type OpenOptions struct {
	HasDataStore   bool
	BadgerOptions  *badger.Options
	HasEVMKeyStore bool
	HasP2PKeyStore bool
}

// OpenStore creates new FS Store under the given 'path'.
// To be opened the Store must be initialized first, otherwise ErrNotInited is thrown.
// OpenStore takes a file Lock on directory, hence only one Store can be opened at a time under the
// given 'path', otherwise ErrOpened is thrown.
func OpenStore(logger tmlog.Logger, path string, options OpenOptions) (*Store, error) {
	path, err := storePath(path)
	if err != nil {
		return nil, err
	}

	flock, err := fslock.Lock(lockPath(path))
	if err != nil {
		if err == fslock.ErrLocked {
			return nil, ErrOpened
		}
		return nil, err
	}

	ok := IsInit(logger, path, InitOptions{
		NeedDataStore:   options.HasDataStore,
		NeedEVMKeyStore: options.HasEVMKeyStore,
		NeedP2PKeyStore: options.HasP2PKeyStore,
	})
	if !ok {
		flock.Unlock() //nolint: errcheck
		return nil, ErrNotInited
	}

	var ds *badger.Datastore
	if options.HasDataStore {
		if options.BadgerOptions == nil {
			return nil, fmt.Errorf("badger data store options needed to open the store")
		}
		ds, err = badger.NewDatastore(dataPath(path), options.BadgerOptions)
		if err != nil {
			return nil, fmt.Errorf("can't open Badger Datastore: %w", err)
		}
	}

	var ks *keystore.KeyStore
	if options.HasEVMKeyStore {
		ks = keystore.NewKeyStore(evmKeyStorePath(path), keystore.StandardScryptN, keystore.StandardScryptP)
	}

	logger.Info("successfully opened store", "path", path)

	return &Store{dirLock: flock, Path: path, DataStore: ds, EVMKeyStore: ks}, nil
}

// Close closes an opened store and removes the lock file.
func (s Store) Close(logger tmlog.Logger, options OpenOptions) error {
	err := s.dirLock.Unlock()
	if err != nil {
		logger.Info("couldn't unlock store", "path", s.Path, "err", err.Error())
		return err
	}
	if options.HasDataStore {
		err = s.DataStore.Close()
		if err != nil {
			logger.Info("couldn't close data store", "path", s.Path, "err", err.Error())
			return err
		}
	}
	logger.Info("successfully closed store", "path", s.Path)
	return nil
}

// lockPath returns the path to the lock file relative to the base directory.
func lockPath(base string) string {
	return filepath.Join(base, "lock")
}

// dataPath returns the data folder path relative to the base
func dataPath(base string) string {
	return filepath.Join(base, DataPath)
}

// evmKeyStorePath returns the evm keystore folder path relative to the base
func evmKeyStorePath(base string) string {
	return filepath.Join(base, EVMKeyStorePath)
}

// p2pKeyStorePath returns the p2p keystore folder path relative to the base
func p2pKeyStorePath(base string) string {
	return filepath.Join(base, P2PKeyStorePath)
}
