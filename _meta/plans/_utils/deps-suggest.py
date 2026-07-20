#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Harvest CANDIDATE native "blocked-by" dependency edges from open-issue bodies.

GitHub-native blocked-by edges are entered by hand and lag the prose: an issue
body says "blocked by #NNN" or "depends on #NNN" while the native edge was never
created. This script reads every open issue body, finds dependency-signal phrases
that precede a `#NNN`, and proposes the edges as candidates for a human to confirm.

It is strictly READ-ONLY. It NEVER writes an edge. Writing is a separate, confirmed
step done per edge via:
    gh api --method POST repos/<owner>/<name>/issues/{n}/dependencies/blocked_by \
        -F issue_id=<blocker-issue-id>

Candidates already present as native blockedBy edges are excluded, as are self-refs
and parent/epic hierarchy phrasing (those are tracking relations, not blockers). A
proposed blocker that is not itself an open issue is still listed, flagged
`(blocker not open)`, so a stale or closed reference is visible rather than dropped.
The repo is derived from `gh` (see _repo.py), so this drops into any project.

Usage (from anywhere):
    uv run _meta/plans/_utils/deps-suggest.py          # or: python3 ...
    ./_meta/plans/_utils/deps-suggest.py --json         # machine-readable candidates

Exit code is always 0 -- this is an advisory harvester, not a gate.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from _repo import owner_name, repo_slug  # noqa: E402  (sibling - repo derived from gh)

OWNER, NAME = owner_name()
REPO = repo_slug()

# A dependency signal ("blocked by", "depends on"...) immediately preceding a `#NNN`.
SIGNAL_RE = re.compile(
    r"(blocked by|depends on|depends upon|requires|needs|gated on|waiting on|after|once)"
    r"\s+#?(\d+)",
    re.I,
)
# Hierarchy phrasing near the ref means parent/epic relation, not a blocker -> skip.
EPIC_RE = re.compile(r"under epic|part of|child of|tracked by|epic |sub-issue", re.I)


def fetch_open_issues() -> list[dict]:
    """[{number, title, body}] for every OPEN issue (one batched gh call)."""
    out = subprocess.run(
        [
            "gh",
            "issue",
            "list",
            "--state",
            "open",
            "--limit",
            "1000",
            "--json",
            "number,title,body",
        ],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return json.loads(out)


def fetch_native_edges() -> set[tuple[int, int]]:
    """Existing native (blocked, blocker) edges, paginated through all open issues."""
    edges: set[tuple[int, int]] = set()
    cursor = "null"
    while True:
        query = (
            '{ repository(owner:"%s",name:"%s"){ '
            "issues(first:100,states:OPEN,after:%s){ "
            "nodes{ number blockedBy(first:20){nodes{number}} } "
            "pageInfo{hasNextPage endCursor} } } }" % (OWNER, NAME, cursor)
        )
        out = subprocess.run(
            ["gh", "api", "graphql", "-f", f"query={query}"],
            capture_output=True,
            text=True,
            check=True,
        ).stdout
        page = json.loads(out)["data"]["repository"]["issues"]
        for node in page["nodes"]:
            for blocker in node["blockedBy"]["nodes"]:
                edges.add((node["number"], blocker["number"]))
        info = page["pageInfo"]
        if not info["hasNextPage"]:
            return edges
        cursor = json.dumps(info["endCursor"])


def snippet(body: str, start: int, end: int) -> str:
    """A <=80-char single-line window of context around a match."""
    lo = max(0, start - 24)
    hi = min(len(body), end + 24)
    text = " ".join(body[lo:hi].split())
    return text[:80]


def harvest(issues: list[dict], native: set[tuple[int, int]]) -> list[dict]:
    """Pure compute: candidate edges minus native/self/epic, with open-status flag."""
    open_set = {it["number"] for it in issues}
    seen: set[tuple[int, int]] = set()
    candidates: list[dict] = []
    for it in issues:
        blocked = it["number"]
        body = it.get("body") or ""
        for m in SIGNAL_RE.finditer(body):
            blocker = int(m.group(2))
            if blocker == blocked:  # self-reference
                continue
            if (blocked, blocker) in native or (blocked, blocker) in seen:
                continue
            ctx = snippet(body, m.start(), m.end())
            if EPIC_RE.search(ctx):  # hierarchy, not a blocker
                continue
            seen.add((blocked, blocker))
            candidates.append(
                {
                    "blocked": blocked,
                    "blocker": blocker,
                    "evidence": ctx,
                    "blocker_open": blocker in open_set,
                }
            )
    candidates.sort(key=lambda c: (c["blocked"], c["blocker"]))
    return candidates


def main() -> int:
    issues = fetch_open_issues()
    native = fetch_native_edges()
    candidates = harvest(issues, native)

    if "--json" in sys.argv[1:]:
        print(json.dumps(candidates, indent=2))
        return 0

    blocked_issues = {c["blocked"] for c in candidates}
    print(
        f"Candidate blocked-by edges from open-issue bodies  "
        f"({len(candidates)} candidate edge(s) across {len(blocked_issues)} issue(s); "
        f"{len(native)} already-native edge(s) excluded)\n"
    )
    if not candidates:
        print("No new candidate edges. Prose dependencies already match native edges.")
        return 0

    last: int | None = None
    for c in candidates:
        if c["blocked"] != last:
            if last is not None:
                print()
            last = c["blocked"]
        flag = "" if c["blocker_open"] else "  (blocker not open)"
        print(
            f"  #{c['blocked']:<5} blocked-by #{c['blocker']:<5}"
            f'{flag}   (evidence: "{c["evidence"]}")'
        )
    print(
        f"\n{len(candidates)} candidate(s) -- confirm each, then write via "
        f"`gh api --method POST repos/{REPO}/issues/{{n}}/dependencies/blocked_by`."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
