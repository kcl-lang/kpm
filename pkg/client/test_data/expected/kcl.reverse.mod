[package]
name = "test_add_deps"
edition = "v0.7.0"
version = "0.0.1"

[dependencies]
name = { git = "test_url", tag = "test_tag" }
oci_name = "test_tag"
