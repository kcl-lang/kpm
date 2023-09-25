// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewLogoutCmd new a Command for `kpm logout`.
func NewLogoutCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "logout",
		Usage:  "logout from a registry",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				reporter.Report("kpm: registry must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm registry help' for more information.")
			}
			err := kpmcli.LogoutOci(c.Args().First())
			if err != nil {
				return err
			}

			return nil
		},
	}
}
