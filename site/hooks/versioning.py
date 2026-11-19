"""Give each published build its own identity.

The site is versioned: `site/scripts/build_site.py` builds one directory per
release line (0.1, 0.2, …), plus `dev` for the current checkout, and mirrors the
newest release as `latest`. A page's *content* comes from the tag that released
it; the *identity* of the build cannot, because every directory is built by the
same toolchain and the same theme, and because Material's version selector needs
to know which directory it is rendering.

That identity is passed through the environment rather than written into
`mkdocs.yml`, so this file is the only place that knows the variable names, and
so a plain local build (`make site-serve`, `make site-build`) keeps the
`mkdocs.yml` defaults without pretending to be a release.

Variables, all set by `site/scripts/build_site.py`:

    ASYNCGO_SITE_URL         https://…/asyncgo/0.1/ — the selector resolves a
                             sibling version as `../<version>/`, so the current
                             version has to be in here
    ASYNCGO_EDIT_URI         edit/v0.1.0/site/content/ — "Edit this page" must
                             open the source the page was built from, not master
    ASYNCGO_VERSION          0.1 / 0.2 / dev — the directory being rendered
    ASYNCGO_VERSION_TAG      v0.1.0 / HEAD — where the content came from
    ASYNCGO_LATEST_VERSION   the newest release line, for the outdated banner
    ASYNCGO_LATEST_URL       absolute link to it

A missing `ASYNCGO_VERSION` means "not a versioned build": nothing is changed.
A present but malformed value is an error — a typo must not silently publish a
site that claims to be a version it is not.
"""

from __future__ import annotations

import os
import re

#: A build directory: a minor release line, or the current checkout.
_VERSION = re.compile(r"^(?:dev|\d+\.\d+)$")


def _required(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise ValueError(f"versioning: {name} is required when ASYNCGO_VERSION is set")
    return value


def _url(name: str) -> str:
    """An absolute directory URL, trailing slash included: the theme resolves
    sibling versions relatively, and a missing slash changes that resolution."""
    value = _required(name)
    if not value.startswith("https://") or not value.endswith("/"):
        raise ValueError(f"versioning: {name} must be an https URL ending in '/', got {value!r}")
    return value


def _edit_uri() -> str:
    value = _required("ASYNCGO_EDIT_URI")
    if not value.startswith("edit/") or not value.endswith("/"):
        raise ValueError(f"versioning: ASYNCGO_EDIT_URI must be 'edit/<ref>/…/', got {value!r}")
    return value


def on_config(config):
    version = os.environ.get("ASYNCGO_VERSION", "").strip()
    if not version:
        return config
    if not _VERSION.match(version):
        raise ValueError(f"versioning: ASYNCGO_VERSION must be 'dev' or MAJOR.MINOR, got {version!r}")

    config.site_url = _url("ASYNCGO_SITE_URL")
    config.edit_uri = _edit_uri()
    config.extra["current_version"] = version
    config.extra["current_tag"] = _required("ASYNCGO_VERSION_TAG")
    config.extra["latest_version"] = _required("ASYNCGO_LATEST_VERSION")
    config.extra["latest_url"] = _url("ASYNCGO_LATEST_URL")
    print(f"versioning: building {version} from {config.extra['current_tag']} at {config.site_url}")
    return config
