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

**In progress.** Additional populated-runner scroll/focus/draft and real native
profile-draft Back/Forward cases are authored in the existing consumers. Strict
TypeScript/emitted build and the local synthetic connection suite pass. Their real
native fixture run is currently blocked on missing local fixture/private inputs
(`localhost:3300` unavailable and the documented credential file absent), not editor
ownership. Neither new case nor populated visual acceptance is claimed passing;
see the leading handoff for the failed attempt and remaining gaps.

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

**Source/procedure preparation implemented; actual execution inputs remain
unselected.** Use `tests/installed/sodaspaces.ts`, its existing guard tests and the
[runner native guide](runners-native-validation.md). The driver is wired. Its native
observer now supplies recursive cgroup membership, boot identity and same-boot prior
PID/start/UID survival checks; local fixture tests cover churn, bounds, PID reuse,
escaped survivors and lifecycle postconditions. The guide's cases A–E specify the
registration/job and idle/active sequences, native UI/CLI overlap plus bounded
cancellation, preserved activation/reboot and provider aftermath using existing tools.
No native/provider scenario or real browser/driver proof follows from those tests.
Fill and review the guide's exact execution proposal before claiming this step's
approval-ready input exit; do not guess targets or credentials.

| Scenario | Remaining preparation and required observation |
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

**Separate approval checkpoint:** select the actual retained target and affected
maintenance window. Existing `soda-test`/validation-VM permissions are not a fresh
rollout grant. Require applicable step-4 parity evidence first.

- Perform fresh read-only inventory of exact running bytes/schema/configuration,
  customizations, runner/project identities, work/credentials, units/policy, jobs
  and browser terminals. Review all interruptions and private inputs.
- Take fresh application/config/key/artifact backups and consistent runner backups
  under explicitly approved quiescence. Otherwise label hashes as observations,
  not recoverable backups. Rehearse actual migrations and artifact pairing on copied
  state, without cloned live provider listeners.
- After concrete cutover approval, stop new management admission, wait for old
  Cockpit/CLI/helper writers, deliver the compatible affected set and retire the
  obsolete helper. Do not rerun setup, regenerate credentials, replace roots or
  restart listeners implicitly. Preserve already-delivered compatible components.
- Verify running bytes, native page/Actions coexistence, both authorities, Cockpit
  fallback and preserved project/runner state. Execute only approved retained-target
  checks, never the destructive disposable-fixture suite.
- On failure preserve new writes and partial state; assess compatible restoration.
  An earlier snapshot is not automatically lossless rollback.

Follow [installation](installation.md#existing-state-dashboard-migration) and
[credential/schema rehearsal](dashboard-credentials.md#controlled-existing-state-rehearsal-before-live-deployment).

**Exit:** the approved target serves the verified combined candidate with bounded
preservation evidence. Cockpit Runners still exists until the next coordinated step.

## 6. Retire only the Cockpit runner presentation

After applicable delivered parity, make one coordinated source/removal change:

- Remove `cockpit/soda-runners/` and exclusively used UI/store/transport/dependencies
  only after their behavioral coverage exists at the new owner. Audit imports.
- Update Vite/package inventory/staging/payload tests and port the installed operator
  check's runner section. Keep its Tailnet and root-authority checks.
- Preserve runner backend/CLI/native tests, launch/client/service/sysusers/tmpfiles,
  upstream Actions templates and all runner accounts/state/provider records.
- Preserve Cockpit Tailnet, React/PatternFly dependencies it uses, Services/Logs and
  ordinary operator/root administration. No whole-Cockpit removal.
- Build/check the changed removal candidate and deliver only its approved delta.
  Remove obsolete page/assets on accepted targets; keep fallback elsewhere.

**Exit:** source and accepted installed targets no longer ship/link the old runner
page, native Runners remains functional and Tailnet/ordinary Cockpit still work.

## 7. Close documentation and the bounded completion record

Keep documentation current throughout, then finish the cross-guide audit:
architecture, API/credentials, installation/staging, CI/operator guidance, Project OS,
Lit/design, native validation and AGENTS. Use the actual native **Runners** destination;
remove interim fallback wording only where retirement really occurred.

Report separately: combined source/browser acceptance, exact native build/export,
isolated page/provider/process proof per architecture, retained-target delivery,
Cockpit retirement and any missing acceptance. Preserve historical failures and
restricted evidence locations. Do not claim whole-product/release acceptance.

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
