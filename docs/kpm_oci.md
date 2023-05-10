# Use OCI-based Registries

Beginning in kpm v0.2.0, you can use container registries with OCI support to store and share kcl packages.

## Quick Start with OCI Registry

### kpm login

login to a registry (with manual password entry)

```shell
$ kpm login -u <account_name> <oci_registry>
Password: # input your password here.
Login succeeded
```

- `<account_name>` is your registry account name.
- `<oci_registry>` is the resgistry url.

### kpm logout

logout from a registry

```shell
kpm logout <registry>
```

- `<registry>` is the resgistry url.

### kpm push

Upload a kcl package to an OCI-based registry.

```shell
kpm push oci://<oci_registry>/<account_name>/<repo_name>
```

- `<oci_registry>` is the oci registry url. e.g. `ghcr.io` or `docker.io`.
- `<account_name>` is your registry account name.
- `<repo_name>` is the repo name in the oci registry.

### kpm pull

Download a kcl package from an OCI-based registry.

```shell
kpm pull --tag <kcl_package_version>  oci://<oci_registry>/<account_name>/<repo_name>
```

- `<oci_registry>` is the oci registry url. e.g. `ghcr.io` or `docker.io`.
- `<account_name>` is your registry account name.
- `<repo_name>` is the repo name in the oci registry.
- `<kcl_package_version>` is the version of the kcl package.

### kpm run

You can directly run an oci url or ref to compile a package in oci registry.

```shell
kpm run --tag <kcl_package_version> oci://<oci_registry>/<account_name>/<repo_name>
```

- `<kcl_package_version>` is the version of the kcl package.
- `<oci_registry>` is the oci registry url. e.g. `ghcr.io` or `docker.io`.
- `<account_name>` is your registry account name.
- `<repo_name>` is the repo name in the oci registry.

If your kcl package is stored on `docker.io`, then you can run the package directly using oci ref.

```shell
kpm run <docker.io_account_name>/<repo_name>:<kcl_package_version>
```

- `<docker.io_account_name>` is your `docker.io` account name.
- `<repo_name>` is the repo name in the oci registry.
- `<kcl_package_version>` is the version of the kcl package.
