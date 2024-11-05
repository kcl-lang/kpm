# Code-Design for Integrating Checker into the Existing KPM Workflow

As concluded in the workflow design, KCL needs to trigger checksum verification (depending on the flag passed) only when a dependency is downloaded from the OCI registry and is not in the local cache. If the dependency is already in the local cache, there is no need to perform checksum verification. This approach allows modifications of dependencies during development, similar to the behavior in Go, Helm, and npm.

This strategy also helps achieve the objective mentioned in [issue #329](https://github.com/kcl-lang/kpm/issues/329), which aims to support the operation of KPM in a no-network environment through offline mode.

Here are some code modifications that can be made to integrate the checker into the existing KPM workflow:

```diff
type KpmClient struct {
	// The writer of the log.
	logWriter io.Writer
	// The downloader of the dependencies.
	DepDownloader *downloader.DepDownloader
	// credential store
	credsClient *downloader.CredClient
	// The home path of kpm for global configuration file and kcl package storage path.
	homePath string
	// The settings of kpm loaded from the global configuration file.
	settings settings.Settings
+	// The checker to validate dependencies
+	DepChecker *checker.DepChecker
	// The flag of whether to check the checksum of the package and update kcl.mod.lock.
	noSumCheck bool
	// The flag of whether to skip the verification of TLS.
	insecureSkipTLSverify bool
}

// NewKpmClient will create a new kpm client with default settings.
func NewKpmClient() (*KpmClient, error) {
	settings := settings.GetSettings()

	if settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
		return nil, settings.ErrorEvent
	}

	homePath, err := env.GetAbsPkgPath()
	if err != nil {
		return nil, err
	}

+	depChecker := checker.NewDepChecker(
+		checker.WithCheckers(checker.NewIdentChecker(), checker.NewVersionChecker(), checker.NewSumChecker(
+			checker.WithSettings(*settings))),
+	)

	return &KpmClient{
		logWriter:     os.Stdout,
		settings:      *settings,
		homePath:      homePath,
+		DepChecker:    depChecker,
		DepDownloader: &downloader.DepDownloader{},
	}, nil
}
```

## Modifications for `kcl mod add` and `kcl mod update`

```diff
// Download will download the dependency to the local path.
func (c *KpmClient) Download(dep *pkg.Dependency, homePath, localPath string) (*pkg.Dependency, error) {
	if dep.Source.Git != nil {
		err := c.DepDownloader.Download(*downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
		))
		if err != nil {
			return nil, err
		}

		dep.FullName = dep.GenDepFullName()

		if dep.GetPackage() != "" {
			localFullPath, err := utils.FindPackage(localPath, dep.GetPackage())
			if err != nil {
				return nil, err
			}
			dep.LocalFullPath = localFullPath
			dep.Name = dep.GetPackage()
		} else {
			dep.LocalFullPath = localPath
		}

		modFile, err := c.LoadModFile(dep.LocalFullPath)
		if err != nil {
			return nil, err
		}
		dep.Version = modFile.Pkg.Version
	}

	if dep.Source.Oci != nil || dep.Source.Registry != nil {
		var ociSource *downloader.Oci
		if dep.Source.Oci != nil {
			ociSource = dep.Source.Oci
		} else if dep.Source.Registry != nil {
			ociSource = dep.Source.Registry.Oci
		}
		// Select the latest tag, if the tag, the user inputed, is empty.
		if ociSource.Tag == "" || ociSource.Tag == constants.LATEST {
			latestTag, err := c.AcquireTheLatestOciVersion(*ociSource)
			if err != nil {
				return nil, err
			}
			ociSource.Tag = latestTag

			if dep.Source.Registry != nil {
				dep.Source.Registry.Tag = latestTag
				dep.Source.Registry.Version = latestTag
			}

			// Complete some information that the local three dependencies depend on.
			// The invalid path such as '$HOME/.kcl/kpm/k8s_' is placed because the version field is missing.
			dep.Version = latestTag
			dep.FullName = dep.GenDepFullName()
			dep.LocalFullPath = filepath.Join(filepath.Dir(localPath), dep.FullName)
			localPath = dep.LocalFullPath

			if utils.DirExists(dep.LocalFullPath) {
				dpkg, err := c.LoadPkgFromPath(localPath)
				if err != nil {
					// If the package is invalid, delete it and re-download it.
					err := os.RemoveAll(dep.LocalFullPath)
					if err != nil {
						return nil, err
					}
				} else {
					dep.FromKclPkg(dpkg)
					return dep, nil
				}
			}
		}

		credCli, err := c.GetCredsClient()
		if err != nil {
			return nil, err
		}
		err = c.DepDownloader.Download(*downloader.NewDownloadOptions(
			downloader.WithLocalPath(localPath),
			downloader.WithSource(dep.Source),
			downloader.WithLogWriter(c.logWriter),
			downloader.WithSettings(c.settings),
			downloader.WithCredsClient(credCli),
			downloader.WithInsecureSkipTLSverify(c.insecureSkipTLSverify),
		))
		if err != nil {
			return nil, err
		}

		// load the package from the local path.
		dpkg, err := c.LoadPkgFromPath(localPath)
		if err != nil {
			return nil, err
		}

		dep.FromKclPkg(dpkg)

-		dep.Sum, err = c.AcquireDepSum(*dep)
-		if err != nil {
-			return nil, err
-		}
-		if dep.Sum == "" {
			dep.Sum, err = utils.HashDir(localPath)
			if err != nil {
				return nil, err
			}
-		}

		if dep.LocalFullPath == "" {
			dep.LocalFullPath = localPath
		}

		if localPath != dep.LocalFullPath {
			err = os.Rename(localPath, dep.LocalFullPath)
			if err != nil {
				return nil, err
			}
		}

+		if err := c.ValidateDependency(dep); err != nil {
+			return nil, err
+		}
	}

	if dep.Source.Local != nil {
		kpkg, err := pkg.FindFirstKclPkgFrom(c.getDepStorePath(homePath, dep, false))
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(kpkg)
	}

	return dep, nil
}

+ func (c *KpmClient) ValidateDependency(dep *pkg.Dependency) error {
+	tmpDep := pkg.Dependency{
+		Name:          dep.Name,
+		FullName:      dep.FullName,
+		Version:       dep.Version,
+		Sum:           dep.Sum,
+		LocalFullPath: dep.LocalFullPath,
+		Source: downloader.Source{
+			Oci: dep.Source.Oci,
+		},
+	}
+
+	tmpKclPkg := pkg.KclPkg{
+		HomePath: dep.LocalFullPath,
+		Dependencies: pkg.Dependencies{Deps: func() *orderedmap.OrderedMap[string, pkg.Dependency] {
+			m := orderedmap.NewOrderedMap[string, pkg.Dependency]()
+			m.Set(tmpDep.Name, tmpDep)
+			return m
+		}()},
+		NoSumCheck: c.GetNoSumCheck(),
+	}
+
+	if err := c.DepChecker.Check(tmpKclPkg); err != nil {
+		return reporter.NewErrorEvent(reporter.InvalidKclPkg, err, fmt.Sprintf("%s package does not match the original kcl package", dep.FullName))
+	}
+
+	return nil
+}
```

## Modifications for `kcl mod pull`

```diff
// PullFromOci will pull a kcl package from oci registry and unpack it.
func (c *KpmClient) PullFromOci(localPath, source, tag string) error {
	localPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}
	if len(source) == 0 {
		return reporter.NewErrorEvent(
			reporter.UnKnownPullWhat,
			errors.FailedPull,
			"oci url or package name must be specified",
		)
	}

	if len(tag) == 0 {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s'", source),
			c.logWriter,
		)
	} else {
		reporter.ReportMsgTo(
			fmt.Sprintf("start to pull '%s' with tag '%s'", source, tag),
			c.logWriter,
		)
	}

	ociOpts, err := c.ParseOciOptionFromString(source, tag)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err, fmt.Sprintf("failed to create temp dir '%s'.", tmpDir))
	}
	// clean the temp dir.
	defer os.RemoveAll(tmpDir)

	storepath := ociOpts.SanitizePathWithSuffix(tmpDir)
	err = c.pullTarFromOci(storepath, ociOpts)
	if err != nil {
		return err
	}

	// Get the (*.tar) file path.
	tarPath := filepath.Join(storepath, constants.KCL_PKG_TAR)
	matches, err := filepath.Glob(tarPath)
	if err != nil || len(matches) != 1 {
		if err == nil {
			err = errors.InvalidPkg
		}

		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to find the kcl package tar from '%s'.", tarPath),
		)
	}

	// Untar the tar file.
	storagePath := ociOpts.SanitizePathWithSuffix(localPath)
	err = utils.UnTarDir(matches[0], storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedUntarKclPkg,
			err,
			fmt.Sprintf("failed to untar the kcl package tar from '%s' into '%s'.", matches[0], storagePath),
		)
	}
+
+	if err := c.ValidatePkgPullFromOci(ociOpts, storagePath); err != nil {
+		return reporter.NewErrorEvent(
+			reporter.InvalidKclPkg,
+			err,
+			fmt.Sprintf("failed to validate kclPkg at %s", storagePath),
+		)
+	}

	reporter.ReportMsgTo(
		fmt.Sprintf("pulled '%s' in '%s' successfully", source, storagePath),
		c.logWriter,
	)
	return nil
}
+
+func (c *KpmClient) ValidatePkgPullFromOci(ociOpts *opt.OciOptions, storagePath string) error {
+	kclPkg, err := c.LoadPkgFromPath(storagePath)
+	if err != nil {
+		return reporter.NewErrorEvent(
+			reporter.FailedGetPkg,
+			err,
+			fmt.Sprintf("failed to load kclPkg at %v", storagePath),
+		)
+	}
+
+	dep := &pkg.Dependency{
+		Name: kclPkg.ModFile.Pkg.Name,
+		Source: downloader.Source{
+			Oci: &downloader.Oci{
+				Reg:  ociOpts.Reg,
+				Repo: ociOpts.Repo,
+				Tag:  ociOpts.Tag,
+			},
+		},
+	}
+
+	dep.FromKclPkg(kclPkg)
+	dep.Sum, err = utils.HashDir(storagePath)
+	if err != nil {
+		return reporter.NewErrorEvent(
+			reporter.FailedHashPkg,
+			err,
+			fmt.Sprintf("failed to hash kclPkg - %s", dep.Name),
+		)
+	}
+	if err := c.ValidateDependency(dep); err != nil {
+		return err
+	}
+	return nil
+}
```

## Modifications for `pullTarFromOci`

To avoid checksum failures when a user pulls a specific version of a package first and then the latest version, a small modification can be made to the `pullTarFromOci` function:

```diff
// pullTarFromOci will pull a kcl package tar file from oci registry.
func (c *KpmClient) pullTarFromOci(localPath string, ociOpts *opt.OciOptions) error {
	absPullPath, err := filepath.Abs(localPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.Bug, err)
	}

	repoPath := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	cred, err := c.GetCredentials(ociOpts.Reg)
	if err != nil {
		return err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(c.GetSettings()),
		oci.WithInsecureSkipTLSverify(ociOpts.InsecureSkipTLSverify),
	)
	if err != nil {
		return err
	}

	ociCli.SetLogWriter(c.logWriter)

	var tagSelected string
	if len(ociOpts.Tag) == 0 {
		tagSelected, err = ociCli.TheLatestTag()
		if err != nil {
			return err
		}
		reporter.ReportMsgTo(
			fmt.Sprintf("the lastest version '%s' will be pulled", tagSelected),
			c.logWriter,
		)
	} else {
		tagSelected = ociOpts.Tag
	}

+	ociOpts.Tag = tagSelected

	full_repo := utils.JoinPath(ociOpts.Reg, ociOpts.Repo)
	reporter.ReportMsgTo(
		fmt.Sprintf("pulling '%s:%s' from '%s'", ociOpts.Repo, tagSelected, full_repo),
		c.logWriter,
	)

	err = ociCli.Pull(absPullPath, tagSelected)
	if err != nil {
		return err
	}

	return nil
}
```

## Conclusion
We have identified each command in KPM that requires the integration of the checksum feature as sub-tasks. The proposed changes will seamlessly integrate the checksum checker while preserving the existing behavior for cached dependencies, ensuring that the development process remains both efficient and secure.

Additionally, if the `--no_sum_check` option is passed, the checksum verification will be bypassed. This is accomplished through the assignment of `NoSumCheck: c.GetNoSumCheck()` in `tmpKclPkg`, which is subsequently checked during the checksum verification process.