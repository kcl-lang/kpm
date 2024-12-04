## Bug Report 

https://github.com/kcl-lang/kcl/issues/1768

The [documentation](https://www.kcl-lang.io/docs/user_docs/guides/package-management/how-to/kpm_oci#kcl-mod-push-to-upload-a-kcl-package) mentions a `--tag` flag for `kcl mod push` to overwrite the version written in `kcl.mod`.

However, it appears this flag doesn't exist.

### 1. Minimal reproduce step (Required)

Run `kcl mod push --tag 0.0.42`

### 2. What did you expect to see? (Required)

### 3. What did you see instead (Required)

```
unknown flag: --tag
```
### 4. What is your KCL components version? (Required)

kcl version: 0.10.10-linux-amd64
