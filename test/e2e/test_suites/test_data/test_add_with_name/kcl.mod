[package]
name = "test_add_with_name"
edition = "v0.12.1"
version = "0.0.1"

[dependencies]
k8s = { oci = "oci://localhost:5002/test/k8s", tag = "1.27" }
