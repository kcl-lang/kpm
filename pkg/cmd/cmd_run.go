// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kcl-go/pkg/kcl"
	"kcl-lang.io/kpm/pkg/api"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/git"
	"kcl-lang.io/kpm/pkg/opt"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/runner"
)

// NewRunCmd new a Command for `kpm run`.
func NewRunCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "run",
		Usage:  "compile kcl package.",
		Flags: []cli.Flag{
			// The entry kcl file.
			&cli.StringSliceFlag{
				Name:  FLAG_INPUT,
				Usage: "a kcl file as the compile entry file",
			},
			&cli.StringFlag{
				Name:  FLAG_TAG,
				Usage: "the tag for oci artifact",
			},
			// '--vendor' will trigger the vendor mode
			// In the vendor mode, the package search path is the subdirectory 'vendor' in current package.
			// In the non-vendor mode, the package search path is the $KCL_PKG_PATH.
			&cli.BoolFlag{
				Name:  FLAG_VENDOR,
				Usage: "run in vendor mode",
			},
			// --no_sum_check
			&cli.BoolFlag{
				Name:  FLAG_NO_SUM_CHECK,
				Usage: "do not check the checksum of the package and update kcl.mod.lock",
			},

			// KCL arg: --setting, -Y
			&cli.StringSliceFlag{
				Name:    FLAG_SETTING,
				Aliases: []string{"Y"},
				Usage:   "specify the input setting file",
			},

			// KCL arg: --argument, -D
			&cli.StringSliceFlag{
				Name:    FLAG_ARGUMENT,
				Aliases: []string{"D"},
				Usage:   "specify the top-level argument",
			},

			// KCL arg: --overrides, -O
			&cli.StringSliceFlag{
				Name:    FLAG_OVERRIDES,
				Aliases: []string{"O"},
				Usage:   "specify the configuration override path and value",
			},

			// KCL arg: --disable_none, -n
			&cli.BoolFlag{
				Name:    FLAG_DISABLE_NONE,
				Aliases: []string{"n"},
				Usage:   "disable dumping None values",
			},

			// KCL arg: --sort_keys -k
			&cli.BoolFlag{
				Name:    FLAG_SORT_KEYS,
				Aliases: []string{"k"},
				Usage:   "sort result keys",
			},

			// KCL arg: --package
			&cli.StringFlag{
				Name:    FLAG_PACKAGE,
				Usage:   "specify the package name",
			},
		},
		Action: func(c *cli.Context) error {
			return KpmRun(c, kpmcli)
		},
	}
}

func KpmRun(c *cli.Context, kpmcli *client.KpmClient) error {
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

	kclOpts := CompileOptionFromCli(c)
	kclOpts.SetNoSumCheck(c.Bool(FLAG_NO_SUM_CHECK))
	runEntry, errEvent := runner.FindRunEntryFrom(c.Args().Slice())
	if errEvent != nil {
		return errEvent
	}

	// 'kpm run' compile the current package under '$pwd'.
	if runEntry.IsEmpty() {
		pwd, err := os.Getwd()
		kclOpts.SetPkgPath(pwd)

		if err != nil {
			return reporter.NewErrorEvent(
				reporter.Bug, err, "internal bugs, please contact us to fix it.",
			)
		}
		compileResult, err := kpmcli.CompileWithOpts(kclOpts)
		if err != nil {
			return err
		}
		fmt.Println(compileResult.GetRawYamlResult())
	} else {
		var compileResult *kcl.KCLResultList
		var err error
		// 'kpm run' compile the package from the local file system.
		if runEntry.IsLocalFile() || runEntry.IsLocalFileWithKclMod() {
			kclOpts.SetPkgPath(runEntry.PackageSource())
			kclOpts.ExtendEntries(runEntry.EntryFiles())
			if runEntry.IsLocalFile() {
				// If there is only kcl file without kcl package,
				compileResult, err = api.RunWithOpt(kclOpts)
			} else {
				// Else compile the kcl pacakge.
				compileResult, err = kpmcli.CompileWithOpts(kclOpts)
			}
		} else if runEntry.IsTar() {
			// 'kpm run' compile the package from the kcl package tar.
			compileResult, err = kpmcli.CompileTarPkg(runEntry.PackageSource(), kclOpts)
		} else if runEntry.IsGit() {
			gitOpts := git.NewCloneOptions(runEntry.PackageSource(), "", c.String(FLAG_TAG), "", "", nil)
			// 'kpm run' compile the package from the git url
			compileResult, err = kpmcli.CompileGitPkg(gitOpts, kclOpts)
		} else {
			// 'kpm run' compile the package from the OCI reference or url.
			compileResult, err = kpmcli.CompileOciPkg(runEntry.PackageSource(), c.String(FLAG_TAG), kclOpts)
		}

		if err != nil {
			return err
		}
		fmt.Println(compileResult.GetRawYamlResult())
	}
	return nil
}

// CompileOptionFromCli will parse the kcl options from the cli context.
func CompileOptionFromCli(c *cli.Context) *opt.CompileOptions {
	opts := opt.DefaultCompileOptions()

	// --input
	opts.ExtendEntries(c.StringSlice(FLAG_INPUT))

	// --vendor
	opts.SetVendor(c.Bool(FLAG_VENDOR))

	// --package
	opts.SetPackage(c.String(FLAG_PACKAGE))

	// --setting, -Y
	settingsOpt := c.StringSlice(FLAG_SETTING)
	if len(settingsOpt) != 0 {
		for _, sPath := range settingsOpt {
			opts.Merge(kcl.WithSettings(sPath))
		}
		opts.SetHasSettingsYaml(true)
	}

	// --argument, -D
	opts.Merge(kcl.WithOptions(c.StringSlice(FLAG_ARGUMENT)...))

	// --overrides, -O
	opts.Merge(kcl.WithOverrides(c.StringSlice(FLAG_OVERRIDES)...))

	// --disable_none, -n
	opts.Merge(kcl.WithDisableNone(c.Bool(FLAG_DISABLE_NONE)))

	// --sort_keys, -k
	opts.Merge(kcl.WithSortKeys(c.Bool(FLAG_SORT_KEYS)))

	return opts
}
