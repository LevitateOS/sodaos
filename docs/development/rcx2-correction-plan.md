# RCX-2 review correction plan

Prepared 2026-10-16 for the eight findings in the independent RCX-2 review.
This is a repair specification, not evidence that the repairs have been made.
The current instruction is planning only: no implementation, runtime mutation,
new Muse launch, commit, push, publication or merge is performed by this plan.

## Scope and controlling contracts

Correct the existing implementation in the canonical `~/Projects/sodaos` and
`~/Projects/forgejo-ext` checkouts. Keep the existing ownership: Fountain owns
generic packages, durable intent and lifecycle admission; Soda owns restricted
host effects; the release tools own acquisition and maintenance. No language
ports, new durable state owner, package format, cleanup daemon, compatibility
layer or replacement test framework is required.

The adopted [RCX-2 revision](../../.artifacts/release-design-refinement/RCX-2.md)
and its [Fountain](../../.artifacts/release-design-refinement/fountain-design.md),
[Tailnet](../../.artifacts/release-design-refinement/tailnet-design.md) and
[release](../../.artifacts/release-design-refinement/release-design.md) contracts
establish the intended behavior. The revision's original “not adopted” banner
describes its preparation history; the owner's subsequent adoption controls.
Current owning guides are [release architecture](../architecture/release.md),
[networking](../architecture/networking.md), [Project OS](../reference/project-os.md)
and Fountain's `docs/fountain-release.md` and `contrib/extensions/README.md`.
Update those owners where a repair needs a clarification; do not copy the plan
into product documentation.

Planning baseline: Soda `04699345` and Fountain `d9120378b2`. Concurrent Factory,
native-operation and issue/snapshot work is outside this scope and must be
preserved. Fountain's actively edited `models/extensionauth/verify.go` is a shared
authorization input: inspect its current change and obtain a handoff if a repair
needs that file; do not stage it with extension fixes by assumption. These
commits identify planning inputs, not instructions to reset either checkout.
Review probes are retained in Fountain's
`.artifacts/rcx2-review-20261016/fountain/`. They demonstrate failures using
isolated state and fake transports; promote the useful cases into normal tests,
then exercise the real callers. The optional-static-validation and extra-gzip
probes were not included in the eight accepted findings and are not extra tasks.

## Ownership, sequencing and commit groups

| ID | Finding | Priority | Lane / language | Source prerequisite | Focused commit |
| --- | --- | --- | --- | --- | --- |
| FX01 | Ordinary admission opens before confirmed activation | P1 | F / Go | P00 | `fix(extensions): fence admission until activation completes` |
| FX02 | Cold disabled intent skips external cleanup | P2 | F / Go | FX01 handoff; shared dispatch owner | `fix(extensions): reconcile disabled effects after restart` |
| FX03 | Host supervision retains maintenance lock | P2 | T / Rust | P00 | `fix(tailnet): release maintenance guard before supervision` |
| FX04 | Package backup recovery is absent from startup | P2 | F / Go | F writer available | `fix(extensions): recover interrupted package swaps before startup` |
| FX05 | Obsolete companions accumulate past inventory bound | P2 | T / Rust | FX03 handoff; shared native fixture owner | `fix(tailnet): retire verified obsolete companions` |
| FX06 | Historical release intent poisons later boot reconciliation | P2 | R / Rust | P00 | `fix(release): reconcile current intent after completed operations` |
| FX07 | Missing catalog entry bypasses declared controller | P2 | F / Go | FX01/FX02 handoff; shared dispatch owner | `fix(extensions): reject missing controlled catalog bindings` |
| FX08 | Acquisition timeouts lose their cause | P3 | R / Rust | R writer available | `fix(image-import): preserve in-flight timeout and cancellation` |

F runs FX01, FX02, FX04, FX07 in that order. T runs FX03 then FX05. R runs
FX06 then FX08. The three lanes can progress concurrently; the ordering within
a lane prevents competing edits and repeated setup. FX02+FX05 and FX01+FX06
join for native acceptance. There is no new SDK/wire dependency between lanes.
If implementation proves a shared interface change necessary, stop that overlap,
document why, and obtain a writer handoff before changing the assignment.

