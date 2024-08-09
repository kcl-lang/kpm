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

## Additional changes based on review

### Checksum Database Design
#### About Go's Checksum Database
The checksum database is served by sum.golang.org, and is built on a Transparent Log (or “Merkle tree”) of hashes backed by Trillian. The main advantage of a Merkle tree is that it is tamper proof and has properties that don’t allow for misbehavior to go undetected, which makes it more trustworthy than a simple database. The go command uses this tree to check “inclusion” proofs (that a specific record exists in the log) and “consistency” proofs (that the tree hasn’t been tampered with) before adding new go.sum lines to your module’s go.sum file.

**NOTE**:- For more Info Check:- https://www.youtube.com/watch?v=KqTySYYhPUE

#### Process behind checksum database in Go
* When a new module version is published, its checksums are calculated.
* These checksums are then submitted to `sum.golang.org` and added to the centralized `Merkle tree`.
* The Merkle tree is updated to reflect the new entries, ensuring the root hash changes to represent the new state.
* When we run `go mod tidy`, `go build`, or other commands that interact with Go modules, our local Go toolchain communicates with sum.golang.org to fetch and verify checksums.
https://github.com/golang/mod/blob/bc151c4e8ccc31931553c47d43e41c0efc246096/sumdb/client.go#L206-L293
* The go command requests inclusion and consistency proofs from sum.golang.org.
https://github.com/golang/mod/blob/bc151c4e8ccc31931553c47d43e41c0efc246096/gosumcheck/main.go#L185-L211
* The inclusion proof ensures that a specific module's checksum is included in the Merkle tree.
* This involves fetching a proof that shows the path from the checksum entry to the root hash of the tree, verifying its inclusion.
https://github.com/golang/mod/blob/bc151c4e8ccc31931553c47d43e41c0efc246096/sumdb/tlog/note.go#L113-L135
* The consistency proof ensures that the Merkle tree has not been tampered with between different states.
* This proof demonstrates that the root hash at a previous state is consistent with the root hash at the current state, indicating no unauthorized changes.(older tree is contained within a newer tree in the transparency log)
https://github.com/golang/mod/blob/bc151c4e8ccc31931553c47d43e41c0efc246096/sumdb/client.go#L295-L476
* If the proofs are valid, the go command updates the go.sum file with the verified checksums.
https://github.com/golang/go/blob/1f0c044d60211e435dc58844127544dd3ecb6a41/src/cmd/go/internal/modfetch/sumdb.go#L69-L80

Go's Checksum Database proposal - https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md#checksum-database

More Info on Go's Checksum Database 
- https://go.dev/blog/module-mirror-launch
- https://go.dev/ref/mod#checksum-database

**Request made to checksum Database** -

Example- GET https://sum.golang.org/lookup/github.com/gogo/protobuf@v1.3.2
Response-
~~~
2451746
github.com/gogo/protobuf v1.3.2 h1:Ov1cvc58UF3b5XjBnZv7+opcTcQFZebYjWzi34vdm4Q=
github.com/gogo/protobuf v1.3.2/go.mod h1:P1XiOD3dCwIKUDQYPy72D8LYyHL2YPYrpS2s69NZV8Q=

go.sum database tree
28852277
IDh4gnKScdn3WD9jflNCi2BmvZvdJ0Z2jeK20t8QF1E=

— sum.golang.org Az3grkvHyq7VdmI/sCI/Jl0SsMAC/Bnol0lsMJHaL4b+mPEnHxDoea72xevZL3vEKp0q3NXFJrJfiaody1x9Dtim9gI=
~~~
So GET request on https://sum.golang.org/lookup/module@version gives response as log record number for the entry about module M version V, followed by the data for the record (that is, the go.sum lines for module M version V) and a signed tree hash for a tree that contains the record.

