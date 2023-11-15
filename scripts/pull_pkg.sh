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


# Check if file exists
if [ ! -d "./ghcr.io/kcl-lang/k8s/1.14" ]; then
  $current_dir/bin/kpm pull k8s:1.14
fi

if [ ! -d "./ghcr.io/kcl-lang/k8s/1.27" ]; then
  $current_dir/bin/kpm pull k8s:1.27
fi

if [ ! -d "./ghcr.io/kcl-lang/helloworld" ]; then
  $current_dir/bin/kpm pull helloworld
fi

cd "$current_dir"
