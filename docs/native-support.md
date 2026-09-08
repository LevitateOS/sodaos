# Native support tools

Implemented outside artifact/VM/SSH/evidence tools with **partial native evidence**.
See the [handoff](implementation-status.md) for exact build/check revisions and
limits; suite participation does not prove every remote/VM/install path. The
historical plan/audit are retired from active docs; unresolved checks are retained
[below](#remaining-validation), not declared fixed by deleting an audit.

The [Sodaspaces plan](sodaspaces-plan.md) and production callers own API/config/schema,
build/stage and product behavior. These tools supply transport/artifacts/observations,
not duplicate product scenarios or a second readiness gate. Optional ISO/QCOW2
wrappers remain unselected. U/P labels in existing CLI arguments and observations
are retained protocol/evidence identifiers, not an active numbered roadmap. No
helper, flag, commit or report grants execution permission.

## Shared contracts and provenance

| Interface | Owner / implementation boundary |
| --- | --- |
| `build-native.sh ARCH`, `check-native.sh ARCH`, `stage.py --arch ARCH` | Production build/stage remain authoritative. Support adds fresh-output locking, OCI archives, resolved image IDs, public input metadata and sealing. No standalone Soda UI payload remains. |
| Containerfile `BASE_IMAGE` argument | The build pins the existing Rocky reference to its resolved native digest reference during that build; unchanged default, no base upgrade or frontend change. |
| `install-native.sh /absolute/bundle/ARCH PRIVATE_SUBNET` | Existing first-install interface. Support adds verified archives, preflight before delivery, existing core tag restoration and a retained partial-install marker. No setup/OAuth/migration implementation is copied. |
| `render-provisioning.py` | Public `appliance/provisioning/base.json` plus private per-instance inputs. Existing extension bootstrap remains the default; `--bootstrap minimal` is a fixture-only alternative without package installation. |
| `tools/soda-artifacts`, `tools/soda-acceptance` | Separate native `tools/` output, never appliance `cmd/`, rootfs or container payload. The bundle carries only the verifier as a transport utility, not an installed program. |
| Installed checks | Host/operator observations stay separate from product-owned developer/shared-tools/workload/persistence journeys. Old standalone browser harnesses are removed; the read-only native-page journey and exported-payload checks passed at the handoff's bounded local scope. |

Reuse and licensing are recorded in [native support notices](native-support-notices.md). The predecessor checkout and `scripts/test-vm.sh` remain separate and preserved.

## Effects and permissions

Every command below is a **later, explicitly authorized recipe**, not a record of execution:

- `exec`: runs exactly the supplied owned check, locally or over pinned SSH. Tests, browser login and provider mutations need their own grants.
- `native`: one remote `prepare`, `build`, `check` or `bundle` phase. No automatic next phase. Preparation clones the canonical repository into a new private checkout; it never copies laptop binaries/dependencies/state.
- `fetch-coreos`: downloads/verifies/decompresses a public base into a fresh private cache. No overwrite, key import, VM or installation.
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

The export's parent must already exist; the `ARCH` directory must not. A bundle contains `rootfs/`, four actual OCI archives, the matching installer, verifier, public dependency/input records, notices, `build-info.json` and `SHA256SUMS`. The inspector checks ELF architecture, blob hashes, config/platform/source/base identity, required existing core payload, modes, symlinks and the exact file inventory. New bundles reject retired standalone React assets/inputs; Soda has no embedded HTML; it is an API/OAuth command. Core packaging tests still own their detailed payload assertions.

`SHA256SUMS` identifies `build-info.json`, which identifies every delivered payload file. Establish that checksum through a trusted external channel **before executing any bundled program**, then verify the inventory. These are integrity records, not signatures or reproducible-build claims. Mutable package repositories and actual resolved RPMs are recorded, not disguised as pinned/reproducible inputs.

The first installer verifies before copying writable prefixes, validates the RFC1918 subnet and subordinate ranges before applying them, and refuses existing/partial Soda state. `/etc/soda/install-started` remains after a partial failure; do not remove it to pretend the attempt was clean. Application setup, HTTPS activation and migrations still use core-owned commands. An extension request needs its separately approved activation reboot before installation. Do not install Soda on the builder.

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
| `operator.mjs ORIGIN PASSWORD_FILE PRIVATE_BROWSER_HOME HOSTNAME --allow-advertisement-refresh` | Actual root login, native PAM/SELinux bridge access, retained pages/read paths and logout. Trust the CA in the isolated browser home first. Tailnet's existing page effect may refresh Forgejo advertisement; that grant is mandatory. No screenshot/trace/provider-body capture. |
| `forgejo-advertisement.sh` | Explicit existing-helper invocation and unchanged core origins; also requires `SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH=1` and an already approved running Tailnet. |
| `probe-ssh --owner P11 --remote FILE …` | Pinned Git endpoint observed from the actual selected client. Use `User: git`; no identity key is used. Not Git auth/project acceptance. |

Shell/PAM checks require `SODA_NATIVE_VALIDATE` equal to the actual host. Browser checks use a fresh profile below a restricted browser home, retain it privately, and never bypass TLS. Native provider CLI package/version evidence is included in build metadata; personal authentication remains product validation.

Registration/start/stop/restart/remove use the **existing** `soda-runners` stdin protocol and Cockpit UI with separately approved IDs/provider grants. Retain actual provider run URL/attempt and job output, not merely listener status. `tests/fixtures/runner/native-support.yaml` is a manually selected trusted-job fixture, outside CI discovery; approve any copy/scheduling in the actual provider repository first. Choose the actual registered label. Record registration removal, service/account cleanup and provider leftovers explicitly. Never dump registrations, runner credentials, container environments or entire provider responses.

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
`tests/build/test_native_support.py` and production/packaging/Cockpit tests. Execution
is revision-scoped in the handoff. Native builds/checks/VM/provider work still need
applicable action/target permission.

## Remaining validation

Condensed from the audit at `58ddc0d` and its subsequent source-remediation record,
retained in full at `git show 9f3baa7:docs/native-porting-audit.md`. The old audit's
line-specific defects are not assertions about today's edited source. These are
remaining proof/coverage obligations, not new work authorizations or a finding
that any retained bundle leaked secrets.

| Boundary | Source follow-up and remaining proof |
| --- | --- |
| Public/private payload | Explicit public paths and real credential-name rejection tests exist; exercise fresh staging/export and private-input contamination. Filename checks cannot prove unknown secrets absent from allowed content. Preserve old bundles with their original verifier. |
| Process ownership | Linux non-reaping leader/group termination and resistant-descendant cases exist; native cancellation, leader-first exit, remote interruption and exact bounded cleanup still need observed results. No stale-PID adoption. |
| Evidence/redaction/finalization | Structured escaped-secret handling and exclusive pending→final publication exist; retain failure-injection coverage and exercise actual partial launch/write/close/scan/cleanup failures. Unknown secrets remain outside exact-match guarantees. |
| Filesystem confinement | Directory-relative scans/hash/copy and parent checks exist; cover changed parents, special files, byte changes during copy/stream and actual native extraction, including legitimate CoreOS `/usr/local` mapping. |
| Artifact identity | Retired SPA payloads are rejected; rebuild/check current Go/Cockpit output. OCI schema/descriptor/rootfs checks and tiny real tar fixtures exist, not a full compressed-layer/native-import proof. Verify exact transfer digest, installer/verifier trust and byte-bound handoff; generic exec identity stays caller-declared. |
| CoreOS/VM inputs | Missing-tool preflight, version capture and bounded tool phases exist. Independently select trusted signer/keyring and matching per-architecture firmware/tools; exercise retrieval/signature/decompression, strict Ignition and fresh KVM boot/restart/shutdown with retained disk/NVRAM. Never adopt the live guest. |
| First installation | Route/container-network collision, writable ancestry and booted-deployment checks exist; complete behavioral rejection fixtures and a genuinely fresh exact-target install/activation without repair edits. Reinstalling the persistent guest is not that proof. |
| Host/operator | Secret/TLS modes, socket/DNAT and enforcing-state checks exist; execute current listener/permission/byte checks, root/non-root Cockpit sessions, interactive/quiet console and native branding. Tailnet mutations and both providers' actual runner lifecycle/jobs/removal require separate grants. |
| Reporting/architecture | Keep invocation/exit/evidence/cleanup/artifact outcomes distinct, including missing/failed/not-reached scopes. Native remote dispatcher/transfer/fixture coverage and independent aarch64 results remain incomplete; no inferred full support acceptance. |

Tests for these boundaries are not an exhaustive recovery framework. Preserve
failed attempts and the worktree retention mappings. Product reachability, native
Git/shared tools/workloads/persistence and browser integration remain in
[native validation](native-validation.md), not a second support suite. Actual
input/package/license/source-delivery closure is still required; see [licensing](licensing.md).
