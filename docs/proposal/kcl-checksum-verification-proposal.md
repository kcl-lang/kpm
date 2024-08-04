# Technical Proposal for the checksum verification of the dependencies in kpm

## Objective
The main objective of this proposal is to redesign and complete the checksum verification of dependencies in KPM (KCL Package Manager) to ensure integrity, correctness, and reliability.

## Main purpose of checking checksum
A checksum is a value that allows the contents of a file to be authenticated. Even small changes to the file content will change its checksum. It is an error detection algorithm where a cryptographic hash (`SHA-1`, `SHA-256`, `SHA-512` etc) is generated based on the dependencies (their versions, content etc.) on the sender's side. When a package is retrieved on the receiver's side, the same algorithm generates the hash, and both values are compared to check for errors during installation.
The main purposes of a checksum are:

1.	**To verify that the source has not been modified**: Ensuring that the correct file is used.
2.	**To validate the integrity of the downloaded package**.
3.	**To confirm that the file hasn't changed since it was first downloaded**: Reporting a security error if the file does not have the correct checksum.
4.	**To ensure that the package didn't get corrupted in transit at install time**.

## Related to the Checksum Check Function
1. **Downloading the Package/Dependency**: The package or dependency is downloaded from the registry.
2. **Calculating the Cryptographic Hash**: A cryptographic hash (e.g., SHA-1, SHA-256, SHA-512) is calculated from the downloaded package in streams.
3. **Comparing the Hash**: The calculated hash is then compared with the actual hash stored in the registry or the lockfile.
4. **Validating the Package**: If the hashes match, the downloaded package is validated, ensuring its integrity and correctness.
5. **Error Handling**: If the hashes do not match, proper error handling is involved to notify the user and prevent the installation of the corrupted or tampered package.

## Checksum verification in some common package managers
1. **Cargo**:- The Rust package manager uses an `SHA-256` algorithm to generate cryptographic hash codes encoded in hexadecimal string (checksum), stored in the `Cargo.lock` file, to verify the integrity of dependencies during installation.

- **Demonstrating how Cargo maintains integrity through checksums**:
```
cargo new hello_world
cd hello_world
```
Add this to your Cargo.toml -
~~~
[dependencies]
rand = "0.8.4"
~~~
```
cargo build
<change the  checksum values in cargo.lock for the given dependency>
cargo build    
```
After the above steps you will see the following error -
~~~
error: checksum for 'rand v0.8.5 changed between lock files this could be indicative of a few possible errors:
* the lock file is corrupt
* a replacement source in use (e.g., a mirror) returned a different checksum
* the source itself may be corrupt in one way or another
unable to verify that 'rand v0.8.5 is the same as when the lockfile was generated
~~~
- Implementation of Checksum in Cargo
    1. Checksum Generation -
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/crates/cargo-test-support/src/registry.rs#L1711-L1714
    
    2. Retrieve the package metadata from the crates.io registry or from the Cargo.lock file if it exists. Get the checksum:
    - If Cargo.lock is present, use the checksum stored in it.
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/src/cargo/core/resolver/encode.rs#L522-L529
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/src/cargo/ops/lockfile.rs#L220-L247
    
    - If Cargo.lock is not present, fetch the checksum from the crate's metadata on crates.io.
    
    3. Use the URL provided in the crate's metadata to download the tarball file (.crate). Compute the checksum using a hashing algorithm (e.g., SHA256) while streaming the data from the tarball.
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/src/cargo/sources/registry/mod.rs#L717-L740
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/src/cargo/sources/registry/mod.rs#L892-L922
    
    4. Compare the calculated checksum with the checksum retrieved earlier (from Cargo.lock or crates.io metadata).
    
    https://github.com/rust-lang/cargo/blob/fa646583675d7c140482bd906145c71b7fb4fc2b/src/cargo/sources/registry/download.rs#L116-L154

2. **NPM** :- The Node Package Manager uses `SHA-1`(deprecated) and `SHA-512`(Default) algorithms to generate hash code checksum values, which are encoded in `BASE64` format and then stored in `package-lock.json` file, to verify the integrity of dependencies during installation.

https://github.com/npm/cli/blob/04eb43f2b2a387987b61a7318908cf18f03d97e0/lib/utils/tar.js#L78-L80

