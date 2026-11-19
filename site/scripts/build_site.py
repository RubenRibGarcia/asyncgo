#!/usr/bin/env python3
"""Build the versioned project site.

The site is published at https://rubenribgarcia.github.io/asyncgo/, and a reader
on an old release must not be handed the docs for a newer one. So every release
line gets its own directory, built from the tag that released it:

    /              meta refresh into latest/
    /latest/       the newest stable release (a copy of its version directory)
    /0.1/ /0.2/…   one directory per minor line, from its highest patch tag
    /dev/          the current checkout — in CI, master
    /versions.json the selector's source of truth (mike's schema)

Two decisions are load-bearing:

* **The root is a redirect, not the newest build.** The theme's selector
  (Material's `extra.version`, no `mike` plugin needed) resolves a sibling link
  as `../<version>/` against the page's `site_url`, so `site_url` has to carry a
  version segment for the links to land on `/0.1/` rather than `/0.1/` one level
  too high. A build served at `/` has no version segment, so `latest/` is a real
  directory and `/` only redirects.

* **Content comes from the tag; tooling comes from this checkout.** A version's
  `site/content/` and `docs/` are checked out at its tag, so old docs really are
  old docs. `mkdocs.yml`, `hooks/` and `overrides/` are copied from the current
  checkout, so every version is built by the same toolchain with the same
  selector, the same link rewrite and the same changelog cutoff. Rebuilding with
  each tag's own pinned requirements would eventually break on a yanked version
  and would leave tags predating a feature without it.

The consequence to watch: today's `nav:` is applied to yesterday's content, so a
page added to the nav after an old tag existed would fail that version's
`--strict` build. That is deliberate — the build fails loudly instead of
publishing a version with a broken nav.

Usage:
    python3 site/scripts/build_site.py all

It writes a complete tree into site/build/. There is one path on purpose: a
single unversioned build could not prove that the versioned tree builds, and
"today's nav applied to yesterday's content" is exactly the thing that breaks.
`make site-build` is the fast single build for iterating on a page; this is the
one CI runs, on pushes and on pull requests alike.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

#: Where the published site lives. Version directories are appended to it.
BASE_URL = "https://rubenribgarcia.github.io/asyncgo/"

#: The directory of the build that `/latest/` mirrors.
LATEST = "latest"

#: The directory of the current checkout's build.
DEV = "dev"

#: A stable release tag. Pre-releases are deliberately skipped: they are
#: short-lived, and publishing them would push a real release out of `latest/`.
_TAG = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")

#: Files and directories taken from this checkout rather than from the tag.
TOOLING = ("mkdocs.yml", "hooks", "overrides")

USAGE = "usage: build_site.py all"


def repo_root() -> Path:
    return Path(_git("rev-parse", "--show-toplevel").strip())


def _git(*args: str, cwd: Path | None = None) -> str:
    return subprocess.run(
        ["git", *args],
        cwd=str(cwd) if cwd else None,
        check=True,
        capture_output=True,
        text=True,
    ).stdout


def _executable(path: Path) -> bool:
    try:
        return path.is_file() and os.access(path, os.X_OK)
    except OSError:
        return False


def _remove(path: Path) -> None:
    """Delete a directory this script owns, failing loudly rather than silently."""
    try:
        if path.exists():
            shutil.rmtree(path)
    except OSError as exc:
        sys.exit(f"build_site: cannot remove {path}: {exc}")


def mkdocs() -> str:
    """The venv's mkdocs when it exists, so `make site-versions` and CI agree."""
    local = repo_root() / "site" / ".venv" / "bin" / "mkdocs"
    if _executable(local):
        return str(local)
    found = shutil.which("mkdocs")
    if not found:
        sys.exit("build_site: no mkdocs on PATH — run `make site-deps` first")
    return found


def versions() -> list[dict]:
    """One entry per minor line, newest first, from the repository's tags.

    A line is built from its highest patch tag, because a patch release is a fix
    rather than a new interface: the docs for 0.1.1 describe 0.1.0 as well. The
    newest entry carries the `latest` alias, which is what the theme compares
    against `extra.version.default` to decide whether to warn about an outdated
    page; `dev` is listed too, so it does not warn about itself.
    """
    tags = [t for t in _git("tag", "--list", "v*").split() if t]
    lines: dict[tuple[int, int], tuple[tuple[int, int, int], str]] = {}
    skipped: list[str] = []
    for tag in tags:
        match = _TAG.match(tag)
        if not match:
            skipped.append(tag)
            continue
        try:
            major, minor, patch = (int(part) for part in match.groups())
        except ValueError:  # unreachable: _TAG only matches digits
            sys.exit(f"build_site: cannot parse release tag {tag}")
        key = (major, minor)
        if key not in lines or (major, minor, patch) > lines[key][0]:
            lines[key] = ((major, minor, patch), tag)

    built = [
        {
            "directory": f"{major}.{minor}",
            "tag": tag,
            "title": f"{major}.{minor}",
            "aliases": [],
        }
        for (major, minor), (_, tag) in sorted(lines.items(), reverse=True)
    ]
    if skipped:
        # Silent truncation would read as "every release is published".
        print(f"build_site: not a stable vX.Y.Z tag, not published: {', '.join(sorted(skipped))}")

    if not built:
        sys.exit("build_site: no stable release tags — nothing to publish but dev/")
    built[0]["aliases"] = [LATEST]
    built.append({"directory": DEV, "tag": "HEAD", "title": "development", "aliases": []})
    return built


