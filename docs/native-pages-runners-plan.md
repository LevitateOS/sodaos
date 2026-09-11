# Native pages and Runners — combined completion plan

This is the active execution plan for finishing **native Soda pages and the
Cockpit-to-native Runners move**, under one implementation owner. It replaces the
remaining execution sequences in [Soda-pages](forgejo-soda-pages-plan.md) and
[Runners](runners-port.md), not their product/security contracts or historical
receipts. It does not expand into every deferred SodaOS feature.

The [current handoff](implementation-status.md) records actual execution. A plan,
source test, native build, installed proof and delivery are different evidence.

## Ownership and fixed boundaries

- One agent owns the combined source, tests, candidate and coordinated delivery
  preparation. Former cross-agent file reservations are superseded. Do not permit
  concurrent edits or introduce another browser/authentication harness, manifest
  format, orchestrator or readiness gate.
- Forgejo owns its native shell, identity, permissions, Git and provider Actions.
  Use its supported templates/assets/APIs; no fork or replacement authority.
- Soda owns its additional sessions/grants, project access and local runner
  capacity. Native site administration is not Soda operator authority.
- Preserve native page/drawer owners, exact terminal IDs, finite retention/Return,
  HTTP End and no-replay semantics. Document disposal detaches, not End. Never
  replay a mutation or automatically create/start/register on navigation/history.
- Preserve all retained roots, accounts, credentials, work, memberships, units and
  later writes. Cockpit Runners remains until delivered replacement parity;
  Tailnet and ordinary Cockpit administration remain afterward.
- Routine authorized local implementation/build/testing can proceed. Real target
  activation, provider resources/jobs, lifecycle/interruption, reboot, destructive
  removal and retained-target delivery require explicit applicable grants. An
  input file or opt-in flag checks scope; it does not confer authorization.

## Starting point — do not redo completed work

Source through `d866c12`, with the ownership update at `f0361a4`, includes:

- All three native bodies: Spaces, operator Runners and repository Spaces settings;
  native connection/consent, coordinated non-atomic logout, navigation retirement,
  schema-v9 context and a versioned module graph.
- Corrections for late entry side effects, real history owner restoration,
  unconfirmed-mutation remount suppression and canonical-epoch template tests.
  The previous stale-URL source blocker is fixed; focused Go regressions passed.
- Runner backend/API/CLI serialization and preservation regressions; gated phase
  inputs, native state/job-proof readers and official exact-run provider calls.
- Installed runner driver wiring with separate actor cookie contexts, fresh role
  checks and page/actor/path/body-bound single-use mutation admission. Local
  source-backed doubles cover the branch/guard, not installed browser execution.
- Isolated checksum-verified Go 1.26.7 and Bun 1.4.2. Candidate `0d0071d` built/sealed
  on native x86_64 after the official Rocky CRB packaging correction, but its native
  check failed on the since-fixed template assertions. That is not a passing
  receipt or export for the latest combined source.

No new editor handoff or brainstorming is required. Remaining implementation and
acceptance gaps are the tasks below, not reasons to pause independent local work.

## 1. Close combined local page and browser acceptance

**In progress.** The additional populated-runner and real native profile-draft
Back/BFCache/Forward cases now pass on the authorized Mac fixture at `0f1d2b1`.
That run corrected subpixel assertion rounding and a real tablet-width native
navbar overflow. All 16 consumers and the native parent passed without skipping
required cases. Runner operation responses remain synthetic; automated geometry
is not complete populated visual acceptance. Wider gaps below remain open. The
Linux builder prerequisites and mandatory page gate subsequently passed in step 3;
the wider gaps below remain open. See the leading handoff for exact evidence.

Review current production callers and existing tests
against the remaining gaps, reusing applicable receipts for unchanged behavior.
Fix reproduced defects directly with their regression tests.

Complete the existing page/Forgejo/workspace fixtures for:

- Real Back/BFCache and the wider combined navigation/history sequence across all
  three pages and the repository drawer: one live owner, no stale listeners or
  duplicate mounts, original-actor validation and usable restored controls.
- Late responses that ignore abort, departed entries, Retry, actor changes and
  coordinated logout/partial failures. No delayed OAuth navigation after retirement,
  secret restoration, mutation replay or unsafe remount of an unconfirmed action.
