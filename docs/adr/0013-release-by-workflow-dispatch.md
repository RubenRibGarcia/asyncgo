# 0013. Release by workflow dispatch with a generated changelog

- Status: accepted
- Deciders: asyncgo maintainers
- Created: 2026-11-16
- Status updated: 2026-11-16

## Context and Problem Statement

A release used to be a `git push` of a `v*` tag, which ran GoReleaser
(`.goreleaser.yaml`) to cross-compile `asyncgo` for linux/darwin/windows ×
amd64/arm64, build an SBOM per archive, checksum the lot, attest the checksums,
and create the GitHub release. None of that underpinned anything:

- The CLI is installed with `go install .../cmd/asyncgo@vX.Y.Z`, and
  `internal/cli/version.go` already recovers the version from
  `debug.ReadBuildInfo()`. The archives were never downloaded because they were
  never published — `git ls-remote --tags origin` is empty, and the only two
  releases (`v0.0.1`, `v0.1.0`) sit as drafts.
- The changelog was GoReleaser's commit list, filtered by exclusion regexes,
  used once as a release body and never shown anywhere else. The project site has
  no changelog, so a release was invisible to anyone reading the docs.
- A tag push cannot express "publish only if this commit's CI passed", nor
  "show me the notes before anything is irreversible", and the tag is the only
  record — a wrong commit in it is a rewrite or a junk release.
- The project has no release cadence yet: the first real version is still ahead.

What is actually needed is small: decide a version, check that commit is green,
write the changelog, tag it, publish it, and show it on the site. That is one
workflow run, not a cross-compilation pipeline.

## Decision Drivers

- One dispatch with one input (the version) must be the whole release process.
- Nothing irreversible may happen before the notes are reviewable.
- The changelog must be generated from conventional commits, not hand-copied,
  and there must be exactly one place to correct an entry.
- The changelog must be readable on the project site, not only on GitHub.
- The install path is `go install`; artifacts nobody installs are not worth a
  build matrix, an SBOM step, and two attestation steps.
- A release must not be possible from a commit whose tests did not pass, nor from
  a branch that moved while the dispatch was being clicked.

## Considered Options

- Keep GoReleaser on the tag push, and add a changelog page generated from the
  releases.
- `workflow_dispatch` + `gh release create --generate-notes` (GitHub's PR-based
  notes) + a committed `CHANGELOG.md` written by the bot.
- `workflow_dispatch` + git-cliff + GitHub Releases as the changelog's source of
  truth, rendered into the site at build time, source-only releases.
- `semantic-release` / `release-please`: a bot infers the version from commit
  types, tags, and publishes.
- Keep the tag-push trigger but move the pipeline into plain `go build` steps.

## Decision Outcome

Chosen option: "`workflow_dispatch` + git-cliff + GitHub Releases as the
changelog's source of truth, rendered into the site at build time, source-only
releases", because the pipeline people actually use is a tag and a changelog, and
everything else GoReleaser did was serving an install path nobody takes.

Concretely:

- **D1 — The release trigger is `workflow_dispatch`, not a tag push.** Inputs:
  `version` (semver without the leading `v`) and `dry_run`. `on: push: tags` is
  removed from `.github/workflows/release.yml`.
- **D2 — The workflow creates the annotated tag and the release; nothing is
  built.** Steps: validate → require a green `test.yaml` run for `HEAD` → render
  the notes → `git tag -a` → `git push origin "$TAG"` → `gh release create
  --verify-tag --notes-file`. The tag is the release, and GitHub attaches the
  source archives, so there is no `dist/`, no SBOM, no checksum file and no
  attestation.
- **D3 — git-cliff, configured in `cliff.toml`, renders the notes.** One command,
  run identically by CI and by `make release-notes VERSION=…`:
  `git cliff --config cliff.toml --tag "v$VERSION" --unreleased --strip all`.
  Only `feat`/`fix`/`perf`/`refactor`/`revert` reach a release; the exclusion
  regexes GoReleaser carried become `commit_parsers` data, merge commits are
  dropped by `filter_unconventional`, and `protect_breaking_commits` marks a
  breaking change inline. `--unreleased` + `--tag` produce the future release
  body *before* the tag exists, which is what `dry_run` prints.