Commit each repair with its focused regression tests and applicable owning-doc
clarification. Keep additional joint fixture/acceptance changes in a separate
integration commit only when they do not belong to one repair. The coordinator
alone stages explicitly released file sets and makes commits; never use
`git add -A`, include another writer's files, amend, or rewrite history.

### P00 — execution preflight

- Refresh local HEADs, dirty paths, index ownership, reservations, active commands,
  native fixture owners and existing Muse monitor before granting write ownership.
  Do not fetch, reset or replace the recorded bases merely to reproduce the review.
- Verify the eight affected code paths still match the findings. If another
  writer has fixed one, review and test that exact change instead of duplicating it.
- Reserve Fountain's extension modules/services/router tests to F; Soda's host
  Tailnet runtime and main entrypoint to T; Soda's maintenance and image importer
  to R. R/T hand off any shared service/configuration edit explicitly.
- Use no worktrees. All host-side writes, builds, caches, temporary directories,
  sockets and logs live under `/home`. Preserve `sodaos-local`, unrelated runtime
  state, retained evidence and unpublished release reservations. Fixture guest
  runtime paths are isolated by the existing disposable native fixtures.
- Reuse the existing Muse coordinator only after checking whether it is alive,
  responsive and still owns the run. Obtain acknowledgment of these task IDs,
  changed assignments and the single central checklist before dispatch. An old
  monitor or sent prompt alone does not establish receipt. Do not automatically
  launch a replacement if the earlier run is closed or unavailable.
- Preserve the requested execution settings: Codex GPT-6.1 Sol / High; Muse
  `muse-spark-1.3-contributor`, Contributor ON, XHIGH for coordinator and native
  workers. Verify actual settings and inheritance before implementation. One
  interactive Muse coordinator, three native lanes, no recurring five-minute
  worker checks or timer fallback. Verify the 25-minute same-chat Codex schedule
  through the supported automation route; report unavailable scheduling rather
  than claiming it is armed. The retained closure record did not establish an
  active schedule. Planning does not arm or restart any timer.

Each task below is complete only after its regression, scoped acceptance,
released file/command handoff, focused commit and independent review. Existing
lane tests or a checked box alone do not complete the integration gate.

## FX01 — close ordinary admission until activation is durable

**Failure and owner.** Fountain's `services/extensions/lifecycle_dispatch.go`
publishes a controlled backend in `startControlledBackend` before final observe
and pending-intent persistence in `runActivateSequence`. `manager.go` `List` and
`Lookup` check liveness/draining, so failed activation remains ordinarily routable.
`cleanupFailedEnable` currently leaves that backend admitted. Native admin
settings have an extra guard, but ordinary proxy routes do not share it.

**Implementation.**

1. Use the manager's existing ordinary-admission/draining mechanism to keep the
   optional target closed before it can be published during enable, retry or
   ordinary boot activation. Treat a running backend as an internal lifecycle
   participant until activation is complete; registration alone must not admit it.
   Reuse one side-effect-free manager predicate for the exact package/runtime
   instance instead of adding a durable enabled flag or per-router copies of
   lifecycle policy. For optional packages require current desired=true, no
   pending/busy/drain/maintenance fence, matching live backend and catalog identity,
   and, for controlled packages, a current controller instance with confirmed
   running effects and admission readiness. Snapshot under the mutex, perform
   disk reads outside it, then recheck instance and busy/drain state before return.
2. Open ordinary admission only after exact current controller/intent/artifact
   checks, running effects with admission readiness, successful backend handshake,
   and durable completion of pending intent. Preserve the existing tuple/CAS
   checks and lock order. Do not hold the manager mutex across callbacks or waits.
3. Every failure after backend registration must retain the closed fence. Stop
   the optional backend through existing teardown where needed, while preserving
   the required controller and its private lifecycle callback path. A
   final-observation failure may use the
   existing bounded failed-enable cleanup exception; a persistence failure must
   stay pending/unconfirmed, never masquerade as disabled or completed enabled.
   Do not replay an uncertain activation automatically.
