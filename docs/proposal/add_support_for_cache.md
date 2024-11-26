# Proposal for Enhancing KPM's Dependency Management

**Author**: [Manoram Sharma](https://github.com/Manoramsharma)

## Introduction

### KPM: The Package Management Tool for KCL

KPM is a robust tool designed to simplify the management, creation, and publishing of KCL packages. As Kubernetes configurations grow in complexity, managing dependencies and versioning becomes increasingly challenging. KPM addresses these challenges by providing an efficient and standardized way to handle KCL packages, ensuring seamless integration and deployment in Kubernetes environments.

### Why KPM is Required

Managing KCL packages manually can be error-prone and time-consuming, especially when dealing with multiple dependencies and versions. KPM automates this process, making it easier for developers to:

- **Manage Dependencies**: Automatically fetch and resolve dependencies from Git repositories, local paths, or OCI registries.
- **Create Packages**: Streamline the creation of KCL packages with consistent structure and metadata.
- **Publish Packages**: Facilitate the distribution of packages to OCI registries, enabling easy sharing and reuse across projects.

## Project Understanding: Dependencies Management in KPM

### Current Implementation

KPM manages KCL package dependencies from multiple sources, ensuring flexibility and efficiency:

- **Git**: Uses the `pkg/git` module to clone and fetch dependencies from Git repositories. 
- **Local Path**: Allows dependencies to be specified from local file paths, facilitating development and testing.
- **OCI Registry**: The default registry for KPM, storing KCL packages in OCI-compliant containers. Managed through the `pkg/oci` module, it provides a centralized and standardized way to distribute packages.

### Problem Statement

The current implementation of KPM faces significant issues with dependency management:

- **Naming Conflicts**: Dependencies from different registries may share the same name, leading to conflicts in the current file naming approach used by KPM.
- **Case Sensitivity**: The local saving path of third-party libraries is case-sensitive, which can cause inconsistencies across different operating systems, particularly between Windows and Unix-based systems.

To address these problems, we propose redesigning the local storage system for third-party dependencies in KPM, drawing inspiration from Cargo's implementation for Rust. This redesign will involve using bare repositories to cache dependencies, ensuring efficient and conflict-free storage and retrieval.


### Cargo's Implementation for Dependency Management

Cargo, Rust's package manager, efficiently manages third-party dependencies by caching Git repositories locally. This mechanism minimizes redundant downloads and speeds up the build process.

#### Use Case: Rust Developer Managing Dependencies with Cargo

**Scenario**: Alex, a Rust developer, needs to add a third-party library, `my-crate`, from GitHub to their project, using a specific commit `a1b2c3d4`.

**Steps**:

1. **Add Dependency**:
   - Alex adds the dependency to the `Cargo.toml` file:
     ```toml
     [dependencies]
     my-crate = { git = "https://github.com/user/my-crate.git", rev = "a1b2c3d4" }
     ```

2. **Run Cargo Build**:
   - Alex runs the build command:
     ```sh
     cargo build
     ```

3. **Fetch Repository**:
   - Cargo checks if the repository is cached in `~/.cargo/git/db/`.
   - If not cached, Cargo clones the repository as a bare repository:
     ```plaintext
     ~/.cargo/git/db/
     ├── my-crate-<hash>/
     │   ├── objects/
     │   ├── refs/
     │   └── ... (bare repository files)
     ```

4. **Create Checkout**:
   - Cargo creates a working copy in `~/.cargo/git/checkouts/` for the specific commit:
     ```plaintext
     ~/.cargo/git/checkouts/
     ├── my-crate-<hash>/
     │   ├── src/
     │   ├── .git/
     │   └── ... (repository files at commit a1b2c3d4)
     ```

5. **Build the Dependency**:
   - Cargo uses the working copy to build the project with the specified commit.

6. **Reuse and Update**:
   - For subsequent builds, Cargo reuses the cached bare repository.
   - If the dependency is updated to a new commit, Cargo fetches updates to the bare repository and creates a new checkout:
     ```toml
     [dependencies]
     my-crate = { git = "https://github.com/user/my-crate.git", rev = "e5f6g7h8" }
     ```
     ```sh
     cargo build
     ```

By adopting a similar approach, KPM can provide KCL developers with a unified and efficient system for managing third-party dependencies, abstracting the complexities of downloading, caching, and version control.

## 3. Proposed Changes to KPM

### New Directory Structure
```bash
└── kpm
    ├── .kpm # all the configuration file for kpm client
    ├── git # all the kcl dependencies from git repo
    ├── oci # all the kcl dependencies from oci registry
```
Under the subdir in `kpm/git`, the storage local system is
```bash
kpm/git
├── checkouts.   # checkout the specific version of git repository from cache bare repository 
│   ├── kcl-2a81898195a215f1
│   │   └── 33bb450. . # All the version of kcl package from git repository will be replaced with commit id
│   ├── kcl-578669463c900b87
│   │   └── 33bb450
└── db    # A bare git repository for cache git repo
    ├── kcl-2a81898195a215f1.      # <NAME>-<HASH> <NAME> is the name of git repo, 
    ├── kcl-578669463c900b87.   # <HASH> is calculated by the git full url.
```
Under the subdir in `kpm/oci`, the storage local system is
```bash
kpm/oci
├── cache # the cache for KCL dependencies tar
│   ├── ghcr.io-2a81898195a215f1    # <HOST>-<HASH> HOST is the name of oci registry,  <HASH> is calculated by the oci full url.
│   │   └── k8s_1.29.tar    # the tar for KCL dependencies
│   ├── docker.io-578669463c900b87
│   │   └── k8s_1.28.tar
└── src                                              	
│   ├── ghcr.io-2a81898195a215f1
│   │   └── k8s_1.29    # the KCL dependencies tar will untar here
│   ├── docker.io-578669463c900b87
│   │   └── k8s_1.28
```
### 1. [PRETEST 1]: Enhance git module to support bare repository clone

**Completed** ✅ - [PRETEST]: [Added support for bare repo in clone function of git module](https://github.com/kcl-lang/kpm/pull/419)


### 2.[PRETEST 2]: Enhance git/oci downloader module to support new local storage path for cache bare repo

#### **Background**
As part of the ongoing enhancements to the `kpm` package management tool for KCL, we have redesigned the local storage system for third-party dependencies, taking inspiration from the Rust package manager (Cargo). The new design aims to provide a unified and organized structure that supports efficient dependency management, caching, and retrieval for both Git-based and OCI-based KCL packages.

#### **New Storage Structure**

The redesigned local storage structure is as follows:

```
└── kpm
    ├── .kpm    # Configuration files for kpm client
    ├── git     # All the KCL dependencies from Git repositories
    └── oci     # All the KCL dependencies from OCI registries
```

### **Refactoring Approach**

To align with this new storage design, the `GitDownloader` and `OciDownloader` functions within the `pkg/downloader/downloader.go` file need to be refactored. The goal is to ensure that third-party dependencies, whether sourced from Git repositories or OCI registries, are downloaded and stored according to the new structure. The following sections outline the refactored approach for both downloaders.

---

### **1. GitDownloader**

**Functionality Overview:**
The `GitDownloader` is responsible for downloading KCL packages from specified Git repositories. Under the new design, these packages will be stored as bare repositories in the `.kpm/git/cache` directory.

**Proposed Refactored Code:**

```go
func (d *GitDownloader) Download(opts DownloadOptions) error {
	var msg string
	if len(opts.Source.Git.Tag) != 0 {
		msg = fmt.Sprintf("with tag '%s'", opts.Source.Git.Tag)
	}

	if len(opts.Source.Git.Commit) != 0 {
		msg = fmt.Sprintf("with commit '%s'", opts.Source.Git.Commit)
	}

	if len(opts.Source.Git.Branch) != 0 {
		msg = fmt.Sprintf("with branch '%s'", opts.Source.Git.Branch)
	}

	reporter.ReportMsgTo(
		fmt.Sprintf("cloning '%s' %s", opts.Source.Git.Url, msg),
		opts.LogWriter,
	)

	gitSource := opts.Source.Git
	if gitSource == nil {
		return errors.New("git source is nil")
	}

	// Determine the cache path for the Git dependency
	cachePath := filepath.Join(kpmBasePath, "git", "cache", gitSource.Url)
	err := os.MkdirAll(cachePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Clone the repository as a bare repository into the cache path
	_, err = git.CloneWithOpts(
		git.WithCommit(gitSource.Commit),
		git.WithBranch(gitSource.Branch),
		git.WithTag(gitSource.Tag),
		git.WithRepoURL(gitSource.Url),
		git.WithLocalPath(cachePath),
		git.WithBare(true),  // Ensure this is a bare clone
	)

	if err != nil {
		return err
	}

	return nil
}
```

**Key Changes:**
- The `GitDownloader` now clones repositories as bare repositories into the designated cache directory under `.kpm/git/cache`.
- This approach ensures that all Git-based KCL dependencies are stored in a consistent and organized manner, facilitating easy retrieval and management.

---

### **2. OciDownloader**

**Functionality Overview:**
The `OciDownloader` handles downloading KCL packages from OCI registries. With the new storage design, these packages will be stored as tarball files in the `.kpm/oci/cache` directory.

**Proposed Refactored Code:**

```go
func (d *OciDownloader) Download(opts DownloadOptions) error {
	ociSource := opts.Source.Oci
	if ociSource == nil {
		return errors.New("oci source is nil")
	}

	// Define the cache directory for OCI dependencies
	cachePath := filepath.Join(kpmBasePath, "oci", "cache", ociSource.Repo)
	err := os.MkdirAll(cachePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)

	var cred *remoteauth.Credential
	if opts.credsClient != nil {
		cred, err = opts.credsClient.Credential(ociSource.Reg)
		if err != nil {
			return err
		}
	} else {
		cred = &remoteauth.Credential{}
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(&opts.Settings),
	)
	if err != nil {
		return err
	}

	ociCli.PullOciOptions.Platform = d.Platform

	if len(ociSource.Tag) == 0 {
		tagSelected, err := ociCli.TheLatestTag()
		if err != nil {
			return err
		}

		reporter.ReportMsgTo(
			fmt.Sprintf("the latest version '%s' will be downloaded", tagSelected),
			opts.LogWriter,
		)

		ociSource.Tag = tagSelected
	}

	reporter.ReportMsgTo(
		fmt.Sprintf(
			"downloading '%s:%s' from '%s/%s:%s'",
			ociSource.Repo, ociSource.Tag, ociSource.Reg, ociSource.Repo, ociSource.Tag,
		),
		opts.LogWriter,
	)

	// Download the OCI package into the cache directory
	err = ociCli.Pull(cachePath, ociSource.Tag)
	if err != nil {
		return err
	}

	// No need to untar since we're treating the cache as a storage of raw tarballs

	return nil
}
```

**Key Changes:**
- The `OciDownloader` stores the downloaded OCI tarball files directly in the `.kpm/oci/cache` directory.
- This method maintains the tarball format, simplifying storage and retrieval operations while ensuring alignment with the new storage design.

---

**Next Steps:**
We seek feedback and approval from the maintainers and mentors on this proposed approach before proceeding with the implementation. Any insights or suggestions for improvement would be highly appreciated.

**After the implementation is approved, we can work forward to implement it and further move to implement hash while storing.**

### 3.[PRETEST 3]: Research work for implementing a unified dependency support system in KPM

### Proposal for Unified Dependency Support in KPM

This document outlines a proposed approach for implementing a unified dependency support system in KPM, inspired by successful practices from other package managers like Cargo (Rust), Helm, Jsonnet-bundler, CUE, and Terraform. The goal is to simplify the management of third-party dependencies for KCL developers, abstracting the underlying complexities and offering a seamless user experience.

### 1. Overview of Unified Dependency Systems in Other Package Managers

#### 1.1 Cargo (Rust)
- **Dependency Management**: Uses `Cargo.toml` for specifying dependencies.
- **Automation**: Commands like `cargo build` and `cargo update` automatically handle fetching, caching, and compiling dependencies.
- **Unified System**: Centralized cache, supports both registries (crates.io) and Git repositories. Abstracts the underlying details from the user.

#### 1.2 Helm
- **Dependency Management**: Dependencies declared in `Chart.yaml`.
- **Automation**: `helm dependency update` fetches and updates dependencies automatically.
- **Unified System**: Abstracts fetching and managing chart dependencies, storing them in the `charts` directory.

#### 1.3 Jsonnet-bundler (jb)
- **Dependency Management**: Manages dependencies stored in a `vendor` directory.
- **Automation**: `jb install` fetches and installs dependencies.
- **Unified System**: Abstracts fetching and managing dependencies, providing a smooth user experience.

#### 1.4 CUE
- **Dependency Management**: Manages dependencies similar to Go modules, stored in `$CUEHOME/pkg`.
- **Automation**: Automatically fetches and caches dependencies.
- **Unified System**: Abstracts the processes, providing a seamless user experience.

#### 1.5 Terraform (HCL)
- **Dependency Management**: Manages providers.
- **Automation**: `terraform init` fetches and caches providers.
- **Unified System**: Abstracts fetching and managing providers, offering a unified dependency management experience.

### Summary of Unified Dependency Systems in Various Configuration Languages

| Tool      | Unified System | Description                                                                                           |
|-----------|----------------|-------------------------------------------------------------------------------------------------------|
| **Cargo** | Yes            | Abstracts fetching, caching, and managing dependencies, providing a seamless user experience.         |
| **Helm**  | Yes            | Abstracts fetching and managing chart dependencies, providing a seamless user experience.             |
| Kustomize | No             | Users handle external resources manually; no built-in automation for fetching remote bases.           |
| Jsonnet   | Yes            | Jsonnet-bundler (`jb`) abstracts fetching and managing dependencies, providing a smooth experience.    |
| CUE       | Yes            | CUE’s module system automatically fetches and manages dependencies, abstracting underlying processes.  |
| Terraform | Yes            | Abstracts fetching and managing providers, providing a unified system for dependencies.               |
| Kpt       | Partially      | Uses Git for managing packages, providing a smooth experience but lacks a centralized cache mechanism. |


#### Conclusion
By implementing support for bare Git repositories in kpm, we can achieve efficient and independent management of third-party dependencies for KCL-lang projects. This enhancement will provide a robust solution for both online and offline development environments, reducing reliance on external services and improving overall development workflow.
