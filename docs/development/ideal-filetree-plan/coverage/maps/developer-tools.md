# Developer tools

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-282444ebc80f"></a>

## [tools/png-equal/main_test.go](../../../../../tools/png-equal/main_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1–13, 34, 42 | Declarations/fixtures and integration for Decoded PNG equality and dimension bounds |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 14–33 | Assert Equal PNGCompares Decoded Pixels; declarations/fields: `TestEqualPNGComparesDecodedPixels` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 35–41 | Assert Equal PNGRejects Unbounded Dimensions; declarations/fields: `TestEqualPNGRejectsUnboundedDimensions` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 43–49 | Test harness writePNG — Decoded PNG equality and dimension bounds; declarations/fields: `writePNG` |

<a id="coverage-e265944c7361"></a>

## [tools/soda-installed-probes/main.go](../../../../../tools/soda-installed-probes/main.go)

Live Go installed-probe dispatcher: tests/build/sodaspaces_test.go:68–81 invokes developer-access, and six actions delegate to existing internal/acceptance probe functions. Distinct from ported Rust acceptance orchestration.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–32 | Expose live native installed probe CLI usage/exit and error reporting; declarations/fields: `main`, `exitCode`, `toolUsage` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 33–37 | Require explicit installed probe subcommand; declarations/fields: `run` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 38–39 | Dispatch scoped developer SSH/access observation; declarations/fields: `acceptance.RunDeveloperAccess` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 40–41 | Dispatch selected native Project lifecycle state observation; declarations/fields: `acceptance.RunLifecycleState` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 42–43 | Dispatch ordinary account's personal native Git credential/access observation; declarations/fields: `acceptance.RunPersonalGit` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 44–45 | Dispatch configured native service HTTPS observation; declarations/fields: `acceptance.RunServiceHTTPS` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 46–47 | Dispatch scoped routed native workload service-access observation; declarations/fields: `acceptance.RunWorkloadAccess` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 48–49 | Dispatch scoped selected native workload execution probe; declarations/fields: `acceptance.RunWorkloadExec` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 50–57 | Refuse unknown probe subcommand with bounded usage error; declarations/fields: `run`, `dispatchUsage` |

<a id="coverage-bb7c2d6f4104"></a>

## [tools/soda-installed-probes/main_test.go](../../../../../tools/soda-installed-probes/main_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–11, 40 | Declarations/fixtures and integration for Installed probe command dispatch/refusal contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 12–39 | Assert Dispatch; declarations/fields: `TestDispatch` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 41–52 | Assert Service HTTPSProbe Failure; declarations/fields: `TestServiceHTTPSProbeFailure` |