4. Apply that predicate to `List`, `Lookup`, browser admission creation and callback
   consumption, `VerifyBackgroundAdmission`, `BootstrapServiceAdmission`, ordinary
   HTTP/WebSocket routing, and `ControlState`'s admission projection. Bind ordinary
   tokens to their exact issuing package/instance. Revoke/cancel browser request
   admissions when fenced. For a retained live backend, dynamically reject its
   runtime/service tokens through the predicate; permanently revoke those tokens
   when the instance stops. Do not revoke its only runtime token and then reuse
   the instance without redelivery: token delivery currently occurs at `startOne`.
   Keep raw backend snapshots available for diagnosis. Required lifecycle
   dispatch, `VerifyLifecycleAdmission`, its
   authorize callbacks and authenticated manager-control routes remain available
   independently of the optional target's fence. Expose an accurate failed state
   and a usable native Disable/retry action; reads perform no controller effects.

**Regression and acceptance.** Extend `lifecycle_controller_test.go`,
`lifecycle_test.go`, `lifecycle_draining_native_regression_test.go` and the
existing router proxy/callback tests as applicable. Pause the real controlled
activation after backend registration and before final confirmation; assert
no navigation and no ordinary proxy/callback admission. Inject final-observe
unknown/unavailable and completion-write failure; admission must remain closed
after the operation returns, with original generation/desired intent preserved.
Successful completion opens routes once, while controller cleanup remains usable
in every failure case. Previously issued browser/background/service tokens cannot
bypass closure. A successful explicit retry must actually use a background/service
capability afterward, proving retained instances have not lost their usable token.
A direct map-state probe is useful regression evidence
but cannot replace a request through an actual registered ordinary route.

**Done when.** No ordinary request reaches the optional backend before durable
success or after failed enable, and native cleanup remains reachable without
granting new authority to the optional backend. No public schema change.

## FX02 — observe stable disabled intent after a cold start

**Failure and owner.** `services/extensions/lifecycle_dispatch.go` `bootDisableRow`
uses only backend liveness and the in-memory observation journal. A new manager
has neither. The supported offline `forgejo extensions disable` command writes
stable false intent without stopping external resources, so restart can report
disabled without observing or cleaning those resources.

**Implementation.**

1. For a controlled optional package with durable disabled intent,
   obtain a bounded authenticated observation through its required controller.
   Do not treat a missing in-memory entry as proof of stopped effects. Keep
   never-activated packages without an installation/intent entirely skipped.
   Missing optional package bytes, receipt or backend must not prevent the
   required controller's observation and cleanup path.
2. If positively observed stopped with admission closed and its ordinary backend
   absent/stopped, leave generation and desired intent unchanged and report disabled.
   If positively observed running, or an ordinary backend still needs termination,
   persist pending deactivate at the same generation, then use the existing
   controlled-disable sequence: close admission, drain/stop verified effects,
   verify stopped, stop the optional backend if present, clear pending durably.
3. Unknown/mismatched effects are failed or unconfirmed, not confirmed disabled;
   do not invent a new automatic effect authorization for an unknown stable row.
   Keep explicit administrator Disable available for bounded cleanup. Unavailable
   controllers use the existing bounded startup retry policy, not a new loop.
4. Apply this behavior to both startup and explicit manager reconciliation through
   their shared path. Keep optional work after core readiness, suppress it during
   maintenance, and never download an optional archive/image merely to observe
   or clean up. Preserve UUID, generation, credentials and private data.
5. Correct `mergeControlledEffect` and related status derivation so absent backend
   plus absent observation cannot produce definitive disabled status. Status reads
   stay pure; an interrupted pending operation retains its separate observe-only
   settlement path and is not routed into automatic effect replay.

**Regression and acceptance.** Add live-controller cases for stopped, running,
unknown and unavailable cold disabled rows in `lifecycle_controller_test.go` and
`manager_control_test.go`. Exercise the actual offline CLI/store path: start an
extension, stop Fountain while external effects survive, disable offline, restart
Fountain, and prove that the controller is called and owned effects stop. Verify
the optional backend stays unstarted, generation does not advance during boot,
core remains available and never-activated packages generate zero calls/pulls.
Include missing optional directory/receipt/backend and stopped external effects
with a still-running ordinary backend; neither may bypass required cleanup.
Use the existing isolated controller fixture before the disposable native case.

