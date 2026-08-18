# desk-standard — the canonical repo task interface. `make help` (the default) lists targets.
#
# One product lane sits under this root: the librarian (a single Go binary, built/tested via its
# own librarian/Makefile). These root targets fan out to that lane plus the repo-wide gates
# (neutrality lint, version-sync, actionlint). CI (`.github/workflows/ci.yml`) runs the same
# checks; the release workflow reuses this VERSION and the librarian ldflags stamp.
#
# The single source of truth for the release version is the root VERSION file; `check-version
# -sync` fails if the shipped manifests drift from it, and the librarian binary is stamped from
# it (see librarian/Makefile).
.DEFAULT_GOAL := help
SHELL := /bin/bash

VERSION := $(shell cat VERSION 2>/dev/null || echo dev)

.PHONY: help setup build install test check verify e2e package media clean version-status release-prep

PREFIX ?= $(HOME)/.local

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

setup: ## Install the git hooks (lefthook install)
	@lefthook install
	@echo "setup: lefthook pre-commit hooks installed."

build: ## Build the librarian binary (go, version-stamped)
	@$(MAKE) -C librarian build

install: ## Build + install the librarian binary to ~/.local/bin (override PREFIX=/usr/local)
	@$(MAKE) -C librarian install PREFIX="$(PREFIX)"

test: ## Fast tests: librarian (go test ./...)
	@$(MAKE) -C librarian test

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
	@shellcheck install.sh librarian/verify.sh librarian/dogfood-agent.sh librarian/dogfood-pm.sh librarian/sandbox/*.sh scripts/record-media.sh librarian/e2e/e2e.sh librarian/e2e/lib.sh librarian/e2e/steps/*.sh
	@actionlint
	@node scripts/check-workflow-pins.mjs
	@node scripts/check-workflow-pins.mjs --self-test
	@node scripts/check-profile-root.mjs
	@node scripts/check-profile-root.mjs --self-test

verify: ## Run the librarian Phase-1 verify gate (throwaway scratch desk; never a real store)
	@bash librarian/verify.sh

e2e: ## Run the end-to-end system-behaviour suite (whole system, throwaway desk; offline, no LLM key)
	@bash librarian/e2e/e2e.sh

package: ## No-op: the marketplace bundle under plugins/ is authored in place, nothing is generated
	@echo "package: nothing to generate — the marketplace bundle under plugins/ is authored in place."

media: ## Record the demo media assets (scripts/record-media.sh)
	@bash scripts/record-media.sh

clean: ## Remove the librarian build artifacts
	@$(MAKE) -C librarian clean
	@echo "cleaned: librarian artifacts."

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
