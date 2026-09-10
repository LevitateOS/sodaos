# CoreOS installer implementation plan

## Manual-install decision — 10 September 2026

The user exercised the ISO through a keyboard-only VM console and selected a
**root-password installation with no SSH public-key prompt**. The same flow must
work when booting physical hardware from a USB stick. This replaces the initial
required-key design below; it is a documented change to implement, not behavior
already present in the delivered ISO.

The current interface exposed three gaps: invalid disk input exits without a retry
screen; key entry assumes clipboard support or an already accessible live-system
file; and successful disk installation still hands off to SSH-based setup. Earlier
VM tests booted an upstream CoreOS QCOW2 with private Ignition inputs already
containing the operator key/password hash, then delivered Soda over SSH. Those
tests exercised running Soda components, not this manual installation journey.
See [recorded VM setup](local-testing.md) and the [handoff](implementation-status.md).

The replacement flow is network → disk → hostname → root password/confirmation →
project subnet → review → explicit erase/install → reboot → local operator login
and guided continuation. The password is native host access, separate from Forgejo
accounts. It must not silently enable ordinary root-password SSH.

### Concrete source work

1. In `internal/installer/install_linux.go`, remove key entry, live-file import,
   required-key validation and key-fingerprint review from disk installation.
   Reuse the existing hidden password entry, confirmation and private hashing.
2. In `internal/installer/inputs.go`, allow destination provisioning with a root
   password hash and no authorized key. Preserve normal key validation for actual
   key-import callers and the existing key-based automated fixture provisioning in
   `scripts/render-provisioning.py`; do not change retained accounts or credentials.
3. Make the console readable at ordinary VM/physical-console sizes, with clear
   choices, Back, correctable input errors and explicit restart after cancellation
   before writing. Keep private input out of later prompts. The existing launcher
   refuses to overwrite its loaded binary, so retry must reuse verified loaded
   code rather than blindly rerun the one-time loader. After disk writing starts,
   retain the attempt marker and failure evidence; no automatic erase retry.
4. Extend the installed continuation and login guidance to show dependency status,
   any required activation reboot, and the separate remote-access step below.
   Removing the initial key requirement must not imply that the existing
   SSH-dependent Forgejo bootstrap now works without an access path.
5. Update focused provisioning/console tests, the usage guide and the handoff.
   Rebuild media and validate the complete authorized fresh-disk journey before
   describing the replacement as a usable installer.

### Remote access after local installation

The recommended follow-up is a **locally armed, short-lived native SSH key-import
window**, separate from disk installation. After logging in locally with the root
password, the operator explicitly enables it on a selected reachable private
address. Show the native SSH host fingerprint and a laptop-side command that reads
the user's public-key file. The laptop transfers that public key through SSH; no
console clipboard, second USB drive, published key URL or private-key transfer is
required. A VM's network mode still has to provide a real client path.

Authenticate with the native operator password only within the explicitly enabled
window, and restrict this temporary connection to bounded key
enrollment: no ordinary shell, PTY, forwarding, SCP or arbitrary commands. Validate and
atomically install the key with native ownership/permissions; preserve other keys.
Close enrollment after one successful import, cancellation, timeout or reboot, and
require authenticated local action to reopen it. After the window closes, verify a
fresh ordinary key-based SSH connection before bundle transfer or native Forgejo
setup through an SSH tunnel; local password access
remains available if that check fails. Normal SSH remains key-based.

This is a design recommendation requiring implementation and exact native
authentication/timeout/failure validation. It is not an existing `soda-install`
subcommand, a persistent root-password SSH policy or a new Forgejo password
authority. The existing native Forgejo installer still uses an operator SSH tunnel;
there is no selected public browser-bootstrap server.

USB file picking, HTTPS public-key retrieval, personalized media and paired-browser
bootstrap were considered alternatives. They are not requirements for the selected
keyboard-only disk-install flow. Merely making the key optional would leave the
later SSH-dependent continuation incomplete.

## Current source status

The original implementation of steps 1–4 has a source candidate and focused local
coverage; see [the implementation/usage guide](coreos-installer.md). That coverage
does not include the password-only replacement specified above.
Step 5 now includes on-media packaging/readback checks; see the handoff for actual
media-generation and bounded diskless BIOS/UEFI boot evidence. Disk writes and the
installed journey remain unrun. The candidate has a tty1 console,
not a graphical or serial-only interface. An invalid/cancelled form exits; native
effects are never automatically retried.

