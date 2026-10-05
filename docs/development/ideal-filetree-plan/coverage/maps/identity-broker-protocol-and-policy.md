# Identity broker protocol and policy

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-3e4294f35465"></a>

<a id="rustsoda-identitysrccontrolrs-1"></a>

## [rust/soda-identity/src/control.rs](../../../../../rust/soda-identity/src/control.rs)

 Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–12 | Enrollment provider/session interface; declarations/fields: `EnrollmentSession`, `snapshot`, `finish`, `close`, `Provider`, `start` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 13 | Enrollment provider/session interface; declaration/member EnrollmentSession; declarations/fields: `EnrollmentSession` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 14 | Enrollment provider/session interface; declaration/member snapshot; declarations/fields: `snapshot` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 15 | Enrollment provider/session interface; declaration/member finish; declarations/fields: `finish` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 16–18 | Enrollment provider/session interface; declaration/member close; declarations/fields: `close` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 19 | Enrollment provider/session interface; declaration/member Provider; declarations/fields: `Provider` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 20–22 | Enrollment provider/session interface; declaration/member start; declarations/fields: `start` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 23 | Exact native runtime binding callback seam; declarations/fields: `Runtime` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 24 | Exact native runtime binding callback seam; declaration/member validate; declarations/fields: `validate` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 25 | Exact native runtime binding callback seam; declaration/member stop; declarations/fields: `stop` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 26–28 | Exact native runtime binding callback seam; declaration/member finish; declarations/fields: `finish` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 29–39 | Controller errors and provider adaptation; declarations/fields: `map_provider_error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 40 | Codex enrollment adapter; declarations/fields: `CodexProvider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 41–42 | Codex enrollment adapter; declaration/member MuseProvider; declarations/fields: `MuseProvider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 43 | Codex enrollment adapter; declaration/member CodexSession; declarations/fields: `CodexSession` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 44–46 | Codex enrollment adapter; declaration/member MuseSession; declarations/fields: `MuseSession` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 47–49, 59–61 | Codex enrollment adapter; declaration/member snapshot; declarations/fields: `snapshot` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 50–52, 62–63 | Codex enrollment adapter; declaration/member finish; declarations/fields: `finish` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 53–58 | Codex enrollment adapter; declaration/member close; declarations/fields: `close` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 64 | Muse enrollment adapter; declarations/fields: `close`, `start` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 65–70 | Muse enrollment adapter; declaration/member close; declarations/fields: `close` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 71–85 | Muse enrollment adapter; declaration/member start; declarations/fields: `start` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 86 | Ephemeral pending enrollment state; declarations/fields: `EnrollmentEntry` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 87 | Ephemeral pending enrollment state; declaration/member EnrollmentEntry.owner; declarations/fields: `EnrollmentEntry.owner` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 88 | Ephemeral pending enrollment state; declaration/member EnrollmentEntry.label; declarations/fields: `EnrollmentEntry.label` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 89 | Ephemeral pending enrollment state; declaration/member EnrollmentEntry.provider_id; declarations/fields: `EnrollmentEntry.provider_id` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 90 | Ephemeral pending enrollment state; declaration/member EnrollmentEntry.session; declarations/fields: `EnrollmentEntry.session` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 91–93 | Ephemeral pending enrollment state; declaration/member EnrollmentEntry.result; declarations/fields: `EnrollmentEntry.result` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 94 | Ephemeral pending enrollment state; declaration/member State; declarations/fields: `State` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 95 | Ephemeral pending enrollment state; declaration/member State.store; declarations/fields: `State.store` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 96 | Ephemeral pending enrollment state; declaration/member State.providers; declarations/fields: `State.providers` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 97 | Ephemeral pending enrollment state; declaration/member State.runtime; declarations/fields: `State.runtime` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 98–100 | Ephemeral pending enrollment state; declaration/member State.enrollments; declarations/fields: `State.enrollments` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 101 | Broker construction and state mutex; declarations/fields: `Controller` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 102–105 | Broker construction and state mutex; declaration/member Controller.state; declarations/fields: `Controller.state` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 106–128 | Broker construction and state mutex; declaration/member new; declarations/fields: `new` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 129–132 | Broker construction and state mutex; declaration/member lock; declarations/fields: `lock` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 133–143 | Pending enrollment bookkeeping; declarations/fields: `pending_enrollment` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 144–151 | Owner connection authority; declarations/fields: `owned` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 152–158 | Owned connections and available delegated metadata; declarations/fields: `connections` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 159–165 | Owned connections and available delegated metadata; declaration/member available; declarations/fields: `available` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 166–202 | Start/poll/cancel owner enrollment; declarations/fields: `start_enrollment` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 203–255 | Start/poll/cancel owner enrollment; declaration/member enrollment; declarations/fields: `enrollment` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 256–272 | Start/poll/cancel owner enrollment; declaration/member cancel_enrollment; declarations/fields: `cancel_enrollment` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 273–278 | Grant observation/creation; declarations/fields: `grants` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 279–297 | Grant observation/creation; declaration/member create_grant; declarations/fields: `create_grant` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 298–316 | Owner lease observation; declarations/fields: `leases` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 317–338 | Acquire/admit/replay/reserve exact execution fence; declarations/fields: `acquire` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 339–358 | Acquire/admit/replay/reserve exact execution fence; declaration/member admit_execution; declarations/fields: `admit_execution` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 359–396 | Acquire/admit/replay/reserve exact execution fence; declaration/member replay_acquisition; declarations/fields: `replay_acquisition` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 397–427 | Acquire/admit/replay/reserve exact execution fence; declaration/member reserve_execution_lease; declarations/fields: `reserve_execution_lease` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 428–438 | Execution fence read; declarations/fields: `get_execution` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 439–476 | Close exact execution; declarations/fields: `close_execution` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 477–501 | Delegated execution authorization; declarations/fields: `authorize_reservation` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 502–522 | Exact binding registration and credential delivery; declarations/fields: `register` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 523–533 | Exact binding registration and credential delivery; declaration/member registration_execution; declarations/fields: `registration_execution` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 534–542 | Exact binding registration and credential delivery; declaration/member observe_execution_binding; declarations/fields: `observe_execution_binding` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 543–557 | Exact binding registration and credential delivery; declaration/member observe_terminal; declarations/fields: `observe_terminal` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 558–574 | Execution release and terminal observation; declarations/fields: `release_execution_lease` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 575–592 | Registration/binding authority attestation; declarations/fields: `registration` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 593–609 | Registration/binding authority attestation; declaration/member registration_authority; declarations/fields: `registration_authority` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 610–632 | Final credential return, uncertainty, retirement and reconcile; declarations/fields: `return_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 633–640 | Final credential return, uncertainty, retirement and reconcile; declaration/member uncertain; declarations/fields: `uncertain` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 641–656 | Final credential return, uncertainty, retirement and reconcile; declaration/member end; declarations/fields: `end` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 657–665 | Final credential return, uncertainty, retirement and reconcile; declaration/member end_lease; declarations/fields: `end_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 666–694 | Final credential return, uncertainty, retirement and reconcile; declaration/member finish_lease; declarations/fields: `finish_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 695–704 | Final credential return, uncertainty, retirement and reconcile; declaration/member reconcile_lease; declarations/fields: `reconcile_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 705–720 | Connection/grant revocation and live native retirement; declarations/fields: `revoke` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 721–739 | Connection/grant revocation and live native retirement; declaration/member revoke_grant; declarations/fields: `revoke_grant` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 740–751 | Periodic lease sweep and rejected-delivery retirement; declarations/fields: `reconcile` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 752–763 | Periodic lease sweep and rejected-delivery retirement; declaration/member sweep; declarations/fields: `sweep` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 764–776 | Periodic lease sweep and rejected-delivery retirement; declaration/member sweep_lease; declarations/fields: `sweep_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 777–786 | Periodic lease sweep and rejected-delivery retirement; declaration/member reject; declarations/fields: `reject` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 787–799 | Connection retirement and controller shutdown; declarations/fields: `retire_connection` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 800–813 | Connection retirement and controller shutdown; declaration/member close; declarations/fields: `close` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 814–821 | Error adaptation and private id primitives; declarations/fields: `join_errors` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 822–827 | Error adaptation and private id primitives; declaration/member new_id; declarations/fields: `new_id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 828–832 | Error adaptation and private id primitives; declaration/member fill_random; declarations/fields: `fill_random` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 833–837 | Error adaptation and private id primitives; declaration/member zeroize; declarations/fields: `zeroize` |

<a id="coverage-917dd4d29da0"></a>

## [rust/soda-identity/src/crypto.rs](../../../../../rust/soda-identity/src/crypto.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 1–83, 101–117 | Authenticated encryption and protected custody key handling; declarations/fields: `KEY_BINDING`, `GRANT_KEY_ERROR`, `GrantCipher`, `new`, `seal`, `open`, `nonce_size`, `fill_random`, `tests`, `rejects_short_keys`, `nist_vector`, `hex` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 84–100 | Source assertion of Native binding and private delivery; declarations/fields: `round_trip_with_binding` |

<a id="coverage-72e250c9050e"></a>

<a id="rustsoda-identitysrchttprs-1"></a>

## [rust/soda-identity/src/http.rs](../../../../../rust/soda-identity/src/http.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–13 | Private Unix broker listener admission and bounds; declarations/fields: `MAX_BODY`, `MAX_HEADER`, `HEADER_TIMEOUT`, `BODY_TIMEOUT`, `REQUEST_FIELDS`, `GRANT_FIELDS`, `ACQUIRE_FIELDS`, `BINDING_FIELDS`, `NESTED`, `Server`, `new`, `serve`, `handle_connection`, `Admission`, `HttpRequest`, `read_request` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 14 | Private Unix broker listener admission and bounds; declaration/member MAX_BODY; declarations/fields: `MAX_BODY` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 15 | Private Unix broker listener admission and bounds; declaration/member MAX_HEADER; declarations/fields: `MAX_HEADER` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 16 | Private Unix broker listener admission and bounds; declaration/member HEADER_TIMEOUT; declarations/fields: `HEADER_TIMEOUT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 17–18 | Private Unix broker listener admission and bounds; declaration/member BODY_TIMEOUT; declarations/fields: `BODY_TIMEOUT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 19–32 | Private Unix broker listener admission and bounds; declaration/member REQUEST_FIELDS; declarations/fields: `REQUEST_FIELDS` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 33–40 | Private Unix broker listener admission and bounds; declaration/member GRANT_FIELDS; declarations/fields: `GRANT_FIELDS` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 41–52 | Private Unix broker listener admission and bounds; declaration/member ACQUIRE_FIELDS; declarations/fields: `ACQUIRE_FIELDS` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 53–66 | Private Unix broker listener admission and bounds; declaration/member BINDING_FIELDS; declarations/fields: `BINDING_FIELDS` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 67–72 | Private Unix broker listener admission and bounds; declaration/member NESTED; declarations/fields: `NESTED` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 73 | Private Unix broker listener admission and bounds; declaration/member Server; declarations/fields: `Server` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 74 | Private Unix broker listener admission and bounds; declaration/member Server.controller; declarations/fields: `Server.controller` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 75 | Private Unix broker listener admission and bounds; declaration/member Server.runtime_allowed; declarations/fields: `Server.runtime_allowed` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 76 | Private Unix broker listener admission and bounds; declaration/member Server.shutdown; declarations/fields: `Server.shutdown` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 77–80 | Private Unix broker listener admission and bounds; declaration/member Server.inflight; declarations/fields: `Server.inflight` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 81–94 | Private Unix broker listener admission and bounds; declaration/member new; declarations/fields: `new` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 95–142 | Private Unix broker listener admission and bounds; declaration/member serve; declarations/fields: `serve` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 143–151 | Private Unix broker listener admission and bounds; declaration/member handle_connection; declarations/fields: `handle_connection` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 152–156 | Private Unix broker listener admission and bounds; declaration/member Admission; declarations/fields: `Admission` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 157 | Private Unix broker listener admission and bounds; declaration/member HttpRequest; declarations/fields: `HttpRequest` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 158 | Private Unix broker listener admission and bounds; declaration/member HttpRequest.method; declarations/fields: `HttpRequest.method` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 159 | Private Unix broker listener admission and bounds; declaration/member HttpRequest.path; declarations/fields: `HttpRequest.path` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 160 | Private Unix broker listener admission and bounds; declaration/member HttpRequest.query; declarations/fields: `HttpRequest.query` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 161–163 | Private Unix broker listener admission and bounds; declaration/member HttpRequest.origin; declarations/fields: `HttpRequest.origin` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 164–260 | Private Unix broker listener admission and bounds; declaration/member read_request; declarations/fields: `read_request` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 261–283 | Strict private path decoding; declarations/fields: `percent_decode` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 284–315 | Private request dispatch and runtime/admin admission; declarations/fields: `dispatch` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 316–322 | Administrative route dispatch envelope; declarations/fields: `route` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 323–328 | Connection/availability route |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 329–332 | Connection revoke route |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 333–344 | Enrollment start/read/cancel routes |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 345–359 | Grant read/create/revoke routes |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 360–362 | Lease read route |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 363–366 | Lease end route |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 367–375 | Runtime authority gate |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 376–381 | Runtime route envelope; declarations/fields: `route_runtime` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 382–387 | Execution acquisition route |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 388–397 | Native binding registration/private delivery route |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 398–419 | Reject/return/reconcile routes |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 420–422 | Execution fence observation |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 423–426 | Execution close route |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 427–430 | Unknown route denial and HTTP response envelope; declarations/fields: `success_response`, `error_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 431–445 | Unknown route denial and HTTP response envelope; declaration/member success_response; declarations/fields: `success_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 446–465 | Unknown route denial and HTTP response envelope; declaration/member error_response; declarations/fields: `error_response` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 466–469 | Private percent-path and HTTP output source vectors; declarations/fields: `tests` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 470–477 | Private percent-path and HTTP output source vectors; declaration/member percent_paths_decode; declarations/fields: `percent_paths_decode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 478–486 | Private percent-path and HTTP output source vectors; declaration/member error_bodies_match_go; declarations/fields: `error_bodies_match_go` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 487–494 | Private percent-path and HTTP output source vectors; declaration/member success_envelope_matches_go; declarations/fields: `success_envelope_matches_go` |

