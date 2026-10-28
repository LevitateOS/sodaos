# Source investigation and boundaries

This audit follows reader-visible responsibilities through their actual source
owners. The 12 slices below group documentation concerns; they do not propose
new packages, processes, languages or runtime work. Paths are relative to the
SodaOS repository unless explicitly identified as website paths.

Three independent read-only specialist investigations covered Factory,
Spaces/Projects/development and installation/operations. The coordinator inspected
product scope, topology, extension mounting, Project preparation definitions,
the current public entrance pages and website discovery/route consumers. Agents
inspected the source identified in their respective traces and challenged the
shared boundaries described below. No tests or installed journeys were run.

## S01 Native navigation, identity and collaboration

**Intended model:** [overview](../../product/overview.md),
[Spaces](../../product/spaces.md) and [trust](../../architecture/trust.md).
Fountain owns native identity/collaboration and extension lifetime. Soda supplies
Spaces/Workspace/Feed rather than another login shell.

**Source trace:** `system/containers/extension/extension.json` declares global
Workspace with New Project/build-request subroutes, repository/global Feed,
Applications and the persistent Workspace panel. Workspace page and panel share
one owner/entry. `cmd/soda-extension/main.go`, `internal/web/extension.go` and
`internal/web/api/extension.go` connect native contribution authority to the
private service; `frontend/spaces/soda-extension.ts` validates the owner/session
binding and requires host request/stream/navigation transport. The public backend
redirect and private contribution routes do not create a second browser login.

**Allocation:** H01/H02, U01/U03/U05, P01 and D01. Retire dashboard OAuth/return
instructions. Native Git, issue/PR permissions and discussion remain native;
Factory Feed discussion is a separate scoped product interaction, not a second
editable issue system. UI naming differences are DEC05 in the plan.

## S02 Product direction, accepted work and native dependencies

**Intended model:** [overview](../../product/overview.md), particularly intent
review, accepted requirements, readiness and planning; adopted behavior in
[Spaces UX](../../design/spaces-ux.md).

**Source trace:** `sodaspaces-intent.ts`, `sodaspaces-intent-view.ts` and
`sodaspaces-intent-walkthrough.ts` in `frontend/spaces/` collect/save a private
objective, exploration/review, delivery approach, repository selection and
generation. `internal/web/api/factory_intent_explore.go`,
`factory_intent_generation.go`, `factory_request_view.go` and
`factory_revision_view.go` expose exploration, generation and durable history;
`factory_issue_view.go::apiFactoryIssue` projects acceptance/readiness.
Native records remain authoritative; request/accepted-input records bind their
revisions and explicit authority rather than copying editable issues.

**Required full journey:** saved draft → reviewed direction → exact repository
and finite planning allowance → conditional native graph generation/readback →
accepted issue inputs → readiness/plan freshness → dependency/result changes and
reassessment. Material edits/questions require authorized adoption; labels,
silence, agent confidence or reopening a cancelled issue cannot supply it. Lost
responses inspect the original effect instead of generating duplicates.

**Allocation:** F02/F04; H02 defines terms. Research/non-code work is included at
the semantic level; exact proposed field names and completion authority remain
the bounded research contract DEC04 writing dependency.

## S03 AI connections, sponsorship, capacity and usage

**Intended model:** [credentials](../../reference/credentials.md),
[trust](../../architecture/trust.md) and the overview's finite allowances.

**Source trace:** `frontend/spaces/soda-providers.ts::SodaProviders` owns actual
enrollment persistence, device verification, polling/cancel/revoke and
connected/reconnect/error states; `soda-identity.ts` integrates personal grants.
`internal/web/api/identity.go` owns connection routes and `identity_grants.go`
checks provisioned Project membership/current repository authority for personal
delegation. `factory_policy.go::{apiFactoryPolicy,apiFactoryCapacity,
apiFactoryOperatorGrant}` separate repository policy and appliance resources.
`factory_sponsorship.go::{apiFactorySponsorship,checkSponsorshipBroker}` requires
the connection owner, current generation and live broker grant.

**Required full journey:** connect account → explicit personal grant or Factory
sponsorship → reserve a bounded authorized execution/lease → observe quota,
retirement and return → reconnect/revoke when needed. Account connection,
repository policy, host capacity and native write authority remain separate.
Unconfirmed return fences the connection; credentials are not prompt context or
transferrable human takeover access.

