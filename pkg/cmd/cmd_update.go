// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/module"
	mvsExt "kcl-lang.io/kpm/pkg/3rdparty/mvs"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/mvs"
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

	pkgInfos := c.Args().Slice()

	pwd, err := os.Getwd()
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	fetchNameAndVersion := func(pkgInfo string) (string, string) {
		// the version is specified after :
		if strings.Contains(pkgInfo, ":") {
			return strings.Split(pkgInfo, ":")[0], strings.Split(pkgInfo, ":")[1]
		}
		return pkgInfo, ""
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

	var (
		modulesToUpgrade   []module.Version
		modulesToDowngrade []module.Version
	)

	// validate if the packages is present in kcl.mod file and
	// find the latest version if version is not specified
	for _, pkgInfo := range pkgInfos {
		pkgInfo = strings.TrimSpace(pkgInfo)
		pkgName, pkgVersion := fetchNameAndVersion(pkgInfo)

		var dep pkg.Dependency
		var ok bool
		if dep, ok = kclPkg.Deps[pkgName]; !ok {
			reporter.Report(fmt.Sprintf("package %s not found in kcl.mod file", pkgName))
			continue
		}

		if pkgVersion == "" {
			var releases []string
			if dep.Git != nil {
				releases, err = mvs.GetReleasesFromSource(dep.GetSourceType(), dep.Git.Url)
			} else if dep.Oci != nil {
				releases, err = mvs.GetReleasesFromSource(dep.GetSourceType(), "fjdkasd")
			}
			if err != nil {
				reporter.ReportEventToStdout(reporter.NewEvent(
					reporter.FailedGetReleases,
					fmt.Sprintf("failed to get releases for %s", pkgName),
				))
				continue
			}
			pkgVersion, err = semver.LatestCompatibleVersion(releases, dep.Version)
			if err != nil {
				reporter.NewErrorEvent(
					reporter.FailedSelectLatestCompatibleVersion,
					err,
					fmt.Sprintf("failed to find the latest version for %s", pkgName),
				)
				continue
			}
		}
		if pkgVersion < dep.Version {
			modulesToDowngrade = append(modulesToDowngrade, module.Version{Path: pkgName, Version: pkgVersion})
		} else if pkgVersion > dep.Version {
			modulesToUpgrade = append(modulesToUpgrade, module.Version{Path: pkgName, Version: pkgVersion})
		}
	}

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	if err != nil {
		return err
	}

	reqs := mvs.ReqsGraph{
		Graph:     depGraph,
		KpmClient: kpmcli,
		KpmPkg:    kclPkg,
	}

	target := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()}
	buildList, err := UpdateBuildList(target, modulesToUpgrade, modulesToDowngrade, &reqs)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedUpdatingBuildList, err, "failed to update build list")
	}

	modules, err := graph.TopologicalSort(depGraph)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedTopologicalSort, err, "failed to sort the dependencies")
	}

	kclPkg.ModFile.Dependencies.Deps = make(map[string]pkg.Dependency)

	for _, module := range modules {
		if module.Path == target.Path || !slices.Contains(buildList, module) {
			continue
		}
		d := pkg.Dependency{
			Name:    module.Path,
			Version: module.Version,
		}
		d.FullName = d.GenDepFullName()
		_, properties, err := depGraph.VertexWithProperties(module)
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedGetVertexProperties, err, "failed to get vertex with properties")
		}
		// there must be one property depending on the download source type
		for sourceType, uri := range properties.Attributes {
			d.Source, err = pkg.GenSource(sourceType, uri, module.Version)
			if err != nil {
				return reporter.NewErrorEvent(reporter.FailedGenerateSource, err, "failed to generate source")
			}
			// if d.Oci != nil && d.Oci.Reg == "ghcr.io" {
			// 	d.Oci = nil
			// }
		}
		kclPkg.ModFile.Dependencies.Deps[module.Path] = d
	}

	err = kpmcli.UpdateDeps(kclPkg)
	if err != nil {
		return err
	}
	return nil
}

func UpdateBuildList(target module.Version, modulesToUpgrade []module.Version, modulesToDowngrade []module.Version, reqs *mvs.ReqsGraph) ([]module.Version, error) {
	var (
		UpdBuildLists []module.Version
		err           error
	)

	if len(modulesToUpgrade) == 0 {
		UpdBuildLists, err = mvsExt.UpgradeAll(target, reqs)
	} else {
		UpdBuildLists, err = mvsExt.Upgrade(target, reqs, modulesToUpgrade...)
	}
	if err != nil {
		return []module.Version{}, err
	}

	if len(modulesToDowngrade) != 0 {
		UpdBuildLists, err = mvsExt.Downgrade(target, reqs, modulesToDowngrade...)
	}
	if err != nil {
		return []module.Version{}, err
	}

	return UpdBuildLists, nil
}