<a id="coverage-f29e4693296d"></a>

## [rust/soda-identity/src/lib.rs](../../../../../rust/soda-identity/src/lib.rs)

Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–4 | Private identity broker service composition |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 5 | Module wiring: control; Execution admission and lease fencing; declarations/fields: `control` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 6 | Module wiring: crypto; Encrypted credential custody; declarations/fields: `crypto` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 7 | Module wiring: http; Private IPC and service lifetime; declarations/fields: `http` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 8 | Module wiring: pg; Storage mechanics; declarations/fields: `pg` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 9 | Module wiring: runtime; Native binding and private delivery; declarations/fields: `runtime` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 10 | Module wiring: schema; Storage mechanics; declarations/fields: `schema` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 11 | Module wiring: store; Storage mechanics; declarations/fields: `store` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 12 | Module wiring: strict; Encoding and parsing; declarations/fields: `strict` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 13 | Module wiring: wire; Encoding and parsing; declarations/fields: `wire` |

<a id="coverage-749b2e8df0a8"></a>

<a id="rustsoda-identitysrcpgrs-1"></a>

## [rust/soda-identity/src/pg.rs](../../../../../rust/soda-identity/src/pg.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–16 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declarations/fields: `IO_TIMEOUT`, `Dsn`, `parse`, `percent_decode`, `Stream`, `read`, `write_all`, `Row`, `text`, `integer`, `bytea`, `Client`, `connect`, `send`, `receive`, `authenticate`, `authenticate_scram`, `query`, `simple`, `command_count`, `error_response`, `tests`, `dsn_shapes_match_go`, `command_tags_count_rows` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 17–18 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member IO_TIMEOUT; declarations/fields: `IO_TIMEOUT` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 19 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn; declarations/fields: `Dsn` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 20 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn.user; declarations/fields: `Dsn.user` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 21 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn.password; declarations/fields: `Dsn.password` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 22 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn.host; declarations/fields: `Dsn.host` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 23 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn.port; declarations/fields: `Dsn.port` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 24–27 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Dsn.database; declarations/fields: `Dsn.database` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 28–102 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member parse; declarations/fields: `parse` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 103–125 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member percent_decode; declarations/fields: `percent_decode` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 126–131 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Stream; declarations/fields: `Stream` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 132–138 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member read; declarations/fields: `read` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 139–146 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member write_all; declarations/fields: `write_all` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 147 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Row; declarations/fields: `Row` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 148 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Row.columns; declarations/fields: `Row.columns` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 149–152 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Row.fields; declarations/fields: `Row.fields` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 153–161 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member text; declarations/fields: `text` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 162–167 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member integer; declarations/fields: `integer` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 168–186 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member bytea; declarations/fields: `bytea` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 187 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Client; declarations/fields: `Client` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 188 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Client.stream; declarations/fields: `Client.stream` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 189–192 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member Client.buffer; declarations/fields: `Client.buffer` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 193–233 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member connect; declarations/fields: `connect` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 234–237 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member send; declarations/fields: `send` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 238–253 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member receive; declarations/fields: `receive` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 254–295 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member authenticate; declarations/fields: `authenticate` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 296–327 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member authenticate_scram; declarations/fields: `authenticate_scram` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 328–411 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member query; declarations/fields: `query` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 412–448 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member simple; declarations/fields: `simple` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 449–455 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member command_count; declarations/fields: `command_count` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 456–476 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member error_response; declarations/fields: `error_response` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 477–480 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member tests; declarations/fields: `tests` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 481–503 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member dsn_shapes_match_go; declarations/fields: `dsn_shapes_match_go` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 504–512 | PostgreSQL wire, SCRAM authentication, query and row mechanics; declaration/member command_tags_count_rows; declarations/fields: `command_tags_count_rows` |

