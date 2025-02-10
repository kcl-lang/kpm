[package]
name = "test_run_hyphen_entries"
edition = "v0.11.1"
version = "0.0.1"

[dependencies]
flask_manifests = { git = "https://github.com/kcl-lang/flask-demo-kcl-manifests.git", commit = "ade147b", version = "0.0.1" }

[profile]
entries = ["main.k", "${flask-manifests:KCL_MOD}/main.k"]
