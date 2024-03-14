[package]
name = "test_add_deps"
edition = "v0.8.0"
version = "0.0.1"

[dependencies]
oci_name = "test_tag"
name = { git = "test_url", tag = "test_tag" }
