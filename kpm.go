// Copyright 2022 The KCL Authors. All rights reserved.

package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/cmd"
	"kusionstack.io/kpm/pkg/reporter"
)

func main() {
	reporter.InitReporter()
	app := cli.NewApp()
	app.Name = "kpm"
	app.Usage = "kpm is a kcl package manager"
	app.Version = "v0.0.1"
	app.UsageText = "kpm  <command> [arguments]..."
	app.Commands = []*cli.Command{
		cmd.NewInitCmd(),
		cmd.NewAddCmd(),
		cmd.NewPkgCmd(),
		cmd.NewRunCmd(),
	}
	err := app.Run(os.Args)
	if err != nil {
		reporter.Fatal(err)
	}
}
