# Unified Soda frontend and backend implementation plan

**Status: official Forgejo template overrides selected; U01 acceptance withdrawn.
Only bounded U08 is accepted (1/20 core milestones).** This plan replaces the
fork-dependent implementation sequence, not the installed system or prior evidence.
The [workflow inventory](forgejo-api-coverage.md), [page/dependency inventory](dashboard-plan.md),
[architecture](architecture.md), [licensing notes](licensing.md), [deferred scope](deferred.md)
and [handoff](implementation-status.md) remain the supporting references.

## Selected direction — official template overrides; partial U01 invalidation

Use upstream Forgejo's **official template overrides and extension points**, native
server-rendered pages, handlers, authentication and business rules. Customize
presentation/navigation/assets instead of recreating every workflow over JSON.
Soda retains its Go/React development-environment/access integration. No React SSR,
production Node service, iframe, scraped HTML fragments or borrowed cookies is
selected. Exact navigation/session/origin integration still needs review and proof.

**Forking Forgejo is an architecture failure path, not an implementation option.**
If a requirement appears to need downstream source patches or a custom Forgejo
executable, stop that approach and revisit the architecture with the user. Calling
it a small API extension does not change the boundary. Check supported configuration,
themes/assets, template extension points/overrides, native workflows/protocols and
existing APIs/integrations first. Do not bypass security or silently omit required
workflows. Separately scoped upstream contributions do not authorize an unmerged
downstream dependency.

The inspected template loader layers `<CustomPath>/templates/` before built-ins.
That is source evidence for an official mechanism, not proof of a custom shell.
Review the exact selected stock version, templates, JavaScript/asset contracts,
configured custom paths and reload/restart requirements before implementing changes.
Prefer small extension points to whole-template overrides where sufficient.

U01's architecture acceptance at `542de21` is withdrawn: it incorrectly treated a
separate all-React/API interface as mandatory and turned missing JSON endpoints
into a requirement for a source-built fork. The source/API findings, authority
findings, original-code license and actual evidence remain useful. The dedicated
source/patch preparer, lock and proposed native auth/read/build contracts have now
been removed, not parked as dormant alternatives. Git history retains them.

This cleanup does not accept U01 or select an upgrade. Its remaining work is the
specific template/navigation/session integration review, not another H01 inventory
or a new decision between a fork and templates.

## Current execution snapshot — U08 completion run

This heading is retained for incoming evidence links.

- Installed affected components on retained `soda-test` remain `8b823db`: dashboard,
  helper, runner companion and default new-project image. React preview is `/app/`;
  default Soda routes are HTMX. Stock Forgejo remains 15.0.7 with its existing ingress.
- Later API/React source has local checks but is not all deployed. No custom Forgejo
  executable, source patch, new auth/read endpoint or template override was delivered
  by the abandoned source-preparation work.
- Four environments, credentials, memberships, dirty checkouts, roots, keys,
  workloads/volumes, backups and failures are preserved. No fifth environment,
  reset/reseed, cleanup, host-network change or reboot is implied. Never reboot infra.
- U08 combines `f233a4a` lifecycle proof for explicitly unchanged mechanisms,
  `c96c108` fresh/different-UID exec proof and `8b823db` matched regressions. It is not
  final-image lifecycle, complete customized frontend, fresh install or aarch64 proof.
- Missing console hook remains P11/U20 work. Tailnet `NeedsLogin` and zero runners
  are not enrollment/provider-job acceptance. Infra reachability does not prove a
  laptop/LAN/Tailnet route. Old backups are not lossless rollback after new writes.

