VERSION := $(shell git describe --tags)
LDFLAGS := -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" kpm.go

COVER_FILE			?= coverage.out
SOURCE_PATHS		?= ./pkg/...

test:  ## Run the tests
	go test -gcflags=all=-l -timeout=20m `go list $(SOURCE_PATHS)` ${TEST_FLAGS} -v

cover:  ## Generates coverage report
	go test -gcflags=all=-l -timeout=20m `go list $(SOURCE_PATHS)` -coverprofile $(COVER_FILE) ${TEST_FLAGS} -v

e2e: ## Run e2e test
	scripts/e2e.sh
