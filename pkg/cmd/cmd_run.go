// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/api"
	"kcl-lang.io/kpm/pkg/errors"
)

// NewRunCmd new a Command for `kpm run`.
func NewRunCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "run",
		Usage:  "compile kcl package.",
		Flags: []cli.Flag{
			// The entry kcl file.
			&cli.StringSliceFlag{
				Name:  FLAG_INPUT,
				Usage: "a kcl file as the compile entry file",
			},
			&cli.StringFlag{
				Name:  FLAG_TAG,
				Usage: "the tag for oci artifact",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "run in vendor mode",
			},

			// '--kcl' will pass the arguments to kcl.
			&cli.StringFlag{
				Name:  FLAG_KCL,
				Value: "",
				Usage: "Arguments for kcl",
			},
		},
		Action: func(c *cli.Context) error {
			pkgWillBeCompiled := c.Args().First()
			// 'kpm run' compile the current package undor '$pwd'.
			if len(pkgWillBeCompiled) == 0 {
				compileResult, err := api.RunPkg(c.StringSlice(FLAG_INPUT), c.Bool(FLAG_VENDOR), c.String(FLAG_KCL))
				if err != nil {
					return err
				}
				fmt.Println(compileResult)
			} else {
				// 'kpm run <package source>' compile the kcl package from the <package source>.
				compileResult, err := api.RunTar(pkgWillBeCompiled, c.StringSlice(FLAG_INPUT), c.Bool(FLAG_VENDOR), c.String(FLAG_KCL))
				if err == errors.InvalidKclPacakgeTar {
					compileResult, err = api.RunOci(pkgWillBeCompiled, c.String(FLAG_TAG), c.StringSlice(FLAG_INPUT), c.Bool(FLAG_VENDOR), c.String(FLAG_KCL))
					if err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
				fmt.Println(compileResult)
			}
			return nil
		},
	}
}
