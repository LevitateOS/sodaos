# Native build and installation

This guide owns installation, build timing and affected-component maintenance.
**Implement the [single-run replacement milestones](release-engineering-plan.md#single-run-build-replacement-implementation)
first.** Public delivery, unattended scheduling, production readiness and migration
follow B6. The existing writable/ISO recipes below remain operational references
until native cutover, not a second build lane to complete or publish beforehand.
Use the [development handoff](development-handoff.md) for retained installed state
and its scoped grants; [release implementation](implementation-status.md) tracks the
single-run replacement and release commissioning. Use
[local testing](local-testing.md) for recorded access paths.

[Sodaspaces](sodaspaces-plan.md) owns product scope; the [credential guide](dashboard-credentials.md)
owns credential/schema contracts. These procedures use production callers.
[Native support](native-support.md) supplies artifact inspection/bundling and provisioning transport, not a second
installer or product gate. The [CoreOS installer implementation](coreos-installer.md)
now has source and focused local tests for upstream ISO customization, a Go console
and installed-host continuation instead of Anaconda. The console ships on media,
not through a required hosting URL; see the handoff for generation evidence.
Bounded diskless BIOS/UEFI boot checks passed; fresh-disk validation remains unrun.
A prepared Soda QCOW2 is a recommended future download; its producer remains
unimplemented. See
[handoff](development-handoff.md) for actual native build/check limits. The recipes
below describe the retiring sealed-bundle/private-provisioning path. Its bundled ISO
source does not contain a configured appliance or install the derived host candidate.
The new lane must install and update the same immutable candidate through the
[FCOS-native handoff review](coreos-installer-plan.md#b1-native-mechanism-review--reopened);
no direct OCI disk-install capability or replacement update engine is presumed.
No preinstalled QCOW2 is supplied. Do not replay first-install as a service upgrade.

## Publication direction

First produce and qualify the replacement locally; public download/service
commissioning follows B6 using those same artifacts, not another build recipe.

| Artifact | Role in the replacement | Current state |
| --- | --- | --- |
| Host candidate and five application images | One immutable candidate shared by installation and updates; B1 must establish supported host transport | Existing host OCI, local x86_64 candidate and signed Internal GHCR snapshots are scoped experimental foundations, not a selected native handoff or install/update proof |
| SodaOS ISO | Minimal network-install consumer of the signed candidate and prebuilt tools; [media/download contract](coreos-installer-plan.md#selected-media--minimal-network-install) | Image-based handoff unimplemented; older required-key media has bounded boot evidence and password-only writable-bundle source exists |
| Final signed release metadata | Binds tested host/app/ISO identities, compatibility and protected evidence without rebuilding | Delivery primitives exist; integration into the run remains B5 |
| Sealed writable Soda bundle | Retiring product of the old producer, not an output of the replacement | Preserve existing artifacts/maintenance readers; remove competing production at B6 |
| SodaOS QCOW2 | Optional later consumer of the same candidate | No producer or distributable product; not a prerequisite for this replacement |

The [release owner](release-engineering-plan.md#after-the-replacement-operational-commissioning)
owns subsequent public GHCR/ISO commissioning, trust and promotion. Checksums alone
are not Soda release signatures. Generic images/media contain no credentials,
initialized databases or cloned test-appliance state; establish passwords, host keys
and machine configuration per installation. Private-network media is not public media.

The [installer contract](coreos-installer-plan.md#image-based-replacement-contract)
owns the native disk/bootstrap handoff. Preserve the selected password-only text flow,
but replace bundle copying and client-side RPM installation with the signed candidate.
The replacement downloads content during installation under the
[minimal-media target](coreos-installer-plan.md#selected-media--minimal-network-install).
Native proof must establish download integrity and complete local payload availability
before installation finishes. Marketplace networking remains separate.
Historical upstream QCOW2/private-Ignition runtime tests are not image-based ISO proof.

The following commands describe **current retiring component/fixture recipes**.
They remain usable under existing contracts until qualified cutover; do not expand
or publish them as a prerequisite for the single-run rewrite. The proposed
`soda-build` interface is documented in the release plan and is not implemented yet.
No documentation grants disk writes, new fixture lifecycle, uploads or migration.

## 1. Prepare the native builder

Use x86_64 first when access exists; repeat independently on aarch64 later. Install Go 1.26.7, Bun 1.4.2, Python >=3.12 and native Podman through the builder's normal mechanisms. Do not cross-compile/emulate and report native evidence. The root `package.json` pins Bun; run `bun install --frozen-lockfile` at the repository root. See [TypeScript development](typescript.md) for local commands and compiler boundaries.

Soda is a Go API/OAuth command with no embedded or external standalone UI. Both original Go/HTMX and React frontends are removed from source; the read-only Sodaspaces hook/drawer passed its isolated local browser journey, not an appliance install. The standalone React directory, lock, build and external asset payload are removed; all custom Cockpit frontend/workspace output is also retired; stock branding is staged independently. New bundles reject retired SPA payloads. Old installed bundles/evidence retain their original verifier and source revision. Dependency/build/deployment actions still require their applicable scope.

Real Go dependency metadata was resolved during the first native x86_64 build and is now in `go.mod`/`go.sum`. For intentional dependency changes, run `go mod tidy` and review the resulting metadata. The root Bun lock records the remaining shared tools/Lit analysis workspace; unused custom Cockpit dependencies are removed. Then invoke the admitted `soda-build` controller for a development candidate or
production run as documented in [native support](native-support.md#local-host-content-image-candidate).
The legacy `scripts/build-native.sh` / `soda-host-image --legacy-native` lane was
removed at B6. Verify a produced candidate without rebuilding:

```sh
bash scripts/check-native.sh x86_64 /ABS/PATH/TO/artifacts
```

Use a clean exact-revision checkout with a fresh soda-build output parent; another attempt requires a fresh output directory, not global artifact removal. soda-build freezes source, builds application/host archives once, and seals candidate/media identities. No publication, install, VM or product test follows automatically unless P9/P10 configs are admitted.

The companion is built with `appliance/tailnet.Containerfile`: the immutable
upstream `tailscale/alpine-base` plus official release archives pinned by SHA-256 in
`appliance/locks/tailscale-image.json`. Only the native CLI/daemon are extracted; the
helper invokes them directly, not `containerboot`. Build metadata refuses a CLI or
daemon version differing from the lock. This preserves the selected release when
upstream has published its binaries but not a matching container tag.

Run `scripts/check-native.sh x86_64 /ABS/PATH/TO/artifacts` separately against a
soda-build candidate. Export with `tools/soda-artifacts bundle` when that tool is
present in the candidate or a retained sealed stage, as shown in
[support recipes](native-support.md#build-and-artifact-contract); do not transfer the entire build tree.

## Build timing and progress implementation plan

**Target contract; current implementation is transitional.** The owner rejected
`d054a60`'s extraction as final architecture and requested the
[single-run replacement](release-engineering-plan.md#single-run-build-replacement-implementation).
This section owns timing/reporting behavior to preserve from `2166333`, not its
Python/shell implementation. The new Go controller owns the only run clock and
process supervision. Existing invocation/output details below describe the retired shell/Python
implementation removed at B6; they are historical. The Go controller owns the only
run clock and process supervision.
Actual native x86_64 host-context preparation at `d054a60` also passed through the
new shared compiler/timing bridge; see the [receipt](implementation-history.md#shared-build-production-and-timing-consolidation).
It did not build application/host images or an ISO. Full native build/installation
evidence remains separate.

### Run the timed build

**Retiring commands below, not the new interface.** `soda-build` now implements
P1–P6; [current invocation/effects](native-support.md#local-host-content-image-candidate)
include its unsigned-candidate boundary. The full release contract remains in the
release plan; media/qualification/signing are not yet connected.

With the [native builder prerequisites](#1-prepare-the-native-builder), committed
source, existing `.artifacts` parent and a fresh native output location:

```sh
bash scripts/build-iso.sh --arch x86_64 \
  --butane /absolute/path/to/butane \
  --coreos-installer /absolute/path/to/coreos-installer \
  --xorriso /usr/bin/xorriso \
  --keyring /absolute/path/to/fedora-signing-keyring.gpg \
  --signer FEDORA_SIGNER_FINGERPRINT \
  --out "$PWD/.artifacts/iso-x86_64-attempt"
```

Substitute the actual trusted tools/keyring/fingerprint from the existing
[ISO recipe](coreos-installer.md). The output and timing log must be new: the example
produces `iso-x86_64-attempt/soda.iso` and the sibling
`iso-x86_64-attempt.timing.log`. Logs are mode 0600, contain only progress records,
and remain outside sealed payloads. Existing outputs are never cleared or replaced.
The native payload remains at `.artifacts/native/x86_64`.

`build-native.sh ARCH` also reports checkpoints when run alone, with its log at
`.artifacts/native/ARCH.timing.log`. The existing standalone `build-installer.py`
command uses `--bundle-source` and reports an ISO-only total. `--build-native` and
`--bundle-source` are mutually exclusive; the shell entrypoint selects the former.
The release-image command is `tools/soda-host-image --build --complete`; its invocation
is documented in [native support](native-support.md#local-host-content-image-candidate).
It calls the same Go producer and emits `<attempt>/timing.log` plus a separate
`build.log`. It does not call the legacy assembler or build an ISO.

The retiring shell/media lane retains its Python timing helper. Go progress now uses
a native monotonic clock, with compatible inherited timing only for the legacy child.
The new controller invokes no Python clock/supervisor; it reuses the existing Go
process-group owner and has its own single run clock and final outcome. The shell retains `errexit`, and captured
image IDs/tool versions remain separate from progress stderr. A subprocess group forwards Ctrl-C/TERM
to the active build and descendants, allowing five seconds for shutdown before
forcing a timed-out group to stop. No monitoring service or dependency was added.

**Local validation:** 11 timing/wrapper tests passed, including both interrupt
signals, descendant shutdown, failure exit codes, existing-output preservation,
standalone timing and native-to-ISO handoff. Existing ISO fixtures passed for both
public and private-network paths (19 tests, one optional real-xorriso test skipped);
metadata/activation tests passed (5 tests, two optional Caddy tests skipped), as did
Tailnet image checks (2) and Spaces staging checks (7). Shell syntax, Python compile
and diff checks passed. Mac staging tests used the real `/private/tmp` parent rather
than the system's symlinked temporary path. These are local fixtures/process tests,
not a new native image/ISO build or boot/install receipt.

### Outputs covered by the build

The ISO is the final output of this selected path, not its only product. Progress
and the final summary must identify the other retained artifacts as well.

| Entry point | Outputs |
| --- | --- |
| `build-native.sh` | Legacy admission/lock and sealing around `soda-host-image --legacy-native`. The common Go producer emits programs/support tools, browser/terminal/Tea assets, staged files and five OCI archives. Legacy Forgejo/Caddy remain upstream images. |
| `build-installer.py` | Installer console and verifier, verified upstream ISO, on-media bundle snapshot, intermediate remastered ISO, final `soda.iso`, configuration, readback evidence and media metadata/checksums. |
| `build-iso.sh` | Runs both existing phases and reports their retained outputs and timings. The name describes the requested final target; it does not imply that only an ISO was produced. |
| `soda-host-image --build [--complete …]` | Existing experimental host-image assembly using that same producer: vendor-tagged programs, immutable Forgejo presentation, host OCI archive, application archives, payload/candidate records and wall timings. No legacy assembly, ISO, signing or publication is implied. |

The final timing summary lists the sealed native payload, its application-image
archives, the final ISO and the timing log. Give native production its own subtotal,
so the cost of producing these independently useful artifacts remains visible.

### Intended terminal experience

Print a flushed message before each section starts, and another when it finishes.
Every completion line includes that section's elapsed wall time and the elapsed
build total. Name the actual component during repeated work, such as Project OS,
dashboard, Forgejo or Caddy. Keep normal command output available under the existing
logging rules; do not print raw command arguments or expose currently suppressed
private provisioning/network input.

Illustrative output only; these are not measured build times:

```text
BUILD   Soda ISO · x86_64 · revision abc1234
START   Native / Compile Soda programs
DONE    Native / Compile Soda programs       section 00:01:12 · total 00:01:20
START   Native / Fetch upstream Tea binary
DONE    Native / Fetch upstream Tea binary   section 00:00:04 · total 00:01:24
START   Native / Build Project OS image
DONE    Native / Build Project OS image      section 00:03:18 · total 00:04:42
...
START   ISO / Verify final image
DONE    ISO / Verify final image             section 00:00:38 · total 00:09:51
SUCCESS Soda ISO ready                      total 00:09:51
OUTPUT  <attempt>/soda.iso
```

A checkpoint is a progress message, not a saved execution state or resume feature.
No estimated percentage or ETA is needed. Start/end messages and the normal tool
output are sufficient for this pass; no background monitoring process is required.

### Timing and failure contract for the replacement

- One Go controller uses native monotonic elapsed time, starting before preflight
  and ending after the last requested phase. No Go-to-Python clock subprocess,
  nested coordinator, duplicate supervisor or timing daemon. Preserve the familiar
  plain `START`, `DONE`, `FAILED`, `CANCELLED` records and `HH:MM:SS` durations.
- Measure each significant section and phase, including copies/hashes, downloads,
  waits, media readback, native qualification and requested publication. Total and
  phase intervals are measured independently, not summed from overlapping children.
  The [release owner](release-engineering-plan.md#single-run-release-build-contract)
  defines phases and qualified/published/held outcomes; timing never changes authority.
- Progress goes to stderr and a restricted progress-only log under the run's `logs/`.
  Captured image IDs and native tool stdout remain intact. Flush before/after each
  section. Retain redacted tool diagnostics separately; no credentials, raw private
  provisioning, passphrases or private network contents enter progress or evidence.
- Preserve the original failed exit status, identify the active failed section,
  stop later phases and retain completed timings. Interrupt/termination stops and
  reaps owned descendants; timeouts fail rather than extending themselves. Emit one
  final outcome only, with retained paths. Uncertain publication is held, not retried.
- Fresh output guards remain. Never clear artifacts/caches or create a worktree for
  timing convenience. Record architecture/revision and explicit digest reuse; label
  unmeasured cache state unknown. No fabricated cold-cache benchmark or ETA.
- Port the existing timing/capture/cancellation assertions to Go; use controlled
  clocks and short process doubles, not exact real-time expectations. Cover failed
  preflight, child/grandchild cancellation, log failure, occupied paths and secret
  suppression. Confirm logs are excluded from immutable image/media payloads.
- Finally record one real native local full-run receipt through installation/recovery
  and signed final metadata, with phase and total timings. Record actual publication
  timing separately when its downstream commissioning is selected. Source tests, host-context
  preparation and media generation remain separately scoped evidence until then.

### Current implementation inventory (retiring)

The shared producer, not a second build graph in documentation, owns execution
order. Each significant operation gets a named start/end checkpoint; repeated
program/image work names the actual component. The media wrapper additionally
reports native/ISO phase durations and the overall total.

**Shared production — [production.go](../internal/nativebuild/production.go)**

| Section | Work included / source boundary |
| --- | --- |
| Compile each program | One pinned `go build` and ELF/mode verification per command. Vendor uses `soda_host_image` and archive-safe VCS mode; legacy uses its existing writable layout. |
| Check frontend toolchain; verify Go dependencies | Manifest-owned Bun version and `go mod verify`. |
| Install dependencies; build frontend assets | Frozen Bun install, `build-forgejo.ts`, selected browser modules/inventory/output writes. |
| Fetch terminal assets; prepare Forgejo translations; fetch Tea | Existing checksum/license/architecture/staging owners, each timed individually. |
| Stage appliance files | `stage.py`: programs, services, configuration, branding, templates/assets/locale and public modes. This precedes both layouts' image builds. |
| Pull and resolve Rocky base | Same selected base for dashboard/Project OS, platform/digest resolution and provenance recording. |
| Build then export/verify each image | Dashboard, Project OS, Forgejo, proxy and Tailnet in producer order. Each export immediately verifies OCI bytes, platform, config identity and applicable revision. |
| Forgejo layout-specific assembly | Vendor builds immutable presentation into the image; legacy exports upstream Forgejo for its writable installation paths. |
| Proxy naming | Same upstream component; `caddy.oci` remains the legacy bundle filename and `proxy.oci` the release filename. |

Command names come from `SodaCommands`, not another maintained list. Support tools
remain outside runtime `cmd/` admission.

**Legacy-only assembly — [build-native.sh](../scripts/build-native.sh)**

Admission/toolchain/clean-source checks, checkout lock and fresh producer output;
compile the shared Go producer; produce the legacy components; collect/inspect
native inputs using `native-build-info.py`; confirm unchanged source; seal the
payload. For a combined ISO build, snapshot the just-produced verifier for reuse.
Each boundary remains timed. Existing outputs and the producer executable are
retained rather than overwritten or cleared.

**Experimental host-image assembly (retiring) — [soda-host-image](../tools/soda-host-image/main.go)**

Admission; freeze committed source with `git archive`; prepare host context;
compile vendor programs once; lock host package inputs; common asset/image
production and immutable Forgejo assembly/inspection; embed payload metadata and
bound-image selections; inventory context; pull locked CoreOS; build, inspect,
export and verify the host; write the detached candidate record. Vendor programs
are copied into the app staging input, not compiled again. Read-only content/ELF/
OCI checks are artifact inspection, not installed qualification.

The Project OS image interval includes Rocky package installation, the signed GitHub
CLI RPM, podman-compose, the upstream mise binary, copying Tea and Soda runtime files,
and final OS/service preparation. Dashboard includes CA-package installation and
copying its binary. Keep Podman's normal step output visible within these intervals.
These are whole-image durations, not separate measured package-install durations;
do not parse/reimplement Podman's build steps or change Containerfile layers just
for timing. Tea and Tailscale are downloaded binaries, not source compilations.

**ISO production — [build-installer.py](../scripts/build-installer.py)**

| Order | Terminal section | Work included / source boundary |
| --- | --- | --- |
| I01 | Check ISO build prerequisites | Native architecture, unchanged source, tool versions, selected ISO inputs, canonical output validation and fresh directories. The outer preflight catches obvious missing inputs earlier; retain the builder's own checks. |
| I02 | Verify installer Go dependencies | `go mod verify`. |
| I03 | Compile installer console | Build `appliance/installer` into the on-media `soda-install`. |
| I04 | Reuse or compile ISO artifact verifier | Combined native/ISO builds reuse the freshly produced, owned verifier snapshot. Standalone ISO builds compile from trusted source, never from the supplied bundle. |
| I05 | Download and verify CoreOS ISO | `fetch-coreos-iso`: input validation, ISO download/checksum, signature download/verification and verified-input record. This is one combined interval around the existing command. |
| I06 | Generate destination configuration | Load public provisioning configuration and convert it with Butane. |
| I07 | Snapshot native bundle onto media payload | `snapshot_bundle`: the existing `bundle` command verifies/copies/seals the admitted payload; hash its resulting manifest. This includes real archive copying and verification, not just passing a path. |
| I08 | Prepare console payload and live configuration | Artwork, licenses, modes, console hash, live configuration generation and the second Butane conversion. |
| I09 | Prepare boot-menu files | Inside `remaster`: inspect upstream ISO entries, read and brand available EFI/BIOS configuration. BIOS-specific work is absent on aarch64. |
| I10 | Write remastered ISO | The `xorriso` invocation in `remaster`, including payload mapping and boot-equipment replay; preserve its existing `remaster.log`. |
| I11 | Prepare private network configuration, if supplied | `snapshot_network`: validate and copy the private input; never show its contents. |
| I12 | Customize ISO | `coreos-installer iso customize` embeds live Ignition and optional networking. |
| I13 | Extract bundle from completed ISO | The extraction command in `verify_bundle_readback`. |
| I14 | Verify extracted bundle | Manifest comparison and `soda-artifacts verify` in the same helper. |
| I15 | Verify private network readback, if supplied | Extract and compare the optional network configuration. |
| I16 | Verify embedded live configuration | Show embedded Ignition through captured output and compare it with the generated configuration. |
| I17 | Verify ISO files and boot structure | `verify_remaster`: file inventories, ownership/modes/links, boot metadata, preservation hashes and payload hashes. No separate timer per file. |
| I18 | Verify live kernel arguments | Compare upstream/final kernel arguments and write the ISO inspection record. |
| I19 | Confirm unchanged source | Final source check before writing the completed media record. |
| I20 | Write final metadata and checksums | Hash completed outputs, write `media-build.json` and `SHA256SUMS`. Includes the existing repeated hashing of large files; total timing must continue through it. |

The inventory above describes the retiring callers, not another implementation
checklist. Their historical native/ISO and image-build totals do not include a
connected qualification/publication run. The replacement's completion criteria
are the timing contract above and the owning release plan's B1–B6 exits.

## 2. Provision the upstream host

Candidate: Fedora CoreOS stable 44.20260817.3.2. Use its upstream installer/image matching the target architecture and native install instructions. Do not add a separate Soda distribution/release pipeline.

For the on-media console recipe, see [CoreOS installation media](coreos-installer.md).
It reuses the bootstrap/bundle/setup contracts below; it is not an offline appliance
image or permission to reinstall an existing host.

Produce private provisioning input using operator-selected files:

```sh
umask 077
scripts/render-provisioning.py --operator-key-file /secure/operator.pub \
  --root-password-hash-file /secure/root.hash --out /secure/soda.bu
/path/to/soda-artifacts convert-butane --arch x86_64 \
  --source /secure/soda.bu --out /secure/soda.ign
```

Use a real mode-0700 parent and new absolute output paths; private inputs/outputs are restricted regular files. The public bootstrap is `appliance/provisioning/base.json`. For fresh QEMU fixtures, the [support guide](native-support.md#fresh-vm-contract) adds a unique hostname and a pre-pinned per-instance SSH host key, verified against one complete `fw_cfg` Ignition input; do not assume automatic disk/fragment merging.

The root password hash is a native crypt(3) hash supplied by the operator, not plaintext or a Forgejo password. Treat both generated provisioning files as secrets. The input configures only operator host access and a one-time native extension install; it never creates developer host accounts.

Use the generated Ignition file with the upstream `coreos-installer` on the **explicitly authorized installation disk/VM**. Disk selection and permission to erase/install are not supplied by this document. The first boot's `soda-extensions.service` requests Cockpit, Tailscale, native Forgejo runner and their host dependencies through rpm-ostree. Inspect its journal and `rpm-ostree status`, then reboot explicitly to activate the extensions. The service does not automatically reboot or implement a Soda updater.

The Tailscale repository is the upstream Fedora stable repository. Native package closure, especially Cockpit/Forgejo runner/.NET dependencies, must be observed and corrected against the chosen deployment; inherited package source evidence is not installed proof.

## 3. Install the built components

Transfer only the sealed matching deployment bundle through the pinned `soda-acceptance transfer` path or another explicitly approved trusted transfer. Establish its `SHA256SUMS` identity over that trusted channel before executing any bundled program. The inventory identifies each delivered file; it is not a signed release. No publication is needed. Choose a **non-overlapping private IPv4 subnet** for project IPs, for example `10.89.0.0/24` only if suitable for that deployment.

```sh
sudo /path/to/bundle/x86_64/install-native.sh /path/to/bundle/x86_64 10.89.0.0/24
```

This is an explicit first-install operation, not an updater. It stages native files, establishes the dedicated service identity and non-conflicting Podman subordinate range, loads the five verified image archives and restores the existing core service references, and starts loopback-only Forgejo/Cockpit plus the root:soda helper socket and native Tailscale daemon. Fresh host configuration opts into Tailnet management and records the verified companion image ID. No policy/credential is supplied and legacy Create stays Off: installation does not enroll a host/project, create provider runners or activate public browser endpoints. It refuses blind reinstall over existing Soda configuration or a retained `/etc/soda/install-started` marker. Bundle/platform/package/subnet/identity preflight precedes payload writes; partial installation still requires an operator recovery decision, not deleting the marker and retrying as if clean.

Follow [operator setup](operator-setup.md). The native Forgejo installer is initially accessible only over an operator SSH tunnel to port 3000. `soda-setup` uses the resulting operator API token to create its actual OAuth application/configuration. Current source uses one Forgejo/Sodaspaces browser origin with Soda routes under `/-/soda/`; `public_url` and `--public-url` are removed. The isolated browser/proxy proof does not establish deployment to an appliance.

For a private IP browser origin, the replacement source supports Caddy's native
local issuer without an owned domain:

```sh
sudo /usr/local/sbin/soda-activate --bind-ip PRIVATE_APPLIANCE_IP --local-tls
```

The configured HTTPS origin must use that same private IP. Explicitly copy and
verify the appliance's public CA certificate over trusted SSH, then trust it on
intended clients; see the [guided setup](coreos-installer.md#configure-private-browser-access).
Automatic trust installation is disabled. A completed command is not a browser
login or reachability test. Existing certificate-based deployment remains available:

```sh
sudo /usr/local/sbin/soda-activate --bind-ip PRIVATE_APPLIANCE_IP \
  --certificate /secure/browser-cert.pem --private-key /secure/browser-key.pem
```

Activation applies file ownership for the unprivileged dashboard, retains operator-only native access, binds Caddy and Forgejo Git SSH to the selected private IP, and starts the actual services. If Tailnet Git access is intended, enroll through the native operator CLI before activation (or the dashboard Tailnet page once reachable) and select that Tailnet private IP; later advertisement refresh refuses to substitute a Tailnet address while Git SSH only binds a LAN IP. Configured browser origins must resolve through the deployment's normal browser/network setup; this is unrelated to project SSH, which uses project IPs directly. Native Forgejo Git SSH uses port 2222; project SSH uses each project IP's port 22.

First activation also derives the local [Soda avatar provider](avatars.md) from
the configured Forgejo origin. Enable provider avatars and disable federation
through Forgejo's native administration settings. Saved database-backed settings
and explicit offline mode are not silently overridden by activation.

### Sodaspaces customization delivery

The earlier merge's partial-payload hold is replaced in source by
`internal/nativebuild/forgejo-payload.json`: an exact source/destination inventory
shared by staging and the embedded Go verifier. It includes all 229 selected template
overrides, shared presentation assets/fonts/notices, the mounted Sodaspaces content/
terminal and five locked renderer/CSS/MIT-notice files, beneath
`/var/lib/soda/forgejo/gitea/`. See the [mounting contract](terminal-integration.md).
Adding an arbitrary template is still refused; expand the reviewed inventory explicitly.
Clean native build/check/export passed at `dad2945`, including public-mode
normalization for private checkouts. The preceding `2aa4960` application/helper
passed bounded installed integration on the isolated fixture; see the handoff for
its exact bytes and recorded legacy public-mode differences. The newer whole
bundle is not an installed-appliance or retained-rollout result.

The build fetches locked terminal distributions into `terminal-assets` and verifies
Forgejo 15.0.7's complete English catalog via `appliance/forgejo/locale.lock.json` before
adding the Soda-only namespace into `forgejo-locales/locale_en-US.ini`. It never installs
a partial replacement catalog or downloads at runtime. Stage checks locked renderer
bytes again; bundle verification requires exact files, modes and LICENSE/NOTICE.
Files are 0644, new readable directories 0755. The installer applies UID/GID 1000 only
to the admitted new files/directories, not recursively to a mutable Forgejo tree.
First-install preflight refuses occupied hook/asset destinations, including
symlinks, before host writes. Resolve conflicts explicitly, never merge or overwrite
operator hooks automatically. This is not an upgrade interface. Actual CustomPath,
labels, reload requirements and browser behavior passed bounded native first-delivery
proof at `bdbce8e`; retained-target cutover and whole-appliance acceptance remain separate.

### Existing-state dashboard migration

The current source dashboard requires `grant_key_file` and the
[supported schema](dashboard-credentials.md#tailnet-credentials-and-schema-v10-return),
retaining session-grant encryption. Do not run first-install or OAuth bootstrap again on an existing target.
Follow the [controlled credential migration and rollback procedure](dashboard-credentials.md),
including a consistent SQLite backup, matching config/key/artifact set and
separately approved rehearsal/deployment. A missing or wrong key fails closed;
a prior binary is not assumed compatible with the new schema.

### Retained Sodaspaces cutover
This section owns affected-component maintenance, not first-install replay or a
fixed historical v3→v5 recipe. Consult the [handoff](development-handoff.md) for
actual installed versions/grants and [AGENTS.md](../AGENTS.md#permissions-and-preservation)
for execution policy. Assess only the change's real compatibility and interruption
effects; do not assume an old target inventory or replay completed maintenance.

1. Identify the exact target, running image IDs/effective units, schema, private
   inputs, customizations and affected project/runner/session state. Select the
   minimal compatible delta and declare its interruptions. A bundle's project
   images or package defaults are not automatically part of that delta.
2. Before replacing active management writers or stopping shared services, close
   their affected admission paths and drain actual pending writers. Replacing a file
   does not stop an old process. Follow the [terminal shutdown contract](terminal-integration.md#managed-terminal-implementation-and-proof-limits)
   for the actual deployed representation (old guard-owned sessions are not silently
   converted to native ownership) and the [runner compatibility contract](runners-port.md#paired-artifact-compatibility)
   for shared CLI/web writers; do not silently stop ordinary workloads or jobs.
3. Take appropriate fresh consistent backups of affected state and matching
   config/key/artifact/unit/custom-file inputs. The [credential migration contract](dashboard-credentials.md#controlled-existing-state-rehearsal-before-live-deployment)
   owns SQLite/key/schema rehearsal and compatible restoration. Reuse applicable
   compatibility evidence; copied state must not authenticate grants or start cloned
   listeners with live credentials. Preserve metadata and operator customizations.
4. Publish only the approved paired changes. Verify the dashboard OCI and actual
   image pin, not just its separately staged executable. Required project-local
   program changes follow [same-root maintenance](project-os.md#deliver-required-additions-without-replacing-roots); do not replace
   roots. Do not rerun setup/activation, regenerate keys/OAuth applications or alter
   callbacks/configuration merely because a historical recipe did so. Any required
   upstream setting change uses its supported owner interface and declared scope.
5. Restart only affected approved services after compatible inputs are ready.
   Verify running bytes, applicable migration/preservation outcomes, native routes
   and protected page/access paths. Keep original accounts, roots, credentials,
   runner state and unaffected services. Use existing actor credentials and the
   declared client route; a failed observation does not justify adding keys,
   projects, permissions or network changes.
6. Record completed versus unconfirmed effects and preserve failed/partial state
   plus later writes. Restoration requires a compatible matching set and a
   later-write preservation decision under the credential contract; never lower a
   schema marker or replay a mutation to fix an observer.

### Cockpit addon maintenance

The [Cockpit guide](cockpit-port.md#selected-root-administration-baseline) owns the
selected page/package baseline. On an authorized existing appliance, use its native
rpm-ostree transaction, not the first installer or application setup again. Inspect
pending deployments and actual workloads, preserve affected configuration (including
PAM, certificates and any navigation override), and preview the package delta.

For a purely additive transaction, the selected native rpm-ostree may support
`install --apply-live`: it records the next-boot deployment and adds files to the
running system without a reboot. The [fresh fixture receipt](implementation-history.md#native-cockpit-administration-additions)
exercises that native path. Do not add `--allow-replacement`, force file replacement,
reset a deployment or reboot as an unreviewed fallback. If replacements/removals or
an activation interruption are required, assess the concrete transaction and its
authorization/preservation scope first. Fresh provisioning retains its explicit
extension-activation reboot; live maintenance is not an implicit bootstrap change.

Verify installed versions, unchanged workloads/boot identity when claiming no
interruption, native page loading and the root-only PAM boundary. Refresh the
operator's login to discover new manifests; avoid restarting Cockpit globally merely
to repair a cached navigation observer. Report generation/upload, policy changes,
container mutations and optional VM/recording/crash-capture setup are separate
operations, not consequences of installing their administration interface.

## 4. Establish real project reachability

The implemented profile is a native routed Podman bridge (`soda0`) on the appliance. Host-to-project access is through that bridge; developer clients need a route for the chosen project subnet via the appliance. Set that route on the deployment's LAN router, or use a native Tailscale subnet route with the required Tailnet administrator approval. Respect existing firewall policy and authorize only the intended private ingress/forwarding. No project DNS, SSH gateway or extra identity authority is required.

Host Tailnet enrollment by itself does not route the project subnet. Port-forward-only access to a builder VM is not proof that real developer clients can reach project IPs. Verify the actual routing/firewall setup in the product journey; do not call a Podman-only address usable because it appears on the dashboard.

## 5. Operator services and state

Cockpit initially binds loopback 9090; use an operator SSH tunnel unless private native access is deliberately configured. Its PAM policy permits only root. Tailnet and local Runners management belong to the configured
Soda operator at `/admin?soda-view=tailnet` and `/admin?soda-view=runners`. Tailnet delivery follows
[its plan](tailnet-integration-plan.md#stage-6--affected-retained-delivery-then-cockpit-retirement);
installed runner fallback removal follows the
[combined retirement gate](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation). The dashboard runs as native service UID/GID 2000 with no host capabilities and reaches only the restricted project helper socket; secret files are root:soda 0640, the database directory soda-owned 0700. The helper is root and exposes only fixed project operations over that Unix socket, never a public control listener.

Project environments are created once and started/stopped as existing containers. Do not run `podman system prune`, delete project containers, use `--rm`, or replace their writable roots as ordinary management. That root stores account records, installed packages/tools and service data. Backups/recovery and automatic image replacement remain deferred.

Required new native support, such as tmux, follows the [Project OS same-root
maintenance contract](project-os.md#deliver-required-additions-without-replacing-roots).
New image builds/defaults do not update retained roots. The handoff records one
exact isolated-root tmux transaction with native signature/dependency checks and
bounded browser proof; the user waived backups only for that target. Other
package/file/service transactions still need their own compatibility review and
applicable target/action scope; first-install/activation are not those recipes.

Run the later [native validation guide](native-validation.md) only with explicit targets and permissions. Installation/activation command success is not validation. Keep source, build and installed evidence separate.
