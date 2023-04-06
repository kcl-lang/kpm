// Copyright 2021 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/ops"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
)

// NewAddCmd new a Command for `kpm add`.
func NewAddCmd() *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "add",
		Usage:  "add new dependancy",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "git",
				Usage: "Git repository location",
			},
			&cli.StringSliceFlag{
				Name:  "tag",
				Usage: "Git repository tag",
			},
		},

		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()

			kpmHome := os.Getenv("KPM_HOME")
			if kpmHome == "" {
				fmt.Println("kpm: KPM_HOME environment variable is not set")
				fmt.Println("kpm: `add` will be downloaded to directory: ", pwd)
			}

			if err != nil {
				reporter.Fatal("kpm: internal bugs, please contact us to fix it")
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)

			if err != nil {
				reporter.Fatal("kpm: could not load `kcl.mod` in `", pwd, "`")
			}

			gitUrl, err := onlyOnceOption(c, "git")

			if err != nil {
				return nil
			}

			gitTag, err := onlyOnceOption(c, "tag")

			if err != nil {
				return err
			}

			return addGitDep(&opt.AddOptions{
				LocalPath: kpmHome,
				RegistryOpts: opt.RegistryOptions{
					Git: &opt.GitOptions{
						Url: *gitUrl,
						Tag: *gitTag,
					},
				},
			}, kclPkg)
		},
	}
}

// onlyOnceOption is used to check that the value of some parameters can only appear once.
func onlyOnceOption(c *cli.Context, name string) (*string, error) {
	inputOpt := c.StringSlice(name)
	if len(inputOpt) > 1 {
		reporter.ExitWithReport("kpm: the argument '", name, "' cannot be used multiple times")
		reporter.ExitWithReport("kpm: run 'kpm add help' for more information.")
		return nil, fmt.Errorf("kpm: Invalid command")
	} else if len(inputOpt) == 1 {
		return &inputOpt[0], nil
	} else {
		reporter.Report("kpm: the following required arguments were not provided: ", name)
		reporter.ExitWithReport("kpm: run 'kpm add help' for more information.")
		return nil, fmt.Errorf("kpm: Invalid command")
	}
}

func addGitDep(opt *opt.AddOptions, kclPkg *pkg.KclPkg) error {
	return ops.KpmAdd(opt, kclPkg)
}
