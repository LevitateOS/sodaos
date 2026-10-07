# Build, installation and operational joins

The source connects command discovery, browser builders, staged services and
payload identities to their installation consumers. One installed-host script
still checks obsolete executable paths. Its documentation also overstates the
identity of running services: checking stored images and selected files does not
establish which image a running container uses. These are separate corrections;
neither establishes that a particular installed machine failed.

Source: `bc28a07c38567f0f8b0ae7939aedb7b4d6f86e4d`, tree
`d5cecabf0409f634debc28acec080224d7bd4fe4`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 already dirty
guidance files and 39 dependency inputs retain their recorded bytes. This is a
source-only audit, with no builds, tests, dependency resolution, network, database,
provider, VM or installed operations. Existing [tasks](implementation-tasks.md)
and [lanes](implementation-lanes.md) remain the execution plan.

Discovery selected **111 build/configuration/service/payload/check paths**, not
111 fresh body reviews. Three Luna medium primaries traced native, release and
browser joins; each independently challenged another packet. Coordinator source
tracing and a small static selector check supplied the shared joins below.
Unchanged saved Cargo metadata supplies **28 workspace packages and 38 binary
targets**. All **22 explicit package/bin requirements** in the runtime, rootfs
tool and outside-tool tables exist in that metadata. This checks target names,
not the complete command discovery algorithm, feature closure, compiler results
or shipping count; no Cargo command was run here.

## Producer, shipping and invocation chain

| Responsibility / defining source | Producer and actual consumer; evidence boundary |
| --- | --- |
| Command discovery: [sys.rs](../../../lib/soda-release-image/src/sys.rs), [build_compile.rs](../../../lib/soda-release-image/src/build_compile.rs) | Snapshot Cargo metadata validates Rust package/bin/default-feature admission. `select_commands` also discovers Go command directories. `compile_soda_commands` dispatches each selected owner and adds moved Rust runtime bins; fixed rootfs/outside-tool tables own destinations. Workspace bin count is not the shipped set |
| Compiler: [production_compile.rs](../../../lib/soda-release-build/src/production_compile.rs) | Production Rust uses `cargo build --release --locked`, then reads the source snapshot's `target/release/<bin>` and stages mode-0755, architecture-checked ELF. This build can resolve dependencies over the network; only metadata selection uses `--offline`. Go uses `-mod=readonly -trimpath -buildvcs=false` and an explicit output path. These recipes are source evidence, not executed builds |
| Shared compiled outputs: [payload_stage.rs](../../../lib/soda-release-image/src/payload_stage.rs) | Discovered command outputs under `rootfs/usr/libexec/soda` are hard-linked to native `bin` for staging. Extension backend reuses the compiled command; dashboard's Go executable is also packaged at container `/usr/local/bin/soda-dashboard`. `tools/soda-installer` hard-links the compiled `soda-install`; sorted `tools.json` records tool hashes/modes/source identity. There is no second installer compilation |
| Host services/configuration: [prepare.rs](../../../lib/soda-release-image/src/prepare.rs), [complete.rs](../../../lib/soda-release-image/src/complete.rs) | Fixed file maps copy units/configuration into the rootfs and rewrite vendor `/usr/local/libexec/soda/` references to `/usr/libexec/soda/`. `/usr/bin` links expose setup, tailnet and installer commands. Completion binds Quadlets and extension-install service to recorded local image identities, retaining `--pull=never`. Authored pre-transformation paths alone do not show an installed service defect |
| Service/socket/config custody: [host units](../../../system/host/services/), [Identity unit](../../../system/host/services/soda-identity.service) | Host socket activation joins the root service to root:soda mode-0660 socket. Identity's named admin/runtime sockets join its inherited listeners; `/etc/soda/identity.json` is deliberately operator-provisioned and conditionally enables those units. Setup/defaults and operator credentials have different producers. Missing automatic Identity configuration is not itself a broken join |
| Project/helper commands: [production_assets.rs](../../../lib/soda-release-build/src/production_assets.rs), [Project Containerfile](../../../system/project/Containerfile), [term_create.rs](../../../cmd/soda-project-terminal/src/term_create.rs) | Project tools are compiled before staging and consumed by the image recipe. Host and guest helper outputs can be separate builds; terminal admission checks a held host executable against the guest-supplied hash. Preserve that actual check rather than assume the two recipes always emit identical bytes |
| Maintenance/install: [PG units](../../../system/host/services/soda-postgres-init.service), [load-console.sh](../../../system/host/installer/load-console.sh), [installer](../../../cmd/soda-install/src/) | PG initialization precedes Forgejo; backup has its explicit timer and restore is an operator action. Media loader admits hash-pinned console bytes into a private file before publishing/executing them. Installer media/release/architecture checks and six-image verification precede disk writes; configuration invokes the separately staged setup/activation commands |