**Done when.** Disabled status is supported by a current observation, the offline
disable/restart path stops residual effects, and unknown outcomes remain visible
with admission closed. This joins FX05 for native host/Project cleanup.

## FX03 — release maintenance admission before host supervision

**Failure and owner.** Soda's `lib/host/src/main.rs` `run_tailnet_host_action`
retains `_maintenance` through the long `runtime.wait` after recreating the host
Tailnet daemon. The shared lock blocks exclusive maintenance until the daemon
stops. Initial adoption of an already-running daemon avoids this path.

**Implementation.** Limit the maintenance guard and start lock to the existing
bounded admission/validation/ensure operation. Drop both, in lock-release order,
before entering the supervision wait. Keep the ensure result and normal signal,
exit and systemd restart behavior unchanged. Do not remove start serialization,
move the wait to a new supervisor, change the lock inode or weaken initial
maintenance exclusion. Both creation and already-running adoption must return to
supervision without holding maintenance admission.

**Regression and acceptance.** Use the existing admission and host-runtime test
infrastructure with a narrow scoped setup helper only if necessary. During
ensure, a competing exclusive lock must be denied; while the daemon supervision
wait is active, it must succeed. Exercise error paths to prove guards are released.
On a disposable native fixture, cause the task-owned daemon to exit, wait for
the actual systemd restart/recreation branch, then take/release the existing
maintenance EX lock while the recreated daemon remains running. Retain PID/CID
and lock observations; no full image build is needed to diagnose this boundary.

**Done when.** Ordinary daemon recreation retains required start exclusion but
does not hold maintenance hostage during supervision. Update lock-lifetime
comments in the owner; no service policy change unless evidence requires one.

## FX04 — recover interrupted package swaps before startup validation

**Failure and owner.** Fountain's `modules/extensions/store.go` calls
`RecoverPackage` only inside replacement when the target already exists. A crash
between `target -> .previous-ID` and staged publication leaves an absent target
that bootstrap/startup never restores.

**Implementation.**

1. Invoke exact per-package recovery under the existing lifetime package-store
   lock before required bootstrap validates/acquires packages and before selected
   optional packages are validated or executed. Route through the lock-bearing
   store API; do not introduce an unlocked second installer.
2. Discover recoverable IDs from trusted catalog selections and existing valid
   installed metadata/intent ownership as appropriate. Handle an absent package
   target explicitly, including exact validated `.previous-<id>` names in the
   supported standalone layout. An ordinary target-directory scan alone cannot
   discover it. Never sweep arbitrary hidden directories, install staging residue,
   or unrelated IDs. Bind recovered manifest/receipt identity to that exact ID.
3. Preserve existing recovery rules: restore a validated fixed backup when the
   target is absent; if both exist, validate the target before retiring that exact
   backup; invalid/incomplete targets or invalid backups stay unavailable and
   preserved. Re-run current catalog/receipt checks after filesystem recovery so
   restoring older bytes does not authorize an obsolete package.
4. Recover before replacement regardless of whether the target initially exists.
   Preserve installation UUID, intent/generation, `.data` and archive cache. A
   missing or mismatched optional package still cannot trigger boot acquisition;
   required fallback acquisition is considered only after recovery and validation.
   A required recovery failure blocks required readiness; an optional failure
   isolates that package and must not execute its bytes. Preserve supported
   standalone prebuilt validation without accepting a missing receipt for a
   catalog-acquired package.

**Regression and acceptance.** Extend `store_recovery_test.go`,
`bootstrap_prepare_test.go`, manager startup tests and offline prepare tests.
Create the actual interruption states at the two rename boundaries; do not merely
call the helper directly. Prove: matching backup + absent target/cache restores
offline with zero downloads; enabled optional backup restores and starts without
acquisition; restored wrong-version bytes fail current catalog admission; valid
target + backup retires only that backup; invalid target retains backup and never
executes. Preserve identity/intent/private-data hashes. With an existing archive
cache, recovery must still resolve the backup coherently rather than bypass it.

