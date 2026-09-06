# Native support and artifact porting plan

**Status: active support source implemented; source checks and native observations remain unexecuted.** P01–P06/P11 tools, callers and authored tests, plus P12/P13 reporting interfaces, are described in the [support guide](native-support.md). No native milestone exit or current-candidate product readiness is claimed. P07/P08 remain core redirects; P09/P10 remain conditional/unselected. Source implementation does not authorize builds, tests, installation, VM/provider operations or publication.

**Precedence:** [dashboard-implementation-plan.md](dashboard-implementation-plan.md) is the leading plan for the **core product**, including its production native environment integration—not just React screens. This plan owns only the surrounding support tools and retained host-operator integrations. If the plans conflict, the dashboard implementation plan wins; change this plan instead of forking the core's behavior, contracts or acceptance criteria.

**Objective:** selectively reuse the predecessor's native VM/SSH/evidence and artifact machinery to help build, install and observe the core product. Supply reusable infrastructure and operator-side results to U08/U20, not another product implementation or qualification pipeline. Keep the laptop as an editor/coordinator or named developer client; build and exercise native Linux targets on matching hardware.

**Quick navigation:** [ownership](#1-baseline-precedence-and-ownership) · [scope/decisions](#2-scope-and-decisions) · [reuse](#3-exact-reuse-inventory) · [entrypoints](#4-source-layout-and-entrypoints) · [artifacts](#5-artifact-and-workspace-contracts) · [milestones](#6-milestones-and-dependency-order) · [evidence handoff](#7-evidence-handoff-not-a-second-product-suite) · [execution gates](#8-execution-gates) · [risks](#10-highest-risks-and-stopping-points) · [handoff](#11-commit-and-handoff-sequence).

## 1. Baseline, precedence and ownership

- Original porting research: SodaOS `6f7b51e9b94e85ec2aae816e2217ba3d6d1b046b`, recorded in `9c8d672`; dashboard plan recorded in `55ce5cb`. Recheck the merged revision before implementing either track.
- Predecessor source: `soda-os` `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. Do not modify that repository. Retain attribution, licenses and provenance for actual reused source.
- The [core plan's coordination contract](dashboard-implementation-plan.md#coordination-with-native-support-porting), [architecture](architecture.md) and [deferred scope](deferred.md) govern. The [initial M01–M18 plan](implementation-plan.md) is historical, not a competing current assignment.
- [Implementation status](implementation-status.md) and [local testing](local-testing.md) record actual evidence. Earlier x86_64 builds and operator browser checks do not validate the merged tree or complete any new milestone.
- Preserve the [existing CoreOS installation path](installation.md): native operator services, separate Forgejo/dashboard/Caddy containers and persistent Rocky environments. This plan does not select a new host or project runtime.

### Single-owner split

| Responsibility | Leading owner | Native support contribution |
| --- | --- | --- |
| React routes, Go API/DTOs, OAuth/session/CSRF, upstream developer/admin adapters, Soda schema | U01–U19 as assigned in the core plan | Supply build/transport/fixture inputs; no alternative handlers, account model, browser runner or database writes |
| Production `internal/host`, project account/SSH setup, shared tools, nested workloads, persistence, image/lifecycle/resource policy | U07/U08 and approved E01–E03 | Run-owned VM/client transport and bounded observation utilities only; report runtime defects to the core owner |
| Product browser/People/repository/join/Git/SSH/workload/persistence tests and fixture meanings | Owning U feature; integrated U08/U20 | Invoke the existing core-owned entrypoints once on exact inputs; retain their results, without copying scenarios into a Go harness |
| Dashboard asset payload, application config/secret/migration semantics and cutover | U02/U03/U04/U18 | P04 packages the specified payload; P05 transports/invokes the specified installer/bootstrap steps; neither redesigns them |
| Build-tool execution, artifact inspection/bundling, native VM/QMP/SSH/evidence helpers | P01–P05 | Reuse concrete tools with real callers, outside the production appliance command set |
| Installed OS/service substrate observations | P06, against core-owned service contracts | Report host/package/unit/permission/listener observations for U08/U20; not proof of application login or usable projects |
| Stock Cockpit/Tailnet/local runner services, native console and native branding delivery | P11 | Preserve/adapt these outside integrations and their focused tests; upstream CI/admin screens remain U14/U16 |
| Optional CoreOS ISO/QCOW2 wrappers | Conditional P09/P10 | Reuse approved core artifacts/installers; do not gate core delivery or change host architecture |
| Native artifact/operator evidence by architecture | P12/P13 | Hand off results; U08/U20 own combined product acceptance and readiness claims |

A root helper used by the application is **core production code**, even though it is native. A QEMU/SSH helper used to test an appliance is **outside support tooling**. Likewise, Forgejo's runner/admin UI is a core upstream-backed feature; host runner capacity/control remains the Cockpit integration. Do not classify work by the word “native” alone.

Shared paths such as `scripts/build-native.sh`, `scripts/stage.py`, `internal/config`, `appliance/` and `tests/installed/` do not have two independent designs. Follow the core plan's ownership matrix and agree the exact interface change before editing a shared contract. Give a runtime/auth/schema defect to its U/E owner; do not repair it by weakening a support check.

## 2. Scope and decisions

### Included outside-support work

1. Bounded command/SSH/evidence utilities, exact-revision remote invocation and native QEMU/KVM ownership with adapted tests.
2. Inspection, checksums, source/platform attribution and allowlisted bundling around the existing build/stage owners.
3. Verified CoreOS inputs, private provisioning transport and fresh guest preparation using the existing installation spine.
4. Read-only installed OS/service observations and retained host-operator integration follow-ups.
5. Conditional CoreOS installer ISO/QCOW2 delivery wrappers with explicit semantics, not implicit product requirements.
6. Independent architecture support for these tools and their own native evidence. Useful x86_64 work need not wait for aarch64 access.

### Outside this plan

- Core frontend/backend implementation, Forgejo capability/scopes research, user/session lifecycle, product fixtures and browser/project acceptance: these stay in U01–U20.
- Project image/account/workload redesign, Fedora selection, browser lifecycle controls and resource policies: these stay in the core U/E decision register.
- A Soda bootc host image, Anaconda/Kickstart/RPM distribution builder, custom updater or the predecessor's reserved Updates platform.
- Registry publication, release tags, promotion/signing, architecture-combining qualification certificates, a release executor account and automatic CI.
- Old host developer/workspace accounts, PAM-backed Forgejo identity, managed checkouts, project deletion, B→A→B update tests and speculative runtime backends.
- Generic scenario registries/DSLs, workflow engines, resumable jobs, reconciliation or a second implementation of installed tests.
- Bare-metal, public-cloud or public-ingress claims inferred from a local KVM guest.

The predecessor builds a **bootc host OCI**, then derives an Anaconda network ISO and a QCOW2 through separate mechanisms. SodaOS builds **application/project OCI images**, not a bootable host OCI. Neither ISO-to-QCOW2 conversion nor application-OCI-to-host-image conversion is an assumed delivery path.

### P01 decisions, subordinate to the core contract

These are proposed defaults. A media decision must not block core implementation, the first U08 journey or independent support-tool source.

| Decision | Safe initial contract | Explicit decision/verification still needed |
| --- | --- | --- |
| Host mechanism | Existing upstream CoreOS plus native Soda installation | A Soda host OCI/bootc alternative needs a separate architecture decision, not a packaging shortcut. |
| Network dependency | Network-assisted installation; record required endpoints and actual package inputs | Offline delivery needs a complete host-extension/package/container closure and disconnected proof. Embedded Ignition alone is not offline support. |
| ISO meaning | No core ISO requirement; if selected, CoreOS media plus approved non-secret bootstrap/payload support | Public versus private/unattended derivative and exact disk-selection behavior; never ship silent disk erasure as the default. |
| QCOW2 meaning | No core QCOW2-delivery requirement; proposed kit is pristine CoreOS disk + exact Soda bundle + per-instance recipe | If a single preinstalled Soda image is required, prove a supported composition design; a kit cannot be relabeled as that image. |
| New native VM profile | Headless matching-native QEMU/KVM; explicit firmware, with proposed x86 UEFI/OVMF | Existing `soda-test` has no explicit OVMF setting. Do not change its firmware or infer BIOS/Secure Boot coverage. |
| Developer transport input | Named client and separately approved private route, supplied to U08/U20 | Product reachability is core-owned; neither QEMU forwarding nor host Tailnet enrollment proves the route. |
| Execution ownership | Exact builder, new run-owned guests and separate operator/provider grants | Name disks, ports, routes, accounts/resources and cleanup limits; flags/environment variables are not authorization. |

## 3. Exact reuse inventory

Predecessor paths below are relative to the pinned [predecessor tree](https://github.com/LevitateOS/soda-os/tree/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c). Recheck source/dependency closure before copying. A listed source is a reuse candidate, not a new mandatory framework.

| Predecessor source | Disposition / owner | Tests or techniques to retain |
| --- | --- | --- |
| `internal/acceptance/qmp.go` | P03: typed QMP handshake/results in `internal/acceptance/` | `qmp_test.go`, cancellation/read deadlines, malformed replies, missing socket |
| `internal/acceptance/processes.go`, `cleanup.go` | P03: exact child/process-group ownership and bounded finalization | `processes_test.go`, `cleanup_test.go`; unexpected VM exit is failure |
| `internal/acceptance/qemu.go`, `qemu_inputs.go` | P03: native preflight, disk/firmware/QMP; no GTK/Cocoa/cloud-init/fixed product forwards | `qemu_test.go`, applicable `preflight_test.go` cases |
| `internal/acceptance/guest.go` | P03: one owner per VM/disk/NVRAM, orderly restart and partial-launch cleanup | `guest_test.go`, `guest_process_test.go`, replacement failure, ISO/disk independence |
| `internal/acceptance/remote.go`, `command.go` | P02: literal arguments, stdin and distinct command outcomes; replace insecure trust behavior | `remote_test.go`, `command_results_test.go`, transport assertions; no refreshed unauthenticated host-key trust |
| `internal/acceptance/evidence.go` | P02: private exclusive writes and pre-write redaction | `evidence_test.go`, `evidence_boundary_test.go`, split-write redaction, unsafe path/symlink cases |
| `internal/acceptance/runner.go`, `itinerary.go`, `runner_init.go`, `runner_vm.go` | P02/P03: extract ordered invocation/finalization patterns only | Focused reporting/finalization tests; do not copy the product itinerary, release model or scenario registry |
| `internal/acceptance/product_scenarios.go`, `project_scenarios.go`, `fixtures.go` | **Core U08/U20 reference only; not a P port** | Core owner may reuse SSH/transfer/HTTP/fixture techniques in existing `tests/installed/` callers; no duplicate Go onboarding/workload suite |
| `internal/acceptance/preservation.go`, `preservation.sh` | **Core U08/U20 reference only; not a P08 implementation** | Core-owned bounded state comparisons and failed-snapshot checks; no old account/path/shadow catalog |
| `internal/acceptance/local_forwarded.go`, `tailnet.go` | P02/P11: transport distinction and run-owned operator enrollment, not project policy | Relevant focused tests, without old endpoints or host-wide peer ownership assumptions |
| `internal/build/installer/qcow2.go` | Conditional P10: no-overwrite/compression/checksum techniques in `internal/nativebuild/`; no bootc invocation | Adapt `qcow2_test.go`: platform, wrong input, collisions and compression failure |
| `internal/build/release/inspection.go` | P04: source/platform/content identity for actual archives, not release/RPM records | Applicable format/tampering fixture checks; no release schema dependency graph |
| `scripts/check-native.sh`, `scripts/soda-release-executor` | P02/P04: native guards, exact-SHA checkout, fresh run paths around current entrypoints | Argument/revision/phase tests and selected `scripts/check_native_test.go` patterns |
| `tests/acceptance/check-native-service-ordering.sh` | P06: current CoreOS/systemd/Quadlet service observations | Replace cloud-init/old Forgejo-init assertions; native command failures stay failures |
| `scripts/place-libvirt-iso.sh` | Unselected; reconsider only for a real libvirt caller | Direct QEMU needs no libvirt placement port |

Do not copy `internal/acceptance/record.go`, `verify.go`, `qualification_test.go`, `registry.go`, `fallback.go`, `cloud_init.go`, deletion scenarios, the old full `cmd/soda-acceptance` tree or release publisher as helper dependencies. `scripts/prepare-native-iso-candidate.sh` builds **and publishes**; it is not a safe local media shortcut. Predecessor acceptance gaps and old zero-exit results are not inherited product qualification.

## 4. Source layout and entrypoints

```text
Laptop — editing and explicit coordination; optionally a named browser/SSH client
    └── named native Linux builder (recorded x86_64: linux-infra.dimensionlab.net)
        ├── exact-revision checkout → existing build/check + support tools
        └── fresh run-owned CoreOS guests → OS/install observations
                                            + invoked core-owned U08/U20 tests

Actual developer client → separately approved LAN/Tailnet route → project IPs
Existing soda-test + its backing image → preserved, not an export/test template
```

This names a proposed reuse of the recorded builder, not a new liveness/access check. Do not install Soda services on the builder. Client CPU architecture does not determine the tested appliance architecture.

Implemented source paths with outside callers (media remains conditional):

```text
tools/soda-artifacts/       inspect/seal/bundle and verified CoreOS input commands
internal/nativebuild/      artifact identity and input verification; no media adapters
tools/soda-acceptance/      explicit VM/transport/evidence phases; invoke owned checks
internal/acceptance/        VM/QMP/process/SSH/evidence helpers, not core scenarios
internal/acceptance/remote_executor.py  embedded exact-revision native phases
appliance/locks/            approved CoreOS/tool/image inputs where needed
tests/installed/            existing owners: core product; P06 host; P11 operator
```

Keep infrastructure commands outside `cmd/`: current build **and staging** loops include every `cmd/*` directory in appliance delivery. Add a separate tools build destination excluded from rootfs and container contexts. Do not install a QEMU harness on the appliance accidentally.

Retain the existing `build-native.sh`, `check-native.sh`, `stage.py`, `render-provisioning.py` and `install-native.sh` entrypoints. P04 adds format/identity/bundle handling; U02 defines React asset packaging and U03/U04/U18 define application configuration/migrations. No competing builder/installer or copied application bootstrap. Keep shell for narrow entrypoints, existing Python staging, and Go for substantial support logic.

Reuse `internal/process/` only where its non-sensitive command API fits. Its traces/error formatting are not a secret-safe transport; do not force credentials/provider output through it or change unrelated production callers to suit the harness.

Each phase has explicit inputs and native results. Evidence is human-readable output, not a database reopened by later phases. Core tests remain at their owning entrypoints: the wrapper may invoke them with an approved target, but must not redefine routes, roles, expected behavior or mutate Soda SQLite directly to satisfy them.

## 5. Artifact and workspace contracts

### Outputs

| Output | Required content / identity |
| --- | --- |
| Native deployment bundle | Core-defined staged rootfs/assets/native/provider payloads and image archives, matching installer and notices; correct modes/symlinks; no provisioning/runtime state |
| Soda image archives | Actual explicitly selected OCI format, platform and source; a `.oci` suffix does not establish format, and current saves need inspection |
| Upstream application inputs | Recorded selected Forgejo/Caddy platform digests and matching loaded bytes; no incidental version change or silent later tag drift |
| Build information | Source/native tool/input references, image manifest/config identities, observed package inventories and `SHA256SUMS`; not a PASS/signature/qualification schema |
| Optional installer ISO | Selected CoreOS media/bootstrap/payload arrangement, checksums, companions and network requirements; private derivative separately identified |
| Optional QCOW2 delivery | Approved kit versus preinstalled-image meaning, standalone disk and any selected fixed compression/checksums; all required companions named |
| Native evidence | Private run log and bounded observations tied to source/artifact hashes, architecture, target and topology; core product results cited from their owner |

P04 may validate/package only the application payload established by U02/U03/U04/U18. The frontend build output, encryption-key provisioning, callback/origin rules, migrations and HTMX cutover are not inferred by an artifact scanner. Changes to a shared manifest or installer need the core owner's review.

Build identity is not bit-for-bit reproducibility. Record mutable RPM repositories, resolver results and actual layered package/deployment identity honestly. Existing locks and real Go metadata remain authoritative; no fabricated digests, incidental dependency upgrades or unsupported offline/reproducibility claim.

### Isolation and retention

- Use fresh run-owned paths for source, outputs, private inputs, disks and evidence; record source/architecture/run identity. Reuse verified immutable caches only as read-only inputs.
- Preserve the existing `.artifacts/native/ARCH` layout within a fresh checkout when useful; export only an allowlisted bundle, never the whole repository or artifact directory.
- Use attempt-specific image references or serialize concrete build operations to avoid racing on `localhost/soda-*:dev`. Bind installation to inspected bytes, not a later mutable-tag lookup.
- Preserve `.artifacts/test-vm/` and `.artifacts/downloads/fedora-coreos.qcow2`: the live overlay depends on that base. Never rebase, flatten, provision or export that existing instance as a candidate template.
- New fixture overlays may use verified immutable bases. A delivered QCOW2 must be standalone; inspect its backing chain before conversion, and never convert an in-use disk.
- Refuse occupied/symlink-escaping output paths or unowned disks. Partial output is a failed attempt retained for inspection, not permission to overwrite on retry.
- Exclude private keys/password hashes, instance TLS/OAuth/token-encryption secrets and personalized configuration, browser profiles, runtime databases, VM disks and logs from image contexts and distributable bundles. Non-secret source templates remain part of the core-defined payload. Permission-safe private per-instance input is not public media content.

## 6. Milestones and dependency order

These are **support deliverables**, not a replacement core milestone graph. Separate source implementation from executed source checks/builds and native observations. P07/P08 have no independent completion state; their product work is assigned to U08/U20.

| ID | Support deliverable / disposition | Source dependency or contract | Native observation prerequisite |
| --- | --- | --- | --- |
| P01 | Outside-tool interfaces, provenance and artifact meanings | Existing contracts; align changes with U01/U02/U03/U04/U18 | Review only; no target action |
| P02 | Command/SSH/evidence and exact-revision remote entrypoint | P01 relevant inputs | Named builder/client grants |
| P03 | Fresh QEMU/KVM fixture ownership | P02; P05 verified-base interface | P05 input retrieval + new-guest permission |
| P04 | Artifact inspection and allowlisted bundle | P01; core build/payload contracts; P02 if remote | Authorized matching-native build/check |
| P05 | Verified CoreOS input and first-install transport | P01; P04 bundle interface; existing core installer | P03 guest + installation/reboot grant |
| P06 | OS/service substrate observations | P02–P05; core service contracts | Fresh installed guest; activation only through core-owned steps |
| P07 | **Moved: developer/workload journey → U08/U20** | No P implementation | Core-owned permission/evidence boundary |
| P08 | **Moved: project persistence → U08/U20** | No P implementation | Core-owned stop/start/reboot permission |
| P09 | **Conditional:** CoreOS ISO wrapper | Explicit media decision, P03–P06 interfaces | Fresh exact-disk/media-install permission |
| P10 | **Conditional:** pristine QCOW2 delivery | Explicit deliverable decision, P03–P06 interfaces; independent of P09 | Fresh independent-instance permission |
| P11 | Host operator/native companion integrations | P06 substrate; current Cockpit/native contracts | Per-feature provider/network/browser grants |
| P12 | x86_64 support/artifact evidence handoff | Selected P04/P06/P11 outputs; P09/P10 only if selected | Named exact candidate and performed checks |
| P13 | Independent aarch64 support evidence | Same source/interfaces, native sibling inputs | Matching Linux hardware and grants |

Recommended support source order: P01 → P02, then P03/P04/P05 interface work → P06/P11 → P12, with conditional P09/P10 separately selected. P03 boot observation uses P05's input retrieval; P05 installation observation uses P03's VM. Those observation dependencies do not require circular source completion or a second downloader. P13 never blocks x86_64 work.

The core can proceed U01 → U02–U07 → U08 with existing authorized tooling. **U08, U18 and U20 do not require this whole port, ISO/QCOW2 delivery or P12 completion.** Consume whichever support outputs exist. P12/P13 report their own scope without waiting for U20 to declare the product verified; U20 can reuse that evidence when applicable. No circular “core waits for native qualification, native waits for core qualification” gate.

### P01 — Confirm outside-tool contracts and provenance

- Inventory selected helper callers/tests/licenses, source/native architecture inputs, absolute fresh paths and approved output meanings. Keep dependency additions minimal; do not pull in a release specification graph for a checksum helper.
- Record core-owned interfaces rather than redesigning them: staged payload/assets, installer arguments, private configuration inputs and core-test entrypoints. Agree any shared-file changes with the owning U milestone first.
- Define architecture vocabulary conversion once at the build boundary; preserve actual tool names (`x86_64`/`amd64`, `aarch64`/`arm64`).
- Research CoreOS/Butane/coreos-installer/QEMU/firmware/inspection inputs from actual upstream metadata. Media-specific locks/recipes wait for their delivery decision.
- Name phase effects and file/stdin secret inputs. A run root, environment flag or artifact filename is not permission.

**Authored checks:** path/architecture/input validation and missing-tool detection before mutation. **Exit:** bounded support interfaces and source provenance are documented; no core contract, media promise or host mechanism is silently replaced.

### P02 — Command, SSH, evidence and remote invocation

- Port only literal-argument/stdin transport, separate stdout/stderr/exit results, bounded connection/cancellation and private evidence helpers with focused tests.
- Require explicit identities and pinned builder/guest keys. Product host-key retrieval/display/verification belongs to U07/U08; accept their trusted public-key result, never silently refresh trust with an unauthenticated keyscan.
- Separate SSH readiness from optional web checks. Use configured origins/trusted CA; no insecure HTTPS/browser bypass or hardcoded predecessor endpoint.
- Sanitize before returning/retaining diagnostics, including split-write matches. Keep exclusive private destinations, reject escaping/symlinked paths, and distinguish evidence failure from expected command denial. A final leak scan is defense in depth only.
- Add thin exact-SHA remote invocation of explicit existing build/check and owned installed-test entrypoints. Prepare one fresh checkout per run; later explicit phases verify/reuse it without silently switching revision or copying laptop binaries/dependencies/private artifacts.
- No automatic VM/install/provider action after a build, release publisher, CI account, login-shell replacement, scenario registry or duplicated core browser suite.

**Authored checks:** empty/metacharacter arguments, stdin, wrong revision/phase/target, host-key failure, cancellation, transport versus expected denial, private-path collisions and redaction failures. **Exit:** one authorized phase can return truthful diagnostics; a transport unit test is not product proof.

### P03 — Fresh native VM ownership

- Port QMP/process/guest ownership for new disks, NVRAM, sockets and child process groups. Resolve native executable/firmware requirements before mutation; use P05's verified base interface.
- Use headless matching-native KVM; fail instead of silently using emulation. Keep QMP private and management forwards loopback-only. No implicit tap/bridge/firewall creation.
- Prepare fresh disk/provisioning/ports; on an authorized restart reuse that instance's disk/NVRAM and detach installer media at the correct boundary. Never share instance identity across fixtures.
- Bound orderly shutdown and emergency owned-child termination independently. Unexpected exit, failed QMP or partial cleanup stays an error; an old PID file alone is not ownership.
- Leave `scripts/test-vm.sh`, its persistent guest and backing files unchanged initially. Do not require consolidation or adoption of that live VM to create disposable fixtures.

**Authored checks:** preflight-before-write, occupied paths/ports, immutable bases, partial launch, malformed QMP, unexpected exit, cancellation, retained NVRAM and exact cleanup. **Native exit, when authorized:** a fresh minimal CoreOS guest boots with pinned SSH and shuts down safely; this proves the fixture, not Soda workflows.

### P04 — Inspect and bundle the core's native artifacts

- Extend existing build/check/stage entrypoints only for concrete format/identity/bundle needs; compile infrastructure tools separately. U02 owns adding the React build/asset payload and dashboard-only build path.
- Select/inspect actual OCI archive format/platform, add source/base attribution and bind binaries/rootfs/assets to one checkout. Retain real tool/dependency metadata.
- Resolve the selected Forgejo/Caddy platform identities without upgrading them. Coordinate any service-image reference change with the core installer/config owner; no separate tag policy hidden in the wrapper.
- Assemble the allowlisted bundle, matching installer/notices and checksums with modes/symlinks intact. Validate hash/platform/source/content and extraction safety before host writes; retain `/usr/local` mapping, parent ownership and delivered-path SELinux behavior.
- Refuse stale/mismatched/missing outputs. Route actual core build/config/runtime defects to the owning U/E milestone; do not bypass checks, fabricate frontend output or recreate project state.

**Authored checks:** formats, identity/tampering, missing assets/payload, safe extraction/symlinks/modes, occupied paths and exclusion of tools/private inputs. **Native exit:** the named bundle matches real matching-native build/check outputs; it is not a core U08/U20 PASS.

### P05 — Verified CoreOS inputs and installation transport

- Add one verified CoreOS retrieval/cache path with selected upstream hashes/signatures and compressed/uncompressed identity where provided. Never replace a live backing input.
- Adapt only the provisioning/transport portions of `render-provisioning.py`: separate public bootstrap source from mode-0600 per-instance inputs, strict Butane conversion and exclusive outputs. Keep operator-only host access.
- Reuse the existing native extension request, explicit activation reboot and first-install script. Deliver/verify the exact P04 bundle through trusted transfer, companion media or an approved trusted endpoint; no mandatory publication.
- Verify one actual per-platform Ignition mechanism and fragment/first-boot behavior; do not assume disk config and QEMU `fw_cfg` merge automatically.
- Application setup, OAuth grants, TLS/config/key files and migrations use the core-owned commands/contracts. The support layer may invoke those explicit steps with permission but must not copy bootstrap API logic, insert sessions/users or rerun first-install over existing state.
- Preflight architecture/packages/subnet/identity/existing state, report partial native results and preserve first-install refusal. No reinstall/reconciliation daemon.

**Authored checks:** bad/missing hashes/signatures, overwrite, private file modes, invalid input/package/platform, existing install and phase ordering. **Native exit:** a fresh target completes the current CoreOS/component installation without copy/label recovery edits; core authentication/workflows still need their U evidence.

### P06 — Installed OS and service substrate

- Extend the existing host check and adapted service-ordering check for actual layered packages, generated Quadlets, helper socket, service UIDs/capabilities, ownership/labels, listeners and failed units.
- Check root-only Cockpit PAM/SELinux behavior and Accounts navigation independently: hiding a link is not authorization. Detailed Tailnet/runner/browser behavior stays P11.
- Observe core-defined configuration/permissions, not a parallel application's expected defaults. Record pre-activation loopback versus post-activation HTTPS distinctly; use only core-owned setup/activation steps if those are separately authorized.
- Hand off exact origins/CA reference, target identity and observed host readiness to the core tests. Do not extend `dashboard.mjs`, create People/repository fixtures or implement OAuth/session assertions under P06.

**Authored checks:** failed native inspection, service ordering/properties, restricted access, bad certificate/origin and missing inputs. **Native exit:** accurate fresh-host observations with host SELinux enforcing. A live listener or successful health response is not login, permission or project acceptance.

### P07 — Routed developer, shared resource and workload journey

**Moved to core U08, with final repetition/coverage owned by U20. No P07 implementation or independent exit.** The useful predecessor techniques and stronger direct-IP/shared-installation/workload assertions are incorporated in the [core native proof detail](dashboard-implementation-plan.md#core-owned-native-proof-detail). P02/P03 supply transport/fixtures; they do not own accounts, browser steps, project policy or runtime corrections.

### P08 — Ordinary persistence

**Moved to core U08/U20. No P08 implementation or independent exit.** Bounded snapshot/comparison techniques, dirty work, public host-key and service-data observations are incorporated in the same [core proof detail](dashboard-implementation-plan.md#core-owned-native-proof-detail). P03 may restart an authorized appliance fixture; project lifecycle expectations and persistence assertions stay with the core. This move does not remove the required proof or reopen recovery/deletion scope.

### P09 — Conditional CoreOS installer ISO wrapper

**Gate:** separately approved media contract; not a prerequisite for U08/U18/U20 using the current installer.

- Inspect and wrap the selected `coreos-installer iso customize` interface, not the predecessor's Anaconda/SquashFS/SELinux patch stack. Reuse P05 inputs and the core-approved P04 payload.
- Keep public media unpersonalized and document companions/network needs. Any private derivative has a fresh restricted output, separate identity and explicit target selection. No default physical disk or hidden unattended erase.
- Validate platform/boot/payload/config contents and secret exclusion. Use canonical branding only at supported surfaces; do not start an unrelated installer artwork toolchain.
- With exact guest-disk permission, install to a fresh run-owned disk, boot without media and verify P05/P06 substrate results against the delivered bytes.

**Authored checks:** wrong formats/platforms, occupied output, missing/mismatched payload, private contamination and unintended unattended settings/destinations. **Native exit:** the selected ISO path installs/boots the intended payload without recovery edits. Claiming product support for that path additionally needs the core's U08/U20 tests on it—not a duplicated P product suite.

### P10 — Conditional QCOW2 delivery and instance independence

**Gate:** separately approved kit versus single-image contract; independent of ISO construction and not a core cutover gate.

- For the proposed kit, preserve the verified upstream disk's identity and supply the exact bundle/per-instance recipe alongside it. Do not rename it to imply preinstalled Soda. A single preinstalled deliverable first needs a supported composition/first-boot design; a kit is not its completion.
- Never export the existing initialized VM or “sanitize” a few files to make a template. Exclude root hashes, SSH/machine identity, initialized application databases, OAuth/TLS/encryption secrets, Tailnet state and runner registrations.
- Inspect non-running QCOW2 integrity/size/backing chains, produce standalone delivery and optional fixed-parameter compression with both checksums, and verify decompression equality.
- Boot fresh independently provisioned instances. P10 checks disk/provisioning/native machine/SSH identity independence; core U04/U08/U20 tests establish application/session/data separation without a second P account/bootstrap implementation.
- Resize only a newly owned fixture if disk growth is selected, and observe filesystem capacity rather than just metadata.

**Authored checks:** in-use/backing-dependent inputs, bad format/compression/hash, output collision, private state and provisioning/instance reuse. **Native exit:** exact approved delivery provisions independent native instances with retained identity on subsequent boot. Product acceptance is invoked from the core owner when that delivery is claimed; compressed bytes alone prove neither.

### P11 — Retained host-operator and companion integrations

- Cockpit/Tailnet: preserve page/backing logic, real native state, root-session PAM/SELinux and separately approved enrollment/approval/exit-node/LAN/routing behavior. Do not create a Tailnet identity database or rewrite it into the dashboard.
- Forgejo advertisement helper: inspect the real Git SSH listener, preserve configured browser/OAuth origins and verify its advertised endpoint from the intended client. Coordinate any shared config/client change with U04/U06; do not alter core login scopes or permissions.
- Local runner services: approved native accounts/capacity, registration/start/stop/restart and provider-scheduled trusted-job evidence, with explicit removal/cleanup. This is not the U14 Actions UI or U16 Forgejo administrator API.
- Console/native branding: interactive welcome, quiet noninteractive SSH/transfers, safe origins, canonical asset delivery and the existing opt-in review/capture tests. Dashboard layout/rendering/screenshot assertions remain U02/U19/U20.
- Provider CLIs: build/delivery/version/permission support for the selected Tea/gh inputs. Personal credentials/API use inside projects is core U08/U20 acceptance; do not create a separate P user fixture or credential broker.

**Exit:** each retained outside integration has its own source/native evidence and cleanup status. Unavailable external grants remain unverified. Report any change to shared project-image packaging to the core owner; E01/profile selection is not authorized here.

### P12 — x86_64 support evidence handoff

- Freeze actual source/bundle/selected-media identities and record performed build/format/install-substrate/operator checks, native tools, topology and cleanup. Optional media is included only when selected and exercised.
- Core product runs are cited by their owning U08/U20 entrypoint, revision and exact inputs, not copied or independently certified. P12 can finish its scoped handoff while product evidence is still pending.
- Preserve failures and skipped/not-reached work. A source/byte correction requires the affected outputs/checks to be repeated before reusing earlier evidence; do not attribute an old bundle to a new revision.
- Update installation/validation/local-testing/reuse/status guidance. Retire duplicate support helpers only with complete callers/tests, never global artifact cleanup.
- Hand off artifacts/private evidence over the approved channel. No push/tag/registry/release/signing/automatic CI is part of this milestone.

**Exit:** a useful, truthful support report for the exact x86_64 candidate. U20, not P12, owns combined product readiness; a partial run is not a deployment certificate.

### P13 — Independent native aarch64 support evidence

Repeat the selected support build/fixture/provisioning/operator/media work using matching-native Linux/aarch64 and its own CoreOS/firmware/package inputs. Hand off independent observations to U20; invoke core-owned product tests only within their permission scope, without reimplementing them. Laptop coordination, cross-compilation or emulation is not sibling appliance evidence. Unavailable aarch64 access does not cancel useful x86_64 results.

## 7. Evidence handoff, not a second product suite

| Observation | Owner / handoff | Insufficient substitute |
| --- | --- | --- |
| Exact source/platform/native artifacts | P04/P12/P13 → U08/U20 | Filename, `--platform`, old log |
| Fresh OS/component install and service facts | P05/P06 → U08/U20 | Repaired existing VM, generated media alone |
| Optional media and native instance independence | P09/P10 → selected U20 delivery coverage | Exported initialized VM, compressed disk, unobserved application independence |
| Native Cockpit/Tailnet/runner/console/branding/CLI delivery | P11 → U20 retained-integration coverage | Hidden navigation, fake job, printed URL or version as authentication proof |
| Browser/OAuth/People/repository/join/roles | Core U04–U08/U16/U20 only | Seeded session, second harness flow, listener health |
| Direct project-IP/Git/shared tools/workloads/persistence | Core U08/U20 only | QEMU forward, ProxyJump, same version strings, unrelated engine or replaced container |

Each observation records owner/milestone, source revision, relevant artifact hashes, native target/client, actual invocation, result and retained evidence location. Reuse identical evidence by reference; do not count it twice or infer that one installation path proves another. Core changes invalidate affected evidence even if the support tool is unchanged.

Use ordinary logs and a concise report with failed, blocked, not-selected and not-reached distinctions. Missing output, cleanup/evidence failure or cancellation is not success. No scenario registry, signed qualification record or architecture-combining verdict is added. The core plan owns product acceptance; the shared [native validation guide](native-validation.md) remains operational instructions, not a third roadmap.

## 8. Execution gates

The execution brief must name actual targets/actions. These are permissions, not a new machine-readable permit platform.

| Gate | Required scope | Does not authorize |
| --- | --- | --- |
| Source implementation | Selected port and reviewed core interfaces | Builds/tests/dependency installation/VM operations |
| Native build/check | Exact native builder/revision, tool retrieval and outputs | Soda install on builder, guest boot, publication |
| VM boot | KVM/firmware, run-owned disks/NVRAM/ports/private provisioning | Existing disk overwrite or host networking changes |
| Install/activation | Exact guest/destination, inputs, extension install and named restarts/reboot | Physical installation, unrelated services, blind reinstall |
| Client routing | Exact client/guest/LAN or Tailnet route and approving authority | Public ingress or arbitrary bridge/firewall changes |
| Core product test invocation | U08/U20-approved users/repos/projects/keys/workloads and entrypoints | Separate P fixtures, production cleanup or database seeding |
| Persistence invocation | Core-owned fixture stop/start and exact guest reboot | Project replacement, volume deletion or disaster recovery |
| Operator/providers | Specific enrollments/registrations/jobs/exit nodes and cleanup | Other resources, untrusted jobs, publication |
| Destructive cleanup | Exact owned resources and retention decision | Shared `.artifacts/`, live bases, operator inputs or unrelated state |

A held gate stays held; do not use automatic CI, another machine or a background agent to bypass it. Author independent source while access is missing and record the gap honestly.

## 9. Failure handling, secrets and cleanup

- Keep private provisioning/browser homes/credentials/disks restricted; never include them in public media, image contexts, screenshots or Git. Password hashes and encryption keys are secrets too.
- No credentials in argv/traces/constructed URLs. Native OAuth codes/state appear in redirects by protocol; omit/redact query strings from diagnostics. Capture selected facts, not full provider responses/pages/cookies/container environments.
- Sanitize before retention, across streaming boundaries. An exact-secret scan cannot prove arbitrary unknown secrets absent. Leak/evidence handling failure remains failure, not a successful product result.
- Use independent bounded cleanup contexts after cancellation; orderly shutdown first, then exact owned children. Never trust a stale PID file as authority to kill a process.
- Separate process cleanup from private disk/evidence retention; report leftovers and delete only with explicit resource ownership/permission. Keep failed attempts for approved diagnosis without reusing them as clean installations.
- Expected denials need their specific result; SSH/lookup failure is not proof of absent users or forbidden access. Preserve command and evidence errors independently.
- Fix core defects in the core's owning source and support defects here, then rebuild/repeat the affected check. Do not manually patch a failed candidate and call it a clean install or automatically retry ambiguous provider mutations.

## 10. Highest risks and stopping points

| Risk | Response / owner |
| --- | --- |
| Core/native scope drifts into duplicate APIs, fixtures or readiness gates | Dashboard plan wins; route work to the U/E owner and remove the duplicate P task. |
| No real developer route or nested-runtime incompatibility | U08 owns investigation/correction; P02/P03 provide transport/VM support, never weaker substitute evidence. |
| Misidentified archive/tag/source | P04 inspects bytes/format and coordinates core loading contracts; no filename/tag-only trust. |
| CoreOS/package drift | P01/P05 verify inputs and record actual layering; no silent version change or reproducibility claim. |
| Unsupported Ignition/media composition | P05 and conditional P09/P10 resolve actual interfaces; no used-VM export or architecture switch. |
| Unattended disk erasure | Explicit private derivative and exact owned guest disk; never default builder/physical disk. |
| Secret leakage or mixed instance identity | P02/P04/P10 boundaries plus core-owned identity tests; fail and investigate. |
| Existing VM mistaken for scratch | Immutable bases, fresh ownership and exact cleanup; never global `.artifacts/` removal. |
| Provider/firmware/architecture unavailable | Record scoped block and continue independent source/authorized native work. |

## 11. Commit and handoff sequence

Keep coherent support commits with callers/tests/attribution. Suggested grouping follows P01 interfaces, P02 transport/remote, P03 guests, P04 bundles, P05 inputs/install transport, P06 host observations and P11 operator integrations. Conditional P09/P10 media follow separate selection. P12/P13 record actual support evidence when execution occurs. There are **no P07/P08 product-test commits**; those changes belong to U08/U20.

Before changing shared configuration/build/staging/test paths, name the U/P owner and input/output contract. Merge coordinated changes rather than create another builder, fixture directory or scenario runner. Handoff records changed paths, authored versus executed checks, exact targets/bytes, failures, preserved state and the next allowed step in [implementation status](implementation-status.md). A commit saying “exercise” is test source unless actual execution is recorded. Git/publication permissions remain separate.

## 12. Completion checklist

- [x] Core precedence, shared-file interfaces and active/conditional/moved P dispositions are explicit.
- [x] Selected helpers/tests have actual outside callers; no copied core scenario/release/account/Updates framework.
- [x] Source build/staging guards exclude infrastructure binaries/private inputs from appliance payloads; existing VM/base remain untouched. Actual packaging/native verification is pending.
- [ ] Exact native artifacts and selected OS/operator observations have revision/byte/target-specific evidence.
- [ ] Media paths have independent proof only if approved; no media or sibling-architecture gate blocks core work.
- [x] U08/U20 own developer/workload/persistence assertions and overall product readiness; support reports link, not duplicate them.
- [x] Source reports preserve failed/missing/unselected work and privacy/cleanup failures; native limitations remain visible.
- [x] Installation/validation/reuse guidance and shared build contracts follow the leading core plan. See the source-only handoff; no execution is implied.

## References

- [Leading core plan](dashboard-implementation-plan.md), [native validation](native-validation.md), [installation](installation.md), [predecessor reuse](predecessor-reuse.md), [historical M plan](implementation-plan.md).
- [Project services](project-services.md), [operator setup](operator-setup.md), [runners](runners-port.md), [project CLIs](project-clis.md), [branding review](branding-review.md).
- [Predecessor build/release guide](https://github.com/LevitateOS/soda-os/blob/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c/docs/build-and-release.md) and [acceptance guide/limitations](https://github.com/LevitateOS/soda-os/blob/bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c/tests/acceptance/README.md), at the reviewed revision.
- [CoreOS customization](https://coreos.github.io/coreos-installer/customizing-install/) and [ISO interface](https://coreos.github.io/coreos-installer/cmd/iso/): verify the selected version before implementing conditional media, not runtime evidence or authorization.
