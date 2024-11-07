[package]
name = "rename"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
newpkg = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.4", package = "subhelloworld", version = "0.0.1" }
