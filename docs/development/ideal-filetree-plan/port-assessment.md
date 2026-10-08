# Port assessment and cutovers

## Current remaining implementation ports

The task schedule was reconciled against `de65ff68` on 2026-10-06; its source
delta from the audit pin is documentation only. **One Go→Rust implementation
port remains**, the native `soda-forgejo-tailnet` helper and its exclusive
configuration rewrite, specified by [N07](reviews/N07.md) and
[C04](implementation-tasks.md#c04-project-tailnet-and-forgejo-helper-port).
**No Rust→Go implementation port is selected.** Package moves, concern splits,
consolidations and retirement of already-ported predecessors are separate tasks.

`soda-candidate-check` is already declared in
`rust/soda-release-tools/Cargo.toml` and calls the current Rust `check_cli`
implementation; its old Go source is absent. Current Rust host Tailnet controls
also exist. Do not derive additional port tasks from historical predecessor rows.
The [Python cutover reconciliation](#python-cutover-closure) below records current
probe/test source, including the already-landed shell/Go implementations.
These are inspected source facts, not compiled or installed qualification.

## Was the Rust port a good choice?

The source supports selective assessment, not a blanket language verdict. No
startup, memory, throughput, binary-size or runtime-defect comparison was run
or found in this review. Source-line totals describe authored maintenance,
including comments and tests; they do not measure performance or safety.

| Port | What the current source establishes | Recommendation for this refactor |
| --- | --- | --- |
| Host project/preparation/Tailnet daemon | Current `f7e9cf9d` source has Rust daemon, native adapters, binary manifest and Rust release compile selection; Go daemon/executor directories are absent | Retain at `lib/host` with the same-package entrypoint under `cmd/soda-host`. Assess real behavior and remaining source debt in slice records; do not recreate predecessors or treat source wiring as installed proof. |
| Project terminal | Replaces four embedded Python programs at one existing installed helper boundary; native file descriptors, PTY and process handling stay inside the Project | Retain one helper and split its real terminal/subscription/key concerns. This is the strongest current boundary for a selective native port; benefit still needs runtime evidence. |
| Identity broker | Real replacement of the former Go broker; retains an established custody service | Consolidate provider packages. Reassess handwritten PostgreSQL transport/codecs and cross-language schema/wire maintenance before expanding them. The separate service predates the language change. |
| Installer + image import | Rust release-build/deliver and native installer/import owners are current; predecessor Go release code is absent. D05/D11 compare their actual guarantees and caller contracts. | Retain both native commands. Select existing release-deliver payload/content as the delivered wire/admission owner and existing release-build OCI/layout as the pure OCI/content owner, as recorded in D11-T3/E8. Rebind only after exact null/duplicate/error/result/confinement/compression and dependency cutover checks. Preserve native installer authority and cancelled importer lifecycle; no FFI, RPC, extra crate or service. |
| Acceptance | Rust owns outside driver/remote handling; Go installed probe orchestration and the personal-Git shell payload remain live | Retain real owners and remote binary; reconcile actual probe assumptions and old guidance. Python predecessors are absent in the inspected current source; no new Python port is scheduled. Preserve live probe input support. |
| Factory CLI + Muse wrappers | Actual thin client/helper duties; Rust adds manual flag, transport and Go compatibility parsing | Retain the recorded Rust owners and assess maintenance/primitive consolidation against actual callers. A language-change recommendation requires an explicit owner decision; it cannot silently override the selected target. The coordinator remains Go. |
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

Historical installer/import duplication was explicit in
`rust/soda-image-import/src/main.rs:1-10,550-669,1078-1200` and
`rust/soda-install/src/deliver.rs:106-284`, compared with predecessor Go owners at
`internal/release/deliver/payload.go`, `content.go` and
`internal/release/build/oci_layout.go`. Those Go release/host daemon paths are
absent at `f7e9cf9d`; current Rust release-build, release-deliver, installer and
image-import owners must be compared by their actual guarantees and callers.
D05/D11 record that comparison and exact retained targets; no absent Go owner
is retained or awaited as a future prerequisite. The earlier host predecessor
used payload loading/native admission at `internal/host/daemon.go:61-80`.
Port history was checked
at `977ebe55` (installer), `0123e689` (import), `c753342a` (acceptance),
`a096a06c` (identity), `c439c209` (host library), `899e9bf3` (terminal),
`999f42ff` (factory CLI), `1c3a7cdc` (Muse), `5abc5733` (stage/render) and
`179a8723` (locales). The release cutover wired tools → image →
build/deliver, preserved candidate-check and payload data, and removed the
Go pipeline: Rust orchestrates the release producer.

The selected reuse defining owners are `lib/soda-release-deliver/src/payload.rs`
and `content.rs` for delivered payload wire/load/admission and six-image binding,
and `lib/soda-release-deliver/src/oci/` for live layout and shared layer scanning.
Build retains `lib/soda-release-build/src/oci.rs` and its archive/manifest/content
submodules. `DEAD-REL-OCI-LAYOUT-1` supersedes the earlier selection of build’s
uncalled `oci_layout.rs`; its exclusive oracle/fixture does not make it a current
product owner. The four Rust crate and native command owners remain. C's actual
guarantee comparison and A's independent source-fit challenge establish that
direction; they do not prove the private copies equivalent. D11-A3 retains the
exact decoder, error, null, duplicate, result, no-follow/confinement and decoded
gzip EOF gates before copy retirement. The present importer is std plus
soda-json while build has HTTP acquisition dependencies: specify that dependency
and footprint cutover without importing acquisition behavior into an installed
operation. Root/native platform, signal/deadline and Podman execution stay with
`cmd/soda-image-import/src/{context,platform,import}.rs`; the generic delivered
NativeEngine adapter does not replace that lifecycle. Disk/authentication and
single-attempt installation remain with `cmd/soda-install`.

### Ownership decisions before further splitting

The live Go Forgejo Tailnet helper has a decided Rust successor, independently
challenged by A: `cmd/soda-forgejo-tailnet/main.rs` is an additional binary in
the existing `soda-host` package; `lib/host/src/tailnet/forgejo.rs` defines its
native behavior and private tests. [Exact cutover](decomposition/host-runtime.md#current-forgejo-tailnet-helper-allocation)
preserves the installed helper identity and listener guard, with explicit
package/binary compile selection. The target excludes its exclusive Go
entrypoint and environment rewrite implementation after that cutover.

| Current code | Observed consumers | Required disposition decision |
| --- | --- | --- |
| Former Go acceptance Evidence/Execute/Remote/Command and uncalled Worker closure | Current live owners are Rust command/evidence and Go installed-probe private inputs/Process. | Retired at `dbdb0615` after same-FD bounded private-input custody landed in `469f47f5`; exclusive tests/API adapters are gone. Existing Go StartCommand/Process remains for actual bounded probe callers and its real cleanup test. The earlier all-process retirement claim is superseded by this caller evidence. No new package, facade or compatibility implementation; installed qualification remains separate. |
| Historical Go deliver/import.go | Predecessor path is absent at f7e9cf9d; current native import is Rust | Already retired; do not recreate or await it. Preserve the selected current Rust pure-owner/native-lifecycle cutover above. |
| Rust release-inputs readers | Forgejo reader is used by release-build `forgejo.rs:10`; signature admission by `coreos.rs:398`. Settings helpers have no external production caller; production.rs duplicates them locally | Retain Forgejo/signature and the other live readers. Consolidate matching recipe/unit/command inventory helpers into the existing settings owner and replace the duplicate release-build helpers, preserving their actual semantics. Do not retire the live reader closure or create a helper service. |
| Rust release-build mirrored progress/clock/BuildExecution closure | Current production uses release-tools progress and release-image Runner; the build mirror is consumed only by its own tests and two exclusive oracle cases. D03 coordinator source inspection and A's independent source/caller census agree. | Retire the mirror, its exclusive clock/tests/oracle/support and module imports at cutover; omit all six predecessor target leaves. Preserve actual live progress/execution and the active build crate/Error/OCI/input/production duties. The definition-only release-tools `record.files_map` also retires; other record duties remain. |
| Go broker credential-write helpers | Seed/fixture callers; metadata reads still live | Move test-only seeding through the real fixture/broker boundary when authorized. Preserve live reads, schema and persisted encryption contract. |

The main tree represents the selected final owners. Retiring sources are
accounted for in disposition tables and review entries, not carried as target
implementations. Pending wiring or consolidation details do not reopen the
decided language choices.

### Current source integration concern

`rust/soda-acceptance/src/driver.rs:476-485` reads
`soda-acceptance-remote` beside the current driver executable, and native action
uses those bytes at lines 528–536. Current Rust candidate assembly in
`rust/soda-release-image/src/build.rs:671–707` compiles only the driver at693–697;
it has no companion selector there. The previous Go image path is absent at the
review baseline. [D03-F3](reviews/D03.md#concrete-findings) records the conditional
missing-sibling integration finding, independently challenged by A. Compile the
existing soda-acceptance package's soda-acceptance-remote binary into the same
artifacts/tools set before its recorded inventory, preserving its current target
binding. No new binary/service or reproduced candidate failure is claimed.

### Host cutover integration

The PR26 observations and pending-port rows below describe the historical branch
inspection. They are superseded as current cutover status by the pinned audit
source: `rust/soda-host/Cargo.toml` declares `src/main.rs`; `main.rs`, `dbackend.rs`,
`iconfig.rs`, `tcontrol*.rs` and daemon/broker/terminal modules implement the
retained service; release-image `build.rs:522-553` compiles it. The Go daemon and
its Project/terminal/Tailnet executor directories are absent. These historical
rows are not executable remaining-port instructions. H01, networking, Project,
identity and H06 records assess current duties and exact target moves before
closing their allocation dimensions. No installed cutover proof is claimed.

The current Rust host has a library and real daemon binary. The earlier inspected
`pr/26` revision, `b306756d00c6801278bb205ea0c8bc9de1d0456a`, lacked that
entrypoint and concrete backend; those limitations describe that historical
revision, not the current pinned source. The target carries one
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
Endpoint client plus its tests. The Go native control, policy publication,
provider/enrollment and RunBinding implementations are retired; Rust remains
their defining owner. The ProjectStatus parser had no surviving Go production
caller after companion retirement and already had a Rust counterpart; it was
retired with the same source cut.

These integration leaves retain existing callers. The daemon rows describe the
historical PR26 gaps; the Tailnet rows below now record the current Rust source
after the N03 primary review and A's independent presence/caller challenge.
Target moves and concern splits are **decided-pending**. Existing implementation
presence does not close behavioral findings, runtime admission questions or
installed proof:

| Target | Existing responsibility and evidence |
| --- | --- |
| `cmd/soda-host/main.rs` | same -config/-tailnet-action/-project CLI/root arguments/exit0,78,1; binary calls soda_host rather than compiling duplicate library modules. Evidence: cmd/soda-host/main.go:20-38,41-70,111-149 |
| `lib/host/src/daemon/mod.rs` | Retain/extract current Rust main.rs:185–440 host construction, root/fd3 activation, Muse listener, signal shutdown and bounded Muse→mux drain ordering. The former Go daemon is absent; this is current same-host ownership, not a pending port. H01 records shutdown/caller limits. |
| `lib/host/src/daemon/config.rs` | Retain/extract current iconfig.rs host JSON/defaults/runtime/Muse configuration and main's admitted configuration wiring. Rebind actual dbackend construction; do not recreate the historical Go parser or treat configured runtime admission as automatically correct. |
| `lib/host/src/daemon/backend.rs` | Retain/extract current dbackend.rs:125–251 concrete construction and actual Project/preparation/Factory/terminal/Tailnet adapters. Preserve current native validators, callers and installed boundaries. Historical PR26 stubs are not the current implementation. |
| `lib/host/src/daemon/websocket.rs` | Retain/extract the live dbackend pump_terminal entry at 1052 and native pump at 1460–1620, with current mux admission/first-frame/expiry/writes. Correct H01-F3's conditional pipe-read/mutex obstruction in this existing owner; source wiring alone is not installed transport proof. |
| `lib/host/src/daemon/broker.rs` | Retain/extract current dbackend FactoryBroker adapters at 614 onward and IdentityBroker adapters at 717 onward, using the one existing iclient::BrokerClient constructed at 137/181/197/251. Rebind the actual client/private seams and tests; keep existing soda-identity service and sockets, with no pending Go port or extra broker process. |
| `lib/host/src/tailnet/control/mod.rs` | Retain/move current `tcontrol.rs:45-341` in the existing host crate/process. `dbackend.rs:158-192,1017-1040` constructs and dispatches the concrete Control; `main.rs:193-224` supplies backend configuration. No missing-port instruction. Configured Project-runtime admission is a separate N04/N06 question. |
| `lib/host/src/tailnet/control/native.rs` | Retain/move current `tcontrol_native.rs`: real bounded LocalAPI request, status observation, host actions/readback and CLI passthrough. [N03](reviews/N03.md) owns the actual HTTP-framing finding and target review; presence is not protocol correctness or native proof. |
| `lib/host/src/tailnet/control/provider.rs` | Retain/move current `tcontrol_provider.rs` and its actual caller from `tcontrol_enroll.rs`: bounded OAuth token/key operations already exist. N04/N06 allocate current duties and assertions; do not create a second provider executor. |
| `lib/host/src/tailnet/control/policy.rs` | Retain/move current `tcontrol_policy.rs`: protected state/locking, exact policy records and publication already exist. N04/N05 reconcile precise policy/Project extraction seams and callers. |
| `lib/host/src/tailnet/control/enrollment.rs` | Extract the actual enrollment duties from current `tcontrol.rs`, `tcontrol_policy.rs` and `tcontrol_enroll.rs`; this is a same-package responsibility split, not an absent implementation. N04/N06 own exact admission, revision and consumption targets. |
| `lib/host/src/tailnet/control/project.rs` | Extract the actual saved Project policy/RunBinding duties from current `tcontrol_policy.rs:742-921`, `tcontrol.rs` and `tcontrol_enroll.rs:152-214`. N05/N06 own precise allocations; preserve the real companion boundary. |
| `lib/host/src/tailnet/control/wire.rs` | Retain/move current `tcontrol_wire.rs` rather than manufacture another set of records. Go `control_types.go`, `control_validation.go` and the surviving read-only status client/selection contract remain Go responsibilities. |
| `lib/host/src/tailnet/control/tests.rs` | Reallocate current `tcontrol_oracle.rs` assertions by their actual host/policy/Project responsibility. Retire only superseded privileged Go Control execution assertions with the corresponding cutover; retain live Go DTO/client/validation tests. N03/N04/N05/N06 record exact units and evidence limits. |

The existing service, fd3 operation listener, Muse unixpacket listener and
same-binary Tailnet action remain the execution boundaries. The Rust backend
must replace the full real operations rather than install a production stub.
Native terminal admission/expiry/first-frame/write/pump lifetime and the current
identity socket contract are part of cutover, not another daemon proposal.

The retained host concern allocations below originated in earlier branch
inspection. Current slice records reconcile the merged implementation, exact
live/obsolete duties and tests at `f7e9cf9d`; historical comments and offsets
do not establish a missing port or current validity. Whole test cases and
fixtures remain coherent. These allocations do not create additional source
coverage beyond the catalog and its explicit review receipts:

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

**rust/soda-host/src/texec.rs** (4,832 lines at f7e9cf9d). Current reviewed cuts below supersede the older branch intervals for those duties; unrefined adjacent selectors remain explicitly historical.

- `lib/host/src/terminal/mod.rs` — 1495–1507: Service/EndIdentityHook plus shared fixed agent/Podman helpers1531–1559; one Service owner. Retire the definitions-only private request predicate and StreamTable at 2367–2440, their exclusive stream_table_flows test at 4561–4584, and only the predecessor predicate assertions at 4586–4591. Keep identity assertions at 4592 onward and actual gmux_admission/TerminalGate authority. [S05's actual caller census and independent challenge](reviews/S05.md) supersede the historical service.go port comments.
- `lib/host/src/terminal/protocol.rs` — 40–251: limits/errors, terminal IDs/names/dimensions and existing strict base64.
- `lib/host/src/terminal/request.rs` — 252–351,384–420,554–658: TerminalRequest/TerminalState whole declarations, specs, codecs and request admission.
- `lib/host/src/terminal/frame.rs` — 352–383,421–553,659–772: TerminalFrame declaration/specs/decode/encode and input/output admission. find_field421–435 has only decode_terminals438 caller and remains one frame helper.
- `lib/host/src/terminal/identity_wire.rs` — 788-983,1065-1080,1119-1178: RFC3339/string-i64 and whole Binding record/spec/codec.
- `lib/host/src/terminal/lease.rs` — 984-1064,1081-1118,1179-1320: Lease/Delivery/AcquireRequest whole record/spec/codecs.
- `lib/host/src/terminal/target.rs` — 1321–1494,1509–1529,1560–1619: TerminalInspection/isolation/id-map/exit code and exact project/factory-container methods.
- `lib/host/src/terminal/identity_protocol.rs` — 773–785,1620–1743,1995–2027: single json_valid/credential_valid predicates plus IdentityRequest, reservation/binding checks and identity_result.
- `lib/host/src/terminal/identity.rs` — 1744-1994,2028-2110: identity_call/prepare/identity/managed_end, target/harness verification and credential-bound transfer.
- `lib/host/src/terminal/native.rs` — 2111–2366: agent path/hash/argv, output line decode and NativeAttach/Drop.
- `lib/host/src/terminal/launch.rs` — 2443-2655: TerminalStart/IdentityBroker, random execution ID and identity_launch/route/action.

Existing test allocation: common; protocol; identity_wire; target; identity; launch. Source grouping: 2656-2752 fixture closure; name/id/request/base64/frame goldens2753-3316; identity wire3317-3553; target/argv3554-3957; native identity3958-4554; stream/start/launch4555-4826.

**rust/soda-host/src/tcodex.rs** (3,089 lines at f7e9cf9d).

The [current shared Factory terminal allocation](decomposition/host-runtime.md#current-shared-factory-terminal-ownership)
is authoritative for both-family run/native/binding/lifecycle/output/artifact
units and their private tests. Remove the inherited `terminal/codex/run.rs`
target. Shared models and mechanics have direct shared owners; providers do not
import them through a Codex facade. The current whole defining units below
replace the former 3,263-line branch selectors.

- `lib/host/src/terminal/codex/mod.rs` — Codex module/private test declarations and imports only; the existing Service definition remains terminal/mod.rs, and shared native methods move to terminal/factory/native.rs.
- `lib/host/src/terminal/codex/paths.rs` — complete provider run-path/guest functions228–252, Codex path declaration/builder284–323 and binding436–471. Preserve the fourth .codex tuple element; generic unit/takeover paths move to their shared owners.
- `lib/host/src/terminal/codex/commands.rs` — supervisor345–372, Codex reserve-exec506–541, setup556–569 and start gate591–600. Shared quoting/unit argv/install/stage-file/retirement/output/export/takeover builders move to their selected shared owners, not duplicate commands.
- `lib/host/src/terminal/codex/reserve.rs` — complete native reserve/setup/stage/stage-host835–1043, retaining Codex harness policy and direct shared native imports.
- `lib/host/src/terminal/codex/start.rs` — start1045–1081, wait1105–1125 and validate1127–1133. Shared stage-file1083–1103 moves to factory/native.rs, preserving opaque bytes and marker order.
- `lib/host/src/terminal/codex/stop.rs` — provider stop1135–1147 and capture/finish/unbound/live1229–1276. Shared retirement/observation/container helpers have one factory/lifecycle.rs owner.
- `lib/host/src/terminal/codex/artifacts.rs` — provider output adapter1278–1296 only. Generic export/takeover/DTO/size and broker family dispatch move to the selected shared owners.

Existing Codex test allocation remains common/wire/reserve/lifecycle/artifacts,
with actual whole defining cases and single private fixtures preserved. Move
ONLY the inspected generic run case1670–1796 and export/takeover cases2839–2916/
2919–3025 to terminal/factory/tests/{run,artifacts}.rs. Preserve whole mixed
quote/path/unit-show/script/output/callback cases at their Codex test destinations
with direct shared subject imports. The current cfg(test)1487/module1488 shell
and final closing shell3089 occur once at Codex tests/mod.rs; shared factory
private tests get their own declaration shell, never a second mock/Service.

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

The candidate verification port is already present: the current
`rust/soda-release-tools/src/bin/soda-candidate-check.rs` calls `check_cli::main`,
and its manifest already declares the binary and release-deliver dependency.
The desired concern owner is `lib/soda-release-tools/src/candidate_check.rs`
with the retained `src/bin/soda-candidate-check.rs` entrypoint. Rebind the
existing implementation/imports as part of the move/split. Keep `--candidate`, `--arch`,
`--soda-revision`, `--forgejo-revision`, current parsing/refusal order, quiet
success and one-line error/exit behavior. Preserve `scripts/check-native.sh`
through this already-Rust binary; retain delivered archive verification and exact Soda/Fountain
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
does not invent a replacement for the existing Compose tool. At the scheduling
source, `git ls-files '*.py'` returns no paths. Inspected personal-Git probe source
is POSIX shell, lifecycle observation is Go, and current probe/build tests use
their native subjects rather than Python drivers. This is a source census, not
a run of the active no-Python gate. The former conversion rows are reconciled
below; do not schedule ports of deleted programs:

| Current source / caller | Target owner and required cutover |
| --- | --- |
| `internal/acceptance/personal_git.go` | Current `gitRemoteTemplate` is POSIX shell piped to `sh -se`, with a real no-Python assertion. Keep Go prepare/exercise/unlock orchestration and preserve the actual key/agent flow. The selected existing Rust remote payload may be rebound only after exact caller equivalence; this is reuse/cutover, not a new Python port. Preserve SSH identity, protected passphrase input, public-key-only output, live-agent refusal and encrypted-key lifetime. |
| Former `internal/acceptance/lifecycle_state.go` | Retired with its exclusive fixtures and CLI selector in `8b10ab14` after independent current-caller review: no installed journey selected the obsolete dashboard SQLite observer. Current PostgreSQL/lifecycle and Rust project-state owners remain; their native qualification is separate. Retain the staged Forgejo SQLite test fixture/module. No replacement observer, query translation or second database is selected. |
| `tools/soda-installed-probes/{cockpit_account,project_state}_test.go`, `tests/build/u08_state_test.go` | Current native test subjects are retained. Rebind moved actual Rust cockpit/project-state/remote owners and preserve PAM denial/error distinction, root/native admission, private-file refusal and snapshot bounds. Do not schedule removal of already-absent Python import drivers. |
| Historical `project-os/rootfs/usr/libexec/soda/{project-account,project-factory-roles}` | Python sources are absent; current compiled Rust helper crates and tests exist. Fold those existing implementations into the Project-terminal package and preserve both installed executable identities, explicit build/hash selectors and real tests. Do not track generated outputs or add wrappers. |
| `tests/build/{project_account,project_factory_roles}_test.go` | Current Rust compiled-helper drivers remain active. Preserve their actual assertions and explicit requested binary selection during Project-terminal package consolidation; historical Python SourceFileLoader disposition is obsolete. Exact current production/test seams are recorded in project-runtime.md and P03/P07. |
| `tests/build/{source_checks,workload_probe,project_foundation}_test.go`, `tests/build/helpers.go` | Current Go/shell fixtures replace the old Python doubles. Rebind actual command/environment/status/ordering assertions; the stale driveModule/Python3 comment is not a live helper. Preserve real admitted helper environment and no-replayed-mutation evidence. |
| `scripts/{check-source,check-native,check-no-python}.sh`, `tests/build/source_checks_test.go` | Preserve the existing Go/Rust/Bun/no-Python sequence and verification CLI. `check-native.sh` already invokes Rust `soda-candidate-check` with the candidate, architecture and exact revisions; only actual moved selectors require rebinding. |
| Former `scripts/{check-ruff-format,check-ruff,check-py-complexity,ruff-env}.sh`, `pyproject.toml`, `requirements-ruff.txt` | These Python-only tool inputs are already absent at the audit pin; they have no successor or remaining deletion task. Retain current source gates and do not recreate Ruff tooling. |
| `docs/development/python.md` | Retain its current active Python-elimination and gate contract at the same path. Its earlier tooling-guide content is gone; the remaining canonical policy document is not dead code. Correct stale Python setup/tooling labels in the owning development indexes during separately scoped guidance upkeep. |

The extensionless helper destinations are installation outputs, not additional
source packages. The two replacement entrypoints share one existing Cargo
package; the ephemeral probe uses one existing acceptance payload. This adds
no daemon, socket, container or sidecar. Python cutover verification must check
retained executable bodies and real test invocations, not only filename suffixes.
