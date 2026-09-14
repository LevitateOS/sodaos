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
native builder execution and exact installed proof remain. Bootc filesystem
installation stays withdrawn. The owner now selects a
[minimal network-install ISO](coreos-installer-plan.md#selected-media--minimal-network-install),
not self-contained offline media; native download/bootstrap and size proof are still
outstanding. **B2 is complete at its source-to-candidate scope:** the Go controller
produced and verified native x86_64 candidate `4c62f68` in one run. B3 is in progress: native bootstrap authentication
and executable builder inputs have been reviewed; VM packaging/install proof awaits
execution under its newly approved exact fixture grant. B4–B6 have not started. B1's media/installed proof remains separate from this unsigned candidate.
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
Soda's existing signed-channel client checks. The initial **2,508 → 2,879**
orchestration baseline remains recorded; B2 deltas are reported separately. No
native media/install/update proof ran in the B1 source review. [Implementation and exit](release-engineering-plan.md#milestone-1--verify-the-native-installation-contract).

### 2. Implement one Go build controller — B2

**Complete — native x86_64 P1–P6 run verified.** `tools/soda-build`
owns clean-source admission/archive, frozen app inputs, dependencies, once-only
programs/tools/assets, prepared tests, five app images and FCOS host verification.
Native Go timing and the existing Go process-group owner replace the Python bridge
in this path. Payload v2 embeds all five images for ordinary Podman; no bootc
installation/update/storage/lint path or Zincati disabling remains in new candidates.
The old host-image command is legacy-only, not a competing host producer. Its
required writable installer lane remains for B6 retirement after native proof.
The new CLI does not claim release success: it exits 2 after the unsigned candidate
until B3–B5 connect media, qualification and protected finalization. The observed run
took **00:06:29 with ordinary caches**, with eleven shipping program/tool compilations,
all five embedded archive hashes and native image/Quadlet checks. Native SIGINT at
P3 exited 130, retained diagnostics and emitted no candidate. [Receipt](implementation-history.md#b2-go-controller-and-native-candidate).
[Implementation and exit](release-engineering-plan.md#milestone-2--implement-one-go-build-controller).

### 3. Make the ISO consume the candidate — B3

**In progress — minimal media and authenticated network boot; installer proof open.**
The streaming verifier passed its scoped integrity checks. Two native imports retained
exact candidate bytes and reached upstream helper VMs, but failed before producing
media. The first three packaging targets are consumed; the owner approved three
fresh targets and the clarified upstream scratch lifecycle after reviewing the hold.
Attempt 04 exposed a `/usr/sbin` overlay bug. Replacement candidate `0959f5c` passed
the controller in 5m28s; attempt 05 packaged it successfully. Native extraction yielded
160,432,128-byte minimal media plus the exact 1,884,586,496-byte rootfs. One live boot
and missing/corrupt/truncated download refusals ran in diskless fixture install-01.
The Go controller/installer now implement the embedded prebuilt-console handoff and
native candidate provisioning. The second/final replacement build is next; no disk
has been installed. Previous candidates remain unchanged.
[Receipt](implementation-history.md#b3-native-import-and-stopped-packaging-attempts),
[original approval](#b3-native-fixture-scope--approved) and
[approved narrow extension](#b3-packaging-extension--approved).
Media-only assembly takes the exact signed host/app candidate and
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

The owner selected completion of B2 and confirmed no bootc use. The controller and
ordinary-Podman candidate path are implemented and native-checked. Continue B1/B3
candidate-derived minimal media, keeping installed proof separate. Neither
source checks nor an unsigned candidate authorize appliance installation or delivery.

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

**Native proof prerequisites:** B3 identified and inspected the exact x86_64
Assembler manifest, OSBuild/live stage and tools. Its native streaming verifier
passed scoped tests; the [installer owner](coreos-installer-plan.md#b3-native-download-authentication-handoff)
records that handoff. The approved scope below covers its helper VMs and fresh
installation targets; its first three packaging targets are consumed and the
extension below is now approved. Required proof
includes unchanged OCI input/installed digest, native osmet reconstruction after
download, minimal ISO size, network failures, private Ignition, enforcing SELinux,
media removal and all-five local image availability. B2 implements v2 storage/import;
B3 still must replace the legacy media/console continuation and prove its runtime.

- Actual disk/VM installation, reboot and recovery need an exact native fixture,
  baseline, resource budget and lifecycle grant. Existing fixtures are not implicit
  release fixtures. Complete independent approved source work while such gates wait.
- Signing/qualification must remain protected from arbitrary build code. Use reviewed
  existing tools and isolated fixture trust for local mechanism tests; synthetic/local
  evidence does not grant production authority. Real worker changes need their grant.
- Do not wait for public package visibility, ISO publishing, timer installation,
  automatic stable promotion, native aarch64 or retained-appliance migration to begin
  or complete the independently scoped replacement work.

### B3 native fixture scope — approved

**Owner approved the exact `5efed3e` request.** The following scope now authorizes
B3's named native fixtures and actions. It does not revive the withdrawn bootc
experiment or extend retained targets' grants. Targets 01–03 were used; no
installation target/listener was created. Upstream scratch housekeeping conflicted
with the original no-pruning/preservation restriction. The owner subsequently
approved the [narrow extension](#b3-packaging-extension--approved).

- **Targets:** new directories only under
  `.artifacts/installer-candidate/b3-ea0dc92-OaDOUt/native/`, with
  `package-{01,02,03}` and `install-{01,02,03}` attempts. Container/VM names
  `soda-b3-package-ea0dc92-{01,02,03}` and `soda-b3-install-ea0dc92-{01,02,03}`;
  no retained appliance, existing disk, project or database may be attached.
- **Packaging:** up to three fresh upstream Assembler imports/live packaging runs,
  using x86_64 manifest
  `quay.io/coreos-assembler/coreos-assembler@sha256:f010dce4d350c1588762bbd5b69d27e14dabe859043489c587dc4d429c067daf`
  and FCOS config `682c839aabbc01564f1605bb41687a7511180031`. Native `cosa import
  --skip-prune`/OSBuild only; no bootc, Soda partition manifest or shipping rebuild.
  Upstream's temporary Python buildroot takes Python from that same pinned builder
  via `BUILDER_IMG`, not a mutable image or new RPM resolution.
- **Packaging privileges:** rootless Podman with upstream-required namespace
  `--privileged`, container-only `label=disable`, `/dev/kvm` and `/dev/fuse`. Mount
  only the new work/cache/tmp roots and admitted inputs; do not mount the real
  `/var/tmp`, home, signing custody, container storage or host block disks. No sudo,
  host SELinux/firewall/trust change, registry auth/key exposure or global cleanup.
  Upstream's helper VM uses its native permissive build environment; installed
  candidate acceptance still requires enforcing SELinux.
- **Inputs:** start with B2 candidate `4c62f68` (host manifest
  `sha256:ac3071fcfb95bbb3a28b487d7b6ab74ba4038016a48f709bcf2ebc6850b9cd49`)
  for packaging proof. Permit up to two replacement candidates from committed B3
  source produced by the one Go controller against the unchanged locked x86_64 FCOS
  base. Record exact candidate, tool and media hashes and local fixture-signature
  admission before each VM use; never modify admitted bytes in place. No externally
  supplied candidate or architecture/base substitution falls within this request.
- **Resources:** one active VM at a time, CPU affinity 0–3 / at most four vCPUs,
  up to 16 GiB RAM; up to three new 64 GiB sparse installation disks, fresh copies
  of native OVMF variables, and upstream packaging's fresh 50 GiB cache / 10 GiB
  supermin roots. Stop at four hours of native execution or 200 GiB aggregate new
  allocated disk usage. These are approved experiment bounds, not product budgets.
- **Installation actions:** start fresh UEFI x86_64 guests, enter fixture-only
  root passwords through the virtual keyboard, explicitly confirm erasure of only
  their named blank disks, detach the ISO, reboot and verify first boot, native
  deployment identity, SELinux and all-five local content/app startup. Exercise
  missing/bad/interrupted downloads, cancellation/no write replay and partial-write
  refusal within those attempts. Never re-erase an attempted disk to repair a test.
  This grants no physical USB proof, retained migration, update/recovery baseline or
  real provider registration/job.
- **Network/access:** QEMU user-mode NAT, a loopback-only content fixture on
  `127.0.0.1:19843`, optional loopback VNC/SSH forwards on `19844–19846`, and private
  QMP sockets under each attempt. No bridge/tap/firewall/global CA/DNS changes or
  public hosting. HTTP fixture content is public candidate/tool data authenticated
  by native bootstrap hashes; no passwords, tokens or keys in served roots.
  Fixture-only signing keys/private inputs stay restricted and outside exports.
- **Lifecycle/preservation:** allow start, interruption, media removal, reboot and
  owned-process shutdown of these exact new targets, including upstream helper VMs
  and the fixture listener. Retain failed/successful disks, CIDs, inputs and redacted
  logs. No `--rm`, `--replace`, pruning, reset/recreation or retained-state cleanup.

### B3 packaging extension — approved

**Owner approved continuation after the plain-language scope clarification.**
This adds exactly `native/package-{04,05,06}` under the
same B3 evidence root and containers `soda-b3-package-ea0dc92-{04,05,06}`, allowing
three further fresh imports/live packaging attempts. Do not reuse or restart the
failed workspaces. Installation targets 01–03 and replacement-candidate allowance
remain unchanged; no additional install disk, candidate, base or architecture is
authorized.

- Keep the approved pinned Assembler; use its metadata-only `USER 0` Python-source
  wrapper with identical rootfs layers. Place the admitted archive in the new
  workspace's `/srv` share before helper use.
- Permit **only upstream automatic housekeeping inside each new packaging workspace
  and fresh helper VM**: guest-local empty-cache `podman system prune --all --force
  --filter until=72h`, cache discard/trim, and normal deletion of generated temporary
  helper roots/initrds/build scratch on exit. Record the prelude and results. Preserve
  original inputs, signatures, CIDs, cache disks, failure consoles and produced media;
  installation disks remain fully preserved. This is not permission for manual
  pruning, old-cache reuse, host-store cleanup or retained-target mutation.
- Keep CPU affinity 0–3 (`taskset`, since the user cgroup lacks cpuset delegation),
  four-CPU quota, at most 16 GiB RAM/one active VM and the original **aggregate**
  four-hour/200 GiB bounds. No reset: conservatively charge **1,747 seconds** through
  the hold (including gaps), leaving **12,653 seconds**. The new evidence tree's
  conservative allocated-block count was **35,930,062,848 bytes** at that check.
- All other original network, input, secret, lifecycle and publication restrictions
  remain. No host trust/SELinux/cgroup delegation change is requested.

This recommends retaining upstream scratch ownership rather than maintaining a
Soda fork merely to suppress empty-cache housekeeping. It does not retroactively
authorize the already observed housekeeping or count failed packaging as B3 proof.

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

- `4c62f68`: native x86_64 Go-controller candidate, all five ordinary-Podman archives,
  391 Forgejo files, 625 RPMs and prepared suites checked in the P1–P6 run. Unsigned;
  no media/install/update qualification. [Receipt](implementation-history.md#b2-go-controller-and-native-candidate).
- `45ac843`: complete local native x86_64 host/app candidate, 391 immutable Forgejo
  files, locked 625-RPM inventory and bootc lint 13 passed/one skipped/no warnings.
  [Receipt](implementation-history.md#complete-local-appliance-candidate). No install/
  update/recovery acceptance is implied. Its bootc-bound storage and disabled Zincati
  are historical choices removed from new production, not replacement requirements. No
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
| B3 bootstrap and native packaging failures | `.artifacts/installer-candidate/b3-ea0dc92-OaDOUt/`; `native/package-01` created/not running, 02–03 exited; CIDs, admitted inputs, fixture trust, caches and consoles retained. Upstream removed temporary helper roots; no installation targets created. |
| B2 native candidate, failures and cancellation | `.artifacts/releases/b2-{42cba33,9837d3e,4c62f68,cancel-4c62f68}-*/`; exact paths in the B2 receipt |
| B2 controller binaries/checks | `.artifacts/controllers/b2-*`, `.artifacts/build-controller/controller-3fe7f18-yIS3y0/` |
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
  was explicitly approved; the owner selected B1, B2 and now B3 implementation. B3
  source/local preparation and the original exact VM/disk/listener scope above were
  approved. The owner subsequently approved the [narrow packaging extension](#b3-packaging-extension--approved)
  after clarification; no broader artifact-folder cleanup is authorized.
  B1's source/upstream audit, local tests and bounded rootless read-only image
  inspections are recorded; B2's initial change was direct vendor asset staging. B3's
  additional grant is limited to the named fixtures above; it adds no real protected
  worker, publication or commissioning authority.
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

Approved B3 native work admitted the unchanged B2 candidate with fixture-only native
signatures and proved exact OCI import twice. Two helper VMs ran; live packaging
failed on the Python source's user, then its VM-invisible archive path. The cgroup
startup failure, failed signature-document fixture and successful corrections remain
in the [receipt](implementation-history.md#b3-native-import-and-stopped-packaging-attempts).
Upstream ran an unanticipated empty-cache prune (0 bytes reclaimed) and removed
helper scratch, contrary to the original restriction. After clarification, the owner
approved the [scope correction and new targets](#b3-packaging-extension--approved);
execution resumes without resetting the original aggregate resource limits.
Attempt 04 then reached native disk assembly but failed bootloader installation:
Soda's context replaced `/usr/sbin → bin`, hiding the existing GRUB tool. The focused
Go-tested source fix stages wrappers in `/usr/bin` and adds a native layout gate
(**12,238 production lines**, +3). That replacement (`0959f5c`) passed its full P1–P6
run and native packaging/minimal network boot. The subsequent Go handoff adds 321
production lines (**12,559 total**): embedded once-built console, pinned Butane public
Ignition output, strict candidate/archive verification and native provisioning using
the unchanged disk safeguards. Source/default/vendor/race checks and strict profile
conversion passed. [Current receipt](implementation-history.md#b3-minimal-media-network-boot-and-candidate-console).
No installation/media-removal/installed-first-boot proof, real signing-custody change,
publication or retained-appliance mutation has occurred. B3 remains incomplete.
