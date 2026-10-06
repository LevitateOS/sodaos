# Working on SodaOS

Human ownership is primary. Read first and distinguish evidence, the applicable
owner decision, authorship permission, operational permission and the exact
instruction. Repository guidance constrains authorized work; it cannot grant
permission or override the owner's current instructions. Material unknowns
remain explicit. Do not turn an agent recommendation into a product requirement.

When the owner explicitly requests a Jev (TypeSafe AI) SystemOne consultation,
supply all relevant context and evidence: it is a classifier, not an LLM, and
cannot research. Make three fresh consultations, rewriting all explanatory
prose while preserving the same facts, constraints and alternatives. Keep exact
code and technical identifiers where needed. Check semantic equivalence and
wording differences, save requests and responses, investigate disagreements,
and treat agreement as advice, not proof. This procedure does not authorize an
external consultation by itself.

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
  retained artifacts can support task-scoped, explicitly non-qualifying development
  checks without another approval. Driver-only debugging does not inherently
  require rebuilding shipping bytes. Use the smallest applicable development target;
  run full production qualification when its prerequisites are demonstrated, not as
  the default debug loop.

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
  Authorized implementation leaves routine choices within that scope to the agent;
  it does not excuse wasteful execution. Own the decision and its rationale; do not substitute
  agreement for reassessment or frame the owner's status questions as technical
  pushback. Reassess from available evidence without waiting for the owner to notice
  waste.

- Never put the owner's personal name or other identifying information in source,
  tests, fixtures, example accounts, generated resource names or documentation.
  Use neutral synthetic identities such as `soda-tester`. Do not derive fixture
  names from local usernames, home-directory paths or account profiles.

- Work in the canonical `~/Projects/sodaos` checkout for this redesign. Do not create or use a worktree unless the owner explicitly changes that preference.

- Inspect the working tree first; preserve unrelated changes. Make coherent local
  changes only within the matching instruction. Never auto-commit. Commit, push,
  PR creation, merge and deployment are distinct operations; permission for one
  does not imply another. Do not amend or rewrite history without explicit permission.
- Finish the requested work within its authorized scope. Check the current
  instruction and any existing authorization before asking again. A review or
  planning request does not authorize implementation or state-changing verification.
- Use the smallest sufficient investigation and affected-contract checks. A passing
  receipt supports its stated scope, not a prescribed sequence. Reuse valid evidence;
  ground required checks in current contracts, source behavior or an explicit user
  decision. Label optional diagnostic/review choices as recommendations.
