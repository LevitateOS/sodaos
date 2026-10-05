# Integration and review boundaries

## Large data and documents retained

Locks, complete data fixtures, attribution and canonical documentation owners are retained. Their line counts do not establish excessive implementation complexity. Any later split of a data inventory needs its real loader and consumer contract, rather than a size-only rule.

| Current file | Lines | Proposed file |
| --- | ---: | --- |
| `Cargo.lock` | 1849 | `Cargo.lock` |
| `appliance/forgejo/README.md` | 782 | `frontend/forgejo/README.md` |
| `appliance/licenses/forgejo-LICENSE` | 674 | `system/licenses/forgejo-LICENSE` |
| `assets/branding/cockpit/provenance/LICENSES.txt` | 510 | `assets/branding/cockpit/provenance/LICENSES.txt` |
| `docs/architecture/trust.md` | 956 | `docs/architecture/trust.md` |
| `docs/development/factory-implementation-plan.md` | 1353 | `docs/development/factory-implementation-plan.md` |
| `docs/development/forgejo-extensions-plan.md` | 1029 | `docs/development/forgejo-extensions-plan.md` |
| `docs/development/ideal-filetree-plan.md` | 5020 at the structural baseline | `docs/development/ideal-filetree-plan/` (one living plan, split into the index and owned topic files) |
| `docs/development/native-support.md` | 475 | `docs/development/native-support.md` |
| `docs/product/overview.md` | 546 | `docs/product/overview.md` |
| `docs/research/factory-capability-map.md` | 532 | `docs/research/factory-capability-map.md` |
| `docs/research/onedev.md` | 413 | `docs/research/onedev.md` |
| `internal/avatar/style-v1.json` | 1465 | `internal/avatar/style-v1.json` |
| `tests/forgejo/presentation/form-native-contracts.json` | 750 | `tests/forgejo/presentation/form-native-contracts.json` |
| `tests/forgejo/presentation/inventory.json` | 9729 | `tests/forgejo/presentation/inventory.json` |
| `tests/forgejo/presentation/repository-settings-native-contracts.json` | 886 | `tests/forgejo/presentation/repository-settings-native-contracts.json` |
| `tests/forgejo/presentation/settings-native-contracts.json` | 611 | `tests/forgejo/presentation/settings-native-contracts.json` |

## Wiring that changes with the tree

- Root Cargo workspace members, consolidated manifests and relative
  dependencies; actual binary/test declarations and private module imports.
  Release-assets keeps seven binaries and three distinct integration-test
  support namespaces. Provider consolidation keeps a single providers module
  root; controller adapters stay with control rather than a competing root.
  Replace the three old asset package selections and library imports with the
  merged soda-release-assets fetch/render/locales namespaces directly. Keep one
  actual provider type owner and rebind the provider-local helper/test paths.
- Release-build and release-image keep `tests/oracle.rs` as their integration
  crate roots. Bind the proposed children explicitly with paths such as
  `#[path = "oracle/oci.rs"] mod oci;` for build and
  `#[path = "oracle/media.rs"] mod media;` for image; plain root `mod` declarations
  would search beside oracle.rs. Preserve each suite's actual golden-vector
  lookup and shared fixtures. Deliver's `tests/oracle/main.rs` and tools'
  `tests/cli/main.rs` use their natural child-module paths.
- Direct Go imports for publication and the surviving host client/wire and
  probe-input closure. Update the existing ownership guide and architecture
  assertions together. Remove imports/tests owned only by the retired host
  executors, acceptance execution, release pipeline and OCI test helper; do not
  create a Go process package or facade to keep those imports compiling.
- Installed command discovery currently in `internal/release/build/production.go`
  and compilation/delivery inventory currently in `internal/release/image`,
  replaced by their Rust owners at cutover. The current loop
  Go-compiles every `cmd/` directory; mixed command directories require adapting
  this existing recipe selection while preserving installed binary destinations.
- Stage renderer checkout detection, payload inventories, Containerfile COPY
  sources, rootfs assembly, service paths, provisioning and release input locks.
- Acceptance remote payload delivery/recording and the real compiled Project
  account/factory-role helper staging/hash closure. Preserve the existing
  entrypoint identities and verify executing bytes rather than retired wrappers.
- Shared schema extraction in the identity broker: `internal/store/schema.go`
  remains the observed schema source of truth. A new physical tree does not
  create a separate schema owner.
- Forgejo and extension build imports, templates, locale additions, CSS imports,
  Bun scripts, TypeScript includes and browser fixtures. The native Forgejo
  presentation and independently installed extension retain separate payloads.
- Direct source-path assertions in retained Rust, Go and browser tests, including
  paths below `rust/`, `appliance/`, `project-os/` and `scripts/ops/`. Remove
  Python import subjects and test drivers as part of their real native cutover.
- Source-check and hook discovery for moved shipping files, including
  `.githooks/pre-commit`, SQL locality and complexity check roots. Preserve the
  Go ownership guide and architecture checks; the proposed `lib/` avoids placing
  Rust-only directories inside the Go directory-name assertion.
- Relative documentation links and source references. The external handbook
  sync contract keeps `docs/public/` stable.

## Review and implementation boundary

This plan folder is the maintained project artifact from this pass; the inventory
and analysis reports remain in ignored .artifacts/filetree-planning/. Source
moves, implementation splits, manifests, operational configuration and Git
operations are not performed by authoring it. The repository asks for three
fresh Jev SystemOne consultations
on difficult design decisions; that tool is not available in this session. The
tree remains a proposal, and consultation agreement would not prove behavior.

Source inspection and inventory checks validate coverage of the proposal. They
do not prove compilation, package imports, source checks, installed paths or
native user journeys. Those require the corresponding checks when the refactor
is authorized and implemented. Preserve current private inputs and unrelated
work, and recheck source drift before each later update of this plan.

The initial review covered 1,586 paths at its recorded baseline. The historical
structural reconciliation accounted for all 1,705 paths at d7e565aa, with 1,488
retained/moved dispositions and 217 retirements, and reviewed all 190 oversized
code files then present. Those are historical counts, not the current source
inventory. The latest coverage ledger accounts for all 1,597 tracked paths at
26d420f2 and gives current mixed/oversized responsibility maps separately.

The primary tree now contains 2,358 unique leaves after excluding 11 obsolete
proposals found by the coverage pass and replacing the single plan document
with its 122 maintained Markdown sections. It retains 27 Cargo package manifests
plus the root workspace manifest. These counts include proposed native
entrypoints and concern splits, not measured final implementation sizes. Full
structural reconciliation after d7e565aa remains pending. Shared destination
leaves consolidate existing providers, assets, HTTP/test support and native
assertion coverage. No file/directory or Rust file/module-root filename conflicts
remain in the proposed tree. It omits pre-port Go acceptance/process/release/
host implementations, Python sources and the newly evidenced historical mockup,
retired provenance overrides and alternate factory recipe. Live input/wire and
probe portions remain separately mapped; unknown lifecycle is not treated as
proof of dead code.

