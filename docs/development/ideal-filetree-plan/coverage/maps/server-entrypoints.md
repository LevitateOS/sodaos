# Server entrypoints

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-ec73c0fae138"></a>

## [cmd/soda-dashboard/main.go](../../../../../cmd/soda-dashboard/main.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 21–27 | main — Private IPC and service lifetime; declarations/fields: `main` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 28, 33–37, 46–73 | run — Private IPC and service lifetime; declarations/fields: `run` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 29 | Configured service input path; declarations/fields: `run` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 30 | Optional native-extension private endpoint configuration; declarations/fields: `run` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 31–32 | Private factory operator endpoint and explicit OS principal configuration; declarations/fields: `run` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 38–42 | Close shared PostgreSQL handle on service exit; declarations/fields: `run` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 43–45 | Acquire exclusive coordinator lifetime before serving; declarations/fields: `run`, `StartCoordinator` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 74–83 | Coordinator operation admission and lifecycle wiring; declarations/fields: `operatorEndpoint` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 84–92 | serveOperatorSocket — Private IPC and service lifetime; declarations/fields: `serveOperatorSocket` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 93, 113–115 | openDashboard — Private IPC and service lifetime; declarations/fields: `openDashboard` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 94–96 | Validate canonical authored avatar definitions before serving; declarations/fields: `openDashboard`, `avatar.Validate` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 97–100 | Load configured service inputs; declarations/fields: `openDashboard`, `config.Load` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 101–104 | Read provisioned credential encryption key; declarations/fields: `openDashboard`, `config.GrantKey` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 105–108 | Read restricted PostgreSQL DSN boundary; declarations/fields: `openDashboard`, `config.Secret` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 109–112 | Open existing credential store with provisioned encryption-key check; declarations/fields: `openDashboard`, `store.OpenEncrypted` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 116–122 | extensionHTTPServer — Private IPC and service lifetime; declarations/fields: `extensionHTTPServer` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 123–131 | serveExtensionSocket — Private IPC and service lifetime; declarations/fields: `serveExtensionSocket` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 132–136, 139–151 | Bounded explicit HTTP/private socket service shutdown; declarations/fields: `beginShutdown` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 137 | Terminate/cancel tracked human terminal operation streams at backend shutdown; declarations/fields: `beginShutdown`, `CloseTerminals` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 138 | Retire exclusive factory coordinator lifetime at backend shutdown; declarations/fields: `beginShutdown`, `CloseCoordinator` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 152–156 | shutdownServer — Private IPC and service lifetime; declarations/fields: `shutdownServer` |

<a id="coverage-e446e6589b09"></a>

## [cmd/soda-dashboard/operator.go](../../../../../cmd/soda-dashboard/operator.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 19–49 | Admitted operator-only factory command endpoint; declarations/fields: `operatorHTTPServer` |

<a id="coverage-370aad109b4e"></a>

## [cmd/soda-dashboard/operator_linux.go](../../../../../cmd/soda-dashboard/operator_linux.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 11–58 | Operator principal boundary for private factory commands; declarations/fields: `operatorPeerListener`, `gateOperatorPeers`, `operatorPeerListener.Accept`, `operatorPeerOK` |

<a id="coverage-eb2c8e223d64"></a>

## [cmd/soda-dashboard/operator_linux_test.go](../../../../../cmd/soda-dashboard/operator_linux_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 18–74 | Operator principal boundary for private factory commands; declarations/fields: `operatorRoundTrip`, `TestOperatorSocketAdmitsConfiguredPeer`, `TestOperatorSocketRebindsStalePath`, `TestOperatorSocketRefusesForeignPeer` |

<a id="coverage-978604ee0fa6"></a>

## [cmd/soda-dashboard/operator_other.go](../../../../../cmd/soda-dashboard/operator_other.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–9 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 10–12 | Operator principal boundary for private factory commands; declarations/fields: `gateOperatorPeers` |

<a id="coverage-6105744cc963"></a>

## [cmd/soda-dashboard/operator_test.go](../../../../../cmd/soda-dashboard/operator_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–9 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 10–19 | Operator principal boundary for private factory commands; declarations/fields: `TestOperatorEndpointRequiresExplicitPeer` |

<a id="coverage-0c5003e81409"></a>

## [cmd/soda-extension/main.go](../../../../../cmd/soda-extension/main.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 19–41 | run — Browser authority and contributions; declarations/fields: `run` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 42–57, 63–71 | Restricted configured operator identity file boundary; declarations/fields: `readOperatorID`, `canonicalOperatorID` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 58–62 | operatorIDFileAvailable — Browser authority and contributions; declarations/fields: `operatorIDFileAvailable` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 72–80 | contributionAuthorizer — Browser authority and contributions; declarations/fields: `contributionAuthorizer` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 81–89 | allowedContribution — Browser authority and contributions; declarations/fields: `allowedContribution` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 90–95 | main — Browser authority and contributions; declarations/fields: `main` |

