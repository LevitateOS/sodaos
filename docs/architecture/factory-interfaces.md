# Factory implementation interfaces

This document owns the **target component interfaces** for the automatic factory:
commands, durable records, transitions and execution bindings. These interfaces
are not yet implemented. The [product contract](../product/overview.md),
[Project model](../product/projects.md), [Spaces contract](../product/spaces.md)
and [trust model](trust.md) own their respective behavior and security rules;
the [API reference](../reference/api.md) continues to describe existing routes.
Package placement belongs to [Go ownership](../development/go.md#factory-interface-placement).

## Service and persistence placement

Run one `factory/control` coordinator inside the existing `soda-dashboard`
backend. It uses that service's `store` database for factory policy, acceptance,
attempts, runs and commands alongside Project records. `web` constructs it and
owns explicit start/shutdown wiring; `web/api` admits requests and invokes it.
Constructing an HTTP handler does not launch work. Operator tools address this
same coordinator; retire their independent execution engine and `execution.db`.
Startup must obtain exclusive coordinator ownership of this database and finish
reconciling outstanding work before reopening dispatch. This is one appliance
process, not a replicated scheduler or leader-election protocol.

The Identity Broker keeps its existing separate service, database and credential
custody. Fixed host operations orchestrate lease acquisition and native launch;
the coordinator receives metadata, never provider credential delivery. The host
persists only protected mechanical preparation/start/stop receipts, not factory
policy or a competing run ledger. Fountain retains its own native records and
operation ledger. No transaction spans these services.

The backend retains the unprivileged `host/publish` executor, adds Git to its
image and uses a private publication directory under its existing writable data
root. Native actor secret files and background service admission stay outside
Projects. Existing read-only root, dropped capabilities and private socket
boundaries remain applicable. Sharing the backend avoids another daemon and
cross-service factory/database coordination, but also shares its availability and trusted
native-actor boundary. Git is an additional backend dependency.

This choice follows the actual startup/store wiring in
[`cmd/soda-dashboard`](../../cmd/soda-dashboard/main.go), the separate ledger in
[`factory/control`](../../internal/factory/control/config.go), existing
[`host` lease delivery](../../internal/host/identity.go), and the current
[backend image](../../system/containers/dashboard/Containerfile). Three fresh Jev
consultations supported backend placement and host delivery; that advice does
not prove either implementation. The [planning evidence](../development/factory-implementation-plan.md#interface-definition)
records their scope and source revisions.

## Common identities and request rules

- Native repository, issue, comment, user, PR, review and check IDs retain native
  meanings. Encode native integer IDs as decimal strings in browser JSON.
  Project ID identifies the persistent Soda Project; full container ID identifies
  its current native incarnation. Never substitute a repository name or PID.
- Soda policy/record revisions, accepted-input IDs, native mutation revisions,
  provider credential generations and process invocation IDs are distinct fields.
  An `AuthorityRef` binds repository policy, operator grant, sponsorship, Project
  lifecycle grant, accepted inputs and preparation revisions. A run also binds its
  immutable assignment, attempt, role, source/candidate/base and selected harness
  identity.
- A mutating command carries a client-generated `command_id`, expected current
  Soda revision and a typed payload. Acceptance uses its existing `decision_id`
  and expected predecessor. Persist admitted identity, normalized payload digest
  and result atomically with the decision. Same ID and payload returns that
  decision; reuse for different content conflicts. Do not replay authorization
  merely because the HTTP response was lost.
- Native browser admission supplies the caller and route/repository binding;
  body fields cannot select an authorizing human. Reads and replayed responses
  still require current visibility. Maintain existing CSRF/origin, strict decode,
  body/stream bounds and private transport admission. No public payload contains
  credential bytes, secret filenames, host paths or a privileged executable.
- Return `400` for invalid input, `401/403` for admission/authority failure,
  `404` for absent or undisclosable resources, `409` for changed source/identity
  conflicts and `412` for stale Soda revision preconditions. A durable asynchronous
  command returns `202` and a command/status reference; `200/201` represents a
  completed read/decision. `503` means unavailable evidence or service, never
  evidence of mutation failure. Responses include bounded reason codes and current
  revisions; clients refresh rather than silently adopting newer inputs.

## Soda API and UI commands

The following are logical routes beneath Soda's existing authenticated extension
API. The extension namespace supplies the public prefix; do not expose a second
unauthenticated backend entrypoint. `R` is native repository ID, `I` native issue
ID, `P` Project ID, `A` attempt ID, `X` run ID and `C` command ID.

| Route | Input / result | Admission and UI action |
| --- | --- | --- |
| `GET /api/repositories/R/factory` | Policy, effective grants, bounded queue summary, revision and missing-authority reasons. Private provider details are redacted by viewer. | Native repository visibility; factory status/settings contribution. |
| `PUT /api/repositories/R/factory/policy` | Command envelope plus enabled/paused state, target branch, role harness family (`harness`), pinned version (`harness_version`) and model selections, configured native actor bindings, nonempty required check set, supported merge method and repository limits, including optional `attempt_limits` with `active_minutes` and `correction_cycles`. Omitted attempt limits resolve to the [operating defaults](../product/overview.md#concurrency-and-usage-limits) when policy is recorded. | Current native owner/admin; **Enable/Configure/Pause factory**. Actor selection references administrator-enrolled bindings; it cannot enroll a credential. |
| `PUT /api/factory/capacity` and `PUT /api/repositories/R/factory/operator-grant` | Appliance caps, or repository resource permission, with separate revisions. | Configured operator; capacity and repository permission controls. Enlarging one grant cannot enlarge another or authorize Project creation on its owner's behalf. |
| `PUT /api/repositories/R/factory/environment-grant` | Standing create/start permission, supported profile, current owner and Project reference when present; revisioned command. | Current human repository owner; **Allow factory environment setup** under existing Project creation rules. Organization-owned creation stays unsupported. |
| `PUT /api/repositories/R/factory/sponsorships/{connection}` | Canonical broker grant reference, permitted Project/roles, explicit usage allowance and active/withdrawn state. | Current connection owner; **Sponsor/Withdraw**. Uses existing connection/grant administration, with no new enrollment or credential API. |
| `GET /api/repositories/R/factory/issues/I` | Current acceptance and native source links, readiness evidence/blockers, attempt, usage, native PR/check/review/operation links, pending controls. | Native issue visibility; **Factory work** view. |
| `POST /api/repositories/R/factory/issues/I/acceptances` | `decision_id`, expected acceptance predecessor, observed native revision and complete selected native source versions/digests, direct dependency occurrences/outcomes, resolution references. Returns immutable receipt. | Current code-write maintainer; **Adopt issue**, **Accept requirements/answer**, **Confirm resolution**. Show exact selected native text and changes before sending. |
| `POST /api/repositories/R/factory/issues/I/actions` | Command envelope, target attempt/revision and one of `pause`, `resume`, `cancel`, `retry`, `takeover`. Takeover identifies the admitted member's Project destination, not a filesystem path. | Current code-write maintainer; named work controls. Human membership/join remains separately required for takeover. Retry is explicit fresh-attempt admission after prior work is reconciled. |
| `GET /api/factory/attempts/A` and `GET /api/factory/commands/C` | Stage, remaining allowances, run/evidence/native result references, requested versus confirmed stopping/cancellation. | Current visibility of the bound repository; refresh after pending commands or lost replies. |
| `GET /api/factory/runs/X/output` | Read-only bounded stream, run/process identity, output cursor, explicit truncation/gap and terminal status. | Current code-write authority at attach and during streaming; **Watch agent** in Spaces. Reject input, resize, signals and human-terminal End frames. Reattachment never launches or resumes an agent. |
| `GET /api/projects/P/preparation` | Current approved preparation ID, maintenance state, role readiness observations and failures. | Authorized Project/repository access; environment readiness view. |
| `POST /api/projects/P/preparation/acceptances` | Decision ID, expected requirement predecessor and exact approved-base source/configuration definitions and effective input digests; returns a maintainer acceptance receipt. | Current code-write maintainer; **Accept environment requirements**. This does not authorize privileged installation. |
| `POST /api/projects/P/preparation/actions` | Command envelope with `hold`, `inspect` or `approve`; expected Project/preparation revision and requirement acceptance ID. Approval records reviewed privileged effects, exact installed tool/service/resource results and readiness evidence. | Current Project administrator, verified through native Project account/privilege mapping, or operator under existing administration authority; **Prepare environment**. Owner status alone does not replace either requirements acceptance or administrative authority. |

Preparation controls coordinate maintenance and record approval. They do not
install arbitrary root commands through HTTP: administrators use the existing
native Project setup path, and the helper performs only its fixed role/layout
operations. Approval clears a maintenance hold only after affected work has
stopped, preparation is verified and other authority remains valid.

Existing Project create/start/stop, Join, human terminals/SSH, provider enrollment,
Tailnet and operator interfaces remain with their current owners. Adapt their
lifecycle checks to outstanding factory work; do not add factory-specific copies.
No control reports **cancelled**, **stopped** or **taken over** while the relevant
native effect or process retirement is still unresolved. A committed write remains
visible even if its initiating command was subsequently withdrawn.

The blocker view displays an agent's structured question, consequence, requested
resolution and exact source references. For a product decision, the maintainer
posts the question and answer through the native issue interface, then selects
both exact comment revisions in Soda's acceptance action. Proposed prerequisites
likewise become native dependencies before adoption. Factual resolutions can use
already approved source evidence under the product rules. Do not add unattended
issue-comment writes or treat a Soda-local answer as an accepted native comment.

Retained `soda-factory` operator commands use a private Unix endpoint on this same
backend: `POST /operator/factory` with the command envelope and `status`, `stop`
or `reconcile`. Package its socket outside Projects and the Fountain extension
host, restrict filesystem access and verify native peer credentials against the
configured appliance operator/root identity. It is not reachable through the
public HTTP mux. Configure the OS peer UID explicitly and verify it in the deployed
socket/process namespaces; `config.OperatorID` is a native forge user ID and must
never be used as an OS UID. Record the OS principal separately, without inventing
a native human author. Status returns bounded operational metadata, not repository
contents or credential data; stop withdraws operator execution permission and
uses the same coordinator cancellation path. Reconcile inspects recorded work,
retires remaining execution and settles accounting; it cannot launch, retry,
spend or publish. Resumption after withdrawal requires the authorized product
controls. These commands cannot supply a
human native identity, accept requirements, change repository policy, enlarge
sponsorship or clear Fountain's native reservation. Such actions use their owning
native/UI or offline operator contracts. Remove the old CLI's manual admission,
execution and independent database behavior.

## Soda durable records

These are record families and required fields, not a second schema implementation.
`store` owns tables, indexes, transactions and record CAS. Domain records live in
`factory`, `project` and `identity` as assigned below; consumers use those types.
Reject obsolete unreleased formats rather than migrating or reading both models.

| Owner / record | Required identity, contents and invariants |
| --- | --- |
| `factory.RepositoryPolicy`, `OperatorGrant`, `Sponsorship` | Repository, granting native identity, revision, active/withdrawn state and the corresponding authorized fields. Sponsorship references `identity.Grant`, connection and credential generation; it does not copy credential custody. Record the separately accountable execution sponsor used as broker `ActorID`, not an issue author or native publication bot substituted by convention. Appliance capacity is separately revisioned. |
| `factory.Acceptance` | Immutable decision ID, predecessor, host-admitted approver, repository/issue, observed native revision, exact sources/digests and the complete approved prerequisites/outcomes/resolutions required by the [acceptance contract](../product/overview.md#accepted-requirements-and-native-records). Preserve attribution and selected text without bypassing native visibility. |
| `factory.IssueControl` / assessment | One current control record per repository/issue: revision, current acceptance, pause/cancel/takeover latch, readiness classification, reason and assessment fingerprint. Assessment binds native evidence, authority and preparation versions; unchanged blocked inputs do not buy another assessment. Native issues and dependency graphs remain authoritative. |
| `factory.Attempt` | Immutable attempt ID and issue link, accepted-input/authority references, Project ID, fixed allowance snapshot, consumed active intervals/correction cycles, stage and CAS revision. Reserve at most one unfinished current attempt per issue and at most one executing attempt per repository; blocked/paused work can retain an attempt without holding an execution slot. |
| `factory.Run` / assignment | Immutable run ID, attempt/role, canonical prompt/assignment digest and protected input snapshot, source/candidate/base, preparation and harness identities, execution sponsor/connection, lease reference, exact process binding, preparation/start/stop/return facts and result/evidence references. One executing or unreconciled run per attempt; no reuse of a consumed run ID. A new conversation is a new run within remaining attempt allowances. |
| `factory.Command` / native dispatch | Immutable command ID plus admitted principal (native human or separately identified appliance OS peer), payload digest, target/revision and durable outcome. A dispatch additionally binds one native operation ID, immutable intent/digest, authority references, submission/cancellation status and last attributable Fountain receipt. These are local intent/outcome references, not a duplicate native operation authority. |
| `factory.UsageReservation` | Attempt/run/connection identities, reserved slot and bounded time allowance, lease start/end facts, cumulative attempt intervals and rolling provider usage. Keep open or uncertain intervals charged/unavailable until reconciled. Pausing, editing or retrying cannot erase prior usage. |
| `project.FactoryLifecycleGrant` / `Preparation` | Separate current-owner standing create/start grant. Preparation binds Project/incarnation, protected repository/setup/tool/service/resource inputs and digest, the code-write maintainer's requirement acceptance ID/revision and separately the native Project administrator's privileged-effect approval/verification, maintenance hold and per-role readiness observations. Reference retained checkout ownership and native resource assignments; do not introduce another editable environment package manager. |
| `identity.Lease` / execution acquisition / grant records | Existing canonical broker records extended with unique `(kind, execution_id)` acquisition and immutable acquisition digest. Retain nonsecret terminal acquisition identity after lease return/deletion; a consumed or closed execution cannot acquire a new lease. Return metadata by that identity; same ID with changed request refuses. The broker remains authoritative for connection availability, grant revocation and credential return. |

All command admission, attempt/run/usage reservations and dispatch registration
check expected revisions in a local transaction. Do not hold it across native
HTTP, Git, host calls or broker acquisition. Store exports concern-specific
methods such as admit acceptance, reserve run, register dispatch and withdraw
authority; callers do not get a SQL escape hatch.

The scheduler reserves Soda capacity and remaining provider allowance **before**
asking the host to acquire a broker lease. Acquisition is a separately reconciled
boundary, not an atomic cross-database reservation. On conflict, release only
confirmed unused reservations and queue the work; on uncertainty, look up its
existing execution ID. Refresh connection/grant metadata before dispatch. Broker
serialization may further constrain the product's configured limits.

## Transitions and ordered cancellation

Readiness is a view over acceptance, evidence and control: `not_authorized`,
`blocked`, `queued`, `active`, `needs_intervention`, `cancelled`, `completed`.
Stage, stopping, resource occupancy and cancellation are separate facts; do not
encode every combination in one status string.

| Transition | Preconditions and durable effect |
| --- | --- |
| Observe → assess/queue | Treat webhook/poll as a hint; use authoritative native snapshots and current authority. Verified original creation or explicit acceptance supplies inputs. Reconcile relevant enabled repositories with bounded work. Native edit/revert and dependency occurrence changes are checked as specified by product rules. |
| Queue → prepare → assess/code | Atomically reserve eligible work and allowances. Prepare exact Project/role/source; require readiness before a provider lease. Persist run assignment and launch intent before any host call. Preparation failure records a blocker/intervention and releases only confirmed unused resources. |
| Code → publish → PR → review | Stop and account for the coding run, validate/export its exact candidate, then register each native write separately. A distinct reviewer run receives the exact candidate and fresh independent Git state. Stage advances only on attributable native effects, not an agent's claim or HTTP timeout. |
| Review → fix/reverify or CI | Persist structured verdict/evidence for exact head/base. Findings go to a coder run; changed code consumes the defined correction/reverification allowance and invalidates prior evidence. Agent output is untrusted bounded structured data and cannot waive policy or mark checks passing. |
| CI → merge → completion | Require complete nonempty native check/review evidence and all current grants for the exact candidate/base. Submit the conditional merge. `committed` alone is not completed factory work: await native completion and confirmed issue outcome/reachability before recording completion and reassessing dependants. |
| Blocker/edit/grant loss/maintenance/pause | Close affected dispatch, preserve work and consumed allowances, and record pending stop/cancel. Remove an execution slot only after the assigned process is retired. Blocked/paused intervals stop charging active attempt time once affected activity is confirmed stopped. Active CI waits count under the product time rule. |
| Cancel / takeover | Latch the request; stop runs and cancel outstanding native operations. Cancellation remains sticky across issue reopen. Takeover waits for process retirement, credential accounting and known native outcomes, then copies retained work into the human's separate checkout. It does not transfer an agent terminal or credential home. |
| Resume / resolved blocker | Revalidate the accepted inputs, environment and all grants. Continue only within the same remaining attempt allowances. Never restart an old process or silently resume its CLI conversation. |
| Explicit retry | After the previous attempt's effects and resources are reconciled, create a new attempt with a new allowance snapshot. Preserve cancellation attribution and rolling provider usage; retry cannot silently adopt changed requirements. |
| Timeout/crash/unknown effect | Keep the recorded identities; inspect host/broker/native operation state, stop and account for any surviving execution. Confirmed exhaustion/failure becomes intervention; unknown ownership/effect remains fenced. Do not infer failure from absence of a response or launch a replacement while an earlier boundary is unresolved. |

For each stage write or launch, the coordinator checks current authority and
dispatch-open state and records its immutable ID in the **same local transaction
order** as withdrawal. Withdrawal atomically closes dispatch and captures all
outstanding IDs; persist cancellation delivery before returning its command
receipt. Then deliver/reconcile host stop and native cancellation by those IDs.
A delayed registration cannot escape this set, and native cancellation tombstones
can precede delayed submission. Local withdrawal acknowledges **requested**, not
that it has already won a native race. A native commit that won first is reported
with its actual receipt and cancellation outcome.

Accepted-input admission reads the exact selected sources and permissions inside
the native revision bracket, then CASes the expected Soda predecessor. A native
edit immediately afterwards can invalidate that receipt; it cannot authorize a
later stage using stale evidence. Revalidate before dispatch and bind the new
native revision to each conditional operation. Do not imply an atomic transaction
between Soda acceptance and the native database.

## Project execution and broker interface

Use canonical Project, factory and identity records over the existing private
`host.Client` transport. The daemon admits and routes; execution leaves own the
fixed operations. Browser requests cannot call these endpoints directly.

| Host operation | Request / response boundary |
| --- | --- |
| `FactoryPrepare` | Immutable preparation ID/digest, Project ID, attempt/run/role, approved preparation revision, operator-bounded resource/deadline limits and exact source identity with a verified source bundle. Resolve fixed accounts and helper-owned checkout paths; execute approved private setup unprivileged. Return preparation receipt, full container identity, source identity and role readiness. Never accept arbitrary root command, UID, image, mount or destination path. |
| `FactoryLaunch` | Run ID, immutable assignment/digest, preparation receipt and canonical `identity.AcquireRequest` with fixed limits/role. Persist admission before acquiring. Obtain the execution-unique lease, reserve a credential-waiting unit, persist/attest its binding, register delivery and consume start once. Return lease/binding and phase metadata only. |
| `FactoryInspect` | Run ID and expected recorded binding; return preparation/start/stop state, exact observed container/unit/incarnation, exit/result/output references and credential-return status. No PID-only success. |
| `FactoryStop` | Stable run/preparation ID plus known binding when available. Persist terminal stop intent even before preparation/launch is observed; refuse all subsequent work under that ID. Retire only its native process boundary, close/reconcile its broker execution and report requested/confirmed/uncertain state. A changed incarnation refuses instead of targeting the replacement. |
| `FactoryExport` | Stopped run, known binding and exact candidate identity; return a bounded native Git bundle and source/result metadata. The unprivileged publisher independently validates it and protected paths before publication. No agent Git config/hooks or general native credential accompany the bundle. |
| `FactoryTakeover` | Reconciled attempt/run and admitted human Project membership; derive the human destination and copy retained source into independent Git state. Return the human checkout/session reference; exclude provider homes, role Git configuration and credentials. |

Preparation must be addressable before completion, with inspect/stop using its
recorded identity as well. Approved setup can execute code and cannot be blindly
rerun after an uncertain reply. Store its intent before execution and its phase
afterwards; interrupted preparation requires inspection/stop and explicit new
preparation once safe. Helper receipts are protected persistent files associated
with the retained Project/run and survive service or container restart. An absent
transient systemd unit is not proof that its prior start never happened.

The private broker interface retains `Acquire`, `Register` and lease reconciliation,
and adds execution-keyed `GetExecution(kind, execution_id)` and
`CloseExecution(kind, execution_id)`. Lookup returns the immutable acquisition
digest, lease/binding metadata and pending/live/terminal state without credentials.
Close creates a terminal acquisition tombstone even before acquisition arrives,
prevents subsequent acquisition/registration and reconciles any existing lease.
It reports pending until custody and process retirement are established. Same-ID
`Acquire` never grants a replacement lease after return or closure. Keep these
records in `identity`/`store`, without storing another credential copy.

For launch, order these boundaries:

1. Persist local run/launch/usage intent; the helper durably accepts that ID or
   its earlier stop tombstone before side effects. Repeated identical requests
   inspect the recorded progress, never re-execute it.
2. Acquire or look up the broker lease by `(factory, run_id)`. Prepare one waiting
   native unit and record its observed binding before credential registration.
3. Broker registration re-attests the exact unit/container/role and delivers only
   for that lease/binding. Store restricted credentials outside checkout; consume
   the protected start marker before starting the fixed CLI entrypoint. A crash
   in that boundary may leave an unstarted run requiring intervention, but cannot
   justify a second start.
4. On lost registration/start response, inspect and retire/reconcile the same run.
   Metadata lookup never redelivers credentials. Do not introduce automatic
   delivery replay or CLI continuation to repair the observer.
5. Broker `Validate`/`Stop`/`Finish` use the same exact binding; `Finish` captures
   provider state privately after the assigned descendants are quiescent. Unknown
   retirement or credential return keeps the connection unavailable. Host or
   coordinator restart reconciles outstanding receipts before admitting more work.

Serialize stop with preparation and each native side effect, including physical
start. Persist the stop barrier before closing broker acquisition and retiring
the unit; late preparation/launch calls cannot pass it. An acquisition or native
creation already in flight keeps stop pending until it is accounted for and any
result is retired. A consumed start marker followed by an absent unit is
insufficient: a delayed spawn must not occur after confirmed stop. Use short
per-execution critical sections for native effect admission and track unfinished
phases durably. Do not hold a lock across broker `Register` that its callback to
host `Validate` needs. After a crash, unresolved phases remain fenced until their
effects can be established, rather than treating a vanished caller as completion.

Reuse `identity.Binding`: `Kind=factory`, `ID=Run.ID=Lease.ExecutionID`,
`Project=full container ID`, `Login/UID/GID=derived fixed role`,
`Scope=fixed factory execution discriminator`, `InvocationID=observed native
systemd incarnation`, `CredentialRoot=helper-derived private path`, and
`Generation=broker credential generation`. `Lease.ProjectID` remains the logical
Project ID. Record the exact unit identity as part of the protected binding;
container ID, PID, UID or credential generation alone cannot identify a process.
Disable role-controlled cgroup migration or alternative service launch that could
escape targeted retirement. Never reuse disposable-container kill/capture here.

Preparation/source/assignment changes require a new preparation/run identity.
Maintainer prompt acceptance is separate from harness input construction: the
controller builds the bounded role prompt from accepted inputs, exact approved
source instructions, authoritative evidence and prior findings according to the
[prompt contract](../product/overview.md#prompt-assembly), records its digest,
and supplies it through protected run input. Agent findings/results flow back as
validated data; they are not executable host commands or native authority.

## Generic Fountain integration

Keep existing browser `NativeClient` admission separate from background admission.
Extend the existing private `/v1/native` dispatcher and SDK, rather than creating
a Soda-specific native service. The target SDK surface is:

```go
RuntimeBackgroundClient() (BackgroundClient, error)
BootstrapServiceBackground(context.Context, ServiceBridgeOptions) (BackgroundClient, error)

type BackgroundClient interface {
    ReadNativeRevision(context.Context) (NativeRevisionObservation, error)
    ReadSnapshot(context.Context, CredentialFile, SnapshotRequest) (NativeSnapshot, error)
    SubmitOperation(context.Context, CredentialFile, OperationIntent) (OperationRecord, error)
    GetOperation(context.Context, string) (OperationLookup, error)
    CancelOperation(context.Context, string) (OperationRecord, error)
}
```

These are proposed signatures. `CredentialFile` is an SDK-local restricted secret
input, never an arbitrary path sent for privileged host reading. Runtime/service
constructors retain ephemeral admission in memory; the native host maps verified
runtime or service peer evidence to its durable installation. Submission and
snapshot reads present the bound native actor credential privately. Lookup and
cancellation use owning installation admission even after actor authority loss.
The [background-authentication contract](trust.md#background-authentication)
owns the exact enrollment, peer and credential binding requirements.

`ReadNativeRevision` returns atomic revision plus idle/busy state. `SnapshotRequest`
selects bounded supported native families: issues/comments/dependencies and
permission/version evidence; PRs/reviews/checks; repository policy/refs. Pages carry
completeness and visibility evidence. Soda brackets the full evidence set with
equal idle observations; a changed/busy observation, missing page or inaccessible
required record cannot authorize work. This exposes native facts, not factory
acceptance or readiness.

`OperationIntent` has the common fields and exactly one of the four typed payloads
owned by [conditional native mutations](trust.md#conditional-native-mutations):
ref publication, PR creation, final review submission or merge. `OperationRecord`
returns immutable intent/provenance, execution status, effect, cancellation,
completion, bounded refusal reason and kind-specific receipt. `OperationLookup`
distinguishes `not_observed` from a saved operation; absence does not by itself
authorize retry. Installation IDs and private execution secrets are host-derived,
not payload-selected. Publication also needs SDK transport setup for one registered
operation and verified native receive origin, with restricted credentials and no
credential-forwarding redirects; it is not a fifth mutation operation.

| Native record family | Required durable boundary |
| --- | --- |
| Installation / native actor binding | Installation UUID survives restart/replacement/disable and is retired on removal. Administrator enrollment binds installation UUID, native token ID, actor ID, repository ID and operation kind; each operation separately captures the current credential-generation fingerprint. The binding is independently revocable. Reuse native credential records; no new generic application-grant service. |
| Mutation reservation | Single native revision and exclusive owner/generation for ordinary and conditional writers; private execution verifier plus allowed resource/phase/effect. Persist ownership before any protected native effect. |
| Operation | Unique `(installation_id, operation_id)`, immutable submitted intent/digest, or pre-submit cancellation tombstone; bound actor provenance, admission/execution state, effect receipt, cancellation and completion. Never reuse terminal identity. |

SDK DTOs stay in Fountain `sdk`; dispatch stays in `routers/web/extensions`.
Generic native persistence/services may live in `models/nativeoperation` and
`services/nativeoperation`; a dependency-neutral `modules/nativeoperation` carries
trusted execution context into participating native writers. These are target
packages, with no Soda policy types. Preserve existing native writer/service
ownership and transaction semantics rather than moving collaboration into an
extension package. The [reservation and recovery contract](trust.md#conditional-native-mutations)
owns writer coverage, atomic SQL receipts, Git prepared admission, quiescence,
offline release authority and all terminal/uncertain transitions.

No runtime success is claimed by this specification. The existing
[acceptance criteria](../development/testing.md#factory-acceptance) still require
native binding/restart/race checks and the composed provider, Project, browser and
collaboration journey. Interface qualification tasks belong in implementation
sequencing; another release build is not required to finish this planning step.
