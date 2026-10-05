# Backend identity

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-388b001b4805"></a>

## [internal/identity/client/broker_compat_test.go](../../../../../internal/identity/client/broker_compat_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Seeds connection directly, runs fake Codex and stub host if executed; exercises protocol compatibility rather than native enrollment/real-provider authorization.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–27 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 28–29 | Fixture/protocol support compatCredential; declarations/fields: `compatCredential` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 30–49 | Fixture/protocol support repoRoot: repository root not found; declarations/fields: `repoRoot` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 50–65 | Fixture/protocol support buildBroker: build soda-identity: %v\n%s; declarations/fields: `buildBroker` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 66–75 | Fixture/protocol support tmpfsRoot; declarations/fields: `tmpfsRoot` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 76–88 | Fixture/protocol support fakeCodex; declarations/fields: `fakeCodex` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 89–124 | Fixture/protocol support stubHost; declarations/fields: `stubHost` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 125–136 | Fixture/protocol support waitForSocket: broker socket %s never appeared; declarations/fields: `waitForSocket` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 137–205 | Opt-in disposable PostgreSQL/native broker/fake provider startup fixture; no enrollment journey; declarations/fields: `TestRustBrokerCompatibility` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 206–214 | Seed one encrypted provider connection directly in shared schema; declarations/fields: `IdentitySaveConnection` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 215–234 | Go/native broker delegated availability/grant compatibility assertions; declarations/fields: `Connections`, `Available`, `CreateGrant` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 235–247 | Exact lease acquisition/execution fence compatibility assertions; declarations/fields: `Acquire`, `GetExecution` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 248–257 | Attested native registration/delivery; public lease metadata strips binding; declarations/fields: `Register`, `Leases` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 258–278 | Credential return/execution closure/grant and connection retirement compatibility assertions; declarations/fields: `Return`, `CloseExecution`, `RevokeGrant`, `Revoke` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 279–283 | Native owner admission rejects invalid actor metadata read; declarations/fields: `Connections` |

<a id="coverage-ccdfbbec350f"></a>

## [internal/identity/client/client.go](../../../../../internal/identity/client/client.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 18–19 | Record, DTO or interface contract Client for Private IPC and service lifetime; declarations/fields: `Client` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 20–25 | New — Private IPC and service lifetime; declarations/fields: `New` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 26–46 | Client.call — Private IPC and service lifetime; declarations/fields: `Client.call` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 47–85 | Bounded broker response decoding mechanics; declarations/fields: `decodeError`, `decodeResponse` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 86–89, 130–133, 140–147, 160–167, 174–176 | Credential retirement and reconciliation protocol operation; declarations/fields: `Client.ReconcileLease`, `Client.RevokeGrant`, `Client.EndLease`, `Client.Revoke`, `Client.Return`, `Client.Reject`, `Client.CloseExecution` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 90–101, 118–129 | Owned connection and delegation metadata protocol operation; declarations/fields: `Client.Connections`, `Client.Available`, `Client.Grants`, `Client.CreateGrant` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 102–117 | Provider enrollment protocol operation; declarations/fields: `Client.StartEnrollment`, `Client.Enrollment`, `Client.CancelEnrollment` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 134–139, 148–153, 168–173 | Lease admission or execution fence protocol operation; declarations/fields: `Client.Leases`, `Client.Acquire`, `Client.GetExecution` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 154–159 | Native execution binding and private delivery protocol operation; declarations/fields: `Client.Register` |

<a id="coverage-d898078466ee"></a>

## [internal/identity/client/client_test.go](../../../../../internal/identity/client/client_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 12–35 | Broker decoder contract assertions; declarations/fields: `TestMetadataArraysAndPrivateDeliveryDecode` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 36–40 | Fixture/protocol support redirectTransport; declarations/fields: `redirectTransport` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 41–45 | Fixture/protocol support redirectTransport.RoundTrip; declarations/fields: `redirectTransport.RoundTrip` |

<a id="coverage-111a23a134a3"></a>

## [internal/identity/enrollment.go](../../../../../internal/identity/enrollment.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 8–9, 11–15 | Record, DTO or interface contract EnrollmentSession for Enrollment and owner consent; declarations/fields: `EnrollmentSession` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 10 | Private enrolled credential result delivered for encrypted custody; declarations/fields: `EnrollmentSession.Finish` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 16–20 | Record, DTO or interface contract Provider for Enrollment and owner consent; declarations/fields: `Provider` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 21, 23–25 | Native registered execution retirement confirmation contract; declarations/fields: `Runtime` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 22 | Attested lease native binding validation; declarations/fields: `Runtime.Validate` |

<a id="coverage-4dd10addd793"></a>

