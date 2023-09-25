// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
)

// NewPullCmd new a Command for `kpm pull`.
func NewPullCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "pull",
		Usage:  "pull kcl package from OCI registry.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  FLAG_TAG,
				Usage: "the tag for oci artifact",
			},
		},
		Action: func(c *cli.Context) error {
			return KpmPull(c, kpmcli)
		},
	}
}

func KpmPull(c *cli.Context, kpmcli *client.KpmClient) error {
	return kpmcli.PullFromOci(c.Args().Get(1), c.Args().Get(0), c.String(FLAG_TAG))
}
