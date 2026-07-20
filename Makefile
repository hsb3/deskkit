# desk-standard — the canonical repo task interface. `make help` (the default) lists targets.
#
# Two product lanes sit under one root: the plugin (harness-pure TS core + stdio MCP server,
# built/tested with bun) and the librarian (single Go binary, built/tested via its own
# librarian/Makefile). These root targets fan out to both lanes plus the repo-wide gates
# (neutrality lint, version-sync, actionlint). CI (`.github/workflows/ci.yml`) runs the same
# checks; the release workflow reuses this VERSION and the librarian ldflags stamp.
#
# The single source of truth for the release version is the root VERSION file; `check-version
# -sync` fails if the three shipped manifests drift from it, and the librarian binary is
# stamped from it (see librarian/Makefile).
.DEFAULT_GOAL := help
SHELL := /bin/bash

VERSION := $(shell cat VERSION 2>/dev/null || echo dev)

.PHONY: help setup build install test check verify package media clean version-status release-prep

PREFIX ?= $(HOME)/.local

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

setup: ## Install plugin deps + git hooks (bun install, lefthook install)
	@cd plugin && bun install
	@lefthook install
	@echo "setup: plugin deps installed; lefthook pre-commit hooks installed."

build: ## Build both lanes: plugin (bun/tsc) + librarian binary (go, version-stamped)
	@cd plugin && bun run build
	@$(MAKE) -C librarian build

install: ## Build + install the librarian binary to ~/.local/bin (override PREFIX=/usr/local)
	@$(MAKE) -C librarian install PREFIX="$(PREFIX)"

test: ## Fast tests both lanes: plugin (bun test) + librarian (go test ./...)
	@cd plugin && bun run test
	@$(MAKE) -C librarian test

check: ## Repo gates: neutrality lint + self-test, kit-manifest drift, scaffold frontmatter, plugin core purity, actionlint
	@node scripts/check-neutrality.mjs
	@node scripts/check-neutrality.mjs --self-test
	@node scripts/check-kits.mjs
	@node scripts/check-scaffold-frontmatter.mjs
	@cd plugin && bun run check:purity
	@actionlint

verify: ## Run the librarian Phase-1 verify gate (throwaway scratch desk; never a real store)
	@bash librarian/verify.sh

package: ## Regenerate the marketplace-distribution plugin bundle (claude-plugin/ artifacts)
	@cd plugin && bun run package

media: ## Record the demo media assets (scripts/record-media.sh)
	@bash scripts/record-media.sh

clean: ## Remove build artifacts from both lanes
	@cd plugin && rm -rf dist
	@$(MAKE) -C librarian clean
	@echo "cleaned: plugin/dist + librarian artifacts."

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
	@echo "(the release workflow asserts the tag matches VERSION, re-runs the gates, and publishes binaries + bundle.)"
