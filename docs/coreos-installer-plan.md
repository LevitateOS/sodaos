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
- Inspect the selected upstream bootc install/local-content mechanism and actual Soda
  caller before coding. Keep stock live boot and supported media customization.
  `coreos-installer install --offline` is not assumed to accept a derived OCI host.
  Do not import the predecessor installer, invent a boot backend or add insecure flags.
- General ISO installation and first boot must have host, bound-app and retained-runtime
  content without a registry dependency. Prove transport, signature policy, storage
  lifetime and media removal. Provider/marketplace networking is outside this claim.
- Native installation provides image-owned software. Machine identity, host keys,
  network/subnet, root access and first-operator/application setup remain per-machine
  work, not baked-in credentials/databases. Do not replay first-install over retained
  or partial state; native failures are not automatic erase/reinstall permission.
- Keep the candidate and console in ordinary ISO files, not oversized Ignition.
  Preserve boot equipment/primary names and verify content, ownership, SELinux and
  space. Private network media remains private. Final qualification stays outside
  the ISO so signing/publication does not rebuild tested bytes.

### Implementation responsibilities

| Owner | Required change in the replacement |
| --- | --- |
| `tools/soda-build` and existing Go build packages | Build the candidate and media programs once; own timing, cancellation, source identity and exact handoff |
| `scripts/build-installer.py` | Become media-only assembly (rename to `assemble-installer.py` if retained); remove native-build invocation, Go compilation and independent supervision |
| `appliance/installer`, `internal/installer` | Invoke the B1-selected image installation path, preserving the text/disk/secret guards below; replace extension/bundle continuation with per-machine setup |
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
- Full ISO is the selected path, not PXE/minimal/fromram. Existing removal of obsolete
  absolute-offset miniso metadata is version-specific evidence, not a generic ISO
  rewrite rule. Do not weaken readback to fit a new payload.
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

Actual product evidence includes the real ISO, confirmed installation, media removal,
password-only local login, offline host/app startup, enforcing SELinux, private setup
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

Existing source has password-only fields, correctable input, pre-write restart,
separate key enrollment/private browser setup, ordinary on-media console packaging
and writable bundle continuation. It does **not** implement image-based installation
or the media-only build handoff. Earlier built required-key media has bounded diskless
BIOS/UEFI proof, not this replacement's fresh-disk acceptance. No producer exists for
a preinstalled SodaOS QCOW2.

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
exact bootc/install caller review. Historical media checks are not new native proof.
