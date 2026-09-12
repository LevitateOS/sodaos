# Native pages and Runners — combined completion plan

This is the sole coordination and retirement plan for native Soda pages and the
Cockpit-to-native Runners move. One owner coordinates the combined changes; there
are no cross-agent file reservations or separate page/runner delivery lanes.

Requirements stay with their owners:

- [Native page integration](forgejo-soda-pages-plan.md): host, entry, connection,
  logout, navigation and browser acceptance contracts.
- [Runner integration](runners-port.md): capacity, authority, operations, secrets,
  compatibility and runner-specific UI behavior.
- [Runner native validation](runners-native-validation.md): installed inputs,
  scenarios and observations, using the existing shared browser driver.
- [Installation](installation.md#retained-sodaspaces-cutover): affected-component
  maintenance, consuming the credential, project and terminal contracts.
- [Current handoff](implementation-status.md): installed state, outstanding work
  and active/consumed grants. [History](implementation-history.md) retains receipts.
- [AGENTS.md](../AGENTS.md): working method and execution/preservation policy.

## Completed execution

The former steps 1–5 are recorded in the handoff and history, with their bounded
acceptance and remaining wider gaps. They are not standing prerequisites for every
change. Their old procedures and intermediate statuses are retained in Git, not
another active execution sequence in this file. Use the relevant owner's checks
for affected behavior and the recorded evidence for unchanged mechanisms.

## 6. Retire only the Cockpit runner presentation

**Entry is accepted delivered parity.** For each proposed target, require no
unresolved delivery-relevant runner/authentication/preservation defect and a reviewed
coverage map from Cockpit responsibilities to the native Runners owner: local
capacity, registration/token handling, lifecycle/confirmation, uncertain outcomes,
refresh and diagnostics/provider guidance. Applicable isolated mutation evidence
can cover unchanged mechanisms; do not create or destroy retained capacity merely
to fill a matrix. Missing applicable evidence keeps the fallback in place.

The step-6 source review is recorded in the
[parity coverage map](implementation-history.md#step-6--source-retirement-parity-review).
Installed removal remains a separate per-target outcome in the handoff.

Then make one coordinated source-removal candidate:

- Remove `cockpit/soda-runners/` and runner-only React presentation/store/transport
  only after their applicable behavioral tests exist at the retained/new owner.
  Audit actual imports; do not remove shared helpers/dependencies by directory name.
- Update Vite/package inventories, staging/payload checks, links and installed
  operator journeys. The overlap scenario remains historical pre-retirement proof;
  port ongoing CLI/native coverage without invoking a removed package or copying
  the authentication harness. Keep ordinary root/Tailnet checks and their gates.
- Preserve `internal/runners`, host/API/CLI/launch/service/client contracts,
  sysusers/tmpfiles and focused native/security tests. Do not remove provider-owned
  Forgejo Actions templates, accounts, units, credentials, work or provider records.
- Preserve Tailnet, its backing logic/tests and React/PatternFly dependencies,
  Cockpit Services/Logs and ordinary administration. No whole-Cockpit removal.

Build/check/export and verify the removal candidate under the existing
[build contract](native-support.md#build-and-artifact-contract), with focused evidence
that native Runners and the remaining operator pages work. Rehearse its packaging/
removal delta and obtain explicit per-target removal-delivery approval. Delete only
inventoried obsolete package/assets/links after checking actual occupants; preserve
customized/unexpected files for a decision. Unaccepted targets retain their fallback.
Use the installation guide's maintenance contract, not a new rollout procedure.

**Exit per approved target:** obsolete runner navigation/package is absent, native
management and ordinary Cockpit/Tailnet work, and unchanged runner/provider state is
verified. Record source removal and each installed removal separately.

## 7. Close documentation and the bounded completion record

Update current guides to name native **Runners** at `/?soda-view=runners`. Remove
fallback wording only for actually retired targets. Update the handoff in place
with the candidate, affected targets, checks, remaining limits and permissions;
keep detailed execution receipts in history rather than extending the plan.

Source acceptance, native build/export, isolated proof, retained delivery and
Cockpit retirement are separate outcomes. Broader terminal/CLI, intended-client
routing, aarch64 and release acceptance stay with their feature/validation owners.

## Scope

Runner OS/OCI migration, cache policy, scheduling, reconciliation, Project OS
desktops, general recovery/updaters and project deletion are not added by this
retirement. Follow [Sodaspaces](sodaspaces-plan.md), the selected feature guides and
[deferred scope](deferred.md) rather than making those separate projects prerequisites.
