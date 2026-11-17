# Design: A tag-and-release workflow without GoReleaser

- **Status**: Accepted
- **Created**: 2026-11-16
- **Status updated**: 2026-11-16
- **Scope**: `.github/workflows/release.yml`, `.goreleaser.yaml`, `cliff.toml`,
  `site/hooks/changelog.py`, `site/mkdocs.yml`, `.github/workflows/docs.yml`,
  `Makefile`, `mise.toml`, `AGENT.md`, `site/content/getting-started.md`

## Summary

Releasing currently means pushing a `v*` tag and letting GoReleaser build six
cross-compiled archives, SBOMs, checksums and attestations. None of that is
load-bearing: the CLI is installed with `go install`, and the archives have never
been published (no `v*` tag exists on `origin`). This design replaces the whole
pipeline with a `workflow_dispatch` release that takes an explicit version,
renders the changelog with [git-cliff](https://git-cliff.org), creates the
annotated tag and the GitHub release, and rebuilds the site so the release notes
appear as a Changelog page.

## Background

What exists today:

| Piece | Current state |
| --- | --- |
| Trigger | `push` on `v*` tags (`.github/workflows/release.yml`) |
| Builder | GoReleaser v2: linux/darwin/windows × amd64/arm64 archives, `tar.gz`/`zip`, checksums, source tarball |
| Extras | Syft SBOM per archive, `actions/attest` on checksums and digests |
| Changelog | GoReleaser's built-in generator, filtered by regexes in `.goreleaser.yaml`, used only as the release body |
| Tag history | `git ls-remote --tags origin` is **empty**; two releases exist as drafts (`v0.0.1`, `v0.1.0`) |
| Site | MkDocs Material in `site/`, published to Pages by `docs.yml` from `master`; `site/hooks/assemble.py` already generates pages at build time from repository files |
| Commits | Conventional, squash-merged: `feat(internal): … (#63)`, `fix(schema): …`, `chore(deps): …` |

Facts the design depends on:

1. `go install github.com/RubenRibGarcia/asyncgo/cmd/asyncgo@vX.Y.Z` already
   works and `internal/cli/version.go` already falls back to
   `debug.ReadBuildInfo()` for the module version. No `-ldflags` stamping is
   required for a correct `asyncgo version`.
2. A tag or commit pushed with `GITHUB_TOKEN` **does not** trigger other
   workflows. Any workflow that must run after the release has to be dispatched
   explicitly — and equally, nothing is silently re-entered by the tag push.
3. `docs.yml` already accepts `workflow_dispatch`, and `workflow_dispatch` can be
   triggered by another workflow holding `actions: write`.
4. `--strict` means every generated page must be in `nav`, and every nav entry
   must exist by the time MkDocs validates the nav. `assemble.py` shows the
   working recipe: write the page in `on_config`, inject the nav entry, rewrite
   only when content changed so `mkdocs serve` does not loop.
5. GitHub attaches `tar.gz` and `zip` source archives to every release for free;
   a "source-only" release has nothing to upload.

## Goals / non-goals

**Goals**

1. One dispatch, one input: `version` (plus `dry_run`). No tag push, no local
   checkout, no GoReleaser.
2. The tag and the GitHub release are created by the workflow itself, from
   `master`, only if the tree for that commit is green.
3. The release body is a human changelog generated from conventional commits by
   git-cliff, with a config in the repository that anyone can run locally.
4. The changelog is readable on the project site at
   `https://rubenribgarcia.github.io/asyncgo/changelog/`, sourced from GitHub
   Releases so there is exactly one place to correct an entry.
5. No new always-on toolchain: git-cliff is a pinned binary in CI and an
   optional `mise` tool locally.

**Non-goals (v1)**

- Prebuilt binaries, SBOMs, checksums and attestations. Deferred until someone
  installs the CLI without a Go toolchain; the design keeps the door open
  (see [Rollout](#rollout-plan), stage 5).
- A committed `CHANGELOG.md`. GitHub Releases is the source of truth; a file can
  be generated from the same config later without redesign.
- Automatic version computation from commit types (semantic-release style).
  The version is an explicit human decision.
- Signing, notarization, Homebrew tap.

## Design decisions

### D1 — Release is a `workflow_dispatch`, not a tag push

The only trigger is `workflow_dispatch` with `version` (semver without the `v`,
e.g. `0.1.0`) and an optional `dry_run` boolean. `on: push: tags` is removed.

**Rationale**: A tag push trigger makes two things impossible to express — "tag
this, but only if the tests on that exact commit passed" and "render the notes I
am about to publish". Dispatch also removes the two-step ritual from `AGENT.md` § Release
(`git tag` then `git push`), which is the
only place a release can go wrong silently. `dry_run` gives the reviewer the
exact text before anything irreversible happens.

### D2 — The workflow creates the annotated tag and the release; nothing is built

Steps: validate → require a green test run → render notes → `git tag -a` →
`git push origin "$TAG"` → `gh release create --verify-tag --notes-file`. No
build job, no `dist/`.

**Rationale**: Source-only releases have no artifacts, so the tag *is* the
release. `--verify-tag` makes the release fail rather than implicitly create a
tag at the wrong ref. GitHub's automatic source archives remove the need for
GoReleaser's `source` block.

### D3 — git-cliff, configured in `cliff.toml`, renders the notes

One command, in CI and locally:

```bash
git cliff --config cliff.toml --tag "v$VERSION" --unreleased \
  --strip all --output release-notes.md
```

**Rationale**: GoReleaser's changelog was a set of exclusion regexes; git-cliff
expresses the same intent as data (`commit_parsers`, groups, `protect_breaking_commits`)
and is the only part of GoReleaser worth keeping. `--unreleased` renders the
commits since the last tag; `--tag` labels them with the version about to be
published, so the output is the release body verbatim, produced *before* the tag
exists (which is what `dry_run` needs). `--strip all` drops the static
`# Changelog` header and footer so the body starts at the version heading.

### D4 — GitHub Releases is the changelog source of truth; the site fetches it

A new MkDocs hook, `site/hooks/changelog.py`, writes
`site/content/changelog.md` at build time from
`GET /repos/RubenRibGarcia/asyncgo/releases`, newest first, drafts excluded.

**Rationale**: There is exactly one editable copy of an entry — the release body
(`gh release edit vX.Y.Z --notes-file` fixes a typo everywhere). The hook, not a
committed file, means the site cannot drift from the published releases, and a
release needs no second commit. The trade-off — the site build now needs the
network — is contained by a cache plus a stub fallback (see
[Edge cases](#edge-cases)).

Rejected alternatives:

| Alternative | Why not |
| --- | --- |
| Keep GoReleaser | Six archives nobody downloads, plus Syft and two attest steps, for a CLI installed with `go install` |
| `semantic-release` / `release-please` | Adds a Node runtime and a bot that infers versions; the version should be a human decision, and the repo has no release cadence yet |
| `gh release create --generate-notes` + `.github/release.yml` | Zero config, but the body format is GitHub's PR-labelled grouping, not conventional commits, and it cannot be previewed before publishing |
| Small Go tool in `tools/` | Fits the `go tool` convention, but reimplements commit parsing, templating and breaking-change detection that git-cliff already tests |
| Committed `CHANGELOG.md` written by the release workflow | A second source of truth; the bot commit needs a push to `master` and a manual `docs.yml` dispatch to reach the site — same dispatch, more state |
| Site renders from `git log` at build time | Would require `fetch-depth: 0` in `docs.yml` and duplicates the changelog rules in Python |

### D5 — The release dispatches the site build

After the release exists, the workflow runs `gh workflow run docs.yml --ref master`
with `actions: write`.

**Rationale**: Nothing else would rebuild the site after a release — the release
touches no path in `docs.yml`'s `on.push.paths`, and a `GITHUB_TOKEN` push would
not start it anyway. Dispatching after the release (not before) is what makes the
new tag visible to the hook.

### D6 — Publish only from a commit whose tests passed

Before tagging, the workflow asserts that the newest `test.yaml` run for
`master` has `headSha == HEAD` and `conclusion == success`, and that
`HEAD == origin/master`.

**Rationale**: The tag-push trigger used to mean "whatever is on the branch gets
published" with no verification. A three-line check is cheaper than duplicating
`make lint && make test` inside the release job, and it also catches a stale
dispatch (branch moved between clicking and running).

## Detailed design

### 1. `.github/workflows/release.yml` (rewritten)

```yaml
name: Release

# A release is a version input, not a tag push: this workflow validates the
# tree, renders the notes with git-cliff, creates the annotated tag and the
# GitHub release, then rebuilds the site so the Changelog page picks it up.
#
# Note: a tag pushed with GITHUB_TOKEN does not trigger other workflows. That is
# why docs.yml is dispatched explicitly at the end.

on:
  workflow_dispatch:
    inputs:
      version:
        description: "Version to release, without the leading v (e.g. 0.1.0)"
        required: true
        type: string
      dry_run:
        description: "Render and print the notes, but do not tag or publish"
        required: false
        type: boolean
        default: false

permissions:
  contents: write   # create the tag and the release
  actions: write    # dispatch docs.yml

concurrency:
  group: release
  cancel-in-progress: false

env:
  VERSION: ${{ inputs.version }}
  TAG: v${{ inputs.version }}

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout master
        uses: actions/checkout@v7
        with:
          ref: master
          fetch-depth: 0        # git-cliff needs the full tag and commit history

      - name: Validate the dispatch
        id: validate
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
            echo "::error::'$VERSION' is not MAJOR.MINOR.PATCH[-PRE]"; exit 1
          fi
          if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/master)" ]]; then
            echo "::error::master moved during the dispatch; re-run"; exit 1
          fi
          # A retry after a half-finished publish may legitimately find its own
          # tag: accept it only when it points at HEAD.
          if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
            if [[ "$(git rev-parse "$TAG^{commit}")" != "$(git rev-parse HEAD)" ]]; then
              echo "::error::$TAG already exists and does not point at HEAD"; exit 1
            fi
            echo "tag_exists=true" >> "$GITHUB_OUTPUT"
          fi
          if gh release view "$TAG" >/dev/null 2>&1; then
            echo "::error::release $TAG already exists — edit it instead"; exit 1
          fi

      - name: Require a green test run for this commit
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          run=$(gh run list --workflow=test.yaml --branch=master --limit=1 \
                  --json headSha,conclusion --jq '.[0] | "\(.headSha) \(.conclusion)"')
          echo "latest test.yaml: $run"
          [[ "$run" == "$(git rev-parse HEAD) success" ]]

      - name: Install git-cliff
        uses: taiki-e/install-action@v2
        with:
          tool: git-cliff@2.14.2

      - name: Render release notes
        run: |
          set -euo pipefail
          git cliff --config cliff.toml --tag "$TAG" --unreleased \
            --strip all --output release-notes.md
          cat release-notes.md
          grep -q '^### ' release-notes.md || {
            echo "::error::nothing releasable since the last tag"; exit 1; }

      - name: Tag and publish
        if: ${{ !inputs.dry_run }}
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          if [[ -z "$(git rev-parse -q --verify "refs/tags/$TAG" || true)" ]]; then
            git config user.name  "github-actions[bot]"
            git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
            git tag -a "$TAG" -m "asyncgo $TAG"
            git push origin "$TAG"
          fi
          flags=()
          [[ "$VERSION" == *-* ]] && flags+=(--prerelease)
          gh release create "$TAG" --title "$TAG" --verify-tag \
            --notes-file release-notes.md "${flags[@]}"

      - name: Rebuild the site
        if: ${{ !inputs.dry_run }}
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: gh workflow run docs.yml --ref master
```

`dry_run` still runs validation, the green-tree check and the render, and prints
the notes in the job log — enough to review the exact text before the real run.

### 2. `cliff.toml` (new, repository root)

```toml
# Changelog for asyncgo, rendered from conventional commits by git-cliff.
#   local preview:  make release-notes VERSION=0.1.0
#   in CI:          .github/workflows/release.yml (release body and site page)
#
# The rules below are the ones .goreleaser.yaml used to apply as regexes: only
# feat / fix / perf / refactor / revert reach a release, everything else is
# developer noise.

[changelog]
header = ""
body = """
{% if version %}\
## [{{ version | trim_start_matches(pat="v") }}](https://github.com/RubenRibGarcia/asyncgo/releases/tag/v{{ version | trim_start_matches(pat="v") }}) - {{ timestamp | date(format="%Y-%m-%d") }}
{% else %}\
## Unreleased
{% endif %}\
{% for group, commits in commits | group_by(attribute="group") %}
### {{ group | striptags | trim | upper_first }}
{% for commit in commits %}
- {% if commit.scope %}**{{ commit.scope }}:** {% endif %}\
{% if commit.breaking %}**breaking** — {% endif %}\
{{ commit.message | upper_first }} ([{{ commit.id | truncate(length=7, end="") }}](https://github.com/RubenRibGarcia/asyncgo/commit/{{ commit.id }}))
{% endfor %}
{% endfor %}
"""
footer = ""

[git]
conventional_commits = true
filter_unconventional = true
split_commits = false
protect_breaking_commits = true
tag_pattern = "v[0-9]*"
sort_commits = "oldest"

commit_parsers = [
  { message = "^feat", group = "Added" },
  { message = "^fix", group = "Fixed" },
  { message = "^perf", group = "Performance" },
  { message = "^refactor", group = "Changed" },
  { message = "^revert", group = "Reverted" },
  { message = "^docs", skip = true },
  { message = "^test", skip = true },
  { message = "^ci", skip = true },
  { message = "^build", skip = true },
  { message = "^chore", skip = true },
  { message = "^style", skip = true },
]
```

Commit IDs render as links so an entry is traceable without opening the release
page. `filter_unconventional = true` drops the merge commits and any stray
non-conventional message, which is how `.goreleaser.yaml`'s
`Merge pull request` / `Merge branch` excludes disappear.

### 3. `site/hooks/changelog.py` (new) and `site/mkdocs.yml`

The hook mirrors `assemble.py`: generate in `on_config`, write only when the
content changed, and fix the page's edit link.

```python
"""Publish the GitHub Releases feed as site/content/changelog.md.

GitHub Releases is the changelog's single source of truth: a released tag's
notes are written by the release workflow (git-cliff), corrected with
`gh release edit`, and rendered here. The page is a build artifact, so it is
gitignored like the Project section's copies.
"""
# The host is a literal on purpose: HTTPSConnection enforces TLS, and only the
# path can be aimed elsewhere (at a 404) to exercise the fallbacks.
API_HOST = "api.github.com"
API_PATH = "/repos/RubenRibGarcia/asyncgo/releases?per_page=100"
TIMEOUT = 8  # seconds

def _fetch() -> list[dict] | None:
    """Published releases, newest first. None when the API cannot be read."""

def _published(payload: list) -> list[dict]:
    """Drop drafts — applied to the API response and to the cache alike."""

def _read_cache(cache: Path) -> list[dict] | None: ...
def _write_cache(cache: Path, releases: list[dict]) -> None: ...

def _render(releases: list[dict]) -> str:
    """Newest first; a release body that already opens with a heading is used
    verbatim, otherwise its heading is added from the API metadata."""

def on_config(config):
    content = Path(config.docs_dir).resolve()
    cache = content.parent / ".cache" / "releases.json"
    releases = _fetch()
    if releases is not None:
        _write_cache(cache, releases)   # an empty list is worth caching too
    else:
        releases = _read_cache(cache)
    page = _render(releases) if releases is not None else _fallback()
    if not (content / "changelog.md").exists() or \
       (content / "changelog.md").read_text(encoding="utf-8") != page:
        (content / "changelog.md").write_text(page, encoding="utf-8")
    return config

def on_page_markdown(markdown, page, config, files):
    if page.file.src_uri == "changelog.md":
        # The page is generated: point "Edit this page" at the only editable copy.
        page.edit_url = RELEASES_URL
    return markdown
```

`_fallback()` returns a short page — an admonition saying the live list is on
GitHub Releases, with a link — so `--strict` never fails on a rate limit or an
offline laptop. `ASYNCGO_RELEASES_PATH` can be aimed at a 404 on the same host to
exercise the fallbacks without mocking the network:

```bash
make site-build                                                 # live API
ASYNCGO_RELEASES_PATH=/repos/RubenRibGarcia/asyncgo/nope make site-build  # cache, then stub
```

Drafts are filtered in `_published()`, which both the API response and the cache
go through: the cache is a file on disk, and trusting its contents would make the
filter depend on who wrote it.

`site/mkdocs.yml` gains:

```yaml
nav:
  - ...
  - Project:
      - Overview: project/index.md
      # Generated at build time from GitHub Releases by hooks/changelog.py.
      - Changelog: changelog.md
      - AsyncAPI 3.1.0 coverage: project/asyncapi-3.1.0-coverage.md
      ...

hooks:
  - hooks/assemble.py
  - hooks/changelog.py

watch:
  - ../docs
  - hooks/changelog.py
```

`page.meta` for Material's version banner is deliberately not wired: the site has
no version selector, and the changelog page carries the versions.

### 4. `.github/workflows/docs.yml`

Add a token to the build step so the GitHub API call is authenticated (public
reads work anonymously, but the shared runner IP has 60 requests/hour):

```yaml
      - name: Build site
        working-directory: site
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: mkdocs build --strict
```

`permissions: contents: read` is already correct and sufficient.

### 5. Local tooling

`Makefile` — one target, the same command CI runs:

```makefile
# The changelog is generated, never edited: release notes come from conventional
# commits via git-cliff (cliff.toml). CI runs exactly this on dispatch.
release-notes:
	@test -n "$(VERSION)" || { echo "usage: make release-notes VERSION=0.1.0"; exit 2; }
	git cliff --config cliff.toml --tag v$(VERSION) --unreleased --strip all
```

`mise.toml` — `git-cliff = "<pinned>"` next to `go`, so local and CI render
identical notes. The pin must match `taiki-e/install-action`'s `tool:` value;
pick the exact patch at implementation time and bump both together.

### 6. Removals and documentation

| Path | Change |
| --- | --- |
| `.goreleaser.yaml` | Deleted |
| `.github/workflows/release.yml` | Rewritten (above); Syft and both `actions/attest` steps go with the archives |
| `AGENT.md` § Release | Rewritten: dispatch with a version, what the workflow does, how to correct a published release (`gh release edit`), and that `vX.Y.Z` is the API contract for `go install` |
| `AGENT.md` repo layout | `.goreleaser.yaml` line dropped; `cliff.toml` and the Changelog hook noted next to `site/` |
| `site/content/getting-started.md` | The goreleaser admonition (≈ lines 22–30) replaced: releases are source-only, install with `go install …@vX.Y.Z`, `asyncgo version` reports the module version |
| `.gitignore` | `site/content/changelog.md`, `site/.cache/`, `/release-notes.md` |
| `docs/designdoc/README.md` | Index row for this document |

## Example (before / after)

Before, `v0.1.0`:

```text
$ git tag v0.1.0 && git push origin v0.1.0
→ 6 archives × (tar.gz|zip) + 6 SBOMs + checksums + 2 attestations + source tarball
→ release body: GoReleaser's grouped commit list
→ site: unchanged (no trigger, no changelog page)
```

After, `v0.1.0`: *Actions → Release → Run workflow*, `version: 0.1.0`.

```markdown
## [0.1.0](https://github.com/RubenRibGarcia/asyncgo/releases/tag/v0.1.0) - 2026-11-20

### Added
- **internal:** write one document per catalog into a directory ([59d896a](https://github.com/RubenRibGarcia/asyncgo/commit/59d896a))
- **schema:** reject unknown asyncapi struct-tag directives ([b4539e6](https://github.com/RubenRibGarcia/asyncgo/commit/b4539e6))

### Fixed
- **schema:** map gofrs/uuid v5 by its /v5 import path ([55bf7c6](https://github.com/RubenRibGarcia/asyncgo/commit/55bf7c6))
```

GitHub's assets: the tag's automatic source archives only. The site gains
`/changelog/` listing every published release with the same text.

## Edge cases

- **`master` moves between dispatch and run** — validation fails with "master
  moved"; re-run. No half-release.
- **Retry after a failed `gh release create`** — the tag already exists and
  points at HEAD, so tagging is skipped and publishing continues; a tag at a
  different commit is an error.
- **`dry_run: true`** — validates, checks the tree, renders and prints; nothing
  is tagged, published or dispatched.
- **Pre-release (`0.3.0-rc.1`)** — semver regex accepts it, `--prerelease` is
  passed, so it does not become `latest`. `tag_pattern = "v[0-9]*"` means
  `v0.3.0-rc.1` is not a changelog baseline; the final `v0.3.0` covers both.
- **Nothing releasable since the last tag** — `grep '^### '` fails the job with
  "nothing releasable"; that is the signal that the version is pointless.
- **Two draft releases exist** (`v0.0.1`, `v0.1.0`, no tags). The hook filters
  `draft: true`; validation refuses to publish a version that already has a
  release. Decide before the first run: delete the drafts (recommended — they
  have no tags and nothing to keep) or publish them from the tag flow. Editing
  them into the flow is out of scope.
- **First real release** — the remote has no tags, so the first dispatch renders
  the entire history under one heading. Review it with `dry_run` first; it is a
  large section and worth one manual pass.
- **Typo in published notes** — `gh release edit vX.Y.Z --notes-file notes.md`,
  then re-run *Docs* (or wait for the next release, which dispatches it). The
  site never holds an independent copy.
- **API rate limit / offline build** — hook falls back to `site/.cache/releases.json`,
  then to the stub page with a link. CI always has a token, so the fallback is a
  local/rare path.
- **Renamed or deleted release** — it simply disappears from the page; releases
  are the deletion mechanism, there is no second copy to reconcile.
- **git-cliff version drift** — CI and `mise` pin the same version, so a local
  preview matches the published body byte for byte.
- **`.golangci.yaml`, coverage gates** — untouched: the release job deliberately
  does not re-run lint/tests (D6 gates on the existing run).

## Rollout plan

1. **Stage 0 — notes without process** (`chore(release)`): add `cliff.toml`,
   the `make release-notes` target and the `mise` pin. Run
   `make release-notes VERSION=0.1.0` on `master` and review the full-history
   output. No CI change, nothing irreversible.
2. **Stage 1 — the site page** (`docs(site)`): `site/hooks/changelog.py`,
   `mkdocs.yml` nav + hook + watch, `docs.yml` token, `.gitignore` entries.
   With zero published releases the page says so; verify `make site-build`
   and the fallbacks via `ASYNCGO_RELEASES_PATH`.
3. **Stage 2 — the workflow** (`ci(release)`): rewrite `release.yml`, delete
   `.goreleaser.yaml`, rewrite AGENT.md § Release, fix the
   `getting-started.md` admonition. Merge, then dispatch with `dry_run: true`
   from `master` and review the rendered notes in the job log.
4. **Stage 3 — first real release** (`v0.1.0`): delete the two stale drafts,
   dispatch `version: 0.1.0`, confirm the tag, the release body and
   `/changelog/`, and that `go install …@v0.1.0` reports `0.1.0`. The decisions
   above are recorded as
   [ADR 0013](../adr/0013-release-by-workflow-dispatch.md).
5. **Stage 4 — deferred**: a `CHANGELOG.md` generated from the same config
   (`git cliff --output CHANGELOG.md`) if a file in the repository is ever
   wanted; prebuilt binaries behind a second dispatch input if someone needs them
   without a Go toolchain — the workflow has room for a build job, and its
   absence is a decision, not an accident.

Each stage is independently mergeable; stage 2 must not land before stage 1, or
the first release has nowhere to appear.

## Testing plan

- **Stage 0**: `make release-notes VERSION=0.1.0` renders groups for
  `feat`/`fix`, skips `chore`/`docs`/`test`, drops merges, and marks a
  synthetic `feat!: …` commit as breaking (`git commit` on a scratch branch).
  - `should_group_by_conventional_type`
  - `should_skip_chore_and_docs`
  - `should_mark_breaking_commits`
- **Stage 1**: `make site-build` passes `--strict` with the page in `nav`;
  `ASYNCGO_RELEASES_PATH=<404 path>` produces the stub and still builds;
  a cached release list renders without network.
  - `should_render_each_published_release`
  - `should_exclude_drafts`
  - `should_fall_back_to_cache_then_stub`
- **Stage 2**: YAML lints (the repo's `actionlint`/editor schema if configured);
  every shell `run:` block exercised via `act`-free dry paths: bad version
  (`0.1`, `v0.1.0`, `0.1.0 `), existing tag at HEAD, existing tag elsewhere,
  existing release, red test run.
  - `should_reject_non_semver_input`
  - `should_reject_a_tag_that_is_not_head`
  - `should_refuse_to_publish_a_red_commit`
- **Stage 3**: the real dispatch is the test; verify tag → release → site page →
  `go install …@v0.1.0 && asyncgo version`.

## Open / deferred

- **`actionlint`** is not wired into `test.yaml`. The new workflow has a
  non-trivial shell block; adding `actionlint` (a single pinned binary, like
  git-cliff) would catch expression and permission mistakes before a dispatch.
  Out of scope here; worth an issue if stage 2 feels fragile.
- **Changelog ordering on the site** — the API returns releases newest first, but
  `sort_commits = "oldest"` groups commits oldest-first inside a release. Fine
  for now; say if the page should lead with the newest tag's biggest group.
- **Release cadence** — the flow makes a release cheap, which invites
  one-commit releases. Nothing enforces a cadence; that stays a human habit.
