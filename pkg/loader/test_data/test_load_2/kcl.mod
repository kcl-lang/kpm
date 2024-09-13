[package]
name = "test_load_2"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.1" }