**Allocation:** U04 owns connection/custody and personal managed CLI use; F03
owns sponsorship/capacity/limits; F10 owns pending run/lease settlement. Human
onboarding links to these owners instead of repeating enrollment.

## S04 Role communication, research, Feed and Library

**Intended model:** [worker design](../../design/factory-worker-mcp.md),
Feed direction and
component semantics. Required
capability and proposed method/profile names are distinct.

**Source trace:** `internal/factory/mcp_profile.go::{MCPProfiles,MCPPhaseOf,
MCPToolAdmitted}` defines seven production profiles plus Library Admin.
`internal/factory/control/library_maintenance.go::{selectInitialIssuePhase,
settleLibraryMaintenance}` binds curation to accepted native metadata.
`internal/web/api/factory_feed_route.go::apiFactoryFeed` checks current per-item
visibility for Project and followed-global feeds. `factory_feed_comment.go`
separates discussion, clarification, existing-work and new-issue intake with
persisted receipts and uncertain-effect reconciliation; GET does not drive work.
`factory_mcp_library.go` exposes scoped library/search/revision operations.
`sodaspaces-factory-feed.ts` and `sodaspaces-factory-library.ts` in the frontend
render actual following, comments, artifacts, knowledge and history.

**Required full journey:** observe an attributable role/controller post → reply
or inspect evidence → see the accepted route/result → retrieve reusable knowledge
→ assess provenance/applicability → curate with preserved revision and decision
authority. Research may produce a knowledge prerequisite rather than a synthetic
code PR. It must create/reuse bounded follow-up work, settle/release the blocked
parent's repository slot, distinguish negative from inconclusive results and
return accepted exact evidence for fresh parent context/reassessment. A worker
report/submission is not a controller-confirmed effect.

**Allocation:** roles are overview subsections; F04 owns research work and output
requirements, F08 owns Feed interactions and F09 owns knowledge/decision lifetime.
No public MCP catalog, persona chapter or speculative Researcher role is needed.

## S05 Execution, review, CI, integrated verification and intervention

**Intended model:** [overview](../../product/overview.md),
[Factory interfaces](../../architecture/factory-interfaces.md) and trust.

**Source trace:** `internal/web/api/factory_lifecycle.go::{apiFactoryActions,
apiFactoryRunActions,apiFactoryCommand}` and
`internal/factory/control/lifecycle.go::{PauseRepository,ResumeRepository,
StopProject}` enforce admission/retirement/settlement boundaries.
`control/completion_drive.go::DriveCompletions` tracks complete-inventory progress;
`stall_serve.go::ServeStallRound` detects stalls and admits bounded investigation.
`internal/factory/planning.go::InvestigationPlan`,
`plan_task.go::ValidatePlanGraph` and `control/task_execute.go::ImportTaskEvents`
bind executable task graphs, lanes, checks and trusted checkpoint/output imports.
`control/publication.go::PublishPass`, `review_cycle.go::CheckPass` and
`merge.go::MergePass` own candidate publication, review progression and merge.
`app_verification.go::{ApplicationVerificationEvidence,verifyApplicationReport,
driveAppVerifications}` binds observed checks, integrated version and coverage.
`frontend/spaces/sodaspaces-factory-navigation-view.ts` and
`sodaspaces-factory-view.ts` distinguish watching output from control.
`internal/factory/control/operator.go` and `cmd/soda-factory/src/main.rs` expose
recorded status/stop/reconcile, not the complete automatic admission workflow.

**Required full journey:** reserve execution within grants → code in its assigned
checkout → publish exact candidate → independent review → correction and fresh
required evidence → confirmed native guarded merge and original issue closure →
reassess dependents →
verify the integrated intent. Native CI capacity is separately managed. A changed
head/base/policy invalidates relevant evidence; missing infrastructure is a wait,
not permission to weaken checks. Pause/cancel/takeover stop new dispatch before
settling real processes, credentials and native effects. Resume/retry have
different authority and accounting consequences.

**Allocation:** F01/F05/F06/F07/F10. Keep current operator commands as support in
F10. Whole-host offline writer recovery belongs to O08. No normal manual-admission
or mandatory human-merge path, fixed single repair or disposable stage container.

