#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Sync each plan folder's issue-body.md against its live GitHub issue body.

Each _meta/plans/<slug>/ folder may hold an `issue-body.md` -- the intended issue
body, the source of truth on the desk where it stays tidy and reviewable. The live
GitHub issue body drifts from it (edited in the web UI, or never seeded). This
script catches that drift in one direction or fixes it in either, reusing
reconcile.py's `plan_body_issue()` (tracking issue) and `disk_folders()` (folders).
No third-party deps -- stdlib diff + `gh`.

Usage (from anywhere):
    uv run _meta/plans/sync-bodies.py          # READ-ONLY drift report (default)
    ./_meta/plans/sync-bodies.py --json        # machine-readable per-folder status
    ./_meta/plans/sync-bodies.py --push        # write local issue-body.md -> GitHub
    ./_meta/plans/sync-bodies.py --pull        # write live GitHub body -> issue-body.md

Exit code: default exits 1 when any folder DIFFERS (drift), 0 when clean -- so it
can gate a wave. --push/--pull exit 0 after acting.
"""

from __future__ import annotations

import difflib
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from reconcile import PLANS_DIR, disk_folders, plan_body_issue  # noqa: E402  (sibling; path set above)

DIFF_CAP = 20  # max unified-diff lines printed per DIFFERS folder


def normalize(body: str) -> str:
    """Strip trailing whitespace per line + collapse to a single trailing newline,
    so cosmetic line-ending / blank-tail differences don't fire a spurious diff."""
    lines = [line.rstrip() for line in body.replace("\r\n", "\n").split("\n")]
    return "\n".join(lines).rstrip("\n") + "\n"


def fetch_live_body(issue: int) -> str:
    """The live GitHub issue body (normalized) for one issue."""
    out = subprocess.run(
        ["gh", "issue", "view", str(issue), "--json", "body", "--jq", ".body"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return normalize(out)


def classify(slug: str) -> dict:
    """Per-folder record: {slug, issue, status, path, local, remote}. status is one of
    in-sync / DIFFERS / no issue-body.md / no tracking issue."""
    issue, _ = plan_body_issue(slug)
    body_path = PLANS_DIR / slug / "issue-body.md"
    rec: dict = {"slug": slug, "issue": issue, "path": body_path}
    if issue is None:
        rec["status"] = "no tracking issue"
        return rec
    if not body_path.is_file():
        rec["status"] = "no issue-body.md"
        return rec
    local = normalize(body_path.read_text())
    remote = fetch_live_body(issue)
    rec["local"], rec["remote"] = local, remote
    rec["status"] = "in-sync" if local == remote else "DIFFERS"
    return rec


def short_diff(local: str, remote: str) -> list[str]:
    """A unified diff (local 'issue-body.md' vs remote 'github'), capped to DIFF_CAP."""
    diff = difflib.unified_diff(
        local.splitlines(),
        remote.splitlines(),
        fromfile="issue-body.md",
        tofile="github",
        lineterm="",
    )
    lines = list(diff)
    if len(lines) > DIFF_CAP:
        lines = lines[:DIFF_CAP] + [f"... ({len(lines) - DIFF_CAP} more diff lines)"]
    return lines


def push(rec: dict) -> None:
    """Write the local issue-body.md to the GitHub issue (the only mutating path)."""
    subprocess.run(
        ["gh", "issue", "edit", str(rec["issue"]), "--body-file", str(rec["path"])],
        check=True,
    )
    print(f"  PUSH  {rec['slug']:<30} #{rec['issue']} <- issue-body.md")


def pull(rec: dict) -> None:
    """Write the live GitHub body into issue-body.md (seeds/refreshes the local file)."""
    rec["path"].write_text(fetch_live_body(rec["issue"]))
    print(f"  PULL  {rec['slug']:<30} #{rec['issue']} -> issue-body.md")


def report(records: list[dict]) -> int:
    """Print per-folder status + summary; exit 1 on any DIFFERS, else 0."""
    in_sync = sum(r["status"] == "in-sync" for r in records)
    differ = sum(r["status"] == "DIFFERS" for r in records)
    missing = sum(r["status"] == "no issue-body.md" for r in records)
    print(f"Sync _meta/plans/*/issue-body.md vs GitHub  ({len(records)} folders)\n")
    for r in records:
        issue = f"#{r['issue']}" if r["issue"] else "--"
        print(f"  {r['status']:<16} {r['slug']:<30} {issue}")
        if r["status"] == "DIFFERS":
            for line in short_diff(r["local"], r["remote"]):
                print(f"        {line}")
    print(f"\n{in_sync} in-sync, {differ} differ, {missing} missing issue-body.md")
    return 1 if differ else 0


def main() -> int:
    args = sys.argv[1:]
    records = [classify(slug) for slug in sorted(disk_folders())]

    if "--push" in args:
        for r in records:
            if r["status"] == "DIFFERS":
                push(r)
        return 0
    if "--pull" in args:
        for r in records:
            if r["issue"] is not None:
                pull(r)
        return 0
    if "--json" in args:
        print(
            json.dumps(
                [
                    {"slug": r["slug"], "issue": r["issue"], "status": r["status"]}
                    for r in records
                ],
                indent=2,
            )
        )
        return 1 if any(r["status"] == "DIFFERS" for r in records) else 0
    return report(records)


if __name__ == "__main__":
    sys.exit(main())
