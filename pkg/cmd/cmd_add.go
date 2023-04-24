// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/opt"
	pkg "kusionstack.io/kpm/pkg/package"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/utils"
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

			if err != nil {
				reporter.Fatal("kpm: internal bugs, please contact us to fix it")
			}

			kpmHome, err := utils.GetAbsKpmHome()
			if err != nil {
				return err
			}

			kclPkg, err := pkg.LoadKclPkg(pwd)
			if err != nil {
				reporter.Fatal("kpm: could not load `kcl.mod` in `", pwd, "`")
			}

			err = kclPkg.ValidateKpmHome(kpmHome)
			if err != nil {
				return err
			}

			gitUrl, err := onlyOnceOption(c, "git")

			if err != nil {
				return nil
			}

			gitTag, err := onlyOnceOption(c, "tag")

			if err != nil {
				return err
			}

			addOpts := opt.AddOptions{
				LocalPath: kpmHome,
				RegistryOpts: opt.RegistryOptions{
					Git: &opt.GitOptions{
						Url: *gitUrl,
						Tag: *gitTag,
					},
				},
			}

			err = addOpts.Validate()
			if err != nil {
				return err
			}

			err = addGitDep(&addOpts, kclPkg)
			if err != nil {
				return err
			}
			reporter.Report("kpm: add dependency '", *gitUrl, "'", "with tag '", *gitTag, "' successfully.")
			return nil
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
	if opt.RegistryOpts.Git == nil {
		reporter.Report("kpm: a value is required for '-git <URI>' but none was supplied")
		reporter.ExitWithReport("kpm: run 'kpm add help' for more information.")
	}

	return kclPkg.AddDeps(opt)
}
