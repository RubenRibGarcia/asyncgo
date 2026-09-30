.SILENT:

.PHONY: deps test pipeline-test build clean lint

deps:
	go mod tidy
	cd test/integration && GOWORK=off go mod tidy
	GOWORK=off go mod vendor

test:
	go tool gotestsum --format pkgname-and-test-fails -- ./... ./test/integration/... -race -coverprofile=coverage.out -covermode=atomic
	go tool go-test-coverage --config=.testcoverage.yaml

pipeline-test:
	go tool gotestsum --format pkgname-and-test-fails -- ./... ./test/integration/... -race -coverprofile=coverage.out -covermode=atomic

build:
	go build ./...

clean:
	go clean ./...

lint:
	go tool golangci-lint run ./... ./test/integration/...
	go tool golangci-lint fmt
