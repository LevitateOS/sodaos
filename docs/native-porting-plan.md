# Native artifact and acceptance porting plan

**Status: proposal and implementation plan only.** Writing this document ports no code, builds no artifacts and establishes no new test results. All P01–P13 milestones below are not started. The current request authorizes planning, not execution of the proposed tools.

**Objective:** selectively reuse the predecessor's native artifact and acceptance machinery, supply CoreOS-compatible delivery tools, and establish the current SodaOS developer/operator journeys on native x86_64 first. Keep the ARM laptop as an editor, coordinator and possible developer client; use matching-native Linux for builds and installed execution.

**Quick navigation:** [decisions](#2-scope-and-decisions) · [file-level reuse](#3-exact-reuse-inventory) · [tool ownership](#4-source-ownership-and-entrypoints) · [artifact contracts](#5-artifact-and-workspace-contracts) · [milestones](#6-milestones-and-dependency-order) · [coverage](#7-acceptance-coverage-and-evidence-boundaries) · [execution gates](#8-execution-gates) · [risks](#10-highest-risks-and-stopping-points) · [commit sequence](#11-commit-and-handoff-sequence).

## 1. Baseline and authority

- Current source reviewed: SodaOS `6f7b51e9b94e85ec2aae816e2217ba3d6d1b046b`.
- Predecessor source reviewed: `soda-os` `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. Do not modify that repository. Preserve applicable attribution, licenses and asset provenance with each port.
- Governing product scope: [architecture](architecture.md) and [deferred work](deferred.md). The old acceptance assertions do not replace either.
- Existing source milestones: M01–M14 in the [implementation plan](implementation-plan.md). This plan supplies tooling and delivery follow-ups for M15–M18; it does not restart completed source work or declare those validation stages complete.
- Recorded native work: [implementation status](implementation-status.md) and [local testing](local-testing.md). Earlier x86_64 builds, installation corrections and operator browser checks do not validate the merged tree. Full developer/runtime/persistence journeys remain pending.
- Existing installation remains the [CoreOS provisioning and native component installation path](installation.md). Preserve the application topology: native operator services, separate Forgejo/dashboard/Caddy containers, and persistent Rocky project containers.

The predecessor builds a **bootc host OCI**, then independently derives an Anaconda network ISO from its published digest and a QCOW2 from its local archive. SodaOS currently builds **application/project OCI images**, not a bootable host OCI. Neither an ISO-to-QCOW2 conversion nor direct application-OCI-to-host-image conversion is the selected mechanism.

## 2. Scope and decisions

### Included

1. Concrete Go helpers for native artifact inspection, QEMU/KVM ownership, SSH execution, evidence and cleanup, with the relevant predecessor tests.
2. A small remote entrypoint for running explicit phases on an exact source revision, without adopting release CI or installing a special executor account.
3. Reliable native build output, deployment bundling, checksums and source/platform attribution.
4. Pinned CoreOS input handling and a common provisioning/component-install path for fresh installer and VM tests.
5. CoreOS installer ISO and QCOW2 delivery recipes with explicit artifact semantics.
6. Real browser, project-IP SSH/transfer, shared-tool, workload, persistence and operator acceptance adapted to the current product.
7. Independent architecture support in source, with x86_64 execution first and aarch64 execution later.

### Excluded

- A Soda bootc host image, Anaconda/Kickstart package stack, RPM distribution builder, custom updater or the predecessor's reserved Updates platform.
- Registry publication, release tags, GitHub Releases, promotion, signing ceremonies, dual-architecture qualification certificates and automatic CI. Artifact integrity checks are not a publication system.
- Old host developer/workspace accounts, PAM-backed Forgejo identity, managed checkouts, project deletion and B→A→B update/fallback tests.
- Generic workflow engines, scenario plugins, resumable jobs, reconciliation, private toolchain/service branches or speculative runtime backends.
- Bare-metal, public-cloud import or public-ingress claims inferred from a local KVM guest.

### Decisions to settle in P01

These are recommended defaults, not choices silently established by this planning change. A blocked media decision must not block independent harness or product-test source work.

| Decision | Recommended initial contract | Consequence / alternative requiring an explicit decision |
| --- | --- | --- |
| Host mechanism | Retain upstream CoreOS plus native Soda installation | A bootable Soda **host OCI** would require a different host-composition decision; do not import bootc merely to produce that filename. |
| Network dependency | Network-assisted initial installation; record required endpoints and resolved package inputs | An offline appliance requires the full host-extension/package/container closure and disconnected tests, not just an embedded Ignition file. Do not label the initial output offline. |
| ISO meaning | CoreOS installation media with non-secret Soda bootstrap/payload support; operator provisioning and destination selection remain explicit | Personalized/unattended media is a separate private derivative and can overwrite disks at boot. It is never a distributable default. |
| QCOW2 meaning | A reusable **QCOW2 deployment kit**: pristine CoreOS disk, the exact Soda bundle, and the supported per-instance Ignition/bootstrap recipe | If the required deliverable is a single preinstalled Soda QCOW2, select and prove a supported offline composition recipe before completing P10. A kit or renamed upstream disk must not be presented as that image. |
| x86 boot profile | Headless native QEMU/KVM; UEFI/OVMF for new media acceptance, with explicitly selected firmware | The existing `soda-test` launcher has no explicit OVMF configuration. Do not change its firmware or claim it already proves this new profile. BIOS/Secure Boot coverage is separately named, not inferred. |
| Developer network | One explicitly approved private routed topology with a real client | Select a LAN-routed VM network or an approved Tailnet subnet route. Existing QEMU user-network SSH forwarding cannot satisfy this requirement. |
| Execution ownership | Native builder and new run-owned guests; separate operator/provider grants | Name hosts, disks, routes, accounts, repositories, provider resources and allowed cleanup before execution. An environment variable is not authorization. |

## 3. Exact reuse inventory

All predecessor paths in this table are relative to the pinned [predecessor tree](https://github.com/LevitateOS/soda-os/tree/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c). Recheck source and dependency closure when implementing; copy only exercised callers and adapted tests.

| Predecessor source | Disposition and proposed destination | Tests to bring or adapt |
| --- | --- | --- |
| `internal/acceptance/qmp.go` | Port into `internal/acceptance/`; retain typed responses, handshake and errors; bound cancellation/read deadlines | `qmp_test.go`; add cancellation, malformed response and missing-socket cases |
| `internal/acceptance/processes.go`, `cleanup.go` | Port exact child/process-group ownership and bounded finalization; distinguish unexpected VM exit from successful cleanup | `processes_test.go`, `cleanup_test.go` |
| `internal/acceptance/qemu.go`, `qemu_inputs.go` | Adapt native preflight, disk creation, firmware selection and QMP wiring; remove GTK/Cocoa, cloud-init and fixed service/development forwards from the default path | `qemu_test.go`, relevant `preflight_test.go` cases |
| `internal/acceptance/guest.go` | Retain one owner per VM/disk, orderly restart and partial-launch cleanup; remove unconditional enrollment/fallback dependencies | `guest_test.go`, `guest_process_test.go`; retain replacement-failure and ISO/disk independence cases |
| `internal/acceptance/remote.go`, `command.go` | Adapt literal-argument SSH execution and returned stdout/stderr/exit result; replace refreshed host-key trust and insecure HTTPS; separate SSH readiness from optional web readiness | `remote_test.go`, `command_results_test.go`, applicable `assertions_test.go` cases |
| `internal/acceptance/evidence.go` | Adapt private exclusive output and error redaction; redact before writing, reject escaping/symlinked paths, retain a final leak check as defense in depth | `evidence_test.go`, `evidence_boundary_test.go`; add split-write redaction and unsafe-path cases |
| `internal/acceptance/runner.go`, `itinerary.go`, `runner_init.go`, `runner_vm.go` | Reuse readable ordered calls and finalization patterns, not their release/registry/account model; write a current-scope runner | Relevant `runner_test.go`, `runner_init_test.go`, `reporting_test.go` behaviors, not obsolete scenario requirements |
| `internal/acceptance/product_scenarios.go`, `project_scenarios.go`, `fixtures.go` | Extract SSH/SCP/SFTP, HTTP content-change and fixture techniques; rewrite setup through Soda/Forgejo and project-local accounts | Adapt focused fixture/transport assertions and shell-boundary tests |
| `internal/acceptance/preservation.go`, `preservation.sh` | Port bounded snapshot/comparison techniques, not old path/account catalogs or shadow output; compare current project state across normal lifecycle operations | `preservation_test.go`, including deliberately changed fixture data and failed snapshot commands |
| `internal/acceptance/local_forwarded.go`, `tailnet.go` | Reuse distinction between forwarded and routed access and guest-specific enrollment ownership only where the operator scenario needs it | Adapt `local_forwarded_test.go`, `tailnet_test.go`; remove old endpoints and host-wide peer discovery assumptions |
| `internal/build/installer/qcow2.go`, `qcow2_test.go` | Reuse no-overwrite, compression and checksum techniques in `internal/nativebuild/`; replace bootc/image-builder invocation completely | Adapt platform, wrong-input, output-collision and compression-failure cases |
| `internal/build/release/inspection.go` | Reuse platform/source/content identity concepts; inspect our actual OCI archives and bundle, not predecessor RPM/release records | Extract applicable cases from release fixtures/tests; add explicit archive-format and tampering checks |
| `scripts/check-native.sh`, `scripts/soda-release-executor` | Reuse native-target guards, exact-SHA checkout and fresh run-directory patterns; keep our Podman build scripts | Adapt `scripts/check_native_test.go` patterns; write remote argument/target/phase tests |
| `tests/acceptance/check-native-service-ordering.sh` | Add a current installed check for real CoreOS/systemd/Quadlet units and boot-journal ordering errors | Replace `cloud-init.target` and old `forgejo-init.service` assertions; test nonzero native inspection results |
| `scripts/place-libvirt-iso.sh` | Optional later port only if libvirt placement becomes a real caller; preserve checksums, no overwrite, QEMU access and SELinux checks | Its focused placement tests, if selected; direct QEMU does not need this script |

Do not copy `internal/acceptance/record.go`, `verify.go`, `qualification_test.go`, `registry.go`, `fallback.go`, `cloud_init.go`, deletion scenarios, the old full `cmd/soda-acceptance` command tree, or the release publisher as dependencies of a small helper. Extract required ordinary file checks from their owners instead.

`scripts/prepare-native-iso-candidate.sh` **builds and publishes** before ISO creation. It is not a safe local candidate-building shortcut. The predecessor's own acceptance README also documents missing installed-onboarding, independent LAN and public-ingress coverage: successful source reuse or a zero exit from its itinerary is not inherited release qualification.

## 4. Source ownership and entrypoints

### Execution topology

```text
ARM laptop — source editing and explicit SSH coordination
    └── linux-infra.dimensionlab.net — recorded native x86_64 builder
        ├── fresh exact-revision checkout → native tools/build/check/bundle
        └── new run-owned CoreOS KVM guests → installation and native scenarios

Named developer client (the laptop can fill this role)
    └── separately approved LAN/Tailnet route → actual project IPs

Existing soda-test VM + backing image → preserved, not an acceptance template
```

The builder is a proposed reuse of the recorded target, not a new liveness or access check. No Soda services are installed on it. Client-side browser/SSH execution has its own explicit permission; its CPU architecture does not turn a native x86_64 guest into an ARM test. A route to the builder is not automatically a route to the guest's project subnet.

### Proposed source layout

Proposed new paths, created only when their milestone supplies a caller:

```text
tools/soda-artifacts/          small Go command: inspect/bundle/CoreOS media operations
internal/nativebuild/         artifact identity, input verification and media adapters
tools/soda-acceptance/         small Go command: explicit native acceptance phases
internal/acceptance/          ported VM/SSH/evidence helpers and concrete scenarios
scripts/native-remote.sh       thin exact-revision SSH phase entrypoint
appliance/locks/               reviewed CoreOS/tool/service-image inputs as needed
tests/installed/               extend existing checks and browser journey
tests/fixtures/                bounded project/Git/workload fixtures
```

Keep both infrastructure commands outside `cmd/`: the current build **and staging** scripts iterate every `cmd/*` directory and would otherwise install a QEMU/acceptance tool on the appliance. Add an explicit native tools build destination, excluded from the appliance rootfs and container contexts.

Retain `scripts/build-native.sh`, `check-native.sh`, `stage.py`, `render-provisioning.py` and `install-native.sh` as their current owners. Extend their concrete inputs rather than writing a competing build or installer. Keep shell for narrow SSH/native-command entrypoints and existing Python staging; new substantial orchestration belongs in Go, not a new Python/Rust framework.

Reuse `internal/process/` for non-sensitive ordinary commands where its API fits. Its command traces/error formatting are not safe for secret-bearing arguments or arbitrary provider output; do not force evidence-aware SSH exchange through it or change production callers incidentally.

Each phase has explicit inputs and returns native results directly. Evidence is an output for people, not a database reopened by the next step. There is no scenario registry, plan DSL, resident controller, SSH login-shell replacement or automatic provider registration.

## 5. Artifact and workspace contracts

### Outputs

| Output | Required content and identity |
| --- | --- |
| Native deployment bundle | Staged rootfs, required native/provider payloads, application image archives, matching installer source and notices. Preserve permissions and intended symlinks; exclude provisioning and runtime state. |
| Soda OCI archives | Real OCI-format project/dashboard archives, independently inspected as the selected Linux platform and source revision. A `.oci` suffix proves nothing: current `podman save` calls do not explicitly select a format. |
| Upstream application inputs | Resolve and record the selected Forgejo/Caddy platform digests; include their archives in the candidate bundle where needed to make media use the same bytes. No unobserved runtime tag drift. |
| Build metadata | Ordinary build-info file, input references, native tool versions, image manifest/config identities, package inventories and `SHA256SUMS`. These identify outputs; they contain no acceptance verdict or signature claim. |
| Installer ISO | Pinned CoreOS media plus the approved public Soda bootstrap/payload mechanism; checksums, required companion files and network requirements. Private unattended derivatives have different identities and storage. |
| QCOW2 delivery | Approved P01 kit or single-image contract, standalone disk with no missing backing dependency, optional fixed-parameter `.qcow2.zst`, checksums and first-boot instructions. Record which files must accompany it. |
| Native evidence | Private per-run log and concise observation summary tied to exact source, artifact hashes, architecture, target and client topology. No combined signed qualification schema. |

Build identity is not bit-for-bit reproducibility. Mutable RPM repositories, unresolved image tags and upstream resolver results must be recorded honestly. Do not claim fully pinned/offline/reproducible installation until the corresponding dependency closure is implemented and tested. Existing dependency locks and real `go.sum` remain authoritative; do not fabricate new digests or incidentally upgrade versions.

A network-assisted recipe may obtain host extensions during provisioning, but must record repository/signature inputs and the actual installed package/deployment identity. The CoreOS version alone does not identify the layered host.

### Isolation and retention

- Use an operator-selected native run root with separate fresh source checkout, build outputs, private inputs, guest disks and sanitized evidence. Identify each attempt by source revision, architecture and a fresh run ID. Reuse verified immutable download caches only as read-only inputs.
- A fresh checkout can retain the existing `.artifacts/native/ARCH` layout internally. This avoids a wholesale output-path rewrite; export only the allowlisted deployment payload, not that entire directory or repository.
- Use attempt-specific image references or serialize the concrete builder operations so concurrent work cannot race on `localhost/soda-*:dev`. Bind installation to the exact inspected images; do not trust a later tag lookup.
- Preserve `.artifacts/test-vm/` and `.artifacts/downloads/fedora-coreos.qcow2`. The existing overlay depends on that exact backing image. Never reuse, flatten, rebase, provision or export this live instance as a candidate template.
- New acceptance disks may use immutable verified backing images. Their published/delivered QCOW2 must be standalone or explicitly remain an internal overlay, never a misleading standalone download. Inspect the backing chain before any conversion.
- Refuse occupied output paths, symlinks to external state and unowned disks. Partial output is a failure retained for inspection, not an implicit success or an invitation to overwrite it on retry.
- `.gitignore`, `.containerignore` and bundle allowlists must keep private keys, password hashes, TLS private keys, OAuth configuration, browser profiles, VM disks and logs out of image contexts and distributable artifacts.

## 6. Milestones and dependency order

All milestone exits have two separate states: **source implemented, execution pending** and **observed native result**. Do not mark the latter from tests merely being authored.

| Milestone | Deliverable | Dependencies |
| --- | --- | --- |
| P01 | Confirm delivery/test contracts and source provenance | None |
| P02 | Safe command, SSH, evidence and remote phase boundary | P01's relevant interface decisions |
| P03 | Native QEMU/KVM lifecycle and fresh guest fixtures | P02; P05's verified base input for native boot proof |
| P04 | Exact native candidate build, inspection and bundle | P01; P02 for remote execution |
| P05 | CoreOS inputs and common first-install provisioning | P01, P04; P03 for native proof |
| P06 | Fresh installed host and operator browser baseline | P02–P05 |
| P07 | Routed two-user/two-project developer and workload journey | P06 plus an approved client route |
| P08 | Persistent project/service-state lifecycle checks | P07 plus stop/start/reboot authorization |
| P09 | CoreOS installer ISO recipe and fresh-disk acceptance | P03–P06 and ISO contract; product qualification also needs P07–P08 |
| P10 | Pristine QCOW2 delivery and independent-instance acceptance | P03–P06 and QCOW2 contract; independent of P09 construction; product proof also needs P07–P08 |
| P11 | Retained operator features and compatible follow-up validation | P06; P07 for project CLIs; routing/provider prerequisites per scenario |
| P12 | Exact-candidate x86_64 validation and handoff | P04–P11 for the selected coverage |
| P13 | Independent native aarch64 build/media/product evidence | Equivalent source ready; matching hardware and permission |

Recommended implementation order is P01–P08, then P09/P10/P11, then P12. Dependencies on **native proof** do not prevent authoring independent source: for example, P03's guest code and P05's verified-input code must both exist before their first boot/install exercise. Do not add a second downloader just to execute P03 first. P09/P10 media source can proceed in parallel once interfaces are settled; do not spend on repeated media builds before investigating the runtime/routing risks. P13 is not a dependency of any x86_64 milestone.

### P01 — Freeze scope, artifact meanings and interfaces

**Source work**

- Resolve the decisions in section 2, record the chosen QCOW2 deliverable and required companions, and distinguish public media from private per-instance provisioning.
- Inventory exact source families, current callers, notices and dependency additions. Do not bring `go-containerregistry` or the release specification graph merely to obtain a checksum helper; prefer existing/native inspection tools or a small justified dependency.
- Specify narrow phase inputs: source revision, native architecture, absolute fresh work/output paths, verified artifact inputs and explicit target/client connections. Secrets are file/stdin inputs, never argument values.
- Define x86_64/amd64 and aarch64/arm64 mappings once at the build boundary and preserve each native tool's actual vocabulary.
- Research and lock the selected CoreOS artifacts, Butane/coreos-installer interfaces, QEMU/firmware inputs and required image inspection/compression tools. Pin actual upstream metadata, URLs and verification material, not remembered names/hashes.

**Authored checks:** input/architecture/path validation and missing-tool detection before mutation.

**Exit:** an implementer knows what each output means, which commands may change state, and what remains a feasibility decision. No media promise depends on treating application OCI as a host OS.

### P02 — Command execution, evidence and native remote entrypoint

**Source work**

- Port the minimal command/SSH/evidence helpers and focused tests. Preserve empty arguments, quoting, stdin programs, distinct stdout/stderr and native exit identity.
- Require pinned builder and guest host keys, strict checking, explicit identities and bounded connection/liveness deadlines. Obtain project public host keys through the already trusted operator connection before first direct project access. Never refresh trust from an unauthenticated keyscan.
- Separate a guest's SSH readiness from Cockpit/dashboard readiness. Use the actual configured origins and trusted CA; no `--insecure` or browser TLS bypass.
- Make evidence writes exclusive and private. Sanitize sensitive diagnostics before writing or returning them; keep the final leak scan as an additional failure check. A log write failure cannot be mistaken for the expected command denial.
- Add a thin remote wrapper selecting only explicit build/check/inspection/acceptance entrypoints. Prepare one fresh exact-SHA checkout per run on the named native builder, not a copy of laptop binaries, dependencies or `.artifacts/`. Subsequent explicit phases verify and reuse that run's checkout/outputs; they must not reclone an empty tree for checks or silently switch revisions.
- Keep native phases separate: checking/building does not install/start a VM or invoke acceptance. Do not install a CI account, change the builder's login shell, publish anything or start automatic workflows.

**Authored checks:** remote shell metacharacters/empty strings, wrong revision/target, prohibited phases, cancellation, transport failure versus expected denial, unavailable evidence destination, secret leakage and output collisions.

**Exit:** source callers can execute one bounded native phase and retain truthful diagnostics without inherited release credentials or endpoint assumptions.

### P03 — Fresh native VM ownership

**Source work**

- Port QMP, process groups, cleanup and guest lifecycle with explicit ownership of disk, NVRAM, sockets and child processes. Resolve native executable/firmware inputs before creating files.
- Use headless Linux/KVM with native CPU architecture; fail if acceleration/firmware is unavailable rather than fall back to TCG/emulation. Keep QMP private and management forwards loopback-only.
- Add CoreOS Ignition delivery, new disk/overlay preparation, unique loopback ports and short private socket paths. No automatic tap/bridge/firewall creation in this helper.
- Reuse the same disk/NVRAM on an authorized restart; detach install media at the appropriate boundary. A fresh instance has fresh NVRAM and provisioning, not another instance's identity.
- Bound shutdown, cancellation and emergency termination separately. A failed QMP request must not lose the child process handle; failed cleanup remains an error. Unexpected VM death is not a successful test.
- Leave `scripts/test-vm.sh` and its persistent instance untouched initially. Do not adopt its PID file or require migrating it just to introduce disposable acceptance guests. Any later wrapper consolidation is separately reviewed.

**Authored checks:** native preflight before disk mutation, occupied ports/paths, immutable backing preservation, malformed QMP, partial launch, unexpected exit, cancellation, restart with retained NVRAM, and exact-resource cleanup.

**Native exit, when authorized:** boot and shut down a new minimal CoreOS fixture on x86_64 KVM, prove pinned SSH and cleanup, and leave the existing instance unchanged. This is harness proof, not Soda installation acceptance.

### P04 — Build and identify one native candidate

**Source work**

- Extend the existing build/check/stage owners for fresh runs; build infrastructure commands separately from appliance commands. Keep real dependency metadata and native Go/Node/pnpm/Tea/GitHub CLI inputs.
- Explicitly select OCI archive output format and inspect actual archive structure/platform. Add source revision/base references to Soda image metadata and tie native binaries/rootfs to the same checkout.
- Resolve selected Forgejo/Caddy images to architecture-specific content identities and arrange consistent staging/loading; stop installation from silently resolving different tags. Preserve current service ownership, mounts and configured origins.
- Assemble the allowlisted deployment bundle, installer and notices with correct modes/symlinks, checksums and build information. Record actual extension/package inputs without claiming an immutable repository snapshot exists.
- Add pre-install bundle checks for content/hash/platform mismatches, missing pieces and unsafe archive paths. Validate before host writes; preserve CoreOS's writable `/usr/local` mapping, parent ownership and delivered-path SELinux labeling.
- Ensure negative/refusal paths do not erase earlier outputs or project state. Fix actual merged-tree build defects in their owning source, not by bypassing checks or reusing stale binaries.

**Authored checks:** archive format, platform/source mismatch, tampering, missing payload, safe extraction including intentional staged symlinks, mode preservation, occupied output paths, infrastructure-tool exclusion and private-context exclusion.

**Native exit:** the exact merged candidate builds on x86_64, all applicable source/build/staging checks execute, and the inspected bundle matches those outputs. Record failures and corrections; do not extrapolate the earlier test counts to this tree.

### P05 — Common CoreOS provisioning and component installation

**Source work**

- Add verified input retrieval using pinned CoreOS stream/release metadata, compressed and uncompressed hashes where supplied, and upstream signature verification. Cache by content; never replace a backing image used by an existing VM.
- Adapt `render-provisioning.py` to separate public bootstrap source from private per-instance operator inputs, preserving exclusive mode-0600 output and strict Butane conversion.
- Keep the current native extension request, inspection, explicitly authorized activation reboot and first-install script as the shared installation spine. Add only the bounded bootstrap/payload transport needed by the chosen media formats.
- Define how the guest receives and verifies the exact P04 bundle: trusted transfer, read-only companion media or an explicitly configured authenticated/trusted local endpoint. Do not require artifact publication merely to test an ISO.
- Select one supported per-platform Ignition delivery mechanism. Prove how configuration fragments combine, when first-boot provisioning runs and when it stops; do not assume embedded disk config and QEMU `fw_cfg` automatically merge.
- Keep operator host access separate from Forgejo installation, OAuth bootstrap and private TLS activation. Preflight subnet, identities, artifact platform, required native packages and existing Soda state before component writes.
- Preserve first-install-only behavior and report partial native operations. Do not create a reinstall/reconciliation service to hide failures.

**Authored checks:** missing/wrong hashes, bad signatures, input overwrite, secret file permissions, invalid subnet/identity, absent native packages, incorrect architecture, existing installation and phase ordering.

**Native exit:** a fresh VM completes the documented CoreOS extension/installation sequence from the identified bundle without manual file-copy/SELinux repairs. A recovered earlier install does not satisfy this exit.

### P06 — Installed host and operator browser baseline

**Source work**

- Extend `tests/installed/host.sh` and add the current service-ordering check. Inspect the actual layered deployment, generated Quadlet services, helper socket, service UIDs/capabilities, ownership, labels, listeners and failed units.
- Preserve Fedora's root-only Cockpit PAM/SELinux transitions and the Accounts navigation override; check authorization independently of navigation hiding.
- Drive real Forgejo operator installation and `soda-setup`/`soda-activate` with approved private inputs. A manual operator interaction may remain explicit; pressing Enter does not turn an unobserved step into a pass.
- Reuse and extend `tests/installed/dashboard.mjs`: isolated trusted browser profile, actual Forgejo sign-in/consent/PKCE callback, session protections, Projects/Profile/People navigation and sign-out. Do not replace the browser path with seeded Soda session rows.
- Retain separate evidence for pre-activation loopback services and configured HTTPS services. A health check is not authentication evidence.

**Authored checks:** native inspection failures, required unit properties, session/role failures, certificate and origin mismatches, safe browser diagnostics and missing bootstrap inputs.

**Native exit:** the fresh target runs the exact bundle and completes the operator browser journey with verified TLS and enforcing host SELinux. No developer or network outcome is inferred.

### P07 — Routed developer, shared resource and workload journey

**Source work**

- Extend the real browser journey to create approved Alice/Bob fixtures through People, handle their native Forgejo initial-password change, authenticate independently and register public development keys. Generate fixture private keys only on the clients that need them.
- Create ordinary repositories through Forgejo's supported native flow. Exercise the populated repository picker and ownership checks, project creation, and **explicit joins for both creator and member**. Assert membership/connection guidance only after successful native account provisioning.
- Verify an actual route from the named client to the displayed project IP, then perform interactive SSH, exact-output noninteractive commands and bidirectional SCP/SFTP with pinned project keys. Do not use `ProxyJump`, host port remapping or a host-shell command as substitute direct-IP evidence.
- Assert Alice is project A's administrator and Bob is an ordinary member; neither has a human host account or appliance administration. Use a second project to check separate roots, host keys and effective runtime authority. Project-local UID numbers may coincide; do not import the predecessor's global host-UID assumptions.
- Adapt Git fixtures for personal homes, independent outbound Git credentials, actual remote commits/pushes and shared file read/write. Joining is not Git authorization; register only fixture public Git keys through the provider's native path.
- Extend `shared-tools.sh` beyond equal version/path strings: inspect one real shared installation's canonical path, file identity/permissions and executable resolution for both users, including noninteractive SSH. Bob consumes but cannot replace the root-owned installation; no duplicate home installation/download completes the check.
- Extend `workloads.sh` and its ordinary Compose fixture: image build, source bind-mount change, HTTP response change, PostgreSQL write/read and reachability from Bob plus the real client. Inspect the actual project-local engine and its socket boundary, not an unrelated host engine.
- Keep expected failures narrow: a lookup/SSH outage is not absence or authorization denial. Missing keys/provider/native failures must not produce joined/ready UI; do not turn these into an exhaustive recovery matrix.

**Native exit:** complete the current architecture's two-user journey plus second-project scoping on the selected route. If nested Podman fails, retain the concrete error and correct its owner. Investigate a scoped host fallback only after a demonstrated blocker; do not substitute a privileged parent, unrestricted socket or VM project backend.

### P08 — Ordinary persistence

**Source work**

- Capture bounded, secret-safe observations of container identity, project account/provider association, ownership/groups, public host-key fingerprints, homes, dirty/untracked Git work, shared files/installations, system configuration and a committed database fixture value.
- Stop/start the **existing** project through its native service after permission; compare returned observations and reconnect as both users. Repeat around a separately authorized appliance guest reboot.
- Validate the documented native workload restart path. If services need an ordinary explicit start after reboot, record that operation and then verify retained data; do not invent automatic service resurrection or rebuild/delete volumes to obtain a passing query.
- Check the second project remains intact and the same project containers/writable roots are used. Exclude volatile PIDs, boot IDs and timestamps from equality checks; use a changed boot ID only to establish that the reboot actually occurred.
- Keep secret values out of exported snapshots. Prefer public fingerprints and in-memory equality checks for necessary sensitive facts; do not port raw shadow/password-hash captures.

**Authored checks:** changed/removed fixture state, failed snapshot subprocesses, wrongly replaced containers, missing data and comparison failures. Snapshot generation failure must abort comparison rather than produce an empty matching result.

**Native exit:** intended state survives both lifecycle boundaries with matching container identities. No backup/restore, updates, deletion or disaster-recovery claim is made.

### P09 — CoreOS installer ISO

**Source work**

- Wrap the selected `coreos-installer iso customize` interface rather than the predecessor's Anaconda/SquashFS/SELinux patch stack. Inspect the exact selected tool version before encoding flags.
- Assemble the P01-approved public bootstrap/payload arrangement from the same CoreOS inputs and P04 bundle. Retain canonical branding only at supported surfaces; do not reintroduce an installer branding toolchain for unrelated Anaconda assets.
- Keep private provisioning out of generic media. Personalization writes a fresh private output, with its own checksum and explicit destination binding. Some customization options enable automatic installation: no hardcoded builder device, default physical disk or hidden unattended erase.
- Validate media metadata and expected boot/payload/config contents before launch. A structurally valid ISO is not an installed proof.
- In an authorized new guest, install only to a fresh run-owned virtual disk with a verified guest-device identity. Boot the installed disk without the ISO, finish the common P05/P06 sequence and record the actual deployed bundle/base.

**Authored checks:** invalid input format/platform, existing output, missing/mismatched payload, secret contamination, accidental unattended configuration and wrong destination selection.

**Native exit:** the identified ISO installs to a clean disk without recovery edits, boots without install media, and supports the selected P07/P08 journey. Record public ISO and any private derivative hashes separately. No GHCR push is required.

### P10 — QCOW2 delivery and first-instance independence

**Source work**

- Implement the P01-selected contract explicitly. For the recommended kit, retain the verified pristine upstream QCOW2 as an upstream artifact, package the exact Soda bundle and supported Ignition/bootstrap recipe alongside it, and document every required companion. Do not rename the disk and imply Soda is preinstalled.
- If a single preinstalled Soda image is required, first establish a supported CoreOS offline composition/first-boot mechanism in a separate design decision. A kit is then an intermediate result, not P10 completion. Do not solve this by booting the existing VM, removing a few credentials and exporting it.
- For either contract, keep the deliverable unpersonalized: no operator/root password hash, instance SSH keys, machine identity, initialized Forgejo/Soda databases, OAuth application, TLS private key, Tailnet state or runner registration.
- Inspect QCOW2 format, virtual size, integrity and backing references with native tools while the file is not in use. Produce a standalone disk, fixed-parameter compression if selected, and checksums for both disk and compressed bytes; verify decompression matches the disk.
- Boot independent writable instances from the same immutable delivery with separate private inputs. Verify each completes provisioning, has independent machine/SSH/application identity, and preserves its own state across reboot. Preserve each instance's identity on its subsequent boot.
- If disk growth is in the approved VM deployment contract, resize only a fresh fixture before provisioning and prove actual root filesystem capacity. File length or virtual-size metadata alone is insufficient.

**Authored checks:** backing dependency, in-use input, bad QCOW2/compression, checksum mismatch, private-state contamination, output collision, provisioning-mode mismatch and attempted instance reuse.

**Native exit:** the exact approved delivery works on fresh independent instances and completes the selected product journey. Record kit versus preinstalled-image semantics prominently; a compressed file alone is not VM acceptance.

### P11 — Operator features, branding, console and provider CLIs

**Source work and separately scoped native scenarios**

- Cockpit/Tailnet: verify real native state in a new authenticated root session; with approval exercise browser enrollment, pending/approval states, exit-node/LAN preferences and routed traffic. Attach cleanup ownership only to the guest/resource created by this run, not to an arbitrary existing enrollment.
- Forgejo advertisement: verify the real Git SSH listener before refresh, preserve browser/OAuth origins, and test the actual advertised clone URL from the intended client. A matching preference value is not connectivity proof.
- Runners: use explicitly approved disposable Forgejo/GitHub registrations, native service accounts/capacity, start/stop/restart and real provider-scheduled trusted jobs. Keep runner removal and provider cleanup explicit, independently authorized operations.
- Console: interactive operator welcome and private-origin safety; exact quiet noninteractive SSH/SCP/SFTP behavior. Printed origins must not be counted as listener observations.
- Branding: actual staged assets and installed pages, using the existing review/capture guidance. Optional renderer/component checks remain separate from the installed UI; generated or mock screenshots do not prove it.
- Project CLIs: native Tea/gh version/availability and permissions. Personal authentication/API compatibility is a separate provider-authorized check with independent per-user credentials, not proof supplied by a version string.

**Exit:** each executed feature has its own bounded evidence and cleanup result. Missing provider/network permissions are recorded as unverified, not replaced by fakes or silently included in an overall product pass.

### P12 — Exact x86_64 candidate handoff

- Freeze the actual source/bundle/media identities for the selected run. Do not build one revision, boot stale archives from another and attach the newer revision to the report.
- Execute the full applicable source/build/staging checks and each selected installation/product/operator path. Attribute shared underlying evidence carefully; a generic QCOW2 boot does not prove the ISO installation path, and an ISO install does not prove the QCOW2 kit's provisioning.
- Record invocation, native tools, architecture, base/layered deployment, artifact checksums, target and client topology, observed results, skipped/blocked work, corrections and cleanup. Retain failed attempts without relabeling them as successful.
- If a correction changes relevant source or delivered bytes, rebuild the affected artifacts and repeat the affected checks. Final candidate evidence must refer to the final bytes; historical results remain separately identified.
- Update installation, validation, local testing, implementation status and reuse guidance. Remove transitional duplicate helpers only after their replacement callers/tests are complete; no indiscriminate `.artifacts/` cleanup.
- Hand off local artifacts and sanitized evidence over the approved channel. No push, tag, registry publication, release signing or automatic CI is part of this milestone.

**Exit:** a named x86_64 candidate has an honest coverage report and usable delivery instructions. A partial operator/provider or network run is explicitly partial, not a ready-to-deploy release certificate.

### P13 — Native aarch64 follow-up

Use matching-native Linux/aarch64 hardware and the architecture's own CoreOS media, firmware, tool/package/provider inputs. Repeat build, both selected delivery paths and native product checks, with independent evidence and failures. Do not treat the ARM laptop, cross-compilation or emulation as a substitute for this installed target. Preserve useful x86_64 completion even if aarch64 access remains unavailable.

## 7. Acceptance coverage and evidence boundaries

| Outcome | Establishing observation | Insufficient substitute |
| --- | --- | --- |
| Native candidate | Exact source/platform, real build/check outputs and inspected bytes | Filename, `--platform` flag, old build log |
| Fresh installation | New disk, real CoreOS extension/installation sequence, cold boot without installer, intended deployed payload | Repaired existing `soda-test`, successful media generation |
| Reusable VM delivery | Independent instances with separate provisioning/identities and documented companion files | Exported initialized VM or compressed disk alone |
| Operator access | Real root Cockpit session/PAM/SELinux behavior and separately authenticated dashboard operator | Hidden Accounts link or active socket |
| Developer onboarding | People → Forgejo password/login → public key → owned repository → create → explicit join → native account | Directly seeded database, fixture-created Linux account, mocked UI |
| Direct project access | Named real client route plus pinned SSH/command/SCP/SFTP to displayed IP | Appliance-local SSH, QEMU forward, ProxyJump |
| Role/project separation | Positive authorized control and specific native/server-side denials; separate second project | Hidden button, transport failure, globally unique project UID requirement |
| Ordinary Git | Independent Git credential setup and real clone/commit/push/readback | Soda login or project SSH key alone |
| Shared installations | Same actual tool files/install tree consumed by both users without duplicate installation | Same version strings or a shared download cache |
| Native workloads | Correct project engine, real build/mount change and HTTP/database traffic from Bob/client | Compose parsing, listing a container, localhost-only health |
| Persistence | Same containers/accounts/files/tools and database fixture after real lifecycle operations | Recreated projects, empty matching snapshots, image rebuild |
| Tailnet/private networking | Actual configured/approved route and successful traffic; listener-consistent Git advertisement | Host enrollment or route preference alone |
| Runner operation | Native runner identity/capacity and provider-scheduled trusted job | Local fake job or installed runner binary |
| Console/branding/CLIs | Real native outputs/pages and separately authorized CLI authentication where claimed | Printed URL, component/mock screenshot or `--version` as login proof |

The primary run remains a straightforward ordered program, not a qualification framework. Only record success at a check's actual successful call site. Retain failure, blocked, not selected and not reached distinctions in ordinary logs/summary. Missing output is not a pass. Cancellation, evidence failure or cleanup failure must remain visible and return a failure for the invoked phase.

The plan adds no signed acceptance record or architecture-combining schema. Artifact metadata identifies bytes; the ordinary run report describes observations. Neither transforms local forwarding into routed access or source tests into installed-browser evidence.

## 8. Execution gates

The eventual execution brief must name each applicable target/action. These are operational permissions, not a new machine-readable permit system.

| Gate | Permission/input needed | Does not authorize |
| --- | --- | --- |
| Source implementation | Selected port scope and reviewed interfaces | Builds, tests, dependency installs or VM operations |
| Native build/check | Named Linux builder, native architecture, exact source, allowed dependency/tool retrieval and output paths | Soda installation on builder, VM boot, publication |
| VM preparation/boot | KVM access, selected firmware, fresh run-owned disk/ports and private provisioning | Existing disk replacement, host bridge/firewall changes |
| Installation/activation | Exact guest/destination, private operator inputs, subnet, extension install, specified reboot/service starts | Physical-disk install, unrelated service restart, production migration |
| Real client routing | Exact client, guest/appliance network, LAN/router or Tailnet route and approving authority | Public ingress or arbitrary host-network reconfiguration |
| People/projects/workloads | Disposable identities/repositories/project fixtures, Git keys, image/network use and workload writes | Unrelated accounts, automatic production cleanup |
| Persistence | Named existing fixture project stop/start and specific guest reboot | Container replacement, volume deletion, backups/restore |
| Providers | Specific Tailnet/Forgejo/GitHub resources, registration/job/exit-node actions and cleanup limits | Other enrollments, untrusted jobs, publication |
| Destructive cleanup | Exact run-created resources and retention decision | Recursive deletion of shared `.artifacts/`, caches/backing images, operator inputs or existing instance state |

Use source-only work to complete independent milestones while an execution gate is unavailable. Do not route held operations through CI, a background agent or another machine to bypass the gate.

## 9. Failure handling, secrets and cleanup

- All private provisioning, browser homes, credentials and instance disks remain restricted runtime inputs/state. Do not include them in distributed media, archive contexts, screenshots, logs or Git. Password hashes are secrets too.
- Never put passwords/provider tokens in argv, tracing or constructed URL query strings. Native OAuth redirects carry codes/state by protocol; omit or redact those query strings from browser diagnostics. Capture selected facts rather than full pages, cookies, container environments or provider responses.
- Sanitize streamed output before retention, including matches across write boundaries. A final exact-secret scan is only an additional diagnostic, not proof arbitrary unknown secrets cannot leak. Leakage fails evidence handling; do not announce successful sanitization as successful validation.
- Cleanup runs with independent bounded contexts even after cancellation. Attempt orderly guest shutdown first, then stop only owned remaining processes. Never terminate a PID merely because an old PID file names it.
- Separate process/network cleanup from disk/evidence retention. Failed private disks may be retained for authorized diagnosis; deletion is explicit. Keep operator-owned files and immutable bases. Report resources that remain after partial cleanup.
- Native negative tests require the expected result, not any error: e.g. account lookup absence is distinct from command/SSH failure. Preserve command outcome and evidence-retention errors independently.
- Repair source, rebuild and use a fresh attempt for a failed first install. Do not quietly patch a candidate VM and label it a clean install, invent rollback logic or rerun a partially successful provider mutation automatically.

## 10. Highest risks and stopping points

| Risk | Detection / response |
| --- | --- |
| No real client route through the current NAT-only VM | P07 is blocked until a specific routed topology is approved; do not weaken the SSH criterion. |
| Nested Podman/cgroup/fuse/security incompatibility | Investigate early in P07 on the selected native host; correct the mechanism or bring back a concrete scoped-runtime decision. |
| Misidentified OCI/archive/tag | Inspect bytes and explicit archive format in P04; bind runtime to exact content, not filename or mutable tag. |
| CoreOS/RPM/provider package drift | Verify upstream inputs and record actual layered packages; fail unavailable or incompatible inputs rather than silently upgrade or claim reproducibility. |
| Ignition composition or reusable-disk semantics unsupported | Resolve in P05/P10 against the selected upstream interface; do not export a used VM or claim the kit is a preinstalled image. |
| Accidental unattended disk installation | Private derivative plus exact guest-disk binding and explicit permission; no default builder/physical disk path. |
| Evidence contains credentials or fixtures share identity | Pre-write filtering, private input separation, artifact inspection and independent-instance tests; fail and investigate. |
| Existing persistent state mistaken for generated scratch | Separate directories, immutable backing inputs and exact-resource ownership; no global cleanup. |
| Old tests reinstate old product assumptions | Map every scenario to current architecture/feature guides; drop incompatible assertions and dependent callers together. |
| External provider/firmware/architecture unavailable | Record that coverage as blocked/unverified; keep independent source and authorized x86 work moving. |

## 11. Commit and handoff sequence

Use coherent source commits, each carrying its focused tests, callers and documentation. Suggested subjects follow the dependency order:

1. `Document native artifact contracts and selected predecessor reuse` — P01.
2. `Port bounded native SSH and evidence helpers` — P02 core.
3. `Add exact-revision native remote phase entrypoint` — P02 caller.
4. `Port isolated KVM guest lifecycle and regression fixtures` — P03.
5. `Build and inspect exact native deployment candidates` — P04.
6. `Unify CoreOS inputs and first-install provisioning` — P05.
7. `Exercise fresh installed operator and browser journeys` — P06 source.
8. `Add routed project, shared-tool and workload acceptance` — P07 source.
9. `Check persistent project state across native lifecycle` — P08 source.
10. `Add CoreOS installer ISO delivery recipe` — P09 source.
11. `Add approved pristine QCOW2 delivery recipe` — P10 source.
12. `Adapt retained operator and compatible follow-up checks` — P11 source.
13. Record actual x86 corrections/evidence separately as they occur — P12.
14. Record independent aarch64 source corrections/evidence when available — P13.

A commit title saying “check” or “exercise” describes code unless the handoff explicitly records execution. Update [implementation status](implementation-status.md) at each substantive source/native step: what changed, source attribution, tests authored, what actually ran, exact candidate/target, failures and next gate. Do not manufacture counts, hashes or PASS records. Commit/push/publication operations remain within their separately authorized scope.

## 12. Completion checklist

- [ ] P01 artifact semantics, network dependency, native profile and execution interfaces are explicit.
- [ ] Selected helpers and tests have current callers; incompatible release/account/Updates dependencies are absent.
- [ ] Infrastructure binaries are excluded from the appliance payload; existing persistent VM state is untouched.
- [ ] The exact merged x86_64 candidate builds and passes applicable source/build/staging checks.
- [ ] ISO and the approved QCOW2 delivery each have fresh-path proof tied to their actual bytes; instance identities are independent.
- [ ] Real Alice/Bob browser, direct-IP transport, shared installations, workloads, second-project boundaries and persistence are observed.
- [ ] Selected operator/provider/branding/console/CLI checks have actual evidence, with unselected or blocked coverage clearly separated.
- [ ] Failed attempts, privacy checks, cleanup outcomes and remaining risks are preserved truthfully.
- [ ] Current installation/validation/reuse guidance is consistent with the implemented tools and outputs.
- [ ] aarch64 status is independent; no x86 result implies sibling coverage, bare-metal proof or publication readiness.

## References

- [Existing implementation plan](implementation-plan.md), [native validation](native-validation.md), [installation](installation.md), [predecessor reuse](predecessor-reuse.md).
- [Developer environment](development-environment.md), [project services](project-services.md), [operator setup](operator-setup.md), [runners](runners-port.md), [project CLIs](project-clis.md).
- [Predecessor build/release guide at the reviewed revision](https://github.com/LevitateOS/soda-os/blob/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c/docs/build-and-release.md).
- [Predecessor acceptance guide and its coverage limitations](https://github.com/LevitateOS/soda-os/blob/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c/tests/acceptance/README.md).
- [CoreOS installer customization](https://coreos.github.io/coreos-installer/customizing-install/) and [ISO command interface](https://coreos.github.io/coreos-installer/cmd/iso/). These describe available upstream interfaces, not a version lock or native validation result; verify the selected version in P01/P09.
