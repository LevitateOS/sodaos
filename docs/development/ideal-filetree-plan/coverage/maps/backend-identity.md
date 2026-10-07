# Backend identity

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-388b001b4805"></a>

## [internal/identity/client/broker_compat_test.go](../../../../../internal/identity/client/broker_compat_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–136; compatCredential; repoRoot; buildBroker; tmpfsRoot; fakeCodex; stubHost; waitForSocket | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Current declaration duty: compatCredential; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 137–283; TestRustBrokerCompatibility | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current declaration duty: TestRustBrokerCompatibility — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ccdfbbec350f"></a>

## [internal/identity/client/client.go](../../../../../internal/identity/client/client.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–46; whole file; Client; New; Client.call | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 47–85; decodeError, decodeResponse | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Bounded broker response decoding mechanics; declarations/fields: `decodeError`, `decodeResponse` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 86–89, 130–133, 140–147, 160–167, 174–176; Client.ReconcileLease, Client.RevokeGrant, Client.EndLease, Client.Revoke, Client.Return, Client.Reject, Client.CloseExecution | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Credential retirement and reconciliation protocol operation; declarations/fields: `Client.ReconcileLease`, `Client.RevokeGrant`, `Client.EndLease`, `Client.Revoke`, `Client.Return`, `Client.Reject`, `Client.CloseExecution` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 90–101, 118–129; Client.Connections, Client.Available, Client.Grants, Client.CreateGrant | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Owned connection and delegation metadata protocol operation; declarations/fields: `Client.Connections`, `Client.Available`, `Client.Grants`, `Client.CreateGrant` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 102–117; Client.StartEnrollment, Client.Enrollment, Client.CancelEnrollment | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Provider enrollment protocol operation; declarations/fields: `Client.StartEnrollment`, `Client.Enrollment`, `Client.CancelEnrollment` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 134–139, 148–153, 168–173; Client.Leases, Client.Acquire, Client.GetExecution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Lease admission or execution fence protocol operation; declarations/fields: `Client.Leases`, `Client.Acquire`, `Client.GetExecution` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 154–159; Client.Register | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Native execution binding and private delivery protocol operation; declarations/fields: `Client.Register` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-d898078466ee"></a>

## [internal/identity/client/client_test.go](../../../../../internal/identity/client/client_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 36–45; whole file; redirectTransport; redirectTransport.RoundTrip | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Fixture/protocol support redirectTransport; declarations/fields: `redirectTransport`; Fixture/protocol support redirectTransport.RoundTrip; declarations/fields: `redirectTransport.RoundTrip` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 12–35; TestMetadataArraysAndPrivateDeliveryDecode | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Broker decoder contract assertions; declarations/fields: `TestMetadataArraysAndPrivateDeliveryDecode` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-111a23a134a3"></a>

## [internal/identity/enrollment.go](../../../../../internal/identity/enrollment.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 11–20; whole file; EnrollmentSession; Provider | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Record, DTO or interface contract EnrollmentSession for Enrollment and owner consent; declarations/fields: `EnrollmentSession`; Record, DTO or interface contract Provider for Enrollment and owner consent; declarations/fields: `Provider` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10; EnrollmentSession.Finish | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Private enrolled credential result delivered for encrypted custody; declarations/fields: `EnrollmentSession.Finish` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 21, 23–25; Runtime | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Native registered execution retirement confirmation contract; declarations/fields: `Runtime` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22; Runtime.Validate | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Attested lease native binding validation; declarations/fields: `Runtime.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-9d4bc5506b5c"></a>

## [internal/identity/event.go](../../../../../internal/identity/event.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19; file scaffold; Event | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current scaffold duty: file scaffold; Current declaration duty: Event — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4dd10addd793"></a>

## [internal/identity/launch.go](../../../../../internal/identity/launch.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–93, 108–156; whole file; MuseLaunchSocket; (declaration group); LaunchRequest.Validate; LaunchRequest.registrationValid; launchAbsolutePath; launchText; launchArgumentsValid; museProviderArguments; museAuthOverride; musePositional; museValueFlag; LaunchRequest.configPathsValid; LaunchRequest.launchSizesValid | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 14 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 94–107; MuseArguments | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | Muse-specific provider launch arguments; declarations/fields: `MuseArguments` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-8ef706c3cc47"></a>

