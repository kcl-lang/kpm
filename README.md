<h1 align="center">Kpm: KCL Package Manager</h1>

<p align="center">
<a href="./README.md">English</a> | <a href="./README-zh.md">简体中文</a>
</p>
<p align="center">
<a href="#introduction">Introduction</a> | <a href="#installation">Installation</a> | <a href="#quick-start">Quick start</a> 
</p>

<p align="center">
<img src="https://coveralls.io/repos/github/KusionStack/kpm/badge.svg">
<img src="https://img.shields.io/badge/license-Apache--2.0-green">
<img src="https://img.shields.io/badge/PRs-welcome-brightgreen">
</p>

## Introduction

`kpm` is the KCL package manager and it is integrated in the `kcl mod` command.

## Contributing

See [contribution guideline](https://kcl-lang.io/docs/community/contribute/).

## Learn More

See [here](https://www.kcl-lang.io/docs/user_docs/guides/package-management/quick-start) for more documents.

## Authentication

`kpm login` supports username/password (`--provider=basic`, the default)
and GCP Workload Identity (`--provider=gcp`) for passwordless login from
GKE pods. See [docs/gcp-workload-identity.md](./docs/gcp-workload-identity.md).
