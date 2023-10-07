// Copyright 2022 The KCL Authors. All rights reserved.

package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/cmd"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/version"
)

func main() {
	reporter.InitReporter()
	kpmcli, err := client.NewKpmClient()
	if err != nil {
		reporter.Fatal(err)
	}
	app := cli.NewApp()
	app.Name = "kpm"
	app.Usage = "kpm is a kcl package manager"
	app.Version = version.GetVersionInStr()
	app.UsageText = "kpm  <command> [arguments]..."
	app.Commands = []*cli.Command{
		cmd.NewInitCmd(kpmcli),
		cmd.NewAddCmd(kpmcli),
		cmd.NewPkgCmd(kpmcli),
		cmd.NewMetadataCmd(kpmcli),

		// todo: The following commands are bound to the oci registry.
		// Refactor them to compatible with the other registry.
		cmd.NewRunCmd(kpmcli),
		cmd.NewLoginCmd(kpmcli),
		cmd.NewLogoutCmd(kpmcli),
		cmd.NewPushCmd(kpmcli),
		cmd.NewPullCmd(kpmcli),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  cmd.FLAG_QUIET,
			Usage: "push in vendor mode",
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.Bool(cmd.FLAG_QUIET) {
			kpmcli.SetLogWriter(nil)
		}
		return nil
	}
	err = app.Run(os.Args)
	if err != nil {
		reporter.Fatal(err)
	}
}
