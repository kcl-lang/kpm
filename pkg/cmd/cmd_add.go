// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewAddCmd new a Command for `kpm add`.
func NewAddCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "add",
		Usage:  "add new dependency",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "git",
				Usage: "Git repository location",
			},
			&cli.StringSliceFlag{
				Name:  "tag",
				Usage: "Oci or Git repository tag",
			},
			&cli.StringSliceFlag{
				Name:  "commit",
				Usage: "Git repository commit",
			},
			&cli.BoolFlag{
				Name:  FLAG_NO_SUM_CHECK,
				Usage: "do not check the checksum of the package and update kcl.mod.lock",
			},
			&cli.StringFlag{
				Name:  "rename",
				Usage: "rename the package name in kcl.mod.lock",
			},
			&cli.StringSliceFlag{
				Name:  "package",
				Usage: "package name to use in case of git",
			},
		},

		Action: func(c *cli.Context) error {
			return KpmAdd(c, kpmcli)
		},
	}
}

func KpmAdd(c *cli.Context, kpmcli *client.KpmClient) error {
	// acquire the lock of the package cache.
	err := kpmcli.AcquirePackageCacheLock()
	if err != nil {
		return err
	}

	defer func() {
		// release the lock of the package cache after the function returns.
		releaseErr := kpmcli.ReleasePackageCacheLock()
		if releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

	pwd, err := os.Getwd()

	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
	}

	globalPkgPath, err := env.GetAbsPkgPath()
	if err != nil {
		return err
	}

	kclPkg, err := kpmcli.LoadPkgFromPath(pwd)
	if err != nil {
		return err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	addOpts, err := parseAddOptions(c, kpmcli, globalPkgPath)
	if err != nil {
		return err
	}

	if addOpts.RegistryOpts.Local != nil {
		absAddPath, err := filepath.Abs(addOpts.RegistryOpts.Local.Path)
		if err != nil {
			return reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, please contact us to fix it.")
		}
		if absAddPath == kclPkg.HomePath {
			return reporter.NewErrorEvent(
				reporter.AddItselfAsDep,
				fmt.Errorf("cannot add '%s' as a dependency to itself", kclPkg.GetPkgName()),
			)
		}
	}

	err = addOpts.Validate()
	if err != nil {
		return err
	}

	_, err = kpmcli.AddDepWithOpts(kclPkg, addOpts)
	if err != nil {
		return err
	}
	return nil
}

// onlyOnceOption is used to check that the value of some parameters can only appear once.
func onlyOnceOption(c *cli.Context, name string) (string, *reporter.KpmEvent) {
	inputOpt := c.StringSlice(name)
	if len(inputOpt) > 1 {
		return "", reporter.NewErrorEvent(reporter.InvalidCmd, fmt.Errorf("the argument '%s' cannot be used multiple times", name))
	} else if len(inputOpt) == 1 {
		return inputOpt[0], nil
	} else {
		return "", nil
	}
}

// parseAddOptions will parse the user cli inputs.
func parseAddOptions(c *cli.Context, kpmcli *client.KpmClient, localPath string) (*opt.AddOptions, error) {
	noSumCheck := c.Bool(FLAG_NO_SUM_CHECK)
	newPkgName := c.String("rename")
	// parse from 'kpm add -git https://xxx/xxx.git -tag v0.0.1'.
	if c.NArg() == 0 {
		gitOpts, err := parseGitRegistryOptions(c)
		if err != (*reporter.KpmEvent)(nil) {
			if err.Type() == reporter.InvalidGitUrl {
				return nil, reporter.NewErrorEvent(reporter.InvalidCmd, errors.InvalidAddOptions)
			}
			return nil, err
		}
		return &opt.AddOptions{
			LocalPath:    localPath,
			NewPkgName:   newPkgName,
			RegistryOpts: *gitOpts,
			NoSumCheck:   noSumCheck,
		}, nil
	} else {
		regOpt, err := opt.NewRegistryOptionsFrom(c.Args().First(), kpmcli.GetSettings())

		if err != nil {
			return nil, err
		}

		if regOpt.Oci != nil {
			tag, err := onlyOnceOption(c, constants.Tag)

			if err != (*reporter.KpmEvent)(nil) {
				return nil, err
			}

			if len(tag) != 0 {
				regOpt.Oci.Tag = tag
			}
		}

		return &opt.AddOptions{
			LocalPath:    localPath,
			NewPkgName:   newPkgName,
			RegistryOpts: *regOpt,
			NoSumCheck:   noSumCheck,
		}, nil
	}
}

// parseGitRegistryOptions will parse the git registry information from user cli inputs.
func parseGitRegistryOptions(c *cli.Context) (*opt.RegistryOptions, *reporter.KpmEvent) {
	gitUrl, err := onlyOnceOption(c, "git")

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	gitTag, err := onlyOnceOption(c, "tag")

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	gitCommit, err := onlyOnceOption(c, "commit")

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	gitPackage, err := onlyOnceOption(c, "package")

	if err != (*reporter.KpmEvent)(nil) {
		return nil, err
	}

	if gitUrl == "" {
		return nil, reporter.NewErrorEvent(reporter.InvalidGitUrl, fmt.Errorf("the argument 'git' is required"))
	}

	if (gitTag == "" && gitCommit == "") || (gitTag != "" && gitCommit != "") {
		return nil, reporter.NewErrorEvent(reporter.WithoutGitTag, fmt.Errorf("invalid arguments, one of commit or tag should be passed"))
	}

	return &opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url:     gitUrl,
			Tag:     gitTag,
			Commit:  gitCommit,
			Package: gitPackage,
		},
	}, nil
}

// parseOciRegistryOptions will parse the oci registry information from user cli inputs.
func parseRegistryOptions(c *cli.Context, kpmcli *client.KpmClient) (*opt.RegistryOptions, error) {
	ociPkgRef := c.Args().First()
	name, version, err := opt.ParseOciPkgNameAndVersion(ociPkgRef)
	if err != nil {
		return nil, err
	}

	return &opt.RegistryOptions{
		Registry: &opt.OciOptions{
			Reg:     kpmcli.GetSettings().DefaultOciRegistry(),
			Repo:    kpmcli.GetSettings().DefaultOciRepo(),
			PkgName: name,
			Tag:     version,
		},
	}, nil
}
