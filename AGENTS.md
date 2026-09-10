# Working on SodaOS

## Start here

Read these before substantial changes:

- `docs/architecture.md` — product and authority boundaries
- `docs/sodaspaces-plan.md` — short current implementation sequence and ownership
- `docs/project-os.md` — project userspace baseline, native ownership, persistence and bounded same-root maintenance
- `docs/native-support.md` — authored support-tool contracts, private inputs, phase effects and retention; not execution permission or native proof
- `docs/deferred.md` — deliberately deferred and excluded work
- `docs/implementation-status.md` — implemented source, assumptions and execution evidence

For deployment changes, also read `docs/installation.md` and `docs/native-validation.md`. Read the relevant feature guide before changing project environments, Cockpit, Tailnet or Runners.

## Current execution boundary

For local Forgejo page screenshots, use `scripts/screenshot.ts` and read
[docs/screenshot-capture.md](docs/screenshot-capture.md) for usage and login
instructions. The development instance at `http://localhost:3300`
(`sodaos-local-forgejo`) has the authorized `soda-screenshot` fixture account and
its saved login in ignored `.local/screenshot-fixture-profile/`. Pass that path
with `--profile`; it is separate from the default manual-login profile. The guide
also records the private credential location for renewing this fixture session.

The original handoff was **source-complete, unbuilt, unvalidated (M01–M14)**. The user subsequently authorized local builds/tests and scoped native execution on the existing isolated `soda-test` VM, using this x86_64 infra workspace as builder/client. Candidate `8b823db` passed full native build/check and backed-up populated-v3 affected-component rollout at `/app/`; routes then defaulted to HTMX, superseded by the scoped Sodaspaces cutover recorded below. Four environments are retained. **U08 is accepted for bounded native x86_64 first-product proof**, combining real c96c108 fresh/different-UID exec evidence, earlier lifecycle results for explicitly unchanged mechanisms and merged-candidate regressions. This is not final-product/release acceptance. The missing console hook remains an explicit delivery gap; full operator/provider and aarch64 acceptance are pending. Both additional-fixture approvals and the original VM reboot have been used; no further fixture, lifecycle action, capability change or target is implied. Preserve every root, later write, private credential input and evidence. The private route serves infra, not the laptop automatically; old backups are not lossless rollback. See `docs/local-testing.md` and `docs/implementation-status.md` for exact bytes, evidence and remaining scope.

The user subsequently authorized local builds and automated tests for the U09
source work on this development machine. This does not authorize deployment,
repository/provider mutations or changes to retained projects. See the handoff
for the exact checks actually run.

The user also explicitly authorized a new isolated local Sodaspaces validation
fixture on this development machine. Its state/evidence is retained under
`.artifacts/local-sodaspaces-31e73bf/`: rootless stock Forgejo/Caddy plus the built
Go backend, synthetic users/repository/OAuth client and fixture-only TLS/browser
trust. Fixture initialization and authentication/browser checks are authorized;
no builder appliance installation, retained VM/project mutation, global trust or
host-network/security-policy change, unrelated provider mutation or cleanup is
implied. See `docs/implementation-status.md` for actual execution and failures.

The user subsequently gave standing implementation/testing approval for this
planned work. Native x86_64 build/check/export and the new isolated delivery fixture
at `.artifacts/delivery-ee8091a/` passed for `ee8091a`; the build/worktree/export are
retained under `.artifacts/stage-validation-ee8091a/` and `.artifacts/worktrees/`.
Step 3's bounded read-only exit is satisfied. Step 4's explicit access actions now
have local source/test coverage and subsequent bounded native proof as recorded below. Routine local implementation/testing does not need repeated permission.
Preserve retained data, credentials, fixtures and evidence; standing testing approval
is not an instruction to erase them or silently perform appliance cutover.

The user then explicitly requested phases 5 and 6. Candidate `bdbce8e` passed bounded
native x86_64 build/export and real create/key/join/Copy/SSH validation on the new
`soda-native-spaces-658f2af` CoreOS/KVM fixture, retained under
`.artifacts/access-vm-658f2af/`. Its separate client bridge namespace is not laptop
routing. Phase 6 covers preserved-state rehearsal before coordinated affected-component
maintenance on retained `soda-test`; no project lifecycle, unrelated provider/network
change, deletion or blanket appliance upgrade is included. Copied-state rehearsal
passed, followed by explicit live-cutover approval. `soda-test` now serves native
Sodaspaces with built `bdbce8e` affected artifacts and schema v5. Native private-page
browser checks and all seven existing memberships' SSH/PTY checks passed from infra;
all four roots, memberships, credentials and unchanged native components were preserved.
See the handoff and `.artifacts/cutover-c007eb6/` for exact bytes, fresh backups and
remaining scope. No further deployment, lifecycle or target is implied.