- **D4 — GitHub Releases is the changelog's single source of truth, and the site
  renders it at build time.** `site/hooks/changelog.py` writes
  `site/content/changelog.md` from the Releases API on every build (drafts
  filtered, cached in `site/.cache/releases.json`, stub page when the API is
  unreachable), following the same pattern `site/hooks/assemble.py` established
  for the development-process records. There is no committed `CHANGELOG.md`, so a
  correction is `gh release edit` and nothing else changes.
- **D5 — The release dispatches the site build.** The last step is
  `gh workflow run docs.yml --ref master`. A tag pushed with `GITHUB_TOKEN` does
  not trigger other workflows, and a release touches no path in the docs
  workflow's `on.push.paths`; dispatching after the release exists is what makes
  the new tag visible to the hook.
- **D6 — Only a commit whose tests passed can be released.** The workflow
  asserts that `HEAD` is `origin/master` and that the newest `Test & Coverage`
  run has `headSha == HEAD` and `conclusion == success`, instead of re-running
  lint and tests inside the release job.
- **Retries are idempotent.** A tag that already exists at `HEAD` is accepted
  (the publish step skips tagging and re-renders with `--latest`, since there are
  no unreleased commits left); a tag at another commit or a release that already
  exists stops the run and points at `gh release edit`.
- **The first release is `v0.1.0`.** The two stale drafts (`v0.0.1`, `v0.1.0`)
  carry no tags and no artifacts and are deleted before the first dispatch —
  `v0.1.0` in particular would otherwise be refused as an existing release. The
  first release therefore renders the whole history under one heading, reviewed
  first with `dry_run: true`.

Rejected: keeping GoReleaser was rejected because six archives, an SBOM per
archive and two attestations serve no install path the project documents;
`--generate-notes` with a committed `CHANGELOG.md` was rejected because the body
format depends on PR labels rather than conventional commits, it cannot be
previewed before publishing, and the file creates a second source of truth that
needs a bot push to `master` and a manual docs dispatch to reach the site;
`semantic-release`/`release-please` were rejected because they add a Node runtime
and infer the version from commit types, when the version is the one decision
here that should stay human; a plain `go build` tag-push pipeline was rejected
because it keeps every property D1 and D6 remove — no green-commit gate, no
preview, and a tag that cannot be taken back.

### Consequences

- Good, because a release is one dispatch with a version, and `dry_run` makes the
  exact notes reviewable before anything is irreversible.
- Good, because `HEAD` must be green and on `master` before a tag exists; a stale
  dispatch fails instead of publishing whatever the branch happened to hold.
- Good, because the changelog is generated from conventional commits into a
  single editable place (the release body) and reaches both GitHub and the site
  without a second commit.
- Good, because the workflow shrank from a cross-compile matrix, Syft and two
  attest steps to seven steps and two permissions (`contents: write`,
  `actions: write`), with the toolchain pinned in `mise.toml` and in CI at the
  same git-cliff version.
- Bad, because the site build now reads the network. `--strict` would fail on a
  rate limit or an offline laptop, so the hook carries a cache and a stub page —
  two failure paths to keep honest, and a local build that can silently render
  the cache instead of live data (it prints which).
- Bad, because there are no prebuilt binaries: anyone without a Go toolchain
  cannot install the CLI. Adding a build job back is a workflow change, not a
  redesign, but it is now a decision someone has to make.
- Bad, because a release now depends on `git-cliff` and on commit-message
  discipline: a `feat:` that ships user-visible change outside the five mapped
  types never appears in the changelog.
- Bad, because the changelog page is only as fresh as the last site build; a
  release edited with `gh release edit` needs a docs run to show up.

## More Information

- Design doc: [release-workflow](../designdoc/release-workflow.md) — includes the
  alternatives table, the site hook's fallbacks, and the rollout stages.
- Supersedes the release process described in `AGENT.md` § Release before this
  ADR: tag push → GoReleaser → archives, checksums, SBOMs, attestations.
- Removes `.goreleaser.yaml`; adds `cliff.toml`,
  `site/hooks/changelog.py`, and the `release-notes` Makefile target.
- Related: [ADR 0009](0009-verbose-cli-logging.md) — the convention that
  user-facing output is deliberate, which the changelog body follows.
