# kcl.mod: The KCL package Manifest File

## The Manifest

The `kcl.mod` file for each module is called its manifest. It is written in the TOML format. It contains metadata that is needed to compile the module.

In the MVP version, the sections we plan to support are as follows:
- Package metadata:
  - [package](#package) - Defines a package.
    - [name](#package) — The name of the package.
    - [version](#package) — The version of the package.
    - [edition](#package) — The KCL compiler edition.
- Dependency tables:
  - [dependencies](#dependencies) - Package library dependencies.
- Compiler settings:
  - [profile] - The compiler settings.
    - [entries](#entries) - The entry points of the package when compiling.

## package

The first section in a `kcl.mod` is [package].

```
[package]
name = "hello_world" # the name of the package
version = "0.1.0"    # the current version, obeying semver
edition = "0.5.0"    # the KCL compiler version
```

## dependencies

Your kcl package can depend on other libraries from OCI registries, git repositories, or subdirectories on your local file system.

You can specify a dependency by following:

```toml
[dependencies]
<package name> = <package_version>
```

You can specify a dependency from git repository.

```toml
[dependencies]
<package name> = { git = "<git repo url>", tag = "<git repo tag>" } 
```

You can specify a dependency from local file path.

```toml
# Find the kcl.mod under "./xxx/xxx".
[dependencies]
<package name> = {path = "<package local path>"} 
```

## entries
You can specify the entry points of the package when compiling.

`entries` is a sub section of `profile` section. 

```toml
[profile]
entries = [
   ...
]
```

`entries` is the relative path of kcl package root path, the `kcl.mod` file path is the package root path. There are two types of file paths formats supported, `normal paths` and `mod relative paths`.

- normal path: The path is relative to the current package root path.
- mod relative path: The path is relative to the vendor package root path that can be found by the section [dependencies](#dependencies) in `kcl.mod` file.

### For example:

1. If the `kcl.mod` is localed in `/usr/my_pkg/kcl.mod`, `kpm run` will take `/usr/my_pkg/entry1.k` and `/usr/my_pkg/subdir/entry2.k` as the entry point of the kcl compiler.

```toml
entries = [
   "entry1.k",
   "subdir/entry2.k",
]
```

2. If the `kcl.mod` is localed in `/usr/my_pkg/kcl.mod`, and the current kcl package depends on the kcl package `k8s`. You can use the `mod relative paths` the take the kcl file in the package `k8s` as the entry point of the kcl compiler.

```toml
entries = [
   "entry1.k",
   "subdir/entry2.k",
   "${k8s:KCL_MOD}/core/api/v1/deployment.k"
]
```

The `mod relative paths` must contains the preffix `${k8s:KCL_MOD}`, `k8s` is the package name, `${k8s:KCL_MOD}` means the package root path of the package `k8s`. Therefore, if the package root path of `k8s` is `/.kcl/kpm/k8s`, the `entries` show above will take `/usr/my_pkg/entry1.k`, `/usr/my_pkg/subdir/entry2.k` and `/.kcl/kpm/k8s/core/api/v1/deployment.k` as the entry point of the kcl compiler. 

### Note
You can use `normal path` to specify the compilation entry point in the current package path, and use `mod relative path` to specify the entry point in a third-party package.

Therefore, the file path specified by `normal path` must come from the same package, that is, the `kcl.mod` path found from the normal path must only find one `kcl.mod` file, otherwise the compiler will output an error.

For example:

In the path `/usr/kcl1`:
```
/usr/kcl1
      |--- kcl.mod
      |--- entry1.k
```

In the path `/usr/kcl2`:
```
/usr/kcl2
      |--- kcl.mod
      |--- entry2.k
```

If you compile with this `kcl.mod` in the path `/usr/kcl1`:
```
entries = [
   "entry1.k", # The corresponding kcl.mod file is /usr/kcl1/kcl.mod
   "/usr/kcl2/entry2.k", # The corresponding kcl.mod file is /usr/kcl2/kcl.mod
]
```

You will get an error:
```
error[E3M38]: conflict kcl.mod file paths
```
