#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Sequence all OPEN issues into a tiered work board (NOW / NEXT / BLOCKED / DEFERRED).

There is no single true linear order, so this emits TIERS, not a 1..N rank - honest
about the imperfection. The signals already in the repo do the work:
  - dated milestones = urgency
  - native `blockedBy` dependency edges = the BLOCKED tier (real GitHub data; see
    `deps-suggest.py` to populate them)
  - `gate:*` labels = promise-level priority (secondary sort)
  - `subIssues` / `blocking` edges = leverage (an issue that unblocks many ranks up)
  - a plan folder in `_meta/plans/` (via reconcile.py) = ready to build now
  - `on-hold:*` labels = DEFERRED

Tiers:
  NOW      - on a dated milestone AND a plan exists AND not blocked. Build these.
  NEXT     - everything actionable that is not NOW (off the deadline, or needs a plan
             first). Grouped by milestone, sorted by the leverage/gate/plan keys.
  BLOCKED  - has an OPEN native blocked-by edge. Listed with its blockers.
  DEFERRED - carries an `on-hold:*` label.

It is a generated VIEW - never hand-maintained. Regenerate after any state change.
The repo is derived from `gh` (see _repo.py), so this drops into any project.

Usage:
    uv run _meta/plans/_utils/sequence.py          # or: python3 ...
    python3 _meta/plans/_utils/sequence.py --json
"""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import reconcile  # noqa: E402, I001  (sibling toolkit - plan readiness; path set above)
import conformance  # noqa: E402  (sibling - is_epic_type)
from _repo import graphql, owner_name  # noqa: E402  (sibling - repo derived from gh)

OWNER, NAME = owner_name()
UNDATED = "9999-99-99"

# `gh issue list` page cap; warn if a call saturates it (results may be truncated).
ISSUE_LIST_LIMIT = 1000

# GraphQL variables (never string-interpolated) keep this injection-safe and robust
# to unexpected owner/name/cursor formats; $cursor is null on the first page.
GRAPH_QUERY = (
    "query($owner: String!, $name: String!, $cursor: String) {"
    "  repository(owner: $owner, name: $name) {"
    "    issues(first: 100, states: OPEN, after: $cursor) {"
    "      nodes {"
    "        number"
    "        blockedBy(first: 30) { nodes { number state } }"
    "        blocking(first: 30) { nodes { number } }"
    "        subIssues(first: 50) { nodes { number } }"
    "      }"
    "      pageInfo { hasNextPage endCursor }"
    "    }"
    "  }"
    "}"
)


def fetch_issues() -> list[dict]:
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
            "number,title,labels,milestone",
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


def fetch_milestone_due() -> dict[str, str]:
    """milestone title -> due date (YYYY-MM-DD), only for dated milestones."""
    out = subprocess.run(
        [
            "gh",
            "api",
            f"repos/{OWNER}/{NAME}/milestones",
            "--jq",
            ".[] | select(.due_on != null) | [.title, .due_on] | @tsv",
        ],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    due = {}
    for line in out.splitlines():
        title, due_on = line.split("\t")
        due[title] = due_on[:10]
    return due


def fetch_graph() -> dict[int, dict]:
    """number -> {blocked_by: [open #], blocking: [#], children: [#]} via GraphQL."""
    graph: dict[int, dict] = {}
    cursor: str | None = None
    while True:
        data = graphql(GRAPH_QUERY, owner=OWNER, name=NAME, cursor=cursor)[
            "data"
        ]["repository"]["issues"]
        for n in data["nodes"]:
            graph[n["number"]] = {
                "blocked_by": [
                    b["number"] for b in n["blockedBy"]["nodes"] if b["state"] == "OPEN"
                ],
                "blocking": [b["number"] for b in n["blocking"]["nodes"]],
                "children": [c["number"] for c in n["subIssues"]["nodes"]],
            }
        page = data["pageInfo"]
        if not page["hasNextPage"]:
            return graph
        cursor = page["endCursor"]


def planned_issues() -> set[int]:
    """Issue numbers that already have a plan folder (reconcile.py is the source)."""
    out = set()
    for slug in reconcile.disk_folders():
        issue, _ = reconcile.plan_body_issue(slug)
        if issue:
            out.add(issue)
    return out


def classify(issues, due, graph, planned) -> tuple[dict[str, list[dict]], list[int]]:
    tiers: dict[str, list[dict]] = {
        "NOW": [],
        "NEXT": [],
        "BLOCKED": [],
        "DEFERRED": [],
    }
    epics: list[int] = []
    for it in issues:
        # Epics/trackers are containers, not work - sequence their children, not them.
        if conformance.is_epic_type(it):
            epics.append(it["number"])
            continue
        num = it["number"]
        labels = {lbl["name"] for lbl in it.get("labels", [])}
        ms = (it.get("milestone") or {}).get("title")
        g = graph.get(num, {"blocked_by": [], "blocking": [], "children": []})
        row = {
            "number": num,
            "title": it["title"],
            "milestone": ms,
            "gates": sorted(lbl for lbl in labels if lbl.startswith("gate:")),
            "planned": num in planned,
            "blocked_by": g["blocked_by"],
            "leverage": len(g["blocking"]) + len(g["children"]),
            "due": due.get(ms, UNDATED),
        }
        if any(lbl.startswith("on-hold:") for lbl in labels):
            tiers["DEFERRED"].append(row)
        elif g["blocked_by"]:
            tiers["BLOCKED"].append(row)
        elif ms in due and row["planned"]:
            tiers["NOW"].append(row)
        else:
            tiers["NEXT"].append(row)
    return tiers, epics


def sort_key(r: dict) -> tuple:
    return (
        r["due"],
        0 if r["gates"] else 1,
        -r["leverage"],
        0 if r["planned"] else 1,
        r["number"],
    )


def fmt(r: dict) -> str:
    plan = "plan" if r["planned"] else "    "
    gates = (
        (" " + ",".join(g.replace("gate:", "@") for g in r["gates"]))
        if r["gates"]
        else ""
    )
    lev = f" ^{r['leverage']}" if r["leverage"] else ""
    blk = (
        f"  blocked-by {','.join('#' + str(b) for b in r['blocked_by'])}"
        if r["blocked_by"]
        else ""
    )
    ms = f"  - {r['milestone']}" if r["milestone"] else ""
    due = f" (due {r['due']})" if r["due"] != UNDATED else ""
    return f"  #{r['number']:<4} [{plan}]{gates}{lev}  {r['title'][:46]}{ms}{due}{blk}"


def main() -> int:
    issues = fetch_issues()
    tiers, epics = classify(
        issues, fetch_milestone_due(), fetch_graph(), planned_issues()
    )
    if "--json" in sys.argv[1:]:
        print(json.dumps({"tiers": tiers, "epics": epics}, indent=2))
        return 0
    blurbs = {
        "NOW": "dated milestone + plan ready - build these",
        "NEXT": "actionable; off the deadline or needs a plan first",
        "BLOCKED": "has an open blocked-by edge",
        "DEFERRED": "on-hold",
    }
    print(
        f"Issue sequencing - {len(issues)} open, {len(epics)} epics/trackers excluded  "
        f"([plan] = plan exists, @gate, ^N = unblocks/parents N)\n"
    )
    for tier in ("NOW", "BLOCKED", "NEXT", "DEFERRED"):
        rows = sorted(tiers[tier], key=sort_key)
        print(f"{tier} - {blurbs[tier]}  ({len(rows)})")
        for r in rows:
            print(fmt(r))
        print()
    print(
        f"EPICS (containers - sequence their children): "
        f"{' '.join('#' + str(n) for n in sorted(epics))}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
