# pocket-librarian — task interface. `make help` lists targets.
#
# Single-writer rule (spec §2.7): the one-shot subcommands (sweep, patrol, propose-fix,
# findings/summary/adoption/orphans/uncollapsed, and the supervised `apply-fix` CLI) open
# the same on-disk SQLite store directly via native DAO. They must NOT run while `serve` is
# up against the same pb_data — run `make stop` first. `verify` is self-contained (it
# manages its own throwaway pb_data + scratch desk) and is safe to run at any time.
#
# `apply-fix` is deliberately NOT a Makefile target (spec §9.5): committing a fix to the
# REAL desk is supervised-only, run by hand as
#   ./pocket-librarian apply-fix --run <run_id>
# (optionally under sandbox-exec — see sandbox/README.md). The throwaway-copy write path
# (propose-fix -> apply-fix) is exercised only inside `verify`.
.DEFAULT_GOAL := help
SHELL := /bin/bash

BIN     := pocket-librarian
PIDFILE := .pocket-librarian.pid
LOGFILE := serve.log

.PHONY: help build gui serve stop sweep patrol propose-fix findings summary \
        adoption orphans uncollapsed verify clean test vet fmt

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build the pocket-librarian binary (./pocket-librarian)
	@go build -o $(BIN) ./cmd/pocket-librarian

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

orphans: build ## Show .md files with no entity_type / not under a meta prefix (query orphans)
	@./$(BIN) query orphans

uncollapsed: build ## Show open R5 judgment findings — graduated-but-not-collapsed docs (query uncollapsed)
	@./$(BIN) query uncollapsed

verify: ## Run the Phase-1 verify gate against a throwaway scratch desk (never the real desk)
	@bash verify.sh

clean: stop ## Stop serve + remove the built binary and local pb_data
	@rm -rf $(BIN) pb_data $(PIDFILE) $(LOGFILE)
	@echo "cleaned: binary, pb_data, pidfile, serve log"

test: ## go test ./...
	@go test ./...

vet: ## go vet ./...
	@go vet ./...

fmt: ## Check gofmt formatting (fails and lists files if any need formatting)
	@bash -c 'out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; exit 1; fi'
