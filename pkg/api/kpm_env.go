package api

import (
	"kcl-lang.io/kpm/pkg/env"
)

// GetKclPkgPath will return the value of $KCL_PKG_PATH.
//
// If $KCL_PKG_PATH does not exist, it will return $XDG_DATA_HOME/kcl/modules on Unix systems
// or the platform-specific equivalent.
func GetKclPkgPath() (string, error) {
	return env.GetAbsPkgPath()
}
