# deskkit — the canonical repo task interface. `make help` (the default) lists targets.
#
# Single-writer rule: the one-shot subcommands (sweep, patrol, propose-fix, findings/summary/
# adoption/orphans/uncollapsed, apply-fix) open the on-disk SQLite store directly, so they must
# NOT run while `serve` is up against the same store — run `make stop` first. `verify` manages
# its own throwaway store home and is safe to run at any time.
#
# These targets pass no --dir, so each command resolves its store to
# $XDG_DATA_HOME/deskkit/<DESK_NAME>/ and needs DESK_NAME resolvable (env or the desk's profile).
#
# `apply-fix` is deliberately NOT a target: committing a fix to a REAL desk is supervised-only,
# run by hand as `./deskkit apply-fix --run <run_id>` (optionally under sandbox-exec). Its write
# path is exercised only inside `verify`.
#
# `example-agent-loop` drives the real in-binary agent loop against a live API key, so it MAKES
# REAL BILLED LLM CALLS and stays out of verify/check/CI. `examples/pm-walkthrough.sh` is the
# free offline counterpart and has no target: run it directly.
.DEFAULT_GOAL := help
SHELL := /bin/bash

# The root VERSION file is the single source of truth; the binary is stamped from it, so a bare
# `go build ./...` that skips these ldflags leaves the default "dev".
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

BIN     := deskkit
PIDFILE := .deskkit.pid
LOGFILE := serve.log

# Install location, mirroring install.sh's default (no root needed); override with
# `make install PREFIX=/usr/local`.
PREFIX ?= $(HOME)/.local
BINDIR := $(PREFIX)/bin

.PHONY: help setup spa build install gui serve stop sweep patrol propose-fix findings summary \
        adoption orphans uncollapsed test vet fmt check shellcheck verify e2e spa-verify example-agent-loop \
        package media clean version-status release-prep

help: ## List targets
	@# 0-9 in the class is load-bearing: without it `e2e` never appears in the listing.
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

setup: ## Install the git hooks (lefthook install)
	@lefthook install
	@echo "setup: lefthook pre-commit hooks installed."

spa: ## Build the web/ SPA into internal/core/spa/dist (go:embed source; .gitkeep preserved)
	@if [ ! -d web/node_modules ]; then cd web && npm ci; fi
	@find internal/core/spa/dist -mindepth 1 -not -name .gitkeep -delete
	@cd web && npm run build

build: spa ## Build the SPA (web/) + the deskkit binary (./deskkit), version-stamped from VERSION
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/deskkit

install: build ## Build + install the version-stamped deskkit binary to $(PREFIX)/bin (override PREFIX=/usr/local)
	@mkdir -p "$(BINDIR)"
	@install -m 0755 $(BIN) "$(BINDIR)/$(BIN)"
	@echo "installed $(BIN) $(VERSION) -> $(BINDIR)/$(BIN)"
	@command -v $(BIN) >/dev/null 2>&1 || echo "note: $(BINDIR) is not on your PATH — add it to use '$(BIN)' as a bare command."

gui: build ## Serve the DB and open the admin GUI in a browser
	@./$(BIN) gui

serve: build ## Start PocketBase serve in the background (pidfile-tracked; see `stop`)
	@if [ -f $(PIDFILE) ] && kill -0 "$$(cat $(PIDFILE))" 2>/dev/null; then \
	  echo "serve already running (pid $$(cat $(PIDFILE)))"; \
	else \
	  nohup ./$(BIN) serve > $(LOGFILE) 2>&1 & echo $$! > $(PIDFILE); \
	  sleep 1; \
	  echo "serve started (pid $$(cat $(PIDFILE))); logs: $(LOGFILE)"; \
	fi

stop: ## Stop the running serve process (via its pid file) — required before any one-shot write command
	@if [ -f $(PIDFILE) ]; then \
	  if kill "$$(cat $(PIDFILE))" 2>/dev/null; then echo "serve stopped (pid $$(cat $(PIDFILE)))"; \
	  else echo "no process for recorded pid (stale pidfile)"; fi; \
	  rm -f $(PIDFILE); \
	else \
	  pkill -f './$(BIN) serve' 2>/dev/null && echo "serve stopped" || echo "no serve running"; \
	fi

sweep: build ## Re-index the desk tree into the files collection (read-only; not while serve is up)
	@./$(BIN) sweep

patrol: build ## Dry-run rule patrol R1-R6 -> findings + one patrol_log row (never writes files)
	@./$(BIN) patrol

propose-fix: build ## Plan mechanical fixes (R1/R2/R3) and record originals to revisions (no fs writes)
	@./$(BIN) propose-fix

