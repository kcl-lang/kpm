[package]
name = "test_add_deps"
edition = "v0.11.0"
version = "0.0.1"

[dependencies]
name = { git = "test_url", tag = "test_tag" }
oci_name = { oci = "oci://test_reg/test_repo", tag = "test_tag" }
