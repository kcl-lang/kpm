package env

import (
	"os"
	"path/filepath"

	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// env name
const PKG_PATH = "KCL_PKG_PATH"
const DEFAULT_PKG_PATH_IN_UER_HOME = ".kcl"
const KPM_SUB_DIR = "kpm"

// GetEnvPkgPath will return the env $KCL_PKG_PATH.
func GetEnvPkgPath() string {
	return os.Getenv(PKG_PATH)
}

// GetKpmSubDir will return the subdir for kpm ".kcl/kpm"
func GetKpmSubDir() string {
	return filepath.Join(DEFAULT_PKG_PATH_IN_UER_HOME, KPM_SUB_DIR)
}

// GetAbsPkgPath will return the absolute path of $KCL_PKG_PATH,
// or the absolute path of the current path if $KCL_PKG_PATH does not exist.
func GetAbsPkgPath() (string, error) {
	kpmHome := GetEnvPkgPath()
	if kpmHome == "" {
		defaultHome, err := utils.CreateSubdirInUserHome(GetKpmSubDir())
		if err != nil {
			return "", reporter.NewErrorEvent(reporter.FailedAccessPkgPath, err, "could not access $KCL_PKG_PATH.")
		}
		kpmHome = defaultHome
	}

	kpmHome, err := filepath.Abs(kpmHome)
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.FailedAccessPkgPath, err, "could not access $KCL_PKG_PATH.")
	}

	return kpmHome, nil
}
