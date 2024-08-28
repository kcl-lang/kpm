[package]
name = "my_package"
edition = "v0.8.0"
version = "0.0.1"

[dependencies]
helloworld = { git = "https://github.com/kcl-lang/modules.git", commit = "ee03122b5f45b09eb48694422fc99a0772f6bba8", package = "helloworld" }
