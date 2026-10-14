# Native support tools

Outside build, transport, VM and evidence tools for Soda development. They supply
artifacts and observations; they do not redefine product API, Project OS or Spaces
contracts.

Permissions follow [AGENTS.md](../../AGENTS.md#permissions-and-preservation).
This document is not an execution grant.

Release model: [Release architecture](../architecture/release.md).
Day-to-day release commands: [Release workflow](release.md).
Media: [Installation media](../guides/media.md).
Notices: [Notices](../research/notices.md).

## Shared contracts and provenance

| Interface | Owner / implementation boundary |
| --- | --- |
| `build-native.sh` / `build-iso.sh` / `soda-host-image` | Use `soda-build` (`lib/soda-release-tools`). |
| `check-native.sh ARCH CANDIDATE_DIR` | Verifies a soda-build candidate artifacts directory; does not build. |
| Containerfile `BASE_IMAGE` argument | The build pins the existing Rocky reference to its resolved native digest reference during that build; unchanged default, no base upgrade or frontend change. |
| `soda-render-provisioning` (`tools/release-assets`) | Public `system/host/provisioning/base.json` and shared host-branding assets plus private per-instance inputs. Existing extension bootstrap remains the default; `--bootstrap minimal` is a fixture-only alternative without package installation. |
| `soda-artifacts` (`lib/soda-release-tools`), `soda-acceptance` (`rust/soda-acceptance`) | Separate native tool output, never appliance `cmd/`, rootfs or container payload. Candidates carry them for archive inspection and media-input conversion, not as installed programs. |
| Installed checks | Host/operator observations stay separate from product-owned developer/shared-tools/workload/persistence journeys. Old standalone browser harnesses are removed; the read-only native-page journey and exported-payload checks passed at their documented local scope. Current `832dfa16` development-appliance results and explicit task-built overlays are recorded in the implementation plan and ignored receipt; they do not qualify fresh media or a release. |

Reuse and licensing are recorded in [native support notices](../research/notices.md). The predecessor checkout remains separate and preserved; `scripts/test-vm.sh` is superseded by the `soda-test-vm` Rust port.

## Effects and permissions

This section owns command effects, not an approval queue. Apply the
[execution policy](../../AGENTS.md#permissions-and-preservation) and the applicable grants:
[release commissioning](../../AGENTS.md#permissions-and-preservation) or
[development targets](../../AGENTS.md#permissions-and-preservation). Already-approved
local work does not need a new grant per command.

- `exec`: runs exactly the supplied owned check, locally or over pinned SSH. Its selected check determines browser, native and provider effects.
- `native`: one remote `prepare`, `build` or `check` phase. No automatic next phase. Preparation clones the canonical repository into a new private checkout; it never copies laptop binaries/dependencies/state.
- `fetch-coreos`: downloads/verifies/decompresses a public QEMU base into a fresh private cache. No overwrite, key import, VM or installation.
- `fetch-coreos-iso`: downloads/verifies the uncompressed upstream ISO resolved live from the stable stream for the requested architecture, using the same trusted-key/signature boundary, into new `coreos.iso` and `verified-iso.json` outputs. No stored version is consulted. It does not customize, boot, publish or install; soda-build media assembly is the concrete media caller.
- `convert-butane`: runs strict native Butane conversion into a new restricted file. No boot or install.
- `vm`: creates a new KVM overlay/NVRAM/process, boots to pinned SSH, then shuts down. `--restart` explicitly tests one restart of those same files. `--hold` keeps it available for separately authorized checks. The selected Ignition may install extensions: that requires installation permission as well as boot permission.
- `probe-ssh`: direct Git-endpoint key exchange using an existing trusted pin, without credentials, proxy, keyscan or Git authentication.
- `report`: reads/cites existing observations and verifies their retained-file hashes. No native work, retry, product verdict or publication.

The existing build now pulls the unchanged selected Forgejo/Caddy references for the matching platform and runs fresh, network-disabled, read-only image-inspection containers for RPM/Tea/gh version metadata. These are build inspection containers, **not project containers**. Include those operations in a build grant. The candidate embeds those exact images for native import rather than looking up mutable tags later.

## Build and artifact contract

`soda-build` produces unqualified release media by default. Its explicit development targets stop at candidate or media; `soda-candidate` remains a development wrapper. Publication and qualification belong to `soda-release`.
Retained `.artifacts/native/` trees may still be read by maintenance tools.

Use a fresh exact-revision checkout on matching-native Linux. A dirty checkout, occupied output or simultaneous build is refused; there is no automatic removal of earlier artifacts. `GOTOOLCHAIN=local` prevents implicit toolchain installation.

```sh
# After soda-build writes candidate artifacts:
bash scripts/check-native.sh x86_64 /ABS/PATH/TO/artifacts
```

When a candidate has a `docker/` artifact directory, the existing candidate
checker also verifies its [Docker metadata and actual OCI contents](../../lib/soda-release-deliver/README.md#docker-metadata-and-provenance).
This is read-only artifact admission; Docker production packaging, registry
publication and native qualification remain separate operations.

soda-build candidates carry payload/candidate/OCI archives under `artifacts/`. The payload contains Fountain branding/templates and the separately packaged Soda extension and service, not a standalone React frontend or Go page shells. Mutable package repositories and actual resolved RPMs are recorded, not disguised as pinned/reproducible inputs.

The disk installer verifies candidate media before writing, validates the explicit project subnet before applying it, and refuses to replay an attempted install. Application setup, HTTPS activation and migrations still use core-owned commands. Do not install Soda on the builder.

## Local host-content image candidate

`soda-build` (`lib/soda-release-tools`) owns the shared candidate/media sequence under the
[release contract](../architecture/release.md).
Explicit [development targets](release.md)
stop this same producer at the requested boundary; they never qualify a release.
Build from committed canonical source with the pinned Go/Rust/Bun/native prerequisites.
The public entry point now requires an operator-admitted, root-owned executable and
restricted `--worker-config`; it does not install accounts, grant sudo or create a
worker policy. Account/helper installation requires applicable task approval.
The measured development-candidate path took **5m30s warm / 8m29s cold** on the selected
four-CPU x86_64 worker; [receipts](release.md)
state that scope. The optional `--media-compression fast` flag applies only to
`--development --target media`: it selects the upstream EROFS/LZMA level 1,
fragments and 1 MiB cluster profile. It does not qualify production media.

Normal candidate production admits the reviewed upstream EROFS settings, then
freezes LZMA level 6, fragments, global deduplication and 64 KiB clusters into
the host's `image.json`. Native compression hints store existing `.xz` and `.gz`
files unchanged; initramfs remains compressed so its boot/deployment copies can
share storage. The hints ship beside `image.json` and native osbuild reads them
from its existing `deployed-tree` input. No extra stage mount or codec is used.
Complete media, including both installer osmet records, must still fit the
GitHub asset limit and pass the [release qualification](release.md).

The configuration names `Executable`, canonical `Source`, the separate clean committed
`ForgejoSource`, an existing private
`OutputParent` below `.artifacts/releases/`, and `BuildHome`, `Runtime`, `Tools` and
`MediaAuthorityDirectory` (needed only for media). `OutputParent`, `BuildHome` and `Runtime` belong to
`soda-build-worker`; setup stages pinned Go, the repository-selected stable Rust
sysroot under `Tools/rust`, and `Tools/bin/bun`. Cargo dependencies are fetched
with the committed lockfile during setup. The worker builds offline with
`CARGO_HOME` and `CARGO_TARGET_DIR` in its writable build home; source and compiler
binds stay read-only. The worker cache SELinux type permits execution of Cargo's
build scripts and proc macros inside that same sandbox. The executable and its
parents are root-owned and not group/world writable, and its bytes must equal the
running dispatcher. The internal `--worker-build` stage refuses other identities.

The operator runs the admitted executable as root from the canonical checkout:

```sh
# Candidate-only development: no rootfs URL or media authority needed/mounted.
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/UNIQUE --development --target candidate \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT

# Installer development: same candidate producer, then authenticated native media.
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/ANOTHER_UNIQUE --development --target media \
  --rootfs-base-url http://FIXTURE_ADDRESS:PORT \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT

# For faster installer iteration, add --media-compression fast to the media command.
# This selects a distinct development host/ISO, not production qualification.
```

`soda-candidate` (`lib/soda-release-tools`, [operator guide](../guides/soda-candidate.md)) wraps the same admitted controller: one overview screen
shows every choice (mode, output with freshness status, controller, worker
config, fixture URL, protected configs; arch stays pinned to the host),
fields are edited by number with inline validation, and `go` starts only a
runnable config. The run shows a live step table (spinner plus live elapsed
per running phase, controller durations on completion; timestamped log
fallback with `--non-interactive`). Flags only pre-seed answers. The wrapper
never admits workers, signs, or publishes, and has no production mode. Fixture-only
media authority is not production signing authority; release delivery and
publication remain separately owned and grant-scoped as below.

Substitute the operator-admitted absolute paths and explicit URL. Omitting `--development` builds release-candidate media without publishing or installing.
Target flags select development boundaries, and candidate mode refuses media-only inputs. Development
cannot select publication/final signing.
Source commands run only inside the separate systemd worker:
read-only source/tools, selected writable output/cache/runtime binds, CPUs 0–3,
a four-CPU quota, 16 GiB ceiling, and no operator-home or real-release-custody access.

`MediaAuthorityDirectory/config.json` is restricted JSON with `Trust` and `Keys`
(`Key`, `Passphrase`), using worker-visible paths below `/run/soda-media-authority/`.
It and the fixture key inputs belong to the build worker but are bound read-only.
They are not copied into frozen source or command environment. This authority is
**fixture-only**: production release keys must never be supplied to the build worker.
Native identity/custody boundary checks do not establish release readiness; the
public-download installation smoke and promotion remain separate operations. The public URL is explicit and receives a
hash-named rootfs file. A local URL does not make the ISO distribution-ready.

The controller's VCS revision must match the clean checkout. The admitted source and
its committed Git objects must be readable by the isolated build identity; private
untracked inputs are not source inputs and must not be made accessible to fix that.
Each output is fresh;
there is no worktree, partial-host mode, resume or skip-tests flag. The internal image-identity prefix defaults to `ghcr.io/levitateos/sodaos`; it does not
create a registry dependency. User downloads are GitHub release assets. Source and app inputs freeze before compilation. Prepared
source/browser suites no longer gate the build; they consume the same emitted
assets when run directly. Native page/provider journeys still belong to their
separately authorized qualification drivers.

P1–P6 builds runtime programs, the installer console and support tools once; stages
vendor assets directly; builds/exports seven runtime images and the native FCOS host; then
checks ELF/OCI identities, RPM inventory, embedded shared blobs and native Quadlets.
All seven runtime images use ordinary Podman through one shared OCI layout.
Podman exports uncompressed OCI layers so native EROFS can compress and deduplicate
shared files across images once; image configs, diffIDs and filesystem contents
stay identical. The copier preserves these exact exported descriptors.
The host producer uses Podman's `--squash-all` before export and removes the
inherited `/sysroot` encoding cache and `ostree.commit`/`ostree.final-diffid` labels,
following the [upstream plain-image conversion](https://github.com/coreos/chunkah#compatibility-with-bootable-bootc-images).
Runtime files and base provenance remain intact; native OSTree imports the plain
filesystem instead of requiring base chunks that flattening removed.
Obsolete payload
readers and storage selectors are removed. No bootc installer, update engine, bound storage,
lint or sysusers workaround is used, and Zincati is not disabled. Fedora's inherited
packages/bootable-container labels are not a Soda bootc integration.

Outputs occupy `work/`, `artifacts/`, `evidence/`, `release/` and `logs/`.
`artifacts/` contains the eight OCI archives (seven under `images/`), tool binaries and
hashes, payload/candidate identities and frozen app provenance. The once-compiled
installer is also embedded in the host; native checks bind it to the exported tool.
For media requests, floating upstream Butane produces public `destination.ign`
and `live.ign`; its observed version is recorded per build. P7 authenticates the
candidate and packaging inputs using existing Sigstore primitives. P8 invokes the
floating latest Assembler in fresh disposable scratch, extracts minimal media,
customizes it once, and reads back Ignition, kernel arguments and native rootfs
chunk hashes.
Assembler scratch uses its upstream `RUNVM_CACHE_SIZE=75G` setting for the complete
runtime image set. This is a sparse task-owned disk on `/home`; the packaging
container retains its four-CPU and 16 GiB memory limits.
`artifacts/media/media.json` binds the ISO/rootfs sizes, hashes, URL, candidate and
actual rootfs filesystem/options. `image-config.json` is checked against the built
host and included in the authenticated packaging inventory. Media builds also record
`logs/media-events.jsonl`: upstream log-arrival windows, not exact CPU profiles.
The Assembler import checksum is labelled separately from installed identity.
Native Go owns timing
and cancellation, using the existing pinned process-group owner plus controller
subreaping. No raw argv/environment is copied into progress or evidence. Tool logs
are restricted. Packaging owns disposable containers/helper scratch; it does not
prune unrelated stores or modify retained appliances.

`evidence/build.json` records purpose, requested/completed target, candidate hash
(binding its payload and all application identities), host/payload identities and
completed checks. Candidate-only output has no media path, live Ignition or packaging
workspace. All current candidate checks and all seven images remain; media-only tools
and signing inputs are not required for that target.

**Current completion boundary:** explicit development targets exit **0** on success
with `development-only; not release-qualified`. Failures/cancellation remain nonzero.

**Wrapper failure output:** a failed run ends with a `finished in ... with exit N`
line plus a failure panel, on terminals and pipes alike, so the cause is the
last thing the operator sees: `why` names the failed step, `cause` gives the
controller's one-line reason, `log` points at the host build log
(`<out>/logs/build.log`; per-phase detail sits beside it under `<out>/logs/`),
and `hint` suggests the fix when the reason matches a failure the wrapper has
actually seen (stale worker provisioning, offline-module denials, leftover
worker units). An unmatched reason gets no guess: the log path is the whole
advice. Preflight refusals happen before any run and say the fix directly: a
dirty tree lists the files to commit or stash, and a reused `--out` asks for a
fresh directory per attempt.
The builder has no implicit installation or promotion phase. Use `soda-release`
after a completed build, following the [release workflow](release.md). Earlier
failures stop immediately; a build-only task does not authorize publication or
disk installation.
The host install floats on current repositories: bare package names plus the
stable Tailscale repository, with the observed RPM inventory recorded per build
as the bill-of-materials. No lock file precedes the install, for either
architecture. When Fedora supersedes a package set, the install fails against
current mirrors instead of substituting; the recorded inventory names the exact
attempt.

The legacy `soda-host-image --legacy-native` / `build-native.sh` lane was removed.
The [superseded experimental recipes](https://github.com/LevitateOS/sodaos/blob/3fe7f18/docs/native-support.md#local-host-content-image-candidate)
remain historical evidence, not current commands or authority to mutate retained
images. Source history remains available; old experiments do not require current readers.
The [release owner](../architecture/release.md)
defines current storage, defaults and machine-state ownership. The base/RPM inventory
is recorded per build, not locked, mirrored or byte-reproducible. No installed-release claim follows
from local image inspection or ordinary build-cache reuse.

## GitHub release delivery

The [release architecture](../architecture/release.md#public-distribution) owns
public hosting and qualification; the [release workflow](release.md#build-once-stage-install-promote)
provides the concrete commands. The Rust `soda-release` entry point in
`lib/soda-release-tools` delegates to `lib/soda-release-deliver` and the native
`gh` client. It does not build, install an image, change host trust or reboot.

These are the implemented appliance interfaces. The
[Docker release operator flow](release.md#docker-release-operator-flow) specifies
the distribution selector, worktree admission and prepared-input commands to
implement; those planned flags and subcommands are not available in this CLI yet.

| Operation | Inputs and effects |
| --- | --- |
| `stage` | Reads exact candidate/media/source artifacts, prepares a fresh asset directory, checks source CI and per-file size limits, creates or verifies the exact tag, uploads a draft, compares remote hashes and sizes, and publishes an immutable prerelease. Requires GitHub repository write access. |
| `promote` | Binds the observed installed revision and explicit native smoke confirmation to the prepared release and downloaded ISO, writes the receipt, checks unchanged GitHub inventory, then updates notes and prerelease/latest flags. No image re-upload or disk mutation. |

Keep private authority inputs and task receipts on the workspace disk. Only public
artifact files enter the upload set; keys, passphrases, credentials and private VM
configuration must never be published. Native OCI artifact signing remains a local
packaging boundary and uses restricted key/passphrase files. GitHub owns published
asset and tag immutability; no local channel state or publication ledger is needed.

An existing or uncertain publication is inspected rather than overwritten. There
is no automatic asset clobbering or cleanup of failed attempts. A prerelease is
explicitly unqualified until the one bounded public-download native smoke passes.

## SSH, commands and exact-source remote phases

Owned process execution (`exec`, `native`, `vm`) now requires Linux's
non-reaping wait support. The leader remains pinned until its group is terminated,
so an exited parent cannot leave descendants behind or let cleanup signal a reused
PID. Non-Linux coordination can use existing approved SSH tooling; report/metadata
and direct key-probe operations remain portable. This is group ownership, not a
sandbox for a privileged child or one deliberately creating a different session.

Connection JSON uses the Go field names below. Keys are file references, never inline contents. Private identities must be absolute regular files inaccessible to group/others; `KnownHosts` is an already trusted, non-writable-by-others regular file. No automatic trust refresh occurs.

```json
{
  "User": "root",
  "Host": "127.0.0.1",
  "Port": 22222,
  "Key": "/absolute/private/operator-key",
  "KnownHosts": "/absolute/private/known_hosts"
}
```

For a builder, select its actual user/host/port. For a fresh fixture, management is root on loopback only. A QEMU/SSH forward is **not a developer project route**.

```sh
/path/to/soda-acceptance exec --owner P06 \
  --revision FULL_COMMIT_SHA --arch x86_64 --target ACTUAL_GUEST_HOSTNAME \
  --remote /absolute/private/guest.json \
  --evidence /absolute/private/new-host-observation \
  -- env SODA_NATIVE_VALIDATE=ACTUAL_GUEST_HOSTNAME bash -s < tests/installed/host.sh
```

Literal argv quoting preserves empty/metacharacter arguments; stdin is not logged. Prefer an existing owned test script over ad hoc shell assertions. For an expected denial, check the **specific native result and evidence result**: an SSH error is not an access-control pass.

`native` takes a restricted request file with exactly these fields:

```json
{
  "Revision": "FULL_40_CHARACTER_COMMIT_SHA",
  "Architecture": "x86_64",
  "Target": "ACTUAL_BUILDER_HOSTNAME",
  "Work": "/absolute/existing-parent/new-run",
  "Phase": "prepare"
}
```

The target is the machine's actual hostname, not an assumed DNS alias. Supply matching `--revision`, `--arch`, `--target`, `--owner P02`, `--remote`, `--request` and a **new** `--evidence` directory. Later requests change only `Phase` and reuse the binding. `check` requires completed `build`. Each phase is exclusive, verifies the unchanged clean checkout, and retains failed work. No automatic retry or revision switch.

Remote commands are bounded with native `timeout` plus a 10-second termination allowance. Closing SSH is **not proof that a remote process has already stopped**; on interruption, retain the failed observation and inspect the authorized target before retrying. Core tests/provider actions own their resource cleanup. Source tools may be built on a separately authorized client for coordination; a client tool build is never appliance-architecture evidence.

## Verified CoreOS and private provisioning

The stable stream's QEMU inputs resolve live per architecture; no stored version
is consulted. The caller must independently select a trusted Fedora keyring and
full signer fingerprint (see [Fedora security](https://fedoraproject.org/security/)). No key is fetched/imported as a trust shortcut.

```sh
/path/to/soda-artifacts fetch-coreos --arch x86_64 \
  --keyring /absolute/trusted/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --out /absolute/cache/new-coreos-attempt
```

Both compressed/uncompressed hashes and the selected GPG signature must match. Success writes a read-only `coreos.qcow2` and private `verified-base.json`. A failed attempt is retained and cannot be overwritten. This is a fixture base, **not a preinstalled Soda disk or selected media deliverable**.

Prepare one private directory, a fresh Ed25519 **host** key, the operator's existing public authentication key, and a private crypt password-hash file. Host and operator keys are different identities. Host-key generation and conversion are explicit actions; never put passwords/hashes/key contents in shell arguments or logs.

```sh
cargo run -p soda-release-assets --bin soda-render-provisioning -- --bootstrap minimal \
  --operator-key-file /absolute/private/operator.pub \
  --root-password-hash-file /absolute/private/root.hash \
  --ssh-host-key-file /absolute/private/instance-host-key \
  --hostname soda-native-example --out /absolute/private/instance.bu
/path/to/soda-artifacts convert-butane --arch x86_64 \
  --source /absolute/private/instance.bu --out /absolute/private/instance.ign
```

The output parent must be real and mode 0700; outputs are exclusive 0600. `extensions` instead of `minimal` retains the existing public package/repository/unit bootstrap, without an automatic reboot. Butane diagnostics can echo secrets and are deliberately not retained.

Build `known_hosts` from the **trusted public half of that freshly generated host key** for `[127.0.0.1]:PORT`; do not keyscan the guest. The VM preflight verifies the pinned key and hostname against the supplied Ignition. Use exactly one complete private `fw_cfg` document; external Ignition merge/replace is rejected rather than guessing how disk config composes with it.

## Fresh VM contract

Use actual native executable/firmware paths from the selected builder; resolve symlinks first. QEMU and its matching UEFI CODE/VARS pair are operator-selected native inputs, recorded by version/hash, not bundled firmware or an emulation fallback.

```json
{
  "Name": "soda-native-example",
  "Architecture": "x86_64",
  "BaseReceipt": "/absolute/cache/new-coreos-attempt/verified-base.json",
  "Ignition": "/absolute/private/instance.ign",
  "QEMU": "/absolute/native/qemu-system-x86_64",
  "Firmware": "/absolute/native/OVMF_CODE.fd",
  "Variables": "/absolute/native/OVMF_VARS.fd",
  "Work": "/absolute/private/new-vm-work",
  "DiskGiB": 64,
  "SSH": {
    "User": "root", "Host": "127.0.0.1", "Port": 22222,
    "Key": "/absolute/private/operator-key",
    "KnownHosts": "/absolute/private/known_hosts"
  }
}
```

Invoke `/path/to/soda-acceptance vm --owner P03 --config FILE --target soda-native-example` with the common revision/architecture/evidence flags. Add `--restart` only with restart permission; `--hold` allows separate install/core-test invocations while the fixture stays owned. Interrupting a held fixture is recorded as cancellation, **not PASS**; its earlier SSH-ready fact and cleanup result remain separate. Without `--hold`, it boots to pinned SSH and shuts down.

Work/evidence must be disjoint fresh directories. Preflight checks matching Linux/KVM, tools, trust, paths and loopback port before creating VM state (private diagnostic evidence may exist on preflight failure). The overlay grows only the new disk, never the base. Orderly shutdown and owned process-group termination are bounded independently. Disks/NVRAM remain for inspection, including on failure; no PID-file adoption, global cleanup, project replacement, bridge/tap/firewall setup or persistent `soda-test` adoption occurs.

Use the native `x86_64` target under the [platform scope](../architecture/release.md#architectures). Firmware/`fw_cfg` boot behavior and final filesystem capacity still need actual native observation.

## Installed substrate and retained integrations

Invoke these existing/new entrypoints only with their named grants and actual target:

| Entry point | Observation, not a substitute for |
| --- | --- |
| `tests/installed/host.sh` | CoreOS/layering, service identities/capabilities, labels, listeners, observed activation phase; not login or a client route. Optional `SODA_HOST_PHASE` asserts the intended phase. Installed bytes and image objects in the local Podman image store are checked against the installed release inventory. |
| `service-ordering.sh` | Actual generated unit dependencies and failed units; no service mutations. |
| `soda-acceptance-remote cockpit-account` (debug build) | Real PAM account stage permits root and denies existing `nobody`; no new account and no password/session proof. |
| `/path/to/soda-installed-probes service-https ORIGIN CA_FILE` | Configured-origin trusted TLS from the selected client, no redirect/login journey or insecure fallback. |
| `operator.sh` | Selected native Tailnet/runner/version/branding/quiet-hook facts; no enrollment, registration or job. |
| `operator.ts ORIGIN PASSWORD_FILE PRIVATE_BROWSER_HOME HOSTNAME --stock-read-only` | Actual root login, stock Overview bridge/PAM/SELinux/native CLI read paths, Services/Logs and logout; reject Soda custom packages. Trust the CA in the isolated browser home first. No advertisement refresh or enrollment; no screenshot/trace/provider-body capture. Authored for the stock-only candidate, not executed on unretired targets. |
| `forgejo-advertisement.sh` | Explicit existing-helper invocation and unchanged core origins; also requires `SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH=1` and an already approved running Tailnet. |
| `probe-ssh --owner P11 --remote FILE …` | Pinned Git endpoint observed from the actual selected client. Use `User: git`; no identity key is used. Not Git auth/project acceptance. |

Shell/PAM checks require `SODA_NATIVE_VALIDATE` equal to the actual host. Browser checks use a fresh profile below a restricted browser home, retain it privately, and never bypass TLS. Native provider CLI package/version evidence is included in build metadata; personal authentication remains product validation.

Local runner execution is unavailable. No runner observation, Stop/Remove,
or `soda-runners` control protocol ships. Historical installed runner execution
receipts and their optional drivers do not qualify the current deferred feature.
Never dump runner credentials, container environments or entire provider responses.

Interactive console/native-branding reviews reuse the existing console welcome, opt-in branding renderer/tests and [capture rules](../design/screenshot-capture.md). Do not call a quiet noninteractive check an interactive/visual pass. Browser/product/user/repository/shared-tool/workload/persistence tests remain at their core-owned entrypoints.

## Evidence records

For candidate-bound host evidence, the host check compares installed content and the local Podman image objects named by `/usr/share/soda/release.json` against its content inventory. It does not inspect service container image selections or compare mutable configuration, databases or existing project containers; those policies and assertions remain core-owned.

Each new private evidence root has bounded, streaming-redacted captures. Structured values are sanitized before JSON encoding; `observation.pending.json` is retained and linked exclusively to `observation.json` only after successful write/close/leak checks. A finalization failure leaves no new final record. The record includes: owner, requested source, actual tool VCS state, client platform, selected target/topology/invocation, separate execution/evidence outcomes, public artifact references, file hashes and cleanup status. Add `--secret-file` for each known private value; SSH/bootstrap paths are not credential contents. Private Ignition values are collected before serial capture. Evidence records show invocation, exit and cleanup context. Generic `exec` source/target fields remain caller-declared unless the invoked owner check verifies them. Redirect queries are omitted. Exact-secret scanning is defense in depth, not proof against unknown secrets; capture selected facts only.

Each capture limits input, expanded retained output, each pending buffer and its
tee buffer to 16 MiB. Structured evidence also has a 16 MiB encoded-output limit,
checked before allocation of the compact and indented serializations. Known-secret
metadata admits at most 16,384 distinct raw/escaped patterns within a 16 MiB
budget, with conservative allowance for escaping before copying inputs. Exceeding
these limits is an evidence failure; structured output is never silently truncated.

Before evidence creation or capture, secret collection separately admits at most
16,384 nonempty candidates and 16 MiB cumulative candidate bytes across all
secret-file and Ignition inputs. Duplicates and required derived variants consume
that budget. Private-file/URI staging keeps its existing per-file bound; gzip
expansion and retained variant copies also respect the remaining collection
budget. The later escaped-pattern admission can refuse earlier and remains a
separate bound. Any refusal prevents capture and final publication.

QMP uses one finite phase across connection, capability negotiation, writes and
response matching, including buffered events. Command and VM capture pumps retain
the operation phase; pipes held past that phase produce an incomplete-capture
error. Finalization joins owned pumps, closes every redacting writer and checks
their errors before using captured bytes. CoreOS metadata needs no trailing
newline. Command capture failures remain separate evidence errors even when
native execution reports an expected denial. Capture failures prevent a completed
observation outcome; failed operations can still publish a failure record when
that record's own write, close and leak checks succeed.

```sh
/path/to/soda-acceptance report --arch x86_64 --revision FULL_COMMIT_SHA \
  --record /absolute/private/host-run/observation.json \
  --record /absolute/private/operator-run/observation.json \
  --out /absolute/private/new-support-evidence.md
```

Missing/failed/cancelled/evidence-failed scopes remain visible. Records from a different source/architecture or changed retained files are refused. Product observations retain their original owner labels (including historical U08/U20), not an independent support certification. No sibling/media/product qualification gate is introduced.

Authored coverage lives in `internal/acceptance/*_test.go`, `internal/release/build/*_test.go`,
`tests/build/test_native_support.py` and production/packaging/browser tests. Execution
is revision-scoped in the evidence record. Native builds/checks/VM/provider work still need
applicable action/target permission.

<a id="remaining-validation"></a>

## Historical audit reference

The reference inventory below is condensed from the audit at `58ddc0d` and its
subsequent remediation record (`git show 9f3baa7:docs/native-porting-audit.md`). It
records historical follow-up recommendations and evidence limits, not a current
list of defects or a mandatory matrix to replay after each change. The concrete
tool contracts above define required behavior. For an affected tool or new claim,
use applicable evidence and identify only the relevant missing proof; an untested
claim stays unverified. This is not a second product readiness gate.

| Boundary | Historical follow-up and evidence limits (not a current checklist) |
| --- | --- |
| Public/private payload | Explicit public paths and real credential-name rejection tests exist; exercise fresh staging/export and private-input contamination. Filename checks cannot prove unknown secrets absent from allowed content. Preserve old bundles with their original verifier. |
| Process ownership | Linux non-reaping leader/group termination and resistant-descendant cases exist; native cancellation, leader-first exit, remote interruption and exact bounded cleanup still need observed results. No stale-PID adoption. |
| Evidence/redaction/finalization | Structured escaped-secret handling and exclusive pending→final publication exist; retain failure-injection coverage and exercise actual partial launch/write/close/scan/cleanup failures. Unknown secrets remain outside exact-match guarantees. |
| Filesystem confinement | Directory-relative scans/hash/copy and parent checks exist; cover changed parents, special files, byte changes during copy/stream and actual native extraction, including legitimate CoreOS `/usr/local` mapping. |
| Artifact identity | Retired SPA payloads are rejected; rebuild/check current Go/Lit/stock-branding output. OCI schema/descriptor/rootfs checks and tiny real tar fixtures exist, not a full compressed-layer/native-import proof. Verify exact transfer digest, installer/verifier trust and byte-bound evidence record; generic exec identity stays caller-declared. |
| CoreOS/VM inputs | Missing-tool preflight, version capture and bounded tool phases exist. Independently select trusted signer/keyring and matching per-architecture firmware/tools; exercise retrieval/signature/decompression, strict Ignition and fresh KVM boot/restart/shutdown with retained disk/NVRAM. Never adopt the live guest. |
| First installation | Route/container-network collision, writable ancestry and booted-deployment checks exist; complete behavioral rejection fixtures and a genuinely fresh exact-target install/activation without repair edits. Reinstalling the persistent guest is not that proof. |
| Host/operator | Secret/TLS modes, socket/DNAT and enforcing-state checks exist; execute current listener/permission/byte checks, root/non-root Cockpit sessions, interactive/quiet console and native branding. Tailnet mutations and retained runner cleanup require their applicable task authorization. |
| Reporting/architecture | Keep invocation/exit/evidence/cleanup/artifact outcomes distinct, including missing/failed/not-reached scopes. Native remote dispatcher/transfer/fixture coverage remains incomplete; no inferred full support acceptance. |

Current results and retained evidence belong with the [development index](README.md) and task records,
not the audit's old worktree mappings. Product reachability, Git/shared tools,
workloads/persistence and browser integration stay with [native validation](testing.md).
[Licensing](../research/licensing.md) owns input/package/license/source-delivery obligations.
Neither this reference table nor a past successful run establishes new execution permission.
