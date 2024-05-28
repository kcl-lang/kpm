[package]
name = "test_kpm_run_with_git_commit_dep"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
flask-demo-kcl-manifests = { git = "https://github.com/kcl-lang/flask-demo-kcl-manifests.git", branch = "main" }
