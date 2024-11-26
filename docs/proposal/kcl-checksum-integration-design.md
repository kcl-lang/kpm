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

4. **`Helm verify`**:

   - The command uses `Run` which verifies the chart using `VerifyChart` based on its location.
   - [Source Code Reference](https://github.com/helm/helm/blob/3a3e3846ca9c929a6966583b461181e70f19bc13/pkg/action/verify.go#L39-L59)

5. **`Helm dependency build`**:

   - The command uses `downloadAll` which Downloads and verifies new charts using `DownloadTo`. Existing charts are only version-checked.
   - [Source Code Reference](https://github.com/helm/helm/blob/main/pkg/downloader/manager.go#L238-L374)

6. **`Helm dependency update`**:
   - Runs `Update`, which triggers `downloadAll` for checksum verification.
   - [Source Code Reference](https://github.com/helm/helm/blob/main/pkg/downloader/manager.go#L152-L219)

### Conclusion

The optimal point for integrating checksum verification in kpm’s command workflow is after the package download and before updating any configuration or lock files. Integrating checksum verification in kpm should follow this pattern, ensuring security without compromising performance.

## Overall Approach

Currently, the `kpm` commands that may require checksum verification are `kcl mod add`, `kcl mod pull` and `kcl mod update`.

### Position of checksum verification in `kcl mod add`

- When `kcl mod add` is executed, it runs `KpmAdd`, which runs `LoadPkgFromPath` (to load the mod file and dependencies from `kcl.mod.lock`). After parsing and validating the user CLI inputs, `AddDepWithOpts` is executed, which adds a dependency to the current KCL package.
- The `AddDepWithOpts` sets the `--no_sum_check` flag and performs the following tasks:

  1. Gets the name and version of the repository/package from the input arguments using `ParseOpt`.
  2. Downloads the dependency to the local path using `AddDepToPkg` (downloads the dependency to local cache if not already present).
  3. Updates `kcl.mod` and `kcl.mod.lock` using `UpdateModAndLockFile`.

**SumCheck Integration**: Since we have already explored other package managers, the most optimal position for checksum verification is after downloading the dependency and before updating the lockfile. We can do checksum verification between steps 2 and 3. Here's how the code might look:

```go
if err := c.DepChecker.Check(*tmpKclPkg); err != nil {
    return nil, reporter.NewErrorEvent(reporter.InvalidKclPkg, err, fmt.Sprintf("%s package does not match the original kcl package", tmpKclPkg.GetPkgFullName()))
}
```
We will first convert the downloaded dependency into a temporary `tmpKclPkg`, perform its checksum verification based on the flag passed, and then add this dependency to the original `kclPkg`. In this way, the `kclPkg` is populated with both already present and new dependencies after verifying the checksum of the newly downloaded dependencies (which are not in the local cache).

We can also check if the dependency exists locally using `dependencyExistsLocal`. If it exists, we can bypass both the download and checksum verification procedures.

### Position of checksum verification in `kcl mod update`

- It runs `KpmUpdate`, which first sets the `--no_sum_check` flag and then runs `LoadPkgFromPath` to load the mod file and dependencies from `kcl.mod.lock`.
- It then runs `InitGraphAndDownloadDeps`, which initializes a dependency graph and calls `DownloadDeps`. `DownloadDeps` triggers the `Download` function to download dependencies (if not present locally).
- The function then updates the build list based on the modules to upgrade or downgrade using `UpdateBuildList`.
- Once the build list is updated, the function sorts the dependency graph topologically using `TopologicalSort`, and dependencies are inserted into the package using `InsertModuleToDeps`.
- Finally, the function updates the package’s dependency list using `UpdateDeps` (ensuring that both the modfile and lockfile are updated).

**SumCheck Integration**: Similar to `kcl mod add`, we can perform checksum verification after downloading the dependency (using `Download`). However, if the dependency is already present in the local cache, we can directly bypass the download and checksum procedures.

In this way, since the `kclPkg` is updated, we can verify the checksum of only the newly downloaded dependencies and not the ones that are already in the local cache.

### Position of checksum verification in `kcl mod pull`
During `kcl mod pull`, it is better to have checksum verification to ensure the consistency of the package downloaded from the registry:
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

1. We can load the pulled `kclPkg` using `LoadPkgFromPath`, convert it into a dependency using `FromKclPkg`, and calculate the checksum using `dep.Sum, err = utils.HashDir(storagePath)`. Then, set this dependency into the `kclPkg` itself and verify both the pulled `kclPkg` and its dependencies together.

**Note**: Bundling dependencies within the package and verifying them together can be more comprehensive but may introduce complexities in package management and updates.

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

2. We can follow an approach similar to `Helm`, where the verification checks the integrity of the pulled package without recursively verifying all its dependencies. Helm’s `helm pull` command fetches the chart, but it does not automatically verify or download dependencies—this is handled separately via the `helm dependency update` command. Similarly, we can download and verify the pulled `kclPkg` and its re-downloaded dependencies without recursively checking those already present in the local cache. This approach simplifies the process, as only the checksum of the newly downloaded package and dependencies is verified, avoiding unnecessary complexity in package management.

## Additional Changes Based on Review

### Re-downloads and Checksum Verification in Package Management

**Question**: When functions trigger re-downloads of third-party libraries, will checksum verification be triggered (e.g., during `go build`)?

#### Workflow of the `go build` Command

**Note**: This workflow occurs only during dependency re-download and verification.

1. The `runBuild` function is executed, which first runs `BuildInit` to set build configurations (most importantly, it sets `cfg.ModulesEnabled=true`). Then, it runs `PackagesAndErrors`, which returns the packages specified by the command line arguments.
   
2. If the pattern has a suffix of ".go", the `GoFilesPackage` function will run for each pattern, creating a package for building a collection of Go files.

3. It will then synthesize a fake "directory" so that local imports resolve consistently. After that, it triggers `ImportFromFiles` (since `cfg.ModulesEnabled` is set to true), which adds modules to the build list as needed to satisfy the imports in the named Go source files.

4. It first loads the mod file using `LoadModFile`, then runs `loadFromRoots`, which attempts to load the build graph needed to process a set of root packages and their dependencies.

5. The `loadFromRoots` function will run `preloadRootModules`, which loads the module requirements needed to identify the selected version of each module. `preloadRootModules` will then run `importFromModules`, which fetches the modules depending on imports by running `fetch` (downloads the given module).

6. The `fetch` function checks if the package is already present in the local filesystem (no verification is performed if it's present locally). If not, it triggers the `Download` function, which downloads the given module and verifies its checksum using `checkMod`.

7. In step 2, if there is no pattern with the suffix ".go", the process moves to `LoadPackages` (since `cfg.ModulesEnabled` is set to true). It will update the dependency requirement and run `checkTidyCompatibility`, which triggers `importFromModules` for each package. The same process as described in steps 5 and 6 will follow.

8. After all imports are resolved, `LoadPackages` will tidy up the `go.mod` and `go.sum` files and finally write the changes using `commitRequirements`.

**NOTE**: Similar commands are executed when `go run` is used.

### Conclusion

We can conclude that checksum verification in `go build` only occurs if the dependency is not in the local cache and is re-downloaded. If the dependency is already present in the local cache, no checksum verification occurs.

---

### In Helm

1. **`Helm install`**:
   - The `LocateChart` function checks if the chart is already in the local cache. If it is, and no `--verify` flag is passed, then there will be no checksum verification. If the chart is not found, it will download the chart using `DownloadTo`, (with checksum verification based on the verification strategy), followed by verification via `VerifyChart`, depending on the flags passed.

2. **`Helm pull`**:
   - The command uses `Run` and `DownloadTo` to download the chart to a specific location and then verifies the pulled chart using `VerifyChart` based on the flag passed. If the `--verify` flag is specified, the requested chart MUST have a provenance file and MUST pass the verification process. Failure in any part of this will result in an error, and the chart will not be saved locally.

3. **`Helm dependency build` and `Helm dependency update`**:
   - The command uses `downloadAll`, which skips the download and checksum verification if the dependency is already downloaded. It downloads and verifies new charts using `DownloadTo`, while existing charts are only version-checked.

### Conclusion

In Helm, we can conclude that checksum verification will only occur for dependencies/charts not in the local cache. If the charts are in the local cache, checksum verification will only be conducted if we pass the `--verify` flag; otherwise, it will be bypassed.

---

### In npm

1. I checked locally by downloading a dependency named `express`, then modified its code. After running `npm install`, it did not return any checksum mismatch error.

2. This behavior can also be checked in the npm CLI source code, specifically [here](https://github.com/npm/cli/blob/63d6a732c3c0e9c19fd4d147eaa5cc27c29b168d/lib/utils/verify-signatures.js#L259-L294), which shows that packages in local workspaces or those not from the registry (e.g., git packages) are skipped. The `getValidPackageInfo` function filters out local workspace packages and those that don't have an installed version, meaning only packages installed from a registry are audited for signatures and integrity.

---

### Modifications to Local Third-Party Libraries

**Question**: If local third-party libraries are modified, how does this affect checksum verification? What are the expected outcomes?

- I checked locally that if we have a dependency already present in the local cache and modify it during development, no checksum failure occurs for `go mod download`, `go mod install`, `go build`, and `go run`.
  
- For instance, `go mod download` and `go mod install` will download a dependency not already present in the local cache. If the dependency is already in the local cache (checked by `downloadCache.Do(mod, some function)` and `downloadZipCache.Do(mod, some function)`), then all checksum verification will be skipped for that module. However, if it is not already in the local cache, the module will be downloaded by `Download`, and checksum verification will be triggered by `checkMod`.

- For the `go build` and `go run`, the steps have already been stated, concluding that if the dependency is already in the local cache, checksum verification is bypassed. This also allows for modifications of third-party libraries during development.

- During the downloading of the dependency and checksum verification, only the re-downloaded module will be checked. The verification process is as follows:
  
  ```Go
  for _, vh := range goSum.m[mod] {
      if h == vh {
          return true
      }
      if strings.HasPrefix(vh, "h1:") {
          base.Fatalf("verifying %s@%s: checksum mismatch\n\tdownloaded: %v\n\t%s:     %v"+goSumMismatch, mod.Path, mod.Version, h, sumFileName, vh)
      }
  }
  ```
- In Go, checksum verification occurs immediately after downloading the dependency or during the `go mod verify`

### Final Conclusion
In KCL, we need to trigger checksum verification (depending on the flag passed) only when a dependency is downloaded from the oci registry and is not in the local cache. If the dependency is already in the local cache, there is no need to perform checksum verification. This approach allows modifications of dependencies during development, similar to the behavior in Go, Helm, and npm.

This strategy also helps achieve the objective mentioned in [issue #329](https://github.com/kcl-lang/kpm/issues/329), aiming for offline mode to support the operation of KPM in a no-network environment.

The purpose of checking the checksum is to prevent security risks caused by malicious replacement of remote third-party dependencies. Therefore, I believe there is no need for checksum verification for already downloaded/cached dependencies.
