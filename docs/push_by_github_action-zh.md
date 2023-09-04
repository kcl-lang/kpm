# 如何在 github action 中使用 kpm 发布您的 KCL 包 

[kpm](https://github.com/KusionStack/kpm) 是一个用于管理 kcl 包的工具。本文将指导您如何在 GitHub Action 中使用 kpm 将您的 kcl 包推送到发布到 ghcr.io 中。

下面是一个简单的步骤，指导您如何使用 kpm 将您的 kcl 包推送到 OCI Registry。

## 步骤 1：安装 kpm

首先，您需要在您的计算机上安装 kpm。您可以按照 [kpm 安装文档](https://kcl-lang.io/docs/user_docs/guides/package-management/installation)中的说明进行操作。

## 步骤 2：创建一个 GitHub 账号

如果您已经有 GitHub 帐号了，您可以选择跳过这一步

[注册新的一个 GitHub 账号](https://docs.github.com/zh/get-started/signing-up-for-github/signing-up-for-a-new-github-account)

## 步骤 3: 为您的 KCL 包创建一个 GitHub 仓库并进行相关配置

### 1. 为您的 KCL 程序包准备仓库
您需要为您的 KCL 程序包准备一个 GitHub 仓库。

[创建一个 GitHub 仓库](https://docs.github.com/zh/get-started/quickstart/create-a-repo) 


在这个仓库中添加您的 KCL 程序，以仓库 https://github.com/awesome-kusion/catalog.git 为例，

```bash
├── .github
│   └── workflows
│       └── push.yaml # github action 文件 
├── LICENSE
├── README.md
├── kcl.mod # kcl.mod 将当前仓库内容定义为一个 kcl 包
├── kcl.mod.lock # kcl.mod.lock 是 kpm 自动生成的文件
└── main.k # 您的 KCL 程序
```

### 2. 为您的仓库设置 OCI Registry，账户和密码

#### 2.1 通过 GitHub action variables 设置您的 OCI Registry，账户

[为 Github 仓库设置 Variables](https://docs.github.com/zh/actions/learn-github-actions/variables#creating-configuration-variables-for-a-repository)

以 docker.io 为例，您可以为您的仓库设置两个 Variables `REG` 和 `REG_ACCOUNT`。`REG` 的值为 `docker.io`，`REG_ACCOUNT` 的值为您的 docker.io 账户。

#### 2.2 通过 GitHub action secrets 设置您的 OCI Registry 密码

[为仓库添加 secrets](https://docs.github.com/zh/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository)

以 `docker.io` 为例，您可以将您的 `docker.io` 登录密码设置为名为 `REG_TOKEN` 的 secrets 。

如果您使用 `ghcr.io` 作为 `Registry`, 您需要使用 GitHub token 作为 secrets。

[创建一个 GitHub Token](https://docs.github.com/zh/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#personal-access-tokens-classic)


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

      - name: Set up Go 1.19
        uses: actions/setup-go@v2
        with:
          go-version: 1.19

      - name: Install kpm
        run: go install kcl-lang.io/kpm@latest

      - name: Login and Push
        env:
          # 通过环境变量指定 OCI Registry 和账户
          KPM_REG: ${{ vars.REG }}
          KPM_REPO: ${{ vars.REG_ACCOUNT }}
          # kpm login 时使用 secrets.REG_TOKEN 
        run: kpm login -u ${{ vars.REG_ACCOUNT }} -p ${{ secrets.REG_TOKEN }} ${{ vars.REG }} && kpm push

      - name: Run kpm project from oci registry
        run: kpm run oci://${{ vars.REG }}/${{ vars.REG_ACCOUNT }}/catalog --tag 0.0.1

```
