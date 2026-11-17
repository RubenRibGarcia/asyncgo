"""Publish the GitHub Releases feed as the site's Changelog page.

GitHub Releases is the changelog's single source of truth: the release workflow
(`.github/workflows/release.yml`) renders a release body with git-cliff from
`cliff.toml` at dispatch time, and a correction is `gh release edit` — there is no
second copy to keep in step. This hook renders whatever is published.

Like the Project section's copies (`assemble.py`), the page is a build artifact:
regenerated on every build, gitignored, and rewritten only when its content
changed so `mkdocs serve` does not rebuild itself in a loop.

The build therefore reads the network, which `--strict` would turn into a hard
failure. Two fallbacks keep it green: the cache written by the last successful
fetch (`site/.cache/releases.json`), then a stub page that points at the releases
page. Set `ASYNCGO_RELEASES_PATH` to a bad or empty path to exercise them.
"""

from __future__ import annotations

import http.client
import json
import os
from datetime import datetime
from pathlib import Path

#: The repository whose releases are the changelog.
REPO = "RubenRibGarcia/asyncgo"
RELEASES_URL = f"https://github.com/{REPO}/releases"

#: The host is a literal on purpose: `http.client.HTTPSConnection` enforces TLS,
#: and only the path below can be aimed elsewhere (at a 404, to test fallbacks).
API_HOST = "api.github.com"
API_PATH = f"/repos/{REPO}/releases?per_page=100"

#: Seconds to wait for the API before falling back to the cache.
TIMEOUT = 8

#: The generated page, relative to `docs_dir`.
PAGE = "changelog.md"


def _api_path() -> str:
    """The request path.

    `ASYNCGO_RELEASES_PATH` exists only so a developer can aim the build at a 404
    or an empty page to verify the cache and stub fallbacks; it cannot leave
    `api.github.com` or pick another path prefix.
    """
    path = os.environ.get("ASYNCGO_RELEASES_PATH", API_PATH)
    if not path.startswith("/repos/") or " " in path or "\n" in path:
        raise ValueError(
            f"ASYNCGO_RELEASES_PATH must start with /repos/, got {path!r}"
        )
    return path


def _fetch() -> list[dict] | None:
    """Published releases, newest first. None when the API cannot be read.

    A token is only needed to raise the anonymous rate limit; the repository is
    public, so an unauthenticated local build normally succeeds.
    """
    headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": "asyncgo-site",
    }
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"

    path = _api_path()
    connection = http.client.HTTPSConnection(API_HOST, timeout=TIMEOUT)
    try:
        connection.request("GET", path, headers=headers)
        response = connection.getresponse()
        body = response.read()
        status = response.status
    except (OSError, http.client.HTTPException) as exc:
        print(f"changelog: cannot read {API_HOST}{path}: {exc}")
        return None
    finally:
        connection.close()

    if status != 200:
        print(f"changelog: {API_HOST}{path} returned HTTP {status}")
        return None
    try:
        payload = json.loads(body)
    except json.JSONDecodeError as exc:
        print(f"changelog: {API_HOST}{path} did not return JSON: {exc}")
        return None

    if not isinstance(payload, list):
        print(f"changelog: {API_HOST}{path} did not return a list")
        return None
    return _published(payload)


def _published(payload: list) -> list[dict]:
    """Drop drafts: they are unpublished by definition and must not reach the site.

    Applied to the API response and to the cache alike — the cache is a file on
    disk, and trusting its contents would make the filter depend on who wrote it.
    """
    return [r for r in payload if isinstance(r, dict) and not r.get("draft")]


def _read_cache(cache: Path) -> list[dict] | None:
    try:
        payload = json.loads(cache.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    return _published(payload) if isinstance(payload, list) else None


def _write_cache(cache: Path, releases: list[dict]) -> None:
    try:
        cache.parent.mkdir(parents=True, exist_ok=True)
        cache.write_text(json.dumps(releases, indent=2), encoding="utf-8")
    except OSError as exc:  # a read-only checkout must not fail the build
        print(f"changelog: cannot write {cache}: {exc}")


def _date(release: dict) -> str:
    stamp = release.get("published_at") or release.get("created_at") or ""
    try:
        return datetime.fromisoformat(stamp.replace("Z", "+00:00")).date().isoformat()
    except ValueError:
        return stamp[:10]


def _heading(release: dict) -> str:
    tag = release.get("tag_name", "")
    label = release.get("name") or tag
    pre = " (pre-release)" if release.get("prerelease") else ""
    url = release.get("html_url") or RELEASES_URL
    return f"## [{label}]({url}) - {_date(release)}{pre}"


def _render(releases: list[dict]) -> str:
    """One section per release, newest first, using the release body verbatim.

    git-cliff bodies already open with a version heading; a body without one (an
    older or hand-written release) gets a heading built from the API metadata
    instead of being rendered headless.
    """
    parts = [
        "# Changelog",
        "",
        "Every published release, newest first, with the notes published on"
        f" [{REPO} releases]({RELEASES_URL}). The notes are the release bodies — a"
        " correction belongs there, not here.",        "",
    ]
    if not releases:
        parts += ["No releases have been published yet.", ""]
    for release in releases:
        body = (release.get("body") or "").strip()
        if not body:
            parts += [_heading(release), "", "_No notes were published for this release._", ""]
            continue
        if body.lstrip().startswith("#"):
            parts += [body, ""]
        else:
            parts += [_heading(release), "", body, ""]
    return "\n".join(parts).rstrip() + "\n"


def _fallback() -> str:
    return f"""# Changelog

!!! warning "The release list is not available in this build"

    This page is generated from the repository's GitHub Releases. The build could
    not reach the GitHub API and had no cached copy, so the releases cannot be
    listed here.

Every published release, with its notes, is on the
[{REPO} releases]({RELEASES_URL}) page.
"""


def on_config(config):
    content_dir = Path(config.docs_dir).resolve()
    cache = content_dir.parent / ".cache" / "releases.json"

    releases = _fetch()
    if releases is not None:
        # Cache even an empty list: "the API answered, nothing is published" is
        # worth keeping across an outage.
        _write_cache(cache, releases)
    else:
        releases = _read_cache(cache)

    page = _render(releases) if releases is not None else _fallback()
    dest = content_dir / PAGE
    if not dest.exists() or dest.read_text(encoding="utf-8") != page:
        dest.write_text(page, encoding="utf-8")
    return config


def on_page_markdown(markdown, page, config, files):
    """Point "Edit this page" at the only editable copy of an entry."""
    if page.file.src_uri == PAGE:
        page.edit_url = RELEASES_URL
    return markdown
