[package]
name = "test_kpm_run_with_git_commit_dep"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
catalog = { git = "https://github.com/KusionStack/catalog.git", commit = "3891e96" }