- Unsaved native forms and project/settings drafts; exact terminal identity,
  retention, Return and named End behavior. Do not replace process proof with a
  rendered terminal or a synthetic history event.
- Operator without site-admin, site-admin without operator, owner/member,
  missing/private repositories and unavailable provider. Keep native-only escape,
  first-consent/missing-scope behavior and JavaScript-disabled limits explicit.
- Genuine old/new-payload cached-client behavior, including transitive modules and
  the final CSP/payload contract. An ordinary cache hit or fresh profile is not an
  upgrade test. Preserve old clients/evidence and never manufacture an old receipt.
- Populated dark/light, narrow/wide, scrolling, focus/keyboard/profile-menu visual
  review. Use [the screenshot guide](screenshot-capture.md) and authorized local
  fixture; no tokens or credential-entry screenshots/traces. Screenshots complement
  behavior tests, not native process acceptance.

Use existing local fixture authorization only. If a required new fixture/action
falls outside it, identify that exact gap without blocking unrelated tests.

**Exit:** an explicit gap list is closed with applicable source/browser evidence,
including the remaining items in the [page review](forgejo-soda-pages-review.md).
No unsupported whole-product, physical-keyboard or installed-process claim.

## 2. Finish installed runner scenarios and execution inputs

**Source/procedure preparation and isolated execution-input selection complete.**
The exact approved fixture/actors/resources and bounded step-4 results are in the
leading handoff; these are not defaults or a grant for another target. Use `tests/installed/sodaspaces.ts`, its existing guard tests and the
[runner native guide](runners-native-validation.md). The driver is wired. Its native
observer now supplies recursive cgroup membership, boot identity and same-boot prior
PID/start/UID survival checks; local fixture tests cover churn, bounds, PID reuse,
escaped survivors and lifecycle postconditions. The guide's cases A–E specify the
registration/job and idle/active sequences, native UI/CLI overlap plus bounded
cancellation, preserved activation/reboot and provider aftermath using existing tools.
No native/provider scenario or real browser/driver proof follows from those tests.
Fill and review the guide's exact execution proposal before claiming this step's
approval-ready input exit; do not guess targets or credentials.

| Scenario | Required preparation and observation |
| --- | --- |
| Entry/admission and operations | Validate real driver/consent integration locally where supported; cover argument/target/actor mismatches, both fresh cookie identities, exact permit consumption, token clearing and failure evidence. Existing doubles are not installed proof. |
| Registration and successful job | Bind an explicitly selected provider registration, unique fixture label, trusted repository/workflow/full commit and observation to exact native runner identity and the returned provider run ID. No latest-run selection, borrowed credentials or automatic registration/publication. |
| Idle and active-job lifecycle | Prepare explicit Start/Stop/Restart sequences, including a bounded hold. Check boot policy, actual native process termination and provider outcome. A listener state or one sampled PID is insufficient for process-tree cleanup. |
| Concurrent callers and faults | Prepare bounded Cockpit/CLI/web contenders on the same disposable runner and separately scoped failures; require serialization, release and truthful partial/unconfirmed outcomes. Existing subprocess/command-double tests do not prove installed overlap. |
| Preserved activation and reboot | Identify a pre-existing baseline, account/credential/work/version/policy comparisons, consistent backup/quiescence needs and separately gated reboot observations. Never start a cloned listener with copied live credentials. |
| Remove and provider aftermath | Verify exact local unit/account/state outcomes and remaining provider record/history. Prepare provider cleanup only as a separately authorized action. Preserve every baseline runner and failed attempt. |

For each case, finish product-owned automation or an exact existing-native-tool
procedure: inputs, commands/actions, expected observations, bounded failure behavior
and retained evidence. Do not leave vague manual checklists or copy scenarios into
support tools. Add focused local failure/admission tests for new code.

Prepare the concrete execution proposal: target/architecture, browser origin/CA,
SSH trust, operator and denied actor, provider repository/registration/workflow,
disposable runner IDs, preservation IDs, restricted credentials and exact effects.
Unknown target/resource choices remain visibly unselected, never guessed from old
VM permissions. Preparing a procedure or credential path does not create resources.

**Exit:** scenarios are reproducible and safely gated, local checks are recorded,
and the exact native/provider proposal is ready for approval. Actual installed
execution belongs to step 4 below; it is not a prerequisite to author these cases.

## 3. Produce and verify one combined native candidate

