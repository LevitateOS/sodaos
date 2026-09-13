# Native support tools

Implemented outside artifact/VM/SSH/evidence tools with **partial native evidence**.
See the [handoff](implementation-status.md) for exact build/check revisions and
limits; suite participation does not prove every remote/VM/install path. The
historical plan/audit are retired from active docs; their follow-up notes remain
in the [historical audit reference](#historical-audit-reference), not a second active work queue.

The [Sodaspaces plan](sodaspaces-plan.md) and production callers own API/config/schema,
build/stage and product behavior. These tools supply transport/artifacts/observations,
not duplicate product scenarios or a second readiness gate. The requested
[CoreOS installer implementation](coreos-installer.md) now has source and local
checks for upstream ISO customization and a production-owned console/continuation,
not Anaconda. Its console is now packaged on the ISO without hosted executable
retrieval; see the handoff for generation/inspection and bounded diskless BIOS/UEFI
boot evidence. Disk installation remains unrun.
A prepared product QCOW2 is a recommended [future download](installation.md#publication-direction);
its producer remains unimplemented and is not an outside-support wrapper.
U/P labels in existing CLI arguments and observations
are retained protocol/evidence identifiers, not an active numbered roadmap. No
helper, flag, commit or report grants execution permission.

## Shared contracts and provenance

| Interface | Owner / implementation boundary |
| --- | --- |
| `build-native.sh ARCH`, `check-native.sh ARCH`, `stage.py --arch ARCH` | Production build/stage remain authoritative. Support adds fresh-output locking, OCI archives, resolved image IDs, public input metadata and sealing. No standalone Soda UI payload remains. |
| Containerfile `BASE_IMAGE` argument | The build pins the existing Rocky reference to its resolved native digest reference during that build; unchanged default, no base upgrade or frontend change. |
| `install-native.sh /absolute/bundle/ARCH PRIVATE_SUBNET` | Existing first-install interface. Support adds verified archives, preflight before delivery, existing core tag restoration and a retained partial-install marker. No setup/OAuth/migration implementation is copied. |
| `render-provisioning.py` | Public `appliance/provisioning/base.json` and shared host-branding assets plus private per-instance inputs. Existing extension bootstrap remains the default; `--bootstrap minimal` is a fixture-only alternative without package installation. |
| `tools/soda-artifacts`, `tools/soda-acceptance` | Separate native `tools/` output, never appliance `cmd/`, rootfs or container payload. The bundle carries only the verifier as a transport utility, not an installed program. |
| Installed checks | Host/operator observations stay separate from product-owned developer/shared-tools/workload/persistence journeys. Old standalone browser harnesses are removed; the read-only native-page journey and exported-payload checks passed at the handoff's bounded local scope. |

Reuse and licensing are recorded in [native support notices](native-support-notices.md). The predecessor checkout and `scripts/test-vm.sh` remain separate and preserved.

## Effects and permissions

This section owns command effects, not an approval queue. Apply the
[execution policy](../AGENTS.md#permissions-and-preservation) and [current grants](implementation-status.md#current-permissions)
to those effects; already-approved local work does not need a new grant per command.

- `exec`: runs exactly the supplied owned check, locally or over pinned SSH. Its selected check determines browser, native and provider effects.
- `native`: one remote `prepare`, `build`, `check` or `bundle` phase. No automatic next phase. Preparation clones the canonical repository into a new private checkout; it never copies laptop binaries/dependencies/state.
- `fetch-coreos`: downloads/verifies/decompresses a public QEMU base into a fresh private cache. No overwrite, key import, VM or installation.
- `fetch-coreos-iso`: downloads/verifies the uncompressed upstream ISO selected by `appliance/locks/coreos-iso.json`, using the same trusted-key/signature boundary, into new `coreos.iso` and `verified-iso.json` outputs. It does not customize, boot, publish or install; `scripts/build-installer.py` is its concrete media caller.
- `convert-butane`: runs strict native Butane conversion into a new restricted file. No boot or install.
- `vm`: creates a new KVM overlay/NVRAM/process, boots to pinned SSH, then shuts down. `--restart` explicitly tests one restart of those same files. `--hold` keeps it available for separately authorized checks. The selected Ignition may install extensions: that requires installation permission as well as boot permission.
- `transfer`: validates/streams only a sealed bundle into a new remote directory, checks the transferred verifier before executing it, and verifies the payload. It does **not** install.
- `probe-ssh`: direct Git-endpoint key exchange using an existing trusted pin, without credentials, proxy, keyscan or Git authentication.
- `report`: reads/cites existing observations and verifies their retained-file hashes. No native work, retry, product verdict or publication.

The existing build now pulls the unchanged selected Forgejo/Caddy references for the matching platform and runs fresh, network-disabled, read-only image-inspection containers for RPM/Tea/gh version metadata. These are build inspection containers, **not project containers**. Include those operations in a build grant. The installer loads those exact archives and restores existing service references rather than looking up mutable tags later.

## Build and artifact contract

Use a fresh exact-revision checkout on matching-native Linux. A dirty checkout, occupied output or simultaneous build is refused; there is no automatic removal of earlier artifacts. `GOTOOLCHAIN=local` prevents implicit toolchain installation.

```sh
bash scripts/build-native.sh x86_64
bash scripts/check-native.sh x86_64
.artifacts/native/x86_64/tools/soda-artifacts bundle \
  --source "$PWD/.artifacts/native/x86_64" \
  --arch x86_64 --revision FULL_COMMIT_SHA \
  --out /absolute/new-export/x86_64
```

The export's parent must already exist; the `ARCH` directory must not. A bundle contains `rootfs/`, five actual OCI archives (including the locked Tailnet companion), the matching installer, verifier, public dependency/input records, notices, `build-info.json` and `SHA256SUMS`. The inspector checks ELF architecture, blob hashes, config/platform/source/base identity, required existing core payload, modes, symlinks and the exact file inventory. The payload contains native Forgejo templates/Lit assets and Soda's API/OAuth backend, not a standalone React frontend or Go page shells. Core packaging tests still own their detailed payload assertions.

`SHA256SUMS` identifies `build-info.json`, which identifies every delivered payload file. Establish that checksum through a trusted external channel **before executing any bundled program**, then verify the inventory. These are integrity records, not signatures or reproducible-build claims. Mutable package repositories and actual resolved RPMs are recorded, not disguised as pinned/reproducible inputs.

The first installer verifies before copying writable prefixes, validates the RFC1918 subnet and subordinate ranges before applying them, and refuses existing/partial Soda state. `/etc/soda/install-started` remains after a partial failure; do not remove it to pretend the attempt was clean. Application setup, HTTPS activation and migrations still use core-owned commands. Fresh extension bootstrap needs its separately approved activation reboot before installation; [existing Cockpit addon maintenance](installation.md#cockpit-addon-maintenance) owns the native additive live-update option. Do not install Soda on the builder.

## Local host-content image candidate

The [release engineering plan](release-engineering-plan.md#single-run-build-replacement-implementation)
owns the replacement of this transitional image path and the writable-bundle builder
with one source-to-release command. The new command is not implemented yet; the
following describes existing tools/effects, not the target build interface. The
current writable installer and sealed bundle remain usable pending native cutover.
`tools/soda-host-image` is a noninteractive local build tool, never an installed
appliance helper or update scheduler. It is also the canonical Go component
producer: the preserved `build-native.sh` adapter invokes its explicit
`--legacy-native` layout before legacy metadata/sealing. Do not mix that compatibility
flag with host build/complete/repository flags or treat its output as a host candidate.
Both callers use `internal/nativebuild/production.go`, not duplicated command recipes.

```sh
# Run from the canonical checkout after committing the candidate source.
# The fresh attempt's parent must exist; never clear/reuse an earlier attempt.
mkdir -p .artifacts/host-image
GOTOOLCHAIN=go1.26.7 go run ./tools/soda-host-image \
  --arch x86_64 --out "$PWD/.artifacts/host-image/UNIQUE-ATTEMPT" --build
```

Without `--build`, the command only snapshots committed source, prepares the public
context and compiles/inspects native binaries. It requires a clean checkout and
matching-native Linux/Go 1.26.7. `git archive` freezes the revision in the new ignored
attempt; it does not create a worktree, switch branches or touch another agent's
uncommitted work. Source archive, context inventory and private build log remain.
The [existing timing owner](installation.md#build-timing-and-progress-implementation-plan)
now supplies named sections, total wall time and failure/cancellation summaries to
this command too. `timing.log` contains progress only; `build.log` retains native
command diagnostics. The same Python supervisor handles the invocation's process
group; no service/timer is installed. A supplied parent timing origin/log is reused.

With `--build`, it pulls the architecture-specific digest in
`appliance/locks/coreos-host.json`, builds using local Podman (never a remote engine),
then runs one networkless read-only package/layout inspection container and exports
`host.oci`. `host.iid`, `inspect.cid`, `packages.txt` and `result.json` identify the
image, retained inspection container, installed package inventory and archive hash.
Build/intermediate/failed outputs are retained; no pruning or automatic cleanup is
performed. These are build effects, not VM/installed-appliance lifecycle effects.
No signing keys, scheduler, GHCR publication or global trust changes are involved.

The `soda_host_image` Go build tag selects fixed vendor helper/unit locations, while
ordinary builds retain the existing writable paths. `appliance/host.Containerfile`
installs the package list extracted from the current provisioning owner and records
resolved RPMs. The base is digest-pinned; live Fedora/Tailscale repositories are not
snapshotted, so this is **not a reproducible-RPM or production-release claim**.

Without `--complete`, this remains **host-content-only**: binaries, vendor units/
Quadlets and basic native configuration, retaining legacy/dev app references.
It does not deliver the application payload or complete first-boot setup.

For the complete local candidate, add an explicit **intended** repository prefix:

```sh
GOTOOLCHAIN=go1.26.7 go run ./tools/soda-host-image \
  --arch x86_64 --out "$PWD/.artifacts/host-image/UNIQUE-COMPLETE-ATTEMPT" \
  --build --complete --repository-prefix ghcr.io/OWNER/PREFIX
```

Replace `OWNER/PREFIX` with lowercase intended names; this does not reserve or create
GHCR resources. `--complete` requires `--build`, the workspace-pinned Bun and the
ordinary native asset/project-tool prerequisites. It reuses the existing recipes,
Forgejo payload manifest, browser/locales/terminal builders and `stage.py` inside the
frozen snapshot. App bases resolve once and are built by digest with `--pull=never`.
The host transaction requires `appliance/locks/host-packages-ARCH.json`; currently
only x86_64 has the qualified NEVRA/inventory input. No ARM lock is synthesized.

Additional effects: public dependency downloads, local app builds and OCI exports,
read-only/networkless Forgejo binary/presentation inspection and native host Quadlet
`--dryrun` generation. Exact inspection containers/CIDs and intermediates remain.
No application service, provider, VM or retained project is started. Output includes
`images/{dashboard,forgejo,proxy,project-os,tailnet}.oci`, `app-inputs.json`, immutable
presentation hashes, `payload.json`, `candidate.json`, `generated-quadlets.txt`,
`host.oci` and the final `result.json`. Only retained-runtime archives are embedded
in the host; three bound app references require later digest-preserving publication.
A failed attempt is not complete merely because some archives or payload.json exist.

The [release plan](release-engineering-plan.md#local-candidate-content-and-machine-state-ownership)
owns defaults, persistent image storage and metadata. Complete candidates are still
unsigned/unpublished, with no native first-install/update/recovery acceptance or
qualified upgrade edges. Do not deploy either mode as an installable Soda release.
Zincati updates are disabled and bootc's automatic timer masked only inside these
candidate images, not on the builder or retained appliances.

## Trusted release-delivery worker

The [release plan](release-engineering-plan.md#implemented-trusted-delivery-contract)
owns authority, protocol and custody. `tools/soda-release` is a noninteractive worker
CLI; it does not run build code, install a policy/image, import Podman state or reboot.
It uses the locked native skopeo from `internal/releasedelivery/tools.json` and the
existing M1 OCI verifier. No new Go cryptographic signing implementation or registry
server is introduced. Run source checks/builds with the repository-pinned Go.

All operations require `--trust PUBLIC-TRUST.json` and an absolute `--out`. Trust is
public but integrity-controlled; it is not build-supplied authority. The exact JSON
types are in [`internal/releasedelivery/model.go`](../internal/releasedelivery/model.go).
Native subprocesses do not inherit ambient credentials/proxy/user configuration.
Public anonymous GHCR access is the implemented commissioning target; private pulls
or an ambient authenticated proxy are not silently substituted.

| Operation | Inputs and effects |
| --- | --- |
| `prepare` | `--input M1-CANDIDATE-DIR --qualification PUBLIC-QUALIFICATION.json`; verifies all six archives/identities and emits one local release OCI layout. Prints its intended digest reference. No signatures/registry effects. |
| `channel` | `--input CHANNEL.json`; validates channel shape/freshness and writes its local OCI layout/reference. Does not authorize publication or prove referenced artifacts available. |
| `policy` | Optional `--base-policy FILE`; writes proposed policy/registries.d into a fresh directory, preserving unrelated scopes or refusing conflicts. Never installs it. |
| `init-state` | `--out NEW-PRIVATE-STATE-FILE`; explicit bootstrap only, refuses an existing file. No discovery/reset/recovery side effect. |
| `init-ledger` | `--repository EXACT-REPOSITORY --out NEW-PRIVATE-LEDGER-FILE`; explicit publisher bootstrap. Existing channel history cannot be silently adopted with a fresh ledger. |
| `sign` | `--input LOCAL-PATH --transport oci\|oci-archive\|dir --permit PRIVATE-PERMIT.json --signer PRIVATE-SIGNER.json`; private snapshot, exact-digest admission, local native signing and verification. No registry write or interactive signing prompt. |
| `publish` | `--input SIGNED-DIR --permit PRIVATE-PERMIT.json --auth-file PRIVATE-AUTH.json --ledger EXISTING-LEDGER`; **real registry writes**, credential use and protected promotion if authorized. Requires fresh output; stops/holds on uncertain effects. |
| `publish --observe` | Same protected permit/ledger, but no input/auth file needed; observes a recorded pending/completed publication without replaying uploads, including after permit expiry. It is not a fresh-offer/activation approval. |
| `fetch` | `--channel candidate\|preview\|stable --arch ARCH --state EXISTING-STATE`; public discovery plus native signature-verified downloads, durable observed-authority state and a verified receipt only after completeness. No installation/activation. |

Except the two initialization operations, `--out` names a **new** directory with an
existing real parent. It is never reused/cleared automatically. Keep local outputs
under `.artifacts/release-delivery/`. Private keys/passphrases/permits/auth/signer
configuration must be real files owned by the worker, with no group/other access,
inside restricted directories. `PRIVATE-SIGNER.json` contains only `Key` and
`Passphrase` absolute filenames; secret values never go in argv. `Permit` specifies
exact `Repository`, `Digest`, expiry and, for promotion, the previously approved
channel digest or explicit `absent` bootstrap. The **protected pipeline worker**
produces these permits after qualification; users do not manually sign releases or
write per-release permits as the intended operating procedure. Build/test workers
must not share the signer's UID, credentials or generic command authority.

Preserve snapshots, signatures, pending ledgers and failed outputs. Skopeo's
`--remove-signatures` applies only to the new private signing snapshot so an existing
signature cannot mask the wrong signing key; it does not remove registry artifacts
or signatures. Every publication uses `--preserve-digests` and verifies an anonymous
native round trip. A complete or uncertain job is observed, not automatically replayed.
There is no tag CAS or distributed publisher lock; obey the owning single-writer
contract. Explicit fault reconciliation is not ordinary per-release manual signing.

Source/process-double checks:

```sh
GOTOOLCHAIN=go1.26.7 go test -race ./internal/releasedelivery ./tools/soda-release
```

Native **filesystem-only** Sigstore proof, using fresh synthetic keys and no network
transport, registry, daemon, fixture lifecycle or global trust changes:

```sh
mkdir -p .artifacts/release-delivery
SODA_RELEASE_NATIVE_OUT="$PWD/.artifacts/release-delivery/UNIQUE-NATIVE-PROOF" \
  GOTOOLCHAIN=go1.26.7 go test ./internal/releasedelivery \
  -run '^TestNativeSigstoreDirectoryRoundTrip$' -count=1 -v
```

The directory must not exist. Keys, private passphrase, failed copies and the receipt
remain restricted there; never commit or publish them. This opt-in native receipt
is not GHCR attachment/anonymous-pull, installed bootc policy/cache, worker isolation
or native boot/upgrade acceptance. Real resource provisioning/publication needs its
exact grant; no release timer or worker service is installed by these commands.

The [retained real bootstrap receipt](implementation-history.md#ghcr-namespace-and-signing-bootstrap)
identifies the root-only builder state at `/var/lib/soda-release`, public trust at
`appliance/keys/release-trust.json`, completed immutable uploads and the remaining
GitHub visibility step. That operational state is separate from synthetic/local-test
outputs. Do not rerun initialization/key generation or recreate its root. Use the
[current handoff](implementation-status.md#current-permissions) for the bounded
candidate commissioning grant; this guide does not authorize additional effects.

## SSH, commands and exact-source remote phases

Owned process execution (`exec`, `native`, `transfer`, `vm`) now requires Linux's
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

The target is the machine's actual hostname, not an assumed DNS alias. Supply matching `--revision`, `--arch`, `--target`, `--owner P02`, `--remote`, `--request` and a **new** `--evidence` directory. Later requests change only `Phase` and reuse the binding. `check` requires completed `build`; `bundle` requires completed `check`. Each phase is exclusive, verifies the unchanged clean checkout, and retains failed work. No automatic retry or revision switch.

Remote commands are bounded with native `timeout` plus a 10-second termination allowance. Closing SSH is **not proof that a remote process has already stopped**; on interruption, retain the failed observation and inspect the authorized target before retrying. Core tests/provider actions own their resource cleanup. Source tools may be built on a separately authorized client for coordination; a client tool build is never appliance-architecture evidence.

## Verified CoreOS and private provisioning

`appliance/locks/coreos-qemu.json` records the stable metadata's selected `44.20260817.3.2` QEMU inputs for both architectures. The metadata was inspected during implementation; the images/signatures were **not downloaded or natively exercised**. The caller must independently select a trusted Fedora keyring and full signer fingerprint (see [Fedora security](https://fedoraproject.org/security/)). No key is fetched/imported as a trust shortcut.

```sh
/path/to/soda-artifacts fetch-coreos --arch x86_64 \
  --lock /absolute/source/appliance/locks/coreos-qemu.json \
  --keyring /absolute/trusted/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --out /absolute/cache/new-coreos-attempt
```

Both compressed/uncompressed hashes and the selected GPG signature must match. Success writes a read-only `coreos.qcow2` and private `verified-base.json`. A failed attempt is retained and cannot be overwritten. This is a fixture base, **not a preinstalled Soda disk or selected media deliverable**.

Prepare one private directory, a fresh Ed25519 **host** key, the operator's existing public authentication key, and a private crypt password-hash file. Host and operator keys are different identities. Host-key generation and conversion are explicit actions; never put passwords/hashes/key contents in shell arguments or logs.

```sh
python3 scripts/render-provisioning.py --bootstrap minimal \
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
  "CoreOSLock": "/absolute/source/appliance/locks/coreos-qemu.json",
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

Invoke `vm --owner P03 --config FILE --target soda-native-example` with the common revision/architecture/evidence flags. Add `--restart` only with restart permission; `--hold` allows separate install/core-test invocations while the fixture stays owned. Interrupting a held fixture is recorded as cancellation, **not PASS**; its earlier SSH-ready fact and cleanup result remain separate. Without `--hold`, it boots to pinned SSH and shuts down.

Work/evidence must be disjoint fresh directories. Preflight checks matching Linux/KVM, tools, trust, paths and loopback port before creating VM state (private diagnostic evidence may exist on preflight failure). The overlay grows only the new disk, never the base. Orderly shutdown and owned process-group termination are bounded independently. Disks/NVRAM remain for inspection, including on failure; no PID-file adoption, global cleanup, project replacement, bridge/tap/firewall setup or persistent `soda-test` adoption occurs.

For aarch64 use its own lock entry, `qemu-system-aarch64`, compatible ARM UEFI pair and matching-native hardware. Availability and success are independent of x86_64. Firmware/`fw_cfg` boot behavior and final filesystem capacity still need actual native observation.

## Installed substrate and retained integrations

Invoke these existing/new entrypoints only with their named grants and actual target:

| Entry point | Observation, not a substitute for |
| --- | --- |
| `tests/installed/host.sh` | CoreOS/layering, service identities/capabilities, labels, listeners, observed activation phase; not login or a client route. Optional `SODA_HOST_PHASE` asserts the intended phase. Supply `SODA_BUNDLE`/`SODA_REVISION` to verify immutable delivered bytes and current service image IDs against that candidate; without them the script explicitly records that identity was not checked. |
| `service-ordering.sh` | Actual generated unit dependencies and failed units; no service mutations. |
| `cockpit-account.py` | Real PAM account stage permits root and denies existing `nobody`; no new account and no password/session proof. |
| `service-https.py ORIGIN CA_FILE` | Configured-origin trusted TLS from the selected client, no redirect/login journey or insecure fallback. |
| `operator.sh` | Selected native Tailnet/runner/version/branding/quiet-hook facts; no enrollment, registration or job. |
| `operator.ts ORIGIN PASSWORD_FILE PRIVATE_BROWSER_HOME HOSTNAME --stock-read-only` | Actual root login, stock Overview bridge/PAM/SELinux/native CLI read paths, Services/Logs and logout; reject Soda custom packages. Trust the CA in the isolated browser home first. No advertisement refresh or enrollment; no screenshot/trace/provider-body capture. Authored for the stock-only candidate, not executed on unretired targets. |
| `forgejo-advertisement.sh` | Explicit existing-helper invocation and unchanged core origins; also requires `SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH=1` and an already approved running Tailnet. |
| `probe-ssh --owner P11 --remote FILE …` | Pinned Git endpoint observed from the actual selected client. Use `User: git`; no identity key is used. Not Git auth/project acceptance. |

Shell/PAM checks require `SODA_NATIVE_VALIDATE` equal to the actual host. Browser checks use a fresh profile below a restricted browser home, retain it privately, and never bypass TLS. Native provider CLI package/version evidence is included in build metadata; personal authentication remains product validation.

Registration/start/stop/restart/remove use native **Runners** at
`/admin?soda-view=runners` and the existing root `soda-runners` stdin protocol with
separately approved IDs/provider grants. The current operator journey checks
Cockpit runner retirement; unretired targets retain their historical check revision. Retain actual provider run URL/attempt and job output, not merely listener status. `tests/fixtures/runner/native-support.yaml` is a manually selected trusted-job fixture, outside CI discovery; approve any copy/scheduling in the actual provider repository first. Choose the actual registered label. Record registration removal, service/account cleanup and provider leftovers explicitly. Never dump registrations, runner credentials, container environments or entire provider responses.

Interactive console/native-branding reviews reuse the existing console welcome, opt-in branding renderer/tests and [capture rules](screenshot-capture.md). Do not call a quiet noninteractive check an interactive/visual pass. Browser/product/user/repository/shared-tool/workload/persistence tests remain at their core-owned entrypoints.

## Evidence and handoff

For candidate-bound host evidence, invoke `verify-installed` (directly or through the host check's bundle/revision variables). It compares the install-attempt revision, immutable delivered files/modes/links and current Forgejo/dashboard/proxy image IDs. It deliberately does not compare mutable configuration, databases or existing project containers, whose policy and assertions remain core-owned.

Each new private evidence root has bounded, streaming-redacted captures. Structured values are sanitized before JSON encoding; `observation.pending.json` is retained and linked exclusively to `observation.json` only after successful write/close/leak checks. A finalization failure leaves no new final record. The record includes: owner, requested source, actual tool VCS state, client platform, selected target/topology/invocation, separate execution/evidence outcomes, public artifact references, file hashes and cleanup status. Add `--secret-file` for each known private value; SSH/bootstrap paths are not credential contents. Private Ignition values are collected before serial capture. Transfers automatically retain the selected manifest hash; handoffs show invocation, exit and cleanup context. Generic `exec` source/target fields remain caller-declared unless the invoked owner check verifies them. Redirect queries are omitted. Exact-secret scanning is defense in depth, not proof against unknown secrets; capture selected facts only.

```sh
/path/to/soda-acceptance report --arch x86_64 --revision FULL_COMMIT_SHA \
  --record /absolute/private/host-run/observation.json \
  --record /absolute/private/operator-run/observation.json \
  --out /absolute/private/new-support-handoff.md
```

Missing/failed/cancelled/evidence-failed scopes remain visible. Records from a different source/architecture or changed retained files are refused. Product observations retain their original owner labels (including historical U08/U20), not an independent support certification. No sibling/media/product qualification gate is introduced.

Authored coverage lives in `internal/acceptance/*_test.go`, `internal/nativebuild/*_test.go`,
`tests/build/test_native_support.py` and production/packaging/browser tests. Execution
is revision-scoped in the handoff. Native builds/checks/VM/provider work still need
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
| Artifact identity | Retired SPA payloads are rejected; rebuild/check current Go/Lit/stock-branding output. OCI schema/descriptor/rootfs checks and tiny real tar fixtures exist, not a full compressed-layer/native-import proof. Verify exact transfer digest, installer/verifier trust and byte-bound handoff; generic exec identity stays caller-declared. |
| CoreOS/VM inputs | Missing-tool preflight, version capture and bounded tool phases exist. Independently select trusted signer/keyring and matching per-architecture firmware/tools; exercise retrieval/signature/decompression, strict Ignition and fresh KVM boot/restart/shutdown with retained disk/NVRAM. Never adopt the live guest. |
| First installation | Route/container-network collision, writable ancestry and booted-deployment checks exist; complete behavioral rejection fixtures and a genuinely fresh exact-target install/activation without repair edits. Reinstalling the persistent guest is not that proof. |
| Host/operator | Secret/TLS modes, socket/DNAT and enforcing-state checks exist; execute current listener/permission/byte checks, root/non-root Cockpit sessions, interactive/quiet console and native branding. Tailnet mutations and Forgejo runner lifecycle/jobs/removal require separate grants. |
| Reporting/architecture | Keep invocation/exit/evidence/cleanup/artifact outcomes distinct, including missing/failed/not-reached scopes. Native remote dispatcher/transfer/fixture coverage and independent aarch64 results remain incomplete; no inferred full support acceptance. |

Current results and retained evidence belong to the [handoff](implementation-status.md),
not the audit's old worktree mappings. Product reachability, Git/shared tools,
workloads/persistence and browser integration stay with [native validation](native-validation.md).
[Licensing](licensing.md) owns input/package/license/source-delivery obligations.
Neither this reference table nor a past successful run establishes new execution permission.
