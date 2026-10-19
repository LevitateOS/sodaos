#!/bin/bash
# SQL locality: product PostgreSQL stays in internal/store.
# See docs/development/go.md. No appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."

violations=""
hits_file="$(mktemp "${TMPDIR:-/tmp}/soda-sql-locality.XXXXXX")"
trap 'rm -f "$hits_file"' EXIT
while IFS= read -r -d '' file; do
  case "$file" in
    ./internal/store/*) continue ;;
    # The workload-access acceptance probe issues PostgreSQL DML to ephemeral
    # test workloads through the psql CLI (ported from
    # tests/installed/workload-access.py). It never touches product Store or
    # database/sql; these exact exclusions keep the product gate intact.
    ./internal/acceptance/workload_access.go) continue ;;
    # This test-only seed writes dependency edges directly into an external
    # staged Forgejo SQLite fixture because it has no REST writer. The
    # production Store remains PostgreSQL-only (COST-GO-SQLITE-FIXTURE-1).
    ./internal/factory/control/staged_seed_test.go) continue ;;
    # Existing adversarial integration fixtures, individually documented in
    # docs/development/go.md. Never exempt a production file or all tests.
    # Force consumed clocks without waiting for real attempt deadlines.
    ./internal/factory/control/dispatch_expiry_test.go) continue ;;
    # Isolate bounded traversal from admission headers; inspect saved cursors.
    ./internal/factory/control/readiness_sweep_test.go) continue ;;
    # Seed and corrupt attributable assignment/run/view history to challenge
    # carried-finding retirement; validating writers gain no corruption hook.
    ./internal/factory/control/review_carried_history_test.go) continue ;;
    # Seed completion/reassessment prefixes and observe budget/cursor rollback
    # with scripted native RPC evidence; not a full native producer.
    ./internal/web/api/factory_completion_budget_test.go) continue ;;
    # Seed and corrupt persisted assignment/publication/completion records for
    # registered-HTTP projection, order and refusal; no exported hook.
    ./internal/web/api/factory_journey_readback_test.go) continue ;;
    # Seed a persisted review assignment to isolate MCP terminal settlement.
    ./internal/factory/control/mcp_settle_pg_test.go) continue ;;
    # Seed finished assignment history to exercise feed projection directly.
    ./internal/web/api/factory_feed_route_test.go) continue ;;
  esac
  if grep -nE '"database/sql"|sql\.Open[[:space:]]*\(|`(SELECT|INSERT|UPDATE|DELETE|CREATE TABLE|ALTER TABLE|PRAGMA)[[:space:]]|"[[:space:]]*(SELECT|INSERT|UPDATE|DELETE|CREATE TABLE|ALTER TABLE|PRAGMA)[[:space:]]' "$file" >"$hits_file" 2>/dev/null; then
    while IFS= read -r hit; do
      violations+="${file}:${hit}"$'\n'
    done <"$hits_file"
  fi
done < <(find ./internal ./cmd ./tools -name '*.go' -print0)

if [ -n "$violations" ]; then
  printf '%s' "$violations"
  printf 'SQL locality failed: database/sql, sql.Open, and SQL verb literals require internal/store or a named test fixture exception (docs/development/go.md).\n'
  exit 1
fi
printf 'SQL locality passed: product SQL stays in internal/store; named fixture exceptions only.\n'