**Done when.** Real bootstrap, optional startup and offline replacement callers
perform the documented crash repair before using package bytes. A retained-source
offline-prepare interruption is recovered without network or fabricated intent.

## FX05 — retire verified obsolete companion containers

**Failure and owner.** Soda's `lib/host/src/tailnet/companion/{start,stop,inventory}.rs`
and `runtime/{mod,wire}.rs` retain persistent run-specific companion containers
after reboot loses `/run` receipts. Inventory counts stopped children too; above
128 entries it remains unknown, while stop-only retries cannot reduce the count.
Global teardown integration belongs to `lib/host/src/tailnet/lifecycle/runtime.rs`.

**Implementation.**

1. Reuse existing native ownership inspection: exact container ID, installation,
   Project/run/parent binding, immutable image and create recipe. Labels or names
   alone never authorize deletion; contradictory surviving receipts remain a
   refusal. Establish obsolescence positively from a different verified current
   run or a confirmed absent parent; a failed observation is not absence.
2. During an already-authorized fresh start, under the existing maintenance/start
   and Project runtime locks, retire only verified stopped obsolete incarnations
   before creating a new child. Preserve the exact current run and valid adopted
   running child. Missing ephemeral receipts must not bypass this inspection.
3. During explicit disable, close the start fence, drain starts, stop verified
   owned children, then retire verified stopped obsolete children. Reinspect by
   exact ID immediately before non-forced removal. Preserve current-run reusable
   children, all foreign/uncertain resources, credentials, volumes and images.
   Remove only matching stale ephemeral receipt/socket files when necessary;
   never leave a current receipt pointing to a container this operation removed.
4. Ensure existing overflow is recoverable without fixed-prefix starvation.
   Separate read-only historical-candidate discovery from the existing 128-item
   processing bound and bounded effect batch. Replace the unconditional sorted
   `take(128)` before candidate selection: inspect beyond a protected current-run
   prefix within the existing output-byte/deadline bounds, then select verified
   obsolete candidates for bounded retirement. Current children must not consume
   every retirement slot. Reuse scoped per-Project discovery where it reduces
   work; no persistent cleanup cursor or new database is needed. A bounded pass
   may leave unknown remainder, but available verified obsolete candidates must
   be reachable and removal must reduce the backlog. Reobserve before claiming
   complete cleanup; exhaustion or unknown ownership remains unconfirmed. This
   is scoped lifecycle cleanup, not a periodic GC or registry/image cache policy.
5. Keep observe/state operations read-only. No `podman prune`, name-only removal,
   `rm --force`, automatic credential deletion or blanket container cleanup.

**Regression and acceptance.** Extend companion identity/lifecycle/runtime tests
with missing `/run` receipts, old boot/run identities, a current reusable child,
foreign or contradictory metadata, inspection/removal failure and a start racing
disable. Script inventory above 128 to prove bounded progress and eventual
confirmation without creating 129 native containers. Verify current adoption
does not restart a daemon or require reenrollment.
Include a sorted prefix of 128 protected current children with obsolete children
after it: the latter must be reached and retired while every current child survives.

Extend `companion/native_tests.rs` and reuse the retained official-image child
fixture. With one disposable Project, create two real incarnations across a
restart/reboot and prove the production path retires the obsolete container.
Assert container counts and preserved state before fixture teardown; its extra
`podman rm` must not be the operation that makes the test pass. Join FX02 to show
offline disable plus Fountain restart cleans native host/Project effects.

**Done when.** Repeated Project/host incarnations do not accumulate obsolete
owned containers, and prior overflow can shrink through bounded authorized
cleanup without touching current/foreign resources. New active populations above
the existing bound are not silently supported by removing legitimate children.

## FX06 — reconcile current intent after a completed release

**Failure and owner.** Soda's `cmd/soda-project-maintain/src/release_operation.rs`
loads `last-operation` whenever `operation.json` is absent. `post_commit` then
uses its frozen generations, so a later legitimate administrator change causes
every boot to return exit 75 despite healthy current extension state.

**Implementation.**

