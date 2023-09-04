# How to Push Your KCL Package by GitHub Action

[kpm](https://github.com/KusionStack/kpm) is a tool for managing kcl packages. This article will guide you how to use kpm in GitHub Action to push your kcl package to OCI registry.

## Step 1: Install kpm

At first, you need to install kpm on your computer. You can follow [kpm installation document](https://kcl-lang.io/docs/user_docs/guides/package-management/installation).

## Step 2: Create a GitHub account

If you already have a GitHub account, you can skip this step.

[Sign up for a new GitHub account](https://docs.github.com/en/get-started/signing-up-for-github/signing-up-for-a-new-github-account)

## Step 3: Create a GitHub repository for your KCL package

You need to prepare a GitHub repository for your KCL package.

[Create a GitHub repository](https://docs.github.com/en/get-started/quickstart/create-a-repo)

You need to prepare a GitHub Token for your GitHub Action to write to ghcr.io.

[create a GitHub Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#personal-access-tokens-classic)

Add the token as secrets to the repository.

[Add the secrets to the repository](https://docs.github.com/en/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository)

Name the token we added as `REG_TOKEN`.

## Step 4: Add your KCL package to the repository and write github action workflow

Add github action file `.github/workflows/push.yml` to the repository, the content is as follows:

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

      # go install kpm 
      - name: Install kpm
        run: go install kcl-lang.io/kpm@latest

      # kpm login and kpm push
      - name: Login and Push kpm project
        run: kpm login -u ${{ github.actor }} -p ${{ secrets.REG_TOKEN }} ghcr.io && kpm push

      - name: Run kpm project
        # test if push success
        run: kpm run oci://ghcr.io/kcl-lang/mykcl --tag 0.0.1
```

## Step 5(optional): Push your KCL package to your custom OCI Registry

In the above steps, we push the KCL package to the default OCI Registry `ghcr.io/kcl-lang` through GitHub action. If you want to push the KCL package to your custom OCI Registry, you can follow the steps below.

Take `docker.io` as an example:

You need to get the token of your custom OCI Registry. If you use `docker.io`, the account password of `docker.io` can be used as token, and refer to [Step 3](#step-3-create-a-github-repository-for-your-kcl-package-and-perform-related-configuration) to set it as GitHub action secrets.

You can specify your OCI Registry account by setting Github Action Variables

[Set Variables for Github repository](https://docs.github.com/en/actions/learn-github-actions/variables#creating-configuration-variables-for-a-repository)

You can set your docker account as variables `REG_ACCOUNT`.
You can set your docker account password as secrets `REG_TOKEN`.

In `.github/workflows/push.yml`.

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

      # go install kpm 
      - name: Install kpm
        run: go install kcl-lang.io/kpm@latest

      # kpm login and kpm push
      - name: Login and Push kpm project
      # Specify a custom OCI Registry through environment variables
        env:
          KPM_REG: "docker.io"
          KPM_REPO: ${{ vars.REG_ACCOUNT }}
        run: kpm login -u ${{ vars.REG_ACCOUNT }} -p ${{ secrets.REG_TOKEN }} docker.io && kpm push

      - name: Run kpm project
        run: kpm run oci://docker.io/${{ vars.REG_ACCOUNT }}/mykcl --tag 0.0.1
```
