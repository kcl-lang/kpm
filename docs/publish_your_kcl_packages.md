# How to Share Your Package using kpm

[kpm](https://github.com/KusionStack/kpm) is a tool for managing kcl packages. This article will guide you on how to use kpm to push your kcl package to an OCI Registry for publication. kpm uses [ghcr.io](https://ghcr.io) as the default OCI Registry, and you can change the default OCI Registry by modifying the kpm configuration file. For information on how to modify the kpm configuration file, see [kpm oci registry](./kpm_oci.md#kpm-registry)

Here is a simple step-by-step guide on how to use kpm to push your kcl package to ghcr.io.

## Step 1: Install kpm

First, you need to install kpm on your computer. You can follow the instructions in the [kpm installation documentation](https://kcl-lang.io/docs/user_docs/guides/package-management/installation).

## Step 2: Create a ghcr.io token

If you are using the default OCI Registry of kpm, to push a kcl package to ghcr.io, you need to create a token for authentication. You can follow the instructions in [ghcr.io authentication](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry).

## Step 3: Log in to ghcr.io

After installing kpm and creating a ghcr.io token, you need to log in to ghcr.io using kpm. You can do this using the following command:

```shell
kpm login ghcr.io -u <USERNAME> -p <TOKEN> <OCI_REGISTRY>
```

Where `<USERNAME>` is your GitHub username, `<TOKEN>` is the token you created in step 2, and `<OCI_REGISTRY>` is ghcr.io.

For more information on how to log in to ghcr.io using kpm, see [kpm login](./kpm_oci.md#kpm-login).

## Step 4: Push your kcl package

Now, you can use kpm to push your kcl package to ghcr.io.

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

After completing these steps, you have successfully pushed your KCL Package to the default OCI Registry.
For more information on how to use `kpm push`, see [kpm push](./kpm_oci.md#kpm-push).
