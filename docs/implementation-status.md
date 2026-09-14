# Implementation status — single-run replacement

**Keep the original B1–B6 plan.** Finish the current source-to-media path, then start
B4 native qualification. Do not substitute an optimization programme or experimental
artifact preservation for that work. The [release plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns the milestone contracts; [history](implementation-history.md) owns detailed
receipts and superseded grants. Other workstreams remain in the [development handoff](development-handoff.md).

## Current position

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
- **The seven pre-B4 tasks are complete; ready to start B4.** Production-controller
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

**Not started.** Test actual installation, native updates/recovery, signatures,
offline content, maintenance and populated-state preservation against unchanged
candidate bytes. Connect protected qualification/evidence using the selected native
update contract. Do not recreate the former custom client updater.

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

Start B4 native qualification using the selected rpm-ostree/Zincati contract and
unchanged-candidate evidence. Do not reopen completed B3 work as a preservation,
compression or general audit programme. A new build must have fresh committed
source/controller/output; a read-only observer correction is not a reason to replay
successful production or disk effects. Public hosting, ARM and minimum-hardware
qualification remain separate work.

The public rootfs base URL is an explicit build input. GitHub Release assets can serve
hash-named ISO/rootfs files; this local work neither publishes them nor requires an
operated download service. Installed offline content means the required content is
local after the authenticated network installation, not that the minimal ISO is a
complete offline installer.

## Current permissions

The owner gave **standing approval for all seven pre-B4 tasks and necessary local
execution without routine handoffs**, including source simplification, media
integration, native installation verification and selection of the native update
contract. This supersedes the prior consumed-slot hold/pending deduplication proposal.

- Use fresh local candidate/media outputs and disposable VM/disk targets. Diagnosing
  failures and making another corrected attempt does not require another routine
  approval. Keep one active VM, affinity 0–3, at most four vCPUs and 16 GiB RAM.
- Measure resources; do not make blanket preservation or a 280 GiB retention increase
  the default. Upstream temporary/helper cleanup and disposable test-container
  lifecycle are permitted. Any cleanup must identify exact task-owned resources and
  protect credentials, unrelated work and explicitly protected state.
- Local fixture signing, loopback content serving, blank-disk installation, media
  removal/reboot and first-boot checks are within scope. Real provider jobs, public
  publishing, unrelated appliance mutation and M4 update/recovery execution are not
  needed to reach this stopping point.
- Existing real delivery authority remains candidate-channel-only within its original
  commissioning scope. It is not permission to replay bootstrap effects, reset keys/
  ledgers or publish preview/stable. This task uses no real signing/registry credentials.
- Stop for a concrete safety issue or an unresolved external blocker, not a routine
  milestone handoff. Report readiness only when the stated B3 checks actually pass.

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
