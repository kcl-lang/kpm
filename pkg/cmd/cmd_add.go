// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/env"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
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
			return KpmAdd(c)
		},
	}
}

func KpmAdd(c *cli.Context) error {
	// 1. get settings from the global config file.
	settings := settings.GetSettings()
	if settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
		return settings.ErrorEvent
	}

	// 2. acquire the lock of the package cache.
	err := settings.AcquirePackageCacheLock()
	if err != nil {
		return err
	}

	defer func() {
		// 3. release the lock of the package cache after the function returns.
		releaseErr := settings.ReleasePackageCacheLock()
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

	kclPkg, err := pkg.LoadKclPkg(pwd)
	if err != nil {
		return err
	}

	err = kclPkg.ValidateKpmHome(globalPkgPath)
	if err != (*reporter.KpmEvent)(nil) {
		return err
	}

	addOpts, err := parseAddOptions(c, globalPkgPath)
	if err != nil {
		return err
	}

	err = addOpts.Validate()
	if err != nil {
		return err
	}

	err = kclPkg.AddDeps(addOpts)
	if err != nil {
		return err
	}
	return nil
}

// onlyOnceOption is used to check that the value of some parameters can only appear once.
func onlyOnceOption(c *cli.Context, name string) (*string, *reporter.KpmEvent) {
	inputOpt := c.StringSlice(name)
	if len(inputOpt) > 1 {
		return nil, reporter.NewErrorEvent(reporter.InvalidCmd, fmt.Errorf("the argument '%s' cannot be used multiple times", name))
	} else if len(inputOpt) == 1 {
		return &inputOpt[0], nil
	} else {
		return nil, nil
	}
}

// parseAddOptions will parse the user cli inputs.
func parseAddOptions(c *cli.Context, localPath string) (*opt.AddOptions, error) {
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
			RegistryOpts: *gitOpts,
		}, nil
	} else {
		// parse from 'kpm add xxx:0.0.1'.
		ociReg, err := parseOciRegistryOptions(c)
		if err != nil {
			return nil, err
		}
		return &opt.AddOptions{
			LocalPath:    localPath,
			RegistryOpts: *ociReg,
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

	if gitUrl == nil {
		return nil, reporter.NewErrorEvent(reporter.InvalidGitUrl, fmt.Errorf("the argument 'git' is required"))
	}

	if gitTag == nil {
		return nil, reporter.NewErrorEvent(reporter.WithoutGitTag, fmt.Errorf("the argument 'tag' is required"))
	}

	return &opt.RegistryOptions{
		Git: &opt.GitOptions{
			Url: *gitUrl,
			Tag: *gitTag,
		},
	}, nil
}

// parseOciRegistryOptions will parse the oci registry information from user cli inputs.
func parseOciRegistryOptions(c *cli.Context) (*opt.RegistryOptions, error) {
	ociPkgRef := c.Args().First()
	name, version, err := parseOciPkgNameAndVersion(ociPkgRef)
	if err != nil {
		return nil, err
	}

	settings := settings.GetSettings()
	if settings.ErrorEvent != nil {
		return nil, settings.ErrorEvent
	}

	return &opt.RegistryOptions{
		Oci: &opt.OciOptions{
			Reg:     settings.DefaultOciRegistry(),
			Repo:    settings.DefaultOciRepo(),
			PkgName: name,
			Tag:     version,
		},
	}, nil
}

// parseOciPkgNameAndVersion will parse package name and version
// from string "<pkg_name>:<pkg_version>".
func parseOciPkgNameAndVersion(s string) (string, string, error) {
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		return parts[0], "", nil
	}

	if len(parts) > 2 {
		return "", "", reporter.NewErrorEvent(reporter.InvalidPkgRef, fmt.Errorf("invalid oci package reference '%s'", s))
	}

	if parts[1] == "" {
		return "", "", reporter.NewErrorEvent(reporter.InvalidPkgRef, fmt.Errorf("invalid oci package reference '%s'", s))
	}

	return parts[0], parts[1], nil
}
