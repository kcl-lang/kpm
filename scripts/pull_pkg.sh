#!/usr/bin/env bash

# Check kpm version
version=$(./bin/kpm --version)
if ! echo "$version" | grep -q "kpm version test_version"; then
  echo "Incorrect version: '$version'."
  exit 1
fi

# set the kpm default registry and repository
export KPM_REG="ghcr.io"
export KPM_REPO="kcl-lang"
export OCI_REG_PLAIN_HTTP=off

current_dir=$(pwd)

mkdir -p ./scripts/pkg_in_reg/

cd ./scripts/pkg_in_reg/

$current_dir/bin/kpm pull k8s:1.14
$current_dir/bin/kpm pull k8s:1.27

cd "$current_dir"
