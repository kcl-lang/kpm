# 如何在 github action 中使用 kpm 发布您的 KCL 包 

[kpm](https://github.com/KusionStack/kpm) 是一个用于管理 kcl 包的工具。本文将指导您如何在 GitHub Action 中使用 kpm 将您的 kcl 包推送到发布到 ghcr.io 中。

下面是一个简单的步骤，指导您如何使用 kpm 将您的 kcl 包推送到 OCI Registry。

## 步骤 1：安装 kpm

首先，您需要在您的计算机上安装 kpm。您可以按照 [kpm 安装文档](https://kcl-lang.io/docs/user_docs/guides/package-management/installation)中的说明进行操作。

## 步骤 2：创建一个 GitHub 账号

如果您已经有 GitHub 帐号了，您可以选择跳过这一步

[注册新的一个 GitHub 账号](https://docs.github.com/zh/get-started/signing-up-for-github/signing-up-for-a-new-github-account)

## 步骤 3: 为您的 KCL 包创建一个 GitHub 仓库并进行相关配置

您需要为您的 KCL 程序包准备一个 GitHub 仓库。

[创建一个GitHub仓库](https://docs.github.com/zh/get-started/quickstart/create-a-repo) 

您需要为您的 GitHub Action 准备一个 GitHub Token，用于向 ghcr.io 写入内容。

[创建一个 GitHub Token](https://docs.github.com/zh/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#personal-access-tokens-classic)

在仓库中添加 token 作为 secrets。

[为仓库添加 secrets](https://docs.github.com/zh/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository)

将我们添加的 token 命令为 `REG_TOKEN`.

## 步骤 4: 将您的 KCL 包添加到仓库中并编写 github action workflow

为这个仓库添加 github action 文件 `.github/workflows/push.yml`，内容如下：

```yaml
name: KPM Push Workflow

on:
  push:
    branches:
      - main

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      # 安装 go 环境
      - name: Set up Go 1.19
        uses: actions/setup-go@v2
        with:
          go-version: 1.19

      # go install 安装 kpm 
      - name: Install kpm
        run: go install kcl-lang.io/kpm@latest

      # kpm login and kpm push
      - name: Login and Push kpm project
        run: kpm login -u ${{ github.actor }} -p ${{ secrets.TEST }} ghcr.io && kpm push

      - name: Run kpm project
        # 测试是否 push 成功
        run: kpm run oci://ghcr.io/kcl-lang/mykcl --tag 0.0.1
```

## 步骤 5(可选): 将您的 KCL 包推送到自定义的 OCI Registry 中

上面的步骤中，通过 GitHub action 将 KCL 包推送到默认的 OCI Registry `ghcr.io/kcl-lang` 中。如果您想将 KCL 包推送到自定义的 OCI Registry 中，您可以按照下面的步骤进行操作。

以 `docker.io` 为例：

您需要获取到您自定义 OCI Registry 仓库的 Token, 如果您使用的是 `docker.io`, `docker.io` 的账户密码就可以作为 token 使用, 并参考[步骤3](#步骤-3-为您的-kcl-包创建一个-github-仓库并进行相关配置)将其设置为 GitHub Action 的 secrets。

您可以通过设置 Github Action Variables 来指定您的 OCI Registry 账户

[为 Github 仓库设置 Variables](https://docs.github.com/zh/actions/learn-github-actions/variables#creating-configuration-variables-for-a-repository)

您可以将您的 docker 账户设置为 Variables `REG_ACCOUNT`。
将您的 docker 账户密码设置为 secrets `REG_TOKEN`。

在 `.github/workflows/push.yml` 中。

```yaml
name: KPM Push Workflow

on:
  push:
    branches:
      - main

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      # 安装 go 环境
      - name: Set up Go 1.19
        uses: actions/setup-go@v2
        with:
          go-version: 1.19

      # go install 安装 kpm 
      - name: Install kpm
        run: go install kcl-lang.io/kpm@latest

      # kpm login and kpm push
      - name: Login and Push kpm project
      # 通过环境变量指定自定义的 OCI Registry
        env:
          KPM_REG: "docker.io"
          KPM_REPO: ${{ vars.REG_ACCOUNT }}
        run: kpm login -u ${{ vars.REG_ACCOUNT }} -p ${{ secrets.REG_TOKEN }} docker.io && kpm push

      - name: Run kpm project
        # 测试是否 push 成功
        run: kpm run oci://docker.io/${{ vars.REG_ACCOUNT }}/mykcl --tag 0.0.1
```
