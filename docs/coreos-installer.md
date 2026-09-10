# CoreOS installation media

Source implementation of the [installer plan](coreos-installer-plan.md), **not yet
built/booted ISO evidence or a fresh-appliance acceptance result**. Anaconda and
Kickstart are not used. No upstream CoreOS Installer patches or custom OS image
are introduced. The legacy repository and canonical artwork remain unchanged.

## Owners and prerequisites

- `scripts/build-installer.py` builds the native Go console and artifact verifier,
  fetches/verifies the selected upstream ISO, converts public Butane configuration
  and invokes stock `coreos-installer iso customize --live-ignition`.
- `appliance/locks/coreos-iso.json` records actual release metadata for both native
  architectures, separate from the existing QEMU lock. Both selected RPM inventories
  report CoreOS Installer **0.26.0**; customization and runtime version checks require
  that version. The source-backed wrapper/readback contract is version-specific.
- `appliance/installer` / `internal/installer` own the interactive disk adapter and
  explicit installed-host continuation. The media-only command stays outside the
  runtime `cmd/` staging loop: the application bundle must not overwrite a running
  bootstrap executable. Ignition delivers it; no media build becomes an application
  build prerequisite.
- `scripts/render-provisioning.py::public_config()` remains the public bootstrap
  owner. Butane strictly converts FCOS 1.6.0 to Ignition 3.5.0 at media-build time.
  The live Go adapter adds only validated per-machine fields to that template;
  Python and Butane are **not required on the live OS**. The standalone renderer
  also accepts `--appliance-hostname`; fixture `--hostname soda-native-*` is unchanged.

### Why the console executable is a separate payload

The upstream ISO has a **256 KiB Ignition embed area**, not space for a Go executable.
The ISO embeds a small live configuration that fetches the exact generated public
console binary through **HTTPS with an Ignition SHA-256 verification hash**. The
URL is revision/architecture-specific. The builder emits that file under `payload/`
but never hosts or publishes it. Use an existing operator-selected HTTPS location
with ordinary trusted certificates; no token in the URL or trust bypass.

This is a **network-assisted installer**, not an offline Soda ISO. DHCP must make
that HTTPS payload reachable before the console can start. A static-only network
can use the explicitly supplied private NetworkManager keyfile below. Later Soda
bootstrap also needs network access for RPM/repository dependencies. CoreOS disk
writing itself uses the full ISO's offline image and never falls forward to a newer
stream download.

## Build recipe (not installation permission)

Use a clean exact-revision checkout on matching-native Linux, the repository-pinned
Go toolchain, native `gpgv`, an independently trusted Fedora keyring/full signer
fingerprint, CoreOS Installer 0.26.0 and a Butane executable supporting FCOS 1.6.0.
Butane's observed version and executable hash are recorded, not an invented tool lock.
No tool installation, key import, host service or publication follows automatically.

Create a new output below an existing real `.artifacts/` parent:

```sh
python3 scripts/build-installer.py --arch x86_64 \
  --butane /absolute/tools/butane \
  --coreos-installer /absolute/tools/coreos-installer \
  --keyring /absolute/trusted/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --payload-base-url https://operator-selected.example/soda-media \
  --out "$PWD/.artifacts/new-installer-attempt"
```

The URL is an operator input/example, not a provided Soda hosting service. Before
booting, place the exact file printed by the builder at its matching URL, under
separately authorized hosting scope. Distribute the matching `soda.iso`, not an
unrelated build's ISO. Inspect `media-build.json`, `SHA256SUMS`, `live.ign` and the
upstream signature-verification receipt. Checksums identify bytes; they are not a
Soda release signature, a signed customized ISO or installed acceptance.

For static/pre-Ignition networking, optionally add:

```sh
--network-keyfile /absolute/private/appliance.nmconnection
```

This input must be a bounded regular mode-0600 file, not a symlink. It is snapshotted
in the private attempt and its ISO-embedded bytes are read back and compared.
**That ISO is private per-machine media**, potentially containing network credentials.
Do not publish it as general media. The separately served console binary is still
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
private file bounds and extension activation. Python doubles cover strict conversion,
private network snapshots, live-only customization/readback and fresh-output refusal.
They never exercise real disk writes, network edits, ISO boot or appliance services.

Actual full-media generation, hosted-payload retrieval, tty1 boot, static networking,
confirmed disk writes, first boot/reboot/continuation and complete operator setup
still require explicit fresh-target/disk approval and native proof. Native aarch64
is independent; cross-compilation or metadata are not installed evidence. See the
[handoff](implementation-status.md) for actual checks and unrelated packaging failures.
