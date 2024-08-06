# Contributing to KPM 

Thank you for considering contributing to KPM! We welcome contributions in various forms, including bug reports, feature requests, code contributions, and documentation improvements. Please read through this guide to get started.

## Table of Contents

- [Introduction](#introduction)
- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Contributing Code](#contributing-code)
- [Local Setup and Usage](#local-setup-and-usage)
  - [Cloning the Repository](#cloning-the-repository)
  - [Building the Project](#building-the-project)
  - [Running the Project](#running-the-project)

## Introduction

KPM is the KCL package manager integrated with the `kcl mod` command. This document outlines the process to contribute to KPM and helps ensure that the community remains welcoming and productive.

## Code of Conduct

Please read and follow our [Code of Conduct](https://github.com/kcl-lang/kcl/blob/main/CODE_OF_CONDUCT.md) to ensure a welcoming environment for all contributors.

## How to Contribute to Code

### Reporting Bugs

If you find a bug, please [open an issue](https://github.com/kcl-lang/kpm/issues) and provide as much detail as possible. Include steps to reproduce the bug, your environment details, and any relevant logs.

### Suggesting Enhancements

We welcome suggestions for new features or improvements! Please [open an issue](https://github.com/kcl-lang/kpm/issues) to discuss your ideas before implementing them to ensure they align with the project’s goals.

For comprehensive instructions on how to contribute, including best practices and additional resources, please refer to the official contribution guide [here](https://www.kcl-lang.io/docs/community/contribute/).

## Local Setup and Usage

### Cloning the Repository

1. Fork the KPM repository on GitHub.
2. Clone your forked repository to your local machine:
   ```
   git clone https://github.com/your-username/kpm.git
   cd kpm
   ```

### Building the Project

1. Ensure you have [Go](https://golang.org/dl/) installed.
2. Install dependencies:
   ```
   go mod tidy
   ```
3. Build the project:
   ```
   make build
   ```

### Running the Project

1. Run the built executable:
   ```
   ./kpm
   ```
2. To see available commands and options, use:
   ```
   ./kpm --help
   ```

### Submitting Changes

1. Create a new branch for your changes: `git checkout -b my-feature`
2. Make your changes and commit them with a clear and concise commit message.
3. Push your changes to your fork: `git push origin my-feature`
4. Open a pull request on the main repository.
