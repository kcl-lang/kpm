build:
	go build kpm.go

COVER_FILE			?= coverage.out
SOURCE_PATHS		?= ./pkg/...

test:  ## Run the tests
	go test -gcflags=all=-l -timeout=10m `go list $(SOURCE_PATHS)` ${TEST_FLAGS} -v

cover:  ## Generates coverage report
	go test -gcflags=all=-l -timeout=10m `go list $(SOURCE_PATHS)` -coverprofile $(COVER_FILE) ${TEST_FLAGS} -v