<a id="coverage-e6620e3beafc"></a>

## [rust/soda-identity/src/runtime.rs](../../../../../rust/soda-identity/src/runtime.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–67 | Private host callback client and bounded transport; declarations/fields: `CALL_TIMEOUT`, `DEFAULT_LIMIT`, `FINISH_LIMIT`, `HostClient`, `new`, `call` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 68–75 | Exact native binding validation callback; declarations/fields: `validate` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 76–93 | Native stop and final credential capture callbacks; declarations/fields: `stop`, `finish` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 94–99 | Callback trait/response transport; declarations/fields: `default_limit` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 100–102 | Native validate trait binding; declarations/fields: `validate` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 103–110 | Native completion trait binding; declarations/fields: `stop`, `finish` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 111–240 | Bounded HTTP callback body/chunk parser; declarations/fields: `read_response`, `read_chunked`, `read_line` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 241–373, 383–390 | Host callback fixtures/assertions; declarations/fields: `tests`, `lease`, `stub`, `rand_suffix`, `delegates_both_kinds_to_host`, `finish_rejects_null_and_oversized_bodies` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 374–382 | Source assertion of Execution admission and lease fencing; declarations/fields: `refuses_unbound_lease_without_host_call` |

<a id="coverage-fac98c2013f2"></a>

