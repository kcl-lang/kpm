package client

import (
	"kcl-lang.io/kpm/pkg/env"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// PackagePkg will package the current kcl package into a "*.tar" file in under the package path.
func (c *KpmClient) PackagePkg(kclPkg *pkg.KclPkg, vendorMode bool) (string, error) {
	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return "", err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return "", err
	}

	err = c.Package(kclPkg, kclPkg.DefaultTarPath(), vendorMode)

	if err != nil {
		reporter.ExitWithReport("failed to package pkg " + kclPkg.GetPkgName() + ".")
		return "", err
	}
	return kclPkg.DefaultTarPath(), nil
}

// Package will package the current kcl package into a "*.tar" file into 'tarPath'.
func (c *KpmClient) Package(kclPkg *pkg.KclPkg, tarPath string, vendorMode bool) error {
	// Vendor all the dependencies into the current kcl package.
	if vendorMode {
		err := c.VendorDeps(kclPkg)
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedVendor, err, "failed to vendor dependencies")
		}
	}

	// Tar the current kcl package into a "*.tar" file.
	err := utils.TarDir(kclPkg.HomePath, tarPath, kclPkg.GetPkgInclude(), kclPkg.GetPkgExclude())
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedPackage, err, "failed to package the kcl module")
	}
	return nil
}
