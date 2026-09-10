# Overengineering review — three passes at 7594458

Recorded on 10 September 2026 at the user's request. All three passes examined
**`7594458025add930ea2b0ffe58cefff5f8f12196`**, after the managed-key writer change.
The production code did not change between reviews; the conclusions changed as
inspection widened and the assistant corrected its reasoning.

These were **three passes by the same assistant**, not three independent reviewers
or an independent peer-review consensus. This records the findings, reversals and
remaining qualifications rather than rewriting the first answer as if it had been
correct all along.

This is a revision-bound supplement to the [ownership audit](upstream-ownership-audit.md),
not another backlog. The [refactoring plan](refactoring-plan.md) owns implementation
decisions and ordering; the [handoff](implementation-status.md) owns actual checks
and delivery. No deletion described here was implemented during these reviews.
The request to record them does not select every recommendation for implementation.

## How the assistant changed its mind

### Review 1 — deletion-focused inspection

**User direction:** investigate suspected overengineering, starting with recent
changes, without deleting unfinished features or producing another sprawling plan.

Before the broader review, the assistant had cited Phase 3's passing correctness
tests when defending the change. Inspection then showed that its new key-error
distinctions were discarded by the real caller. Passing tests established the
checked behavior, not the necessity of the added error layer.

The first review proposed six candidates: key-error classes, the root-to-root runner
subprocess, terminal standalone mode, unused process-wrapper surface, historical
support-tool taxonomy and presentation inventory bookkeeping. It distinguished
these from necessary session, locking, durability and terminal-lifetime protections,
but presented candidates of very different certainty too uniformly.

**First change of mind:** from defending added machinery with test results to looking
for specific deletions. **Remaining mistake:** the review too readily promoted
"this could be simplified" into "this is demonstrated overengineering," especially
for compatibility surfaces and regression snapshots. No project-wide maintenance
cost or performance improvement had been measured.

### Review 2 — adversarial self-review

**User direction:** "Do a peer review of yourself."

The second pass challenged the proposed deletions against their contracts:

- The runner helper enforces real/effective UID and `PKEXEC_UID` checks that the
  coordinator's authorizer does not completely duplicate. Removing the subprocess
  must preserve the combined admission policy.
- Terminal standalone mounting is documented, not merely an arbitrary test option.
  Production workspace tests already exercise the locator-based mode; the older
  fixture does not establish that production behavior is untested.
- Expected caller lists and hashes can catch lost native gates and fields. Their
  being derivable or sensitive to formatting does not make them dispensable.
- Support labels are explicitly retained protocol/evidence identifiers, and missing
  observations are deliberately reported without certifying readiness.

**Second change of mind:** the verdict became **two clear small deletions, two
conditional refactors and two weak or low-priority recommendations**, rather than
six similarly actionable findings. Broad presentation-check removal was withdrawn;
security and compatibility qualifications became explicit.

**Remaining mistake:** this correction leaned too far toward treating the existence
of tests, documentation or other dependants as a reason to retain the implementation,
without fully asking whether those dependants themselves had an independent purpose.

### Review 3 — investigation through the dependants

**User direction:** re-review the dependants and whether they also represent
unnecessary machinery.

The third pass separated actual product consumers from scaffolding created by an
implementation. It traced production entrypoints, interfaces, fixtures, packaging
requirements and documentation rather than stopping at a reference search.

This exposed an unused Tailnet classification chain, confirmed the runner hop's
supporting duplication, and found historical template reconstruction being applied
to current-behavior tests. It also found preservation requirements deeper in the
chain: the live workspace consumes `pending:` locator events, and the native runner
lock still coordinates independent CLI and daemon callers.

**Third change of mind:** tests and packaging lists do not independently justify the
thing they test or package. Remove an unnecessary mechanism with its incidental
dependants, port useful behavioral tests, and retain independently required authority,
persistence and compatibility behavior. This strengthened particular findings; it
did not restore the original blanket confidence or justify a broad rewrite.

## Disposition across the three reviews

| Original candidate | Review 1 | Review 2 | Review 3 |
| --- | --- | --- | --- |
| Key-error distinctions | Remove classes and unused diagnostics | Confirmed, but small | Dedicated diagnostic expectations/docs depend on the classes; concurrency protections do not |
| Runner subprocess hop | Strong larger removal candidate | Conditional: helper-only admission checks matter | Extra hop generates interfaces, executable, duplicate tests and packaging requirements; retain real CLI adaptation and combined root policy |
| Terminal standalone mode | No production caller found; port fixture | Documented interface; not established disposable code | Port tests rather than preserving the mode for them; interface retirement remains explicit and live locator/legacy import behavior stays |
| Generic process surface | Remove unused methods/configuration | Output, cancellation and errors are used | Trace/stdin tests support unused configuration; a further Tailnet identity/classification chain has only test callers |
| Support-tool taxonomy | Remove historical label/action coupling and checklist | Small retained protocol; benefit not established | Still low priority; no independent scheduling/readiness engine was found that warrants a subsystem rewrite |
| Presentation inventory | Shrink bookkeeping and eventually retire broad hashes | Broad deletion withdrawn: equivalent coverage not demonstrated | Preserve parity/permission checks; the stronger finding is historical normalization leaking into behavior tests |

