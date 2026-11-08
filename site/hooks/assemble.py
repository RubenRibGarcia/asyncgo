"""Build-time assembly of the site's Project section.

`docs/adr/`, `docs/designdoc/` and `docs/asyncapi-3.1.0-coverage.md` are the
repository's development-process records: AGENT.md pins them there, they cross-
reference each other with relative links, and moving them into `site/content/`
would break both. This hook publishes them instead.

On every build it copies them into `site/content/project/` under their original
relative layout — so a link such as `../asyncapi-3.1.0-coverage.md#b7` from an ADR
still resolves — and generates the matching nav entries, so adding an ADR needs
no edit to `mkdocs.yml` and never trips `--strict` with "page not in nav".

The copies are gitignored build input, regenerated on every build. Files whose
content is unchanged are not rewritten, which keeps `mkdocs serve` from
rebuilding itself in a loop.
"""

from __future__ import annotations

import re
from pathlib import Path

#: Nav entries in mkdocs.yml that this hook replaces with generated sections.
ADR_SENTINEL = "ADRs"
DESIGNDOC_SENTINEL = "Design docs"

#: Copied verbatim, at the top level of the Project section.
COVERAGE_FILE = "asyncapi-3.1.0-coverage.md"

#: A markdown inline link whose target is a fragment.
_FRAGMENT_LINK = re.compile(r"\]\(([^)\s]*#)([^)\s]+)\)")


def _mkdocs_anchors(text: str) -> str:
    """Collapse GitHub-style fragments to the anchors Markdown generates.

    GitHub slugs `#### B8 — Binding zero values are dropped` to
    `b8--binding-zero-values-are-dropped`; Markdown drops the em dash and
    collapses the doubled dash to `b8-binding-zero-values-are-dropped`. Only the
    build copies are rewritten, so the records in docs/ keep working on GitHub.
    """

    def collapse(match: re.Match[str]) -> str:
        return f"]({match.group(1)}{re.sub('-{2,}', '-', match.group(2))})"

    return _FRAGMENT_LINK.sub(collapse, text)


def _first_heading(path: Path) -> str:
    """The document's H1, used as its nav label."""
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return path.stem


def _sync(sources: list[Path], dest_dir: Path, prune: bool = True) -> dict[str, Path]:
    """Copy `sources` into `dest_dir`, and (optionally) drop copies with no source."""
    dest_dir.mkdir(parents=True, exist_ok=True)
    written: dict[str, Path] = {}
    for src in sources:
        dest = dest_dir / src.name
        text = _mkdocs_anchors(src.read_text(encoding="utf-8"))
        if not dest.exists() or dest.read_text(encoding="utf-8") != text:
            dest.write_text(text, encoding="utf-8")
        written[dest.name] = src
    for stale in dest_dir.glob("*.md"):
        if prune and stale.name not in written:
            stale.unlink()
    return written


def _nav_children(copies: dict[str, Path], prefix: str) -> list:
    """Nav entries for one directory: the README first, then by filename."""
    children: list = []
    if "README.md" in copies:
        children.append({"Overview": f"{prefix}/README.md"})
    for name in sorted(copies):
        if name == "README.md":
            continue
        children.append({_first_heading(copies[name]): f"{prefix}/{name}"})
    return children


def _project_section(nav: list) -> list | None:
    for item in nav:
        if isinstance(item, dict) and isinstance(item.get("Project"), list):
            return item["Project"]
    return None


def on_config(config):
    content_dir = Path(config.docs_dir).resolve()
    repo_root = content_dir.parent.parent
    docs_src = repo_root / "docs"
    project_dir = content_dir / "project"

    adrs = _sync(sorted(docs_src.glob("adr/*.md")), project_dir / "adr")
    designs = _sync(sorted(docs_src.glob("designdoc/*.md")), project_dir / "designdoc")
    _sync([docs_src / COVERAGE_FILE], project_dir, prune=False)

    section = _project_section(config.nav)
    if section is None:
        return config

    rebuilt: list = []
    for item in section:
        if isinstance(item, dict) and ADR_SENTINEL in item:
            label = "Architecture Decision Records"
            rebuilt.append({label: _nav_children(adrs, "project/adr")})
        elif isinstance(item, dict) and DESIGNDOC_SENTINEL in item:
            rebuilt.append({"Design docs": _nav_children(designs, "project/designdoc")})
        else:
            rebuilt.append(item)
    section[:] = rebuilt
    return config


def _edit_url(config, repo_path: str) -> str:
    """An edit link for a file in the repository.

    The branch comes from `edit_uri` rather than `remote_branch`: the latter
    defaults to `gh-pages`, which is the branch ghp-import *publishes* to, not the
    branch the documents are edited on.
    """
    branch = config.edit_uri.split("/")[1] if config.edit_uri else "master"
    return f"{config.repo_url.rstrip('/')}/edit/{branch}/{repo_path}"


def on_page_markdown(markdown, page, config, files):
    """Point "Edit this page" at the real file in docs/, not the build copy."""
    src = page.file.src_uri
    for sub in ("adr/", "designdoc/"):
        if src.startswith(f"project/{sub}"):
            page.edit_url = _edit_url(config, f"docs/{sub}{Path(src).name}")
            return markdown
    if src == f"project/{COVERAGE_FILE}":
        page.edit_url = _edit_url(config, f"docs/{COVERAGE_FILE}")
    return markdown