1. Preserve the active-operation path: verify/finalize `operation.json` under EX,
   retain its frozen record, drop EX, and validate that operation's post-commit
   outcome against its exact snapshot. Concurrent later intent changes during
   that operation's completion remain detectable and must never be overwritten.
2. When no operation is active, release EX and use the currently deployed Payload
   plus a current authenticated snapshot for normal bounded reconciliation.
   Do not load historical rollback evidence as today's desired state. Reuse the
   existing `post_commit(None)` behavior by returning `Ok(None)` from the absent
   record branch, rather than introducing another journal. Retain its current
   fail-closed checks for pending intent or mismatched selected installed packages;
   this repair does not authorize a broader automatic healing path.
3. Keep `last-operation` intact for explicit rollback and evidence. Rollback must
   still apply its frozen identity/generation checks and reject changed intent.
   Do not weaken `same_intent` globally, delete retained evidence, restore an old
   intent, stage a host, or restart a completed core transaction.
4. Keep real extension-not-ready outcomes typed as exit 75, the existing unit
   retry policy, current catalog/package verification and the 300-second bound.
   Manual restart after exit 75 must inspect current health and can succeed after
   repair without a new host transaction. EX must be released before callbacks.

**Regression and acceptance.** Extend release maintenance tests using existing
operation/snapshot fixtures: completed release, later legitimate Tailnet toggle,
then no-marker boot reconciliation succeeds with current healthy state. Repeat
with a genuinely unavailable extension: exit 75 remains. Assert active-operation
intent changes still fail, no-marker reconciliation does not stage/finalize, and
the retained record remains byte-identical and available to rollback checks.
Also test an unreadable/malformed historical record with no active transaction:
ordinary current-state reconciliation must not depend on parsing it, while explicit
rollback must continue to reject it. Preserve `run()`'s immediate frozen comparison
for the operation just completed.

Use the retained disposable maintenance fixture for one native sequence: completed
release -> native admin toggle -> reconcile service restart (and boot check if
needed) -> healthy current state. Preserve the rollback record and private data.
There is no reason to rebuild an entire source/target pair to debug record selection.

**Done when.** Ordinary boot follows current intent; incomplete current health
still fails accurately; active recovery and explicit rollback retain frozen
transaction protections.

## FX07 — reject missing catalog/controller relationships

**Failure and owner.** Fountain's `services/extensions/lifecycle_dispatch.go`
`adoptMutation` accepts an installed manifest for an ID absent from the configured
catalog. `executeMutation` then chooses backend-only handling despite a declared
controller, potentially reporting successful disable with live external effects.

**Implementation.**

1. Validate the trusted target/controller relationship before acquiring a mutation
   slot, allocating installation identity or persisting changed intent. When a
   configured catalog omits the target, return a clear configuration/catalog
   failure. Never infer lifecycle authority from an installed manifest alone.
2. Permit existing standalone backend-only operation only where no configured
   catalog/controller contract applies. A manifest declaring a lifecycle controller
   without the trusted catalog relationship always fails closed. Do not reconstruct
   a controller tuple from stale receipt/manifest data or silently remove the
   controller field to make the request pass.
3. Make native inventory/status consistent: the retained package remains visible
   with the configuration problem; do not show a successful backend-only mutation
   as recovery or label its effects `not_required`. Exclude it from ordinary
   admission using FX01's predicate. Direct POST remains protected even if a stale
   UI action is submitted.
   Restoring the trusted catalog and explicitly retrying makes proper cleanup
   available again through the required controller.

**Regression and acceptance.** Extend manager mutation tests and native admin
inventory/POST tests. Remove a controlled target from an otherwise valid configured
catalog, retain installed bytes and intent, then submit enable and disable. Both
must fail before generation/UUID/pending changes and perform zero backend or
controller effects. Also cover a missing/unready required controller, a restored
valid relationship performing real cleanup, and legitimate no-catalog standalone
backend-only behavior. Current default Tailnet catalog behavior stays unchanged.

**Done when.** Catalog misconfiguration cannot downgrade a controller-backed
package into backend-only semantics or manufacture a successful disable.

## FX08 — preserve acquisition deadline and cancellation causes

