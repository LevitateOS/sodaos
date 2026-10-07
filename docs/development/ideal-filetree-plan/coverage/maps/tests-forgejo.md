# Tests forgejo

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-0dfdb2c69293"></a>

## [tests/forgejo/branding.test.ts](../../../../../tests/forgejo/branding.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–128; Forgejo branding test suite | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Checks native Forgejo branding assets, contrast, icons, and stylesheet payload against the retained product branding contract. — Current source inspected at tests/forgejo/branding.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-94f4433cc4ed"></a>
<a id="testsforgejocomponent-boundariestestts-1"></a>

## [tests/forgejo/component-boundaries.test.ts](../../../../../tests/forgejo/component-boundaries.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–217; Forgejo component boundary test suite | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Checks native Forgejo presentation component boundaries and their allowed local imports. — Current source inspected at tests/forgejo/component-boundaries.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-83aea1d8957a"></a>

## [tests/forgejo/lit-build.test.ts](../../../../../tests/forgejo/lit-build.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; Forgejo Lit build test suite | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Checks bundled Forgejo browser output and packaging/build constraints for the staged native extension payload. — Current source inspected at tests/forgejo/lit-build.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-bfe6d6b2777e"></a>

## [tests/forgejo/lit-runtime.test.ts](../../../../../tests/forgejo/lit-runtime.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–212; Forgejo Lit runtime test suite | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Checks runtime startup and component behavior of the native Forgejo presentation in its browser environment. — Current source inspected at tests/forgejo/lit-runtime.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-6e10460ea92a"></a>

## [tests/forgejo/login-theme.test.ts](../../../../../tests/forgejo/login-theme.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6, 68–132; prior detailed responsibility interval | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Declarations/fixtures and integration for Native Forgejo login theme presentation/behavior contracts; 8 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/login-theme.test.ts; manifest identical_prior confirms byte identity |
| 7–67; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness page — Native Forgejo login theme presentation/behavior contracts; declarations/fields: `page` — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/login-theme.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-138cd13c9c57"></a>

## [tests/forgejo/presentation/form-native-contracts.json](../../../../../tests/forgejo/presentation/form-native-contracts.json)

Inherited prior inventory row

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–750; lines 1–750 (all top-level items and imports) | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current configuration scope; split declarations retain the path's release/install/qualification duty. — tests/forgejo/presentation/form-native-contracts.json; prior ledger docs/development/ideal-filetree-plan/coverage/inventory/tests.md row:  /  [tests/forgejo/presentation/form-native-contracts.json](../../../../../tests/forgejo/presentation/form-native-contracts.json)  /  asset / active  /  [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation)  /  |

<a id="coverage-09c13cc40c79"></a>

## [tests/forgejo/presentation/inventory.json](../../../../../tests/forgejo/presentation/inventory.json)

Inherited prior inventory row

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10008; lines 1–10008 (all top-level items and imports) | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current configuration scope; split declarations retain the path's release/install/qualification duty. — tests/forgejo/presentation/inventory.json; prior ledger docs/development/ideal-filetree-plan/coverage/inventory/tests.md row:  /  [tests/forgejo/presentation/inventory.json](../../../../../tests/forgejo/presentation/inventory.json)  /  asset / active  /  [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation)  /  |

<a id="coverage-91d72f81e84e"></a>

## [tests/forgejo/presentation/inventory.test.ts](../../../../../tests/forgejo/presentation/inventory.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; Forgejo presentation inventory test suite | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Checks the presentation inventory covers production overrides and embedded local callers. — Current source inspected at tests/forgejo/presentation/inventory.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-b23706f4e0cd"></a>

## [tests/forgejo/presentation/locales.test.ts](../../../../../tests/forgejo/presentation/locales.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–53; Forgejo presentation locale test suite | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Checks native Forgejo presentation locale resources and catalog behavior. — Current source inspected at tests/forgejo/presentation/locales.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-cebddad57992"></a>

## [tests/forgejo/presentation/refinement-browser.test.ts](../../../../../tests/forgejo/presentation/refinement-browser.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6, 11–150; prior detailed responsibility interval | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Declarations/fixtures and integration for Native Forgejo refinement browser presentation/reference contracts; 4 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/presentation/refinement-browser.test.ts; manifest identical_prior confirms byte identity |
| 7–10; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness fixture — Native Forgejo refinement browser presentation/reference contracts; declarations/fields: `fixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/presentation/refinement-browser.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-4debca98445e"></a>

## [tests/forgejo/presentation/repository-settings-native-contracts.json](../../../../../tests/forgejo/presentation/repository-settings-native-contracts.json)

Inherited prior inventory row

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–886; lines 1–886 (all top-level items and imports) | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current configuration scope; split declarations retain the path's release/install/qualification duty. — tests/forgejo/presentation/repository-settings-native-contracts.json; prior ledger docs/development/ideal-filetree-plan/coverage/inventory/tests.md row:  /  [tests/forgejo/presentation/repository-settings-native-contracts.json](../../../../../tests/forgejo/presentation/repository-settings-native-contracts.json)  /  asset / active  /  [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation)  /  |

<a id="coverage-c0b0e7f03a7e"></a>

## [tests/forgejo/presentation/settings-native-contracts.json](../../../../../tests/forgejo/presentation/settings-native-contracts.json)

Inherited prior inventory row

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–611; lines 1–611 (all top-level items and imports) | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current configuration scope; split declarations retain the path's release/install/qualification duty. — tests/forgejo/presentation/settings-native-contracts.json; prior ledger docs/development/ideal-filetree-plan/coverage/inventory/tests.md row:  /  [tests/forgejo/presentation/settings-native-contracts.json](../../../../../tests/forgejo/presentation/settings-native-contracts.json)  /  asset / active  /  [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation)  /  |

<a id="coverage-d352a05a26e0"></a>

## [tests/forgejo/presentation/workspace-tokens.test.ts](../../../../../tests/forgejo/presentation/workspace-tokens.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 55–61; prior detailed responsibility interval | [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) | retained | Declarations/fixtures and integration for Authored workspace presentation consumes canonical visual roles; Assert every authored workspace CSS/TS module consumes canonical visual roles; declarations/fields: `test: every authored workspace CSS/TS module consumes canonical visual roles` — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/presentation/workspace-tokens.test.ts; manifest identical_prior confirms byte identity |
| 8–54; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness visualFaults — Authored workspace presentation consumes canonical visual roles; declarations/fields: `visualFaults`; Assert workspace raw-visual guard rejects literals including fallbacks and inline templates; declarations/fields: `test: workspace raw-visual guard rejects literals including fallbacks and inline templates` — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/presentation/workspace-tokens.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-b5d879c5fc55"></a>

## [tests/forgejo/repository-actions.test.ts](../../../../../tests/forgejo/repository-actions.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 14, 82–194; prior detailed responsibility interval | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Declarations/fixtures and integration for Native Forgejo repository actions presentation/behavior contracts; 8 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/repository-actions.test.ts; manifest identical_prior confirms byte identity |
| 8–13, 15–81; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness runActions — Native Forgejo repository actions presentation/behavior contracts; declarations/fields: `runActions`; Test harness fixture — Native Forgejo repository actions presentation/behavior contracts; declarations/fields: `fixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-forgejo.md#tests/forgejo/repository-actions.test.ts; manifest identical_prior confirms byte identity |