def _environment(version: dict, latest: str) -> dict[str, str]:
    """Per-build configuration, read by mkdocs.yml through `!ENV`."""
    directory = version["directory"]
    pinned = "" if directory == DEV else directory
    return {
        **os.environ,
        "ASYNCGO_SITE_URL": f"{BASE_URL}{directory}/",
        # "Edit this page" has to open the source the page was built from, and
        # for a release that is the tag, not master.
        "ASYNCGO_EDIT_URI": f"edit/{version['tag']}/site/content/",
        "ASYNCGO_VERSION": directory,
        "ASYNCGO_VERSION_TAG": version["tag"],
        "ASYNCGO_LATEST_VERSION": latest,
        "ASYNCGO_LATEST_URL": f"{BASE_URL}{LATEST}/",
        # The changelog stops at the version being built: a 0.1 build must not
        # list 0.2's releases.
        "ASYNCGO_MAX_VERSION": pinned,
    }


def _build(source: Path, out: Path, version: dict, latest: str) -> None:
    print(f"build_site: {version['directory']} from {version['tag']} -> {out}")
    subprocess.run(
        [mkdocs(), "build", "--strict", "-d", str(out)],
        cwd=str(source / "site"),
        env=_environment(version, latest),
        check=True,
    )


def _overlay_tooling(tree: Path, ours: Path) -> None:
    """Replace the tag's build tooling with this checkout's, keeping its content."""
    for name in TOOLING:
        source, dest = ours / "site" / name, tree / "site" / name
        if source.is_dir():
            _remove(dest)
            shutil.copytree(source, dest)
        else:
            shutil.copy2(source, dest)


def build_all(out: Path) -> None:
    plan = versions()
    latest = plan[0]["directory"]
    root = repo_root()
    ours = Path(__file__).resolve().parent.parent.parent

    with tempfile.TemporaryDirectory(prefix="asyncgo-site-") as scratch:
        for version in plan:
            if version["directory"] == DEV:
                _build(root, out / DEV, version, latest)
                continue
            tree = Path(scratch) / version["directory"]
            _git("worktree", "add", "--detach", str(tree), version["tag"], cwd=root)
            try:
                _overlay_tooling(tree, ours)
                _build(tree, out / version["directory"], version, latest)
            finally:
                _git("worktree", "remove", "--force", str(tree), cwd=root)

    # `/latest/` is a copy rather than a second build: the copy keeps the tag's
    # own site_url, so its canonical link still points at the version directory
    # it duplicates.
    shutil.copytree(out / latest, out / LATEST, dirs_exist_ok=True)
    assemble(out, plan)


def assemble(out: Path, plan: list[dict]) -> None:
    """Write the two files that tie the version directories together."""
    # The selector reads each entry as mike's schema spells it — `version` is
    # the directory it links to and what it matches the current page against,
    # `title` the label, `aliases` the extra names (`latest`) the version also
    # answers to. The plan's own key is `directory`, so the mapping is explicit
    # here: published under that name, every selector link resolved to
    # `../undefined/` and no page matched its own version.
    published = [
        {
            "version": entry["directory"],
            "title": entry["title"],
            "aliases": entry["aliases"],
        }
        for entry in plan
    ]
    (out / "versions.json").write_text(
        json.dumps(published, indent=2) + "\n", encoding="utf-8"
    )

    # GitHub Pages serves / as a directory index, so the redirect is a file.
    (out / "index.html").write_text(
        f"""<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>asyncgo</title>
    <link rel="canonical" href="{BASE_URL}{LATEST}/">
    <meta http-equiv="refresh" content="0; url={LATEST}/">
  </head>
  <body>
    <p>Redirecting to the <a href="{LATEST}/">asyncgo documentation</a>.</p>
  </body>
</html>
""",
        encoding="utf-8",
    )


def main(argv: list[str]) -> None:
    if len(argv) != 2 or argv[1] != "all":
        sys.exit(USAGE)

    out = repo_root() / "site" / "build"
    # Only this script writes here, and a stale directory would publish pages
    # for versions that no longer exist.
    _remove(out)
    out.mkdir(parents=True)

    build_all(out)


if __name__ == "__main__":
    main(sys.argv)
