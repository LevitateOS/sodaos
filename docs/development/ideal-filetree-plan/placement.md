# File granularity and placement

## File granularity and unresolved extraction seams

The 400-line threshold is an inspection trigger. Avoid a file per helper,
record, method or test assertion. Keep small private functions with the real
filesystem, command or wire operation they serve; reuse existing view modules
and mature primitives. The refined Muse maintenance proposal combines pinned
tool/path/native-error handling into filesystem ownership and quote diagnostics
with config wire formatting. It does not create utility crates or adapters.

Go concern files stay in the same package unless the [package ownership review](package-ownership.md) earns
a new owner. SQL stays local to store; one table does not imply one package.
Rust tests should remain descendants of their implementation modules, with the
minimum needed internal visibility. A module split does not justify promoting
private APIs or preserving useless forwarding facades.

For stateful TypeScript, retain one actor/binding/epoch/terminal lifetime owner.
First extract whole stateless views and already self-contained measurement or
screen helpers. The detailed stateful concerns are allocation proposals, not
implementation-ready helper signatures: reads, actions, layouts and resources
still need exact field/callback/admission/update seams. Preserve the mounted
project cache in its original owner, combine New/Rename with terminal
restoration/selection, and place Space in inventory response composition rather
than factory records. These refinements account for the concrete workspace
navigation, pane, eligibility and dialog functions omitted by broad first-pass
ranges. No mixins, request buses or duplicate state controllers are implied.

Current native-helper verification is present in source at 0d8d3b8e:
`tests/build/project_factory_roles_test.go` drives the compiled Rust helper, and
`internal/factory/control/st15_demo_native_test.go:277-294,348-353,399-410`
builds, stages and hashes its compiled bytes. Preserve those real observations
during package consolidation. Neither verification path was executed for upkeep.
Existing source-reading tests, shell imports and discovery roots
must follow the actual owning files rather than manufacture passing substitutes.
### Every current Go package

All 40 current Go packages with non-test source were checked for production imports and callers. This table records current direct in-repository imports, rather than the desired future graph. Commands are entrypoints; a missing Go importer does not mean an installed command is dead. `internal/testoci` has test callers, and `internal/archcheck` is test-only.

