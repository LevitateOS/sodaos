# CoreOS installation media

Implementation of the [installer plan](coreos-installer-plan.md); see the
[handoff](implementation-status.md) for actual media-generation evidence.
A native x86_64 ISO is built with the console on media.
**ISO generation/inspection is not boot or fresh-appliance acceptance.** Anaconda
and Kickstart are not used. No upstream installer patches or OS filesystem
replacements are introduced. Legacy source and canonical artwork remain unchanged.

## Owners and prerequisites

- `scripts/build-installer.py` builds the native Go console and artifact verifier,
  fetches/verifies the selected upstream ISO, converts public Butane configuration
  and adds the console with xorriso's imported boot-equipment replay before stock
  `coreos-installer iso customize --live-ignition`. It compares upstream file hashes,
  boot references, volume identity and native kernel-argument/Ignition readback.
- `appliance/locks/coreos-iso.json` records actual release metadata for both native
  architectures, separate from the existing QEMU lock. Both selected RPM inventories
  report CoreOS Installer **0.26.0**; customization and runtime version checks require
  that version. The source-backed wrapper/readback contract is version-specific,
  including its locked Rust serializer's known absent-to-null optional fields.
- `appliance/installer` / `internal/installer` own the interactive disk adapter and
  explicit installed-host continuation. The media-only command stays outside the
  runtime `cmd/` staging loop: the application bundle must not overwrite a running
  bootstrap executable. The live launcher loads it from the ISO; no media build
  becomes an application build prerequisite.
- `scripts/render-provisioning.py::public_config()` remains the public bootstrap
  owner, including shared `branding_files()` for live and installed display identity.
  Butane strictly converts FCOS 1.6.0 to Ignition 3.5.0 at media-build time.
  The live Go adapter adds only validated per-machine fields to that template;
  Python and Butane are **not required on the live OS**. The standalone renderer
  also accepts `--appliance-hostname`; fixture `--hostname soda-native-*` is unchanged.

### SodaOS branding

ISO GRUB/ISOLINUX entries display **SodaOS Installer**, and the BIOS menu title and
GRUB theme class use SodaOS. Exact same-length text substitutions preserve native
kernel-argument embed offsets; readback rejects changes beyond those substitutions.
The signed EFI executable, kernel, initramfs and OS payload are not patched. Their
machine-readable boot-volume identifier and upstream attribution remain intact.

The console uses the canonical artwork and clears boot output on startup; its
systemd unit uses `Type=idle` to reduce status-message interleaving. Public live and
destination provisioning share `assets/branding/host/os-release` and the canonical
SVG icon. `/etc/os-release` supplies SodaOS display identity while retaining the
Fedora/CoreOS compatibility IDs. It does not copy a stale base version: the Go
installer validates the immutable `/usr/lib/os-release` and exact `IMAGE_VERSION`.
The existing vendor `/etc/os-release` entry and live `/etc/motd` are explicitly
replaced; other existing-file protections remain.

**Installed-disk GRUB titles are not yet rebranded.** Selected OSTree 2026.3 derives
BLS titles from `/usr/lib/os-release`, preferentially over `/etc/os-release`.
Completing that part requires an explicit native release-identity packaging choice,
not an assumed effect of the display override or a hidden boot-entry rewrite loop.
There is no installed-system branding acceptance yet. The ISO boot menus and
installed-disk boot entries are distinct owners.

### The console ships on the ISO

The **256 KiB limit applies to the Ignition embed area, not ISO capacity**. The
previous mandatory HTTPS executable payload was unnecessary and is removed.
The executable, LICENSE and NOTICE live in the ISO's ordinary `/soda/` directory.
Small live Ignition embeds `appliance/installer/load-console.sh` and the expected
console SHA-256. The launcher uses CoreOS's read-only `/run/media/iso` mount, copies
the executable to restricted live storage, verifies the copied bytes, publishes
without replacing an existing file and applies normal SELinux labeling before
making it executable. Failed copies remain non-executable for inspection.
**No web server, payload URL or executable download is needed.**

