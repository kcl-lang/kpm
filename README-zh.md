<h1 align="center">Kpm: KCL 包管理器</h1>

<p align="center">
<a href="./README.md">English</a> | <a href="./README-zh.md">简体中文</a>
</p>
<p align="center">
<a href="#introduction">介绍</a> | <a href="#installation">安装</a> | <a href="#quick-start">快速开始</a>
</p>

<p align="center">
<img src="https://coveralls.io/repos/github/KusionStack/kpm/badge.svg">
<img src="https://img.shields.io/badge/license-Apache--2.0-green">
<img src="https://img.shields.io/badge/PRs-welcome-brightgreen">
</p>

## 介绍

`kpm` 是 KCL 包管理器。`kpm` 会下载您的 KCL 包的依赖项、编译您的 KCL 包、制作可分发的包并将其上传到 KCL 包的仓库中。

## 安装

### 安装 KCL

`kpm` 将调用 [KCL 编译器](https://github.com/KusionStack/KCLVM) 来编译 KCL 程序。在使用 `kpm` 之前，您需要确保 KCL 编译器 已经成功安装。

[如需了解如何安装 KCL 的更多信息，请参考此处。](https://kcl-lang.io/docs/user_docs/getting-started/install)

使用以下命令来确保您已成功安装 `KCL`。

```shell
kcl -V
```

### 安装 `kpm`

#### 使用 `go install` 安装

您可以使用 `go install` 命令安装 `kpm`。

```shell
go install kcl-lang.io/kpm@latest
```

如果您在执行完上述命令后,使用 `kpm` 时,无法找到命令 `kpm` 请参考:

- [go install 安装后找不到命令。](#q-我在使用go-install安装kpm后出现了command-not-found的错误)

#### 从 Github release 页面手动安装

您也可以从 Github release 中获取 `kpm` ，并将 `kpm` 的二进制文件路径设置到环境变量 PATH 中。

```shell
# KPM_INSTALLATION_PATH 是 `kpm` 二进制文件的所在目录.
export PATH=$KPM_INSTALLATION_PATH:$PATH  
```

请使用以下命令以确保您成功安装了 `kpm`。

```shell
kpm --help
```

如果你看到以下输出信息，那么你已经成功安装了 `kpm`，可以继续执行下一步操作。

<img src="./docs/gifs/kpm_help.gif" width="600" align="center" />

## 快速开始

### 初始化一个空的 KCL 包

使用 `kpm init` 命令创建一个名为 `my_package` 的 kcl 程序包, 并且在我们创建完成一个名为 `my_package` 的包后，我们需要通过命令 `cd my_package` 进入这个包来进行后续的操作。

```shell
kpm init my_package
```

<img src="./docs/gifs/kpm_init.gif" width="600" align="center" />

`kpm` 将会在执行 `kpm init my_package` 命令的目录下创建两个默认的配置文件 `kcl.mod` 和 `kcl.mod.lock`。

```shell
- my_package
        |- kcl.mod
        |- kcl.mod.lock
        |- # 你可以直接在这个目录下写你的kcl程序。
```

`kcl.mod.lock` 是 `kpm` 用来固定依赖版本的文件，是自动生成的，请不要人工修改这个文件。

`kpm` 将会为这个新包创建一个默认的 `kcl.mod`。如下所示:

```shell
[package]
name = "my_package"
edition = "0.0.1"
version = "0.0.1"
```

### 为 KCL 包添加依赖

然后，您可以通过 `kpm add` 命令来为您当前的库添加一个外部依赖。

如下面的命令所示，为当前包添加一个版本号为 `1.27` 并且名为 `k8s` 的依赖包。

```shell
kpm add k8s:1.27
```

<img src="./docs/gifs/kpm_add_k8s.gif" width="600" align="center" />

`kpm` 会为您将依赖添加到 kcl.mod 文件中.

```shell
[package]
name = "my_package"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
k8s = "1.27" # The dependency 'k8s' with version '1.27'
```

### 编写一个程序使用包 `konfig` 中的内容

在当前包中创建 `main.k`。

```shell
- my_package
        |- kcl.mod
        |- kcl.mod.lock
        |- main.k # Your KCL program.
```

并且将下面的内容写入 `main.k` 文件中。

```kcl
# 导入并使用外部依赖 `k8s` 包中的内容。
import k8s.api.core.v1 as k8core

k8core.Pod {
    metadata.name = "web-app"
    spec.containers = [{
        name = "main-container"
        image = "nginx"
        ports = [{containerPort = 80}]
    }]
}

```

### 使用 `kpm` 编译 kcl 包

你可以使用 kpm 编译刚才编写的 `main.k` 文件, 得到编译后的结果。

```shell
kpm run
```

<img src="./docs/gifs/kpm_run.gif" width="600" align="center" />

## OCI Registry 的支持

从 kpm v0.2.0 版本开始，kpm 支持通过 OCI Registries 保存和分享 KCL 包。

了解更多如何在 kpm 中使用，查看 [OCI registry 支持](./docs/kpm_oci-zh.md).

## 常见问题 (FAQ)

##### Q: 我在使用 `go install` 安装 `kpm` 后，出现了 `command not found` 的错误。

A: `go install` 默认会将二进制文件安装到 `$GOPATH/bin` 目录下，您需要将 `$GOPATH/bin` 添加到环境变量 `PATH` 中。

## 了解更多

- [OCI registry 支持](./docs/kpm_oci-zh.md).
- [如何使用 kpm 与他人分享您的 kcl 包](./docs/publish_your_kcl_packages-zh.md)
- [如何使用 kpm 在 docker.io 上与他人分享您的 kcl 包](./docs/publish_to_docker_reg-zh.md)
- [kpm 命令参考](./docs/command-reference-zh/index.md)
- [kcl.mod: KCL 包清单文件](./docs/kcl_mod-zh.md)
