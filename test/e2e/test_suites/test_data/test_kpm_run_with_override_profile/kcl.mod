[package]
name = "test_kpm_run_with_override"
edition = "0.0.1"
version = "0.0.1"

[profile]
overrides = [ "__main__:a2.image=\"new-a2-image:v123\"" ]