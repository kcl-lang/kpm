// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/mvs"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/semver"
)

// NewUpdateCmd new a Command for `kpm update`.
func NewUpdateCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "update",
		Usage:  "Update dependencies listed in kcl.mod.lock based on kcl.mod",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  FLAG_NO_SUM_CHECK,
				Usage: "do not check the checksum of the package and update kcl.mod.lock",
			},
		},
		Action: func(c *cli.Context) error {
			return KpmUpdate(c, kpmcli)
		},
	}
}

func KpmUpdate(c *cli.Context, kpmcli *client.KpmClient) error {
	kpmcli.SetNoSumCheck(c.Bool(FLAG_NO_SUM_CHECK))

	// acquire the lock of the package cache.
	err := kpmcli.AcquirePackageCacheLock()
	if err != nil {
		return err
	}

	defer func() {
		// release the lock of the package cache after the function returns.
		releaseErr := kpmcli.ReleasePackageCacheLock()
		if releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

	pwd, err := os.Getwd()
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	kclPkg, err := kpmcli.LoadPkgFromPath(pwd)
	if err != nil {
		return err
	}

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	err = kpmcli.UpdateDeps(kclPkg)
	if err != nil {
		return err
	}
	return nil
}

// GetModulesToUpdate validates if the packages is present in kcl.mod file and
// find the latest version if version is not specified. Depending on the value of pkgVersion,
// modulesToUpgrade or modulesToDowngrade will be updated.
func GetModulesToUpdate(kclPkg *pkg.KclPkg, modulesToUpgrade []module.Version, modulesToDowngrade []module.Version, pkgInfo string) error {
	pkgInfo = strings.TrimSpace(pkgInfo)
	pkgName, pkgVersion, err := opt.ParseOciPkgNameAndVersion(pkgInfo)
	if err != nil {
		return err
	}

	var dep pkg.Dependency
	var ok bool
	if dep, ok = kclPkg.Deps.Get(pkgName); !ok {
		return err
	}

	if pkgVersion == "" {
		var releases []string
		releases, err = client.GetReleasesFromSource(dep.GetSourceType(), dep.GetDownloadPath())
		if err != nil {
			return reporter.NewErrorEvent(
				reporter.FailedGetReleases,
				err,
				fmt.Sprintf("failed to get releases for %s", pkgName),
			)
		}
		pkgVersion, err = semver.LatestCompatibleVersion(releases, dep.Version)
		if err != nil {
			return reporter.NewErrorEvent(
				reporter.FailedSelectLatestCompatibleVersion,
				err,
				fmt.Sprintf("failed to find the latest version for %s", pkgName),
			)
		}
	}
	if pkgVersion < dep.Version {
		modulesToDowngrade = append(modulesToDowngrade, module.Version{Path: pkgName, Version: pkgVersion})
	} else if pkgVersion > dep.Version {
		modulesToUpgrade = append(modulesToUpgrade, module.Version{Path: pkgName, Version: pkgVersion})
	}
	return nil
}

// InsertModuleToDeps checks whether module is present in the buildList and it is not the same as the target module,
// and inserts it to the dependencies of kclPkg
func InsertModuleToDeps(kclPkg *pkg.KclPkg, module module.Version, target module.Version, buildList []module.Version, reqs mvs.ReqsGraph) error {
	if module.Path == target.Path || !slices.Contains(buildList, module) {
		return nil
	}
	d := pkg.Dependency{
		Name:    module.Path,
		Version: module.Version,
	}
	d.FullName = d.GenDepFullName()
	_, properties, err := reqs.VertexWithProperties(module)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedGetVertexProperties, err, "failed to get vertex with properties")
	}
	// there must be one property depending on the download source type
	for sourceType, uri := range properties.Attributes {
		d.Source, err = pkg.GenSource(sourceType, uri, module.Version)
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedGenerateSource, err, "failed to generate source")
		}
	}
	kclPkg.ModFile.Dependencies.Deps.Set(module.Path, d)
	return nil
}
