# Tests forgejo

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-0dfdb2c69293"></a>

## [tests/forgejo/branding.test.ts](../../../../../tests/forgejo/branding.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1–6, 69, 81, 108 | Declarations/fixtures and integration for Canonical imagery/functional icons/licenses and shared palette contrast |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 7–68 | Assert Forgejo delivers no decorative robot artwork and retains the new identity; declarations/fields: `test: Forgejo delivers no decorative robot artwork and retains the new identity` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 70–80 | Assert Spaces functional icons are bounded static masks staged with their MIT notice; declarations/fields: `test: Spaces functional icons are bounded static masks staged with their MIT notice` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 82–107 | Assert shared neutral surfaces, links, actions and control edges retain contrast in both modes; declarations/fields: `test: shared neutral surfaces, links, actions and control edges retain contrast in both modes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 109–123 | Assert every stylesheet referenced by the Forgejo header is in the canonical payload; declarations/fields: `test: every stylesheet referenced by the Forgejo header is in the canonical payload` |

<a id="coverage-94f4433cc4ed"></a>

<a id="testsforgejocomponent-boundariestestts-1"></a>

## [tests/forgejo/component-boundaries.test.ts](../../../../../tests/forgejo/component-boundaries.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–8 | Declarations/fixtures and integration for Native Forgejo component boundaries presentation/behavior contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 9–30 | Load exact candidate/native CSS cascade and test-owned browser renderer; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 31–120 | Assert native settings headings, inset bodies and empty-state boundaries across themes/widths; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 121–155 | Assert native normal/error/focused field styling preserves control state; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 156–211 | Assert native repository clone/actions toolbar control geometry and compact boundaries; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 212–277 | Assert semantic neutral/primary/native action control states and variants; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 278–360 | Assert attached settings surfaces, repository navigation, status canvas and form roles; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 361–469 | Assert native primary/disabled/loading/red/positive and public profile control state; declarations/fields: `test: expanded components preserve native state and layout boundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 470–510 | Assert package cleanup preview table remains horizontally reachable across widths; declarations/fields: `test: expanded components preserve native state and layout boundaries` |

<a id="coverage-83aea1d8957a"></a>

## [tests/forgejo/lit-build.test.ts](../../../../../tests/forgejo/lit-build.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–10, 28, 47, 64, 79 | Declarations/fixtures and integration for Native browser build runtime/import/source closure and notice staging |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 11–27 | Assert Lit submodules cannot silently become unresolved or duplicate runtimes; declarations/fields: `test: Lit submodules cannot silently become unresolved or duplicate runtimes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 29–46 | Assert analysis tools cannot enter browser payloads through external imports; declarations/fields: `test: analysis tools cannot enter browser payloads through external imports` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 48–63 | Assert emitted Forgejo imports share the presentation cache epoch; declarations/fields: `test: emitted Forgejo imports share the presentation cache epoch` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 65–78 | Assert staged Lit notice matches the resolved browser dependency licenses; declarations/fields: `test: staged Lit notice matches the resolved browser dependency licenses` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 80–85 | Assert Lit scaffolding does not eagerly load or replace native page controls; declarations/fields: `test: Lit scaffolding does not eagerly load or replace native page controls` |

<a id="coverage-bfe6d6b2777e"></a>

## [tests/forgejo/lit-runtime.test.ts](../../../../../tests/forgejo/lit-runtime.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–16, 26, 41, 176 | Declarations/fixtures and integration for Bundled Lit runtime source, browser smoke and installed asset-serving assertions |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 17–25 | Test harness buildFixtureModules — Bundled Lit runtime source, browser smoke and installed asset-serving assertions; declarations/fields: `buildFixtureModules` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 27–40 | Assert Lit production runtime is staged as a self-contained browser module; declarations/fields: `test: Lit production runtime is staged as a self-contained browser module` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 42–175 | Assert Lit production runtime upgrades and updates independent elements in Chromium; declarations/fields: `test: Lit production runtime upgrades and updates independent elements in Chromium` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 177–212 | Assert Installed appliance serves its first-party assets in Chromium; declarations/fields: `test: Installed appliance serves its first-party assets in Chromium` |

<a id="coverage-6e10460ea92a"></a>

## [tests/forgejo/login-theme.test.ts](../../../../../tests/forgejo/login-theme.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–6 | Declarations/fixtures and integration for Native Forgejo login theme presentation/behavior contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 7–67 | Test harness page — Native Forgejo login theme presentation/behavior contracts; declarations/fields: `page` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 68–76 | Assert system preference applies before DOM readiness and follows system changes; declarations/fields: `test: system preference applies before DOM readiness and follows system changes` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 77–82 | Assert explicit preference wins over the system; declarations/fields: `test: explicit preference wins over the system` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 83–96 | Assert toggle persists the next mode and exposes its next action; declarations/fields: `test: toggle persists the next mode and exposes its next action` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 97–103 | Assert pages without a guest toggle still apply the pre-paint preference safely; declarations/fields: `test: pages without a guest toggle still apply the pre-paint preference safely` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 104–111 | Assert blocked storage still allows an in-memory choice; declarations/fields: `test: blocked storage still allows an in-memory choice` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 112–123 | Assert cross-tab changes synchronize; clearing preference restores system mode; declarations/fields: `test: cross-tab changes synchronize; clearing preference restores system mode` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 124–132 | Assert invalid stored values fall back safely and app subpaths have distinct keys; declarations/fields: `test: invalid stored values fall back safely and app subpaths have distinct keys` |

