[package]
name = "test_add_git_dep"
edition = "v0.11.0"
version = "0.0.1"

[dependencies]
flask_manifests = { git = "git://github.com/kcl-lang/flask-demo-kcl-manifests.git", tag = "v0.1.0" }

