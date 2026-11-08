.SILENT:

.PHONY: deps test pipeline-test build clean lint

deps:
	go mod tidy
	cd test/integration && GOWORK=off go mod tidy
	GOWORK=off go mod vendor

# test is pipeline-test plus the coverage gates: CI runs pipeline-test on its own
# so a coverage-threshold failure is a separate signal from a test failure.
test: pipeline-test
	go tool go-test-coverage --config=.testcoverage.yaml

pipeline-test:
	go tool gotestsum --format pkgname-and-test-fails -- ./... ./test/integration/... -race -coverprofile=coverage.out -covermode=atomic

build:
	go build ./...

clean:
	go clean ./...
	rm -f coverage.out

lint:
	go tool golangci-lint run ./... ./test/integration/...
	go tool golangci-lint fmt
