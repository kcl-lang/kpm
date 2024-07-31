[package]
name = "test"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
dep1 = { oci = "oci://ghcr.io/kcl-lang/dep1", tag = "0.0.1", package = "dep1" }