Keep the existing Go server/domain/coordinator/Store/client and Rust
system/privileged/runtime owners. GUIDANCE-01/03/04/05/06 in the
[decision register](review-assignments.md#guidance-conflicts-and-controlling-decisions)
continue to control language/ownership; GUIDANCE-08 preserves intentional
all-page Forgejo presentation. No universal target manifest, linker package or
new state owner follows from these joins.

## Browser products and their actual consumers

Forgejo presentation and the Soda extension have separate producers and routes.
[Production assets](../../../lib/soda-release-build/src/production_assets.rs)
invokes [build-forgejo.ts](../../../scripts/build-forgejo.ts) using the
[presentation map](../../../frontend/forgejo/payload.json). Inventory closure
checks connect authored modules to emitted ESM and `@build/forgejo-js` staging
entries. The Forgejo Containerfile copies the custom tree to
`/usr/share/soda/forgejo`, configures its custom path, and keeps it read-only to
the service user. Templates use Forgejo's asset URLs and Caddy routes that origin
to Forgejo. The private extension transport is a separate backend path.

[build-soda-extension.ts](../../../scripts/build-soda-extension.ts) consumes the
[extension manifest](../../../system/containers/extension/extension.json), builds
declared pages/panels and styles, verifies locked terminal asset digests and emits
a sorted file inventory. Staging copies its listed assets, manifest and launcher,
hard-links the backend and creates the separate extension image. The stopped-host
extension-install unit uses the patched Forgejo CLI with mounted `/data` before
Forgejo starts; the launcher executes the backend with its private socket. The
sibling SDK owns manifest/protocol behavior and actual registration still needs
its installed workflow evidence.

The presentation digest covers staged public paths/bytes; extension's file list
is a staging inventory, with subsequent image/archive and candidate content hashes
binding packaged bytes. Local browser tests use local generated assets; release
production independently calls the same builders into its native output. A
prepared-suite invocation does not rebuild assets. Source inventory tests,
temporary builder outputs and template fixtures do not prove the bytes served
by an installed Forgejo. These distinctions refine [test evidence](test-evidence.md),
not a new browser harness or compatibility requirement.

**JOIN-BROWSER-CACHE-1 / B, conditional:** generated relative imports use the
header's presentation epoch; direct template script/style URLs carry literal
version tokens. No current stale token was established. The Forgejo presentation
owner should decide whether direct-entry cache invalidation is intentionally
manual or required to follow every rebuilt module. Only a required automatic
contract justifies changing that join; preserve the existing design/asset scope.

## Payload and installed identity: what is actually checked

[record.rs](../../../lib/soda-release-image/src/record.rs) binds serialized payload,
source/tool provenance, OCI archives and selected packaged content into candidate
records. The [candidate checker](../../../lib/soda-release-tools/src/)
binds requested source/sibling revision and architecture to archives and inspected
OCI identities. [check-native.sh](../../../scripts/check-native.sh) additionally
requires exact clean source/tools and runs source checks; it can compile the
checker and is larger than a direct selected candidate check. Neither operation
builds the candidate, signs it, installs it or qualifies complete workflows.

Installer media and image-import consumers admit fixed release records and all
six OCI images before their respective disk/import effects. This preserves actual
payload bytes and image identities; a newly generated record is not signing
authority. Candidate metadata and archive checks remain distinct from native
boot, destructive installation and delivery qualification.

The installed [host-content probe](../../../tools/acceptance/src/host_probes/content.rs)
checks architecture, package-inventory hash, expected stored image Config IDs,
selected content hashes from those images and three actual host service files.
In the activated phase it also verifies the extension's installed directory and
exact file set/hashes. Dashboard process checks in [host.sh](../../../tests/installed/host.sh)
observe PID, uid/gid, capabilities, `NoNewPrivs` and read-only rootfs.
[host_deployments](../../../tools/acceptance/src/host_probes/deployments.rs) prints
a deployment summary; it does not compare the booted checksum with a requested
release. Neither stored-image inspection nor active-unit status compares the
running service container's `.Image` with the payload's Config ID. Host executable
mode/label checks also do not hash every binary or identify a service's open executable.
No live mismatch was observed; full invoked-byte identity is not established here.

## Bounded correction packets

| Finding / accountable execution owner | Scope and prerequisites | Acceptance |
| --- | --- | --- |
| **JOIN-HOST-PROBE-PATH-1 / C**, installed-host qualification owner, consuming the release-image inventory | `host.sh` checks five `/usr/local/libexec/soda` paths while all five actual Go/Rust outputs are staged at `/usr/libexec/soda`. Fix that prefix and retain command set, mode, owner and SELinux checks. Vendor service rewriting does not transform the script passed directly as stdin by the native guide. Requires the current command inventory and explicit host selector; dashboard also has a container output and tailnet has a genuine host producer | Source assertions match all five staged paths. Later selected installed evidence checks those paths/modes/labels; do not remove dashboard or invent a missing tailnet producer. No current installed failure is claimed |
| **JOIN-INVOKED-IDENTITY-1 / C**, installed-host qualification/evidence owner | The native-support host-probe description claims current service image IDs are verified; current probes establish stored-image/content checks and process restrictions. Narrow the owning guide to those facts. For any future running-image claim, compare only the claimed containers' actual `.Image` to the recorded Config ID in the existing probe. Requires the selected phase/containers and admitted release record; booted-deployment or all-executable identity needs its own actual observation | Description and final evidence claim exactly match reached checks. A running-image claim fails on a mismatched actual container ID; stored expected images alone cannot satisfy it. No new attestation subsystem, broad executable inventory or automatic native run is selected |

These refine existing C-owned qualification duties; they do not reopen completed
library replacements or structural work. **TEST-CARGO-RESULT-1** remains the
separate Go test-helper process-result repair; it is not an observed production
compiler failure. Existing observation/custody findings remain prerequisites for
trusting their affected captures. No arbitrary execution request or observation's
requested revision field proves discovered installed identity.

Use Luna low for the settled path and description corrections; Luna medium for
any newly required runtime-identity admission and independent review. The
coordinator retains shared staging/manifests and costly qualification ownership.

## Smallest sufficient later verification

1. Reconcile the affected source selectors, staged transformations and evidence
   wording; preserve defined command roles and configuration producers. This audit
   performed source tracing and static target-name validation only.
2. During authorized fixes, select existing focused cases for the changed join
   and a development build of only the affected command if executable behavior is
   needed. Reuse valid artifacts for unchanged bytes; fix a driver without
   rebuilding the whole appliance merely to debug its assertions.
3. When a matching candidate exists, use the already-owned candidate checker for
   exact archive/content/source identity. Use `check-native.sh` only when its
   additional source/tool checks answer the current question. Neither substitutes
   for installed observation.
4. For installed claims, select the actual host/phase and read back the paths,
   image/process or served assets the claim names using existing drivers. A
   deployment summary, stored image, local browser bundle or fake compiler hook
   has a narrower scope than the currently invoked product.
5. Proceed to native boot/install and complete browser/project/provider journeys
   only after their existing source, capture, artifact and operational prerequisites
   are demonstrated. Keep later execution, cleanup and qualification evidence at
   its actual revision and workflow scope.

Fresh source joins and their independent challenge are complete for this focused
scope. Future behavior, installed identity and qualification remain unperformed;
conditional cache policy remains with its one established owner. Exact source
selectors and local evidence reside in ignored
`.artifacts/operational-joins-20261007-bc28a07c/`; this chapter is the maintained
planning disposition, not a successful build or installation receipt.
