[package]
name = "aaa"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
bbb = { path = "../bbb" }
helloworld = { oci = "oci://localhost:5001/test/helloworld", tag = "0.1.2" }
k8s = { oci = "oci://localhost:5001/test/k8s", tag = "1.27" }

