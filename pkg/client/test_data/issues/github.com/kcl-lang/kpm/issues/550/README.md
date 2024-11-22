# bug: path information is lost in kcl mod metadata


## 1. Minimal reproduce step (Required)

```shell
kcl mod init demo
cd demo
kcl mod add k8s --git https://github.com/kcl-lang/modules
kcl mod add k8s --rename k8soci
kcl mod add json_merge_patch --git https://github.com/kcl-lang/modules --tag v0.1.0
kcl mod metadata
```

## 2. What did you expect to see? (Required)

Show the metadata message with local storage path.

```shell
{"packages":{"json_merge_patch":{"name":"json_merge_patch","manifest_path":"<real_path>"},"k8s":{"name":"k8s","manifest_path":"<real_path>"},"k8soci":{"name":"k8soci","manifest_path":"<real_path>"}}}
```

## 3. What did you see instead (Required)

The real path is empty.

```shell
{"packages":{"json_merge_patch":{"name":"json_merge_patch","manifest_path":""},"k8s":{"name":"k8s","manifest_path":""},"k8soci":{"name":"k8soci","manifest_path":""}}}
```

## 4. What is your KCL components version? (Required)

v0.11.0-alpha.1
