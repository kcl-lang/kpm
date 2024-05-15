// Copyright 2024 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/dominikbraun/graph"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/module"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/env"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewGraphCmd new a Command for `kpm graph`.
func NewGraphCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "graph",
		Usage:  "prints the module dependency graph",
		Action: func(c *cli.Context) error {
			return KpmGraph(c, kpmcli)
		},
	}
}

func KpmGraph(c *cli.Context, kpmcli *client.KpmClient) error {
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

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return err
	}

	kclPkg, err := pkg.LoadKclPkg(pwd)
	if err != nil {
		return err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	_, depGraph, err := kpmcli.InitGraphAndDownloadDeps(kclPkg)
	if err != nil {
		return err
	}

	adjMap, err := depGraph.AdjacencyMap()
	if err != nil {
		return err
	}

	format := func(m module.Version) string {
		formattedMsg := m.Path
		if m.Version != "" {
			formattedMsg +=  "@" + m.Version
		}
		return formattedMsg
	}

	// print the dependency graph to stdout.
	root := module.Version{Path: kclPkg.GetPkgName(), Version: kclPkg.GetPkgVersion()} 
	err = graph.BFS(depGraph, root, func(source module.Version) bool {
		for target := range adjMap[source] {
			reporter.ReportMsgTo(
				fmt.Sprint(format(source), " ", format(target)),
				kpmcli.GetLogWriter(),
			)
		}
		return false
	})
	if err != nil {
		return err
	}
	return nil
}