**Failure and owner.** Soda's `cmd/soda-image-import/src/import.rs` converts a
runner failure during pull into import-unconfirmed and maps all inspection errors
to identity mismatch, hiding an in-flight timeout or cancellation.

**Implementation.** Recheck `ImportCtx` after runner failure and before mapping
errors at initial existence, pull, post-pull existence and inspection boundaries.
Preserve timeout/cancellation with their current distinct diagnostic and exit
behavior; genuine digest/platform mismatches and non-timeout command failures
retain their distinct existing meanings. Recheck before reporting a completed
acquisition if the final call exceeded the deadline. Prefer this local change to
a new runner/protocol hierarchy; keep child termination and output bounds intact.
Update `src/context.rs`'s stale comment about flattening context errors. The outer
inspection error mapping must not overwrite an inner context error.

**Regression and acceptance.** Extend `src/tests/import.rs` beyond already-expired
contexts: the fake runner advances/cancels context during pull and inspection,
then returns failure; assert the actual cause survives. Cover the other runner
boundaries using the same driver and verify no later pull or success is emitted.
Retain true identity mismatch and ordinary command-failure cases. Exercise the
existing bounded child-process test where needed to confirm timeout terminates
and reaps the child; no network, image build or native VM is required.

**Done when.** In-flight deadlines and cancellations remain distinguishable from
transport and identity failures without changing required/optional selection.

## Validation ladder and integration gates

### G1 — focused regressions before packaging

Set task-specific `TMPDIR`, `GOTMPDIR`, `GOCACHE`, `GOMODCACHE`,
`CARGO_TARGET_DIR` and any needed Cargo/runtime cache paths to existing compatible
or task-owned directories under `/home`; do not repurpose `HOME` or share an
actively written cache without ownership. Use pinned toolchains/lockfiles.
Run the narrow named regressions first, then these affected package suites once:

| Checkout | Check | Scope |
| --- | --- | --- |
| Fountain | `go test ./modules/extensions ./services/extensions ./routers/web/extensions ./cmd` | Package/store, live lifecycle, actual native routes, offline CLI |
| Fountain | Focused `go test -race` for the new activation/cleanup concurrency cases in `./services/extensions` | Closed admission, callbacks and mutation races; no blanket repeated race suite |
| Soda | `cargo test --offline --locked -p soda-image-import` | FX08, real mismatch/selection regressions |
| Soda | `cargo test --offline --locked -p soda-project-maintain` | FX06, existing recovery/rollback checks |
| Soda | `cargo test --offline --locked -p soda-host --lib tailnet -- --test-threads=1` and `cargo test --offline --locked -p soda-host --bin soda-host -- --test-threads=1` | FX03/FX05 and main/guard regression; no unrelated host integration suite |

Respect the existing build tags and test prerequisites documented by each repo.
Add no network acquisition to tests to hide a missing cache. Run applicable
format/lint checks and `git diff --check` on changed source. SDK/browser builds
are needed only if those inputs actually change; this plan does not require a
wire change. Test failures must be classified as product failure, test setup or
unrelated concurrent work rather than patched indiscriminately.

### G2 — small native boundary checks

Reuse released retained drivers/artifacts after inspecting their resource names,
ports, image identities and reservations. Use disposable fixtures or a released
copy-on-write derivative; never run hard-coded drivers against `sodaos-local` or
an owned live fixture. Prefer SSH/serial/journal text logs over VM screenshots.
Keep command handles, bounded deadlines and exact cleanup ownership.

1. **FX01:** real Fountain backend/controller plus ordinary HTTP route: hold final
   confirmation, induce failure, and prove closed admission and reachable cleanup.
2. **FX02+FX05:** surviving owned effects + offline disable + fresh manager,
   missing runtime receipts and two Project incarnations. Verify no optional
   acquisition, cleanup, current identity preservation and obsolete-container
   removal before fixture teardown.
3. **FX03:** actual task-owned tailscaled failure/systemd recreation, live daemon,
   then successful maintenance EX acquisition. Also retain evidence that EX
   excludes a start still in its bounded ensure phase.
