# SodaOS ideal file tree plan

This is the living plan for reorganizing the entire SodaOS repository by
responsibility and decomposing oversized implementations into concrete concern
files. It covers every tracked source, test, document, asset, manifest, hidden
file, and system definition. The proposed paths below do not implement a move or
change product behavior. Existing ownership and trust boundaries remain the
starting point for the refactor.

Implementation is deferred until time is available. Keeping this plan current
is a separate, smaller task: update it after every merge, including merges that
do not change the proposed structure. This is the single forward-looking plan;
revise it in place rather than create a replacement plan or accumulate a merge
diary. Maintaining it does not authorize executing the refactor.

Last maintained: **2026-10-05**. The reconciled source baseline and coverage are
recorded below. This maintenance pass checks the six landed port PRs, closes
obsolete Go ownership in the target, and reviews the newly oversized files.
Pending source cutovers remain explicit; proposed paths are not landed code.

Decided language policy (owner): network-facing servers stay Go, system and
privileged applications are Rust, Python is eliminated from the tracked tree.
Open language options from earlier revisions are closed: the Rust host, installer,
import, acceptance and release crates are the retained owners, and their Go or
Python predecessors retire at cutover. The proposed tree below is the
post-cutover target: it contains no pre-port implementation, even where the
cutover commit is still pending. Pending retirements are called out as
decided-pending with the PR or cutover that executes them.

The current source has 30 Cargo packages under `rust/`. The recommendation merges
the three release-asset packages into one and folds the sole-consumer identity
providers into the broker; the Rust host library is retained as the daemon
foundation. That leaves 27 Cargo packages. Its daemon entrypoint belongs to the
same `soda-host` Cargo package, with a `[[bin]]` path under `cmd/soda-host`; it
does not add a package or a second service. The four
landed release crates (`soda-release-build`, `soda-release-deliver`,
`soda-release-image`, `soda-release-tools`) are placed 1:1 under `lib/`; whether
they consolidate further, and where their duplicated build helpers dedupe, is an
open owner decision in the pending filetree layout. The tree places
installed commands in `cmd/`, named implementation libraries in `lib/`, support
tools in `tools/`, authored browser surfaces in `frontend/`, and image/service
definitions in `system/`. Canonical assets stay in `assets/`; the website handbook
source stays at `docs/public/`.

Navigation: [package ownership](#package-ownership-consolidation-and-splitting),
[execution topology](#existing-execution-topology),
[port assessment](#was-the-rust-port-a-good-choice),
[complete tree](#complete-proposed-tree), and
[file decomposition](#detailed-decomposition). The
[post-merge workflow](#keeping-the-plan-current-after-every-merge) owns ongoing
maintenance.

## Source coverage

Last reconciled source: `d7e565aa1019753997a99fd430ba103d7a472b48`. Inventory at that baseline: **1705 tracked paths**. This correction changes only this document.

Initial full structural review: `899e9bf3b9b57883df42cbb520329679b3b8186d`. Latest reconciliation scope: six landed port PRs (tests to Go #29/#31, release tools #30, release image #32, release deliver #33, release build #34) plus the decided language policy. Keep this initial review separate from later merge checkpoints.

| Current root | Tracked paths | Proposed disposition |
| --- | ---: | --- |
| `(root files)` | 24 | Retain live tool inputs and ownership files; omit two tracked ELF outputs and the two Python-only tooling inputs at their cutover. |
| `.agents` | 1 | Retain ownership; include every leaf below. |
| `.githooks` | 1 | Retain ownership; include every leaf below. |
| `appliance` | 306 | System definitions to system; Forgejo presentation to frontend. |
| `assets` | 169 | Retain ownership; include every leaf below. |
| `cmd` | 21 | Preserve installed command identities; replace the Go host entrypoint with the same-package Rust binary and retire its Go-only tests. |
| `docs` | 92 | Preserve canonical owners; retire the Python-only tooling guide when its final callers move. |
| `factory-os` | 1 | system/factory. |
| `frontend` | 32 | Retain ownership; include every leaf below. |
| `internal` | 502 | Preserve domain/trust boundaries and live installed probes; relocate publication to Forgejo. Retire the obsolete acceptance execution/evidence/process closure, Go release pipeline and privileged host server/executors. Keep host clients and their pure wire types. |
| `project-os` | 17 | system/project, preserving rootfs paths and installed identities. Two extensionless Python helpers port at PR42. |
| `rust` | 283 | All 30 packages accounted for: 27 proposed packages after consolidation; host library and daemon ownership retained. |
| `scripts` | 83 | Retain build/check entrypoints; move SELinux and rootfs service definitions to their owners. |
| `tests` | 133 | Preserve actual assertion coverage; retire `.py` predecessors and replace Python import drivers with tests of their native owners. |
| `tools` | 40 | Retain ownership; ported Go tools retire at release cutover. |

Standalone Python inventory at this baseline: **25 tracked programs** — 23 `.py` files (21 in `tests/build`, 2 in `tests/installed`), plus 2 extensionless libexec helpers (`project-account`, `project-factory-roles`). Go test/probe files have landed, but several still execute Python source or import these predecessors. The Python cutover must replace that embedded execution and its test drivers as well as the 25 files; it is not complete merely because assertions moved to Go. Installed helper paths remain stable as compiled outputs, not tracked Python sources. See [Python cutover closure](#python-cutover-closure).

There are **207 text files over 400 lines**, of which **190 are executable source, style, template, or test files**. Counts include embedded tests and are not production-only line counts. The 17 remaining files are data, locks, licenses or documents, including this plan. A size flag prompts a responsibility review; it does not justify arbitrary chunks or new product machinery.

Detailed concern reviews cover **190 oversized code files**; **0 remain unreviewed in this snapshot**. This includes all 31 newly oversized Rust release files and `tests/build/project_factory_roles_test.go`, in addition to the original 158 entries. Retiring sources have explicit removal dispositions; their old decomposition suggestions are not target leaves. Proposed filenames are responsibility seams, not measured final file sizes.

## Package ownership, consolidation and splitting

Directory renaming is only the first part of this proposal. Package ownership
must reduce duplicated implementation while keeping domain policy, credential
custody, browser authorization, persistence and privileged execution distinct.
The review traced all current Go package imports and all 30 Cargo workspace
members, then checked the actual binary compilation and installation recipes.
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
| `rust/soda-host` | Retain at `lib/host`; one package also owns `cmd/soda-host/main.rs` (decided-pending) | The host cutover wires terminal/project executors and mux into the existing privileged service and removes the Go server/execution half. Current local `pr/26` contains executor/mux library changes, not a daemon binary or Go deletion. The post-cutover tree carries the Rust owner and Go client/wire support only. |

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
| soda-host + project/terminal/Tailnet executors (Rust after pending PR26; Go until its cutover) | Existing root service; root:soda Unix operation socket; real host namespaces | Keep the fixed privileged surface and existing launch handling in this process. The cutover replaces the daemon implementation inside the same service; it adds no second host daemon. |
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

## Was the Rust port a good choice?

The source supports selective assessment, not a blanket language verdict. No
startup, memory, throughput, binary-size or runtime-defect comparison was run
or found in this review. Source-line totals describe authored maintenance,
including comments and tests; they do not measure performance or safety.

| Port | What the current source establishes | Recommendation for this refactor |
| --- | --- | --- |
| Host project/preparation/Tailnet library | Decided Rust owner; local `pr/26` adds terminal/project executors and daemon mux library code, while the Go daemon remains installed | Retain at `lib/host`. Complete the entrypoint/service adapter and Go server/executor deletion at cutover. The proposed tree excludes those Go predecessors already. |
| Project terminal | Replaces four embedded Python programs at one existing installed helper boundary; native file descriptors, PTY and process handling stay inside the Project | Retain one helper and split its real terminal/subscription/key concerns. This is the strongest current boundary for a selective native port; benefit still needs runtime evidence. |
| Identity broker | Real replacement of the former Go broker; retains an established custody service | Consolidate provider packages. Reassess handwritten PostgreSQL transport/codecs and cross-language schema/wire maintenance before expanding them. The separate service predates the language change. |
| Installer + image import | Decided Rust owners; the Go release code they duplicate retires at the release cutover, resolving the duplication from the Go side | Retain both. After the Go pipeline retires, select the single surviving payload/OCI implementation per guarantee; do not solve reuse with FFI, RPC or another service. |
| Acceptance | Rust owns outside driver/remote handling; Go installed probe orchestration remains live | Retain Rust acceptance and its remote binary. Retire obsolete Go driver/evidence/process execution and tests; preserve live probe input support. Replace embedded Python payloads and tests that still execute predecessors before claiming cutover complete. |
| Factory CLI + Muse wrappers | Actual thin client/helper duties; Rust adds manual flag, transport and Go compatibility parsing | Reconsider language alongside shared primitive consolidation. The current source provides no reason to migrate the dashboard coordinator or add a client-side daemon. |
| Asset fetch/render/locales | Actual build-time replacements under release orchestration (Rust after cutover); several recreate Python parsing/CLI behavior | Consolidate the seven binaries into one build-tools package. The language decision is made; no alternative-stack migration follows. |
| Release pipeline + tools (new) | Four Rust crates landed (#30/#32/#33/#34) and cut over: tools → image → build/deliver is wired, candidate-check and payload data preserved, Go removed | Retain the four crates under `lib/` and split their actual concerns below. |

The installer contains **17,697 Rust source lines**, including comments,
blank lines and embedded tests. The previous installer at the parent of
`977ebe55` had **3,590 lines in production Go files** and **2,184 in Go test
files**. Scope differs: the Rust tree copied shared release code and recreated
upstream/standard-library behavior. This is evidence of retained implementation
cost, not a like-for-like production statement ratio. In particular,
`rust/soda-install/src/x509.rs` is 3,624 lines and explains its Go DER/error
fidelity at lines 1-27; its local-CA caller is
`rust/soda-install/src/setup.rs:602-619`. Smaller files
would leave that emulation cost intact.

Installer/import duplication is explicit in
`rust/soda-image-import/src/main.rs:1-10,550-669,1078-1200` and
`rust/soda-install/src/deliver.rs:106-284`, compared with live Go owners at
`internal/release/deliver/payload.go`, `content.go` and
`internal/release/build/oci_layout.go`. The live host still uses payload loading
and native admission at `internal/host/daemon.go:61-80`. Port history was checked
at `977ebe55` (installer), `0123e689` (import), `c753342a` (acceptance),
`a096a06c` (identity), `c439c209` (host library), `899e9bf3` (terminal),
`999f42ff` (factory CLI), `1c3a7cdc` (Muse), `5abc5733` (stage/render) and
`179a8723` (locales). The release cutover wired tools → image →
build/deliver, preserved candidate-check and payload data, and removed the
Go pipeline: Rust orchestrates the release producer.

### Ownership decisions before further splitting

| Current code | Observed consumers | Required disposition decision |
| --- | --- | --- |
| Go acceptance Evidence/Execute/Remote/Command/process closure | Legacy tests and retiring Go release/worker callers; PrivateFile remains needed by installed probes | Retire the obsolete closure and its tests. Retain PrivateFile in probe input support. The Rust harness choice is closed; no permanent Go process package remains. |
| Go deliver/import.go | Whole Go deliver package retires at the release cutover (Rust decided) | Retire with the package; the thin-Go-command alternative is void. |
| Rust release-inputs readers | Forgejo reader is used by release-build `forgejo.rs:10`; signature admission by `coreos.rs:398`. Settings helpers have no external production caller; production.rs duplicates them locally | Retain Forgejo/signature and the other live readers. Consolidate matching recipe/unit/command inventory helpers into the existing settings owner and replace the duplicate release-build helpers, preserving their actual semantics. Do not retire the live reader closure or create a helper service. |
| Go broker credential-write helpers | Seed/fixture callers; metadata reads still live | Move test-only seeding through the real fixture/broker boundary when authorized. Preserve live reads, schema and persisted encryption contract. |

The main tree represents the selected final owners. Retiring sources are
accounted for in disposition tables and review entries, not carried as target
implementations. Pending wiring or consolidation details do not reopen the
decided language choices.

### Current source integration concern

`rust/soda-acceptance/src/driver.rs:476-485` reads
`soda-acceptance-remote` beside the current driver executable, and native action
uses those bytes at lines 528-536. Candidate assembly at
`internal/release/image/build.go:490-508` copies the acceptance driver but has no
entry for that sibling. Compiling the Cargo package does not copy every binary
into the candidate payload. The Rust image owner must deliver and record the
paired payload with its correct target binding at cutover. This is a source-level
integration finding, not a reproduced candidate-run failure.

### Host cutover integration

The current Rust host is a library. The locally inspected `pr/26` revision is
`b306756d00c6801278bb205ea0c8bc9de1d0456a`: its executor/mux modules are pending
source, not the reconciled main baseline. It has no real daemon entrypoint or
concrete native backend, and does not delete Go. The target carries one
`lib/host/Cargo.toml`, with a `soda-host` binary declared at
`../../cmd/soda-host/main.rs`. That entry imports the same library, not copied
modules; `cmd/soda-host` has no second manifest.

Keep these Go client files whole: access_keys.go, client.go,
factory_candidate.go, factory_client.go, lifecycle.go, os.go, prepare.go,
profiles.go and terminal_client.go under `internal/host`. Keep only client
methods in identity.go and tailnet.go. Move existing TerminalRequest/State/Frame,
FrameLimit and name validation from terminal/types.go into host/terminal.go;
move the bounded Write operation from terminal/service.go:299-311 into
terminal_client.go. Remove executor aliases, native server methods and imports.
All Go host project/terminal/Tailnet executor directories retire. Keep canonical
Go project/identity records and actual client tests; do not delete those simply
because their filenames mention a privileged operation.

Go `internal/tailnet` retains control_types.go, control_validation.go,
ProjectSelection/Validate from project_runtime.go, and the actual status/
Endpoint client plus its tests. Protected native control, policy publication,
provider/enrollment and RunBinding execution move to Rust. The ProjectStatus
parser has no surviving Go production caller after companion retirement and
already has a Rust counterpart; it retires too.

These integration leaves are required by existing callers and are
**decided-pending**, not supplied by the inspected branch:

| Target | Existing responsibility and evidence |
| --- | --- |
| `cmd/soda-host/main.rs` | same -config/-tailnet-action/-project CLI/root arguments/exit0,78,1; binary calls soda_host rather than compiling duplicate library modules. Evidence: cmd/soda-host/main.go:20-38,41-70,111-149 |
| `lib/host/src/daemon/mod.rs` | one host construction and root fd3 HTTP plus current Muse unixpacket listener, signals/draining/terminal closure and same-binary tailnet action; not implemented on inspected PR26. Evidence: cmd/soda-host/main.go:72-110;launch.go:12-57;internal/host/daemon.go:212-254 |
| `lib/host/src/daemon/config.rs` | full current host.json/defaults/runtime/Muse config; pending port. Evidence: internal/host/daemon.go:31-210,636-644;pr/26 project.rs:78-84 subset |
| `lib/host/src/daemon/backend.rs` | one concrete native project/prepare/factory/terminal/Tailnet/identity adapter; pending port, never production 501 stub. Evidence: pr/26 gmux_backend.rs:64-128 trait and130-222 stub;GMUX_PATCHES sections2,4 |
| `lib/host/src/daemon/websocket.rs` | current first-frame/expiry/write/bidirectional native terminal pump and stream lifetime; pending port under same transport owner. Evidence: Go terminal/service.go:120-311;pr/26 texec.rs:8-10 excludes Write/pumps/Handler;gmux_backend.rs:119-127 requires pump_terminal |
| `lib/host/src/daemon/broker.rs` | private adapters to existing soda-identity Unix client contract; pending port, same broker and sockets. Evidence: Go daemon.go:212-236;pr/26 pfactory.rs:1326 FactoryBroker,texec.rs:2523 IdentityBroker,muse.rs:499 MuseHooks |
| `lib/host/src/tailnet/control/mod.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go control.go:32-59,412-490,624-651; current Rust tailnet_companion.rs:90-103 is only TailnetControl trait. One real same-package native Control owner must be ported. |
| `lib/host/src/tailnet/control/native.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go control.go:27-30,102-411,490-628: bounded Tailscale Unix HTTP/process observations and confirmed host actions. |
| `lib/host/src/tailnet/control/provider.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go control.go:61-101,629-639;enrollment.go:36-181: scoped bounded OAuth credential check/project auth-key issuance. |
| `lib/host/src/tailnet/control/policy.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go policy.go:23-235: protected directories/lock and atomic policy publication. |
| `lib/host/src/tailnet/control/enrollment.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go policy_enrollment.go:10-211;enrollment.go:183-252: enrollment CAS/default/admission/rotate/disable and fenced consumption. |
| `lib/host/src/tailnet/control/project.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go policy_project.go:9-167;project_runtime.go:29-71: per-project saved policy/binding/CAS and native RunBinding callback. |
| `lib/host/src/tailnet/control/wire.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go control_types.go:30-288;control_validation.go:5-207: Rust mirror of actual Go request/response contracts, not alternate records. |
| `lib/host/src/tailnet/control/tests.rs` | decided-pending port; neither current main nor inspected PR26 implements this native control/policy adapter. Evidence: Go control/policy/enrollment/recovery/project runtime server cases: preserve existing semantics in same Rust owner before predecessor retirement. |

The existing service, fd3 operation listener, Muse unixpacket listener and
same-binary Tailnet action remain the execution boundaries. The Rust backend
must replace the full real operations rather than install a production stub.
Native terminal admission/expiry/first-frame/write/pump lifetime and the current
identity socket contract are part of cutover, not another daemon proposal.

The pending branch also contains oversized implementations. Their inspected
concern allocations are below; individual test cases and fixtures stay intact.
They do not increase the current-main coverage count of 190. Reconcile their
final merged bytes before claiming a fresh review of them:

**rust/soda-host/src/gmux_admission.rs** (403 lines on the inspected branch).

- `lib/host/src/daemon/admission.rs` — 1-289: path/body/action validators, AdmissionGate/Guard and TerminalGate/Slot.
- `lib/host/src/daemon/peer.rs` — 290-403: PeerCred/MusePeer, peer_cred, muse_peer and close_pidfd; existing Unix attestation.

**rust/soda-host/src/gmux_backend.rs** (222 lines on the inspected branch).

- `lib/host/src/daemon/backend.rs` — 1-128: BackendError/TerminalSession and ExecBackend trait; concrete production adapter remains pending.

**rust/soda-host/src/gmux_routes.rs** (523 lines on the inspected branch).

- `lib/host/src/daemon/routes.rs` — 1-341: current 33-route table, DaemonConfig/RouteOutcome and native/identity/tailnet/terminal dispatch.
- `lib/host/src/daemon/response.rs` — 342-425: status reasons, error/not-found/JSON/Tailnet response wire.
- `lib/host/src/daemon/websocket.rs` — 426-523: upgrade/accept key and inline RFC6455 SHA-1/base64; pending terminal pumping belongs this same transport owner.

**rust/soda-host/src/gmux_server.rs** (364 lines on the inspected branch).

- `lib/host/src/daemon/http.rs` — 1-364: accept loop, handle_connection, read_request, drain_head, percent_decode and systemd_listener; keep the existing algorithm cohesive.

**rust/soda-host/src/pfactory.rs** (5728 lines on the inspected branch).

- `lib/host/src/factory/mod.rs` — 1269-1371,1577-1625: FactoryTerminal/FactoryBroker, Secret/RunLock/Factory, open_factory/harness_pin; one state/lock owner.
- `lib/host/src/factory/run.rs` — 40-198,363-547: factory phase/run/path validators, FactoryRun and FactoryLaunch.
- `lib/host/src/factory/deadline.rs` — 199-362: RFC3339Nano parse and current native deadline math.
- `lib/host/src/factory/identity.rs` — 548-848: whole Binding/Lease/AcquireRequest declarations and their Spec tables/codecs.
- `lib/host/src/factory/state.rs` — 849-983: OutputSlice, FactoryState and FactoryHarnessPin.
- `lib/host/src/factory/requests.rs` — 984-1268: whole inspect/stop/takeover/output/export/candidate request and response declarations/codecs.
- `lib/host/src/factory/receipt.rs` — 1372-1576,1626-1795: FactoryReceipt decode/state mapping and lock/load/store/tombstone/write.
- `lib/host/src/factory/launch.rs` — 1796-1936,2023-2067: launch/drive/consume_start/update_receipt; same sequence and custody.
- `lib/host/src/factory/finish.rs` — 1937-2022,2068-2255: fail/abandon/refresh_stopped/yielded/start-failure/timeout/finish and uncertainty mapping.
- `lib/host/src/factory/stop.rs` — 2256-2395,2467-2507: stop, reconcile_run_credential and record_stop_outcome.
- `lib/host/src/factory/inspect.rs` — 2396-2466: inspect/takeover and exact copy destination.
- `lib/host/src/factory/artifacts.rs` — 2508-2605: existing bounded output/export operations.
- `lib/host/src/factory/candidate.rs` — 2606-2751: inspect_candidate/run_reason/protected candidate script.
- `lib/host/src/factory/confirmation.rs` — 2752-2968: confirm_factory_*, confirm_prepare* and exact status mappings.

Existing test allocation: common; wire; launch; stop; artifacts; candidate; confirmation. Source grouping: 2969-3536 existing fixture/mock closure; wire/receipt cases3537-4162; launch4163-4564; stop/inspect4565-4809; takeover/output/export4810-5182; candidate5183-5452; confirmations5453-5728. Keep cohesive vectors/fixtures intact; individual case boundaries rather than arbitrary ranges determine implementation placement.

**rust/soda-host/src/texec.rs** (4826 lines on the inspected branch).

- `lib/host/src/terminal/mod.rs` — 1495-1505,2365-2442: Service/EndIdentityHook, private request admission and StreamTable; one Service owner.
- `lib/host/src/terminal/protocol.rs` — 40-255: limits/errors, terminal IDs/names/dimensions and existing strict base64.
- `lib/host/src/terminal/request.rs` — 256-360,384-420,554-658: TerminalRequest/TerminalState whole declarations, specs, codecs and request admission.
- `lib/host/src/terminal/frame.rs` — 361-383,421-553,659-787: TerminalFrame declaration/specs/decode/encode and frame/credential validity.
- `lib/host/src/terminal/identity_wire.rs` — 788-983,1065-1080,1119-1178: RFC3339/string-i64 and whole Binding record/spec/codec.
- `lib/host/src/terminal/lease.rs` — 984-1064,1081-1118,1179-1320: Lease/Delivery/AcquireRequest whole record/spec/codecs.
- `lib/host/src/terminal/target.rs` — 1321-1494,1506-1619: TerminalInspection/isolation/id-map/exit code and exact project/factory-container methods.
- `lib/host/src/terminal/identity_protocol.rs` — 1620-1743,1995-2027: IdentityRequest, reservation/binding checks and identity_result.
- `lib/host/src/terminal/identity.rs` — 1744-1994,2028-2110: identity_call/prepare/identity/managed_end, target/harness verification and credential-bound transfer.
- `lib/host/src/terminal/native.rs` — 2111-2364: agent path/hash/argv, output line decode and NativeAttach/Drop.
- `lib/host/src/terminal/launch.rs` — 2443-2655: TerminalStart/IdentityBroker, random execution ID and identity_launch/route/action.

Existing test allocation: common; protocol; identity_wire; target; identity; launch. Source grouping: 2656-2752 fixture closure; name/id/request/base64/frame goldens2753-3316; identity wire3317-3553; target/argv3554-3957; native identity3958-4554; stream/start/launch4555-4826.

**rust/soda-host/src/tcodex.rs** (3263 lines on the inspected branch).

- `lib/host/src/terminal/codex/mod.rs` — 685-838: unit-show/role-ID/bounded wait and Service native systemctl/systemd_run methods; extends the existing terminal Service.
- `lib/host/src/terminal/codex/run.rs` — 24-219: constants and whole FactoryRun fields/spec/decode/validator.
- `lib/host/src/terminal/codex/paths.rs` — 220-316,429-477: exact role/checkout/run/user paths and lease binding.
- `lib/host/src/terminal/codex/commands.rs` — 317-428,478-684: supervisor/retire/setup/install/start-gate scripts and reserve/output/takeover/export argv.
- `lib/host/src/terminal/codex/reserve.rs` — 839-1048: reserve/setup/stage/stage_host and native unit reservation.
- `lib/host/src/terminal/codex/start.rs` — 1049-1185: start/stage_file/wait/validate; same unit-incarnation gates.
- `lib/host/src/terminal/codex/stop.rs` — 1186-1364: stop/inactive/retire/PID/container/capture/finish/unbound/live.
- `lib/host/src/terminal/codex/artifacts.rs` — 1365-1665: output binding/window, export bundle, takeover copy, factory identity operation and output size.

Existing test allocation: common; wire; reserve; lifecycle; artifacts. Source grouping: 1666-1825 fixture closure; domain/path/script/binding/unit1826-2172; reservation2173-2470; lifecycle2471-2910; artifacts/callbacks2911-3263.

**rust/soda-host/src/muse.rs** (5038 lines on the inspected branch).

- `lib/host/src/muse/mod.rs` — 450-542,959-997: existing peer/caller/nested/execution records, MuseHooks/MuseRuntime and fixed podman/guest helpers.
- `lib/host/src/muse/wire.rs` — 28-316: whole NestedRegistration/LaunchRequest/LaunchControl/LaunchExit and validators/specs/codecs.
- `lib/host/src/muse/arguments.rs` — 317-449: existing provider argument validation and connection selection/authorization.
- `lib/host/src/muse/caller.rs` — 550-636,998-1220: cgroup/UID/account/mode/registration and project/kernel/registered caller resolution.
- `lib/host/src/muse/nested.rs` — 637-693,1221-1459: child PID/readonly mount and nested registration/authority/child/namespace/account.
- `lib/host/src/muse/program.rs` — 694-739,771-958,1460-1519: ELF/arch/signals/environment/command/unit argv and pinned guest binary proof.
- `lib/host/src/muse/execution.rs` — 740-770,1520-1697,1926-2083: delivery/root validity, prepare/reserve/deliver/control/stop/validate/await/Muse dispatch.
- `lib/host/src/muse/config.rs` — 1698-1925,2831-2865: admitted files/config/auth mount/copy_nested_config/decode_config_view.
- `lib/host/src/muse/cleanup.rs` — 2084-2205: cleanup_execution_state/invoke_state/retire_mount/retire_execution_files.
- `lib/host/src/muse/socket.rs` — 2242-2434,2458-2497: existing listener directory, pidfd/rights/request/descriptors and split-json.
- `lib/host/src/muse/launch.rs` — 2206-2241,2435-2457,2498-2830: resize/state-container/command-exit and MuseLaunch serve/shell/control/spawn; same process.

Existing test allocation: common; wire; caller; execution; program; socket. Source grouping: 2866-3067 fixture closure; wire/selection3068-3382; native caller vectors3383-3608 and3716-3789; program commands3609-3715; custody/config/retirement3790-4382; config/control/binary4383-4669; socket/control-loop/shutdown4670-5020; environment5021-5038. Shared actual fixtures remain descendant-test accessible.

**project.rs extension** (2177 lines on that branch): `lib/host/src/project/lifecycle.rs`, `lib/host/src/project/confirmation.rs`, `lib/host/src/project/lifecycle_tests.rs`, `lib/host/src/project/confirmation_tests.rs`. Evidence: 86-113,710-780,811-880,888-912: Lifecycle/LifecycleState, exact native unit read/start/stop and confirmation.; 881-887,913-974: address/profile/OS/key pure confirmations.; 1250-1646: existing main project fixtures/tests shifted by pending source.; 1647-2025: unit-state/lifecycle cases.; 2026-2177: address/profile/OS/key confirmations.

Keep branch smoke-test coverage at `lib/host/tests/daemon.rs`, using actual
library imports; move its current StubBackend into test support. Preserve the
Go stream/admission/shutdown regressions in `lib/host/tests/terminal_transport.rs`
against the actual Rust transport. The opt-in native terminal Go test can retain
its marker/probe journey, but its helper currently constructs Go Daemon/Native/
Service (`terminal_native_test.go:115-142`); rebind it to the Rust binary and
real native binding before executor deletion. Client-only candidate/export/
prepare/Tailnet refusal tests stay with the Go client, with server cases removed.
Smoke tests or stub backends do not prove installed host cutover.

Retire `internal/platform/platform.go` with its Go native callers: all actual
imports belong to the removed daemon, project lifecycle or Tailnet Control.
Keep its fixed installed paths at the corresponding Rust configuration/unit
admission owners, without an unused Go constants package. Also retire
`tests/build/{project_os_observation,project_keys,terminal}_test.go`: these are
source-existence guards for superseded Go/Python subjects, not behavior tests.
The Rust OS/key/terminal owners retain the actual behavior coverage; do not port
guards that require obsolete files to remain.

### Release cutover integration

The four release crates are retained source owners, but the current Rust CLI
is not a connected replacement producer. `build_cli.rs:305-318` stops at
unimplemented isolated-worker dispatch, and `349-353` stops at unimplemented
worker image execution. The image crate accepts a production factory from its
caller (`build.rs:185-198`); the tools manifest currently has no build/image/
deliver dependencies. Connect these existing crates and CLI phases directly,
preserving worker identity, source admission, exact outputs and cleanup. Do not
delete Go merely because the standalone Rust oracle suites pass.

The candidate verification successor is concrete:
`lib/soda-release-tools/src/candidate_check.rs` and
`src/bin/soda-candidate-check.rs`, in the existing package, call
`soda_release_deliver::check::check_candidate`. Declare that binary and the real
deliver dependency in its manifest. Keep `--candidate`, `--arch`,
`--soda-revision`, `--forgejo-revision`, current parsing/refusal order, quiet
success and one-line error/exit behavior. Repoint `scripts/check-native.sh:29`
to this binary; retain delivered archive verification and exact Soda/Fountain
revision binding. Its existing CLI suite gains `soda_candidate_check.rs` rather
than another test harness or package. The Go command is absent from the target.

The new concern splits expose remaining duplication rather than certify it
removed. Build and deliver already own identities, payload/trust admission and
OCI reads; image mirrors several of those models. Compare actual binding and
byte emission before reusing the existing owner: image currently represents
`upgrade_from` as a vector while deliver preserves optional-vector `null`
semantics (`image/model.rs:416,520-525`, `deliver/payload.rs:73-75`). Do not treat
matching type names as equivalent APIs. The repeated host sealing sequence in
`image/build.rs:711-771,1059-1121` can have one implementation with the existing
phase callback. Consolidate matching settings and HTTP helpers at their real
owners; no facade crate, FFI, RPC or additional service follows from this work.

### Python cutover closure

The target has no Soda-authored Python program, including executable strings
inside Go tests or probes. Third-party package implementation languages and
historical prose are separate from this authored-source inventory; this plan
does not invent a replacement for the existing Compose tool. The source still
contains Python execution. These are pending conversions, not completed ports:

| Current source / caller | Target owner and required cutover |
| --- | --- |
| `internal/acceptance/personal_git.go:33-66,164-173,238-244` | Keep the Go probe's prepare/exercise/unlock orchestration. Move the existing user-scoped key/agent payload into `tools/acceptance/src/personal_git.rs`, dispatched through the existing ephemeral `soda-acceptance-remote` payload. Preserve the SSH login identity, protected passphrase input, public-key-only output, live-agent refusal and encrypted-key lifetime. Remove the Python template and `python3 -` transport; do not add a service or credential broker. |
| `internal/acceptance/lifecycle_state.go:27-34,150,199` | Keep the Go snapshot/comparison orchestration. Execute the existing Rust `project_state` payload through `soda-acceptance-remote project-state`. The embedded SQLite host query is obsolete against the canonical PostgreSQL store; reconcile the host observation's actual records and read-only caller before implementing its replacement. Do not port that stale query or introduce a second database. |
| `tools/soda-installed-probes/{cockpit_account,project_state}_test.go`, `tests/build/u08_state_test.go` | Exercise the existing Rust cockpit/project-state implementation and remote binary under the actual fixtures. Remove Python import drivers and assertions against deleted `.py` source. Keep the PAM denial/error distinction, root/native admission, private-file content refusal and snapshot bounds. |
| `project-os/rootfs/usr/libexec/soda/{project-account,project-factory-roles}` | Retain these two installed executable identities as compiled outputs of the existing project-terminal Cargo package. Concrete entrypoints are `cmd/soda-project-terminal/src/bin/project-account.rs` and `project-factory-roles.rs`; a private library root shares the real project-local modules. Account provisioning and factory-role duties stay separate from terminal dispatch. Do not track generated binaries or Python wrappers in rootfs. |
| `tests/build/{project_account,project_factory_roles}_test.go` | Replace embedded Python `SourceFileLoader`/mock drivers with tests exercising the real Rust helper functions and compiled entrypoints. Preserve established file, lock, account, digest, receipt and process effects; retire redundant Go wrappers only after their actual assertions have moved to the native owner. |
| `tests/build/{source_checks,workload_probe}_test.go`, `tests/build/helpers.go:142-160` | Replace Python PATH-tool doubles with the existing Go subprocess-test pattern or inert shell fixtures where sufficient. Keep command order, environment, status and no-replayed-mutation observations. Remove `Python3` after its callers move. |
| `tests/build/project_foundation_test.go:36` | Replace the Python `os.defpath` observation with the real admitted helper environment/argv contract; do not retain an interpreter merely for this assertion. |
| `scripts/{check-source,check-native}.sh`, `tests/build/source_checks_test.go` | Remove obsolete unittest invocations and update the actual source/native check sequence. `check-native.sh` invokes the Rust `soda-candidate-check` binary with the same candidate, architecture and exact revisions; it must keep verification enabled. |
| `scripts/{check-ruff-format,check-ruff,check-py-complexity,ruff-env}.sh`, `pyproject.toml`, `requirements-ruff.txt`, `docs/development/python.md` | Retire Soda's Python-only tooling and guide with their final authored callers. Remove the matching package scripts and documentation links in that cutover. Shared source gates, root manifests and guides remain and are updated at their existing owners. |

The extensionless helper destinations are installation outputs, not additional
source packages. The two replacement entrypoints share one existing Cargo
package; the ephemeral probe uses one existing acceptance payload. This adds
no daemon, socket, container or sidecar. Python cutover verification must check
retained executable bodies and real test invocations, not only filename suffixes.

## File granularity and unresolved extraction seams

The 400-line threshold is an inspection trigger. Avoid a file per helper,
record, method or test assertion. Keep small private functions with the real
filesystem, command or wire operation they serve; reuse existing view modules
and mature primitives. The refined Muse maintenance proposal combines pinned
tool/path/native-error handling into filesystem ownership and quote diagnostics
with config wire formatting. It does not create utility crates or adapters.

Go concern files stay in the same package unless the caller review above earns
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

The extensionless Project factory-role program is part of the source review.
Its Rust successor's real compiled bytes must be staged and hashed:
`tests/build/project_factory_roles_test.go` still imports the Python program,
and `internal/factory/control/st15_demo_native_test.go:321-325,371-381`
copies/hashes that source. Repoint both to the compiled native helper, preserving
their actual observations rather than keeping a dead wrapper for the checks.
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
| `factory-os/` | `system/factory/` |
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

The tracked root files `installer` and `soda-candidate` are ELF build outputs. Their proposed disposition is removal from the tracked source tree; current tools build into ignored `.artifacts/`. The `rust/soda-host/` files are retained at `lib/host/` for pending PR26. Detailed decomposition entries for cutover-deleted files stay as review history; the proposed tree below omits them. This document deletes or executes none of these files.

## Complete proposed tree

Every tracked source has a destination or an explicit disposition. The tree applies the package consolidations above and the decided language policy: it shows the post-cutover target, so pre-port Go/Python implementations are omitted even where their cutover commit is still pending. It retains the host crate at `lib/host/`; the decomposition section's host entries are now the retained plan, not a conditional alternative. Existing small files keep their names unless package consolidation requires a namespace. The three release-assets integration-test namespaces retain distinct support modules.

The existing `sodaspaces-factory-view.ts` and `sodaspaces-terminal-view.ts` receive
view concerns from their larger state owners; their combined size still needs
implementation review. Providers and assets merge into their existing owners.
Shared HTTP/test support and native assertion destinations also intentionally
consolidate existing concerns. Each shared leaf has one implementation owner;
duplicate file leaves or competing Rust module roots are not intended.

```text
sodaos/
├── .agents/
│   └── plans/
│       └── 2026-09-16-remove-rpm-pinning.md
├── .githooks/
│   └── pre-commit
├── assets/
│   ├── animated-wave-background/
│   │   ├── animated-waves.svg
│   │   ├── index.html
│   │   └── styles.css
│   ├── branding/
│   │   ├── cockpit/
│   │   │   ├── provenance/
│   │   │   │   ├── patternfly/
│   │   │   │   │   ├── _fonts.scss
│   │   │   │   │   └── patternfly-6-cockpit.scss
│   │   │   │   ├── LICENSES.txt
│   │   │   │   ├── README.md
│   │   │   │   ├── _global-variables.scss
│   │   │   │   ├── cockpit-dark-theme.ts
│   │   │   │   ├── patternfly-MIT.txt
│   │   │   │   ├── patternfly-react-MIT.txt
│   │   │   │   └── redhat-fonts-OFL.txt
│   │   │   ├── README.md
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── branding.css
│   │   │   ├── favicon-16.png
│   │   │   ├── favicon-32.png
│   │   │   ├── favicon-48.png
│   │   │   ├── favicon.ico
│   │   │   ├── login-background-dark.svg
│   │   │   ├── login-background-light.svg
│   │   │   ├── preview.css
│   │   │   ├── preview.html
│   │   │   ├── soda.css
│   │   │   └── theme.css
│   │   ├── fonts/
│   │   │   ├── barlow/
│   │   │   │   ├── LICENSE
│   │   │   │   ├── barlow-latin-400-normal.woff2
│   │   │   │   └── barlow-latin-600-normal.woff2
│   │   │   ├── barlow-condensed/
│   │   │   │   ├── LICENSE
│   │   │   │   └── barlow-condensed-latin-800-normal.woff2
│   │   │   ├── fraunces/
│   │   │   │   ├── LICENSE
│   │   │   │   └── fraunces-latin-wght-normal.woff2
│   │   │   ├── ibm-plex-mono/
│   │   │   │   ├── LICENSE
│   │   │   │   ├── ibm-plex-mono-latin-400-normal.woff2
│   │   │   │   └── ibm-plex-mono-latin-500-normal.woff2
│   │   │   ├── README.md
│   │   │   ├── fonts.css
│   │   │   └── sources.json
│   │   ├── forgejo/
│   │   │   ├── backgrounds/
│   │   │   │   ├── masters/
│   │   │   │   │   ├── desktop-day.png
│   │   │   │   │   ├── desktop-night.png
│   │   │   │   │   ├── mobile-day.png
│   │   │   │   │   ├── mobile-night.png
│   │   │   │   │   ├── tablet-day.png
│   │   │   │   │   └── tablet-night.png
│   │   │   │   ├── README.md
│   │   │   │   ├── manifest.json
│   │   │   │   ├── subway-desktop-day.webp
│   │   │   │   ├── subway-desktop-night.webp
│   │   │   │   ├── subway-mobile-day.webp
│   │   │   │   ├── subway-mobile-night.webp
│   │   │   │   ├── subway-tablet-day.webp
│   │   │   │   └── subway-tablet-night.webp
│   │   │   ├── css/
│   │   │   │   ├── soda-controls.css
│   │   │   │   ├── theme-soda-auto.css
│   │   │   │   ├── theme-soda-dark.css
│   │   │   │   └── theme-soda-light.css
│   │   │   ├── login-station/
│   │   │   │   ├── masters/
│   │   │   │   │   └── approaching-train.png
│   │   │   │   ├── README.md
│   │   │   │   ├── approaching-train.webp
│   │   │   │   └── manifest.json
│   │   │   ├── README.md
│   │   │   ├── account-details.css
│   │   │   ├── account-settings.css
│   │   │   ├── admin-details.css
│   │   │   ├── admin-monitoring.css
│   │   │   ├── admin.css
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── auth.css
│   │   │   ├── code-search.css
│   │   │   ├── components-buttons.css
│   │   │   ├── components-empty.css
│   │   │   ├── components-forms.css
│   │   │   ├── components-guest.css
│   │   │   ├── components-intro.css
│   │   │   ├── components-list.css
│   │   │   ├── components-navigation.css
│   │   │   ├── components-repository-toolbar.css
│   │   │   ├── components-settings.css
│   │   │   ├── components-toolbar-layout.css
│   │   │   ├── components-toolbar.css
│   │   │   ├── components.css
│   │   │   ├── configuration.css
│   │   │   ├── create.css
│   │   │   ├── dashboard.css
│   │   │   ├── explore.css
│   │   │   ├── favicon-16.png
│   │   │   ├── favicon.png
│   │   │   ├── federated-auth.css
│   │   │   ├── forgejo-events.d.ts
│   │   │   ├── forgejo-setup.css
│   │   │   ├── form-pages.css
│   │   │   ├── home.css
│   │   │   ├── insights.css
│   │   │   ├── lit.ts
│   │   │   ├── login-theme.ts
│   │   │   ├── login.css
│   │   │   ├── logo.png
│   │   │   ├── manifest.tsv
│   │   │   ├── milestones.css
│   │   │   ├── moderation.css
│   │   │   ├── notification-preview.css
│   │   │   ├── notification-preview.ts
│   │   │   ├── notifications.css
│   │   │   ├── onboarding.css
│   │   │   ├── org-create.css
│   │   │   ├── org-details.css
│   │   │   ├── org-home.css
│   │   │   ├── organization.css
│   │   │   ├── packages.css
│   │   │   ├── personal-settings.ts
│   │   │   ├── profiles.css
│   │   │   ├── projects.css
│   │   │   ├── quota.css
│   │   │   ├── repository-actions.ts
│   │   │   ├── repository-code-actions.css
│   │   │   ├── repository-code-browser.css
│   │   │   ├── repository-code-editing.css
│   │   │   ├── repository-code.css
│   │   │   ├── repository-content.css
│   │   │   ├── repository-issues.css
│   │   │   ├── repository-settings-details.css
│   │   │   ├── repository-switcher.css
│   │   │   ├── repository-switcher.ts
│   │   │   ├── repository.css
│   │   │   ├── runners.css
│   │   │   ├── status.css
│   │   │   ├── theme-preview.html
│   │   │   └── webhooks.css
│   │   ├── host/
│   │   │   ├── README.md
│   │   │   └── os-release
│   │   ├── icons/
│   │   │   ├── hicolor/
│   │   │   │   ├── 128x128/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 16x16/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 24x24/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 256x256/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 32x32/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 48x48/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 512x512/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   └── 64x64/
│   │   │   │       └── apps/
│   │   │   │           └── soda-os.png
│   │   │   └── octicons/
│   │   │       ├── LICENSE
│   │   │       ├── README.md
│   │   │       ├── repo-19.14.0.svg
│   │   │       └── terminal-19.14.0.svg
│   │   ├── installer/
│   │   │   ├── manifest.tsv
│   │   │   ├── soda-logo-black.png
│   │   │   ├── soda-logo-horizontal-dark.png
│   │   │   ├── soda-logo-horizontal.png
│   │   │   ├── soda-logo-navy.png
│   │   │   ├── soda-logo-white.png
│   │   │   ├── soda-symbol-black.png
│   │   │   ├── soda-symbol-navy.png
│   │   │   ├── soda-symbol-white.png
│   │   │   ├── soda-symbol.png
│   │   │   └── soda.css
│   │   ├── source/
│   │   │   ├── soda-logo-black.svg
│   │   │   ├── soda-logo-horizontal-dark.svg
│   │   │   ├── soda-logo-horizontal.svg
│   │   │   ├── soda-logo-navy.svg
│   │   │   ├── soda-logo-white.svg
│   │   │   ├── soda-symbol-black.svg
│   │   │   ├── soda-symbol-brutalist-dark.svg
│   │   │   ├── soda-symbol-brutalist.svg
│   │   │   ├── soda-symbol-navy.svg
│   │   │   ├── soda-symbol-white.svg
│   │   │   └── soda-symbol.svg
│   │   ├── terminal/
│   │   │   ├── README.md
│   │   │   ├── fastfetch.jsonc
│   │   │   ├── motd.txt
│   │   │   └── sodaos.txt
│   │   ├── theme/
│   │   │   ├── README.md
│   │   │   └── palette.css
│   │   ├── web/
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── favicon-16.png
│   │   │   └── favicon-32.png
│   │   └── soda-os-logo-concept-v3.png
│   └── README.md
├── cmd/
│   ├── soda-activate/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── activation.rs
│   │   │   │   ├── cli.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── origin.rs
│   │   │   ├── activation.rs
│   │   │   ├── cli.rs
│   │   │   ├── forgejo_env.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   ├── soda-console-welcome/
│   │   ├── src/
│   │   │   ├── config.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   ├── tests.rs
│   │   │   └── welcome.rs
│   │   └── Cargo.toml
│   ├── soda-dashboard/
│   │   ├── extension.go
│   │   ├── extension_test.go
│   │   ├── main.go
│   │   ├── operator.go
│   │   ├── operator_linux.go
│   │   ├── operator_linux_test.go
│   │   ├── operator_other.go
│   │   ├── operator_test.go
│   │   └── postgres_fixture_test.go
│   ├── soda-extension/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── soda-factory/
│   │   ├── src/
│   │   │   ├── main.rs
│   │   │   └── operator_tests.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-domain/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── cli.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── domain.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   └── mod.rs
│   │   │   ├── cli.rs
│   │   │   ├── config.rs
│   │   │   ├── domain.rs
│   │   │   ├── main.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-migrate/
│   │   ├── src/
│   │   │   └── main.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-tailnet/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── soda-host/
│   │   └── main.rs
│   ├── soda-identity/
│   │   ├── src/
│   │   │   ├── providers/
│   │   │   │   ├── codex/
│   │   │   │   │   ├── config.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── protocol.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── muse/
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── sha256.rs
│   │   │   │   └── types.rs
│   │   │   ├── acquisition.rs
│   │   │   ├── control.rs
│   │   │   ├── crypto.rs
│   │   │   ├── enrollment.rs
│   │   │   ├── grants.rs
│   │   │   ├── http.rs
│   │   │   ├── http_routes.rs
│   │   │   ├── http_tests.rs
│   │   │   ├── http_wire.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── pg.rs
│   │   │   ├── pg_dsn.rs
│   │   │   ├── pg_query.rs
│   │   │   ├── pg_tests.rs
│   │   │   ├── registration.rs
│   │   │   ├── retirement.rs
│   │   │   ├── runtime.rs
│   │   │   ├── schema.rs
│   │   │   ├── service.rs
│   │   │   ├── store.rs
│   │   │   ├── store_connections.rs
│   │   │   ├── store_events.rs
│   │   │   ├── store_executions.rs
│   │   │   ├── store_grants.rs
│   │   │   ├── store_leases.rs
│   │   │   ├── store_schema.rs
│   │   │   ├── store_tests.rs
│   │   │   ├── strict.rs
│   │   │   ├── strict_tests.rs
│   │   │   ├── wire.rs
│   │   │   ├── wire_errors.rs
│   │   │   ├── wire_execution.rs
│   │   │   ├── wire_grants.rs
│   │   │   ├── wire_scalars.rs
│   │   │   ├── wire_tests.rs
│   │   │   └── wire_time.rs
│   │   ├── tests/
│   │   │   ├── common/
│   │   │   │   └── mod.rs
│   │   │   ├── broker.rs
│   │   │   ├── enrollment.rs
│   │   │   └── http.rs
│   │   └── Cargo.toml
│   ├── soda-identity-compose/
│   │   ├── src/
│   │   │   ├── compose.rs
│   │   │   ├── compose_tests.rs
│   │   │   ├── launch_json.rs
│   │   │   ├── launch_wire.rs
│   │   │   ├── main.rs
│   │   │   ├── options.rs
│   │   │   └── registration.rs
│   │   └── Cargo.toml
│   ├── soda-image-import/
│   │   ├── src/
│   │   │   ├── oci/
│   │   │   │   ├── inspection.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── metadata.rs
│   │   │   │   └── mod.rs
│   │   │   ├── tests/
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── import.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── oci.rs
│   │   │   │   ├── payload.rs
│   │   │   │   └── primitives.rs
│   │   │   ├── context.rs
│   │   │   ├── import.rs
│   │   │   ├── json_binding.rs
│   │   │   ├── main.rs
│   │   │   ├── payload.rs
│   │   │   ├── platform.rs
│   │   │   └── sha256.rs
│   │   └── Cargo.toml
│   ├── soda-install/
│   │   ├── src/
│   │   │   ├── candidate/
│   │   │   │   └── tests.rs
│   │   │   ├── console/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── network.rs
│   │   │   │   ├── terminal.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── deliver/
│   │   │   │   └── tests.rs
│   │   │   ├── disks/
│   │   │   │   └── tests.rs
│   │   │   ├── enroll/
│   │   │   │   ├── keys/
│   │   │   │   │   ├── authorized_keys.rs
│   │   │   │   │   ├── directory.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── arm.rs
│   │   │   │   ├── arm_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── native_config.rs
│   │   │   │   ├── peer.rs
│   │   │   │   ├── receive.rs
│   │   │   │   ├── selinux.rs
│   │   │   │   ├── serve.rs
│   │   │   │   ├── session.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── execute/
│   │   │   │   └── tests.rs
│   │   │   ├── inputs/
│   │   │   │   └── tests.rs
│   │   │   ├── netip/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── address.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── prefix.rs
│   │   │   │   ├── address.rs
│   │   │   │   ├── address_format.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── prefix.rs
│   │   │   ├── oci/
│   │   │   │   ├── inspection.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── metadata.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── setup/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── access.rs
│   │   │   │   │   ├── configure.rs
│   │   │   │   │   ├── fixtures.rs
│   │   │   │   │   └── mod.rs
│   │   │   │   ├── access.rs
│   │   │   │   ├── address.rs
│   │   │   │   ├── configure.rs
│   │   │   │   ├── local_ca.rs
│   │   │   │   └── mod.rs
│   │   │   ├── sshkey/
│   │   │   │   ├── authorized_keys.rs
│   │   │   │   ├── base64.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── wire.rs
│   │   │   ├── urlx/
│   │   │   │   └── tests.rs
│   │   │   ├── wizard/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── review.rs
│   │   │   │   ├── steps.rs
│   │   │   │   └── tests.rs
│   │   │   ├── x509/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── algorithms.rs
│   │   │   │   │   ├── extensions.rs
│   │   │   │   │   ├── fixtures.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── public_key.rs
│   │   │   │   │   ├── structure.rs
│   │   │   │   │   ├── time.rs
│   │   │   │   │   └── verify.rs
│   │   │   │   ├── algorithms.rs
│   │   │   │   ├── certificate.rs
│   │   │   │   ├── der.rs
│   │   │   │   ├── extensions.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── name_constraints.rs
│   │   │   │   ├── names.rs
│   │   │   │   ├── public_key.rs
│   │   │   │   ├── time.rs
│   │   │   │   ├── types.rs
│   │   │   │   └── verify.rs
│   │   │   ├── buildx.rs
│   │   │   ├── candidate.rs
│   │   │   ├── command.rs
│   │   │   ├── deliver.rs
│   │   │   ├── disks.rs
│   │   │   ├── errors.rs
│   │   │   ├── execute.rs
│   │   │   ├── fmtx.rs
│   │   │   ├── hostadmit.rs
│   │   │   ├── inputs.rs
│   │   │   ├── jsongo.rs
│   │   │   ├── main.rs
│   │   │   ├── pathx.rs
│   │   │   ├── pemx.rs
│   │   │   ├── run.rs
│   │   │   ├── signal.rs
│   │   │   └── urlx.rs
│   │   └── Cargo.toml
│   ├── soda-muse/
│   │   ├── src/
│   │   │   ├── account.rs
│   │   │   ├── config.rs
│   │   │   ├── config_tests.rs
│   │   │   ├── execution.rs
│   │   │   ├── launch.rs
│   │   │   ├── launch_json.rs
│   │   │   ├── launch_tests.rs
│   │   │   ├── launch_wire.rs
│   │   │   ├── main.rs
│   │   │   ├── paths.rs
│   │   │   ├── runtime.rs
│   │   │   ├── shell.rs
│   │   │   └── shell_tests.rs
│   │   └── Cargo.toml
│   ├── soda-muse-maintain/
│   │   ├── src/
│   │   │   ├── archive.rs
│   │   │   ├── archive_tests.rs
│   │   │   ├── command.rs
│   │   │   ├── config.rs
│   │   │   ├── config_tests.rs
│   │   │   ├── config_validation.rs
│   │   │   ├── config_wire.rs
│   │   │   ├── filesystem.rs
│   │   │   ├── filesystem_tests.rs
│   │   │   ├── interface.rs
│   │   │   ├── interface_admission.rs
│   │   │   ├── interface_tests.rs
│   │   │   ├── json.rs
│   │   │   ├── json_string.rs
│   │   │   ├── main.rs
│   │   │   ├── network.rs
│   │   │   ├── network_tests.rs
│   │   │   ├── options.rs
│   │   │   ├── options_tests.rs
│   │   │   ├── project.rs
│   │   │   ├── project_tests.rs
│   │   │   ├── release.rs
│   │   │   ├── release_tests.rs
│   │   │   ├── release_validation.rs
│   │   │   ├── release_wire.rs
│   │   │   ├── sha256.rs
│   │   │   ├── stage.rs
│   │   │   └── test_support.rs
│   │   └── Cargo.toml
│   ├── soda-pg-maintenance/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-pg-backup.rs
│   │   │   │   ├── soda-pg-init-roles.rs
│   │   │   │   └── soda-pg-restore.rs
│   │   │   └── lib.rs
│   │   └── Cargo.toml
│   ├── soda-project-terminal/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── project-account.rs
│   │   │   │   └── project-factory-roles.rs
│   │   │   ├── factory_roles/
│   │   │   │   ├── accounts.rs
│   │   │   │   ├── execution.rs
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── records.rs
│   │   │   │   └── tests.rs
│   │   │   ├── account.rs
│   │   │   ├── b64.rs
│   │   │   ├── broker.rs
│   │   │   ├── broker_tests.rs
│   │   │   ├── cgroup.rs
│   │   │   ├── fs.rs
│   │   │   ├── fs_tests.rs
│   │   │   ├── key_lines.rs
│   │   │   ├── key_request.rs
│   │   │   ├── keys.rs
│   │   │   ├── keys_tests.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── project_account.rs
│   │   │   ├── proto.rs
│   │   │   ├── pty.rs
│   │   │   ├── pty_io.rs
│   │   │   ├── pty_process.rs
│   │   │   ├── pty_relay.rs
│   │   │   ├── pty_tests.rs
│   │   │   ├── pyemit.rs
│   │   │   ├── sha.rs
│   │   │   ├── socket.rs
│   │   │   ├── subscription_cgroup.rs
│   │   │   ├── subscription_credentials.rs
│   │   │   ├── subscription_prepare.rs
│   │   │   ├── subscription_profile.rs
│   │   │   ├── subscription_retire.rs
│   │   │   ├── subscription_start.rs
│   │   │   ├── subscription_wire.rs
│   │   │   ├── svc.rs
│   │   │   ├── svc_tests.rs
│   │   │   ├── sys.rs
│   │   │   ├── term.rs
│   │   │   ├── term_attach.rs
│   │   │   ├── term_binding.rs
│   │   │   ├── term_binding_tests.rs
│   │   │   ├── term_collect.rs
│   │   │   ├── term_create.rs
│   │   │   ├── term_native_tests.rs
│   │   │   ├── term_paths.rs
│   │   │   ├── term_prepare.rs
│   │   │   ├── term_protocol_tests.rs
│   │   │   ├── term_status.rs
│   │   │   ├── timex.rs
│   │   │   ├── timex_tests.rs
│   │   │   └── tmux.rs
│   │   ├── tests/
│   │   │   └── cli.rs
│   │   └── Cargo.toml
│   ├── soda-setup/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── admission.rs
│   │   │   │   ├── encoding.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── postgres.rs
│   │   │   │   └── setup.rs
│   │   │   ├── cli.rs
│   │   │   ├── config.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── json.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   ├── secrets.rs
│   │   │   ├── setup.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   └── soda-tailnet/
│       ├── command.go
│       ├── command_test.go
│       └── main.go
├── docs/
│   ├── architecture/
│   │   ├── factory-interfaces.md
│   │   ├── networking.md
│   │   ├── overview.md
│   │   ├── release.md
│   │   └── trust.md
│   ├── design/
│   │   ├── spaces/
│   │   │   ├── README.md
│   │   │   ├── drawer-compact.svg
│   │   │   ├── drawer-sessions.svg
│   │   │   ├── drawer-work.svg
│   │   │   ├── focus.svg
│   │   │   ├── index.html
│   │   │   ├── mobile-states.svg
│   │   │   ├── preview.css
│   │   │   ├── preview.ts
│   │   │   ├── render-sheets.ts
│   │   │   ├── review.ts
│   │   │   ├── sheets.css
│   │   │   ├── split.svg
│   │   │   ├── tsconfig.json
│   │   │   └── tsconfig.tools.json
│   │   ├── avatars.md
│   │   ├── branding.md
│   │   ├── console-welcome.md
│   │   ├── forms.md
│   │   ├── screenshot-capture.md
│   │   └── spaces-ux.md
│   ├── development/
│   │   ├── README.md
│   │   ├── cockpit.md
│   │   ├── factory-implementation-plan.md
│   │   ├── forgejo-extensions-plan.md
│   │   ├── go-packages.md
│   │   ├── go.md
│   │   ├── ideal-filetree-plan.md
│   │   ├── lit.md
│   │   ├── native-support.md
│   │   ├── release.md
│   │   ├── testing.md
│   │   └── typescript.md
│   ├── factory/
│   │   └── decision-gate.md
│   ├── guides/
│   │   ├── develop.md
│   │   ├── installation.md
│   │   ├── local-testing.md
│   │   ├── media.md
│   │   ├── operator-setup.md
│   │   ├── project-clis.md
│   │   └── project-services.md
│   ├── operator/
│   │   └── enroll-key.md
│   ├── ops/
│   │   ├── gc-policy.md
│   │   └── retention.md
│   ├── product/
│   │   ├── overview.md
│   │   ├── projects.md
│   │   ├── scope.md
│   │   └── spaces.md
│   ├── public/
│   │   ├── 10-Start-here/
│   │   │   ├── 10-index.md
│   │   │   └── 20-product-model.md
│   │   ├── 20-Deploy/
│   │   │   ├── 05-verify-downloads.md
│   │   │   ├── 10-deploy-to-cloud.md
│   │   │   ├── 20-install-on-premises.md
│   │   │   ├── 25-operator-setup.md
│   │   │   └── 30-first-connection.md
│   │   ├── 30-Use-Soda/
│   │   │   ├── 05-dashboard.md
│   │   │   ├── 10-cockpit.md
│   │   │   ├── 15-software-factory.md
│   │   │   ├── 20-projects-and-workspaces.md
│   │   │   ├── 30-forgejo.md
│   │   │   ├── 35-collaboration.md
│   │   │   ├── 40-tailscale.md
│   │   │   ├── 50-ci-runners.md
│   │   │   └── 60-updates-and-fallback.md
│   │   ├── 40-Develop/
│   │   │   ├── 10-connect-and-develop.md
│   │   │   ├── 20-shared-tools-and-files.md
│   │   │   └── 30-project-services.md
│   │   ├── 50-Operate/
│   │   │   ├── 10-people-and-access.md
│   │   │   ├── 20-administration.md
│   │   │   ├── 30-backups-and-restoration.md
│   │   │   └── 40-data-safety-and-removal.md
│   │   └── README.md
│   ├── reference/
│   │   ├── api.md
│   │   ├── configuration.md
│   │   ├── credentials.md
│   │   ├── factory.md
│   │   ├── forgejo.md
│   │   ├── project-os.md
│   │   └── terminal.md
│   ├── research/
│   │   ├── factory-capability-map.md
│   │   ├── host-strategy.md
│   │   ├── licensing.md
│   │   ├── notices.md
│   │   ├── onedev.md
│   │   └── predecessor-reuse.md
│   └── README.md
├── frontend/
│   ├── forgejo/
│   │   ├── locales/
│   │   │   ├── README.md
│   │   │   └── en-US.ini
│   │   ├── templates/
│   │   │   ├── admin/
│   │   │   │   ├── applications/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── oauth2_edit.tmpl
│   │   │   │   ├── auth/
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── edit_ldap.tmpl
│   │   │   │   │   ├── edit_oauth.tmpl
│   │   │   │   │   ├── edit_pam.tmpl
│   │   │   │   │   ├── edit_smtp.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── emails/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── repo/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── runners/
│   │   │   │   │   └── create.tmpl
│   │   │   │   ├── user/
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── config.tmpl
│   │   │   │   ├── cron.tmpl
│   │   │   │   ├── dashboard.tmpl
│   │   │   │   ├── hook_new.tmpl
│   │   │   │   ├── layout_head.tmpl
│   │   │   │   ├── notice.tmpl
│   │   │   │   ├── queue.tmpl
│   │   │   │   ├── queue_manage.tmpl
│   │   │   │   ├── self_check.tmpl
│   │   │   │   └── stacktrace.tmpl
│   │   │   ├── custom/
│   │   │   │   ├── soda/
│   │   │   │   │   ├── empty_content.tmpl
│   │   │   │   │   ├── guest_theme.tmpl
│   │   │   │   │   ├── milestone_row.tmpl
│   │   │   │   │   ├── notification_preview.tmpl
│   │   │   │   │   ├── page_intro.tmpl
│   │   │   │   │   ├── profile_block_dialog.tmpl
│   │   │   │   │   ├── profile_repositories.tmpl
│   │   │   │   │   └── theme_toggle.tmpl
│   │   │   │   ├── explore_empty.tmpl
│   │   │   │   ├── explore_navbar.tmpl
│   │   │   │   ├── extra_links.tmpl
│   │   │   │   ├── footer.tmpl
│   │   │   │   └── header.tmpl
│   │   │   ├── explore/
│   │   │   │   ├── code.tmpl
│   │   │   │   ├── repos.tmpl
│   │   │   │   └── users.tmpl
│   │   │   ├── moderation/
│   │   │   │   └── new_abuse_report.tmpl
│   │   │   ├── org/
│   │   │   │   ├── member/
│   │   │   │   │   └── members.tmpl
│   │   │   │   ├── projects/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── hook_new.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_edit.tmpl
│   │   │   │   │   └── runners_create.tmpl
│   │   │   │   ├── team/
│   │   │   │   │   ├── invite.tmpl
│   │   │   │   │   ├── members.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   ├── repositories.tmpl
│   │   │   │   │   └── teams.tmpl
│   │   │   │   ├── create.tmpl
│   │   │   │   ├── header.tmpl
│   │   │   │   └── home.tmpl
│   │   │   ├── package/
│   │   │   │   ├── shared/
│   │   │   │   │   ├── cleanup_rules/
│   │   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   │   ├── list.tmpl
│   │   │   │   │   │   └── preview.tmpl
│   │   │   │   │   ├── cargo.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── versionlist.tmpl
│   │   │   │   ├── settings.tmpl
│   │   │   │   └── view.tmpl
│   │   │   ├── projects/
│   │   │   │   ├── list.tmpl
│   │   │   │   ├── new.tmpl
│   │   │   │   └── view.tmpl
│   │   │   ├── repo/
│   │   │   │   ├── actions/
│   │   │   │   │   ├── dispatch.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── list_inner.tmpl
│   │   │   │   │   ├── no_workflows.tmpl
│   │   │   │   │   ├── runs_list.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── branch/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── diff/
│   │   │   │   │   ├── box.tmpl
│   │   │   │   │   └── compare.tmpl
│   │   │   │   ├── editor/
│   │   │   │   │   ├── cherry_pick.tmpl
│   │   │   │   │   ├── commit_form.tmpl
│   │   │   │   │   ├── delete.tmpl
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── patch.tmpl
│   │   │   │   │   └── upload.tmpl
│   │   │   │   ├── find/
│   │   │   │   │   └── files.tmpl
│   │   │   │   ├── issue/
│   │   │   │   │   ├── choose.tmpl
│   │   │   │   │   ├── labels.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── milestone_issues.tmpl
│   │   │   │   │   ├── milestone_new.tmpl
│   │   │   │   │   ├── milestones.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── migrate/
│   │   │   │   │   ├── codebase.tmpl
│   │   │   │   │   ├── git.tmpl
│   │   │   │   │   ├── gitbucket.tmpl
│   │   │   │   │   ├── gitea.tmpl
│   │   │   │   │   ├── github.tmpl
│   │   │   │   │   ├── gitlab.tmpl
│   │   │   │   │   ├── gogs.tmpl
│   │   │   │   │   ├── migrate.tmpl
│   │   │   │   │   ├── migrating.tmpl
│   │   │   │   │   ├── onedev.tmpl
│   │   │   │   │   ├── options.tmpl
│   │   │   │   │   └── pagure.tmpl
│   │   │   │   ├── projects/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── pulls/
│   │   │   │   │   ├── commits.tmpl
│   │   │   │   │   ├── files.tmpl
│   │   │   │   │   ├── fork.tmpl
│   │   │   │   │   ├── status.tmpl
│   │   │   │   │   ├── tab_menu.tmpl
│   │   │   │   │   └── trust.tmpl
│   │   │   │   ├── release/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── units/
│   │   │   │   │   │   ├── issues.tmpl
│   │   │   │   │   │   ├── overview.tmpl
│   │   │   │   │   │   ├── pulls.tmpl
│   │   │   │   │   │   └── wiki.tmpl
│   │   │   │   │   ├── webhook/
│   │   │   │   │   │   ├── base.tmpl
│   │   │   │   │   │   ├── base_list.tmpl
│   │   │   │   │   │   ├── history.tmpl
│   │   │   │   │   │   └── new.tmpl
│   │   │   │   │   ├── actions.tmpl
│   │   │   │   │   ├── branches.tmpl
│   │   │   │   │   ├── collaboration.tmpl
│   │   │   │   │   ├── deploy_keys.tmpl
│   │   │   │   │   ├── githook_edit.tmpl
│   │   │   │   │   ├── githooks.tmpl
│   │   │   │   │   ├── layout_footer.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── lfs.tmpl
│   │   │   │   │   ├── lfs_file.tmpl
│   │   │   │   │   ├── lfs_file_find.tmpl
│   │   │   │   │   ├── lfs_locks.tmpl
│   │   │   │   │   ├── lfs_pointers.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── options.tmpl
│   │   │   │   │   ├── options_danger.tmpl
│   │   │   │   │   ├── options_federation.tmpl
│   │   │   │   │   ├── options_maintenance.tmpl
│   │   │   │   │   ├── options_mirrors.tmpl
│   │   │   │   │   ├── options_modals.tmpl
│   │   │   │   │   ├── options_repository.tmpl
│   │   │   │   │   ├── options_trust.tmpl
│   │   │   │   │   ├── protected_branch.tmpl
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_setup.tmpl
│   │   │   │   │   ├── secrets.tmpl
│   │   │   │   │   ├── tags.tmpl
│   │   │   │   │   └── units.tmpl
│   │   │   │   ├── tag/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── wiki/
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   ├── pages.tmpl
│   │   │   │   │   ├── revision.tmpl
│   │   │   │   │   ├── search.tmpl
│   │   │   │   │   ├── start.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── activity.tmpl
│   │   │   │   ├── branch_dropdown.tmpl
│   │   │   │   ├── clone_buttons.tmpl
│   │   │   │   ├── commit_header.tmpl
│   │   │   │   ├── commit_page.tmpl
│   │   │   │   ├── commits.tmpl
│   │   │   │   ├── commits_list.tmpl
│   │   │   │   ├── commits_table.tmpl
│   │   │   │   ├── create.tmpl
│   │   │   │   ├── create_basic.tmpl
│   │   │   │   ├── empty.tmpl
│   │   │   │   ├── forks.tmpl
│   │   │   │   ├── graph.tmpl
│   │   │   │   ├── header.tmpl
│   │   │   │   ├── home.tmpl
│   │   │   │   ├── release_tag_header.tmpl
│   │   │   │   ├── search.tmpl
│   │   │   │   ├── sub_menu.tmpl
│   │   │   │   ├── user_cards.tmpl
│   │   │   │   ├── view_file.tmpl
│   │   │   │   ├── view_list.tmpl
│   │   │   │   └── watchers.tmpl
│   │   │   ├── shared/
│   │   │   │   ├── actions/
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_list.tmpl
│   │   │   │   │   └── runner_setup.tmpl
│   │   │   │   ├── secrets/
│   │   │   │   │   └── add_list.tmpl
│   │   │   │   ├── user/
│   │   │   │   │   └── profile_big_avatar.tmpl
│   │   │   │   ├── variables/
│   │   │   │   │   └── variable_list.tmpl
│   │   │   │   ├── blocked_users_list.tmpl
│   │   │   │   └── quota_overview.tmpl
│   │   │   ├── status/
│   │   │   │   ├── 404.tmpl
│   │   │   │   └── 413.tmpl
│   │   │   ├── user/
│   │   │   │   ├── auth/
│   │   │   │   │   ├── activate.tmpl
│   │   │   │   │   ├── change_passwd.tmpl
│   │   │   │   │   ├── forgot_passwd.tmpl
│   │   │   │   │   ├── grant.tmpl
│   │   │   │   │   ├── grant_error.tmpl
│   │   │   │   │   ├── link_account.tmpl
│   │   │   │   │   ├── prohibit_login.tmpl
│   │   │   │   │   ├── reset_passwd.tmpl
│   │   │   │   │   ├── signin.tmpl
│   │   │   │   │   ├── signin_openid.tmpl
│   │   │   │   │   ├── signup.tmpl
│   │   │   │   │   ├── signup_openid_connect.tmpl
│   │   │   │   │   ├── signup_openid_register.tmpl
│   │   │   │   │   ├── twofa.tmpl
│   │   │   │   │   ├── twofa_scratch.tmpl
│   │   │   │   │   └── webauthn.tmpl
│   │   │   │   ├── dashboard/
│   │   │   │   │   ├── dashboard.tmpl
│   │   │   │   │   ├── issues.tmpl
│   │   │   │   │   └── milestones.tmpl
│   │   │   │   ├── notification/
│   │   │   │   │   ├── notification_div.tmpl
│   │   │   │   │   └── notification_subscriptions.tmpl
│   │   │   │   ├── overview/
│   │   │   │   │   ├── header.tmpl
│   │   │   │   │   ├── package_versions.tmpl
│   │   │   │   │   └── packages.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── security/
│   │   │   │   │   │   ├── accountlinks.tmpl
│   │   │   │   │   │   ├── openid.tmpl
│   │   │   │   │   │   ├── security.tmpl
│   │   │   │   │   │   ├── twofa.tmpl
│   │   │   │   │   │   ├── twofa_enroll.tmpl
│   │   │   │   │   │   └── webauthn.tmpl
│   │   │   │   │   ├── access_token_edit.tmpl
│   │   │   │   │   ├── account.tmpl
│   │   │   │   │   ├── actions.tmpl
│   │   │   │   │   ├── appearance.tmpl
│   │   │   │   │   ├── applications.tmpl
│   │   │   │   │   ├── applications_oauth2.tmpl
│   │   │   │   │   ├── applications_oauth2_edit.tmpl
│   │   │   │   │   ├── applications_oauth2_edit_form.tmpl
│   │   │   │   │   ├── applications_oauth2_list.tmpl
│   │   │   │   │   ├── blocked_users.tmpl
│   │   │   │   │   ├── grants_oauth2.tmpl
│   │   │   │   │   ├── hook_new.tmpl
│   │   │   │   │   ├── hooks.tmpl
│   │   │   │   │   ├── keys.tmpl
│   │   │   │   │   ├── keys_gpg.tmpl
│   │   │   │   │   ├── keys_principal.tmpl
│   │   │   │   │   ├── keys_ssh.tmpl
│   │   │   │   │   ├── layout_footer.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── organization.tmpl
│   │   │   │   │   ├── packages.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_edit.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_preview.tmpl
│   │   │   │   │   ├── profile.tmpl
│   │   │   │   │   ├── repos.tmpl
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_setup.tmpl
│   │   │   │   │   └── storage_overview.tmpl
│   │   │   │   ├── code.tmpl
│   │   │   │   └── profile.tmpl
│   │   │   ├── webhook/
│   │   │   │   ├── new.tmpl
│   │   │   │   └── shared-settings.tmpl
│   │   │   ├── home.tmpl
│   │   │   ├── install.tmpl
│   │   │   └── post-install.tmpl
│   │   ├── README.md
│   │   ├── locale.lock.json
│   │   └── payload.json
│   ├── spaces/
│   │   ├── README.md
│   │   ├── soda-extension.ts
│   │   ├── soda-identity-response.ts
│   │   ├── soda-identity.ts
│   │   ├── soda-native-paths.ts
│   │   ├── soda-spaces-entry.ts
│   │   ├── soda-workspace-panel-entry.ts
│   │   ├── sodaspaces-api.ts
│   │   ├── sodaspaces-attention.ts
│   │   ├── sodaspaces-environment-view.ts
│   │   ├── sodaspaces-factory-navigation-view.ts
│   │   ├── sodaspaces-factory-response.ts
│   │   ├── sodaspaces-factory-screen.ts
│   │   ├── sodaspaces-factory-stream-response.ts
│   │   ├── sodaspaces-factory-view.ts
│   │   ├── sodaspaces-factory.ts
│   │   ├── sodaspaces-inventory-response.ts
│   │   ├── sodaspaces-keys-response.ts
│   │   ├── sodaspaces-layout.ts
│   │   ├── sodaspaces-network.ts
│   │   ├── sodaspaces-page.ts
│   │   ├── sodaspaces-project-access.ts
│   │   ├── sodaspaces-project-connection.ts
│   │   ├── sodaspaces-project-journey-view.ts
│   │   ├── sodaspaces-project-mutations.ts
│   │   ├── sodaspaces-project-network.ts
│   │   ├── sodaspaces-project-refresh.ts
│   │   ├── sodaspaces-project-request.ts
│   │   ├── sodaspaces-project-response.ts
│   │   ├── sodaspaces-project-runtime.ts
│   │   ├── sodaspaces-project-settings-view.ts
│   │   ├── sodaspaces-project-view.ts
│   │   ├── sodaspaces-project.css
│   │   ├── sodaspaces-project.ts
│   │   ├── sodaspaces-repository-picker-view.ts
│   │   ├── sodaspaces-repository-response.ts
│   │   ├── sodaspaces-session-navigation-view.ts
│   │   ├── sodaspaces-terminal-actions.ts
│   │   ├── sodaspaces-terminal-attachment.ts
│   │   ├── sodaspaces-terminal-dialog-view.ts
│   │   ├── sodaspaces-terminal-response.ts
│   │   ├── sodaspaces-terminal-screen.ts
│   │   ├── sodaspaces-terminal-view.ts
│   │   ├── sodaspaces-terminal.css
│   │   ├── sodaspaces-terminal.ts
│   │   ├── sodaspaces-widths.ts
│   │   ├── sodaspaces-workspace-drawer.ts
│   │   ├── sodaspaces-workspace-factory.ts
│   │   ├── sodaspaces-workspace-focus.ts
│   │   ├── sodaspaces-workspace-inventory.ts
│   │   ├── sodaspaces-workspace-layout.ts
│   │   ├── sodaspaces-workspace-measurement.ts
│   │   ├── sodaspaces-workspace-navigation.ts
│   │   ├── sodaspaces-workspace-pane-view.ts
│   │   ├── sodaspaces-workspace-setup.ts
│   │   ├── sodaspaces-workspace-shell-view.ts
│   │   ├── sodaspaces-workspace-terminal-hosts.ts
│   │   ├── sodaspaces-workspace-terminals.ts
│   │   ├── sodaspaces-workspace-toolbar-view.ts
│   │   ├── sodaspaces-workspace-types.ts
│   │   ├── sodaspaces-workspace-view.ts
│   │   ├── sodaspaces-workspace.css
│   │   ├── sodaspaces-workspace.ts
│   │   ├── sodaspaces.css
│   │   └── terminal-vendor.d.ts
│   └── tailnet/
│       ├── soda-settings.css
│       ├── soda-tailnet-actions.ts
│       ├── soda-tailnet-confirmation-view.ts
│       ├── soda-tailnet-enrollment-view.ts
│       ├── soda-tailnet-entry.ts
│       ├── soda-tailnet-host-view.ts
│       ├── soda-tailnet-observation.ts
│       ├── soda-tailnet-page.ts
│       ├── soda-tailnet-response.ts
│       └── soda-tailnet.css
├── internal/
│   ├── acceptance/
│   │   ├── developer_access.go
│   │   ├── developer_access_journey_test.go
│   │   ├── developer_access_request_test.go
│   │   ├── developer_access_session.go
│   │   ├── developer_access_test.go
│   │   ├── developer_access_transfer.go
│   │   ├── developer_access_users.go
│   │   ├── installed.go
│   │   ├── lifecycle_state.go
│   │   ├── lifecycle_state_test.go
│   │   ├── personal_git.go
│   │   ├── personal_git_exercise.go
│   │   ├── personal_git_keys.go
│   │   ├── personal_git_test.go
│   │   ├── personal_git_transport.go
│   │   ├── service_https.go
│   │   ├── service_https_test.go
│   │   ├── workload_access.go
│   │   ├── workload_access_test.go
│   │   ├── workload_exec.go
│   │   └── workload_exec_test.go
│   ├── archcheck/
│   │   └── arch_test.go
│   ├── avatar/
│   │   ├── testdata/
│   │   │   └── v1-snapshots.json
│   │   ├── README.md
│   │   ├── avatar.go
│   │   ├── avatar_test.go
│   │   └── style-v1.json
│   ├── config/
│   │   ├── background_test.go
│   │   ├── config.go
│   │   ├── config_test.go
│   │   ├── grant_key_test.go
│   │   ├── intake_test.go
│   │   ├── load_test.go
│   │   └── review_credential_test.go
│   ├── factory/
│   │   ├── control/
│   │   │   ├── acceptance.go
│   │   │   ├── acceptance_evidence.go
│   │   │   ├── acceptance_initial.go
│   │   │   ├── acceptance_initial_test.go
│   │   │   ├── acceptance_status.go
│   │   │   ├── acceptance_status_test.go
│   │   │   ├── acceptance_test.go
│   │   │   ├── checks_native_fixture_test.go
│   │   │   ├── checks_native_stale_test.go
│   │   │   ├── checks_native_test.go
│   │   │   ├── checks_pass.go
│   │   │   ├── checks_pass_test.go
│   │   │   ├── coordinator.go
│   │   │   ├── coordinator_test.go
│   │   │   ├── correction.go
│   │   │   ├── correction_test.go
│   │   │   ├── cycles.go
│   │   │   ├── dispatch.go
│   │   │   ├── dispatch_accounting_test.go
│   │   │   ├── dispatch_attempt.go
│   │   │   ├── dispatch_concurrency_test.go
│   │   │   ├── dispatch_fixture_test.go
│   │   │   ├── dispatch_inputs.go
│   │   │   ├── dispatch_launch.go
│   │   │   ├── dispatch_occupancy.go
│   │   │   ├── dispatch_recovery.go
│   │   │   ├── dispatch_recovery_test.go
│   │   │   ├── dispatch_registration.go
│   │   │   ├── dispatch_result.go
│   │   │   ├── dispatch_selection.go
│   │   │   ├── dispatch_test.go
│   │   │   ├── dispatch_wait_test.go
│   │   │   ├── grant_authority.go
│   │   │   ├── grants.go
│   │   │   ├── grants_test.go
│   │   │   ├── inventory_test.go
│   │   │   ├── lifecycle.go
│   │   │   ├── lifecycle_retry.go
│   │   │   ├── lifecycle_takeover.go
│   │   │   ├── lifecycle_test.go
│   │   │   ├── merge.go
│   │   │   ├── merge_effect_test.go
│   │   │   ├── merge_evidence.go
│   │   │   ├── merge_fixture_test.go
│   │   │   ├── merge_native_completion_test.go
│   │   │   ├── merge_native_effect_test.go
│   │   │   ├── merge_native_fixture_test.go
│   │   │   ├── merge_native_setup_test.go
│   │   │   ├── merge_native_stale_test.go
│   │   │   ├── merge_native_test.go
│   │   │   ├── merge_reconcile.go
│   │   │   ├── merge_test.go
│   │   │   ├── merge_withdraw.go
│   │   │   ├── operator.go
│   │   │   ├── operator_test.go
│   │   │   ├── postgres_fixture_external_test.go
│   │   │   ├── postgres_fixture_test.go
│   │   │   ├── preparation_decisions.go
│   │   │   ├── prerequisites.go
│   │   │   ├── publication.go
│   │   │   ├── publication_authority.go
│   │   │   ├── publication_effect_test.go
│   │   │   ├── publication_export.go
│   │   │   ├── publication_fixture_test.go
│   │   │   ├── publication_hooks_test.go
│   │   │   ├── publication_native_effect_test.go
│   │   │   ├── publication_native_fixture_test.go
│   │   │   ├── publication_native_test.go
│   │   │   ├── publication_reconcile.go
│   │   │   ├── publication_recovery_test.go
│   │   │   ├── publication_test.go
│   │   │   ├── publication_withdraw.go
│   │   │   ├── readiness.go
│   │   │   ├── readiness_fixture_test.go
│   │   │   ├── readiness_prerequisite_test.go
│   │   │   ├── readiness_sweep.go
│   │   │   ├── readiness_sweep_test.go
│   │   │   ├── readiness_test.go
│   │   │   ├── readiness_visibility_test.go
│   │   │   ├── review_cycle.go
│   │   │   ├── review_cycle_test.go
│   │   │   ├── review_executor.go
│   │   │   ├── review_native_primitive_test.go
│   │   │   ├── settle.go
│   │   │   ├── st15_demo_accept_test.go
│   │   │   ├── st15_demo_broker_test.go
│   │   │   ├── st15_demo_coding_test.go
│   │   │   ├── st15_demo_completion_test.go
│   │   │   ├── st15_demo_coordinator_test.go
│   │   │   ├── st15_demo_journey_test.go
│   │   │   ├── st15_demo_native_test.go
│   │   │   ├── st15_demo_project_test.go
│   │   │   ├── st15_demo_review_test.go
│   │   │   ├── st15_demo_runs_test.go
│   │   │   ├── st15_demo_seed_test.go
│   │   │   ├── st15_demo_stack_test.go
│   │   │   ├── traversal.go
│   │   │   └── traversal_test.go
│   │   ├── acceptance.go
│   │   ├── acceptance_test.go
│   │   ├── allowance.go
│   │   ├── allowance_test.go
│   │   ├── assignment.go
│   │   ├── assignment_resources.go
│   │   ├── assignment_result.go
│   │   ├── assignment_test.go
│   │   ├── checks.go
│   │   ├── checks_assessment.go
│   │   ├── checks_observation.go
│   │   ├── checks_test.go
│   │   ├── dispatch_prompt.go
│   │   ├── effective_authority.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── lifecycle.go
│   │   ├── lifecycle_test.go
│   │   ├── merge.go
│   │   ├── merge_evidence.go
│   │   ├── merge_operation.go
│   │   ├── merge_test.go
│   │   ├── publication.go
│   │   ├── publication_correction.go
│   │   ├── publication_intent.go
│   │   ├── publication_operation.go
│   │   ├── publication_refusal.go
│   │   ├── publication_test.go
│   │   ├── readiness.go
│   │   ├── readiness_test.go
│   │   ├── result.go
│   │   ├── review_native.go
│   │   ├── review_role_test.go
│   │   ├── run.go
│   │   ├── run_test.go
│   │   ├── sponsorship.go
│   │   ├── types.go
│   │   ├── views.go
│   │   └── views_test.go
│   ├── filelock/
│   │   ├── filelock.go
│   │   └── filelock_test.go
│   ├── forgejo/
│   │   ├── publish/
│   │   │   ├── candidate_validation.go
│   │   │   ├── credentials.go
│   │   │   ├── credentials_test.go
│   │   │   ├── git.go
│   │   │   ├── operation.go
│   │   │   ├── operation_git_test.go
│   │   │   ├── operation_observation.go
│   │   │   ├── operation_push.go
│   │   │   ├── operation_receipts.go
│   │   │   ├── operation_receipts_test.go
│   │   │   ├── operation_test.go
│   │   │   ├── publish.go
│   │   │   ├── publish_test.go
│   │   │   ├── review_role_test.go
│   │   │   └── source.go
│   │   ├── background.go
│   │   ├── background_admission.go
│   │   ├── background_test.go
│   │   ├── background_transport.go
│   │   ├── checks.go
│   │   ├── checks_test.go
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── errors.go
│   │   ├── errors_test.go
│   │   ├── merge.go
│   │   ├── merge_completion.go
│   │   ├── merge_observation.go
│   │   ├── merge_test.go
│   │   ├── observe.go
│   │   ├── observe_test.go
│   │   ├── own_keys.go
│   │   ├── ownership.go
│   │   ├── publish.go
│   │   ├── publish_test.go
│   │   ├── repositories.go
│   │   ├── repositories_test.go
│   │   ├── review.go
│   │   ├── review_test.go
│   │   ├── snapshot.go
│   │   ├── snapshot_issue.go
│   │   ├── snapshot_pull.go
│   │   ├── snapshot_request.go
│   │   ├── snapshot_test.go
│   │   ├── snapshot_transport.go
│   │   ├── snapshot_transport_test.go
│   │   ├── tailnet.go
│   │   └── tailnet_test.go
│   ├── host/
│   │   ├── access_keys.go
│   │   ├── client.go
│   │   ├── factory_candidate.go
│   │   ├── factory_candidate_test.go
│   │   ├── factory_client.go
│   │   ├── factory_export_test.go
│   │   ├── identity.go
│   │   ├── lifecycle.go
│   │   ├── os.go
│   │   ├── prepare.go
│   │   ├── prepare_test.go
│   │   ├── profiles.go
│   │   ├── tailnet.go
│   │   ├── tailnet_test.go
│   │   ├── terminal.go
│   │   ├── terminal_boundary_native_test.go
│   │   ├── terminal_client.go
│   │   └── terminal_native_test.go
│   ├── identity/
│   │   ├── client/
│   │   │   ├── broker_compat_test.go
│   │   │   ├── client.go
│   │   │   └── client_test.go
│   │   ├── enrollment.go
│   │   ├── event.go
│   │   ├── launch.go
│   │   ├── launch_test.go
│   │   ├── selection.go
│   │   ├── selection_test.go
│   │   ├── terminal.go
│   │   ├── transport.go
│   │   ├── types.go
│   │   └── types_test.go
│   ├── project/
│   │   ├── factory.go
│   │   ├── factory_candidate.go
│   │   ├── factory_candidate_test.go
│   │   ├── factory_export.go
│   │   ├── factory_export_test.go
│   │   ├── factory_harness.go
│   │   ├── factory_harness_test.go
│   │   ├── factory_output.go
│   │   ├── factory_output_test.go
│   │   ├── factory_test.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── preparation.go
│   │   ├── preparation_test.go
│   │   ├── profile.go
│   │   ├── project.go
│   │   ├── project_test.go
│   │   ├── takeover.go
│   │   ├── takeover_test.go
│   │   └── types.go
│   ├── store/
│   │   ├── corruption.go
│   │   ├── ephemeral.go
│   │   ├── factory.go
│   │   ├── factory_assignments.go
│   │   ├── factory_checks.go
│   │   ├── factory_checks_test.go
│   │   ├── factory_dispatch_packet.go
│   │   ├── factory_dispatch_queue.go
│   │   ├── factory_dispatch_queue_test.go
│   │   ├── factory_dispatch_test.go
│   │   ├── factory_grants.go
│   │   ├── factory_grants_test.go
│   │   ├── factory_inventory.go
│   │   ├── factory_inventory_test.go
│   │   ├── factory_lifecycle.go
│   │   ├── factory_lifecycle_test.go
│   │   ├── factory_merges.go
│   │   ├── factory_merges_test.go
│   │   ├── factory_publication_intent_test.go
│   │   ├── factory_publications.go
│   │   ├── factory_publications_test.go
│   │   ├── factory_reservations.go
│   │   ├── factory_retry_packet.go
│   │   ├── factory_retry_packet_test.go
│   │   ├── factory_review_role_test.go
│   │   ├── factory_test.go
│   │   ├── factory_views.go
│   │   ├── factory_views_test.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── identity.go
│   │   ├── identity_events.go
│   │   ├── identity_test.go
│   │   ├── issue_acceptances.go
│   │   ├── issue_acceptances_test.go
│   │   ├── issue_controls.go
│   │   ├── issue_controls_test.go
│   │   ├── members.go
│   │   ├── observe.go
│   │   ├── observe_test.go
│   │   ├── postgres_fixture_test.go
│   │   ├── preparation.go
│   │   ├── preparation_test.go
│   │   ├── project_grants.go
│   │   ├── project_grants_test.go
│   │   ├── project_profile_test.go
│   │   ├── schema.go
│   │   ├── schema_test.go
│   │   ├── staged_seed.go
│   │   ├── store.go
│   │   └── store_test.go
│   ├── strictjson/
│   │   ├── decode.go
│   │   └── decode_test.go
│   ├── tailnet/
│   │   ├── control_types.go
│   │   ├── control_validation.go
│   │   ├── project_runtime.go
│   │   ├── tailnet.go
│   │   └── tailnet_test.go
│   └── web/
│       ├── api/
│       │   ├── access_keys.go
│       │   ├── api.go
│       │   ├── dispatch_inputs.go
│       │   ├── dispatch_inputs_test.go
│       │   ├── environment_authority.go
│       │   ├── environment_os.go
│       │   ├── environments_api.go
│       │   ├── environments_create.go
│       │   ├── environments_join.go
│       │   ├── environments_join_test.go
│       │   ├── environments_preparation.go
│       │   ├── extension.go
│       │   ├── extension_native.go
│       │   ├── extension_terminal.go
│       │   ├── extension_terminal_authority.go
│       │   ├── extension_terminal_stream.go
│       │   ├── extension_test.go
│       │   ├── factory_assignments.go
│       │   ├── factory_assignments_test.go
│       │   ├── factory_intake.go
│       │   ├── factory_intake_test.go
│       │   ├── factory_issue_view.go
│       │   ├── factory_lifecycle.go
│       │   ├── factory_output.go
│       │   ├── factory_output_test.go
│       │   ├── factory_policy.go
│       │   ├── factory_readiness.go
│       │   ├── factory_settings.go
│       │   ├── factory_sponsorship.go
│       │   ├── factory_status.go
│       │   ├── factory_views.go
│       │   ├── factory_views_test.go
│       │   ├── identity.go
│       │   ├── identity_grants.go
│       │   ├── identity_launch.go
│       │   ├── issue_acceptance_evidence.go
│       │   ├── issue_acceptances.go
│       │   ├── issue_acceptances_test.go
│       │   ├── lifecycle.go
│       │   ├── operator.go
│       │   ├── preparation_decisions.go
│       │   ├── project_profiles.go
│       │   ├── provisioning.go
│       │   ├── repositories.go
│       │   ├── spaces.go
│       │   ├── spaces_authority.go
│       │   ├── spaces_inspection.go
│       │   ├── spaces_inventory.go
│       │   ├── tailnet.go
│       │   └── terminal_registry.go
│       ├── auth/
│       │   ├── auth.go
│       │   ├── development_key.go
│       │   ├── errors.go
│       │   ├── errors_test.go
│       │   ├── extension.go
│       │   ├── extension_service.go
│       │   ├── extension_test.go
│       │   ├── forgejo_keys.go
│       │   ├── http.go
│       │   ├── postgres_fixture_test.go
│       │   ├── service.go
│       │   └── session.go
│       ├── testdata/
│       │   └── avatar-browser.ts
│       ├── avatars.go
│       ├── avatars_browser_test.go
│       ├── avatars_test.go
│       ├── browser_join_test.go
│       ├── environment_authority_test.go
│       ├── environment_os_test.go
│       ├── environment_read_publication_test.go
│       ├── environments_api_test.go
│       ├── execution_access_test.go
│       ├── extension.go
│       ├── extension_terminal_test.go
│       ├── extension_test.go
│       ├── factory_checks_view_test.go
│       ├── factory_intake_route_test.go
│       ├── factory_lifecycle_test.go
│       ├── factory_output_fixture_test.go
│       ├── factory_output_stream_test.go
│       ├── factory_settings_test.go
│       ├── factory_views_test.go
│       ├── forgejo_keys_test.go
│       ├── identity_native_test.go
│       ├── issue_acceptances_test.go
│       ├── join_boundary_test.go
│       ├── lifecycle_access_keys_test.go
│       ├── mutation_admission_test.go
│       ├── postgres_fixture_test.go
│       ├── preparation_test.go
│       ├── project_profiles_test.go
│       ├── provisioning_lifetime_test.go
│       ├── repositories_test.go
│       ├── repository_access_test.go
│       ├── repository_settings_test.go
│       ├── retired_frontend_test.go
│       ├── server.go
│       ├── server_test.go
│       ├── spaces_inventory_test.go
│       ├── spaces_test.go
│       ├── tailnet_test.go
│       ├── terminal_test.go
│       └── test_helpers_test.go
├── lib/
│   ├── host/
│   │   ├── src/
│   │   │   ├── account/
│   │   │   │   ├── access_keys_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── daemon/
│   │   │   │   ├── admission.rs
│   │   │   │   ├── backend.rs
│   │   │   │   ├── broker.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── http.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── peer.rs
│   │   │   │   ├── response.rs
│   │   │   │   ├── routes.rs
│   │   │   │   └── websocket.rs
│   │   │   ├── domain/
│   │   │   │   ├── account.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── os.rs
│   │   │   │   ├── profile.rs
│   │   │   │   └── tests.rs
│   │   │   ├── factory/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   ├── candidate.rs
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── confirmation.rs
│   │   │   │   │   ├── launch.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── stop.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── artifacts.rs
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── confirmation.rs
│   │   │   │   ├── deadline.rs
│   │   │   │   ├── finish.rs
│   │   │   │   ├── identity.rs
│   │   │   │   ├── inspect.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── receipt.rs
│   │   │   │   ├── requests.rs
│   │   │   │   ├── run.rs
│   │   │   │   ├── state.rs
│   │   │   │   └── stop.rs
│   │   │   ├── json/
│   │   │   │   ├── bind.rs
│   │   │   │   ├── binding_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── number.rs
│   │   │   │   ├── scan.rs
│   │   │   │   ├── specs.rs
│   │   │   │   ├── strict_tests.rs
│   │   │   │   └── string.rs
│   │   │   ├── muse/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── caller.rs
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── execution.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── program.rs
│   │   │   │   │   ├── socket.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── arguments.rs
│   │   │   │   ├── caller.rs
│   │   │   │   ├── cleanup.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── execution.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── nested.rs
│   │   │   │   ├── program.rs
│   │   │   │   ├── socket.rs
│   │   │   │   └── wire.rs
│   │   │   ├── preparation/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── decisions.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── setup.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── validation_tests.rs
│   │   │   │   └── wire_tests.rs
│   │   │   ├── prepare/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── candidate_tests.rs
│   │   │   │   ├── execution_tests.rs
│   │   │   │   ├── helper.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── paths.rs
│   │   │   │   ├── source.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── tools.rs
│   │   │   ├── project/
│   │   │   │   ├── confirmation.rs
│   │   │   │   ├── confirmation_tests.rs
│   │   │   │   ├── connection.rs
│   │   │   │   ├── create.rs
│   │   │   │   ├── executor.rs
│   │   │   │   ├── inspect.rs
│   │   │   │   ├── lifecycle.rs
│   │   │   │   ├── lifecycle_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── os.rs
│   │   │   │   ├── profile.rs
│   │   │   │   └── tests.rs
│   │   │   ├── ssh/
│   │   │   │   ├── base64.rs
│   │   │   │   ├── certificate.rs
│   │   │   │   ├── material.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── mpint.rs
│   │   │   │   └── tests.rs
│   │   │   ├── tailnet/
│   │   │   │   ├── companion/
│   │   │   │   │   ├── enroll.rs
│   │   │   │   │   ├── execute.rs
│   │   │   │   │   ├── identity.rs
│   │   │   │   │   ├── identity_tests.rs
│   │   │   │   │   ├── lifecycle_tests.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── start.rs
│   │   │   │   │   ├── stop.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   ├── view.rs
│   │   │   │   │   └── view_tests.rs
│   │   │   │   ├── control/
│   │   │   │   │   ├── enrollment.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── native.rs
│   │   │   │   │   ├── policy.rs
│   │   │   │   │   ├── project.rs
│   │   │   │   │   ├── provider.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── domain/
│   │   │   │   │   ├── address_tests.rs
│   │   │   │   │   ├── addresses.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── native.rs
│   │   │   │   │   ├── node_tests.rs
│   │   │   │   │   ├── status.rs
│   │   │   │   │   ├── status_tests.rs
│   │   │   │   │   └── time.rs
│   │   │   │   ├── files/
│   │   │   │   │   ├── keys.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── resolver.rs
│   │   │   │   │   ├── run.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── runtime/
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── process.rs
│   │   │   │   │   ├── project.rs
│   │   │   │   │   ├── project_tests.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   └── mod.rs
│   │   │   ├── terminal/
│   │   │   │   ├── codex/
│   │   │   │   │   ├── tests/
│   │   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   │   ├── common.rs
│   │   │   │   │   │   ├── lifecycle.rs
│   │   │   │   │   │   ├── mod.rs
│   │   │   │   │   │   ├── reserve.rs
│   │   │   │   │   │   └── wire.rs
│   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   ├── commands.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── paths.rs
│   │   │   │   │   ├── reserve.rs
│   │   │   │   │   ├── run.rs
│   │   │   │   │   ├── start.rs
│   │   │   │   │   └── stop.rs
│   │   │   │   ├── tests/
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── identity.rs
│   │   │   │   │   ├── identity_wire.rs
│   │   │   │   │   ├── launch.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── protocol.rs
│   │   │   │   │   └── target.rs
│   │   │   │   ├── frame.rs
│   │   │   │   ├── identity.rs
│   │   │   │   ├── identity_protocol.rs
│   │   │   │   ├── identity_wire.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── lease.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── native.rs
│   │   │   │   ├── protocol.rs
│   │   │   │   ├── request.rs
│   │   │   │   └── target.rs
│   │   │   ├── lib.rs
│   │   │   ├── net.rs
│   │   │   ├── nist.rs
│   │   │   └── sha256.rs
│   │   ├── tests/
│   │   │   ├── common/
│   │   │   │   ├── backend.rs
│   │   │   │   └── mod.rs
│   │   │   ├── daemon.rs
│   │   │   └── terminal_transport.rs
│   │   └── Cargo.toml
│   ├── json/
│   │   ├── src/
│   │   │   └── lib.rs
│   │   └── Cargo.toml
│   ├── release-inputs/
│   │   ├── src/
│   │   │   ├── reader/
│   │   │   │   ├── forgejo.rs
│   │   │   │   ├── muse.rs
│   │   │   │   ├── settings.rs
│   │   │   │   ├── signature.rs
│   │   │   │   ├── stream.rs
│   │   │   │   └── url.rs
│   │   │   ├── lib.rs
│   │   │   └── reader.rs
│   │   └── Cargo.toml
│   ├── soda-release-build/
│   │   ├── src/
│   │   │   ├── coreos/
│   │   │   │   └── process.rs
│   │   │   ├── files/
│   │   │   │   └── tests.rs
│   │   │   ├── json_go/
│   │   │   │   └── tests.rs
│   │   │   ├── oci/
│   │   │   │   ├── archive.rs
│   │   │   │   ├── content.rs
│   │   │   │   ├── layers.rs
│   │   │   │   ├── manifest.rs
│   │   │   │   └── tests.rs
│   │   │   ├── production/
│   │   │   │   └── tests.rs
│   │   │   ├── progress/
│   │   │   │   └── tests.rs
│   │   │   ├── build_execution.rs
│   │   │   ├── clock.rs
│   │   │   ├── confined_files.rs
│   │   │   ├── coreos.rs
│   │   │   ├── coreos_iso.rs
│   │   │   ├── coreos_registry.rs
│   │   │   ├── coreos_stream.rs
│   │   │   ├── elf.rs
│   │   │   ├── files.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── http.rs
│   │   │   ├── json_emit.rs
│   │   │   ├── json_go.rs
│   │   │   ├── json_input.rs
│   │   │   ├── lib.rs
│   │   │   ├── live_inputs.rs
│   │   │   ├── oci.rs
│   │   │   ├── oci_layout.rs
│   │   │   ├── production.rs
│   │   │   ├── production_assets.rs
│   │   │   ├── production_compile.rs
│   │   │   ├── production_images.rs
│   │   │   ├── production_inputs.rs
│   │   │   ├── progress.rs
│   │   │   ├── tailnet_inputs.rs
│   │   │   └── test_support.rs
│   │   ├── tests/
│   │   │   ├── data/
│   │   │   │   ├── go-layout/
│   │   │   │   │   ├── blobs/
│   │   │   │   │   │   └── sha256/
│   │   │   │   │   │       ├── 098b60ba449c4b81d38cca87e08b16ff83522b9edb36bdb36025d9d370a99295
│   │   │   │   │   │       ├── 9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
│   │   │   │   │   │       └── 973a9bd7fe238b1604434f944e1ad4bff0629795321210dc11fcecf853ad3dce
│   │   │   │   │   ├── index.json
│   │   │   │   │   └── oci-layout
│   │   │   │   └── go-fixture.oci
│   │   │   ├── oracle/
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── oci.rs
│   │   │   │   ├── production.rs
│   │   │   │   └── progress.rs
│   │   │   ├── support/
│   │   │   │   └── buffer.rs
│   │   │   ├── oracle.rs
│   │   │   └── oracle_vectors.rs
│   │   └── Cargo.toml
│   ├── soda-release-deliver/
│   │   ├── src/
│   │   │   ├── buildx/
│   │   │   │   ├── filesystem.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── fetch/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── verification.rs
│   │   │   ├── jsonx/
│   │   │   │   ├── decode.rs
│   │   │   │   ├── emit.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── model/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── channel.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── release.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── trust.rs
│   │   │   ├── native/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── policy.rs
│   │   │   │   └── sign.rs
│   │   │   ├── oci/
│   │   │   │   ├── archive.rs
│   │   │   │   ├── layers.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── schema.rs
│   │   │   ├── payload/
│   │   │   │   └── tests.rs
│   │   │   ├── publish/
│   │   │   │   ├── channel.rs
│   │   │   │   ├── ledger.rs
│   │   │   │   └── mod.rs
│   │   │   ├── admission.rs
│   │   │   ├── check.rs
│   │   │   ├── content.rs
│   │   │   ├── document.rs
│   │   │   ├── finalize.rs
│   │   │   ├── import.rs
│   │   │   ├── lib.rs
│   │   │   ├── payload.rs
│   │   │   └── prepare.rs
│   │   ├── tests/
│   │   │   ├── goldens/
│   │   │   │   └── deliver.json
│   │   │   └── oracle/
│   │   │       ├── artifacts.rs
│   │   │       ├── fetch_state.rs
│   │   │       └── main.rs
│   │   ├── Cargo.toml
│   │   └── tools.json
│   ├── soda-release-image/
│   │   ├── src/
│   │   │   ├── build_context/
│   │   │   │   └── tests.rs
│   │   │   ├── build_runner/
│   │   │   │   └── tests.rs
│   │   │   ├── media/
│   │   │   │   └── tests.rs
│   │   │   ├── model/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── images.rs
│   │   │   │   ├── live_inputs.rs
│   │   │   │   ├── payload.rs
│   │   │   │   ├── tests.rs
│   │   │   │   ├── trust.rs
│   │   │   │   └── url.rs
│   │   │   ├── prepare/
│   │   │   │   └── tests.rs
│   │   │   ├── build.rs
│   │   │   ├── build_candidate.rs
│   │   │   ├── build_compile.rs
│   │   │   ├── build_context.rs
│   │   │   ├── build_media.rs
│   │   │   ├── build_runner.rs
│   │   │   ├── build_source.rs
│   │   │   ├── complete.rs
│   │   │   ├── compression.rs
│   │   │   ├── error.rs
│   │   │   ├── events.rs
│   │   │   ├── extension.rs
│   │   │   ├── files.rs
│   │   │   ├── foreign.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── host.rs
│   │   │   ├── ignition.rs
│   │   │   ├── inspect.rs
│   │   │   ├── jsonio.rs
│   │   │   ├── layout.rs
│   │   │   ├── lib.rs
│   │   │   ├── media.rs
│   │   │   ├── media_assembler.rs
│   │   │   ├── media_authentication.rs
│   │   │   ├── media_container.rs
│   │   │   ├── media_installer.rs
│   │   │   ├── model.rs
│   │   │   ├── packages.rs
│   │   │   ├── payload_stage.rs
│   │   │   ├── prepare.rs
│   │   │   ├── quadlet.rs
│   │   │   ├── recall.rs
│   │   │   ├── record.rs
│   │   │   ├── request.rs
│   │   │   ├── rootfs.rs
│   │   │   └── sys.rs
│   │   ├── tests/
│   │   │   ├── oracle/
│   │   │   │   ├── host.rs
│   │   │   │   ├── media.rs
│   │   │   │   └── staging.rs
│   │   │   └── oracle.rs
│   │   └── Cargo.toml
│   └── soda-release-tools/
│       ├── src/
│       │   ├── artifacts/
│       │   │   └── tests.rs
│       │   ├── bin/
│       │   │   ├── soda-artifacts.rs
│       │   │   ├── soda-build.rs
│       │   │   ├── soda-candidate-check.rs
│       │   │   └── soda-candidate.rs
│       │   ├── build_cli/
│       │   │   └── tests.rs
│       │   ├── candidate/
│       │   │   ├── mod.rs
│       │   │   ├── options.rs
│       │   │   └── tests.rs
│       │   ├── candidate_display/
│       │   │   ├── events.rs
│       │   │   ├── mod.rs
│       │   │   └── tests.rs
│       │   ├── candidate_fixture/
│       │   │   └── tests.rs
│       │   ├── candidate_prompts/
│       │   │   ├── defaults.rs
│       │   │   ├── mod.rs
│       │   │   └── tests.rs
│       │   ├── progress/
│       │   │   └── tests.rs
│       │   ├── worker/
│       │   │   ├── config.rs
│       │   │   ├── mod.rs
│       │   │   ├── runtime.rs
│       │   │   └── tests.rs
│       │   ├── artifacts.rs
│       │   ├── build_cli.rs
│       │   ├── build_spec.rs
│       │   ├── candidate_check.rs
│       │   ├── candidate_controller.rs
│       │   ├── candidate_fixture.rs
│       │   ├── candidate_hints.rs
│       │   ├── digest.rs
│       │   ├── exitcode.rs
│       │   ├── goflag.rs
│       │   ├── lib.rs
│       │   └── progress.rs
│       ├── tests/
│       │   └── cli/
│       │       ├── main.rs
│       │       ├── soda_artifacts.rs
│       │       ├── soda_build.rs
│       │       ├── soda_candidate.rs
│       │       └── soda_candidate_check.rs
│       ├── Cargo.toml
│       └── build.rs
├── scripts/
│   ├── fixtures/
│   │   ├── portcontracts/
│   │   │   ├── cli_surface.json
│   │   │   ├── dashboard_config_vectors.json
│   │   │   ├── host_config_bytes.json
│   │   │   ├── operator_wire.json
│   │   │   ├── strictjson_vectors.json
│   │   │   └── systemd_wiring.json
│   │   ├── spaces-review-client.ts
│   │   ├── spaces-review.css
│   │   ├── spaces-review.html
│   │   └── spaces-scenarios.ts
│   ├── build-forgejo-preview.ts
│   ├── build-forgejo.test.ts
│   ├── build-forgejo.ts
│   ├── build-soda-extension.test.ts
│   ├── build-soda-extension.ts
│   ├── check-complexity.sh
│   ├── check-errcheck.sh
│   ├── check-forgejo-branding.ts
│   ├── check-gofumpt.sh
│   ├── check-lit.ts
│   ├── check-native.sh
│   ├── check-no-npm.sh
│   ├── check-oxfmt.sh
│   ├── check-oxlint.sh
│   ├── check-source.sh
│   ├── check-sql-locality.sh
│   ├── check-staticcheck.sh
│   ├── check-ts-complexity.sh
│   ├── console_welcome_test.go
│   ├── forgejo_account_details_test.go
│   ├── forgejo_account_settings_test.go
│   ├── forgejo_admin_details_test.go
│   ├── forgejo_admin_monitoring_test.go
│   ├── forgejo_admin_org_test.go
│   ├── forgejo_auth_test.go
│   ├── forgejo_cargo_test.go
│   ├── forgejo_code_search_test.go
│   ├── forgejo_components_test.go
│   ├── forgejo_federated_auth_test.go
│   ├── forgejo_form_components_test.go
│   ├── forgejo_form_layout_test.go
│   ├── forgejo_home_redesign_test.go
│   ├── forgejo_insights_test.go
│   ├── forgejo_migrate_test.go
│   ├── forgejo_native_pages_test.go
│   ├── forgejo_notification_preview_test.go
│   ├── forgejo_onboarding_test.go
│   ├── forgejo_org_details_test.go
│   ├── forgejo_org_home_test.go
│   ├── forgejo_org_projects_test.go
│   ├── forgejo_owner_code_test.go
│   ├── forgejo_packages_test.go
│   ├── forgejo_presentation_test.go
│   ├── forgejo_profiles_test.go
│   ├── forgejo_project_board_test.go
│   ├── forgejo_repository_code_test.go
│   ├── forgejo_repository_content_test.go
│   ├── forgejo_repository_general_settings_test.go
│   ├── forgejo_repository_issues_test.go
│   ├── forgejo_repository_settings_collections_test.go
│   ├── forgejo_repository_settings_details_test.go
│   ├── forgejo_repository_settings_navigation_test.go
│   ├── forgejo_repository_test.go
│   ├── forgejo_setup_test.go
│   ├── forgejo_shared_projects_test.go
│   ├── forgejo_soda_settings_test.go
│   ├── forgejo_status_test.go
│   ├── forgejo_template_fixture_test.go
│   ├── forgejo_theme_components_test.go
│   ├── pg_backup_test.go
│   ├── pg_runtime_test.go
│   ├── preview-spaces.ts
│   ├── render-forgejo-branding.sh
│   ├── render_forgejo_branding_test.go
│   ├── render_forgejo_native_test.go
│   ├── screenshot.ts
│   ├── sodaspaces_templates_test.go
│   ├── system_formats_test.go
│   ├── terminal_branding_test.go
│   └── wire_contracts_test.go
├── system/
│   ├── containers/
│   │   ├── dashboard/
│   │   │   └── Containerfile
│   │   ├── extension/
│   │   │   ├── Containerfile
│   │   │   ├── extension.json
│   │   │   └── run
│   │   ├── forgejo/
│   │   │   └── Containerfile
│   │   └── tailnet/
│   │       └── Containerfile
│   ├── factory/
│   │   └── Containerfile
│   ├── host/
│   │   ├── config/
│   │   │   ├── 90-soda-routing.conf
│   │   │   ├── cockpit.conf
│   │   │   ├── cockpit.pam
│   │   │   ├── cockpit.socket.conf
│   │   │   ├── console-welcome.sh
│   │   │   ├── forgejo.env
│   │   │   ├── host.example.json
│   │   │   ├── proxy.Caddyfile
│   │   │   ├── soda.sysusers
│   │   │   └── soda.tmpfiles
│   │   ├── image/
│   │   │   ├── packages.tmpfiles
│   │   │   ├── retained-images.conf
│   │   │   └── soda-image-import.service
│   │   ├── installer/
│   │   │   └── load-console.sh
│   │   ├── provisioning/
│   │   │   ├── base.json
│   │   │   └── candidate.json
│   │   ├── selinux/
│   │   │   └── soda-build-worker.te
│   │   ├── services/
│   │   │   ├── README.md
│   │   │   ├── forgejo.container
│   │   │   ├── soda-console.service
│   │   │   ├── soda-dashboard.container
│   │   │   ├── soda-extension-install.service
│   │   │   ├── soda-forgejo-migrate.service
│   │   │   ├── soda-host.service
│   │   │   ├── soda-host.socket
│   │   │   ├── soda-identity-runtime.socket
│   │   │   ├── soda-identity.service
│   │   │   ├── soda-identity.socket
│   │   │   ├── soda-pg-provision.service
│   │   │   ├── soda-postgres-backup.service
│   │   │   ├── soda-postgres-backup.timer
│   │   │   ├── soda-postgres-init.service
│   │   │   ├── soda-postgres.container
│   │   │   ├── soda-project@.service
│   │   │   ├── soda-proxy.container
│   │   │   ├── soda-tailnet@.service
│   │   │   └── soda.network
│   │   ├── trust/
│   │   │   └── release-trust.json
│   │   └── Containerfile
│   ├── licenses/
│   │   ├── avatar-dependencies.txt
│   │   ├── forgejo-LICENSE
│   │   ├── lit-LICENSE
│   │   └── tailscale-LICENSE
│   └── project/
│       ├── licenses/
│       │   └── tea-LICENSE
│       ├── rootfs/
│       │   ├── etc/
│       │   │   ├── containers/
│       │   │   │   ├── containers.conf
│       │   │   │   └── storage.conf
│       │   │   ├── mise/
│       │   │   │   └── config.toml
│       │   │   ├── profile.d/
│       │   │   │   ├── soda-mise.sh
│       │   │   │   └── soda-podman.sh
│       │   │   ├── ssh/
│       │   │   │   └── sshd_config.d/
│       │   │   │       └── 10-soda.conf
│       │   │   ├── sudoers.d/
│       │   │   │   └── soda-project
│       │   │   ├── systemd/
│       │   │   │   └── system/
│       │   │   │       ├── soda-podman.service
│       │   │   │       ├── soda-podman.socket
│       │   │   │       └── soda-project-init.service
│       │   │   └── yum.repos.d/
│       │   │       └── gh-cli.repo
│       │   └── usr/
│       │       └── libexec/
│       │           └── soda/
│       │               └── project-init
│       ├── Containerfile
│       └── muse-release.json
├── tests/
│   ├── build/
│   │   ├── avatar_integration_test.go
│   │   ├── candidate_gate_test.go
│   │   ├── complexity_scope_test.go
│   │   ├── forgejo_domain_test.go
│   │   ├── forgejo_payload_test.go
│   │   ├── helpers.go
│   │   ├── native_support_test.go
│   │   ├── operator_probe_test.go
│   │   ├── project_account_test.go
│   │   ├── project_foundation_test.go
│   │   ├── project_runtime_test.go
│   │   ├── proxy_image_test.go
│   │   ├── sodaspaces_test.go
│   │   ├── source_checks_test.go
│   │   ├── tailnet_image_test.go
│   │   ├── terminal_assets_test.go
│   │   ├── u08_state_test.go
│   │   └── workload_probe_test.go
│   ├── fixtures/
│   │   └── workload/
│   │       ├── public/
│   │       │   └── index.html
│   │       ├── Containerfile
│   │       └── compose.yaml
│   ├── forgejo/
│   │   ├── fixtures/
│   │   │   ├── component-browser.ts
│   │   │   └── lit-smoke.ts
│   │   ├── presentation/
│   │   │   ├── dashboard-sidebar-browser.test.ts
│   │   │   ├── form-browser.test.ts
│   │   │   ├── form-native-contracts.json
│   │   │   ├── form-presentation-deltas.json
│   │   │   ├── form-source.test.ts
│   │   │   ├── gallery.test.ts
│   │   │   ├── gallery.tmpl
│   │   │   ├── home-background-browser.test.ts
│   │   │   ├── inventory.json
│   │   │   ├── inventory.test.ts
│   │   │   ├── locales.test.ts
│   │   │   ├── login-station-browser.test.ts
│   │   │   ├── migration-browser.test.ts
│   │   │   ├── notification-layout-browser.test.ts
│   │   │   ├── profile-browser.test.ts
│   │   │   ├── refinement-browser.test.ts
│   │   │   ├── refinement-gallery.tmpl
│   │   │   ├── repository-actionbar-browser.test.ts
│   │   │   ├── repository-settings-browser.test.ts
│   │   │   ├── repository-settings-gallery.tmpl
│   │   │   ├── repository-settings-native-contracts.json
│   │   │   ├── repository-settings-source.test.ts
│   │   │   ├── settings-browser.test.ts
│   │   │   ├── settings-contracts.ts
│   │   │   ├── settings-native-contracts.json
│   │   │   ├── settings-source.test.ts
│   │   │   └── workspace-tokens.test.ts
│   │   ├── branding.test.ts
│   │   ├── cockpit-branding.test.ts
│   │   ├── component-boundaries.test.ts
│   │   ├── component-layout-boundaries.test.ts
│   │   ├── component-toolbar-boundaries.test.ts
│   │   ├── explore-overflow.test.ts
│   │   ├── lit-build.test.ts
│   │   ├── lit-runtime.test.ts
│   │   ├── login-theme.test.ts
│   │   ├── milestone-page-boundaries.json
│   │   ├── milestone-source.test.ts
│   │   ├── milestones-layout.test.ts
│   │   ├── notification-preview.test.ts
│   │   ├── repository-actions.test.ts
│   │   ├── repository-container.test.ts
│   │   ├── repository-switcher.test.ts
│   │   └── work-item-lists.test.ts
│   ├── frontend/
│   │   ├── fixtures/
│   │   │   ├── drawer-fixture.ts
│   │   │   ├── persistent-panel-fixture.ts
│   │   │   ├── project-controls-driver.ts
│   │   │   ├── terminal-driver.ts
│   │   │   ├── terminal-fixture.ts
│   │   │   ├── workspace-driver.ts
│   │   │   ├── workspace-fixture.ts
│   │   │   ├── workspace-measurement-probe.ts
│   │   │   └── workspace-model.ts
│   │   ├── drawer-controls.test.ts
│   │   ├── factory-view.test.ts
│   │   ├── identity.test.ts
│   │   ├── journey-input.test.ts
│   │   ├── native-browser.test.ts
│   │   ├── native-paths.test.ts
│   │   ├── native-project-controls.test.ts
│   │   ├── persistent-panel.test.ts
│   │   ├── project-access-controls.test.ts
│   │   ├── project-environment-controls.test.ts
│   │   ├── repository-settings.test.ts
│   │   ├── sodaspaces-http.test.ts
│   │   ├── spaces-api.test.ts
│   │   ├── spaces-attention.test.ts
│   │   ├── spaces-layout.test.ts
│   │   ├── spaces-preview.test.ts
│   │   ├── tailnet.test.ts
│   │   ├── terminal-end.test.ts
│   │   ├── terminal-retirement.test.ts
│   │   ├── terminal-stream.test.ts
│   │   ├── terminal.test.ts
│   │   ├── workspace-attention.test.ts
│   │   ├── workspace-first-use.test.ts
│   │   ├── workspace-journey.test.ts
│   │   ├── workspace-layout.test.ts
│   │   ├── workspace-navigation.test.ts
│   │   ├── workspace-persistence.test.ts
│   │   ├── workspace-recovery.test.ts
│   │   ├── workspace-responsive.test.ts
│   │   ├── workspace-setup.test.ts
│   │   └── workspace.test.ts
│   └── installed/
│       ├── cockpit-types.ts
│       ├── forgejo-advertisement.sh
│       ├── host.sh
│       ├── native-browser.ts
│       ├── operator.sh
│       ├── operator.ts
│       ├── project-foundation.sh
│       ├── project-os.sh
│       ├── service-ordering.sh
│       ├── shared-tools.sh
│       ├── sodaspaces-cli.ts
│       ├── sodaspaces-controls.ts
│       ├── sodaspaces-first-use-journey.ts
│       ├── sodaspaces-http.ts
│       ├── sodaspaces-input.ts
│       ├── sodaspaces-journey-evidence.ts
│       ├── sodaspaces-matrix-input.ts
│       ├── sodaspaces-matrix-native.ts
│       ├── sodaspaces-workspace-journey.ts
│       └── workloads.sh
├── tools/
│   ├── acceptance/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-acceptance-remote.rs
│   │   │   │   └── soda-host-probes.rs
│   │   │   ├── command/
│   │   │   │   ├── execute.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── ssh.rs
│   │   │   │   └── tests.rs
│   │   │   ├── coreos/
│   │   │   │   └── tests.rs
│   │   │   ├── driver/
│   │   │   │   ├── actions.rs
│   │   │   │   ├── finalization.rs
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── options.rs
│   │   │   │   └── tests.rs
│   │   │   ├── evidence/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── redaction.rs
│   │   │   │   ├── store.rs
│   │   │   │   └── tests.rs
│   │   │   ├── files/
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── owned_directory.rs
│   │   │   │   ├── temporary.rs
│   │   │   │   └── tests.rs
│   │   │   ├── host_probes/
│   │   │   │   ├── content.rs
│   │   │   │   ├── deployments.rs
│   │   │   │   ├── forgejo.rs
│   │   │   │   ├── listeners.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── tailnet.rs
│   │   │   │   └── tests.rs
│   │   │   ├── jsonio/
│   │   │   │   └── tests.rs
│   │   │   ├── native_phase/
│   │   │   │   └── tests.rs
│   │   │   ├── process/
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── owned_process.rs
│   │   │   │   ├── phase.rs
│   │   │   │   └── tests.rs
│   │   │   ├── project_state/
│   │   │   │   ├── command.rs
│   │   │   │   ├── files.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── snapshot.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── workloads.rs
│   │   │   ├── report/
│   │   │   │   ├── handoff.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── observation.rs
│   │   │   │   └── tests.rs
│   │   │   ├── timestamps/
│   │   │   │   └── tests.rs
│   │   │   ├── trust/
│   │   │   │   ├── host_key.rs
│   │   │   │   ├── inline_data.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── vm/
│   │   │   │   ├── base.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── lifecycle.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── cockpit.rs
│   │   │   ├── coreos.rs
│   │   │   ├── error.rs
│   │   │   ├── jsonio.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── native_phase.rs
│   │   │   ├── personal_git.rs
│   │   │   ├── probe.rs
│   │   │   ├── provisioning.rs
│   │   │   ├── qmp.rs
│   │   │   ├── remote.rs
│   │   │   ├── sha256.rs
│   │   │   └── timestamps.rs
│   │   ├── Cargo.toml
│   │   └── build.rs
│   ├── candidate-setup/
│   │   ├── src/
│   │   │   ├── config.rs
│   │   │   ├── controller.rs
│   │   │   ├── fixture_authority.rs
│   │   │   ├── main.rs
│   │   │   ├── preflight.rs
│   │   │   ├── process.rs
│   │   │   ├── selinux.rs
│   │   │   ├── storage.rs
│   │   │   ├── tests.rs
│   │   │   ├── worker_caches.rs
│   │   │   └── worker_tools.rs
│   │   ├── tests/
│   │   │   ├── support/
│   │   │   │   └── mod.rs
│   │   │   └── cli.rs
│   │   ├── Cargo.toml
│   │   └── README.md
│   ├── lab-credentials/
│   │   ├── src/
│   │   │   ├── fixture_authority.rs
│   │   │   ├── inventory.rs
│   │   │   ├── main.rs
│   │   │   ├── process.rs
│   │   │   ├── runbooks.rs
│   │   │   └── tests.rs
│   │   ├── tests/
│   │   │   └── cli.rs
│   │   ├── Cargo.toml
│   │   └── README.md
│   ├── lit-check/
│   │   ├── fixtures/
│   │   │   ├── aria.ts
│   │   │   ├── boolean.ts
│   │   │   ├── customProperty.ts
│   │   │   ├── directive.ts
│   │   │   ├── event.ts
│   │   │   ├── nullable.ts
│   │   │   ├── positive.ts
│   │   │   ├── property.ts
│   │   │   ├── unclosed.ts
│   │   │   ├── unknownEvent.ts
│   │   │   └── unknownProperty.ts
│   │   ├── README.md
│   │   ├── check.ts
│   │   ├── package.json
│   │   └── tsconfig.json
│   ├── png-equal/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── postgres-fixture/
│   │   ├── src/
│   │   │   └── main.rs
│   │   └── Cargo.toml
│   ├── release-assets/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-fetch-muse.rs
│   │   │   │   ├── soda-fetch-tea.rs
│   │   │   │   ├── soda-fetch-terminal.rs
│   │   │   │   ├── soda-forgejo-locales.rs
│   │   │   │   ├── soda-render-provisioning.rs
│   │   │   │   ├── soda-render-terminal-logo.rs
│   │   │   │   └── soda-stage.rs
│   │   │   ├── fetch/
│   │   │   │   ├── muse/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── tea/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── muse.rs
│   │   │   │   ├── tea.rs
│   │   │   │   ├── terminal.rs
│   │   │   │   └── test_server.rs
│   │   │   ├── locales/
│   │   │   │   ├── merge/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── merge.rs
│   │   │   │   └── mod.rs
│   │   │   ├── render/
│   │   │   │   ├── provisioning/
│   │   │   │   │   ├── document.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── private_files.rs
│   │   │   │   │   ├── render.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── stage/
│   │   │   │   │   ├── branding.rs
│   │   │   │   │   ├── files.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── payload.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── terminal_logo/
│   │   │   │   │   ├── geometry.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── svg.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   └── mod.rs
│   │   │   └── lib.rs
│   │   ├── tests/
│   │   │   ├── fixtures/
│   │   │   │   ├── prov-root/
│   │   │   │   │   ├── assets/
│   │   │   │   │   │   └── branding/
│   │   │   │   │   │       └── source/
│   │   │   │   │   │           └── soda-symbol.svg
│   │   │   │   │   ├── frontend/
│   │   │   │   │   │   └── forgejo/
│   │   │   │   │   │       └── payload.json
│   │   │   │   │   └── system/
│   │   │   │   │       └── host/
│   │   │   │   │           └── provisioning/
│   │   │   │   │               └── base.json
│   │   │   │   ├── ext-host.bu
│   │   │   │   ├── ext-hostkey.bu
│   │   │   │   ├── ext-product.bu
│   │   │   │   └── min-host.bu
│   │   │   ├── locales_support/
│   │   │   │   └── mod.rs
│   │   │   ├── render_support/
│   │   │   │   └── mod.rs
│   │   │   ├── fetch_cli.rs
│   │   │   ├── locales_cli.rs
│   │   │   ├── locales_locked_input.rs
│   │   │   ├── locales_native_merge.rs
│   │   │   ├── render_provisioning.rs
│   │   │   ├── render_staging.rs
│   │   │   └── render_terminal_logo.rs
│   │   ├── Cargo.toml
│   │   └── terminal-assets.lock.json
│   ├── soda-avatars/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── soda-installed-probes/
│   │   ├── cockpit_account_test.go
│   │   ├── installed_probes_test.go
│   │   ├── main.go
│   │   ├── main_test.go
│   │   └── project_state_test.go
│   ├── soda-rootfs-server/
│   │   ├── main.go
│   │   ├── main_test.go
│   │   └── soda-rootfs-server.service
│   └── test-vm/
│       ├── src/
│       │   ├── main.rs
│       │   ├── process.rs
│       │   ├── start.rs
│       │   ├── state.rs
│       │   ├── tests.rs
│       │   └── transport.rs
│       ├── tests/
│       │   ├── support/
│       │   │   └── mod.rs
│       │   ├── cli.rs
│       │   ├── start.rs
│       │   ├── status.rs
│       │   └── transport.rs
│       ├── Cargo.toml
│       └── README.md
├── .containerignore
├── .gitignore
├── .oxfmtrc.json
├── .oxlintrc.json
├── AGENTS.md
├── Cargo.lock
├── Cargo.toml
├── LICENSE
├── NOTICE
├── README.md
├── bun.lock
├── bunfig.toml
├── go.mod
├── go.sum
├── package.json
├── rust-toolchain.toml
├── tsconfig.base.json
├── tsconfig.browser.json
├── tsconfig.json
└── tsconfig.tests.json
```

Ignored roots remain `.artifacts/` for build/evidence outputs, `.local/` for local inputs, `target/` for Cargo outputs, and `node_modules/` for dependencies. Their generated contents and private fixtures are outside the tracked-source inventory and are not enumerated here.

## Detailed decomposition

The following names describe existing concern seams. The Rust host, installer,
importer and acceptance entries are the retained plan under the decided language
policy. Retiring Go/Python sources have removal dispositions and successor
references instead of target files for obsolete implementations. Historical
mockups are design material, not production owners. Embedded Rust tests stay
descendants of their implementation module so private access can survive. A
stateful frontend extraction requires an actual private seam; moving methods
across files alone cannot preserve one TypeScript class owner.

### appliance/forgejo/templates/admin/auth/edit.tmpl

Observed size: 449 lines, including tests where embedded. Separate existing provider-specific form sections; preserve the native entry name, actions, input IDs/names and common form owner.

- `frontend/forgejo/templates/admin/auth/edit.tmpl` — Native edit entry, common source identity/status fields, one form, CSRF, submit/delete and tips/modal wiring.
- `frontend/forgejo/templates/admin/auth/edit_ldap.tmpl` — Existing LDAP/direct-LDAP bind, search, group, attributes and sync fields.
- `frontend/forgejo/templates/admin/auth/edit_smtp.tmpl` — Existing SMTP source fields.
- `frontend/forgejo/templates/admin/auth/edit_pam.tmpl` — Existing PAM source fields.
- `frontend/forgejo/templates/admin/auth/edit_oauth.tmpl` — Existing OAuth2 provider URLs, scopes, claims, tenant and quota mapping fields.

Evidence: admin/auth/edit.tmpl:10-23 common form; 24-183 LDAP/DLDAP; 184-245 SMTP; 246-265 PAM; 266-402 OAuth2; 404-449 shared status/actions/tips/modal.

Open detail: Go template invocations do not inherit lexical $cfg variables. Pass the same admitted native context and provider config explicitly; update the actual Forgejo payload inventory and existing source/browser contracts.

### appliance/forgejo/templates/repo/settings/options.tmpl

Observed size: 808 lines, including tests where embedded. The 808-line entry owns several already distinct native forms. Extract complete sections and confirmation markup, leaving native authorization branches and form ownership intact.

- `frontend/forgejo/templates/repo/settings/options.tmpl` — Native entry, layout and conditional composition of existing settings sections.
- `frontend/forgejo/templates/repo/settings/options_repository.tmpl` — Repository name, description, visibility/template fields and avatar form.
- `frontend/forgejo/templates/repo/settings/options_federation.tmpl` — Existing FederationEnabled-gated following-repositories form.
- `frontend/forgejo/templates/repo/settings/options_mirrors.tmpl` — Existing pull/push mirror conditions, forms and local mirror-selection variables.
- `frontend/forgejo/templates/repo/settings/options_trust.tmpl` — Existing signature trust-model radios and help.
- `frontend/forgejo/templates/repo/settings/options_maintenance.tmpl` — Existing fsck/health, code/issue/statistics indexer administration forms.
- `frontend/forgejo/templates/repo/settings/options_danger.tmpl` — Existing conversion, transfer, archive, delete and wiki action controls.
- `frontend/forgejo/templates/repo/settings/options_modals.tmpl` — Existing conversion/transfer/delete/wiki/archive confirmation modals and push mirror sync include.

Evidence: repo/settings/options.tmpl:1-70 identity/avatar; 72-94 federation; 96-348 mirrors; 350-393 trust; 396-460 maintenance; 467-575 danger controls; 579-808 modals.

Open detail: Keep mirror variables with their entire section; retain native template entry and dialog IDs, POST action values, CSRF tokens and the manifest-described source closure.

### assets/branding/forgejo/components-toolbar.css

Observed size: 476 lines, including tests where embedded. Split by existing selector ownership; canonical styles remain assets rather than becoming a second frontend theme.

- `assets/branding/forgejo/components-toolbar.css` — Preserved public stylesheet entry, composing the canonical concern sheets in reviewed cascade order.
- `assets/branding/forgejo/components-toolbar-layout.css` — Toolbar shell, native search/action inputs, disclosure controls and their responsive rules.
- `assets/branding/forgejo/components-navigation.css` — Native context switcher, jump menu and overflow-menu tabs, including active/responsive states.
- `assets/branding/forgejo/components-repository-toolbar.css` — Native repository compact-size button variants and joined clone/folder controls.

Evidence: components-toolbar.css:3-79 .soda-toolbar/search; 80-247 .soda-context-switcher/.soda-tabs; 248-366 native controls; 367-420 responsive overrides; 421-476 repository toolbar.

Open detail: Preserve cascade and installed URL closure. component-boundaries.test.ts:16-27 reads selected CSS files and strips @import; it must resolve the real new stylesheet sources instead of silently dropping imports.

### assets/branding/forgejo/repository-code.css

Observed size: 465 lines, including tests where embedded. Split the current browser, editing/compare and native Actions selectors, preserving upstream editor/polling/dispatch ownership.

- `assets/branding/forgejo/repository-code.css` — Preserved public stylesheet entry composing existing code-workflow sheets.
- `assets/branding/forgejo/repository-code-browser.css` — Finder/file listing, branch/tag/commit history, quickstart/forks/people and landing sidebar/README presentation.
- `assets/branding/forgejo/repository-code-editing.css` — Existing file headers, editor chrome, branch picker and compare/diff framing.
- `assets/branding/forgejo/repository-code-actions.css` — Existing native Actions lists/filter/dispatch and corresponding narrow-screen rules.

Evidence: repository-code.css:3-194 browsing/history; 195-287 editor/compare; 288-324 quickstart/forks/people; 325-400 native Actions; 401-465 landing sidebar/README.

Open detail: Retain CSS cascade, relative imports, native selectors and payload paths; update fixtures to read the actual stylesheet closure.

### docs/design/spaces/preview.css

Observed size: 1025 lines, including tests where embedded. The owning README identifies this as superseded 8fe3468 history rather than a current product/design candidate. Do not convert an obsolete fake state machine into a newly maintained module graph.

- `docs/design/spaces/preview.css` — Historical superseded mockup source, retained intact pending an explicit disposition decision.

Evidence: docs/design/spaces/README.md:64-70 superseded mockup and historical retention; preview.ts:1-2 design-only/no API/socket/auth/storage/shell; review.ts:19-27 current historical source loading.

Open detail: If the owner chooses retirement, remove the whole obsolete preview/review/compiler/capture group coherently. If the owner explicitly wants it maintained, a separate design-only split can use preview-state.ts, preview-navigation.ts, preview-panes.ts and preview-dialogs.ts with preview.css split into base/navigation/panes/dialogs sheets; this does not establish production requirements.

### docs/design/spaces/preview.ts

Observed size: 706 lines, including tests where embedded. The owning README identifies this as superseded 8fe3468 history rather than a current product/design candidate. Do not convert an obsolete fake state machine into a newly maintained module graph.

- `docs/design/spaces/preview.ts` — Historical superseded mockup source, retained intact pending an explicit disposition decision.

Evidence: docs/design/spaces/README.md:64-70 superseded mockup and historical retention; preview.ts:1-2 design-only/no API/socket/auth/storage/shell; review.ts:19-27 current historical source loading.

Open detail: If the owner chooses retirement, remove the whole obsolete preview/review/compiler/capture group coherently. If the owner explicitly wants it maintained, a separate design-only split can use preview-state.ts, preview-navigation.ts, preview-panes.ts and preview-dialogs.ts with preview.css split into base/navigation/panes/dialogs sheets; this does not establish production requirements.

### frontend/spaces/sodaspaces-api.ts

Observed size: 858 lines, including tests where embedded. Move each existing response contract to its current product concern. Keep one definition per DTO and import the actual owner directly.

- `frontend/spaces/sodaspaces-api.ts` — Bounded JSON/object primitives, common identifier admission and SodaRequestError; no forwarding exports for moved contracts.
- `frontend/spaces/sodaspaces-project-response.ts` — Canonical browser CreationProfile/Environment/Detail/OSObservation types and their existing decoders.
- `frontend/spaces/sodaspaces-keys-response.ts` — Saved/profile key and preview response types/decoders.
- `frontend/spaces/sodaspaces-terminal-response.ts` — Terminal identity/metadata types and exact binding/state response admission.
- `frontend/spaces/sodaspaces-factory-response.ts` — Existing factory authority/control/run types, status/detail admission and bounded display text.
- `frontend/spaces/sodaspaces-factory-stream-response.ts` — Existing factory status/output frame admission, cursor accounting and closed reason.
- `frontend/spaces/sodaspaces-inventory-response.ts` — Canonical Space composition (Environment/Detail plus TerminalMetadata and FactoryAuthority/FactoryControl/FactoryRun references), spaceNetwork and existing collection admission; observed inventory and incomplete-page facts remain here.
- `frontend/spaces/sodaspaces-repository-response.ts` — Existing repository-choice/path/pagination admission.

Evidence: sodaspaces-api.ts:4-47 common JSON/object/identifier primitives -> sodaspaces-api.ts; 48-185 profile/environment/OS definitions and admission -> project-response; 186-239 key definitions/preview admission -> keys-response; 240-298 terminal metadata/terminalResponse -> terminal-response.; sodaspaces-api.ts:299-317 FactoryAuthority/FactoryControl/FactoryRun definitions, 332-356 factory display/control command functions, 362-650 factory-field/run/detail admission -> factory-response; 651-703 status/output stream frames -> factory-stream-response.; sodaspaces-api.ts:318-331 Space interface and 357-361 spaceNetwork -> inventory-response; 704-814 actual Space/collection admission -> inventory-response. Space imports Environment/Detail, TerminalMetadata and the canonical Factory records directly; no duplicate DTOs.; sodaspaces-api.ts:815-858 repository choices/path/pagination and request error; repository contracts -> repository-response, SodaRequestError -> original common API owner..

Open detail: Space belongs to the collection/inventory owner and composes the canonical project, terminal and factory records through direct imports. Factory-field admission is reused by the Space decoder rather than recreated. Update every real caller, including installed journeys and Tailnet common JSON imports; final module visibility/signatures require implementation review.

### frontend/spaces/sodaspaces-factory.ts

Observed size: 529 lines, including tests where embedded. Extract screen setup and pure presentation from the existing read-only watcher without creating another run controller or input-capable transport.

- `frontend/spaces/sodaspaces-factory.ts` — One immutable factory run binding, generation/lifetime, read-only socket watch and retirement owner; public mount remains here.
- `frontend/spaces/sodaspaces-factory-screen.ts` — Existing disabled-input xterm construction, font/theme/fit and OSC behavior.
- `frontend/spaces/sodaspaces-factory-view.ts` — Reuse existing stateless view owner; absorb watch presentation/title/status and menu projection from the class.

Evidence: sodaspaces-factory.ts:40-42 read-only ownership contract; 123-188 view/menu projection; 236-304 status/font/theme/xterm; 305-475 current watch/socket/lifetime; 478-529 mount admission.

Open detail: Keep generation/cursor/retirement in the same owner. Screen helpers receive the existing owner callbacks/resources; no generic terminal/factory lifecycle abstraction is implied.

### frontend/spaces/sodaspaces-project.ts

Observed size: 1895 lines, including tests where embedded. Separate the current journey/settings projections and existing request/read/mutation capabilities. Keep original-target identity, epoch cancellation and dispatch admission in the same component; reuse environment/project/network view modules.

- `frontend/spaces/sodaspaces-project.ts` — SodaProjectControls state, original native binding, epoch/lifetime gate, Lit setup/render coordination and public mount.
- `frontend/spaces/sodaspaces-project-request.ts` — Existing native transport request preparation, error-code admission and authorization-loss callback.
- `frontend/spaces/sodaspaces-project-journey-view.ts` — Existing welcome/unavailable/join-failed/join/existing/configure journey projections.
- `frontend/spaces/sodaspaces-project-settings-view.ts` — Existing repository identity/tab/environment/access/network view composition, reusing existing concern views.
- `frontend/spaces/sodaspaces-project-refresh.ts` — Existing account/environment/profile/detail refresh sequencing and admitted read results.
- `frontend/spaces/sodaspaces-project-connection.ts` — Existing joined-account SSH connection observation and deliberate copy action.
- `frontend/spaces/sodaspaces-project-mutations.ts` — Existing dispatch, result checks, uncertain outcome explanations and completion observation.
- `frontend/spaces/sodaspaces-project-network.ts` — Existing Tailnet option/observation/change branches and confirmation admission.
- `frontend/spaces/sodaspaces-project-access.ts` — Existing explicit keyless/keyed Join, profile/saved key review, save and Apply actions.
- `frontend/spaces/sodaspaces-project-runtime.ts` — Existing OS read, observed runtime state, explicit Start/Stop and shared-impact confirmation.

Evidence: sodaspaces-project.ts:1-37 existing view/response imports; 172-347 component state/configure; 373-427 epoch/command/create gate; 430-780 journey; 802-1034 settings views; 1036-1080 join/copy; 1110-1494 reads; 1495-1687 mutations; 1688-1863 network/OS/lifecycle/keys; 1864-1895 disposal/mount.

Open detail: These are module boundaries, not permission for separate mutable controllers, new policy, or parallel request models. Exact exported helper signatures and post-extraction line sizes still need implementation review.

### frontend/spaces/sodaspaces-terminal.ts

Observed size: 964 lines, including tests where embedded. Separate existing renderer, attachment protocol and explicit native End operation while retaining one original actor and imperative lifetime owner.

- `frontend/spaces/sodaspaces-terminal.ts` — SodaTerminal original account/binding/generation, component lifecycle, visibility and public exact-locator mount.
- `frontend/spaces/sodaspaces-terminal-attachment.ts` — Existing metadata inspect, reservation/create/attach handshake, native-generation frame dispatch and socket callbacks.
- `frontend/spaces/sodaspaces-terminal-screen.ts` — Existing xterm/font/theme/input/fit/geometry/minimum-size and renderer readiness code.
- `frontend/spaces/sodaspaces-terminal-actions.ts` — Existing explicit End confirmation, target-preserving HTTP action and unconfirmed-outcome handling.
- `frontend/spaces/sodaspaces-terminal-view.ts` — Reuse existing stateless view; absorb terminal presentation labels/menu projections rather than making another rendered terminal owner.

Evidence: sodaspaces-terminal.ts:54-55 immutable account/no rendering effects; 127-152 configure; 157-273 presentation/confirmation; 274-404 generation/detach/End; 405-508 send/geometry; 550-856 metadata/renderer/reservation/socket; 869-964 visibility/dispose/mount.

Open detail: There must still be one generation and one socket/screen owner. Preserve bounded queues, readiness gating, input focus and no replay; helper extraction must not duplicate those state machines.

### frontend/spaces/sodaspaces-workspace-view.ts

Observed size: 473 lines, including tests where embedded. Extract complete stateless Lit view functions by existing screen concern; keep view data as presentation-only projections.

- `frontend/spaces/sodaspaces-workspace-view.ts` — Welcome and workspace introduction, plus existing menu primitive.
- `frontend/spaces/sodaspaces-repository-picker-view.ts` — Existing repository search, choice, empty result and pagination presentation.
- `frontend/spaces/sodaspaces-session-navigation-view.ts` — Existing session tabs and project/session navigation rows.
- `frontend/spaces/sodaspaces-factory-navigation-view.ts` — Existing factory run rows and watched-run navigation presentation.
- `frontend/spaces/sodaspaces-terminal-dialog-view.ts` — Existing Rename/New terminal form presentation and callbacks.

Evidence: sodaspaces-workspace-view.ts:7-81 welcome/intro; 82-192 repository picker; 193-200 menu; 201-319 tabs/project navigation; 320-374 factory rows; 375-473 rename/create.

Open detail: Update importing owners directly; do not keep a forwarding export barrel solely to preserve old source imports.

### frontend/spaces/sodaspaces-workspace.ts

Observed size: 3125 lines, including tests where embedded. Split existing presentation and resource concerns while preserving one SodaSpaces component, one actor/generation/lifetime gate, and flat live terminal hosts. New/Rename and locator restoration form one current terminal concern; the mounted project-controls cache remains with the component that owns its actor and retirement.

- `frontend/spaces/sodaspaces-workspace.ts` — Single SodaSpaces actor/epoch/lifetime/availability and component-state owner, constructor/createRenderRoot and derived-state getters/projections, public mount, markViewed/live admission, invalidation/disposal, and existing mounted project-controls cache/presentation/event delegation. Keep managementAdmitted/mountProject/refreshMountedProject/showManagement/presentManagement together here; stateless helpers receive admitted current projections.
- `frontend/spaces/sodaspaces-workspace-types.ts` — Move current WorkspaceContext, Slot, Row, PaneSession, FactoryWatch and Creation records to their single shared definition.
- `frontend/spaces/sodaspaces-workspace-measurement.ts` — Existing WorkspaceMeasurement subscriptions and owner-triggered geometry/minimum measurements.
- `frontend/spaces/sodaspaces-workspace-shell-view.ts` — Current workspace frame, status banners, canvas/navigation visibility and first-terminal introductions.
- `frontend/spaces/sodaspaces-workspace-toolbar-view.ts` — Current native/page toolbar, project label, sidebar/back/drawer/management disclosures.
- `frontend/spaces/sodaspaces-workspace-setup.ts` — Existing repository search/cursors, welcome/configure journey, selected repository and cancel/back flow.
- `frontend/spaces/sodaspaces-workspace-navigation.ts` — Current authorized project/session rows, query filters, unread/attention ordering and explicit selectProject; original owner remains responsible for project selection, back and management admission.
- `frontend/spaces/sodaspaces-workspace-factory.ts` — Existing factory watch selection, immutable watch context, display and command routing.
- `frontend/spaces/sodaspaces-workspace-pane-view.ts` — Current pane chrome, keyed tabs, overflow/move/drop targets, pane switcher/menu/layout-help projections. Receive original-owner focus/split/maximize/consolidate callbacks; do not own layout state.
- `frontend/spaces/sodaspaces-workspace-focus.ts` — Current workspace key/click/menu/focus handling and session-switcher focus restoration.
- `frontend/spaces/sodaspaces-workspace-inventory.ts` — Existing bounded collection refresh/pagination, incomplete facts, slot metadata reconciliation and journey synchronization.
- `frontend/spaces/sodaspaces-workspace-layout.ts` — Existing projection/split/move/divider/sidebar keyboard operations, toggleSidebar/toggleMaximizedPane/consolidatePanes, and load/persist calls into the existing pure layout owner. Original owner performs admitted state/focus/menu updates.
- `frontend/spaces/sodaspaces-workspace-drawer.ts` — Current native-navigation handoff, exact original entry, safe repository path and stale-request checks.
- `frontend/spaces/sodaspaces-workspace-terminal-hosts.ts` — Existing visibleSlot projection, stable flat DOM host mounting, exact locator/metadata/observation routing and geometry display; original-owner epoch/lifetime callbacks admit all observations.
- `frontend/spaces/sodaspaces-workspace-terminals.ts` — One existing terminal concern: exact saved/existing locator restoration and selection, creation eligibility/context/form admission, deliberate New and Rename, destination-pane admission and uncertain result handling. Reuse existing dialog presentation functions; no invented session inventory or second actor/lifetime owner.

Evidence: sodaspaces-workspace.ts:60-139 current shared records -> workspace-types; 140-183 existing WorkspaceMeasurement -> workspace-measurement; 185-186 flat-host/no render-effects contract retained; 261-315 original binding/actor/epoch/lifetime/slots/watch/project cache state -> workspace.ts.; sodaspaces-workspace.ts:316-430 constructor/createRenderRoot, activeSurface/selectedSpace/setupScreen/welcomeScreen, pageReadyForFirstTerminal/spaceReadyForFirstTerminal/projectRunnable/firstTerminal/inventoryRecovery/workspaceIntro, projectState/projectStatus, selected/compact/projection/locator/paneMinimum/tabHeight remain original-owner construction and derived-state projections. Stateless view/layout helpers receive these currently admitted values; no additional module or state owner.; sodaspaces-workspace.ts:431-561 configure/measure/project event callbacks -> original owner plus existing measurement concern; 562-934 shell projections -> shell-view; 935-1078 toolbar projections -> toolbar-view; 1080-1295 repository search/setup/configure flow -> setup.; sodaspaces-workspace.ts:1296-1302 selectProject -> navigation through original-owner project/back/management callbacks; 1303-1315 toggleSidebar/toggleMaximizedPane/consolidatePanes -> layout; 1316-1361 showPaneSwitcher/onFocusPaneChange/renderPaneSwitcher/renderPaneLayoutExtras/renderPaneSplitHelp/renderPaneMenu/paneActions -> pane-view through original-owner layout/focus callbacks.; sodaspaces-workspace.ts:1362-1371 visibleSlot -> terminal-hosts; 1372-1385 markViewed remains original-owner epoch/readRequested/unread retirement; 1386-1574 row attention/navigation -> navigation; 1576-1677 watches -> factory; 1678-1880 pane chrome -> pane-view.; sodaspaces-workspace.ts:1882-1934 creationEligible/creationExplanation/creationContext/creationDisabled/onCreationEnvironment/onCreationName/cancelCreation/creationForm -> terminals; existing renderCreation receives presentation data and original-owner draft/create/focus callbacks. 1935-2037 key/click/menu/focus/back -> focus with original-owner state updates.; sodaspaces-workspace.ts:2038-2098 defaultTerminalName/creationSpace/pickCreationSpace/pageBlocksNewTerminal/newTerminalBlocked/focusCreationDialog/newTerminal -> terminals; 2099-2101 live remains original owner; 2102-2312 bounded collection reads/paging/reconciliation -> inventory.; sodaspaces-workspace.ts:2313-2334 loadLayout/persist -> layout; 2335-2392 native handoff -> drawer; 2394-2497 exact restoration/open/confirmed End -> terminals; 2499-2717 stable host mount/locator/observation/metadata/focus/geometry -> terminal-hosts; 2718-2855 arrange/focus/split/move/divider/sidebar/tab operations -> layout with original-owner admitted state updates.; sodaspaces-workspace.ts:2856-2944 createAdmitted/createSpaceReady/openCreatedSlot/createTerminal/openExistingBlocked/existingOrNewEntry/openExisting -> terminals; 2945-2999 managementAdmitted/mountProject/defaultManagementMode/refreshMountedProject/focusConfigureIfNeeded/showManagement/presentManagement remain original owner; 3000-3043 renameAdmitted/applyRenamedMetadata/rename -> terminals; 3044-3125 lifecycle/disposal/mount remain original owner..

Open detail: Private methods currently read and write the original class state: binding/actor/epoch/lifetime, slots/watches/projects/layout, creation/creating/renaming, selection/view and focus. File targets are proposed responsibility owners, not complete helper APIs. Extract stateless views and the existing measurement controller first; each later helper must identify the exact snapshot/read inputs, original-owner admission callbacks and returned observations/state changes, preserving current checks before and after await. Keep markViewed/live and mounted-project cache/lifecycle in the original owner. Do not implement mixins, forwarding barrels, duplicate controllers, a generic request bus, new restoration policy or alternative navigation. Concrete signatures and final line sizes require implementation review.

### frontend/tailnet/soda-tailnet-page.ts

Observed size: 836 lines, including tests where embedded. Separate already distinct native observation, host/enrollment actions and pure Lit sections; keep private secrets/drafts and authorization retirement in the original page owner.

- `frontend/tailnet/soda-tailnet-page.ts` — Single configured native transport, original authorization lifetime, pending confirmation, draft ownership, component mount and retirement.
- `frontend/tailnet/soda-tailnet-observation.ts` — Existing bounded request/refresh and admitted native settings reconciliation.
- `frontend/tailnet/soda-tailnet-actions.ts` — Existing explicit host/enrollment mutation selection, payload/result admission and unconfirmed outcome messaging.
- `frontend/tailnet/soda-tailnet-host-view.ts` — Existing status/auth link, exit-node preference choices, peer list and host controls.
- `frontend/tailnet/soda-tailnet-enrollment-view.ts` — Existing enrollment summary/form/admission presentation.
- `frontend/tailnet/soda-tailnet-confirmation-view.ts` — Existing pending confirmation/reconnect projection and consequence labels.

Evidence: soda-tailnet-page.ts:7-74 current helpers/types; 76-190 transport/state/lifetime; 191-286 observation; 287-457 mutations/confirmation; 458-558 handlers; 559-581 confirmation; 582-697 host view; 698-809 enrollment view; 810-836 composition/mount.

Open detail: All scope/revision/native authorization behavior remains existing. Do not create a second settings store, add refresh retries, or move secrets into presentation modules.

### internal/acceptance/developer_access_test.go

Observed size: 447 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/acceptance/developer_access_test.go` — Existing access input, user, UID, subnet and SSH options cases.
- `internal/acceptance/developer_access_request_test.go` — Private request/support file loading and refusal cases.
- `internal/acceptance/developer_access_journey_test.go` — Bound transport fixture and native access journey.

Evidence: TestValidateAccessUser at 58; TestAccessSSHOptions at 140; TestLoadAccessRequest at 276; TestRunDeveloperAccessValidation at 181; stubAccessTransport at 319; TestRunDeveloperAccessEndToEnd at 368.

### internal/acceptance/evidence.go

Observed size: 466 lines, including tests where embedded. Retire the obsolete
Evidence/Execute/Remote/Command closure and its tests. Its remaining production
use is PrivateFile, called by installed probes; preserve that real operation
in probe input support before removing the rest. No permanent Go process
package or replacement evidence facade is proposed.

- `internal/acceptance/installed.go` — Existing restricted regular-file input operation folded into its actual private-input caller; preserve the present mode, path and size checks and existing probe tests.

Evidence: CreateEvidence at 30; WriteJSON at 188; PublishObservation at 241; Hashes at 217; scanEvidenceBytes at 277; CheckSecrets at 327; PrivateFile at 454; RedactError at 361; redactingWriter at 370; redactPendingSecrets at 409.

Disposition: Rust acceptance is the retained harness. Remove the obsolete
Go release-build hashing import with Evidence; do not keep legacy machinery
to retain its unit tests.

### internal/acceptance/personal_git.go

Observed size: 487 lines, including tests where embedded. Keep the Go probe's
orchestration, exported contracts and execution order. The embedded remote
Python program is an implementation predecessor: replace it with the Rust
user-scoped payload in the existing acceptance package, keeping the actual
SSH login, passphrase custody and key/agent lifetime. Merely moving its string
to personal_git_keys.go would not complete the Python cutover.

- `internal/acceptance/personal_git.go` — Explicit prepare/exercise admission and orchestration.
- `internal/acceptance/personal_git_keys.go` — Passphrase custody and exported public-key checks.
- `internal/acceptance/personal_git_transport.go` — Target decoding, SSH invocation and exact Git URL checks.
- `internal/acceptance/personal_git_exercise.go` — Per-user clone/commit/push exercise and outcome records.
- `tools/acceptance/src/personal_git.rs` — Native replacement for the existing remote key/agent operation, called through the existing ephemeral acceptance payload; no new installed service.

Evidence: RunPersonalGit at 451; runPersonalGit at 458; preparePassfile at 176; gitKeyUser at 213; fetchExportedKey at 238; loadGitTarget at 82; gitSSHBase at 126; validateGitURL at 267; exerciseCommand at 316; exerciseUser at 381; writeGitOutcomes at 405.

Open detail: Update the real Go SSH invocation, Rust remote dispatch and
personal_git_test.go together. Keep secret inputs off argv and output; target
payload binding and ephemeral staging must follow the existing acceptance
delivery rather than requiring a target compiler or installing another agent.

### internal/factory/assignment.go

Observed size: 442 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/assignment.go` — Canonical assignment identity, stage and validation.
- `internal/factory/assignment_result.go` — Reported and synthesized harness results.
- `internal/factory/assignment_resources.go` — Canonical reservation and usage records.
- `internal/factory/dispatch_prompt.go` — Accepted input sections and bounded prompt construction.

Evidence: Assignment at 67; ValidAssignmentStage at 36; AssignmentResult at 184; ParseHarnessResult at 248; Reservation at 297; Usage at 326; PromptInputs at 359; BuildDispatchPrompt at 388.

### internal/factory/checks.go

Observed size: 480 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/checks.go` — Check verdicts, resolution reasons and target/adopted definitions.
- `internal/factory/checks_observation.go` — Observed context records and severity classification.
- `internal/factory/checks_assessment.go` — Canonical assessment validation and exact evidence verification.

Evidence: CheckTarget at 123; AdoptedChecks at 161; ObservedChecks at 212; checkStateSeverity at 279; CheckAssessment at 304; VerifyChecks at 406.

### internal/factory/control/acceptance.go

Observed size: 584 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/acceptance.go` — Explicit acceptance admission, immutable command replay and receipts.
- `internal/factory/control/acceptance_evidence.go` — Read and verify native acceptance evidence.
- `internal/factory/control/acceptance_initial.go` — Initial creator-authorized acceptance without overwrite.
- `internal/factory/control/acceptance_status.go` — Current validity assessment and explicit withdrawal.

Evidence: AdmitAcceptance at 125; replayAcceptance at 191; readAcceptanceEvidence at 218; verifyAcceptance at 246; AdmitInitialAcceptance at 322; acceptanceStatus at 410; assessAcceptance at 470; WithdrawAcceptance at 525.

### internal/factory/control/acceptance_test.go

Observed size: 472 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/acceptance_test.go` — Explicit admission and code-route refusal cases.
- `internal/factory/control/acceptance_initial_test.go` — Initial authority, non-overwrite and creator eligibility cases.
- `internal/factory/control/acceptance_status_test.go` — Current validity and withdrawal/replay cases.

Evidence: TestAdmitAcceptanceVerifiesAndRecords at 71; TestAdmitAcceptanceGuardsCodeRoutes at 180; TestAdmitInitialAcceptanceVerifiesCreation at 240; TestAdmitInitialAcceptanceCannotOverwrite at 312; TestAcceptanceStatusAssessesValidity at 329; TestWithdrawAcceptanceLatchesAndReplays at 441.

### internal/factory/control/checks_native_test.go

Observed size: 565 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/checks_native_test.go` — Native verdict, context and policy scenarios.
- `internal/factory/control/checks_native_fixture_test.go` — Shared exact native check fixture and status transport.
- `internal/factory/control/checks_native_stale_test.go` — Head/base/definition and workflow snapshot cases.

Evidence: TestNativeCheckPass at 283; TestNativeCheckUnknownState at 375; TestNativeCheckPolicyEmpty at 466; nativeCheckPublish at 115; nativePostStatus at 170; nativeVerifyChecks at 242; TestNativeCheckStaleHead at 409; TestNativeCheckStaleBase at 483; TestNativeCheckWorkflowPendingSnapshot at 504.

### internal/factory/control/dispatch.go

Observed size: 742 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch.go` — Dispatch dependency records, pass entry and coordinator hooks.
- `internal/factory/control/dispatch_occupancy.go` — Current held capacity and sponsorship occupancy.
- `internal/factory/control/dispatch_recovery.go` — Existing assignment recovery without launching fresh work.
- `internal/factory/control/dispatch_result.go` — Terminal outcome, exact harness result and usage accounting.

Evidence: DispatchDeps at 176; DispatchPass at 212; dispatchDeps at 703; passOccupancy at 256; snapshotOccupancy at 262; heldConn at 290; recoverAssigned at 306; recoverOne at 324; recoverSettled at 438; recoverUnused at 511; finishFromRun at 603; deriveAttemptResult at 638; AccountSettledRun at 672.

### internal/factory/control/dispatch_attempt.go

Observed size: 620 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch_attempt.go` — Attempt planning and existing readiness refusals.
- `internal/factory/control/dispatch_selection.go` — Sponsorship, resource limits, preparation and harness selection.
- `internal/factory/control/dispatch_inputs.go` — Current accepted native input reads and prompt sections.
- `internal/factory/control/dispatch_launch.go` — Fresh/retry execution, launch receipts and unused launch settlement.

Evidence: attemptPlan at 19; planAttempt at 121; selectSponsorship at 200; checkLimits at 228; selectPreparation at 277; checkHarness at 313; readAttemptInputs at 332; promptSections at 403; executeFreshAttempt at 424; retryAttempt at 500; launchTail at 565; settleUnusedLaunch at 603.

### internal/factory/control/dispatch_test.go

Observed size: 1236 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch_test.go` — Current queue ordering, short limit and required deps cases.
- `internal/factory/control/dispatch_fixture_test.go` — Shared host/broker/read fakes and canonical seed fixture.
- `internal/factory/control/dispatch_accounting_test.go` — Existing interrupt/fence/resource accounting cases.
- `internal/factory/control/dispatch_wait_test.go` — Wait reasons and explicit provider selection refusals.
- `internal/factory/control/dispatch_recovery_test.go` — Settled recovery, dependant reassessment and automatic intake.
- `internal/factory/control/dispatch_concurrency_test.go` — Existing simultaneous dispatch capacity/budget case.

Evidence: TestDispatchPassLaunchesOldestWithinShortLimit at 247; TestDispatchPassWithoutDepsReports at 1060; fakeDispatchHost at 26; fakeDispatchBroker at 61; dispatchSeed at 113; TestDispatchAccountingNeverRefreshes at 347; TestDispatchFencedLaunchStaysHeld at 544; TestAccountSettledRunResults at 838; TestDispatchWaitReasons at 574; TestDispatchNeverFallsBackToAnotherConnection at 787; TestRecoveryConsumesSettledRunWithoutHook at 929; TestCompletionTriggersDependantReassessment at 979; TestIntakeTriggersAutomaticDispatch at 1067; TestConcurrentDispatchPassesPreserveCapacityAndBudget at 1179.

### internal/factory/control/grants.go

Observed size: 502 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/grants.go` — Standing policy/capacity/operator/sponsorship command admission and replay.
- `internal/factory/control/preparation_decisions.go` — Requirement and approval admission with exact replay.
- `internal/factory/control/grant_authority.go` — Effective authority and preparation readiness reads.
- `internal/factory/control/dispatch_registration.go` — Explicit dispatch registration and reopening.

Evidence: ApplyPolicy at 49; ApplySponsorship at 87; applyGrant at 123; AdmitRequirement at 203; AdmitApproval at 221; admitDecision at 246; EffectiveAuthority at 339; preparationReadiness at 392; RegisterDispatch at 423; ReopenDispatch at 445.

### internal/factory/control/lifecycle.go

Observed size: 487 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/lifecycle.go` — Repository pause/resume and tracked run stop.
- `internal/factory/control/lifecycle_retry.go` — Explicit retry and replay.
- `internal/factory/control/lifecycle_takeover.go` — Human takeover and project start checks.

Evidence: PauseRepository at 33; ResumeRepository at 168; StopProject at 155; RetryRun at 251; replayRetry at 342; TakeoverRun at 357; VerifyProjectStart at 437.

### internal/factory/control/merge.go

Observed size: 642 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/merge.go` — Pass orchestration, report and persistence helpers.
- `internal/factory/control/merge_reconcile.go` — Existing conditional-operation reconciliation and evidence adoption.
- `internal/factory/control/merge_evidence.go` — Exact snapshot, authority and completion observation.
- `internal/factory/control/merge_withdraw.go` — Repository/acceptance cancellation and withdrawal ordering.

Evidence: MergePass at 76; mergeOne at 108; finishMerge at 629; reconcileMerge at 146; adoptMergeReceipt at 435; adoptMergeObserved at 592; observeMergeEvidence at 308; mergeAuthority at 380; completeMerge at 459; cancelMerges at 519; withdrawMerge at 555.

### internal/factory/control/merge_native_test.go

Observed size: 1408 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/merge_native_test.go` — Exact native wire, review and confirmation snapshots.
- `internal/factory/control/merge_native_fixture_test.go` — Native config/receipts and committed operation transport.
- `internal/factory/control/merge_native_setup_test.go` — Canonical assignment, approval, assessment and native merge drive.
- `internal/factory/control/merge_native_stale_test.go` — Current authority/head/base/withdrawal refusal cases.
- `internal/factory/control/merge_native_effect_test.go` — Cancel/lost-reply/incomplete-effect and protection cases.
- `internal/factory/control/merge_native_completion_test.go` — Complete merge and dependant readiness journeys.

Evidence: TestNativeMergeWire at 347; TestNativeMergeReviewSnapshot at 372; TestNativeMergeConfirmationSnapshot at 421; loadNativeST12 at 41; nativeMergeSubmitMergeCommitted at 200; nativeMergeAPI at 279; nativeMergeAssignment at 477; nativeMergeApprove at 539; nativeMergeDrive at 581; TestNativeMergeStaleHead at 771; TestNativeMergeStaleBase at 805; TestNativeMergeAuthorityChanged at 857; TestNativeMergeCancelAfterCommit at 988; TestNativeMergeLostReply at 1091; TestNativeMergeProtectionRefuses at 1350; TestNativeMergeFullPass at 666; TestNativeMergeDependantRunnable at 1254.

### internal/factory/control/merge_test.go

Observed size: 571 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/merge_test.go` — Exact merge pass, current check/review/authority cases.
- `internal/factory/control/merge_fixture_test.go` — Existing merger fake and canonical seed.
- `internal/factory/control/merge_effect_test.go` — Lost reply, completion, withdrawal and dependant cases.

Evidence: TestMergePassMergesOneExactPR at 180; TestMergePassRequiresCurrentAuthority at 435; happyMerger at 81; mergeSeed at 117; TestMergePassReconcilesLostSubmitReply at 208; TestMergePassWithdrawalAfterCommitConfirmsCompletion at 492; TestMergeCompletionReleasesCodeDependant at 526.

### internal/factory/control/publication.go

Observed size: 701 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication.go` — Pass orchestration, assignment lookup and persistence hooks.
- `internal/factory/control/publication_reconcile.go` — Existing branch/PR effects and receipt reconciliation.
- `internal/factory/control/publication_authority.go` — Current authority and exact native work construction.
- `internal/factory/control/publication_withdraw.go` — Cancellation, withdrawal and late observed-effect adoption.
- `internal/factory/control/publication_export.go` — Confirmed candidate export and bounded bundle decoding.

Evidence: PublishPass at 78; publishOne at 119; assignmentForPublication at 148; reconcilePublication at 163; adoptPublicationReceipts at 448; publicationAuthority at 388; publicationWork at 433; cancelPublications at 507; withdrawPublication at 576; adoptObserved at 620; exportCandidate at 671; decodeExportBundle at 688.

### internal/factory/control/publication_native_test.go

Observed size: 701 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication_native_test.go` — Native publish, correction and unexpected-ref scenarios.
- `internal/factory/control/publication_native_fixture_test.go` — Existing native config, Git/API and seed fixture.
- `internal/factory/control/publication_native_effect_test.go` — Lost branch/PR reply, withdrawal and partial committed effects.

Evidence: TestNativePublishCandidate at 365; TestNativePublishCorrection at 430; TestNativePublishWrongRefs at 519; loadNativeST09 at 53; nativeAPI at 90; nativeSeed at 223; TestNativePublishLostReplies at 562; TestNativePublishWithdrawal at 615; TestNativePublishBranchCommittedPRFailed at 683.

### internal/factory/control/publication_test.go

Observed size: 615 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication_test.go` — Exact PR link and source/ref refusal cases.
- `internal/factory/control/publication_fixture_test.go` — Existing publication fakes, native outcomes and seed.
- `internal/factory/control/publication_effect_test.go` — Lost reply, immutable replay and partial/fenced effects.
- `internal/factory/control/publication_recovery_test.go` — Late committed links, export recovery and unavailable executor.

Evidence: TestPublishPassLinksOneExactPR at 314; TestPublishPassRejectsChangedAcceptedSource at 413; publicationTestDB at 20; publishSeed at 167; happyPublisher at 262; TestPublishPassReconcilesLostSubmitReply at 333; TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent at 356; TestPublishPassFencesIndeterminateEffect at 494; TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion at 541; TestPublishPassRecoversCommittedPRWithoutExport at 577; TestPublishPassWaitsWithoutExecutor at 609.

### internal/factory/control/readiness_test.go

Observed size: 625 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/readiness_test.go` — Native event intake, duplicate observation and explicit adoption.
- `internal/factory/control/readiness_fixture_test.go` — Existing exact native-read fakes and seed inputs.
- `internal/factory/control/readiness_prerequisite_test.go` — Accepted code/result prerequisite outcome cases.
- `internal/factory/control/readiness_visibility_test.go` — Hidden/incomplete/cycle/withdrawn native evidence cases.

Evidence: TestObserveIssueEventCreationQueues at 152; TestObserveIssueEventDuplicateSuppresses at 178; TestEditedCreationNeedsExplicitAdoption at 258; readinessCoordinator at 103; readinessDecision at 95; readinessHint at 111; TestClosureAloneCannotSatisfyCodeOutcome at 296; TestResultPrereqNeedsAcceptedResolution at 382; TestHiddenEndpointBlocksWithoutLeaking at 412; TestLongerCycleBlocks at 471; TestWithdrawnEndpointInvalidatesDependent at 505.

### internal/factory/control/st15_demo_journey_test.go

Observed size: 694 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_journey_test.go` — One composed demo entry and final receipt.
- `internal/factory/control/st15_demo_coding_test.go` — Original coding dispatch, native publication and browser observations.
- `internal/factory/control/st15_demo_review_test.go` — Independent reviewer/fix/new-head/fresh-review sequence.
- `internal/factory/control/st15_demo_completion_test.go` — Native CI, merge/dependant and retention proof.

Evidence: TestST15ComposedDemo at 664; writeFinalReceipt at 651; dispatchA at 99; publishA at 193; runBrowser at 71; prepareReviewer at 238; reviewLeg1 at 326; correctA at 376; reviewLeg2 at 422; ciPass at 458; mergeAndDependants at 474; proveRetention at 590.

### internal/factory/control/st15_demo_native_test.go

Observed size: 548 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_native_test.go` — Existing fixture identity/configuration, receipts and command helpers.
- `internal/factory/control/st15_demo_stack_test.go` — Explicit task fixture host/broker/runtime stack setup and cleanup.
- `internal/factory/control/st15_demo_broker_test.go` — Current fixture broker build, tmpfs and socket wait.

Evidence: loadST15 at 69; st15Receipt at 96; st15Podman at 166; setupHostStack at 309; openFactoryStore at 536; st15BuildBroker at 261; st15TmpfsRoot at 277; st15WaitSocket at 295.

### internal/factory/control/st15_demo_seed_test.go

Observed size: 454 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_seed_test.go` — Native repository, source and issue/comment setup.
- `internal/factory/control/st15_demo_project_test.go` — Existing retained Project, role preparation and human work setup.
- `internal/factory/control/st15_demo_coordinator_test.go` — Original coordinator construction and native intake hints.

Evidence: seedRepository at 128; createIssue at 386; insertEdge at 430; setupProject at 63; prepareRoles at 169; seedHumanWork at 272; wireCoordinator at 323; postIntake at 361; issueOpenedHint at 436.

### internal/factory/grants.go

Observed size: 438 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/grants.go` — Standing grant identity, capacity and operator records.
- `internal/factory/sponsorship.go` — Provider sponsorship records and allowance validation.
- `internal/factory/effective_authority.go` — Evaluation of separately recorded authority inputs.

Evidence: SettingsCommandType at 63; SettingsDigest at 77; ValidTargetBranch at 129; Sponsorship at 240; EvaluateAuthority at 333.

### internal/factory/merge.go

Observed size: 592 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/merge.go` — Canonical merge stage, identity and aggregate record validation.
- `internal/factory/merge_operation.go` — Immutable conditional operation, work and native intent.
- `internal/factory/merge_evidence.go` — Exact check evidence and native observation validation.

Evidence: Merge at 213; ValidMergeStage at 31; MergeOperationID at 204; MergeIntent at 372; MergeWork at 348; VerifyMergeCheckEvidence at 490; MergeTargetChanged at 587.

### internal/factory/publication.go

Observed size: 736 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/publication.go` — Canonical publication aggregate, identity and outcome.
- `internal/factory/publication_operation.go` — Native effect/cancellation/completion and exact operation record.
- `internal/factory/publication_intent.go` — Immutable branch and PR creation intent binding.
- `internal/factory/publication_refusal.go` — Typed refusal and transport error classification.

Evidence: Publication at 290; ValidPublicationStage at 31; ValidOpEffect at 84; ValidOpCancellation at 105; PublicationOperationID at 270; PublicationWork at 480; PRTitleFor at 623; PRBodyFor at 629; Error at 682.

### internal/forgejo/background.go

Observed size: 450 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/background.go` — Existing background service/client API methods and callback contract.
- `internal/forgejo/background_admission.go` — Serialized bootstrap/admission custody and peer verification.
- `internal/forgejo/background_transport.go` — Bounded POST transport and exact operation lookup validation.

Evidence: ServiceBackground at 35; NewServiceBackground at 49; admissionForCall at 299; bootstrap at 313; verifiedPeer at 430; checkBackgroundRecord at 241; checkBackgroundLookup at 251; post at 383.

### internal/forgejo/merge.go

Observed size: 459 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/merge.go` — Conditional merge submit/lookup/cancel adapter.
- `internal/forgejo/merge_observation.go` — Native actor, target and complete check evidence mapping.
- `internal/forgejo/merge_completion.go` — Receipt adoption and confirmed native issue outcome.

Evidence: NewMerger at 30; SubmitMerge at 247; LookupOp at 276; CancelOp at 296; ObserveMerge at 59; matchMergeTarget at 94; matchMergeChecks at 170; AdoptMerge at 374; ObserveCompletion at 402; matchMergeConfirmation at 427.

### internal/forgejo/snapshot.go

Observed size: 716 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/snapshot.go` — Revision-bound snapshot aggregate and bracketed read.
- `internal/forgejo/snapshot_request.go` — Evidence-family identifiers and exact request validation.
- `internal/forgejo/snapshot_issue.go` — Issue, comment and dependency DTOs with existing validators.
- `internal/forgejo/snapshot_pull.go` — Pull, review, check and ref DTOs with existing validators.

Evidence: NativeSnapshot at 383; BracketedRead at 412; ValidateRequest at 124; ContentDigest at 204; validIssue at 516; validCommentPage at 538; validDependencyPage at 559; validPull at 577; validReviewPage at 596; validCheckSet at 614; validRefs at 635.

### internal/host/daemon.go

Observed size: 644 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: NewDaemon at 212; NewCompanionDaemon at 238; ServeHTTP at 570; LoadConfig at 55; validateRuntimeConfig at 104; validateMuseRuntime at 636; acquireAdmission at 257; validateNativeOperationRequest at 307; makeBodyDecoder at 329; dispatchCreate at 348; dispatchPrepare at 416; dispatchFactory at 457.

### internal/host/project/factory.go

Observed size: 837 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: OpenFactory at 56; HarnessPin at 76; Launch at 249; loadReceipt at 137; storeReceipt at 168; storeTombstone at 684; drive at 280; acquireRunLease at 346; consumeStart at 452; finishRun at 565; Stop at 611; stopTimedOut at 537; reconcileRunCredential at 712; Inspect at 743; Takeover at 776; recordStopOutcome at 810.

### internal/host/project/prepare.go

Observed size: 514 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Prepare at 187; prepareTargetReady at 68; mapPreparationState at 136; clonePreparationSource at 283; confirmPreparationHead at 313; verifyLauncherEnvironment at 342; resolvePreparationTools at 405; recordPreparationTools at 429; InspectPreparation at 442; StopPreparation at 456; HoldPreparation at 488.

### internal/host/publish/operation.go

Observed size: 769 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/publish/operation.go` — Conditional branch/PR submission, lookup/cancel and native outcome mapping.
- `internal/forgejo/publish/operation_observation.go` — Exact Git ref advertisements and bracketed observation.
- `internal/forgejo/publish/candidate_validation.go` — Existing candidate bundle and credential validation in temporary repository.
- `internal/forgejo/publish/operation_push.go` — Bounded credentialed Git push and observed-effect reconciliation.
- `internal/forgejo/publish/operation_receipts.go` — Strict branch/PR receipt decoding and expiry extraction.

Evidence: SubmitPublish at 194; SubmitPRCreate at 260; LookupOperation at 368; CancelOperation at 421; ObserveForPublish at 60; observeTips at 126; observeTip at 146; PrepareValidated at 529; verifyCandidateBundle at 558; validationVerdict at 576; PushBranch at 602; reconcilePush at 633; DecodePublishReceipt at 672; DecodePRCreateReceipt at 722; OperationNotAfter at 755.

### internal/host/publish/operation_test.go

Observed size: 574 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/publish/operation_test.go` — Exact intent mapping, submission/lookup/cancel and lost reply.
- `internal/forgejo/publish/operation_receipts_test.go` — Strict receipt scope and trailing-document refusal.
- `internal/forgejo/publish/operation_git_test.go` — Native advertisement and exact branch mutation cases.

Evidence: TestSubmitPublishMapsTerminalDispatchVerdicts at 84; TestSubmitPublishReconcilesLostReplies at 130; TestLookupAndCancelMapHonestly at 285; TestDecodePublishReceiptRefusesLookalikes at 323; TestDecodePRCreateReceiptRefusesLookalikes at 365; TestReceiptRejectsTrailingDocument at 561; TestPushBranchMovesExactlyItsTarget at 397; TestObserveTipParsesExactAdvertisement at 438.

### internal/host/tailnet/companion.go

Observed size: 730 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: validateCompanionRecord at 100; inspectCompanion at 133; companionCLI at 169; current at 196; saveCurrent at 218; consumeRunKey at 246; prepareCompanionState at 390; createFreshCompanion at 413; enrollCompanion at 482; StartTailnet at 522; StopTailnet at 571; logoutAndStopCompanion at 627; queueProjectTailnetDisable at 640; applyCompanionIdleState at 673; ObserveProjectTailnet at 705.

### internal/host/terminal/factory_codex_linux.go

Observed size: 539 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: FactoryCodexReserve at 111; FactoryCodexStart at 265; FactoryCodexWait at 311; factoryCodexSetup at 180; factoryCodexStage at 201; factoryCodexStageHost at 233; factoryUnitState at 54; factoryActiveInvocation at 62; factoryRoleIDs at 88; FactoryCodexValidate at 350; FactoryCodexStop at 379; FactoryCodexFinish at 465.

### internal/host/terminal/muse_execution_linux.go

Observed size: 439 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Start at 29; reserveExecution at 87; deliverExecution at 108; controlExecution at 137; stageConfig at 238; populateConfig at 251; copyNestedConfig at 385; museHostEnvironment at 414; stopExecution at 268; ValidateMuseBinding at 293; retireMount at 426.

### internal/host/terminal/muse_linux.go

Observed size: 436 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: resolveProject at 86; resolve at 161; projectAccount at 189; authorizedCaller at 231; RegisterNested at 265; registeredChild at 311; validateNested at 344; nestedNamespace at 359.

### internal/host/terminal_native_test.go

Observed size: 418 lines, including tests where embedded. Preserve the opt-in
native terminal marker/probe and journey assertions, rebinding its temporary
helper to the Rust host binary and actual native binding. The current fixture
constructs Go Daemon/Native/terminal.Service at 115-142 and cannot survive
executor deletion unchanged. Keep installed proof distinct from source tests.

- `internal/host/terminal_native_test.go` — Existing native terminal escape/control sequence probe.
- `internal/host/terminal_boundary_native_test.go` — Installed actor/terminal boundary journey.

Evidence: TestTerminalProbeNativeControlSequences at 38; TestInstalledTerminalBoundary at 48.

### internal/release/build/coreos_stream.go

Observed size: 518 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: resolveStreamBuild at 232; ResolveCoreOSISO at 359; ResolveCoreOSQEMU at 376; ResolveTailnetInputs at 68; latestTailnetRelease at 95; latestTailnetBaseTag at 122; fetchCappedJSON at 190; resolveRegistryDigests at 289; WriteLiveInputs at 438; ReadLiveInputs at 450; ValidLiveInputs at 462.

### internal/release/build/oci.go

Observed size: 711 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: InspectOCI at 175; InspectOCIContent at 475; openOCIArchive at 46; readOCIArchiveEntries at 144; readOCIIndex at 545; whiteoutTarget at 220; scanOCILayer at 307; scanOCIArchiveLayers at 399; parseOCIManifest at 603; validateOCIRootFS at 628; validateOCIAttribution at 644.

### internal/release/build/production.go

Observed size: 591 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Production at 24; step at 35; validate at 42; SodaCommands at 52; CompileRust at 77; Compile at 108; Assets at 127; Dependencies at 156; assetSteps at 178; admitResolvedInputRecord at 276; pullResolvedInput at 285; ResolveInputs at 349; buildImage at 376; exportAppImages at 448; exportImages at 538.

### internal/release/deliver/model.go

Observed size: 570 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-deliver`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Hash at 29; Digest at 30; admitTrustRoleKeys at 73; Role at 107; ValidCandidateContent at 156; requiredCandidateContent at 179; MediaBinding at 230; References at 334; EmptyState at 368; AdmitChannel at 483; AdmitRelease at 535.

### internal/release/image/assemble.go

Observed size: 736 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-image`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: assembleMedia at 585; admitMediaInputs at 273; fetchAssemblerConfig at 72; verifyAssemblerLayers at 119; prepareAssembler at 168; collectMediaInventory at 288; verifyMediaInventory at 327; authenticatePackagingInputs at 337; buildMediaContainer at 375; assembleNativeMedia at 387; verifyBuildMeta at 429; customizeInstallerISO at 483; verifyMediaReadback at 508; sealMedia at 545; VerifyLiveIgnition at 651; VerifyRootfsChunks at 711.

### internal/release/image/build.go

Observed size: 671 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-image`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: ValidateTarget at 65; Build at 148; runBuild at 571; verifyCheckoutSource at 198; extractBuildSnapshot at 300; setupBuildWorkspace at 315; compileSodaCommands at 405; compileRustTools at 463; compileShippingTools at 476; runBuildCommand at 623; linkPreparedAssets at 658; failureReason at 184.

### internal/store/factory_assignments.go

Observed size: 504 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_assignments.go` — Canonical assignment reads, queries and finishing.
- `internal/store/factory_dispatch_packet.go` — Atomic first dispatch packet and current admission/capacity check.
- `internal/store/factory_retry_packet.go` — Atomic retry packet under the same transaction owner.
- `internal/store/factory_dispatch_queue.go` — Queued controls and active run counts.

Evidence: Assignment at 51; IssueAssignments at 268; FinishAssignment at 401; RecordDispatchPacket at 51; checkAdmissionTx at 130; RecordRetryPacket at 298; QueuedControls at 457; ActiveRunCounts at 486.

### internal/store/factory_dispatch_test.go

Observed size: 459 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_dispatch_test.go` — Dispatch packet, existing fixture and limit enforcement.
- `internal/store/factory_retry_packet_test.go` — Retry packet bounds and resource reholding.
- `internal/store/factory_dispatch_queue_test.go` — Reservation/usage/queue/current-count scenarios.

Evidence: dispatchStoreFixture at 16; TestRecordDispatchPacket at 88; TestRecordDispatchPacketEnforcesLimits at 203; TestRecordRetryPacketBoundsAttemptsAndFinishes at 138; TestRecordRetryPacketReholdsAndRefusesLimits at 259; TestReservationTransitions at 304; TestRunUsageFirstWriteWins at 349; TestQueuedControlsOldestFirst at 395.

### internal/store/factory_publications_test.go

Observed size: 435 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_publications_test.go` — Record round-trip, CAS and selectable assignment/record cases.
- `internal/store/factory_publication_intent_test.go` — Immutable intent/receipt and withdrawal/authority registration ordering.

Evidence: TestRecordPublicationRoundTrip at 71; TestUpdatePublicationComparesAndSwaps at 93; TestPublishableAssignmentsSelectsReportedCompleted at 165; TestPublicationRegistrationPersistsImmutableIntent at 244; TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation at 294; TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight at 370.

### internal/tailnet/control.go

Observed size: 651 lines, including tests where embedded.

Disposition: retire native Go Control execution and its server-only tests at
host cutover. The existing protected policy/provider/enrollment/host-action
behavior belongs to `lib/host/src/tailnet/control/`, with its actual tests.
Retain Go request/response validators and read-only clients; this source is
not a new Go split owner in the target.

Evidence: NewControl at 40; NewProjectControl at 53; RunNative at 117; RoundTrip at 84; request at 130; checkCredential at 629; fetchNativeStatus at 264; populateHostPreferences at 296; observe at 356; HostAction at 427; executeSignin at 490; executeExitNode at 544; verifyHostActionOutcome at 591.

### internal/tailnet/control_test.go

Observed size: 495 lines, including tests where embedded.

Disposition: retire native Go Control execution and its server-only tests at
host cutover. The existing protected policy/provider/enrollment/host-action
behavior belongs to `lib/host/src/tailnet/control/`, with its actual tests.
Retain Go request/response validators and read-only clients; this source is
not a new Go split owner in the target.

Evidence: TestTailnetHostProjectionAndPassiveReads at 51; TestTailnetHostNodeKeyOptionalButStrict at 97; TestTailnetInitialLoginUsesBoundedUpNotReset at 228; TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration at 255; TestTailnetOfflineExitNodeAndUnconfirmedClear at 392; TestTailnetBoundedOutputChild at 481; TestTailnetCommandOutputBounds at 488.

### internal/web/api/extension_terminal.go

Observed size: 411 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/extension_terminal.go` — Existing extension terminal route/session operations.
- `internal/web/api/extension_terminal_authority.go` — Original actor/account/membership and contribution binding.
- `internal/web/api/extension_terminal_stream.go` — Exact generation handshake, websocket attach and control pump.

Evidence: ExtensionHandler at 24; extensionTerminalSession at 243; extensionTerminalAccount at 87; extensionTerminalCurrent at 138; extensionTerminalAttach at 296; validNativeTerminalHandshake at 353; pumpExtensionControls at 387.

### internal/web/api/factory_settings.go

Observed size: 499 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/factory_settings.go` — Common native actor/repository/operator admission and errors.
- `internal/web/api/factory_status.go` — Current factory and preparation status projection.
- `internal/web/api/factory_policy.go` — Policy, capacity and operator grant routes.
- `internal/web/api/factory_sponsorship.go` — Environment/provider grant routes and broker sponsorship check.

Evidence: factoryRepository at 39; factoryOperator at 76; settingsCommandID at 86; apiFactoryStatus at 128; factoryPreparationStatus at 195; apiFactoryPolicy at 260; apiFactoryCapacity at 314; apiFactoryOperatorGrant at 349; apiFactoryEnvironmentGrant at 382; apiFactorySponsorship at 434; checkSponsorshipBroker at 467.

### internal/web/api/issue_acceptances.go

Observed size: 479 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/issue_acceptances.go` — Explicit acceptance and withdrawal routes.
- `internal/web/api/issue_acceptance_evidence.go` — Authorized native acceptance snapshots and refusal mapping.
- `internal/web/api/factory_issue_view.go` — Current issue/check/merge response projection.

Evidence: apiIssueAcceptances at 159; apiIssueWithdrawal at 247; ReadAcceptanceEvidence at 32; mapAcceptanceSnapshotError at 97; checksViewDTO at 326; mergeViewDTO at 358; apiFactoryIssue at 372.

### internal/web/api/spaces.go

Observed size: 404 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/spaces.go` — Inventory pagination, session checks and Spaces response entry.
- `internal/web/api/spaces_inspection.go` — Current per-project native/terminal/Tailnet/run projection.
- `internal/web/api/spaces_authority.go` — Existing authority and allowed-control projection.

Evidence: inspectSpaces at 284; apiSpaces at 361; parseSpacesCursor at 391; inspectSpaceNative at 78; inspectSpaceTerminals at 91; inspectSpaceTailnet at 118; inspectSpaceFactoryRuns at 186; resolveSpaceAuthority at 69; inspectSpaceAuthority at 223; inspectSpaceControl at 245.

### internal/web/factory_views_test.go

Observed size: 439 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/factory_views_test.go` — Current run status and Spaces inventory response cases.
- `internal/web/factory_output_fixture_test.go` — Existing output proxy and handshake fixtures.
- `internal/web/factory_output_stream_test.go` — Bound output/status/EOF and stale/unauthorized/input refusal cases.

Evidence: TestFactoryRunStatusPendingLiveAndExcerpt at 63; TestSpacesShowsFactoryRuns at 168; factoryOutputProxyFixture at 206; dialFactoryOutput at 252; factoryOutputHandshakeFor at 290; TestFactoryOutputStreamDeliversStatusOutputAndEOF at 294; TestFactoryOutputStreamRefusesStaleHandshake at 377; TestFactoryOutputStreamRequiresWrite at 419.

### project-os/rootfs/usr/libexec/soda/project-factory-roles

Observed size: 800 lines, including tests where embedded. Replace this Python
helper in the existing project-terminal Cargo package. Separate its current
role/layout, approved input custody, receipt observation and process lifecycle
concerns without widening the privileged protocol or adding a daemon.

- `cmd/soda-project-terminal/src/bin/project-factory-roles.rs` — Thin entrypoint; compiled installation remains `/usr/libexec/soda/project-factory-roles`.
- `cmd/soda-project-terminal/src/factory_roles/mod.rs` — Bounded stdin, exact operation dispatch and lock scope.
- `cmd/soda-project-terminal/src/factory_roles/layout.rs` — Fixed paths, caps, identifiers, roles and root-owned file/layout admission (25-121).
- `cmd/soda-project-terminal/src/factory_roles/accounts.rs` — Fixed coder/reviewer lookup, creation and restricted homes (121-165).
- `cmd/soda-project-terminal/src/factory_roles/inputs.rs` — Approved digest, Git bundle, credential-reference custody and immutable snapshot creation (176-325).
- `cmd/soda-project-terminal/src/factory_roles/records.rs` — Verified receipts and log/phase/status/inspect observations (326-377, 541-623).
- `cmd/soda-project-terminal/src/factory_roles/execution.rs` — Role/environment execution, detached preparation, process Start/Stop, hold/release and confirmed ownership (378-540, 624-800).
- `cmd/soda-project-terminal/src/factory_roles/tests.rs` — Existing helper behavior tested against the actual native implementation; no imported Python subject.

Evidence: project-factory-roles:1-6 fixed contract; 25-121 fixed paths/admission/layout; 121-165 roles; 176-325 approved inputs/bundle/credential/snapshot; 326-377 verified records; 378-540 execution; 541-623 inspect; 624-773 process stop/hold/release; 774-800 main.

Open detail: Declare the two helper binaries and private shared library in the
existing project-terminal manifest. Install the compiled helper root-owned at
its existing path, and update project_factory_roles_test.go and the ST15 native
copy/hash at st15_demo_native_test.go:321-325,371-381 to stage/verify those real
executing bytes. The old Python source and import driver both retire.

### rust/identity-providers/src/codex.rs

Observed size: 702 lines, including tests where embedded. Separate the existing pinned-provider/process/enrollment lifecycle from app-server request/reply dispatch and notification parsing. Keep Config admission, environment filtering, binary digest and version checks together in codex_config.rs. codex.rs retains Provider and Session ownership, process stop before credential reading, and private tmpfs cleanup. Move the embedded tests without changing their fixtures. All files stay in the same provider crate; no Soda grant policy moves into this provider.

- `cmd/soda-identity/src/providers/codex/mod.rs`
- `cmd/soda-identity/src/providers/codex/protocol.rs`
- `cmd/soda-identity/src/providers/codex/config.rs`
- `cmd/soda-identity/src/providers/codex/tests.rs`

Evidence: 21-157: Config/Provider, new/start/create_session and split_env -> codex.rs and codex_config.rs; 161-242,304-397: Inner/Session, snapshot/cancel, device enrollment, finish/account/credential_file/stop/close -> codex.rs; 180-190,243-302,400-500: Message, send/call/protocol_error/read_loop/notify -> codex_protocol.rs; 502-565: environment/filter_env/validate_config/check_binary/check_version -> codex_config.rs; 566-702: environment/config fixtures, managed enrollment, cancellation, disabled-device tests -> codex_tests.rs.

### rust/identity-providers/src/muse.rs

Observed size: 561 lines, including tests where embedded. The production provider is 347 lines and owns one native enrollment lifecycle. Its process, prompt recognition, immutable subscription validation and private credential-file checks are cohesive; do not invent additional provider layers merely because tests bring the whole file to 561 lines. Extract the 213-line test module, keeping its real CLI-script fixtures and its existing environment/config/digest/version assertions.

- `cmd/soda-identity/src/providers/muse/mod.rs`
- `cmd/soda-identity/src/providers/muse/tests.rs`

Evidence: 22-163: Config/Provider/Inner/Session, native login process, finish and confirmed close; 165-242: bounded prompt read_loop/complete/scan_device_url and fixed environment; 244-346: pinned configuration/binary/version checks plus credential_valid/credential_file; 348-561: native_subscription_and_private_file, environment/device prompt, digest/version, presentation-private fixtures -> muse_tests.rs.

### rust/soda-acceptance/src/command.rs

Observed size: 550 lines, including tests where embedded. Separate pinned SSH argument construction from redacted command execution and its distinct execution/evidence outcomes. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/command/mod.rs`
- `tools/acceptance/src/command/ssh.rs`
- `tools/acceptance/src/command/execute.rs`
- `tools/acceptance/src/command/tests.rs`

Evidence: rust/soda-acceptance/src/command.rs:72-230 Remote, decode_remote, Remote::args and Remote::command; rust/soda-acceptance/src/command.rs:297-399 execute; rust/soda-acceptance/src/command.rs:402-550 tests for quoting, SSH pins, cancellation and outcome separation.

### rust/soda-acceptance/src/coreos.rs

Observed size: 503 lines, including tests where embedded. Keep the connected CoreOS resolution closure together; its bounded curl transport, stable stream and registry confirmation are one VM-base resolution operation. Extract the six existing unit tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/coreos.rs`
- `tools/acceptance/src/coreos/tests.rs`

Evidence: rust/soda-acceptance/src/coreos.rs:117-203 fetch_capped; rust/soda-acceptance/src/coreos.rs:261-284 resolve_stream_build; rust/soda-acceptance/src/coreos.rs:304-400 resolve_registry_digest and resolve_qemu; rust/soda-acceptance/src/coreos.rs:404-503 bounded-source and document tests.

### rust/soda-acceptance/src/driver.rs

Observed size: 1047 lines, including tests where embedded. Separate option/duration parsing, private input collection, action dispatch and observation finalization; keep run as the entry orchestration. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/driver/mod.rs`
- `tools/acceptance/src/driver/options.rs`
- `tools/acceptance/src/driver/inputs.rs`
- `tools/acceptance/src/driver/actions.rs`
- `tools/acceptance/src/driver/finalization.rs`
- `tools/acceptance/src/driver/tests.rs`

Evidence: rust/soda-acceptance/src/driver.rs:62-392 parse_duration, parse_flags and parse_run_options; rust/soda-acceptance/src/driver.rs:393-434 secret and VM input collection; rust/soda-acceptance/src/driver.rs:489-715 execute_exec_or_native, execute_vm and execute_action; rust/soda-acceptance/src/driver.rs:717-830 observation finalization and run; rust/soda-acceptance/src/driver.rs:861-1047 duration, admission and cancelled-execution tests.

### rust/soda-acceptance/src/evidence.rs

Observed size: 909 lines, including tests where embedded. Separate confined evidence creation/publication from streaming secret and URL redaction; keep the streaming state machine intact. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/evidence/mod.rs`
- `tools/acceptance/src/evidence/store.rs`
- `tools/acceptance/src/evidence/redaction.rs`
- `tools/acceptance/src/evidence/tests.rs`

Evidence: rust/soda-acceptance/src/evidence.rs:34-271 create_evidence and Evidence methods; rust/soda-acceptance/src/evidence.rs:341-368 scan_evidence_bytes; rust/soda-acceptance/src/evidence.rs:369-646 URL sanitization and RedactingWriter; rust/soda-acceptance/src/evidence.rs:648-909 retention, split-secret and byte-fidelity tests.

### rust/soda-acceptance/src/files.rs

Observed size: 696 lines, including tests where embedded. Keep descriptor-confined directory traversal as one OwnedDir implementation; separate private file/hash reads and temporary/exclusive destinations. Preserve descriptor identity and symlink rules. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/files/mod.rs`
- `tools/acceptance/src/files/owned_directory.rs`
- `tools/acceptance/src/files/inputs.rs`
- `tools/acceptance/src/files/temporary.rs`
- `tools/acceptance/src/files/tests.rs`

Evidence: rust/soda-acceptance/src/files.rs:111-370 OwnedDir and its fd-relative operations; rust/soda-acceptance/src/files.rs:372-437 private_file, hash_file and hash_at; rust/soda-acceptance/src/files.rs:438-522 fresh_directory, private_destination, TempDir and write_new; rust/soda-acceptance/src/files.rs:525-696 filesystem confinement and input tests.

### rust/soda-acceptance/src/host_probes.rs

Observed size: 939 lines, including tests where embedded. Put the already distinct installed probe commands in their own modules; retain common failure, command and JSON emission helpers locally. These remain installed host observations, not the outside driver. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/host_probes/mod.rs`
- `tools/acceptance/src/host_probes/content.rs`
- `tools/acceptance/src/host_probes/listeners.rs`
- `tools/acceptance/src/host_probes/deployments.rs`
- `tools/acceptance/src/host_probes/tailnet.rs`
- `tools/acceptance/src/host_probes/forgejo.rs`
- `tools/acceptance/src/host_probes/tests.rs`

Evidence: rust/soda-acceptance/src/host_probes.rs:184-413 host_content and extension package checks; rust/soda-acceptance/src/host_probes.rs:415-678 listener parsing and host_listeners; rust/soda-acceptance/src/host_probes.rs:680-731 host_deployments and operator_tailscale; rust/soda-acceptance/src/host_probes.rs:733-819 Forgejo origins, tailnet and advertisement probes; rust/soda-acceptance/src/host_probes.rs:821-939 probe parser/summary tests.

### rust/soda-acceptance/src/jsonio.rs

Observed size: 511 lines, including tests where embedded. Separate RFC3339 clock parsing/formatting from bounded JSON reads and exact JSON emission. Both are current behavior; no new serialization layer is proposed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/jsonio.rs`
- `tools/acceptance/src/timestamps.rs`
- `tools/acceptance/src/jsonio/tests.rs`
- `tools/acceptance/src/timestamps/tests.rs`

Evidence: rust/soda-acceptance/src/jsonio.rs:25-96 bounded regular JSON reads; rust/soda-acceptance/src/jsonio.rs:98-268 typed fields and compact/indented emission; rust/soda-acceptance/src/jsonio.rs:270-412 RFC3339 date conversion and validation; rust/soda-acceptance/src/jsonio.rs:422-509 byte-emission, date and strict-read tests.

### rust/soda-acceptance/src/native_phase.rs

Observed size: 654 lines, including tests where embedded. Retain the current native prepare/build/check admission and receipt sequence as one cohesive operation; extract its fake-runner fixtures and tests. The existing build phase admits candidate bytes and does not run a producer. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/native_phase.rs`
- `tools/acceptance/src/native_phase/tests.rs`

Evidence: rust/soda-acceptance/src/native_phase.rs:179-250 phase_request admission; rust/soda-acceptance/src/native_phase.rs:311-445 prepare_checkout, phase_command and run_phase; rust/soda-acceptance/src/native_phase.rs:557-588 explicit_phase_order_without_automatic_work; rust/soda-acceptance/src/native_phase.rs:448-654 native phase test module.

### rust/soda-acceptance/src/process.rs

Observed size: 739 lines, including tests where embedded. Separate cancellable Phase context from child launch/pumps and the owned process-group wait/reap/stop lifecycle. Keep the leader-unreaped/PGID ownership sequence together. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/process/mod.rs`
- `tools/acceptance/src/process/phase.rs`
- `tools/acceptance/src/process/launch.rs`
- `tools/acceptance/src/process/owned_process.rs`
- `tools/acceptance/src/process/tests.rs`

Evidence: rust/soda-acceptance/src/process.rs:28-104 Phase and cancellation/deadline methods; rust/soda-acceptance/src/process.rs:144-282 process launch and stdout/stderr pumps; rust/soda-acceptance/src/process.rs:283-442 Linux non-reaping/group cleanup; rust/soda-acceptance/src/process.rs:452-605 Process methods and signal_group; rust/soda-acceptance/src/process.rs:608-739 cancellation and resistant-descendant tests.

### rust/soda-acceptance/src/project_state.rs

Observed size: 943 lines, including tests where embedded. Separate bounded command capture, filesystem-entry/traversal snapshots, account/Git state collection and the explicitly requested workload/database observations. Preserve one run_snapshot result shape and its existing gates. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/project_state/mod.rs`
- `tools/acceptance/src/project_state/command.rs`
- `tools/acceptance/src/project_state/files.rs`
- `tools/acceptance/src/project_state/snapshot.rs`
- `tools/acceptance/src/project_state/workloads.rs`
- `tools/acceptance/src/project_state/tests.rs`

Evidence: rust/soda-acceptance/src/project_state.rs:165-240 command capture and output_lines; rust/soda-acceptance/src/project_state.rs:242-472 Entry, sorted JSON and traversal; rust/soda-acceptance/src/project_state.rs:499-672 root/account/Git/shared-file observations; rust/soda-acceptance/src/project_state.rs:673-819 declared workload, volume and native database observations; rust/soda-acceptance/src/project_state.rs:822-943 entry, command, gate and network tests.

### rust/soda-acceptance/src/report.rs

Observed size: 665 lines, including tests where embedded. Separate observation data/JSON conversion from retained-file hash verification and handoff rendering; keep incomplete/failed scope reporting. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/report/mod.rs`
- `tools/acceptance/src/report/observation.rs`
- `tools/acceptance/src/report/handoff.rs`
- `tools/acceptance/src/report/tests.rs`

Evidence: rust/soda-acceptance/src/report.rs:77-294 Observation wire fields and JSON conversion; rust/soda-acceptance/src/report.rs:296-381 evidence hashes and read_observation; rust/soda-acceptance/src/report.rs:383-500 handoff admission/rendering; rust/soda-acceptance/src/report.rs:503-665 round-trip and unsafe-reference tests.

### rust/soda-acceptance/src/trust.rs

Observed size: 506 lines, including tests where embedded. Separate bounded Ignition inline-data decoding from the existing fixture SSH host-key trust comparison. Keep all current encoding and key pin semantics. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/trust/mod.rs`
- `tools/acceptance/src/trust/inline_data.rs`
- `tools/acceptance/src/trust/host_key.rs`
- `tools/acceptance/src/trust/tests.rs`

Evidence: rust/soda-acceptance/src/trust.rs:31-217 base64/data URI/gzip and IgnitionFile decoding; rust/soda-acceptance/src/trust.rs:223-332 Ignition parse and fixture pinned-host-key verification; rust/soda-acceptance/src/trust.rs:334-506 codec, bounded compression and trust-mismatch tests.

### rust/soda-acceptance/src/vm.rs

Observed size: 944 lines, including tests where embedded. Separate VM configuration/preflight, verified base receipt, owned work/launch preparation and the VM readiness/restart/shutdown lifecycle. Keep failure ownership and retained-work reporting. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/vm/mod.rs`
- `tools/acceptance/src/vm/config.rs`
- `tools/acceptance/src/vm/base.rs`
- `tools/acceptance/src/vm/launch.rs`
- `tools/acceptance/src/vm/lifecycle.rs`
- `tools/acceptance/src/vm/tests.rs`

Evidence: rust/soda-acceptance/src/vm.rs:30-257 VmConfig/RemoteConfig decoding and preflight; rust/soda-acceptance/src/vm.rs:259-341 VerifiedBase and launch receipt verification; rust/soda-acceptance/src/vm.rs:343-480 QEMU checks, work preparation and owned arguments; rust/soda-acceptance/src/vm.rs:483-758 Vm, launch_vm, readiness and shutdown; rust/soda-acceptance/src/vm.rs:761-944 config, pin, disjointness and QMP argument tests.

### rust/soda-activate/src/main.rs

Observed size: 1374 lines, including tests where embedded. Make main a thin entry; use the already present CliArgs, Paths/Sys, origin/IP admission, activation operation and Forgejo environment rewrite boundaries. Group existing unit tests by those concerns. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-activate/src/main.rs`
- `cmd/soda-activate/src/cli.rs`
- `cmd/soda-activate/src/origin.rs`
- `cmd/soda-activate/src/system.rs`
- `cmd/soda-activate/src/activation.rs`
- `cmd/soda-activate/src/forgejo_env.rs`
- `cmd/soda-activate/src/tests/mod.rs`
- `cmd/soda-activate/src/tests/fixtures.rs`
- `cmd/soda-activate/src/tests/activation.rs`
- `cmd/soda-activate/src/tests/origin.rs`
- `cmd/soda-activate/src/tests/cli.rs`

Evidence: rust/soda-activate/src/main.rs:75-186 CliArgs and parse_args; rust/soda-activate/src/main.rs:188-279 Paths, ActivateError and Sys/RealSys; rust/soda-activate/src/main.rs:289-513 IP/origin admission; rust/soda-activate/src/main.rs:515-775 activate; rust/soda-activate/src/main.rs:794-886 rewrite_forgejo_env; rust/soda-activate/src/main.rs:888-1374 fixtures and activation/origin/CLI tests.

### rust/soda-asset-fetchers/src/muse.rs

Observed size: 521 lines, including tests where embedded. Keep the 222-line pinned Muse fetch/stage path together; most of this file is test fixtures and negative cases. Extract the descendant tests rather than split the small fetcher into extra production modules. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/fetch/muse.rs`
- `tools/release-assets/src/fetch/muse/tests.rs`

Evidence: rust/soda-asset-fetchers/src/muse.rs:68-120 load_release; rust/soda-asset-fetchers/src/muse.rs:151-220 streaming stage and fetch_muse; rust/soda-asset-fetchers/src/muse.rs:224-521 manifest, integrity, output and transport tests.

### rust/soda-asset-fetchers/src/tea.rs

Observed size: 542 lines, including tests where embedded. Keep the 163-line Tea fetch/license staging path intact; extract its 377 lines of fixtures and integrity/exclusivity cases. No new fetcher framework is needed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/fetch/tea.rs`
- `tools/release-assets/src/fetch/tea/tests.rs`

Evidence: rust/soda-asset-fetchers/src/tea.rs:75-162 upstream tag/checksum/ELF/license fetch and staging; rust/soda-asset-fetchers/src/tea.rs:165-228 local test fixtures; rust/soda-asset-fetchers/src/tea.rs:230-542 integrity, output and upstream refusal tests.

### rust/soda-candidate-setup/src/main.rs

Observed size: 1558 lines, including tests where embedded. Extract the existing setup stages and subprocess/status helpers; keep their current order and task-owned cleanup in one small orchestration. Fixture authority stays separate from release keys. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/candidate-setup/src/main.rs`
- `tools/candidate-setup/src/process.rs`
- `tools/candidate-setup/src/preflight.rs`
- `tools/candidate-setup/src/storage.rs`
- `tools/candidate-setup/src/controller.rs`
- `tools/candidate-setup/src/worker_tools.rs`
- `tools/candidate-setup/src/worker_caches.rs`
- `tools/candidate-setup/src/selinux.rs`
- `tools/candidate-setup/src/fixture_authority.rs`
- `tools/candidate-setup/src/config.rs`
- `tools/candidate-setup/src/tests.rs`

Evidence: rust/soda-candidate-setup/src/main.rs:399-470 worker/trust/config JSON emission; rust/soda-candidate-setup/src/main.rs:496-603 active-build refusal, storage migration and SELinux helpers; rust/soda-candidate-setup/src/main.rs:605-745 input/native/toolchain admission and lease; rust/soda-candidate-setup/src/main.rs:755-818 build/admit controller and wrapper; rust/soda-candidate-setup/src/main.rs:820-949 worker directories and tool installation; rust/soda-candidate-setup/src/main.rs:950-1084 cache warming; rust/soda-candidate-setup/src/main.rs:1086-1166 worker SELinux installation; rust/soda-candidate-setup/src/main.rs:1191-1309 restricted worker configuration and fixture authority; rust/soda-candidate-setup/src/main.rs:1375-1558 helper unit tests.

Open detail: The long run_setup body currently interleaves local variables and ordered system mutations. Extract stage functions without a generic workflow/state machine; current privileged path is source-inspected, not executed here.

### rust/soda-candidate-setup/tests/cli.rs

Observed size: 463 lines, including tests where embedded. Keep this coherent pre-mutation admission suite together after extracting existing fixture/fake-tool/lease helpers. It explicitly does not prove the later privileged setup flow.

- `tools/candidate-setup/tests/cli.rs`
- `tools/candidate-setup/tests/support/mod.rs`

Evidence: rust/soda-candidate-setup/tests/cli.rs:1-5 pre-mutation-only scope comment; rust/soda-candidate-setup/tests/cli.rs:20-147 TempDir, command isolation, fake tools and HeldLock; rust/soda-candidate-setup/tests/cli.rs:149-463 source/toolchain/lease/active-build refusal tests.

### rust/soda-console-welcome/src/main.rs

Observed size: 790 lines, including tests where embedded. Separate operator observation/banner composition from its top-level config parser and display-origin validation. Keep read-only behavior and exactly the existing permissive Python-like JSON cases. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-console-welcome/src/main.rs`
- `cmd/soda-console-welcome/src/welcome.rs`
- `cmd/soda-console-welcome/src/config.rs`
- `cmd/soda-console-welcome/src/origin.rs`
- `cmd/soda-console-welcome/src/tests.rs`

Evidence: rust/soda-console-welcome/src/main.rs:19-182 root gate, observed uplinks, banner and subprocess helpers; rust/soda-console-welcome/src/main.rs:184-240 render_config; rust/soda-console-welcome/src/main.rs:245-578 JsonParser and top-object extraction; rust/soda-console-welcome/src/main.rs:581-707 listen/origin validation; rust/soda-console-welcome/src/main.rs:713-733 duplicate-key/string/NaN/Infinity tests.

Open detail: This parser accepts NaN and Infinity while other Rust JSON readers use different rules. Consolidating parsers or replacing its dependency-free implementation needs contract reconciliation, not a file move.

### rust/soda-factory/src/main.rs

Observed size: 593 lines, including tests where embedded. The operator CLI has 351 lines of production code and no coordinator or database. Retain the existing argument parsing, command envelope, Unix HTTP send and bounded response reader together; extract the 240-line test module. This avoids creating a second factory runtime or unnecessary transport abstraction.

- `cmd/soda-factory/src/main.rs`
- `cmd/soda-factory/src/operator_tests.rs`

Evidence: 1-4: stated thin operator role and no database; 45-151: parse_args/dispatch for status, stop and reconcile; 153-351: sorted Go envelope encoding, send, read_response and response-size/chunk handling; 352-593: operator_server/request_parts, envelope/status/response/misuse tests -> operator_tests.rs.

### rust/soda-forgejo-domain/src/main.rs

Observed size: 1021 lines, including tests where embedded. Separate the existing native-writer control verbs from deployment app.ini/env parsing and host marker mapping. Retain native offline-marker name and quiescence/inhibition semantics. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-forgejo-domain/src/main.rs`
- `cmd/soda-forgejo-domain/src/cli.rs`
- `cmd/soda-forgejo-domain/src/system.rs`
- `cmd/soda-forgejo-domain/src/config.rs`
- `cmd/soda-forgejo-domain/src/domain.rs`
- `cmd/soda-forgejo-domain/src/tests/mod.rs`
- `cmd/soda-forgejo-domain/src/tests/fixtures.rs`
- `cmd/soda-forgejo-domain/src/tests/config.rs`
- `cmd/soda-forgejo-domain/src/tests/domain.rs`
- `cmd/soda-forgejo-domain/src/tests/cli.rs`

Evidence: rust/soda-forgejo-domain/src/main.rs:80-187 CLI, Paths and Sys; rust/soda-forgejo-domain/src/main.rs:237-493 INI interpolation, AppDataPath and marker mapping; rust/soda-forgejo-domain/src/main.rs:495-625 stop/inhibit/status/lift/start; rust/soda-forgejo-domain/src/main.rs:628-1021 fake system and configuration/control tests.

### rust/soda-forgejo-locales/src/locales.rs

Observed size: 568 lines, including tests where embedded. Keep the 319-line catalog parser/merge and locked-input operation intact after extracting tests. Its precise ConfigParser-like behavior and byte-preserving merge are already bounded; extra production modules would not clarify much. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/locales/merge.rs`
- `tools/release-assets/src/locales/merge/tests.rs`

Evidence: rust/soda-forgejo-locales/src/locales.rs:115-201 parse_ini and byte-preserving merge; rust/soda-forgejo-locales/src/locales.rs:214-318 locked/native input and exclusive output; rust/soda-forgejo-locales/src/locales.rs:321-568 parser/merge/source-integrity tests.

### rust/soda-forgejo-locales/tests/cli.rs

Observed size: 462 lines, including tests where embedded. Group current help/flag admission separately from local catalog merge/output behavior and lock-document admission. Reuse the existing local TempDir/command helpers; keep tests that reject a lock before network access.

- `tools/release-assets/tests/locales_cli.rs`
- `tools/release-assets/tests/locales_native_merge.rs`
- `tools/release-assets/tests/locales_locked_input.rs`
- `tools/release-assets/tests/locales_support/mod.rs`

Evidence: rust/soda-forgejo-locales/tests/cli.rs:23-71 shared fixture and binary invocation helpers; rust/soda-forgejo-locales/tests/cli.rs:73-184 native merge, defaults and size bound; rust/soda-forgejo-locales/tests/cli.rs:185-253 CLI argument/help checks; rust/soda-forgejo-locales/tests/cli.rs:254-330 lock source/document refusal; rust/soda-forgejo-locales/tests/cli.rs:332-462 merge namespace/exclusive-output refusals.

### rust/soda-host/src/account.rs

Observed size: 764 lines, including tests where embedded. Keep the 254-line production account/key implementation; its distinction between provisioning key normalization and revision-checked own-account replacement is established. Split the 509-line embedded test body into existing account provisioning cases and own-account key preview/apply/drift cases. Move Mock and fixture builders with account_tests.rs and share those existing helpers within the test module only; do not introduce a new fixture service.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/account/mod.rs`
- `lib/host/src/account/tests.rs`
- `lib/host/src/account/access_keys_tests.rs`

Evidence: 19-143: revision/key validation, canonicalize_account_keys/canonical_keys and access-key state decoding; 146-254: Runtime::account and Runtime::access_keys; preserve preview before apply and target incarnation checks; 255-598: Mock, fixture payloads, key normalization and account provisioning/rejection -> account_tests.rs; 599-764: access_keys_preview_observes_without_applying/apply_round_trips_preview_and_confirms_set/rejects_drift -> access_keys_tests.rs.

### rust/soda-host/src/domain.rs

Observed size: 709 lines, including tests where embedded. Separate already-existing pure domain records by concern inside the same host crate: profile/Create; account/AccessKeys; OS release/observation; and core project validators plus Environment/Connection in domain.rs. Move each type with its decoder/encoder/spec constants, retaining pure no-I/O validation and wire field order. The 532 lines before tests and the 177-line embedded test module are distinct; no new canonical DTOs should be introduced.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/domain/mod.rs`
- `lib/host/src/domain/profile.rs`
- `lib/host/src/domain/account.rs`
- `lib/host/src/domain/os.rs`
- `lib/host/src/domain/tests.rs`

Evidence: 1-6: pure mirrors of the established project Unix contract; 12-93,262-325: Unicode/ID/login/image/container validation, Environment/Connection -> domain.rs; 95-261: Profile specs/validation/encoding and Create -> profile.rs; 327-408: OsRelease/OsObservation and OS validators -> domain_os.rs; 411-531: Account/AccessKeys/AccessKeyState and their typed specs -> domain_account.rs; 533-709: profile/domain/account/OS wire and validator tests -> domain_tests.rs.

Open detail: The Go project domain remains the established owner; this physical split preserves the current Rust wire mirror and does not authorize a second competing definition or complete the host port.

### rust/soda-host/src/json.rs

Observed size: 1661 lines, including tests where embedded. Split the current host-local codec into value/public decode/output encoding, structural scanner, string token decoding, number scanning, typed field specifications/BoundMap access, and binding/type-error handling. The production code is 1,307 lines before tests. run_machine is one 276-line structural state machine with object/array/value branches; retain that algorithm together after removing unrelated lexical and binder code rather than creating one file per state. Separate existing strict/scanner/depth/UTF8 tests from binder/escaping/tolerant tests. Keep strict request policy distinct from tolerant Podman decoding and from lib/json.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/json/mod.rs`
- `lib/host/src/json/scan.rs`
- `lib/host/src/json/string.rs`
- `lib/host/src/json/number.rs`
- `lib/host/src/json/specs.rs`
- `lib/host/src/json/bind.rs`
- `lib/host/src/json/strict_tests.rs`
- `lib/host/src/json/binding_tests.rs`

Evidence: 23-74,662-805,1242-1306: Value/Error, decode_strict/decode_tolerant, Value accessors and output/tolerant map encoding -> json.rs; 76-190,191-465: Parser structural methods and run_machine object/array/value stack handling -> json_scan.rs; extract quote_byte with lexical diagnostics if needed to keep this file bounded; 467-585: hex4/parse_string, including surrogate substitution -> json_string.rs; 586-660,806-815: parse_number/parse_go_int64/parse_go_uint32 -> json_number.rs; 817-953: Spec/Kind/Bound/BoundMap and field access -> json_specs.rs; 954-1241: type_error, byte binding, bind_value/match_spec/bind_struct/bind_root -> json_bind.rs; 1308-1554: strict shapes/duplicates/scanner/errors/depth matrices -> json_strict_tests.rs; 1555-1661: binder fixtures, surrogate/output escaping and tolerant semantics -> json_binding_tests.rs.

Open detail: Scanner visibility/imports must be resolved inside this crate during implementation. Different duplicate-key and escaping contracts preclude an unverified blanket replacement with the shared minimal parser.

### rust/soda-host/src/preparation.rs

Observed size: 1209 lines, including tests where embedded. This is the pure preparation record/validation surface, not the executor. Group the existing validators and Preparation/Prepare identity in preparation.rs; RequirementAcceptance/AdminApproval in preparation_decisions.rs; ApprovedSetup/digest in preparation_setup.rs; resolved-tool, inspect/stop/hold and state wire in preparation_state.rs; FactoryCandidate in preparation_candidate.rs. Split 452 lines of tests into validation/digest/candidate and exact decoding/state-output cases. Move typed Spec tables with their consuming records.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/preparation/mod.rs`
- `lib/host/src/preparation/decisions.rs`
- `lib/host/src/preparation/setup.rs`
- `lib/host/src/preparation/state.rs`
- `lib/host/src/preparation/candidate.rs`
- `lib/host/src/preparation/validation_tests.rs`
- `lib/host/src/preparation/wire_tests.rs`

Evidence: 38-101,207-315,390-437: role/phase/ID validators, Preparation and Prepare -> preparation.rs; 102-206: RequirementAcceptance/AdminApproval and validation -> preparation_decisions.rs; 316-388: ApprovedSetup allowed files/bundle and setup_digest_of -> preparation_setup.rs; 438-698: ResolvedTool, PrepareState/Inspect/Stop, HoldState/PrepareHold -> preparation_state.rs; 699-755: FactoryCandidate validation/from_value/from_map -> preparation_candidate.rs; 756-1044: current record fixtures and validator/setup-digest/candidate cases -> preparation_validation_tests.rs; 1045-1209: prep_json/prepare_json, exact decoding messages and omitempty state output -> preparation_wire_tests.rs.

### rust/soda-host/src/prepare.rs

Observed size: 1955 lines, including tests where embedded. Split 1,113 production lines at the existing execution phases: orchestrating prepare/inspect/stop/hold; clean role paths and ID maps; fixed helper exchange/approval; source clone/head binding and role execution; launcher/tool verification and recorded evidence; protected candidate snapshot reads; and native state decode. Keep the prepare sequence and its re-observation on error together and preserve fixed argv/body bytes. The 840-line test module has existing pure/path helpers, full preparation-chain cases, and protected candidate cases suitable for separate concern test files.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/prepare/mod.rs`
- `lib/host/src/prepare/paths.rs`
- `lib/host/src/prepare/helper.rs`
- `lib/host/src/prepare/source.rs`
- `lib/host/src/prepare/tools.rs`
- `lib/host/src/prepare/candidate.rs`
- `lib/host/src/prepare/state.rs`
- `lib/host/src/prepare/tests.rs`
- `lib/host/src/prepare/execution_tests.rs`
- `lib/host/src/prepare/candidate_tests.rs`

Evidence: 21-150: path_clean/path_join/prepare_id_map/preparation_paths/single_line/tool path policy -> prepare_paths.rs; 151-348,1055-1113: inspection Spec tables, LauncherEvidence, map_preparation_state -> prepare_state.rs; approval/saved-request tables move with their actual decoder instead; 351-425,484-537: factory_helper/prepare_container/approve_preparation and files encoding -> prepare_helper.rs; 426-483,820-881: inspect/prepare orchestration/stop/hold -> prepare.rs; 538-637: role_exec/must_role_exec/clone_preparation_source/confirm_preparation_head -> prepare_source.rs; 638-819: verify_launcher_environment/resolve_preparation_tools/record_preparation_tools -> prepare_tools.rs; 882-1054: prepare_candidate/candidate_protected_file/candidate_approved_files -> prepare_candidate.rs; 1114-1456: existing Mock/config/preparation/setup fixtures and pure/native admission cases -> prepare_tests.rs; 1457-1788: full-chain/error/blocked/inspect/stop/hold cases -> prepare_execution_tests.rs; 1789-1955: candidate_reuses_ready_source/rejects_bad_snapshots -> prepare_candidate_tests.rs.

### rust/soda-host/src/project.rs

Observed size: 1383 lines, including tests where embedded. Separate Native/Executor and fixed process deadline/output handling from project operations. Keep creation/network setup/start/readiness in project_create.rs; immutable profile resolution in project_profile.rs; container observation/isolation in project_inspect.rs; OS-release reading and its shlex parser in project_os.rs; host-key reading/fingerprint in project_connection.rs. project.rs retains Config/Runtime and the common Podman entry. Existing tests occupy 396 lines and can remain one concern test module initially. Do not combine domain types with privileged execution.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/project/mod.rs`
- `lib/host/src/project/executor.rs`
- `lib/host/src/project/profile.rs`
- `lib/host/src/project/create.rs`
- `lib/host/src/project/inspect.rs`
- `lib/host/src/project/os.rs`
- `lib/host/src/project/connection.rs`
- `lib/host/src/project/tests.rs`

Evidence: 78-119,215-230: Config/Runtime/podman -> project.rs; 120-213: Native Executor, bounded subprocess I/O/deadlines and exit_text -> project_executor.rs; 231-306,743-777: resolve_profile/apply_creation_profile -> project_profile.rs; 408-539,682-711: create/create_container/start_created/wait_project_ready, go_arch/go_dir -> project_create.rs; 307-407,540-593,712-742,778-790: inspect/project_container, label decoding and ID-map isolation -> project_inspect.rs; 594-644,791-986: observe_os, split_lines/shlex_posix and parse_os_release -> project_os.rs; 645-681: connection host key/fingerprint -> project_connection.rs; 987-1383: mock runtime/profile/create/inspect/OS/connection cases -> project_tests.rs.

Open detail: soda-host/Cargo.toml currently defines only a library and no Rust binary; these are physical splits of existing library code, not evidence of daemon cutover.

### rust/soda-host/src/ssh.rs

Observed size: 1028 lines, including tests where embedded. Separate authorized_keys text/options scanning and public parse/marshal/fingerprint entrypoints from standard/Go base64 semantics, SSH binary/mpint primitives, non-certificate key material, and certificate field/tuple handling. Preserve supported algorithms, NIST on-curve checks, canonical signed-minimal mpints, sorted certificate tuples, and multiline trailing-data policy. The longest existing algorithm is parse_key_fields at about 97 lines and remains cohesive; no new key algorithm or dependency is proposed.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/ssh/mod.rs`
- `lib/host/src/ssh/base64.rs`
- `lib/host/src/ssh/mpint.rs`
- `lib/host/src/ssh/material.rs`
- `lib/host/src/ssh/certificate.rs`
- `lib/host/src/ssh/tests.rs`

Evidence: 26-64,756-907: trim_ws, parse_public_key/parse_key_text/scan_options/parse_authorized_key and public marshal/fingerprint -> ssh.rs; 65-229: strict standard and Go-specific base64 decoders/encoders -> ssh_base64.rs; 231-412: read_string/read_u32/read_u64/put_string, Mpint parsing/comparison/marshalling -> ssh_mpint.rs; 436-471,489-594,698-755: ParsedKey/KeyMaterial, curve_for, ordinary key parsing/marshalling -> ssh_material.rs; 413-435,472-488,595-697: certificate algorithm names/material, parse_tuples/parse_cert/marshal_tuples -> ssh_certificate.rs; certificate arm of marshal_fields stays with certificate responsibility; 908-1028: base64, canonical/authorized-key and fingerprint fixtures -> ssh_tests.rs.

### rust/soda-host/src/tailnet_companion.rs

Observed size: 2188 lines, including tests where embedded. The 1,111-line production companion module combines incarnation proof, process execution, state reconciliation/start, enrollment key consumption, confirmed logout/stop, and user-facing observation. Split these existing phases within the host crate while retaining one Companion executor and TailnetControl authority boundary. Keep repeated runtime/resolver/namespace checks at their current lifecycle points. Its 1,075-line test module needs the same concern grouping: existing fixture/mock scaffolding, identity/record cases, lifecycle cases, and view/action cases.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/companion/mod.rs`
- `lib/host/src/tailnet/companion/identity.rs`
- `lib/host/src/tailnet/companion/execute.rs`
- `lib/host/src/tailnet/companion/start.rs`
- `lib/host/src/tailnet/companion/enroll.rs`
- `lib/host/src/tailnet/companion/stop.rs`
- `lib/host/src/tailnet/companion/view.rs`
- `lib/host/src/tailnet/companion/tests.rs`
- `lib/host/src/tailnet/companion/identity_tests.rs`
- `lib/host/src/tailnet/companion/lifecycle_tests.rs`
- `lib/host/src/tailnet/companion/view_tests.rs`

Evidence: 33-89,133-360: CompanionRecord decode/recipe/exec validation, proc namespace identity and resolver match -> tailnet_companion_identity.rs; 90-132,370-393: TailnetControl/Companion, staged errors and freshness/idle helpers -> tailnet_companion.rs; 394-555,843-869: bounded native runtime execution/inspect/CLI plus wait_tailnet -> tailnet_companion_execute.rs; 556-624,671-773,811-842: start gating, admit run, prepare state/container activation/node wait, final recheck and start_tailnet -> tailnet_companion_start.rs; 774-810,938-980: enroll_companion/ensure_companion_enrolled and consume_run_key -> tailnet_companion_enroll.rs; 625-670,870-937: previous-run retirement/reconciliation and confirmed stop/logout/resolver restoration -> tailnet_companion_stop.rs; 981-1111: disable/start queue, stopped-project and companion/project status observation -> tailnet_companion_view.rs; 1112-1308: existing temp/run/view/binding/request helpers and MockExec/MockTailnet -> tailnet_companion_tests.rs; 1309-1605: record/recipe/proc/resolver/scanner/freshness identity cases -> tailnet_companion_identity_tests.rs; 1606-1955: native execution/start/wait/enrollment/stop/reconciliation cases -> tailnet_companion_lifecycle_tests.rs; 1956-2188: queued intent, stopped/observed status and project-status maps -> tailnet_companion_view_tests.rs.

### rust/soda-host/src/tailnet_domain.rs

Observed size: 1538 lines, including tests where embedded. Keep pure Tailnet DTO/error/ID definitions separate from RFC3339Nano validity, native status/preferences JSON binding with Go case folding, DNS/IP parsing/canonicalization, and project status/binding decisions. The 892-line production portion has clear existing seams. Split tests by has-node, project-status/native-binding, and address/DNS/time vectors; retain cohesive tables together rather than one file per case. None of these files gains runtime or SQL authority.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/domain/mod.rs`
- `lib/host/src/tailnet/domain/time.rs`
- `lib/host/src/tailnet/domain/native.rs`
- `lib/host/src/tailnet/domain/addresses.rs`
- `lib/host/src/tailnet/domain/status.rs`
- `lib/host/src/tailnet/domain/node_tests.rs`
- `lib/host/src/tailnet/domain/status_tests.rs`
- `lib/host/src/tailnet/domain/address_tests.rs`

Evidence: 1-84: errors, reader types, RunTarget/RunBinding/ProjectRequest/ProjectView and ID validators -> tailnet_domain.rs; 85-219: time math/parse_rfc3339_nano/go_escape -> tailnet_time.rs; 220-454: case folding/native_object/field binders and SelfPeer/NativeStatus/NativePrefs decoding -> tailnet_native.rs; 455-775: DNS/address parsers/canonicalization/global-unicast validation and peer resolution -> tailnet_addresses.rs; 776-892: parse_project_status/validate_project_preferences/match_project_self/binding/project_status/project_has_node -> tailnet_status.rs; 893-1026: binding/status/prefs fixtures and has_node vectors/state table -> tailnet_node_tests.rs; 1027-1294: connected/fresh/non-running/unsafe/malformed/casefold/limit project-status cases -> tailnet_status_tests.rs; 1295-1538: address/DNS/simple-fold/ID/RFC3339/escaping vectors -> tailnet_address_tests.rs.

### rust/soda-host/src/tailnet_files.rs

Observed size: 874 lines, including tests where embedded. Separate protected runtime-root/open/locking from current-run records/subdirectory preparation, ephemeral secret-file and companion-ID custody, and resolver inode/text/network-device admission. Keep file-descriptor identity and exclusive lock checks at existing mutation boundaries. Production is 566 lines; tests are 307 and can remain together because their fixtures exercise these linked filesystem boundaries.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/files/mod.rs`
- `lib/host/src/tailnet/files/run.rs`
- `lib/host/src/tailnet/files/keys.rs`
- `lib/host/src/tailnet/files/resolver.rs`
- `lib/host/src/tailnet/files/tests.rs`

Evidence: 36-225: Root/RunFiles structs, protected root/project ownership, open and exclusive lock -> tailnet_files.rs; 233-342: run subdirectories and RunFiles::current/save_current/prepare -> tailnet_run_files.rs; 343-496: key create/retire/pending-key inode handling and companion ID records -> tailnet_keys.rs; 497-566: resolver generated path/inode/text/device conflict checks -> tailnet_resolver.rs; 567-874: existing TempDir/run fixtures, locks, symlink/mode/key-inode/resolver/explicit-retry cases -> tailnet_files_tests.rs.

### rust/soda-host/src/tailnet_runtime.rs

Observed size: 1121 lines, including tests where embedded. Split the 687-line production runtime into native proc identity reading, run snapshot/record encoding plus identity digest, orchestration/companion recipe, and project container isolation/inspection. Keep snapshot rereads and exact process identity comparison before admitting a run. Extract the 433-line test portion into run/process/recipe cases and container-isolation/inspection cases; do not broaden supported isolation mappings.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/runtime/mod.rs`
- `lib/host/src/tailnet/runtime/process.rs`
- `lib/host/src/tailnet/runtime/wire.rs`
- `lib/host/src/tailnet/runtime/project.rs`
- `lib/host/src/tailnet/runtime/tests.rs`
- `lib/host/src/tailnet/runtime/project_tests.rs`

Evidence: 23-67,386-549: ProjectRun/runtime declarations, actual readers, admit/confirm/inspect/project_run and companion_create_args/recheck -> tailnet_runtime.rs; 74-210: proc stat/start/id-map/namespace/boot readers and process_run_identity -> tailnet_process.rs; 211-385: RUN_INSPECT specs, run decode/encode/assemble_project_run digest binding -> tailnet_run_wire.rs; 550-687: id_map/project_isolation/inspect_project/project_container/project_running -> tailnet_project.rs; 688-1014: synthetic proc readers, exact recipe/run wire/admission cases -> tailnet_runtime_tests.rs; 1015-1121: inspection executor/JSON fixture and argv/isolation gates -> tailnet_project_tests.rs.

### rust/soda-identity-compose/src/main.rs

Observed size: 910 lines, including tests where embedded. Keep main as the explicit one-service registration sequence. Move existing flag decisions to options.rs; Compose override/up/ps/immutable-child selection to compose.rs; private tmpfs registration root, provisioned account marker and launch-socket registration to registration.rs; request/exit wire handling to launch_wire.rs; generic scalar/skip token routines to launch_json.rs. The production portion is 750 lines and tests 158. Preserve service opt-in, immutable Podman child attribution, launch-only socket access and current error/JSON behavior; do not add Compose orchestration features.

- `cmd/soda-identity-compose/src/main.rs`
- `cmd/soda-identity-compose/src/options.rs`
- `cmd/soda-identity-compose/src/compose.rs`
- `cmd/soda-identity-compose/src/registration.rs`
- `cmd/soda-identity-compose/src/launch_wire.rs`
- `cmd/soda-identity-compose/src/launch_json.rs`
- `cmd/soda-identity-compose/src/compose_tests.rs`

Evidence: 20-47: load options, attest account, create runtime root, launch one service and register -> main.rs; 13-18,49-165: Options and flag/validation/usage helpers -> options.rs; 245-287,316-345,697-750: launch_compose/write_override/select_compose_child and immutable ID/service attribution -> compose.rs; 166-244,288-315,626-696: private tmpfs root/random/mkdir, account marker and SOCK_SEQPACKET registration -> registration.rs; 346-470: exact string/request encoding and LaunchExit decode -> launch_wire.rs; 471-625: parse_json_string/parse_json_integer/skip_json_value -> launch_json.rs; 751-910: flags, exact override/wire response, runtime-root and Compose-child vectors -> compose_tests.rs.

### rust/soda-identity/src/control.rs

Observed size: 837 lines, including tests where embedded. The 837-line controller has no embedded tests; separate its already-existing broker concerns while retaining the single mutex-protected State and one Controller. Provider adapter traits/mapping are transport-neutral broker integration; enrollment and grants remain distinct from acquisition/execution admission, registration/credential delivery, and confirmed retirement/reconciliation. Keep all current lock/transaction and native runtime attestation sequencing. This is a file split within one broker crate, not independent controllers or a new authorization layer. Keep the small existing provider/runtime adapter block in control.rs; providers/mod.rs is the one root for the consolidated native provider implementations.

- `cmd/soda-identity/src/control.rs` — Provider/runtime traits and existing session adapters remain with controller construction, State, lock ownership and shared controller helpers.
- `cmd/soda-identity/src/enrollment.rs`
- `cmd/soda-identity/src/grants.rs`
- `cmd/soda-identity/src/acquisition.rs`
- `cmd/soda-identity/src/registration.rs`
- `cmd/soda-identity/src/retirement.rs`

Evidence: 13-85: provider/runtime traits, error mapping and Codex/Muse session adapters -> control.rs; 86-165,800-837: EnrollmentEntry/State/Controller, construction/lock/ownership, close/error join/random-ID/zeroize -> control.rs; 133-143,166-272: pending/start/read/cancel enrollment -> enrollment.rs; 273-298,705-739: grant listing/create plus connection/grant revoke -> grants.rs; retirement helpers remain called from retirement.rs; 298-501: lease list, acquire/admit/replay/reserve/get/close execution and reservation authorization -> acquisition.rs; 502-632: register/registration authority/execution binding/terminal attestation/release and return_lease -> registration.rs; 633-704,740-799: uncertain/end/finish/reconcile/sweep/reject/retire_connection -> retirement.rs.

### rust/soda-identity/src/http.rs

Observed size: 494 lines, including tests where embedded. Separate socket serving/shutdown/inflight ownership from bounded HTTP head/body framing and encoding, and from current admin/runtime route admission. Production is 464 lines and tests are 28. Keep runtime_allowed enforcement on the existing service socket path and POST/query/Origin checks intact; route files do not own broker custody or policy. Do not replace this with a new server framework.

- `cmd/soda-identity/src/http.rs`
- `cmd/soda-identity/src/http_wire.rs`
- `cmd/soda-identity/src/http_routes.rs`
- `cmd/soda-identity/src/http_tests.rs`

Evidence: 73-151: Server construction/serve/handle_connection, shutdown/inflight and dead-listener behavior -> http.rs; 152-283,431-464: Admission/HttpRequest/read_request/percent_decode/success_response/error_response -> http_wire.rs; 1-72,284-430: allowed wire fields/nested specs, dispatch/error mapping, admin and runtime routes -> http_routes.rs; 465-494: percent-path, error-body and success-envelope cases -> http_tests.rs.

### rust/soda-identity/src/pg.rs

Observed size: 512 lines, including tests where embedded. Separate PostgreSQL DSN parsing/percent decoding, connection/transport/authentication and message receipt, and row decoding/query/simple-command result handling. Keep the current upstream postgres-protocol framing/SCRAM use and one Client connection/buffer. Production is 475 lines and tests 35; no new pooling/retry/TLS support is proposed. Authentication phases already have authenticate and authenticate_scram functions and need no extra state machine.

- `cmd/soda-identity/src/pg.rs`
- `cmd/soda-identity/src/pg_dsn.rs`
- `cmd/soda-identity/src/pg_query.rs`
- `cmd/soda-identity/src/pg_tests.rs`

Evidence: 19-125: Dsn parse/percent_decode -> pg_dsn.rs; 126-146,187-327,456-475: Stream/Client/connect/send/receive/authenticate/authenticate_scram and backend error -> pg.rs; 147-186,328-455: Row scalar/bytea conversions and query/simple/command_count -> pg_query.rs; 476-512: DSN forms and command-tag row counts -> pg_tests.rs.

### rust/soda-identity/src/store.rs

Observed size: 798 lines, including tests where embedded. Split existing SQL locality by connection custody, grants, leases, execution records and audit events, retaining Store/Tx plus parameter binding/encoding/transaction primitives in store.rs. Schema bootstrap and grant-key admission belong together in store_schema.rs; preserve internal/store/schema.go as the existing authoritative schema and leave schema.rs a mechanically synchronized representation. Production is 772 lines and tests 24. Do not create separate databases, duplicate schema ownership, or alter transaction atomicity while moving methods.

- `cmd/soda-identity/src/store.rs`
- `cmd/soda-identity/src/store_schema.rs`
- `cmd/soda-identity/src/store_connections.rs`
- `cmd/soda-identity/src/store_grants.rs`
- `cmd/soda-identity/src/store_leases.rs`
- `cmd/soda-identity/src/store_executions.rs`
- `cmd/soda-identity/src/store_events.rs`
- `cmd/soda-identity/src/store_tests.rs`

Evidence: 13-117,576-587,704-710,729-772: placeholder bind, Store/Tx query/transaction, changed and Param encoding -> store.rs; 118-206,588-639: grant-key/schema admission/bootstrap, load_schema_version/required columns/trigger checks -> store_schema.rs; 207-333: encrypted save_connection/read credential/list/available/state -> store_connections.rs; 334-412: grant insert/list/read/revoke -> store_grants.rs; 413-488,544-555,668-703: reserve/register/return/forget lease and transactional maintain_credential -> store_leases.rs; 489-543: execution/admit_execution/observe_execution -> store_executions.rs; 556-575,640-667,711-728: events/append_event/lease_event -> store_events.rs; 773-798: placeholder and bytea encoding cases -> store_tests.rs.

### rust/soda-identity/src/strict.rs

Observed size: 402 lines, including tests where embedded. Keep the cohesive 314-line request codec: bounded UTF-8 object decode, decoded duplicate-key scanner, case remapping and known-field checking are one admission path. Extract the 86-line test module. The total only barely exceeds 400 because of tests; do not create multiple scanners or replace it with a different duplicate/case/null policy.

- `cmd/soda-identity/src/strict.rs`
- `cmd/soda-identity/src/strict_tests.rs`

Evidence: 1-86: decode/remap_case/check_known_fields and declared strict request semantics; 87-314: Scanner lexical handling/check_unique_keys/check_value recursion; 315-402: strict shape/size/casefold/decoded-Unicode duplicate vectors -> strict_tests.rs.

### rust/soda-identity/src/wire.rs

Observed size: 981 lines, including tests where embedded. Split the 833-line production wire mirror by existing record responsibility, keeping canonical field names/order/string integers/null semantics and provider Connection/Enrollment reuse. wire.rs owns Request/DeliveryWire and provider/constants; wire_time owns UnixTime/date formatting; wire_scalars owns current int/null/base64 serde routines; grant/acquisition and lease/binding/execution records each keep their validators/digest; wire_errors keeps the fixed failure taxonomy. The 147-line tests can remain a single exact-wire/time/digest module.

- `cmd/soda-identity/src/wire.rs`
- `cmd/soda-identity/src/wire_time.rs`
- `cmd/soda-identity/src/wire_scalars.rs`
- `cmd/soda-identity/src/wire_grants.rs`
- `cmd/soda-identity/src/wire_execution.rs`
- `cmd/soda-identity/src/wire_errors.rs`
- `cmd/soda-identity/src/wire_tests.rs`

Evidence: 1-23,578-631,655-661: existing provider record reuse/constants, Request including its derive/credential field and DeliveryWire including its derive -> wire.rs; preserve the single Connection/Enrollment definitions under providers/types.rs; 24-219: UnixTime/civil date math/parse_rfc3339_nano/format_rfc3339_nano -> wire_time.rs; scalar-codec documentation starts separately at221; 221-354,462-464,633-653: complete int-string/null/base64 serde modules, is_zero_i32 and complete base64_bytes_option module -> wire_scalars.rs; 356-410,747-775,817-832: Grant/GrantRequest/AcquireRequest including derive attributes, validation and acquisition_digest -> wire_grants.rs; 412-460,466-576,777-815: Binding/Lease/Execution/Event including derive attributes and binding/execution validation -> wire_execution.rs; 663-745: ErrorKind/Error including derive attributes, mapping and conversions -> wire_errors.rs; 834-981: cfg(test) descendant module for Go time/wire/base64/null/provider ID/acquisition digest vectors -> wire_tests.rs.

Open detail: Module imports and serde with/deserialize_with/skip_serializing_if paths must follow each complete codec declaration. Keep tests as a descendant of the owning wire module and preserve one Rust definition per existing record; moving codecs does not authorize a new wire shape or forwarding DTO.

### rust/soda-identity/tests/broker.rs

Observed size: 543 lines, including tests where embedded. This whole 543-line file is integration-test code, not 203 lines of production before the first #[test]. Extract only the existing environment-gated PostgreSQL/StubProvider/StubRuntime fixture support into tests/common/mod.rs; group persistent custody/fencing/revoke checks in broker.rs, enrollment-to-lease path in enrollment.rs, and admin/runtime HTTP plus dead-listener shutdown in http.rs. Keep the same real broker/store under test, fixture opt-in, native cleanup and skips; do not create a new test harness.

- `cmd/soda-identity/tests/broker.rs`
- `cmd/soda-identity/tests/enrollment.rs`
- `cmd/soda-identity/tests/http.rs`
- `cmd/soda-identity/tests/common/mod.rs`

Evidence: 1-203: super_dsn/Ephemeral create/store/drop, fixed credentials/connections, StubSession/Provider/Runtime and controller/binding -> tests/common/mod.rs; 204-238,305-367: store_round_trip/close_execution_fences_late_registration/revoke_retires_live_leases -> tests/broker.rs; 239-304: enrollment_to_lease_lifecycle -> tests/enrollment.rs; 368-543: http_admission_matches_go and dead_listener_fails_fast including thread/FD guards -> tests/http.rs.

### rust/soda-image-import/src/main.rs

Observed size: 2172 lines, including tests where embedded. Separate already marked platform, hash, Go-compatible JSON binding, payload, OCI metadata/layout verification and Podman import sections. Keep whole-layout verification before any import and group the 872 test lines by current concern. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-image-import/src/main.rs`
- `cmd/soda-image-import/src/context.rs`
- `cmd/soda-image-import/src/platform.rs`
- `cmd/soda-image-import/src/sha256.rs`
- `cmd/soda-image-import/src/json_binding.rs`
- `cmd/soda-image-import/src/payload.rs`
- `cmd/soda-image-import/src/oci/mod.rs`
- `cmd/soda-image-import/src/oci/metadata.rs`
- `cmd/soda-image-import/src/oci/layout.rs`
- `cmd/soda-image-import/src/oci/inspection.rs`
- `cmd/soda-image-import/src/import.rs`
- `cmd/soda-image-import/src/tests/mod.rs`
- `cmd/soda-image-import/src/tests/fixtures.rs`
- `cmd/soda-image-import/src/tests/payload.rs`
- `cmd/soda-image-import/src/tests/oci.rs`
- `cmd/soda-image-import/src/tests/import.rs`
- `cmd/soda-image-import/src/tests/primitives.rs`

Evidence: rust/soda-image-import/src/main.rs:49-120 entry admission and import cancellation context; rust/soda-image-import/src/main.rs:122-205 native platform and identifier shapes; rust/soda-image-import/src/main.rs:207-521 streaming SHA and JSON binding; rust/soda-image-import/src/main.rs:524-692 Payload decoding/validation/load; rust/soda-image-import/src/main.rs:693-1135 OCI metadata/layout/content inspection; rust/soda-image-import/src/main.rs:1136-1298 verified native import; rust/soda-image-import/src/main.rs:1301-2172 fixture, payload, layout and import tests.

Open detail: This duplicates installer release/OCI verification with somewhat different result shapes. A shared release/OCI library would require a caller/type/error comparison beyond a filetree split; do not create it solely from visual similarity.

### rust/soda-install/src/candidate.rs

Observed size: 453 lines, including tests where embedded. Keep the 216-line candidate-media authentication and destination assembly together; extract its substantial OCI/template fixtures and tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/candidate.rs`
- `cmd/soda-install/src/candidate/tests.rs`

Evidence: rust/soda-install/src/candidate.rs:18-67 candidate media/payload/image authentication; rust/soda-install/src/candidate.rs:113-215 host-file rewrite and candidate_destination; rust/soda-install/src/candidate.rs:217-453 fixtures and requirement/destination tests.

### rust/soda-install/src/console.rs

Observed size: 700 lines, including tests where embedded. Separate raw TTY read/write/echo handling from the established network editor/review interaction. Move the existing PTY test support into a test-only child module still shared by installer tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/console/mod.rs`
- `cmd/soda-install/src/console/terminal.rs`
- `cmd/soda-install/src/console/network.rs`
- `cmd/soda-install/src/console/test_support.rs`
- `cmd/soda-install/src/console/tests.rs`

Evidence: rust/soda-install/src/console.rs:21-217 terminal IO, line/secret input and echo restoration; rust/soda-install/src/console.rs:219-391 nmtui and live-network interaction; rust/soda-install/src/console.rs:396-482 byte trimming and errno formatting; rust/soda-install/src/console.rs:485-593 existing shared test_support PTY helpers; rust/soda-install/src/console.rs:595-700 console and network tests.

### rust/soda-install/src/deliver.rs

Observed size: 524 lines, including tests where embedded. Keep the 288-line installer payload reader and content-binding entry together; extract payload and OCI fixtures/tests. This is installer verification, not release publication. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/deliver.rs`
- `cmd/soda-install/src/deliver/tests.rs`

Evidence: rust/soda-install/src/deliver.rs:28-240 Payload shapes, validation, decoding and load; rust/soda-install/src/deliver.rs:243-286 revision/layout identity binding and verify_content; rust/soda-install/src/deliver.rs:290-524 validation/load/OCI-binding tests.

### rust/soda-install/src/disks.rs

Observed size: 588 lines, including tests where embedded. Keep the 403-line disk inventory/identity safety operation intact; its sysfs, live-media and holders checks jointly determine one disk selection. Extract tests rather than split this safety sequence by size. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/disks.rs`
- `cmd/soda-install/src/disks/tests.rs`

Evidence: rust/soda-install/src/disks.rs:173-315 sysfs/live-media/removable/holders reads; rust/soda-install/src/disks.rs:317-388 scan_disks and same_disk; rust/soda-install/src/disks.rs:405-588 inventory, live-media and identity-change tests.

### rust/soda-install/src/enroll/arm.rs

Observed size: 636 lines, including tests where embedded. Retain the current 397-line enrollment address selection/intent/arming refusal path; extract descendant tests for admission, expiry and preserving existing state. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/arm.rs`
- `cmd/soda-install/src/enroll/arm_tests.rs`

Evidence: rust/soda-install/src/enroll/arm.rs:31-185 interface/state/window input; rust/soda-install/src/enroll/arm.rs:202-315 select address, fingerprint and explicit intent; rust/soda-install/src/enroll/arm.rs:329-396 guard_existing_enrollment_state; rust/soda-install/src/enroll/arm.rs:398-636 address/state/unit admission tests.

### rust/soda-install/src/enroll/keys.rs

Observed size: 791 lines, including tests where embedded. Separate descriptor-owned directory/stat helpers from the cohesive authorized_keys append/create/confirm transaction. Preserve one-write uncertainty and concurrent-native-editor protection. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/keys/mod.rs`
- `cmd/soda-install/src/enroll/keys/directory.rs`
- `cmd/soda-install/src/enroll/keys/authorized_keys.rs`
- `cmd/soda-install/src/enroll/keys/tests.rs`

Evidence: rust/soda-install/src/enroll/keys.rs:40-145 descriptor, ownership, lock and key-stat helpers; rust/soda-install/src/enroll/keys.rs:149-308 existing-key append and exclusive creation; rust/soda-install/src/enroll/keys.rs:361-465 append_enrollment_key_with_writer and confirmations; rust/soda-install/src/enroll/keys.rs:467-491 safe directory and native root-home admission; rust/soda-install/src/enroll/keys.rs:494-791 preservation/race/partial-write tests.

### rust/soda-install/src/enroll/mod.rs

Observed size: 519 lines, including tests where embedded. Leave the enrollment entry module small; separate generated native SSH/systemd configuration from socket peer/cgroup provenance. Move already shared test keys, temporary-directory helpers and environment lock into a test-only sibling. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/mod.rs`
- `cmd/soda-install/src/enroll/native_config.rs`
- `cmd/soda-install/src/enroll/peer.rs`
- `cmd/soda-install/src/enroll/tests.rs`
- `cmd/soda-install/src/enroll/test_support.rs`

Evidence: rust/soda-install/src/enroll/mod.rs:19-27 fixed enrollment paths and limits; rust/soda-install/src/enroll/mod.rs:39-81 fixed restricted sshd configuration; rust/soda-install/src/enroll/mod.rs:131-197 start/socket/template unit configuration; rust/soda-install/src/enroll/mod.rs:199-252 root peer credentials and native unit provenance; rust/soda-install/src/enroll/mod.rs:255-355 existing shared test-only constants and helpers; rust/soda-install/src/enroll/mod.rs:357-519 configuration/provenance tests.

### rust/soda-install/src/execute.rs

Observed size: 695 lines, including tests where embedded. Keep the existing 393-line disk-attempt execution, retry and diagnostic landing flow together; extract tests. The fresh inventory, intentional-removable confirmation and started marker remain ordered before disk writing. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/execute.rs`
- `cmd/soda-install/src/execute/tests.rs`

Evidence: rust/soda-install/src/execute.rs:69-112 failure/retry flow; rust/soda-install/src/execute.rs:151-211 diagnostic console landing; rust/soda-install/src/execute.rs:261-389 verified disk attempt and execute_disk write boundary; rust/soda-install/src/execute.rs:394-695 disk execution/refusal/landing tests.

### rust/soda-install/src/inputs.rs

Observed size: 454 lines, including tests where embedded. Keep the 211-line private provisioning input validation/template extension together; extract input vectors and destination assertions. No new schema or validation rule is proposed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/inputs.rs`
- `cmd/soda-install/src/inputs/tests.rs`

Evidence: rust/soda-install/src/inputs.rs:13-60 hostname/subnet/password admission; rust/soda-install/src/inputs.rs:62-160 template decoding and path collisions; rust/soda-install/src/inputs.rs:162-208 destination construction; rust/soda-install/src/inputs.rs:212-454 vectors and provisioning template tests.

### rust/soda-install/src/netip.rs

Observed size: 836 lines, including tests where embedded. Separate existing address parse, byte-preserving address formatting/classification and prefix operations, with grouped oracle vectors. Keep one canonical Addr/Prefix representation and parent-private helper access. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/netip/mod.rs`
- `cmd/soda-install/src/netip/address.rs`
- `cmd/soda-install/src/netip/address_format.rs`
- `cmd/soda-install/src/netip/prefix.rs`
- `cmd/soda-install/src/netip/tests/mod.rs`
- `cmd/soda-install/src/netip/tests/address.rs`
- `cmd/soda-install/src/netip/tests/prefix.rs`

Evidence: rust/soda-install/src/netip.rs:1-9 documented Go byte/error fidelity; rust/soda-install/src/netip.rs:69-331 IPv4/IPv6 byte parsing; rust/soda-install/src/netip.rs:333-462 address classification/string/mask helpers; rust/soda-install/src/netip.rs:463-543 Prefix parse/mask/contains; rust/soda-install/src/netip.rs:546-836 oracle address/prefix and error-text tests.

Open detail: Std IpAddr replacement is not equivalent to the existing arbitrary byte-zone/error contract. Upstream replacement is a separate behavior decision, not presumed by relocation.

### rust/soda-install/src/oci.rs

Observed size: 1175 lines, including tests where embedded. Separate fd-relative layout/blob loading, index/manifest/config binding and image/rootfs/attribution inspection. Preserve hard versus soft JSON decoding distinctions. Move the existing shared OCI fixture into test-only support. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/oci/mod.rs`
- `cmd/soda-install/src/oci/layout.rs`
- `cmd/soda-install/src/oci/metadata.rs`
- `cmd/soda-install/src/oci/inspection.rs`
- `cmd/soda-install/src/oci/test_support.rs`
- `cmd/soda-install/src/oci/tests.rs`

Evidence: rust/soda-install/src/oci.rs:27-258 layout types and descriptor-confined file/blob loading; rust/soda-install/src/oci.rs:260-617 soft/hard descriptor and manifest/config decoding; rust/soda-install/src/oci.rs:618-863 rootfs/attribution/image/layout inspection; rust/soda-install/src/oci.rs:866-1020 existing test fixture module; rust/soda-install/src/oci.rs:1023-1175 identity/count/substitution tests.

### rust/soda-install/src/setup.rs

Observed size: 1140 lines, including tests where embedded. Separate existing address and token prompting, reserved setup/activation attempt, configured-access guidance and public CA parsing/fingerprint logic. Group the 517 lines of unit/PTY tests by flow and access concern. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/setup/mod.rs`
- `cmd/soda-install/src/setup/address.rs`
- `cmd/soda-install/src/setup/configure.rs`
- `cmd/soda-install/src/setup/access.rs`
- `cmd/soda-install/src/setup/local_ca.rs`
- `cmd/soda-install/src/setup/tests/mod.rs`
- `cmd/soda-install/src/setup/tests/fixtures.rs`
- `cmd/soda-install/src/setup/tests/configure.rs`
- `cmd/soda-install/src/setup/tests/access.rs`

Evidence: rust/soda-install/src/setup.rs:18-156 setup address decoding/origin selection; rust/soda-install/src/setup.rs:209-435 prompting, live-address recheck and setup/activation operation; rust/soda-install/src/setup.rs:472-600 installed origin/unit admission and client trust guidance; rust/soda-install/src/setup.rs:602-619 public CA fingerprint verification; rust/soda-install/src/setup.rs:623-1140 CA/configuration/rollback/secret-transcript tests.

### rust/soda-install/src/sshkey.rs

Observed size: 904 lines, including tests where embedded. Separate Base64 codec, existing SSH key/certificate wire parse/marshal and authorized_keys text/options policy. Preserve wire truncation and canonicalization rules; keep key representation singular. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/sshkey/mod.rs`
- `cmd/soda-install/src/sshkey/base64.rs`
- `cmd/soda-install/src/sshkey/wire.rs`
- `cmd/soda-install/src/sshkey/authorized_keys.rs`
- `cmd/soda-install/src/sshkey/tests.rs`

Evidence: rust/soda-install/src/sshkey.rs:15-113 Go-compatible Base64 decode/encode; rust/soda-install/src/sshkey.rs:115-568 SSH wire integers, keys, certificates and marshaling; rust/soda-install/src/sshkey.rs:570-721 authorized_keys parsing, operator key policy and fingerprint; rust/soda-install/src/sshkey.rs:724-904 key vectors, unsupported types, canonicalization and codec tests.

Open detail: Replacing wire/key readers with a library requires checking certificate/security-key/trailing-byte and error-taxonomy behavior already exercised here. This plan proposes module boundaries only.

### rust/soda-install/src/urlx.rs

Observed size: 730 lines, including tests where embedded. Keep the coherent 373-line byte-preserving URL parser intact; extract its extensive oracle/error vectors as a descendant test module. Do not split its mutually dependent authority/unescape parser into arbitrary pieces. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/urlx.rs`
- `cmd/soda-install/src/urlx/tests.rs`

Evidence: rust/soda-install/src/urlx.rs:1-9 documented net/url byte/error semantics; rust/soda-install/src/urlx.rs:105-355 unescape, host/authority/scheme and parse; rust/soda-install/src/urlx.rs:356-370 hostname; rust/soda-install/src/urlx.rs:374-730 oracle URL and exact-error tests.

Open detail: Standard or third-party URL parsing cannot be assumed equivalent to decoded non-UTF8 fields and the existing scheme-specific host rules.

### rust/soda-install/src/wizard.rs

Observed size: 627 lines, including tests where embedded. Separate the established wizard state/navigation loop from individual input steps and the final disk-erasure review/confirmation. Preserve back-navigation resets and password-to-hash handling. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/wizard/mod.rs`
- `cmd/soda-install/src/wizard/steps.rs`
- `cmd/soda-install/src/wizard/review.rs`
- `cmd/soda-install/src/wizard/tests.rs`

Evidence: rust/soda-install/src/wizard.rs:25-284 navigation and network/disk/hostname/password/subnet steps; rust/soda-install/src/wizard.rs:285-366 final review and erase confirmation; rust/soda-install/src/wizard.rs:368-447 step dispatch and collect_disk_install_choices; rust/soda-install/src/wizard.rs:450-627 selection and complete PTY flow tests.

### rust/soda-install/src/x509.rs

Observed size: 3624 lines, including tests where embedded. Use the explicit DER, algorithm/type, validity, name/constraint, extension, key, ParseCertificate and CheckSignatureFrom sections as private modules. Group the 1286 test lines by those same protocols and keep existing DER fixture builders test-only. Signature math continues to use the existing crypto crates; no new cryptographic implementation is proposed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/x509/mod.rs`
- `cmd/soda-install/src/x509/der.rs`
- `cmd/soda-install/src/x509/algorithms.rs`
- `cmd/soda-install/src/x509/types.rs`
- `cmd/soda-install/src/x509/time.rs`
- `cmd/soda-install/src/x509/names.rs`
- `cmd/soda-install/src/x509/name_constraints.rs`
- `cmd/soda-install/src/x509/extensions.rs`
- `cmd/soda-install/src/x509/public_key.rs`
- `cmd/soda-install/src/x509/certificate.rs`
- `cmd/soda-install/src/x509/verify.rs`
- `cmd/soda-install/src/x509/tests/mod.rs`
- `cmd/soda-install/src/x509/tests/fixtures.rs`
- `cmd/soda-install/src/x509/tests/structure.rs`
- `cmd/soda-install/src/x509/tests/public_key.rs`
- `cmd/soda-install/src/x509/tests/algorithms.rs`
- `cmd/soda-install/src/x509/tests/time.rs`
- `cmd/soda-install/src/x509/tests/extensions.rs`
- `cmd/soda-install/src/x509/tests/verify.rs`

Evidence: rust/soda-install/src/x509.rs:1-27 scope, exact Go parser/error semantics, rejected x509-cert profile and existing crypto delegates; rust/soda-install/src/x509.rs:38-378 DER reader and tag/OID constants; rust/soda-install/src/x509.rs:388-506 algorithm enums/public key/certificate types; rust/soda-install/src/x509.rs:512-781 validity time and ASN1 name/algorithm parsing; rust/soda-install/src/x509.rs:783-1481 extension and name-constraint parsing; rust/soda-install/src/x509.rs:1488-1862 signature algorithm and public-key parsing; rust/soda-install/src/x509.rs:1870-2046 ParseCertificate sequence; rust/soda-install/src/x509.rs:2052-2336 CheckSignatureFrom and existing signature verification delegates; rust/soda-install/src/x509.rs:2339-3624 structural/key/time/extension/signature vectors and DER fixture builders.

Open detail: The source explicitly documents why x509-cert did not match accepted serial/UTCTime behavior. Removing this port or narrowing certificate validation needs an owner decision and primary-source/caller validation. Moving modules must preserve parser error precedence and uses of netip/urlx; no public visibility should be added solely for tests.

### rust/soda-muse-maintain/src/main.rs

Observed size: 4075 lines, including tests where embedded. The file contains 2,892 lines before tests plus a 1,183-line embedded test module, spanning every maintenance phase. Preserve one ordered maintain operation: admit pinned public tool sources, bind exact project incarnation, optionally ensure system bus/stage tools, prepare public directory, then attach the restricted mount. Split current flags, host-config wire/validation, native tool files with their path/errno operations, existing config wire diagnostics, JSON lexical parser, release-payload read/decode/validation, network parsing, bounded command execution, project observation, archive/staging and interface mount/admission into concern files. Parser string decoding separates from scalar/structural scanner so neither lexical file must carry the unrelated release/host object bindings. Retain fixed scripts with their existing phase. Test files follow existing groups and shared TestDir/synthetic archive helpers move to test_support.rs only. Keep path/errno support with the real pinned-file operations in filesystem.rs (about 309 production lines); the existing Go quote renderer stays with config wire diagnostics rather than a tiny standalone module. These shared crate-private functions may be reused by current flags/network/interface callers without introducing a new package.

- `cmd/soda-muse-maintain/src/main.rs`
- `cmd/soda-muse-maintain/src/options.rs`
- `cmd/soda-muse-maintain/src/config.rs`
- `cmd/soda-muse-maintain/src/config_wire.rs`
- `cmd/soda-muse-maintain/src/config_validation.rs`
- `cmd/soda-muse-maintain/src/filesystem.rs`
- `cmd/soda-muse-maintain/src/json.rs`
- `cmd/soda-muse-maintain/src/json_string.rs`
- `cmd/soda-muse-maintain/src/release.rs`
- `cmd/soda-muse-maintain/src/release_wire.rs`
- `cmd/soda-muse-maintain/src/release_validation.rs`
- `cmd/soda-muse-maintain/src/network.rs`
- `cmd/soda-muse-maintain/src/sha256.rs`
- `cmd/soda-muse-maintain/src/project.rs`
- `cmd/soda-muse-maintain/src/command.rs`
- `cmd/soda-muse-maintain/src/stage.rs`
- `cmd/soda-muse-maintain/src/archive.rs`
- `cmd/soda-muse-maintain/src/interface.rs`
- `cmd/soda-muse-maintain/src/interface_admission.rs`
- `cmd/soda-muse-maintain/src/test_support.rs`
- `cmd/soda-muse-maintain/src/options_tests.rs`
- `cmd/soda-muse-maintain/src/config_tests.rs`
- `cmd/soda-muse-maintain/src/network_tests.rs`
- `cmd/soda-muse-maintain/src/release_tests.rs`
- `cmd/soda-muse-maintain/src/project_tests.rs`
- `cmd/soda-muse-maintain/src/filesystem_tests.rs`
- `cmd/soda-muse-maintain/src/interface_tests.rs`
- `cmd/soda-muse-maintain/src/archive_tests.rs`

Evidence: 59-212: Options/default_tools/parse/usage/bool/project-ID flag policy -> options.rs; 66-91 and 268-279 -> main.rs run/maintain; 281-338: go_clean/go_base/go_dir and 366-513: path_error/go_errno/last_errno -> filesystem.rs with real pinned-file operations; 216-267: go_quoted/nonprint -> config_wire.rs with existing diagnostic formatting; 339-365: Config/load_config -> config.rs; 514-725: host decode/field slot/string setters with saved error semantics -> config_wire.rs; 1690-1790: current runtime validation order -> config_validation.rs; 726-792,904-1073,1074-1100: JsonParser basic methods, literal/number/skip scanning and UTF8 unit decoding -> json.rs; 793-903: parse_string/surrogate_tail -> json_string.rs; 1101-1230: apply_release_images, ReleasePayload/Image and confined payload file admission -> release.rs; 1231-1366,1481-1689: typed release parsing/image/string-list binding -> release_wire.rs; 1367-1480: payload/digest/architecture/reference validation -> release_validation.rs; 1791-1993: observed netip prefix/address/IPv4/IPv6 error-parity routines -> network.rs; 1994-2095: Tool FD ownership/load_tools/open_tool/trusted_tool/verify_native -> filesystem.rs; 2096-2238: hex/streaming Sha256 -> sha256.rs; 2239-2246,2347-2532: Observation, inspect/validate/decode/slot/wait/confirm project incarnation -> project.rs; 2247-2346: podman/podman_streamed/wait_output with O_CLOEXEC, drain and deadline precedence -> command.rs; 14-55,2533-2573: BUS/INSTALL/DESTINATIONS and ensure_system_bus/stage_tools -> stage.rs; 2574-2683: feed_archive/emit_archive/tar_header and existing deadline/CopyN/USTAR byte contract -> archive.rs; 22,2684-2841: prepare_interface/FdGuard/open_tree/mount_setattr/attach_project_mount; exact incarnation recheck before unshare/setns -> interface.rs; 2842-2892: public_socket_directory/validate_public_socket/validate_interface_directory -> interface_admission.rs; 2893-2937,3451-3498,3896-3946: existing TestDir and synthetic release/archive fixture helpers -> test_support.rs; keep release-specific helpers with release_tests if simpler; 2975-2991: quote vectors -> config_tests.rs; 3027-3041 and 2992-3026: path/errno vectors -> filesystem_tests.rs; 3042-3119: flags -> options_tests.rs; 3120-3245,3616-3665: config/error-order vectors -> config_tests.rs; 3246-3450: netip prefix/address matrix -> network_tests.rs; 3499-3615: payload admission/image conflict vectors -> release_tests.rs; 3666-3762: strict observation/isolation vectors -> project_tests.rs; 2938-2974,3763-3819: streaming SHA and root-owned pinned tool admission -> filesystem_tests.rs; 3820-3871: public launch socket/directory protection -> interface_tests.rs; 3872-3895,3947-4075: USTAR/install replacement/streaming/wait-output/short-read cases -> archive_tests.rs.

Open detail: Host config uses duplicate-last-wins/null-no-op/trailing-data/error-priority behavior; release and observation decoders have stricter policies. These remain separate bindings over the existing scanner. Consolidation with other crates or replacing the parser requires further behavior comparison, outside this physical split plan.

### rust/soda-muse/src/main.rs

Observed size: 1637 lines, including tests where embedded. Split 1,233 production lines along the existing launcher/helper responsibilities. Keep command/native-action selection in main; shell input/validation in shell; seqpacket+FD passing/signal/resize forwarding in launch; credential-view admission/native exec/environment in execution; account marker/passwd checks in account; config copy/read in config; pinned-version/private clean subprocess in runtime; Go path/errno/quoting semantics in paths. Exact request/exit output codec stays distinct from scalar JSON token scanning. Split the 401-line test module into shell/wire/paths, FD/signal transport, and config/account/native-dispatch tests. Preserve one launch-only socket and broker confirmation gates.

- `cmd/soda-muse/src/main.rs`
- `cmd/soda-muse/src/shell.rs`
- `cmd/soda-muse/src/launch.rs`
- `cmd/soda-muse/src/execution.rs`
- `cmd/soda-muse/src/account.rs`
- `cmd/soda-muse/src/config.rs`
- `cmd/soda-muse/src/runtime.rs`
- `cmd/soda-muse/src/paths.rs`
- `cmd/soda-muse/src/launch_wire.rs`
- `cmd/soda-muse/src/launch_json.rs`
- `cmd/soda-muse/src/shell_tests.rs`
- `cmd/soda-muse/src/launch_tests.rs`
- `cmd/soda-muse/src/config_tests.rs`

Evidence: 14-79: main/run/native_action/metadata_action -> main.rs; 80-187: ShellRequest, shell_request/validation and path/text/argument admission -> shell.rs; 242-405: seqpacket_connect/send_with_fds/launch_shell/signal controls -> launch.rs; 406-547: execute/await_admission/execution_state/private runtime directory and fixed environment -> execution.rs; 548-631,661-664: Go path/error/rune quoting -> paths.rs; 632-660,665-734: account_for/account_entry/account_node/lookup_user -> account.rs; 735-831: check_runtime/make_private_dir/run_pristine -> runtime.rs; 832-954: copy_config tree/filter/file and read_config settings/trust view -> config.rs; 188-241,955-1079: shell request serialization/base64/json_string/LaunchExit object decode -> launch_wire.rs; 1080-1233: parse_json_string/parse_json_integer/skip_json_value -> launch_json.rs; 1234-1403: ShellRequest fixture/wire/validation/exit/base64/path/quote/environment cases -> shell_tests.rs; 1469-1617: FD passing/exit and signal forwarding -> launch_tests.rs; 1404-1468,1618-1637: config copy/read/account safety/native action errors -> config_tests.rs.

### rust/soda-project-terminal/src/broker.rs

Observed size: 1342 lines, including tests where embedded. This is the fixed project-local identity execution helper inside project-terminal, distinct from the host soda-identity custody service. Split 1,118 lines before tests into existing JSON/wire helpers, reserve/private-profile preparation, stored profile/binding/unit-incarnation observation, protected credential seed/capture, pinned harness start, cgroup freeze/control, and confirmed finish/stop/model retirement. broker.rs retains dispatch plus bounded stdin/stdout, 45-second alarm, exclusive TERMINALS parent lock and fixed non-secret stderr. Preserve prepare cleanup precedence, freeze before capture, kill before stop, retirement only after confirmed stop, and Python-compatible numeric/dict semantics. Its 223-line existing test module remains together; no new subscription operation or separate daemon is proposed.

- `cmd/soda-project-terminal/src/broker.rs`
- `cmd/soda-project-terminal/src/subscription_wire.rs`
- `cmd/soda-project-terminal/src/subscription_prepare.rs`
- `cmd/soda-project-terminal/src/subscription_profile.rs`
- `cmd/soda-project-terminal/src/subscription_credentials.rs`
- `cmd/soda-project-terminal/src/subscription_start.rs`
- `cmd/soda-project-terminal/src/subscription_cgroup.rs`
- `cmd/soda-project-terminal/src/subscription_retire.rs`
- `cmd/soda-project-terminal/src/broker_tests.rs`

Evidence: 1-53,1011-1118: documented guest stdin/stdout protocol, subscription_dispatch, BrokerFail/read/write/alarm/broker_main; broker_inner owns exclusive parent lock across dispatch -> broker.rs; 60-240: json_equal/json_int/lease_with_binding/deadline_ok/result/decode_request/native_binding/profile_object -> subscription_wire.rs; 269-280,298-423: mount argv, subscription_prepare/provision, exact reserve/provision/cleanup order -> subscription_prepare.rs; 241-268,281-297,424-524,933-1010: paths/lease actor/execution/deadline, ProfileHit/subscription_profile, invocation/check_unit, lookup/resolve -> subscription_profile.rs; 525-600,773-800: no-follow owner changes/auth directory, subscription_seed/stage and capture -> subscription_credentials.rs; 601-689: root-owned harness digest, fixed respawn argv and subscription_start repeated binding/unit proof -> subscription_start.rs; 690-772: subscription_cgroup/kernel write/cgroup.events parse and confirmed freeze -> subscription_cgroup.rs; 801-932: bounded missing-ok removal/is_mount, subscription_retire and subscription_finish; preserve finally kill/stop/body-error precedence -> subscription_retire.rs; 1119-1342: numeric deep equality/coercion, binding/deadline/decode/result/argv/cgroup/mount/removal/dispatch cases -> broker_tests.rs.

### rust/soda-project-terminal/src/fs.rs

Observed size: 534 lines, including tests where embedded. Keep the cohesive descriptor-relative protected record/filesystem boundary: root-chain/regular-file UID/mode/link checks, bounded record read, create/replace/unlink, chmod/chown and added full-fstat/read-up-to helpers. Production spans lines 1-269 with a small test-only predicate at 60-64; the embedded test module spans 270-534. Extract those tests and the test-only predicate instead of adding another filesystem wrapper layer or new permission rule.

- `cmd/soda-project-terminal/src/fs.rs`
- `cmd/soda-project-terminal/src/fs_tests.rs`

Evidence: 14-59,65-269: component checks, root_chain/root_file/read_record/new_file and descriptor-relative operations; fstat_all at 238 and read_up_to at 247 remain reusable actual source helpers; 60-64,270-534: test-only s_isdir plus root-chain/safety/file/record/create/wrapper/chown/chmod cases -> fs_tests.rs.

### rust/soda-project-terminal/src/keys.rs

Observed size: 614 lines, including tests where embedded. Split 442 lines before tests into canonical managed authorized-key line parsing, strict request/truthiness/duplicate handling, and the existing locked filesystem preview/replace plus fixed CLI failure/result boundary. Keep update and replace_inner together: stable directory lock, login-to-identity admission, preview hash, exclusive temporary creation, fsync, bytes+FileId reread before rename, parent fsync and result reread form one atomic publication path. Extract the 171-line existing tests without inventing a new SSH parser, update mode or validation policy.

- `cmd/soda-project-terminal/src/keys.rs`
- `cmd/soda-project-terminal/src/key_request.rs`
- `cmd/soda-project-terminal/src/key_lines.rs`
- `cmd/soda-project-terminal/src/keys_tests.rs`

Evidence: 1-37,39-112: fixed limits/key algorithm line shape, canonical_key_ok/canonical_lines -> key_lines.rs; 113-239: Python truthiness, decoded duplicate rejection, KeyRequest/decode_key_request/state_object -> key_request.rs; 240-380: protected keys directory, FileId/read_keys/candidate_name/update/replace_inner -> keys.rs; 381-442: bounded stdin/write/EPIPE/fixed-stderr key_main/key_inner -> keys.rs; retain with atomic operation boundary; 443-614: key shape/canonical lines/truthiness/request/candidate/state and unprivileged refusal -> keys_tests.rs.

### rust/soda-project-terminal/src/pty.rs

Observed size: 1002 lines, including tests where embedded. Split the 756-line production PTY implementation into public ready/output/closed orchestration, fork/login/resize/child termination, relay control/backpressure/deadline handling, and raw descriptor/readiness I/O. Keep relay_pty_session as the existing loop over apply/ingest/read/stop/flush helpers; do not add another session state machine. Preserve no terminal bytes in diagnostics, parent size gate before child exec, heartbeat/lifetime caps and existing EINTR behavior. Extract the 244-line tests as one module.

- `cmd/soda-project-terminal/src/pty.rs`
- `cmd/soda-project-terminal/src/pty_process.rs`
- `cmd/soda-project-terminal/src/pty_relay.rs`
- `cmd/soda-project-terminal/src/pty_io.rs`
- `cmd/soda-project-terminal/src/pty_tests.rs`

Evidence: 23-54,115-150,621-655,693-756: stop signal handling, output frames, bounded closed flush and run_terminal orchestration -> pty.rs; 151-332: tmux attach argv, size, child wait/end and spawn_login_pty with fork/ready gate -> pty_process.rs; 333-443,519-620: apply_control_frame/ingest/read output/session_should_stop/take_control_input/flush queues/relay loop -> pty_relay.rs; 55-114,444-518,656-692: nonblocking/read/write/select/wait_readable -> pty_io.rs; 757-1002: exact frames, ingest/backpressure/deadlines, child lifecycle/resize/bounds cases -> pty_tests.rs.

### rust/soda-project-terminal/src/svc.rs

Observed size: 882 lines, including tests where embedded. Separate owned systemd unit field/argv/state/stop handling, cgroup filesystem/emptiness admission, tmux socket ownership/peer identity, and tmux subprocess control. Production is 665 lines; tests are 215 and can remain together while moved methods keep private access inside the same crate. Preserve stop confirmation with empty cgroup, no-follow descriptors, socket peer PID/UID/inode checks and no arbitrary control commands.

- `cmd/soda-project-terminal/src/svc.rs`
- `cmd/soda-project-terminal/src/cgroup.rs`
- `cmd/soda-project-terminal/src/socket.rs`
- `cmd/soda-project-terminal/src/tmux.rs`
- `cmd/soda-project-terminal/src/svc_tests.rs`

Evidence: 19-57,87-200: fixed unit naming/properties/argv, parse/verify service fields, state/confirmed stop -> svc.rs; 58-68,201-458: bounded statfs subprocess/read-only filesystem checks and cgroup_parent/events/empty -> cgroup.rs; shared existing wait/exit helpers stay crate-private to consumers; 15-18,459-604: SocketCheck/socket_identity_kinded/connect classification/exact socket_identity -> socket.rs; 69-86,605-665: infocmp/tmux fixed argv and tmux_control -> tmux.rs; 666-882: exact argv/fields/cgroup/socket/statfs/supervisor/tmux-negative vectors -> svc_tests.rs.

### rust/soda-project-terminal/src/term.rs

Observed size: 1821 lines, including tests where embedded. Split 1,363 lines before the embedded test module along already-defined terminal phases: held path/record helpers; account binding/reservation records and serialization; native status; confirmed finished-terminal collection; pinned-program reservation/create; exclusive writer attach/subscription lifetime; list/mutate/control dispatcher; and systemd post-start tmux preparation. Keep the parent TERMINALS shared/exclusive lock in control_terminal spanning its current operations, and keep the writer lock held throughout relay. Split 457 embedded-test lines into binding/reservation/status, exact argv/output/lifetime, and program-hash/native entry smoke groups. Do not move terminal transcript/bytes into persisted state or add lifecycle behavior.

- `cmd/soda-project-terminal/src/term.rs`
- `cmd/soda-project-terminal/src/term_paths.rs`
- `cmd/soda-project-terminal/src/term_binding.rs`
- `cmd/soda-project-terminal/src/term_status.rs`
- `cmd/soda-project-terminal/src/term_collect.rs`
- `cmd/soda-project-terminal/src/term_create.rs`
- `cmd/soda-project-terminal/src/term_attach.rs`
- `cmd/soda-project-terminal/src/term_prepare.rs`
- `cmd/soda-project-terminal/src/term_binding_tests.rs`
- `cmd/soda-project-terminal/src/term_protocol_tests.rs`
- `cmd/soda-project-terminal/src/term_native_tests.rs`

Evidence: 1066-1181: list_owned_terminals/mutate_terminal/control_terminal and parent lock scope -> term.rs; 33-146: terminal constants/path/checked_chain/list/fstatat/existence/output failure helpers -> term_paths.rs; 147-314,714-787: BindingRecord/Reservation validation/read/account match, JSON record/name emission -> term_binding.rs; 315-489,1243-1258: unit/state classification, ready parsing/writer/socket observation/status and ready_object -> term_status.rs; 490-657: terminal listing/stub retirement/screen and owned-file removal/collect_finished -> term_collect.rs; remove_owned_files is now crate-private for broker caller; 658-713,788-967: regular root-owned program hash/full-file verification and reserve/systemd_run/create -> term_create.rs; 968-1065: lease lifetime/subscription command and held-writer attach -> term_attach.rs; 1182-1242,1259-1363: tmux session argv/subscription setup/prepare/prepare_inner -> term_prepare.rs; 1364-1578: account/binding fixture, path/binding/account/reservation/status matrices -> term_binding_tests.rs; 1579-1736: exact systemd/tmux/subscription/lifetime/output vectors -> term_protocol_tests.rs; 1737-1821: full binary hash beyond 64KB, ready/native entry/tmux configuration cases -> term_native_tests.rs; src/main.rs:18-30 wires term/broker/keys into the fixed single argv-dispatched project-terminal binary; retain this crate and installed binary identity.

### rust/soda-project-terminal/src/timex.rs

Observed size: 403 lines, including tests where embedded. The 234-line production ISO deadline parser is one cohesive grammar with integer date math and explicit documented CPython deviations. Extract the 167-line vectors instead of fragmenting date/time/offset into tiny files. Preserve required timezone, truncation toward zero, accepted offset forms, and the existing rejection of naive/week-date/unsplit inputs.

- `cmd/soda-project-terminal/src/timex.rs`
- `cmd/soda-project-terminal/src/timex_tests.rs`

Evidence: 1-31: exact CPython grammar and three documented deviations; 33-234: now_secs/civil date, parse_date/time/offset and parse_iso_deadline; 235-403: CPython accepted/rejected vectors, deviations, civil math and current-clock case -> timex_tests.rs.

### rust/soda-rotate-lab-creds/src/main.rs

Observed size: 831 lines, including tests where embedded. Separate read-only lab inventory/runbooks from the explicitly acknowledged fixture-key rotation operation and process/status helpers. Retain class-specific acknowledgement and no-secret-output behavior. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/lab-credentials/src/main.rs`
- `tools/lab-credentials/src/process.rs`
- `tools/lab-credentials/src/inventory.rs`
- `tools/lab-credentials/src/fixture_authority.rs`
- `tools/lab-credentials/src/runbooks.rs`
- `tools/lab-credentials/src/tests.rs`

Evidence: rust/soda-rotate-lab-creds/src/main.rs:168-334 process capture and trust/config output; rust/soda-rotate-lab-creds/src/main.rs:337-499 runbooks and credential file metadata inventory; rust/soda-rotate-lab-creds/src/main.rs:537-622 fixture-only authority rotation; rust/soda-rotate-lab-creds/src/main.rs:624-708 command dispatch and cleanup; rust/soda-rotate-lab-creds/src/main.rs:711-831 serialization/metadata/helper tests.

### rust/soda-setup/src/main.rs

Observed size: 1731 lines, including tests where embedded. Separate existing CLI/origin admission, Forgejo bootstrap HTTP, local JSON decoding/config emission, PostgreSQL/grant secrets and setup/revocation orchestration. Keep native ownership helpers local and group the 541 lines of existing tests by those concerns. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-setup/src/main.rs`
- `cmd/soda-setup/src/cli.rs`
- `cmd/soda-setup/src/origin.rs`
- `cmd/soda-setup/src/forgejo.rs`
- `cmd/soda-setup/src/json.rs`
- `cmd/soda-setup/src/config.rs`
- `cmd/soda-setup/src/secrets.rs`
- `cmd/soda-setup/src/setup.rs`
- `cmd/soda-setup/src/system.rs`
- `cmd/soda-setup/src/tests/mod.rs`
- `cmd/soda-setup/src/tests/fixtures.rs`
- `cmd/soda-setup/src/tests/setup.rs`
- `cmd/soda-setup/src/tests/postgres.rs`
- `cmd/soda-setup/src/tests/admission.rs`
- `cmd/soda-setup/src/tests/encoding.rs`

Evidence: rust/soda-setup/src/main.rs:79-246 entry and Go-style CLI parsing; rust/soda-setup/src/main.rs:247-333 origin and bootstrap credential admission; rust/soda-setup/src/main.rs:334-563 Forgejo user/token HTTP path; rust/soda-setup/src/main.rs:567-843 local JSON parse and string emission; rust/soda-setup/src/main.rs:845-905 dashboard config encoding; rust/soda-setup/src/main.rs:907-1067 randomness/encoding/PostgreSQL secret generation/reuse; rust/soda-setup/src/main.rs:1090-1187 setup publication/revocation and exclusive output; rust/soda-setup/src/main.rs:1190-1731 HTTP fixtures, secret/admission/encoding tests.

Open detail: The local setup decoder, console parser and shared soda-json do not establish a single common acceptance/error contract. Keep local behavior until source/caller semantics are reconciled.

### rust/soda-stage-render/src/provisioning.rs

Observed size: 614 lines, including tests where embedded. Separate current Python-compatible JSON/document construction from private-input/host-public-key/output handling and render orchestration. Keep secret-file handling and exclusive destination admission. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/render/provisioning/mod.rs`
- `tools/release-assets/src/render/provisioning/document.rs`
- `tools/release-assets/src/render/provisioning/private_files.rs`
- `tools/release-assets/src/render/provisioning/render.rs`
- `tools/release-assets/src/render/provisioning/tests.rs`

Evidence: rust/soda-stage-render/src/provisioning.rs:79-144 regular/private file and hostname input checks; rust/soda-stage-render/src/provisioning.rs:146-331 Python JSON emission, file entries and public template; rust/soda-stage-render/src/provisioning.rs:336-469 public host-key derivation and exclusive output; rust/soda-stage-render/src/provisioning.rs:471-564 RenderInputs and render; rust/soda-stage-render/src/provisioning.rs:566-614 hostname and exact JSON tests.

### rust/soda-stage-render/src/stage.rs

Observed size: 518 lines, including tests where embedded. Separate confined destination copying/tree normalization from current payload/terminal locks and host/Forgejo branding adaptation. Retain one run sequence and the existing current generated browser inputs. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/render/stage/mod.rs`
- `tools/release-assets/src/render/stage/files.rs`
- `tools/release-assets/src/render/stage/payload.rs`
- `tools/release-assets/src/render/stage/branding.rs`
- `tools/release-assets/src/render/stage/tests.rs`

Evidence: rust/soda-stage-render/src/stage.rs:36-161 native platform, directory/copy/normalization helpers; rust/soda-stage-render/src/stage.rs:162-262 payload manifest/terminal pins/favicon/build-source resolution; rust/soda-stage-render/src/stage.rs:264-478 context admission and appliance/Forgejo/terminal staging; rust/soda-stage-render/src/stage.rs:482-518 ICO layout and source-token tests.

Open detail: Source-root detection and payload/asset path strings must change together with the project-wide tree. Existing runtime binary destinations remain the installed contract.

### rust/soda-stage-render/src/terminal_logo.rs

Observed size: 630 lines, including tests where embedded. Separate the existing restricted SVG/XML cursor from polygon/raster-to-text geometry and output/check orchestration. Preserve canonical emblem syntax and exact text output. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/render/terminal_logo/mod.rs`
- `tools/release-assets/src/render/terminal_logo/svg.rs`
- `tools/release-assets/src/render/terminal_logo/geometry.rs`
- `tools/release-assets/src/render/terminal_logo/tests.rs`

Evidence: rust/soda-stage-render/src/terminal_logo.rs:21-119 path tokenization and polygon extraction; rust/soda-stage-render/src/terminal_logo.rs:121-176 evenodd raster and terminal coloring; rust/soda-stage-render/src/terminal_logo.rs:178-462 restricted SVG cursor/parser; rust/soda-stage-render/src/terminal_logo.rs:464-516 render_svg and canonical output/check; rust/soda-stage-render/src/terminal_logo.rs:519-630 tokenizer/geometry/SVG/output tests.

### rust/soda-stage-render/tests/cli.rs

Observed size: 768 lines, including tests where embedded. Split the three CLI suites at their existing labeled sections and share only the current TempDir/command/copy helpers. Keep golden provisioning files and canonical branding comparisons as their existing byte assertions.

- `tools/release-assets/tests/render_staging.rs`
- `tools/release-assets/tests/render_provisioning.rs`
- `tools/release-assets/tests/render_terminal_logo.rs`
- `tools/release-assets/tests/render_support/mod.rs`

Evidence: rust/soda-stage-render/tests/cli.rs:22-118 common fixture, root lookup, binary and command helpers; rust/soda-stage-render/tests/cli.rs:120-284 soda-stage CLI suite; rust/soda-stage-render/tests/cli.rs:286-670 soda-render-provisioning CLI suite; rust/soda-stage-render/tests/cli.rs:672-768 soda-render-terminal-logo CLI suite.

Open detail: repo_root currently assumes the crate sits two levels below the checkout root at lines 64-69. Fixture paths include the old payload manifest path and require coordinated updates.

### rust/soda-test-vm/src/main.rs

Observed size: 768 lines, including tests where embedded. Separate shell-status/exec/diagnostic helpers, current pidfile/liveness state, locked QEMU start, and SSH/tunnel/console actions. Keep the exact native/KVM gates and fixed management forwarding. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/test-vm/src/main.rs`
- `tools/test-vm/src/process.rs`
- `tools/test-vm/src/state.rs`
- `tools/test-vm/src/start.rs`
- `tools/test-vm/src/transport.rs`
- `tools/test-vm/src/tests.rs`

Evidence: rust/soda-test-vm/src/main.rs:132-363 process/access/exec diagnostics and signal handling; rust/soda-test-vm/src/main.rs:365-449 pidfile state, VM directory and SSH arguments; rust/soda-test-vm/src/main.rs:451-535 lock/native-input/QEMU start; rust/soda-test-vm/src/main.rs:537-624 status and transport action dispatch; rust/soda-test-vm/src/main.rs:652-768 pid/SSH/status diagnostics tests.

### rust/soda-test-vm/tests/cli.rs

Observed size: 777 lines, including tests where embedded. Group existing status/pidfile, native start/lock/QEMU argument, and exec/SSH/tunnel tests; retain CLI/default-action checks separately and share existing fake commands/VM fixtures.

- `tools/test-vm/tests/cli.rs`
- `tools/test-vm/tests/status.rs`
- `tools/test-vm/tests/start.rs`
- `tools/test-vm/tests/transport.rs`
- `tools/test-vm/tests/support/mod.rs`

Evidence: rust/soda-test-vm/tests/cli.rs:20-82 KVM gate, TempDir, command and fake VM helpers; rust/soda-test-vm/tests/cli.rs:84-169 usage/status/pidfile tests; rust/soda-test-vm/tests/cli.rs:170-566 start host/input/lock/QEMU/refusal tests; rust/soda-test-vm/tests/cli.rs:568-725 SSH failure and exact transport argument tests; rust/soda-test-vm/tests/cli.rs:727-777 working-directory and closed-stdout checks.

### scripts/forgejo_components_test.go

Observed size: 691 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `scripts/forgejo_components_test.go` — Native intro/empty/explore composition checks.
- `scripts/forgejo_theme_components_test.go` — Guest/theme-toggle singleton and route selection cases.
- `scripts/forgejo_form_components_test.go` — Original creation permission/form composition and adapter boundaries.
- `scripts/forgejo_template_fixture_test.go` — Shared native template dictionary/read/call fixtures.

Evidence: TestForgejoPageIntroComposition at 48; TestForgejoExplorePagesComposeNativeControlsWithOriginalContext at 204; TestForgejoThemeToggleIsSingletonAtEachPlacement at 347; TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes at 421; TestForgejoRepositoryCreationKeepsNativePermissionBranches at 473; TestForgejoNativeFormAdapterSelectsMainFormsOnly at 627; forgejoTemplateDict at 33; readForgejoTemplate at 662; requireForgejoTemplateCalls at 682.

### tests/build/test_project_keys.py

Observed size: 416 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to the native owner and live tests
listed in [Python cutover closure](#python-cutover-closure).
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: test_project_keys.py:20-55 real source loader/setup; 56-142 basic preservation/publication; 143-300 writer/admission races; 301-397 uncertain publication/cleanup; 398-416 CLI/refusal.

### tests/build/test_terminal.py

Observed size: 817 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to the native owner and live tests
listed in [Python cutover closure](#python-cutover-closure).
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: test_terminal.py:1-5 limited proof contract; 22-25 real-source load; 28-151 TerminalProtocol; 154-203 managed fixture; 204-600 admission/supervision tests; 602-817 LocalTerminalProcess.

### tests/forgejo/component-boundaries.test.ts

Observed size: 510 lines, including tests where embedded. Split current nested component subtests by layout and control concerns; keep their existing native CSS fixture and authorization guard.

- `tests/forgejo/component-boundaries.test.ts` — Existing native form focus/icon/help/heading and settings-panel/table boundary assertions.
- `tests/forgejo/component-toolbar-boundaries.test.ts` — Existing repository toolbar/control size/button state and adjoining-input assertions.
- `tests/forgejo/component-layout-boundaries.test.ts` — Existing compact/empty/fluid/repository status/package/profile narrow-layout assertions.
- `tests/forgejo/fixtures/component-browser.ts` — Extract current guarded native origin, real palette/styles loading, sandboxed browser/page and render fixture.

Evidence: component-boundaries.test.ts:9-29 guarded fixture; 30-154 form/heading/focus/empty; 155-277 toolbar/button sizes; 278-359 table/fluid/status/settings; 360-448 primary states; 449-510 packages/cleanup.

Open detail: The shared stylesheet loader must include actual split canonical files and preserve existing style order. Do not broaden to installed-product proof or introduce new test contracts.

### tests/frontend/drawer-controls.test.ts

Observed size: 630 lines, including tests where embedded. Split regression cases along the production environment/access/control responsibilities using the existing drawer-fixture as the real browser entry.

- `tests/frontend/drawer-controls.test.ts` — Existing inert mount/visibility/tab/actor retirement, draft preservation and no terminal ownership tests.
- `tests/frontend/project-environment-controls.test.ts` — Existing explicit Create/network/OS/Start/Stop observations and confirmed administration controls.
- `tests/frontend/project-access-controls.test.ts` — Existing deliberate Join, key selection/review/Apply/save/refusal/copy tests.
- `tests/frontend/fixtures/project-controls-driver.ts` — Extract existing drawer browser/server fixture, refresh/click helpers and cleanup once.

Evidence: drawer-controls.test.ts:7-65 browser/server/driver; 67-126 inert controls; 127-325 Create/network/OS/lifecycle; 326-451 keys/Join; 452-629 uncertainty/retirement/hidden controls/identity.

Open detail: Fixture extraction must not repeat production logic or reset browser state between assertions that currently form one regression scenario.

### tests/frontend/terminal.test.ts

Observed size: 559 lines, including tests where embedded. Split existing protocol, retirement, explicit End and stream regressions; use the current terminal browser fixture rather than a new terminal model.

- `tests/frontend/terminal.test.ts` — Existing inert exact-locator mount, reserve/create/attach readiness, generation/binding and native observation contracts.
- `tests/frontend/terminal-retirement.test.ts` — Existing visibility/focus/actor/late import/reservation/renderer/socket retirement cases.
- `tests/frontend/terminal-end.test.ts` — Existing separately confirmed End and unconfirmed/absent native outcome handling.
- `tests/frontend/terminal-stream.test.ts` — Existing Unicode IO, bounded queues, overload and invalid-frame cases.
- `tests/frontend/fixtures/terminal-driver.ts` — Extract current terminal-fixture browser/server setup, action/open/ready/End helpers and cleanup once.

Evidence: terminal.test.ts:11-100 current fixture/helpers; 102-182 mount/readiness/subURL; 183-309 retirement/admission; 310-357 End; 358-520 identity/visibility/observation/retired callbacks; 521-559 bounds/frames.

Open detail: A single scenario spanning readiness and retirement stays intact; splitting file placement cannot weaken original actor/native-generation assertions.

### tests/frontend/workspace.test.ts

Observed size: 1843 lines, including tests where embedded. Split the many current regression groups by established workspace concern. Keep shared driver and actual emitted component fixture; do not copy the product state machine into new tests.

- `tests/frontend/workspace.test.ts` — Existing page/panel shared original-target Create/End and native drawer handoff/renderer continuity cases.
- `tests/frontend/workspace-attention.test.ts` — Existing unread coalescing, authorized stable attention order and stale-generation observations.
- `tests/frontend/workspace-navigation.test.ts` — Existing project/detail/search/This-page switching, actor invalidation, pending rename and Hide ownership cases.
- `tests/frontend/workspace-persistence.test.ts` — Existing exact arrangement restoration, obsolete/corrupt caches and failed storage cases.
- `tests/frontend/workspace-layout.test.ts` — Existing real xterm owner retention through splits/moves/dividers/overflow/maximize and measurement retirement.
- `tests/frontend/workspace-first-use.test.ts` — Existing complete selected-repository welcome/Create/Join/terminal/re-entry scenario.
- `tests/frontend/workspace-setup.test.ts` — Existing pending creation, repository keyboard search/back/change, superseded choice and second-project cancellation cases.
- `tests/frontend/workspace-recovery.test.ts` — Existing stopped/incomplete/failed/native-generation/private-context recovery and partial collection cases.
- `tests/frontend/workspace-responsive.test.ts` — Existing welcome/intro/coherence short viewport, theme/native CSS precedence and long-name menu/focus assertions.
- `tests/frontend/fixtures/workspace-driver.ts` — Extract current browser/server/fixture lifetime and real UI action helpers once, importing the existing workspace-fixture.

Evidence: workspace.test.ts:11-146 fixture/driver; 148-293 drawer/layout surfaces; 294-404 attention; 405-556 exact operations/navigation; 557-640 persistence; 641-766 layout; 767-893 installed controls/measurement; 894-955 first-use helpers; 956-1235 complete first use; 1236-1463 keyboard/setup/responsive; 1464-1596 state/Join/cancellation/inventory; 1597-1843 responsive/native recovery/generation/CSS/partial inventory.

Open detail: Some existing parameterized scenarios cover several concerns and must remain whole. Browser/server hooks and shared mutable fixture lifetime need isolation review when distributing test modules; do not assume parallel execution is safe.

### tests/installed/sodaspaces-workspace-journey.ts

Observed size: 441 lines, including tests where embedded. Separate the two already exported installed scenarios and their shared evidence records; retain their guarded native callers and independent local-fixture proof limits.

- `tests/installed/sodaspaces-workspace-journey.ts` — Existing workspace matrix scenario, exact native transport observation and bounded UI operations.
- `tests/installed/sodaspaces-first-use-journey.ts` — Existing selected-actor/repository Create, keyless Join, terminal and re-entry first-use scenario.
- `tests/installed/sodaspaces-journey-evidence.ts` — Move existing FirstUseEvidence/MatrixSession/MatrixFacts/MatrixEvidence/MatrixNative definitions and native mount identity check to one shared owner.

Evidence: sodaspaces-workspace-journey.ts:1-2 proof scope; 9-24 native mount/session check; 26-171 FirstUseEvidence/exerciseFirstUse; 172-210 matrix records; 211-441 exerciseWorkspaceMatrix.

Open detail: Update actual installed and fixture imports directly without alias modules. Splitting retained scenario source does not establish new installed qualification.

### tools/soda-candidate/display.go

Observed size: 415 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: renderer at 114; buildLines at 250; startTicker at 303; parseEvent at 31; parseArtifactEvent at 93; hostArtifactPath at 81; printWhyPanelLocked at 356; exitMeaning at 373; wallDur at 384.

### tools/soda-candidate/main_test.go

Observed size: 619 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: TestValidateArchFlagAdmitsOnlyX8664 at 16; TestResolveOptionsDefersWorkerAdmissionToController at 552; TestOverviewStartsWhenValid at 113; TestOverviewBlocksBadStartWithoutLosingAnswers at 133; TestServeAndFileRootfs at 254; TestFileBuiltRootfsCoversNonLoopbackMedia at 292; TestParseControllerEvents at 27; TestRunningPhaseShowsLiveElapsed at 379; TestFailedRunPrintsWhyPanelWithHostPaths at 397; TestCheckCleanTreeRefusesUntrackedFiles at 319; TestCopyFileRefusesOccupiedPickup at 349; TestReadyRunTouchesNoWorkerState at 573.

### tools/soda-candidate/prompts.go

Observed size: 425 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: prompter at 17; line at 56; choice at 76; askAbsolute at 333; askOut at 380; overviewFields at 130; baseFields at 140; mediaFields at 151; editMode at 160; overview at 240; renderOverview at 272; dispatchOverviewCmd at 288; startIfValid at 299.

### rust/soda-release-build/src/coreos.rs

Observed size: 476 lines, including tests where embedded. Keep the existing ordered QEMU authentication chain, its CoreOSImage/VerifiedBase records, bounded HTTPS copy and three small unit cases together. Extract the command drain used by both xz expansion and gpgv verification into a private CoreOS child; its timeout/nonblocking/output-limit mechanics are a real shared concern. This reduces the parent without inventing a general executor or changing the download, checksum, signature, expansion and retained-record order.

- `lib/soda-release-build/src/coreos.rs` — Existing records, native/signer/keyring admission, exclusive bounded HTTPS download, compressed checksum and detached-signature binding, QEMU expansion/hash and verified-base record; retain the current three tests here.
- `lib/soda-release-build/src/coreos/process.rs` — Existing run_bounded and set_nonblocking implementation used by xz and gpgv; child visibility only as required by those callers.

Evidence: 15-64: CoreOSImage/VerifiedBase and emit/marshal; 65-85: https_url/valid_signer/admit_coreos_fetch; 86-165: download_http/download/download_verified_archive; coreos_iso.rs:4,26-38 uses the same download/signature closure; 166-268: run_bounded/set_nonblocking; 269-315 and 365-402 call it for xz and gpgv; 316-364: fetch_coreos/fetch_coreos_with; 404-476: URL, exclusive bounded-download and record-shape unit cases

Open detail: Do not substitute BuildExecution or image run_build_command merely because all spawn processes: this drain carries different fixed limits/timeouts. Its helper is private, not a new public or service boundary.

### rust/soda-release-build/src/coreos_stream.rs

Observed size: 856 lines, including tests where embedded. Separate admitted live-input records and validation, Tailnet release/base selection, CoreOS stream selection and registry image-index resolution. Reuse the existing http module for the already shared capped metadata/text fetch rather than create another transport package. Keep test fixtures shared only where production unit tests already consume them; retain actual resolver and validation cases with their owner.

- `lib/soda-release-build/src/coreos_stream.rs` — Stable CoreOS stream endpoint, stream document and release-location parsing, stream triple validation, ISO/QEMU/full resolution entrypoints and their existing stub cases.
- `lib/soda-release-build/src/live_inputs.rs` — TailnetInputs/ResolvedCoreOS/LiveInputs emit/decode, exact write/read and admitted-input validation; existing round-trip/refusal case stays with this owner.
- `lib/soda-release-build/src/tailnet_inputs.rs` — Tailnet endpoints, newest stable archive selection, checksum/base-tag selection, resolve_tailnet_inputs and its existing release-selection case.
- `lib/soda-release-build/src/coreos_registry.rs` — Existing registry image-index request, bounded body decode and matching x86_64 digest selection.
- `lib/soda-release-build/src/http.rs` — Existing transport plus current fetch_capped_json/fetch_capped_text helpers reused by CoreOS and Tailnet callers.
- `lib/soda-release-build/src/test_support.rs` — Current cfg(test) fixture_live_inputs used by coreos_stream tests and production.rs:843; keep fixture ownership test-only.

Evidence: 24-65: current endpoint accessors; 67-257: TailnetInputs/ResolvedCoreOS/LiveInputs plus decode/read/write/validation; 258-301: fetch_capped_json/fetch_capped_text; existing http.rs is 296 lines including its current unit tests; 302-438: Tailnet resolution, release scan and newest base tag; 439-559: stream document/location/triple parsing; 560-645: resolve_registry_digests_with; 646-706: ISO/QEMU/full CoreOS resolution; 707-856: fixture_live_inputs, stream/index fixtures and four existing cases; production.rs:843-844 imports fixture/read/write

Open detail: Existing release-inputs reader validators are used here at 198-203,216-228,544-556 and693. Keep those actual calls; moving types does not prove the independently mirrored image DTOs equivalent.

### rust/soda-release-build/src/files.rs

Observed size: 621 lines, including tests where embedded. Separate ordinary artifact inventory/output operations from the descriptor-relative directory owner and the strict bounded JSON read. Keep read_json_at, exact-byte digest and its trailing-data scanner together. Root metadata/inode comparison belongs with confined operations; never replace these checks with an unconstrained path read. Unit tests may be extracted as cfg(test) descendants while preserving their actual implementation inputs.

- `lib/soda-release-build/src/files.rs` — File inventory emit/decode, architecture/revision/digest admission, regular-file hash, fresh/private output admission, exclusive write/chmod and absolute-path helper.
- `lib/soda-release-build/src/confined_files.rs` — Root/FileMeta, openat traversal, no-follow lstat/open, inode identity checks and hash_at; shared with OCI layout reads and strict JSON input.
- `lib/soda-release-build/src/json_input.rs` — read_json/read_json_at with the 4 MiB bound, regular/inode check, exact-byte hash, strict binder and existing first-value/trailing-data classifier.
- `lib/soda-release-build/src/files/tests.rs` — Existing validator/hash/directory/exclusive-write/JSON/root-escape cases; retain unit scope and real Root reads.

Evidence: 18-83: File inventory and admission; 84-109: hash_file/hash_at;110-165: output admission/write/chmod;476-483: abs_path; 166-221: read_json/read_json_at;222-388: Root/FileMeta and descriptor-relative helper closure; 389-475: first_json_end/skip_ws/skip_string/skip_number/skip_literal/skip_value, used only to classify trailing input; 484-621: six existing unit cases; oci_layout.rs:5-6,45-54,116 onward consumes Root and FileMeta

Open detail: Update current imports directly. Preserve duplicate/fold/null/unknown-field behavior from Strict and keep the JSON size, inode and exact-byte guarantees together.

### rust/soda-release-build/src/json_go.rs

Observed size: 451 lines, including tests where embedded. Split decoding/binding from byte emission at the existing Emit boundary. Keep the lenient Fields and strict Strict name/type handling together because they intentionally encode the same Go field semantics. Preserve ordered struct fields, sorted-map responsibility, HTML escaping and indent bytes; neither the generic JSON parser nor strictjson is automatically interchangeable.

- `lib/soda-release-build/src/json_go.rs` — FieldError/Fields plus Strict binder and diagnostics, exact-or-folded last-wins lookup, null handling and unknown-field finish.
- `lib/soda-release-build/src/json_emit.rs` — Emit values, sorted_object and marshal_indent/emit_value/emit_indent byte renderer.
- `lib/soda-release-build/src/json_go/tests.rs` — Existing marshal, unknown-field, folded/null and lenient-last-wins cases, kept unit-scoped and targeting the real decoder/emitter.

Evidence: 14-122: FieldError/Fields lenient extraction;125-312: Strict/json_kind/type and unknown-field diagnostics; 315-389: Emit/sorted_object/marshal_indent/emit_indent/emit_value; 390-451: four existing unit cases; coreos.rs,coreos_stream.rs,files.rs,forgejo.rs,production.rs import these real semantics

Open detail: The deliver crate jsonx and image crate jsonio also emulate Go JSON, but have different binders and emission APIs. This split alone does not establish safe consolidation.

### rust/soda-release-build/src/oci.rs

Observed size: 1286 lines, including tests where embedded. Keep one OCI inspection owner. Separate bounded outer archive/blob admission, index/manifest/config identity validation, single-layer whiteout/path semantics and multi-layer compressed-content resolution. Share Descriptor/Blob/Image with the existing oci_layout caller rather than copying them. Preserve two-pass archive verification, requested-file regularity, whiteout/opaque/ancestor behavior, trailer drain and conservative zstd refusal. Unit fixtures already shared with production and layout tests stay test-only.

- `lib/soda-release-build/src/oci.rs` — Image/Descriptor/Blob/LoadBlobs and existing inspect_oci/inspect_oci_content entrypoints orchestrating the real archive, manifest and layer owners.
- `lib/soda-release-build/src/oci/archive.rs` — Archive input admission, tar entry uniqueness/path/type gates, bounded blob copy/hash and outer archive collection.
- `lib/soda-release-build/src/oci/manifest.rs` — Layout/index/descriptor/manifest/config parsing, local bounded blob resolution, rootfs/layer/media/platform/source/base validation and image identity construction.
- `lib/soda-release-build/src/oci/layers.rs` — Requested member and layer path gates, whiteout/opaque/non-directory ancestor tracking, per-member hashing and one-layer tar scan.
- `lib/soda-release-build/src/oci/content.rs` — Descriptor-to-archive indexes, hash tee, gzip drain/zstd block handling, archive-layer collection and reverse overlay member resolution.
- `lib/soda-release-build/src/oci/tests.rs` — Existing identity/platform, content-member, scanner and gzip/zstd cases with their tar helper; preserve unit access to actual layer implementations.
- `lib/soda-release-build/src/test_support.rs` — Existing cfg(test) fixture_oci_bytes/FIXTURE_REVISION used by OCI, oci_layout.rs tests and production.rs:845; one shared fixture owner.

Evidence: 26-84: Image/Descriptor/Blob/LayerMember/LoadBlobs;85-294: archive/blob admission and read_oci_archive_entries; 295-548: read_oci_index/parse_oci_manifest/fetch_oci_blob/config/rootfs/attribution/inspect_oci_image;549-573: archive identity entrypoint; 574-782: requested_oci_paths/clean_layer_name/whiteout_target/record_* and scan_oci_layer; 783-952: layer_archive_indexes/HashReader/scan_layer_reader/scan_archive_layer/scan_oci_archive_layers/resolve_oci_members;953-1001: content inspection entrypoint; 1002-1286: shared OCI fixture and four current test groups; oci_layout.rs:5-6 reuses inspect_oci_image/read_oci_blob/read_oci_index/Blob/Image/LoadBlobs

Open detail: deliver/src/oci.rs ports the same Go archive/content surface; identify one surviving Rust owner at caller cutover, comparing guarantees first. Do not present another permanent OCI package or widen helpers solely for tests.

### rust/soda-release-build/src/production.rs

Observed size: 1179 lines, including tests where embedded. Split the existing Production methods by compiler/dependency recipes, public asset staging, resolved image admission and application image production/export. Keep one Production value and its private inputs across resolve_inputs and images, preserving one recorded attempt and execution order. Keep the substantial scripted fixture and all production-sequence/refusal cases in one unit descendant; no new orchestrator or generalized plugin layer follows.

- `lib/soda-release-build/src/production.rs` — Existing Production state, hooks, validation/step delegation, images entrypoint and Forgejo method delegation; shared ProducedImage result.
- `lib/soda-release-build/src/production_compile.rs` — Current Go/Cargo compile recipes, ELF/output mode checks, pinned Bun and dependency admission and Soda command discovery.
- `lib/soda-release-build/src/production_assets.rs` — Existing one-time frontend/terminal/locales/upstream-tools staging and project helper compilation sequence.
- `lib/soda-release-build/src/production_inputs.rs` — ResolvedInput record, frozen selection, pull/digest admission and app-inputs record, recipe/unit references, live Tailnet inputs and resolve_inputs.
- `lib/soda-release-build/src/production_images.rs` — Current Podman build/export recipe, Rocky base/project/dashboard/Forgejo/extension/proxy/Tailnet role sequence and lexical relative paths used by those recipes.
- `lib/soda-release-build/src/production/tests.rs` — Current production_fixture and seven scripted sequence/failure/refusal/compile/discovery cases; unit access to actual private Production state.

Evidence: 15-83: hooks/Production/state/validation;85-138: compile_rust/compile;167-204: pinned Bun/dependencies;805-838: soda_commands; 139-166,205-325: assets/asset_steps and current package/binary/path strings; 326-480,734-759,777-804: frozen/pulled/recorded inputs and recipe parsers;354-368: images public entrypoint; 481-714: build/export application image roles;715-749: Forgejo delegation and current result records;760-776: lexical_rel; 840-1179: fixture plus oracle_production_sequence/failure_stops/refusals/asset_destinations_refuse_early/compile_recipes/soda_commands/image_repo_parsing

Open detail: The planned release-assets package must update -p names and locale/source paths here plus recorded oracle strings. settings helpers duplicate release-inputs reader settings and image/sys command discovery; compare real errors/order before reuse. The Rust Production currently has no in-tree Cargo production dependent.

### rust/soda-release-build/src/progress.rs

Observed size: 857 lines, including tests where embedded. Separate monotonic timing/log transitions from subprocess execution, cancellation, output and structured exit observation. Preserve one timing origin and child log admission. Move only the SharedBuffer test writer to test support; unit and integration cases must continue exercising the real progress/execution owner. This is a module seam within the current crate, not another process service.

- `lib/soda-release-build/src/progress.rs` — BuildProgress, inherited monotonic origin, section/phase/reason/log transitions, summary/finish/error joining and build_exit_code.
- `lib/soda-release-build/src/build_execution.rs` — BuildExecution, Go tool selection, subprocess stream/capture, cancellation and native exit/signal observation.
- `lib/soda-release-build/src/progress/tests.rs` — Existing clock/origin/phase/reason/occupied-log/child and cancellation/compiler cases, with their current synthetic clock and progress fixture.
- `lib/soda-release-build/tests/support/buffer.rs` — Existing test-only SharedBuffer writer, shared by unit test inclusion and integration oracle modules without publishing test helpers.

Evidence: 14-353: BuildProgress and transitions/origin/summary/finish, error joins, new_build_progress/build_exit_code; 354-383: SharedBuffer;384-654: BuildExecution/environment/tool selection/execute/capture; 657-857: progress_fixture/fake_clock and six existing cases; tests/oracle.rs:426 onward uses progress bytes and shared output; image/build.rs:36-170,1122-1244 has a distinct runner/environment/execution implementation; tools/src/progress.rs and exitcode.rs duplicate timing/exit concerns

Open detail: No in-tree production caller currently constructs build crate BuildProgress/BuildExecution. tools progress is live in its CLI but pipeline dispatch is unfinished. Select one real Rust caller/owner at cutover; error metadata, environment pinning, streaming and cancellation differences prevent a blind helper swap.

### rust/soda-release-build/tests/oracle.rs

Observed size: 534 lines, including tests where embedded. Retain the frozen Go vectors and OCI fixture/layout bytes as test data while dividing the heterogeneous battery into four concern modules under one integration-test entrypoint. Reuse data_path/scratch and byte comparison helpers once. Each module still invokes the actual Rust owner; no Go implementation or temporary generator remains required for execution.

- `lib/soda-release-build/tests/oracle.rs` — Existing integration entrypoint, oracle_vectors import and shared data_path/scratch; declares the concern modules.
- `lib/soda-release-build/tests/oracle/oci.rs` — Existing frozen archive/layout identity/content/rejection cases and artifact validator vectors.
- `lib/soda-release-build/tests/oracle/inputs.rs` — Existing live-input/verified-base/resolved-input byte cases and strict bounded-JSON read assertions.
- `lib/soda-release-build/tests/oracle/production.rs` — Existing Forgejo argv/script/toolchain and scripted real Production sequence oracle.
- `lib/soda-release-build/tests/oracle/progress.rs` — Existing timing byte and structured exit-code oracle cases using the actual progress implementation.

Evidence: 1-53: frozen-vector imports/data_path/scratch;55-155: OCI archive/content/layout cases; 156-248: live-input/base/resolved-input serialization;249-284: Forgejo args/script/toolchain and ELF fixture; 285-425: scripted Production sequence;426-484: progress bytes and exit classes; 485-509: miscellaneous artifact/URL validators;510-534: read_json strictness; oracle_vectors.rs and tests/data remain existing frozen fixture leaves

Open detail: After an intentional CLI/path/package change, update the corresponding vector from the agreed current output contract. Deleting a Go predecessor does not justify deleting these Rust assertions or claiming their old captured recipes are still the new installed layout.

### rust/soda-release-deliver/src/buildx.rs

Observed size: 595 lines, including tests where embedded. Separate the inspected build identity from filesystem custody. Keep the fd, no-follow checks, same-inode observations, bounded reads and exact byte hashing in one filesystem module; do not spread the unsafe access sequence across generic helpers.

- `lib/soda-release-deliver/src/buildx/mod.rs` — Inspected image and pinned Forgejo toolchain types, validation/decoding/emission, shape reexports, and existing build-JSON error/byte adapters.
- `lib/soda-release-deliver/src/buildx/filesystem.rs` — Root owns the fd and every confined open/stat/read/hash/JSON read, plus fresh/private/create-new output admission and writes.
- `lib/soda-release-deliver/src/buildx/tests.rs` — Existing toolchain validation and file-helper error tests.

Evidence: 18-166: compiler pins, Image, ForgejoToolchain and APK provenance; 167-386: os_error, Root and confined readers/hash; 387-526: output admission, create-new write, decode_build_json/read_json_at; 527-537: unknown_field/decode_bytes_value; 538-595: two existing tests.

Open detail: Keep crate::buildx public paths through the module root; Root fields and fd/stat/open helpers remain inside filesystem. Reexport only current callers' existing functions. prepare.rs, admission.rs, document.rs and oci.rs continue to use these same custody operations. cfg(test) descendant tests retain real filesystem assertions.

### rust/soda-release-deliver/src/fetch.rs

Observed size: 467 lines, including tests where embedded. Durable state custody and authenticated image verification are separate existing responsibilities. Keep partial authenticated highwater results and error propagation intact; the caller still persists the returned state before propagating image verification failure.

- `lib/soda-release-deliver/src/fetch/mod.rs` — Discovery/document transport and fetch request/orchestration: admit, save authenticated progress before later failures, and emit verification-only receipt.
- `lib/soda-release-deliver/src/fetch/state.rs` — StateLock/drop, flock admission, fresh private temporary write, fsync/rename/directory sync.
- `lib/soda-release-deliver/src/fetch/verification.rs` — Architecture selection, verified image metadata/config binding, image copy checks, mixed-architecture identity tracker and highest authenticated state.
- `lib/soda-release-deliver/src/fetch/tests.rs` — Current lock/save serialization and request-refusal tests with their private-directory fixture.

Evidence: 19-41 discover; 42-50 init_state; 51-129 StateLock/lock_state/create_temp/save_state; 130-141 fetch_document; 142-358 resolve_verification_architectures through verify_releases; 359-418 admit_fetch_request/complete_fetch/fetch; 419-467 tests.

Open detail: Preserve crate::fetch::{discover,fetch_document,lock_state,save_state,verify_releases} visibility for publish.rs. Publish and Fetch share this existing lock/write code, not a new process or persistence abstraction. verification retains ReleaseIdentityTracker and refs_seen within its closure.

### rust/soda-release-deliver/src/jsonx.rs

Observed size: 623 lines, including tests where embedded. Group decode and emission by their actual wire duties. This adapter deliberately preserves distinct strict and lenient Go semantics; do not substitute one parser or introduce a new JSON crate merely to shorten the file. Keep emitter implementations beside its private state.

- `lib/soda-release-deliver/src/jsonx/mod.rs` — Current JSON contract constants, DecodeError and exact standard-base64 byte encoding/decoding; existing decode/emit public exports.
- `lib/soda-release-deliver/src/jsonx/decode.rs` — Recursive duplicate rejection, Binder/Soft with all their methods, strict/lenient parse, last-wins normalization and integer range checks.
- `lib/soda-release-deliver/src/jsonx/emit.rs` — Emitter private buffer/indent state, Emit trait and every scalar/map/array/JsonValue implementation, marshal.
- `lib/soda-release-deliver/src/jsonx/tests.rs` — Current base64 vectors and strict object/type/duplicate/unknown-field assertions.

Evidence: 1-18 MAX_STRICT_BYTES/DecodeError; 22-96 exact base64; 97-352 duplicate traversal, Binder/Soft, parse_strict/parse_lenient/dedupe_last_wins; 353-558 Emitter/Emit implementations; 559-573 numeric ranges/marshal; 574-623 tests.

Open detail: Preserve crate::jsonx public names and field ordering/newline/base64 bytes. decode reads base64_decode from its parent; emit reads base64_encode/DecodeError only as already needed. No new package or generic serializer API. cfg(test) descendant tests retain private Binder/Emitter access where needed.

### rust/soda-release-deliver/src/model.rs

Observed size: 1312 lines, including tests where embedded. The file already names distinct release-authority documents. Keep validation, decoding and emission with each actual type rather than create separate model/validator/codec layers. Channel highwater and its admission remain together; release serial admission stays with Release. Permit remains meaningful code in the public root.

- `lib/soda-release-deliver/src/model/mod.rs` — Public model exports and the existing protected Permit type, validate/decode/Emit implementation.
- `lib/soda-release-deliver/src/model/trust.rs` — Trust envelope/timing, PEM/DER P-256 shape, signer-role separation and fingerprinting, role/reference lookup, typed decode/emission.
- `lib/soda-release-deliver/src/model/candidate.rs` — Candidate host/source/toolchain/content binding, asset/path rules, decode/emission and existing path_clean.
- `lib/soda-release-deliver/src/model/release.rs` — MediaFile/MediaBinding and media admission, Release validation/provenance/evidence/reference/decode/emission, release serial admission and its exact helpers.
- `lib/soda-release-deliver/src/model/channel.rs` — Channel/Seen/Highwater shapes and decode/emission/state validation; channel reference/identity/timing/progression helpers and admit_channel.
- `lib/soda-release-deliver/src/model/tests.rs` — Current content/path and trust-reference unit tests plus their full_content fixture.

Evidence: 24-299 Trust and all PEM/DER/key-role helpers; 300-535 Candidate, exact content set, path_clean and code/asset binding; 536-615 media; 616-798 Release; 799-1021 Channel/Seen/Highwater; 1022-1125 channel admission; 1126-1179 release admission; 1180-1231 Permit; 1232-1312 tests.

Open detail: Retain crate::model paths through explicit reexports, including pub(crate) path_clean and valid_media_* used by OCI/admission. Channel admission owns validate_release_reference (used by validate_channel_releases); release admission owns admit_channel_ref/admit_release_digest. Cross-type calls use the same Trust/Payload/Candidate/Highwater, never new adapter types.

### rust/soda-release-deliver/src/native.rs

Observed size: 614 lines, including tests where embedded. Separate trust-policy construction and signer custody from the existing native runner/copy operation. Keep signing admission before effects and the final fresh verification in its current order. The 49-line native test module can stay inline in mod.rs rather than gain another tiny leaf.

- `lib/soda-release-deliver/src/native/mod.rs` — Runner/Native command execution, locked version check, private-file/write-json custody, verified copy and manifest check, existing focused native tests.
- `lib/soda-release-deliver/src/native/policy.rs` — Requirement shape/emission, exact role policy construction/merge/conflict refusal, local policy and registry Sigstore configuration.
- `lib/soda-release-deliver/src/native/sign.rs` — SecretFiles decode, protected permit/key admission, snapshot source, signed document admission, signature emission and round-trip verification.

Evidence: 18-69 Runner/Native/check_native; 71-106 private_file/write_json; 107-370 Requirement/policy/merge/registries; 371-434 verify source/copy/digest; 435-565 SecretFiles/sign closure; 566-614 version/private-file tests.

Open detail: Preserve crate::native public and crate-visible paths used by fetch/publish. Keep policy_for/local_policy/registry_config visible only to their native parent/siblings as needed. Moving TOOL_LOCK into src/native/mod.rs changes include_str! from ../tools.json to ../../tools.json; preserve the same tracked lock bytes, skopeo env clearing, command timeout and output cap.

### rust/soda-release-deliver/src/oci.rs

Observed size: 1109 lines, including tests where embedded. Split wire identity, outer archive custody and rootfs layer resolution. Keep whiteouts, ancestor replacement, unsupported zstd blocking and topmost-member resolution in one layer owner; these are one algorithm. The layout loader and public orchestration stay together and consume the same schema/archive/layer internals.

- `lib/soda-release-deliver/src/oci/mod.rs` — OCI constants, OciLayout/shared Blob, three existing public inspect entrypoints, content identity orchestration, confined LayoutLoader/image-set closure, existing focused tests.
- `lib/soda-release-deliver/src/oci/schema.rs` — Descriptor/OciManifest/OciConfig and exact index/manifest/config decoding; bounded blob hashing, layer/rootfs/attribution checks and inspected image identity.
- `lib/soda-release-deliver/src/oci/archive.rs` — Regular archive admission, safe unique bounded archive-entry collection, archive index selection and outer layer-archive traversal.
- `lib/soda-release-deliver/src/oci/layers.rs` — LayerMember, exact requested paths, clean layer names, whiteout/opaque/ancestor replacement rules, member hashes, descriptor index, TeeHasher/drain/layer scan and reverse member resolution.

Evidence: 29-67 OciLayout/Blob/Descriptor/OciManifest/LayerMember; 68-421 exact index/manifest/config/blob/image inspection; 422-518 outer archive admission/read/index; 519-528 inspect_oci; 529-855 member path/whiteout/hash/layer resolution; 856-928 content orchestration/outer scan; 929-1080 layout input/Root/loader/image set; 1081-1109 current tests.

Open detail: Keep crate::oci::{inspect_oci,inspect_oci_content,inspect_oci_layout,OciLayout}. Blob stays common only within this module; Descriptor/OciManifest/OciConfig stay schema-owned, LayerMember layer-owned, and sibling access is bounded to OCI (pub(super) where required). scan_oci_archive_layers delegates existing scan_archive_layer; do not create a public archive abstraction or extract helpers with no caller.

### rust/soda-release-deliver/src/payload.rs

Observed size: 406 lines, including tests where embedded. The production owner is 366 lines and is one delivered payload contract. Extract the existing 40-line tests only; retain shared optional decode helpers and load with their current owner to avoid a new generic codec layer.

- `lib/soda-release-deliver/src/payload.rs` — Current Image/Payload wire types, validity checks/decode/emission, typed optional field adapters, repository prefix validation and confined load operation.
- `lib/soda-release-deliver/src/payload/tests.rs` — Current repository-prefix and CoreOS-version tests.

Evidence: 24-213 Image/Payload and validation/decode; 214-262 decode_opt_* / decode_string_map (used by model/buildx/native/publish); 263-346 Emit/repository prefix; 347-366 load via Root/read_json_at; 367-406 tests.

Open detail: Preserve crate::payload public names, build decoding last-wins behavior versus strict document callers, and exact load errors/digest checks. Use #[cfg(test)] mod tests; src/payload.rs + src/payload/tests.rs is an ordinary Rust module/descendant pair.

### rust/soda-release-deliver/src/publish.rs

Observed size: 533 lines, including tests where embedded. Ledger state and channel promotion are distinct duties within one publication operation. Keep the ordered publication state machine in mod.rs: lock, hold pending uncertainty, admit/signature-check, record pending, immutable upload, optional channel promotion, observe and record complete. Existing 47-line tests remain inline.

- `lib/soda-release-deliver/src/publish/mod.rs` — Upload/observed publication, ledger admission, observe-only completion, signed snapshot admission, immutable commit, final receipt and publish sequencing; current focused tests.
- `lib/soda-release-deliver/src/publish/ledger.rs` — Ledger type/decode/validation/emission, phase rule and explicit private init_ledger.
- `lib/soda-release-deliver/src/publish/channel.rs` — Protected channel history/offer admission, anonymous tag observation, prior-channel verification and tag promotion.

Evidence: 19-102 Ledger/init_ledger/phase; 103-219 upload/observe/admit/observe-only; 220-257 channel history/offer; 258-325 admit_signed/commit_immutable; 326-428 tag/prior-channel/promotion; 429-485 finalization/publish; 486-533 tests.

Open detail: Preserve crate::publish::{Ledger,init_ledger,publish} and private helper order. Channel functions use the actual fetch::verify_releases and native::verify_copy, ledger uses the same fetch::lock_state/save_state, and no blind publication retry or new recovery machinery appears.

### rust/soda-release-deliver/tests/oracle.rs

Observed size: 680 lines, including tests where embedded. Group oracle assertions by pure authority documents, delivered artifact bytes, and durable fetch state. Retain the single existing golden JSON fixture without mechanical splitting; no live Go subprocess is needed to consume retained byte/outcome evidence.

- `lib/soda-release-deliver/tests/oracle/main.rs` — Cargo-discovered oracle test root: golden decode/result/time helpers, model byte-roundtrip/validation/channel/release/policy/strict-decode cases, shared temp/private-dir and current small formatting/hash helpers.
- `lib/soda-release-deliver/tests/oracle/artifacts.rs` — Document layout round-trip/copy fixture, plain/gzip OCI/content tests, payload load errors, candidate binding-before-archives and qualification/fixture-refusal cases.
- `lib/soda-release-deliver/tests/oracle/fetch_state.rs` — Real private state/ledger init observations and the existing scripted Runner/fetch wiring refusal case.

Evidence: 24-83 shared retained golden readers; 84-313 model/channel/release/policy parity; 314-431 document/OCI fixture; 432-463 strict/load; 464-486 state/ledger; 487-588 candidate/qualification; 589-628 helpers; 629-680 ScriptRunner/fetch refusal.

Open detail: Use Cargo's tests/oracle/main.rs integration-test discovery, with mod artifacts; mod fetch_state; as natural child modules and private helpers accessible through super. Keep tests/goldens/deliver.json; update include_str! to ../goldens/deliver.json. Preserve all assertions and observed fixture bytes; scripted fetch proves wiring/refusal, qualification JSON tests admission, neither establishes installed or live publication proof.

### rust/soda-release-image/src/build.rs

Observed size: 1446 lines, including tests where embedded. Separate source admission/snapshot, command/recall/log execution, host-context preparation, shipping compilation inventory and candidate sealing from the single ordered build orchestration. The two host-candidate bodies duplicate the same observation-build/package-hash/seal/final-build/readback sequence; use the existing callback-based implementation once while retaining phase labels. Keep the narrow RunnerProduction bootstrap with source extraction: its intentionally refusing foreign methods are not a completed build/deliver adapter.

- `lib/soda-release-image/src/build.rs` — Existing ProductionInputs/build/run_build/run_build_inner, ordered phase orchestration, production factory, prepare/execute/finalize joins and record-result boundary.
- `lib/soda-release-image/src/build_runner.rs` — Cancel/SharedFile/Runner/LogCloser, recall/media log attachment, child environment/tool resolution and run_build_command.
- `lib/soda-release-image/src/build_source.rs` — Canonical clean checkout/revision/compiler/output admission, directories/source archive/workspace/log lifecycle and current Forgejo extraction bootstrap.
- `lib/soda-release-image/src/build_context.rs` — Freeze selected base-image config, prepare host context and existing link_prepared_assets.
- `lib/soda-release-image/src/build_compile.rs` — Go command discovery, Rust installed-command/tool tables, shipping tool compilation/copy/hardlink and exact tools.json record.
- `lib/soda-release-image/src/build_candidate.rs` — One current observation-build/package-hash/payload-seal/inventory/final-build/readback implementation, shared by callback/progress callers.
- `lib/soda-release-image/src/build_runner/tests.rs` — Existing command environment/capture/failure case against the real execution function.
- `lib/soda-release-image/src/build_context/tests.rs` — Existing prepared-assets/no-command assertion with its current panic-on-command fixture, preserving that specific boundary.

Evidence: 36-170: Cancel/SharedFile/Runner/LogCloser;171-204: ProductionInputs/build;205-378: source/input/output/snapshot/workspace admission; 379-490: freeze_base_image_config/prepare_build_host_context;491-521: prepare_build_production; 522-710: compile_soda_commands/record_tool_files/RUST_TOOLS/compile_rust_tools/compile_shipping_tools; 711-771 and1059-1121: same host-candidate sealing/build sequence with next callback versus Progress;772-864,976-1058: orchestration/finalization; 865-975: narrow RunnerProduction bootstrap for forgejo::extract_forgejo_snapshot;1122-1244: environment/tool resolution/run_build_command;1246-1264: link_prepared_assets; 1265-1326: real child command test;1327-1446: no-command prepared-assets test and specific stub

Open detail: Actual pipeline wiring is pending: image depends on neither release-build nor release-deliver; tools/src/build_cli.rs:305-318 and349-353 stop with explicit not-yet-implemented errors. compile_shipping_tools:687-691 still compiles deleted-at-cutover Go soda-artifacts and694-699 copies only acceptance driver. Target wiring must choose the Rust tools owner and record acceptance remote beside the driver. Do not infer installed proof from crate presence.

### rust/soda-release-image/src/media.rs

Observed size: 1204 lines, including tests where embedded. Separate upstream assembler preparation, authenticated packaging inventory, packaging-container/meta verification and installer customization/readback. Keep Media/MediaLock/MediaAuthority/MediaInputs, admitted-input sequencing, final media binding and assemble_media orchestration together. Preserve exact authority/digest/inventory admission, native container cleanup and ISO/rootfs readback before sealing; no new media service or alternate signing path is implied.

- `lib/soda-release-image/src/media.rs` — Current media/lock/authority records and public-base URL gate, authority/candidate/compression admission, final seal_media binding and one assemble_media sequence.
- `lib/soda-release-image/src/media_assembler.rs` — Upstream config revision fetch, frozen buildroot arg, assembler digest/wrapper/layer verification, prepare_assembler and builder_id.
- `lib/soda-release-image/src/media_authentication.rs` — Existing sign/verify delegation and auxiliary input inventory collection/binding/recheck before native packaging.
- `lib/soda-release-image/src/media_container.rs` — Current run-owned packaging-container stop/build, native assembly and MediaMeta parse/identity verification.
- `lib/soda-release-image/src/media_installer.rs` — Rootfs placement, coreos-installer ISO customization/verification, initrd readback/chunk verification and prepare_and_verify_media.
- `lib/soda-release-image/src/media/tests.rs` — Existing public URL and buildroot selection tests against the actual owning functions.

Evidence: 20-174: callback aliases, MediaLock/MediaAuthority/Media and media_base_url;175-398: assembler preparation/identity; 399-439,533-639: sign_media_input/inventory/authenticate_packaging_inputs;440-532: MediaInputs and authority/candidate/compression admission; 640-740: owned-container cleanup/build/native assembly;741-819: MediaMeta/image/meta validation; 820-1035: setup_media_rootfs/verify_customized_iso/customize_installer_iso/verify_media_readback/prepare_and_verify_media; 1036-1153: seal_media/assemble_media;1154-1204: URL and assembler buildroot tests; build_media.rs consumes prepare_assembler/assemble_media

Open detail: The existing Production trait delegates signing/verification to a caller that still needs release-deliver wiring. The physical decomposition preserves this seam; it is not evidence of native packaging/signing success.

### rust/soda-release-image/src/model.rs

Observed size: 1475 lines, including tests where embedded. Separate current artifact primitives/records, URL+IP admission, payload, candidate+Forgejo provenance, trust+permit+secret-file binding and admitted live CoreOS/Tailnet records. This documents the current mirrored concerns, not permission to preserve duplicate owners permanently. At the actual tools/image/build/deliver cutover, replace matching mirrored DTO/validator closures with their existing Rust owner after reconciling exact decoder, null/empty, map-order and error guarantees. Keep private trust DER/SPKI helpers with trust rather than a generic certificate package.

- `lib/soda-release-image/src/model.rs` — Current format/source/schema and image-role constants plus shared digest/revision/architecture/repository/CoreOS identity primitives; schema remains derived from the real store owner.
- `lib/soda-release-image/src/model/url.rs` — Current UrlParts/parser, HTTPS shape, loopback and IPv4/IPv6 helpers used by media URL admission and live-input validation.
- `lib/soda-release-image/src/model/images.rs` — Current Image/ProducedImage records and their JSON interface for foreign OCI/production operations.
- `lib/soda-release-image/src/model/payload.rs` — Current PayloadImage/Payload parse/emit/load and exact identity/base/independent-image/no-upgrade admission closure.
- `lib/soda-release-image/src/model/candidate.rs` — Current ForgejoToolchain/package provenance and Candidate parsing/emission/payload/source/host/content binding.
- `lib/soda-release-image/src/model/trust.rs` — Current Trust role keys/timing/minimum-sequence admission, P-256 PEM/DER/SPKI checks, Permit and SecretFiles models.
- `lib/soda-release-image/src/model/live_inputs.rs` — Current CoreOSImage/ResolvedCoreOS/TailnetInputs/LiveInputs decoding, container selection and validators.
- `lib/soda-release-image/src/model/tests.rs` — Existing primitive/URL/payload cases remain unit scoped; attach each case to its actual concerned module without exporting parser helpers for tests.

Evidence: 12-104: current constants, identity/digest/architecture/repository gates;105-290: URL/IPv4/IPv6/loopback implementation; 291-365: Image/ProducedImage;366-608: PayloadImage/Payload JSON/load/identity/base/image/upgrade validation; 609-695: ForgejoToolchain/APK provenance;696-939: Candidate and exact payload/source/host/content binding; 940-1176: Trust and private PEM/DER/SPKI/role-key admission;1177-1244: Permit/SecretFiles; 1245-1427: admitted CoreOS/Tailnet inputs;1428-1475: three unit groups; actual consumers are build,complete,host,inspect,layout,payload_stage,prepare,record,media and foreign trait signatures

Open detail: Reuse candidates are build::{oci::Image,production::ProducedImage,forgejo::ForgejoToolchain,coreos::CoreOSImage,coreos_stream::*} and deliver::{payload::Payload,model::{Candidate,Trust,Permit},native::SecretFiles}. Equivalence is not proven: image uses Vec upgrade_from and emits [] while deliver/payload.rs:81-85 preserves Option for nil→null; image ProducedImage contains manifest/config while build ProducedImage wraps full Image. Do not replace these via alias/FFI/RPC or create a new schema owner.

### rust/soda-release-image/src/prepare.rs

Observed size: 501 lines, including tests where embedded. Keep the cohesive public-only host context staging operation and explicit source-to-installed file map together. The production body is 376 lines; the excess comes from a 125-line unit case largely implementing the existing foreign trait. Extract that test as a descendant instead of manufacturing separate writer/map/base orchestration packages. Preserve current live-versus-admitted base paths, public-file modes/vendor normalization, exact revision and inventory semantics.

- `lib/soda-release-image/src/prepare.rs` — Base/live/file resolution, PreparedWriter/public copies and rootfs_file_map, one prepare/prepare_resolved/finish_prepare sequence and exact context inventory.
- `lib/soda-release-image/src/prepare/tests.rs` — Existing base-input architecture/revision gate case and its original foreign trait stub; keep private finish_base_inputs access.

Evidence: 20-68: Base and live/admitted base loaders;69-184: PreparedWriter/public-file normalization/links/build record; 185-254: rootfs_file_map;255-340: base input/revision/provisioning admission and prepare/prepare_resolved/finish_prepare; 341-376: inventory modes/hash/link record;377-501: oracle_base_inputs_require_exact_revision and current stub; build.rs:451-490 consumes prepare/prepare_resolved;build.rs:748,1095 consumes inventory

Open detail: All current appliance source paths and installed libexec/service destinations must follow the chosen system tree. These are production inventories, not obsolete Go logic; retain their staged bytes and update path-reading test fixtures directly.

### rust/soda-release-image/tests/oracle.rs

Observed size: 684 lines, including tests where embedded. Retain the frozen Go outputs as Rust test data and split the heterogeneous battery into three existing responsibility groups under one integration entrypoint. Shared byte/error helpers remain in the entrypoint. Keep the refusal-only Production stub local to staging; do not turn its deliberately unreachable operations into another general production adapter. Large base64 literals stay beside the cases they qualify.

- `lib/soda-release-image/tests/oracle.rs` — Existing single integration-test entrypoint, frozen-output note and b64/check_ok/check_err helpers.
- `lib/soda-release-image/tests/oracle/media.rs` — Existing public media URL, compression, native media log-event and rootfs-chunk assertions.
- `lib/soda-release-image/tests/oracle/host.rs` — Existing local quadlet, live Ignition, package/RPM input and candidate-live-config byte/refusal assertions.
- `lib/soda-release-image/tests/oracle/staging.rs` — Existing asset-name/failure-reason/stage-layout cases and the stage-only refusing Production stub.

Evidence: 1-23: frozen-output helpers;24-115: media public-base URL cases;116-218: quadlet/live Ignition byte cases; 219-236 and292-324: package/RPM admission;237-291: compression;325-392: media event bytes; 393-442: extension asset/failure reason;443-579: stage-only Production stub;580-612: stage layout refusals; 613-648: rootfs chunk verification;649-684: candidate-live-config frozen byte/refusal cases

Open detail: The source contains frozen expected byte strings, not a runtime dependency on an old Go oracle. Preserve the exact proven cases while adjusting owner imports only as part of the real caller cutover.

### rust/soda-release-tools/src/artifacts.rs

Observed size: 506 lines, including tests where embedded. Production is 330 lines and cohesive. Extract its 176-line tests, retaining the exact command/validation order and current explicit unimplemented pipeline boundaries. No separate filesystem library or new subprocess wrapper is justified by this size.

- `lib/soda-release-tools/src/artifacts.rs` — One artifact command owner: flag admission, exact native tools/output/Butane execution and OCI/CoreOS boundary dispatch.
- `lib/soda-release-tools/src/artifacts/tests.rs` — Existing unit tests with private destination/Butane/CLI/OCI/CoreOS inputs and refusal observations.

Evidence: 55-89 flags; 90-161 executable lookup/private output admission; 162-235 Butane admission/execution/new-file completion; 236-302 CoreOS/OCI admission and current boundary errors; 303-330 action dispatch/main; 331-506 tests.

Open detail: Keep crate::artifacts public paths and its current String errors. cfg(test) descendants use current private helpers, without widening production APIs. Current CoreOS fetch/OCI inspection boundary stubs remain evidence of pending wiring, not a completed pipeline claim.

### rust/soda-release-tools/src/build_cli.rs

Observed size: 514 lines, including tests where embedded. The 377-line production file is one controller entry workflow; its flags, source binding and branch/exit sequence should remain readable together. Extract tests instead of distributing these steps into independent tiny files.

- `lib/soda-release-tools/src/build_cli.rs` — Existing soda-build flags/admission, source/VCS binding, environment/signals/progress, parent/worker branch and exit handling.
- `lib/soda-release-tools/src/build_cli/tests.rs` — Current exact flags/worker dispatch, progress/source/environment boundary tests.

Evidence: 1-173 FLAG_SPECS/BuildFlags/parse and dispatch admission; 174-202 env/signals/progress; 203-292 Soda/Fountain revisions and canonical checkout binding; 293-320 artifact printing/run_parent_build; 321-377 run/main; 378-514 tests.

Open detail: Preserve crate::build_cli paths and the existing build.rs VCS stamp. Cargo manifest depth is still two directories below repository root after rust/soda-release-tools -> lib/soda-release-tools, so build.rs ../.. root lookup remains valid. Current parent/worker execution explicit boundary errors are pending integration; file moves do not complete them.

### rust/soda-release-tools/src/candidate.rs

Observed size: 522 lines, including tests where embedded. Separate the actual answer/flag contract from executing an admitted candidate run. Keep Options plus all current validation together; keep fixture startup, controller wait, stop and filing order in the run owner.

- `lib/soda-release-tools/src/candidate/mod.rs` — Checkout cleanliness/fresh-output preflight, monotonic origin, summary, terminal option resolution, controller/fixture lifecycle, failure/exit handling.
- `lib/soda-release-tools/src/candidate/options.rs` — Options, flag specs/default/usage/native architecture, mode/arch/parse and answered path/media validation.
- `lib/soda-release-tools/src/candidate/tests.rs` — Current base_options fixture and option/answer/preflight unit tests.

Evidence: 14-203 Options/flag_specs/ARCH_DEFAULT/usage/native_arch/parse_options/valid_out_leaf/validate_*; 204-279 checkout/Git/fresh-out preflight; 280-327 monotonic/describe/resolve_options/ready_run; 328-405 ExitError/run_candidate/CandidateError/main; 406-522 tests.

Open detail: Reexport current crate::candidate Options and option functions so candidate_prompts/fixture/controller callers stay stable. Preserve #[cfg(test)] pub mod tests and base_options, which candidate_prompts' existing test module uses; do not silently sever that test-only caller. Keep existing callback to prompter_overview, not a new run interface.

### rust/soda-release-tools/src/candidate_display.rs

Observed size: 686 lines, including tests where embedded. Event parsing/path translation is an existing stateless seam. Keep the entire mutable renderer lifecycle, both impl blocks, ticker ownership and its private state in one file; do not split drawing from lifecycle into stand-in UI APIs.

- `lib/soda-release-tools/src/candidate_display/mod.rs` — Phase/RendererInner/Renderer/TickerHandle, both Renderer impls, every locked feed/note/draw/finish method, phase/display/time/terminal helpers and constants.
- `lib/soda-release-tools/src/candidate_display/events.rs` — Controller Event and exact START/DONE/FAILED/CANCELLED/artifact parsing, duration fields and current sandbox-to-host artifact path translation.
- `lib/soda-release-tools/src/candidate_display/tests.rs` — Current captured-writer event/parser/render/failure/panel/path/viewport cases, including private renderer state checks.

Evidence: 11-103 Event/parse_event and all parser helpers/host_artifact_path; 104-271 renderer fields and first impl; 272-352 phase mutation/line rendering; 353-435 ticker/finish/why panel impl; 436-477 exit/time/truncate/terminal helpers; 478-686 tests.

Open detail: Keep crate::candidate_display exports consumed by candidate_controller and candidate. tests stays a cfg(test) descendant of the root and retains direct private RendererInner/Phase access; Renderer is not made public beyond its current API. Event wire padding quirks, lock order, first failed cause and existing ticker stop/join behavior stay unchanged.

### rust/soda-release-tools/src/candidate_fixture.rs

Observed size: 403 lines, including tests where embedded. Production is 281 lines covering one existing local media pickup workflow. Extract the 122-line tests; keep the selected loopback-only fixture and its stop handle/file custody together. This is a development fixture, not a newly designed product server.

- `lib/soda-release-tools/src/candidate_fixture.rs` — Existing development rootfs pickup defaults/URL loopback detection/listen/HTTP refusal/copy ownership and fresh destination rules.
- `lib/soda-release-tools/src/candidate_fixture/tests.rs` — Current fixture URL/address matrices, HTTP/rootfs filing, busy-port behavior and occupied/missing output refusal tests.

Evidence: 11-97 rootfs defaults/URL host/port/fixture predicates; 98-215 FixtureServer/serve_fixture/request/percent decode/response; 216-281 copy_built_rootfs/chown_name/copy_file; 282-403 tests.

Open detail: Preserve current fixture_wanted and busy-port no-op semantics and candidate_controller caller paths. HTTP tests exercise the real existing serve_fixture/copy_file operations. No new listener, daemon, service or fallback is proposed.

### rust/soda-release-tools/src/candidate_prompts.rs

Observed size: 656 lines, including tests where embedded. Separate the actual default-answer/controller-argv calculations from the interactive Prompter owner. Keep its full private implementation, field order, renderer and edit methods together; overview_rows continues to call Prompter::show_fast_compress within that owner.

- `lib/soda-release-tools/src/candidate_prompts/mod.rs` — Prompter fields and entire impl, private editing/line/choice/render/overview state, overview_rows/output status/path observations, real-terminal entry.
- `lib/soda-release-tools/src/candidate_prompts/defaults.rs` — Existing constants/mode labels/default path/timestamp suggestion/controller arguments and default_overview answer initialization.
- `lib/soda-release-tools/src/candidate_prompts/tests.rs` — Existing Shared capture/scripted prompt fixtures, controller args/defaults/answer edits/refusal/output suggestion assertions.

Evidence: 7-115 constants/mode/default_path/file_exists/suggest_out/utc_stamp/controller_args; 116-377 full Prompter impl and existing injected exists callback; 378-423 overview_rows/out_status/parent_dir_exists/path_absent; 424-442 default_overview; 443-451 terminal entry; 452-656 tests.

Open detail: Preserve crate::candidate_prompts public exports, constructor existing file_exists callback and cfg(test) fixture dependency crate::candidate::tests::base_options. Only current private default_overview/file_exists access crosses to defaults (pub(super)); no detached Prompter method or new state/callback abstraction is introduced.

### rust/soda-release-tools/src/progress.rs

Observed size: 407 lines, including tests where embedded. Production is 318 lines and one ordered progress lifecycle. Extract the existing 89-line tests; keep event emission, inherited timing origin/log admission, phase/section completion and summary/exit in the same owner.

- `lib/soda-release-tools/src/progress.rs` — One BuildProgress timing/event/log owner with current private clock/phase/reason/finish state and all methods.
- `lib/soda-release-tools/src/progress/tests.rs` — Current environment restoration/locking, fixed clock/captured output and phase/failure/origin assertions.

Evidence: 12-47 monotonic/duration/failure/finish text; 48-318 BuildProgress with full impl, log/events/phase/next/end/finish; 319-407 existing environment/clock fixtures and tests.

Open detail: Keep crate::progress names and the START/DONE/FAILED/CANCELLED wire consumed by candidate_display. Retain the real injected clock/capture API and test environment locking/restoration; no new progress sink/service or event protocol is introduced.

### rust/soda-release-tools/src/worker.rs

Observed size: 978 lines, including tests where embedded. Group restricted configuration custody and per-attempt runtime ownership separately from the pure worker description/result boundary. Keep all bind/environment/argv choices and receipt/host-path checks visible in the original worker owner, not spread through a generic sandbox API.

- `lib/soda-release-tools/src/worker/mod.rs` — Worker description/constants, current uid/user lookup/identity, sandbox bind/environment/argument construction and exact result decode/validation/host-path rebinding.
- `lib/soda-release-tools/src/worker/config.rs` — WorkerConfig, path/trusted-executable/private-file custody, task/config/storage admission, bounded decode/read/load.
- `lib/soda-release-tools/src/worker/runtime.rs` — Fresh per-attempt runtime claim/nonce/private mode/chown and constrained direct-child release.
- `lib/soda-release-tools/src/worker/tests.rs` — All current request/config/runtime/name/result unit scenarios and existing temp-dir/config fixtures.

Evidence: 19-31 WorkerConfig; 32-81 Worker/identity helpers; 82-324 config/path/trust/private input admission and load; 325-404 runtime claim/release; 405-515 join/rel/name/build_worker; 516-594 result binding/decode/read; 595-978 twelve existing tests.

Open detail: Preserve crate::worker public paths and pub(crate) go_clean, used by build_cli::validate_forgejo_checkout_root. config keeps all its filesystem helper closure; runtime reuses existing uid/gid inputs without new authority. Test descendants can reach private helpers through appropriate parent-only imports, not widened production APIs. Current build_worker constructs a description; this is not evidence that the isolated execution port is wired.

### rust/soda-release-tools/tests/cli.rs

Observed size: 472 lines, including tests where embedded. Organize tests by the three actual existing command identities, sharing only the current temporary directory and command-result fixtures. Keep each exact usage golden and every code/stdout/stderr assertion beside the command it verifies.

- `lib/soda-release-tools/tests/cli/main.rs` — Cargo-discovered CLI suite root with existing TempDir/drop/counter and binary command result capture.
- `lib/soda-release-tools/tests/cli/soda_build.rs` — Current build binary lookup/usage golden, help/flag/bool/admission matrix and deep refusal.
- `lib/soda-release-tools/tests/cli/soda_candidate.rs` — Current candidate binary lookup/usage golden/help/flags/answer/preflight refusals.
- `lib/soda-release-tools/tests/cli/soda_artifacts.rs` — Current artifacts binary lookup/action/native/Butane/archive-shape refusals.

Evidence: 1-60 TempDir/bin/run fixtures; 61-176 build CLI cases; 177-327 candidate cases; 328-434 artifact cases; 435-472 build deep refusal.

Open detail: Use Cargo's tests/cli/main.rs automatic integration-test discovery with mod soda_build; mod soda_candidate; mod soda_artifacts; nested files call existing private root run/TempDir through super. Keep CARGO_BIN_EXE_soda-build/candidate/artifacts identities. Existing explicit pipeline-boundary refusal tests remain pending behavior evidence, not completed success-path proof.

### tests/build/project_factory_roles_test.go

Observed size: 414 lines, including tests where embedded. Retire the Go wrapper when its actual assertions and fixtures exercise the Rust helper successor. This file is 414 lines chiefly because its 230-line rolesDriver embeds Python, SourceFileLoader and mocks; moving it to smaller Go files would preserve Python execution and violate the decided cutover. Keep one native concern suite until real Rust fixture code warrants another measured split.

- `cmd/soda-project-terminal/src/factory_roles/tests.rs` — Native successor owns all 12 established role/account/custody/phase/start/stop/hold/process/log scenarios; reuse the already proposed factory_roles test owner rather than add duplicate test suites.

Evidence: 15-244 rolesDriver imports the real project-os/rootfs/usr/libexec/soda/project-factory-roles, patches account/process/filesystem effects and implements named cases; 246-293 runRolesCase/Python3/JSON observations/MRO/base64 adapters; 295-351 ensure/approve/reject-before-effects/record assertions; 352-414 launcher refusal/start/interruption/stop/hold/proc-group/zombie/bounded-output assertions.

Open detail: Reuse the pending Rust private owner cmd/soda-project-terminal/src/factory_roles/{mod,layout,accounts,inputs,records,execution,tests}.rs and compiled src/bin/project-factory-roles.rs. Preserve all existing scenario observations: locked nologin accounts/no extra groups/idempotence; immutable mode-0644 snapshot/bundle verification/repeat and invalid-input no effects; waiting/missing stability and launcher failure; supervisor running/interruption; stop retirement/identity bar; hold revision/quiescence; pgrp versus session/uid/gid/zombie checks; bounded log bytes/exit3/truncation. Remove rolesDriver/SourceFileLoader/Python3/exception-MRO dependency, with native refusal/effect assertions. Compiled entrypoint protocol and installed staging/hash qualification remain separate evidence. PR42 native helper port is pending: exact test fixture execution/injection seams must follow the real Rust implementation, with no placeholder dispatcher, new public test API or Python subprocess retained. Existing /proc fixture executable names such as python3 are inert test bytes, not permission to retain Python execution. Retire the Go source only after the real native owner preserves the assertions; no port completion is claimed.

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
| `docs/development/ideal-filetree-plan.md` | 5020 at the reconciled commit | `docs/development/ideal-filetree-plan.md` (living plan, length changes during maintenance) |
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

This document is the maintained project artifact from this pass; the inventory
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
work, and recheck source drift before each later update of this file.

The initial review covered 1,586 paths at its recorded baseline. At the current
reconciled commit, all 1,705 tracked paths have destinations or dispositions:
1,488 map to retained/moved concern owners (including partially retained input
or wire files), and 217 are retirement dispositions. The original 158 oversized
reviewed sources still match their stored content hashes; the 32 new entries
complete coverage of all 190 current oversized code files.

The primary tree contains 2,248 unique leaves and 27 Cargo package manifests,
plus the root workspace manifest. Those counts include explicit pending native
entrypoints/adapters and concern splits, not measured final implementation
sizes. Shared destination leaves intentionally consolidate providers, assets,
HTTP/test support and moved native assertion coverage. No file/directory or
Rust file/module-root filename conflicts remain. The tree omits obsolete Go
acceptance/process/release/native-host implementations and Python sources;
pending conversions inside retained files are listed with their real callers.

## Keeping the plan current after every merge

The person or agent handling a merge owns the post-merge update of this file as
part of that work. Update the plan after every merge into the maintained branch,
before considering the merge task complete. Pulling merged changes into this
checkout also requires reconciliation if this file still records an older
baseline. A planning update never starts the deferred refactor.

1. Read the current HEAD, working-tree changes and this plan's last reconciled
   source. Compare the accumulated changes since that source, not only the last
   merge commit. Preserve unrelated work; if the baseline is unavailable,
   inventory the current source before claiming reconciliation.
2. Refresh the tracked file inventory and size flags. Inspect changed source
   plus affected callers, tests, manifests and build/install payloads. Reuse the
   existing concern reviews for unchanged code; a full architectural audit or
   application build is not required for routine plan upkeep. Add a real concern
   review for every newly oversized code file before claiming zero unreviewed
   files. Keep pending-branch observations separate from merged-source coverage.
3. Update the complete desired tree, package ownership, port recommendations,
   affected concern notes, source references and coverage counts together.
   Account for additions, deletions, renamed files, changed large-file seams
   and any actual new process boundary. Do not infer sidecars from package splits.
4. Remove proposals already implemented or superseded and mark unsettled choices
   honestly. Retiring a port removes its conditional split targets; retaining it
   requires closing its documented import, state and delivery seams. Keep one
   current plan rather than an append-only log of merges. Omit dead or decided
   predecessor implementations from the primary target even before physical
   deletion; record the retirement and its real cutover dependencies separately.
   Check live data, mixed client/wire files, command callers and embedded programs
   before deleting a whole language/package subtree from the plan.
5. Record the actual reconciled source commit, maintenance date and review scope.
   Preserve the initial full-review baseline; a delta review does not claim a
   new full architectural review or fresh runtime verification. Check source
   coverage, destinations, merge targets, links and filename/module conflicts.
   For a merge with no structural impact, still advance that baseline/date and
   state that the reviewed delta required no proposal changes.

Use the resulting commit recorded after a merge, squash or rebase, rather than
the pre-merge branch tip. If upkeep is incomplete, keep the previous reconciled
baseline and state which source changes still need review; never label the plan
current solely because a pull succeeded. A follow-up documentation change can
record the code merge it reconciles without creating an endless self-update
cycle for its own bookkeeping-only commit.

The maintained artifact is this Markdown file. Ignored scratch inventories and
renderers are optional analysis aids, not prerequisites for future maintainers.
Do not blindly regenerate from old split reports after source changes: re-read
the affected definitions and preserve later owner decisions in this document.

## Executing when time is available

Keep implementation deferred until the owner selects work that fits available
time. Before that work starts, reconcile this plan against current source and
resolve the affected ownership/language choices. Select a small, coherent
boundary including its real callers, manifests, tests and installation wiring;
do not begin the entire repository migration merely because the plan exists.
Replace proposed seams with implemented paths as work lands, remove obsolete
alternatives, and record build, source-check and native verification separately.
Completed choices belong in their owning guides; this file remains the current
plan for outstanding restructuring while that work remains deferred.
