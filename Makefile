.SILENT:

.PHONY: deps test pipeline-test build clean lint site-deps site-serve site-build site-versions release-notes

# The project site (site/) is built with MkDocs Material and published to GitHub
# Pages by .github/workflows/docs.yml. It is versioned: every release line is
# built from its tag into its own directory, latest/ mirrors the newest release
# and dev/ is this checkout (site/scripts/build_site.py). The virtualenv lives
# inside site/ and is gitignored; site-build and site-versions run exactly what
# CI runs — site-build for a pull request, site-versions for a push to master.
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

# The full versioned tree: every stable tag in its own directory, latest/, dev/,
# versions.json and the root redirect. The same script CI runs on a push.
site-versions: $(SITE_MKDOCS)
	python3 site/scripts/build_site.py all

# The changelog is generated, never edited: release notes come from conventional
# commits via git-cliff (cliff.toml). .github/workflows/release.yml runs the same
# command at dispatch time, with the tag of the version about to be published.
release-notes:
	@test -n "$(VERSION)" || { echo "usage: make release-notes VERSION=0.1.0"; exit 2; }
	git-cliff --config cliff.toml --tag v$(VERSION) --unreleased --strip all

$(SITE_MKDOCS):
	$(MAKE) site-deps
