# KPM sparse checkout

**Author**: Asish Kumar

## Abstract

`kpm` manages third-party libraries through Git repositories, requiring a `kcl.mod` file at the root directory. It treats the entire Git repository as a single `kcl` package, which is inefficient for monorepos containing multiple `kcl` packages. Often, a `kcl` project depends on just one package within a monorepo, but `kpm` downloads the entire repository. Therefore, `kpm` needs to allow adding a subdirectory of a Git repository as a dependency, enabling it to download only the necessary parts and improve performance.

## User Interface

I will add a new flag called `--subdir` in `kcl mod add` command.  This flag will specify the path to the desired subdirectory within the Git repository. Below is the syntax for the enhanced kpm add command:

```
kcl mod add --git <git-repo-url> --subdir <subdir> 
```

The `--subdir` flag will be optional. If the flag is not provided, `kpm` will download the entire repository as it does now. If the flag is provided, `kpm` will download only the specified subdirectory of the git repo.

Example usage: 

```
kcl mod add --git https://github.com/kcl-lang/modules --subdir add-certificates-volume 
```

This command will download the `add-certificates-volume` subdirectory from the `modules` repository and append it in the subdir array of the `kcl.mod` file.

The `kcl.mod` file will look like this:

```
[dependencies]
bbb = { git = "https://github.com/kcl-lang/modules", commit = "ade147b", subdir = ["add-ndots"]}
```

The subdir is a list because in the future if user wants to add another subdir from the same git repo then it can be added without overwritting the current subdir.

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

Whenever we want to access the subdirectory using any command, we can refer to `kcl.mod` file of the project and iterate over the `subdir` array to get the path to the subdirectory. The `kcl.mod` file will automatically get updated whenever `kcl mod add` command is run.

### Additional information

1. To avoid creating a new root for each subdirectory download, I can add some check functions.

2. The subdir flag is only for git options. If we pass it as a flag after oci, for example: `kpm add k8s --subdir 1.21/*`, it will not work. We can add a check [here](https://github.com/kcl-lang/kpm/blob/92158183556d39545bc0734a1e24284344ff3d9e/pkg/cmd/cmd_add.go#L154) that will give a warning if the subdir flag is passed. Furthermore, the subdir flag will only work for git repositories since it will insert the flag value into the field variable of the [Git](https://github.com/kcl-lang/kpm/blob/92158183556d39545bc0734a1e24284344ff3d9e/pkg/package/modfile.go#L375) struct.

## References 

1. https://medium.com/@marcoscannabrava/git-download-a-repositorys-specific-subfolder-ceeabc6023e2
2. https://pkg.go.dev/github.com/go-git/go-git/v5
3. https://pkg.go.dev/github.com/hashicorp/go-getter
