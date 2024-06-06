# KPM sparse checkout

**Author**: Asish Kumar

## Abstract

`kpm` manages third-party libraries through Git repositories, requiring a `kcl.mod` file at the root directory. It treats the entire Git repository as a single `kcl` package, which is inefficient for monorepos containing multiple `kcl` packages. Often, a `kcl` project depends on just one package within a monorepo, but `kpm` downloads the entire repository. Therefore, `kpm` needs to allow adding a subdirectory of a Git repository as a dependency, enabling it to download only the necessary parts and improve performance.

## User Interface

The user can just provide the git url of the subdir they want to install. An example command will look like this:

```
kcl mod add --git https://github.com/kcl-lang/modules/tree/main/argoproj --tag <tag>
```

kpm would parse the git url and extract the subdirectory path using `GetPath()` function from github.com/kubescape/go-git-url subdir. It will then download the subdirectory and append it in the subdir array of the `kcl.mod` file. 

The `kcl.mod` file will look like this:

```
[dependencies]
bbb = { git = "https://github.com/kcl-lang/modules", commit = "ade147b", subdir = ["add-ndots"]}
```

The subdir is a list because in the future if user wants to add another subdir from the same git repo then it can be added without overwritting the current subdir.

## Design

The path to the directory will be passed to `CloneOptions` in [pkg/git/git.go](https://github.com/kcl-lang/kpm/blob/d20b1acdc988f600c8f8465ecd9fe04225e19149/pkg/git/git.go#L19) as subdir.  

As mentioned in the [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter#readme-subdirectories) docs, we can append our subdir from `CloneOptions` (only if subdir is not empty) in `WithRepoURL` function. 

## References 

1. https://medium.com/@marcoscannabrava/git-download-a-repositorys-specific-subfolder-ceeabc6023e2
2. https://pkg.go.dev/github.com/hashicorp/go-getter
3. https://pkg.go.dev/github.com/kubescape/go-git-url
