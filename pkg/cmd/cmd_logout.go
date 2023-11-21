// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"

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
				return reporter.NewErrorEvent(
					reporter.InvalidCmd,
					fmt.Errorf("registry must be specified"),
				)
			}
			err := kpmcli.LogoutOci(c.Args().First())
			if err != nil {
				return err
			}
			reporter.ReportMsgTo("Logout Succeeded", kpmcli.GetLogWriter())
			return nil
		},
	}
}