<a id="rustsoda-identitysrcstrictrs-1"></a>

## [rust/soda-identity/src/strict.rs](../../../../../rust/soda-identity/src/strict.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–6, 368–390 | Bounded strict JSON admission and null/unknown-field compatibility; declarations/fields: `MAX_DOCUMENT`, `decode`, `remap_case`, `check_known_fields`, `Scanner`, `new`, `skip_ws`, `peek`, `eat`, `string`, `skip_string`, `skip_scalar`, `check_unique_keys`, `check_value`, `tests`, `Envelope`, `FIELDS`, `decode_envelope`, `limits_match_go` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 7–8 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member MAX_DOCUMENT; declarations/fields: `MAX_DOCUMENT` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 9–35 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member decode; declarations/fields: `decode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 36–64 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member remap_case; declarations/fields: `remap_case` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 65–86 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member check_known_fields; declarations/fields: `check_known_fields` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 87 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Scanner; declarations/fields: `Scanner` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 88 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Scanner.bytes; declarations/fields: `Scanner.bytes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 89–92 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Scanner.pos; declarations/fields: `Scanner.pos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 93–99 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member new; declarations/fields: `new` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 100–107 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member skip_ws; declarations/fields: `skip_ws` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 108–112 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member peek; declarations/fields: `peek` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 113–125 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member eat; declarations/fields: `eat` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 126–199 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member string; declarations/fields: `string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 200–203 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member skip_string; declarations/fields: `skip_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 204–217 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member skip_scalar; declarations/fields: `skip_scalar` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 218–248 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member check_unique_keys; declarations/fields: `check_unique_keys` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 249–315 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member check_value; declarations/fields: `check_value` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 316–320 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member tests; declarations/fields: `tests` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 321–322 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Envelope; declarations/fields: `Envelope` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 323–324 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Envelope.command_id; declarations/fields: `Envelope.command_id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 325–326 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Envelope.wire_type; declarations/fields: `Envelope.wire_type` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 327–329 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member Envelope.target; declarations/fields: `Envelope.target` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 330–331 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member FIELDS; declarations/fields: `FIELDS` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 332–336 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member decode_envelope; declarations/fields: `decode_envelope` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 337–367 | Source assertion of Encoding and parsing; declarations/fields: `strict_vectors_match_go` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 391–398 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member case_fold_matches_go_fallback; declarations/fields: `case_fold_matches_go_fallback` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 399–402 | Bounded strict JSON admission and null/unknown-field compatibility; declaration/member unicode_keys_compare_decoded; declarations/fields: `unicode_keys_compare_decoded` |

<a id="coverage-8cb18c62534a"></a>

<a id="rustsoda-identitysrcwirers-1"></a>

