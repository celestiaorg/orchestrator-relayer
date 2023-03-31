package store

import (
	"os"
	"path/filepath"

	"github.com/celestiaorg/orchestrator-relayer/store/fslock"
	"github.com/mitchellh/go-homedir"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// DataPath the subdir for the data folder containing the p2p data relative to the path.
const DataPath = "data"

// storePath clean up the store path.
func storePath(path string) (string, error) {
	return homedir.Expand(filepath.Clean(path))
}

// Init initializes the qgb file system in the directory under
// 'path'.
// It also creates a lock under that directory, so it can't be used
// by multiple processes.
func Init(log tmlog.Logger, path string) error {
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
		if err == fslock.ErrLocked {
			return ErrOpened
		}
		return err
	}

	err = initDir(dataPath(path))
	if err != nil {
		return err
	}

	log.Info("data dir initialized", "path", dataPath(path))

	err = flock.Unlock()
	if err != nil {
		return err
	}

	log.Info("qgb store initialized", "path", path)

	return nil
}

// IsInit checks whether FileSystem Store was set up under given 'path'.
// If the path doesn't contain the data folder, then it returns false.
// Other validation will be added when the keystores are added.
func IsInit(logger tmlog.Logger, path string) bool {
	path, err := storePath(path)
	if err != nil {
		logger.Error("parsing store path", "path", path, "err", err)
		return false
	}

	return Exists(dataPath(path))
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
	return os.Mkdir(path, perms)
}

// Exists checks whether file or directory exists under the given 'path' on the system.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
