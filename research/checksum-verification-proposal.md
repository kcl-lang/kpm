# KPM Checksum Verification

**Author**: Mintu Gogoi

## Introduction

This document proposes a comprehensive approach to implement checksum verification for dependencies in the KPM (KCL Package Manager). The proposal draws inspiration from common package management tools such as Cargo, npm, and Go modules.

## Purpose of Checksum Verification

The main purposes of implementing checksum verification are:

- **Integrity Verification**: Ensure that downloaded packages haven't been tampered with or corrupted during transmission.
- **Consistency**: Verify that the downloaded files are exactly the same as those originally published.
- **Authenticity**: Protect against malicious alterations and potential supply chain attacks.
- **Reproducibility**: Ensure consistent builds across different environments by verifying exact package contents.

## Checksum Verification in Common Package Managers

1. **Cargo (Rust)**: Uses SHA256 checksums stored in `Cargo.lock`.
2. **npm (JavaScript)**: Uses SHA512 checksums stored in `package-lock.json`.
3. **Go Modules**: Uses SHA256 checksums stored in `go.sum`.

These package managers typically verify checksums during package installation or update processes.

## Implementation Requirements

To implement checksum verification in KPM, we need to:

1. **Checksum Generation**: Implement functionality to calculate checksums for all files in a package during publication or addition to a project.
2. **Checksum Storage**: Extend the `kcl.mod.lock` file format to include a `checksums` section for storing generated checksums.
3. **Checksum Verification**: Implement logic to verify checksums during package installation, update, and compilation processes.
4. **Error Handling**: Implement clear and informative error messages for checksum mismatches.

1. Checksum Generation

We'll modify the existing HashDir function in pkg/utils/utils.go to use SHA-256:

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/utils/utils.go#L32


2. Checksum Storage:

- Extend the kcl.mod.lock file format to include a checksums section.
- Store the generated checksums for each dependency in this section.
- Example `kcl.mod.lock`structure:

```bash
[[dependencies]]
name = "example-dep"
version = "1.0.0"
checksum = "a1b2c3d4e5f6..."

[checksums]
"example-dep@1.0.0" = "a1b2c3d4e5f6..."
```
Implement functions to read and write checksums to the kcl.mod.lock file

```go
func writeChecksumToLockFile(dep Dependency, checksum string) error {
    // Implementation to write checksum to kcl.mod.lock
    // This would involve parsing the existing TOML file,
    // updating the checksums section, and writing it back
}

func readChecksumFromLockFile(dep Dependency) (string, error) {
    // Implementation to read checksum from kcl.mod.lock
    // This would involve parsing the TOML file and
    // retrieving the checksum for the specific dependency
}
```

3. Checksum Verification:


- Implement logic to verify checksums during package installation, update, and compilation processes.
- This should be integrated into existing functions like [ResolvePkgDepsMetadata](https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L305), [UpdateDeps](https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L445), and [Compile](https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L492).


Example implementation:

```go
func verifyChecksum(dep Dependency) error {
    storedChecksum, err := readChecksumFromLockFile(dep)
    if err != nil {
        return err
    }
    calculatedChecksum, err := generateChecksum(dep.LocalFullPath)
    if err != nil {
        return err
    }
    if storedChecksum != calculatedChecksum {
        return fmt.Errorf("checksum mismatch for dependency %s", dep.Name)
    }
    return nil
}
```
4. Error Handling:

- Implement clear and informative error messages for checksum mismatches.
- Create custom error types for different checksum-related issues.
- Example implementation:

```go
type ChecksumMismatchError struct {
    DependencyName string
    ExpectedChecksum string
    ActualChecksum string
}

func (e *ChecksumMismatchError) Error() string {
    return fmt.Sprintf("checksum mismatch for dependency %s: expected %s, got %s", 
        e.DependencyName, e.ExpectedChecksum, e.ActualChecksum)
}

// Usage
if storedChecksum != calculatedChecksum {
    return &ChecksumMismatchError{
        DependencyName: dep.Name,
        ExpectedChecksum: storedChecksum,
        ActualChecksum: calculatedChecksum,
    }
}
```
- Integrate these custom errors into the existing error handling system:
```go
if err != nil {
    if checksumErr, ok := err.(*ChecksumMismatchError); ok {
        reporter.ReportError(reporter.ChecksumMismatch, checksumErr)
    } else {
        reporter.ReportError(reporter.UnknownError, err)
    }
}
```

## Functions in KPM to Add Checksum

1. Generate checksum when adding a new dependency and store the generated checksun in the `kcl.mod.lock` file

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L746

```go
func (c *KpmClient) AddDepWithOpts(kclPkg *pkg.KclPkg, dep *pkg.Dependency, opts *opt.AddOptions) error {
    // Generate checksum
    checksum, err := HashDir(dep.LocalFullPath)
    if err != nil {
        return fmt.Errorf("failed to generate checksum: %w", err)
    }

    // Store checksum in Dependency struct
    dep.Checksum = checksum

    // Store checksum in kcl.mod.lock
    err = writeChecksumToLockFile(*dep, checksum)
    if err != nil {
        return fmt.Errorf("failed to write checksum to lock file: %w", err)
    }
}
```

2. Verify checksums when resolving dependencies

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L261

