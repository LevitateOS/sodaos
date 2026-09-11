# Working on SodaOS

## Working style

- Inspect the working tree first; preserve unrelated changes. Make coherent commits;
  do not amend or rewrite history without permission.
- Finish approved work without repeated handoffs. Ask again only for actions outside
  approval or a concrete safety issue requiring the user's decision.
- Use the smallest sufficient investigation and affected-contract checks. Reuse valid
  evidence; historical execution sequences are not standing gates.
- Prefer upstream mechanisms and direct, concrete code over duplicated authority,
  speculative frameworks or new orchestration. Check the selected upstream version
  and actual Soda caller before adding an adapter or declaring a limitation.
- Report changes, checks actually run and remaining limitations concisely. For
  substantial changes, update `docs/implementation-status.md`; do not add timelines here.

## Product and authority boundaries

- Stock Forgejo owns identity, authentication, Git, permissions and administration.
  Use supported configuration, templates, assets and APIs. No Forgejo fork, direct
  database access or replacement authority; revisit the design if one seems necessary.
- Soda's Go backend owns environment/access integration. Use native Forgejo pages
  with Lit management/terminal controls, not a standalone Soda dashboard. Cockpit
  retains its separate React/PatternFly operator UI, loopback-first.
- Project owners administer projects, not the appliance. Native root, the configured
  Soda operator and Forgejo site administrators are distinct authorities. Developer
  Linux accounts belong inside projects, not on the host.
- Enforce authorization server-side from trusted state; preserve session, OAuth,
  CSRF and input protections. The host helper exposes fixed operations, not arbitrary
  commands or unrestricted host Podman access.
- Projects have persistent roots, accounts, homes, keys, installed tools and service
  data. Normal startup starts the existing container. Joining must provision real
  access; shared resources mean actual shared tools/files/services. Keep Git, mise
  and SSH native; a bridge address alone does not prove client reachability.
- Providers own CI workflows, scheduling and results; Soda manages Forgejo-only local
  runner capacity. This does not remove project GitHub CLI tools or native imports.
  Preserve stock Cockpit and Tailnet. Keep Cockpit Runners, its backing logic and tests
  until replacement parity and separately approved retirement.

## Permissions and preservation

- Routine local implementation, builds and tests for the planned work are authorized.
  Appliance installation, service/VM lifecycle, real provider registration/jobs,
  publishing/automatic CI, network/trust changes and cleanup require applicable
  target/action approval. A command, input file or old approval is not a new grant.
- Before affecting retained state, consult the target's latest handoff and current
  approval. Preserve roots, credentials, fixtures, evidence and later writes. Take
  appropriate consistent backups for authorized maintenance; never blindly restore
  an older database over later writes or replay a mutation to repair an observer.
- Clean up only explicitly authorized, exact resources. Do not use `--rm`, `--replace`,
  pruning or root recreation as repair shortcuts for persistent projects.
- Use restricted secret-file inputs. Never expose credentials in source, argv, logs,
  screenshots or evidence, or request a developer's private SSH key for onboarding.
  Full container inspection and provisioning outputs can contain secrets.
- Distinguish authored checks, local tests, native builds and installed evidence.
  x86_64 and aarch64 are targets; cross-compilation/emulation is not native proof,
  and one architecture's work does not require an unrelated sibling-build gate.

## Code and tooling

- Use Go for backend/setup/privileged integration. New JS-family source is TypeScript;
  retain strict checks, validate untrusted data from `unknown`, and use `import type`.
  Do not bypass checking with blanket `any`, unchecked casts or disabled checks.
- Use the root Bun workspace and single `bun.lock`. Tool/dependency versions belong
  in `go.mod`, manifests and source lockfiles, not this guide. Avoid incidental upgrades.
- Add focused behavior, failure and authorization tests with changes. Keep callers,
  generated browser assets and staging wired together; reuse existing test drivers.
- Generated outputs belong in ignored `.artifacts/`; private inputs stay untracked.
  Preserve canonical `assets/`, attribution and licenses. Leave the separate
  predecessor repository and its Updates platform outside this work.

## Commands

Run from the repository root with the pinned toolchain and documented prerequisites.
Choose checks for the change; this table is not a mandatory sequence.

| Command | Purpose |
| --- | --- |
| `bun install --frozen-lockfile` | Install the locked workspace dependencies. |
| `go test ./internal/runners` | Example focused Go package test; select the affected package/tests. |
| `bun run typecheck` | Strict TypeScript and Lit checks. |
| `bun run test:frontend` / `bun run test:forgejo` | Build browser assets and run the selected suite. |
| `bun run test:pages` | Native-page fixture checks; requires the authorized local Forgejo fixture. |
| `bun run --cwd cockpit test` | Cockpit tests. |
| `bun run check:source` | Broad Go, TypeScript, browser/Cockpit and Python source checks. |
| `bash scripts/build-native.sh ARCH` | Resolve/build/stage native artifacts; does not install or publish. |
| `bash scripts/check-native.sh ARCH` | Native checks against a prepared matching-architecture stage. |

`ARCH` is `x86_64` or `aarch64`. Read the deployment/support guides before using
install/activation, provisioning, VM tools or `tests/installed/`; they can change
real state. Provisioning output contains sensitive password hashes.

## Task-specific documentation

Read the guides relevant to the change, not this entire list. Service/image source
in `appliance/services/` and `project-os/` establishes implemented topology.

| Area | Guide |
| --- | --- |
| Product scope and ownership | [Architecture](docs/architecture.md), [Sodaspaces](docs/sodaspaces-plan.md), [deferred work](docs/deferred.md) |
| Native pages and Runners | [Active combined plan](docs/native-pages-runners-plan.md), [page integration](docs/forgejo-soda-pages-plan.md), [runner contracts](docs/runners-port.md) |
| Forgejo customization and UI | [Supported integration](docs/forgejo-frontend-integration.md), [Lit](docs/lit.md), [TypeScript and test prerequisites](docs/typescript.md) |
| Project runtime and access | [Project OS](docs/project-os.md), [terminals](docs/terminal-integration.md), [API](docs/dashboard-api.md), [credentials](docs/dashboard-credentials.md) |
| Cockpit and Tailnet | [Cockpit](docs/cockpit-port.md), [operator setup](docs/operator-setup.md) |
| Build, deployment and native tools | [Installation](docs/installation.md), [native validation](docs/native-validation.md), [support-tool effects](docs/native-support.md) |
| Browser screenshots | [Capture and fixture login](docs/screenshot-capture.md); use `scripts/screenshot.ts`. |
| Refactoring | [Upstream-first review](docs/refactoring-plan.md#1-upstream-first-review) |
| Retained state, permissions and evidence | [Current handoff](docs/implementation-status.md), [local testing](docs/local-testing.md); historical receipts do not renew permissions. |
