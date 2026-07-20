#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Reconcile the _meta/plans/README.md status table against live GitHub issue
state and the plan folders on disk.

The README table is hand-maintained, so it drifts: a plan stays in the ACTIVE
section after its tracking issue closes, a folder appears with no row, the row
names a different issue than the plan body. This script catches all of that in
one batched `gh` call. No third-party deps.

Usage (from anywhere):
    uv run _meta/plans/reconcile.py          # or: python3 _meta/plans/reconcile.py
    ./_meta/plans/reconcile.py --json        # machine-readable drift list

Exit code is 1 when any drift is found, 0 when clean -- so it can gate a wave.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

# This script lives in _meta/plans/_utils/; the plans desk it scans is the parent dir.
PLANS_DIR = Path(__file__).resolve().parent.parent
README = PLANS_DIR / "README.md"
ISSUE_RE = re.compile(r"#(\d+)")

# Section markers in README.md (substring match on the line).
ACTIVE_MARKER = "ACTIVE plans"
ARCHIVED_MARKER = "ARCHIVED ("

# `gh issue list` page cap; warn if a call saturates it (results may be truncated).
ISSUE_LIST_LIMIT = 1000


def parse_readme_rows() -> tuple[list[dict], list[dict]]:
    """Return (active_rows, archived_rows). Each row: {plan, issues:[int], raw}."""
    active: list[dict] = []
    archived: list[dict] = []
    mode: str | None = None
    for line in README.read_text().splitlines():
        if ACTIVE_MARKER in line:
            mode = "active"
            continue
        if ARCHIVED_MARKER in line:
            mode = "archived"
            continue
        if mode is None or not line.lstrip().startswith("|"):
            continue
        cells = [c.strip() for c in line.strip().strip("|").split("|")]
        if len(cells) < 2:
            continue
        plan, issue_cell = cells[0], cells[1]
        # Skip the header and separator rows.
        if plan in ("Plan", "Plan (archived path)") or set(plan) <= {"-", " ", ":"}:
            continue
        issues = [int(n) for n in ISSUE_RE.findall(issue_cell)]
        (active if mode == "active" else archived).append(
            {"plan": plan, "issues": issues, "issue_cell": issue_cell}
        )
    return active, archived


# Words that, when they precede the first `#` in the Tracking section, mean the
# leading issue is an EPIC/relation, not this plan's tracking issue -- which would
# make the "first #" heuristic grab the wrong number. The convention is: the
# tracking issue comes FIRST, epics/relations after. We warn if that's violated.
EPIC_SIGNAL = re.compile(r"\b(epic|under|child of|relates?|sibling)\b", re.I)


def plan_body_issue(slug: str) -> tuple[int | None, str | None]:
    """(tracking issue, warning). The tracking issue is the first `#NNN` in the
    plan.md `## Tracking` section. Warn (non-fatal) if the section is missing or
    the leading issue looks epic-flagged -- i.e. the convention may be violated and
    the parse should be double-checked against the README issue."""
    plan_md = PLANS_DIR / slug / "plan.md"
    if not plan_md.is_file():
        return None, "no plan.md"
    text = plan_md.read_text()
    m = re.search(r"^##\s*Tracking\b.*?(?=^##\s|\Z)", text, re.S | re.M)
    if not m:
        return None, "no `## Tracking` section"
    section = m.group(0)
    found = ISSUE_RE.search(section)
    if not found:
        return None, "`## Tracking` names no issue"
    issue = int(found.group(1))
    # Is the leading `#` preceded by an epic-signal word on the same line?
    line_start = section.rfind("\n", 0, found.start()) + 1
    if EPIC_SIGNAL.search(section[line_start : found.start()]):
        return (
            issue,
            f"leading issue #{issue} looks epic-flagged; verify it is the tracking issue",
        )
    return issue, None


