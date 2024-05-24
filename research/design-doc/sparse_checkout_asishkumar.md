# KPM sparse checkout

**Author**: Asish Kumar

## Abstract

`kpm` manages third-party libraries through Git repositories, requiring a `kcl.mod` file at the root directory. It treats the entire Git repository as a single `kcl` package, which is inefficient for monorepos containing multiple `kcl` packages. Often, a `kcl` project depends on just one package within a monorepo, but `kpm` downloads the entire repository. Therefore, `kpm` needs to allow adding a subdirectory of a Git repository as a dependency, enabling it to download only the necessary parts and improve performance.

## User Interface

I will add a new flag called `--subdir` in `kpm add` command.  This flag will specify the path to the desired subdirectory within the Git repository. Below is the syntax for the enhanced kpm add command:

```
kpm add --subdir <subdir> <git-repo-url>
```

The `--subdir` flag will be optional. If the flag is not provided, `kpm` will download the entire repository as it does now. If the flag is provided, `kpm` will download only the specified subdirectory. The `kcl.mod` file will be generated with the path to the subdirectory.

Example usage: 

```
kpm add --subdir 1.21/* k8s
```

This command will download the `1.21` directory and all its contents from the `k8s` repository hosted in https://github.com/kcl-lang/modules


The `kcl.mod` file of the users project will also contain an array of path to the subdirectories. 

```
[dependencies]
bbb = { path = "../bbb", subdir = ["test-*", "test-*"]}
```

## Design 

The path to the directory will be passed to `CloneOptions` in [pkg/git/git.go](https://github.com/kcl-lang/kpm/blob/d20b1acdc988f600c8f8465ecd9fe04225e19149/pkg/git/git.go#L19) as subDir.  

### using go-getter

As mentioned in the [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter#readme-subdirectories) docs, we can append our subDir from `CloneOptions` (only if subDir is not empty) in `WithRepoURL` function. 

### using go-git

This process will involve using the `sparse-checkout` feature of git. 

1. Initialize a new git repository in the local `.kcl/kpm/` directory using [PlainInit](https://pkg.go.dev/github.com/go-git/go-git#PlainInit). The repository name will be the PackageName_version.

2. Create a new worktree using [Worktree](https://pkg.go.dev/github.com/go-git/go-git/v5#Repository.Worktree)

3. Enable the sparse-checkout feature using [SparseCheckout](https://pkg.go.dev/github.com/go-git/go-git/v5#Worktree.SparseCheckout). The second argument will be a slice of strings containing the subdirectory path.

4. Add the remote repository using [AddRemote](https://pkg.go.dev/github.com/go-git/go-git/v5#Repository.CreateRemote)

5. Pull the repository using [Pull](https://pkg.go.dev/github.com/go-git/go-git/v5#Worktree.Pull)

Whenever we want to access the subdirectory using any command, we can refer to `kcl.mod` file of the project and iterate over the `subdir` array to get the path to the subdirectory. The `kcl.mod` file will automatically get updated whenever `kpm add` command is run.


### Additional modifications

To avoid creating a new root for each subdirectory download, I can add some check functions.

## References 

1. https://medium.com/@marcoscannabrava/git-download-a-repositorys-specific-subfolder-ceeabc6023e2
2. https://pkg.go.dev/github.com/go-git/go-git/v5
3. https://pkg.go.dev/github.com/hashicorp/go-getter











