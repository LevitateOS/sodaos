# CoreOS installation media

Implementation of the [installer plan](coreos-installer-plan.md); see the
[handoff](implementation-status.md) for actual media-generation evidence.
A native x86_64 ISO is built and has bounded diskless BIOS/UEFI boot proof, with
the console on media.
**ISO generation/inspection is not boot or fresh-appliance acceptance.** Anaconda
and Kickstart are not used. No upstream installer patches or OS filesystem
replacements are introduced. Legacy source and canonical artwork remain unchanged.

**Replacement source; rebuilt media still pending:** the USB/VM installer now asks
for a root password and confirmation with **no public-key prompt**, with correction,
Back and pre-write restart controls. Separate commands provide local key enrollment
and private browser setup. The previously delivered ISO still requires a key; these
source changes do not change that ISO. Native x86_64 media build and the complete
fresh-disk journey are deferred until the x86_64 machine is available again.
See the [concrete change plan](coreos-installer-plan.md#manual-install-decision--10-september-2026)
for local-password provisioning, subsequent SSH enrollment and required validation,
and the [publication direction](installation.md#publication-direction) for including
the Soda payload and the recommended QCOW2 download.

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
Its per-file limit is 4 GiB minus one byte; the builder refuses larger individual
payload files before remastering. A larger archive requires a separately verified
packaging change that preserves the upstream primary-name contract.
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
Prepare the sealed native Soda stage or exported bundle for that same revision and
architecture first. The media builder uses the existing bundle verifier/exporter
to snapshot it into ordinary ISO files, then extracts and verifies the final ISO's
bundle again. It does not rebuild or execute the received bundle's programs.
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
  --bundle-source /absolute/sealed/bundle/x86_64 \
  --xorriso /usr/bin/xorriso \
  --out "$PWD/.artifacts/new-installer-attempt"
```

The output `soda.iso` carries its matching console and sealed Soda bundle. Inspect `media-build.json`,
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

## Replacement disk installation interaction

This section describes the replacement source, not the previously delivered ISO.
Use the handoff to distinguish local checks from actual installed behavior.

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
3. Supply hostname (default `soda`), hidden native root/operator password and
   confirmation, and a private project subnet. No public key or file is requested.
   The password is SHA-512-crypt hashed by native OpenSSL over stdin, never argv.
4. Review the shared effects and type **`ERASE /dev/SELECTED_DISK`** exactly.
   Immediately recheck the kernel disk sequence, identity, partition inventory,
   mounts and holders before invoking stock `coreos-installer install --offline`.
   Do not hotplug the selected disk during installation; this is not a hardware
   identity/locking guarantee against arbitrary concurrent privileged changes.
5. The matching media payload must be copied and verified on the selected installed
   root before the console offers media removal. Success means CoreOS and the
   continuation payload are on disk, not that Soda is ready. Remove installation
   media and reboot explicitly. The UI never initiates a reboot.

Invalid field input offers correction. Back revisits the previous step, and
cancellation before disk writing offers an explicit restart using the already
loaded installer. The console service does not restart itself. After an attempted
write, the per-boot marker blocks replay. On an error/interruption, preserve the live boot for inspection:
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

The live copy uses the revalidated selected disk's exact root partition and the
installed OSTree stateroot's persistent `/var`. It measures available space and
requires room for the bundle plus a one-GiB reserve. The initial stock root has not
yet grown on first boot: a large destination disk alone does not prove the payload
fits. Insufficient space stops the copy without custom partition growth. Copy,
verification, labeling, persistence or unmount failures retain the incomplete
attempt and withhold media-removal guidance. Real bundle size, on-media metadata,
SELinux labels and the remove-before-first-boot journey require native proof.

The normal media path uses the bundle copied before USB removal. It requires a
completed media-copy receipt and rechecks its architecture, revision, trusted
checksum and protected contents with the compiled native verifier before executing
**any** bundled program. There is no bundle-transfer, file-path or checksum-entry
question. Missing or partial copy state stops continuation for inspection; it does
not select another directory or download a substitute payload. The existing trusted
component/fixture transfer recipe remains in [installation](installation.md).

After subnet revalidation and explicit **`INSTALL SODA`** confirmation, it calls
existing `install-native.sh`. That script retains package/network/container/identity
preflight, image loading and first-install refusal. Existing or partial Soda state
and a continuation-attempt marker block replay. No live-environment post-install
hook calls the appliance installer; no retry, rollback or old-backup restoration
is provided on failure.

Finally use the separate access and setup steps below. Cockpit stays loopback-first;
neither provider enrollment nor project creation/client routing occurs implicitly.

## Import a laptop key after local password login

At the installed machine's local keyboard/monitor console, log in as root with the
password chosen during installation, then run:

```sh
/usr/local/libexec/soda/soda-install enroll-key
```

Select a reachable private IPv4 address, inspect the displayed native SSH host
fingerprint and explicitly arm the five-minute window. Run the displayed command
on the laptop; it reads the laptop's public `.pub` file and authenticates with the
native root password. Nothing needs to be pasted or typed into the VM/USB console.
The private key stays on the laptop. Compare the host fingerprint before accepting
the new connection; an appliance bridge address alone does not prove reachability.

The dedicated listener uses port 22222, separate from ordinary SSH and Forgejo Git.
It accepts one validated key through a fixed command, with no shell, PTY, forwarding,
SCP or arbitrary command execution. Successful import, local cancellation, timeout
or reboot closes it. Opening another window requires another local action. It
preserves existing authorized keys and asks for a fresh ordinary key-only SSH login
to verify access. Serial, tmux and SSH terminals cannot arm this first candidate.

Existing key files are appended through a validated open inode; a missing key file
is published exclusively. Soda never replaces a concurrently edited key file with
an old snapshot. Detected concurrent edits or uncertain writes preserve the result
for inspection, close the window and require checking native access before another
import. This is not a lock against later changes by a native administrator.

This is a separate stock OpenSSH inetd configuration. Native systemd socket/service
units accept and limit connections and bind their lifetime to the short-lived
enrollment service. Soda has no TCP accept/fork supervisor or password verifier;
its protected local receiver validates and commits the one public key. Runtime
units are exclusively created for the explicit window, never enabled at boot.
It verifies the native shadow password with `UsePAM no`; additional
PAM account/authentication/session policies are not inherited. Ordinary sshd/PAM
configuration is unchanged. This avoids letting PAM session management move a
connection outside the temporary unit's lifetime. Native Fedora OpenSSH, SELinux,
account-policy, timeout/concurrent-connection and real login proof remain required;
local parser/filesystem tests do not establish them. The selected release's
[RPM lock](https://github.com/coreos/fedora-coreos-config/blob/682c839aabbc01564f1605bb41687a7511180031/manifest-lock.x86_64.json)
identifies systemd 259.8 and OpenSSH 10.2p1; the source review covers native
[socket connection limits](https://github.com/systemd/systemd/blob/v259/man/systemd.socket.xml)
and [unit lifetime dependencies](https://github.com/systemd/systemd/blob/v259/man/systemd.unit.xml).

## Configure private browser access

After verifying ordinary key-based SSH, run the following **in the laptop's SSH
terminal**, where pasting the later Forgejo token is possible:

```sh
/usr/local/libexec/soda/soda-install configure
```

Select the appliance's reachable private IP. The guide uses an SSH tunnel for
Forgejo's native installer and administrator account, then asks for a native
operator token with hidden input. It passes that token to the existing `soda-setup`
through a restricted file, never command arguments, and uses the existing
`soda-activate --local-tls`. Partial setup is retained for inspection rather than
creating duplicate OAuth applications. Rerunning after activation checks each
native service and displays the saved access/trust guidance without replaying
setup. The activation marker admits services through their startup conditions;
it is not evidence that they started successfully.

The selected IP becomes the HTTPS browser address. No domain purchase, public DNS,
public certificate service or Internet-facing listener is part of this path.
Use a stable address or DHCP reservation. Caddy's
[internal certificate issuer](https://caddyserver.com/docs/caddyfile/directives/tls)
supplies HTTPS; [automatic trust installation is disabled](https://caddyserver.com/docs/caddyfile/options#skip-install-trust).
Once available, the guide displays the public root certificate fingerprint and an
SSH copy command for **only `root.crt`**, followed by explicit client trust. Never
copy the CA private key. This is private appliance access, not a completed browser
login or project test. Local HTTPS here does not implement Services app ingress.

Finish by signing in to native Forgejo, creating or selecting a repository,
creating/joining its Sodaspaces environment and opening a real browser terminal.
Direct project SSH still requires the separate client route described in
[installation](installation.md#4-establish-real-project-reachability).

## Validation boundary

Local Go/race tests cover inputs, private terminal echo/cancellation, disk inventory
changes/use, pre-write cancellation/marker failures, command-failure redaction,
private file bounds and extension activation. Python tests cover strict conversion,
private network snapshots, live-only customization/readback and fresh-output refusal.
Launcher tests substitute only fixed paths and root/mount/SELinux commands, exercising
real copy/hash/exec and failure handling on synthetic files. A deliberately nonbootable
synthetic EFI ISO tests real xorriso preservation, primary names and tamper rejection;
synthetic BIOS bytes test relocated boot-info/PVD checks. These are not native proof.

Native x86_64 diskless BIOS/UEFI boots reached tty1 through the enforcing-SELinux
guard. Network-editor open/return and cancellation also have bounded native proof.
Static networking, confirmed disk writes, first boot/reboot/continuation and complete
operator setup still require explicit fresh-target/disk approval and native proof. Native aarch64
is independent; cross-compilation or metadata are not installed evidence. See the
[handoff](implementation-status.md) for actual checks and unrelated packaging failures.