## S06 Persistent Project creation, preparation and lifecycle

**Intended model:** [Projects](../../product/projects.md) and
[Project OS](../../reference/project-os.md).

**Source trace:** `internal/web/api/environments_create.go::{apiCreateEnvironment,
reconcileCreate,provisionAndSaveProject}` reserves and reconciles creation rather
than blindly recreating uncertain state. `environments_join.go` admits explicit
account-only or selected-key joining and persists observed native membership.
`environment_authority.go` separates repository visibility, code-write,
administrator and operator authority. `internal/project/profile.go` and
`lib/host/src/project/create.rs` admit the supported Rocky headless profile.
`internal/project/preparation.go` defines distinct requirement acceptance,
administrator approval, lifecycle grant, preparation and maintenance-hold state;
`environments_preparation.go` supplies its user operations.
`internal/web/api/lifecycle.go` stops Factory work under a hold and starts the
existing root without reviving prior runs. Frontend Project journey/environment
views expose retained reservations, actual state and explicit Create/Join/Start.

**Required full journey:** create/reuse a repository-bound root → explicitly Join
humans → accept exact setup source → approve/install shared prerequisites →
prepare coder/reviewer accounts/checkouts → use readiness → hold shared changes
→ stop/start or same-root refresh. Human Join is not a Factory prerequisite.
Unknown state is inspected; unrelated homes, dirty work and services persist.

**Allocation:** P02 and the distinct P03 preparation chapter; U05 links first
Join. O03 owns retained-root support refresh. Profile labels in old guides do
not authorize KDE/Fedora/graphical variants.

## S07 Sessions, browser views and terminal continuity

**Intended model:** [Spaces](../../product/spaces.md) and
[terminal lifetime](../../reference/terminal.md).

**Source trace:** `frontend/spaces/soda-spaces-entry.ts` and
`sodaspaces-workspace*.ts` retain one Workspace owner and original targets.
`sodaspaces-terminal-attachment.ts::{inspectExisting,reserveCreate,connect}`
uses exact terminal identities; `sodaspaces-terminal-actions.ts::{endTerminal,
endUnconfirmed}` distinguishes request from confirmed termination.
`sodaspaces-terminal-error.ts` separates sign-in/access from support-maintenance
failure. `internal/web/api/extension_terminal.go` reserves/looks up/acts on exact
sessions; `extension_terminal_authority.go` rechecks original account, Project,
current native session and code-write authority.

**Required full journey:** explicit terminal creation → attachment → layout and
native-page navigation → hide/detach/reconnect the exact session → confirmed End.
An uncertain reservation is inspected rather than replayed. Factory output is
read-only; takeover requires separate settled human authority.

**Allocation:** P01 navigation, P04 session procedures, D02 direct SSH and F10
takeover. Browser presence never owns an agent's execution lifetime.

## S08 Human tools, files, native Git and development services

**Intended model:** Projects/Project OS, [project services](../../guides/project-services.md),
[project CLIs](../../guides/project-clis.md) and the adopted private-app guide.

**Source trace:** `internal/web/api/access_keys.go` checks current fingerprints,
key revision and explicit last-key removal on application. Frontend Project views
separate saved-key review/application and show current SSH details/fingerprint.
`system/project/Containerfile` supplies SSH/tmux, nested Podman/Compose, mise,
tea/gh, Muse and the offline Canlang tools.
`system/project/rootfs/etc/profile.d/soda-mise.sh` selects shared mise paths;
`soda-podman.sh` selects the Project engine socket; the Project sudoers policy
keeps wheel administration inside Project userspace.

**Required full journey:** explicitly Join and open a browser terminal, or
optionally install public-key access and reach the Project over SSH → clone with
native personal Git credentials → use shared tools/files
and persistent development services → diagnose native listeners, volumes and
readiness → refresh retained tools under the separate maintenance procedure.
Agent access is explicitly assigned within the common Project. Neither terminal
End nor a merged commit restarts workloads, applies migrations or deletes data.

**Allocation:** D01–D05; U03 owns identity/offboarding effects, U04 managed AI
connections, O03 maintenance and O06 backup. Historical `guides/develop.md`
desktop claims do not override the selected headless profile.

