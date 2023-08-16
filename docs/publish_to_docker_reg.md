# How to Share Your Package using kpm on docker.io

[kpm](https://github.com/KusionStack/kpm) is a tool for managing kcl packages. This article will show you how to use kpm to push your kcl packages to docker.io.

Here is a simple step-by-step guide on how to use kpm to push your kcl package to docker.io.

## Step 1: Install kpm

First, you need to install kpm on your computer. You can follow the instructions in the [kpm installation documentation](https://kcl-lang.io/docs/user_docs/guides/package-management/installation).

## Step 2: Create a docker.io account

If you want to use `docker.io` as the OCI registry to share your kcl package with others, you need to create a `docker.io` account to support the push of your kcl package.

## Step 3: Tell kpm the registry and the repository

By default, kpm uses `ghcr.io` as the OCI registry. More information about how to use `ghcr.io` as the OCI registry to share your kcl package with others, see [How to Share Your KCL Packages with Others on ghcr.io using kpm](./publish_your_kcl_packages.md).

Therefore, if you want to use `docker.io`, you need to specify the registry and repository by setting the environment variables `KPM_REG` and `KPM_REPO`.

```shell
export KPM_REG=docker.io
export KPM_REPO=<USERNAME>
```

`<USERNAME>` is your `docker.io` username created in [Step 2](#step-2-create-a-dockerio-account).

## Step 4: Log in to docker.io

You can use the following command to log in to docker.io.

```shell
kpm login -u <USERNAME> -p <PASSWORD> docker.io
```

Where `<USERNAME>` is your `docker.io` username, and `<PASSWORD>` is your `docker.io` password.

For more information on how to log in to docker.io using kpm, see [kpm login](./kpm_oci.md#kpm-login).

## Step 5: Push your kcl package

Now, you can use kpm to push your kcl package to `docker.io`.

### 1. A valid kcl package

First, you need to make sure that what you are pushing conforms to the specifications of a kcl package, i.e., it must contain valid kcl.mod and kcl.mod.lock files.

If you don't know how to get a valid kcl.mod and kcl.mod.lock, you can use the `kpm init` command.

```shell
# Create a new kcl package named my_package
kpm init my_package
```

The `kpm init my_package` command will create a new kcl package `my_package` for you and create the `kcl.mod` and `kcl.mod.lock` files for this package.

If you already have a directory containing kcl files `exist_kcl_package`, you can use the following command to convert it into a kcl package and create valid `kcl.mod` and `kcl.mod.lock` files for it.

```shell
# In the exist_kcl_package directory
$ pwd 
/home/user/exist_kcl_package

# Run the `kpm init` command to create the `kcl.mod` and `kcl.mod.lock` files
$ kpm init 
```

For more information on how to use `kpm init`, see [kpm init](./command-reference/1.init.md).

### 2. Pushing the KCL Package

You can use the following command in the root directory of your `kcl` package:

```shell
# In the root directory of the exist_kcl_package package
$ pwd 
/home/user/exist_kcl_package

# Pushing the KCL Package to Default OCI Registry
$ kpm push
```

After completing these steps, you have successfully pushed your KCL Package to `docker.io`.
For more information on how to use `kpm push`, see [kpm push](./kpm_oci.md#kpm-push).
