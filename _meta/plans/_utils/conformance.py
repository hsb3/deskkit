#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Audit every OPEN issue's body against its issue-template's required sections.

The repo's `.github/ISSUE_TEMPLATE/` prescribes:
  - feature.md / bug.md (non-epic): an "Acceptance criteria" section AND a
    "Dependencies & gates" section.
  - epic.md (epic-type): a "Close when" section (its acceptance equivalent).

This flags issues that drift from those (and issues with no template structure at
all), so the "every non-epic issue is conformant with meaningful acceptance criteria"
rule has a guard instead of a manual read. Heading matching is tolerant (an issue may
say "Acceptance" or "Definition of done"); it checks the load-bearing sections, not
every heading. No third-party deps; one batched `gh` call.

Usage:
    uv run _meta/plans/conformance.py          # or: python3 ...
    python3 _meta/plans/conformance.py --json   # machine-readable

Exit 1 when any open issue is non-conformant, 0 when all conform.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys

HEADING_RE = re.compile(r"^#{1,6}\s+(.+?)\s*$", re.M)
AC_RE = re.compile(r"accept|definition of done|done when", re.I)
GATE_RE = re.compile(r"gate|depend", re.I)
CLOSE_RE = re.compile(r"close when|close criteria|done / foundation", re.I)
# An issue is epic-type (judged by "Close when", not "Acceptance criteria") if it
# carries the `epic` label or reads as a tracker.
EPIC_TITLE_RE = re.compile(r"tracking:|^epic\b|\U0001F4CC", re.I)

# `gh issue list` page cap; warn if a call saturates it (results may be truncated).
ISSUE_LIST_LIMIT = 1000


def fetch_open_issues() -> list[dict]:
    out = subprocess.run(
        [
            "gh",
            "issue",
            "list",
            "--state",
            "open",
            "--limit",
            str(ISSUE_LIST_LIMIT),
            "--json",
            "number,title,labels,body",
        ],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    items = json.loads(out)
    if len(items) >= ISSUE_LIST_LIMIT:
        print(
            f"WARNING: issue list hit the {ISSUE_LIST_LIMIT}-issue limit; "
            "results may be incomplete",
            file=sys.stderr,
        )
    return items


def is_epic_type(issue: dict) -> bool:
    labels = {lbl["name"] for lbl in issue.get("labels", [])}
    return "epic" in labels or bool(EPIC_TITLE_RE.search(issue.get("title", "")))


def audit_issue(issue: dict) -> list[str]:
    """Return the list of missing required sections ([] = conformant)."""
    headings = HEADING_RE.findall(issue.get("body") or "")
    blob = "\n".join(headings)
    if not headings:
        return ["no headings / no template structure"]
    if is_epic_type(issue):
        return [] if CLOSE_RE.search(blob) else ["Close when (epic acceptance)"]
    missing = []
    if not AC_RE.search(blob):
        missing.append("Acceptance criteria")
    if not GATE_RE.search(blob):
        missing.append("Dependencies & gates")
    return missing


def severity(missing: list[str], epic: bool) -> str:
    """Triage band. Missing acceptance criteria (or any structure) is the hard
    requirement -> CRITICAL; an epic with no close-condition -> EPIC; having AC but
    lacking only the gates checklist -> MINOR."""
    if any("Acceptance" in m or "no headings" in m for m in missing):
        return "CRITICAL"
    if epic:
        return "EPIC"
    return "MINOR"


def run() -> dict:
    issues = fetch_open_issues()
    bad: list[dict] = []
    for it in sorted(issues, key=lambda i: -i["number"]):
        missing = audit_issue(it)
        if missing:
            epic = is_epic_type(it)
            bad.append(
                {
                    "number": it["number"],
                    "title": it["title"],
                    "epic": epic,
                    "missing": missing,
                    "severity": severity(missing, epic),
                }
            )
    return {"total": len(issues), "bad": bad}


def main() -> int:
    result = run()
    if "--json" in sys.argv[1:]:
        print(json.dumps(result, indent=2))
        return 1 if result["bad"] else 0

    total, bad = result["total"], result["bad"]
    rate = 100 * (total - len(bad)) / total if total else 100
    print(
        f"Issue-template conformance: {total - len(bad)}/{total} "
        f"open issues conform ({rate:.0f}%)\n"
    )
    if not bad:
        print("All open issues carry their template's required sections.")
        return 0
    bands = {
        "CRITICAL": "no meaningful acceptance criteria (the hard requirement)",
        "EPIC": "epic missing a close-condition",
        "MINOR": "has AC; missing only the dependencies/gates checklist",
    }
    for band, blurb in bands.items():
        rows = [b for b in bad if b["severity"] == band]
        if not rows:
            continue
        print(f"{band} - {blurb}  ({len(rows)})")
        for b in rows:
            print(
                f"  #{b['number']:<4} missing: {', '.join(b['missing'])}"
                f"  -  {b['title'][:50]}"
            )
        print()
    print(
        f"{len(bad)} non-conformant. Conform to the matching "
        f".github/ISSUE_TEMPLATE option, then re-run."
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
