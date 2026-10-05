# Package ownership and execution topology

## Source coverage

Structural reconciliation baseline: `d7e565aa1019753997a99fd430ba103d7a472b48`, with **1705 tracked paths** at that commit. Latest slice coverage is recorded separately at `0d8d3b8e` in the [coverage index](coverage/README.md). The table and full structural counts below remain historical; latest upkeep reconciles the five source/test changes after `26d420f2` and the documentation split without claiming a new full structural review.

Initial full structural review: `899e9bf3b9b57883df42cbb520329679b3b8186d`. Latest reconciliation scope: six landed port PRs (tests to Go #29/#31, release tools #30, release image #32, release deliver #33, release build #34) plus the decided language policy. Keep this initial review separate from later merge checkpoints.

At audit source `f7e9cf9d`, the workspace has 32 Cargo members, including
`soda-project-account` and `soda-project-factory-roles`. The proposed 27-package
tree folds both into the existing Project-terminal package in addition to the
provider/assets consolidations. Rust `soda-host` already declares its binary;
release-image `build.rs:522-553` compiles it, and the superseded Go daemon and
executor directories are absent. These are current source facts, not installed
qualification. The root-count table and deleted-source citations below are
historical; they cannot authorize recreating predecessors or count as current
cutover tasks. Actual per-slice records reconcile remaining responsibilities.

| Current root | Tracked paths | Proposed disposition |
| --- | ---: | --- |
| `(root files)` | 24 | Retain live tool inputs and ownership files; omit two tracked ELF outputs and the two Python-only tooling inputs at their cutover. |
| `.agents` | 1 | Retain ownership; include every leaf in the [proposed tree](proposed-tree.md). |
| `.githooks` | 1 | Retain ownership; include every leaf in the [proposed tree](proposed-tree.md). |
| `appliance` | 306 | System definitions to system; Forgejo presentation to frontend. |
| `assets` | 169 | Retain ownership; include every leaf in the [proposed tree](proposed-tree.md). |
| `cmd` | 21 | Preserve installed command identities; replace the Go host entrypoint with the same-package Rust binary and retire its Go-only tests. |
| `docs` | 92 | Preserve canonical owners; retire the Python-only tooling guide when its final callers move. |
| `factory-os` | 1 | Retire unselected alternate image recipe; excluded by the latest coverage correction. |
| `frontend` | 32 | Retain ownership; include every leaf in the [proposed tree](proposed-tree.md). |
| `internal` | 502 | Preserve domain/trust boundaries and live installed probes; relocate publication to Forgejo. Retire the obsolete acceptance execution/evidence/process closure, Go release pipeline and privileged host server/executors. Keep host clients and their pure wire types. |
| `project-os` | 17 | system/project, preserving rootfs paths and installed identities. Two extensionless Python helpers port at PR42. |
| `rust` | 283 | All 30 packages accounted for: 27 proposed packages after consolidation; host library and daemon ownership retained. |
| `scripts` | 83 | Retain build/check entrypoints; move SELinux and rootfs service definitions to their owners. |
| `tests` | 133 | Preserve actual assertion coverage; retire `.py` predecessors and replace Python import drivers with tests of their native owners. |
| `tools` | 40 | Retain ownership; ported Go tools retire at release cutover. |

Standalone Python inventory at this baseline: **25 tracked programs** — 23 `.py` files (21 in `tests/build`, 2 in `tests/installed`), plus 2 extensionless libexec helpers (`project-account`, `project-factory-roles`). Go test/probe files have landed, but several still execute Python source or import these predecessors. The Python cutover must replace that embedded execution and its test drivers as well as the 25 files; it is not complete merely because assertions moved to Go. Installed helper paths remain stable as compiled outputs, not tracked Python sources. See [Python cutover closure](port-assessment.md#python-cutover-closure).

That program count is historical. At the audit pin, `docs/development/python.md`
owns the active elimination/gate contract and remains a target document. The
compiled helper crates/build inputs are present; their proposed package fold is
separate from the landed ports. H06/D06 review remaining probe assumptions and
actual test subjects rather than repeating earlier Python retirement tasks.

There are **207 text files over 400 lines**, of which **190 are executable source, style, template, or test files**. Counts include embedded tests and are not production-only line counts. The 17 remaining files are data, locks, licenses or documents, including this plan. A size flag prompts a responsibility review; it does not justify arbitrary chunks or new product machinery.

Detailed concern reviews cover **190 oversized code files**; **0 remain unreviewed in this snapshot**. This includes all 31 newly oversized Rust release files and `tests/build/project_factory_roles_test.go`, in addition to the original 158 entries. Retiring sources have explicit removal dispositions; their old decomposition suggestions are not target leaves. Proposed filenames are responsibility seams, not measured final file sizes.

## Package ownership, consolidation and splitting

Directory renaming is only the first part of this proposal. Package ownership
must reduce duplicated implementation while keeping domain policy, credential
custody, browser authorization, persistence and privileged execution distinct.
The structural review traced its Go imports and 30 Cargo workspace members;
the current [placement census](placement.md#every-current-go-package) and slice
records distinguish later source changes from that historical evidence.
The following recommendations are design proposals for the owner to assess.

### Recommended package changes

| Current ownership | Proposed ownership | Reason and implementation closure |
| --- | --- | --- |
| `internal/host/publish` | `internal/forgejo/publish` | Its only production caller is `internal/forgejo`; it executes unprivileged repository publication. Keep the existing `BackgroundOperations` interface, credential isolation and factory/project domain inputs. Update the existing architecture rules to name this publication leaf correctly. |
| Go acceptance Command/Evidence/process closure | Retire obsolete execution/evidence closure and its tests; no `internal/process` package | Its demonstrated process callers are the retiring Go image builder, Go worker and legacy acceptance harness. The Rust owners retain their real execution and descendant-cleanup behavior. Fold the live `PrivateFile` implementation into the existing installed probe input support before deleting `evidence.go`. |
| `tools/soda-build/worker_linux.go` | `lib/soda-release-tools` (`worker.rs`, landed #30) | The Rust worker ports the tool-side worker and is the single worker owner. Retire the Go file at the release cutover. Preserve both admitted worker identities, trusted executable checks, mount/environment rules, root dispatch and exact systemd unit cleanup. |
| `internal/acceptance/worker_linux*` | Retire at the release cutover (no production caller left) | Its only production caller is the retired Go tool. Do not move it into the deleted tool as previously proposed. |
| `internal/release/{build,deliver,image}` (all Go) | Retire at the decided release cutover; `lib/soda-release-{build,deliver,image,tools}` (landed #30/#32/#33/#34) are the owners | The earlier Go-internal consolidation (parent `payload.go`/`identity.go`/`json.go`) is void: the decided language policy keeps the pipeline in Rust, so the Go side is deleted, not reshaped. Keep construction in build, orchestration in image and trust/signing/publication in deliver, now as Rust crates. |
| `internal/testoci` | Retire with the Go release tests at the release cutover | All actual consumers are Go release tests, which retire with the Go pipeline. Fixture duty moves to the Rust oracle suites. |
| `rust/identity-providers` + `rust/soda-identity` | One `cmd/soda-identity` Cargo package, with private `providers/codex`, `providers/muse` and existing provider types/helpers | The provider crate has exactly one direct production dependent. Consolidate package ownership while retaining provider protocol, broker policy, storage and wire modules. Fold its manifest/dependencies into the broker; remove the old package boundary and aliases. |
| `rust/soda-asset-fetchers` + `rust/soda-stage-render` + `rust/soda-forgejo-locales` | One `tools/release-assets` Cargo package with `fetch/`, `render/`, `locales/` modules and the existing seven binaries | These are build-input operations under the existing release producer. Keep separate CLI entrypoints and test support namespaces. Existing transport/hash helpers can be reused after matching their caller behavior. This merges three manifests and library roots rather than adding another facade crate. |
| `rust/soda-host` | Retain at `lib/host`; one package also owns the moved `cmd/soda-host/main.rs` | Current source already owns the Rust daemon and native adapters; its manifest and release compile path select that binary. The proposed move preserves this one package/service and the surviving Go client/wire surface. No Go daemon or retired executor is a target. |
| `rust/soda-project-account` + `rust/soda-project-factory-roles` | Existing `cmd/soda-project-terminal` package; `src/bin/project-account.rs`, `src/bin/project-factory-roles.rs`, `src/account.rs` and `src/factory_roles/` | Both native ports already compile into Project tools via release-build `production.rs:222-236` and install through Project Containerfile lines 40-44. Consolidate their package roots/dependencies and update these actual compile selectors; preserve binary names, installed paths and real compiled-helper tests. P03/P06 and shared owners challenge the exact split. |

Publication evidence: `internal/forgejo/publish.go:37-58` and
`internal/host/publish/operation.go:19-35`. Process/worker evidence:
`internal/release/image/build.go:639-645`,
`internal/acceptance/worker_linux.go:137-176`, and
`tools/soda-build/worker_linux.go:89,281-307`. Release input evidence:
`internal/host/daemon.go:61-80`, `internal/release/deliver/payload.go:16-113`,
`internal/release/build/files.go:28-46,140-215`, and
`internal/release/doc.go:1-16`. Asset ownership is visible in
`internal/release/build/production.go:178-210` and the three Cargo manifests.

Provider consolidation includes an actual binary/library seam:
`rust/soda-identity/src/main.rs:245-278` currently constructs providers through
the external crate, while the binary imports the soda_identity library.
The proposed `main.rs` keeps the existing entrypoint; `service.rs` moves the
existing service initialization, settings, provider construction and listener
closure into that library. Keep provider implementation modules private there
and choose the minimum entry facade for the existing invocation. Do not copy
the modules into both binary and library or publish the provider internals to
make the import compile. Provider code's current crate-root helper imports and
TestDir references must rebind to its providers owner. Connection/Enrollment
keep their one existing definition, shared with wire; removing the old crate
name does not mean cloning those records.

The merged release-assets library root owns fetch, render and locales modules.
Replace current `-p` selections and old library imports in the existing
binaries/tests directly, without compatibility facade crates. The provisioning
fixture's relative path follows its real input to
`tests/fixtures/prov-root/system/host/provisioning/base.json`. Move the live
`internal/release/build/forgejo-payload.json` data to
`frontend/forgejo/payload.json`, together with its provisioning fixture copy.
Update the stage renderer's checkout sentinel and loader, Go payload/Spaces
tests, source-path assertions and owning references.
Keep installed destination keys stable; rewrite only moved source values.

The former Go process extraction is withdrawn. `StartCommand` is called by
`internal/release/image/build.go:639`, `internal/acceptance/worker_linux.go:157`
and the obsolete Command wrapper; none survives the decided cutovers. Do not
create a permanent package for that closure. `PrivateFile` remains live through
`internal/acceptance/installed.go:65`; fold its small implementation into that
existing private-input owner and preserve the real service/access probe tests.
Do not add a file per helper. Preserve installed probe behavior without retaining
legacy Evidence, Execute, Remote, Command or their release-build import.

The immutable release move is superseded by the landed Rust pipeline (#32/#33/#34):
the ReadJSON/ReadJSONAt helper closure (regular-file/inode confinement, the
4 MiB bound, exact-byte digest, unknown/trailing rejection) now lives in the
Rust crates instead of a reshaped Go parent. Payload validation still supplies
no signature, qualification or upgrade authority. Shared delivery test fixtures
move to the Rust oracle suites with their consumers at cutover.

### Boundaries that should remain

Keep `internal/project`, `internal/identity` and `internal/factory` as canonical
Go meaning/validation owners. Keep `internal/factory/control` as the existing
coordinator and `internal/store` as the one SQL/schema owner. Keep host project,
terminal and Tailnet execution separate from those pure records, SQL and HTTP.
Keep web wiring, API admission and browser authorization separate within the
existing dashboard process. Small shared packages such as strictjson, filelock,
avatar, config and the identity client have surviving callers; small size
does not justify replacing them with a general helpers package.

The schema source remains `internal/store/schema.go`. Rust identity's schema
copy explicitly follows that source and has a drift assertion
(`rust/soda-identity/src/schema.rs:1-6,190-196`). Current storage is PostgreSQL;
older SQLite wording in credentials/development documentation is stale.
Go metadata reads remain live in `internal/web/api/factory_settings.go:467-485`.
IdentitySaveConnection/IdentitySaveGrant now have test callers only, but moving
them into a package-local test file would break fixtures in other packages.
Any retirement must preserve the real cross-package fixture/broker boundary;
it does not justify a second production broker store or schema.

`lib/json` has twelve direct current Cargo dependents (up from eight; all four landed release crates use it) and earns shared ownership.
Its duplicate-last-wins value parser is not equivalent to strict broker
admission, Go structured binding or Python-compatible terminal emission.
Consolidate exact encoding or hashing primitives where contracts match; retain
caller-specific policy. The existing lock contains SHA implementations used by
other live crates, so retained callers can reuse a mature implementation rather
than maintain custom algorithm copies. A shared primitive is no reason to create
a credential, hashing, database or JSON process.

## Existing execution topology

These are source-verified deployment definitions and callers, not observations
of the currently running machine. The proposed package/file moves add **zero
daemons, containers, sockets or sidecars**.

| Existing owner | Execution boundary | Refactor consequence |
| --- | --- | --- |
| Dashboard + web/API/auth + factory/control + store | Existing unprivileged dashboard process; private host/identity calls and PostgreSQL | Factory coordination remains an object inside this process. No new factory service. |
| soda-host + project/terminal/Tailnet executors (Rust in current source) | Existing root service; root:soda Unix operation socket; real host namespaces | Keep the fixed privileged surface and existing launch handling in this process. Moving the package and binary entrypoint adds no host daemon. Installed behavior remains separately unverified. |
| Identity broker + providers | Existing soda-identity service; administration socket 0660, runtime socket 0600 | Consolidating providers changes a package boundary, not credential custody or the root host execution boundary. |
| Project terminal prepare/broker/keys/control | One existing installed helper executed inside a Project | Split modules inside the same crate and executable. The guest broker concern is distinct from the host credential broker. |
| Muse CLI, Compose registration and maintenance | Existing entrypoints under their established caller identities | Share proven primitives, retain their different invocation/authority duties. No general identity runtime service. |
| Factory CLI | One-shot client of dashboard's operator endpoint | Keep one client; it owns no independent database or reconciliation loop. |
| Optional per-Project Tailnet companion | Existing tailscaled container supervised by soda-host service phases | Preserve its existing network/credential lifetime. Removing a real trust boundary merely to reduce a process count is a separate product decision. |
| Release assets, installer, import and acceptance tools | Build/operator invocations; acceptance has outside driver, SSH payload and installed host-probe roles | Package consolidation does not turn command roles into daemons. Retain required binaries as actual payloads. |

Topology evidence: `internal/web/server.go:25-50`,
`appliance/services/soda-dashboard.container:7-35`,
`appliance/services/soda-host.service:7-15`,
`appliance/services/soda-identity.service:8-22`,
`appliance/services/soda-identity.socket:7-12`,
`appliance/services/soda-identity-runtime.socket:7-12`,
`internal/host/terminal/agent.go:13-16,67-75`,
`project-os/Containerfile:32-39`, `rust/soda-factory/src/main.rs:1-16`, and
`internal/host/tailnet/runtime.go:242-258`. Existing services, binaries, crates
and concern files are separate counts; none implies another.
