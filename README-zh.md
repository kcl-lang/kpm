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

### 安装 KCLVM

`kpm` 将调用 KCLVM 来编译 KCL 程序。在使用 `kpm` 之前，您需要确保 KCLVM 已经成功安装。

[如需了解如何安装 KCLVM 的更多信息，请参考此处。](https://kcl-lang.io/docs/user_docs/getting-started/install)

使用以下命令来确保您已成功安装KCLVM。

```shell
kclvm_cli version
```

如果上面的命令能为您正常展示 KCLVM 的版本信息，意味着您已经成功安装了 KCLVM 您就可以继续下一步操作。

### 安装 `kpm`

您可以从 Github release 中获取 `kpm` ，并将 `kpm` 的二进制文件路径设置到环境变量 PATH 中。

```shell
# KPM_INSTALLATION_PATH 是 `kpm` 二进制文件的所在目录.
export PATH=$KPM_INSTALLATION_PATH:$PATH  
```

请使用以下命令以确保您成功安装了`kpm`。

```shell
kpm --help
```

如果你看到以下输出信息，那么你已经成功安装了`kpm`，可以继续执行下一步操作。

```shell
NAME:
   kpm - kpm is a kcl package manager

USAGE:
   kpm  <command> [arguments]...

VERSION:
   v0.0.1

COMMANDS:
   init     initialize new module in current directory
   add      add new dependancy
   pkg      package a kcl package into tar
   run      compile kcl package.
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

### 设置环境变量

你需要设置一个环境变量 KPM_HOME 来声明 `kpm` 下载的 KCL 包的位置。

```shell
# Kpm 下载的包将会保存在 /user/xxx/xxx/path 目录下. 
export KPM_HOME="/user/xxx/xxx/path" 
```

在成功安装 KCLVM 后，为了确保 KCLVM 可以找到 `kpm` 下载的包，你需要设置环境变量 $KCLVM_VENDOR_HOME 并将其指向 $KPM_HOME。

```shell
export KCLVM_VENDOR_HOME=$KPM_HOME
```

## 快速开始

### 初始化一个空的 KCL 包

创建一个叫做 `mykcl` 的包。

```shell
kpm init mykcl
```

`kpm` 将会在执行`kpm init mykcl`命令的目录下创建两个默认的配置文件 `kcl.mod` 和 `kcl.mod.lock`。

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

如果你想要使用 [Konfig](https://github.com/awesome-kusion/konfig.git) 中的 KCL 程序.

```shell
kpm add -git https://github.com/awesome-kusion/konfig.git -tag v0.0.1
```

在环境变量 `$KPM_HOME` 描述的目录下, 你可以看到被下载的包。

```shell
- $KPM_HOME
      | - konfig_v0.0.1
```

并且 `kpm` 会为您将依赖添加到 kcl.mod 文件中.

```shell
[package]
name = "my_package"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
# 'konfig' 是依赖的包的名称
# 如果你想在你的 kcl 程序中使用包 'konfig' 中的内容，
# 你需要在 import 语句中使用包名 'konfig' 作为导入内容的前缀。
konfig = { git = "https://github.com/awesome-kusion/konfig.git", tag = "v0.0.1" }
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
import konfig.base.examples.native.nginx_deployment as nd

demo = nd.demo
```

### 使用 `kpm` 编译 kcl 包

你可以使用 kpm 编译刚才编写的 `main.k` 文件。

```shell
kpm run --input /my_package/main.k
```

如果你得到如下输出，恭喜你！你成功使用 `kpm` 编译了一个 kcl 包。

```shell
demo:
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: nginx-deployment
  spec:
    replicas: 3
    selector:
      matchLabels:
        app: nginx
    template:
      metadata:
        labels:
          app: nginx
      spec:
        containers:
          - image: "nginx:1.14.2"
            name: nginx
            ports:
              - containerPort: 80
```

### 打包您的 kcl 包

你可以使用 `kpm pkg` 将您的包与其对应的依赖打包在一起.

```shell
kpm pkg --target /my_package/my_package.tar
```

这个命令执行后，您可以看到您的 kcl 包已经被打包到了 `/my_package/my_package.tar` 文件中，并且 `my_package` 的依赖也都被复制到了当前包的 `vendor` 子目录下。

```shell
- my_package
        |- kcl.mod
        |- kcl.mod.lock
        |- main.k
        |- my_package.tar # `kpm pkg` 命令生成的 tar 包。
        |- vendor # 当前包所有的依赖都将被复制到 `vendor`中。 
             |- konfig_v0.0.1
```
