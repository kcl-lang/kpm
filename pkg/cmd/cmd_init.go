// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/env"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	reporter "kusionstack.io/kpm/pkg/reporter"
)

// NewInitCmd new a Command for `kpm init`.
func NewInitCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "init",
		Usage:  "initialize new module in current directory",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				reporter.Report("kpm: module name must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm init help' for more information.")
			}
			modName := c.Args().First()
			pwd, err := os.Getwd()

			if err != nil {
				reporter.Fatal("kpm: internal bugs, please contact us to fix it")
			}

			initOpts := opt.InitOptions{
				Name:     modName,
				InitPath: pwd,
			}

			err = initOpts.Validate()
			if err != nil {
				return err
			}

			kclPkg := pkg.NewKclPkg(&initOpts)

			globalPkgPath, err := env.GetAbsPkgPath()

			if err != nil {
				return err
			}

			err = kclPkg.ValidateKpmHome(globalPkgPath)

			if err != nil {
				return err
			}

			err = kclPkg.InitEmptyPkg()

			if err == nil {
				reporter.Report("kpm: package '", modName, "' init finished")
			} else {
				reporter.ExitWithReport(err)
			}

			return err
		},
	}
}