## [rust/soda-identity/src/wire.rs](../../../../../rust/soda-identity/src/wire.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–7 | Provider constants and exact Go timestamp/null/integer codecs; declarations/fields: `CODEX`, `MUSE`, `READY`, `REAUTH`, `REVOKED`, `FACTORY`, `TERMINAL`, `EXECUTION_PENDING`, `EXECUTION_LIVE`, `EXECUTION_TERMINAL`, `provider_valid`, `UnixTime`, `now`, `add_hours`, `as_system_time`, `serialize`, `deserialize`, `is_leap`, `days_in_month`, `days_from_civil`, `civil_from_days`, `parse_rfc3339_nano`, `format_rfc3339_nano`, `i64_string`, `i64_string_omitted`, `null_tolerant`, `string`, `boolean`, `integer`, `integer32`, `time`, `is_zero`, `base64_bytes`, `ALPHABET`, `encode`, `decode_value`, `decode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 8 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member CODEX; declarations/fields: `CODEX` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 9 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member MUSE; declarations/fields: `MUSE` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 10 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member READY; declarations/fields: `READY` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 11 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member REAUTH; declarations/fields: `REAUTH` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 12 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member REVOKED; declarations/fields: `REVOKED` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 13 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member FACTORY; declarations/fields: `FACTORY` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 14–15 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member TERMINAL; declarations/fields: `TERMINAL` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 16 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member EXECUTION_PENDING; declarations/fields: `EXECUTION_PENDING` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 17 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member EXECUTION_LIVE; declarations/fields: `EXECUTION_LIVE` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 18–19 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member EXECUTION_TERMINAL; declarations/fields: `EXECUTION_TERMINAL` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 20–26 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member provider_valid; declarations/fields: `provider_valid` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 27 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member UnixTime; declarations/fields: `UnixTime` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 28 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member UnixTime.sec; declarations/fields: `UnixTime.sec` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 29–32 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member UnixTime.nanos; declarations/fields: `UnixTime.nanos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 33–42 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member now; declarations/fields: `now` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 43–49 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member add_hours; declarations/fields: `add_hours` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 50–60 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member as_system_time; declarations/fields: `as_system_time` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 61–66, 226–229, 243–246, 345–348 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member serialize; declarations/fields: `serialize` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 67–73, 230–239, 247–254, 349–356 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member deserialize; declarations/fields: `deserialize` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 74–77 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member is_leap; declarations/fields: `is_leap` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 78–88 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member days_in_month; declarations/fields: `days_in_month` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 89–98 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member days_from_civil; declarations/fields: `days_from_civil` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 99–114 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member civil_from_days; declarations/fields: `civil_from_days` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 115–195 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member parse_rfc3339_nano; declarations/fields: `parse_rfc3339_nano` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 196–222 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member format_rfc3339_nano; declarations/fields: `format_rfc3339_nano` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 223–225 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member i64_string; declarations/fields: `i64_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 240–242 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member i64_string_omitted; declarations/fields: `i64_string_omitted` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 255–257 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member null_tolerant; declarations/fields: `null_tolerant` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 258–261 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member string; declarations/fields: `string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 262–265 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member boolean; declarations/fields: `boolean` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 266–269 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member integer; declarations/fields: `integer` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 270–273 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member integer32; declarations/fields: `integer32` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 274–278 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member time; declarations/fields: `time` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 279–283 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member is_zero; declarations/fields: `is_zero` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 284–286 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member base64_bytes; declarations/fields: `base64_bytes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 287–288 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member ALPHABET; declarations/fields: `ALPHABET` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 289–308 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member encode; declarations/fields: `encode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 309–319 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member decode_value; declarations/fields: `decode_value` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 320–344 | Provider constants and exact Go timestamp/null/integer codecs; declaration/member decode; declarations/fields: `decode` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 357–358 | Grant and delegated grant request contracts; declarations/fields: `Grant` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 359–360 | Grant and delegated grant request contracts; declaration/member Grant.id; declarations/fields: `Grant.id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 361–362 | Grant and delegated grant request contracts; declaration/member Grant.connection_id; declarations/fields: `Grant.connection_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 363–364 | Grant and delegated grant request contracts; declaration/member Grant.user_id; declarations/fields: `Grant.user_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 365–366 | Grant and delegated grant request contracts; declaration/member Grant.project_id; declarations/fields: `Grant.project_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 367–368 | Grant and delegated grant request contracts; declaration/member Grant.revision; declarations/fields: `Grant.revision` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 369–372 | Grant and delegated grant request contracts; declaration/member Grant.revoked; declarations/fields: `Grant.revoked` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 373–374 | Grant and delegated grant request contracts; declaration/member GrantRequest; declarations/fields: `GrantRequest` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 375–376 | Grant and delegated grant request contracts; declaration/member GrantRequest.connection_id; declarations/fields: `GrantRequest.connection_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 377–378 | Grant and delegated grant request contracts; declaration/member GrantRequest.user_id; declarations/fields: `GrantRequest.user_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 379–380 | Grant and delegated grant request contracts; declaration/member GrantRequest.project_id; declarations/fields: `GrantRequest.project_id` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 381–382 | Grant and delegated grant request contracts; declaration/member GrantRequest.confirm_subscription; declarations/fields: `GrantRequest.confirm_subscription` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 383–386 | Grant and delegated grant request contracts; declaration/member GrantRequest.confirm_credential_exposure; declarations/fields: `GrantRequest.confirm_credential_exposure` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 387–388 | Execution acquisition contract; declarations/fields: `AcquireRequest` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 389–390 | Execution acquisition contract; declaration/member AcquireRequest.repository_id; declarations/fields: `AcquireRequest.repository_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 391–392 | Execution acquisition contract; declaration/member AcquireRequest.provider_id; declarations/fields: `AcquireRequest.provider_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 393–394 | Execution acquisition contract; declaration/member AcquireRequest.execution_id; declarations/fields: `AcquireRequest.execution_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 395–396 | Execution acquisition contract; declaration/member AcquireRequest.actor_id; declarations/fields: `AcquireRequest.actor_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 397–398 | Execution acquisition contract; declaration/member AcquireRequest.connection_id; declarations/fields: `AcquireRequest.connection_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 399–400 | Execution acquisition contract; declaration/member AcquireRequest.project_id; declarations/fields: `AcquireRequest.project_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 401–402 | Execution acquisition contract; declaration/member AcquireRequest.kind; declarations/fields: `AcquireRequest.kind` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 403–408 | Execution acquisition contract; declaration/member AcquireRequest.deadline; declarations/fields: `AcquireRequest.deadline` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 409–412 | Execution acquisition contract; declaration/member AcquireRequest.role; declarations/fields: `AcquireRequest.role` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 413–418 | Exact native binding contract; declarations/fields: `Binding` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 419–424 | Exact native binding contract; declaration/member Binding.child_id; declarations/fields: `Binding.child_id` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 425–430 | Exact native binding contract; declaration/member Binding.uid; declarations/fields: `Binding.uid` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 431–436 | Exact native binding contract; declaration/member Binding.gid; declarations/fields: `Binding.gid` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 437–442 | Exact native binding contract; declaration/member Binding.scope; declarations/fields: `Binding.scope` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 443–448 | Exact native binding contract; declaration/member Binding.credential_root; declarations/fields: `Binding.credential_root` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 449–450 | Exact native binding contract; declaration/member Binding.invocation_id; declarations/fields: `Binding.invocation_id` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 451–452 | Exact native binding contract; declaration/member Binding.kind; declarations/fields: `Binding.kind` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 453–454 | Exact native binding contract; declaration/member Binding.id; declarations/fields: `Binding.id` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 455–456 | Exact native binding contract; declaration/member Binding.project; declarations/fields: `Binding.project` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 457–458 | Exact native binding contract; declaration/member Binding.login; declarations/fields: `Binding.login` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 459–461 | Exact native binding contract; declaration/member Binding.generation; declarations/fields: `Binding.generation` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 462–466 | Exact native binding contract; declaration/member is_zero_i32; declarations/fields: `is_zero_i32` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 467–468 | Lease contract; declarations/fields: `Lease` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 469–470 | Lease contract; declaration/member Lease.repository_id; declarations/fields: `Lease.repository_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 471–472 | Lease contract; declaration/member Lease.provider_id; declarations/fields: `Lease.provider_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 473–474 | Lease contract; declaration/member Lease.id; declarations/fields: `Lease.id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 475–476 | Lease contract; declaration/member Lease.connection_id; declarations/fields: `Lease.connection_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 477–478 | Lease contract; declaration/member Lease.generation; declarations/fields: `Lease.generation` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 479–480 | Lease contract; declaration/member Lease.actor_id; declarations/fields: `Lease.actor_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 481–482 | Lease contract; declaration/member Lease.project_id; declarations/fields: `Lease.project_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 483–484 | Lease contract; declaration/member Lease.execution_id; declarations/fields: `Lease.execution_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 485–490 | Lease contract; declaration/member Lease.kind; declarations/fields: `Lease.kind` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 491–492 | Lease contract; declaration/member Lease.role; declarations/fields: `Lease.role` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 493–498 | Lease contract; declaration/member Lease.deadline; declarations/fields: `Lease.deadline` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 499–504 | Lease contract; declaration/member Lease.grant_id; declarations/fields: `Lease.grant_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 505–506 | Lease contract; declaration/member Lease.grant_revision; declarations/fields: `Lease.grant_revision` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 507–510 | Lease contract; declaration/member Lease.binding; declarations/fields: `Lease.binding` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 511–512 | Persistent execution fence contract; declarations/fields: `Execution` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 513–514 | Persistent execution fence contract; declaration/member Execution.binding; declarations/fields: `Execution.binding` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 515–516 | Persistent execution fence contract; declaration/member Execution.kind; declarations/fields: `Execution.kind` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 517–518 | Persistent execution fence contract; declaration/member Execution.execution_id; declarations/fields: `Execution.execution_id` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 519–520 | Persistent execution fence contract; declaration/member Execution.digest; declarations/fields: `Execution.digest` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 521–526 | Persistent execution fence contract; declaration/member Execution.state; declarations/fields: `Execution.state` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 527–530 | Persistent execution fence contract; declaration/member Execution.lease_id; declarations/fields: `Execution.lease_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 531–532 | Credential-free identity event contract; declarations/fields: `Event` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 533–534 | Credential-free identity event contract; declaration/member Event.id; declarations/fields: `Event.id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 535–536 | Credential-free identity event contract; declaration/member Event.time; declarations/fields: `Event.time` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 537–538 | Credential-free identity event contract; declaration/member Event.action; declarations/fields: `Event.action` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 539–540 | Credential-free identity event contract; declaration/member Event.owner_id; declarations/fields: `Event.owner_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 541–542 | Credential-free identity event contract; declaration/member Event.actor_id; declarations/fields: `Event.actor_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 543–548 | Credential-free identity event contract; declaration/member Event.connection_id; declarations/fields: `Event.connection_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 549–554 | Credential-free identity event contract; declaration/member Event.lease_id; declarations/fields: `Event.lease_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 555–560 | Credential-free identity event contract; declaration/member Event.project_id; declarations/fields: `Event.project_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 561–566 | Credential-free identity event contract; declaration/member Event.grant_id; declarations/fields: `Event.grant_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 567–572 | Credential-free identity event contract; declaration/member Event.execution_id; declarations/fields: `Event.execution_id` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 573–574 | Credential-free identity event contract; declaration/member Event.kind; declarations/fields: `Event.kind` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 575–579 | Credential-free identity event contract; declaration/member Event.generation; declarations/fields: `Event.generation` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 580–585 | Private administrative/runtime request envelope; declarations/fields: `Request` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 586–587 | Private administrative/runtime request envelope; declaration/member Request.provider_id; declarations/fields: `Request.provider_id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 588–593 | Private administrative/runtime request envelope; declaration/member Request.owner_id; declarations/fields: `Request.owner_id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 594–599 | Private administrative/runtime request envelope; declaration/member Request.id; declarations/fields: `Request.id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 600–605 | Private administrative/runtime request envelope; declaration/member Request.label; declarations/fields: `Request.label` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 606–611 | Private administrative/runtime request envelope; declaration/member Request.project_id; declarations/fields: `Request.project_id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 612–617 | Private administrative/runtime request envelope; declaration/member Request.kind; declarations/fields: `Request.kind` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 618–619 | Private administrative/runtime request envelope; declaration/member Request.execution_id; declarations/fields: `Request.execution_id` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 620–621 | Private administrative/runtime request envelope; declaration/member Request.grant; declarations/fields: `Request.grant` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 622–623 | Private administrative/runtime request envelope; declaration/member Request.acquire; declarations/fields: `Request.acquire` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 624–629 | Private administrative/runtime request envelope; declaration/member Request.binding; declarations/fields: `Request.binding` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 630–635 | Private administrative/runtime request envelope; declaration/member Request.credential; declarations/fields: `Request.credential` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 636–639 | Private administrative/runtime request envelope; declaration/member base64_bytes_option; declarations/fields: `base64_bytes_option` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 640–646 | Private administrative/runtime request envelope; declaration/member serialize; declarations/fields: `serialize` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 647–656 | Private administrative/runtime request envelope; declaration/member deserialize; declarations/fields: `deserialize` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 657 | Private credential delivery contract; declarations/fields: `DeliveryWire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 658–659 | Private credential delivery contract; declaration/member DeliveryWire.lease; declarations/fields: `DeliveryWire.lease` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 660–663 | Private credential delivery contract; declaration/member DeliveryWire.credential; declarations/fields: `DeliveryWire.credential` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 664–673 | Private broker error forms; declarations/fields: `ErrorKind` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 674 | Private broker error forms; declaration/member Error; declarations/fields: `Error` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 675 | Private broker error forms; declaration/member Error.kind; declarations/fields: `Error.kind` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 676–679 | Private broker error forms; declaration/member Error.message; declarations/fields: `Error.message` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 680–685 | Private broker error forms; declaration/member denied; declarations/fields: `denied` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 686–691 | Private broker error forms; declaration/member busy; declarations/fields: `busy` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 692–697 | Private broker error forms; declaration/member stale; declarations/fields: `stale` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 698–703 | Private broker error forms; declaration/member uncertain; declarations/fields: `uncertain` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 704–709 | Private broker error forms; declaration/member not_found; declarations/fields: `not_found` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 710–715 | Private broker error forms; declaration/member internal; declarations/fields: `internal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 716–718 | Private broker error forms; declaration/member kind; declarations/fields: `kind` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 719–721 | Private broker error forms; declaration/member is_denied; declarations/fields: `is_denied` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 722–727 | Private broker error forms; declaration/member is_not_found; declarations/fields: `is_not_found` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 728–735 | Private broker error forms; declaration/member fmt; declarations/fields: `fmt` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 736–746 | Private broker error forms; declaration/member from; declarations/fields: `from` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 747 | Grant delegation validation; declarations/fields: `validate` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 748–760 | Grant delegation validation; declaration/member validate; declarations/fields: `validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 761 | Acquisition admission validation; declarations/fields: `validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 762–776 | Acquisition admission validation; declaration/member validate; declarations/fields: `validate` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 777 | Native binding validation; declarations/fields: `validate` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 778–791 | Native binding validation; declaration/member validate; declarations/fields: `validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 792 | Exact execution fence validation; declarations/fields: `validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 793–818 | Exact execution fence validation; declaration/member validate; declarations/fields: `validate` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 819–834 | Digest/phase encoding helpers; declarations/fields: `acquisition_digest` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 835–838, 904–929 | Wire/validation source vectors; declarations/fields: `tests`, `base64_matches_go_byte_form` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 839–858 | Source assertion of Encoding and parsing; declarations/fields: `go_time_vectors_round_trip` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 859–876 | Source assertion of Encoding and parsing; declaration/member timestamps_reject_malformed_input; declarations/fields: `timestamps_reject_malformed_input` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 877–903 | Source assertion of Execution admission and lease fencing; declarations/fields: `lease_wire_shape_matches_go` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 930–950 | Wire/validation source vectors; declaration/member null_scalars_match_go_noop; declarations/fields: `null_scalars_match_go_noop` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 951–958 | Wire/validation source vectors; declaration/member provider_ids_match_go; declarations/fields: `provider_ids_match_go` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 959–981 | Wire/validation source vectors; declaration/member digest_matches_go_acquisition; declarations/fields: `digest_matches_go_acquisition` |

