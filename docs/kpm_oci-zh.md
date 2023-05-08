# 使用 OCI Registries

从 kpm v0.2.0 版本开始，kpm 支持通过 OCI Registries 保存和分享 KCL 包。

## 快速开始在 kpm 中使用 OCI Registry

### kpm login

使用 kpm 登陆 OCI registry，需要手动输入密码。

```shell
$ kpm login -u <account_name> <oci_registry>
Password: # 手动输入密码。
Login succeeded
```

- `<account_name>` 是用来登陆 OCI registry 的账户名称。
- `<oci_registry>` 是 OCI registry 的 URL，例如：'docker.io', 'ghcr.io'等。

### kpm logout

使用 kpm 登出 OCI registry

```shell
kpm logout <registry>
```

- `<oci_registry>` 是 OCI registry 的 URL，例如：'docker.io', 'ghcr.io'等。

### kpm push

向 OCI registry 中上传一个 kcl 包。

```shell
kpm push oci://<oci_registry>/<account_name>/<repo_name>
```

- `<oci_registry>` 是 OCI registry 的 URL，例如：'docker.io', 'ghcr.io'等。
- `<account_name>` 是用来登陆 OCI registry 的账户名称。
- `<repo_name>` 是用来保存 kcl 包的仓库名称。

### kpm pull

从 OCI registry 中下载一个 kcl 包。

```shell
kpm pull --tag <kcl_package_version>  oci://<oci_registry>/<account_name>/<repo_name>
```

- `<oci_registry>` 是 OCI registry 的 URL，例如：'docker.io', 'ghcr.io'等。
- `<account_name>` 是用来登陆 OCI registry 的账户名称。
- `<repo_name>` 是用来保存 kcl 包的仓库名称。
- `<kcl_package_version>` kcl 包的版本号，例如：v0.0.1。

### kpm run

kpm 可以直接通过 OCI 的 url 编译 kcl 包。

```shell
kpm run --tag <kcl_package_version> oci://<oci_registry>/<account_name>/<repo_name>
```

- `<oci_registry>` 是 OCI registry 的 URL，例如：'docker.io', 'ghcr.io'等。
- `<account_name>` 是用来登陆 OCI registry 的账户名称。
- `<repo_name>` 是用来保存 kcl 包的仓库名称。
- `<kcl_package_version>` kcl 包的版本号，例如：v0.0.1。

如果你的 kcl 是保存在 docker.io 上的，那么 kpm 还支持通过如下方式直接编译 kcl 包。

```shell
kpm run <docker.io_account_name>/<repo_name>:<kcl_package_version>
```

- `<docker.io_account_name>` 'docker.io'的账户名称。
- `<repo_name>` 是用来保存 kcl 包的仓库名称。
- `<kcl_package_version>` kcl 包的版本号，例如：v0.0.1。
