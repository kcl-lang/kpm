[package]
name = "test_run_hyphen_entries"
edition = "v0.10.0"
version = "0.0.1"

[dependencies]
hello-world = { package = "helloworld", version = "0.1.4" }

[profile]
entries = ["main.k", "${hello-world:KCL_MOD}/main.k"]
