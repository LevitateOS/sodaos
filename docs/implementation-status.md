# Implementation status — single-run replacement

**Active priority: finish B4.** The owner explicitly resumed remaining production
qualification after the completed
[fast-development side path](fast-development-build-plan.md). B4 is not complete.
F1–F3 are done separately; the original B1–B6 milestones below are retained, not
renumbered or replaced. The
[release plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns production contracts; [history](implementation-history.md) owns detailed
receipts and superseded grants. Other workstreams remain in the [development handoff](development-handoff.md).

## Current position

- **Integrated P9 development driver passed:** `development-driver-02` exercised the
  reconnected fixed scenario against retained `baseline-02` and unchanged
  `b4-production-02` candidate/media (**~9m20s**, exit 0). Evidence covers B ISO
  install/media-free boot, wrong-key and required-content refusals with A unchanged,
  Zincati maintenance/offline B activation, later-write rollback to A, and both state
  generations. Baseline disk hash unchanged; VMs/listeners stopped. This is
  non-qualifying; no production `qualified.json`. Cancelled `development-driver-01`
  remains retained (SIGINT after Forgejo `/api/v1/version` 404 on unlocked install).
- **Fast-development F1/F2 delivered:** `bb17280` adds explicit development
  candidate/media targets through the existing isolated producer. Native candidate
  runs passed in **8m29s cold / 5m30s warm**, with all five apps/checks and no media
  authority, packaging or VM. Focused race/vet checks passed. F3 is also complete (`0fd8def`):
  default/fast media took **21m24s / 18m35s**, with fast packaging saving **2m56s**
  for a **2.85% larger rootfs**. Native readback and a diskless fast-media welcome boot
  passed; task workers, VM and listener stopped. Production settings remain unchanged. [F1–F3 status](fast-development-build-plan.md#implementation-order) remains
  separate from B1–B6. B4 fixtures and real release custody remain untouched.
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

**Resumed; integrated development driver passed; protected production receipt still
required.** The prior approval and completed groundwork remain in force. The
[owning B4 contract](release-engineering-plan.md#milestone-4--connect-native-qualification)
selects P9 installation, same-base x86_64 A → B update and compatible native recovery.
A is the verified current-layout `33ea3f5` artifact; B for the final production run is
the next necessary clean committed controller build. Development used retained
`b4-production-02` bytes only as a non-qualifying check.

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

**M4 remains incomplete; premature orchestration removed at the owner's request.**
Removed the end-to-end driver, registry/graph/fault servers, synthetic receipt gate,
root snapshot dispatcher and production CLI coupling. Artifact admission, shared
acceptance isolation/VM primitives and fixed guest-state/content checks remain.
The resumed implementation now reconnects the fixed scenario after native prerequisite
checks, with the demonstrated trust path, native lock/API observations, one unchanged
maintenance window and real profile writes. Integrated retained-artifact development
validation precedes a fresh production run. B5 remains disconnected.

Two retained production attempts completed P1–P8 in 21m41s and 22m03s before P9
failed (`production-03.log` and `production-04.log` under
`.artifacts/b4-qualification/`). The latter verified B installation/media-free boot
and populated A, but did not qualify update/recovery. Registry networking required
a loopback-only correction. Native inspection found `/run/containers` mode 0700
blocking the unprivileged fetch user; the tmpfiles correction is committed in `39871b5`.
FCOS also binds a separate policy into rpm-ostreed: changing ordinary Podman policy
did not configure native update trust. A diagnostic copy unexpectedly staged
wrong-signed B under that stock policy and remains stopped as failure evidence.
The original corrected policy probe rejected missing signatures only. On resumption,
`attachment-probe-02` enabled native Sigstore attachments for the physical registry
endpoint and obtained an actual cryptographic wrong-key refusal (exit 0). This is
non-qualifying development evidence; its VM and registry stopped successfully.

All artifacts, diagnostic copies and installed historical helpers are retained,
not adopted as a resume path or current controller. The original seed and populated
`baseline-02` remain untouched by these attempts. Development probe 02 verified signed required-content refusal with A unchanged,
then Zincati selected and staged exact B through the local HTTPS graph. The observer
incorrectly expected a JSON finalization-lock field absent from this native version;
`ostree admin status` independently showed the exact B commit as finalization locked,
and Zincati reported reboot pending on its periodic strategy. The guest was powered
off cleanly; the failed observer receipt remains. Probe 05 subsequently observed exact
B activation with the registry stopped and no payload requests. Its first state read
failed because Forgejo's API was not available despite the unit being active; complete
application readiness/state preservation and later-write rollback were then proved
in probes 06–07. Probe 07 exited 0 after native rollback to exact A, preserving both
Git/project generations and the actual B profile write. These are development
receipts, not a complete protected P9 production receipt.
Upstream `fcos.upgrade.basic` remains unsuitable: it uses unverified rebase and
synthesizes a different commit.

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

The owner resumed B4 to completion. Native policy/attachment, update and recovery
prerequisites are demonstrated in retained non-qualifying probes. The integrated
driver is reconnected; `development-driver-01` installed B and reached media-free
first boot, then was SIGINT'd while polling unconfigured Forgejo `/api/v1/version`
(404 until install lock). Readiness now requires that API only after fixture setup.
Next: `development-driver-02` against retained artifacts, then one fresh protected
production P9 run. Development evidence is not release qualification. Public hosting,
ARM and minimum-hardware work remain separate. The required final qualification scope
is unchanged in the
[B4 contract](release-engineering-plan.md#milestone-4--connect-native-qualification).
Preserve failed targets and the original seed.

The public rootfs base URL is an explicit media input. GitHub Release assets can serve
hash-named ISO/rootfs files; this local work neither publishes them nor requires an
operated download service. Installed offline content means the required content is
local after the authenticated network installation, not that the minimal ISO is a
complete offline installer.

## Current permissions

**Current task: finish B4.** The owner explicitly resumed implementation and bounded
native execution under the B4 scope below. Development checks come first; do not
repeat production builds to debug native boundaries. No artifact deletion or broader
provider, publication, trust or resource grant is added.

Active development target completed: `development-driver-02` passed. Next production
target: fresh `b4-production-05` under `.artifacts/releases/isolated/` with admitted
controller/helpers, new `p9-production-05` work root, `production-worker-04.json` and
`production-p9-04.json` (includes BaselinePassword). Use stopped `baseline-02` only as
a verified copy source. No resume of failed production-01..04. Exact disposable
registry/listeners/VMs stop on exit; retain all artifacts.

Completed earlier probe targets remain retained for evidence. First development
target was `workers/soda-qualifier/attachment-probe-02` beneath
`.artifacts/b4-qualification/`, with a fresh copy of the stopped `p9-production-01`
diagnostic disk/NVRAM. Reuse the exact stopped `p9-production-02` fixture registry
and TLS inputs; stop that registry and the new VM/listener after the probe. Admit a
separately named `probe-native-attachments-02` helper under the existing protected
helper directory, recording its hash before execution. This probe is non-qualifying;
original disks, baseline, keys, historical helpers and failed receipts remain retained.

Next development target: `workers/soda-qualifier/update-probe-01`, copied from the
stopped attachment probe. The separately admitted `probe-native-update-01` helper
may publish a correct fixture signature into the same local registry, temporarily
withhold one required blob, serve a local HTTPS graph on port 19444 and exercise
Zincati staging/maintenance/offline activation on this copy only. Record helper and
disk hashes before execution. No new production build; recovery follows demonstrated
update assumptions. Probe 01 stopped at a missing guest-state helper, before signing
or update dispatch. Probe 02 uses a fresh copy under `update-probe-02`, admits the
current guest-state CLI by hash through private SSH, and repeats the same bounded
check. Its host helpers are separately named `guest-state-39871b5` and
`probe-native-update-02`. All listeners, this VM and the exact registry stop on exit.

Development recovery target: `workers/soda-qualifier/recovery-probe-03`, a fresh copy
of stopped probe 02's staged disk/NVRAM. The separately admitted
`probe-native-recovery-03` uses native `ostree admin status` for the lock observation,
then checks offline maintenance activation and compatible later-write rollback.
Probe 03 found that the locked staged deployment did not survive the diagnostic
shutdown, and stopped before update dispatch. Probe 04 uses a fresh copy under
`recovery-probe-04`, with admitted `probe-native-recovery-04`, and lets Zincati stage
again from the retained candidate/cache before testing activation in the same boot.
Probe 04 showed that restarting Zincati after withholding content causes it to
restage and fail fetching registry metadata. The guest was powered off, then its
exact worker stopped. Probe 05 uses a fresh copy under `recovery-probe-05` and
admitted `probe-native-recovery-05`: select one maintenance window before staging,
then keep that agent running while the registry is stopped. These are non-qualifying
development attempts, not qualified-release resumes.
Probe 06 uses a fresh copy of probe 05's stopped B disk under `recovery-probe-06`,
with admitted `probe-native-recovery-06`. Keep the exact registry stopped, wait for
Forgejo's actual version endpoint as well as native units, compare the retained A
snapshot, then exercise later-write recovery. It must not repeat a mutation to repair
an observer. A/B store, web/Forgejo code, Go dependencies and runtime locks were
compared before recovery: no differences; only the Assembler media lock changed.
Probe 06 passed B's actual API readiness, local content and retained state with the
registry stopped, then committed B Git/project data. The profile assertion exposed
a fixture bug: `UpsertUser` intentionally does not rename an existing profile.
`GuestState(later)` now calls `RenameProfile`. Probe 07 uses a fresh copy under
`recovery-probe-07`, preserves those committed writes, and invokes that actual profile
operation once through a separately admitted `rename-fixture-profile-07` helper before
rollback (`probe-native-recovery-07`). It does not rerun `later`, recreate data or
restore a database.

Integrated driver development target: fresh `development-driver-01` roots under
`.artifacts/b4-qualification/` and `workers/soda-qualifier/`. Admit
`soda-build-p9-development-01` and `development-p9-01` helpers by hash; use the stopped
original `baseline-02` only as a verified copy source, its existing private console
credential, and the unchanged `b4-production-02` candidate/media. Exercise current P9
without running a build or creating a protected production `qualified.json`. The
launcher records explicit non-qualifying admission/results. Exact new registry,
listeners and both sequential disposable VMs stop on exit; retain all artifacts.

**Retained fast-development scope:** F1–F3 execution is complete; no benchmark/VM is
left running.
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

**Resumed B4 approval:** the grant below records the resumed target/action bounds,
not permission to repeat the rejected production-build debugging strategy.
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
