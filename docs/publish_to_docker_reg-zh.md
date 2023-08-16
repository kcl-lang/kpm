# 如何使用 kpm 在 docker.io 上与他人分享您的 kcl 包

[kpm](https://github.com/KusionStack/kpm) 是一个用于管理 kcl 包的工具。本文将指导您如何使用 kpm 将您的 kcl 包推送到发布到 docker.io 中。

下面是一个简单的步骤，指导您如何使用 kpm 将您的 kcl 包推送到 docker.io。

## 步骤 1：安装 kpm

首先，您需要在您的计算机上安装 kpm。您可以按照 [kpm 安装文档](https://kcl-lang.io/docs/user_docs/guides/package-management/installation)中的说明进行操作。

## 步骤 2：创建一个 docker.io 账户

您需要创建一个 docker.io 账户以支持您的 kcl 包的推送。

## 步骤 3：登录 docker.io

您可以直接使用 docker.io 的账户名和密码登录。

```shell
kpm login -u <USERNAME> -p <PASSWORD> docker.io
```

其中 `<USERNAME>` 是您的 docker.io 用户名，`<PASSWORD>` 是您 docker.io 账户的密码。

关于如何使用 kpm 登录 docker.io 的更多信息，请参阅 [kpm login](./kpm_oci-zh.md#kpm-login)。

## 步骤 4：推送您的 kcl 包

现在，您可以使用 kpm 将您的 kcl 包推送到 docker.io。

### 1. 一个合法的 kcl 包

首先，您需要确保您推送的内容是符合一个 kcl 包的规范，即必须包含合法的 kcl.mod 和 kcl.mod.lock 文件。

如果您不知道如何得到一个合法的 `kcl.mod` 和 `kcl.mod.lock`。您可以使用 `kpm init` 命令。

```shell
# 创建一个名为 my_package 的 kcl 包
kpm init my_package
```

`kpm init my_package` 命令将会为您创建一个新的 kcl 包 `my_package`, 并为这个包创建 `kcl.mod` 和 `kcl.mod.lock` 文件。

如果您已经有了一个包含 kcl 文件的目录 `exist_kcl_package`，您可以使用以下命令将其转换为一个 kcl 包，并为其创建合法的 `kcl.mod` 和 `kcl.mod.lock`。

```shell
# 在 exist_kcl_package 目录下
$ pwd 
/home/user/exist_kcl_package

# 执行 kpm init 命令来创建 kcl.mod 和 kcl.mod.lock
$ kpm init 
```

关于如何使用 kpm init 的更多信息，请参阅 [kpm init](./command-reference-zh/1.init.md)。

### 2. 推送 kcl 包

您可以在 `kcl` 包的根目录下使用以下命令进行操作：

```shell
# 在 exist_kcl_package 包的根目录下
$ pwd 
/home/user/exist_kcl_package

# 推送 kcl 包到默认的 OCI Registry
$ kpm push oci://docker.io/<USERNAME>/exist_kcl_package
```

完成上述步骤后，您就成功地将您的 kcl 包 `exist_kcl_package` 推送到了 docker.io 中。
关于如何使用 kpm push 的更多信息，请参阅 [kpm push](./kpm_oci-zh.md#kpm-push)。
