package env

import (
	"os"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// env name
const PKG_PATH = "KCL_PKG_PATH"
const MODULES_SUB_DIR = "modules"
const KCL_DATA_DIR = "kcl"

// GetEnvPkgPath will return the env $KCL_PKG_PATH.
func GetEnvPkgPath() string {
	return os.Getenv(PKG_PATH)
}

// GetKpmDataDir will return the data directory for kpm following XDG Base Directory Specification.
// It returns $XDG_DATA_HOME/kcl/modules on Unix systems, or the platform-specific equivalent.
func GetKpmDataDir() string {
	dataDir, err := utils.DataDir()
	if err != nil {
		// Fallback to home directory if DataDir fails
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".kcl", MODULES_SUB_DIR)
	}
	return filepath.Join(dataDir, KCL_DATA_DIR, MODULES_SUB_DIR)
}

// GetAbsPkgPath will return the absolute path of $KCL_PKG_PATH,
// or the absolute path of the XDG data directory if $KCL_PKG_PATH does not exist.
func GetAbsPkgPath() (string, error) {
	kpmHome := GetEnvPkgPath()
	if kpmHome == "" {
		kpmHome = GetKpmDataDir()
		// Create the directory if it doesn't exist
		if !utils.DirExists(kpmHome) {
			err := os.MkdirAll(kpmHome, 0755)
			if err != nil {
				return "", reporter.NewErrorEvent(reporter.FailedAccessPkgPath, err, "could not create kpm data directory.")
			}
		}
	}

	kpmHome, err := filepath.Abs(kpmHome)
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedAccessPkgPath, err, "could not access $KCL_PKG_PATH.")
	}

	return kpmHome, nil
}
