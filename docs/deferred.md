# Deferred and excluded work

This is the scope boundary for the [current architecture](architecture.md). It records parked ideas and edge-case work so they are not repeatedly promoted into prerequisites for the first version.

The [Sodaspaces minimum-controls plan](sodaspaces-plan.md#minimum-end-to-end-user-controls)
now selects basic Start/Stop and explicit own-project development-key apply/revoke as
bounded follow-up work. These do not reopen broader lifecycle/reconciliation work.
OS-profile selection and resource-limit/usage extensions remain unselected; destruction
still requires its explicit scope decision. [Outside support tools](native-support.md)
cannot change production policy or make optional installer media a product prerequisite.

**Deferred is not a roadmap commitment or a judgment that an idea is bad.** Reopening an item requires an explicit scope decision. **Not pursuing** identifies approaches outside the selected direction, rather than features waiting for implementation.

The current baseline still includes usable project-local accounts and SSH, genuine shared installed tools, reachable project IPs, a usable native service runtime and ordinary persistent project state. The predecessor's operator Cockpit **Tailnet and Runners pages and backing logic are selected for reuse, not deferred**. Deferral is not a request to strip their existing working validation or focused error handling.

Deferring machinery does not mean hiding native errors or granting developers unrestricted host administration.

## 1. Deferred: private toolchain and service branches

The longer-term idea is to start with shared project tools and pods, temporarily branch only what a developer needs, and return to the shared environment after their work is integrated. Toolchain and service branches could be independent. Those temporary resources would be ephemeral in purpose, not automatically disposable on a timer or Git event.

**The entire Soda-managed branch/copy/select/return experience is deferred.** Do not build dashboard selectors, private-resource database records, clone APIs or acceptance gates for it now. This includes:

- cloning installed toolchains, including portability and independence of their files;
- copying service definitions and state, including consistent PostgreSQL/database snapshots;
- switching a user's endpoints between shared and private services;
- defining effects on existing shells, applications and database connections;
- automatic environment propagation, reconnects or restarts.

Developers may use ordinary native tools to create private installations or workloads themselves. No special Soda machinery is required to permit normal development.

If this idea is revisited, the expected workflow is still developer-owned: commit and merge reproducible changes, apply them through normal tools, remove temporary toolchains/pods when no longer needed, and reconnect or restart the affected processes as necessary. The exact restart requirements are deferred with the switching mechanism.

**Soda does not monitor Git merges to promote live state, merge databases or installed binaries, delete temporary resources, or restart processes.** A merge is not a cleanup instruction. Expected developer cleanup is not an automatic lifetime or deletion guarantee.

## 2. Deferred: access-lifecycle edge cases

Keep out of the first-version implementation scope:

- automatically propagating later key rotations/revocations across memberships;
- coordinating Forgejo disablement, Soda membership removal and Linux access revocation;
- terminating already authenticated sessions;
- automated member departure, deprovisioning and manual account/key drift repair;
- invitation/approval systems and generalized permission synchronization.

Initial Forgejo-backed authentication, public-key registration and join-time account/key installation remain in scope. The minimum-controls revision additionally requires own saved-key removal and an explicit, authorized apply/revoke action for one's existing project account, with real native effects and SSH verification. Saving/removing a preference alone is not revocation; other projects and already authenticated SSH sessions stay untouched. These bounded operations now have source implementations; native proof/delivery remain separate, and this is not execution permission. Automatic synchronization and generalized offboarding remain deferred. These deferrals concern broader Soda cross-system/project access machinery, not Forgejo's own required invitations, account-security, grant/session revocation or identity lifecycle workflows. Preserve native workflows and conforming Soda associations without generalized Linux offboarding. The requested existing-account browser terminal includes its own connection/session lifetime; that is not permission to terminate unrelated SSH sessions or shared workloads. Do not request private SSH keys or claim that changing one system automatically revokes access in the others.

## 3. Deferred: identity and ownership remapping

Username renames, reused names, collision-resolution policies, historical UID/GID remapping and ownership migration do not need a new Soda subsystem now. Retain the stable Forgejo identity association and use the ordinary native account path; native validation errors should be reported rather than hidden behind automatic remapping.

The working project-admin rule is the owner of the associated Forgejo project/repository. Forgejo's native organization ownership and repository transfers remain required under the leading plan. Resolve current native authority rather than treating the stored creator as permanently authorized; preserve Linux accounts/data. Generalized Linux ownership migration and elaborate multi-repository/role mappings remain deferred, not a reason to build a parallel permissions product.

## 4. Deferred: general operation-recovery machinery

Do not expand the first join/create/service flow into a general solution for every interrupted request, concurrent retry, partially created resource or cross-system failure.

Automatic reconciliation, rollback/compensation, durable cross-system workflows and exhaustive recovery matrices are deferred. Use the selected native operations, report their actual outcomes and do not claim that a failed provision succeeded. This is not permission to destructively recreate existing work as a shortcut.

## 5. Deferred: broader recovery and lifecycle management

The following are not first-version requirements:

- coordinated backup/restore of Forgejo, Soda and project environments;
- disaster recovery and comprehensive service-resurrection/availability policies;
- quota systems, capacity admission and automated disk/resource-pressure management;
- long-term project image replacement and preservation/migration of arbitrary system modifications;
- separate environment archival, transfer, deletion and destructive rebuild workflows (not Forgejo's required repository lifecycle actions).

Basic persistence across normal project stop/start and host reboot stays in scope. That is not a claim of disaster recovery or high availability. This deferral does not cancel or redirect the predecessor's separately reserved Updates work in issue #61.

## 6. Not pursuing

- **A Soda-specific repository/toolchain/service format.** Repositories can use normal files, optional native mise configuration, image recipes and native workload definitions. Soda is not a replacement version manager, package catalog or Git workflow engine.
- **A project DNS or custom SSH-gateway project.** Show the project's reachable IP and use ordinary `ssh user@ip`; project labels are not a hostname-routing requirement.
- **Unrestricted host Podman access presented as project scoping.** Prefer a workable nested runtime; investigate project-scoped host execution if needed. A remote connection alone is not that boundary.
- **Restoring the old host-account developer model.** Developer accounts belong inside project environments. Both old Soda frontends are removed; the selected UI is native Forgejo plus Sodaspaces. The explicitly retained operator Cockpit Tailnet and Runners pages do not restore custom Cockpit developer workspaces.
- **Frameworks justified only by deferred scenarios.** Do not add generic authorization, reconciliation or orchestration platforms just because a future edge case could use one.

## 7. Still to prove, not deferred away

A simpler scope still needs real native evidence: shared installation identity and
permissions, direct-IP reachability, nested workloads/persistence and operator/project
separation. Bounded x86_64 results are in the [handoff](implementation-status.md), not
universal/final-product proof. No host fallback is implemented or selected without a
concrete nested-runtime blocker and a decision.

Follow [native validation](native-validation.md) with applicable target/action permission.
Outside helpers do not duplicate product tests or gate work on optional media. Native
evidence is required before claiming a verified product, not before independent source
work. Do not restore parked branching/selection/recovery machinery without a scope change.