## [internal/identity/launch.go](../../../../../internal/identity/launch.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 8–9 | Declared identifiers/bounds MuseLaunchSocket for Provider execution integration; declarations/fields: `MuseLaunchSocket` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 10–31, 81–93 | Record, DTO or interface contract (declaration group) for Provider execution integration; declarations/fields: `(declaration group)` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 32–47 | LaunchRequest.Validate — Provider execution integration; declarations/fields: `LaunchRequest.Validate` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 48–54 | LaunchRequest.registrationValid — Provider execution integration; declarations/fields: `LaunchRequest.registrationValid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 55–61 | launchAbsolutePath — Provider execution integration; declarations/fields: `launchAbsolutePath` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 62–65 | launchText — Provider execution integration; declarations/fields: `launchText` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 66–80 | launchArgumentsValid — Provider execution integration; declarations/fields: `launchArgumentsValid` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 94–107 | Muse-specific provider launch arguments; declarations/fields: `MuseArguments` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 108–120 | museProviderArguments — Provider execution integration; declarations/fields: `museProviderArguments` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 121–125 | museAuthOverride — Provider execution integration; declarations/fields: `museAuthOverride` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 126–141 | musePositional — Provider execution integration; declarations/fields: `musePositional` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 142–149 | museValueFlag — Provider execution integration; declarations/fields: `museValueFlag` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 150–153 | LaunchRequest.configPathsValid — Provider execution integration; declarations/fields: `LaunchRequest.configPathsValid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 154–156 | LaunchRequest.launchSizesValid — Provider execution integration; declarations/fields: `LaunchRequest.launchSizesValid` |

<a id="coverage-8ef706c3cc47"></a>

## [internal/identity/launch_test.go](../../../../../internal/identity/launch_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 8–18 | Assertions TestLaunchRequestHasNoAuthority: invalid request admitted: %#v; declarations/fields: `TestLaunchRequestHasNoAuthority` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 19–30 | Muse argument-placement assertions; declarations/fields: `TestMuseNativeCommandProviderPlacement` |

<a id="coverage-b5248fddcfbc"></a>

## [internal/identity/transport.go](../../../../../internal/identity/transport.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–3 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 4, 16–17 | Record, DTO or interface contract Request for Private IPC and service lifetime; declarations/fields: `Request` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 5–8 | Provider enrollment/owner consent request identity; declarations/fields: `Request.ProviderID`, `Request.OwnerID`, `Request.ID`, `Request.Label` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 9 | Delegated connection availability Project scope; declarations/fields: `Request.ProjectID` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 10–11 | Execution-fence protocol identity; declarations/fields: `Request.Kind`, `Request.ExecutionID` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 12 | Explicit delegated grant request; declarations/fields: `Request.Grant` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 13 | Lease acquisition request; declarations/fields: `Request.Acquire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 14–15 | Attested native registration/return credential payload; declarations/fields: `Request.Binding`, `Request.Credential` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 18–22 | Private runtime delivery only, never admin/browser response; declarations/fields: `DeliveryWire` |

<a id="coverage-ac0839ea542a"></a>

## [internal/identity/types.go](../../../../../internal/identity/types.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 14–21, 31–37, 42–43 | Declared identifiers/bounds (declaration group) for Execution admission and lease fencing; declarations/fields: `(declaration group)` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 22–24 | Supported enrollment provider identifiers; declarations/fields: `Codex`, `Muse` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 25–27 | Connection availability lifecycle metadata; declarations/fields: `Ready`, `Reauth`, `Revoked` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 28–30 | Allowed native execution consumer kinds; declarations/fields: `Factory`, `Terminal` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 38–41 | Terminal execution fence prevents reacquisition after retirement; declarations/fields: `ExecutionTerminal` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 44–71, 134–140 | Connection and delegated grant metadata; declarations/fields: `Connection`, `Grant`, `GrantRequest`, `GrantRequest.Validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 72–85 | Record, DTO or interface contract AcquireRequest for Execution admission and lease fencing; declarations/fields: `AcquireRequest` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 86–99, 119–123, 148–157 | Attested native binding and private delivery DTOs; declarations/fields: `Binding`, `Delivery`, `Binding.Validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 100–114, 116–118 | Record, DTO or interface contract Lease for Execution admission and lease fencing; declarations/fields: `Lease` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 115 | Lease references separately attested native binding; declarations/fields: `Lease.Binding` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 124–133, 216 | Provider enrollment metadata and supported provider identities; declarations/fields: `Enrollment`, `ProviderValid` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 141–147 | AcquireRequest.Validate — Execution admission and lease fencing; declarations/fields: `AcquireRequest.Validate` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 158–164 | Bounded private credential custody input shape; declarations/fields: `CredentialValid` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 165–173 | Record, DTO or interface contract Execution for Execution admission and lease fencing; declarations/fields: `Execution` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 174–201 | Execution.Validate — Execution admission and lease fencing; declarations/fields: `Execution.Validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 202–215 | AcquisitionDigest — Execution admission and lease fencing; declarations/fields: `AcquisitionDigest` |

<a id="coverage-6bfe7af613fe"></a>

## [internal/identity/types_test.go](../../../../../internal/identity/types_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 8–23 | Delegation metadata validation assertions; declarations/fields: `TestGrantRequiresBothConfirmations` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 24–38 | Assertions TestLeaseRequiresBoundedDeadlineAndExecution: expired request admitted; declarations/fields: `TestLeaseRequiresBoundedDeadlineAndExecution` |

