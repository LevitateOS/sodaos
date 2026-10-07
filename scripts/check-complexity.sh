#!/bin/bash
# Cyclomatic complexity gate for shipping Go: every non-test function in
# internal/, cmd/, tools/, appliance/, and system/project/ must stay below 10
# (gocyclo flags 10 and above). scripts/, tests/, and *_test.go are out of
# scope. No appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local

selected=()
if [ "$#" -gt 0 ]; then
  for path in "$@"; do
    case "$path" in
      internal/*|cmd/*|tools/*|appliance/*|system/project/*)
        case "$path" in
          *_test.go) ;;
          *) selected+=("$path") ;;
        esac
        ;;
    esac
  done
else
  scratch_dir="$PWD/.artifacts/complexity"
  if ! mkdir -p "$scratch_dir"; then
    printf 'Complexity gate failed: cannot create discovery scratch directory: %s\n' "$scratch_dir" >&2
    exit 1
  fi
  discovery_file=$(mktemp "$scratch_dir/check-complexity-find.XXXXXX") || {
    printf 'Complexity gate failed: cannot create discovery scratch file.\n' >&2
    exit 1
  }
  trap 'rm -f "$discovery_file"' EXIT
  for root in internal cmd tools appliance system/project; do
    if ! find "$root" -name '*.go' ! -name '*_test.go' -print0 >"$discovery_file"; then
      printf 'Complexity gate failed: could not discover Go files under %s.\n' "$root" >&2
      exit 1
    fi
    while IFS= read -r -d '' path; do
      selected+=("$path")
    done <"$discovery_file"
  done
fi
if [ "${#selected[@]}" -eq 0 ]; then
  printf 'Complexity gate passed: no shipping Go files to check.\n'
  exit 0
fi

for path in "${selected[@]}"; do
  if [ ! -f "$path" ]; then
    printf 'Complexity gate failed: selected input is missing: %s\n' "$path" >&2
    exit 1
  fi
done

if analysis=$(go tool gocyclo -over 9 "${selected[@]}" 2>&1); then
  analysis_status=0
else
  analysis_status=$?
fi
if [ "$analysis_status" -eq 0 ]; then
  if [ -n "$analysis" ]; then
    printf 'Complexity gate failed: analyzer returned unexpected output:\n%s\n' "$analysis" >&2
    exit 1
  fi
elif [ "$analysis_status" -eq 1 ] && [ -n "$analysis" ] && \
     ! printf '%s\n' "$analysis" | grep -Ev '^[0-9]+ ' >/dev/null; then
  printf '%s\n' "$analysis"
  count=$(printf '%s\n' "$analysis" | wc -l)
  printf 'Complexity gate failed: %s shipping function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
else
  printf 'Complexity gate failed: gocyclo analysis did not complete successfully (exit %s):\n%s\n' \
    "$analysis_status" "$analysis" >&2
  exit 1
fi
printf 'Complexity gate passed: all shipping Go functions below 10.\n'
