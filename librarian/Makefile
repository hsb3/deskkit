# deskkit — task interface. `make help` lists targets.
#
# Single-writer rule (spec §2.7): the one-shot subcommands (sweep, patrol, propose-fix,
# findings/summary/adoption/orphans/uncollapsed, and the supervised `apply-fix` CLI) open
# the same on-disk SQLite store directly via native DAO. They must NOT run while `serve` is
# up against the same store — run `make stop` first. `verify` is self-contained (it manages
# its own throwaway XDG store home + scratch desk) and is safe to run at any time.
#
# Store location (ADR 0002 §2): with no --dir, every command below resolves its store to
# $XDG_DATA_HOME/deskkit/<DESK_NAME>/ (fallback ~/.local/share/...). These targets
# pass no --dir, so they use that canonical per-desk home and require DESK_NAME to be
# resolvable (env / walk-up .env / the desk's _knowledge/profile.*).
#
# `apply-fix` is deliberately NOT a Makefile target (spec §9.5): committing a fix to the
# REAL desk is supervised-only, run by hand as
#   ./deskkit apply-fix --run <run_id>
# (optionally under sandbox-exec — see sandbox/README.md). The throwaway-copy write path
# (propose-fix -> apply-fix) is exercised only inside `verify`.
#
# `dogfood-agent` is a separate, deliberately-manual harness: it drives the REAL in-binary
# eino agent loop (`deskkit agent`) against a throwaway scratch desk, using a real
# ANTHROPIC_API_KEY. It is NOT part of `verify`/`make check`/CI (real LLM calls cost money and
# aren't deterministic) — run it by hand when you want to dogfood the agent loop itself.
.DEFAULT_GOAL := help
SHELL := /bin/bash

BIN     := deskkit
PIDFILE := .deskkit.pid
LOGFILE := serve.log

# Install location. Mirrors install.sh's default (~/.local/bin — no root needed); override with
# `make install PREFIX=/usr/local` to land the binary in $(PREFIX)/bin.
PREFIX  ?= $(HOME)/.local
BINDIR  := $(PREFIX)/bin

# Version stamp: the single source of truth is the repo-root VERSION file. The build target
# passes it to the binary via `-ldflags -X main.version=<VERSION>` so `deskkit
# --version` reports the release (a bare `go build ./...` leaves the default "dev"). The
# release workflow stamps the same way for cross-compiled artifacts.
VERSION  := $(shell cat ../VERSION 2>/dev/null || echo dev)
LDFLAGS  := -X main.version=$(VERSION)

.PHONY: help build install gui serve stop sweep patrol propose-fix findings summary \
        adoption orphans uncollapsed verify dogfood-agent clean test vet fmt

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build the deskkit binary (./deskkit), version-stamped from ../VERSION
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/deskkit

install: build ## Install the version-stamped binary to $(PREFIX)/bin (default ~/.local/bin; override PREFIX=...)
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

verify: ## Run the Phase-1 verify gate against a throwaway scratch desk (never the real desk)
	@bash verify.sh

dogfood-agent: build ## Manual dogfood: drive the REAL agent LLM loop end-to-end (needs ANTHROPIC_API_KEY; not part of verify/CI)
	@bash dogfood-agent.sh

clean: stop ## Stop serve + remove build artifacts (binary, pidfile, serve log)
	@rm -rf $(BIN) $(PIDFILE) $(LOGFILE)
	@rm -rf pb_data  # legacy cwd-relative store from pre-ADR-0002 checkouts, if any lingers
	@echo "cleaned: binary, pidfile, serve log (+ any legacy ./pb_data)"
	@echo "note: the canonical per-desk store lives at \$$XDG_DATA_HOME/deskkit/<DESK_NAME>/"
	@echo "      (fallback ~/.local/share/...); it is persistent and NOT removed by clean — rm it by hand."

test: ## go test ./...
	@go test ./...

vet: ## go vet ./...
	@go vet ./...

fmt: ## Check gofmt formatting (fails and lists files if any need formatting)
	@bash -c 'out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; exit 1; fi'
