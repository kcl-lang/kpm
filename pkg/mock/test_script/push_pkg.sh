#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$(dirname "$(realpath "$0")")"

# Move to the root directory
cd "$SCRIPT_DIR/../../../"

# Build kpm binary
LDFLAGS="-X kcl-lang.io/kpm/pkg/version.version=test_version"
go build -ldflags "$LDFLAGS" -o ./bin/kpm

# Check kpm version
version=$(./bin/kpm --version)
if ! echo "$version" | grep -q "kpm version test_version"; then
  echo "Incorrect version: '$version'."
  exit 1
fi

export KPM_REG="localhost:5002"
export KPM_REPO="test"

# Prepare the package on the registry
current_dir=$(pwd)
echo $current_dir

# Log in to the local registry
"$current_dir/bin/kpm" login -u test -p 1234 localhost:5002

# Push the test_data package to the registry
cd "$SCRIPT_DIR/../test_data"
"$current_dir/bin/kpm" push oci://$KPM_REG/$KPM_REPO/test_data
