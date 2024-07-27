# PreTest
**Author**:- **Vinayak Raj Ranjan**
## Objective
The primary objective of this technical proposal is to redesign and complete the checksum verification mechanism for the kpm package manager, enhancing its security and integrity. 

## Purpose of Checksum Verification
- Ensure that the downloaded package has not been tampered with.
- To prevent potential security vulnerabilities arising from corrupted or malicious packages.
- To provide users with confidence that the packages they are using are exactly what the authors intended.
- Refereance:- [https://www.bsi.bund.de/EN/Themen/Verbraucherinnen-und-Verbraucher/Informationen-und-Empfehlungen/Cyber-Sicherheitsempfehlungen/Virenschutz-Firewall/Pruefsummencheck/pruefsummencheck.html?nn=920646#doc921790bodyText3](https://www.bsi.bund.de/EN/Themen/Verbraucherinnen-und-Verbraucher/Informationen-und-Empfehlungen/Cyber-Sicherheitsempfehlungen/Virenschutz-Firewall/Pruefsummencheck/pruefsummencheck.html?nn=920646#doc921790bodyText3)
## Related Checksum Check Functions in Common Package Managers
- Cargo (Rust): Uses SHA256 hashes stored in Cargo.lock.
- NPM (JavaScript): Uses SHA1 hashes stored in package-lock.json.
- Go Modules: Uses SHA256 hashes stored in go.sum.
- Analyze how popular package management tools like Cargo, npm, and Go implement checksum verification.
- Understand their methods for generating, storing, and verifying checksums.
- Identify best practices and common challenges associated with checksum verification.
- Reference:-[https://docs.rs/cargo-lock/latest/cargo_lock/package/enum.Checksum.html](https://docs.rs/cargo-lock/latest/cargo_lock/package/enum.Checksum.html)

## Parts Required for Implementing Checksum Verification
- Functionality to calculate the checksum of a downloaded package.
- Store the checksums in a lock file (e.g., kcl.mod.lock).
- Verify the calculated checksum against the stored checksum.
- Proper error handling when checksums do not match.

## Integration with kpm
- Add Command:
    - Calculate the checksum when adding a new dependency.
    - Store the checksum in kcl.mod.lock.
- Update Command:
    - Verify checksums during the update process.
    - Update checksums for any new versions of dependencies.
- Global Configurations:
    - Introduce a FLAG_NO_SUM_CHECK to skip checksum verification if needed.
## Backward Compatibility
- Check if the checksum exists in kcl.mod.lock. If not, skip verification but log a warning.
- Gradually introduce checksum verification by updating kcl.mod.lock on the first update.

# Changes to be made in Codebase:
To enhance the `kpm` package manager with checksum verification for dependencies, the following changes will be made to the codebase:

## Add Checksum Field to Dependency Struct

   - **Change:** Introduce a new field `Checksum` in the `Dependency` struct to store the checksum value of each dependency.

   ```go
   type Dependency struct {
       Name        string
       Version     string
       Source      Source
       Local       *Local
       Checksum    string // New field for storing checksum
   }
```

## Checksum Verification in ResolvePkgDepsMetadata Method
- Implement checksum verification during dependency resolution. Ensure that the checksum of each dependency matches the expected value.
``` go
for _, name := range kclPkg.Dependencies.Deps.Keys() {
        d, ok := kclPkg.Dependencies.Deps.Get(name)
        if !ok {
            break
        }
        // Verify checksum if not in '--no_sum_check' mode
        if !c.noSumCheck {
            expectedChecksum, err := c.AcquireDepSum(d)
            if err != nil {
                return err
            }
            if d.Checksum != expectedChecksum {
                return reporter.NewErrorEvent(reporter.ChecksumMismatch, fmt.Errorf("checksum mismatch for dependency '%s'", d.Name))
            }
        }
```


## Add Checksum Calculation and Storage
```go
func calculateChecksum(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func addChecksumToLockFile(pkgName, checksum string) error {
    // Add logic to update kcl.mod.lock with the checksum
} 
```
- Update pkg/cmd/cmd_add.go
``` go
checksum, err := client.calculateChecksum(pkgFilePath)
    if err != nil {
        return err
    }

    // Add checksum to lock file
    err = client.addChecksumToLockFile(kclPkg.GetPkgName(), checksum)
    if err != nil {
        return err
    }
```

## Verify Checksum During Update
``` go
func verifyChecksum(filePath, expectedChecksum string) error {
    actualChecksum, err := calculateChecksum(filePath)
    if err != nil {
        return err
    }

    if actualChecksum != expectedChecksum {
        return fmt.Errorf("checksum verification failed for %s: expected %s, got %s", filePath, expectedChecksum, actualChecksum)
    }
```
- pkg/cmd/cmd_update.go
``` go
 for _, dep := range kclPkg.ModFile.Dependencies.Deps {
        // Verify checksum
        err := client.verifyChecksum(dep.FilePath, dep.Checksum)
        if err != nil {
            return err
        }
    }
```

## Updating Documentation
- Update the user guide to include information on checksum verification, its importance, and how to disable it using FLAG_NO_SUM_CHECK
- Update the developer guide to provide detailed instructions on how the checksum verification process works internally.

