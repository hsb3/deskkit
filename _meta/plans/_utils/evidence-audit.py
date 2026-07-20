#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Flag recently CLOSED issues that closed WITHOUT evidence of acceptance.

The convention is "review the acceptance criteria + attach evidence before you
close." This is the AUDIT backstop for that rule: it lists closed issues with
neither a linked PR nor an evidence-bearing comment, so a sloppy close surfaces
after the fact.

"Evidence" detection is HEURISTIC: a closed-by PR reference is the strong signal;
comment-text matching (evidence/verified/proof/acceptance/screenshot/links/...)
is best-effort and will both miss prose evidence and over-credit a passing
mention. Treat a flag as "go look", not "proven unverified". This is an
after-the-fact backstop, not a hard gate. The repo is the one `gh` is pointed at,
so this drops into any project.

Usage (from anywhere):
    uv run _meta/plans/_utils/evidence-audit.py                # last 40 closed
    ./_meta/plans/_utils/evidence-audit.py --limit 80          # widen the window
    ./_meta/plans/_utils/evidence-audit.py --since 2026-06-10  # closed on/after a date
    ./_meta/plans/_utils/evidence-audit.py --json              # machine-readable

Exit code is 1 when any closed issue lacks evidence, 0 when all are covered --
so it can gate a wave.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys

DEFAULT_LIMIT = 40
# Best-effort evidence signals in a comment body. A linked PR is the strong
# signal handled separately; this is the soft fallback.
EVIDENCE_RE = re.compile(
    r"evidence|verified|proof|acceptance|screenshot|attached|merged in|PR #|https?://",
    re.I,
)


def fetch_closed_issues(limit: int) -> list[dict]:
    """Recently closed issues, newest first (one batched gh call against the
    current repo -- no --repo flag, so it follows the working tree like the
    sibling scripts)."""
    out = subprocess.run(
        [
            "gh",
            "issue",
            "list",
            "--state",
            "closed",
            "--limit",
            str(limit),
            "--json",
            "number,title,closedAt,closedByPullRequestsReferences,comments",
        ],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return json.loads(out)


def has_evidence_comment(issue: dict) -> bool:
    return any(
        EVIDENCE_RE.search(c.get("body") or "") for c in issue.get("comments") or []
    )


def closed_with_evidence(issue: dict) -> bool:
    """Strong signal: a closing PR. Soft signal: an evidence-bearing comment."""
    has_pr = bool(issue.get("closedByPullRequestsReferences"))
    return has_pr or has_evidence_comment(issue)


def audit(issues: list[dict], since: str | None) -> dict:
    """Compute the flagged list. `since` filters by closedAt date (YYYY-MM-DD)."""
    checked: list[dict] = []
    flagged: list[dict] = []
    for it in issues:
        closed_at = it.get("closedAt") or ""
        if since and closed_at[:10] < since:
            continue
        checked.append(it)
        if not closed_with_evidence(it):
            flagged.append(
                {
                    "number": it["number"],
                    "title": it["title"],
                    "closedAt": closed_at,
                }
            )
    flagged.sort(key=lambda f: -f["number"])
    return {"checked": len(checked), "flagged": flagged}


def parse_args(argv: list[str]) -> tuple[int, str | None, bool]:
    limit, since, as_json = DEFAULT_LIMIT, None, False
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg == "--json":
            as_json = True
        elif arg == "--limit":
            i += 1
            if i >= len(argv):
                print("--limit requires a value", file=sys.stderr)
                sys.exit(2)
            try:
                limit = int(argv[i])
            except ValueError:
                print(f"--limit requires an integer, got: {argv[i]!r}", file=sys.stderr)
                sys.exit(2)
        elif arg == "--since":
            i += 1
            if i >= len(argv):
                print("--since requires a value", file=sys.stderr)
                sys.exit(2)
            since = argv[i]
        else:
            print(f"unknown arg: {arg}", file=sys.stderr)
            sys.exit(2)
        i += 1
    return limit, since, as_json


def main() -> int:
    limit, since, as_json = parse_args(sys.argv[1:])
    result = audit(fetch_closed_issues(limit), since)
    if as_json:
        print(json.dumps(result, indent=2))
        return 1 if result["flagged"] else 0

    checked, flagged = result["checked"], result["flagged"]
    scope = f" closed on/after {since}" if since else f" (last {limit} closed)"
    print(
        f"Evidence audit: {len(flagged)} of {checked} recently-closed "
        f"issues lack attached evidence{scope}.\n"
    )
    if not flagged:
        print("Every closed issue has a linked PR or an evidence-bearing comment.")
        return 0
    for f in flagged:
        date = (f["closedAt"] or "")[:10] or "?"
        print(f"  #{f['number']:<4} closed {date}  -  {f['title'][:60]}")
    print(
        "\nHeuristic: a linked PR is the strong signal; comment matching is "
        "best-effort. Review each, attach evidence, then re-run."
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
