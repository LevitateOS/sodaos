# Package ownership and execution topology

## Current package and path mapping

Source `e0e2413f` (2026-10-09): 2,541 tracked paths and 28 tracked Cargo
packages (`cmd`: 14, `lib`: 8, `tools`: 6). These are source/package counts,
separate from the historical audit and runtime evidence below.

| Current root | Tracked paths |
| --- | ---: |
| (root files) | 20 |
| .agents | 1 |
| .githooks | 1 |
| assets | 175 |
| cmd | 336 |
| docs | 318 |
| frontend | 345 |
| internal | 453 |
| lib | 420 |
| scripts | 80 |
| system | 65 |
| tests | 138 |
| tools | 189 |

Current crate, binary, module and caller selectors govern the target tree.
The Project-terminal package uses `src/main.rs` and its two `src/bin` helpers;
there is no selected `src/lib.rs`. `lib/host/Cargo.toml` keeps `src/main.rs`
as the daemon entrypoint. Current Tailnet enrollment/Project methods remain
at their explicit defining owners; optional directory changes are superseded.
The D06 Rust personal-Git payload remains planned, with its real caller and
credential cutover unresolved. Historical root/port counts below are preserved
as earlier observations, rather than current source facts.

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

| Historical root | Tracked paths | Historical proposed disposition |
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
| `internal/forgejo/publish` | Retain `internal/forgejo/publish`; predecessor `internal/host/publish` retired | B07.M's publication move and operation/observation/validation/push/receipt plus test splits landed in `33682cfc`; its merge-completion split landed in `6cb54cf9`. Direct callers and architecture assertions name the successor, with one `BackgroundOperations`/`StatusError` owner. Native effects, F09/F10/F12 corrections and Q5 remain separate; see current reconciliation in [G04](reviews/G04.md) and [G07](reviews/G07.md). |
| Go acceptance execution/evidence closure | Retired; no `internal/process` package | The old `Command`, `Evidence`, `Execute`, `Remote` and uncalled worker closure are retired. Go `StartCommand`/`Process` remains live for actual installed-probe callers. `privateFile` remains live in installed-probe input support. |
| `tools/soda-build/worker_linux.go` | Retired; `lib/soda-release-tools` owns the current worker | The Go worker implementation and exclusive tests have been removed. Preserve the Rust worker's real tool duties; this is completed source ownership, separate from runtime qualification. |
| `internal/acceptance/worker_linux*` | Retired with the obsolete Go worker caller | No target implementation or compatibility worker remains. Go `StartCommand`/`Process` is retained independently for installed probes. |
| `internal/release/{build,deliver,image}` (Go) | Retired; `lib/soda-release-{build,deliver,image,tools}` are the current owners | The Rust release pipeline is wired and the Go implementation is gone. Preserve Rust construction, orchestration and trust/signing/publication duties; do not treat the old Go files as pending cutover work. |
| `internal/testoci` | Retired without replacement in `eaed66a9` / L18-DEAD01 | Current import/caller/build registration checks confirmed the orphan. Active OCI fixtures stay with their actual Rust owners; testify retains live Go test consumers. No Go pipeline retirement prerequisite or replacement adapter. |
| `rust/identity-providers` + `rust/soda-identity` | One `cmd/soda-identity` Cargo package, with private `providers/codex`, `providers/muse` and existing provider types/helpers | The provider crate has exactly one direct production dependent. Consolidate package ownership while retaining provider protocol, broker policy, storage and wire modules. Fold its manifest/dependencies into the broker; remove the old package boundary and aliases..* **[run 20261005:** LANDED (A05+A06: cmd/soda-identity + providers/codex+muse; predecessors retired).] |
| `rust/soda-asset-fetchers` + `rust/soda-stage-render` + `rust/soda-forgejo-locales` | One `tools/release-assets` Cargo package with `fetch/`, `render/`, `locales/` modules and the existing seven binaries | These are build-input operations under the existing release producer. Keep separate CLI entrypoints and test support namespaces. Existing transport/hash helpers can be reused after matching their caller behavior. This merges three manifests and library roots rather than adding another facade crate. | **[run 20261005:** LANDED (C09: tools/release-assets + 7 bins; 3 predecessors retired).]
| `rust/soda-host` | Retain `lib/host` and its defining entrypoint `lib/host/src/main.rs`; supersede proposed `cmd/soda-host/main.rs` placement | The existing Cargo manifest and fixed runtime package/bin selector already build the 478-line entrypoint. A source-only `cmd/soda-host` directory would be classified as Go and compiled alongside the Rust target. Changing discovery to select it as Rust would also affect `inventory.commands()`, which feeds candidate asset hardlinks through `build_candidate` and `payload_stage::link_candidate_commands`. That requires additional release policy for no caller, engine-removal or ownership gain. Retain the current package/bin/service and language boundaries. | **[run 20261005:** A00 in-place roots LANDED.] **[adoption reconciliation:** the optional entrypoint placement is superseded after current source and actual Cargo metadata inspection; native/installed qualification remains separate.]
| `rust/soda-project-account` + `rust/soda-project-factory-roles` | Existing `cmd/soda-project-terminal` package; `src/bin/project-account.rs`, `src/bin/project-factory-roles.rs`, `src/account.rs` and `src/factory_roles/` | Both native ports already compile into Project tools via release-build `production.rs:222-236` and install through Project Containerfile lines 40-44. Consolidate their package roots/dependencies and update these actual compile selectors; preserve binary names, installed paths and real compiled-helper tests. P03/P06 and shared owners challenge the exact split. | **[adoption reconciliation:** package fold and installed executable identities are canonical at the investigation pin; preserve completed placement and distinguish parked later seams.]

