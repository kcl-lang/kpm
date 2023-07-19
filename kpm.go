// Copyright 2022 The KCL Authors. All rights reserved.

package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/cmd"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/version"
)

func main() {
	reporter.InitReporter()
	setting := settings.GetSettings()
	if setting.ErrorEvent != nil {
		reporter.Fatal(setting.ErrorEvent)
	}
	app := cli.NewApp()
	app.Name = "kpm"
	app.Usage = "kpm is a kcl package manager"
	app.Version = version.GetVersionInStr()
	app.UsageText = "kpm  <command> [arguments]..."
	app.Commands = []*cli.Command{
		cmd.NewInitCmd(),
		cmd.NewAddCmd(),
		cmd.NewPkgCmd(),
		cmd.NewMetadataCmd(),

		// todo: The following commands are bound to the oci registry.
		// Refactor them to compatible with the other registry.
		cmd.NewRunCmd(),
		cmd.NewLoginCmd(setting),
		cmd.NewLogoutCmd(setting),
		cmd.NewPushCmd(setting),
		cmd.NewPullCmd(),
	}
	err := app.Run(os.Args)
	if err != nil {
		reporter.Fatal(err)
	}
}