| Current package | Proposed owner | Current direct repository imports |
| --- | --- | --- |
| `cmd/soda-dashboard` | cmd/soda-dashboard | `internal/avatar`, `internal/config`, `internal/factory/control`, `internal/store`, `internal/web` |
| `cmd/soda-extension` | cmd/soda-extension | `internal/web` |
| `cmd/soda-forgejo-tailnet` | cmd/soda-forgejo-tailnet | `internal/forgejo`, `internal/tailnet` |
| `cmd/soda-host` | cmd/soda-host (Rust binary after pending PR26; Go until cutover) | `internal/host`, `internal/tailnet` |
| `cmd/soda-rootfs-server` | tools/soda-rootfs-server | None |
| `cmd/soda-tailnet` | cmd/soda-tailnet | `internal/tailnet` |
| `internal/acceptance` | Live installed probes/private input support; obsolete harness/process/worker closure retires | `internal/release/build` (obsolete Evidence import, removed with its owner) |
| `internal/avatar` | internal/avatar | None |
| `internal/config` | internal/config | None |
| `internal/factory` | internal/factory | `internal/identity`, `internal/project` |
| `internal/factory/control` | internal/factory/control | `internal/factory`, `internal/filelock`, `internal/identity`, `internal/project`, `internal/store`, `internal/strictjson` |
| `internal/filelock` | internal/filelock | None |
| `internal/forgejo` | internal/forgejo | `internal/config`, `internal/factory`, `internal/host/publish` |
| `internal/host` | internal/host (server half retires at pending PR26; Go client surface stays) | `internal/host/project`, `internal/host/tailnet`, `internal/host/terminal`, `internal/identity`, `internal/identity/client`, `internal/platform`, `internal/project`, `internal/release/build`, `internal/release/deliver`, `internal/strictjson`, `internal/tailnet` |
| `internal/host/project` | Retire privileged executor; Rust `lib/host` owns it after cutover | `internal/filelock`, `internal/host/terminal`, `internal/identity`, `internal/identity/client`, `internal/platform`, `internal/project`, `internal/strictjson` |
| `internal/host/publish` | internal/forgejo/publish | `internal/factory`, `internal/project` |
| `internal/host/tailnet` | Retire privileged companion executor; canonical Go `internal/tailnet` policy stays | `internal/filelock`, `internal/strictjson`, `internal/tailnet` |
| `internal/host/terminal` | Pure client frame/types only; privileged terminal executor retires | `internal/identity`, `internal/project`, `internal/strictjson` |
| `internal/identity` | internal/identity | None |
| `internal/identity/client` | internal/identity/client | `internal/identity` |
| `internal/platform` | Retire Go constants with their native host/Control callers; Rust configuration/unit owners retain the installed paths | None |
| `internal/project` | internal/project | `internal/strictjson` |
| `internal/release` | Retire whole subtree at the release cutover (Rust crates landed #32/#33/#34) | None |
| `internal/release/build` | Retire; owner is `lib/soda-release-build` | None |
| `internal/release/deliver` | Retire; owner is `lib/soda-release-deliver` | `internal/release/build`, `internal/strictjson` |
| `internal/release/image` | Retire; owner is `lib/soda-release-image` | `internal/acceptance`, `internal/release/build`, `internal/release/deliver`, `internal/store` |
| `internal/store` | internal/store | `internal/factory`, `internal/identity`, `internal/project` |
| `internal/strictjson` | internal/strictjson | None |
| `internal/tailnet` | Go wire/validation/selection and live status client; protected Control/policy/provider execution moves to `lib/host/src/tailnet/control` | `internal/filelock`, `internal/platform`, `internal/strictjson` (native-owner imports retire with that code) |
| `internal/testoci` | Retire with Go release tests at cutover | None |
| `internal/web` | internal/web | `internal/avatar`, `internal/config`, `internal/factory/control`, `internal/forgejo`, `internal/host`, `internal/identity/client`, `internal/project`, `internal/store`, `internal/web/api`, `internal/web/auth` |
| `internal/web/api` | internal/web/api | `internal/config`, `internal/factory`, `internal/factory/control`, `internal/forgejo`, `internal/host`, `internal/identity`, `internal/project`, `internal/store`, `internal/strictjson`, `internal/tailnet`, `internal/web/auth` |
| `internal/web/auth` | internal/web/auth | `internal/config`, `internal/forgejo`, `internal/store`, `internal/strictjson` |
| `tools/png-equal` | tools/png-equal | None |
| `tools/soda-artifacts` | Retire at release cutover; owner is `lib/soda-release-tools` (#30) | `internal/release/build` |
| `tools/soda-avatars` | tools/soda-avatars | `internal/avatar` |
| `tools/soda-build` | Retire at release cutover; owner is `lib/soda-release-tools` (#30) | `internal/acceptance`, `internal/release/build`, `internal/release/deliver`, `internal/release/image` |
| `tools/soda-candidate` | Retire at release cutover; owner is `lib/soda-release-tools` (#30) | None |
| `tools/soda-candidate-check` | Retire Go entrypoint; existing `lib/soda-release-tools` package owns the `soda-candidate-check` binary and delegates to Rust deliver | `internal/release/deliver` |
| `tools/soda-installed-probes` | tools/soda-installed-probes | `internal/acceptance` |

Retain architecture tests for surviving owners and update their actual path and
import assertions. Remove tests belonging solely to retired Go implementations;
move surviving wire/input behavior coverage with its live owner. The Go worker
and process package proposals are withdrawn.

## Placement rules

Installed binary identities and protocols stay stable in this candidate. The
release-assets package is the proposed `soda-release-assets` crate, with seven
existing binaries; the three old manifests merge into its single manifest.
The provider manifest folds into `soda-identity`, and its library root becomes
a private providers module. The four release crates keep their identities under
`lib/`. The existing host and project-terminal packages also own the replacement
entrypoints described here; multiple binaries do not require extra packages.
PostgreSQL maintenance and acceptance keep their multi-binary packages. File
and package boundaries follow callers and authority; none requires a new daemon.

| Current Cargo directory | Proposed directory |
| --- | --- |
| `rust/identity-providers/` | `cmd/soda-identity/src/providers/` |
| `rust/soda-acceptance/` | `tools/acceptance/` |
| `rust/soda-activate/` | `cmd/soda-activate/` |
| `rust/soda-asset-fetchers/` | `tools/release-assets/` |
| `rust/soda-build-tools/` | `lib/release-inputs/` |
| `rust/soda-candidate-setup/` | `tools/candidate-setup/` |
| `rust/soda-console-welcome/` | `cmd/soda-console-welcome/` |
| `rust/soda-factory/` | `cmd/soda-factory/` |
| `rust/soda-forgejo-domain/` | `cmd/soda-forgejo-domain/` |
| `rust/soda-forgejo-locales/` | `tools/release-assets/` |
| `rust/soda-forgejo-migrate/` | `cmd/soda-forgejo-migrate/` |
| `rust/soda-host/` | `lib/host/` (retained; pending PR26 cutover) |
| `rust/soda-release-build/` | `lib/soda-release-build/` (landed #34; 1:1, consolidation open) |
| `rust/soda-release-deliver/` | `lib/soda-release-deliver/` (landed #33; 1:1, consolidation open) |
| `rust/soda-release-image/` | `lib/soda-release-image/` (landed #32; 1:1, consolidation open) |
| `rust/soda-release-tools/` | `lib/soda-release-tools/` (landed #30; 1:1, consolidation open) |
| `rust/soda-identity/` | `cmd/soda-identity/` |
| `rust/soda-identity-compose/` | `cmd/soda-identity-compose/` |
| `rust/soda-image-import/` | `cmd/soda-image-import/` |
| `rust/soda-install/` | `cmd/soda-install/` |
| `rust/soda-json/` | `lib/json/` |
| `rust/soda-muse/` | `cmd/soda-muse/` |
| `rust/soda-muse-maintain/` | `cmd/soda-muse-maintain/` |
| `rust/soda-pg-fixture/` | `tools/postgres-fixture/` |
| `rust/soda-pg-maintenance/` | `cmd/soda-pg-maintenance/` |
| `rust/soda-project-terminal/` | `cmd/soda-project-terminal/` |
| `rust/soda-rotate-lab-creds/` | `tools/lab-credentials/` |
| `rust/soda-setup/` | `cmd/soda-setup/` |
| `rust/soda-stage-render/` | `tools/release-assets/` |
| `rust/soda-test-vm/` | `tools/test-vm/` |

Additional relocations are exhaustive prefix rules; the detailed tree expands every surviving leaf.

| Current path | Proposed path |
| --- | --- |
| `internal/host/publish/` | `internal/forgejo/publish/` |
| `internal/testoci/` | Retire with Go release tests at cutover |
| `appliance/forgejo/templates/` | `frontend/forgejo/templates/` |
| `appliance/forgejo/i18n/` | `frontend/forgejo/locales/` |
| `appliance/forgejo/` | `frontend/forgejo/` |
| `appliance/config/` | `system/host/config/` |
| `appliance/host-image/` | `system/host/image/` |
| `appliance/installer/` | `system/host/installer/` |
| `appliance/keys/` | `system/host/trust/` |
| `appliance/provisioning/` | `system/host/provisioning/` |
| `appliance/services/` | `system/host/services/` |
| `appliance/licenses/` | `system/licenses/` |
| `appliance/soda-extension/` | `system/containers/extension/` |
| `project-os/` | `system/project/` |
| `factory-os/` | Retire unselected alternate image recipe; no active target |
| `cmd/soda-rootfs-server/` | `tools/soda-rootfs-server/` |
| `scripts/selinux/` | `system/host/selinux/` |
| `scripts/ops/` | `tools/soda-rootfs-server/` |
| `appliance/host.Containerfile` | `system/host/Containerfile` |
| `appliance/dashboard.Containerfile` | `system/containers/dashboard/Containerfile` |
| `appliance/forgejo.Containerfile` | `system/containers/forgejo/Containerfile` |
| `appliance/soda-extension.Containerfile` | `system/containers/extension/Containerfile` |
| `appliance/tailnet.Containerfile` | `system/containers/tailnet/Containerfile` |
| `appliance/terminal-assets.lock.json` | `tools/release-assets/terminal-assets.lock.json` |
| `internal/acceptance/process_wait_linux.go` | Retire with obsolete Go execution closure |
| `internal/acceptance/process_wait_other.go` | Retire with obsolete Go execution closure |
| `internal/acceptance/process_linux_test.go` | Retire with obsolete Go execution closure |
| `internal/acceptance/worker_linux.go` | Retire at release cutover (no production caller left) |
| `internal/acceptance/worker_linux_test.go` | Retire at release cutover |
| `internal/release/**` (Go implementation/tests) | Retire at release cutover; owners are `lib/soda-release-*` |
| `internal/release/build/forgejo-payload.json` | `frontend/forgejo/payload.json`; live data and all loaders/fixtures move together |
| `tools/soda-build/**`, `tools/soda-candidate/**`, `tools/soda-artifacts/**` | Retire at release cutover; owner is `lib/soda-release-tools` |
| `tests/build/*.py`, `tests/installed/*.py` | Retire at PR42; Go successors already landed |
| `project-os/.../project-account`, `project-factory-roles` (extensionless Python) | Rust entrypoints/modules in the existing project-terminal package; compiled installed paths unchanged; source cutover pending |
| `internal/acceptance/process.go` | Retire; do not create `internal/process` or `process_command.go` |
| `internal/acceptance/evidence.go` | Retire Evidence; fold live PrivateFile into existing `internal/acceptance/installed.go` input support |
| `internal/platform/platform.go` | Retire with Go native callers; retain fixed installed paths at actual Rust configuration/unit admission owners |
| `tests/build/{project_os_observation,project_keys,terminal}_test.go` | Retire guards that pin obsolete Go/Python file existence; retain actual Rust behavior tests |
| `tools/soda-candidate-check/main.go` | Rust `lib/soda-release-tools/src/bin/soda-candidate-check.rs` plus `candidate_check.rs`; preserve the existing verification CLI |
| `rust/soda-identity/src/main.rs` | `cmd/soda-identity/src/main.rs`, `cmd/soda-identity/src/service.rs` |

The tracked root files `installer` and `soda-candidate` are ELF build outputs. Their proposed disposition is removal from the tracked source tree; current tools build into ignored `.artifacts/`. The `rust/soda-host/` files are retained at `lib/host/` for pending PR26. Detailed decomposition entries for cutover-deleted files stay as review history; the [proposed tree](proposed-tree.md) omits them. This document deletes or executes none of these files.