Subsequent step-4 driver corrections each received newly frozen applicable checks;
latest installed/exported `19824ec` passed the full native x86_64 build/check/export
and independent verification. The original step-3 exit below remains historical,
not a substitute receipt for later bytes.

**Bounded native x86_64 build/check/export complete for
`2cdc23861de5829bb0010e292475b9e7c35add6f`.** Under explicit builder-fixture setup
authorization, created the isolated localhost fixture and restricted inputs and
installed/prepared the required browsers. The fresh repository-owner case exposed
an unsupported read in the existing Go fixture double; it now supplies the existing
synthetic creation profile without allowing native mutations. This supersedes
Mac candidate `0f1d2b1` with a newly frozen, freshly checked revision.

The focused page gate and full `scripts/check-native.sh x86_64` passed, including
all 16 consumers and the real native profile-draft history parent. Fresh build/seal,
11 staging tests, bundle export and independent export verification passed.
Verified export: `.artifacts/combined-candidate-2cdc238/export/x86_64/`.
See the leading handoff for manifest checksum, tools, failures, skips and exact
receipts. No prior candidate's PASS was transferred. The
[mandatory page-gate setup](native-validation.md#mandatory-page-gate-on-a-final-native-builder)
remains required for later candidates. Step 1's wider acceptance gaps, step 2's
exact proposal and all installed/provider/delivery acceptance remain separate;
this completion is not permission to advance into unapproved native actions.

**Earlier builder evidence:** `894b9e8` built/sealed natively on x86_64.
Its mandatory check reached the page gate and failed on absent fixture credentials;
the local Forgejo service and required Chrome distribution are also unavailable.
An independent staging run exposed stale Caddy indentation assertions; those were
corrected without changing routing, and all 11 staging cases passed against the
built tree. Retain the failed aggregate and mixed-source staging receipts; use the
resulting exact revision for subsequent candidate validation. Fresh `6d9a9d9` now
built/sealed and passed all 11 staging tests and integrity verification from its own
frozen worktree; paired Go metadata matches that clean revision. Its full native
check/export remains held until fixture prerequisites are restored. Restoring them
is necessary, not permission to skip the page gate or claim step 1's
remaining acceptance gaps closed. See the leading handoff for exact artifacts.

Freeze a clean committed source after the applicable changes above. Use a fresh
worktree/output; retain previous failed attempts and later writes.

1. Run focused checks during implementation. Use `scripts/build-native.sh ARCH`,
   `scripts/check-native.sh ARCH` and the existing sealed-bundle producer for the
   final native candidate. The native check includes `bun run check:source` and
   staging tests; do not invent a duplicate gate or repeat full suites solely for
   separate plan labels. Run affected race/browser checks not covered by that gate.
2. Confirm the pulled template fixes in the full gate. Inspect the actual sealed
   artifact set, native architecture, versions, hashes and compatibility contract;
   export only after applicable checks pass. Old `0d0071d` output cannot be relabelled
   as the new candidate.
3. Review the affected set together: dashboard image, canonical Forgejo payload and
   cache epoch, required schema, paired `soda-host`/`soda-runners`, Forgejo-only
   launcher and only necessary unit/package changes. Plan retirement of the
   independently callable obsolete `soda-runner-helper` after old writers finish.
4. Keep Cockpit payloads. The producer's Project OS images do not authorize replacing
   existing roots or broadening delivery. Declare effects on active browser
   terminals/projects when restarting shared `soda-host`, not merely runner effects.

Record one exact candidate manifest and preparation receipt. Use available native
x86_64 hardware now; record aarch64 independently, without an emulation claim or
sibling-architecture barrier. Changed bytes or incompatible target state require
new applicable evidence, not rewriting an old PASS record.

**Exit:** passing applicable source/native-build/staging checks, verified export,
reviewed affected set and a concrete rehearsable compatibility recipe. This is
build/staging evidence, not installation or provider parity.

## 4. Execute approved isolated native and provider proof

**Complete for the explicitly approved isolated x86_64 fixture only.** Latest
installed `19824ec` passed fresh full build/check/export, verification and backed-up
paired delivery on `soda-native-runners-2cdc238`. Final native/Cockpit/CLI mutation
overlap, dispatched departure/no-replay, exact probe Remove/provider aftermath and
native page/auth/BFCache checks passed. Registration/exact successful job, idle and
active lifecycle, bounded CLI cancellation and the **single already-used reboot**
have original native receipts for explicitly unchanged production mechanisms;
full changed-path inspection and the evidence composition are in the handoff.
Earlier failed/unconfirmed receipts remain failed, not relabelled or erased.

Only baseline remains running as local capacity. Probe's native account/state were
removed after a verified private evidence archive; both provider records and all
three exact job histories remain. State/inputs/archives/evidence remain under
`.artifacts/runners-vm-2cdc238/`, with the original bounded VM-owner deadline.
This is not whole-product/release or aarch64 acceptance; step 1's wider gaps remain.
Keep Cockpit Runners. No retained-target rollout, new target, additional reboot,
provider cleanup or retirement follows.

**Approval checkpoint:** obtain the exact fixture, activation, provider, lifecycle,
fault, reboot and cleanup grants needed for the selected cases. Approvals can be
bounded separately; omitted cases remain open, not silently waived.

Before activation, inventory the preservation baseline and unsupported provider
state. Resolve unsupported saved providers explicitly; never reinterpret/delete
them. Back up consistently under approved quiescence, rehearse required schema and
paired compatibility changes on copied state, and drain old management writers.
Use one executor/receipt for common activation, not separate UI and runner rollouts.

On the approved fixture, execute the prepared cases through the real proxy, backend,
root socket, systemd and provider. Verify actual delivered bytes, both authority
combinations, registration and successful trusted job, idle/active lifecycle,
concurrent callers/failures, pre-existing state survival and approved reboot.
Perform exact local Remove only on declared disposable capacity and observe provider
aftermath separately. Retain partial states, later writes and all evidence.

Also verify the integrated native page/drawer/authentication/cache behavior on the
actual delivered payload. Cite unchanged local contract receipts rather than
repeating their implementation, but do not call them installed proof.

**Exit:** explicit native page and runner/provider results for the exact candidate,
architecture and action scope. Missing provider or platform evidence stays open;
source rendering and local listener health cannot substitute for it.

## 5. Rehearse and deliver to the approved retained target

**Status: targets inspected; approved checkpoint-B backups and isolated rehearsals passed.** The user selected
step 5 only, `soda-test` first and retained `soda-native-spaces-658f2af` second, and
approved starting powered-off targets. Both were already running on their original
disks; neither was started/rebooted. Checkpoint A observations are retained under
`.artifacts/retained-step5-c71e5ba/`. The user subsequently approved B; both dashboards
were briefly stopped/backed up and resumed on unchanged v6 installations. Actual
copied v6→v9, missing/wrong-key, prior-v9 refusal and matching-prior-v6 cases passed;
private archives were retained on guests and hash-verified on the builder. The
live cutover recipe/window (C), remaining delivery checks and target acceptance
remain pending; step 6 is excluded.
`19824ec` is the current verified x86_64 candidate, exported at
`.artifacts/runner-projection-19824ec/export/x86_64/`, not an instruction to install
it everywhere. Step 4 supplies bounded isolated evidence and explicit continuity
for unchanged production mechanisms, not a retained-state rehearsal. Earlier
`soda-test`/validation-VM grants and the isolated fixture's used reboot/removal do
not transfer here. The new target-specific inspection/start selection above is
separate from those historical grants; it does not approve other targets.

Use one owner, the existing producer/verifier, installed journeys and private
exact-target recipes. Do not build another updater, orchestration layer, manifest
format or readiness database. Follow the current
[installation constraints](installation.md#existing-state-dashboard-migration) and
[credential/schema rehearsal](dashboard-credentials.md#controlled-existing-state-rehearsal-before-live-deployment),
not the historical v3→v5 target recipe as a command sequence to replay.

### 5a. Finish the delivery proposal and relevant page acceptance

Prepare source-backed work locally now. Fill target facts only after separately
approved inspection; keep unknowns visibly unselected.

| Proposal field | Required concrete decision/evidence |
| --- | --- |
| Target and client | Exact hostname/architecture, running versus powered-off state, browser and Cockpit origins, management/client path, trusted CA and SSH pin. Power-on/reboot and routing are separate effects, not implicit prerequisites. |
| Candidate and delta | Full source revision, verified export/manifest chain, actual installed identities and exact added/replaced/removed paths, images, effective units and configuration fields. Recheck compatibility with installed upstream versions/customizations. |
| Authorities and private inputs | Configured Soda operator, nonoperator site administrator and applicable repository actors; separate host-root/Cockpit authority. Select their own approved input files, not isolated-fixture or borrowed CLI credentials. |
| Preserved state | All projects/roots, original Linux accounts/memberships, keys, runner accounts/registrations/work, boot policies, active jobs and exact managed terminal IDs/process scopes. Identify every writer and required backup consistency. |
| Effects and window | Explicit service interruptions and admission closure, client/tab handling, root/project/terminal implications of shared `soda-host`, and any individually necessary package/configuration changes. No blanket upgrade or project/runner lifecycle permission. |
| Checks and failure handling | Named pre/post observations, candidate-specific copied-state results, permitted authentication/refresh effects, failure stop points, matching prior artifacts and a decision owner for restoration with later writes. |

Classify step 1's remaining work by the actual delivery boundary. Do not equate a
missing broader matrix with a reproduced defect, or use bounded acceptance to
waive a known authorization, persistence or data-loss failure.

| Coverage | Before live cutover | Remaining broader acceptance |
| --- | --- | --- |
| Old/new clients and module graph | Exercise genuine predecessor bytes for the selected target through first new navigation, transitive imports and Back/Forward. Verify actual installed revalidation/CSP/epoch behavior, not candidate bytes warmed under old URLs or the preview's six-hour cache. Establish safe retirement/no replay for old entries and document the explicit reload/reconnect policy. If the predecessor cannot be obtained, this transition remains open; do not fabricate an upgrade PASS. | Wider predecessor releases, browser/platform combinations and long-lived client matrices beyond the selected transition. |
| Authority and authentication | Preserve applicable real two-actor results and cover the selected target's actual private/public repository and owner/member boundaries, consent/cancellation, logout, unavailable responses and stale actors. Verify server denials without destructive writes or live provider permission changes. Missing actors/resources require an explicit approved test arrangement, not role equivalence or credential substitution. | Broader provider outages, permission-change and repository combinations not exercised on this delivery. |
| Native UI and drafts | Review populated Spaces, Runners and repository settings in both themes and representative narrow/tablet/wide layouts using the existing fixture/capture tools. Cover usable confirmation/error controls, scrolling/focus, native profile menu and unsaved native/Soda drafts; fix reproduced delivery-relevant defects. | Exhaustive visual/browser and physical-keyboard acceptance. Geometry alone is not screenshot review. |
| Terminals and shared helper | Dashboard/helper shutdown cancels managed terminal ownership, not merely attachment; the selected transition retains this existing behavior. Require a freshly established quiescent managed-terminal set after declared admission closure and writer drain. If any exact session, pending owner or unconfirmed runtime state remains, stop for an explicit decision; do not use restart/logout/expiry forcing to drain it or recreate it afterward. Preserve ordinary SSH/workloads separately and exercise affected maintenance on approved resources. | Broader terminal/CLI suitability and native concurrency matrices outside the affected maintenance boundary. |

The existing native fixture now has a passing, manifest-bound **b8af68c asset-byte**
transition using zero-age revalidation, a fresh browser context, first candidate
module graph and Back/Forward; all normal consumers and the native parent still run.
See [the review](forgejo-soda-pages-review.md#coverage-gaps-that-are-not-additional-proven-production-defects)
and [fixture inputs](typescript.md#local-source-checks). The additional bound-binary
phase now also executes the actual predecessor page/backend, keeps that page open
across a same-fixture-DB v6→v9 transition, exercises its real Refresh control and
observes pagehide retirement without API mutation replay. Actual Back performed a
network reload/current owner, not predecessor BFCache restoration; keep that limit
explicit. All consumers/native parent passed, including a current-handler race run.
This does not replace installed cache/CSP proof or native terminal preservation. The existing
consumers also passed their bounded populated visual slice using the shared screenshot
tool: light/dark at 390/768/1440, runner inventory/confirmation/registration,
repository metadata/Join and workspace navigation. Keep its synthetic-operation and
empty-terminal limits explicit; broader visual/access/profile-menu and native
maintenance evidence remain separate. See the leading handoff for reviewed captures
and preserved failures, not a whole-product visual acceptance claim.

Record applicable existing receipts and the remaining bounded checks in the
handoff. Changed production bytes require a fresh frozen candidate and applicable
build/browser/native regression evidence; retain the original step-4 receipts
rather than relabelling them. No aarch64 sibling barrier is added to an approved
x86_64 delivery; neither cross-build nor x86_64 proof establishes aarch64 acceptance.

### 5b. Approved inspection, backup and copied-state rehearsal

**Approval checkpoint A:** select the exact target and read-only inspection scope.
Inventory running artifacts/image digests and effective unit pins, actual schema,
config consumers, Forgejo version/CustomPath/template/cache overrides, native
clients/units and the preservation set above. Do not dump full container inspection,
secrets or Forgejo's database. Unexpected custom files, unsupported saved providers,
unknown identities or an upstream mismatch stop delta preparation for a decision;
do not reinterpret, overwrite or delete them.

The first inspection found schema v6 on both targets, four/two running projects,
empty local runner inventories and no listed managed terminal units. All 391 current
custom payload files match the prior b8af68c export; selected Forgejo version,
CustomPath and zero static-cache setting match the candidate's upstream contract.
Dashboard/helper bytes and runner-command bytes have different historical provenance;
do not describe either target as a whole b8af68c installation. No old runner-helper
process was observed, but this is not a future writer-drain receipt.

**Concrete same-root prerequisite:** all six retained roots still contain the original
unlocked `project-account` program. The candidate host's embedded key updater must
be paired with the current account program under the
[project maintenance contract](project-os.md#managed-key-writer-contract).
The exact single-file update passed bounded copied-file/ancestor rehearsals on all
six project-derived copies; originals stayed unchanged. This does not prove live
writer exclusion, native provisioning, terminal preservation or label delivery.
Finish those applicable maintenance checks before seeking C;
leaving the old writer or replacing roots is not an acceptable shortcut. Directory
inodes, accounts, key bytes, homes and live SSH/process identities must survive.
Neither this observation nor an empty runner inventory establishes delivered parity.

The later official Forgejo CLI observation found operator ID 1 on each target also
has native site-admin authority; the other two/one native users are nonadmins.
Do not promote users or borrow credentials to manufacture step 4's cross-role pair.
The existing installed **list-only** driver now accepts explicit actual native role
expectations, while still checking fresh native and Soda identities independently.
All effectful phases retain the original cross-role gate. Keep the separate step-4
proof and record actual retained-target outcomes after delivery.

Both targets also retain the same 76-file historical Cockpit payload, matching
multiple prior sealed inventories through `bdbce8e`, not the b8af68c Cockpit export.
The candidate changes the Runners presentation/protocol to Forgejo-only. Pair that
retained entry with its candidate assets (two added files and one index replacement),
retain its prior hashed assets and back up the package freshly. This is **not removal
of Cockpit Runners**. Leave the Tailnet package and ordinary Cockpit configuration
unchanged; no whole-Cockpit upgrade is selected.

Produce the smallest coordinated delta from those facts: dashboard/API image and
canonical template/module graph/cache epoch, schema/config compatibility, paired
`soda-host`/`soda-runners`, launcher and only required unit/package changes. List
unchanged components explicitly. Exported Forgejo/Caddy/Project OS images are not
an instruction to load every image, replace roots or change future-creation defaults.
Identify any obsolete `soda-runner-helper` caller/process before planning its exact
retirement; removing a file does not drain an already running writer.

**Approval checkpoint B:** approve the backup/rehearsal location, private-copy
custody and exact quiescence/interruption effects. Use new exclusive private
outputs. Back up Soda SQLite with its supported backup API, including committed
WAL data, and verify integrity. Retain matching key/config/credentials, prior
artifacts, image pins, units and custom files with ownership/modes/SELinux labels.
Never regenerate an existing grant key. Any Forgejo backup uses supported upstream
mechanisms, not direct database access. Consistent runner/project backups require
their own approved quiescence; otherwise record bounded observations, not a
recoverable live snapshot or permission to stop jobs/projects.

Rehearse the actual source-schema→candidate-v9 migration/config consumers on the
controlled copy. Verify preserved IDs/columns/ciphertext and failure on missing or
wrong keys, plus paired artifact/route compatibility. Do not assume the target is
still v6 or that all intermediate migrations are already proven for its state.
Keep copied grants/provider credentials from authenticating or refreshing against
live services; never start cloned listeners with copied registration tokens.
Use the existing synthetic/isolated tools for effects, not live resources merely
because their state was copied. Retain successful and failed copies privately.

**Rehearsal exit:** reviewed exact delta, successful applicable page/compatibility
checks, a preservation/interruption record and a runnable bounded cutover recipe.
Rehearsal success is not live-cutover permission, and its backup is not current
rollback data for a later maintenance window.

The follow-up read-only sample found all six `/run/soda-terminals` directories
present and empty, with no listed managed units. The prior account/key observations
and 18 sampled system PID/start pairs remain unchanged. This is stronger than a unit
list alone, but **not closed-admission quiescence or future cutover permission**.
The unchanged terminal/dashboard/unit source chain and focused Go/race shutdown
checks are recorded in the handoff. In particular, Hide or document departure is
not End; detached sessions still have owners. Do not mistake browser disconnects
for a safe service-restart baseline. Exact admission/drain and ordinary SSH/workload
preservation observations still belong in the checkpoint-C recipe.

### 5c. Separately approved live cutover and verification

Checkpoint C is now approved for the two named targets in order, with a 15-minute
maintenance allowance per target. `soda-test` has received paired `19824ec`/v9 and
passed bounded native private-page/BFCache, operator/nonoperator Runners reads,
retained Cockpit Runners and all seven memberships' management-forwarded SSH/PTY
checks. Its four roots and sampled system processes were preserved. The first
observer and direct-routing failures remain recorded. Validation VM delivery is
next, not yet performed; step 6 remains excluded. See the leading handoff.

**Approval checkpoint C:** present the candidate/delta, rehearsal evidence, target,
window, exact interruptions and retained-target checks for explicit cutover approval.
If target bytes/state or the proposed effects changed, reassess before writing.

1. Reconfirm the preservation baseline and take fresh consistent paired backups
   under the approved window. Close the declared web/socket/CLI/Cockpit management
   admission paths and establish that old writers have finished. Do not introduce
   a product drain service or assume stopping the socket closes existing handlers.
2. Apply only the reviewed paired delta; preserve explicit image pins, modes,
   labels, credentials and native origins/callbacks. Do not run first-install,
   `soda-setup` or `soda-activate` as an upgrade, recreate roots, rewrite runner
   descriptors, regenerate OAuth applications or implicitly restart listeners.
   If an actual callback change is necessary, use its supported owner workflow
   under a separately named effect; no database edit or secret regeneration.
3. Restart only the approved affected services after their compatible inputs are
   ready. Record the real terminal/access interruption, schema migration and
   running-byte verifier results before reopening management admission.
4. Verify native Forgejo pages/protocols and Actions coexistence, native Spaces/
   Runners/settings, scoped OAuth/cookies, both authority boundaries and cache/
   history behavior using approved actors. Compare preserved project/root/account/
   key/runner/work/boot state and exact terminal identities under the declared
   client/reconnect policy. Use existing approved access, not new keys, joins,
   project starts or routing changes to hide failures.
5. Keep Cockpit Runners as fallback and verify its compatible read path plus native
   root access. Run only named retained-target checks: no fixture registration,
   job dispatch, contention hold, lifecycle, Remove or reboot is inherited from
   step 4. Opening Tailnet may refresh Forgejo advertisement; that existing effect
   needs explicit scope rather than being hidden in a harmless-looking UI check.

**Failure stop:** preserve failed DB/WAL, partial files, jobs, credentials and all
later writes; stop further automatic actions and report what completed versus
what is unconfirmed. Reopening access or restoring files requires a reviewed
compatible state. Never run a pre-v9 binary on v9, lower a schema marker, replay a
mutation or restore an old DB/root snapshot blindly. A prior image alone is not
rollback; authorize any restoration using the matching prior set and an explicit
later-write preservation decision. This is not a new general recovery mechanism.

**Step-5 exit:** target-specific acceptance of the recorded affected delivery and
preservation checks, with all failures/limitations disclosed. An empty runner
inventory proves only the read path, not retained-runner migration parity. If the
actual target cannot establish relevant parity, keep that limitation and Cockpit
fallback explicit. No automatic progression into retirement or release acceptance.

## 6. Retire only the Cockpit runner presentation

**Entry is delivered parity, not step-4 success alone.** Require accepted step-5
results on each target proposed for retirement, no unresolved delivery-relevant
runner/authentication/preservation defect, and a reviewed coverage map from the
Cockpit responsibilities to the native Runners owner. This includes local capacity,
registration/token handling, lifecycle/confirmation, uncertain outcomes, refresh
and diagnostics/provider guidance. Applicable isolated mutation evidence can cover
unchanged mechanisms; do not create or destroy retained capacity merely to fill a
matrix. Missing applicable evidence keeps the fallback in place.

Then make one coordinated source-removal candidate:

- Remove `cockpit/soda-runners/` and runner-only React presentation/store/transport
  only after their applicable behavioral tests exist at the retained/new owner.
  Audit actual imports; no removal of shared helpers/dependencies by directory name.
- Update Vite/package inventories, staging/payload checks, links and installed
  operator journeys. The overlap scenario remains historical pre-retirement proof;
  port ongoing CLI/native coverage without invoking a removed package or copying
  the authentication harness. Keep the ordinary root/Tailnet checks and their gates.
- Preserve `internal/runners`, host/API/CLI/launch/service/client contracts,
  sysusers/tmpfiles and focused native/security tests. Do not remove provider-owned
  Forgejo Actions templates, accounts, units, credentials, work or provider records.
- Preserve Tailnet, its backing logic/tests and React/PatternFly dependencies,
  Cockpit Services/Logs and ordinary administration. No whole-Cockpit removal.

Build/check/export and verify this **new removal candidate**, with focused evidence
that native Runners still works and remaining operator pages are intact. Rehearse
its packaging/removal delta. Obtain explicit per-target removal-delivery approval;
step-5 approval does not authorize this later source/payload change. Delete only
inventoried obsolete package/assets/links after checking actual occupants. Preserve
customized/unexpected files for a decision. Unaccepted targets retain their installed
fallback and are not silently upgraded by publication of a new source tree.

**Exit per approved target:** no obsolete runner navigation/package remains, native
management and ordinary Cockpit/Tailnet still work, and unchanged runner/provider
state is verified. Record source removal and each installed removal separately.

## 7. Close documentation and the bounded completion record

This is documentation/evidence closure, **not a further deployment, cleanup or
release grant**. Update guides alongside each actual phase, then audit architecture,
API/credentials, installation/staging, CI/operator usage, Project OS, Lit/design,
native validation and AGENTS. Name native **Runners** at `/?soda-view=runners`.
Remove interim fallback wording only for actually retired targets; distinguish
historical receipts from current-source contracts without erasing failures.

Keep the existing handoff as the completion record, with these separate outcomes:

| Boundary | Record independently |
| --- | --- |
| Source/page acceptance | Exact revisions/checks and remaining step-1 predecessor, authority, visual and browser/keyboard gaps; no generic “UI complete” claim. |
| Build/export | Candidate, architecture, verified manifest/artifacts and actual checks/skips; no claim of reproducibility or installed proof from a build. |
| Isolated native proof | Original per-case revisions/results and explicit unchanged-mechanism continuity; `19824ec` bounded x86_64 completion does not relabel earlier jobs/reboot. |
| Retained delivery | Selected target, exact installed delta, schema, backups, interruption/preservation receipts, acceptance and residual risks. Unselected/unexecuted targets stay explicit. |
| Cockpit retirement | New source candidate and separately approved target removals, including remaining fallback installations and preserved operator functionality. |
| Broader acceptance | Outstanding aarch64, whole-product/release, intended-client routing, CLI/terminal and wider matrix work with its existing feature owner. No sibling-build barrier or silent waiver. |

Retain all private evidence, credentials, archives, fixtures and later writes under
their existing custody rules. A tool-owned VM hold/deadline is not an extension or
new power-cycle grant. Do not clean up provider records, fixtures or old worktrees
as part of closeout. If checks remain open, finish a bounded handoff with explicit
next work—not a percentage, whole-product acceptance claim or another parallel plan.

## Explicit exclusions and decisions still needed

No Runner OS/OCI migration, cache service, scheduler, drain/admission product,
reconciliation, GitHub runner restoration, Project OS desktop expansion, new
provider authority, general recovery/updater or project deletion is added here.
Those remain under their existing plans/deferred boundaries.

No user design decision is needed to continue local work. User input is needed
when selecting/approving real targets, provider resources, restricted credentials
and effects for steps 4–6. If a concrete architectural gap appears, present the
source-backed limitation and smallest decision needed rather than inventing a
new subsystem or silently expanding authority.
