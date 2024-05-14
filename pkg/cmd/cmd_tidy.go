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

// NewTidyCmd new a Command for `kpm graph`.
func NewTidyCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "graph",
		Usage:  "prints the module dependency graph",
		Action: func(c *cli.Context) error {
			return KpmTidy(c, kpmcli)
		},
	}
}

func KpmTidy(c *cli.Context, kpmcli *client.KpmClient) error {
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