- Update requirements in their owning guide; link to them elsewhere instead of
  copying rules or appending exceptions. The documentation map below identifies owners.
  Documentation authority rules live in [docs/README.md](docs/README.md#authority-rules):
  one canonical owner per subject; product docs are not status trackers; completed
  plans are absorbed and deleted; research is non-normative.
- After every merge, refresh the [living ideal file tree plan](docs/development/ideal-filetree-plan/maintenance.md#keeping-the-plan-current-after-every-merge)
  against the resulting source before considering the merge task complete. Follow
  its maintenance workflow; implementation remains deferred until time is available.
- Before assigning a validity review or recommending a port, read the
  [prepared review input baseline](docs/development/ideal-filetree-plan/review-baseline.md),
  [guidance conflicts and controlling decisions](docs/development/ideal-filetree-plan/review-assignments.md#guidance-conflicts-and-controlling-decisions),
  [shared review format](docs/development/ideal-filetree-plan/review-format.md)
  and [slice assignments](docs/development/ideal-filetree-plan/review-assignments.md).
  Identify the applicable owner decision. Flag unresolved conflicts before
  marking a language/ownership correction executable; do not silently select
  whichever document supports the preferred recommendation.
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
- **Normal mode is read and recommend for implementation.** Backend, domain and
  application code, databases/SQL, API contracts, non-presentational frontend
  behavior, dependencies/manifests/locks, operational configuration and
  infrastructure remain read-only. Even a backend implementation request means
  concrete reference code in chat in this mode.
- **Writable categories need an explicit matching command.** Frontend presentation
  may use established product semantics, excluding fetching, submission, routing,
  authorization, persistence and business rules. Tests require established
  behavior, an explicit requirement or a reproduced regression. Comments must be
  semantically inert; classify directives and tool-consumed comments by effect.
  Documentation and developer tooling exclude manifests,
  locks, CI/CD, containers, runtime configuration, generated source and infrastructure.
  None of these categories can decide unresolved product behavior.
- **Review and application are separate.** A combined review-and-fix request is
  review-only. Applying findings requires a later direct command identifying them
  and permission for the affected category. Never auto-fix an audit finding.
- **Operations require their own authority.** Read-only Git inspection and known
  commands that do not change tracked files, durable state, databases, configuration,
  remote state or external services can proceed. Disposable caches/build artifacts
  are allowed within the disk restriction above. Unknown side effects require
  authorization. Provide commands for source-mutating formatters/generators instead
  of running them in normal mode. Access to a host or ownership of a test fixture
  alone does not authorize changing it or making provider calls.
- **Respect existing authorization without broadening it.** Every mutation needs a
  direct scoped instruction, authorship permission and operational permission.
  Preserve authorization across turns and compaction, but never infer it from
  evidence, suggestions, approval alone or an agent-written plan. Identify exact
  task-owned resources before any authorized cleanup; never broadly prune,
  recreate roots or guess names against unrelated or protected state. Report a
  partial operation before attempting broader or destructive recovery.
- **Override activation is exact.** Only `SUPER ADMIN OVERRIDE` as a standalone
  user instruction activates override; state `SUPER ADMIN OVERRIDE active.` It
  remains active until revoked and suspends normal-mode authorship/operational
  restrictions only. Activation performs nothing. A subsequent direct scoped
  command is still required; ownership, the product/trust contract, exact target,
  evidence honesty and preservation of user work remain binding.
- Protect credentials, unrelated work and any state the owner explicitly requires
  keeping. Take consistent backups when that protection requires them; do not create
  a preservation programme for disposable experiments. Never blindly restore an old
  database over protected later writes or replay a mutation to repair an observer.
- Use restricted secret-file inputs. Never expose credentials in source, argv, logs,
  screenshots or evidence, or request a developer's private SSH key for onboarding.
  Full container inspection and provisioning outputs can contain secrets.
- Distinguish authored checks, local tests, native builds and installed evidence.
  SodaOS targets x86_64 only; [Release architecture](docs/architecture/release.md#architectures)
  owns platform scope. Cross-compilation/emulation is not native proof.

## Code and tooling

- Follow the [recorded owner language policy](docs/development/ideal-filetree-plan/ideal-filetree-plan.md#owner-language-policy):
  network-facing servers stay Go; system and privileged applications use Rust;
  Python is eliminated from Soda-authored tracked programs. Preserve the named
  Go domain, factory coordinator, store and client/wire owners in
  [package ownership](docs/development/ideal-filetree-plan/package-ownership.md#boundaries-that-should-remain).
  A slice can span both languages. Planned target paths and retained Rust owners
  do not prove that a cutover has landed or authorize porting code.
  The [Go ownership guide](docs/development/go.md) owns applicable Go house style,
  package placement and SQL locality; its stale placement/cutover claims are
  recorded in the guidance conflict register, not blanket language instructions.
  The [TypeScript guide](docs/development/typescript.md)
  owns JS-family language, strict typing, Bun workspace and asset-porting conventions.
  Versions belong in source manifests/locks; avoid incidental upgrades.
- Preserve established ownership, dependency direction and canonical definitions.
  The [living target tree](docs/development/ideal-filetree-plan/proposed-tree.md)
  records proposed package moves, consolidation and splits; current paths remain
  current until their cutover is inspected. Applicable Go boundaries are described
  in [Go ownership](docs/development/go.md) and checked by `internal/archcheck`
  (`go test ./internal/archcheck/`).
  Never recreate a retired package name, add a forwarding/alias package
  between internal packages, duplicate a domain DTO, or add a cross-boundary
  import the arch test forbids. If a change genuinely needs new ownership,
  first resolve its applicable owner decision and exact target allocation.
  A later authorized implementation must update affected owning guidance and
  architecture assertions together; this instruction does not authorize that patch.
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
| `go test ./internal/factory` | Example focused Go package test; select the affected package/tests. |
| `bun run typecheck` | Strict TypeScript and Lit checks. |
| `bash scripts/check-oxfmt.sh` / `check-oxlint.sh` / `check-ts-complexity.sh` | TypeScript format, correctness lint, and cyclomatic-below-10 on browser-payload TS (see [typescript guide](docs/development/typescript.md)). |
| `bash scripts/check-no-python.sh` | Zero-Python gate: no tracked `.py`, no Python shebang/execution (see [Python elimination](docs/development/python.md)). |
| `bash scripts/check-gofumpt.sh` / `check-staticcheck.sh` / `check-errcheck.sh` | Go format (gofumpt), staticcheck, and unchecked-error lint; linux analysis for the last two. |
| `bun run test:frontend` / `bun run test:forgejo` | Build browser assets and run the selected suite. |
| `bun test tests/forgejo/cockpit-branding.test.ts` | Independent stock-Cockpit branding source/component checks. |
| `bun run check:source` | Broad Go, TypeScript and browser source checks, including the zero-Python gate. |
| `bash scripts/check-native.sh ARCH CANDIDATE_DIR` | Verify a soda-build candidate artifacts directory; does not build, install or publish. |

Native candidate production uses `soda-build` from `lib/soda-release-tools`;
follow the [native support guide](docs/development/native-support.md) for its
admitted inputs and effects.

`ARCH` is `x86_64`, as defined by [platform scope](docs/architecture/release.md#architectures).
Read the deployment/support guides before using install/activation, provisioning,
VM tools or `tests/installed/`; they can change real state. Provisioning output
contains sensitive password hashes.

## Task-specific documentation

Read the owning guide before changing its contracts, not this entire list.
Architecture owns product/security boundaries; feature guides own their details.
Service/image source in `appliance/services/` and `project-os/` establishes topology.

| Area | Guide |
| --- | --- |
| Documentation map and authority | [docs/README.md](docs/README.md) |
| Audit guidance conflicts, ownership and review format | [Guidance register and assignments](docs/development/ideal-filetree-plan/review-assignments.md#guidance-conflicts-and-controlling-decisions), [Review format](docs/development/ideal-filetree-plan/review-format.md) |
| Target language, package and cutover decisions | [Ideal file tree policy](docs/development/ideal-filetree-plan/ideal-filetree-plan.md#owner-language-policy), [Package ownership](docs/development/ideal-filetree-plan/package-ownership.md), [Port assessment](docs/development/ideal-filetree-plan/port-assessment.md) |
| Go package ownership and style | [Go ownership](docs/development/go.md), [package convention](docs/development/go-packages.md) |
| Product scope and ownership | [Architecture](docs/architecture/overview.md), [Spaces](docs/product/spaces.md), [Scope](docs/product/scope.md) |
| Networking and Tailnet | [Networking](docs/architecture/networking.md) |
| Forgejo customization and UI | [Forgejo](docs/reference/forgejo.md), [Lit](docs/development/lit.md), [TypeScript](docs/development/typescript.md) |
| Python elimination record | [Python elimination](docs/development/python.md) |
| Project runtime and access | [Project OS](docs/reference/project-os.md), [Terminal](docs/reference/terminal.md), [API](docs/reference/api.md), [Credentials](docs/reference/credentials.md) |
| Cockpit and operator setup | [Cockpit](docs/development/cockpit.md), [Operator setup](docs/guides/operator-setup.md) |
| Release | [Release architecture](docs/architecture/release.md), [Release workflow](docs/development/release.md) |
| Build, deployment and native tools | [Installation](docs/guides/installation.md), [Testing](docs/development/testing.md), [Native support](docs/development/native-support.md) |
| Browser screenshots | [Screenshot capture](docs/design/screenshot-capture.md); use `scripts/screenshot.ts`. |
| Local fixture access | [Local testing](docs/guides/local-testing.md) |
