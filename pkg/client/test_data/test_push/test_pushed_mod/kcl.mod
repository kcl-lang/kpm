[package]
name = "test_pushed_mod"
edition = "v0.11.2"
version = "0.0.1"

[dependencies]
push_0 = { oci = "oci://localhost:5002/test/push_0", tag = "0.0.1", version = "0.0.1" }
