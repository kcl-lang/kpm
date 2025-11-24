[package]
name = "a"
edition = "v0.12.0-rc.1"
version = "0.0.1"

[dependencies]
b = { path = "../b", version = "0.0.1" }
fluxcd-helm-controller = "v1.0.3"
fluxcd-source-controller = "v1.3.2"
