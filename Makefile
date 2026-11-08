.SILENT:

.PHONY: deps test pipeline-test build clean lint site-deps site-serve site-build

# The project site (site/) is built with MkDocs Material and published to GitHub
# Pages by .github/workflows/docs.yml. The virtualenv lives inside site/ and is
# gitignored; site-build runs exactly what CI runs.
SITE_VENV := site/.venv
SITE_MKDOCS := $(SITE_VENV)/bin/mkdocs

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

site-deps:
	python3 -m venv $(SITE_VENV)
	$(SITE_VENV)/bin/pip install --upgrade pip
	$(SITE_VENV)/bin/pip install -r site/requirements.txt

site-serve: $(SITE_MKDOCS)
	cd site && .venv/bin/mkdocs serve

site-build: $(SITE_MKDOCS)
	cd site && .venv/bin/mkdocs build --strict

$(SITE_MKDOCS):
	$(MAKE) site-deps
