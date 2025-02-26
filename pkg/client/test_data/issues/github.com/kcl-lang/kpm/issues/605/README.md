# bug: kcl cli slow because of downloading checksums

https://github.com/kcl-lang/kpm/issues/605

## 1. Minimal reproduce step (Required)

```shell
kcl mod init demo
cd demo
```

The `kcl.mod`:

```toml
[package]
name = "claims"
edition = "v0.11.0"
version = "0.0.1"

[dependencies]
crossplane = "1.17.3"
json_merge_patch = "0.1.1"
```

The `kcl.mod.lock`

```toml
[dependencies]
  [dependencies.crossplane]
    name = "crossplane"
    full_name = "crossplane_1.17.3"
    version = "1.17.3"
    sum = "8hx/1t6Gn0UZjaycAJIJ2Xj1R7vefZq7mMpAim21COc="
    reg = "ghcr.io"
    repo = "kcl-lang/crossplane"
    oci_tag = "1.17.3"
  [dependencies.json_merge_patch]
    name = "json_merge_patch"
    full_name = "json_merge_patch_0.1.1"
    version = "0.1.1"
    sum = "o1aamShk1L2MGjnN9u3IErRZ3xBNDxgmFxXsGVMt8Wk="
    reg = "ghcr.io"
    repo = "kcl-lang/json_merge_patch"
    oci_tag = "0.1.1"
  [dependencies.k8s]
    name = "k8s"
    full_name = "k8s_1.31.2"
    version = "1.31.2"
    sum = "xBZgPsnpVVyWBpahuPQHReeRx28eUHGFoaPeqbct+vs="
    reg = "ghcr.io"
    repo = "kcl-lang/k8s"
    oci_tag = "1.31.2"
```

Compile the current project and record the time:

```
time kcl run
```

## 2. What did you expect to see? (Required)

```
$ time kcl run
The_first_kcl_program: Hello World!
kcl run  0.03s user 0.04s system 10% cpu 0.648 total
$ time kcl run
The_first_kcl_program: Hello World!
kcl run  0.03s user 0.01s system 114% cpu 0.037 total
```

## 3. What did you see instead (Required)

```
$ time kcl run
The_first_kcl_program: Hello World!
kcl run  0.09s user 0.08s system 2% cpu 6.023 total
$ time kcl run
The_first_kcl_program: Hello World!
kcl run  0.07s user 0.05s system 1% cpu 7.077 total
```

## 4. What is your KCL components version? (Required)

the main branch