def fetch_issue_states() -> dict[int, str]:
    """number -> 'OPEN'|'CLOSED' for every issue in the repo (one gh call)."""
    out = subprocess.run(
        [
            "gh",
            "issue",
            "list",
            "--state",
            "all",
            "--limit",
            str(ISSUE_LIST_LIMIT),
            "--json",
            "number,state",
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
    return {it["number"]: it["state"] for it in items}


def disk_folders() -> set[str]:
    return {
        p.name for p in PLANS_DIR.iterdir() if p.is_dir() and (p / "plan.md").is_file()
    }


def reconcile() -> dict:
    active, archived = parse_readme_rows()
    states = fetch_issue_states()
    folders = disk_folders()
    drift: list[dict] = []
    warns: list[dict] = []
    ok: list[str] = []

    active_slugs = {r["plan"] for r in active}

    for row in active:
        slug, issues = row["plan"], row["issues"]
        tracking = issues[0] if issues else None
        # 1. tracking issue should be OPEN
        if tracking is None:
            drift.append(
                {
                    "kind": "active-no-issue",
                    "plan": slug,
                    "detail": "ACTIVE row names no issue number",
                }
            )
        elif tracking not in states:
            drift.append(
                {
                    "kind": "issue-missing",
                    "plan": slug,
                    "issue": tracking,
                    "detail": f"#{tracking} not found in repo",
                }
            )
        elif states[tracking] == "CLOSED":
            drift.append(
                {
                    "kind": "active-but-closed",
                    "plan": slug,
                    "issue": tracking,
                    "detail": f"#{tracking} is CLOSED -> human decision: "
                    "reopen the issue (criteria unmet) OR archive the plan (done)",
                }
            )
        else:
            ok.append(f"{slug:<30} #{tracking} OPEN")
        # 2. folder must exist on disk
        if slug not in folders:
            drift.append(
                {
                    "kind": "row-no-folder",
                    "plan": slug,
                    "detail": "ACTIVE row has no plan folder on disk",
                }
            )
        # 3. README issue vs plan.md Tracking issue
        body, warn = plan_body_issue(slug)
        if warn:
            warns.append({"plan": slug, "detail": warn})
        if tracking and body and body != tracking:
            drift.append(
                {
                    "kind": "body-mismatch",
                    "plan": slug,
                    "issue": tracking,
                    "detail": f"README says #{tracking}, plan.md says #{body}",
                }
            )

    for row in archived:
        slug, issues = row["plan"], row["issues"]
        first = issues[0] if issues else None
        if first is None:  # 'enablement (no issue)', 'repo hygiene' -- nothing to check
            continue
        if first not in states:
            drift.append(
                {
                    "kind": "issue-missing",
                    "plan": slug,
                    "issue": first,
                    "detail": f"#{first} not found in repo",
                }
            )
        elif states[first] == "OPEN":
            drift.append(
                {
                    "kind": "archived-but-open",
                    "plan": slug,
                    "issue": first,
                    "detail": f"#{first} is OPEN but plan is archived",
                }
            )

    # 4. disk folders with no ACTIVE row
    for slug in sorted(folders - active_slugs):
        drift.append(
            {
                "kind": "folder-no-row",
                "plan": slug,
                "detail": "plan folder on disk has no ACTIVE README row",
            }
        )

    return {
        "ok": ok,
        "drift": drift,
        "warns": warns,
        "counts": {
            "active": len(active),
            "archived": len(archived),
            "folders": len(folders),
        },
    }


def main() -> int:
    result = reconcile()
    if "--json" in sys.argv[1:]:
        print(json.dumps(result, indent=2))
        return 1 if result["drift"] else 0

    c = result["counts"]
    print(
        f"Reconcile _meta/plans/README.md vs GitHub  "
        f"({c['active']} active rows, {c['archived']} archived, "
        f"{c['folders']} folders)\n"
    )
    for line in result["ok"]:
        print(f"  OK   {line}")
    for w in result.get("warns", []):
        print(f"  WARN {w['plan']}: {w['detail']}")
    if not result["drift"]:
        print("\nNo drift. README table agrees with live issue state and disk.")
        return 0
    print(f"\n{len(result['drift'])} drift item(s):")
    for d in result["drift"]:
        print(f"  DRIFT [{d['kind']}] {d['plan']}: {d['detail']}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
