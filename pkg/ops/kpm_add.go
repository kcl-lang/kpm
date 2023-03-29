// Copyright 2022 The KCL Authors. All rights reserved.

package ops

import (
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
)

// KpmInit initializes an empty kcl module.
func KpmAdd(opt *opt.AddOptions, kclPkg *pkg.KclPkg) error {

	if opt.RegistryOpts.Git == nil {
		reporter.Report("kpm: a value is required for '-git <URI>' but none was supplied")
		reporter.ExitWithReport("kpm: run 'kpm add help' for more information.")
	}

	return kclPkg.AddDeps(opt)
}
