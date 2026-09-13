# Implementation status — single-run replacement

**Current priority: replace the two build lanes first.** One Go command must build
and qualify one immutable host/application candidate used by both installation and
updates, then the old producers must be removed. Unattended scheduling, production
readiness and launch follow that replacement; they are not prerequisites for it.

The [release engineering plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns these six milestones and their detailed contracts/tests. B1–B6 identify the
milestones themselves, not a subordinate checklist beneath the old release roadmap.
Other workstream/fixture records stay in the [development handoff](development-handoff.md),
not this implementation queue.

## Current position

**B1 source review identifies a native FCOS packaging/update route; not yet native-
proved.** Assembler/OSBuild can consume the candidate for live/osmet packaging, and
Zincati 0.0.32 supports OCI through rpm-ostree. A client-trust contract decision,
matching builder capsule admission and exact native proof remain. Bootc filesystem
installation stays withdrawn. The owner now selects a
[minimal network-install ISO](coreos-installer-plan.md#selected-media--minimal-network-install),
not self-contained offline media; native download/bootstrap and size proof are still
outstanding. All six milestones are open; B2–B6 have not started.
The `d054a60` shared-command extraction retained two assemblers and grew orchestration;
the owner rejected it as sufficient simplification. Its scoped tests and `fde23d0`
host-context preparation remain evidence, not completion of the replacement.

Current production still splits into writable native bundle → stock CoreOS ISO,
and derived host/app candidate → separate release-delivery tooling. The replacement
must remove this split, not wrap it. The former milestone labels marking the local
candidate complete and delivery next no longer describe the active execution order.

## Milestones

### 1. Verify the native installation contract — B1

**Source-backed route found; native proof and update-authority choice outstanding.**
Recommend the locked FCOS producer's OCI import → native metal/live/osmet path,
unchanged CoreOS Installer/Ignition, and Zincati/rpm-ostree OCI updates. The stock
Fedora graph does not qualify Soda images; native graph trust is not equivalent to
Soda's existing signed-channel client checks. The **2,508 → 2,879** orchestration
baseline remains unchanged. No native media/install/update proof ran in this pass. [Implementation and exit](release-engineering-plan.md#milestone-1--verify-the-native-installation-contract).

### 2. Implement one Go build controller — B2

**Not started.** One controller owns frozen inputs, direct timing/cancellation,
shipping compilation/assets, prepared tests, app images, direct vendor host assembly
and candidate verification. No second producer or writable staging translation.
[Implementation and exit](release-engineering-plan.md#milestone-2--implement-one-go-build-controller).

### 3. Make the ISO consume the candidate — B3

**Not started.** Media-only assembly takes the exact signed host/app candidate and
prebuilt tools. Prove minimal ISO size, authenticated network installation, media
removal and first boot with the same digests used by updates; preserve the
password-only wizard and disk safeguards.
[Implementation and exit](release-engineering-plan.md#milestone-3--make-the-iso-consume-the-candidate).

### 4. Connect native qualification — B4

**Not started.** Reviewed drivers test the actual ISO and admitted update/recovery
baselines, populated-state preservation, signatures/cache and maintenance ownership.
Evidence binds exact bytes; required tests cannot be skipped or rebuild the candidate.
[Implementation and exit](release-engineering-plan.md#milestone-4--connect-native-qualification).

### 5. Integrate protected signing and delivery — B5

**Not started.** Connect existing native signing/verification and publisher interfaces
to the run, with separate protected authority, final ISO/evidence binding and channel
last. Native local signing and failure tests establish integration; public GHCR/ISO
commissioning is not a gate before deleting the old builders.
[Implementation and exit](release-engineering-plan.md#milestone-5--integrate-protected-signing-and-delivery).

### 6. Retire old lanes and prove the replacement — B6

**Not started.** After native installer/qualification proof, remove the competing
producers, adapters and timing bridge; update every build/media/check caller. Prove
one complete native local run through signed final metadata, unchanged tested bytes
and smaller production orchestration. No timer or production launch is part of this
milestone. [Implementation and exit](release-engineering-plan.md#milestone-6--retire-old-lanes-and-prove-the-replacement).

## Immediate prerequisites and next action

The [installation findings](coreos-installer-plan.md#source-backed-packaging-route--native-proof-outstanding)
and [native update/authority findings](release-engineering-plan.md#b1-native-update-and-authority-findings)
now identify concrete upstream calls and boundaries, not a custom disk/updater design.

**Selected media:** minimize ISO size and download installation content using native
FCOS where possible. Apply the [owning media contract](coreos-installer-plan.md#selected-media--minimal-network-install);
do not retain the full/offline ISO requirement or assume GHCR must serve every file.
B1 must verify native minimal extraction, pre-live networking and authenticated
rootfs/bootstrap binding before implementing that handoff.

**Decision needed:** adopt native Zincati graph/image trust and maintenance as the
appliance update contract, or retain all existing client-side signed-channel checks.
Recommend the native model for minimum FCOS deviation, with protected qualification/
publication and signed final release evidence. This changes role-scoped channel
admission, expiry/high-water and withdrawal semantics; no equivalence or permission
to weaken the existing contract is assumed. Do not implement a second client updater
or publish an unsigned graph as a silent replacement for those checks.

**Native proof prerequisites:** admit an exact Assembler container digest with the
matching OSBuild/live-stage/tool versions; the reviewed source commit alone is not
that executable pin. Then scope its supermin build VM and a fresh x86_64 install
fixture. Required proof includes unchanged OCI input/installed digest, native osmet
reconstruction after download, minimal ISO size and network-failure behavior, private
Ignition/SELinux/boot, media removal and local availability of all five application images. The existing candidate needs versioned storage/
import changes; it is not already a suitable complete fixture. No new VM request or
lifecycle approval is inferred from the withdrawn experiment.

- Actual disk/VM installation, reboot and recovery need an exact native fixture,
  baseline, resource budget and lifecycle grant. Existing fixtures are not implicit
  release fixtures. Complete independent approved source work while such gates wait.
- Signing/qualification must remain protected from arbitrary build code. Use reviewed
  existing tools and isolated fixture trust for local mechanism tests; synthetic/local
  evidence does not grant production authority. Real worker changes need their grant.
- Do not wait for public package visibility, ISO publishing, timer installation,
  automatic stable promotion, native aarch64 or retained-appliance migration to begin
  or complete the independently scoped replacement work.

### Withdrawn native experiment

The `6a360d3` proposal for bootc filesystem installation with Soda-owned partitioning
and up to three VM/disk attempts is withdrawn, not awaiting approval. No proposed
VM/disk was created or installation performed. The original request remains in Git
and the [historical receipt](implementation-history.md#b1-native-installation-contract-and-removal-baseline),
not as a current grant. A new fixture request must follow the corrected mechanism
review and describe only the native path it actually needs to prove.

## Reusable foundations — not completed replacement milestones

- Reopened B1 source evidence: `.artifacts/single-run-b1/fcos-bf4b4fa-SbnzJa/` contains
  39 pinned upstream files, resolved commits, public source hashes and scoped contract
  checks. [Receipt](implementation-history.md#b1-fcos-native-handoff-source-findings).
  No Assembler image, media build, native installation or update was executed.
- Earlier B1 evidence: `.artifacts/single-run-b1/e4f485a-KUmwaG/` contains commit-pinned bootc
  source, native public configuration/help, both retained rootless inspection CIDs,
  original failed lookups, artifact sizes, exact LOC inventories and passing focused
  Go tests. [Receipt](implementation-history.md#b1-native-installation-contract-and-removal-baseline).

- `45ac843`: complete local native x86_64 host/app candidate, 391 immutable Forgejo
  files, locked 625-RPM inventory and bootc lint 13 passed/one skipped/no warnings.
  [Receipt](implementation-history.md#complete-local-appliance-candidate). No install/
  update/recovery acceptance is implied. Its bootc-bound storage and disabled Zincati
  are experimental source choices to revisit, not replacement requirements. No
  runtime policy is changed or existing artifact relabelled by this correction.
- Existing trusted-delivery models, native Sigstore, exact permits and durable
  publication/high-water handling have source/local native proof.
  [Receipt](implementation-history.md#trusted-delivery-source-and-native-filesystem-proof).
- Real protected keys/eight immutable GHCR packages have authenticated native
  signature/digest round trips; visibility was last observed Internal and no mutable
  candidate channel was selected. [Receipt](implementation-history.md#ghcr-namespace-and-signing-bootstrap).
- Transitional build tests and native x86_64 host-context preparation compiled/
  ELF-checked eight programs once with working timings, not images or an ISO.
  [Receipt](implementation-history.md#shared-build-production-and-timing-consolidation).

## After B6 — separate commissioning

The [downstream operations plan](release-engineering-plan.md#after-the-replacement-operational-commissioning)
then resumes public GHCR/ISO delivery, isolated unattended scheduling, supported
architecture/migration readiness, operational recovery drills and progressive launch.
None is marked complete by replacement source or local qualification.

Pending custody facts: package Public visibility/anonymous proof remain outstanding;
the sequence-1 candidate expired at `2026-09-14T18:00:29Z` and requires fresh higher
sequence/permits, never reset state. Off-machine recovery and untrusted-job isolation
remain unproved. Native aarch64 needs its own lock/worker/proof; GitHub Releases ISO
publication and retained-install migration are separately authorized work.

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
  was explicitly approved; the owner has now selected B1. Its source/upstream audit,
  local tests and bounded rootless read-only image inspections are recorded. This
  does not add a VM/disk, protected worker, publication or commissioning grant.
  The current correction restores the FCOS-native baseline and withdraws the bootc
  filesystem experiment; it does not authorize another installation path.
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

Recorded the owner's minimal network-install selection in the installer contract;
removed the active self-contained/offline ISO requirement and linked the release/
installation guides to the new size/download target. Preserved exact candidate
identity, local content after installation, authentication, native FCOS ownership
and disk safeguards. The prior source receipt remains in history; this pass changed
documentation only and checked affected links/contracts. No native execution,
publication, trust change or new lifecycle grant occurred. B1 remains open.