The subsequent integrated E2E request authorized build/check and scoped native
execution on `soda-native-spaces-658f2af`. Backed-up affected-component delivery,
real browser terminals, Stop/Start persistence, temporary-key rotation/revocation
and fresh Create/Join/SSH checks passed there. Two projects are now retained in
that fixture; originals and all later writes/keys/evidence must survive. The
client used management forwarding, not laptop routing. See the leading handoff
for exact tested bytes, failed probes and the public-mode packaging correction.
This does not authorize further targets, deletion or a `soda-test` rollout.

The user subsequently requested the remainder of Lit step 6, including deployment.
Fresh native x86_64 build/check/export passed for `5a21ad6`; artifacts and three
failed preceding attempts are retained under `.artifacts/step6c-*/`. Read-only
preflight contacted only the existing isolated validation VM. The user then selected
both targets, validation first, and approved starting the offline existing VMs.
Both are now running their original disks/projects. Fresh backup/copied-state rehearsal
and affected `5a21ad6`/schema-v6 delivery passed on the isolated validation VM, not
whole-product acceptance. The reproduced tmux startup flush/input-loss correction
then passed fresh `b8af68c` build/check/export and backed-up paired helper/program
same-root delivery there; the full native terminal boundary and two-actor, two-project
six-session matrix passed, including independent cleanup of all twelve named Ends.
`soda-test` subsequently received matching b8af68c/schema-v6 affected components after
fresh paired application/root backups and copied-state rehearsals. Its four original
Rocky 9 roots received only signed tmux, the exact program and terminal-directory init
line; accounts, keys, homes, shared data, legacy agent socket inodes, original CIDs and
all 24 original system-process PID/start pairs were preserved. Native private-browser
checks/BFCache and all seven memberships' SSH/PTY/tmux-version checks passed there.
Both targets retain all roots/later writes; only future creation uses the loaded
Rocky 10.2 image. Probe-only viewport/logout/foreground/HTTPS corrections and failed
attempts remain recorded. CLI/provider, physical-keyboard, broader native and aarch64
acceptance remain unrun, not waived. No new target, lifecycle action, cleanup,
capability change or borrowed CLI credentials follow. See the leading handoff.

This is not blanket authorization for other targets, host-network changes, provider resources or destructive lifecycle checks. Outside the recorded local execution scope, until explicit authorization:

- Edit source and author tests, configuration and build/install recipes.
- Source inspection, upstream source/metadata research, formatting and Git operations are allowed.
- Do not run builds, compilation, type checks, tests, dependency resolution/installation, generated provisioning or native validation.
- Do not install/restart services, enroll Tailscale, register runners, execute provider jobs, publish artifacts or enable automatic CI.
- Do not describe authored tests or inspected source as passing runtime evidence.

Later authorization is action-specific: permission to build is not permission to install, erase disks, restart an appliance, register provider resources or delete data. Update the handoff with actual evidence when an authorized phase occurs; do not silently treat this hold as lifted.

## Implemented topology — do not confuse it with terminology

- Host: Fedora CoreOS with native rpm-ostree package layering, initially exercised on the isolated x86_64 test VM; full product compatibility remains unvalidated.
- Cockpit, `tailscaled`, the project helper and CI runner services run natively on the host.
- Forgejo is currently a **standalone Podman container**, not a Podman pod. Dashboard and Caddy are separate containers too.
- Project environments are persistent Rocky + mise containers with project-local accounts and mutable writable roots.
- The dashboard has its own SQLite database. Forgejo maintains its own persistent data/database.
- Project workloads use the implemented nested Podman candidate. Do not assume it is proven or claim that a fallback already exists.

Use the actual files in `appliance/services/` and `project-os/` as implementation references. Keep documentation consistent when changing deployment structure.

## Product and security boundaries

