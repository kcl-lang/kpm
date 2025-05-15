#!/bin/bash

# Build kpm binary
LDFLAGS="-X kcl-lang.io/kpm/pkg/version.version=test_version"
go build -ldflags "$LDFLAGS" -o ./bin/kpm

# Check kpm version
version=$(./bin/kpm --version)
if ! echo "$version" | grep -q "kpm version test_version"; then
  echo "Incorrect version: '$version'."
  exit 1
fi

# pull the package 'k8s' from 'ghcr.io/kcl-lang/k8s'
./scripts/pull_pkg.sh

# push the package 'k8s' to 'localhost:5002/test'
./scripts/push_pkg.sh
