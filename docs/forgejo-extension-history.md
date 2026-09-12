# Forgejo extension implementation history

Execution receipts for this workstream only. The
[current status](forgejo-extension-status.md) owns implementation order and remaining
work; these records do not grant appliance/provider execution or reopen completed
steps. The original source audit remains in its
[existing receipt](implementation-history.md#forgejo-extension-source-audit).

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
