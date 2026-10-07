# Host protocols and clients

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-19e2e83b9c06"></a>

## [lib/host/src/iclient.rs](../../../../../lib/host/src/iclient.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–34, 168–262, 280–453; current module/import/attribute shell; declaration check_response_limit; declaration ConnectionWire; fields provider_id, id, owner_id, label, email, plan, generation, state; declaration deserialize; declaration ConnectionVisitor; declaration Value; declaration expecting; declaration visit_map; declaration BindingWire; fields child_id, uid, gid, scope, credential_root, invocation_id, kind, id, project, login, generation; declaration BindingVisitor; declaration ExecutionWire; fields binding, kind, execution_id, digest, state, lease_id; declaration ExecutionVisitor | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/iclient.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 35–72, 152–167, 481–515; declaration RESPONSE_LIMIT; declaration ERROR_PREFIX_LIMIT; declaration HEAD_LIMIT; declaration ERR_BUSY; declaration BrokerClient; fields socket_path; declaration Execution; fields binding, kind, execution_id, digest, state, lease_id; declaration execution_is_terminal; declaration unavailable; declaration map_error; declaration new; declaration call | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | RESPONSE_LIMIT: implement the current Unix-socket service and process lifetime duty in iclient.rs.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 73–114, 454–480, 516–524, 590–601; declaration encode_acquire; declaration decode_execution; declaration acquire; declaration get_execution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | encode_acquire: implement the current execution admission and lease fencing duty in iclient.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 115–151; declaration encode_request | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | encode_request: implement the current wire decoding, encoding, and representation conversion duty in iclient.rs. — current source lib/host/src/iclient.rs; lines 115-151; module/caller wiring inspected |
| 263–279, 554–566; declaration decode_connections; declaration available | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | decode_connections: implement the current identity grants and connection availability duty in iclient.rs.; available: implement the current identity grants and connection availability duty in iclient.rs. — current source lib/host/src/iclient.rs; lines 263-279; module/caller wiring inspected; current source lib/host/src/iclient.rs; lines 554-566; module/caller wiring inspected |
| 525–539; declaration register | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | register: implement the current native binding and private delivery duty in iclient.rs. — current source lib/host/src/iclient.rs; lines 525-539; module/caller wiring inspected |
| 540–553, 567–589, 602–612; declaration reconcile_lease; declaration end_lease; declaration return_lease; declaration close_execution | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | reconcile_lease: implement the current completion, revocation, and reconciliation duty in iclient.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-779d14ff834f"></a>
<a id="coverage-910eae97c1db"></a>

## [lib/host/src/tcontrol_wire.rs](../../../../../lib/host/src/tcontrol_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–386, 409–426, 496–708, 778–793, 825–875, 886–1046, 1115–1343; current module/import/attribute shell; declaration err_invalid; declaration err_conflict; declaration err_unsupported; declaration err_unavailable; declaration err_unconfirmed; declaration err_not_enrolled; declaration err_ipv4_unavailable; declaration zero_string; declaration WIRE_BODY_LIMIT; declaration WIRE_RESPONSE_LIMIT; declaration is_lower_hex; declaration is_token_char; declaration is_hex32; declaration valid_revision; declaration valid_tag; declaration valid_tskey; declaration valid_client_secret; declaration valid_auth_key; declaration valid_client_id; declaration valid_network; declaration lower_char; declaration valid_label; declaration canonical_magic_dns_name; declaration ParsedAddr; fields addr, zone; declaration parse_addr; declaration v4_global_unicast; declaration is_global_unicast; declaration check_addresses; declaration canonical; declaration valid_exit_node_ip; declaration parseable_addr; declaration valid_prefix; declaration first_ipv4; declaration HostPreferences; fields want_running, exit_node_id, exit_node_ip, allow_lan, advertise_exit_node; declaration Peer; fields id, dns_name, addresses, online, exit_node, expired; declaration peer_view; declaration HostView; fields tailnet, magic_dns_enabled, revision, state, have_node_key, expired, dns_name, addresses, peers, health_issues, preferences; declaration HostRequest; fields action, revision, confirm, exit_node, allow_lan, advertise; declaration HostResult; fields outcome, host, readback_unavailable, auth_url; declaration deserialize; declaration RequestVisitor; declaration Value; declaration expecting; declaration visit_map; declaration SelectionVisitor; declaration strict_body; declaration decode_host_request; declaration decode_enrollment_request; declaration decode_project_selection; declaration has_extra_fields; declaration validate_signin; declaration validate_confirmed; declaration validate_exit_node; declaration validate_advertise; declaration validate; declaration authentication_url; declaration valid_tailnet; declaration valid_inventory; declaration valid_backend_state; declaration valid_peers; declaration valid_exit_node; declaration push_comma; declaration push_str; declaration push_bool; declaration push_int; declaration push_str_list; declaration encode_into; declaration encode | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol_wire.rs into its current native target.; 96 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 387–408, 427–454, 709–777, 876–885; declaration EnrollmentView; fields revision, binding, tailnet, tags, configured, admission, default, preauthorized, credential_checked, enrollment_verified, runtime_supported; declaration SettingsView; fields host, host_unavailable, enrollment; declaration EnrollmentRequest; fields action, revision, tailnet, tags, preauthorized, client_id, client_secret, default; declaration zero_secret; declaration EnrollmentResult; fields outcome, saved, credential_checked, enrollment; declaration validate_enrollment_tags; declaration validate_enrollment_policy; declaration validate_mutation; declaration has_payload; declaration validate_toggle; declaration valid_identity | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | EnrollmentView: implement the current Project Tailnet enrollment policy duty in tcontrol_wire.rs.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 455–495, 794–824, 1047–1114; declaration ProjectOptions; fields revision, binding, tailnet, available, default; declaration ProjectSelection; fields enabled, revision, binding; declaration check_project_request; declaration valid_project_availability; declaration valid_project_identity; declaration valid_project_persistence; declaration valid_project_runtime; declaration validate_project_view | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | ProjectOptions: implement the current Project Tailnet selection duty in tcontrol_wire.rs.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-922e896428d2"></a>

## [lib/unix-http/src/lib.rs](../../../../../lib/unix-http/src/lib.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–325; current module/import/attribute shell; declaration Limits; fields header_bytes, body_bytes; declaration Response; fields status, body; declaration Error; declaration request; declaration request_inner; declaration collect_response; declaration map_protocol_error; declaration tests; declaration NEXT_SOCKET; declaration socket_path; declaration serve; declaration serve_owned; declaration call; declaration accepts_chunked_body_and_trailers; declaration accepts_close_delimited_body; declaration rejects_conflicting_framing_and_body_over_cap; declaration enforces_response_header_byte_cap; declaration deadline_covers_trickling_response | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/unix-http/src/lib.rs into its current native target.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-2f335531b13c"></a>

Former source `rust/soda-host/src/iclient.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-4bb6abcac2d5"></a>
<a id="rustsoda-hostsrcjsonrs-1"></a>

Former source `rust/soda-host/src/json.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-bfc29b0aa778"></a>

Former source `rust/soda-host/tests/iclient_oracle.rs`; consult its pinned earlier Git source and the current coverage disposition.