```go
func (c *KpmClient) ResolveDepsIntoMap(kclPkg *pkg.KclPkg, deps *pkg.Dependencies, update bool) error {

    for _, name := range deps.Deps.Keys() {
        dep, _ := deps.Deps.Get(name)
        if !c.noSumCheck {
            err := verifyChecksum(dep)
            if err != nil {
                return fmt.Errorf("checksum verification failed for %s: %w", dep.Name, err)
            }
        }
    }
}
```

3. Update checksums when updating dependencies

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L445

```go
func (c *KpmClient) UpdateDeps(kclPkg *pkg.KclPkg) error {
    for _, name := range kclPkg.Dependencies.Deps.Keys() {
        dep, _ := kclPkg.Dependencies.Deps.Get(name)
        
        // Generate new checksum
        newChecksum, err := HashDir(dep.LocalFullPath)
        if err != nil {
            return fmt.Errorf("failed to generate new checksum for %s: %w", dep.Name, err)
        }

        // Update checksum in Dependency struct
        dep.Checksum = newChecksum

        // Update checksum in kcl.mod.lock
        err = writeChecksumToLockFile(dep, newChecksum)
        if err != nil {
            return fmt.Errorf("failed to update checksum in lock file for %s: %w", dep.Name, err)
        }
    }
}
```

4. Verify checksums when vendoring dependencies

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L953

```go
func (c *KpmClient) VendorDeps(kclPkg *pkg.KclPkg) error {
    // Existing code...

    for _, name := range kclPkg.Dependencies.Deps.Keys() {
        dep, _ := kclPkg.Dependencies.Deps.Get(name)
        if !c.noSumCheck {
            err := verifyChecksum(dep)
            if err != nil {
                return fmt.Errorf("checksum verification failed for %s during vendoring: %w", dep.Name, err)
            }
        }
        // Existing vendoring logic...
    }
}
```

5. Verify checksums before compilation

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L492

```go
func (c *KpmClient) Compile(kclPkg *pkg.KclPkg, opts *opt.CompileOptions) error {
    // Verify checksums before compilation
    if !c.noSumCheck {
        for _, name := range kclPkg.Dependencies.Deps.Keys() {
            dep, _ := kclPkg.Dependencies.Deps.Get(name)
            err := verifyChecksum(dep)
            if err != nil {
                return fmt.Errorf("checksum verification failed for %s before compilation: %w", dep.Name, err)
            }
        }
    }
}
```

These changes will integrate checksum verification throughout the KPM workflow, ensuring the integrity of dependencies during addition, resolution, updating, vendoring, and compilation processes. The `FLAG_NO_SUM_CHECK` is respected in all cases, allowing for backward compatibility and gradual rollout of the checksum verification feature.

## Compatibility solution

To maintain compatibility with existing KPM dependencies while introducing checksum verification:

- Implement a gradual rollout strategy using the existing FLAG_NO_SUM_CHECK.
- Generate log warnings for packages without checksums initially.
- Make checksum verification optional during a transition period.
- After the transition period, make checksum verification mandatory for all packages.

## Checker Module

To improve modularity and testability, we'll introduce a separate Checker module for validating KCL dependencies. This module will handle name validation, version checking, and checksum verification.

### Checker Module Design

Create a new file pkg/checker/checker.go
```go
package checker

import (
    "fmt"
    "regexp"

    "github.com/Masterminds/semver/v3"
    "kcl-lang.io/kpm/pkg/utils"
    "kcl-lang.io/kpm/pkg/package"
)

type Checker struct {
    NoSumCheck bool
}

func NewChecker(noSumCheck bool) *Checker {
    return &Checker{NoSumCheck: noSumCheck}
}

func (c *Checker) ValidateDependency(dep pkg.Dependency, localPath string) error {
    if err := c.validateName(dep.Name); err != nil {
        return err
    }

    if err := c.validateVersion(dep.Version); err != nil {
        return err
    }

    if !c.NoSumCheck {
        if err := c.validateChecksum(dep, localPath); err != nil {
            return err
        }
    }

    return nil
}

func (c *Checker) validateName(name string) error {
    validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    if !validName.MatchString(name) {
        return fmt.Errorf("invalid dependency name: %s", name)
    }
    return nil
}

func (c *Checker) validateVersion(version string) error {
    _, err := semver.NewVersion(version)
    if err != nil {
        return fmt.Errorf("invalid version: %s", version)
    }
    return nil
}

func (c *Checker) validateChecksum(dep pkg.Dependency, localPath string) error {
    if dep.Sum == "" {
        return fmt.Errorf("missing checksum for dependency: %s", dep.Name)
    }

    calculatedSum, err := utils.HashDir(localPath)
    if err != nil {
        return fmt.Errorf("failed to calculate checksum: %w", err)
    }

    if dep.Sum != calculatedSum {
        return fmt.Errorf("checksum mismatch for dependency %s: expected %s, got %s", dep.Name, dep.Sum, calculatedSum)
    }

    return nil
}
```

**Integration with KpmClient**

Update the `KpmClient` struct in `pkg/client/client.go` to include the Checker:
```go
type KpmClient struct {
    // ... existing fields
    checker *checker.Checker
}
```
Modify the `NewKpmClient` function to initialize the Checker:

```go
func NewKpmClient() (*KpmClient, error) {
    return &KpmClient{
        // ... existing fields
        checker: checker.NewChecker(false),
    }, nil
}
```

Update the `SetNoSumCheck` method to also update the Checker:

```go
func (c *KpmClient) SetNoSumCheck(noSumCheck bool) {
    c.noSumCheck = noSumCheck
    c.checker = checker.NewChecker(noSumCheck)
}
```