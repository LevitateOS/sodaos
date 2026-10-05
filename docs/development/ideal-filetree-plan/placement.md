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

At `f7e9cf9d`, 29 tracked directories contain Go files without the `_test.go`
filename suffix: 28 under `cmd/`, `internal/` and `tools/`, plus test support
under `tests/build`. The table records their literal direct repository import
statements across platform variants; no build or reachability qualification was
run. `internal/archcheck` is test-only and omitted from that count. A package
with no importer can still be an entrypoint, or have only test callers; its
lifecycle is assessed in the slice record. Deleted Go host/release/tool
directories are excluded from this current census.

| Current package | Proposed owner | Current direct repository imports |
| --- | --- | --- |
| `cmd/soda-dashboard` | cmd/soda-dashboard | `internal/avatar`, `internal/config`, `internal/factory/control`, `internal/store`, `internal/web` |
| `cmd/soda-extension` | cmd/soda-extension | `internal/web` |
| `cmd/soda-forgejo-tailnet` | Rust `cmd/soda-forgejo-tailnet/main.rs`, additional binary in existing `soda-host` package; native duties/private tests at `lib/host/src/tailnet/forgejo.rs` | Current Go imports `internal/forgejo`, `internal/tailnet`; retire only helper-exclusive execution/rewrite duties after the specified Rust cutover |
| `cmd/soda-rootfs-server` | tools/soda-rootfs-server | None |
| `cmd/soda-tailnet` | cmd/soda-tailnet | `internal/tailnet` |
| `internal/acceptance` | Live installed probes/private inputs; obsolete Go harness/process/worker closure has a separate retirement disposition | None |
| `internal/avatar` | internal/avatar | None |
| `internal/config` | internal/config | None |
| `internal/factory` | internal/factory | `internal/identity`, `internal/project` |
| `internal/factory/control` | internal/factory/control | `internal/factory`, `internal/filelock`, `internal/identity`, `internal/project`, `internal/store`, `internal/strictjson` |
| `internal/filelock` | internal/filelock | None |
| `internal/forgejo` | internal/forgejo | `internal/config`, `internal/factory`, `internal/host/publish` |
| `internal/host` | internal/host (surviving Go client/wire surface; Rust owns daemon/executors) | `internal/identity`, `internal/project`, `internal/strictjson`, `internal/tailnet` |
| `internal/host/publish` | internal/forgejo/publish | `internal/factory`, `internal/project` |
| `internal/identity` | internal/identity | None |
| `internal/identity/client` | internal/identity/client | `internal/identity` |
| `internal/platform` | Retire Go constants with their native host/Control callers; Rust configuration/unit owners retain the installed paths | None |
| `internal/project` | internal/project | `internal/strictjson` |
| `internal/store` | internal/store | `internal/factory`, `internal/identity`, `internal/project` |
| `internal/strictjson` | internal/strictjson | None |
| `internal/tailnet` | Go wire/validation/selection and live status client; protected Control/policy/provider execution moves to `lib/host/src/tailnet/control` | `internal/filelock`, `internal/platform`, `internal/strictjson` |
| `internal/testoci` | Retire with Go release tests at cutover | None |
| `internal/web` | internal/web | `internal/avatar`, `internal/config`, `internal/factory/control`, `internal/forgejo`, `internal/host`, `internal/identity/client`, `internal/project`, `internal/store`, `internal/web/api`, `internal/web/auth` |
| `internal/web/api` | internal/web/api | `internal/config`, `internal/factory`, `internal/factory/control`, `internal/forgejo`, `internal/host`, `internal/host/publish`, `internal/identity`, `internal/project`, `internal/store`, `internal/strictjson`, `internal/tailnet`, `internal/web/auth` |
| `internal/web/auth` | internal/web/auth | `internal/config`, `internal/forgejo`, `internal/store`, `internal/strictjson` |
| `tests/build` | tests/build (Go support for existing test drivers) | None |
| `tools/png-equal` | tools/png-equal | None |
| `tools/soda-avatars` | tools/soda-avatars | `internal/avatar` |
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
| `rust/soda-host/` | `lib/host/` (retained Rust daemon; entrypoint moves within same package) |
| `rust/soda-release-build/` | `lib/soda-release-build/`; retain the package, with the selected pure OCI helper allocation in [port assessment](port-assessment.md) |
| `rust/soda-release-deliver/` | `lib/soda-release-deliver/`; retain the package and selected delivered-content/payload owners |
| `rust/soda-release-image/` | `lib/soda-release-image/`; retain the package and actual image traits, importing selected existing helper owners at cutover |
| `rust/soda-release-tools/` | `lib/soda-release-tools/`; retain the controller/composition package and current binary identities |
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
| `rust/soda-project-account/` | Fold into existing `cmd/soda-project-terminal/`; keep `project-account` binary |
| `rust/soda-project-factory-roles/` | Fold into existing `cmd/soda-project-terminal/`; keep `project-factory-roles` binary |
| `rust/soda-rotate-lab-creds/` | `tools/lab-credentials/` |
| `rust/soda-setup/` | `cmd/soda-setup/` |
| `rust/soda-stage-render/` | `tools/release-assets/` |
| `rust/soda-test-vm/` | `tools/test-vm/` |

