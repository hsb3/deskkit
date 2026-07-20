#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Report which OPEN issues have NO plan folder in _meta/plans/ -- the planning
backlog.

reconcile.py answers "do the plans on disk agree with the README and live issue
state?". This script answers the inverse: "which open work has no plan at all?".
An issue with a plan folder is covered; everything else (minus epics/trackers) is
unplanned and shows up here.

Per an owner decision only NON-TRIVIAL work strictly needs a plan -- trivial
bugs/docs may be waived -- so this REPORTS and TAGS likely-trivial items rather
than auto-excluding them; the owner waives them by judgment.

Logic is reused from the sibling scripts in this directory: reconcile.py's
`disk_folders()` + `plan_body_issue()` map each plan folder to its tracking issue
(the "planned" set), and conformance.py's `is_epic_type()` drops epics/trackers.

Usage (from anywhere):
    uv run _meta/plans/coverage.py          # or: python3 _meta/plans/coverage.py
    ./_meta/plans/coverage.py --json        # machine-readable backlog list

Exit code is 1 when any open non-epic issue has no plan, 0 when all are covered.
"""

from __future__ import annotations

import importlib.util
import json
import re
import subprocess
import sys
from pathlib import Path

PLANS_DIR = Path(__file__).resolve().parent


def _load_sibling(name: str):
    """Import a sibling single-file script by path so its logic can be reused."""
    spec = importlib.util.spec_from_file_location(name, PLANS_DIR / f"{name}.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


reconcile = _load_sibling("reconcile")
conformance = _load_sibling("conformance")

# A `bug` is tagged maybe-trivial only when its title looks small (no broad
# scope words). This is advisory -- the owner makes the actual waive call.
BIG_TITLE_RE = re.compile(
    r"\b(refactor|redesign|migrat|architect|pipeline|engine|epic|overhaul|rewrite)\b",
    re.I,
)


def planned_issue_numbers() -> set[int]:
    """The set of tracking issue numbers that already have a plan folder."""
    planned: set[int] = set()
    for slug in reconcile.disk_folders():
        issue, _warn = reconcile.plan_body_issue(slug)
        if issue is not None:
            planned.add(issue)
    return planned


def fetch_open_issues() -> list[dict]:
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
            "number,title,labels,milestone",
        ],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return json.loads(out)


def maybe_trivial(issue: dict) -> bool:
    """Advisory tag: a docs issue, or a small-looking bug, may be waivable."""
    labels = {lbl["name"] for lbl in issue.get("labels", [])}
    if "documentation" in labels:
        return True
    if "bug" in labels and not BIG_TITLE_RE.search(issue.get("title", "")):
        return True
    return False


def compute() -> dict:
    """Pure compute: open non-epic issues with no plan folder, tagged + grouped."""
    planned = planned_issue_numbers()
    issues = fetch_open_issues()
    non_epic = [it for it in issues if not conformance.is_epic_type(it)]
    unplanned: list[dict] = []
    for it in sorted(non_epic, key=lambda i: -i["number"]):
        if it["number"] in planned:
            continue
        milestone = (it.get("milestone") or {}).get("title") or "(no milestone)"
        unplanned.append(
            {
                "number": it["number"],
                "title": it["title"],
                "milestone": milestone,
                "maybe_trivial": maybe_trivial(it),
            }
        )
    return {
        "planned_count": len(planned),
        "non_epic_count": len(non_epic),
        "unplanned": unplanned,
    }


def _grouped(unplanned: list[dict]) -> dict[str, list[dict]]:
    groups: dict[str, list[dict]] = {}
    for u in unplanned:
        groups.setdefault(u["milestone"], []).append(u)
    return groups


def main() -> int:
    result = compute()
    if "--json" in sys.argv[1:]:
        print(json.dumps(result, indent=2))
        return 1 if result["unplanned"] else 0

    unplanned = result["unplanned"]
    total = result["non_epic_count"]
    trivial = sum(1 for u in unplanned if u["maybe_trivial"])
    print(
        f"Plan coverage: {len(unplanned)} of {total} open non-epic issues have "
        f"no plan ({trivial} tagged maybe-trivial)\n"
    )
    if not unplanned:
        print("Every open non-epic issue has a plan folder. Backlog is covered.")
        return 0

    groups = _grouped(unplanned)
    # (no milestone) last; everything else alphabetical.
    order = sorted(groups, key=lambda m: (m == "(no milestone)", m))
    for milestone in order:
        rows = groups[milestone]
        print(f"{milestone}  ({len(rows)})")
        for u in rows:
            tag = "[trivial?]" if u["maybe_trivial"] else "          "
            print(f"  #{u['number']:<4} {tag}  {u['title'][:54]:<54}  -  {milestone}")
        print()
    print(
        f"{len(unplanned)} open issue(s) have no plan. Draft a plan folder under "
        f"_meta/plans/ for each non-trivial one (trivial tags are advisory)."
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
