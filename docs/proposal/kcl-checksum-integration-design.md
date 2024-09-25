# Integrating Checker into the Existing KPM Workflow

## Community Research

### Overview

To integrate checksum verification into KCL, I conducted community research on how different package managers handle checksum verification in their workflows. Below is an analysis of various package managers and where checksum verification is performed within their command workflows. This analysis will help determine the optimal position for integrating checksum verification in KCL's command workflow.

### NPM Workflow

- **Checksum Verification Position**: NPM verifies checksums using `checkData` and `getAction`, which compare the actual and ideal integrity values.
  - [Source Code Reference 1](https://github.com/npm/ssri/blob/0fa39645b10dd39680c708f325b90a52b35f0d30/lib/index.js#L455-L493)
  - [Source Code Reference 2](https://github.com/npm/cli/blob/4e81a6a4106e4e125b0eefda042b75cfae0a5f23/workspaces/arborist/lib/diff.js#L104-L154)

### Go Workflow

1. **`go mod download`**:

   - The command initiates `DownloadModule`, which downloads the module using `Download`. After the download, `checkMod` verifies the checksum(for each module), and `WriteGoMod` updates the `go.mod` file with the current build list.
   - [Source Code Reference](https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modfetch/fetch.go#L45-L73)

2. **`go mod verify`**:

   - The `runVerify` command calls `verifyMod`, which verifies modules in the local Go module cache, checking both the downloaded ZIP files and their extracted directories against stored hashes.
   - [Source Code Reference](https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modcmd/verify.go#L46-L143)

3. **`go get`**:
   - The command executes `runGet`, which calls `checkPackageProblems`. After downloading missing packages(not present in local cache), it verifies the checksum using `DownloadZip` before updating the `go.mod` file.
   - [Source Code Reference 1](https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modget/get.go#L277-L426)
   - [Source Code Reference 2](https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modget/get.go#L1511-L1704)

### Helm Workflow

1. **Checksum Verification**:

   - Helm verifies downloaded files using the `Verify` function, which compares generated checksums with expected values.
   - [Verification Function Reference](https://github.com/helm/helm/blob/3a3e3846ca9c929a6966583b461181e70f19bc13/pkg/provenance/sign.go#L251-L296)
   - The `DigestFile` function generates checksums for downloaded files.
   - [Checksum Generation](https://github.com/helm/helm/blob/main/pkg/provenance/sign.go#L403-L416)

2. **`Helm install`**:

   - The `runInstall` command uses `LocateChart` to check if chart is already in local cache or download the chart using `DownloadTo` if not found, followed by verification via `VerifyChart` depending on the flags passed.
   - [Source Code Reference](https://github.com/helm/helm/blob/3a3e3846ca9c929a6966583b461181e70f19bc13/pkg/action/install.go#L730-L831)

3. **`Helm pull`**:

   - The command uses `Run` and `DownloadTo` which will download the chart to specific location and then verify the pulled chart using `VerifyChart` based on the flag passed
   - [Source Code Reference](https://github.com/helm/helm/blob/main/pkg/action/pull.go#L79-L172)

   - **Note**: If the checksum fails, Helm does not automatically delete the chart.

   - **Reasons for Retaining Failed Charts**:
     - **Performance**: Avoids repeated downloads, which improves performance when troubleshooting.
     - **Diagnostics**: Retaining failed charts helps in diagnosing verification errors.

   - **Potential Improvements**:
     - **Warnings**: Inform users about retaining failed charts and offer cleanup instructions.
     - **Optional Cleanup**: Introduce a flag like --verify-clean to automate the deletion of failed charts.

4. **`Helm verify`**:

   - The command uses `Run` which verifies the chart using `VerifyChart` based on its location.
   - [Source Code Reference](https://github.com/helm/helm/blob/3a3e3846ca9c929a6966583b461181e70f19bc13/pkg/action/verify.go#L39-L59)

5. **`Helm dependency build`**:

   - The command uses `downloadAll` which Downloads and verifies new charts using `DownloadTo`. Existing charts are only version-checked.
   - [Source Code Reference](https://github.com/helm/helm/blob/main/pkg/downloader/manager.go#L238-L374)

6. **`Helm dependency update`**:
   - Runs `Update`, which triggers `DownloadTo` for checksum verification.
   - [Source Code Reference](https://github.com/helm/helm/blob/main/pkg/downloader/manager.go#L152-L219)

### Conclusion

The optimal point for integrating checksum verification in kpm’s command workflow is after the package download and before updating any configuration or lock files. Integrating checksum verification in kpm should follow this pattern, ensuring security without compromising performance.

## Overall Approach

Currently, the `kpm` commands that may require checksum verification are `kcl mod add`, `kcl mod pull`, and `kcl mod update`.

### Position of checksum verification in `kcl mod add`

- When `kcl mod add` is executed, it runs `KpmAdd`, which runs `LoadPkgFromPath` (to load the mod file and dependencies from `kcl.mod.lock`). After parsing and validating the user CLI inputs, `AddDepWithOpts` is executed, which adds a dependency to the current KCL package.
- The `AddDepWithOpts` sets the `--no_sum_check` flag and performs the following tasks:

  1. Gets the name and version of the repository/package from the input arguments using `ParseOpt`.
  2. Downloads the dependency to the local path using `AddDepToPkg` (downloads the dependency to local cache if not already present).
  3. Updates `kcl.mod` and `kcl.mod.lock` using `UpdateModAndLockFile`.

**SumCheck Integration**: Since we have already explored other package managers, the most optimal position for checksum verification is after downloading the dependency and before updating the lockfile. We can do checksum verification between steps 2 and 3. Here's how the code might look:

```go
if err := c.DepChecker.Check(*kclPkg); err != nil {
    return nil, reporter.NewErrorEvent(reporter.InvalidKclPkg, err, fmt.Sprintf("%s package does not match the original kcl package", kclPkg.GetPkgFullName()))
}
```

In this way, since the `kclPkg` is populated with already present and new dependencies, we can verify the checksum of all the dependencies (both new and existing).

We can also modify `dependencyExistsLocal` (which checks whether the dependency exists in the local filesystem) by calculating checksums for checksum verification of already added dependencies. Here's the modified code:

```Go
func (c *KpmClient) dependencyExistsLocal(searchPath string, dep *pkg.Dependency) (*pkg.Dependency, error) {
	deppath := c.getDepStorePath(searchPath, dep, false)
	if utils.DirExists(deppath) {
		depPkg, err := c.LoadPkgFromPath(deppath)
		if err != nil {
			return nil, err
		}
		dep.FromKclPkg(depPkg)
      // Modified Part
		dep.Sum, err = utils.HashDir(deppath)
		if err != nil {
			return nil, err
		}
		return dep, nil
	}
	return nil, nil
}
```

### Position of checksum verification in `kcl mod update`

- It runs `KpmUpdate`, which first sets the `--no_sum_check` flag and then runs `LoadPkgFromPath` to load the mod file and dependencies from `kcl.mod.lock`.
- It then runs `InitGraphAndDownloadDeps`, which initializes a dependency graph and calls `DownloadDeps`. `DownloadDeps` triggers the `Download` function to download dependencies (if not present locally).
- The function then updates the build list based on the modules to upgrade or downgrade using `UpdateBuildList`.
- Once the build list is updated, the function sorts the dependency graph topologically using `TopologicalSort`, and dependencies are inserted into the package using `InsertModuleToDeps`.
- Finally, the function updates the package’s dependency list using `UpdateDeps` (ensuring that both the modfile and lockfile are updated).

**SumCheck Integration**: Similar to `kcl mod add`, we can perform checksum verification after updating the dependencies and just before updating the `kcl.mod.lock` files. Here's how the updated code of `UpdateDeps` might look:

```Go
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
	_, err := c.ResolveDepsMetadataInJsonStr(kclPkg, true)
	if err != nil {
		return err
	}
   // Modified part
	if err := c.DepChecker.Check(*kclPkg); err != nil {
		return err
	}

	// update kcl.mod
	err = kclPkg.ModFile.StoreModFile()
	if err != nil {
		return err
	}

	// Generate file kcl.mod.lock.
	if !kclPkg.NoSumCheck {
		err := kclPkg.LockDepsVersion()
		if err != nil {
			return err
		}
	}
	return nil
}
```

In this way, since the `kclPkg` is updated, we can verify the checksum of all dependencies (both new and existing).

### Position of checksum verification in `kpm mod pull`
During `kpm mod pull`, it is better to have checksum verification to ensure the consistency of the package downloaded from the registry:
- `KpmPull` runs `PullFromOci`, which pulls a KCL package from an OCI registry and unpacks it.
- It fetches the tag of the dependency to be pulled and parses the `ociOpts` using `ParseOciOptionFromString`.
- It then runs `pullTarFromOci`, which pulls a KCL package tar file from the OCI registry.
- After pulling, `PullFromOci` untars the tar file, and finally, a success message is displayed indicating that the specified package has been pulled successfully.

**SumCheck Integration**: Once the tar file is successfully untarred, we can perform checksum verification before displaying the success message. Here’s how the updated `PullFromOci` might look:

```Go
// Validate the pulled kclPkg
	if err := c.ValidatePkgPullFromOci(ociOpts, storagePath); err != nil {
		return reporter.NewErrorEvent(
			reporter.InvalidKclPkg,
			err,
			fmt.Sprintf("failed to validate kclPkg at %s", storagePath),
		)
	}
```

For the implementation of `ValidatePkgPullFromOci`, we have two options:
1. We can load the pulled `kclPkg` using `LoadPkgFromPath`, convert it into a dependency using `FromKclPkg`, and calculate the checksum using `dep.Sum, err = utils.HashDir(storagePath)`. Then, set this dependency into `kclPkg` itself and verify both the pulled kclPkg and its dependencies together. 

**Note**: This bundling dependencies within the package and verifying them together can be more comprehensive but may introduce complexities in package management and updates.

Here’s the code:
```Go
func (c *KpmClient) ValidatePkgPullFromOci(ociOpts *opt.OciOptions, storagePath string) error {
	kclPkg, err := c.LoadPkgFromPath(storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedGetPkg,
			err,
			fmt.Sprintf("failed to load kclPkg at %v", storagePath),
		)
	}

	dep := &pkg.Dependency{
		Name: kclPkg.ModFile.Pkg.Name,
		Source: downloader.Source{
			Oci: &downloader.Oci{
				Reg:  ociOpts.Reg,
				Repo: ociOpts.Repo,
				Tag:  kclPkg.ModFile.Pkg.Version,
			},
		},
	}

	dep.FromKclPkg(kclPkg)
	dep.Sum, err = utils.HashDir(storagePath)
	if err != nil {
		return reporter.NewErrorEvent(
			reporter.FailedHashPkg,
			err,
			fmt.Sprintf("failed to hash kclPkg - %s", dep.Name),
		)
	}
	kclPkg.Dependencies.Deps.Set(dep.Name, *dep)

	err = c.DepChecker.Check(*kclPkg)
	if err != nil {
		return reporter.NewErrorEvent(reporter.InvalidKclPkg, err, fmt.Sprintf("%s package does not match the original kcl package", kclPkg.GetPkgFullName()))
	}
	return nil
}
```

2. We can follow an approach similar to `Helm`, where verification checks the signature and integrity of the pulled chart without recursively verifying its dependencies. This means we can set the pulled `kclPkg` as a dependency, calculate its checksum, and then simply verify it without recursively checking its dependencies. This approach helps in verifying the checksum of the pulled package while avoiding complexities in package management.
