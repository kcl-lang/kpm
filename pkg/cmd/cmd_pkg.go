// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/env"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
)

// NewPkgCmd new a Command for `kpm pkg`.
func NewPkgCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "pkg",
		Usage:  "package a kcl package into tar",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "target",
				Usage: "Packaged target path",
			},
		},
		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()

			if err != nil {
				reporter.ExitWithReport("kpm: internal bug: failed to load working directory")
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)

			if err != nil {
				reporter.ExitWithReport("kpm: failed to load package in " + pwd + ".")
				return err
			}

			globalPkgPath, err := env.GetAbsPkgPath()
			if err != nil {
				return err
			}

			err = kclPkg.ValidateKpmHome(globalPkgPath)
			if err != nil {
				return err
			}

			tarPath := c.String("target")

			if len(tarPath) == 0 {
				reporter.Report("kpm: The directory where the tar is generated is required.")
				reporter.ExitWithReport("kpm: run 'kpm pkg help' for more information.")
			}

			err = kclPkg.PackageKclPkg(globalPkgPath, tarPath)

			if err != nil {
				reporter.ExitWithReport("kpm: failed to package pkg " + kclPkg.GetPkgName() + ".")
				return err
			}
			return nil
		},
	}
}
