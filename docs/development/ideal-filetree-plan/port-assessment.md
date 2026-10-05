# Port assessment and cutovers

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
| Release pipeline + tools (new) | Four Rust crates landed (#30/#32/#33/#34) and cut over: tools → image → build/deliver is wired, candidate-check and payload data preserved, Go removed | Retain the four crates under `lib/` and split their actual concerns in the [release decomposition reviews](decomposition/release-production.md). |

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

**rust/soda-host/src/pfactory.rs** (6081 lines at 0d8d3b8e).

- `lib/host/src/factory/mod.rs` — 1–39,1457–1556,1773–1821: shared preamble/imports, FactoryTerminal/FactoryBroker, Secret/RunLock/Factory, open_factory/harness_pin; one state/lock owner.
- `lib/host/src/factory/run.rs` — 40–201,365–588: factory phase/run/path validators, FactoryRun and FactoryLaunch.
- `lib/host/src/factory/deadline.rs` — 202–364: RFC3339Nano parse and current native deadline math.
- `lib/host/src/factory/identity.rs` — 589–887: whole Binding/Lease/AcquireRequest declarations and their Spec tables/codecs.
- `lib/host/src/factory/state.rs` — 888–1021: OutputSlice, FactoryState and FactoryHarnessPin.
- `lib/host/src/factory/requests.rs` — 1022–1456: whole inspect/stop/takeover/output/export/candidate request and response declarations/codecs.
- `lib/host/src/factory/receipt.rs` — 1557–1772,1822–1991: FactoryReceipt decode/state mapping, stop-owned receipt guard and lock/load/store/tombstone/write.
- `lib/host/src/factory/launch.rs` — 1992–2139,2226–2270: launch/drive/consume_start/update_receipt; same sequence and custody.
- `lib/host/src/factory/finish.rs` — 2140–2225,2271–2458: fail/abandon/refresh_stopped/yielded/start-failure/timeout/finish and uncertainty mapping.
- `lib/host/src/factory/stop.rs` — 2459–2620,2692–2732: stop, bounded broker-close attempts, pre-delivery settlement, reconcile_run_credential and record_stop_outcome.
- `lib/host/src/factory/inspect.rs` — 2621–2691: inspect/takeover and exact copy destination.
- `lib/host/src/factory/artifacts.rs` — 2733–2830: existing bounded output/export operations.
- `lib/host/src/factory/candidate.rs` — 2831–2968: inspect_candidate/run_reason/protected candidate script.
- `lib/host/src/factory/confirmation.rs` — 2969–3194: confirm_factory_*, confirm_prepare* and exact status mappings.

Existing test allocation: common; wire; launch; stop; artifacts; candidate; confirmation. The embedded test module is 3195–6081 (2887 lines); production/pre-test material is 1–3194. New scripted regressions at 4882–4976 cover preservation of stop-owned uncertainty, pre-delivery stop and transient broker closure, and belong with the existing stop tests. Keep cohesive vectors/fixtures intact; the exact current responsibility map, rather than older branch ranges, determines case placement. This upkeep adds no module, process or API and establishes no native race proof.

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
