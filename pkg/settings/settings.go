package settings

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// The config.json used to persist user information
const CONFIG_JSON_PATH = ".kpm/config/config.json"

// The kpm.json used to describe the default configuration of kpm.
const KPM_JSON_PATH = ".kpm/config/kpm.json"

// The package-cache path, kpm will try lock 'package-cache' before downloading a package.
const PACKAGE_CACHE_PATH = ".kpm/config/package-cache"

// The kpm configuration
type KpmConf struct {
	DefaultOciRegistry  string
	DefaultOciRepo      string
	DefaultOciPlainHttp *bool `json:",omitempty"`
}

const ON = "on"
const OFF = "off"
const DEFAULT_REGISTRY = "ghcr.io"
const DEFAULT_REPO = "kcl-lang"
const DEFAULT_OCI_PLAIN_HTTP = ON
const DEFAULT_REGISTRY_ENV = "KPM_REG"
const DEFAULT_REPO_ENV = "KPM_REPO"
const DEFAULT_OCI_PLAIN_HTTP_ENV = "OCI_REG_PLAIN_HTTP"

// This is a singleton that loads kpm settings from 'kpm.json'
// and is only initialized on the first call by 'Init()' or 'GetSettings()'
var kpm_settings *Settings
var once sync.Once

// DefaultKpmConf create a default configuration for kpm.
func DefaultKpmConf() KpmConf {
	return KpmConf{
		DefaultOciRegistry:  DEFAULT_REGISTRY,
		DefaultOciRepo:      DEFAULT_REPO,
		DefaultOciPlainHttp: nil,
	}
}

type Settings struct {
	CredentialsFile string
	KpmConfFile     string
	// the default configuration for kpm.
	Conf KpmConf

	// the flock used to lock the 'package-cache' file.
	PackageCacheLock *flock.Flock

	// the error catch from the closure in once.Do()
	ErrorEvent *reporter.KpmEvent
}

// AcquirePackageCacheLock will try to lock the 'package-cache' file.
func (settings *Settings) AcquirePackageCacheLock(logWriter io.Writer) error {
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
		reporter.ReportEventTo(reporter.NewEvent(reporter.WaitingLock, "waiting for package-cache lock..."), logWriter)
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

// DefaultOciRegistry return the default OCI registry 'ghcr.io'.
func (settings *Settings) DefaultOciRegistry() string {
	return settings.Conf.DefaultOciRegistry
}

// DefaultOciRepo return the default OCI repo 'kcl-lang'.
func (settings *Settings) DefaultOciRepo() string {
	return settings.Conf.DefaultOciRepo
}

// DefaultOciPlainHttp return the default OCI plain http 'false'.
// Deprecated: DefaultOciPlainHttp is deprecated, please use ForceOciPlainHttp() instead.
func (settings *Settings) DefaultOciPlainHttp() bool {
	plainHttp, force := settings.ForceOciPlainHttp()
	if force {
		return plainHttp
	}

	// If the force OCI plain http is not set,
	// return the default OCI plain http.
	return true
}

// ForceOciPlainHttp return the force OCI plain http.
func (settings *Settings) ForceOciPlainHttp() (bool, bool) {
	if settings.Conf.DefaultOciPlainHttp == nil {
		return false, false
	}

	return *settings.Conf.DefaultOciPlainHttp, true
}

// DefaultOciRef return the default OCI ref 'ghcr.io/kcl-lang'.
func (settings *Settings) DefaultOciRef() string {
	return utils.JoinPath(settings.Conf.DefaultOciRegistry, settings.Conf.DefaultOciRepo)
}

// / LoadSettingsFromEnv will load the KPM settings from environment variables.
func (settings *Settings) LoadSettingsFromEnv() (*Settings, *reporter.KpmEvent) {
	// Load the environment variable for OCI registry
	reg := os.Getenv(DEFAULT_REGISTRY_ENV)
	if len(reg) > 0 {
		settings.Conf.DefaultOciRegistry = reg
	}

	// Load the environment variable for OCI repository
	repo := os.Getenv(DEFAULT_REPO_ENV)
	if len(repo) > 0 {
		settings.Conf.DefaultOciRepo = repo
	}

	// Load the environment variable for OCI plain HTTP
	plainHttp := os.Getenv(DEFAULT_OCI_PLAIN_HTTP_ENV)
	if len(plainHttp) > 0 {
		isPlainHttp, err := isOn(plainHttp)
		if err != nil {
			return settings, reporter.NewErrorEvent(
				reporter.UnknownEnv,
				err,
				fmt.Sprintf("Invalid value for environment variable '%s': %s", DEFAULT_OCI_PLAIN_HTTP_ENV, plainHttp),
			)
		}
		settings.Conf.DefaultOciPlainHttp = &isPlainHttp
	}

	return settings, nil
}

// isOn converts ON/OFF values from environment variables to boolean.
func isOn(value string) (bool, error) {
	lowerValue := strings.ToLower(value)
	if lowerValue == "on" {
		return true, nil
	} else if lowerValue == "off" {
		return false, nil
	}
	return false, fmt.Errorf("expected 'on' or 'off', got '%s'", value)
}

// GetFullPath returns the full path file path under '$HOME/.kpm/config/'
func GetFullPath(jsonFileName string) (string, error) {
	home, err := env.GetAbsPkgPath()
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to load working directory")
	}

	return filepath.Join(home, jsonFileName), nil
}