## [internal/identity/launch_test.go](../../../../../internal/identity/launch_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; whole file; TestLaunchRequestHasNoAuthority | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Assertions TestLaunchRequestHasNoAuthority: invalid request admitted: %#v; declarations/fields: `TestLaunchRequestHasNoAuthority` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 19–30; TestMuseNativeCommandProviderPlacement | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | Muse argument-placement assertions; declarations/fields: `TestMuseNativeCommandProviderPlacement` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-d0b7ebb10315"></a>

## [internal/identity/selection.go](../../../../../internal/identity/selection.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–29; file scaffold; SelectMuseConnection; selectConnection | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current scaffold duty: file scaffold; Current declaration duty: SelectMuseConnection; Current declaration duty: selectConnection — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ca36592c4f4"></a>

## [internal/identity/selection_test.go](../../../../../internal/identity/selection_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; file scaffold; TestSelectMuseConnectionRequiresCurrentAuthorizedProvider | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestSelectMuseConnectionRequiresCurrentAuthorizedProvider — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-78f11a18660c"></a>

## [internal/identity/terminal.go](../../../../../internal/identity/terminal.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; file scaffold; TerminalStart | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Current scaffold duty: file scaffold; Current declaration duty: TerminalStart — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b5248fddcfbc"></a>

## [internal/identity/transport.go](../../../../../internal/identity/transport.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 16–17; whole file; Request | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Record, DTO or interface contract Request for Private IPC and service lifetime; declarations/fields: `Request` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 5–8; Request.ProviderID, Request.OwnerID, Request.ID, Request.Label | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Provider enrollment/owner consent request identity; declarations/fields: `Request.ProviderID`, `Request.OwnerID`, `Request.ID`, `Request.Label` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 9, 12; Request.ProjectID; Request.Grant | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Delegated connection availability Project scope; declarations/fields: `Request.ProjectID`; Explicit delegated grant request; declarations/fields: `Request.Grant` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10–11, 13; Request.Kind, Request.ExecutionID; Request.Acquire | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Execution-fence protocol identity; declarations/fields: `Request.Kind`, `Request.ExecutionID`; Lease acquisition request; declarations/fields: `Request.Acquire` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 14–15, 18–22; Request.Binding, Request.Credential; DeliveryWire | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Attested native registration/return credential payload; declarations/fields: `Request.Binding`, `Request.Credential`; Private runtime delivery only, never admin/browser response; declarations/fields: `DeliveryWire` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-ac0839ea542a"></a>

## [internal/identity/types.go](../../../../../internal/identity/types.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 28–37, 42–43, 72–85, 100–114, 116–118, 141–147, 165–215; whole file; (declaration group); Factory, Terminal; AcquireRequest; Lease; AcquireRequest.Validate; Execution; Execution.Validate; AcquisitionDigest | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22–24, 124–133, 216; Codex, Muse; Enrollment, ProviderValid | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Supported enrollment provider identifiers; declarations/fields: `Codex`, `Muse`; Provider enrollment metadata and supported provider identities; declarations/fields: `Enrollment`, `ProviderValid` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 25–27, 44–71, 134–140; Ready, Reauth, Revoked; Connection, Grant, GrantRequest, GrantRequest.Validate | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Connection availability lifecycle metadata; declarations/fields: `Ready`, `Reauth`, `Revoked`; Connection and delegated grant metadata; declarations/fields: `Connection`, `Grant`, `GrantRequest`, `GrantRequest.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 38–41; ExecutionTerminal | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Terminal execution fence prevents reacquisition after retirement; declarations/fields: `ExecutionTerminal` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 86–99, 115, 119–123, 148–157; Binding, Delivery, Binding.Validate; Lease.Binding | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Attested native binding and private delivery DTOs; declarations/fields: `Binding`, `Delivery`, `Binding.Validate`; Lease references separately attested native binding; declarations/fields: `Lease.Binding` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 158–164; CredentialValid | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Bounded private credential custody input shape; declarations/fields: `CredentialValid` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-6bfe7af613fe"></a>

## [internal/identity/types_test.go](../../../../../internal/identity/types_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 24–38; whole file; TestLeaseRequiresBoundedDeadlineAndExecution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Assertions TestLeaseRequiresBoundedDeadlineAndExecution: expired request admitted; declarations/fields: `TestLeaseRequiresBoundedDeadlineAndExecution` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 8–23; TestGrantRequiresBothConfirmations | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Delegation metadata validation assertions; declarations/fields: `TestGrantRequiresBothConfirmations` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
