// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// NewPkgCmd new a Command for `kpm pkg`.
func NewPkgCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "pkg",
		Usage:  "package a kcl package into tar",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "target",
				Usage: "Packaged target path",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "push in vendor mode",
			},
		},
		Action: func(c *cli.Context) error {
			tarPath := c.String("target")

			if len(tarPath) == 0 {
				return reporter.NewErrorEvent(
					reporter.InvalidCmd,
					fmt.Errorf("the directory where the tar is generated is required"),
				)
			}

			pwd, err := os.Getwd()

			if err != nil {
				return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to load working directory.")
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)

			if err != nil {
				reporter.ExitWithReport("failed to load package in " + pwd + ".")
				return err
			}

			// If the file path used to save the package tar file does not exist, create this file path.
			if !utils.DirExists(tarPath) {
				err := os.MkdirAll(tarPath, os.ModePerm)
				if err != nil {
					return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to create the target directory")
				}
			}

			return kpmcli.Package(kclPkg, filepath.Join(tarPath, kclPkg.GetPkgTarName()), c.Bool(FLAG_VENDOR))
		},
	}
}
