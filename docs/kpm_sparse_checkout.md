## KPM Sparse-Checkout feature

## 1. Introduction
kpm which is the package management tool for KCL, does not support sparse-checkout, which means an issue arises when dealing with monorepos, which may contain many KCL packages. We need to get a solution to download specific packages instead of all in a monorepo.

## 2. User Worklow
- Adding a Subdirectory Dependency via Command Line
- Specifying a Subdirectory in `kcl.mod`

## 3. Command Line Inteface 
We will try to keep the command line interface as simple and straightforward as possible for adding a Git repository subdirectory as a dependency. The below steps show how a user can use the feature:

- Adding a Dependency :
The following command will lead to the addition of a dependency:
`kpm add <repository_url> <subdirectory_path>`

This command will add the specified subdirectory as a dependency in the current KCL project.

## 4. Example Use-case
Considering the nginx-ingres module, on typing the command 

```
kpm add https://github.com/kcl-lang/modules/tree/main/nginx-ingress /restrict-ingress-anotations
``` 

The following command will lead to the addition of restrict-ingress-anotations package to the current KCL project and will update the kcl.mod accordingly.

## 5. Specifying how kcl.mod will specify the added subdirectories
To support the specification of a subdirectory for dependencies that are sourced from Git repositories, we need to extend the kcl.mod file structure. This involves adding an optional `subdirectory` field under each dependency.
A sample kcl mod for the same would look like this.

```
[package]
name = "test"
edition = "v0.9.0"
version = "0.0.1"

[dependencies]
my_dependency = { git = "https://github.com/example/repo", subdirectory = "path/to/package" }
```

## 6. Integration and the use of go-getter to download the specific subdirectories

The repoUrl field in the struct `CloneOptions` in kpm/pkg/git/git.go will be given the subdir url accordingly, which then downloads each selected subdirectory one by one. KCL currently uses go-getter to download using URL's with ease.

We can also provide a fallback mechanism to download the entire repo if the subdirectory download fails repeatedly by something like - 

```Go
err := downloadSubdir(repoUrl, subdir)
if err != nil {
    log.Printf("Failed to download subdirectory, falling back to full repository: %v", err)
    return downloadRepo(repoUrl)
}

```

## 7. Conclusion
By extending the kcl.mod file to include a subdirectory field for Git-based dependencies, users can now specify and manage dependencies that reside in specific subdirectories of a Git repository. This enhancement will enable kpm to download only the necessary parts of large repositories, significantly improving performance when dealing with monorepos. The user interface remains intuitive, and the implementation leverages Git's sparse-checkout feature to achieve the desired functionality.