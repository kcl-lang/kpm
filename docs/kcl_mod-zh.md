# kcl.mod: KCL 包清单文件

## 1. KCL 包清单

每个模块的 `kcl.mod` 文件都被称为其清单。它采用 TOML 格式编写，包含编译模块所需的元数据。

目前 `kcl.mod` 中支持如下内容：

- 包元数据：
  - [package](#package) - 定义一个包。
    - [name](#package) — 包的名称。
    - [version](#package) — 包的版本。
    - [edition](#package) — KCL 编译器版本。
- 依赖表：
  - [dependencies](#dependencies) - 包库依赖项。
- 编译器设置：
  - [profile](#entries) - 编译器设置。
    - [entries](#entries) - 编译包的入口点。

## 2. package
`kcl.mod` 中的第一个部分是 `[package]`。主要包含 `name`, `version` 和 `edition` 三个字段。

### 2.1. name
`name` 是包的名称，它是一个字符串, 这是一个必要的字段, 注意，包的名称中不可以包含`"."`。

例如: 一个包名为 my_pkg 的 kcl 程序包。
```
[package]
name = "my_pkg"
```

### 2.2. version
`version` 是包的版本，它是一个字符串, 这是一个必要的字段。注意，目前 KCL 程序包的版本号仅支持语义化版本号。

例如: `my_pkg` 程序包的版本号为 `0.1.0`。
```
[package]
name = "my_pkg"
version = "0.1.0"
```

### 2.3. edition
`edition` 是 KCL 编译器版本，它是一个字符串, 这是一个必要的字段。注意，目前 KCL 编译器版本号仅支持语义化版本号。

例如: `my_pkg` 程序包的版本号为 `0.1.0`, 并且与 0.5.1 的 KCL 编译器兼容。
```
[package]
name = "my_pkg"
version = "0.1.0"
edition = "0.5.0"
```

## 3. dependencies

你的 kcl 包可以依赖于来自 OCI 仓库、Git 存储库或本地文件系统子目录的其他库。

### 3.1. oci dependency

kpm 默认将包保存在 oci registry 上，默认使用的 oci registry 是 `ghcr.io/kcl-lang`。
更多内容关于 oci registry 请参考 [kpm 支持 OCI registry](./docs/kpm_oci-zh.md)。

你可以按照以下方式指定依赖项：

```toml
[dependencies]
<package name> = <package_version>
```

这将会从 oci registry 中拉取名称为 `<package name>` 的包，版本为 `<package_version>`。

如果您希望拉取 `k8s` 包的 `1.27` 版本:

```toml
[dependencies]
k8s = "1.27"
```

### 3.2. git dependency

从 Git 存储库指定依赖项:
```
[dependencies]
<package name> = { git = "<git repo url>", tag = "<git repo tag>" } 
```

这将会从 Git 存储库`<git repo url>`中拉取名称为 `<package name>` 的包，`tag` 为 `<git repo tag>`。

## 4. entries

你可以在编译时指定包的入口点。

`entries` 是 `[profile]` 部分的子部分。entries 是一个字符串数组，包含编译器的入口点。这是一个可选的字段，如果没有指定，则默认为包根目录下的所有 `*.k` 文件。

```toml
[profile]
entries = [
   ...
]
```

entries 中可以定义绝对路径和相对路径，如果定义的是相对路径，那么就会以当前包的 

`entries` 是 kcl 包根路径的相对路径，`kcl.mod` 文件路径是包的根路径。支持两种文件路径格式，即 `normal paths` 和 `mod relative paths`。

- normal path：相对于当前包的根路径。
- mod relative path：相对于 kcl.mod 中 [dependencies](#dependencies) 部分中的三方包的根路径。

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
   "${k8s:KCL_MOD}/core/api/v1/deployment.k"
]
```

`mod relative paths` 必须包含前缀 `${k8s:KCL_MOD}`，其中 `k8s` 是包名，`${k8s:KCL_MOD}` 表示包 k8s 的包根路径。因此，如果 `k8s` 的包根路径是 `/.kcl/kpm/k8s`，则上面的 `entries` 将把 `/usr/my_pkg/entry1.k`、`/usr/my_pkg/subdir/entry2.k` 和 `/.kcl/kpm/k8s/core/api/v1/deployment.k` 作为 `kcl` 编译器的入口点。

### 注意
你可以使用 `normal path` 指定当前包路径中的编译入口点，使用 `mod relative path` 指定三方包中的入口点。

因此，使用 `normal path` 制定的文件路径必须来自于同一个包，即从 `normal path` 开始寻找的 `kcl.mod` 路径必须只能找到一个 `kcl.mod` 文件，不然编译器将输出错误。

例如:

在路径 `/usr/kcl1` 下
```
/usr/kcl1
      |--- kcl.mod
      |--- entry1.k
```

在路径 `/usr/kcl2` 下
```
/usr/kcl2
      |--- kcl.mod
      |--- entry2.k
```

如果你在路径`/usr/kcl1`下使用这样的 kcl.mod 编译：
```
entries = [
   "entry1.k", # 对应的 kcl.mod 文件是 /usr/kcl1/kcl.mod
   "/usr/kcl2/entry2.k", # 对应的 kcl.mod 文件是 /usr/kcl2/kcl.mod
]
```

将会得到错误：
```
error[E3M38]: conflict kcl.mod file paths
```