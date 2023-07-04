// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// NewLoginCmd new a Command for `kpm login`.
func NewLoginCmd(settings *settings.Settings) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "login",
		Usage:  "login to a registry",
		Flags: []cli.Flag{
			// The registry username.
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Usage:   "registry username",
			},
			// The registry registry password or identity token.
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "registry password or identity token",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				reporter.Report("kpm: registry must be specified.")
				reporter.ExitWithReport("kpm: run 'kpm registry help' for more information.")
			}
			registry := c.Args().First()

			username, password, err := utils.GetUsernamePassword(c.String("username"), c.String("password"), c.Bool("password-stdin"))
			if err != nil {
				return err
			}

			err = oci.Login(registry, username, password, settings)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