## S09 Distribution, installation, bootstrap and activation

**Intended model:** [release](../../architecture/release.md),
[installation](../../guides/installation.md), [media](../../guides/media.md)
and [Docker](../../guides/docker.md).

**Source trace:** `cmd/soda-install/src/hostadmit.rs` admits live media separately
from installed hosts. `wizard/steps.rs` and `review.rs` bind selected disk,
password, subnet and explicit erasure; `setup/configure.rs` observes existing or
partial installation rather than blindly rerunning setup.
`cmd/soda-setup/src/setup.rs` verifies native site administration, writes the
configured operator binding and revokes its bootstrap token.
`factory_enrollment.rs` is separate from initial appliance setup.
`cmd/soda-activate/src/activation.rs` admits/writes inputs and starts services.
`system/docker/compose.yaml` and `install-docker.sh` define the privileged Linux
Docker contract and persistent config/state/home volumes. Release payload and
Docker model owners bind distribution bytes and architecture.

**Required full journey:** select distribution/release → verify → prepare host
or VM → install/review destructive target → boot/acquire selected release inputs
→ native setup/operator binding → private activation. The appliance guided
configure path requires installed FCOS; Docker uses its released installer or
lower-level setup/activation in the runtime context. Source build/stage and
installation are separate lifecycles; the public user does not assemble images.

**Allocation:** I01–I05 and U01. VM preparation reuses the common ISO install
owner; no named-provider disk-image promise. x86_64 only. Numeric hardware/firmware
support remains DEC07 rather than guessed from builders.

## S10 Private networking, TLS, Tailnet and Cockpit access

**Intended model:** [networking](../../architecture/networking.md),
[operator setup](../../guides/operator-setup.md) and trust.

**Source trace:** `system/host/config/proxy.Caddyfile` separates private origins
and TLS policy. `lib/host/src/tailnet/host.rs`, `tailnet/lifecycle.rs` and
`tailnet/lifecycle/runtime.rs` implement optional pinned Tailscale container
effects under Fountain enablement/generation grants and observed manager state;
`system/host/services/soda-tailnet-host.service` is their unit.
Cockpit PAM/config/socket overrides retain native root/operator authentication
and the 9090 listener. App ingress is a separately bound hostname/route under
the existing proxy, not shared Forgejo cookie authority.

**Required full journey:** configure native origin/advertisement → establish
private routing and client TLS trust → optionally enable/enroll Tailnet and
Project companions → verify each real endpoint from the client → refresh
advertised Git access when necessary. Address discovery, a running container,
saved setting or enrolled Tailnet node is not proof of end-to-end reachability.

**Allocation:** U02/U06, D02/D04 for their specific access tasks and O02 for
Cockpit. Initial setup hands off to the shared network owner. No public-hosting
or purchased-domain prerequisite.

## S11 Canlang development and independent private applications

**Intended model:** [Canlang applications](../../guides/canlang-applications.md),
Projects and scope. Keep runtime evidence chronology out of public instructions.

**Source trace:** Project recipe/tool closure supplies installed compiler,
preview/examples/LSP and full-profile target tools.
`frontend/spaces/sodaspaces-applications.ts` offers passive inventory and explicit
Inspect/Install/Start/confirmed Stop with Project association.
`internal/web/api/applications.go` and `application_actions.go` separate
operator-only installation/replacement from app-administrator lifecycle and
check exact revision/generation. `lib/host/src/application/runtime.rs::{create,
run_install,run_start,run_stop,run_replace,same_storage}` and `ingress.rs` bind
exact images, app-only data and private routes.
`system/host/services/soda-canlang@.service` owns the independent runtime lifetime.
`application/snapshot.rs::{backup,restore}` takes the writer lock, inventories
authoritative state and restores only into fresh storage.

**Required full journey:** compile/check/examples in separate development state →
assemble compatible bundle/image → stopped app-local owner/member provisioning
→ operator registration/install → configure private client access → explicit
Start/readiness → compatible same-data replacement → stopped backup/fresh restore.
Project Stop does not stop the app. Forgejo/provider identities do not become
app-local accounts. Factory merge does not deploy; schema-incompatible replacement
is not a migration or data rollback.

