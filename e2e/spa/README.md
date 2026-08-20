# SPA browser gate

`make spa-verify` — seeds a throwaway desk, serves it, and drives the embedded SPA in a real
Chromium. Numbered PASS/FAIL lines, non-zero exit on any failure.

## Why a real browser and not a DOM harness

Ruled on the board. The short version: the defects this codebase actually shipped were a
destructive action firing at the wrong level, a stolen OS shortcut, and a frontmatter key
written as `""` on every save. Only a real browser plus a look at the file afterwards catches
the third, and the third is the one that destroyed data. A harness that stops at the DOM buys
the cheap half of the coverage for the price of a dependency.

Several checks here read the file on disk after a save. That is the point of the suite, not a
detail of it.

## Prerequisite

Playwright is **not** a dependency of `web/` — the SPA build and the container image run
`npm ci` and have no use for a browser driver. Install it once, here:

```
npm install --prefix e2e/spa
npx --prefix e2e/spa playwright install chromium
```

A globally installed `playwright` is also found. If neither is present the gate **exits
non-zero** with those two commands rather than skipping: a silent skip would read as "the app
is verified" when nothing ran.

## Files

| File | What it is |
|---|---|
| `run.sh` | Orchestration: build, seed, sweep, serve on an ephemeral port, run, tear down |
| `fixture.mjs` | The fixture desk — 30 documents across all five non-meta status families |
| `checks.mjs` | The assertions, against the browser and against the bytes on disk |

## Two things that will bite you

**`synopsis` comes from frontmatter, not from the body.** A fixture document without a
`synopsis:` key renders no preview line at all, so the "rows are fat enough to preview" check
fails against a perfectly working app. Every fixture document carries one on purpose.

**Re-pressing the mode you are already in is a no-op**, because the level reset keys on the mode
id. It therefore cannot bring the finder back from inside an edit — `⌘B` is the toggle that can.
`openDoc()` in `checks.mjs` uses `⌘B` for exactly this reason.

## When behaviour is ruled to change

The write-boundary check asserts *today's* behaviour: a save into `_structure/decisions/` is
refused. A ruled change makes a decision's structured fields writable while its prose body stays
protected. When that lands, that check inverts — the status save should succeed with a
byte-identical body, and the refusal check moves to a write that would alter the body. The
comment above it in `checks.mjs` says so too.
