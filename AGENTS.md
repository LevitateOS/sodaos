# Working on SodaOS

## Start here

Read these before substantial changes:

- `docs/architecture.md` — product and authority boundaries
- `docs/dashboard-plan.md` — selected frontend/ownership constraints and page/dependency inventory
- `docs/dashboard-implementation-plan.md` — leading core U01–U20 plan, including production native integration and product acceptance; conditional E01–E03 extensions
- `docs/native-porting-plan.md` — subordinate outside VM/SSH/evidence/artifact tools, provisioning support and retained host-operator integrations
- `docs/native-support.md` — authored support-tool contracts, private inputs, phase effects and retention; not execution permission or native proof
- `docs/deferred.md` — deliberately deferred and excluded work
- `docs/implementation-status.md` — implemented source, assumptions and execution evidence
- `docs/implementation-plan.md` — historical initial M01–M18 plan; not the new dashboard implementation sequence

For deployment changes, also read `docs/installation.md` and `docs/native-validation.md`. Read the relevant feature guide before changing project environments, Cockpit, Tailnet or Runners.

## Current execution boundary

The original handoff was **source-complete, unbuilt, unvalidated (M01–M14)**. The user's request to begin testing/start the host initiated native execution on `linux-infra.dimensionlab.net` (x86_64 builder) and a new isolated `soda-test` CoreOS KVM VM. Recorded M15 source/build checks passed; the local dashboard is activated and its operator OAuth/browser journey is verified, while full developer/project/workload journeys remain pending. Dashboard candidate `35df189` has since been built with pinned tools, migrated with private backup/isolated rehearsal, and installed at `/app/` on `soda-test`; its operator React OAuth/read/navigation/logout journey passed. The full x86_64 core build/stage sealed at `8417a90` after packaging fixes; its helper/project image are installed. Two retained developer fixtures now have real private repositories, project accounts/keys and memberships, with browser authorization checks passed. The user subsequently approved the private project tunnel: routed SSH/PTY/SCP/SFTP and sudo boundaries passed from infra. Shared tools, workloads, persistence and final product acceptance remain pending; the tunnel does not route a laptop automatically. Preserve these later writes; the old pre-migration backup is no longer a lossless rollback. See `docs/local-testing.md` and `docs/implementation-status.md` for actual evidence and limits.

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

- Forgejo is upstream. Do not take over its business rules, data, permissions or
  administration, fork its backend for UI parity, or access its database directly.
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
| Projects | Persistent Rocky + mise containers with project-local accounts and writable roots; nested Podman is the unvalidated workload candidate |
| Application code | Go backend/setup/native integration; Go + HTMX dashboard; TypeScript/React Cockpit pages |

Use actual service/image/configuration source to establish details. A pod groups
containers; it is not itself a Linux user database, init system or filesystem.
Do not call the nested runtime proven or describe the unused host fallback as
implemented. Introduce Rust only for a concrete need and an agreed responsibility.

## Human-maintainable engineering

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
configuration. Separate protocol requirements, defaults, optional capabilities,
packaging conventions and hypotheses. If a mechanism fails, identify its actual
constraint; do not silently change the product or invent a new subsystem.

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

**Plan precedence:** `docs/dashboard-implementation-plan.md` wins over `docs/native-porting-plan.md` for core behavior, API/auth/data/build contracts and product acceptance. Production `internal/host/` and `project-os/` are core, not outside harness code. Former P07/P08 developer/workload/persistence work belongs only to U08/U20. P tools can invoke those core-owned tests and hand off evidence, not copy scenarios or create a second readiness gate. Coordinate shared build/config/staging files by the core plan's ownership matrix. Optional native media and unfinished helper ports do not block core work using existing authorized tools.

Keep ordinary Git, mise and container workflows. Shared resources mean actual shared files, installed tools and services—not just a shared download cache.

Do not add managed private toolchain/service branching, selectors, process switching, merge-triggered promotion/cleanup, generalized identity remapping, reconciliation, recovery or project deletion/archival machinery. Consult `docs/deferred.md` rather than expanding scope to handle every hypothetical case.

Investigate a project-scoped host workload fallback only after a concrete nested-runtime blocker. Do not implement both backends speculatively or substitute unrestricted host access, a privileged parent or a VM backend without revisiting the design with the user.

## Source conventions

- Go for the dashboard/backend, setup commands and privileged integration. Do not introduce Rust without a concrete need and an agreed responsibility.
- The current dashboard is Go + HTMX. The user selected a client-rendered TypeScript/React + PatternFly + Vite+ + Zustand dashboard with a Go API next, without SSR, Tailwind or TanStack. See `docs/dashboard-plan.md` for the inventory and `docs/dashboard-implementation-plan.md` for the implementation sequence. The React preview/JSON adapters and schema-v3 encrypted grants are installed on `soda-test` at candidate `35df189`, with operator and first developer browser evidence; default browser routes remain HTMX, and full developer/native and cutover acceptance remain pending. Follow the handoff rather than treating partial source as completed milestones. Retain the separate Cockpit Tailnet/Runners frontend and native boundaries.
- Prefer native configuration and small bounded helpers over new orchestration frameworks.
- Author focused tests with behavior changes, including failure/authorization paths; execution remains subject to the phase boundary.
- Keep build and staging paths consistent with their actual callers. Generated outputs belong in ignored `.artifacts/`; private local inputs belong outside tracked source.
- Do not fabricate `go.sum`, dependency checksums, binaries or validation records. Real Go metadata was resolved during the first native x86_64 build; review intentional dependency changes.
- Both native x86_64 and aarch64 are targets. Do not add a sibling-build barrier or call cross-compilation/emulation native installed evidence.

## Retained source and assets

Preserve both Cockpit pages **and their backing logic/tests**, not just their appearance. Providers own CI workflows, scheduling, registration authority and results; Soda manages local capacity.

The predecessor repository is separate. Do not modify it, close its issues or import its separately reserved Updates platform as part of this work. Preserve attribution and licenses. Canonical branding in `assets/` must not be casually regenerated or removed; adapt installation paths in staging source where necessary.

## Entrypoint effects — execution requires the applicable authorization

| Entrypoint | Effects to account for |
| --- | --- |
| `scripts/build-native.sh ARCH` | Resolves Go/frontend dependencies, builds native commands/project CLIs/images (including Tea's version execution), fetches locked inputs and stages artifacts; does not install or publish the appliance |
| `scripts/check-native.sh ARCH` | Runs Go tests, TypeScript/UI checks, Python build-fixture and staging tests; needs prepared dependencies and the native stage |
| `scripts/stage.py --arch ARCH` | Writes a deployment tree from existing outputs; does not install it |
| `scripts/render-provisioning.py` | Writes private Butane input containing an operator password hash; not a harmless documentation preview |
| `scripts/install-native.sh`, `appliance/bin/soda-activate` | Change the real host/configuration, load images and start/restart services; require explicit target/action authorization |
| `scripts/test-vm.sh` | Starts the prepared KVM guest, opens SSH/tunnels or reads status/console; SSH commands may mutate the guest, so inspect the subcommand and target first |
| `tests/installed/` | Opt-in installed journeys; some read state, others build/start real workloads. Inspect each before execution |

Dependency baselines belong in `go.mod`, Cockpit manifests/lockfile, image recipes
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
