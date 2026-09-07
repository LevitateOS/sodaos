# Unified Soda frontend and backend implementation plan

**Selected: native Forgejo frontend plus Sodaspaces. Only bounded U08 is accepted
(1/20); U01's architecture acceptance remains withdrawn.** The standalone React
frontend and duplicate forge adapters are removed. No replacement component library,
Bootstrap frontend, React shell or downstream Forgejo build is selected.

Supporting references: [workflow register](forgejo-api-coverage.md),
[frontend integration](forgejo-frontend-integration.md), [inventory](dashboard-plan.md),
[architecture](architecture.md), [API](dashboard-api.md), [licensing](licensing.md),
[deferred scope](deferred.md) and [handoff](implementation-status.md).

## Frontend decision update — retire Soda React and PatternFly

The later decision is **Forgejo's frontend everywhere**, with a repository
**Sodaspaces** tab and proposed right-side environment drawer. Go retains Soda's
legitimate environment/access/native integration. Use official native templates,
styles/scripts and bounded additions, not a separate forge frontend. No Codespaces
infrastructure/lifecycle model is imported: environments remain shared, persistent
and backed by project-local human accounts. This UI name does not rename stored
projects, service commands, container identities or database paths.

Root `dashboard/`, SPA loading, duplicate forge feature routes/clients and their
exclusive build/tests are removed. Cockpit's separate React/PatternFly pages remain.
The existing Go/HTMX pages remain functional until replaced by Sodaspaces. Deletion
is not proof of the new tab/drawer or its authenticated backend connection.

## Selected direction — official template overrides; partial U01 invalidation

Forgejo owns native rendering, authentication, authorization, Git/collaboration and
administration. Customize through supported configuration, themes/assets, dedicated
template hooks and only necessary whole-template overrides. Preserve native forms,
scripts and security gates. The selected source's loader layers custom templates
before built-ins; this is deliberately supported, not a backend patch mechanism.

**Forking Forgejo is an architectural failure path.** If a requirement appears to
need a downstream patch/custom executable, stop and revisit the concrete conflict
with the user. No API descriptor, source/patch preparer, generic proxy, borrowed
cookie or scraping solution is selected. Separately scoped upstream contributions
are not permission to ship an unmerged fork.

U01 acceptance at `542de21` was withdrawn because API-only presentation was wrongly
hardened into a requirement. Keep factual H01/license/authority findings and actual
evidence, not the abandoned headless/native-auth/read/build proposals. Do not repeat
the broad audit or treat missing JSON as missing native functionality.

## Current execution snapshot — U08 completion run

Retained evidence anchor; actual installation is not changed by source cleanup.

- Affected components on `soda-test` remain `8b823db`, with historical React `/app/`
  preview and HTMX defaults. Forgejo remains stock 15.0.7. New source no longer serves
  that preview; no source cleanup is an installed upgrade/rollback instruction.
- Four environments, accounts, keys, dirty checkouts, workloads/volumes, credentials,
  backups and failed evidence stay intact. No fifth environment, reset/reseed,
  capability/network change, project replacement or infra reboot is implied.
- U08 combines `f233a4a` unchanged-mechanism lifecycle, `c96c108` fresh/different-UID
  nested exec and `8b823db` matched regressions. It is not final-image/fresh/aarch64
  or Sodaspaces acceptance. Deleted browser scripts remain available at their
  historical Git revisions; old evidence is not reassigned to new source.
- Missing console hook, Tailnet `NeedsLogin`, zero runners and actual remote-client
  routing remain explicit gaps, not inferred successful provider/operator proof.

