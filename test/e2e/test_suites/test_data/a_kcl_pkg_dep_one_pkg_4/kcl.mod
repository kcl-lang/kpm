[package]
name = "a_kcl_pkg_dep_pkg_name"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
kcl2 = "0.0.1"
a_kcl_pkg_dep_pkg_name = { path = "../a_kcl_pkg_dep_one_pkg_3" }
kcl1 = "0.0.1"
k8s = "1.27"