Xorriso replays the imported BIOS/EFI hybrid boot equipment. The volume label,
live kernel arguments, EFI image, kernel, initramfs and OS image are preserved;
only the bounded boot-menu display substitutions are allowed in GRUB/ISOLINUX configs;
BIOS boot-info addresses/checksum and outer ISO partition/layout metadata are
regenerated for the relocated files. ISO level 1 preserves primary names such as
`COREOS/KARGS.JSO`, which stock Installer 0.26.0 reads rather than Rock Ridge names.
The obsolete `/coreos/miniso.dat` absolute-offset copy table is removed: this is
**full-ISO delivery**, not a minimal-ISO/PXE export recipe. The normal live-ISO boot
is selected; `coreos.liveiso.fromram`/eject-before-start is not supported here.

Later Soda bootstrap still needs network access for RPM/repository dependencies;
this is not a fully offline appliance. CoreOS disk writing uses the full ISO's
offline image and never falls forward to a newer stream download. Optional static
networking remains available below, but is not needed to fetch the interface.

## Build recipe (not installation permission)

Use a clean exact-revision checkout on matching-native Linux, the repository-pinned
Go toolchain, native `gpgv`, an independently trusted Fedora keyring/full signer
fingerprint, xorriso, CoreOS Installer 0.26.0 and Butane supporting FCOS 1.6.0.
Observed tool versions and supplied executable hashes are recorded, not invented
locks. An auditable matching-native tool-container wrapper is allowed; its recorded
hash is the wrapper's, so retain its exact image digest/identity as well.
No tool installation, key import, host service or publication follows automatically.

Create a new output below an existing real `.artifacts/` parent:

```sh
python3 scripts/build-installer.py --arch x86_64 \
  --butane /absolute/tools/butane \
  --coreos-installer /absolute/tools/coreos-installer \
  --keyring /absolute/trusted/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --xorriso /usr/bin/xorriso \
  --out "$PWD/.artifacts/new-installer-attempt"
```

The output `soda.iso` carries its matching console. Inspect `media-build.json`,
`iso-inspection.json`, `SHA256SUMS`, `live.ign`, `remaster.log` and the upstream
signature-verification receipt. Checksums identify bytes; the upstream signature
covers the original image, **not the customized ISO**. No Soda release signature
or installed acceptance is implied.

For static/pre-Ignition networking, optionally add:

```sh
--network-keyfile /absolute/private/appliance.nmconnection
```

This input must be a bounded regular mode-0600 file, not a symlink. It is snapshotted
in the private attempt and its ISO-embedded bytes are read back and compared.
**That ISO is private per-machine media**, potentially containing network credentials.
Do not publish it as general media. The console binary itself is still
public/credential-free. General-purpose builds have no network keyfile, passwords,
SSH keys or destination disk. Occupied output directories are refused; failed
attempts are retained, never automatically removed or overwritten.

## Disk installation interaction

On the live ISO the bounded UI runs as root on **tty1** through a dedicated service,
not a passwordless root shell. Supplying live Ignition disables CoreOS's default
console autologin, so the UI has its own controlling terminal. This first candidate
requires a usable local text console; serial-only delivery is not yet implemented.
There is no automatic service restart or automatic disk selection/installation.

1. Keep current DHCP/networking or explicitly open native **nmtui** for address,
   prefix, gateway and DNS editing. Review observed interfaces and confirm network
   copying. This edits the live environment, not host policy on the builder.
2. Select a disk by number after reviewing model, size, serial, WWN and partitions.
   Mounted/swap, read-only, optical/live media and active mapped/held devices are
   refused. Btrfs/LVM/RAID/encrypted members are conservatively unavailable rather
   than treated as safe unused disks. Native CoreOS Installer owns final storage
   suitability and exclusive-access checks; no custom partitioning is selected.
