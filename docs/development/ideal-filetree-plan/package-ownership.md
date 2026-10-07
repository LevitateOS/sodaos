# Package ownership and execution topology

## Source coverage

Structural reconciliation baseline: `d7e565aa1019753997a99fd430ba103d7a472b48`, with **1705 tracked paths** at that commit. Latest slice coverage is recorded separately at `0d8d3b8e` in the [coverage index](coverage/README.md). The table and full structural counts below remain historical; latest upkeep reconciles the five source/test changes after `26d420f2` and the documentation split without claiming a new full structural review.

The selective library-adoption reconciliation uses source
`72e4bb9015b6d6a622b45638104c74851a137473`. The
[library chapter](library-adoption.md#finding-allocation) overrides pending
generic-engine allocation directions selected there, while completed structural
work stays complete at its recorded scope. Historical counts and source spans
below are not refreshed by this change. Package boundaries continue to describe
application duties and thin library adapters; library adoption does not create
new services or a general helpers crate.

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
| `internal` | 502 | Preserve domain/trust boundaries and live installed probes; relocate publication to Forgejo. Retire the obsolete acceptance execution/evidence/process closure, Go release pipeline and privileged host server/executors. Keep host clients and their pure wire types. | **[run 20261005:** publication leaf moved (see row above); B01-B05 Go splits LANDED inside existing owners; retirements pending cutovers.]
| `project-os` | 17 | system/project, preserving rootfs paths and installed identities. Two extensionless Python helpers port at PR42. | **[run 20261005:** move LANDED (C09 c0a19d5a + R01 d296d340).]
| `rust` | 283 | All 30 packages accounted for: 27 proposed packages after consolidation; host library and daemon ownership retained. | **[run 20261005:** counts are baseline-historical; recount after implementation lands.]
| `scripts` | 83 | Retain build/check entrypoints; move SELinux and rootfs service definitions to their owners. |
| `tests` | 133 | Preserve actual assertion coverage; retire `.py` predecessors and replace Python import drivers with tests of their native owners. |
| `tools` | 40 | Retain ownership; ported Go tools retire at release cutover. |

R02 recount @HEAD `1714f7d0` (baseline counts above preserved as audit;
current tracked paths per root): root files 24→22, .agents 1→1,
.githooks 1→1, appliance 306→257, assets 169→170, cmd 21→80, docs 92→300,
factory-os 1→1, frontend 32→32, internal 502→430, project-os 17→0 (moved
to system/project), rust 283→120, scripts 83→77, tests 133→110, tools
40→76, lib —→206 (new), system —→64 (new). Total 1705→1947. Deltas reflect
landed A00/A05/A06/A01, B01–B05, C01/C08/C09 + CORRs/DELTA (moves, splits,
folds, retirements) plus run plan upkeep under docs/.

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
| `internal/host/publish` | `internal/forgejo/publish` | Its only production caller is `internal/forgejo`; it executes unprivileged repository publication. Keep the existing `BackgroundOperations` interface, credential isolation and factory/project domain inputs. Update the existing architecture rules to name this publication leaf correctly..* **[run 20261005:** PARTIAL — StatusError leaf moved to forgejo/publish (alias deleted, merged); package move retained for B07-full.] |
| Go acceptance Command/Evidence/process closure | Retire obsolete execution/evidence closure and its tests; no `internal/process` package | Its demonstrated process callers are the retiring Go image builder, Go worker and legacy acceptance harness. The Rust owners retain their real execution and descendant-cleanup behavior. Fold the live `PrivateFile` implementation into the existing installed probe input support before deleting `evidence.go`. |
| `tools/soda-build/worker_linux.go` | `lib/soda-release-tools` (`worker.rs`, landed #30) | The Rust worker ports the tool-side worker and is the single worker owner. Retire the Go file at the release cutover. Preserve both admitted worker identities, trusted executable checks, mount/environment rules, root dispatch and exact systemd unit cleanup..* **[run 20261005:** crate path rust/->lib/ LANDED (C08); Go file still retires at cutover.] |
| `internal/acceptance/worker_linux*` | Retire at the release cutover (no production caller left) | Its only production caller is the retired Go tool. Do not move it into the deleted tool as previously proposed. |
| `internal/release/{build,deliver,image}` (all Go) | Retire at the decided release cutover; `lib/soda-release-{build,deliver,image,tools}` (landed #30/#32/#33/#34) are the owners | The earlier Go-internal consolidation (parent `payload.go`/`identity.go`/`json.go`) is void: the decided language policy keeps the pipeline in Rust, so the Go side is deleted, not reshaped. Keep construction in build, orchestration in image and trust/signing/publication in deliver, now as Rust crates..* **[run 20261005:** lib/ crate paths LANDED (C08); Go deletion still at cutover.] |
| `internal/testoci` | Delete without replacement under L18/DEAD01 | At the investigation pin this orphaned fixture package has no imports/callers. Recheck references and delete; no Go pipeline retirement prerequisite. Keep testify and other dependencies with live consumers. |
| `rust/identity-providers` + `rust/soda-identity` | One `cmd/soda-identity` Cargo package, with private `providers/codex`, `providers/muse` and existing provider types/helpers | The provider crate has exactly one direct production dependent. Consolidate package ownership while retaining provider protocol, broker policy, storage and wire modules. Fold its manifest/dependencies into the broker; remove the old package boundary and aliases..* **[run 20261005:** LANDED (A05+A06: cmd/soda-identity + providers/codex+muse; predecessors retired).] |
| `rust/soda-asset-fetchers` + `rust/soda-stage-render` + `rust/soda-forgejo-locales` | One `tools/release-assets` Cargo package with `fetch/`, `render/`, `locales/` modules and the existing seven binaries | These are build-input operations under the existing release producer. Keep separate CLI entrypoints and test support namespaces. Existing transport/hash helpers can be reused after matching their caller behavior. This merges three manifests and library roots rather than adding another facade crate. | **[run 20261005:** LANDED (C09: tools/release-assets + 7 bins; 3 predecessors retired).]
| `rust/soda-host` | Retain at `lib/host`; one package also owns the moved `cmd/soda-host/main.rs` | Current source already owns the Rust daemon and native adapters; its manifest and release compile path select that binary. The proposed move preserves this one package/service and the surviving Go client/wire surface. No Go daemon or retired executor is a target. | **[run 20261005:** A00 in-place roots LANDED.] **[adoption reconciliation:** the crate is canonical at lib/host, with its binary still at lib/host/src/main.rs. The desired cmd/soda-host entrypoint placement remains a separate pending structural duty; parked A is assessed separately.]
| `rust/soda-project-account` + `rust/soda-project-factory-roles` | Existing `cmd/soda-project-terminal` package; `src/bin/project-account.rs`, `src/bin/project-factory-roles.rs`, `src/account.rs` and `src/factory_roles/` | Both native ports already compile into Project tools via release-build `production.rs:222-236` and install through Project Containerfile lines 40-44. Consolidate their package roots/dependencies and update these actual compile selectors; preserve binary names, installed paths and real compiled-helper tests. P03/P06 and shared owners challenge the exact split. | **[adoption reconciliation:** package fold and installed executable identities are canonical at the investigation pin; preserve completed placement and distinguish parked later seams.]

Publication evidence: `internal/forgejo/publish.go:37-58` and
`internal/host/publish/operation.go:19-35`. Process/worker evidence:
`internal/release/image/build.go:639-645`,
`internal/acceptance/worker_linux.go:137-176`, and
`tools/soda-build/worker_linux.go:89,281-307`. Release input evidence:
`internal/host/daemon.go:61-80`, `internal/release/deliver/payload.go:16-113`,
`internal/release/build/files.go:28-46,140-215`, and
`internal/release/doc.go:1-16`. Asset ownership is visible in
`internal/release/build/production.go:178-210` and the three Cargo manifests.

Provider consolidation is complete at `cmd/soda-identity`; the entrypoint stays
in `main.rs`, as recorded by the completed A05/A06 work. The former proposed
`service.rs` extraction is not a new pending task. Keep provider implementations
private, one definition of Connection/Enrollment, and actual library imports in
binary/tests. Library adoption replaces generic provider hashing/transport and
PG mechanics without duplicating modules or publishing private internals.

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
(`rust/soda-identity/src/schema.rs:1-6,190-196`) (`cmd/soda-identity/src/schema.rs:1-6,190-196` **[run 20261005:** path moved by A05; drift assertion intact.]). Current storage is PostgreSQL;
older SQLite wording in credentials/development documentation is stale.
Go metadata reads remain live in `internal/web/api/factory_settings.go:467-485`.
IdentitySaveConnection/IdentitySaveGrant now have test callers only, but moving
them into a package-local test file would break fixtures in other packages.
Any retirement must preserve the real cross-package fixture/broker boundary;
it does not justify a second production broker store or schema.

`lib/json` had twelve direct Cargo dependents at the investigation pin.
That caller count does not justify keeping its custom syntax engine. Under L04,
Serde owns syntax/binding/emission; shared profile policy stays here only where
actual callers need the same contract. Retain strict broker admission, Go binding
and terminal emission differences at their existing owners. Do not introduce a
universal permissive facade or preserve lexer modules merely because they were
previously allocated. Signed raw bytes remain raw.

Every selected Rust JSON owner has removed its engine callers and dependency
edges; `229e9cce` deletes the shared crate and workspace/lock entries. Current
responsibilities are bounded raw-byte custody, concrete record admission and
required producer formatters. Image retains ordered metadata for duplicate/raw
number identity, stable sorting and Ignition comparison. Guest retains ordinary
dictionary semantics for factory/subscription state. Acceptance retains arbitrary
QMP/Ignition/probe/evidence data and bounded structural redaction/publication.
Those owner-local trees use borrowed raw child tokens, explicit 127-container
admission and Serde grammar/escaping; they grant no operation authority. Delivery
retains its stricter depth 100 policy. JSON02 Go admission remains unchanged.

Host terminal/factory/preparation records now use caller-owned Serde DTOs
(`df4b2cf5`), with signed token and byte adapters preserving their distinct
admission policies. The final host callers and custom scanner/string/binder engine are retired in
`3bf7e75b`; strict admission, scalar/byte policy and producer formatting remain
small Serde adapters at their existing owners.
Activation and candidate/lab credential producers (`eba27412`) retain only
configuration validation and required Python-compatible output formatters;
their removed JSON machinery has no pending structural split.

### Selected engine and adapter ownership

The [packet definitions](library-adoption.md#execution-packets) supply exact
callers, selected libraries, prerequisites and acceptance. Physical writers stay
in the lane schedule; this table describes remaining responsibilities.

| Selected machinery | Remaining application owner and adapter duty |
| --- | --- |
| Host/identity Unix HTTP and WebSocket engines (N1/N2/N5/N6) | L09 source adoption complete: A owns `lib/unix-http`, the shared bounded Hyper client adapter; four callers retain endpoint, socket, credential and status policy. A-owned listeners/route/admission/backend and coupled upgrade/pump use Hyper/Tokio/tungstenite with bounded, joined ownership. C owns the in-process streaming candidate fixture and stop/file custody. Installed/native qualification remains separate |
| Native PostgreSQL parameters (SQL01) | Completed as L07 in `d12bf6d3`; B leads the Go caller conversion with an explicit A-owned Rust handoff. Both translators and Go's rebind-only wrappers are retired; direct Go DB/Tx calls and retained Rust encoding/query helpers preserve domain authority |
| Identity PG wire/DSN engine (PG01) | L08 source adapter implemented: A-owned synchronous Store/Tx uses tokio-postgres typed values, a total operation deadline, cancellation/discard/join and one connection guard for the full transaction. B retains canonical Go schema ownership and A maintains its Rust mirror |
| Hash and curve engines (CF-01/02) | Completed L03 in `52eee7ee`: sha2 and typed NIST point libraries own primitives; existing callers retain fingerprint recipes, raw bytes and uncompressed-point admission |
| SSH and Base64 engines (CF-03/04) | L05 source adoption is complete in `7b42671d`: A-owned host and C-owned installer adapters delegate formats to upstream ssh-key, retaining allowlists, scalar/point/line policy, canonical raw fingerprints and unchanged certificate wire. CF-04 retains caller-local padding, CRLF and trailing-bit profiles; broad wire/mpint/certificate engines are retired |
| Entropy callers (RNG01) | Implemented getrandom acquisition; A owns host/identity IDs, nonces and policy revision failure propagation, C owns installer/setup/maintenance/release callers and their retained custody |
| Release trust-key intake (CF-05) | C owns the shared typed P-256 adapter in `lib/release-inputs/src/trust_key.rs`; image/delivery retain role/timing/reference policy and original-DER fingerprints |
| Installer X509/DER/calendar/URL engines (X50901/CF-06) | L06 source adoption is complete in `7d063f15`: C owns bounded one-root PEM, typed strict CA/critical/algorithm policy, original DER/TBS and RustCrypto verification. The actual pinned Caddy root admits that profile; custom DER/name/SAN/calendar/SPKI engines are retired. L11 retires setup urlx with its last display caller; installed qualification stays separate |
| JSON grammars/emitters (JSON01) | L04 complete through `229e9cce`: caller-owned Serde DTO/admission/formatting and bounded dynamic-data adapters; shared grammar crate retired |
| Generic file/FD/process mechanics (L12) | Source complete through `c5cca5e7`: C operator/release and A host/guest/identity adapters use typed rustix/OwnedFd, libc/std process/path, scoped tempfile owners and Muse WalkDir. Existing callers own same-FD bounds, rooted admission, private modes, sync/exclusive publication, cancellation/EOF and direct-child vs exact-unit/container cleanup. NSS/PTY/signal, registration traversal and failed-state evidence policy remain local; B Go ownership is unchanged. No broad framework |
| Archive/OCI/XML/CLI/Cargo emulators (L13/L14) | Source complete through `c82125ee`: C-owned tar/flate2/roxmltree/svgtypes and Clap/humantime adapters retain format/renderer/command/shipping policy. Delivery owns shared bounded OCI scanning with an acyclic build dependency; foundation owns the bounded ELF accessor, with A retaining host purpose gates. One release-image Cargo metadata inventory owns snapshot identity/target/feature validation and flows through compile/staging; fixed shipping tables remain authority. Installer uses Rust literal formatting. Installed/shipping qualification stays separate |
| External HTTP (N3/N4) | C-owned setup ureq adapter retains credential/status policy. Host provider curl/Executor remains held on resolver-inclusive deadline and host-native custody; C owns proof, then A owns the host adapter |
| URL/IP/time (N7/N8/N9) | A owns the shared strict `lib/wire-time` profile and host/identity/guest adapters; C owns operator/release handoffs. url/percent-encoding/std IP/time own grammars; callers retain raw authenticated literals, zones/ranges/expiry and shared Linux timing origin |
| Evidence matching/deadlines (RED01/N13/N14) | C-owned evidence publication and bounded overlap adapter; url owns valid token interpretation and malformed tokens are omitted. Small Phase/QMP lifecycle policy remains |
| Native effective configuration (CFG01) | C owns evidence and effective-key policy; parser cutover remains blocked on the actual Forgejo corpus |

KEEP01, N12, JSON02, CFG02, CLI01 and SQLITE01 retain their narrow application
contracts. Consolidating primitives does not reopen completed provider/package
moves, duplicate domain records or create credential/database/codec services.

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
