[package]
name = "test_add_deps"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
oci_name = { oci = "oci://test_reg/test_repo", tag = "test_tag", package = "oci_name" }
name = { git = "test_url", tag = "test_tag", package = "name" }