- **Use Forgejo's official template overrides for Forgejo-owned workflows.**
  Native server-rendered pages/handlers/authentication are selected, with custom
  shell/navigation/assets; the earlier all-React/no-Forgejo-HTML requirement is
  superseded. See `docs/sodaspaces-plan.md`.
  Use Forgejo's native frontend throughout, with the selected Sodaspaces repository
  button/right-drawer addition, not a new repository tab. Drawer view/session tabs
  are selected. A global Spaces link and bounded Soda-owned Go/template listing page
  now have a local source implementation; the shell uses Soda-owned Go HTML, canonical
  assets, fixed native links and a labelled Soda account, not fabricated native
  authentication/notification context. [Lit](docs/lit.md) now owns management and
  terminal-control rendering and the shared multi-session page/drawer. Bounded v2
  layout, measured stable panes, shared chrome/compact projections and current journey
  source ports, observed attention and extended six-session driver fixtures have local
  coverage. Bounded x86_64 native matrix/delivery proof is recorded; broader
  native/CLI acceptance remains pending. The [implementation sequence](docs/lit-migration-plan.md) owns that work. No Bootstrap UI is selected.
  Root `dashboard/`, original Go/HTMX pages/forms/assets and duplicate forge
  adapters are removed; retain the Go API environment/access backend,
  OAuth/security/native integration and separate
  Cockpit React/PatternFly pages. Current hooks mount one complete management/
  terminal component through a non-modal aside; the historical duplicate API
  caller is removed. Managed tmux now preserves exact native sessions across
  attachment loss; same-target Refresh and focus/view/Hide preserve the live mount.
  Isolated `22d8591` has bounded same-shell reload/cleanup evidence, not full native
  safety/UX acceptance. Lit ports must preserve `restore`, finite retain/Return and
  HTTP End; document disposal detaches, not End. The ID-keyed backend, exact
  correlation/cleanup receipts and bounded authorized collection now have local
  Go/race/browser coverage; the multi-session UI also has local browser coverage,
  with bounded native concurrency/cleanup proof now recorded in the handoff. Do not revive the singleton metadata endpoint or select a newest session.
  Do not force ordinary SSH/tmux into Soda, auto-create on reconnect or call local
  rendering checks native process proof. Follow the current plan and terminal guide.
  The read-only hook/drawer/context caller passed
  its native x86_64 build/stage/export and isolated exported-payload browser checks;
  explicit mutation controls have local coverage and bounded native create/key/join/Copy/SSH proof on the separately recorded fresh fixture, not retained-appliance cutover. Source has same-origin `/-/soda/` routing/scoped cookies,
  expected-actor API guards, schema-v5 login cancellation and OAuth repository/actor
  context. The actor hint is not native-session authentication. No backend fork/rebuild, iframe, scraping, borrowed
  cookies or replacement password/permission authority is implied. Template and
  asset compatibility still need review/tests; preserve working login and native
  protocols. Retained operator Cockpit remains a separate selected boundary.

- Forgejo is upstream. Do not take over its business rules, data, permissions or
  administration, or access its database directly. Supported customization is
  described in `docs/forgejo-frontend-integration.md`. The fork-specific preparer
  and proposals have been removed; Git history retains them. The source-backed
  historical audit/plans remain in Git at `9f3baa7`, not an active feature checklist.
  **Architecture lesson:** the assistant wrongly promoted an API-only preference
  into a requirement. Verify official extension points and exact upstream source
  before declaring a limitation; missing JSON is not missing native functionality.
  **Forking Forgejo is an architectural failure path, not an implementation
  option.** A downstream source patch set/custom executable counts as a fork
  even if called a small adapter or API extension. If a requirement appears to
  need one, stop that approach, explain the exact limitation and revisit the
  architecture with the user. Do not proceed under previous patch/build plans.
  First use supported configuration, themes/assets, template extension points/
  overrides, native workflows/protocols and existing APIs/integrations. Official
  customization is not a fork; bypassing native security is not an alternative.
  Upstream contributions require separate scope and must not become a permanent
  downstream dependency disguised as upstream work.
  The unified frontend includes Forgejo developer **and administrator** views;
  those views delegate through supported upstream interfaces, not a replacement
  Soda forge/administration backend.
- Soda's backend is an environment/access extension plus the necessary web/API
  adapter. Forgejo owns identity and authentication; Soda owns only its additional
  profile/preferences data, development-access public keys, environment
  associations and memberships—not a second password authority or provider-role
  inventory. Forgejo Git key management remains upstream-owned.
