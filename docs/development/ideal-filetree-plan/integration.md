# Integration wiring

Preserve canonical assets, licenses, fixtures and technical documentation.

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
- Current command discovery is in Rust release-image `sys.rs`; compilation is
  in `build.rs:522-553`, using Go for discovered `cmd/` commands and explicit
  Rust selectors for the landed ports. Moving Rust entrypoints into `cmd/`
  requires updating this existing discovery/recipe selection together; preserve
  installed binary destinations and avoid a duplicate compile or Go invocation
  on a Rust-only command directory.
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
  Preserve the full intentional Soda design overrides across all Forgejo pages,
  as the owner confirmed in GUIDANCE-08.
  Correctness and native integration findings cannot infer permission to remove
  the overrides, restore stock pages or redesign the presentation.
- Direct source-path assertions in retained Rust, Go and browser tests, including
  paths below `rust/`, `appliance/`, `project-os/` and `scripts/ops/`. Remove
  Python import subjects and test drivers as part of their real native cutover.
- Source-check and hook discovery for moved shipping files, including
  `.githooks/pre-commit`, SQL locality and complexity check roots. Preserve the
  Go ownership guide and architecture checks; the proposed `lib/` avoids placing
  Rust-only directories inside the Go directory-name assertion.
- Relative documentation links and source references. The external handbook
  sync contract keeps `docs/public/` stable.
