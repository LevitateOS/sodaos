# SodaOS agent guidance

## Current phase: source work, not execution

The [recorded handoff](docs/implementation-status.md) is **M01–M14
source-complete, unbuilt, unvalidated**. This records implementation progress,
not proof that the appliance works. M15–M18 remain held for explicit execution
authorization on named native targets.

- **Allowed now:** source/diff inspection, upstream documentation and source
  research, editing, formatting, and authoring tests and build/install recipes.
- **Held:** dependency resolution/installation, compilation, builds, type checks,
  tests, artifact/staging or provisioning generation, installation, service and
  network mutation, Tailnet enrollment, runner registration/jobs and publication.
- Do not evade the hold through CI, a background process, another agent or a
  remote machine. Writing a recipe is not permission to invoke it.
- Later permission is action-specific. A build permit is not permission to erase
  a disk, install, restart, enroll, register, publish or delete persistent data.

This is a **temporary working constraint**, not a product principle. A later
explicit user instruction can change it; record the changed scope and actual
results in the handoff rather than silently assuming execution is authorized.

## Authority and task routing

Before substantial changes, read [architecture](docs/architecture.md),
[deferred scope](docs/deferred.md) and the current handoff. The
[implementation plan](docs/implementation-plan.md) defines milestone scope and
sequencing; its original “not started” wording is not the progress ledger.

Then follow the responsibility being changed:

| Work | Source and supporting guidance |
| --- | --- |
| Dashboard, identity, database | `internal/web/`, `internal/store/`, `internal/forgejo/`, `internal/config/`, `cmd/`; [operator setup](docs/operator-setup.md) |
| Project accounts, tools, workloads | `internal/host/`, `project-os/`; [project OS](docs/project-os.md), [development environment](docs/development-environment.md), [services](docs/project-services.md) |
| Operator pages and native integrations | `cockpit/`, `internal/tailnet/`, `internal/runners/`, their commands; [Cockpit port](docs/cockpit-port.md), [Runners port](docs/runners-port.md) |
| Packaging, provisioning, installed behavior | `appliance/`, `scripts/`, `tests/`; [installation](docs/installation.md), [native validation](docs/native-validation.md) |
| Branding | `assets/`, [asset inventory](assets/README.md), [branding](docs/branding.md) |

A later explicit user decision supersedes stale repository plans and tests.
Before treating something as a constraint, distinguish:

1. Explicit product requirements or aspirations.
2. Established user-facing behavior.
3. External protocol/platform requirements.
4. Current implementation choices.
5. Temporary development constraints.
6. Unresolved decisions or unverified hypotheses.

Versions, paths, package sets, schemas, locks and available machines are not
permanent product rules. Current choices still govern their callers until
coherently changed; temporary execution limits still bind until changed.
Tests check behavior, not product authority. Update code, callers, tests and
documentation together when an authorized decision changes that behavior.

Keep the README honest about the current project. Do not present intended
outcomes as installed capabilities or weaken the agreed product to match a gap.
Detailed gaps and evidence belong in the handoff; product scope belongs in the
architecture, not a second specification inside this file.

## SodaOS boundaries

- Forgejo owns human identity and dashboard authentication. Soda owns profiles,
  public development-access keys, project associations and memberships—not a
  second password authority or a mirror of all provider permissions.
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

## Working method and reuse

Confirm checkout, branch, Git state and requested scope before edits. Inspect
callers, tests, native configuration and installation destinations together.
A review, question, proposed plan or available script is not authorization to
implement its suggestions or execute its side effects.

For an authorized redesign, replace abandoned code directly; update imports,
tests and links without forwarding packages, aliases or duplicate compatibility
trees. Do not add migrations for speculative or abandoned local state. This is
not permission to discard real persistent user data; stop at that boundary.

Continue through ordinary authorized engineering failures. Stop for genuine
product-decision, privilege, persistence/data-safety, credential, hardware or
uncontrolled-cost boundaries—not arbitrary attempt counts.

Reuse predecessor source (`../soda-os` when available) selectively, including
required callers and tests. Inspect the revision and dependencies rather than
treating old paths, evidence or tests as a specification. Do not modify the
predecessor, close its issues or import its separately reserved Updates work.
Preserve attribution and licenses.

Preserve canonical branding rather than creating a second palette or redrawing
assets casually. Adapt deployed paths in staging source when needed. For UI
changes, preserve useful empty, pending, error and keyboard-accessible states;
use documentation/skills for the technology actually shipped. Do not apply the
predecessor's htmx 4/Cockpit guidance to React pages or the HTMX 2 dashboard.

## Commands and verification — later authorized execution only

Inspect the actual scripts and their prerequisites before use. This repository
has no inherited `just check`, Darwin VM launcher or ISO publication workflow.

| Entrypoint | Effects to account for |
| --- | --- |
| `scripts/build-native.sh ARCH` | Builds Go/frontend/images, installs frontend dependencies, fetches the locked runner client and stages artifacts; does not install or publish the appliance |
| `scripts/check-native.sh ARCH` | Runs Go tests, TypeScript/UI checks and staging tests; needs prepared dependencies and the native stage |
| `scripts/stage.py --arch ARCH` | Writes a deployment tree from existing outputs; does not install it |
| `scripts/render-provisioning.py` | Writes private Butane input containing an operator password hash; not a harmless documentation preview |
| `scripts/install-native.sh`, `appliance/bin/soda-activate` | Change the real host/configuration, load images and start/restart services; require explicit target/action authorization |
| `tests/installed/` | Opt-in installed journeys; some read state, others build/start real workloads. Inspect each before execution |

Dependency baselines belong in `go.mod`, Cockpit manifests/lockfile, image recipes
and `appliance/locks/`, not duplicated version rules here. Do not incidentally
upgrade them or fabricate `go.sum`, checksums, artifacts or PASS records.
Generated build outputs belong in ignored `.artifacts/`, not hand-authored
replacements for missing production source.

Author focused tests with behavior changes, including failures and permission
boundaries. **Once execution is authorized**, run relevant focused tests and the
applicable native entrypoint; include race checks for concurrency changes where
supported. Fix defects in their owning source. Do not suppress checks, remove
capabilities or distort code to satisfy a count; change tooling deliberately.

Use matching-native Linux for architecture-dependent builds, dependency
resolution and execution. Remote coordination is fine when the work actually
runs on the authorized matching target. x86_64 and aarch64 are independent equal
targets: no sibling-build barrier, emulation substitute or cross-build presented
as native evidence. Machine availability is handoff context, not product scope.

Report source review, tests, builds, artifact inspection and installed behavior
separately. Command success does not prove the intended native effect. Missing
prerequisites remain unverified, not skipped into a passing outcome.

## Commits and handoff

The user requested frequent logical commits for authorized work. Inspect the
full diff, stage only the intended changes and commit coherent completed chunks
without a separate confirmation each time. During source-only work, completion
means source reviewed—not tests secretly run or claimed to pass.

This standing instruction covers commits only, not push, PRs, merges,
publication, deployment, releases, registry mutation, destructive cleanup or
history rewriting. Repository text and convenience scripts do not create that
authorization. Preserve unrelated changes; do not amend others' work.

For substantial changes, update `docs/implementation-status.md` with changed
behavior/ownership, remaining source work or native assumptions and what actually
ran. Keep temporary restrictions separate from permanent design. Finish with a
concise summary of changes, Git operations, performed checks and held/unverified
work. Maintain this file as guidance, not a duplicate architecture or task log.