Additional relocations are exhaustive prefix rules; the detailed tree expands every surviving leaf.

| Current path | Proposed path |
| --- | --- |
| `internal/host/publish/` | `internal/forgejo/publish/` |
| `internal/testoci/` | Retire with Go release tests at cutover |
| `appliance/forgejo/templates/` | `frontend/forgejo/templates/`; retain the intentional Soda design overrides across all pages, per [GUIDANCE-08](review-assignments.md#guidance-conflicts-and-controlling-decisions) and [G08](reviews/G08.md). Relocation does not reduce override scope. |
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
| `assets/branding/forgejo/forgejo-payload.json` | `frontend/forgejo/payload.json`; this is the current live manifest at f7. Move all actual loaders, build/stage inputs and fixtures with it; preserve installed destinations and the full intentional page overrides. The former Go release manifest path is a historical predecessor, with no separate future leaf. |
| `tools/soda-build/**`, `tools/soda-candidate/**`, `tools/soda-artifacts/**` | Retire at release cutover; owner is `lib/soda-release-tools` |
| Historical `tests/build/*.py`, `tests/installed/*.py` | Already absent at the scheduling source; preserve current Go/Rust test subjects, not another Python port/deletion task |
| Historical `project-os/.../project-account`, `project-factory-roles` (extensionless Python) | Already replaced by compiled Rust helpers; fold existing crates into the Project-terminal package, preserving compiled installed paths and actual helper tests |
| `internal/acceptance/process.go` | Retire; do not create `internal/process` or `process_command.go` |
| `internal/acceptance/evidence.go` | Retire Evidence; fold live PrivateFile into existing `internal/acceptance/installed.go` input support |
| `internal/platform/platform.go` | Retire with Go native callers; retain fixed installed paths at actual Rust configuration/unit admission owners |
| `tests/build/{project_os_observation,project_keys,terminal}_test.go` | Retire guards that pin obsolete Go/Python file existence; retain actual Rust behavior tests |
| Historical `tools/soda-candidate-check/main.go` | Already absent; current Rust `soda-release-tools` binary and `check_cli` owner move/split under `lib/`; preserve the existing verification CLI, no new port |
| `rust/soda-identity/src/main.rs` | `cmd/soda-identity/src/main.rs`, `cmd/soda-identity/src/service.rs` |
| `frontend/spaces/sodaspaces-widths.ts` | Retire the unused predecessor utility; no target leaf. [S03](reviews/S03.md#s03-q2-independent-retirement-disposition) and its independent challenge found definitions only, no production imports or build entry. Active workspace layout and geometry remain with their current owner. |
| `rust/soda-release-build/src/{progress,clock}.rs` and exclusive progress oracle/support duties | Retire the unused mirrored closure; no target leaves for progress, build_execution, clock, progress/tests, oracle/progress or support/buffer. [D03](reviews/D03.md) and A's independent source/caller challenge support this exact disposition. Current release-tools progress and release-image Runner remain. |
| `rust/soda-release-tools/src/record.rs:256-259` (`files_map`) | Retire this definition-only helper at cutover; retain active candidate/build record duties in the existing Rust release-tools owner. |

The tracked root files `installer` and `soda-candidate` are ELF build outputs. Their proposed disposition is removal from the tracked source tree; current tools build into ignored `.artifacts/`. The `rust/soda-host/` files and its actual daemon are retained at `lib/host/`, with the same-package entrypoint moved to `cmd/soda-host/main.rs`. Detailed decomposition entries for cutover-deleted files stay as review history; the [proposed tree](proposed-tree.md) omits them. This document deletes or executes none of these files.
