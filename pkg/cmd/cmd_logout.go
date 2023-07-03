// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
)

// NewLogoutCmd new a Command for `kpm logout`.
func NewLogoutCmd(settings *settings.Settings) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "logout",
		Usage:  "logout from a registry",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				reporter.Report("kpm: registry must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm registry help' for more information.")
			}
			registry := c.Args().First()

			err := oci.Logout(registry, settings)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
