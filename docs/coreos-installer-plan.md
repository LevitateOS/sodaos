# CoreOS installer implementation plan

## Source status

The user subsequently requested implementation. Steps 1–4 now have a source
candidate and focused local coverage; see [the implementation/usage guide](coreos-installer.md).
Step 5's real media generation, hosted-payload retrieval, boot/disk writes and full
installed journey remain unrun, not accepted. The candidate has a tty1 console,
not a graphical or serial-only interface. An invalid/cancelled form exits; native
effects are never automatically retried.

Source-backed packaging correction: the upstream 256 KiB Ignition embed area cannot
hold the Go console. The small live config fetches the exact revision/architecture
binary from operator-selected HTTPS with an Ignition SHA-256 hash. The builder emits
but does not host/publish that public payload. Optional private NetworkManager input
supports pre-Ignition/static networking; it makes that ISO private per-machine media.
This is network-assisted, not an offline Soda appliance. Public bootstrap conversion
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
installer. QCOW2 delivery and a release/update platform remain outside this slice.

## 1. Lock and customize upstream media

- Inspect the exact selected CoreOS release and bundled installer capabilities.
  Add verified upstream live-ISO inputs for x86_64 and aarch64; the current
  `appliance/locks/coreos-qemu.json` covers QEMU disks, not ISOs.
- Use `coreos-installer iso customize --live-ignition` to deliver the bounded
  interface and Soda welcome/instructions. Keep upstream boot, storage and SELinux
  mechanisms; do not carry over `enforcing=0` or Anaconda CSS/profile files.
- General-purpose media contains neither credentials nor a fixed destination disk.
  `--dest-device` enables destructive unattended installation without confirmation;
  do not use it for general-purpose media. Destination options such as
  `--dest-ignition` also enable the automatic installer: keep the interactive path
  distinct from any later explicitly selected machine-specific automation.
- Write new outputs under ignored `.artifacts/`; retain source input identity,
  checksums and customization metadata. Do not fabricate locks or signatures.

## 2. Add bounded input and confirmation UI

| Input | Initial behavior |
| --- | --- |
| Installation disk | Explicit selection showing model, size, serial and existing partitions; exclude live backing media and reject mounted/in-use targets. Recheck identity immediately before writing. |
| Erase confirmation | Name the selected disk and require explicit confirmation; cancellation before execution makes no disk writes. |
| Hostname | Default `soda`, with ordinary hostname validation. |
| Network | DHCP default; optional static address/prefix, gateway and DNS through NetworkManager. |
| Operator SSH public key | Required under the current provisioning contract; import/paste and validate; never request a private key. |
| Operator password | Hidden entry and confirmation; privately generate the native crypt hash without plaintext in argv, echo, logs or evidence. |
| Project subnet | Collect for subsequent Soda installation, validate RFC1918 and known overlaps, and recheck on the installed host. Explain that developer-client routing is separate. |
| Review/progress | Show non-secret settings, destructive scope and distinct installation stages; retain redacted failure diagnostics. |

Use CoreOS's standard disk layout initially, not an Anaconda partition editor.
Do not add package selection, developer host accounts, Forgejo passwords, provider
registration or Tailnet enrollment to this interface. Console branding is new
source work; existing PNGs do not imply a graphical installer.

## 3. Reuse provisioning and invoke the engine

- Reuse `scripts/render-provisioning.py` and strict Butane conversion. Add a product
  hostname input without weakening the existing fixture-only hostname contract.
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
  The candidate selects operator SSH/SCP delivery into a root-owned tree plus an
  independently trusted SHA-256 of `SHA256SUMS`. Its already trusted compiled
  verifier checks the bundle before executing any bundled tool. No artifact
  publication service is assumed.
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

1. Local focused tests: input/hostname/key validation, hidden secret handling,
   network errors, wrong architecture, disk identity changes, in-use disks,
   cancellation, command failures and prevention of unintended auto-install.
2. Inspect customized output for expected live configuration, absence of credentials
   in general media and preservation of upstream verification/security settings.
3. With explicit fresh-target/disk approval, test boot, confirmed installation,
   destination Ignition, networking, extension activation, bundle installation and
   operator setup. Include failure paths without touching retained targets.
4. Record actual native x86_64 and aarch64 results independently; source tests,
   cross-compilation or ISO generation are not installed proof. Update the
   [handoff](implementation-status.md) with exact inputs and remaining gaps.

## Research references

- [Fedora bare-metal installation](https://docs.fedoraproject.org/en-US/fedora-coreos/bare-metal/)
- [CoreOS Installer ISO customization](https://coreos.github.io/coreos-installer/cmd/iso/)
- [CoreOS Installer install options](https://coreos.github.io/coreos-installer/cmd/install/)
- [Fedora live-media reference](https://docs.fedoraproject.org/en-US/fedora-coreos/live-reference/)

These upstream pages and exact CoreOS Installer v0.26.0 source were consulted.
Selected release/commit metadata is retained under `.artifacts/coreos-installer-research/`.
Local code checks are recorded in the handoff; no actual ISO generation, disk
installation or native validation accompanied the implementation.
