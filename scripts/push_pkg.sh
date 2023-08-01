
#!/usr/bin/env bash

# Check kpm version

version=$(./bin/kpm --version)
if ! echo "$version" | grep -q "kpm version test_version"; then
  echo "Incorrect version: '$version'."
  exit 1
fi

export KPM_REG="localhost:5001"
export KPM_REPO="test"
export OCI_REG_PLAIN_HTTP=on

# Prepare the package on the registry
current_dir=$(pwd)
echo $current_dir

$current_dir/bin/kpm login -u test -p 1234 localhost:5001

# Push the package k8s/1.14 to the registry
cd ./scripts/pkg_in_reg/ghcr.io/kcl-lang/k8s/1.14
$current_dir/bin/kpm push

cd "$current_dir"

# Push the package k8s/1.17 to the registry
cd ./scripts/pkg_in_reg/ghcr.io/kcl-lang/k8s/1.27
$current_dir/bin/kpm push

cd "$current_dir"

# Push the package 'kcl1' depends on 'k8s' to the registry
cd ./scripts/pkg_in_reg/kcl1
$current_dir/bin/kpm push

cd "$current_dir"

# Push the package 'kcl2' depends on 'k8s' to the registry
cd ./scripts/pkg_in_reg/kcl2
$current_dir/bin/kpm push
