# Host service admission

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-eae04619505c"></a>

## [rust/soda-host/GMUX_PATCHES.md](../../../../../rust/soda-host/GMUX_PATCHES.md)

Historical integrator checklist with partially superseded instructions; active current runtime is source-grounded in main.rs/dbackend.rs, not this narrative.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–2, 5–188 | Privileged host daemon composition, private native operations and listener lifetime; declarations/fields: `gmux_admission`, `gmux_backend`, `gmux_routes`, `gmux_server`, `DaemonBackend`, `main` |
| Historical; no active owner / obsolete | 3–4 | Superseded wiring/predecessor-owner narrative; current module/daemon/native release callers contradict it |

<a id="coverage-7768dd1418a0"></a>

## [lib/host/src/gmux_admission.rs](../../../../../lib/host/src/gmux_admission.rs)

Scoped R02 reconciliation at `eaed66a9` (2026-10-07). B27's concern extraction
is already integrated. The current facade contains only policy module wiring
and exports (lines 1–16); the defining policy is
[daemon/admission.rs](../../../../../lib/host/src/daemon/admission.rs).
N11's duplicate daemon peer types/functions and two equivalence smoke tests are
retired. HTTP keeps filesystem/systemd socket authorization. Active Muse
credentials/pidfd ownership is defined separately by
[muse/socket.rs](../../../../../lib/host/src/muse/socket.rs) and
[MusePeer](../../../../../lib/host/src/muse/runtime_types.rs), exercised by
`muse::tests::peer_attestation`. No peer-UID gate is added to HTTP. These are
current locations/ranges and L18 retained-duty checks, not a fresh full-slice
validity audit or installed qualification.

