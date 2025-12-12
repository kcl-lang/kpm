[package]
name = "test_add_order"
edition = "v0.12.3"
version = "0.0.1"

[dependencies]
helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.2" }
jsonpatch = { oci = "oci://ghcr.io/kcl-lang/jsonpatch", tag = "0.0.5" }

[profile]
entries = ["./sub/main.k"]
