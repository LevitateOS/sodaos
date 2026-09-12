# Forgejo extension implementation history

Execution receipts for this workstream only. The
[current status](forgejo-extension-status.md) owns implementation order and remaining
work; these records do not grant appliance/provider execution or reopen completed
steps. The original source audit remains in its
[existing receipt](implementation-history.md#forgejo-extension-source-audit).

## Environment read session publication

**Baseline `b1ed488`; source and synthetic-peer Go checks only.** The user approved
investigating the detail/member/connection read boundaries as work independent of
the active Forgejo redesign. Main was clean; the redesign worktree had no overlapping
changes in the backend or the documentation edited here. Its uncommitted visual
work was preserved, and localization was deferred for redesign coordination.

The targeted reproduction confirmed that initially authenticated reads could publish
data after completed Soda logout or replacement of the original session during I/O.
Existing inventory/profile/OS reads already rechecked the original session before
publication. The three direct-ID handlers lacked that final check. This was a late
response from an admitted request, not anonymous access, a role bypass or evidence
of atomic native/Soda logout.

`internal/web/environments_api.go` now calls the existing `requireCurrentSession`
immediately before successful detail/member/connection responses, returning the
existing inventory-style `401 unauthenticated` JSON without data on failure. Early
authorization, degraded member reads, explicit operator inspection and own-membership
connection access remain intact. Earlier error responses keep their existing status.
No shared auth helper, schema, host protocol, mutation handler, browser asset or
redesign/Tailnet file changed. The existing project browser caller already retires
its session/view on 401/403; it was inspected, not modified or freshly browser-tested.

### Checks actually run

Evidence: `.artifacts/environment-read-publication-HCb4VL/`; `checks.json` records
commands/environment/test names. `environment_read_publication_test.go` reuses the
existing management fixture, transport doubles, store APIs and routed logout.

| Check | Result and scope |
| --- | --- |
| New regressions before the fix | 46 refusal cases returned 200/data. Eight unchanged-session controls passed, as did two membership-store/cancellation errors that already refused before publication. `regression-before.jsonl`. |
| Focused Go checks after the fix | All 56 new cases passed within 14 top-level tests / 102 subcases. Includes existing ownership/degraded/member/operator, incomplete reservation, connection, current-session, neighboring read-publication and API error/logout checks. `focused-go.jsonl`. |
| Same selection with `-race` | All passed; no skips or race reports. `focused-race.jsonl`. |
| Formatting/documentation | Pinned `gofmt`, local documentation links/anchors and whitespace checked. |

The eight read paths cover late provider/organization-owner and helper results,
operator inspection, degraded member observations and unavailable native inspection.
Each has unchanged/logout/user/context/CSRF/store/cancel cases. Replacements preserve
unrelated token/actor/context/CSRF fields to test the original binding; cancellation
is deliberately ignored by the transport double. Assertions preserve the project,
members and the other actor's session, and forbid mutation RPCs or secret/error leakage.

Go **1.26.7 darwin/arm64**, `GOTOOLCHAIN=local`, `GOWORK=off`, `-mod=readonly`,
`-count=1`; `CGO_ENABLED=0` ordinarily and `1` for race instrumentation.

### Outcome and limits

**Source-fixed, not installed.** Logout/session change/cancellation completed before
the final check withholds the data. No lock spans external I/O or response writing:
logout after that check is not serialized with publication, and already admitted
helper reads are not undone or guaranteed cancelled. Provider/native session parity
and previously issued Linux access are unchanged. No real provider, retained fixture,
credential, service/VM lifecycle, native build, deployment or frontend operation was
performed. The shared target/Tailnet handoff and redesign worktree were untouched.

## Native navigation current-page cues

**Baseline `9ebc3a8`; local source/browser work only.** The user requested further
fixes and clarified that deployment belongs on the x86 machine, not this arm64
macOS source session. The working tree was clean; no overlapping Tailnet edits were
present when the affected files were inspected.

The selected Forgejo 15.0.7 navbar already uses `active` and `aria-current="page"`;
Soda's existing component CSS styles that native class. `soda-settings-link.ts` now
uses those cues for Spaces/Runners from the matching actor in the existing validated
`main.soda-native-page #soda-native-content` host. It does not parse the query again
or infer authority from a selected view. Spaces has one stable template target;
Runners remains absent unless its existing Soda-session/operator check succeeds.
Generation-based retirement and ordinary links/drafts are preserved.

The header/footer/dashboard entry URLs and compiler-owned transitive imports now
share epoch `2026-09-12.native-pages-8`. No new runtime, dependency, CSS rule or payload
entry was needed. Dashboard/footer selector logic, OAuth, backend/helper code and
Tailnet feature implementation were not changed.

### Checks actually run

Evidence: `.artifacts/forgejo-navigation-YFesQX/`.

| Check | Result and scope |
| --- | --- |
| New checks before the fix | Both emitted-browser tests failed on missing active state (Runners and Spaces); the Go navigation test failed on the missing stable Spaces target. Original logs retained as `browser-before.log` and `templates-before.log`. |
| `bun run build:forgejo` | Passed before and after; `build-before.log`, `build-after.log`. Emitted browser assets only. |
| Selected Go template checks | 7 top-level tests passed: native host/invalid locators/ordinary dashboard, operator navigation, drawer/context escaping and request logging. `templates-after.log`. |
| Selected frontend/Forgejo checks | 10 tests passed across `settings-link`, `connection` and `lit-build`. The final navigation tests cover 25 fixture scenarios: operator/nonoperator/mismatch/expiry, both valid global views, repository settings, absent/invalid/unknown hosts, unrelated URLs, markers outside the native host, foreign actors, root/prefixed asset paths and expired Soda access on Spaces. `browser-final.log`; the earlier passing run is retained as `browser-after.log`. |
| `bun run typecheck` | Passed all configured TypeScript checks, actual-source Lit analysis and checker fixtures. `typecheck.log`; test-config typing was rechecked after the final test-only host-boundary addition (`test-types-final.log`). |
| Formatting/documentation | Pinned `gofmt`, local documentation links/anchors and whitespace checked. |

Bun **1.4.2**. Go **1.26.7 darwin/arm64**, `GOTOOLCHAIN=local`, `GOWORK=off`,
`CGO_ENABLED=0`, read-only modules and `-count=1`. Browser tests used
`SODA_LIT_BROWSER=1` with the existing emitted-asset Playwright fixture pattern;
no native fixture/provider opt-ins were enabled. `checks.json` records exact commands.

### Outcome and limits

**Source-complete navigation slice.** Browser authority data and host markup were
synthetic; Go tests separately verify which native-template branches emit the host.
Pagehide/pageshow checks simulate events rather than prove actual BFCache. Prefix
checks establish navigation/module paths, not end-to-end subpath support. Full
native visual/accessibility acceptance and installed cache transitions were not run.
No native fixture login, real credentials, retained-state/lifecycle operation,
Tailnet implementation, native appliance build or deployment occurred. Localization
and conditional customization reduction remain separate queued work.

## Runner post-decode mutation admission

**Baseline `61a84b6`; source correction and local tests only.** The user requested
the runner logout-race fix. The working tree was clean before this slice.

`internal/web/runners.go` now uses the existing `requireCurrentSession` after body
decoding/validation, immediately before `RunnerCreate` and `RunnerAction`. The
original configured-operator/provider/actor/Origin/CSRF gates remain before decoding.
Final admission failure returns the existing `401 reauthentication_required`
response with no helper request. No shared authentication helper, host protocol,
schema, browser asset or Tailnet implementation was changed.

`internal/web/runner_admission_test.go` reuses `runnerAPIRequests`,
`runnerWebFixture` and `mutationAdmissionBody`. Across create/start/stop/restart/remove,
it covers unchanged success, completed real routed logout, changed user/context/CSRF,
closed store and request cancellation. Context replacement preserves the token,
actor, CSRF and grant; user/CSRF replacements preserve the original context so each
comparison is exercised independently. Transport attempts are counted before
transport cancellation, not merely at a synthetic helper's HTTP listener.

### Checks actually run

Evidence: `.artifacts/runner-admission-fix-ut6yVW/`; `checks.json` records exact
commands, environment and test names. All provider/helper peers were synthetic.

| Check | Result |
| --- | --- |
| New regressions before the fix | 5 unchanged controls passed; 30 refusal cases failed. Logout/user/context/CSRF/store cases returned 200; cancelled requests returned 502. Every refusal case attempted one helper dispatch. `regression-before.log`. |
| Focused Go checks after the fix | 18 top-level tests and 360 subcases passed, including all 35 new admission cases and existing early-authorization/request-boundary/single-dispatch checks. `focused-go.log`. |
| Same selection with `-race` | All passed with no race report and no skips. `focused-race.log`. |
| Formatting/documentation | Pinned `gofmt` reported no changes needed; local documentation links/anchors and whitespace checked. |

Go **1.26.7 darwin/arm64**, `GOTOOLCHAIN=local`, `GOWORK=off`, `-mod=readonly`,
`-count=1`; ordinary checks used `CGO_ENABLED=0`, race checks used `CGO_ENABLED=1`.
The cached pinned toolchain was used, not the shell's newer default Go.

### Outcome and limits

**Source-fixed, not installed.** Session retirement or request cancellation completed
before the final admission check prevents dispatch. An operation already admitted
is not rolled back or guaranteed cancelled by later logout. No native build,
full frontend/source suite, appliance/provider operation, retained fixture lifecycle,
credential use or cleanup was performed. Tailnet work and shared retained-target
state were untouched.
