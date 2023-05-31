package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"kusionstack.io/kpm/pkg/env"
	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/utils"
)

// The config.json used to persist user information
const CONFIG_JSON_PATH = ".kpm/config/config.json"

// The kpm.json used to describe the default configuration of kpm.
const KPM_JSON_PATH = ".kpm/config/kpm.json"

// The package-cache path, kpm will try lock 'package-cache' before downloading a package.
const PACKAGE_CACHE_PATH = ".kpm/config/package-cache"

// The kpm configuration
type KpmConf struct {
	DefaultOciRegistry string
	DefaultOciRepo     string
}

const DEFAULT_REGISTRY = "ghcr.io"
const DEFAULT_REPO = "kusionstack"

// This is a singleton that loads kpm settings from 'kpm.json'
// and is only initialized on the first call by 'Init()' or 'GetSettings()'
var kpm_settings *Settings
var once sync.Once

// DefaultKpmConf create a default configuration for kpm.
func DefaultKpmConf() KpmConf {
	return KpmConf{
		DefaultOciRegistry: DEFAULT_REGISTRY,
		DefaultOciRepo:     DEFAULT_REPO,
	}
}

type Settings struct {
	CredentialsFile string
	// the default configuration for kpm.
	Conf KpmConf

	// the flock used to lock the 'package-cache' file.
	PackageCacheLock *flock.Flock
}

// AcquirePackageCacheLock will try to lock the 'package-cache' file.
func (settings *Settings) AcquirePackageCacheLock() error {
	// if the 'package-cache' file is not initialized, this is an internal bug.
	if settings.PackageCacheLock == nil {
		return errors.InternalBug
	}

	// try to lock the 'package-cache' file
	locked, err := settings.PackageCacheLock.TryLock()
	if err != nil {
		return err
	}

	// if failed to lock the 'package-cache' file, wait until it is unlocked.
	if !locked {
		reporter.Report("kpm: waiting for package-cache lock...")
		for {
			// try to lock the 'package-cache' file
			locked, err = settings.PackageCacheLock.TryLock()
			if err != nil {
				return err
			}
			// if locked, break the loop.
			if locked {
				break
			}
			// when waiting for a file lock, the program will continuously attempt to acquire the lock.
			// without adding a sleep, the program will rapidly try to acquire the lock, consuming a large amount of CPU resources.
			// by adding a sleep, the program can pause for a period of time between each attempt to acquire the lock,
			// reducing the consumption of CPU resources.
			time.Sleep(2 * time.Millisecond)
		}
	}

	return nil
}

// ReleasePackageCacheLock will try to unlock the 'package-cache' file.
func (settings *Settings) ReleasePackageCacheLock() error {
	// if the 'package-cache' file is not initialized, this is an internal bug.
	if settings.PackageCacheLock == nil {
		return errors.InternalBug
	}

	// try to unlock the 'package-cache' file.
	err := settings.PackageCacheLock.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// DefaultOciRepo return the default OCI registry 'ghcr.io'.
func (settings *Settings) DefaultOciRegistry() string {
	return settings.Conf.DefaultOciRegistry
}

// DefaultOciRepo return the default OCI repo 'kusionstack'.
func (settings *Settings) DefaultOciRepo() string {
	return settings.Conf.DefaultOciRepo
}

// GetFullPath returns the full path file path under '$HOME/.kpm/config/'
func GetFullPath(jsonFileName string) (string, error) {
	home, err := env.GetAbsPkgPath()
	if err != nil {
		return "", errors.InternalBug
	}

	return filepath.Join(home, jsonFileName), nil
}

// GetSettings will return the kpm setting singleton.
func GetSettings() (*Settings, error) {
	var err error
	once.Do(func() {
		credentialsFile, err := GetFullPath(CONFIG_JSON_PATH)
		if err != nil {
			return
		}

		conf, err := loadOrCreateDefaultKpmJson()
		if err != nil {
			return
		}

		lockPath, err := GetFullPath(PACKAGE_CACHE_PATH)
		if err != nil {
			return
		}

		// If the 'lockPath' file exists, do nothing.
		// If the 'lockPath' file does not exist, recursively create the 'lockPath' path.
		// If the 'lockPath' path cannot be created, return an error.
		// 'lockPath' is a file path not a directory path.
		if !utils.DirExists(lockPath) {
			// recursively create the 'lockPath' path.
			err = os.MkdirAll(filepath.Dir(lockPath), 0755)
			if err != nil {
				return
			}
			// create a empty file named 'package-cache'.
			_, err = os.Create(lockPath)
			if err != nil {
				return
			}
		}

		kpm_settings = &Settings{
			CredentialsFile:  credentialsFile,
			Conf:             *conf,
			PackageCacheLock: flock.New(lockPath),
		}
	})

	return kpm_settings, err
}

// loadOrCreateDefaultKpmJson will load the 'kpm.json' file from '$KCL_PKG_PATH/.kpm/config',
// and create a default 'kpm.json' file if the file does not exist.
func loadOrCreateDefaultKpmJson() (*KpmConf, error) {
	kpmConfpath, err := GetFullPath(KPM_JSON_PATH)
	if err != nil {
		return nil, err
	}

	defaultKpmConf := DefaultKpmConf()

	b, err := os.ReadFile(kpmConfpath)
	// if the file '$KCL_PKG_PATH/.kpm/config/kpm.json' does not exist
	if os.IsNotExist(err) {
		// create the default kpm.json.
		err = os.MkdirAll(filepath.Dir(kpmConfpath), 0755)
		if err != nil {
			return nil, err
		}

		b, err := json.Marshal(defaultKpmConf)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(kpmConfpath, b, 0644)
		if err != nil {
			return nil, err
		}
		return &defaultKpmConf, nil
	} else {
		err = json.Unmarshal(b, &defaultKpmConf)
		if err != nil {
			return nil, err
		}
		return &defaultKpmConf, nil
	}
}
