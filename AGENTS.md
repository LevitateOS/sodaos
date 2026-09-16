# Working on SodaOS

## Unreleased: optimize for simplicity

Build as if nobody is using the product. Experimental data and build artifacts are
not worth permanent code complexity. Older plans to retain experiments do not
justify keeping obsolete implementations alive.

- **One current implementation.** Replace obsolete code, formats, callers and test
  fixtures together. Do not add legacy readers, compatibility branches, migrations
  or version negotiation for unreleased experiments unless the owner explicitly
  requires them. A format identifier can reject old input without supporting it.
- **Solve the requested problem, not a larger imagined one.** Implement the current
  working path first. Add other paths only for demonstrated failures, actual callers
  or explicit requirements. Briefly note useful unresolved concerns; do not build
  speculative recovery, fallback or prevention subsystems around them.
- **Removal should remove code.** A removal or simplification request is not a reason
  to add compatibility scaffolding or a subsystem that checks the removed feature
  never returns. Count moved/new equivalents when assessing complexity. If the
  solution grows instead of simplifying, reconsider it before expanding the patch.
  Do not weaken necessary verification or merely compress code to improve LOC totals.
- **Use mature upstream implementations directly.** Check the selected version and
  actual Soda caller before adding an adapter or declaring a limitation. Do not
  recreate an upstream service, protocol or state machine for an optional feature;
  drop or defer the feature when its maintenance cost outweighs its value.
- **The owner's limited weekly token budget is a hard engineering constraint.**
  Before costly investigation, implementation or execution, identify the unresolved
  fact, the evidence the action will produce, and the cheapest sufficient way to
  obtain it. Reuse evidence, artifacts and test drivers. Prove critical native
  integration assumptions before building orchestration around them. Do not turn a
  narrow request into a broad audit, test framework or documentation project.
  Reconsider the approach when failures invalidate its assumptions; do not merely
  keep patching the chosen plan. Stop once sufficient evidence exists, and keep
  reasoning, polling and reporting concise.
- **Separate development experiments from final qualification.** A failed release
  run must not be relabeled successful or resumed as a qualified release, but its
  retained artifacts can support authorized, explicitly non-qualifying development
  checks. Driver-only debugging does not inherently require rebuilding shipping
  bytes. Use the smallest applicable development target; run full production
  qualification when its prerequisites are demonstrated, not as the default debug loop.

## Working style

- Prioritize engineering correctness over agreement. Challenge flawed assumptions
  and technically weaker proposals directly, including the owner's; explain the
  concrete tradeoff and recommend the stronger option. Pushback alone is not new
  technical evidence. When changing a recommendation, identify the evidence,
  corrected reasoning, requirement or priority that changed it. State uncertainty;
  do not invent certainty or disagreement. Respect explicit owner decisions within
  approval, but do not recast a chosen tradeoff as the technically stronger option.

- **Learn from the M4 execution failure.** The agent wrote roughly 1,500 lines of
  qualification orchestration before validating critical native boundaries, then
  repeated approximately 22-minute production builds while those boundaries were
  unresolved. Passing tests and compliance with a self-selected plan did not justify
  that sequencing or expense. Review necessity and cost independently of correctness.
  Broad task approval does not approve every implementation choice. Own the actual
  decision and its rationale; do not substitute agreement for reassessment or frame
  the owner's status questions as technical pushback. Reassess from available
  evidence without waiting for the owner to notice waste.

- Never put the owner's personal name or other identifying information in source,
  tests, fixtures, example accounts, generated resource names or documentation.
  Use neutral synthetic identities such as `soda-tester`. Do not derive fixture
  names from local usernames, home-directory paths or account profiles.

- Work in the canonical `~/Projects/sodaos` checkout for this redesign. Do not create or use a worktree unless the owner explicitly changes that preference.

- Inspect the working tree first; preserve unrelated changes. Make coherent commits;
  do not amend or rewrite history without permission.
- Finish approved work without repeated handoffs. Ask again only for actions outside
  approval or a concrete safety issue requiring the user's decision.
- Use the smallest sufficient investigation and affected-contract checks. A passing
  receipt supports its stated scope, not a prescribed sequence. Reuse valid evidence;
  ground required checks in current contracts, source behavior or an explicit user
  decision. Label optional diagnostic/review choices as recommendations.