- **Demonstrating how npm maintains integrity through checksums**:
```
mkdir npm-checksum
cd npm-checksum
npm init
npm install express
<change the integrity sha-512 checksum values in package-lock.json>
npm install
```
After the above steps you will see the following error -
~~~
npm warn tarball tarball data for accepts@https://registry-npmjs.org/accepts/-/accepts-1.3.8.tgz (sha 512-PYAtNishanthTa2m2VKxuvSD3DPC/Gy+U+s0A1LAuT8mkmRuvw+NACSaeXEQ÷NHcVF7rONL6qcaxV3Uuemwawk+7+SJLw==)
seems to be corrupted. Trying again.
npm warn tarball tarball data for accepts@https://registry.pmjs.org/accepts/-/accepts-1.3.8.tgz (sha 512-PYAtNishanthTa2m2VKxuvSD3DPC/Gy+U+0A1LAuT8mkmRuvw+NACSaeXEQ+NHcVF7rON16qcaxV3Uuemwawk+7+SJLw==)
seems to be corrupted. Trying again.
npm error code EINTEGRITY
npm error sha512-PYAtNishanthTa2m2VKxuvSD3DPC/Gy+U+s0A1LAuT8mkmRuvw+NACSaeXEQ+NHcVF70N16qCaxV3Uuemwa
wk+7+SJLw= integrity checksum failed when using sha512: wanted sha512-PYAtNishanthTa2m2VKxuvSD3DPC/G y+U+s0A1LAuT8mkmRuvw+NACSaeXEQ+NHcVF7rON16qcaxV3Uuemwawk+7+SJLw= but got sha512-PYAthTa2m2VKxuvSD3DP
C/Gy+U+s0A1LAuT8mkmRuvw+NACSaeXEQ+NHcVF7rON16qcaxV3Uuemwawk+7+SJLw=. (5403 bytes)
~~~
- **Implementation of Checksum in npm**
    
    1. Retrieve tarball (.tgz) file and checksum (integrity) from npm registry's metadata or `package-lock.json/yarn-lock.json` (if present)
    
    https://registry.npmjs.com/package-name
    - If package-lock.json is present, use the checksum stored in it.
    - If package-lock.json is not present, fetch the checksum from the npm registry's metadata.
    
    https://github.com/npm/cli/blob/4e81a6a4106e4e125b0eefda042b75cfae0a5f23/workspaces/arborist/lib/yarn-lock.js#L344-L375
    
    2. Download the tarball from the provided URL and calculate its checksum based on the streamed data inside the tarball
    
    https://github.com/npm/cli/blob/04eb43f2b2a387987b61a7318908cf18f03d97e0/lib/utils/tar.js#L53-L109
    
    3. Compare the calculated checksum with the checksum from the npm registry's metadata or package-lock.json
    
    https://github.com/npm/cli/blob/4e81a6a4106e4e125b0eefda042b75cfae0a5f23/workspaces/arborist/lib/diff.js#L104-L154
    https://github.com/npm/ssri/blob/0fa39645b10dd39680c708f325b90a52b35f0d30/lib/index.js#L92-L128

3. **Go** :- The `go` command downloads a module and computes a hash. The hash consists of an algorithm name `SHA-256 (h1)`, and a base64-encoded cryptographic hash, which is stored in the `go.sum` file to verify the reliability of the modules during installation.
- **Demonstrating how go maintains reliability through checksums**:
```
mkdir go-checksum
cd go-checksum
go mod init go-checksum
go get github.com/prometheus/client_golang/prometheus 
<change the cryptographic hash(h1: ) checksum values in go.sum>
 go mod verify      
```
After the above steps you will see the following error -
~~~
verifying google.golang.org/protobuf@v1.33.0/go.mod: checksum mismatch downloaded: h1: c6P6GXX6sHbq/GpV6MGZEdwhWPcYBgnhAHhKbcUYpos=
go. sum:
h1: c6P6GXXNishant6sHbq/GpV6MGZEdwhWPcYBgnhAHhKbcUYpos=
SECURITY ERROR
This download does NOT match an earlier download recorded in go. sum.
The bits may have been replaced on the origin server, or an attacker may have intercepted the download attempt.
For more information, see 'go help module-auth'.
~~~
- **Implementation of Checksum in Go**
    1. Download the module file and compute the SHA-256 checksum.
    https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/vendor/golang.org/x/mod/sumdb/dirhash/hash.go#L26-L29
    https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/vendor/golang.org/x/mod/sumdb/dirhash/hash.go#L44-L65
    2. The go command checks the main module’s go.sum file for a corresponding hash entry for the downloaded file.
    https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modfetch/fetch.go#L718-L745
    - If the go.sum file is not present or doesn’t contain a hash for the downloaded file, the go command may use the checksum database, a global source of hashes for publicly available modules, to verify the hash.
    https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modfetch/fetch.go#L761-L785
    3. Once the hash is verified, the go command adds the verified hash entry to the go.sum file.
    https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modfetch/fetch.go#L835-L886
    4. **Special Case** - If a module is private (matched by the GOPRIVATE or GONOSUMDB environment variables) or if the checksum database is disabled (by setting GOSUMDB=off):
    - The go command accepts the computed hash without verifying it against the checksum database.
    - The module file is added to the module cache without further verification.


