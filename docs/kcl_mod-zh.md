# kcl.mod: KCL 包清单文件

## KCL 包清单

每个模块的 `kcl.mod` 文件都被称为其清单。它采用 TOML 格式编写，包含编译模块所需的元数据。

目前 `kcl.mod` 中支持如下内容：

- 包元数据：
  - [package](#package) - 定义一个包。
    - [name](#package) — 包的名称。
    - [version](#package) — 包的版本。
    - [edition](#package) — KCLVM 版本。
- 依赖表：
  - [dependencies](#dependencies) - 包库依赖项。
- 编译器设置：
  - [profile](#entries) - 编译器设置。
    - [entries](#entries) - 编译包的入口点。

## package
`kcl.mod` 中的第一个部分是 `[package]`。

```
[package] 
name = "hello_world" # 包的名称 
version = "0.1.0" # 当前版本，遵循 semver 规范 
edition = "0.1.1-alpha.1" # KCL 编译器版本
```

## dependencies

你的 kcl 包可以依赖于来自 OCI 仓库、Git 存储库或本地文件系统子目录的其他库。

你可以按照以下方式指定依赖项：

```toml
[dependencies]
<package name> = <package_version>
```

你可以从 Git 存储库指定依赖项:
```
[dependencies]
<package name> = { git = "<git repo url>", tag = "<git repo tag>" } 
```

你可以从本地文件路径指定依赖项:
```
[dependencies]
<package name> = {path = "<package local path>"} 
```

## entries

你可以在编译时指定包的入口点。

`entries` 是 `[profile]` 部分的子部分。

```toml
[profile]
entries = [
   ...
]
```

`entries` 是 kcl 包根路径的相对路径，`kcl.mod` 文件路径是包的根路径。支持两种文件路径格式，即 `normal paths` 和 `mod relative paths`。

- normal path：路径相对于当前包的根路径。
- mod relative path：路径相对于可以在 kcl.mod 文件的 [dependencies](#dependencies) 部分中的三方包的根路径。

例如：
1. 如果 `kcl.mod` 位于 `/usr/my_pkg/kcl.mod`，则 `kpm run` 将把 `/usr/my_pkg/entry1.k` 和 `/usr/my_pkg/subdir/entry2.k` 作为 `kcl` 编译器的入口点。

```
entries = [
   "entry1.k",
   "subdir/entry2.k",
]
```

2. 如果 `kcl.mod` 位于 `/usr/my_pkg/kcl.mod`，并且当前 `kcl` 包依赖于 `kcl` 包 `k8s`。你可以使用 `mod relative paths` 将来自包 `k8s` 中的 `kcl` 文件作为 `kcl` 编译器的入口点。

```
entries = [
   "entry1.k",
   "subdir/entry2.k",
   "{k8s:KCL_MOD}/core/api/v1/deployment.k"
]
```

`mod relative paths` 必须包含前缀 `{k8s:KCL_MOD}`，其中 `k8s` 是包名，`{k8s:KCL_MOD}` 表示包 k8s 的包根路径。因此，如果 `k8s` 的包根路径是 `/.kcl/kpm/k8s`，则上面的 `entries` 将把 `/usr/my_pkg/entry1.k`、`/usr/my_pkg/subdir/entry2.k` 和 `/.kcl/kpm/k8s/core/api/v1/deployment.k` 作为 `kcl` 编译器的入口点。

### 注意
你可以使用 `normal path` 指定当前包路径中的编译入口点，使用 `mod relative path` 指定三方包中的入口点。

因此，使用 `normal path` 指定来自不同包的入口点时，必须包含在当前包路径中。如果你使用 `normal path` 指定来自不同包的入口点，kcl 编译器将报告错误。 