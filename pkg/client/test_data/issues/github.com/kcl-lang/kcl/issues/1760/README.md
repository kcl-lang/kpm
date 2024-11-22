## Bug Report

https://github.com/kcl-lang/kcl/issues/1760

### 1. Minimal reproduce step (Required)

1. Two empty modules within the same directory:

|- modules
|   |- a
|   |   |- kcl.mod
|   |   |- main.k
|   |- b
|   |   |- kcl.mod
|   |   |- main.k

2. Content `/modules/a/kcl.mod` :

```
[package]
name = "a"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
b = { path = "../b" }
fluxcd-source-controller = "v1.3.2"
fluxcd-helm-controller = "v1.0.3"
```

3. Content `/modules/b/kcl.mod` :

```
[package]
name = "b"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
fluxcd-source-controller = "v1.3.2"
```

4. Ensure your `$USER/.kcl/kpm/` directory is empty.


### 2. What did you expect to see? (Required)
```
$ kcl run /modules/a/main.k
downloading 'kcl-lang/fluxcd-helm-controller:v1.0.3' from 'ghcr.io/kcl-lang/fluxcd-helm-controller:v1.0.3'
downloading 'kcl-lang/fluxcd-source-controller:v1.3.2' from 'ghcr.io/kcl-lang/fluxcd-source-controller:v1.3.2'
downloading 'kcl-lang/k8s:1.31.2' from 'ghcr.io/kcl-lang/k8s:1.31.2'
The_first_kcl_program: Hello World!
```

### 3. What did you see instead (Required)
```
$ kcl run /modules/a/main.k
downloading 'kcl-lang/fluxcd-helm-controller:v1.0.3' from 'ghcr.io/kcl-lang/fluxcd-helm-controller:v1.0.3'
downloading 'kcl-lang/fluxcd-source-controller:v1.3.2' from 'ghcr.io/kcl-lang/fluxcd-source-controller:v1.3.2'
downloading 'kcl-lang/k8s:1.31.2' from 'ghcr.io/kcl-lang/k8s:1.31.2'
edge already exists
```

--> **kcl exited with error code 1**

### 4. What is your KCL components version? (Required)

0.10.8-linux-amd64
