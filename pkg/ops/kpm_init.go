// Copyright 2022 The KCL Authors. All rights reserved.

package ops

import (
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
)

// KpmInit initializes an empty kcl package with default 'kcl.mod' and 'kcl.mod.lock'.
func KpmInit(opt *opt.InitOptions) error {
	return pkg.NewKclPkg(opt).InitEmptyPkg()
}
