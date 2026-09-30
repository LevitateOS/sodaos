# Testing and native validation

How to validate Soda changes. Product journeys and native proof requirements live
here; tool transport lives in [Native support](native-support.md).

## Layers

| Layer | What it proves |
| --- | --- |
| Source checks | Go tests, TypeScript typecheck/lint, focused frontend suites |
| Authored browser fixtures | UI contracts against local/fixture servers |
| Native stage checks | `scripts/check-native.sh` against a prepared matching-architecture stage |
| Installed journeys | Real appliance/project behavior on an authorized target |

Cross-compilation or emulation is not native proof. x86_64 and aarch64 evidence
are independent.

## Source checks

Choose checks for the change. Common entry points are listed in
[AGENTS.md](../../AGENTS.md#commands). Examples:

```sh
go test ./internal/project ./internal/web/...
bun run typecheck
bun run test:frontend
bun run check:source
```

## Native stage

```sh
bash scripts/check-native.sh ARCH /ABS/PATH/TO/stage
```

Stage preparation and candidate production: [Native support](native-support.md).

## Product journeys to cover when touching those areas

- Project foundation: join, terminal, files, Git, shared tools, nested service
- Spaces: create/key/join, native extension page and persistent panel across
  Forgejo navigation and account changes
- Managed terminal: reserve/create/exact attach/End, reload survival
- Operator runners: retained observation/cleanup and refusal of deferred local
  creation; native CI runs on separately managed capacity
- Operator Tailnet: host settings and project enrollment policy
- Installation media and activate flows when those owners change

Exact script names and fixtures live under `tests/` and `scripts/`. Prefer extending
existing drivers over inventing new harnesses.

## Factory acceptance

This section owns how to demonstrate the finished factory, not an implementation
sequence or a checklist required for every documentation or code change. Behavior
and limits belong to the [product contract](../product/overview.md), environment
ownership to [Projects](../product/projects.md), session behavior to
[Spaces](../product/spaces.md), and authority to [Trust](../architecture/trust.md).
The [architecture decisions](../architecture/overview.md#factory-architecture-decisions)
select the boundaries these checks exercise.

### Finished product demonstration

On an authorized installed appliance with the selected Soda, Fountain and Project
OS versions, enable a repository under its separate repository, capacity and
provider grants. Use a supported subscription CLI, a small real codebase with an
approved toolchain/service requirement, and native CI on separately managed
capacity. Keep a joined human's dirty checkout, terminal and service data present.

Demonstrate an issue and a dependent issue progressing through the actual product:

1. Readiness exposes both a declared dependency and a material unanswered
   requirement in the fixture. An authorized answer resolves the question;
   merely closing an unmerged code prerequisite does not satisfy its outcome.
2. Ready work automatically acquires the environment and runs the selected coding
   CLI through the broker, without an operator `admit` command. Spaces displays
   that process's live activity and issue/attempt identity.
3. Coding publishes the exact native PR candidate. Another agent reviews it in the
   same Project with fresh Git state. Demonstrate a blocking finding or actionable
   CI failure returning to the coder, a changed candidate, fresh review and new
   required checks. If the ordinary candidate needs no correction, use a separate
   bounded case to demonstrate this path; do not fabricate an agent finding.
4. The exact reviewed head and verified base pass the complete configured check
   set and the conditional native merge. Soda confirms the merge and required
   issue outcome, then automatically reconsiders and starts the eligible dependent.
   No person performs the routine merge or manually admits the dependent issue.
5. The recorded container identity stays the same through coding/review/fix.
   Human dirty files and service data remain intact; ordinary factory run
   termination leaves the human terminal and service running. Ending a browser
   attachment does not stop factory work.

The demonstration must include real provider executions, native collaboration
records, the real broker and rendered Spaces routes. Mock transport, synthetic
credentials, a terminal marker or an agent's success narrative cannot substitute
for this composed result. Short deterministic fixtures remain appropriate for
the controllable rejection and race cases below; do not spend provider quota
merely to exhaust timers or simulate duplicate events.

### Acceptance cases

| Area | Observable pass and refusal criteria | Required evidence |
| --- | --- | --- |
| Authorization and accepted input | Separately recorded grants permit unattended accepted work even after browser logout. Missing/withdrawn grants or unadopted issue changes deny affected execution and writes. Agent instructions cannot adopt objectives, alter policy or enlarge authority. | Focused policy checks plus native unattended writes and invalidation ordering under the [mutation contract](../architecture/trust.md#conditional-native-mutations). |
| Readiness and observations | Native issue/dependency/comment evidence produces a specific blocker and accepted resolution causes reassessment. Cycles and inaccessible evidence stay blocked. A missed notification is recovered by bounded reconciliation; duplicates and unchanged blockers cause no duplicate agent or repeated model assessment. | Native issue/answer/dependency journey; deterministic duplicate, missed-event and cycle checks against current native rereads. |
| Assignment and preparation | Actual runs identify accepted inputs, approved base, template, role, checkout, harness/model/connection and required outputs. Authorized absent-Project creation and existing-Project reuse both prepare approved tools and assigned services. Unsupported ownership/profile or incomplete preparation prevents launch without destructive replacement. | Inspect native run assignments and selected Project OS creation/reuse with a representative repository tool and service. |
| Role and credential boundary | Each selectable factory harness runs as its assigned non-login role with fresh home/config/auth and a current process binding. Role-private Git state and human data remain protected; publisher secrets and privileged sockets are absent. Capture and stop cover descendants while unrelated workloads continue. Failed return/termination prevents connection reuse; stale container/run bindings are rejected. | Bounded real CLI/broker execution for every harness exposed as selectable, plus targeted process, permission and stale-binding checks. Enrollment or an old disposable-worker receipt alone does not qualify a factory harness. |
| Candidate, review and correction | Native issue-linked publication identifies the exact head; a distinct reviewer receives accepted requirements and candidate evidence without the coding conversation. Independent writable Git metadata prevents shared local configuration. Reviewer edits cannot publish or approve themselves; correction changes the candidate and invalidates earlier review/checks. Disputed/non-progressing work reaches intervention. | Real coding/review/correction evidence in the same Project, native commit-bound reviews and focused role/revision enforcement checks. |
| CI and conditional mutation | The nonempty required-check set and native protections remain effective. Missing, skipped, pending, cancelled or failed evidence, unauthorized changes to approval definitions, or stale head/base/accepted-input/policy/authority cannot produce an automatic merge. A lost response is reconciled before any retry. | Native selected-Fountain checks for head/base changes after preflight/preparation and authority invalidation racing the effective write. Record both orderings: invalidation first rejects; a write committed first is reported accurately. Pending invalidation must not appear as completed cancellation/revocation. Also exercise publication/review authority loss, native protection refusal and ambiguous outcome lookup. |
| Capacity and spending | The [configured limits](../product/overview.md#concurrency-and-usage-limits) bound appliance/repository concurrency, processes, active time, correction cycles and aggregate leased connection time. Queues identify their resource wait and select the oldest eligible ready issue; blocked work holds no execution slot. Duplicate observations, input edits and resume do not replenish allowances; explicit retry preserves history and rolling use. No account/provider or paid fallback occurs. | Deterministic accounting and reservation checks, including multiple repositories sharing a connection, plus native resource/lease enforcement. Use short test limits or controlled clocks instead of hours of provider execution. |
| Spaces and human access | Authorized viewers see actual factory output with correct issue/run identity; input and human-terminal End cannot control it. Navigation, splits, hide/show and closing/reopening views preserve the execution binding. Session/permission loss disconnects views and cached pages cannot reveal another actor's private state. Human Join, member-only terminals/End, explicit-key SSH/SCP/SFTP and ordinary Git remain usable. | Actual factory and human browser journeys through the native host, including forged input/control requests and access loss. Mocked components and native tmux attachment alone do not pass. |
| Intervention and persistence | Block/pause/cancel stop affected agents, preserve work and deny further writes. Cancellation survives issue reopening and new events; only an authorized explicit retry can create another attempt. Takeover follows confirmed stop and credential return/revocation into a member-owned checkout. Project Stop holds admission; Start preserves roots without restoring old runs/leases or independent grants. Controller interruption never silently resumes or duplicates work; failed cleanup remains distinct from outcome. | Coordinated native pause/resume, cancel, takeover, Stop/Start and interruption checks with dirty work and service data present; inspect remaining allowances, process identities and lifecycle authority refusals. |
| Retained appliance support | LAN routes and service endpoints work without purchased-domain or Tailnet prerequisites; optional Tailnet preserves operator/opt-in policy. Private HTTPS/native origin, bootstrap/helper bounds, stock Cockpit, branding/attribution and separately managed native CI remain usable. Local runner creation remains deferred. | Reuse applicable existing operator, connectivity, human-development and UI checks; verify changed installed paths without building parallel support services. |
| Delivery and preservation | Exact selected source/runtime versions qualify on each affected native architecture. Installation/update/recovery preserve protected existing state. Obsolete auth/factory callers and fixtures are replaced together; no parallel legacy runtime is needed. | Existing [release qualification](release.md) and [native support](native-support.md) evidence for the shipping candidate. One architecture's receipt or cross-compilation does not qualify the other. |

Every advertised coding/review harness must pass its actual launch, result,
credential and stop contract. Do not infer primary Muse factory support from
enrollment or optional tool-worker support. Qualify each selectable harness's
boundary with a small native case; one representative complete product journey
need not be repeated for every combination of otherwise unchanged components.

The first [conditional operation contract](../architecture/trust.md#operation-identity-and-authorization)
specifies `pull_request.merge`. Its focused acceptance additionally exercises:

- Identical-ID replay without a second execution, changed-intent refusal, and
  cancellation arriving before submission or after a lost submission response.
- Spoofed actor/installation/target refusal, expiry at guarded commit admission, and
  cancellation by the owning installation after its native actor loses write rights.
- Both cancellation/write orderings and restart with an in-flight write, preserving
  operation identity and denial of replay rather than inferring failure from timeout.
- Ref success followed by native bookkeeping failure: retain the committed result,
  withhold factory completion/dependent pickup, and reconcile without another merge.
  Where attribution cannot be recovered, report indeterminate and require intervention.

The [enforcement span](../architecture/trust.md#native-enforcement-placement) needs
controlled pauses after prepared-hook approval but before ref publication. Race
cancellation, a source-branch update and a relevant native policy/input change
against that paused writer; also exercise controller death while the receiver
survives. Prove coverage for selected HTTP/SSH and direct native writers. Do not
infer this from a preparation check, PR mutex, reflog entry or missing completion
callback. Under the [selected reservation protocol](../architecture/trust.md#selected-coordination-mechanism),
also demonstrate:

- Atomic idle/revision observations bracket authoritative input reads; changes before
  claim, including issue/dependency changes before Soda receives an event, reject
  the old revision. Unrelated participating changes may conservatively reject too.
- Competing native ref and policy/input mutations join the gate before taking
  native locks. Required synchronous callbacks reenter without deadlock; forged
  bindings and deferred jobs cannot borrow the owner's authority.
- Controller restart and elapsed deadlines cannot clear an occupied reservation.
  Recovery distinguishes writer quiescence from effect attribution; an unresolved
  outcome retains the fence and does not launch replacement work.
- Expiry before guarded admission rejects. Admission before `not_after` may publish
  afterward; a paused admitted writer keeps cancellation pending until resolved.
  This tests the explicit admission deadline, not a physical-publication deadline.

First prove one real native merge with a competing input/ref mutation, cancellation
and controller loss around the prepared checkpoint. Do not build broad factory
orchestration to test this primitive. That focused proof does not replace the
required inventory and coverage checks for every participating writer family.

The [selected integration boundaries](../architecture/trust.md#background-authentication)
add these focused cases during implementation:

- Runtime and external-service bootstrap work without a browser. Verify both Unix
  peers in the actual container/user namespaces; reject an unbound/ambiguous peer,
  spoofed installation/actor, stale admission and native PAT scope/resource loss.
  Owned cancellation remains available after actor-token or binding withdrawal.
- Replacement/restart retain installation identity; remove/reinstall cannot adopt
  old operations. Rotate service admissions without resubmitting accepted work.
  A duplicate cannot replace its recorded native credential with a fresh token;
  regeneration under the same token ID also invalidates the old credential binding.
  Initial submission requires full current-secret verification even after a cache hit.
- Native hooks require the execution capability as well as internal-channel
  authentication. Reject forged generation, phase, ref effects and policy push
  options; preserve only the existing unresolved execution's bounded callbacks
  across restart. Check credential redaction and retirement after reconciliation.
- Exercise a transaction-plus-direct-Git writer such as `DeleteBranch`, an Actions
  task/status update and a deferred push worker. Missing ownership refuses before
  effects; busy work remains queued and no callback waits on its parent's gate.
  Authority revoked between route authentication and claim cannot survive as a
  cached permission. Complete the participating-writer inventory before enablement.
  Native post-publication Actions notifications retain their bounded completion
  ownership without deadlock or dropped effects; runner updates claim fresh ownership.
- Qualify offline reconciliation with the whole native writer domain stopped and
  restart inhibited. Wrong owner/generation and uncertain attribution refuse
  release. Native admin cancellation alone never provides a force-unlock path.
  Include an interrupted ordinary writer; quiescence or a merge-tip comparison
  cannot substitute for accounting for its own native effects and consistency.
- The first conditional method is `fast-forward-only`: require the exact reviewed
  result, native permission for that method and a real target change. Other methods,
  divergence and no-op updates refuse without policy changes or fallback.

### Recording completion

Retain a scoped receipt with exact source/image/CLI versions and native architecture;
issue, PR, repository, Project, attempt/run and native process identities; accepted
input and policy revisions; reviewed head/base and check results; confirmed native
merge/issue outcome; usage and cleanup outcome. Use non-secret credential/lease
identifiers only. Record what was actually exercised, including failures and any
unsupported boundary. Useful diagnostics and detailed receipts belong in task
artifacts/history, not additional product status documents.

Focused development proof precedes production qualification when the native
boundary is uncertain. Reuse valid unchanged evidence at its stated scope. A failed
release run stays failed; its artifacts may inform a separately identified
development check, not a relabeled or resumed successful qualification. Never
claim the finished factory from source checks, the narrow
[synthetic experiment](../research/factory-capability-map.md#focused-native-findings),
or Jev agreement. Do not build the release merely to approve architecture prose.

## Evidence rules

- Distinguish authored checks, local tests, native builds and installed evidence.
- Protect credentials, unrelated work and explicitly retained state. Keep useful
  failure diagnostics; exact task-owned disposable fixtures and artifacts may be
  cleaned within task authorization.
- A passing receipt supports its stated scope only.
