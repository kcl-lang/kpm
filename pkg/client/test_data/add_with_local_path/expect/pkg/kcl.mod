[package]
name = "pkg"
edition = "v0.12.1"
version = "0.0.1"

[dependencies]
dep_pkg = { path = "../dep_pkg" }
helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.2" }
