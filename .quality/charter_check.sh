#!/usr/bin/env bash
# Charter <-> active-gate consistency check (lean charter, 2026-06-16).
# Fails if a config file required by an ENABLED gate in .quality/charter.yml is
# missing, i.e. the active gate set has drifted from the declared closed set.
# Pure shell + grep (no Python) to honor the QZT commit contract.
set -euo pipefail

CHARTER=".quality/charter.yml"
missing=0

require() {
  # require <path> <gate-name>
  if [ ! -e "$1" ]; then
    echo "charter-check: MISSING $1 (required by enabled gate '$2')" >&2
    missing=1
  fi
}

[ -f "$CHARTER" ] || { echo "charter-check: $CHARTER not found" >&2; exit 1; }

# gate 1 + gate 5: pre-commit drives golangci-lint + gitleaks
require ".pre-commit-config.yaml" "lint_format_sec_lint"
require ".golangci.yml"           "lint_format_sec_lint"
# gate 3: tests + coverage
require "scripts/coverage-gate.sh" "tests_coverage"
# gate 4: SAST
require ".quality/opengrep"        "sast"
# gate 5: secrets
require ".gitleaks.toml"           "secrets"
# gate 6: deps
require "osv-scanner.toml"             "deps"
require ".github/dependabot.yml"       "deps"

if [ "$missing" -ne 0 ]; then
  echo "charter-check: FAILED — active gate set drifted from .quality/charter.yml" >&2
  exit 1
fi
echo "charter-check: OK — 6-gate charter configs present."