// GetSettings will return the kpm setting singleton.
func GetSettings() *Settings {
	once.Do(func() {
		kpm_settings = &Settings{}
		credentialsFile, err := GetFullPath(CONFIG_JSON_PATH)
		if err != nil {
			kpm_settings.ErrorEvent = reporter.NewErrorEvent(
				reporter.FailedLoadSettings,
				err,
				fmt.Sprintf("failed to load config file '%s' for kpm.", credentialsFile),
			)
			return
		}
		kpm_settings.CredentialsFile = credentialsFile
		kpm_settings.KpmConfFile, err = GetFullPath(KPM_JSON_PATH)
		if err != nil {
			kpm_settings.ErrorEvent = reporter.NewErrorEvent(
				reporter.FailedLoadSettings,
				err,
				fmt.Sprintf("failed to load config file '%s' for kpm.", kpm_settings.KpmConfFile),
			)
			return
		}

		conf, err := loadOrCreateDefaultKpmJson()
		if err != nil {
			kpm_settings.ErrorEvent = reporter.NewErrorEvent(
				reporter.FailedLoadSettings,
				err,
				fmt.Sprintf("failed to load config file '%s' for kpm.", kpm_settings.KpmConfFile),
			)
			return
		}

		lockPath, err := GetFullPath(PACKAGE_CACHE_PATH)
		if err != nil {
			kpm_settings.ErrorEvent = reporter.NewErrorEvent(
				reporter.FailedLoadSettings,
				err,
				fmt.Sprintf("failed to load config file '%s' for kpm.", lockPath),
			)
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
				kpm_settings.ErrorEvent = reporter.NewErrorEvent(
					reporter.FailedLoadSettings,
					err,
					fmt.Sprintf("failed to create lock file '%s' for kpm.", lockPath),
				)
				return
			}
			// create a empty file named 'package-cache'.
			_, err = os.Create(lockPath)
			if err != nil {
				kpm_settings.ErrorEvent = reporter.NewErrorEvent(
					reporter.FailedLoadSettings,
					err,
					fmt.Sprintf("failed to create lock file '%s' for kpm.", lockPath),
				)
				return
			}
		}

		kpm_settings.Conf = *conf
		kpm_settings.PackageCacheLock = flock.New(lockPath)
	})

	kpm_settings, err := kpm_settings.LoadSettingsFromEnv()
	if err != (*reporter.KpmEvent)(nil) {
		if kpm_settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
			kpm_settings.ErrorEvent = reporter.NewErrorEvent(
				reporter.UnknownEnv,
				err,
			)
		} else {
			kpm_settings.ErrorEvent = err
		}
	}

	return kpm_settings
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
