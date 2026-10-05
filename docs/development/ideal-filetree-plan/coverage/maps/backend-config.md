# Backend config

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-54c7c1af5fa8"></a>

## [internal/config/config.go](../../../../../internal/config/config.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 19–21 | Fixed native-extension browser namespace; declarations/fields: `SodaPath` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 22 | Declared identifiers/bounds SodaPath for Configuration and filesystem primitives; declarations/fields: `SodaPath` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 23, 57–60 | Record, DTO or interface contract Config for Configuration and filesystem primitives; declarations/fields: `Config` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 24–25 | Configured native browser/private listener origin; declarations/fields: `Config.Listen`, `Config.ForgejoURL` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 26 | Internal native authoritative read origin; declarations/fields: `Config.ForgejoInternalURL` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 27–30 | Restricted PostgreSQL connection-secret file configuration; declarations/fields: `Config.DatabaseDSNFile` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 31–32 | Private host/broker IPC endpoints; declarations/fields: `Config.HostSocket`, `Config.IdentitySocket` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 33 | Provisioned identity credential encryption-key file; declarations/fields: `Config.GrantKeyFile` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 34 | Configured native operator identity established by bootstrap; declarations/fields: `Config.OperatorID` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 35–41 | Shared background service socket, kernel peer and restricted native credential; declarations/fields: `Config.ForgejoBackgroundSocket`, `Config.ForgejoBackgroundHostUID`, `Config.ForgejoBackgroundCredentialFile` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 42–44 | Distinct native reviewer credential configuration; declarations/fields: `Config.ForgejoReviewCredentialFile` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 45–47 | Distinct native merge credential configuration; declarations/fields: `Config.ForgejoMergeCredentialFile` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 48–51 | Native webhook intake HMAC secret configuration; declarations/fields: `Config.FactoryIntakeSecretFile` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 52–56 | Backend-private candidate bundle and Git workspace root; declarations/fields: `Config.FactoryPublicationRoot` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 61–64 | Optional shared native background admission configuration; declarations/fields: `Config.BackgroundServiceConfigured` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 65–89 | decodeConfig — Configuration and filesystem primitives; declarations/fields: `decodeConfig` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 90–114, 183–196 | Configured native private origin validation; declarations/fields: `validateListen`, `validateConfigURLs`, `originURL`, `BaseURL` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 115–162 | validateConfigPaths — Configuration and filesystem primitives; declarations/fields: `validateConfigPaths` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 163–182 | Load — Configuration and filesystem primitives; declarations/fields: `Load` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 197–212 | Read the provisioned encryption master key without replacement; declarations/fields: `GrantKey` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 213–232 | Secret — Configuration and filesystem primitives; declarations/fields: `Secret` |

<a id="coverage-6c9bcddd3c22"></a>

## [internal/config/load_test.go](../../../../../internal/config/load_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 11–39 | Assertions TestLoadRejectsTrailingDataPastSizeLimit: small trailing object was not rejected; declarations/fields: `TestLoadRejectsTrailingDataPastSizeLimit` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 40–75 | Configured private-origin validation assertions; declarations/fields: `TestPrivateHTTPSDeploymentBoundary` |