These are the review's changing assessments, not completion flags or an execution order.

## Findings after the third pass

### Tailnet: unused classification and process configuration

The actual command chains are:

```text
cmd/soda-tailnet         -> Client.Status()
cmd/soda-forgejo-tailnet -> Client.Endpoint()
```

In [internal/tailnet/tailnet.go](../internal/tailnet/tailnet.go), `Client.Identity()`
(lines 98–112 at the reviewed revision) instead depends on `Status.EnrollmentState()`,
the `EnrollmentState` type and three constants, and `ErrIdentityUnavailable`.
That chain has only [test callers](../internal/tailnet/tailnet_test.go). The console
uses [its real status/guidance adapter](../cmd/soda-tailnet/command.go), not that enum.

The same client constructs `process.OSRunner{}`. Its production use needs `Output`,
not `Run`, supplied stdin or configurable trace streams. The
trace-failure tests (`internal/process/output_test.go` at the reviewed revision) and
stdin-wiring test (`internal/process/process_test.go`) exercise those unused options;
their existence does not supply a product requirement for them.

**Removal boundary:** the unused identity/classification chain and process surface
can be removed together with implementation-only expectations. Useful parsing tests
can target `Status()`/`Endpoint()`. Preserve the `Status.Identity` field, canonical
address validation, console guidance, Forgejo Git advertisement, cancellation and
command-error handling. Do not remove Tailnet, its Cockpit page or its real callers.

### Runners: redundant transport, not redundant native ownership

```text
Cockpit -> soda-runners -> Coordinator
  -> PrivilegedRunners / PKExecInvoker
  -> soda-runner-helper -> Helper -> Operations -> Native
```

