# CoreOS installer implementation plan — single-run candidate consumer

## Priority and implementation order

The active installer work is part of the [six single-run replacement milestones](release-engineering-plan.md#single-run-build-replacement-implementation),
not a separate writable-bundle release to finish first. B1 proves the upstream
mechanism; B2 produces programs/assets/candidate; B3 consumes them in media and
installs the image; B4 proves install/update/recovery. B5 integrates protected release
finalization; B6 removes the old producers after native proof.

Current password-only console, media verification and setup code are reusable
foundations. Preserve their tested behavior while replacing the software-installation
backend. Do not require another old-style ISO release, public registry commissioning,
a timer or production migration before implementing the replacement.

## Image-based replacement contract

- Media consumes an already built, signed host/application candidate and prebuilt
  console/tools. No program/image build, mutable image lookup, second OS assembly,
  client-side RPM layering or bundled `install-native.sh` occurs in the target lane.
  Install and update use identical immutable host/app digests.
- Apply the [minimum-deviation FCOS contract](release-engineering-plan.md#minimum-deviation-fcos-contract):
  retain CoreOS Installer disk installation and Ignition provisioning, with stock
  live boot and supported media customization. Verify exact native inputs before
  coding; `coreos-installer install --offline` is not assumed to accept a derived OCI
  host. No Soda partitioning/boot backend, predecessor installer or insecure flags.
- Follow the [selected minimal network-install target](#selected-media--minimal-network-install),
  replacing the earlier self-contained/offline installation requirement. Prove download
  authentication, exact candidate selection, storage lifetime and media removal.
  Provider/marketplace networking remains a separate runtime requirement.
- Native installation provides image-owned software. Machine identity, host keys,
  network/subnet, root access and first-operator/application setup remain per-machine
  work, not baked-in credentials/databases. Do not replay first-install over retained
  or partial state; native failures are not automatic erase/reinstall permission.
- Keep bulk software off the distributed ISO; bounded bootstrap must not become
  oversized Ignition. Preserve boot equipment/primary names and verify content,
  ownership, SELinux and space. Private network media remains private. Final
  qualification stays outside the ISO so signing does not rebuild tested bytes.

### Selected media — minimal network install

**Owner-selected priority: less on the ISO; download the payload during installation
where supported by native FCOS.** Minimize actual ISO size, not merely fit under the
ceiling: the distributed ISO must be below **2 GB (2,000,000,000 bytes)**. This is a
conservative bound for [GitHub Release's per-asset limit](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases),
not a GHCR 2 GB limit. Do not invent a smaller numerical target before measuring.

- Prefer the upstream minimal-ISO/network-boot mechanism. Keep only necessary boot
  and small bootstrap/trust material on media; download the matching live filesystem
  and host/application content during the installation session. A full image may
  remain an internal upstream packaging input, not a second release/build lane or
  a required public download. Networkless installation is no longer a requirement.
- Preserve the one immutable install/update candidate. Downloads select its exact
  admitted content, never mutable latest versions or every package/tag in a registry.
  Fetching a candidate-derived live filesystem containing the application archives
  is compatible with this goal; separate GHCR pulls for each image are not required.
  GHCR is not mandated as the sole download source. Every separately hosted artifact
  must fit its host's limits; moving a large rootfs beside the ISO on GitHub Releases
  does not evade the per-asset limit. Hosting/publication requires its own grant.
- Authenticate the bootstrap and all downloaded executable/software content before
  use; verify the native rootfs-to-bootstrap identity binding. No insecure fallback,
  custom boot downloader or stock-install/second-conversion lane to save ISO bytes.
  Measure ISO size, downloaded bytes and memory/disk requirements separately.
- Minimal boot may require networking before the Soda wizard can run. Prove supported
  pre-live network configuration and honest network/download failure handling; the
  current post-boot wizard and ISO-mounted console loader are not proof of that path.
  Preserve password-only setup, disk/erase guards, cancellation and no write replay.
- Installation must finish with the required host and all five application images
  available locally; do not merely defer missing installation payload to normal
  first boot. Media removal and normal host/app startup without further payload
  downloads remain required. This is not a claim of a network-independent appliance.

This changes the media/download requirement, not the native ownership or client-trust
contract. B3 now has scoped candidate-media, authenticated DHCP network boot,
installation/media-removal and exact first-boot evidence; the
[native fixture receipt](implementation-history.md#b3-native-candidate-installation-and-interrupted-write)
records sizes, resources and limitations. Protected production phase integration and
broader qualification remain outstanding.

### B1 native mechanism review — reopened

**Native candidate-derived installation is fixture-proved; B1 update/trust work
remains open.** The owner clarified
that minimum deviation from mature FCOS installation/updating takes priority over
the earlier bootc bound-image recommendation. The `bootc to-filesystem` proposal,
Soda-managed `sfdisk`/`mkfs` layout, hand-written first-boot marker/BLS handling and
manual bootc finalization sequence are withdrawn. Calling upstream tools within
that sequence would still make Soda responsible for reproducing FCOS integration.
No such adapter was implemented or disk installation performed.

Current Soda source in `internal/installer/install_linux.go` invokes **CoreOS
Installer 0.26.0** with `install --offline --ignition-file ... --copy-network` after
its disk/confirmation guards. Public Butane/Ignition owns provisioning. The retiring
profile requests packages through an rpm-ostree extension unit; the candidate profile
uses image-owned software without that continuation. Merely appending an OCI archive
to stock media does not change the host/application set its osmet reconstructs.
Preserve the native installation owner while replacing the separate software
producer/continuation, rather than replacing disk installation to suit its output.

#### Source-backed packaging route

The locked FCOS `44.20260817.3.2` build metadata records **OCI import by CoreOS
Assembler**, at commit `fa114018875a04c3df39dca17ab57274764bf563`, using FCOS config
`682c839aabbc01564f1605bb41687a7511180031`. This supplies an upstream route to
investigate without Soda disk-layout code or a CoreOS source rebuild:

```text
one admitted host OCI archive (including required application archives)
  → upstream cosa import --skip-prune
  → upstream cosa buildextend-live / OSBuild metal + metal4k dependencies
  → candidate-derived minimal ISO + separately hosted matching live rootfs/osmet
  → native network boot downloads and authenticates the matching live content
  → CoreOS Installer --offline (local osmet after download) + destination Ignition
  → native installed OSTree deployment with the candidate's OCI update identity
```

Native fixtures now prove unchanged OCI import, live packaging and installation
with the exact booted OCI digest. This is not yet a runnable Soda release command. It changes the earlier stock-ISO-remaster-only assumption: adding a
Soda OCI tar to an unchanged Fedora live ISO does not change the stock disk image
that `--offline` reconstructs. Media generation is an upstream packaging consumer
of the already-built candidate, not a second compilation of Soda or the OS. The
owner's minimal-network target changes distribution of that content, not its identity.
The disk writer's `--offline` flag describes use of already-downloaded osmet,
not an offline installation session. Native minimal extraction/customization and
rootfs authentication have scoped fixture proof; preserve that order and boundary.

- [Assembler import](https://github.com/coreos/coreos-assembler/blob/fa114018875a04c3df39dca17ab57274764bf563/src/cmd-import)
  copies an `oci-archive:` input unchanged and records its manifest/archive identity.
  Use a fresh dedicated workspace and `--skip-prune`; never reuse retained roots.
  Its import trusts local input, so protected native signature/content admission must
  precede it. Its required `containers.bootc=1` label is inherited FCOS metadata, not
  selection of the bootc installation engine.
- [Native image assembly](https://github.com/coreos/coreos-assembler/blob/fa114018875a04c3df39dca17ab57274764bf563/src/cmd-osbuild)
  defaults `bootc-install-to-fs` to false; the locked FCOS image config does not enable
  it. Upstream OSBuild owns OSTree deployment, Ignition setup, partitions, filesystems,
  boot integration and labeling. Keep those upstream manifests, not Soda copies.
  The builder explicitly avoids a containers-storage optimization because it changed
  deployed digests and broke Zincati; preserve the admitted OCI archive path.
- [Live packaging](https://github.com/coreos/coreos-assembler/blob/fa114018875a04c3df39dca17ab57274764bf563/src/osbuild-manifests/platform.live.ipp.yaml)
  consumes native metal/metal4k output. [CoreOS Installer osmet](https://github.com/coreos/coreos-installer/blob/22d9f23e9c35ee035ed632a20d38eae000536687/docs/osmet.md)
  reconstructs the original raw image bit-for-bit from live-root objects plus packed
  disk metadata, checking its checksum. Installer then owns destination Ignition,
  network copying and first-boot arguments. Do not manually recreate those operations.
- **Trust boundary:** Installer trusts osmet in the live environment; its checksum is
  not independent authentication. Verify the final Soda-signed ISO binding before
  privileged boot/use. Fedora's signature on its original ISO does not authenticate
  a derivative. Direct `--image-file` instead requires a detached GPG signature using
  compiled-in keys in Installer 0.26.0; the existing P-256 Sigstore keys are not that
  keyring. Neither `--insecure`, a GPG wrapper nor a custom installer build is selected.
- **Locally available apps after download:** recommend embedding all five exact application archives in the
  immutable host and importing into ordinary Podman storage through native commands
  and systemd ordering. This uses the signed host as their integrity boundary and
  avoids bootc-bound storage. B2 now implements [payload v2 and all-five import](release-engineering-plan.md#local-candidate-content-and-machine-state-ownership),
  with embedded archive identities checked in the native host candidate. Strict v1
  readers retain the historical meanings. B3 verified installed archive hashes, five
  local Podman IDs and successful import before Forgejo startup without external
  networking. Full configured-appliance qualification remains separate. Preserve existing
  projects and later writes; image import is not permission to replace workloads.

B3's executable review now selects the x86_64 Assembler manifest
`sha256:f010dce4d350c1588762bbd5b69d27e14dabe859043489c587dc4d429c067daf`
from `quay.io/coreos-assembler/coreos-assembler`, revision
`53330beeb45bb0a6f51987fc8df243e8a62d62bd`. Its inspected OSBuild 191 live stage
owns the stream-hash generation described below. Native packaging subsequently
succeeded with this pinned toolchain. Initial import/helper-VM attempts exposed two
actual caller requirements: the temporary Python source must select root (the pinned
Assembler defaults to `builder`), and local archives used inside supermin must live
under its shared `/srv`, not a container-only `/inputs` mount. A metadata-only `USER 0`
wrapper retains the pinned builder's identical rootfs layers; it changes no shipping
candidate bytes or tool packages. The corrected shared-path handoff passed in
packages 04–06; package 04 then exposed the separately fixed merged-bin overlay bug.

Assembler's supermin prelude also prunes its guest-local cache and removes temporary
helper roots on exit, including failed commands. `cosa import --skip-prune` does not
suppress those separate actions. Do not run the helper without applicable authority
for its full scratch lifecycle. The [approved fixture extension](implementation-status.md#b3-packaging-extension--approved)
owns that permission, not a Soda patch to the native disk manifests.

Before VM execution, record resource/effect bounds for
its supermin build VM and a separate fresh installation target. Prove reconstructed
raw checksums, installed OCI digest/origin, untouched shipping content, native boot/
Ignition/SELinux, minimal-media size/download integrity, media removal and local
image availability after installation. Preserve original
Soda tool/readback guards while adapting them to upstream-generated media. The
[native installation receipt](implementation-history.md#b3-native-candidate-installation-and-interrupted-write)
records the successful exact-candidate fixture and interrupted-write limits. The
[initial bootstrap receipt](implementation-history.md#b3-native-bootstrap-and-builder-admission)
and [initial packaging failures](implementation-history.md#b3-native-import-and-stopped-packaging-attempts)
remain historical evidence, including the preservation limitation.

The [update and authority findings](release-engineering-plan.md#b1-native-update-and-authority-findings)
identify a remaining client-trust decision. Do not claim this packaging route also
implements the existing signed-channel admission contract.

The [earlier B1 receipt](implementation-history.md#b1-native-installation-contract-and-removal-baseline)
retains useful version/config/source observations, archive sizing and tests. Its
separate-boot/Ignition mismatch is evidence against treating the simple bootc disk
path as a drop-in FCOS installer. It does not establish that FCOS lacks a supported
solution, nor qualify the withdrawn workaround. Existing signature/tamper, media
readback and no-replay requirements remain; this correction does not weaken them.

### B3 native download authentication handoff

The inspected OSBuild `org.osbuild.coreos.live-artifacts.mono` stage writes SHA-256
hashes of **2 MiB chunks** of the completed live rootfs into
`/etc/coreos-live-want-rootfs` in the initramfs. FCOS's selected live boot service
uses `curl → rdcore stream-hash → bsdtar`; rdcore buffers and verifies each chunk
**before releasing it to the extractor**. Missing, changed, truncated or extra input
fails, and the boot service isolates to the native emergency target. Its executable
passed scoped native valid/corrupt/truncated/extra/missing-stream tests in B3.

The trust chain is independently authenticated final ISO → included initramfs hash
list → authenticated downloaded rootfs chunks. A version stamp or a hash supplied
by the download server is not a substitute. Preserve this native mechanism; no Soda
boot downloader or extra GPG adapter is needed. Upstream curl deliberately does not
rely on TLS certificate validation here; the authenticated chunk list is the content
authority. This is not permission to pass CoreOS Installer `--insecure`, disable
Ignition verification, carry credentials in URLs or accept unsigned media.

Extract a **clean minimal ISO**, then perform one native customization carrying
both the exact rootfs URL and live Ignition. Supplying the URL during extraction
already changes kernel arguments; subsequent `iso customize` refuses that input
without force. Preserve the failed probe, rather than forcing or resetting old media.
Minimal extraction removes `coreos.liveiso`; the old mounted-ISO loader is not used.

The Go controller now links the once-compiled installer into the candidate at
`/usr/libexec/soda/soda-install`, checks its equality with the prebuilt tool and emits
public live/destination Ignition using pinned native Butane. The same authenticated
rootfs supplies this console: **no separate executable download or loader is needed**.
Live Ignition starts only the bounded tty1 wizard and masks appliance workloads in
the temporary live OS. Candidate media binds the console/payload hashes and expected
host manifest. The wizard verifies all five local archives, retains the existing
password/disk/revalidation/no-replay guards, and invokes unchanged CoreOS Installer
with native osmet. No writable bundle copy, package transaction or continuation is
part of the candidate install. Historical media format 0 remains supported separately.
The destination uses native Ignition for private machine configuration and root
password, keeping ordinary SSH key-only. Installed setup/enrollment uses the same
vendor console; no key transcription or private developer key is introduced.

Native probes reached the live environment with the 160 MB ISO and matching rootfs;
missing, corrupted and truncated downloads entered upstream emergency mode without
running the live probe. `rpm-ostree status` is not a valid live identity observer here:
the live EROFS root has no installed `/boot/loader`. The subsequent final-candidate
fixture ran the embedded console, installed through native osmet, removed media and
verified exact booted deployment/origin, enforcing SELinux and all-five local content.
Cancellation before writing left the disk untouched; interruption during writing
reported partial state and did not automatically replay. Explicit same-boot
re-invocation refusal remains source-tested, not independently exercised in the guest.
See the [scoped receipt](implementation-history.md#b3-native-candidate-installation-and-interrupted-write);
these checks do not establish update/recovery, physical USB, minimum RAM or a complete
protected source-to-qualified-release run.

### Implementation responsibilities

| Owner | Required change in the replacement |
| --- | --- |
| `tools/soda-build` and existing Go build packages | Build the candidate and media programs once; own timing, cancellation, source identity and exact handoff |
| `scripts/build-installer.py` | Become media-only assembly (rename to `assemble-installer.py` if retained); remove native-build invocation, Go compilation and independent supervision |
| `appliance/installer`, `internal/installer` | Preserve CoreOS Installer/Ignition ownership and the text/disk/secret guards; change candidate handoff only after B1 proves the supported path |
| Public provisioning/branding owners | Reuse public bootstrap and canonical artwork; separate live from destination inputs and bounded private additions |
| Existing media/native test drivers | Verify actual media readback and native installation/update/recovery; consume prebuilt artifacts, never another production build |

No new numbered installer backlog competes with B1–B6. Detailed build and qualification
exits belong to the release plan; this guide owns the installation interaction and
media/bootstrap contract.

## Manual-install decision — 10 September 2026

The owner selected **root-password installation with no SSH public-key prompt**,
usable through a keyboard-only VM console and physical USB boot. This remains the
interaction contract for the image-based replacement, not a reason to retain the
writable-bundle backend. A graphical session, Anaconda and Kickstart are not selected.

The native root password provides local host access, separate from Forgejo accounts.
It must not silently enable ordinary root-password SSH. Previously exercised fixtures
used upstream QCOW2 and private Ignition with preloaded operator credentials; that
runtime evidence does not prove the manual installation journey.

### Selected text interface and completion contract

| Screen or phase | Required behavior |
| --- | --- |
| Welcome | Clear boot clutter, explain stages and start only on operator input |
| Network | Keep DHCP/current settings or open native nmtui; show observed settings and allow correction |
| Disk | Show model/size/serial/partitions and unavailable reasons; select a listed disk, never a default; reject live/in-use media and recheck identity just before writing |
| Hostname | Default `soda`; preserve valid non-secret values on Back and explain field validation |
| Password | Hidden entry/confirmation for local root access; correct mismatch without restarting; never echo/review plaintext or enable ordinary password SSH |
| Project network | Validate RFC1918/known overlaps and recheck on the installed host; preserve useful choices and distinguish client routing |
| Final review | Show non-secret settings and actual disk; require the exact named-disk ERASE phrase; a typo leaves review open with no writes |
| Installation | Verify the admitted candidate, invoke the selected native image-install operation once, confirm complete handoff before media removal; no automatic retry or reboot |
| Installed setup | Establish/verify machine-owned identity and explicit operator/application setup; do not install a second writable software bundle |
| Access and browser setup | Separate locally armed public-key enrollment, verified laptop SSH and private-IP HTTPS with explicit client trust; no console key transcription or domain purchase |

Back/restart/cancel are available where applicable. Back preserves non-secret choices;
returning through password entry requires a fresh secret. Pre-write restart reuses
the verified loaded executable, not the one-time exclusive loader. After an attempted
write, cancellation/failure preserves the marker and redacted diagnostics, reports
possible partial state and does not imply rollback or silently start another erase.
Redraw/reconnect never initiates a native effect.

No partition editor, package selection, developer host accounts, provider enrollment,
Forgejo password or automatic project creation belongs in disk installation. Use the
selected upstream standard layout after B1 confirms it for the image mechanism.

### Remote access after local installation

Keep a **locally armed, short-lived native SSH key-import window** separate from disk
installation. After local root-password login, the operator explicitly enables it on
a selected reachable private address. Display the native host fingerprint and a laptop
command reading the user's public-key file; never request or transfer a private key.
A VM network mode must provide an actual client route.

Password authentication is allowed only within that explicitly enabled enrollment
window, restricted to key import: no shell, PTY, forwarding, SCP or arbitrary command.
Validate native ownership/modes and preserve other keys. Publish a missing file
exclusively or append to the validated existing inode without replacing it. Concurrent
native edits/partial writes report uncertainty, not stale restoration or automatic retry.

Close after one successful import, cancellation, timeout or reboot; reopening needs
local authenticated action. Verify a fresh ordinary key-based SSH connection before
SSH-dependent Forgejo/bootstrap work. Normal SSH remains key-based and local password
access remains available if remote checks fail. The existing `soda-install enroll-key`
source and private browser setup are reusable; their native authentication/failure
contract still needs proof with the replacement installation.

The native Forgejo setup still uses an operator SSH tunnel, not a public bootstrap
server. Private browser activation retains existing certificate/OAuth/setup owners.
USB key picking, hosted key retrieval, personalized media and paired-browser bootstrap
are not prerequisites for this selected keyboard-only installation.

## Media and bootstrap verification

- Use exact selected architecture-specific live-ISO inputs, trusted upstream keyring/
  signer and version-checked native media tools. QEMU locks do not substitute for ISO
  locks. Preserve upstream attribution, signed EFI executable, kernel and initramfs.
- General media has no credentials or destination disk. Do not use automatic-disk
  `--dest-device` or other options such as `--dest-ignition` that activate unattended
  installation in place of the interactive path. Machine-specific automation would
  require separate selection and exact disk/action authority.
- Live and destination Ignition stay separate. Reuse strict public Butane conversion
  and fixture/product hostname contracts; the live Go adapter adds only bounded
  private fields because the selected live OS lacks Python/Butane. Keep password
  hashes/private network data in restricted files, never argv/logs or general media.
- The 256 KiB limit is the Ignition embed area, not total ISO capacity. Existing
  ordinary `/soda/` media content and a restricted hash-verifying launcher are reusable.
  Prove larger candidate/blob sizes against per-file ISO limits without breaking the
  primary names used by CoreOS Installer 0.26.0. Preserve imported boot-equipment
  replay, volume identity, native kargs/Ignition readback and bounded branding edits.
- The [minimal network-install target](#selected-media--minimal-network-install)
  supersedes the earlier full-ISO selection. The retiring remaster removes obsolete
  absolute-offset miniso metadata; do not apply that operation to native minimal
  extraction inputs. Prove upstream extraction/customization order and readback,
  rather than treating the old full-ISO recipe as a minimal-ISO producer.
- The final signed release record authenticates the resulting ISO hash via independent
  trust. An embedded hash or public key alone cannot authenticate malicious media to
  an outside consumer. Qualification reports are produced after testing, outside ISO.

## Qualification and source retirement

B3/B4 use exact authorized native fixture/disk/baseline inputs. Preserve existing
local tests and add image-handoff cases: wrong candidate/architecture/signature,
oversized/missing content, disk changes/in-use disks, private input/Back/restart,
command failure, incomplete setup, no unintended auto-install and no write replay.
Media readback checks content/permissions/boot identity and confirms no component
compiler/image-builder was invoked.

Actual product evidence includes measured minimal ISO size, authenticated network
boot/downloads, missing network/content and interrupted-download failures, confirmed
installation, media removal, password-only local login, host/app startup without
further payload downloads, enforcing SELinux, private setup
and applicable first-project journey; then the same candidate's supported update/
recovery tests. Prove key enrollment and normal key login, timeout/reboot closure,
restricted enrollment commands and unchanged ordinary SSH policy. Keyboard-only VM
and physical USB boot are distinct claims; clipboard injection/preloaded keys or
cross-compilation do not establish them. Native architectures have separate receipts.

After these exits, B6 removes old producers and bundle/extension installation from
the new lane. Preserve existing installer usability until that point and retained
artifacts/maintenance readers afterward. Do not finish an old-style public ISO or
commission scheduling before replacing the build. Public delivery and production
migration follow B6 under their own grants.

## Current source status

Source supports the candidate's embedded prebuilt console/native image installation
alongside the retiring writable path. The fixed Go run still stops before protected
media assembly/qualification. The [current status](implementation-status.md#3-make-the-iso-consume-the-candidate--b3)
and [native receipt](implementation-history.md#b3-native-candidate-installation-and-interrupted-write)
distinguish implemented code, exact tested candidate and outstanding integration.
No producer exists for a preinstalled SodaOS QCOW2.

The [implementation status](implementation-status.md) tracks B1–B6 and current release
grants. [Development custody](development-handoff.md) retains earlier media/fixture
evidence; earlier holds or a recorded machine-unavailable deferral are not fresh
lifecycle grants or a reason to block independent source work. This plan is not
permission to write disks, publish, change trust or migrate a retained machine.

## Research references

- [Fedora bare-metal installation](https://docs.fedoraproject.org/en-US/fedora-coreos/bare-metal/)
- [CoreOS Installer ISO customization](https://coreos.github.io/coreos-installer/cmd/iso/)
- [CoreOS Installer install options](https://coreos.github.io/coreos-installer/cmd/install/)
- [Fedora live-media reference](https://docs.fedoraproject.org/en-US/fedora-coreos/live-reference/)
- [Selected live generator](https://github.com/coreos/fedora-coreos-config/blob/682c839aabbc01564f1605bb41687a7511180031/overlay.d/05core/usr/lib/dracut/modules.d/35coreos-live/live-generator)
- [Installer 0.26.0 embed metadata](https://github.com/coreos/coreos-installer/blob/v0.26.0/src/live/embed.rs)
- [Installer 0.26.0 miniso copy table](https://github.com/coreos/coreos-installer/blob/v0.26.0/src/miniso.rs)

These sources were consulted for the earlier implementation. Inputs remain under
`.artifacts/coreos-installer-research/`; the build replacement still requires B1's
reopened FCOS-native install/update caller review. Historical media checks are not
new native proof.