<a id="coverage-91d72f81e84e"></a>

## [tests/forgejo/presentation/inventory.test.ts](../../../../../tests/forgejo/presentation/inventory.test.ts)

Source/assertion inspection only; no suite or native scenario executed. Local source/component/native-session scopes differ; fixture rendering is not installed-release qualification.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–5, 47–49 | Declarations/fixtures and integration for Native Forgejo inventory presentation/reference contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 6–14 | Test harness templates — Native Forgejo inventory presentation/reference contracts; declarations/fields: `templates` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 15–46 | Assert presentation inventory covers every production override and its local callers; declarations/fields: `test: presentation inventory covers every production override and its local callers` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 50–79 | Assert embedded native callers match the inventory when the local export is available; declarations/fields: `test: embedded native callers match the inventory when the local export is available` |

<a id="coverage-b23706f4e0cd"></a>

## [tests/forgejo/presentation/locales.test.ts](../../../../../tests/forgejo/presentation/locales.test.ts)

Source/assertion inspection only; no suite or native scenario executed. Local source/component/native-session scopes differ; fixture rendering is not installed-release qualification.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–9, 40 | Declarations/fixtures and integration for Native Forgejo locales presentation/reference contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 10–39 | Test harness merge — Native Forgejo locales presentation/reference contracts; declarations/fields: `merge` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 41–53 | Assert locale generation preserves native bytes and rejects namespace/duplicate-key collisions; declarations/fields: `test: locale generation preserves native bytes and rejects namespace/duplicate-key collisions` |

<a id="coverage-cebddad57992"></a>

## [tests/forgejo/presentation/refinement-browser.test.ts](../../../../../tests/forgejo/presentation/refinement-browser.test.ts)

Source/assertion inspection only; no suite or native scenario executed. Local source/component/native-session scopes differ; fixture rendering is not installed-release qualification.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–6 | Declarations/fixtures and integration for Native Forgejo refinement browser presentation/reference contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 7–10 | Test harness fixture — Native Forgejo refinement browser presentation/reference contracts; declarations/fields: `fixture` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 11–18 | luminance — Native Forgejo refinement browser presentation/reference contracts; declarations/fields: `luminance` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 19–23 | contrast — Native Forgejo refinement browser presentation/reference contracts; declarations/fields: `contrast` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 24–150 | Assert refined page families fit narrow screens and keep content readable; declarations/fields: `test: refined page families fit narrow screens and keep content readable` |

<a id="coverage-d352a05a26e0"></a>

## [tests/forgejo/presentation/workspace-tokens.test.ts](../../../../../tests/forgejo/presentation/workspace-tokens.test.ts)

Source/assertion inspection only; no suite or native scenario executed. Local source/component/native-session scopes differ; fixture rendering is not installed-release qualification.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 1–7 | Declarations/fixtures and integration for Authored workspace presentation consumes canonical visual roles |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 8–29 | Test harness visualFaults — Authored workspace presentation consumes canonical visual roles; declarations/fields: `visualFaults` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 30–54 | Assert workspace raw-visual guard rejects literals including fallbacks and inline templates; declarations/fields: `test: workspace raw-visual guard rejects literals including fallbacks and inline templates` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 55–61 | Assert every authored workspace CSS/TS module consumes canonical visual roles; declarations/fields: `test: every authored workspace CSS/TS module consumes canonical visual roles` |

<a id="coverage-b5d879c5fc55"></a>

## [tests/forgejo/repository-actions.test.ts](../../../../../tests/forgejo/repository-actions.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–7, 14, 82, 99, 116, 128, 144, 163, 181 | Declarations/fixtures and integration for Native Forgejo repository actions presentation/behavior contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 8–13 | Test harness runActions — Native Forgejo repository actions presentation/behavior contracts; declarations/fields: `runActions` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 15–81 | Test harness fixture — Native Forgejo repository actions presentation/behavior contracts; declarations/fields: `fixture` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 83–98 | Assert initial mode follows CSS and wide actions ignore dismissal; declarations/fields: `test: initial mode follows CSS and wide actions ignore dismissal` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 100–115 | Assert mode transitions open wide actions and preserve a focused action on compaction; declarations/fields: `test: mode transitions open wide actions and preserve a focused action on compaction` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 117–127 | Assert expanding to wide from its hidden summary focuses the first available native action; declarations/fields: `test: expanding to wide from its hidden summary focuses the first available native action` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 129–143 | Assert Escape closes compact actions and setup remains idempotent; declarations/fields: `test: Escape closes compact actions and setup remains idempotent` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 145–162 | Assert outside click and focusout close compact actions without stealing focus; declarations/fields: `test: outside click and focusout close compact actions without stealing focus` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 164–180 | Assert HTMX-like form replacement restores focus without closing compact actions; declarations/fields: `test: HTMX-like form replacement restores focus without closing compact actions` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 182–194 | Assert native modal and action events remain untouched in compact mode; declarations/fields: `test: native modal and action events remain untouched in compact mode` |

