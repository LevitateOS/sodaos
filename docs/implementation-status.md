# Implementation status — single-run build and release

This handoff is exclusively about replacing SodaOS's two build lanes with **one
source-to-qualified-release run**, producing the same immutable host/application
candidate for installation and updates. The [release engineering plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns implementation order, contracts, tests and acceptance criteria; this file tracks
current progress, blockers, custody and permissions, not a second task list.

Unrelated workstream and development-fixture records were moved intact in scope to
[the retained development handoff](development-handoff.md). They are not active tasks
in this implementation and are not implicitly available as qualification fixtures.

## Current position

**Full replacement planned; implementation has not started.** The plan is committed
at `f2f3a63`. No replacement controller, image-based installer, connected native
qualification or unattended publisher has been implemented by that planning pass.

The existing `d054a60` extraction shared component commands but retained both
assemblers and grew production orchestration. The owner rejected it as sufficient
simplification. Its source tests and `fde23d0` host-context preparation receipt remain
valid only for their stated scope; they do not satisfy the new lane's acceptance.

Current production still splits into:

- Native writable bundle → stock CoreOS installer ISO.
- Derived immutable host/application candidate → separate release-delivery tooling.

The replacement must remove that split, not put another wrapper around it.

## Milestones

These are the six release milestones from the [owning plan](release-engineering-plan.md#9-implementation-stages-and-exits).
The summaries below report their scope and current exit; detailed requirements and
checks remain in that plan. Earlier scoped evidence is retained, not reset by the
rewrite, and does not imply the new single-run implementation is complete.

### 1. Complete appliance candidate

**Status: complete for the existing local x86_64 candidate only.** Criteria 1–7.

- Covers the derived CoreOS host, five application images, immutable Forgejo
  presentation, image-owned packages/defaults, retained project-image storage and
  complete digest/provenance records.
- Local exit passed at `45ac843`: a complete host/app candidate built and inspected
  from frozen source. Installation, update and recovery were not proved.
- The replacement still has to produce this complete artifact set through one
  controller without writable staging or duplicate component builds.

[Detailed acceptance](release-engineering-plan.md#milestone-1--complete-appliance-candidate).

### 2. Trusted delivery

**Status: partially implemented; commissioning incomplete.** Criteria 8–10.

- Covers artifact/release/channel verification, protected automated signing,
  repository-scoped authority, replay/freshness protection and channel-last publication.
- Source/local native proof and authenticated immutable GHCR round trips passed.
  Public/anonymous delivery, completed promotion commissioning, protected worker
  isolation and recovery custody remain outstanding.
- Exit: intended clients can verify the exact published release without developer
  credentials; interrupted, duplicate, stale and withdrawn offers behave safely.

[Detailed acceptance](release-engineering-plan.md#milestone-2--trusted-delivery).

### 3. Native update and recovery

**Status: pending.** Criteria 11–17.

- Covers the native update caller and exact isolated fixture; actual ISO installation
  and first boot; same-base and new-base upgrades; populated-state preservation;
  interruption, boot failure and compatibility-aware recovery.
- Includes one maintenance/update owner, activation policy, useful operator status
  and resolution of the installed Cockpit update incompatibility.
- Exit: native install/update/recovery evidence for the exact candidate, with later
  writes preserved and no independent updater bypassing Soda qualification.

[Detailed acceptance](release-engineering-plan.md#milestone-3--native-update-and-recovery).

### 4. Automated release builder

**Status: pending; the shared-producer extraction does not satisfy it.** Criteria 18–22.

- Covers the isolated builder/resource contract, CoreOS stable polling and frozen
  candidate admission, the single Go source-to-release run, direct timing/cancellation,
  protected qualification/signing, delivery and normal/emergency serialization.
- Includes deletion of competing producers and a demonstrable reduction in production
  orchestration, followed by authorized timer/one-shot and progressive-promotion setup.
- Exit: the same reviewed command completes the qualified pipeline; commissioned
  normal operation needs no person to build, sign or publish each release. Failed or
  uncertain gates stop downstream effects and notify the owner.

[Detailed acceptance](release-engineering-plan.md#milestone-4--automated-release-builder).

### 5. Production readiness

**Status: pending.** Criteria 23–26.

- Covers native qualification for every advertised architecture/upgrade path,
  rehearsed writable-install migration, and distributable ISO media consuming the
  exact signed host/application candidate used by updates.
- Includes signing rotation/recovery, builder-loss recovery, emergency/withdrawal and
  interrupted-publication drills, retained artifact availability and operations ownership.
- Exit: advertised downloads, supported starting states and operating procedures have
  actual matching evidence. x86_64 may progress independently; full two-architecture
  completion still requires native aarch64 proof. QCOW2 is not an implemented product.

[Detailed acceptance](release-engineering-plan.md#milestone-5--production-readiness).

### 6. Production launch

**Status: pending; no production launch authorized by candidate commissioning.** Criteria 27–28.

- Covers progressive publication/deployment of the first qualified production release
  to exact approved appliances, with observation and maintenance-controlled activation.
- Then demonstrates an actual new CoreOS stable event flowing through approved Soda
  changes, build, native qualification, signing, publication and promotion automatically,
  plus an independently triggered emergency release through the same pipeline.
- Exit: approved appliances consume the tested release according to policy, preservation
  and failure/withdrawal behavior hold, and ongoing release ownership is established.
  A synthetic trigger or GHCR upload alone is not completion.

[Detailed acceptance](release-engineering-plan.md#milestone-6--production-launch).

## Replacement execution order

B1–B6 below are the ordered implementation packages that deliver those milestones,
not another set of milestones or additional approval rounds.

| Package | Status | Next exit |
| --- | --- | --- |
| B1 — Contracts and native-install feasibility | Not started | Verify the selected upstream image-install/offline-content path; record exact dataflow, file-removal/production-LOC baseline and qualification resource requests. |
| B2 — One artifact execution owner | Not started | One Go controller, direct timing/cancellation, programs/assets/images produced once and direct image-owned staging. |
| B3 — Installer consumes the candidate | Not started | Actual ISO installation and first boot of the same host/app digests used by updates; no writable bundle or compilation in media assembly. |
| B4 — Connected native qualification | Not started | Protected evidence from the actual ISO and approved update/recovery baselines, without rebuilding tested artifacts. |
| B5 — Protected finalization and delivery | Not started | Automated candidate signing, final signed release/media binding, verified immutable delivery and channel selection last. |
| B6 — Retire old producers and commission automation | Not started | Competing producers removed, smaller production orchestration, full-run receipt, then separately authorized unattended operation. |

**Next action: B1.** Confirm the native bootc installation/media mechanism and exact
Soda callers before implementing an adapter. Record the deletion/LOC baseline against
`830ca94` and current source. Resolve concrete fixture/effect requirements without
blocking independent approved source work on unrelated architecture or provider gates.

## Existing foundations and evidence

These are reusable inputs, not completion of the replacement:

- **Complete local x86_64 candidate:** `45ac843` produced the host/app payload, 391
  verified immutable Forgejo files, a locked 625-RPM inventory and bootc lint with
  13 passed/one skipped/no warnings. [Receipt](implementation-history.md#complete-local-appliance-candidate).
  This did not prove installation, update, persistence or recovery.
- **Trusted-delivery implementation:** strict release/channel models, native
  Sigstore verification, role-scoped permits and durable publication/high-water
  handling have source/local native proof. [Receipt](implementation-history.md#trusted-delivery-source-and-native-filesystem-proof).
- **Real signing/GHCR bootstrap:** eight immutable packages and signatures were
  staged with authenticated native digest/signature round trips. Last observed
  visibility is **Internal**, not Public; no mutable candidate channel was selected.
  [Receipt](implementation-history.md#ghcr-namespace-and-signing-bootstrap).
- **Transitional build extraction:** focused Go/race/vet, timing, CLI, ISO and staging
  tests passed. Native x86_64 host-context preparation compiled/ELF-checked eight
  vendor programs once with working timings; it did not build images or an ISO.
  [Receipt](implementation-history.md#shared-build-production-and-timing-consolidation).

No complete source → signed candidate → ISO → native install/update/recovery → final
release → publication run has passed. No distributable SodaOS QCOW2 is produced.

## Open qualification and commissioning gates

- Native image-based installation/offline app-content handoff, signature/cache
  enforcement, maintenance-controlled updates, persistence and recovery remain
  unqualified. Exact native fixture/disk/baseline and lifecycle grants are needed.
- GHCR anonymous/public acceptance remains pending the owner's one-time Public
  visibility change for the eight new packages. Observe retained state before any
  further writes. The staged sequence-1 candidate expired at
  `2026-09-14T18:00:29Z`; do not publish it or reset state. Fresh higher-sequence
  admission/permits are required for a new offer.
- Off-machine signing recovery and isolation from untrusted build jobs remain
  unproved. Root-only files plus a sudo-capable builder account are not that isolation.
- GitHub Releases ISO delivery is proposed, not commissioned or authorized by the
  GHCR grant. Preview/stable promotion and unattended worker/timer activation also
  retain their separate commissioning boundaries.
- Native aarch64 still needs its own host-package lock, worker and qualification;
  x86_64 evidence does not cover it. It is not an unrelated prerequisite for B1's
  x86_64 work. Writable-install migration remains separately qualified/authorized.

## Retained release state

Preserve these inputs, failed attempts and later writes; this is last recorded
custody, not a fresh filesystem/registry observation:

| Resource | Custody |
| --- | --- |
| Complete M1 candidate | `.artifacts/host-image/complete-45ac843/` |
| Earlier image attempts and evidence | `.artifacts/host-image/` |
| Delivery source/native proof and bootstrap receipts | `.artifacts/release-delivery/`, including `commission-e8323d5/` |
| Transitional source checks/preparation | `.artifacts/build-consolidation/`, including `prepared-d054a60/` |
| Protected signing/publication authority | `/var/lib/soda-release`: reviewed worker, source identity, encrypted role keys, passphrases/signer files, registry auth, trust, permits, inputs, ledgers, attempts and local encrypted-key backup |
| Public bootstrap trust | `appliance/keys/release-trust.json`; not installed into global host policy |

Do not recreate signing state, regenerate keys, replay bootstrap uploads, prune
artifacts or restore an older ledger/database to repair an observer. The preexisting
GHCR `soda-os` package is unrelated and untouched. No retained appliance has been
migrated to the new release mechanism.

## Current permissions

[AGENTS.md](../AGENTS.md#permissions-and-preservation) owns execution policy. Current
grants belong to the user's task and exact target/action, not this plan's commands.

- **Source/local work:** routine implementation, builds and tests for selected work
  remain authorized within their existing scope. The shared-build/timing extraction
  was explicitly approved; the latest replacement work produced the requested plan
  only. Refocusing this status does not claim implementation or commission effects.
- **Bounded real delivery:** the owner confirmed `LevitateOS`, selected current
  GitHub identity `veighnsche` and approved protected signing setup plus public
  `ghcr.io/levitateos/sodaos-*` namespace/signature commissioning and the **candidate
  channel only**. Preserve the existing protected authority and completed attempts;
  this is not approval to reinitialize them. Routine signing is to be automated,
  not a per-release human ceremony.
- **Not granted by that commissioning or this documentation change:** GitHub
  Release/ISO publication, preview/stable promotion, automatic CI or unattended
  worker/timer installation, global trust/network changes, appliance installation/
  migration, VM/service lifecycle, provider registration/jobs or cleanup. Obtain
  applicable exact target/action authority before these effects.
- Retained development targets have their own [handoff and scoped grants](development-handoff.md#current-permissions).
  None becomes a release qualification fixture because it already exists. Historical
  receipts, time-bounded holds and restricted input files are not renewed permission.

## Latest change

All six release milestones are now explicit here, with scope, current status and
completion summaries linked to their owning acceptance criteria. B1–B6 remain the
replacement implementation order. Unrelated workstream/fixture records stay in their
separate handoff. No production source, build output, key, registry package, VM,
service or appliance state changed.