**Allocation:** D05 development, O04 independent app administration, U02 private
access and O06 backup. This avoids both the old handbook's complete omission of
apps and a duplicated app-specific network/backup manual.

## S12 Maintenance, updates, backup, host recovery and removal

**Intended model:** release, operator setup, Project OS and scope.

**Source trace:** `cmd/soda-project-maintain/src/release_operation.rs::{run,
reconcile_boot,run_operation,rollback}` and `release_extensions.rs` close
admission, include retained stopped roots, reconcile operation journals and bind
support/extension transitions. `post_commit`/`Failure::exit_code` distinguish core
commit from uncertain extension readiness, including exit 75.
`cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs` writes verified separate
Forgejo/Soda dumps and globals; the timer/retention contract does not quiesce
Project/app state or provide simultaneous cross-database consistency.
`soda-pg-restore.rs` requires explicit destructive intent.
`cmd/soda-forgejo-domain/src/domain.rs` and `marker.rs` own stopped/offline native
writer recovery. App snapshot mechanics are traced in S11. Retained Project roots
also live in Podman storage; copying `/home` alone is not a whole-appliance backup.

**Required full journey:** identify the actual state/writers → stop managed work
and take consistent backups → admit a supported release or same-root refresh →
perform/reconcile the exact operation → verify native and extension state →
reopen admission. Restore and software rollback remain distinct. Native data
removal needs explicit actor/target; no destructive recreate/rebuild recovery.

**Allocation:** O01–O03/O05–O09; F10 retains per-run settlement. Docker lifecycle
is separate from FCOS upgrade qualification. No bespoke Updates platform,
coordinated one-click backup, fleet repair or archival programme is inferred.

## Independent boundary challenges and resulting choices

| Challenge | Evidence examined and objection | Reconciled ownership |
| --- | --- | --- |
| Factory versus Workspace | Request/Feed/library frontend and Project journey/terminal controllers; a single “Spaces” manual would mix spending, accepted scope and passive session views | F02/F04/F08/F09 own work; P01/P02/P04 own navigation, Project and session lifetime. Retain a separate P03 because accepted setup source, admin effects and coder/reviewer readiness are distinct tasks. |
| Initial setup versus Factory enablement | `soda-setup/setup.rs`, `factory_enrollment.rs`, policy/sponsorship APIs and provider UI; a single operator setup page would obscure separate grant owners | U01 bootstrap, U04 account custody, F03 standing policy/capacity/sponsorship. Personal connection is not Factory spending authority. |
| Project service versus app versus backup | App runtime/actions/unit/snapshot source and nested Project engine; putting apps under development services would imply the wrong lifetime and custody | D04 Project workloads; D05 app development; O04 independent installation/lifecycle/app identity; U02 network; O06 consistent backup/fresh restore. |
| Run recovery versus host recovery | Factory lifecycle/operator source and release journal/offline native-domain source; “reconcile” has different scope and writer prerequisites | F10 exact run/native-effect settlement; O08 whole-host/release/offline writer procedure. No blanket retry instructions in diagnostics. |
| Retained-root tools versus image update | Project root/persistence model and explicit maintenance source; updating a default image cannot refresh existing writable roots | O03 gets a distinct same-root maintenance owner; O05 handles qualified appliance transitions. |
| Public routes versus folder overhaul | Website discovery/profiles plus source consumers; preserving a filename is not necessary to preserve its slug, while preserving obsolete subject pages duplicates content | Retain 22 subjects/slugs; explicitly retire dashboard/forgejo routes and update consumers. Keep all planning metadata outside `docs/public/`. |

## Website integration evidence

The website's `scripts/docs-sync/source.mjs::discoverDocumentation` permits only
the root README, numbered sections with direct Markdown children and optional
image assets. Its profiles define globally unique flat slugs and `index`.
`snapshotFromRepository` requires clean, committed public source and verifies
the source repository identity; directory snapshot generation can validate a
draft without publishing or changing generated website files.

The existing compatibility anchor is keyed to the old source filename in
`scripts/docs-sync/profiles.mjs`, not just the `software-factory` slug. A folder
move therefore requires deliberate fragment handling. Authored website consumers
also use the old `dashboard` title, `/docs/forgejo` links/fragments and the
operator-setup trust anchor. Exact integration owners are listed in the cutover.
