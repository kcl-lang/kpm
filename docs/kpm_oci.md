# Use OCI-based Registries

Beginning in kpm v0.2.0, you can use container registries with OCI support to store and share kcl packages.

## kpm registry

kpm supports using OCI registries to store and share kcl packages. kpm uses `ghcr.io` to store kcl packages by default.

kpm default registry - [https://github.com/orgs/KusionStack/packages](https://github.com/orgs/KusionStack/packages)

You can adjust the registry url and repo name in the kpm configuration file. The kpm configuration file is located at `$KCL_PKG_PATH/.kpm/config.json`, and if the environment variable `KCL_PKG_PATH` is not set, it is saved by default at `$HOME/.kcl/kpm/.kpm/config.json`.

The default content of the configuration file is as follows:

```json
{
    "DefaultOciRegistry":"ghcr.io",
    "DefaultOciRepo":"kcl-lang"
}
```

## Quick Start with OCI Registry

In the following sections, a temporary OCI registry was set up on `localhost:5001` in a local environment, and an account `test` with the password `1234` was added for this OCI registry.

### kpm login

You can use `kpm login` in the following four ways.

#### 1. login to a registry with account and password

```shell
$ kpm login -u <account_name> -p <password> <oci_registry>
Login succeeded
```

<img src="./gifs/kpm_login.gif" width="600" align="center" />

#### 2. login to a registry with account, and enter the password interactively.

```shell
$ kpm login -u <account_name> <oci_registry>
Password:
Login succeeded
```

<img src="./gifs/kpm_login_with_pwd.gif" width="600" align="center" />

#### 3. login to a registry, and enter the account and password interactively.

```shell
$ kpm login <oci_registry>
Username: <account_name>
Password:
Login succeeded
```

<img src="./gifs/kpm_login_with_both.gif" width="600" align="center" />

### kpm logout

You can use `kpm logout` to logout from a registry

```shell
kpm logout <registry>
```

<img src="./gifs/kpm_logout.gif" width="600" align="center" />

### kpm push

You can use `kpm push` under the kcl package root directory to upload a kcl package to an OCI-based registry.

```shell
# create a new kcl package.
$ kpm init <package_name> 
# enter the kcl package root directory
$ cd <package_name> 
# push it to the default oci registry specified in the configuration file 'kpm.json'.
$ kpm push
```

<img src="./gifs/kpm_push.gif" width="600" align="center" />

You can also use `kpm push` to upload a kcl package to an OCI-based registry by specifying the registry url.

```shell
# create a new kcl package.
$ kpm init <package_name> 
# enter the kcl package root directory
$ cd <package_name> 
# push it to an oci registry
$ kpm push <oci_url>
```

<img src="./gifs/kpm_push_with_url.gif" width="600" align="center" />

### kpm pull

You can use `kpm pull` to download a kcl package from the default OCI registry by kcl package name.
`kpm` will download the kcl package from the default OCI registry specified in the configuration file `kpm.json`.

```shell
kpm pull <package_name>:<package_version>
```

<img src="./gifs/kpm_pull.gif" width="600" align="center" />

Or you can download a kcl package from an OCI-based registry url.

```shell
kpm pull --tag <kcl_package_version> <oci_url>
```

<img src="./gifs/kpm_pull_with_url.gif" width="600" align="center" />

### kpm run

Run an oci url or ref to compile a package in oci registry.

```shell
kpm run --tag <kcl_package_version> <oci_url>
```

<img src="./gifs/kpm_run_oci_url.gif" width="600" align="center" />

Alternatively, you can compile a kcl package directly from an oci ref using `kpm run`.

```shell
kpm run <oci_ref>
```

<img src="./gifs/kpm_run_oci_ref.gif" width="600" align="center" />