See the [accepted U08 handoff](implementation-status.md#u08-accepted--bounded-native-x86_64-first-product-proof).

## 1. Authority and preservation

Forgejo owns identities, passwords, factors, permissions, ownership/transfers, Git,
collaboration, CI and administration. Native pages retain native verification,
forms, sessions, CSRF, redirects and operation results. Template visibility never
authorizes an action. Soda owns only its additional preferences, development keys,
environment associations/memberships and secure adapter/session state.

Keep native site admin, organization/repository authority, Soda operator, project
administrator and host root distinct. Ordinary Soda API calls use acting-user
grants; privileged credentials remain confined to authorized setup/operator work.
Do not mirror provider roles, issue codes/grants, inspect Forgejo's database or
implement its Git algorithms. Missing API coverage is not missing native functionality.

Retain real provisioning, explicit join, project-local accounts, shared tools/files,
direct-IP SSH/SCP/SFTP and persistent existing-container startup. Joining does not
confer Git access. Browser logout does not imply Linux offboarding; current native
ownership must authorize new Soda operations without preserving stale creator
rights. Preserve Linux accounts/data and explain existing-access limits truthfully.

## 2. Supported integration and existing source

- Forgejo templates/assets/native configuration: use supported custom paths and
  extension points, retaining native scripts, form contracts and security gates.
  No alternate auth transport, extension-descriptor API or downstream build system.
- `dashboard/src/`, `internal/{web,forgejo,store,host}/`: retain current working
  Soda environment and stock-API code/tests. Existing Forgejo-specific React screens
  are not fork machinery; retire duplicated paths only when their replacement and
  callers are verified. Do not break login or delete useful adapters blindly.
- `scripts/build-native.sh`, staging/sealing/install and service units: retain the
  stock Forgejo image acquisition/IID/OCI path. Package custom templates/assets as
  ordinary configuration content after review, not by recompiling Forgejo.
- Keep current dependency pins, separate Cockpit frontend, Go backend, SQLite
  extension records, native project helper and canonical branding. No incidental
  package upgrade or replacement framework. See [licensing](licensing.md).

## 3. Outstanding integration review

U01 must establish the smallest shell/navigation override against the selected stock
version, which native content/scripts remain unchanged, how users enter/return to
Soda environments, and the supported browser/OAuth/session/origin arrangement.
A shared visual shell is not a shared security session. Keep current login until
its integrated replacement is proven. Preserve enrolled WebAuthn RP-ID/origins,
mail/IdP callbacks, native Git advertisement and separate Cockpit listeners.

Any concrete supported-interface limitation returns to the user for architectural
resolution. It does not create a patch backlog. Conditional native features remain
required when enabled, not disabled to make coverage easier. E01–E03/media stay
unselected; no ISO exists or is required for core progress.

## 4. Packaging and coordination

### Coordination with native support porting

This plan leads the core product. The [native support plan](native-porting-plan.md)
provides outside VM/SSH/evidence/artifact tools and retained operator integrations,
not a second frontend, provider backend or readiness gate.

| Shared responsibility | Core owner | Support boundary |
| --- | --- | --- |
| Stock Forgejo image/version, template/assets and Soda frontend payload | U01/U02; feature owners | P04 inspects/packages the selected bytes; no source-built fork |
| API/session/CSRF/config/schema and integration | U03/U04/U18 | P05 transports approved inputs/steps, not copied bootstrap or database edits |
| Project helper, accounts, runtime, direct access and terminal | U07; U08 retained/U20 final | P02/P03 transport only; product scenarios stay in core tests |
| Cockpit, Tailnet, local runners, console, branding and CLIs | Preserve U02/U18; acceptance U20 | P11 retains backing logic/dependencies/tests and scoped native evidence |
| Staging, image identity, install and evidence | U02/U03/U18/U20 | P04/P05/P12/P13 consume exact contracts; no duplicate build/acceptance platform |

Keep existing `forgejo.iid`, `images/forgejo.oci`, stage/seal/install callers and
actual stock image identity. Version/platform checks, override compatibility and
license/source delivery are still necessary without building Forgejo. No direct
source/patch receipt or custom API capability manifest is needed.

## 5. Soda API, state and migration contracts

Retain explicit typed operations, complete bounded errors/results, canonical stable
IDs, per-operation read/write scopes, native target checks and all unsafe-method
CSRF/origin protections. No arbitrary proxy, operator fallback or uncertain mutation
replay. Apply these to the APIs Soda actually consumes, not every native screen.

Keep encrypted schema-v3 session/provider-bound grants, restricted external keys,
serialized refresh and logout-winning persistence. No passwords/provider roles in
Soda SQLite. Native rendered login and Soda OAuth/session cleanup retain their
separate native semantics. No guessed global logout or security-key reset.

Preserve populated data through ordered Soda migrations and Forgejo's own supported
upgrade mechanism. Test prior/current/newer-schema refusal, missing/wrong keys,
partial failure and repeat startup. Rehearse consistent backups and actual rollback
limits; do not restore old databases over later writes or rerun first-install.

Soda React requests/drafts remain actor/target-bound; abort obsolete reads, clear
private state on logout, render content inertly and keep tokens out of the browser.
Native Forgejo JavaScript/form contracts remain native; template edits must not
break their initialization, URL handling, accessibility or request protections.

## 6. Milestone map and execution order

Keep U01–U20 and their action owners; these numbers are not a demand for serial
acceptance. The former fork-dependent tasks are removed. Only U08 is accepted.

Next: U01's focused override/integration review → U02 packaging plus U03/U04 session/
configuration integration → feature-owned native page compatibility and Soda
integration → U17 complete rehearsal → separately approved U18 cutover → U19/U20.
No new roadmap, native source-build gate or repeated broad audit is introduced.

## 7. Detailed core milestones

### U01 — Capability, authority and baseline audit

Reuse H01's 179 groups and retained licensing/authority findings. Review exact stock
customization points, selected templates/assets and navigation/session/origin
integration. Confirm which existing Soda routes are kept versus replaced; preserve
working login. **Exit:** a concrete supported override/integration design and
packaging/test responsibilities, without a fork requirement. **Still open.**

Three inspected Soda defects remain actual source work, not resolved by deleting
the preparer: shared-admin credentials in legacy projects/People (U04/U05/U06/U16),
independent Linux/password rules in native account creation (U05/U16), and stale
`Project.OwnerID` authority in environment APIs (U07/U12). Fix active paths or retire
them after verified native replacements; do not leave them until final cleanup.
Retain malformed-input/security bounds, Linux provisioning restrictions and real
Soda membership enforcement. This is not an exhaustive authority audit.

### U02 — Stock Forgejo customization and Soda asset packaging

Package supported templates/themes/assets with stock Forgejo through existing
staging/install inputs; retain native image/entrypoint/data/protocols and Soda/Cockpit
builds. Validate actual override discovery, missing assets/notices, path safety,
private-input exclusion and exact image/platform pairing. **Exit:** repeatable
packaging and tested custom asset/template loading; no Forgejo compilation.

### U03 — Soda API, compatibility and migration foundation

Finish bounded stock-API inputs/results/scopes, protected grants, strict shared
configuration consumers and populated migration/refusal tests. Compatibility means
actual stock interfaces and native templates/scripts, not a new descriptor endpoint.
**Exit:** tested adapter/config/data compatibility and safe upgrade/refusal behavior.

### U04 — Native Forgejo authentication and Soda sessions

Keep native login/password/MFA/recovery/consent/IdP flows and their templates/forms.
Integrate Soda OAuth and safe return navigation without borrowed cookies or a new
native session-handle API. Preserve RP-ID, callbacks, expiry, revocation and actual
logout effects. **Exit:** real integrated browser onboarding/security and negative/
rotation/CSRF/actor-isolation proof, with native authority intact.

### U05 — Profiles, account security, keys and onboarding

Use native account/security/admin-user pages and mechanisms; retain only Soda's
additional preferences/development keys and onboarding associations. Distinguish
native Git SSH/GPG keys, development keys and Linux eligibility. Remove independent
Soda account/password rules. **Exit:** native account/onboarding/security journeys
through the customized presentation, with actual admin/non-admin/operator separation.

### U06 — Repository discovery, creation and basic browsing

Preserve native repository/create/template/README/tree/content/archive/LFS workflows.
Use acting-user stock APIs for legitimate Soda repository/environment associations.
No operator-token discovery. **Exit:** integrated native browsing/creation plus
Soda selection, private/reader/writer/paging/error and safe-navigation proof.

### U07 — Environments, direct access and own-workspace terminal

Retain one eligible repository/environment, stable IDs, real create/explicit join/
account-key installation and own-login/IP/public-host-key inspection. Resolve current
native ownership for privileged Soda actions; preserve state on denial/unavailability.
Finish native/DB partial failures and forged-target cases without retries/recreation.

Implement the requested browser terminal as the signed-in user's **existing**
project-local account/home, not host root/shared admin. Review one bounded transport,
PTY/backpressure, CSRF/origin, expiry/disconnect/process and secret/audit behavior
with U02/U03/U04. No implicit create/join/start, arbitrary host flags or key upload.
**Exit:** real adapter/native/browser authorization, own-workspace and direct-access
proof, preserving stopped/failed states and shared workloads. U08 is not reopened.

### U08 — First installed product proof

**Accepted only for the retained bounded native x86_64 proof.** No new criteria are
retroactively added; template/session changes and final repetition have other owners.

#### U08 completion execution plan — baseline `0d4c4eb`

Historical evidence anchor. Execution ran through `952f3b3`, `935dbdf`, `f233a4a`,
`c96c108` and `8b823db`; the old checklist is retained in Git/history and the handoff.
It is not permission to repeat fixture creation, stop/start or reboot.

#### U08 closure reconciliation — merged candidate

| Accepted evidence | Boundary retained |
| --- | --- |
| `8b823db` native build/check and matched backed-up rollout | Dashboard/helper/runner/default image; not whole-host upgrade |
| Operator/Alice/Bob OAuth, keys, repositories and explicit native joins | Historical native account flows, not final customized-template acceptance |
| Routed personal SSH/PTY/SCP/SFTP, shared tools/files and personal Git | Actual project-local identity/permissions; not laptop routing by inference |
| `c96c108` fresh/different-UID nested exec, image build, HTTP/SQL | Namespaced capabilities/default seccomp, not a privileged parent or hostile-tenant guarantee |
| `f233a4a` same-project stop/start and only `soda-test` reboot | Reused for explicitly unchanged mechanisms; not new final-image lifecycle proof |
| Preserved four roots, dirty data, identities and failed evidence | No automatic workload resurrection, data rollback or cleanup claim |
| Console hook missing; Tailnet `NeedsLogin`; zero runners | P11/U20 gaps, not successful provider/operator acceptance |

#### Core-owned native proof detail

U08/U20 own these assertions, not a duplicate P product suite:

- Real browser/native identity/repository selection and explicit join, including
  the creator; no seeded sessions/rows in place of account/key provisioning.
- Current project IP and independently pinned public host keys from the named
  routed client; direct SSH/SCP/SFTP, not forwarding mistaken for reachability.
- Independent project roots/authority and specific forbidden outcomes, not a
  transport failure counted as an authorization denial.
- Ordinary personal checkouts and own native Git credentials/agents for real
  clone/commit/push/readback; never borrowed admin tokens or private-key uploads.
- The same project-shared installed tools/files and intended ownership for two
  users, including noninteractive access—not matching versions or shared caches.
- Project-local engine image build, bind-mount HTTP edits and committed SQL reached
  by users/client; verify different-UID exec and member/cross-project denials.
- Before authorized lifecycle changes, bounded complete allowlisted state snapshots:
  account/group/public-key/home/dirty/tool/config/workload/volume/data identities.
  Fail incomplete snapshots; exclude shadow/password/private-key material.
- Stop/start the existing environment and only an explicitly approved appliance
  reboot; compare persistent state without replacement/pruning/reseed. Record
  actual workload restart behavior; explicit starts are not automatic recovery.
- U20 adds matching final revision, fresh/populated upgrade, templates/session/
  collaboration/admin, own-workspace terminal and independent native aarch64 proof.

### U09 — Native code, history, comparison and Git workflows

Use native blame/diff/history/refs/write/fork/import/task workflows through customized
pages; do not add read APIs or copy Git algorithms. Retain useful existing Soda
stock adapters until their callers are coherently retired. **Exit:** complete
native code/ref/copy navigation and real Git outcomes, private/protected/stale/
large/binary/error behavior, with no environment copy or mutation as a side effect.

### U10 — Issues, time/dependencies and boards

Preserve native issue/template/comment/attachment/timer/dependency and repository/
organization/personal board workflows. **Exit:** complete customized-page behavior,
real two-user outcomes and native visibility/authorization, with no Soda board store.

### U11 — Pull requests, reviews and merge

Preserve native review/reply/resolve/viewed/dismiss, code ranges, checks/protection,
merge/rebase/auto-merge and branch handling. **Exit:** real multi-user private/
stale/conflict/check-failure/native Git proof; no copied merge or eligibility engine.

### U12 — Repository and organization administration

Preserve native settings/units/protection/collaborators/teams/invitations/hooks/
clients and rename/transfer/archive/delete pages. Keep current native ownership
checks for linked Soda records without Linux remapping or stale-creator privileges.
**Exit:** native owner/admin/reader/organization/site distinctions, genuine mutation
outcomes and retained project data. Hook delivery/network mutations need exact scope.

### U13 — Work, search, notifications and activity

Use native work/search/profile/activity/graph/notification workflows; compose only
legitimate Soda environment information. **Exit:** actual permission-filtered native
results and integrated navigation/paging/actor isolation, not a local index/inventory.

### U14 — Actions and provider configuration

Preserve native jobs/logs/artifacts/dispatch/rerun/cancel/trust, secrets/variables and
provider-runner pages. Forgejo owns scheduling/results/registration; Cockpit owns
local capacity. **Exit:** native conditional/unit/actor/expiry/secret behavior and
separately scoped real provider outcomes; no runner-token impersonation or scheduler.

### U15 — Releases, wiki and packages

Preserve native release/tag/assets, wiki/history and package/settings/protocol
workflows. **Exit:** actual private/native mutation/byte/large/error behavior and
safe integrated navigation. Do not invent stronger write semantics or a registry.

### U16 — Forgejo site administration

Preserve native People/security/auth-source/configuration/maintenance/moderation/
quota/runner administration through native pages. The acting user's native authority
is required, not Soda operator identity. **Exit:** complete conditional native admin
flows and real permission/redaction/reauth outcomes under explicitly scoped actions.
No generic configuration editor, shell proxy or site-admin-to-host escalation.

### U17 — Complete coverage, upgrade compatibility and rehearsal

Use the single 179-group inventory as required user outcomes, not an API-creation
backlog. Verify native custom templates, scripts/forms/assets, Soda integration and
all feature-owned browser/security tests. Check overrides against a reviewed stock
upgrade and retire unnecessary overrides. **Exit:** complete candidate/ingress/
upgrade rehearsal with native protocols and Cockpit intact, before U18 cutover.

### U18 — Preserved cutover and coherent legacy removal

After U17 proof and exact target/action approval, install matching stock-image/
customization/Soda/configuration assets with consistent backups and enrolled-key/
data preservation. Retire replaced routes/templates/React screens/adapters/tests
coherently; preserve legitimate setup and protocol callers. No first-install replay,
redirected POST replay or stranded login. **Exit:** real installed integrated native
Forgejo/Soda journeys and unchanged retained state, not a successful bundle alone.

### U19 — Measured usability and performance

Improve real native/customized/Soda tasks: navigation, responsive behavior, keyboard/
focus/contrast, large content and measured request/render costs. **Exit:** measured
improvements without another UI framework or weaker native authority. Basic
accessibility and error handling are required earlier, not deferred to polish.

### U20 — Final native verification and handoff

Run matching-revision source/adapter/browser/asset/upgrade/operator suites, fresh
install and populated upgrade, the core proof and terminal/native workflows on
independent native x86_64 and aarch64 targets. Consume P results, not a second
readiness certificate. Resolve console/Tailnet/runner gaps with actual evidence.
**Exit:** full required product proof on both architectures with honest permissions,
retention and operational handoff. Publication and additional targets remain separate.

## 8. Conditional Soda extensions

Unselected; do not add dormant helpers/schemas or use these as core gates.

### E01 — Creation-time OS profiles

Only after explicit selection: approved images with native account/runtime/tool/
asset/build tests. No change to existing roots or inferred creation-image history.

### E02 — Existing-environment lifecycle controls

Only after explicit selection: authorized start/stop/restart of trusted existing
units, with persistent state and precise cross-project denials. No deletion/rebuild.

### E03 — Resource limits and usage

Only after explicit selection: native cgroup/storage design and enforcement proof;
no quota/scheduler/billing/pressure-recovery platform or implicit existing-root resize.

## 9. Inventory coverage cross-reference

The [179-group source register](forgejo-api-coverage.md) remains the single inventory.
These owners integrate/test native rendered workflows and legitimate Soda adapters;
they do not implement missing Forgejo JSON APIs. Preserve every conditional-enabled
workflow. Historical API classes are observations, not required patch assignments.

| Audit groups | Primary owner | Integration responsibility |
| --- | --- | --- |
| AU01–AU10 | U04 | Native authentication, Soda session/navigation |
| AU11–AU22 | U05 | Native account/security; U04/U16 shared integration |
| AC01–AC05, AC07, AC11–AC13 | U05 | Native account plus Soda-only preferences/development keys |
| AC06, AC08 | U13 | Profiles/follow/activity |
| AC09, CO01–CO02 | U06 | Repository selection/content and environment entry |
| AC10 | U02 | Native/customized about/help and actual licensing |
| CO03–CO16, CO18 | U09 | Native code/ref/copy workflows |
| CO17, WK01–WK06 | U13 | Native search/activity/work |
| IS01–IS15, BD01–BD04 | U10 | Native issues/boards; U11 conversation integration |
| PR01–PR12 | U11 | Native reviews/merge; U09/U14 code/checks |
| RS01–RS16, HK01–HK06, OR01–OR09 | U12 | Native settings/organization authority and Soda associations |
| CI01–CI15 | U14 | Native Actions/provider administration; Cockpit local capacity |
| RE01–RE04, WI01–WI05, PK01–PK06 | U15 | Native releases/wiki/packages |
| AD01–AD23 | U16 | Native site administration and conditional ordinary-user moderation |
| UI01–UI04 | U17 | Complete integrated presentation/rehearsal; U18 installed cutover |
| UI05 | U07 | Own-workspace terminal; U20 final native proof |

## 10. Definition of done, execution and handoff

Connected source/configuration/templates and authored tests are not executed proof.
Source/build tests, native provider results, browser outcomes and installed acceptance
remain separate. No stub, mocked success or unimplemented override completes a task.
Native workflow preservation replaces the old requirement to reimplement every
screen, not the requirement to test that our integration actually works.

Already authorized local development checks remain available. Deployment, provider/
account/repository mutations, fixtures, origins/networks, enrollment, lifecycle,
cleanup, publication and upstream submission retain their exact separate scopes.
Preserve all failed evidence/private inputs; never expose credentials in logs/argv/
screenshots or remove retained project state. E/media and missing sibling hardware
do not block useful independent source work or certify the untested architecture.

Update the handoff with actual changed paths, checks, revision/target identities,
retained state, failures and remaining work. **Next: U01's concrete supported
template/integration review.** No fork preparer, patch protocol or source-build
work package remains a prerequisite.
