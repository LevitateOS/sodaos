# Implementation status — single-run replacement

**Active priority: resume and complete M4.** The owner explicitly requested resumption
of the remaining production qualification work after the completed
[fast-development side path](fast-development-build-plan.md). M4 is not complete. It owns F1–F3; the original
B1–B6 milestones below are retained, not renumbered or replaced. The
[release plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns production contracts; [history](implementation-history.md) owns detailed
receipts and superseded grants. Other workstreams remain in the [development handoff](development-handoff.md).

## Current position

- **Fast-development F1/F2 delivered:** `bb17280` adds explicit development
  candidate/media targets through the existing isolated producer. Native candidate
  runs passed in **8m29s cold / 5m30s warm**, with all five apps/checks and no media
  authority, packaging or VM. Focused race/vet checks passed. F3 is also complete (`0fd8def`):
  default/fast media took **21m24s / 18m35s**, with fast packaging saving **2m56s**
  for a **2.85% larger rootfs**. Native readback and a diskless fast-media welcome boot
  passed; task workers, VM and listener stopped. Production settings remain unchanged. [F1–F3 status](fast-development-build-plan.md#implementation-order) remains
  separate from B1–B6. M4 fixtures and real release custody remain untouched.
- B2's source-to-candidate controller was natively proved. Earlier B3 fixtures proved
  candidate-derived installation, media removal, exact-candidate first boot, five local
  application images, SELinux and interruption/cancellation boundaries.
- `425b863` removes obsolete payload readers/storage selectors and their compatibility
  tests: **334 net lines removed** across code/tests/docs. There is one current shared
  OCI layout; the format marker rejects old input rather than selecting old readers.
  Across the pre-B4 source work, production is **+437 lines**, tests **−219**;
  necessary media integration adds code. The fixed production inventory is now
  **13,255 lines**. This does not satisfy B6's eventual orchestration reduction.
- `165064c` connects media assembly/readback to the Go controller. Subsequent fixes
  use the exact cached Assembler digest through native pull policy, the correct OCI
  document signing transport, and remove the obsolete archive-only host preflight.
- **Pre-B4 implementation and native installation verified; ready to plan B4 execution.** Production-controller
  candidate/media source `33ea3f5` installed and booted without media or a download
  listener. All five exact Podman images, all 49 embedded file hashes, enforcing
  SELinux and SSH restrictions passed. Evidence: `.artifacts/b3-completion/` and
  [the receipt](implementation-history.md#b3-production-media-and-current-layout-installation).
- The controller run took **21m34s and exited 1** at a false Ignition readback refusal:
  native serialization adds optional `null` fields. `5c237e4` corrects that observer;
  its race tests and independent readback passed against the **unchanged** media.
  No candidate, packaging or disk mutation was replayed to repair observation. This
  is not a claim of a fresh all-green CLI run or B6's qualified-release proof.
- Native rpm-ostree/Zincati trust and maintenance are selected under the owner's
  standing approval. The [owning contract](release-engineering-plan.md#b1-native-update-and-authority-findings)
  explicitly describes the removed custom client-channel semantics.

## Milestones

### 1. Verify the native installation contract — B1

Installation handoff is established: unchanged host OCI → Assembler/OSBuild →
CoreOS Installer/Ignition. Bootc installation is not selected. Local application
content is part of the candidate. Native update trust/maintenance is selected;
actual update and recovery qualification belongs to B4, not another B1 restart.

### 2. Implement one Go build controller — B2

Source-to-candidate execution is implemented and tested: clean source/input freeze,
once-only programs/assets/images, direct staging, verification, timing and cancellation.
The media extension uses the same controller and unchanged candidate; no second
shipping-program build or competing new producer is introduced.

### 3. Make the ISO consume the candidate — B3

**Production path implemented and native installation verified.**
P7 uses existing local Sigstore primitives for candidate/input admission. P8 invokes
pinned Assembler/OSBuild in fresh disposable scratch, extracts clean minimal media,
applies URL/live Ignition once and reads back Ignition, kernel arguments and the
native rootfs chunk binding from the actual ISO. `artifacts/media/media.json` binds
candidate/tool identities, ISO/rootfs sizes/hashes and the exact rootfs URL.

The once-compiled console and all five application images remain in the candidate.
CoreOS Installer/Ignition own disk installation and provisioning; the existing
password-only interaction, disk identity/revalidation and no-replay safeguards remain.
The Assembler import checksum is labelled separately from the actual booted OSTree
commit. Neither is substituted for the OCI image digest.

The controller-produced media passed independent fixture signature admission,
installation to its blank 64 GiB disk, actual ISO removal and no-media/no-listener
first boot. Host OCI identity and the actual OSBuild/booted deployment agree; image
import and expected service/security states passed. Earlier unchanged cancellation,
interruption and corrupt-download evidence remains scoped to its original tests.

Measured ISO: **160,432,128 bytes**; rootfs/download body: **1,802,472,960 bytes**,
**88,116,224 bytes smaller** than the earlier tested rootfs. This is an actual
rebuilt-media comparison, not a prediction from archive sizes. The receipt records
RAM and installed-disk observations separately; it does not claim minimum hardware.

Local fixture signing demonstrates the mechanism, **not adversarial untrusted-job
isolation or final release authority**. Real credentials are not used. The CLI retains
exit 2 after media because qualification/final protected release evidence are absent.
A fixture download URL is not a distribution-ready public installer.

### 4. Connect native qualification — B4

**Resumed under the owner's explicit “do it” instruction.** The prior M4 approval
and completed groundwork remain in force; M4 is not complete. The
[owning B4 contract](release-engineering-plan.md#milestone-4--connect-native-qualification)
selects P9 installation, same-base x86_64 A → B update and compatible native recovery.
A is the verified current-layout `33ea3f5` artifact; B is the next necessary clean
controller build. The existing stopped disk may seed a fresh isolated test clone only
after receipt/identity verification and execution approval.

Required evidence covers native signatures, HTTPS graph offers, maintenance,
staged offline content, populated-state preservation including later writes, and
independent byte/evidence custody. Base changes, schema migrations and automatic
boot-failure recovery are not claimed by that first scenario. Local fixture signing
alone cannot satisfy protected qualification. B5 retains final release authority.
The next B build must exercise corrected media readback and result reporting; no
standalone B3 replay is required solely to replace the recorded nonzero exit.

Execution groundwork: the stopped standalone A disk and its current-layout native
receipt were checked without mutation; disk/receipt hashes are recorded under
`.artifacts/b4-qualification/admission/baseline.json`. The two separate non-login
worker accounts now exist, without sudo/service grants. Native service checks passed
for exact non-root UIDs, read-only input/write refusal, builder refusal to read the
qualifier's private custody marker, and cancellation of the exact running service.
The dispatcher uses systemd's service/cgroup boundary; anonymous output pipes retain
log-file ownership in the controller. An isolated build-worker preflight passed
pinned Go/Bun, the canonical source view and rootless Podman checks.

Quay no longer serves the former Assembler manifest, and native Skopeo refused a
preserving transfer from the administrator's cached store. The selected available
replacement in `media-tools.json` was pulled under the isolated build identity: it
reports the **same** Assembler source revision and Installer 0.26.0. This is a required
input-availability correction, not an incidental source/tool-version upgrade.
The build CLI now dispatches through an admitted root-owned worker configuration;
archive admission and disk-VM/console support are being connected to P9 using existing
acceptance/delivery primitives. A verified independent clone has now booted under
`soda-qualifier`: reviewed console prompt matching gated password entry; fixture-only
SSH enrollment completed; native rpm-ostree observations matched A's exact host
manifest and deployed commit. The VM/service stopped successfully and the original
seed hash remained unchanged. The first QEMU launch refused a misplaced device
property before boot (clone hash unchanged); a later public-host-key observation was
corrected without replaying enrollment or regenerating keys. Original failures and
protected observations remain retained.

A second fresh clone now has native local Forgejo/Soda setup, synthetic operator and
repository, an A-generation Git commit, an actual native Soda project and project
filesystem data. Its protected snapshot binds schema 10, the exact project container,
creation profile, repository commit, settings, machine identity and public-key hashes.
Project creation exposed an actual image-lane omission: the `containers` subordinate
ID pool existed only in the old installer script. The immutable host recipe and
read-only producer check now declare it; the prepared A fixture received the exact
non-overlapping operator correction. Its successful native project creation followed
that correction, without replaying account/repository/OAuth setup or restoring data.

The resumed implementation now connects a fixed protected P9 driver to production
controller dispatch: independent root-held artifact snapshots, a verified disposable
copy of populated A, native B installation, upstream registry/Sigstore fixtures,
Zincati maintenance/offline activation, and native rollback/state comparisons.
Source race tests and vet pass for the affected qualification/controller packages;
these are authored checks, not native scenario evidence. Development remains separate,
and production still cannot report release success before B5.

**No B build, update, recovery or complete P9 qualification has run yet.** Native
groundwork receipts remain under `.artifacts/b4-qualification/`; `project-map.log`
and protected `workers/soda-qualifier/baseline-02/evidence-map01/` record populated A.
Both task clones and all tested worker VMs are stopped. The original seed remains
unchanged. The image correction is source-checked, not yet built/installed in B.
Upstream `fcos.upgrade.basic` remains unsuitable:
it uses unverified rebase and synthesizes a different commit.

### 5. Integrate protected signing and delivery — B5

**Not started.** Connect protected authority to final candidate/ISO/evidence bindings
and existing delivery primitives. Test failure handling and channel-last publication
interfaces locally. B3 fixture signatures do not count as this completion.

### 6. Retire old lanes and prove the replacement — B6

**Not started.** Delete competing producers and obsolete adapters, rewire callers,
and demonstrate a complete native qualified-but-unpublished run with intact
verification and smaller production orchestration. Remove obsolete code when its
actual caller is replaced; do not invent compatibility to keep experiments usable.

## Immediate prerequisites and next action

Complete protected P9 integration and the production-default B build/install, signed
native update/refusals, maintenance/offline activation and state-preserving rollback.
Reuse the populated stopped `baseline-02` fixture without reseeding; preserve failed
targets and original seed. The completed fast-development side path is not production
qualification. Public hosting, ARM and minimum-hardware qualification remain separate.

The public rootfs base URL is an explicit media input. GitHub Release assets can serve
hash-named ISO/rootfs files; this local work neither publishes them nor requires an
operated download service. Installed offline content means the required content is
local after the authenticated network installation, not that the minimal ISO is a
complete offline installer.

## Current permissions

**Current task: implement `docs/fast-development-build-plan.md`.** The owner's latest
request authorizes its source work and bounded local verification, superseding its
planning-only status. F1–F3 execution is complete; no benchmark/VM is left running.
The owner's subsequent “allowed” approved F3 with distinct development-only compression
metadata; the previous
unchanged-candidate comparison constraint is superseded for this benchmark only.
The source-read correction granted the existing build identity read-only ACLs on
13 current-commit loose Git objects, not private/untracked inputs; exact paths remain
in `.artifacts/fast-development/source-read-access.txt`. Scope: reuse the existing isolated build identity/tools/caches;
install a separately named admitted development controller/test helper under
`/usr/local/libexec/soda-qualification/`; write task receipts/configuration under
`.artifacts/fast-development/` and fresh candidate/media outputs beneath the existing
`.artifacts/releases/isolated/` parent. Record exact helper hashes/run names before
effects. Do not replace the M4 controller/configuration or touch its qualifier disks.

Run the necessary native candidate timing and at most one successful baseline/fast
packaging comparison using default/fast development candidates from the same source,
retaining each candidate's unchanged bytes throughout its packaging/boot checks and
retaining failures. Use
fixture-only media signing and a task-local rootfs listener/diskless boot observation
if needed to validate the selected compression; no installation or real provider
activity. Preserve one active VM, CPUs 0–3, at most four vCPUs/16 GiB. Stop only exact
task helpers/containers/listeners; no pruning, retained-state deletion, host trust or
network policy changes, real release authority, publication or increased resources.

**Current B4 approval — execution resumed:** the owner explicitly requested resuming
M4 and then instructed “do it.” The earlier “Now do M4 completely” scope below applies.
Read-only inspection confirms the populated `baseline-02` disk is standalone, not
dirty/corrupt, with its private SSH/key/NVRAM inputs present; no QEMU was running.
No reseeding, baseline reset, production key use or broader effect is added.

**Retained B4 execution scope:**

- Fresh task-owned outputs/targets under `.artifacts/b4-qualification/`, using a
  verified copy of the stopped B3 disk, not mutating the retained seed or adopting an
  unrelated VM. Record exact run paths before effects. Carry forward the
  one-active-VM, affinity 0–3, four-vCPU/16-GiB ceiling; measure storage rather than
  reserve a preservation-driven expansion.
- Native build/package and disposable VM/disk lifecycle for the selected P9 scenario,
  including installation, reboot, synthetic data writes, controlled content-download
  interruption and native rollback. No real provider registration/jobs or restoration
  of an older database over later writes.
- Fixture-only signing authority, local OCI content and HTTPS graph serving, with
  endpoint routing/CA/policy changes confined to the disposable guests. No public
  publication, host-wide DNS/firewall/trust changes or use of real release credentials.
- Use separate `soda-build-worker` and `soda-qualifier` identities and protected
  evidence custody. Check for conflicting existing accounts before creating them;
  record exact worker/run paths and privileges before effects. Same-user fixture signatures
  are not proof of that boundary. Use existing protected primitives; do not reset
  `/var/lib/soda-release` to make a local test work.
- Specify permitted task-container/helper cleanup and exact disposable resources.
  No unrelated cleanup or deletion of retained baseline artifacts is authorized.

Existing real delivery authority remains candidate-channel-only within its original
commissioning scope; no bootstrap replay, key/ledger reset or preview/stable publishing
is added. Historical B3 grants and execution receipts remain in history.

## Retained release state

Real protected custody remains `/var/lib/soda-release` (worker, encrypted keys,
registry authentication, trust, permits, ledgers and backup). Public bootstrap trust
is `appliance/keys/release-trust.json`. Do not reset/replay either. Unrelated development
state and grants remain in the [development handoff](development-handoff.md#current-permissions).

Earlier B2/B3 experiments and receipts remain where recorded in history; they do not
require current compatibility code. The completed current-layout disk is stopped at
`.artifacts/b3-completion/install-01/private/disk.qcow2`; its server is stopped too.
The original copied ISO at
`/home/libvirt/images/sodaos-9577645-x86_64-network.iso` is unchanged and points to the
stopped old fixture URL. It is not the new production-controller output.

## After B6 — separate commissioning

Public ISO/GHCR delivery, unattended scheduling, real trust installation, additional
architectures, migration and launch remain downstream work. Package visibility,
anonymous delivery, off-machine recovery and untrusted-job isolation are not proved
by current local fixtures. Historical channel sequences/permits must not be replayed
or reset; future authorized publication needs fresh applicable authority.
