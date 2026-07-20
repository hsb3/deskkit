#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Derive the current GitHub repo from `gh` so the toolkit is repo-agnostic.

The other scripts that need an owner/name (the GraphQL ones + evidence-audit)
import these helpers instead of hardcoding a repo. `gh` is already authed and
already knows which repo the working tree belongs to, so we just ask it -- the
desk then drops into any project unchanged.

Usage (as a library):
    from _repo import repo_slug, owner_name
    slug = repo_slug()            # "owner/name"
    owner, name = owner_name()    # ("owner", "name")
"""

from __future__ import annotations

import functools
import subprocess


@functools.lru_cache(maxsize=1)
def repo_slug() -> str:
    """The current repo as "owner/name" (from `gh repo view`)."""
    out = subprocess.run(
        ["gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    if "/" not in out:
        raise RuntimeError(f"could not resolve repo from gh: {out!r}")
    return out


def owner_name() -> tuple[str, str]:
    """The current repo as (owner, name)."""
    owner, name = repo_slug().split("/", 1)
    return owner, name
