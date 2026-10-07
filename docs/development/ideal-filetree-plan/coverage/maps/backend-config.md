# Backend config

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-75dd7a657b49"></a>

## [internal/config/background_test.go](../../../../../internal/config/background_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; file scaffold; TestBackgroundServiceInputs | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestBackgroundServiceInputs — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-54c7c1af5fa8"></a>

## [internal/config/config.go](../../../../../internal/config/config.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18, 22–23, 57–60, 65–89, 115–182, 213–232; whole file; SodaPath; Config; decodeConfig; validateConfigPaths; Load; Secret | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 7 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 19–21; SodaPath | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Fixed native-extension browser namespace; declarations/fields: `SodaPath` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 24–25, 90–114, 183–196; Config.Listen, Config.ForgejoURL; validateListen, validateConfigURLs, originURL, BaseURL | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Configured native browser/private listener origin; declarations/fields: `Config.Listen`, `Config.ForgejoURL`; Configured native private origin validation; declarations/fields: `validateListen`, `validateConfigURLs`, `originURL`, `BaseURL` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 26; Config.ForgejoInternalURL | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Internal native authoritative read origin; declarations/fields: `Config.ForgejoInternalURL` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 27–30; Config.DatabaseDSNFile | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Restricted PostgreSQL connection-secret file configuration; declarations/fields: `Config.DatabaseDSNFile` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 31–32; Config.HostSocket, Config.IdentitySocket | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private host/broker IPC endpoints; declarations/fields: `Config.HostSocket`, `Config.IdentitySocket` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 33, 197–212; Config.GrantKeyFile; GrantKey | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Provisioned identity credential encryption-key file; declarations/fields: `Config.GrantKeyFile`; Read the provisioned encryption master key without replacement; declarations/fields: `GrantKey` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 34; Config.OperatorID | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Configured native operator identity established by bootstrap; declarations/fields: `Config.OperatorID` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 35–41, 61–64; Config.ForgejoBackgroundSocket, Config.ForgejoBackgroundHostUID, Config.ForgejoBackgroundCredentialFile; Config.BackgroundServiceConfigured | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | Shared background service socket, kernel peer and restricted native credential; declarations/fields: `Config.ForgejoBackgroundSocket`, `Config.ForgejoBackgroundHostUID`, `Config.ForgejoBackgroundCredentialFile`; Optional shared native background admission configuration; declarations/fields: `Config.BackgroundServiceConfigured` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 42–44; Config.ForgejoReviewCredentialFile | [G05](../../slices/forgejo-integration.md#g05-native-review-submission) | retained | Distinct native reviewer credential configuration; declarations/fields: `Config.ForgejoReviewCredentialFile` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 45–47; Config.ForgejoMergeCredentialFile | [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) | retained | Distinct native merge credential configuration; declarations/fields: `Config.ForgejoMergeCredentialFile` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 48–51; Config.FactoryIntakeSecretFile | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Native webhook intake HMAC secret configuration; declarations/fields: `Config.FactoryIntakeSecretFile` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 52–56; Config.FactoryPublicationRoot | [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | retained | Backend-private candidate bundle and Git workspace root; declarations/fields: `Config.FactoryPublicationRoot` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-78b78dc33eda"></a>

## [internal/config/config_test.go](../../../../../internal/config/config_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; file scaffold; TestBaseURL | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestBaseURL — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a4c4d4b34dd9"></a>

## [internal/config/grant_key_test.go](../../../../../internal/config/grant_key_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–46; file scaffold; TestGrantKeyRestrictedAndExact | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestGrantKeyRestrictedAndExact — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a9939b5d64eb"></a>

## [internal/config/intake_test.go](../../../../../internal/config/intake_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; file scaffold; TestFactoryIntakeSecretFile | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryIntakeSecretFile — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6c9bcddd3c22"></a>

## [internal/config/load_test.go](../../../../../internal/config/load_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; whole file; TestLoadRejectsTrailingDataPastSizeLimit | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Assertions TestLoadRejectsTrailingDataPastSizeLimit: small trailing object was not rejected; declarations/fields: `TestLoadRejectsTrailingDataPastSizeLimit` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 40–75; TestPrivateHTTPSDeploymentBoundary | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Configured private-origin validation assertions; declarations/fields: `TestPrivateHTTPSDeploymentBoundary` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-91aa2f184643"></a>

## [internal/config/review_credential_test.go](../../../../../internal/config/review_credential_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; file scaffold; TestReviewCredentialFile | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestReviewCredentialFile — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