| Slice / lifecycle | Defining file / current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | gmux_admission.rs 1–16 | Module `admission` and direct policy exports; no peer wrapper or DTO duplication |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | daemon/admission.rs 1–63 | BODY_LIMIT_DEFAULT/LARGE/IDENTITY, NATIVE_CLEAN_PATHS, ADMITTED_MUTATION_PATHS, IDENTITY_ACTIONS, TAILNET_ACTIONS |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | daemon/admission.rs 64–71 | TERMINAL_STREAM_CAP, TERMINAL_REQUEST_LIMIT, TERMINAL_FRAME_LIMIT |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | daemon/admission.rs 72–158 | RequestHead method/path/has_query/escaped/origin_present/upgrade_websocket/ws_key/websocket_request and post constructor; NativeRejection status/message; native validation, mutation-path and body-limit policy |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery), [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | daemon/admission.rs 159–164 | valid_identity_request shape/admission |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | daemon/admission.rs 165–177 | validate_tailnet_request action/shape/admission |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | daemon/admission.rs 178–182 | valid_terminal_request shape/admission |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | daemon/admission.rs 183–225 | AdmissionGate.held/new/acquire/try_acquire and AdmissionGuard._guard; mutex custody |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | daemon/admission.rs 226–273 | TerminalGate.live/new/try_register and TerminalSlot.live/drop; held slot lifetime |
| H01/I05 duplicate / retired by N11 | daemon/peer.rs removed | PeerCred and raw-pidfd MusePeer duplicate, peer_cred, muse_peer, close_pidfd; meaningful attestation remains in the actual Muse owner |

<a id="coverage-1c0dfc31b87c"></a>

## [rust/soda-host/src/gmux_backend.rs](../../../../../rust/soda-host/src/gmux_backend.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–66, 137–139 | Privileged host daemon composition, private native operations and listener lifetime; declarations/fields: `BackendError`, `TerminalSession`, `ExecBackend`, `StubBackend` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 67, 140–142 | Profile and runtime readiness adapter: profile; declarations/fields: `profile` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 68, 143–145 | Repository association and creation adapter: create; declarations/fields: `create` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 69, 146–148 | Profile and runtime readiness adapter: inspect; declarations/fields: `inspect` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 70, 149–151 | Profile and runtime readiness adapter: observe_os; declarations/fields: `observe_os` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 71, 152–154 | Development SSH access adapter: connection; declarations/fields: `connection` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 72, 155–157 | Project Start/Stop adapter: lifecycle; declarations/fields: `lifecycle` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 73, 158–160 | Development SSH access adapter: access_keys; declarations/fields: `access_keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 74–76, 161–163 | Human membership and accounts adapter: account; declarations/fields: `account` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 77, 164–166 | Checkout allocation and preparation adapter: prepare; declarations/fields: `prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 78, 167–169 | Checkout allocation and preparation adapter: prepare_candidate; declarations/fields: `prepare_candidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 79, 170–172 | Checkout allocation and preparation adapter: inspect_preparation; declarations/fields: `inspect_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 80, 173–175 | Checkout allocation and preparation adapter: stop_preparation; declarations/fields: `stop_preparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 81–83, 176–178 | Maintenance holds adapter: hold_preparation; declarations/fields: `hold_preparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 84, 179–181 | Assignment and dispatch adapter: factory_launch; declarations/fields: `factory_launch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 85, 182–184 | Run lifecycle and intervention adapter: factory_inspect; declarations/fields: `factory_inspect` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 86, 185–187 | Run lifecycle and intervention adapter: factory_stop; declarations/fields: `factory_stop` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 87, 188–190 | Run lifecycle and intervention adapter: factory_takeover; declarations/fields: `factory_takeover` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 88–91, 191–193 | Factory activity presentation adapter: factory_output; declarations/fields: `factory_output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 92, 194–196 | Provider execution integration adapter: factory_harness; declarations/fields: `factory_harness` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 93, 197–199 | Publication progression adapter: factory_export; declarations/fields: `factory_export` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 94–96, 200–202 | Candidate verification assessment adapter: factory_candidate_inspect; declarations/fields: `factory_candidate_inspect` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 97–102, 203–205 | Provider execution integration adapter: identity_launch; declarations/fields: `identity_launch` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 103–110, 206–208 | Provider execution integration adapter: identity_action; declarations/fields: `identity_action` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 111–118, 209–211 | Host Tailnet control adapter: tailnet; declarations/fields: `tailnet` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 119–122, 212–214 | Interactive attachment adapter: terminal_accept; declarations/fields: `terminal_accept` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 123–136, 215–222 | Interactive attachment adapter: pump_terminal; declarations/fields: `pump_terminal` |

<a id="coverage-f38bae4e0df3"></a>

## [rust/soda-host/src/gmux_routes.rs](../../../../../rust/soda-host/src/gmux_routes.rs)

 Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–25 | Private route table, availability config and response outcomes; declarations/fields: `ROUTE_TABLE`, `DaemonConfig`, `all_enabled`, `RouteOutcome`, `into_response`, `dispatch` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 26–67 | Private route table, availability config and response outcomes; declaration/member ROUTE_TABLE; declarations/fields: `ROUTE_TABLE` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 68–69 | Private route table, availability config and response outcomes; declaration/member DaemonConfig; declarations/fields: `DaemonConfig` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 70–71 | Private route table, availability config and response outcomes; declaration/member DaemonConfig.image; declarations/fields: `DaemonConfig.image` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 72–73 | Private route table, availability config and response outcomes; declaration/member DaemonConfig.tailnet_management; declarations/fields: `DaemonConfig.tailnet_management` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 74–76 | Private route table, availability config and response outcomes; declaration/member DaemonConfig.terminal_available; declarations/fields: `DaemonConfig.terminal_available` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 77–81 | Private route table, availability config and response outcomes; declaration/member DaemonConfig.identity_available; declarations/fields: `DaemonConfig.identity_available` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 82–96 | Private route table, availability config and response outcomes; declaration/member all_enabled; declarations/fields: `all_enabled` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 97–107 | Private route table, availability config and response outcomes; declaration/member RouteOutcome; declarations/fields: `RouteOutcome` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 108–122 | Private route table, availability config and response outcomes; declaration/member into_response; declarations/fields: `into_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 123–158 | Private route table, availability config and response outcomes; declaration/member dispatch; declarations/fields: `dispatch` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 159–181 | Shared native request admission and mutation gate; declarations/fields: `dispatch_native` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 182 | Profile dispatch |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 183 | Create dispatch |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 184–185 | Environment/OS dispatch |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 186 | SSH endpoint dispatch |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 187 | Project Start/Stop dispatch |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 188 | Developer key dispatch |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 189 | Human account dispatch |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 190–193 | Native preparation operation dispatch |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 194 | Maintenance hold dispatch |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 195 | Factory launch dispatch |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 196–198 | Run inspect/stop/takeover dispatch |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 199 | Factory output dispatch |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 200 | Native harness pin dispatch |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 201 | Candidate export dispatch |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 202 | Native candidate inspect dispatch |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 203–221 | Shared native response mapping |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 222–254 | Provider execution operation dispatch; declarations/fields: `dispatch_identity` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 255–296 | Tailnet host/policy/selection operation transport; declarations/fields: `dispatch_tailnet` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 297–302 | Tailnet host/policy/selection operation transport; declaration/member tailnet_unavailable; declarations/fields: `tailnet_unavailable` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 303–341 | Terminal upgrade, held stream slot and launch handoff; declarations/fields: `dispatch_terminal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 342–361 | Private HTTP response envelope rendering; declarations/fields: `reason` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 362–374 | Private HTTP response envelope rendering; declaration/member error_response; declarations/fields: `error_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 375–379 | Private HTTP response envelope rendering; declaration/member not_found_response; declarations/fields: `not_found_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 380–392 | Private HTTP response envelope rendering; declaration/member json_response; declarations/fields: `json_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 393–404 | Private HTTP response envelope rendering; declaration/member tailnet_json_response; declarations/fields: `tailnet_json_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 405 | Private HTTP response envelope rendering; declaration/member tailnet_response; declarations/fields: `tailnet_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 406–416 | Private HTTP response envelope rendering; declaration/member NO_STORE; declarations/fields: `NO_STORE` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 417–425 | WebSocket upgrade/accept key format; declarations/fields: `find_header_end` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 426–436 | WebSocket upgrade/accept key format; declaration/member websocket_upgrade_response; declarations/fields: `websocket_upgrade_response` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 437 | WebSocket upgrade/accept key format; declaration/member websocket_accept_key; declarations/fields: `websocket_accept_key` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 438–443 | WebSocket upgrade/accept key format; declaration/member GUID; declarations/fields: `GUID` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 444–504 | WebSocket upgrade/accept key format; declaration/member sha1; declarations/fields: `sha1` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 505 | WebSocket upgrade/accept key format; declaration/member base64_encode; declarations/fields: `base64_encode` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 506–523 | WebSocket upgrade/accept key format; declaration/member ALPHABET; declarations/fields: `ALPHABET` |

<a id="coverage-401a73ec277c"></a>

## [rust/soda-host/tests/gmux_smoke.rs](../../../../../rust/soda-host/tests/gmux_smoke.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–2, 5–7, 212–235, 430–477, 518–584, 703–729, 933–1030 | Privileged host daemon composition, private native operations and listener lifetime; declarations/fields: `gmux_admission`, `gmux_backend`, `gmux_routes`, `gmux_server`, `head`, `status_of`, `text_of`, `body_of`, `dispatch_stub`, `ScriptBackend`, `err`, `ok`, `replay`, `native_clean_paths_reject_query_and_escapes`, `route_table_has_every_go_route`, `native_error_mapping_matches_go`, `identity_errors_collapse_to_409_like_go`, `terminal_rejections_match_go` |
| Historical; no active owner / obsolete | 3–4 | Superseded wiring/predecessor-owner narrative; current module/daemon/native release callers contradict it |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 8–9 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member gmux_admission; declarations/fields: `gmux_admission` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 10–11 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member gmux_backend; declarations/fields: `gmux_backend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 12–13 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member gmux_routes; declarations/fields: `gmux_routes` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 14–32 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member gmux_server; declarations/fields: `gmux_server` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 33–38 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member head; declarations/fields: `head` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 39–44 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member status_of; declarations/fields: `status_of` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 45–48 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member text_of; declarations/fields: `text_of` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 49–56 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member body_of; declarations/fields: `body_of` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 57–74 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member dispatch_stub; declarations/fields: `dispatch_stub` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 75 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend; declarations/fields: `ScriptBackend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 76 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend.result; declarations/fields: `ScriptBackend.result` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 77 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend.session; declarations/fields: `ScriptBackend.session` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 78 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend.seen_bodies; declarations/fields: `ScriptBackend.seen_bodies` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 79 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend.seen_image; declarations/fields: `ScriptBackend.seen_image` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 80–83 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ScriptBackend.pumped; declarations/fields: `ScriptBackend.pumped` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 84–93 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member err; declarations/fields: `err` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 94–103 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member ok; declarations/fields: `ok` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 104–110 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member replay; declarations/fields: `replay` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 111–113 | Profile and runtime readiness adapter: profile; declarations/fields: `profile` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 114–116 | Repository association and creation adapter: create; declarations/fields: `create` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 117–119 | Profile and runtime readiness adapter: inspect; declarations/fields: `inspect` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 120–122 | Profile and runtime readiness adapter: observe_os; declarations/fields: `observe_os` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 123–125 | Development SSH access adapter: connection; declarations/fields: `connection` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 126–128 | Project Start/Stop adapter: lifecycle; declarations/fields: `lifecycle` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 129–131 | Development SSH access adapter: access_keys; declarations/fields: `access_keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 132–134 | Human membership and accounts adapter: account; declarations/fields: `account` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 135–137 | Checkout allocation and preparation adapter: prepare; declarations/fields: `prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 138–140 | Checkout allocation and preparation adapter: prepare_candidate; declarations/fields: `prepare_candidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 141–143 | Checkout allocation and preparation adapter: inspect_preparation; declarations/fields: `inspect_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 144–146 | Checkout allocation and preparation adapter: stop_preparation; declarations/fields: `stop_preparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 147–149 | Maintenance holds adapter: hold_preparation; declarations/fields: `hold_preparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 150–152 | Assignment and dispatch adapter: factory_launch; declarations/fields: `factory_launch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 153–155 | Run lifecycle and intervention adapter: factory_inspect; declarations/fields: `factory_inspect` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 156–158 | Run lifecycle and intervention adapter: factory_stop; declarations/fields: `factory_stop` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 159–161 | Run lifecycle and intervention adapter: factory_takeover; declarations/fields: `factory_takeover` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 162–164 | Factory activity presentation adapter: factory_output; declarations/fields: `factory_output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 165–168 | Provider execution integration adapter: factory_harness; declarations/fields: `factory_harness` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 169–171 | Publication progression adapter: factory_export; declarations/fields: `factory_export` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 172–174 | Candidate verification assessment adapter: factory_candidate_inspect; declarations/fields: `factory_candidate_inspect` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 175–177 | Provider execution integration adapter: identity_launch; declarations/fields: `identity_launch` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 178–184 | Provider execution integration adapter: identity_action; declarations/fields: `identity_action` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 185–191 | Host Tailnet control adapter: tailnet; declarations/fields: `tailnet` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 192–198 | Interactive attachment adapter: terminal_accept; declarations/fields: `terminal_accept` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 199–211 | Interactive attachment adapter: pump_terminal; declarations/fields: `pump_terminal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 236–263 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member native_method_check_runs_after_clean_path_check; declarations/fields: `native_method_check_runs_after_clean_path_check` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 264–283 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member mutation_gate_covers_exactly_the_go_paths; declarations/fields: `mutation_gate_covers_exactly_the_go_paths` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 284–297 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member body_limits_match_go; declarations/fields: `body_limits_match_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 298–308 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member terminal_constants_match_go; declarations/fields: `terminal_constants_match_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 309–323 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member identity_validator_matches_go; declarations/fields: `identity_validator_matches_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 324–344 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member tailnet_validator_matches_go; declarations/fields: `tailnet_validator_matches_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 345–364 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member terminal_validator_matches_go; declarations/fields: `terminal_validator_matches_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 365–373 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member mutation_gate_serializes; declarations/fields: `mutation_gate_serializes` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 374–390 | Source assertion of Execution admission and lease fencing; declarations/fields: `mutation_gate_blocking_acquire_hands_off` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 391–406, 891–898 | Source assertion of Interactive attachment; declarations/fields: `terminal_gate_caps_and_releases`, `websocket_accept_key_matches_rfc6455_vector` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 407–417 | Source assertion of Encrypted credential custody; declarations/fields: `peer_credentials_attest_self` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 418–429 | Source assertion of Native binding and private delivery; declarations/fields: `muse_peer_carries_pidfd_pin` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 478–492 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member every_post_route_dispatches_through_the_stub; declarations/fields: `every_post_route_dispatches_through_the_stub` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 493–517 | Source assertion of Provider execution integration; declarations/fields: `bodies_and_harness_image_pass_through_verbatim` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 585–596 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member native_unknown_path_is_404_and_terminal_prefix_is_405; declarations/fields: `native_unknown_path_is_404_and_terminal_prefix_is_405` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 597–616 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member native_success_envelope_matches_go_encoder; declarations/fields: `native_success_envelope_matches_go_encoder` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 617–643 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member over_limit_bodies_fail_like_go_maxbytesreader; declarations/fields: `over_limit_bodies_fail_like_go_maxbytesreader` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 644–702 | Source assertion of Repository association and creation; declarations/fields: `create_and_mutations_take_the_gate_reads_do_not` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 730–780 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member identity_rejects_bad_shape_unknown_actions_and_missing_runtime; declarations/fields: `identity_rejects_bad_shape_unknown_actions_and_missing_runtime` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 781–828 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member tailnet_error_mapping_matches_go; declarations/fields: `tailnet_error_mapping_matches_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 829–882 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member tailnet_success_has_no_trailing_newline_and_disabled_is_503; declarations/fields: `tailnet_success_has_no_trailing_newline_and_disabled_is_503` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 883–890 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member terminal_head; declarations/fields: `terminal_head` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 899–932 | Actual daemon terminal upgrade/101 and concurrency-slot lifetime assertion; `terminal_upgrade_holds_slot_and_renders_101`. Current validity review corrected the earlier maintenance-hold keyword classification; A/root/C inspected the actual subject. |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1031–1040 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member stub_pump_reports_unimplemented; declarations/fields: `stub_pump_reports_unimplemented` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1041–1047 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member read_all; declarations/fields: `read_all` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1048–1103 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member server_serves_stub_routes_and_parser_rejections; declarations/fields: `server_serves_stub_routes_and_parser_rejections` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1104–1131 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member server_runs_terminal_upgrade_and_pump; declarations/fields: `server_runs_terminal_upgrade_and_pump` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1132–1138 | Privileged host daemon composition, private native operations and listener lifetime; declaration/member systemd_listener_refuses_without_activation; declarations/fields: `systemd_listener_refuses_without_activation` |