- Update requirements in their owning guide; link to them elsewhere instead of
  copying rules or appending exceptions. The documentation map below identifies owners.
  Documentation authority rules live in [docs/README.md](docs/README.md#authority-rules):
  one canonical owner per subject; product docs are not status trackers; completed
  plans are absorbed and deleted; research is non-normative.
- Report changes, checks actually run and remaining limitations concisely. Prefer
  issues, milestones and Git history for transient status. Detailed receipts belong
  in history, not additional durable docs.

## Permissions and preservation

- The small root disk is permanently and completely off-limits, forever. Never put
  anything on `/`: no builds, state, caches, temp files, installs, cleanups or
  other writes. Never propose the root disk as an option, a fallback or a
  shortcut, and never present it as one. The roomy disk (`/home`) is the only
  workspace. If work cannot be done there, stop and say so instead of reaching
  for `/`. This covers the root filesystem only; explicitly approved privilege
  use is unaffected.
- Destructive or externally visible work needs explicit task approval in the current
  conversation or task brief. Documentation is not a grant. Appliance installation,
  service/VM lifecycle, real provider registration/jobs, publishing/automatic CI,
  network/trust changes and cleanup require applicable target/action approval.
  A command, input file or old approval is not a new grant.
- Artifact retention is not a compatibility requirement. Experimental roots, fixtures
  and evidence may be retired rather than supported by current code. This does not
  itself authorize deletion: consult the target's current approval, and clean up only
  authorized, exact resources. Do not use `--rm`, `--replace`, pruning or root recreation
  as shortcuts against unrelated or explicitly protected state.
- Protect credentials, unrelated work and any state the owner explicitly requires
  keeping. Take consistent backups when that protection requires them; do not create
  a preservation programme for disposable experiments. Never blindly restore an old
  database over protected later writes or replay a mutation to repair an observer.
- Use restricted secret-file inputs. Never expose credentials in source, argv, logs,
  screenshots or evidence, or request a developer's private SSH key for onboarding.
  Full container inspection and provisioning outputs can contain secrets.
- Distinguish authored checks, local tests, native builds and installed evidence.
  x86_64 and aarch64 are targets; cross-compilation/emulation is not native proof,
  and one architecture's work does not require an unrelated sibling-build gate.

## Code and tooling

- Use Go for backend/setup/privileged integration. The [Go ownership guide](docs/development/go.md)
  owns package placement, SQL locality and house style. The [TypeScript guide](docs/development/typescript.md)
  owns JS-family language, strict typing, Bun workspace and asset-porting conventions.
  Versions belong in source manifests/locks; avoid incidental upgrades.
- Do not reshape the Go package topology: `internal/` hierarchy, ownership
  boundaries and dependency direction are owned by [Go ownership](docs/development/go.md)
  and enforced by `internal/archcheck` (`go test ./internal/archcheck/`).
  Never recreate a retired package name, add a forwarding/alias package
  between internal packages, duplicate a domain DTO, or add a cross-boundary
  import the arch test forbids. If a change genuinely needs new ownership,
  update the owning guide and the arch test in the same patch.
- Test the changed working path, demonstrated failures and relevant authorization or
  destructive-operation boundaries. Do not invent exhaustive hypothetical test matrices
  or new harnesses where existing tests suffice. Keep callers, generated browser assets
  and staging wired together.
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
| `bash scripts/check-oxfmt.sh` / `check-oxlint.sh` / `check-ts-complexity.sh` | TypeScript format, correctness lint, and cyclomatic-below-10 on browser-payload TS (see [typescript guide](docs/development/typescript.md)). |
| `bash scripts/check-ruff-format.sh` / `check-ruff.sh` / `check-py-complexity.sh` | Python format, correctness lint, and cyclomatic-below-10 on shipping Python (see [Python tooling](docs/development/python.md)). |
| `bash scripts/check-gofumpt.sh` / `check-staticcheck.sh` / `check-errcheck.sh` | Go format (gofumpt), staticcheck, and unchecked-error lint; linux analysis for the last two. |
| `bun run test:frontend` / `bun run test:forgejo` | Build browser assets and run the selected suite. |
| `bun run test:pages` | Native-page fixture checks; requires the authorized local Forgejo fixture. |
| `bun test tests/forgejo/cockpit-branding.test.ts` | Independent stock-Cockpit branding source/component checks. |
| `bun run check:source` | Broad Go, TypeScript, browser and Python source checks. |
| `bash scripts/check-native.sh ARCH CANDIDATE_DIR` | Verify a soda-build candidate artifacts directory; does not build, install or publish. |
| `bash scripts/build-native.sh ARCH` | Removed at B6; use `tools/soda-build`. |

`ARCH` is `x86_64` or `aarch64`. Read the deployment/support guides before using
install/activation, provisioning, VM tools or `tests/installed/`; they can change
real state. Provisioning output contains sensitive password hashes.

## Task-specific documentation

Read the owning guide before changing its contracts, not this entire list.
Architecture owns product/security boundaries; feature guides own their details.
Service/image source in `appliance/services/` and `project-os/` establishes topology.

| Area | Guide |
| --- | --- |
| Documentation map and authority | [docs/README.md](docs/README.md) |
| Go package ownership and style | [Go ownership](docs/development/go.md), [package convention](docs/development/go-packages.md) |
| Product scope and ownership | [Architecture](docs/architecture/overview.md), [Spaces](docs/product/spaces.md), [Scope](docs/product/scope.md) |
| Runners and Tailnet | [Runners](docs/reference/runners.md), [Networking](docs/architecture/networking.md) |
| Forgejo customization and UI | [Forgejo](docs/reference/forgejo.md), [Lit](docs/development/lit.md), [TypeScript](docs/development/typescript.md) |
| Python format/lint/complexity | [Python tooling](docs/development/python.md) |
| Project runtime and access | [Project OS](docs/reference/project-os.md), [Terminal](docs/reference/terminal.md), [API](docs/reference/api.md), [Credentials](docs/reference/credentials.md) |
| Cockpit and operator setup | [Cockpit](docs/development/cockpit.md), [Operator setup](docs/guides/operator-setup.md) |
| Release | [Release architecture](docs/architecture/release.md), [Release workflow](docs/development/release.md) |
| Build, deployment and native tools | [Installation](docs/guides/installation.md), [Testing](docs/development/testing.md), [Native support](docs/development/native-support.md) |
| Browser screenshots | [Screenshot capture](docs/design/screenshot-capture.md); use `scripts/screenshot.ts`. |
| Local fixture access | [Local testing](docs/guides/local-testing.md) |
