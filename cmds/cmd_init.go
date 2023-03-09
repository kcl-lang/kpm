// Copyright 2021 The KCL Authors. All rights reserved.

package command

import (
	"os"

	"github.com/urfave/cli/v2"
	ops "kusionstack.io/kpm/pkg/ops"
	"kusionstack.io/kpm/pkg/opt"
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

			err = ops.KpmInit(&opt.InitOptions{
				Name:     modName,
				InitPath: pwd,
			})

			if err == nil {
				reporter.Report("kpm: package '", modName, "' init finished")
			} else {
				reporter.ExitWithReport(err)
			}

			return err
		},
	}
}
