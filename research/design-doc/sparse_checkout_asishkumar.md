# KPM sparse checkout

**Author**: Asish Kumar

## Solution

In order to add the feature of sparse-checkout, kpm will add the package specified in the kcl.mod file and later when running `kcl mod run` it will checkout that destination directory which contains that package recursively. 

## User Interface

In order to use a specified package within a repository, the user will have to specify the package during the `kcl mod add` command. For example

```
kcl mod add --git https://github.com/officialasishkumar/check-kcl.git --commit 831fada36e155c8758f07f293c8267c869af69d3 --package k8s
```

This command will recursively search for the package name within all the existing kcl.mod files in the repository and load it by specifying the package name in `kcl.mod` under `dependencies`.

The package flag is optional. If not provided kpm will work normally as before and you will need to have a `kcl.mod` file in the root of the repository. 

There is also an option to manually write the package flag in `kcl.mod` file. For example: 

```
[dependencies]
agent = { git = "https://github.com/kcl-lang/modules.git", commit = "ee03122b5f45b09eb48694422fc99a0772f6bba8", package = "agent" }
```

This will work the same way as before only thing to note is you need to have a `kcl.mod` file in the root of the repository when running the command: 

```
kcl mod add --git <url> --commit <hash>
```

The user can then run `kcl mod run` or `kcl run` to run the code: 

`kcl mod run`

`kcl run`

This will checkout the destination directory which contains that package within the repository. You can then use the loaded dependencies in your code.

The user can also run the following commands with package in there `kcl.mod` file: 

```
kcl mod metadata
```

```
kcl mod metadata --update
```

```
kcl mod metadata --vendor
```

```
kcl mod graph
```

## Design

In order to use this feature, a new field `package` will be added to the `kcl.mod` file. This field will contain the package name that the user wants to use. 

Earlier the download only happens, in case of git, when there is a `kcl.mod` file in the root. Enabling this feature, will allow download of git repository even when there is no `kcl.mod` file in the root but this will only work if a package flag is passed. 


# Implementation and conclusion

The idea implemented in the following PR was mentioned in https://github.com/kcl-lang/kpm/pull/335#issuecomment-2151338180.  

Here are the merged PRs: 

- https://github.com/kcl-lang/kpm/pull/453
- https://github.com/kcl-lang/kpm/pull/457

The changes made are tested by unit tests. 