[Cockpit's caller](../cockpit/src/runners/native.ts) depends on the public
`soda-runners` protocol, not on the internal subprocess hop. The hop supports the
invoker (`internal/runners/invoker.go` at the reviewed revision), parallel
`PrivilegedRunners` interface, helper executable (`cmd/soda-runner-helper/main.go`),
helper wrapper and invoker-specific tests (`internal/runners/invoker_test.go`). The two
[CLI](../cmd/soda-runners/command_test.go) and
helper (`cmd/soda-runner-helper/command_test.go`) command-test files are byte-identical,
103 lines each; their command wrappers differ only in executable name. Packaging
and installed presence checks name the helper too, but are not independent users
of its behavior.

The [coordinator](../internal/runners/coordinator.go) still performs real work:
configured Forgejo URL substitution, strict input handling and public list-response
shaping. The simpler candidate connects that adapter to native lifecycle operations,
not to another newly encoded JSON round trip.

**Preservation boundary:** the [native root gate](../internal/linuxhost/operator.go)
checks real/effective UID and rejects a non-root `PKEXEC_UID`; retain that policy
alongside the [authorizer's native identity check](../internal/runners/auth.go).
`Operations` is not an authorization replacement. Keep secret-input handling,
provider/native lifecycle checks and cancellation. The cross-process lock remains
necessary because the CLI and [host daemon](../cmd/soda-host/main.go) are still
independent callers. Removing a source executable does not authorize deleting or
replacing retained installed files; packaging and delivery remain coordinated work.

### Template tests: historical reconstruction escaped its parity-only purpose

[readForgejoTemplate](../scripts/forgejo_components_test.go) (lines 662–671) reads
current source, calls [withoutForgejoFormLayout](../scripts/forgejo_form_layout_test.go),
and strips presentation roles before returning it. The
[delta file](../tests/forgejo/presentation/form-presentation-deltas.json) contains
82 ordered replacements across 41 template paths.

This is not confined to exact-upstream hash comparisons. For affected templates,
`TestForgejoRepositoryCreationKeepsNativePermissionBranches` and
[TestForgejoSharedProjectsFormExecutesNewAndEditBranches](../scripts/forgejo_shared_projects_test.go)
also consume reconstructed older markup. The normalization is documented, but its
placement in the general reader makes current-behavior tests depend on historical
presentation reconstruction.

**Simplification boundary:** behavior tests should read current authored templates;
historical normalization belongs explicitly in upstream-parity comparisons. Port
obsolete presentation assertions rather than maintaining them through reverse patches.
This does not establish that every replacement or parity test can be deleted now.

The earlier inventory-size argument remains withdrawn: at this revision the
[inventory](../tests/forgejo/presentation/inventory.json) has 253 entries and 9,725
lines, but size alone demonstrates neither redundancy nor maintenance cost. Expected
caller snapshots and hashes can detect lost native conditions. The
[control/gate extractor](../tests/forgejo/presentation/settings-contracts.ts) is not
proven equivalent to full source parity or real branch behavior; do not substitute
it wholesale and declare security coverage preserved.

### Terminal: port useful tests without retaining an alternate mode for them

The [workspace](../frontend/spaces/sodaspaces-workspace.ts) always supplies an explicit
locator to `mountTerminal`. The older [terminal fixture](../tests/frontend/fixtures/terminal-fixture.ts)
omits it and depends on implicit session storage, toolbar selectors and immediate
End behavior. Its useful tests concern authorization, original identities, transport
bounds, cancellation and uncertain outcomes; those requirements do not require a
second mounting/UI mode. Production-mode coverage already exists through the
[workspace fixture](../tests/frontend/fixtures/workspace-fixture.ts) and its tests.

**Conditional removal boundary:** retire the
[documented no-locator interface](terminal-integration.md#terminal-only-mounting-surface)
with its fixture/UI/storage branches, not the whole terminal or transport tests.
Keep `new`/`existing`/`pending` locators, exact IDs, `restore`, finite retention,
explicit End and legacy storage import. In particular, `pending:` is also the
**live locator-event format** consumed by the workspace around line 993, not only
old storage syntax. No external-consumer compatibility proof was obtained.

### Managed keys: unused diagnostics, necessary filesystem behavior

[project_keys.py](../internal/host/project_keys.py) defines `KeyWriterBusy` and
`KeyRevisionChanged` and emits distinct fixed diagnostics. Its real
[Go caller](../internal/host/management.go) (lines 184–186) discards the execution
error and reports `native key operation not confirmed`.

The [diagnostic test](../tests/build/test_project_keys.py) and explanatory guide/plan
text depend on those distinctions; no product caller consumes them. Simplification
means ordinary exceptions and one sanitized failure, with the corresponding assertion
changes—not a new public error protocol or deletion of the filesystem suite.

Keep cooperative writer locking, revision checks, flush/fsync ordering, exclusive
temporary-file ownership, no restoration after uncertainty and tests of those real
requirements. This small finding does not invalidate the whole Phase 3 fix or turn
its 613-line commit diff into a demonstrated deletion budget. Historical execution
evidence remains intact; repetitive current contract prose can be consolidated.

### Support labels: bounded cleanup, not a new evidence-system project

[ValidOwner and Handoff](../internal/acceptance/report.go) and the
[support command](../tools/soda-acceptance/main.go) bind historical P/U labels to
particular actions and report rows. Dependants are observation/report validation,
focused tests and documented CLI usage, not an independent scheduler or product
readiness engine.

The [native-support contract](native-support.md) deliberately retains those labels
as protocol/evidence identifiers. Label/action decoupling is at most a small cleanup
candidate: preserve old records and arguments, actual action/target validation,
missing/failed-evidence honesty, hashes, redaction and cleanup accounting. Neither
retiring the roadmap labels nor their continued appearance justifies rewriting the
evidence subsystem. The third pass did not establish higher urgency here.

## Lessons and evidence limits

- A test **about** an abstraction is not necessarily a product consumer **requiring**
  that abstraction. Preserve the behavioral assertion; do not automatically preserve
  its old fixture or helper structure.
- Conversely, tests across separate trust boundaries are not redundant merely because
  they repeat a check. Identify the complete admission policy before removing a layer.
- Documentation can record a real compatibility promise or merely describe existing
  machinery. Investigate which it is; neither ignore the promise nor treat every
  descriptive paragraph as a permanent architectural requirement.
- Judge the complete removal boundary, including interfaces, fixtures and packaging.
  Do not replace a removable chain with a new registry, framework or compatibility engine.
- Large files, unusual failure cases and passing tests are insufficient evidence for
  either "overengineered" or "justified." The first review overstated certainty; the
  second overcorrected; the third improved the evidence without proving every deletion safe.
- None of this selects removal of unfinished installer/media, profiles/KDE, runners,
  marketplace/AI, credentials, onboarding or native/CLI work. The
  [architecture](architecture.md) and [scope boundaries](deferred.md) remain authoritative.

The reviews used source/contract inspection, Git/reference searches and read-only
standard-library file comparisons/counts. They ran no product builds, compilation,
type checks, tests, native/provider actions or deployment, and changed no source or
retained state. Counts above are static observations, not runtime coverage or measured
productivity savings. Earlier Phase 3 test results belong to their existing handoff;
they are not fresh evidence for these proposed removals.
