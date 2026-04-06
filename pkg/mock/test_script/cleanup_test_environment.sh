#!/bin/bash

# Determine the directory where this script is located
SCRIPT_DIR="$(dirname "$(realpath "$0")")"
ROOT_DIR="$(realpath "$SCRIPT_DIR/../../../")"

# Stop and remove the Docker container then remove the Docker image
docker stop kcl-registry
docker rm kcl-registry
docker rmi registry

# Delete all data stored in the Docker registry volume
rm -rf /var/lib/registry/*

# Remove generated registry authentication data.
rm -rf "$ROOT_DIR/scripts/registry_auth"

# Delete the 'kpm' binary
cd "$ROOT_DIR"
rm -rf ./bin/
