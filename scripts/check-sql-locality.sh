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
    # Exercise checkpoint recovery/clamping with retained elapsed-time facts.
    ./internal/factory/control/dispatch_accounting_test.go) continue ;;
    # Change a retained sponsor receipt without adding a public corruption API.
    ./internal/factory/control/dispatch_restore_helpers_test.go) continue ;;
    # Expire private admission receipts and simulate an unavailable table.
    ./internal/factory/control/dispatch_preparation_test.go) continue ;;
    ./internal/factory/control/dispatch_source_preparation_test.go) continue ;;
    # Isolate bounded traversal from admission headers.
    ./internal/factory/control/traversal_test.go) continue ;;
    # Isolate bounded traversal from admission headers; inspect saved cursors.
    ./internal/factory/control/readiness_sweep_test.go) continue ;;
    # Remove emitted admission headers to isolate priority traversal, fresh
    # members and cursor bounds; same traversal-test reason as above.
    ./internal/factory/control/serving_priority_starvation_test.go) continue ;;
    # Seed and corrupt attributable assignment/run/view history to challenge
    # carried-finding retirement; validating writers gain no corruption hook.
    ./internal/factory/control/review_carried_history_test.go) continue ;;
    # Seed completion/reassessment prefixes and observe budget/cursor rollback
    # with scripted native RPC evidence; not a full native producer.
    ./internal/web/api/factory_completion_budget_test.go) continue ;;
    # Seed and corrupt persisted assignment/publication/completion records for
    # registered-HTTP projection, order and refusal; no exported hook.
    ./internal/web/api/factory_journey_readback_test.go) continue ;;
    # Seed a merged completion event so delivered intent admits a scoped Test
    # Pilot; read allowance data/closed state by root assignment. No public
    # writer records a standalone completion outside the merge flow.
    ./internal/factory/control/delivery_admission_test.go) continue ;;
    # Seed a finished assignment with its merged completion to challenge
    # application verification; count trusted F10 completion posts. No public
    # writer records a standalone finished assignment outside dispatch.
    ./internal/factory/control/app_verification_test.go) continue ;;
    # Seed an assigned assignment, ready preparation, open dispatch and active
    # run to activate a staged worker MCP capability. No public writer records
    # this exact bound set outside the dispatch admission flow under test.
    ./internal/web/api/factory_mcp_live_test.go) continue ;;
    # Seed a persisted review assignment to isolate MCP terminal settlement.
    ./internal/factory/control/mcp_settle_pg_test.go) continue ;;
    # Seed canonical merge evidence and corrupt generation/coverage bindings.
    ./internal/factory/control/intent_plan_choice_test.go) continue ;;
    # Challenge protected repair, immutable spending and atomic report/feed
    # recovery with complete native RPC fixtures and adversarial stored rows.
    ./internal/factory/control/stall_serve_test.go) continue ;;
    # Seed exact active worker/cause/preparation tuples and corrupt attributed
    # read projections through the authenticated MCP transport.
    ./internal/web/api/factory_mcp_bound_test.go) continue ;;
    # Seed finished assignment history to exercise feed projection directly.
    ./internal/web/api/factory_feed_route_test.go) continue ;;
    # Seed an eligible assignment, finished record assignment and open run to
    # challenge FUX14 record-invitation threading through registered owners;
    # these rows have no public Store writer. Disposable fixture only.
    ./internal/web/api/factory_feed_comment_test.go) continue ;;
    # Seed a merged completion, finished assignment and F10 comment rows to
    # challenge comment-product HTTP projection/readback (incl. unavailable
    # lane); proves registered-HTTP readback of fixtures, not a producer
    # chain. No public writer records these rows outside the test closure.
    ./internal/web/api/factory_intent_review_test.go) continue ;;
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
