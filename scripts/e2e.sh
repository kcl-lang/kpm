#!/usr/bin/env bash

# start registry at 'localhost:5001'
# include account 'test' and password '1234'
./scripts/reg.sh

# set the kpm default registry and repository
export KPM_REG="localhost:5001"
export KPM_REPO="test"
export OCI_REG_PLAIN_HTTP=on

set -o errexit
set -o nounset
set -o pipefail

# Install ginkgo
GO111MODULE=on go install github.com/onsi/ginkgo/v2/ginkgo@v2.0.0

# Prepare e2e test env
./scripts/e2e_prepare.sh

# Run e2e
set +e
ginkgo  ./test/e2e/ 
TESTING_RESULT=$?


exit $TESTING_RESULT
