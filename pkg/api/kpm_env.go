package api

import (
	"kcl-lang.io/kpm/pkg/env"
)

// GetKclPkgPath will return the value of $KCL_PKG_PATH.
//
// If $KCL_PKG_PATH does not exist, it will return '$HOME/.kcl/kpm' by default.
func GetKclPkgPath() (string, error) {
	return env.GetAbsPkgPath()
}