- Developers have Linux accounts **inside projects**, not human host accounts.
  The repository's human owner administers its project, not the appliance.
  Native root, the configured dashboard operator and arbitrary Forgejo site
  administrators are not interchangeable authorities.
- **Add me to this project** must reach real account/key provisioning. A row,
  mock, unwired helper or manual-command checklist does not complete the feature.
  Joining and native Git authorization are separate; do not promise automatic
  later key/revocation synchronization.
- Shared resources mean actual shared installed tools, files and services, with
  ordinary personal checkouts. Keep Git, mise and workload operations native;
  Soda does not interpret merges as live-state promotion or cleanup.
- Preserve stock operator Cockpit and the retained Tailnet/Runners pages,
  including their backing logic, dependencies and focused tests. Providers own
  CI scheduling, workflows, registration authority and results; Soda owns local
  capacity, not a scheduler or developer workspace UI in Cockpit.

The current **source implementation**, not an immutable deployment prescription:

| Placement | Current mechanism |
| --- | --- |
| Host | Fedora CoreOS candidate with rpm-ostree layering; native Cockpit, `tailscaled`, project helper and CI runner services |
| Appliance applications | Separate Podman containers for Forgejo, the dashboard and Caddy; Forgejo is **not currently a Podman pod** |
| Projects | Persistent Rocky + mise containers with project-local accounts and writable roots; nested Podman has only bounded native x86_64 evidence |
| Application code | Go API/OAuth/setup/native integration; native Forgejo frontend; TypeScript/React Cockpit pages |

Use actual service/image/configuration source to establish details. A pod groups
containers; it is not itself a Linux user database, init system or filesystem.
Do not infer full runtime compatibility from bounded evidence or describe the
unused host fallback as implemented. Introduce Rust only for a concrete need and an agreed responsibility.

## Human-maintainable engineering

**Upstream-first implementation is a requirement.** Do not build simplified
imitations of established, mature upstream implementations. Use their supported
configuration, APIs, extension points and native workflows before writing a Soda
replacement. A shorter implementation, an inconvenient interface or an assumed
missing feature is not a justification for taking over their responsibility.
This applies during implementation and review, not only in a future audit.

Prefer a coherent system a person can understand, modify and remove:

1. Delete dead or duplicated decisions rather than wrapping them.
2. Use one direct representation instead of synchronized copies of authority.
3. Separate genuinely independent responsibilities through explicit inputs and
   outputs; keep concrete owners and callers visible.

Do not move branches into arbitrary helpers, parameter bags, assertion utilities
or vague `common`/`utils`/`services` packages to satisfy structural metrics.
Small interfaces for external commands, HTTP and test injection are useful;
speculative frameworks and interchangeable-backend scaffolding are not. Add
machinery for a current requirement, external contract, reproduced failure or
concrete correctness/data-loss concern—not imagined future consumers.

Keep both sides of the boundary: do not take over upstream mechanisms, and do
not simplify Soda into a catalog that leaves developers to assemble its missing
integration. A bounded adapter may mutate native state and Soda's legitimate
records without requiring durable jobs, copied permissions or reconciliation.

Before adding an adapter, inspect the exact selected upstream version and
configuration and the actual Soda caller. Identify what upstream already owns,
the concrete Soda requirement it does not supply, and the smallest integration
needed between them. Separate protocol requirements, defaults, optional capabilities,
packaging conventions and hypotheses. Prefer deleting proven duplication over
wrapping or renaming it. If a mechanism fails, identify its actual constraint;
do not silently change the product or invent a new subsystem.