#### Information stored by other package managers
- Cargo uses a registry (like crates.io) that includes an index with information about all available crates. The registry is a Git repository where each crate’s metadata is stored(https://github.com/rust-lang/crates.io-index).
Each commit in this repository is signed using GPG, ensuring that the metadata cannot be tampered with without invalidating the signatures.

If someone tampers with the cargo-index data (e.g. by altering crate metadata or checksums), the GPG signature of the affected commit(s) will no longer match the expected signature.
When Cargo attempts to verify the tampered commit, the GPG verification process will fail because the content hash will differ from the hash that was originally signed.
https://github.com/rust-lang/cargo/issues/4768

Cargo Index format-
https://doc.rust-lang.org/cargo/reference/registry-index.html#index-format

- Go Uses `modulePath@moduleVersion` as unique key to get the ID of particular dependency in database.
https://github.com/golang/mod/blob/bc151c4e8ccc31931553c47d43e41c0efc246096/sumdb/test.go#L78-L117

- The npm public registry is powered by a CouchDB database, of which there is a public mirror at- https://skimdb.npmjs.com/registry
To know what it store for a package use - `skimdb.npmjs.com/registry/<package-name>`

#### What Information to store in Checksum Database for kpm
I think we can store data like -
- **Kpm dependency records**:- Path, Version and Checksum for the each Dependency.
```Go
type ChecksumDBSchema struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Path        string             `bson:"path" json:"path"`
	Version     string             `bson:"version" json:"version"`
	Checksum    string             `bson:"checksum" json:"checksum"`
	PathVersion string             `bson:"path_version" json:"path_version"`
}
```
We can use `Path@Version` (PathVersion) as a unique key to identify an entry for the dependency in the checksum database.

- Other then this if we followed merkle tree strcuture to store data in database then we need additional things to store:- NodeID, ParentID, HASH, ROOT_HASH, Signature(digital signature for verifying the authenticity of the Signed tree head(STH)- need public/private key for that),Signature algorithm, Hash algorithm, TimeStamp(creation time of the STH), TREE_SIZE etc.

OR

(Instead of using merkle tree for checksum database)We can take motivation from Cargo - where we store `Kpm module records` in Git Repository and each commit to this repository is signed with GPG keys. When we fetch metadata of the module we will also fetch its signed commit, after that we can verify the GPG signature of the commit, This verification ensures that the commits were made by an authorized entity and that the commit data has not been altered.


### Work flow of kpm process and the effective position of checksum verification

#### Position of checksum verification in kpm mod update
- Step 1. `ModUpdate` sets `--no_sum_check` then run `LoadPkgFromPath`(load mod file and dependency from kcl.mod.lock). After that it runs `ResolveDepsMetadataInJsonStr`(re-downloads the non-existsent packages, and return the calculated metadata of dependent packages serialized into a json string)
- Step 2. `ResolveDepsMetadataInJsonStr` runs `ResolvePkgDepsMetadata`(re-downloads the non-existsent packages) which runs `resolvePkgDeps`.
- Step 3. Now `resolvePkgDeps` will function accordingly depending on `--no_sum_check` flag is set or not.
- Step 4. Then it triggers `AddDepToPkg` to redownload a package if does not exist, now `AddDepToPkg` will run `InitGraphAndDownloadDeps` initializes a dependency graph and calls `DownloadDeps`
- Step 5. `DownloadDeps`(**Require sum here**) will trigger `Download`(**Verify sum here**) function to download dependencies.
- Step 6. During Donwload it calculates the hash of the dependency and will match it with the actual sum. If it matches `DownloadDeps` will set the new dependencies otherwise it returns an error.
- Step 7. If the sum matches, it updates mod and lock file using `UpdateModAndLockFile`.

**Suggestion**- Although the checksum is implemented in `kcl mod update`, some modifications can be made as suggested above to make checksum verification more secure.Also I think in Step 3 we can introduce checksum for existing dependency if `--no_sum_check` flag is not set to ensure the existing packages are consistent.

**Note**- Similar procedure can be followed for `kcl mod add`(Since most of the steps coincide with kcl mod update including `InitGraphAndDownloadDeps`,`DownloadDeps`,`Download` and `UpdateModAndLockFile` )

#### Position of checksum verification in kpm mod pull
During `kpm mod pull` it would be better if there is some checksum verification to ensure consistent package is downloaded from registry-

- Step 1. `pull` will set up the `source`(from where package to be downloaded) and `localPath`(local place where package should be downloaded), then runs `Pull`(to pull the package from source)
- Step 2. `Pull` will run `downloadPkg` which runs `Download`(dispatches the download to the specific downloader by package source)
- Step 3a.(**Require sum here**) If the source is git repository `(d *GitDownloader) Download` will be triggered which will clone the repository will specified options.
- Step 3b.(**Require sum here**) If the source is OCI registry `(d *OciDownloader) Download` will be triggered which runs `Pull`(pull the oci artifacts from oci registry), `Download` will run `UnTarDir`/`ExtractTarball` to get the contents of the package from the `localPath`.
- Step 4.(**Verify sum here**)The `downloadPkg` will then runs `LoadPkgFromPath`(load mod file and dependency from kcl.mod.lock) for the specified `localPath`.

**Suggestion**- I think we need to compute the checksum in step 3a and 3b when we are cloning a git repo, or extracting a tarball. Since this is first download we need to verify its checksum against the package entry in checksum database to ensure consistent package is downloaded.

#### Suggestions of kcl mod push
Whenever user pushes the package to oci registry we should create some mechanism such that required `Kpm module records` are also be pushed to checksum database for checksum verification during the pull.

- Step 1. `ModPush` runs `pushCurrentPackage`/`pushTarPackage` depending on whether tar package is specified or not.
- Step 2. `pushTarPackage` will load the kcl package from the tar path using `LoadKclPkgFromTar`.
- Step 3. Now `LoadKclPkg` runs to load modfile and to get dependencies from kcl.mod.lock, after this `pushPackage` runs to push the kcl package to the oci registry.
- Step 4. `pushPackage` will generate OCI options from oci url and the version of current kcl package from `ParseOciOptionFromOciUrl` and then runs `GenOciManifestFromPkg`(generate the oci manifest from the kcl package).
- Step 5. `pushPackage` runs `PushToOci`(push a kcl package to oci registry) which in turn runs `PushWithOciManifest`(push the oci artifacts to oci registry from local path).

**Suggestions** - After the Step 5 we should compute the checksum of the pushed package and add the corresponding entry in checksum database.