Corrected packaging: 256 KiB is the **Ignition embed** limit, not the ISO's capacity.
Xorriso replays the imported boot equipment and adds the console to `/soda/` on the
ISO. A small live launcher copies it from CoreOS's read-only ISO mount and verifies
its SHA-256 before execution; no hosting URL or executable download is required.
ISO level-1 primary names preserve Installer 0.26.0's metadata lookups. Obsolete
absolute-offset miniso metadata is removed; full ISO, not PXE/minimal/fromram, is
selected. Optional private NetworkManager input makes the ISO private per-machine
media. Soda's later RPM setup remains network-assisted, not an offline appliance. Public bootstrap conversion
runs with Butane at build time; the live Go adapter adds only bounded private fields
because the selected live OS does not provide Python/Butane.

## Decision and boundary

Use the upstream Fedora CoreOS live ISO and stock `coreos-installer`, not
Anaconda, Kickstart or the predecessor's bootc installer. Add a small Soda-owned
installation interface; do not patch the upstream installer. The implemented first candidate is a console interface,
not an existing CoreOS form or a selected graphical framework. Preserve canonical artwork and the unchanged legacy repository.

This plan and its source candidate are not built/booted media evidence or permission to
write disks, start new fixtures, publish images or reinstall retained appliances.
The [installation guide](installation.md) owns existing installation contracts;
[native support](native-support.md) supplies artifacts/transport, not a second
installer. The [media delivery direction](installation.md#publication-direction)
adds a recommended QCOW2 deliverable and bundled Soda payload as separate remaining
packaging work. It does not select a host OCI/bootc migration or a release/update
platform, and discussion of downloads is not publication permission.

## 1. Lock and customize upstream media

- Inspect the exact selected CoreOS release and bundled installer capabilities.
  Add verified upstream live-ISO inputs for x86_64 and aarch64; the current
  `appliance/locks/coreos-qemu.json` covers QEMU disks, not ISOs.
- Add the console through xorriso boot replay, then use
  `coreos-installer iso customize --live-ignition` for its bounded verified launcher
  and Soda welcome/instructions. Keep upstream boot, storage and SELinux
  mechanisms; do not carry over `enforcing=0` or Anaconda CSS/profile files.
- General-purpose media contains neither credentials nor a fixed destination disk.
  `--dest-device` enables destructive unattended installation without confirmation;
  do not use it for general-purpose media. Destination options such as
  `--dest-ignition` also enable the automatic installer: keep the interactive path
  distinct from any later explicitly selected machine-specific automation.
- Write new outputs under ignored `.artifacts/`; retain source input identity,
  checksums and customization metadata. Do not fabricate locks or signatures.

## 2. Add bounded input and confirmation UI

| Input | Selected replacement behavior; implementation pending |
| --- | --- |
| Installation disk | Explicit selection showing model, size, serial and existing partitions; exclude live backing media and reject mounted/in-use targets. Recheck identity immediately before writing. |
| Erase confirmation | Name the selected disk and require explicit confirmation; cancellation before execution makes no disk writes. |
| Hostname | Default `soda`, with ordinary hostname validation. |
| Network | DHCP default; optional static address/prefix, gateway and DNS through NetworkManager. |
| Operator SSH public key | No prompt during disk installation. Remote key enrollment follows local password login as a separate step. |
| Operator password | Hidden entry and confirmation; privately generate the native crypt hash without plaintext in argv, echo, logs or evidence. |
| Project subnet | Collect for subsequent Soda installation, validate RFC1918 and known overlaps, and recheck on the installed host. Explain that developer-client routing is separate. |
| Review/progress | Show non-secret settings, destructive scope and distinct installation stages; allow correction/cancellation before writing and retain redacted failure diagnostics. |

Use CoreOS's standard disk layout initially, not an Anaconda partition editor.
Do not add package selection, developer host accounts, Forgejo passwords, provider
registration or Tailnet enrollment to this interface. Console branding is new
source work; existing PNGs do not imply a graphical installer.

## 3. Reuse provisioning and invoke the engine

- Reuse the public bootstrap shared with `scripts/render-provisioning.py` and
  strict Butane conversion. Preserve product-hostname and fixture-hostname contracts.
  The interactive destination must accept a password without an SSH key; automated
  fixture provisioning retains its separately supplied key inputs.
- Keep live-environment Ignition separate from destination Ignition. Store private
  inputs in restricted files/directories; per-machine media containing those inputs
  must itself be treated as private, never published as a general ISO.
- After confirmation, invoke stock `coreos-installer install` with the checked disk,
  destination Ignition and selected network-copy behavior. Preserve signature
  verification; do not add insecure flags or reconstruct secrets into arguments.
- Report installation failures honestly, including possible partial disk writes.
  Do not automatically erase again, retry a partial installation or reboot on error.

## 4. Complete Soda installation after boot

- Destination Ignition establishes native operator access and the existing
  `soda-extensions.service`. Show dependency installation status and require a clear
  activation-reboot step; successful CoreOS disk writing is not Soda readiness.
- Provide a bounded continuation on the installed host: obtain the matching sealed
  Soda bundle through a trusted delivery channel, establish its integrity before
  executing bundled tools, and call existing `install-native.sh` after RPM activation.
  The current source selects operator SSH/SCP delivery into a root-owned tree plus an
  independently trusted SHA-256 of `SHA256SUMS`. Its already trusted compiled
  verifier checks the bundle before executing any bundled tool. No artifact
  publication service is assumed. The selected media direction includes the matching
  sealed Soda payload in installation media, with a protected handoff onto the
  destination before media removal and verification before execution. That replaces
  manual bundle transfer for the normal media journey; it does not bypass the
  existing native installer, activation ordering or payload trust checks.
- Do not run that script in the live environment's `--post-install` hook: it expects
  the running installed CoreOS system and activated native prerequisites.
- Preserve first-install refusal/partial-state markers. A continuation must not
  replay first-install over existing or partial Soda state or become a recovery daemon.
- Hand off to existing native Forgejo setup, Soda OAuth configuration and private
  HTTPS activation. Keep Cockpit loopback-first and operator-only. No automatic
  provider enrollment, project creation or network exposure.

CoreOS's full ISO can install its base OS offline. The complete Soda path currently
needs network access for repository configuration and RPM dependencies; do not claim
an offline appliance. Embedding/caching those dependencies is separate work.

## 5. Validate before delivery

1. Local focused tests: password-only destination provisioning, imported-key validation,
   preserved key-based fixture inputs, hidden secret handling and password mismatch,
   network errors, wrong architecture, disk identity changes, in-use disks,
   Back/retry/cancellation before writing, command failures and prevention of unintended
   auto-install or retry after a disk attempt.
2. Inspect customized output for expected on-media console bytes/live configuration,
   no credentials in general media, native Ignition/kargs readback, preserved upstream
   file hashes and boot references. Cover primary ISO names and BIOS relocation.
3. With explicit fresh-target/disk approval, test boot, confirmed installation,
   destination Ignition, password-only local login after reboot, networking,
   extension activation, included-payload handoff/verification and operator setup.
   Prove key enrollment from a separate client, normal key login, closure on
   success/timeout/reboot, denial of shell/forwarding in enrollment mode and unchanged
   normal SSH password policy. Include failure paths without touching retained targets.
   Exercise keyboard-only VM input and actual USB-media boot separately; neither
   clipboard injection nor preloaded operator keys substitute for those journeys.
4. Record actual native x86_64 and aarch64 results independently; source tests,
   cross-compilation or ISO generation are not installed proof. Update the
   [handoff](implementation-status.md) with exact inputs and remaining gaps.

## Research references

- [Fedora bare-metal installation](https://docs.fedoraproject.org/en-US/fedora-coreos/bare-metal/)
- [CoreOS Installer ISO customization](https://coreos.github.io/coreos-installer/cmd/iso/)
- [CoreOS Installer install options](https://coreos.github.io/coreos-installer/cmd/install/)
- [Fedora live-media reference](https://docs.fedoraproject.org/en-US/fedora-coreos/live-reference/)
- [Selected live generator](https://github.com/coreos/fedora-coreos-config/blob/682c839aabbc01564f1605bb41687a7511180031/overlay.d/05core/usr/lib/dracut/modules.d/35coreos-live/live-generator)
- [Installer 0.26.0 embed metadata](https://github.com/coreos/coreos-installer/blob/v0.26.0/src/live/embed.rs)
- [Installer 0.26.0 miniso copy table](https://github.com/coreos/coreos-installer/blob/v0.26.0/src/miniso.rs)

These upstream pages and exact CoreOS Installer v0.26.0 source were consulted.
Selected release/commit metadata is retained under `.artifacts/coreos-installer-research/`.
The selected live generator's `/run/media/iso` mount and Installer v0.26.0's primary
ISO-name lookup/miniso copy-table code were also inspected. Source checks and actual
media generation/inspection are recorded in the handoff, separately from still-held
boot/disk installation and native acceptance.