Current scoped placement also retains `tools/soda-rootfs-server`: `d89f4563`
moved its Go command/tests and rebound service references. These B07/rootfs
updates do not refresh the historical root counts or import census above.
Current R02 target/owner/source-join mapping has its separate accepted
[source crosswalk](build-and-operational-joins.md#current-r02-source-join-crosswalk).

Historical publication evidence: `internal/forgejo/publish.go:37-58` and
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

The former Go process extraction is withdrawn, but Go process support is not
wholly retired. `internal/acceptance/installed.go` is a live caller of
`StartCommand`; its bounded `Process` implementation and cleanup test remain
part of installed-probe ownership. `privateFile` remains live through the same
installed-probe input owner, including its developer-access callers. Retired
release-image, worker and legacy Command callers do not justify keeping the
separate Evidence/Execute/Remote/Command closure. Preserve bounded probe behavior
without a new process package or compatibility wrapper.

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
Release-image's `model::SCHEMA_VERSION` supplies the emitted `Payload.schema`.
The coordinator updates that C-owned version with the B schema and A broker
mirror; the existing release-model equality regression checks both owners.
Current sponsorship metadata admission in
`internal/web/api/factory_sponsorship.go` uses the existing private
`internal/identity/client` broker-admin connection and grant readers. Rust broker
routes/controller/Store remain the production metadata owner; the dashboard's
former `Identity*` metadata methods and `internal/store/identity.go` are retired.
Cross-package tests and native-demo fixtures use `SeedIdentityConnection` and
`SeedIdentityGrant` in the single `internal/store/identity_fixture.go` owner,
retaining canonical sealing and atomic append. Keep that real fixture boundary;
it does not introduce a second production broker store or schema. Separate
factory execution actor and Project attribution remains a product/correctness
question, independent of this completed metadata cutover.

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
configuration validation and current Python-style output formatters; their
spelling is implementation behavior, not a permanent unsigned byte contract.
REP-FMT-1 supersedes pending historical-format equivalence with actual consumer,
semantic and current deterministic-output checks. Their retired JSON engines
have no pending structural split.

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
| Archive/OCI/XML/CLI/Cargo emulators (L13/L14) | Source complete through `c82125ee`: C-owned tar/flate2/roxmltree/svgtypes and Clap/humantime adapters retain format/renderer/command/shipping policy. Delivery owns canonical OCI archive/content inspection and shared bounded scanning with an acyclic build dependency. SIMP-REL-OCI-ARCHIVE-1 at `2cba04d1` retires the duplicate build engine and transfers callers/types directly; build retains export/config-ID/raw-hash policy; tools retains distinct ProducedImage/archive, payload, trust, permit and credential projections, while SIMP-REL-OCI-IDENTITY-1 removes the equivalent seven-field identity copy. No new package or forwarding facade; foundation owns the bounded ELF accessor, with A retaining host purpose gates. One release-image Cargo metadata inventory owns snapshot identity/target/feature validation and flows through compile/staging; fixed shipping tables remain authority. Installer uses Rust literal formatting. Installed/shipping qualification stays separate |
| External HTTP (N3/N4) | Setup retains its ureq adapter and credential/status policy. L10.N4 source scope is complete: the existing host curl/Executor path and generated host package inputs are retained. Resolver-inclusive deadline, cancellation, joined transfer, native marker and uncertain-operation behavior have source checks; shipped executable/RPM, TLS and live-provider qualification remain open under R04 |
| URL/IP/time (N7/N8/N9) | A owns the shared strict `lib/wire-time` profile and host/identity/guest adapters; C owns operator/release handoffs. url/percent-encoding/std IP/time own grammars; callers retain raw authenticated literals, zones/ranges/expiry and shared Linux timing origin |
| Evidence matching/deadlines (RED01/N13/N14) | C retains current matching, evidence publication and completed L01/L12 deadline/read/output policy; url owns valid token interpretation and malformed-token omission. L16.G aggregate collection is complete with its selected count/byte limits and refusal-before-capture checks. Optional Aho-Corasick remains deferred; escaped-pattern admission and bounded per-file staging remain separate |
| Native effective configuration (CFG01) | C retains `cmd/soda-forgejo-domain` and its descriptor-based marker custody. L17 removes the host INI/env reader and its parser split. Recovery derives the fixed Quadlet declaration; the sourced native setup overlay enforces it after config-alias consumption, and the extension oneshot repairs the same path after migration. Actual stage/binding checks require both startup declarations to agree. Deployed generator/service and installed identity remain R04; CFG02 locale scanner remains independently retained |

KEEP01, N12, JSON02, CFG02, CLI01 and SQLITE01 retain their narrow application
contracts. Consolidating primitives does not reopen completed provider/package
moves, duplicate domain records or create credential/database/codec services.

### Current correction and simplification reservations

The [finding allocation](execution-findings.md) defines exact post-adoption
subtasks under the existing plan. It changes no process/language boundary and
adds no generic helper, third DTO, runtime or state owner. Selective reservations:

| Current defining duty | Accountable finding owner / physical reservation | Integration boundary |
| --- | --- | --- |
| Identity Controller/Store/Tx, settings/probe/list producers, typed rows and strict readers | A / existing cmd/soda-identity and host readers | New acquisition/close/bounds follow-ups remain distinct from completed broker fence. B retains canonical schema and explicit Go list-consumer handoff; profiles precede representations |
| CoreOS/Tailnet live-input records and shared validation | C / `lib/release-inputs/src/reader/stream.rs` | One Serde wire owner for build, image and tools. `SIMP-REL-WIRE-1` source cut is complete at `3ccbc127`: build retains network resolution, bounded read/fresh exclusive write and file/hash lifecycle; image consumes canonical types; the current `lib/soda-release-tools/src/pipeline.rs` has no record clones or sort bridges. ORDERED-1 and INSTALL-OCI-1 remain separate. Native/installed qualification is outside this source receipt. |
| Read-only build/image OCI identity | C / `lib/release-inputs/src/reader/image.rs` | SIMP-REL-OCI-IDENTITY-1 is source complete in `e98bbb0e`: the foundation reader is the defining seven-field handoff owner, with direct delivery/buildx and image/model re-exports. The duplicate records, unused image-only alias parser and pipeline field-copy adapter are removed; local Candidate, ProducedImage, payload validation and original-byte hashes stay at their existing owners. Existing dependency edges/versions are unchanged. The installer retains its internal directory-layout result without a new foundation runtime dependency; no workspace-wide single-result-type or native qualification claim. |
| Go dispatch/publication/review/shutdown and saved-key mutation | B / existing dashboard/domain/Store/API/browser | One Store/publication writer across F08/F09; B sends only exact stop-receipt changes to A. Saved preferences differ from installed Project access; original operation identity is retained without a second ledger |
| Host Muse child/listener and stop-owned run receipts | A physical writer; B remains F08-F3 accountable | F08-F3 receipt settlement is source complete under the existing per-run lock, with deterministic stale-snapshot regressions and independent medium review. Exclusive current launch/finish/receipt files, supervisor/FD joins, identity-safe signal/reap and bounded workers remain with A. Preserve terminal repair and parked C41; no new process framework or native qualification claim |
| Go installed capture and C verification tooling | OBS-G01 B via temporary exclusive C11 handoff; remaining scripts/tools C | Reserve installed.go/process.go together, return after integration. Per-finding ownership overrides the generic directory default; one current writer and actual consumers |
| Release worker/evidence, provisioning and installer CA/output duties | C / existing source owners | Same-file failure/admission repairs precede formatter removal. Feature-only trim preserves current CA algorithms; exact permanent profile consumes actual producer/guide decision. Real Runner/Production, signed originals and shipping selectors remain |
| Dependencies, shared schemas/compile/payload joins and full target refresh | Coordinator integration, with canonical B schema, A broker mirror and C emitted release version | Workers supply exact edge/tuple changes. Full tree/decomposition regeneration waits for implemented boundaries, while current source inventory and selective ownership remain maintained |

## Existing execution topology

These are source-verified deployment definitions and callers, not observations
of the currently running machine. The proposed package/file moves add **zero
daemons, containers, sockets or sidecars**.

| Existing owner | Execution boundary | Refactor consequence |
| --- | --- | --- |
| Dashboard + web/API/auth + factory/control + store | Existing unprivileged dashboard process; private host/identity calls and PostgreSQL | Factory coordination remains an object inside this process. No new factory service. |
| soda-host + project/terminal/Tailnet executors (Rust in current source) | Existing root service; root:soda Unix operation socket; real host namespaces | Keep the fixed privileged surface, package/bin entrypoint and launch handling in this process. The proposed entrypoint placement is superseded as recorded above. Installed behavior remains separately unverified. |
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