<a id="coverage-2ad53709a47b"></a>

<a id="rustsoda-identitytestsbrokerrs-1"></a>

## [rust/soda-identity/tests/broker.rs](../../../../../rust/soda-identity/tests/broker.rs)

Source-only integration suite gated on SODA_PG_HOST/PORT/SUPER_PASSWORD_FILE; tests skip without fixture. No execution or native provider success claimed. Compound lifecycle tests reference several distinct broker responsibilities. Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1–10, 149–173, 454 | Ephemeral PostgreSQL broker lifecycle and Unix HTTP source integration tests; declarations/fields: `super_dsn`, `Ephemeral`, `create`, `store`, `drop`, `hex`, `fixture_key`, `subscription`, `connection`, `StubSession`, `snapshot`, `finish`, `close`, `StubProvider`, `start`, `StubRuntime`, `validate`, `stop`, `stub_provider`, `controller`, `binding`, `Guard` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 11–27 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declarations/fields: `super_dsn` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 28 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member Ephemeral; declarations/fields: `Ephemeral` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 29 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member Ephemeral.dsn; declarations/fields: `Ephemeral.dsn` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 30 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member Ephemeral.super_dsn; declarations/fields: `Ephemeral.super_dsn` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 31–34 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member Ephemeral.name; declarations/fields: `Ephemeral.name` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 35–54 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member create; declarations/fields: `create` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 55–60 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member store; declarations/fields: `store` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 61–69 | Explicit ephemeral PostgreSQL developer test fixture; skip without configured fixture, no execution claim; declaration/member drop; declarations/fields: `drop` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 70–73 | Synthetic encrypted-custody credential/connection fixtures; declarations/fields: `hex` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 74–77 | Synthetic encrypted-custody credential/connection fixtures; declaration/member fixture_key; declarations/fields: `fixture_key` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 78–81 | Synthetic encrypted-custody credential/connection fixtures; declaration/member subscription; declarations/fields: `subscription` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 82–94 | Synthetic encrypted-custody credential/connection fixtures; declaration/member connection; declarations/fields: `connection` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 95 | Stub enrollment session/provider fixtures; declarations/fields: `StubSession` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 96 | Stub enrollment session/provider fixtures; declaration/member StubSession.snapshot; declarations/fields: `StubSession.snapshot` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 97 | Stub enrollment session/provider fixtures; declaration/member StubSession.connection; declarations/fields: `StubSession.connection` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 98–101 | Stub enrollment session/provider fixtures; declaration/member StubSession.credential; declarations/fields: `StubSession.credential` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 102–104 | Stub enrollment session/provider fixtures; declaration/member snapshot; declarations/fields: `snapshot` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 105–107 | Stub enrollment session/provider fixtures; declaration/member finish; declarations/fields: `finish` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 108–112 | Stub enrollment session/provider fixtures; declaration/member close; declarations/fields: `close` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 113 | Stub enrollment session/provider fixtures; declaration/member StubProvider; declarations/fields: `StubProvider` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 114 | Stub enrollment session/provider fixtures; declaration/member StubProvider.enrollment; declarations/fields: `StubProvider.enrollment` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 115 | Stub enrollment session/provider fixtures; declaration/member StubProvider.connection; declarations/fields: `StubProvider.connection` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 116–119 | Stub enrollment session/provider fixtures; declaration/member StubProvider.credential; declarations/fields: `StubProvider.credential` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 120–128 | Stub enrollment session/provider fixtures; declaration/member start; declarations/fields: `start` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 129 | Stub exact native binding validation callback; declarations/fields: `StubRuntime` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 130 | Stub exact native binding validation callback; declaration/member StubRuntime.credential; declarations/fields: `StubRuntime.credential` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 131–134 | Stub exact native binding validation callback; declaration/member StubRuntime.calls; declarations/fields: `StubRuntime.calls` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 135–138 | Stub exact native binding validation callback; declaration/member validate; declarations/fields: `validate` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 139–142 | Stub native stop/final credential capture callbacks; declarations/fields: `stop` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 143–148 | Stub native stop/final credential capture callbacks; declaration/member finish; declarations/fields: `finish` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 174–187 | Ephemeral PostgreSQL broker lifecycle and Unix HTTP source integration tests; declaration/member controller; declarations/fields: `controller` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 188–204 | Exact native binding fixture; declarations/fields: `binding` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 205–217, 237–238 | Source assertion of Encrypted credential custody; declarations/fields: `store_round_trip` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 218–236 | Source assertions of owned/delegated available metadata, grant and email stripping; declarations/fields: `store.available`, `store.save_grant`, `store.revoke_grant` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 239–254, 304–305 | Source assertion of Enrollment and owner consent; declarations/fields: `enrollment_to_lease_lifecycle` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 255–288 | Exact execution acquisition and same-input lease replay assertions; declarations/fields: `broker.acquire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 289–291 | Native registration/private credential delivery assertions through StubRuntime; declarations/fields: `broker.register` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 292–295 | Owner lease-listing binding stripping assertion; declarations/fields: `broker.leases` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 296–303 | Codex final return credential generation and terminal execution assertions; declarations/fields: `broker.return_lease`, `broker.get_execution` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 306–310, 357–358, 388–389, 420–421 | Source assertion of Completion, revocation and reconciliation; declarations/fields: `muse_lease_returns_by_forget`, `close_execution_fences_late_registration`, `revoke_retires_live_leases` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 311–326 | Synthetic Muse connection/provider fixture; enrollment explicitly excluded; declarations/fields: `Controller::new` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 327–343 | Muse exact execution acquisition fixture/assertion; declarations/fields: `broker.acquire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 344–346 | Muse registration/private credential assertion through StubRuntime; declarations/fields: `broker.register` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 347–356 | Muse final return forgets lease, keeps immutable connection generation and closes execution; declarations/fields: `broker.return_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 359–364 | Source assertion of Completion, revocation and reconciliation; declaration/member close_execution_fences_late_registration; declarations/fields: `close_execution_fences_late_registration` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 365–367 | Stub enrollment fixture used by terminal fence regression; declarations/fields: `broker.start_enrollment`, `broker.enrollment` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 368–383 | Exact execution acquisition before close regression; declarations/fields: `AcquireRequest`, `broker.acquire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 384 | Close exact execution in terminal-fence regression; declarations/fields: `broker.close_execution` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 385 | Registration refuses terminal execution assertion; declarations/fields: `broker.register` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 386–387 | Acquisition refuses terminal execution assertion; declarations/fields: `broker.acquire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 390–395 | Source assertion of Completion, revocation and reconciliation; declaration/member revoke_retires_live_leases; declarations/fields: `revoke_retires_live_leases` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 396–398 | Stub enrollment fixture used by revoke regression; declarations/fields: `broker.start_enrollment`, `broker.enrollment` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 399–415 | Acquire live lease before revoke regression; declarations/fields: `broker.acquire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 416–419 | Connection revoke retires live leases assertion through StubRuntime; declarations/fields: `broker.revoke` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 422–453 | Source assertion of Private IPC and service lifetime; declarations/fields: `http_admission_matches_go` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 455–457 | Ephemeral PostgreSQL broker lifecycle and Unix HTTP source integration tests; declaration/member Guard.shutdown; declarations/fields: `Guard.shutdown` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 458–546 | Ephemeral PostgreSQL broker lifecycle and Unix HTTP source integration tests; declaration/member drop; declarations/fields: `drop` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 547–596 | Ephemeral PostgreSQL broker lifecycle and Unix HTTP source integration tests; declaration/member dead_listener_fails_fast; declarations/fields: `dead_listener_fails_fast` |