See [bounded U08 acceptance](implementation-status.md#u08-accepted--bounded-native-x86_64-first-product-proof).

## 1. Authority and preservation

Forgejo owns identity, passwords/factors, native permissions/ownership/transfers,
repository workflows, CI scheduling and administration. Soda owns additional
preferences, development public keys, environment associations/memberships and
necessary protected adapter sessions. Use acting-user grants for ordinary APIs;
setup/native operator credentials are not substitutable human authority.

Native site admin, org owner/admin, repository owner, Soda operator and host root
are distinct. Current native ownership authorizes new privileged Soda operations;
creator snapshots must not retain authority after transfer. This does not remap
Linux users/groups, revoke already-installed keys or terminate existing SSH access.

Retain real explicit provisioning/join, shared tools/files, native workloads,
ordinary personal Git checkouts and direct-IP SSH/SCP/SFTP. Joining does not grant
Git access. Opening a tab/drawer must not create, join, start or repair anything.

## 2. Supported integration and existing source

- `internal/web/` embeds the working Go/HTMX pages/static assets and retains bounded
  Soda session/preferences/development-key/environment APIs. `cmd/soda-dashboard`
  is still the backend command, not the deleted React directory.
- `internal/forgejo/` retains only actual identity/OAuth/setup/advertisement,
  native-authorized People, repository-selection and ownership callers. No local
  Git engine, copied permissions, replacement boards/CI or forge administration API.
- `internal/store/`, `internal/host/` and `project-os/` remain production integration.
  Preserve grant encryption, memberships, helper restrictions and persistent roots.
- Stock Forgejo image/IID/OCI/stage/install paths and canonical assets remain.
  Soda HTML is embedded in Go; new bundles refuse the retired SPA asset/input tree.
  Cockpit retains its own frontend build, dependencies and focused tests.

## 3. Outstanding integration review

Close the **small supported tab/drawer integration**, not another frontend selection.
Inspect exact 15.0.7 hooks/context, native markup/styles/scripts and effective custom
path/reload behavior. `custom/extra_tabs` receives repository context; the new drawer
and authenticated Soda connection are not implemented or proven by that fact.

Settle how a native-page action reaches Soda's protected backend without shared-
cookie assumptions, credential forwarding, generic proxying or weakening CSRF.
Different ports on one hostname do not isolate cookies. Preserve enrolled WebAuthn
RP-ID/origins, native login/mail/IdP callbacks, Git advertisement and separate Cockpit
listeners. Show actual acting identity and handle mismatched native/Soda sessions.

First prove a read-only native repository tab/drawer against real environment state;
then explicit create/join and connection details. Keep ordinary navigation and clear
unavailable/denied states. No mocked success or manually assembled account setup
completes integration. A genuine unsupported boundary returns to the user.

## 4. Packaging and coordination

### Coordination with native support porting

This is the leading core plan. The [native support plan](native-porting-plan.md)
owns outside transport/artifact/VM tools and retained host-operator integration,
not another frontend, schema, auth design or product readiness gate.

| Shared responsibility | Core owner | Support boundary |
| --- | --- | --- |
| Stock Forgejo, custom templates/assets and Go HTML payload | U01/U02 | P04 inspects specified image/payload; no custom Forgejo build |
| Soda API/session/CSRF/config/schema | U03/U04/U18 | P05 transports approved inputs; no copied bootstrap/database writes |
| Accounts/helper/runtime/terminal | U07; U08 retained/U20 final | P02/P03 transport only; core owns product scenarios |
| Cockpit/Tailnet/runners/console/branding/CLIs | U02/U18 preserve, U20 accepts | P11 retains native logic/dependencies/tests |
| Stage/install and evidence | U02/U03/U18/U20 | P04/P05/P12/P13 consume core contracts |

Deliver actual supported overrides and public asset inputs with required notices.
No template renderer, custom path, new payload placeholder or automatic reactivation
is supplied by this removal. Add only concrete inputs with real production callers;
verify paths/modes/SELinux discovery and conflicts without replacing persistent data.
New bundles must match the new verifier; old populated bundles retain their own
verifier and history, not silent resealing or blind first-install replay.

## 5. Soda API, state and migration contracts

Keep typed, bounded operations and sanitized failures, stable IDs, current native
resource/actor checks, unsafe-method CSRF/origin protection and no mutation replay.
Retained source contracts are in [dashboard-api.md](dashboard-api.md). New consent
requests only current read scopes plus explicit admin scope for retained People;
old grants remain intact and actual introspected scopes stay authoritative.

Keep schema-v3 session/provider-bound encrypted grants, restricted external keys,
serialized refresh, logout-winning writes and strict configuration loading. No
password/provider-role inventory or second native grant issuer. Native and Soda
logout/revocation have distinct effects; do not promise Linux offboarding.

Preserve populated data, prior/current/newer-schema refusal, missing/wrong-key and
partial-failure checks. Forgejo owns its migrations. Old image/database backups are
not lossless rollback over later writes. Source deletion requires no DB migration.

## 6. Milestone map and execution order

Keep U01–U20 and the single 179-group register. Only U08 is accepted. Next is U01's
concrete Sodaspaces tab/drawer/auth contract, then U02/U03/U04/U07 connected source
and tests, feature-owned native page compatibility, U17 rehearsal, separately
approved U18 cutover and U19/U20. No React migration or new component-library plan.

### Preparation sequence — native templates and Soda environment integration

1. Close the bounded native hooks, drawer markup/behavior and authenticated request
   contract described above. Preserve working login; explain constraints before
   implementing a cost-changing alternative.
2. Retain `fed66cb`'s active authority fixes: acting-user legacy projects/People,
   native account validation, current-owner environment visibility and honest
   ownership-unavailable state. They are locally checked source, not deployed proof.
3. Implement repository-context entry using stable native ID and current actor grant,
   a concrete reservation lookup and protected read-only drawer state. Reuse the
   existing Soda API where it fits; do not add a second representation just to name
   a new endpoint. Only then extend validated OAuth returns for the exact new route.
4. Add the smallest official hooks/assets and actual build/stage/activation callers.
   Public URL delivery must be explicit, escaped and free of secrets. Preserve
   upstream notices, existing custom files and native scripts. No global CSS reset,
   whole template-tree copy, new framework or custom Forgejo executable.
5. Wire explicit create/join/key/connection actions to the existing Go/helper path,
   with actor/target/CSRF/partial-failure tests. Keep incomplete reservations rather
   than recreate them. Scope terminal separately under U07, not a fake root shell.
6. Author focused native browser tests replacing the retired React orchestration.
   Prove native tab/drawer loading, keyboard/focus, login/consent/expiry/identity,
   read versus write denials and real outcomes. Native execution still needs exact
   fixture/action/origin/target approval; no fifth environment or automatic rollout.

## 7. Detailed core milestones

### U01 — Capability, authority and baseline audit

Still open: concrete supported Sodaspaces integration and packaging/test contract.
Reuse H01 and license/authority findings; no broad re-audit or fork alternative.

### U02 — Stock Forgejo customization and Soda asset packaging

Deliver supported overrides/assets through existing stock image and stage callers.
Test discovery, missing assets/notices, path safety, modes and private-input exclusion.
React build/manifest machinery is removed, not replaced by fabricated payloads.

### U03 — Soda API, compatibility and migration foundation

Retain tested bounded Soda APIs/grants/migrations. Add only concrete tab/drawer
inputs/results and stable-ID lookup needed by the chosen authenticated integration.
Compatibility is actual stock behavior, not a custom descriptor endpoint.

### U04 — Native Forgejo authentication and Soda sessions

Preserve native login/password/MFA/recovery/consent/IdP flows. Prove protected Soda
entry/return, cookie/CSRF and actual expiry/logout behavior without borrowed sessions.

### U05 — Profiles, account security, keys and onboarding

Native account/security/Git keys remain Forgejo-owned; Soda owns additional profile/
development public keys. Linux eligibility is provisioning-only. Preserve the fixed
acting-admin legacy People path until a verified native replacement takes over.

### U06 — Repository discovery, creation and basic browsing

Native repository creation/content/archive/LFS stay native. Soda integrates legitimate
repository selection/environment entry using acting grants, never operator discovery.

### U07 — Environments, direct access and own-workspace terminal

Implement the Sodaspaces tab/drawer with real existing/reserved environment state,
explicit create/join and own connection information. Preserve current native owner
checks, legitimate memberships, fixed helper targets and honest partial failures.
The requested browser terminal uses the user's existing project-local account/home,
with bounded PTY/transport/backpressure, origin/CSRF and connection-lifetime checks.
No implicit create/join/start, host shell, private-key upload or workload deletion.

### U08 — First installed product proof

Accepted only for the retained bounded native x86_64 first-product proof.

#### U08 completion execution plan — baseline `0d4c4eb`

Historical anchor. Runs through `952f3b3`, `935dbdf`, `f233a4a`, `c96c108`, `8b823db`
remain in Git and the handoff, not permission to repeat fixtures/lifecycle changes.

#### U08 closure reconciliation — merged candidate

Retain `8b823db` build/check/backed-up affected rollout and actual Alice/Bob/operator
journeys; `c96c108` fresh/different-UID nested exec, Git/build/HTTP/SQL; `f233a4a`
lifecycle only for explicitly unchanged mechanisms. Preserve four roots/dirty state,
accounts/public host keys, workloads/volumes, credentials and failed evidence. Missing
console/Tailnet/runner/client-route/fresh/aarch64/final-revision proof remains explicit.

#### Core-owned native proof detail

U08/U20, not a P duplicate suite, own real browser/native identity/repository/explicit
join; current direct-IP/pinned-host-key SSH/PTY/SCP/SFTP; personal native Git credentials
and clone/commit/push/readback; two users sharing the same installed tools/files;
project-local image build/bind-mount HTTP/committed SQL and different-UID exec; exact
cross-project/role denials; complete bounded public-state snapshots excluding secrets;
separately authorized existing-container stop/start and appliance-only reboot without
replacement/pruning/reseed. Record actual workload restart behavior and preserve all
failures. U20 adds final revision, fresh/populated upgrade, Sodaspaces/terminal/native
collaboration/admin and independent native aarch64 proof. Deleted React orchestration
needs new native-page callers, not reclassification of historical results.

### U09 — Native code, history, comparison and Git workflows

Preserve/test native blame/diff/history/refs/write/fork/import/task flows, private/
protected/stale/large/binary/error behavior and real Git. No new read API/Git engine.

### U10 — Issues, time/dependencies and boards

Preserve native templates/comments/assets/timers/dependencies/boards and actual
multi-user outcomes/visibility. No Soda issue/board database or replacement API.

### U11 — Pull requests, reviews and merge

Preserve native reviews/ranges/resolve/viewed/checks/protection/merge and branch
handling. Prove actual multi-user/stale/conflict/native outcomes, not copied rules.

### U12 — Repository and organization administration

Preserve native settings/units/protection/team/invitation/hook/transfer/lifecycle
workflows and current-authority checks for linked Soda state. No Linux remapping.

### U13 — Work, search, notifications and activity

Use native work/search/profile/feed/graph/notifications and permission filtering.
Compose legitimate Soda state only, not a local index or forge inventory.

### U14 — Actions and provider configuration

Preserve native workflow/jobs/logs/artifacts/controls/trust/secrets/provider runners.
Providers own scheduling/results/registration; Cockpit owns local capacity. Real
jobs/network/registration tests need scope; no runner-token impersonation.

### U15 — Releases, wiki and packages

Preserve native publishing/assets/history/settings and package protocols, with actual
private/mutation/byte/large/error proof. No copied registry or stronger write promises.

### U16 — Forgejo site administration

Native admin/security/auth-source/configuration/maintenance/moderation/quota/runner
flows remain native. Require acting native authority, not Soda operator identity.

### U17 — Complete coverage, upgrade compatibility and rehearsal

Verify all 179 outcomes, native enabled/disabled conditions, templates/scripts/forms,
Soda integration and focused tests. Review overrides against a supported stock
upgrade. Complete exact candidate/ingress/preserved-state rehearsal before U18.

### U18 — Preserved cutover and coherent legacy removal

After proof and exact approval, deliver matching stock/customization/Go/config assets
with consistent backup and enrolled-key/data preservation. Source React removal is
not installed cutover. Retire replaced HTMX/adapters with actual callers/tests;
no blind first-install/OAuth replay, redirected POST replay or stranded login.

### U19 — Measured usability and performance

Measure real native/Sodaspaces navigation, responsive/keyboard/focus/contrast and
large-state/request costs. Basic accessibility/error handling belongs earlier too.

### U20 — Final native verification and handoff

Prove matching final source/asset/browser/operator suites, fresh/populated upgrade,
core native journeys and terminal on independent native x86_64/aarch64. Resolve real
console/Tailnet/runner gaps. P evidence is input, not another readiness certificate.

## 8. Conditional Soda extensions

Unselected. No dormant code/schema or core completion gate.

### E01 — Creation-time OS profiles

Only after selection: approved image choices/native validation; no existing-root swap.

### E02 — Existing-environment lifecycle controls

Only after selection: scoped existing-unit start/stop/restart; no deletion/rebuild.

### E03 — Resource limits and usage

Only after selection: actual native enforcement/observation; no quota/recovery platform.

## 9. Inventory coverage cross-reference

The [179-group register](forgejo-api-coverage.md) is a required-outcome inventory,
not a missing-JSON implementation backlog. Historical Soda source classes no longer
mean the deleted React adapters exist. Conditional-enabled native workflows remain.

| Audit groups | Primary owner | Integration responsibility |
| --- | --- | --- |
| AU01–AU10 | U04 | Native authentication and protected Soda integration |
| AU11–AU22 | U05 | Native account/security; U04/U16 share integration |
| AC01–AC05, AC07, AC11–AC13 | U05 | Native account and Soda preferences/development keys |
| AC06, AC08 | U13 | Native profiles/activity |
| AC09, CO01–CO02 | U06 | Native repositories and Sodaspaces entry |
| AC10 | U02 | About/help and licensing |
| CO03–CO16, CO18 | U09 | Native code/ref/copy |
| CO17, WK01–WK06 | U13 | Native search/work/activity |
| IS01–IS15, BD01–BD04 | U10 | Native issues/boards; U11 shared conversation |
| PR01–PR12 | U11 | Native review/merge; U09/U14 shared code/checks |
| RS01–RS16, HK01–HK06, OR01–OR09 | U12 | Native settings/org authority and Soda associations |
| CI01–CI15 | U14 | Native Actions/provider configuration |
| RE01–RE04, WI01–WI05, PK01–PK06 | U15 | Native publishing/wiki/packages |
| AD01–AD23 | U16 | Native administration/conditional moderation |
| UI01–UI04 | U17 | Complete presentation/rehearsal; U18 cutover |
| UI05 | U07 | Own-account terminal; U20 native proof |

## 10. Definition of done, execution and handoff

Code, authored tests, source checks, image builds and installed acceptance are
separate. No stub, mock, listener, historical log or deleted frontend completes
Sodaspaces. Keep native workflows rather than duplicating them, but test the actual
integration. Local checks remain scoped; live fixture/provider/network/lifecycle/
installation/publication/cleanup actions require exact permission.

Preserve all failed/private evidence and retained resources. Do not print secrets
or treat old backups as lossless rollback. Optional media, new frameworks and
unavailable sibling hardware do not block independent source or prove missing work.
Record actual paths/bytes/revision/targets/checks and remaining gaps in the handoff.
