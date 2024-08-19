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

The user can then run `kcl mod run` to run the code: 

`kcl mod run`

This will checkout the destination directory which contains that package within the repository. 

## Design

In order to use this feature, a new field `package` will be added to the `kcl.mod` file. This field will contain the package name that the user wants to use. 

Earlier the download only happens, in case of git, when there is a `kcl.mod` file in the root. Enabling this feature, will allow download of git repository even when there is no `kcl.mod` file in the root but this will only work if a package flag is passed. 
