# 如何在 github action 中使用 kpm 发布您的 KCL 包 

[kpm](https://github.com/KusionStack/kpm) 是一个用于管理 kcl 包的工具。本文将指导您如何在 GitHub Action 中使用 kpm 将您的 kcl 包推送到发布到 ghcr.io 中。

下面是一个简单的步骤，指导您如何使用 kpm 将您的 kcl 包推送到 ghcr.io。

## 步骤 1：安装 kpm

首先，您需要在您的计算机上安装 kpm。您可以按照 [kpm 安装文档](https://kcl-lang.io/docs/user_docs/guides/package-management/installation)中的说明进行操作。

## 步骤 2：创建一个 GitHub 账号

如果您已经有 GitHub 帐号了，您可以选择跳过这一步

[注册新的一个 GitHub 账号](https://docs.github.com/zh/get-started/signing-up-for-github/signing-up-for-a-new-github-account)

## 步骤 3: 为您的 KCL 包创建一个 GitHub 仓库并进行相关配置

您需要为您的 KCL 程序包准备一个 GitHub 仓库。

[创建一个GitHub仓库](https://docs.github.com/zh/get-started/quickstart/create-a-repo) 

为您的 GitHub Action 增加向 ghcr.io 写入内容的权限。

[GitHub Action 权限管理](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#configuring-the-default-github_token-permissions)

您需要为您的 GitHub Action 准备一个 GitHub Token，用于向 ghcr.io 写入内容。

[创建一个 GitHub Token](https://docs.github.com/zh/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#personal-access-tokens-classic)

在仓库中添加 token 作为 secrets。

[为仓库添加 secrets](https://docs.github.com/zh/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository)

将我们添加的 token 命令为 `TEST`.

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
      - name: Push kpm project
        run: kpm login -u ${{ github.actor }} -p ${{ secrets.TEST }} ghcr.io && kpm push

      - name: Run kpm project
        # 测试是否 push 成功
        run: kpm run oci://ghcr.io/kcl-lang/mykcl --tag 0.0.1
```