Audit responsibilities across their complete call chains, including internal
packages and native helpers. A package boundary does not establish independent
ownership. Compare current code against actual upstream behavior before labelling
it redundant; required authorization, persistence and user-facing integration
must survive any simplification. Follow the source-backed
[upstream-first review](docs/refactoring-plan.md#1-upstream-first-review).

Implement one source-backed candidate. Investigate project-scoped host workloads
only after a concrete nested-runtime blocker. Do not add dormant backends, a
privileged-parent shortcut, an unrestricted host socket or a VM substitute to
hide uncertainty. Bring a genuine architecture gap back for a decision.

[Deferred work](docs/deferred.md) is not permission to omit ordinary validation,
authorization, error handling or persistence. Nor is it an invitation to add
private-resource branching/selectors, identity-remapping policy, generalized
recovery, project deletion/archival or a new release/update platform.

## Security, networking and data

- Enforce authorization server-side. Resolve project ownership and native
  targets from trusted state, not hidden buttons or caller-selected privileges.
  The current Unix-socket helper is a fixed-operation boundary, not an arbitrary
  command endpoint or forwarding surface for host Podman flags.
- Preserve normal session, OAuth, CSRF and input protections. Trusted-team scope
  does not remove the operator/project boundary or justify weakening checks.
- Use restricted secret-file/input channels. Never expose real credentials in
  source, argv, tracing, terminal echo, logs, fixtures, screenshots or evidence.
  Full container inspection can contain secrets. Provisioning password hashes
  are sensitive too. Never request a developer's private SSH key for onboarding.
- Project access is ordinary `user@project-ip`, SSH/SCP/SFTP—not project DNS or a
  custom SSH gateway. A bridge address is not proof of client reachability.
  Host Tailnet enrollment does not establish an advertised/approved subnet route.
- Keep browser/OAuth origins, service listeners and Forgejo Git advertisement
  distinct. Do not reconstruct endpoints from predecessor ports or an unrelated
  browser hostname. Cockpit is loopback-first and operator-only; do not silently
  expose administration or development services publicly.
- Normal project startup starts the existing container. Preserve accounts,
  homes, SSH host keys, installed tools, shared files, configuration and service
  data. Do not use `--rm`, `--replace`, pruning or deletion as a repair shortcut.
- Clean up only explicitly authorized, exact run-owned resources. Preserve
  operator inputs, unrelated work and evidence; a disposable fixture does not
  make its surrounding host, project or credentials disposable.

## Scope discipline

**Ownership:** `docs/sodaspaces-plan.md`, production callers and the API/credential guides govern product behavior and shared build/config/staging contracts. `internal/host/` and `project-os/` are production integration, not outside harness code. Support tools may invoke product-owned tests and hand off evidence, not copy scenarios or create a second readiness gate. Optional native media and unfinished helper ports do not block work using existing authorized tools. Old M/U/P plans and the forge API register are retired; historical labels remain only for existing evidence/tool contracts, not a progress tally.

Keep ordinary Git, mise and container workflows. Shared resources mean actual shared files, installed tools and services—not just a shared download cache.

Do not add managed private toolchain/service branching, selectors, process switching, merge-triggered promotion/cleanup, generalized identity remapping, reconciliation, recovery or project deletion/archival machinery. Consult `docs/deferred.md` rather than expanding scope to handle every hypothetical case.

Investigate a project-scoped host workload fallback only after a concrete nested-runtime blocker. Do not implement both backends speculatively or substitute unrestricted host access, a privileged parent or a VM backend without revisiting the design with the user.

## Source conventions

- Go for the dashboard/backend, setup commands and privileged integration. Do not introduce Rust without a concrete need and an agreed responsibility.
- Native Forgejo frontend plus Sodaspaces is selected. Root `dashboard/`, its React/PatternFly/Vite+/Zustand support and duplicate forge adapters are removed. The original Go/HTMX frontend is also removed, including its form routes/assets/clients. The Go command retains legitimate Soda APIs, OAuth/encrypted grants and native integration. Root and ordinary OAuth return to configured native Forgejo. The bounded Spaces Go/template page now serves Soda-owned HTML at `/-/soda/spaces`, not a replacement Forgejo frontend or SPA. Its fixed transaction-bound OAuth return uses schema v6; the shared page/drawer multi-session workspace has local source/browser coverage. Follow the leading plan and [Lit sequence](docs/lit-migration-plan.md). Management and terminal controls use Lit; xterm/transport remain imperative. Bounded v2 layout, measured panes, compact projections and journey source ports, observed attention and extended driver fixtures have local coverage and bounded native matrix proof; broader native/CLI acceptance remains pending. No rollout follows from local source checks. Preserve native boundaries and follow [the authoring guide](docs/lit.md). Keep Cockpit's separate React/PatternFly frontend, dependencies and native boundaries. Installed `soda-test` now uses matching `b8af68c` affected artifacts and schema-v6 grants after approved preserved-state maintenance; four original Rocky 9 roots are retained with bounded tmux/program/init additions, while only future creation uses the candidate Rocky 10.2 image. Bounded native browser/access checks passed, not whole-product acceptance. See the leading plan and handoff; only bounded U08 is accepted.
- Prefer native configuration and small bounded helpers over new orchestration frameworks.
- Author focused tests with behavior changes, including failure/authorization paths; execution remains subject to the phase boundary.
- Keep build and staging paths consistent with their actual callers. Generated outputs belong in ignored `.artifacts/`; private local inputs belong outside tracked source.
- Do not fabricate `go.sum`, dependency checksums, binaries or validation records. Real Go metadata was resolved during the first native x86_64 build; review intentional dependency changes.
- Both native x86_64 and aarch64 are targets. Do not add a sibling-build barrier or call cross-compilation/emulation native installed evidence.

## Bun and TypeScript conventions

- Use the root Bun workspace and single `bun.lock`; root `package.json` owns the
  Bun pin and shared tooling. Cockpit keeps its UI dependencies and Vite+ build.
- Follow [TypeScript development](docs/typescript.md). New authored JS-family code
  is `.ts`/`.tsx`; keep browser, tooling/test and Cockpit compiler boundaries clear.
- Pass `bun run typecheck` with strict checking, checked indexed access and exact
  optional properties. Do not reintroduce unchecked JavaScript compiler inputs.
- Keep types with their owner, use `import type`, and validate untrusted data from
  `unknown`. No blanket `any`, disabled checks or unchecked casts to evade errors.
- Port callers/tests with each source group. Generate browser JS into ignored build
  output and wire staging/previews before changing served sources. Preserve upstream
  assets and the current explicit installed-test authorization gates.

## Retained source and assets

Preserve both Cockpit pages **and their backing logic/tests**, not just their appearance, until an explicitly coordinated replacement. The user subsequently selected moving Soda's runner capacity/service configuration into operator-only settings in the unified native SodaOS/Forgejo interface. This is remaining work: inspect official administrator extension points, reuse the backing logic/tests and keep the Cockpit Runners page until its replacement works. Tailnet remains in Cockpit. Do not revive a standalone dashboard or equate Forgejo site administration with Soda operator authority. Providers own CI workflows, scheduling, registration authority and results; Soda manages local capacity.

The predecessor repository is separate. Do not modify it, close its issues or import its separately reserved Updates platform as part of this work. Preserve attribution and licenses. Canonical branding in `assets/` must not be casually regenerated or removed; adapt installation paths in staging source where necessary.

## Entrypoint effects — execution requires the applicable authorization

| Entrypoint | Effects to account for |
| --- | --- |
| `scripts/build-native.sh ARCH` | Resolves Go/frontend dependencies, builds native commands/project CLIs/images (including Tea's version execution), fetches locked inputs and stages artifacts; does not install or publish the appliance |
| `scripts/build-installer.py` | Builds native media commands, fetches/verifies the locked CoreOS ISO, adds the console with xorriso and embeds live Ignition with stock CoreOS Installer; writes fresh media/evidence only, never boots, writes a destination disk or deploys Soda |
| `scripts/check-native.sh ARCH` | Runs Go tests, TypeScript/UI checks, Python build-fixture and staging tests; needs prepared dependencies and the native stage |
| `scripts/stage.py --arch ARCH` | Writes a deployment tree from existing outputs; does not install it |
| `scripts/render-provisioning.py` | Writes private Butane input containing an operator password hash; not a harmless documentation preview |
| `scripts/install-native.sh`, `appliance/bin/soda-activate` | Change the real host/configuration, load images and start/restart services; require explicit target/action authorization |
| `scripts/test-vm.sh` | Starts the prepared KVM guest, opens SSH/tunnels or reads status/console; SSH commands may mutate the guest, so inspect the subcommand and target first |
| `tests/installed/` | Opt-in installed journeys; some read state, others build/start real workloads. Inspect each before execution |

Dependency baselines belong in `go.mod`, root/Cockpit manifests and root `bun.lock`, image recipes
and the lockfiles under `appliance/` and `project-os/`, not duplicated version
rules here. Do not incidentally upgrade them or fabricate `go.sum`, checksums, artifacts or PASS records.
Generated build outputs belong in ignored `.artifacts/`, not hand-authored
replacements for missing production source.

## Handoff and commits

Inspect the working tree before editing and preserve unrelated user changes. Keep commits coherent and frequent; do not amend/rewrite history without permission.

For substantial changes, update `docs/implementation-status.md` with changed
behavior/ownership, remaining source work or native assumptions and what actually
ran. Keep temporary restrictions separate from permanent design. Finish with a
concise summary of changes, Git operations, performed checks and held/unverified
work. Maintain this file as guidance, not a duplicate architecture or task log.
