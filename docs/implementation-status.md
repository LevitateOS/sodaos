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

**B1 in progress: source/caller review, native CLI/config inspection, handoffs and
removal baseline are recorded. Disk/boot feasibility remains unrun pending the exact
fixture grant below. All six milestones remain open; B2–B6 have not started.**
The `d054a60` shared-command extraction retained two assemblers and grew orchestration;
the owner rejected it as sufficient simplification. Its scoped tests and `fde23d0`
host-context preparation remain evidence, not completion of the replacement.

Current production still splits into writable native bundle → stock CoreOS ISO,
and derived host/app candidate → separate release-delivery tooling. The replacement
must remove this split, not wrap it. The former milestone labels marking the local
candidate complete and delivery next no longer describe the active execution order.

## Milestones

### 1. Verify the native installation contract — B1

**Source/inspection portion complete; native disk proof pending.** Bootc 1.16.7's
`to-filesystem` plus preloaded bound images is selected for testing: its simple
`to-disk` direct layout lacks the separate boot filesystem required by this FCOS
Ignition path. Native CLI/config and existing installer/verification tests passed;
no actual bootc installation or stored-image copy has run. Artifact/schema/security
handoffs and the 2,508 → 2,879 production-orchestration baseline are recorded. [Implementation and exit](release-engineering-plan.md#milestone-1--verify-the-native-installation-contract).

### 2. Implement one Go build controller — B2

**Not started.** One controller owns frozen inputs, direct timing/cancellation,
shipping compilation/assets, prepared tests, app images, direct vendor host assembly
and candidate verification. No second producer or writable staging translation.
[Implementation and exit](release-engineering-plan.md#milestone-2--implement-one-go-build-controller).

### 3. Make the ISO consume the candidate — B3

**Not started.** Media-only assembly takes the exact signed host/app candidate and
prebuilt tools. Prove actual ISO installation, media removal and first boot with the
same digests used by updates; preserve the password-only wizard and disk safeguards.
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

Finish B1 with the bounded native proof below. The [selected install contract](coreos-installer-plan.md#b1-selected-native-mechanism--source-and-cli-proof)
and [handoff/removal baseline](release-engineering-plan.md#b1-artifact-and-authority-handoffs)
are now recorded from `e4f485a` and pinned upstream source. Bootc cached-source import
is not itself signature verification; preverification and protected store ownership
remain mandatory. The old writable installer stays usable until native cutover.

- Actual disk/VM installation, reboot and recovery need an exact native fixture,
  baseline, resource budget and lifecycle grant. Existing fixtures are not implicit
  release fixtures. Complete independent approved source work while such gates wait.
- Signing/qualification must remain protected from arbitrary build code. Use reviewed
  existing tools and isolated fixture trust for local mechanism tests; synthetic/local
  evidence does not grant production authority. Real worker changes need their grant.
- Do not wait for public package visibility, ISO publishing, timer installation,
  automatic stable promotion, native aarch64 or retained-appliance migration to begin
  or complete the independently scoped replacement work.

### Requested native feasibility scope — not yet approved

- One active x86_64 fixture at a time, with up to three fresh attempts named
  `soda-b1-install-e4f485a-{1,2,3}`, rooted exclusively at
  `.artifacts/single-run-b1/e4f485a-KUmwaG/native-vm/attempt-{1,2,3}/` (parent absent).
- Four vCPUs and 16 GiB RAM maximum active; one new 64 GiB disk and private copied
  UEFI variables per attempt (192 GiB maximum logical disk allocation in total);
  `/usr/libexec/qemu-kvm` 10.1.0 and the matching `/usr/share/edk2/ovmf/OVMF_CODE.fd` /
  `OVMF_VARS.fd` inputs, with hashes recorded before launch. This is not Secure Boot
  or physical hardware acceptance.
- Verified selected FCOS live media and a separate read-only candidate-content medium;
  isolated fixture trust/password files, no production keys/auth. No virtual NIC,
  inbound tunnel, bridge/firewall change or real provider operation.
- Boot live media, perform one confirmed install onto that new disk using the selected
  native filesystem path, remove media, boot the installed candidate and reboot once
  to check one-time Ignition/state behavior. Inspect host identity, enforcing SELinux,
  bound-image availability and embedded retained-image import offline. This is engine
  feasibility, not the full B3/B4 user journey or supported upgrade matrix.
- Shut down within four hours total; retain every disk/NVRAM/input/log afterward.
  A failed install stops with its partial disk intact. A corrected attempt may use
  only the next fresh named directory/disk above, never wipe/recreate/replay the old
  one. No cleanup, host installation or contact with retained appliances. Any missing prerequisite needing
  installation on the builder requires its own applicable grant.

Observed builder capacity supports this proposed scope: native x86_64, 16 logical
CPUs, 62 GiB RAM (44 GiB available), 518 GiB home and 40 GiB root free; KVM is readable/
writable. These are observations, not reserved resources or permission to start.

## Reusable foundations — not completed replacement milestones

- B1 evidence: `.artifacts/single-run-b1/e4f485a-KUmwaG/` contains commit-pinned bootc
  source, native public configuration/help, both retained rootless inspection CIDs,
  original failed lookups, artifact sizes, exact LOC inventories and passing focused
  Go tests. [Receipt](implementation-history.md#b1-native-installation-contract-and-removal-baseline).

- `45ac843`: complete local native x86_64 host/app candidate, 391 immutable Forgejo
  files, locked 625-RPM inventory and bootc lint 13 passed/one skipped/no warnings.
  [Receipt](implementation-history.md#complete-local-appliance-candidate). No install/
  update/recovery acceptance is implied.
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

B1 now records the concrete source-backed filesystem-install path, the offline cached-
image trust boundary, minimal metadata handoffs, measured removal baseline and exact
native fixture request. Native CLI/config inspection and focused existing source tests
ran; no new release image/ISO, disk installation, VM, protected-key operation, registry
write or retained-appliance change occurred. Rootless inspection containers and all
attempts/evidence remain retained; B1 is not marked complete.