findings: build ## Show open findings, grouped by rule (query findings)
	@./$(BIN) query findings

summary: build ## Show the file + open-findings summary counts (query summary)
	@./$(BIN) query summary

adoption: build ## Show the apply-fix adoption log (query adoption)
	@./$(BIN) query adoption

orphans: build ## Show .md files with no doctype / not under a meta prefix (query orphans)
	@./$(BIN) query orphans

uncollapsed: build ## Show open R5 judgment findings — graduated-but-not-collapsed docs (query uncollapsed)
	@./$(BIN) query uncollapsed

test: ## Fast unit tests: go test ./...
	@go test ./...

vet: ## go vet ./...
	@go vet ./...

fmt: ## Check gofmt formatting (fails and lists files if any need formatting)
	@bash -c 'out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; exit 1; fi'

check: ## Repo gates: neutrality lint + self-test, kit-manifest drift, scaffold frontmatter, persona drift, textfield-max, query-kind drift + self-test, doc-link integrity + self-test, shellcheck, actionlint, workflow SHA-pin drift + self-test, profile-root drift + self-test
	@node scripts/check-neutrality.mjs
	@node scripts/check-neutrality.mjs --self-test
	@node scripts/check-kits.mjs
	@node scripts/check-scaffold-frontmatter.mjs
	@node scripts/check-persona-drift.mjs
	@node scripts/check-textfield-max.mjs
	@node scripts/check-query-kinds.mjs
	@node scripts/check-query-kinds.mjs --self-test
	@node scripts/check-doc-links.mjs
	@node scripts/check-doc-links.mjs --self-test
	@$(MAKE) --no-print-directory shellcheck
	@actionlint
	@node scripts/check-workflow-pins.mjs
	@node scripts/check-workflow-pins.mjs --self-test
	@node scripts/check-profile-root.mjs
	@node scripts/check-profile-root.mjs --self-test

shellcheck: ## Lint every shell entry point (the single list — CI and the release gate call this target)
	@shellcheck install.sh docker-entrypoint.sh verify.sh examples/*.sh sandbox/*.sh scripts/record-media.sh scripts/docker-smoke.sh e2e/e2e.sh e2e/lib.sh e2e/steps/*.sh e2e/spa/run.sh

verify: ## Run the Phase-1 verify gate against a throwaway scratch desk (never a real store)
	@bash verify.sh

e2e: ## Run the end-to-end system-behaviour suite (whole system, throwaway desk; offline, no LLM key)
	@bash e2e/e2e.sh

spa-verify: ## Drive the embedded SPA in a real browser against a throwaway desk (needs playwright; see e2e/spa/README.md)
	@bash e2e/spa/run.sh

example-agent-loop: build ## Manual walkthrough: drive the REAL agent LLM loop end-to-end (REAL BILLED CALLS; needs ANTHROPIC_API_KEY; never in CI)
	@bash examples/agent-loop.sh

package: ## No-op: the marketplace bundle under plugins/ is authored in place, nothing is generated
	@echo "package: nothing to generate — the marketplace bundle under plugins/ is authored in place."

media: ## Record the demo media assets (scripts/record-media.sh)
	@bash scripts/record-media.sh

clean: stop ## Stop serve + remove build artifacts (binary, pidfile, serve log)
	@rm -rf $(BIN) $(PIDFILE) $(LOGFILE)
	@rm -rf pb_data
	@echo "cleaned: binary, pidfile, serve log (+ any legacy ./pb_data)"
	@echo "note: the canonical per-desk store lives at \$$XDG_DATA_HOME/deskkit/<DESK_NAME>/"
	@echo "      (fallback ~/.local/share/...); it is persistent and NOT removed by clean — rm it by hand."

version-status: ## Advisory (non-blocking): unreleased product changes since the last tag vs VERSION
	@node scripts/check-version-status.mjs

release-prep: ## Pre-tag gate: assert clean main, run check+test, print the tag/push commands (no auto-tag)
	@if [ -n "$$(git status --porcelain)" ]; then \
	  echo "release-prep: working tree is not clean — commit or stash first." >&2; exit 1; fi
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	  if [ "$$branch" != "main" ]; then \
	    echo "release-prep: on branch '$$branch', not main — release from main." >&2; exit 1; fi
	@node scripts/check-version-sync.mjs
	@node scripts/check-changelog.mjs
	@node scripts/check-version-status.mjs
	@$(MAKE) check
	@$(MAKE) test
	@echo ""
	@echo "release-prep: OK — gates green at v$(VERSION). To cut the release, run:"
	@echo "    git tag v$(VERSION) && git push --tags"
	@echo "(the release workflow asserts the tag matches VERSION, re-runs the gates, and publishes the binaries.)"