3. Supply hostname (default `soda`), one operator SSH public key (paste or file),
   hidden native root/operator password and confirmation, and a private project
   subnet. Key comments/options are not copied as authorization directives.
   The password is SHA-512-crypt hashed by native OpenSSL over stdin, never argv.
4. Review the shared effects and type **`ERASE /dev/SELECTED_DISK`** exactly.
   Immediately recheck the kernel disk sequence, identity, partition inventory,
   mounts and holders before invoking stock `coreos-installer install --offline`.
   Do not hotplug the selected disk during installation; this is not a hardware
   identity/locking guarantee against arbitrary concurrent privileged changes.
5. Success means the CoreOS disk installation completed, not that Soda is ready.
   Remove installation media and reboot explicitly. The UI never initiates a reboot.

Invalid/cancelled input exits without invoking disk installation; it does not retry
native effects. The console service does not restart itself. Before a disk attempt,
a new operator-started review is possible; after an attempted write, the per-boot
marker blocks replay. On an error/interruption, preserve the live boot for inspection:
the disk may be partially written and started native work is not rollback. Private
Ignition remains mode 0600 in a mode-0700 run directory on live tmpfs; raw native
command diagnostics and passwords are not copied to the terminal/journal/evidence.

## Installed-host continuation

The destination receives the trusted console executable, selected project subnet
and a quiet noninteractive / guidance-only interactive root login hook. It does not
receive an automatic first-install/recovery daemon. After first boot:

```sh
sudo /usr/local/libexec/soda/soda-install continue
```

The command requires installed CoreOS with SELinux enforcing and checks the existing
extension-request marker and observed rpm-ostree boot state. If extensions are still
running/failed or a deployment awaits activation, it stops with guidance to inspect
native service status and perform the necessary explicit activation reboot.

Transfer the **sealed matching bundle** over independently trusted operator SSH/SCP
into a new root-owned directory, with root-owned contents and non-writable-by-others
ancestors. Preserve its final `x86_64` or `aarch64` directory name. Supply the SHA-256
of `SHA256SUMS` obtained through the trusted builder/channel—not a value copied from
the received bundle as a trust shortcut. The installer uses its already trusted
compiled native bundle verifier before executing **any** bundled program. A bundle
outside that verifier's source inventory needs matching media, not a bypass.

After subnet revalidation and explicit **`INSTALL SODA`** confirmation, it calls
existing `install-native.sh`. That script retains package/network/container/identity
preflight, image loading and first-install refusal. Existing or partial Soda state
and a continuation-attempt marker block replay. No live-environment post-install
hook calls the appliance installer; no retry, rollback or old-backup restoration
is provided on failure.

Finally follow [operator setup](operator-setup.md) for native Forgejo installation,
Soda OAuth and real private HTTPS activation. Cockpit stays loopback-first; neither
provider enrollment nor project creation/client routing occurs implicitly.

## Validation boundary

Local Go/race tests cover inputs, private terminal echo/cancellation, disk inventory
changes/use, pre-write cancellation/marker failures, command-failure redaction,
private file bounds and extension activation. Python tests cover strict conversion,
private network snapshots, live-only customization/readback and fresh-output refusal.
Launcher tests substitute only fixed paths and root/mount/SELinux commands, exercising
real copy/hash/exec and failure handling on synthetic files. A deliberately nonbootable
synthetic EFI ISO tests real xorriso preservation, primary names and tamper rejection;
synthetic BIOS bytes test relocated boot-info/PVD checks. These are not native proof.

Actual tty1 boot and enforcing-SELinux launch, static networking, confirmed disk
writes, first boot/reboot/continuation and complete operator setup still require
explicit fresh-target/disk approval and native proof. Native aarch64
is independent; cross-compilation or metadata are not installed evidence. See the
[handoff](implementation-status.md) for actual checks and unrelated packaging failures.
