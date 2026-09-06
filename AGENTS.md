# Working on SodaOS

## Start here

Read these before substantial changes:

- `docs/architecture.md` — product and authority boundaries
- `docs/deferred.md` — deliberately deferred and excluded work
- `docs/implementation-status.md` — implemented source, assumptions and execution evidence
- `docs/implementation-plan.md` — milestone scope and later validation stages

For deployment changes, also read `docs/installation.md` and `docs/native-validation.md`. Read the relevant feature guide before changing project environments, Cockpit, Tailnet or Runners.

## Current execution boundary

The recorded handoff is **source-complete, unbuilt, unvalidated (M01–M14)**. M15–M18 are held until the user explicitly authorizes execution on named native targets.

Until that authorization:

- Edit source and author tests, configuration and build/install recipes.
- Source inspection, upstream source/metadata research, formatting and Git operations are allowed.
- Do not run builds, compilation, type checks, tests, dependency resolution/installation, generated provisioning or native validation.
- Do not install/restart services, enroll Tailscale, register runners, execute provider jobs, publish artifacts or enable automatic CI.
- Do not describe authored tests or inspected source as passing runtime evidence.

Later authorization is action-specific: permission to build is not permission to install, erase disks, restart an appliance, register provider resources or delete data. Update the handoff with actual evidence when an authorized phase occurs; do not silently treat this hold as lifted.

## Implemented topology — do not confuse it with terminology

- Host candidate: Fedora CoreOS with native rpm-ostree package layering, unvalidated.
- Cockpit, `tailscaled`, the project helper and CI runner services run natively on the host.
- Forgejo is currently a **standalone Podman container**, not a Podman pod. Dashboard and Caddy are separate containers too.
- Project environments are persistent Rocky + mise containers with project-local accounts and mutable writable roots.
- The dashboard has its own SQLite database. Forgejo maintains its own persistent data/database.
- Project workloads use the implemented nested Podman candidate. Do not assume it is proven or claim that a fallback already exists.

Use the actual files in `appliance/services/` and `project-os/` as implementation references. Keep documentation consistent when changing deployment structure.

## Product and security boundaries

- Forgejo owns human identity and dashboard authentication; Soda is not a second password authority.
- Soda owns profiles, public development-access keys, project associations and memberships.
- Developers have Linux accounts **inside projects**, not human host accounts.
- The repository's human owner administers its project environment, never the appliance merely by ownership.
- Native root/operator access and the explicitly configured Soda dashboard operator are separate from arbitrary Forgejo site administrators.
- Enforce authorization server-side, not merely through hidden navigation.
- Keep host operations behind the restricted Unix-socket helper. Never expose arbitrary commands, caller-selected host Podman flags or an unrestricted host engine socket.
- Secrets belong in restricted runtime files, not source, command arguments, logs, fixtures or screenshots. Never request private SSH keys for onboarding.
- Membership must follow successful native account setup; a database row is not proof of a usable environment.

## Networking and persistence

- Project SSH is direct `user@project-ip`, with ordinary SSH/SCP/SFTP. Do not add project DNS or an SSH gateway.
- The implemented bridge subnet needs actual client routing. Host Tailnet enrollment alone does not advertise/approve that subnet.
- Browser/OAuth origins, native listeners and advertised Forgejo Git SSH addresses are distinct facts. Do not reconstruct them from predecessor ports or assume an advertised address is listening.
- Cockpit is loopback-first and operator-only. Do not silently expose administration or development services publicly.
- Project containers are created once and started as existing containers. Preserve accounts, homes, host keys, tools, configuration and service data.
- Do not use `--rm`, `--replace`, pruning or container deletion as ordinary project startup or a shortcut around a failure.

## Scope discipline

Keep ordinary Git, mise and container workflows. Shared resources mean actual shared files, installed tools and services—not just a shared download cache.

Do not add managed private toolchain/service branching, selectors, process switching, merge-triggered promotion/cleanup, generalized identity remapping, reconciliation, recovery or project deletion/archival machinery. Consult `docs/deferred.md` rather than expanding scope to handle every hypothetical case.

Investigate a project-scoped host workload fallback only after a concrete nested-runtime blocker. Do not implement both backends speculatively or substitute unrestricted host access, a privileged parent or a VM backend without revisiting the design with the user.

## Source conventions

- Go for the dashboard/backend, setup commands and privileged integration. Do not introduce Rust without a concrete need and an agreed responsibility.
- HTMX for the developer dashboard; retain TypeScript/React for Cockpit Tailnet/Runners.
- Prefer native configuration and small bounded helpers over new orchestration frameworks.
- Author focused tests with behavior changes, including failure/authorization paths; execution remains subject to the phase boundary.
- Keep build and staging paths consistent with their actual callers. Generated outputs belong in ignored `.artifacts/`; private local inputs belong outside tracked source.
- Do not fabricate `go.sum`, dependency checksums, binaries or validation records. Initial Go resolution remains a later authorized step.
- Both native x86_64 and aarch64 are targets. Do not add a sibling-build barrier or call cross-compilation/emulation native installed evidence.

## Retained source and assets

Preserve both Cockpit pages **and their backing logic/tests**, not just their appearance. Providers own CI workflows, scheduling, registration authority and results; Soda manages local capacity.

The predecessor repository is separate. Do not modify it, close its issues or import its separately reserved Updates platform as part of this work. Preserve attribution and licenses. Canonical branding in `assets/` must not be casually regenerated or removed; adapt installation paths in staging source where necessary.

## Handoff and commits

Inspect the working tree before editing and preserve unrelated user changes. Keep commits coherent and frequent; do not amend/rewrite history without permission.

For substantial changes, update `docs/implementation-status.md` with what changed, remaining source work/native assumptions, and what actually ran. Keep implementation claims separate from validation claims. Final responses should identify the change, commit/check status and any execution still held—without claiming the appliance works before native evidence exists.
