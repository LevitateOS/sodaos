# Production gaps

Gaps found while hardening after the 28 October 2026 founder-demo run. Each
is open: nothing below is fixed on `hardening/production-gaps`. The fixes
that did land there are the coder-time allowance hold, the JSON-null array
guards and the actionable MCP refusals.

## Release and versioning

- **Releases are hand-built with an unstamped `0.1.0` extension version.**
  Both extension manifests carry a literal `"version": "0.1.0"`
  (`system/containers/extension/extension.json:5`,
  `system/containers/extension/tailnet-package/extension.json:5`), and the
  package filename and staged manifest take it as given
  (`lib/soda-release-image/src/native_package.rs:380`,
  `lib/release-inputs/src/payload.rs:363`). `scripts/build-soda-extension.ts`
  checks the manifest entries but stamps nothing. Two different builds ship
  under the same version, so an installed host cannot tell which Feed or
  Factory code it runs. The release build should stamp the version from the
  release tag or commit, and refuse a plain `0.1.0`. Every `cmd/*/Cargo.toml`
  is also still at `0.1.0`.

## Coder-time allowance

- **The default 120-minute sponsorship is too small for one issue.** One
  attempt alone may use 120 active minutes
  (`internal/factory/allowance.go:36`), so a 120-minute sponsorship is used
  up by a single attempt before planning, review and verification runs.
  The Settings form suggests 60 (`frontend/spaces/sodaspaces-factory-settings.ts:1575`).
  Tonight 121 of 120 minutes were used before the Builder could finish. The
  default should cover at least one issue's full path (plan, build, review,
  verify, one correction), and Settings should say how far the allowance
  goes in issues, not only minutes.
- **Runs get one per-run limit.** On `simplify/over-engineering` a run's
  deadline is the attempt's `ActiveMinutes`, capped at the host's 179.
  Leftover sponsorship or rolling minutes no longer shorten it. The
  sponsorship minutes are a soft spend cap checked at admission, and the
  attempt clock no longer runs down across CI, review and judge waits. The
  short-allowance hold and `MinUsefulRunMinutes` went with the problem they
  patched. Retry is available once all of a root's runs have settled.
- **The used-up notice is visible only to owners and admins, and only until restart.**
  Dispatch waits live in the coordinator's in-memory observation
  (`internal/factory/control/dispatch.go`, `observeDispatch`) and the
  factory status exposes them only to repository admins
  (`internal/web/api/factory_status.go:110-118`). The Feed and Work notices
  read that status, so a non-admin collaborator sees nothing, and a
  restart clears the notice until the next dispatch pass. A waiting issue
  should leave a durable, visible record (a Feed post or an issue-control
  blocker).
- **The notice names "Settings → SodaOS → Factory control".** The current
  Settings page calls the section "Provider allowance and sponsorship" under
  "Factory authority" (`frontend/spaces/sodaspaces-factory-settings.ts:1037`,
  `:1556`). The wording in `frontend/spaces/sodaspaces-factory-allowance.ts`
  must match the Settings names the `ui/repo-places*` work lands.

## JEV and context selection

- **A failed JEV context-bundle choice is never retried for the same run.**
  `PrepareContextBundleSelection`
  (`internal/factory/control/context_bundle_prepare.go:13-17`) funds the
  choice once per immutable capability: a failed or uncertain consultation
  keeps its reservation, and later attempts reuse the frozen selection
  without another call (`:53-68`). One provider timeout holds the run on
  `context_selection_unavailable` until the owner retries the whole run.
  A bounded retry (for example one more call after a transport failure, not
  after a confident answer) would let the run continue.

## Robustness

- **JSON `null` where SQL expects an array.** Go nil slices are stored as
  JSON `null`, which `COALESCE(x,'[]')` does not catch. The store queries
  known to expand arrays are now guarded with `jsonb_typeof`, but the root
  cause remains: record types with nil slices
  (for example `Corrections`, `ReviewOperations`, `Prerequisites`) still
  write `null`. Normalising them to `[]` before insert, or a lint that
  flags `jsonb_array_elements(` without a `jsonb_typeof` guard in
  `internal/store`, would close the class.
  `internal/store/corruption.go:61-62` is test-only and left as is.
- **Capability lookup failures are all "not found".** `admitCall`
  (`internal/web/api/factory_mcp.go:260-262`) reports a store outage the same
  as an unknown `capability_id`, so a worker told to fix its id may instead
  be facing an outage. It should tell the two apart.

## Verification

- **Dispatch and store paths have no coverage without `SODA_PG_*`.** The
  control package's dispatch, admission and publication tests need a
  disposable Postgres fixture and fail without it
  (`internal/factory/control/mcp_capability_producer_test.go:68`). None of
  the allowance hold or JSON-null guards ran under tests on this branch.
  They should run on a lane that has the fixture before release.
- **The founder-demo journey does not typecheck.**
  `tests/installed/founder-demo-e2e.ts:456-464` (merged from
  `fix/issue-plan-bindings`) fails `tsconfig.tests.json` on possibly
  undefined `href` values, so `bun run typecheck` fails on this branch.
- **Source checks are not clean on `main`.** `oxfmt` lists about 40
  unformatted TypeScript files, `oxlint` reports errors in
  `tests/frontend/factory-work-controls.test.ts` and
  `tests/frontend/factory-feed-browser.test.ts`, and `gofmt` flags
  `internal/factory/control/acceptance_initial.go`. The gates
  (`scripts/check-oxfmt.sh`, `scripts/check-oxlint.sh`) cannot block
  regressions while they already fail.
