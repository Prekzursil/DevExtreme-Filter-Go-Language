#!/usr/bin/env bash
# Lean gate-3: Go tests + STRICT 100% coverage.
#
# Mirrors the PROVEN per-package coverage recipe already used by
# .github/workflows/sonar.yml (per-package profiles concatenated; -coverpkg is
# deliberately avoided because it leaves other packages' statements at 0 hits in
# the merged profile and produces false sub-100% numbers).
#
# Coverage denominator = hand-written packages. Excluded, matching
# sonar-project.properties' coverage.exclusions, because they are generated or
# build-shim, not hand-authored source:
#   - transaction-filter-backend/ent/...  (ent codegen, // generated)
#   - main_prod.go                        (prod-only entrypoint, built -tags prod)
#   - main_test_stub.go                   (empty linker shim, //go:build !prod)
set -euo pipefail

PROFILE="coverage.out"
PART="$(mktemp)"
trap 'rm -f "$PART"' EXIT

mapfile -t PKGS < <(go list ./... | grep -vE '/ent($|/)')

rm -f "$PROFILE"
first=1
for pkg in "${PKGS[@]}"; do
  go test "$pkg" -coverprofile="$PART" -covermode=count
  if [ -f "$PART" ]; then
    if [ "$first" -eq 1 ]; then
      cp "$PART" "$PROFILE"; first=0
    else
      tail -n +2 "$PART" >> "$PROFILE"
    fi
  fi
done

go vet ./...

# Drop build-shim lines that Sonar also excludes from the coverage denominator.
grep -vE '(^|/)(main_prod\.go|main_test_stub\.go):' "$PROFILE" > "${PROFILE}.f" && mv "${PROFILE}.f" "$PROFILE"

TOTAL="$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub("%","",$3); print $3}')"
echo "Measured coverage (hand-written packages): ${TOTAL}%"

if [ "$(awk -v t="$TOTAL" 'BEGIN{print (t+0 >= 100.0) ? 1 : 0}')" -ne 1 ]; then
  echo "FAILED: coverage ${TOTAL}% < 100% (lean charter gate-3)." >&2
  go tool cover -func="$PROFILE" | awk '$3 != "100.0%" && $1 !~ /^total:/'
  exit 1
fi

echo "SUCCESS: coverage gate (100%)."
