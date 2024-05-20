VERSION := $(shell git describe --tags)
LDFLAGS := -X kcl-lang.io/kpm/pkg/version.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" kpm.go

COVER_FILE			?= coverage.out
SOURCE_PATHS		?= ./pkg/...

test: ## Run unit tests
	go test -gcflags=all=-l -timeout=20m `go list $(SOURCE_PATHS)` ${TEST_FLAGS} -v

cover:  ## Generates coverage report
	go test -gcflags=all=-l -timeout=20m `go list $(SOURCE_PATHS)` -coverprofile $(COVER_FILE) ${TEST_FLAGS} -v

e2e: ## Run e2e test
	scripts/e2e.sh

fmt:
	go fmt ./...