4. **FX04:** exercise actual stopped-package preparation/startup against an
   interrupted swap with network unavailable and valid retained bytes.
5. **FX06:** completed release evidence + later native admin intent change +
   reconciliation, with current readiness and unchanged rollback evidence.

Use ordinary local subprocesses for generic Fountain tests. Native Podman/systemd
checks run inside the retained disposable native environment. Existing G-R
compression and G-T upstream compatibility evidence need not be rerun merely
because this plan exists; changed runtime behavior needs the specific checks above.
If a native assumption fails, return the exact command, identity and result for
design resolution before broadening implementation or building production media.

### G3 — integrated candidate and independent acceptance

After G1/G2 pass and writers release their files, integrate the focused commits.
Build the smallest existing combined development target for the affected Fountain
and Soda binaries/packages. Bind the new Fountain commit/image, package hashes and
Soda payload to the new candidate; retained old image digests cannot prove these
repairs. Reuse compatible retained dependencies and unchanged content.

Verify the repaired paths through the actual packaged services on disposable
native ISO/Docker fixtures as applicable, including enabled/disabled restart,
failed-enable cleanup, host daemon restart with maintenance and post-release
administrator changes. Package thinning and absent-by-default optional acquisition
must remain intact. A full candidate/media build is a final shipping-wiring check,
not a debug loop; if new media is produced, measure its complete rootfs against the
existing fixed compression/size contract. Do not repeat passed full builds without
a changed relevant input or unresolved failure that requires one.

Codex independently checks the eight original triggers on the exact integrated
revision, including the negative authorization/foreign-resource boundaries.
Record compact results with source IDs, native image/package IDs, commands, exit
codes, data-preservation observations and limitations. Keep implementation,
source tests, local native evidence and public qualification separate.

### G4 — closure and external qualification

- All eight findings have a reviewed fix, focused regression and applicable G2/G3
  evidence. No substantive finding is closed only from a worker summary.
- Owning guides describe the corrected behavior; keep transient logs in ignored
  artifacts. Once complete, absorb needed contract clarifications and remove this
  plan under the documentation authority rules.
- Focused local commits exclude concurrent work. No push, publication or merge.
- Clean only exact released task resources, retain unique evidence and unpublished
  assets, and report the actual supervision/cleanup state. Do not stop another
  chat's monitor, daemon, fixture or terminal.
- Public-download qualification remains a separate external gate: the exact
  Fountain image, release metadata and corresponding source archives, plus Soda
  images and native packages, must be published and anonymously retrievable before
  public-byte ISO/Docker and supported-upgrade qualification can pass. This plan
  authorizes neither publication nor a claim that this gate passed.
- Retained native results at Soda `730fc205` / Fountain `a4fd2221` remain evidence
  only for their recorded revisions and scenarios. Unenrolled official Tailscale
  compatibility does not prove external enrollment/connectivity; an unchanged
  Project-support pair does not prove native changed-helper replacement.

## Execution checklist

- [ ] P00: refresh source/runtime ownership; verify execution route and handoffs.
- [ ] FX01: confirmed and durable ordinary admission; focused commit/evidence.
- [ ] FX02: cold disabled observation/cleanup; focused commit/evidence.
- [ ] FX03: bounded maintenance guard lifetime; focused commit/evidence.
- [ ] FX04: startup/offline package swap recovery; focused commit/evidence.
- [ ] FX05: verified obsolete companion retirement; focused commit/evidence.
- [ ] FX06: current-intent boot reconciliation; focused commit/evidence.
- [ ] FX07: trusted catalog/controller relationship required; focused commit/evidence.
- [ ] FX08: in-flight timeout/cancellation diagnostics; focused commit/evidence.
- [ ] G1/G2: affected suites and small native boundaries pass on released revisions.
- [ ] G3: packaged integration and independent review accept all eight repairs.
- [ ] G4: owning docs, focused local commits, precise cleanup and residual gate report.

At execution, incorporate these IDs into the existing coordinator's one central
checklist. Keep this file as the task specification, not a second live progress
record. Workers report code/evidence and released ownership; the coordinator owns
integration, and Codex owns independent acceptance.
