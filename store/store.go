package store

import (
	"errors"
	"fmt"
	"path/filepath"

	keystore2 "github.com/ipfs/boxo/keystore"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/celestiaorg/orchestrator-relayer/store/fslock"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// Store contains relevant information about the BlobStream store.
type Store struct {
	// DataStore provides a Datastore - a KV store for dht p2p data to be stored on disk.
	DataStore datastore.Batching

	// SignatureStore provides a signature store - a KV store for all orchestrator signatures to be stored on disk.
	SignatureStore *badger.Datastore

	// EVMKeyStore provides a keystore for EVM private keys.
	EVMKeyStore *keystore.KeyStore

	// P2PKeyStore provides a keystore for P2P private keys.
	P2PKeyStore *keystore2.FSKeystore

	// Path the path to the BlobStream storage root.
	Path string

	// storeLock protects directory when the data store is open.
	storeLock *fslock.Locker
}

// OpenOptions contains the options used to create the store
type OpenOptions struct {
	HasDataStore      bool
	BadgerOptions     *badger.Options
	HasSignatureStore bool
	HasEVMKeyStore    bool
	HasP2PKeyStore    bool
}

// OpenStore creates new FS Store under the given 'path'.
// To be opened, the Store must be initialized first, otherwise ErrNotInited is thrown.
// OpenStore takes a file Lock on directory, hence only one Store can be opened at a time under the
// given 'path', otherwise ErrOpened is thrown.
// The store is locked only in the case of also opening the data store, however, in the case
// of the keys, the store can still be opened.
func OpenStore(logger tmlog.Logger, path string, options OpenOptions) (*Store, error) {
	path, err := storePath(path)
	if err != nil {
		return nil, err
	}

	ok := IsInit(logger, path, InitOptions{
		NeedDataStore:      options.HasDataStore,
		NeedEVMKeyStore:    options.HasEVMKeyStore,
		NeedP2PKeyStore:    options.HasP2PKeyStore,
		NeedSignatureStore: options.HasSignatureStore,
	})
	if !ok {
		return nil, ErrNotInited
	}

	var flock *fslock.Locker
	if options.HasDataStore || options.HasSignatureStore {
		flock, err = fslock.Lock(lockPath(path))
		if err != nil {
			if errors.Is(err, fslock.ErrLocked) {
				return nil, ErrOpened
			}
			return nil, err
		}
		if options.BadgerOptions == nil {
			flock.Unlock() //nolint: errcheck
			return nil, fmt.Errorf("badger store options needed to open the store")
		}
	}

	var ds *badger.Datastore
	if options.HasDataStore {
		ds, err = badger.NewDatastore(dataPath(path), options.BadgerOptions)
		if err != nil {
			flock.Unlock() //nolint: errcheck
			return nil, fmt.Errorf("can't open Badger Datastore: %w", err)
		}
	}

	var sigStore *badger.Datastore
	if options.HasSignatureStore {
		sigStore, err = badger.NewDatastore(signaturePath(path), options.BadgerOptions)
		if err != nil {
			flock.Unlock() //nolint: errcheck
			return nil, fmt.Errorf("can't open Badger SignatureStore: %w", err)
		}
	}

	var evmKs *keystore.KeyStore
	if options.HasEVMKeyStore {
		evmKs = keystore.NewKeyStore(evmKeyStorePath(path), keystore.StandardScryptN, keystore.StandardScryptP)
	}

	var p2pKs *keystore2.FSKeystore
	if options.HasP2PKeyStore {
		p2pKs, err = keystore2.NewFSKeystore(p2pKeyStorePath(path))
		if err != nil {
			logger.Error("couldn't open p2p keystore", "path", p2pKeyStorePath(path))
			return nil, err
		}
	}

	logger.Info("successfully opened store", "path", path)

	return &Store{
		storeLock:      flock,
		Path:           path,
		DataStore:      ds,
		SignatureStore: sigStore,
		EVMKeyStore:    evmKs,
		P2PKeyStore:    p2pKs,
	}, nil
}

// Close closes an opened store and removes the lock file.
func (s Store) Close(logger tmlog.Logger, options OpenOptions) error {
	if options.HasDataStore || options.HasSignatureStore {
		err := s.storeLock.Unlock()
		if err != nil {
			logger.Info("couldn't unlock store", "path", s.Path, "err", err.Error())
			return err
		}
	}
	if options.HasDataStore {
		err := s.DataStore.Close()
		if err != nil {
			logger.Info("couldn't close data store", "path", s.Path, "err", err.Error())
			return err
		}
	}
	if options.HasSignatureStore {
		err := s.SignatureStore.Close()
		if err != nil {
			logger.Info("couldn't close signature store", "path", s.Path, "err", err.Error())
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

// signaturePath returns the relayer signatures folder path relative to the base
func signaturePath(base string) string {
	return filepath.Join(base, SignaturePath)
}

// evmKeyStorePath returns the evm keystore folder path relative to the base
func evmKeyStorePath(base string) string {
	return filepath.Join(base, EVMKeyStorePath)
}

// p2pKeyStorePath returns the p2p keystore folder path relative to the base
func p2pKeyStorePath(base string) string {
	return filepath.Join(base, P2PKeyStorePath)
}
