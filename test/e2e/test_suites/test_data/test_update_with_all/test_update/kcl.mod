[package]
name = "test_update"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
test_update_1 = { path = "./../test_update_1" }
helloworld = "0.1.1"
catalog = { git = "https://github.com/KusionStack/catalog.git", tag = "0.1.2" }