## To implement the checksum check, which parts need to finished
1. **Generate Cryptographic Hash Codes**: We need to write a function that will generate cryptographic hash codes for `SHA-256`(Generally faster on 32-bit systems and offers sufficient security for most applications) or `SHA-512`(Can be faster on 64-bit systems due to optimizations in hardware and software) etc.
2. **Generate Checksum While Downloading**: Then write a function that will use the cryptographic function to generate checksums while downloading the package/dependency in streams.
3. **Retrieve Actual Checksum**: Complete a function to get the actual checksum. If it's the first time downloading, get the checksum from the registry; otherwise, get it from the lockfile(`kcl.mod.lock`).
4. **Compare Checksums**: Complete a function that will compare the actual checksum with the current checksum. If the checksums match, this function will store the verified hash in the `kcl.mod.lock` file in the checksum field and load the package in our directory. If the checksums do not match, it will log a proper error message to the user to avoid downloading the malicious package.

## Combined with kpm, which functions of kpm need to add checksum

1. Generate Cryptographic Hash Codes:
- We already have a `sum (checksum)` field in Dependency which we will store in `kcl.mod.lock`:

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/package/modfile.go#L191-L201

2. Change the CheckSum Generate Function:
- Modify the CheckSum function to read data from dependencies in the form of streams and calculate a consistent checksum based on the data streams:
Existing Checksum Function:- https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/utils/utils.go#L32-L70

We can take some motivation from Go's Hash function -
https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/vendor/golang.org/x/mod/sumdb/dirhash/hash.go#L44-L65

Check for Go Implementations of generating hash while downloading the package in form of stream:- https://github.com/golang/go/blob/master/src/cmd/vendor/golang.org/x/mod/sumdb/dirhash/hash.go

Fetching Example -
https://github.com/golang/go/blob/f428c7b729d3d9b37ed4dacddcd7ff88f4213f70/src/cmd/go/internal/modfetch/fetch.go#L647-L652

3. Add custom logic in `Download` function to generate the checksum from data streams using the updated `HashDir` function:

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L1092-L1267

4. `Download` function will load the package from `kcl.mod.lock` using `LoadPkgFromPath` 
https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/client/client.go#L143-L187

5. Compare the actual checksum with the current checksum using `check` function that will compare the checksums and, if matched, store the verified hash in the `kcl.mod.lock` file and load the package:

https://github.com/kcl-lang/kpm/blob/2b1e0b91d8488da804158728063308ca271a3350/pkg/package/package.go#L238-L250

6. If the checksums match, update the mod and lock file using `UpdateModAndLockFile`; otherwise, log an error message to the user to avoid downloading a malicious package.

Additionally, we can extend it to verify this checksum every time for all existing or upcoming packages/dependencies if there is any modification in the `kcl.mod` and `kcl.mod.lock` files. Currently, KPM shows the sum only for the first time and does not give errors if the sum is changed and a download event is triggered. This enhancement would bring KPM in line with other package managers like Go, npm, Cargo, etc., which trigger reverification.

## For the existing kpm dependencies, we need a solution that does not break compatibility for starting this feature
We can have two methods to ensure backward compatibility:

1. Checksum Registry/Database:

We can take some motivation from npm, where we create a `registry` that contains metadata for all the packages and dependencies known to KPM. This is similar to the Go checksum database, which has `checksum database` for all the modules known to Go.
If we are not able to find the checksum entry in the lockfile, we can retrieve it from the registry or `checksum database`. We can also add a flag, similar to Go, allowing users to skip checking the registry or checksum database and download the package even if it might be corrupted.

2. FLAG_NO_SUM_CHECK:

We already have `FLAG_NO_SUM_CHECK`, which we will apply to all existing KPM dependencies by default. New dependencies will not have this flag.
Whenever a user downloads a new dependency, we will log a warning if `FLAG_NO_SUM_CHECK` is enabled, indicating that the checksum is not being verified.

## Improvements
- We need to document the entire procedure for checksum verification in the Developer's Guide to ensure developer's/user's can navigate and troubleshoot errors effectively.

- We also need to create a User Guide to help users understand how to rectify errors related to checksum verification that may occur while downloading a dependency through KPM. This guide will include instructions on:
    - Changing the FLAG_NO_SUM_CHECK: How users can modify the FLAG_NO_SUM_CHECK setting to bypass checksum verification for specific dependencies.
    - Avoiding Checksum Database or Registry Checks: Instructions on how to configure KPM to avoid checking the checksum database or registry, if necessary, for particular scenarios.
