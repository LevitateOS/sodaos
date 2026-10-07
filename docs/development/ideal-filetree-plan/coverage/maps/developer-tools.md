# Developer tools

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-a60de7de07ef"></a>

## [internal/archcheck/arch_test.go](../../../../../internal/archcheck/arch_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–269; file scaffold; modulePath; retired; moduleRoot; internalPackageDirs; productionImports; importsOf; forbid; allowOnly; TestRetiredPackagesStayDeleted; TestPackageNamesMatchDirectories; TestDependencyDirection | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-14dd0303dec0"></a>

## [tools/pg-fixture/src/main.rs](../../../../../tools/pg-fixture/src/main.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–623; whole file: disposable postgresql test fixture controller: protected password handoff, container startup/readiness, role/database setup and cleanup | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Disposable PostgreSQL test fixture controller: protected password handoff, container startup/readiness, role/database setup and cleanup. — tools/pg-fixture/src/main.rs module docs; consumed by local database/test fixture callers |

<a id="coverage-8d3f78fb972a"></a>

## [tools/png-equal/main.go](../../../../../tools/png-equal/main.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; file scaffold; main; decodeBoundedPNG; equalPNG; rgbaPixels | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-282444ebc80f"></a>

## [tools/png-equal/main_test.go](../../../../../tools/png-equal/main_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42; whole file; TestEqualPNGComparesDecodedPixels; TestEqualPNGRejectsUnboundedDimensions | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Declarations/fixtures and integration for Decoded PNG equality and dimension bounds; Assert Equal PNGCompares Decoded Pixels; declarations/fields: `TestEqualPNGComparesDecodedPixels`; Assert Equal PNGRejects Unbounded Dimensions; declarations/fields: `TestEqualPNGRejectsUnboundedDimensions` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 43–49; writePNG | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness writePNG — Decoded PNG equality and dimension bounds; declarations/fields: `writePNG` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-53a5cedbbc3b"></a>

## [tools/soda-avatars/main.go](../../../../../tools/soda-avatars/main.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–226; file scaffold; sheet; main; avatarBaseline; writeAvatarSVG; variantSection; colorSection; robotSamples; writeAvatarIndex; catalogAvatarSheet; prepareAvatarWorkspace; generate; page | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bdfb103da8d3"></a>

## [tools/soda-avatars/main_test.go](../../../../../tools/soda-avatars/main_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–49; file scaffold; TestCatalogUsesAllPartsAndProductionSamples | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestCatalogUsesAllPartsAndProductionSamples — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b593b83c2299"></a>

## [tools/soda-installed-probes/cockpit_account_test.go](../../../../../tools/soda-installed-probes/cockpit_account_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45; file scaffold; TestCockpitAccountRefusesOutsideNativeRoot; TestCockpitAccountContracts | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestCockpitAccountRefusesOutsideNativeRoot; Current declaration duty: TestCockpitAccountContracts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dcdfd4e01239"></a>

## [tools/soda-installed-probes/installed_probes_test.go](../../../../../tools/soda-installed-probes/installed_probes_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; probeRoot; checkProbe; probeResult; runProbe; readProbeSource; remoteProbeBinary | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e265944c7361"></a>

## [tools/soda-installed-probes/main.go](../../../../../tools/soda-installed-probes/main.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 50–57; main, exitCode, toolUsage; run; run, dispatchUsage | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Expose live native installed probe CLI usage/exit and error reporting; declarations/fields: `main`, `exitCode`, `toolUsage`; Require explicit installed probe subcommand; declarations/fields: `run`; Refuse unknown probe subcommand with bounded usage error; declarations/fields: `run`, `dispatchUsage` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 38–39, 42–43; acceptance.RunDeveloperAccess; acceptance.RunPersonalGit | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Dispatch scoped developer SSH/access observation; declarations/fields: `acceptance.RunDeveloperAccess`; Dispatch ordinary account's personal native Git credential/access observation; declarations/fields: `acceptance.RunPersonalGit` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 40–41; acceptance.RunLifecycleState | [P05](../../slices/projects.md#p05-project-startstop) | retained | Dispatch selected native Project lifecycle state observation; declarations/fields: `acceptance.RunLifecycleState` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 44–45; acceptance.RunServiceHTTPS | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Dispatch configured native service HTTPS observation; declarations/fields: `acceptance.RunServiceHTTPS` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 46–47; acceptance.RunWorkloadAccess | [N02](../../slices/networking.md#n02-project-lan-access) | retained | Dispatch scoped routed native workload service-access observation; declarations/fields: `acceptance.RunWorkloadAccess` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 48–49; acceptance.RunWorkloadExec | [P11](../../slices/projects.md#p11-nested-services-and-volumes) | retained | Dispatch scoped selected native workload execution probe; declarations/fields: `acceptance.RunWorkloadExec` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-bb7c2d6f4104"></a>

## [tools/soda-installed-probes/main_test.go](../../../../../tools/soda-installed-probes/main_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–40; whole file; TestDispatch | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Declarations/fixtures and integration for Installed probe command dispatch/refusal contracts; Assert Dispatch; declarations/fields: `TestDispatch` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 41–52; TestServiceHTTPSProbeFailure | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Assert Service HTTPSProbe Failure; declarations/fields: `TestServiceHTTPSProbeFailure` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-26e9dfa543f2"></a>

## [tools/soda-installed-probes/project_state_test.go](../../../../../tools/soda-installed-probes/project_state_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; file scaffold; TestProjectStateRefusesOutsideRootContainer; TestProjectStateSnapshotContracts | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestProjectStateRefusesOutsideRootContainer; Current declaration duty: TestProjectStateSnapshotContracts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cdc0dffc6296"></a>

## [tools/soda-rootfs-server/main.go](../../../../../tools/soda-rootfs-server/main.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–142; file scaffold; rootDir; allowName; requestPath; openAllowed; serveRootfs; run; main | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d9d2d0167c70"></a>

## [tools/soda-rootfs-server/main_test.go](../../../../../tools/soda-rootfs-server/main_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–205; file scaffold; liveGuestImages; allowNameFixture; repoPath; setupRootfsDir; TestSetupAndServerConvergeOnOneHomeDirectory; TestUnitRunsTheSingleServerSource; TestExactFileServing | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
