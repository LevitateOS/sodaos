# Native Forgejo integration

`forgejo` connects Soda's backend to native Forgejo identity, observations and
conditional operations. It is an internal Go integration package; repository
users interact through native Forgejo and Soda's project/factory controls.
Forgejo owns its data and authorization rules. Soda uses supported HTTP and
extension interfaces rather than reading Forgejo's database.

The [issue-intake boundary](../factory/issue_intake.md) now interprets retained
Fountain grants and attaches them to neutral semantic requests only at dispatch.
The store retains the opaque proof and no longer imports the SDK. Claim decoding
does not authenticate a bearer or replace native host admission.
`FountainIdentities` decodes claim actors/repositories through lookup-only
mappings on the adopted host. `FountainIssueIntake` shares `ServiceBackground`
and returns canonical issue scope while preserving the native proof, exact
mutation/precondition values and validated receipt facts. Missing mappings refuse before a credential-bearing
request; local identity numbers never substitute for native authorization IDs.

## REST client

The dashboard constructs `forgejo.New(config.ForgejoInternalURL)`. A read of the
current token owner has this shape:

```go
client := forgejo.New(internalOrigin)
user, err := client.Current(ctx, token)
```

`internalOrigin` is the configured native origin, and `token` is loaded through
the caller's private credential path. `Current` calls `/api/v1/user`.
`RevokeCurrentToken` revokes that token; `OwnPublicKeys` reads the bound actor's
native public keys. The default client uses a 30-second timeout, refuses
redirect following and bounds decoded responses. Handle transport, HTTP and
invalid-response errors through the package's error vocabulary.

## Factory adapters

The native SDK repository read now carries the configured `DefaultBranch` as
navigation metadata. Fountain's admitted repository access maps that fact to both
canonical identity responses as `default_branch`; an empty string means unknown
or unconfigured. It does not prove a Git ref exists, pin a tip or grant mutation
authority. Current code visibility and native session checks still gate every
reply. The browser host's `fountainRepositoryNavigation` decoder supplies this
fact for the later provider-specific Markdown/route binding.

`HumanReads` projects the current request's human-admitted issue and ordinary
discussion reads into the defining `collaboration` records, through `FountainIdentities` on the adopted
host. It never substitutes an automation PAT. Native and canonical actor/repo
keys resolve before provider access. Native IDs stay distinct from issue numbers;
authors/assignees receive host-scoped canonical identities. Successful reads
refresh current repository names without changing their identity. Bodies are
explicitly requested in full; clipped composer previews cannot become complete
product observations. Capabilities stay null/unknown until their per-action
provider facts exist, with no invented connection for a Fountain session.

Issue lists support open, closed and all states. Issue/PR discussion comments
retain separate typed parents and native parent IDs; a PR's ID is not its native
issue row ID. Their shared scope check resolves the adopted namespace and current
human before resource access. The native callback reads only ordinary comment
rows, retains full Markdown and rechecks the parent and matching issue/PR unit
after I/O. Reviews, inline threads and lifecycle events stay separate resources.

The paired Fountain SDK now exposes `Authority.RepositoryCommentSnapshot` for
the pending human comment writer. This native preparation read binds the current
human, exact issue/PR parent plus underlying issue row, current comment capability
and a stable idle native mutation revision. It brackets both resource/permission
reads, refuses busy/revised/session-changed state and writes no comment or
operation. These are native conditions which the future canonical/journaled
consumer must translate and recheck at commit. They do not use the automation
completion-note operation or confer execution/write authority. Host and SDK
must be rebuilt together; no production human-comment endpoint is enabled yet.

The paired host also has a native `SubmitHumanComment` producer under
`services/nativeoperation`. It uses the existing conditional journal/reservation
and an exact preparation revision, with live human admission and current native
parent/permission checks inside the atomic comment/receipt transaction. Replays
retain the original receipt; recovery does not publish another comment or repeat
uncertain notification completion. It has no background credential or enrollment.
The typed SDK now consumes it through `Authority.SubmitHumanComment`,
`GetHumanComment` and `CancelHumanComment`, with the separately declared
`native.repository.comments.write` capability. IDs are scoped to native human
and repository; the host derives and rechecks the live installation. Exact
request/receipt wire definitions live once in the SDK. Native POST admission is
required for submit/cancel; receipt lookup and cancellation can recover a human's
own operation after repository permission loss. A closed runtime releases no
receipt, and lost transport replies do not prove absence of an effect.

The canonical application/HTTP comment consumer now uses these commands through
the captured command journal and live admitted adapter. The Fountain frontend
bridge admits exact issue/PR comment submission and scoped command lookup/cancel
over the private Unix transport, with the larger escaped-writing allowance only
on submission. It resolves canonical
identities before native access, pins the installation and reconciles lost replies
under the original namespace. Installed UI and desktop gateway qualification
remain later work; the shared HTTP boundary is described in
[`internal/factory/httpapi`](../factory/httpapi/README.md).

The paired SDK also exposes `Authority.RepositoryEditSnapshot`, with exact
original title/body digest/content version and separate native title/body
capabilities. Its `SubmitHumanEdit` service producer atomically changes requested
fields, native history and the receipt through the existing conditional journal.
Live current-human and field-specific permission checks run at commit; offline
recovery never repeats an edit or uncertain notification. Canonical request and
receipt types live in `sdk/native_human_edit.go`. Typed SDK submit/get/cancel now
cross the private callback under `native.repository.discussions.edit`, with
human/repository-scoped operation IDs and captured installation equality before
submit/cancel and again at commit. Lookup includes the current installation even
for absence; an earlier installation's outcome remains in its original namespace.
Live current-human admission also gates cleanup after repository permission loss.
Factory's [`HumanEditCommand` and `HumanEditCapture`](../factory/human_edit.go)
now retain exact requested field presence and original writing. The existing
Store command journal captures opaque credential-free provider preparation before
dispatch, resolves concurrent identical requests to the winning preparation and
preserves it across retained upgrades/restarts. [`HumanEdits`](human_edits.go)
now implements the provider-neutral edit writer port: preparation resolves
canonical identities and current native field permissions, and submit/lookup/
cancel use the saved installation and exact native intent. Native receipt IDs
and bytes stay intact while canonical command bindings use local identities
and their separate application digest. The SDK owns exact edit receipt checks.
An optional editor original-content condition is compared to the same native
snapshot's subject ID/title/body digest. The captured observed base is verified
against the opaque preparation on every decode. A mismatch can only reconcile
or cancel; the direct submit port refuses it. Matching conditions retain native
revision/title/body CAS rather than turning an earlier UI read into authority.
The shared [`Application.HumanEdits`](../factory/service/human_edits.go) now
delivers through this port, reconciling the exact saved capture under current
admission before dispatch. Validated observations advance under the existing
command row lock; terminal receipts/tombstones stay immutable, lost replies stay
unknown until lookup, and committed native completion needing intervention stays
unfinished. Scoped lookup/cancel recover original server-retained writing.
The shared [HTTP edit consumer](../factory/httpapi/README.md) now exposes scoped
submission/lookup/cancel through Fountain's frontend bridge and admitted Product
composition, with safe command results and exact optional-field semantics.
Only edit submission receives the larger escaped-writing allowance; cancellation
retains the ordinary envelope. Installed editor and
desktop gateway journeys remain later work. This adds no automation authority.

The paired SDK now supplies `Authority.RepositoryPullCreateSnapshot` for human
PR-creation preparation. Explicit base/head repositories and branch names bind
their current exact tips and idle native revision. Native base Code/Pulls and
head Code reads, target archive, owner block and existing open-PR facts are
rechecked across two reads. Readable private forks and explicit readable source
repositories require no writer role or owned-fork substitute. Tags, revision and
reflog expressions, malformed refs and mixed selectors refuse. Native access,
session, ref and revision changes withhold stale preparation. The read allocates
no issue/index/operation or Git ref/remote and grants no write authority.
The paired native `SubmitHumanPullCreatePrimary` now claims that revision and
binds the immutable human request, explicit source repository and both exact
tips. It checks live admission and native comparison outside SQL, then commits
the issue/index/PR and exact receipt together under the held generation. Current
policy is rechecked inside the transaction; rollback leaves no partial primary.
Replay returns retained truth without creating another PR. The primary grants
no Git write admission. The separate native `CompleteHumanPullCreateRef` now
consumes that committed receipt, rechecks native rows and both tips, and claims
the exact new internal PR ref under the retained generation/proof. Object-only
fetch copies the captured private-source commit without refs, tags, a configured
remote or FETCH_HEAD; create-only CAS and generated native hooks admit only the
receipt's zero-to-head ref tuple. Replay of a completed ref performs no write;
an uncertain started writer stays fenced for separate recovery. Native
`FinishHumanPullCreate` now claims ordinary push-history/code-owner work and
review/mention/new-PR notification dispatch once from the exact committed SQL
and ref evidence. It reuses the ordinary service's finishing helper without
rewriting refs. Partial or uncertain finishing retains its claim and ownership;
it never resends from this method. `CompleteHumanPullCreate` atomically records
successful completion and revokes the exact owner's proof before retiring its
private file. Retained lookup/replay preserves the original receipt. These use
the existing native notification dispatch contract, rather than promising
external message delivery. Stopped/inhibited `Recover` now reconciles this human
kind from its exact immutable capture, native SQL, both tips and derived-ref phase.
An uncommitted primary becomes known not committed; matching committed creation
keeps its receipt and incomplete status unless recorded finishing and native
push history prove completion. Unknown or contradictory evidence keeps ownership
held. Recovery atomically records known outcomes and revokes ownership before
retiring stale proof files; it never recreates a PR/ref or resends notifications.
`SubmitHumanPullCreate` now composes capture, primary, exact derived ref,
one-shot finishing and release through these same owners. Its primary-only
producer shares that implementation. Once the primary commits, bounded
completion survives request cancellation or session withdrawal; returning the
result still requires current admission. Existing operations return retained
truth without resuming work, including incomplete and lost-reply outcomes.
Actual owned-worker SIGKILL/reap checks now qualify all six saved checkpoints,
including rollback of an open primary transaction and retained finishing after
database reopen. Separate active-hook checks now stop, SIGKILL and reap the owned
worker/Git/generated-hook tree before derived-ref admission, after admission and
after the actual committed decision. Recovery preserves the exact SQL primary
and derived-ref evidence without ref or notification replay; unfinished completion
requires intervention. Altered receipt/ref evidence keeps ownership held across
reopen. Fresh writes after these stopped recoveries are now qualified: a new
commit/branch receives fresh SDK preparation and completes native creation,
derived-ref establishment, history/review/notification work and release in a new
worker/generation. Historical operation/PR/ref/lock state remains unchanged;
old/new replay preserves counts and read notifications. Contradictory recovery
keeps a new command busy without a new operation or effects. These checks qualify
the native stopped-domain contract; installed lifecycle remains separate work.
Recovery also preserves read notifications and ref/config state at the saved checkpoints.
The paired SDK now exposes
`Authority.SubmitHumanPullCreate`, scoped lookup and cancellation through the
private callback under `native.repository.pulls.create`. Current actor, admitted
installation and base Code/Pulls reads bind access; submit/cancel require a POST
contribution and the captured installation. Strict selector/result unions and
canonical receipt checks bind identities, tips, content digest and intent. The
submit callback accepts bounded full writing; read/cancel retain the small limit.
Actual manager/socket tests cover reply loss, changed results, installation
replacement and withdrawn access without duplicate creation or notification.
Canonical capture and monotonic outcomes now retain exact creation intent and
winning preparation in the existing PostgreSQL journal. [`HumanPullCreates`](human_pull_creates.go)
implements the shared writer port: prepare resolves existing base/source/actor
mappings, checks the current human and reads exact native preparation. Submit,
lookup and cancel decode that original capture without refreshing tips, revision
or deadline. Native operation/digest/installation and receipt checks bind source,
target, branches, writing and maintainer delegation before projecting canonical
facts; provider IDs and receipt bytes remain separate from canonical bindings.
Actual private SDK/native SQL/Git and PostgreSQL checks cover full/empty writing,
explicit same-repository source, delegation permission, refusal, cancellation,
unfinished completion, reply loss and session/access/installation changes without
duplicate creation. Source access loss permits saved receipt lookup while base
Code/Pulls reads remain required by the native callback. Application now composes
this writer with the single Store, saving preparation before any dispatch and
reconciling the original capture under current admission. Scoped lookup/cancel
accept only command ID/host/base repository/actor and recover the original source,
branches, writing and delegation server-side. Known effects never resubmit;
uncertainty never refreshes preparation. Shared HTTP/Product and Fountain's
frontend/private composition now expose strict canonical creation and scoped
lookup/cancel through this Application path, with native create capability and
bounded full writing. Responses keep preparation/receipts private. Installed
controls, capability presentation and desktop gateway remain subsequent work;
the native capability alone does not advertise product `pull.create` availability.

The paired SDK also provides `Authority.RepositoryIssueCreateSnapshot` for
human issue creation. Its private callback returns exact native
actor/repository, current internal-issue creation capability and an idle native
revision, with reader/archive/owner-block checks bracketed by fresh reads. It
allocates no issue or index and grants no creation or execution authority. The
paired native creation producer now checks live human/repository admission and
atomically allocates the issue/index and receipt through the existing journal.
Actual SQL rollback and SIGKILL/reopen checks pass, without duplicate creation
or notification retries. The paired SDK's submit/get/cancel dispatch now uses
current native human admission, actor/repository-scoped operation IDs and the
captured installation condition for writes. Real manager/private-socket checks
cover lost replies, session/permission changes and installation replacement.
Canonical issue-creation capture now retains the exact request and provider
preparation in the existing command journal, with real native PostgreSQL upgrade,
concurrency and restart/crash checks. [`HumanIssueCreates`](human_issue_creates.go)
now implements the shared creation writer port: preparation resolves existing
canonical mappings and the current native human, then reads the issue-specific
snapshot without requiring code-unit access. Dispatch/lookup/cancel decode the
saved preparation, enforce exact writing, identity, operation and installation
bindings, and check the native intent digest before projecting canonical facts.
Native receipt bytes/IDs and provider digest remain separate from the application
command binding. Actual private callback/native SQL and PostgreSQL checks cover
full/empty writing, tombstones, refused changes and lost-reply/reopen without
duplicate creation. The shared `Application.HumanIssueCreates` now composes this
adapter with capture-first delivery and validated outcome persistence. Actual
native/PostgreSQL checks cover concurrent single creation, cancellation, committed
completion intervention and lost-reply/reopen recovery. Shared creation HTTP now
consumes that application through Fountain's admitted Product composition, with
strict writing, scoped command recovery and safe result projection. Actual HTTP
checks cover native/outer reply loss, retained reopen, full escaped bodies and
post-commit session refusal. The writer also resolves a created issue from the
original authenticated receipt, rechecks normal issue visibility and compares
its exact external ID/index before returning the canonical reference. This
projection does not search by title/body, grant access after permission loss or
change the committed outcome when the destination is unreadable.
The frontend bridge now explicitly admits issue
collection submit and scoped creation lookup/cancel, using the shared issue/PR
creation route matcher. Its submit-only body allowance preserves full writing
and the ordinary cancellation bound. Actual frontend/private Unix/native SDK/
PostgreSQL qualification preserves existing reads and PR creation behavior.
Installed UI and desktop gateway journeys remain pending; local HTTP checks do
not qualify those paths.

The paired SDK provides `Authority.RepositoryLockSnapshot` for
human lock/unlock. It returns exact issue/PR identities, current lock
state, separate action facts and an idle native revision, with fresh native
subject-read/writer/archive checks. Actual callback/SDK state, permission,
identity, revision and session races refuse without issue/comment/operation
effects. This read grants no mutation authority. The paired native delivery
producer now checks live human/writer/parent/state/reason conditions and commits
native lock/history/receipt atomically. Actual SQL rollback and lock/unlock
SIGKILL/reopen checks pass; recovery never reapplies a mutation. The private SDK
now supports scoped submit/get/cancel with captured installation equality.
[`HumanLocks`](human_locks.go) implements the neutral human lock writer port over
canonical mappings and saved command captures. Preparation uses current human
and lock facts, including in repositories with Issues/Pulls and no Code unit.
Submit/lookup/cancel bind the exact saved native intent and preserve raw native
receipt bytes; their provider digest stays separate from Factory's canonical
command digest. Actual native/PostgreSQL checks cover full/blank/Unicode reasons,
issue/PR/closed-issue lock/unlock, replay after later state changes, refusal,
tombstones and lost-reply/reopen recovery. `Application.HumanLocks` now delivers
through the adapter and existing capture journal, reconciling under current
admission before every dispatch/replay and retaining validated observations.
Actual native/PostgreSQL checks cover concurrent single effects, scoped command
recovery, native refusal, cancellation, immutable terminal outcomes and cached
receipt refusal when the native session changes. Shared lock HTTP now consumes
this Application through Fountain’s admitted Product and actual frontend/Unix
bridge. Strict state/reason input, safe scoped results, body limits, lost replies
and post-commit session refusal are qualified; installed UI and desktop gateway
journeys remain subsequent work.

The paired SDK also provides `Authority.RepositoryStateSnapshot` for direct
issue/PR close/reopen preparation. Exact parent/wrapper IDs, closed/merged state
and separate action facts are bracketed by an idle native revision and repeated
fresh human/permission checks. It follows native direct state author-or-writer,
owner/poster block, dependency and archive rules, without a Code-unit prerequisite.
An imported poster gains no author right; merged PRs deny both changes. Open
dependencies can block close while allowing reopen. Actual private-callback
checks cover changing state, merged flag, parent identity, block/dependency/config,
unit, permission, revision and current human, with no mutation/operation effects.
This read grants no authority. The paired native state producer now uses live
human admission, atomically commits status/history/receipt, and performs native
stopwatch/notification completion afterward. Actual rollback, partial stopwatch
failure and process-death recovery retain exact primary evidence without
repeating uncertain completion. Typed SDK submit/get/cancel now dispatch through
live private callback admission with actor/repository-scoped operation IDs and
captured installation equality. Actual manager/socket checks cover receipt replay,
partial completion, refusal, lost replies and installation replacement. Canonical
`HumanStateCapture` now retains exact close/reopen intent and opaque preparation
in Factory's existing immutable command journal, with winning-capture replay and
real PostgreSQL restart/recovery checks. [`HumanStates`](human_states.go) now
implements the neutral state writer port over existing same-host mappings and
the current native human. Preparation retains operation absence, installation,
exact native snapshot and a five-minute deadline. Submit/lookup/cancel strictly
bind saved scope, desired state and native intent; changed installations refuse
instead of becoming absence. Native receipt bytes, IDs and digest remain intact
and distinct from Factory's canonical binding. Explicit stopwatch/notification
completion needing intervention stays unresolved. `Application.HumanStates` now
composes the adapter with canonical capture and monotonic outcome persistence,
with live admission on delivery and receipt recovery. Actual native/PostgreSQL
checks cover concurrent delivery, scoped command recovery, reply loss, changed
human refusal and retained completion intervention without repeating completion.
Shared HTTP now delivers through the admitted Fountain Product/Application with
strict boolean input, safe scoped results and ordinary request bounds. Fountain
declares `native.repository.discussions.state` and rechecks current session after
delivery. Installed UI and desktop HTTPS remain subsequent qualification.
Combined comment posting or PR ref synchronization is not implied by this
direct-state primitive.

The paired SDK now also provides `Authority.RepositoryReviewSnapshot` for human
PR review preparation. It captures exact PR/issue identities, current base/head
commits, a Git-derived merge base, idle native revision and only the current
human's pending draft ID/commit. Separate comment/draft/approve/request-changes
facts follow native code/PR read, self-review/imported-author, block and archive
rules. Private fork identities/branch names and other users' draft contents stay
private. Actual SDK/private callback checks qualify live Git/draft/identity/
permission/revision/session races and no read effects. This preparation grants
no write authority. The paired native model now supplies
`SubmitPreparedHumanReview`, an ownership-required SQL primary for draft
creation/update and final comment/approve/request-changes, including own draft
consumption and fork PRs. Exact native target, current Git proof and live policy
checks join the caller's transaction; draft commit/inline context and native
approval bookkeeping are preserved. Actual SQL/outer rollback checks pass.
The paired native `SubmitHumanReview` service now supplies live request admission,
revision/deadline claim, atomic journal receipt, native completion and conservative
SQL-evidence recovery. Exact receipts retain draft commit/stale state and composite
inline evidence; final completion failure remains committed needing intervention.
Returned-writer recovery never repeats primary or uncertain completion. Actual
native delivery/replay/refusal/rollback/recovery checks pass. Six actual owned-child
SIGKILL/reap cases now qualify persistent native database/Git reopen before and
after draft creation/update and final submission, with exact evidence fencing,
original inline context and no repeated primary or uncertain completion. The paired
SDK now supplies scoped human review submit/get/cancel through live private
callback admission and captured installation equality, with exact intent/receipt
validation. Actual manager/private-channel checks qualify native draft/final
reviews, escaped 64 KiB prose, lost replies, Git/permission/session refusal,
completion intervention without replay and genuine installation replacement.
The production Factory source manifest now declares
`native.repository.reviews.write` for its shared admitted HTTP consumer. Canonical review capture now exists in the
neutral Factory journal, including large prose and the schema-75 upgrade.
`forgejo.HumanReviews` now implements the neutral review writer port with current
native admission, strict saved event/prose/PR/intent binding, exact provider
receipts and separate canonical/native digests. Actual PostgreSQL/private-channel
checks qualify draft/final delivery, original inline evidence, lost replies after
Store reopen, expiry/scope/permission/session refusal, incomplete completion
without replay and genuine installation replacement. `Application.HumanReviews`
now saves exact captures before dispatch and retains validated outcomes
monotonically, including scoped cleanup, concurrency, lost-reply recovery and
intervention across Store reopen. Shared review HTTP now joins the same Product,
Application and current admission through the frontend and private host wrappers.
Actual HTTP checks qualify full prose, strict input/scope, both reply losses,
intervention without completion replay and withheld post-commit session results.
New inline creation, installed review UI and desktop HTTPS remain subsequent
consumers.

The paired SDK now also provides `Authority.RepositoryInlineSnapshot` for exact
inline preparation. It pins immutable merge-base/head, literal path, side and
file/range with current PR/draft/policy facts, selected entry and diff digest.
Two complete proofs and idle native revision checks refuse live changes without
creating comments/reviews/operations or granting write authority. Actual native
Git/SDK checks qualify rename/context/EOF/gap/binary/gitlink/large-file semantics
and current permission/session/draft/source races. The paired model now supplies
`CreatePreparedHumanInlineComment`, requiring native ownership and trusted current
Git/anchor proof, preserving exact file/range data and own older draft context,
and composing draft append or summary-preserving immediate review finalization
inside the caller transaction. Real SQL/outer rollback and shared-review
regression checks pass. Thread grouping and conditional snapshots now preserve
distinct immutable contexts sharing a legacy coordinate and retain unknown
legacy context explicitly. Native journaling, process recovery and admitted SDK
submit/get/cancel are qualified through the private callback. Canonical inline
capture retains the full anchor and exact opaque preparation in Factory's
existing journal, with the schema-76 bounded payload allowance.

[`HumanInlines`](human_inlines.go) implements the neutral inline writer port over
the current native human and existing same-host identity mappings. Preparation
retains operation absence, installation, exact diff/PR/draft facts and a
five-minute deadline. Submit/lookup/cancel strictly bind canonical scope, complete
anchor, mode, prose and the original scoped operation identity. They never refresh
captured proof or deadline. Native receipt bytes/digest stay distinct from the
canonical command binding; changed installations refuse rather than become
absence. Real PostgreSQL/native manager/private callback checks cover draft and
immediate effects, summary preservation, lost replies with Store reopen, exact
replay, current Git/permission/session refusal and completion intervention without
repeating native work. `Application.HumanInlines` now composes this adapter with
capture before dispatch, admitted reconciliation and monotonic PostgreSQL outcome
persistence. Actual checks qualify concurrent delivery, scoped cleanup, exact
capture/anchor binding, stale/conflicting observations, Store reopen and retained
completion intervention. The shared inline HTTP consumer now joins the same
Product/Application and current admission through Fountain's frontend/private
wrappers. Actual HTTP checks qualify full required anchors, full prose, ordinary
thread read-back, both reply losses and withheld post-commit session results.
Shared installed inline UI and the desktop gateway remain subsequent slices.

The paired SDK now supplies `Authority.RepositoryThreadSnapshot` for exact native
thread resolve/unresolve preparation. It observes the complete bounded native
conversation and retains separate prose/context and exact metadata digests,
including resolver identity. Original unknown context remains null and foreign
drafts remain hidden. Native author/writer/official-reviewer policy and archive
rules are separate from review-event permission. Two observations, idle native
revision and a final current-human check refuse live changes. Actual SQL/Git/
private SDK checks qualify file/range/tree distinctions, legacy context, policy,
bounds and session/content/Git races without read effects. This read grants no
authority. The paired native `ApplyPreparedHumanThread` now supplies the
ownership-required resolve/unresolve primary with trusted actor/revision binding,
fresh policy/Git/conversation proof and exact root-only read-back in the caller's
transaction. No-op resolution preserves an existing resolver; original anchors,
prose/patch/imported author and review/draft facts remain intact. Real SQL/Git
checks qualify stale/refused operations and joined outer/SQL/corruption rollback.
The paired native `SubmitHumanThread` now commits root resolution and exact
before/after receipts together in the existing human operation journal, with
replay, cancellation, expiry and current admission checks. Offline recovery proves
exact intent/receipt/scope and complete SQL post-state without reapplying the
mutation. Twelve actual owned-worker SIGKILL/reap and persistent-reopen cases now
qualify before/after SQL for resolve/unresolve, both no-ops, unknown context and
own older drafts. Exact receipts/native rows survive, altered evidence remains
fenced and recovery/replay never repeats the mutation. The paired SDK now exposes
`Authority.SubmitHumanThread`, `GetHumanThread` and `CancelHumanThread` through the
admitted private callback, using actor/repository operation IDs, captured
installation equality and exact intent/receipt/context validation. Actual
manager/Unix/SQL/Git checks cover full/unknown/own-draft transitions and no-ops,
lost replies, live admission refusal and genuine package replacement without
adopting the old journal. [`HumanThreads`](human_threads.go) now implements the
neutral thread writer port over canonical captures. It resolves existing native
identity mappings and the current human, refuses already-observed preparation,
and binds exact thread/original context and desired state to saved provider JSON.
Lookup/cancel keep current admission and captured installation checks after
repository permission loss. Canonical command bindings and native receipt
digests remain separate. Actual PostgreSQL/manager/Unix/SQL/Git checks qualify
full/unknown/own-draft effects, no-ops, replay, lost replies across Store reopen,
live admission refusal and genuine installation replacement.
`Application.HumanThreads` now joins this adapter to exact capture/observation
persistence, retaining scoped cleanup and uncertainty without resubmitting known
records. Real Application/PostgreSQL/native checks cover concurrent requests,
complete escaped receipts, stale/contradictory observation refusal, Store reopen,
and an interrupted committed primary that finishes only after native recovery.
The shared thread HTTP consumer now joins the same Product/Application through
Fountain's frontend/private wrappers, with explicit full/unknown original
selection and desired state, safe results, native read-back, both reply losses,
pending completion and post-commit session refusal. Stored selection and proven
Git placement stay distinct. `HumanThreads.ReadHumanThreadSelection` implements
the separate neutral reader port: it projects only canonical scope, exact stored
original anchor or explicit unknown, resolution state and resolve/unresolve facts
from the admitted snapshot, then checks the live human again. Native actor/repo/
review/resolver identities, revision, evidence digests and preparation stay private.
Unavailable original paths/trees retain selection without inventing render
placement. The read creates no operation and grants no write authority. Shared
installed actions and desktop HTTPS remain subsequent work.

The paired SDK's `Authority.RepositoryReviewRequestSnapshot` now prepares one
explicit user or team selection for adding/removing review requests. The native
read reuses current PR/Git/read admission and Forgejo's distinct user/team policy,
binds the complete selected request rows in a bounded exact digest, and repeats
the observation plus a final current-human/revision check. The native evidence
owner excludes pending/completed review prose; the response carries no private
fork identity, branch or draft text. Native closed-request/no-op behavior and
archive refusal remain distinct from review-event permission. Preparation
allocates no operation or review and grants no write authority. The native
`ApplyPreparedHumanReviewRequest` now supplies the ownership-required desired-state
SQL primary. Preparation and mutation share one native policy helper; the primary
repeats current human/Git/policy/request proof and joins the caller's transaction.
It preserves already-satisfied requests, removes the complete selected legacy
duplicate set, and proves native history plus historical official-flag changes
without exposing completed/pending prose. Historical proof is bounded to 10,000
selected rows and 2 MiB retained text; incomplete evidence refuses. Real SQL/Git
checks qualify user/team effects, no-ops, draft/history preservation, stale refusal
and outer/SQL/corruption rollback. The paired native `SubmitHumanReviewRequest`
now journals the captured desired state and commits exact request/review/history
receipts with the primary. Normal post-commit completion uses the same native
notification owner as ordinary request changes. Recovery proves complete SQL
post-state under the stopped writer's fence, never reapplies requests, and retains
uncertain notification completion as needing intervention. Actual SQLite/Git and
UI-notification queue/database checks qualify effects, replay, cancellation,
expiry, atomic rollback and stopped-writer/altered-evidence recovery. Twenty-four
actual owned-child SIGKILL/reap cases now qualify both sides of SQL commit for
user/team add/remove/no-op/duplicate selections and self-requests. Two persistent
database reopens retain exact receipts and full request/review/history evidence;
altered proof stays fenced and recovery/replay leave SQL and native UI
notifications unchanged. The paired SDK now exposes `SubmitHumanReviewRequest`,
`GetHumanReviewRequest` and `CancelHumanReviewRequest` over the admitted private
callback with actor/repository-scoped operation IDs, current human-review
capability and captured installation equality. Actual manager/Unix/SQL/Git checks
qualify user/team effects and no-ops, exact replay, lost replies, scoped cleanup
after permission loss, pending/intervention completion, live-session withholding
and genuine package replacement without adopting its old journal. The canonical store supplies
separate host-scoped `TeamID`/`TeamRef` observations and lookup-only resolution as
a prerequisite for team targets. Factory now captures canonical user/team reviewer
desired state and exact opaque preparation in its existing command journal.
`HumanReviewRequests` implements the neutral reviewer writer port using the live
human admission and existing mappings. `FountainIdentities.NativeTeam` is
lookup-only; `LocalTeam` observes authenticated incoming native subjects in their
separate canonical namespace. Neither grants membership or permission.

Reviewer preparation observes exact selected request evidence and separate
add/remove policy. Submit/lookup/cancel decode the retained preparation strictly,
bind its native user/team to the canonical request and check the current human.
Operation IDs remain actor/repository scoped; installation is a captured equality
condition. Native effect/receipt/digest and completion facts project into canonical
bindings while retaining exact native receipt bytes and the separate provider
digest. Cleanup resolves identity and current principal without requiring current
repository access; replacement installation refuses old captures even on absence.
Actual manager/private Unix/native SQL/Git and PostgreSQL-capture integration
qualifies user/team changes and no-ops, replay/cancel, lost replies, stale/busy
refusal, session changes, permission-loss cleanup, pending/intervention completion
and genuine package replacement. Canonical `ObserveHumanReviewRequest` now persists
this admitted evidence with immutable capture/receipt binding and monotonic
effect/completion. Real PostgreSQL checks qualify overlapping stale replies,
complete 1,000-record native no-op/removal receipts, restart and WAL recovery.
`Application.HumanReviewRequests` now consumes the writer and canonical journal,
persisting before dispatch and reconciling exact captures under live admission.
Known effects are never resubmitted, including pending/intervention completion;
scoped cleanup recovers the original target/state from the server. Real native
Application checks qualify lost replies, Store reopen and admission/installation
refusal. Shared HTTP/Product and Fountain frontend/private API composition now
expose reviewer commands with canonical user/team IDs, strict JSON, scoped cleanup
and safe result fields. Final current-session checks withhold post-commit results;
provider preparation/receipt bytes remain server-side. Installed UI remains pending.

Human merge preparation now has a separate admitted read boundary in the paired
SDK: `Authority.RepositoryMergeSnapshot`, with an explicit ordinary method
(`merge`, `rebase`, `rebase-merge`, `squash`, or `fast-forward-only`). It returns
exact native repository/operator/PR/issue identity, target head/base tips, derived
merge base, selected method, closed/merged facts, native eligibility and the
bracketed idle revision. Target tip and merge base remain distinct. Native policy,
permission, draft/checking/protection/archive and method settings are reused;
non-fast-forward ancestry denies the fast-forward method. Manual marking, delayed
automerge and branch update are excluded from this read contract.

Two complete observations compare private source/author identity facts without
exposing private fork IDs/branch names or draft/review prose. Missing refs/units,
held ownership, changed metadata/Git/revision and current-human loss refuse.
Actual private Unix/SDK/SQL/Git checks qualify all five method policy facts and
the refusals without creating operations, reviews or comments. These are native
eligibility observations, not proof that a journaled human merge writer exists.
Conditional primary/journal/transport and canonical Application/HTTP/UI consumers
remain subsequent work. The existing Factory delivery merger keeps its own
separate authority and supported method contract.

The paired native `PrepareHumanMerge` now stages exact local commits under the
caller's held native revision/generation and live human admission. It rechecks
native PR/repository/method/permission/protection and actual head/base/source
before and after computation. `pull.PrepareExactMerge` shares the existing native
method engine with ordinary merges, pinning the copied base before computation.
All five immediate methods and private source forks use that same engine.
Its `PreparedExactMerge` owns a private temporary repository and PR working-pool
hold; callers must `Close` it. Result commit/tree/parent facts describe local
objects, not a provider receipt. Close and refusal retire the local repository.
No LFS/ref push, PR completion, operation allocation or notification occurs here.
Real native SQL/Git checks qualify topology, ownership/admission/staleness and
cleanup. The native gate below consumes this exact stage; its journal and push
execution remain required before a merge action can be exposed.

The paired SDK also defines `HumanMergeRequest` and its canonical immutable
intent. Its operation ID binds the durable command to the native actor/repository;
the intent captures exact preparation, message bytes and deadline. It has no
client-selected result commit. `PrepareHumanMergeOperation` consumes that capture
only when the native journal already contains the identical pending human
operation, with no automation credential, and its exact conditional scope and
owner/generation are held. It rechecks those facts and current human admission
around local computation. Real SQL/Git checks cover saved-command reconstruction,
all five methods, changed replay, cancellation, expiry and cleanup. Background
submission rejects this human kind. Public submit, push execution,
receipts/recovery and canonical consumers remain pending; no human merge endpoint
or successful merge is implied by command capture.

`BindHumanMergeStage` now binds the actual computed commit once to the held
native scope, with the same immutable command, owner and generation. The native
service retains a live human admission callback only for that stage's lifetime;
callers must retire it after their push finishes and close the stage. At Git's
prepared checkpoint, exact single-ref effects, current session/request,
deadline, native PR/method/permission/protection/source and private stage inputs
are checked again. The transaction locks the exact scope/generation and uses
the existing atomic cancellation/duplicate ordering. A closed stage, retired
callback or fresh service without its callback refuses. No session is stored in
the journal. Actual native SQLite/Git checks qualify all methods and private
fork admission without publishing; disposable native PostgreSQL qualifies
competing scope binders, eight admissions and both cancellation orderings.
The native primary/journal must still compose effect attribution, completion and
interrupted-writer recovery before an end-user merge action is enabled.

`PushHumanMergeOperation` consumes an already saved command and open stage,
binds it, and uses the existing native LFS/push owner with full pushing identity,
the PR merge trigger and the host-private execution capability. It checks the
actual Git-selected hook path and generated delegates before launching, then
pushes the immutable staged commit through the native hooks. Local stages are
attempted once; Close waits for the push without blocking its prepared callback.
The producer returns raw admission/tip observations and retains the pending
operation, owner and capability on success and error. It creates no merge receipt
and does not enable completion. Actual generated hooks from the current built
executable, authenticated private routes and Git ref writes qualify all five
methods, a private fork, live refusals and lost replies on either side of commit.
The LFS payload and interrupted-process qualification is described below. A result
which does not advance the ref refuses here. Manual metadata marking has its own
preparation boundary below and still needs its separate journaled SQL consumer.
No human merge mutation endpoint is enabled.

`RecordHumanMergeEffect` now records primary ref truth from the exact saved
command, held scope/generation, execution capability, persisted admission and
authoritative Git ref/commit/tree/parents. Its transaction locks both the scope
and operation before observation, ordering cancellation and competing decisions.
Committed effects carry a typed immutable `HumanMergeReceipt` and pending
completion; an unadmitted unchanged target is not committed, while missing,
unattributed or conflicting evidence stays indeterminate. Every outcome retains
the owner and capability. Finalized operations refuse late prepared admission;
attribution never proves an interrupted process tree stopped or releases its
fence. Replays preserve terminal receipts, and later authoritative evidence can
resolve an indeterminate observation. Actual generated-hook pushes qualify all
methods and a private fork, lost replies, concurrent observations and retained
native lookup. PostgreSQL qualifies one durable decision and fresh cancellation
observation under real row locks. This producer performs no PR completion,
notification, submit orchestration or public callback/HTTP operation.

`ApplyHumanMergeMetadata` now consumes that exact committed receipt under the
same held capability, scope and operation locks. It rechecks the current target
and immutable Git object facts, then validates the PR/issue/base identity before
calling the shared native `pull.ApplyMergeMetadata` owner. The ordinary
post-receive merge handler uses that owner too. Native `SetMerged` closes the
issue, creates its merge history and updates counters/cache; scheduled-merge
consumption joins the same transaction. Exact replay produces no second history
event, and a conflicting prior merge refuses. Real generated-hook/SQL/Git checks
cover all five methods, a private fork, concurrent replay and a late SQL failure
that rolls closure/history/schedule changes back together. PostgreSQL qualifies
the model transaction and receipt/scope guards with an opaque domain fixture.
The primary receipt and pending completion state remain unchanged, with owner
and capability held. This metadata step leaves notifications, cross references,
remaining push bookkeeping, safe release and interrupted-writer recovery to
their consumers; it enables no end-user merge operation by itself.

`SyncHumanMergeBranch` now mirrors the committed target into the native branch
catalog after exact PR metadata matches the receipt. It shares the held
capability, immutable object/identity checks and SQL locks with metadata
completion. The existing branch-sync implementation and branch mutex now live
in `modules/repository/branch_sync.go`; the ordinary service still claims its
reservation and delegates to that same owner. Held ownership is required at the
shared entry point. Native fresh-catalog bootstrap, stale/deleted-row update and
up-to-date replay behavior are preserved. Actual generated-hook/Git/SQL checks
cover all methods and a private fork, fresh catalogs, concurrent replay,
rollback and the ordinary idle-to-owned caller. Receipt, pending completion and
human operation ownership remain unchanged. This performs no push-event queueing,
notification, cross-reference action, ref write or final release.

`FinishHumanMerge` consumes that exact merged PR and synchronized branch catalog.
It durably claims a finish phase before calling the shared native
`pull.FinishMerge` notification/cross-reference implementation outside SQL. One
caller dispatches; repeated or concurrent claims refuse. Success records a done
phase while overall completion remains pending. A partial native failure records
`needs_intervention` with the started phase; neither it nor a lost result triggers
automatic resend. Receipt, owner and capability stay held for push-event and
safe-release/recovery consumers. The normal native merge also calls the same
finishing implementation, which now propagates loading/cross-reference errors.
Actual Git/hook/SQL checks use the real UI notifier and parsed closing references,
including read-notification replay suppression and a failure after notification
dispatch. They do not establish external notification delivery or full merge
completion. The native completion column is now VARCHAR(32): the ordered
`v17c_widen-native-operation-completion` migration preserves retained operation
and reservation data while allowing `needs_intervention` on PostgreSQL. The
retained original-width PostgreSQL upgrade and SQLite migration path pass.

The existing push-event worker now delegates to
`services/repository.ProcessPushUpdates` under its ordinary reservation. This
shared engine requires matching held owner/generation and a consistent repository
batch; its caller supplies authorization, exact ref attribution and single
dispatch. It retains native size/time updates, deferred PR synchronization,
commit-reference issue actions, commit feed/notifier fanout, cache and tag work,
without claiming or releasing ownership itself. A local Git/SQL integration with
the real feed notifier proves issue closure/reference attribution, held-owner
refusal/retention and the ordinary worker's claim/release path. The human merge
consumer now binds this engine through `CompleteHumanMergePush` and
`DispatchHumanMergePushEvent`. The operation service validates the exact committed
receipt/current native objects, completed PR metadata, synchronized branch and
done finishing phase, then commits a push-event started phase before dispatch.
Repository/ref range, actor and names come from that native evidence. A
server-side function argument joins the engine without a service import cycle;
it supplies no authorization or caller-selected event facts.

Finishing and push events share the existing SQL claim/result implementation,
now named `ClaimHumanMergeDispatch` and `RecordHumanMergeDispatch`. Both dispatch
entry points refuse an outer SQL transaction so their claims commit before native
effects begin. Successful push dispatch records a done phase with overall
completion still pending. Partial failure records `needs_intervention` and retains
the started phase; a fresh service refuses resend and reads the retained receipt.
Owner and capability stay held. Actual generated-hook/Git/SQL checks cover all
five merge methods, private forks, concurrent claims, changed receipt/catalog
refusal, outer transactions, feed preservation on replay and a repository SQL
failure after feed creation. The real deferred PR queue is initialized/drained;
an unrelated fixture PR is closed, so PR revalidation is not qualified here.
These dispatch stages leave safe release to the completion consumer below;
external event delivery and PR revalidation remain independently queued work.

`CompleteHumanMerge` requires the original native stage to have attempted its
single push and finished `Close`. `WithQuiescentExactTarget` shares the existing
private target check with the live stage gate; open, unused or foreign stages
cannot substitute for that closed writer. The completion service additionally
requires the done push-event phase, exact current receipt/ref/objects, merged PR
metadata, synchronized branch and matching staged result/tree/parents. It joins
those guards to the existing atomic completion/release SQL implementation. A
release failure rolls completion back. Success preserves the primary receipt,
clears owner/scope/verifier and then deletes the private capability file; that
proof is already revoked by the SQL release if file cleanup fails. Retained
lookup resolves lost completion replies. Outer transactions refuse before effects.
Real native checks cover all methods/private forks, the existing concurrent
Git/Close case, temporary-resource cleanup, open/foreign-stage refusal, competing
completion, partial-state fencing and release rollback. PostgreSQL separately
qualifies joined SQL rollback and concurrent release with opaque domain evidence.
This completes the committed live-writer path; refusal cleanup is separate below.

`ReleaseHumanMergeNotCommitted` requires a durable unadmitted non-commit with no
receipt/completion state, then re-observes the native ref under the exact scope/
operation locks. The current target must still equal the original base. The
same private stage checks now support `WithClosedExactTarget` for a closed
pre-launch refusal as well as an attempted/refused writer; committed completion
retains its attempted-push requirement. Pending, admitted, committed, uncertain,
open/foreign or changed-current-evidence cases keep ownership. Successful cleanup
clears ownership for that exact generation and deletes its revoked capability file while
preserving refusal, cancellation and intent. It grants no new write authority.
Native hook/Git/SQL checks qualify pre-launch and prepared-hook refusals, session/
source withdrawal, cancellation, changed-ref fencing, rollback and competing
release. PostgreSQL qualifies the model gates and preserved refusal record.
`SubmitHumanMerge` now composes these owners through the repository service's
fixed native push-engine join. It uses the existing conditional claim/capability
lifetime, checks current human admission before capture/replay and before returning
operation truth, and preserves cancellation tombstones and intent conflicts.
One new capture prepares a local stage and attempts one native push. The stage
closes before authoritative effect observation. Committed effects consume metadata,
branch sync, finishing, push dispatch and safe completion; proved refusals use
closed-stage cleanup. Indeterminate or partial outcomes retain the fence/capability.
Raw push/transport errors never substitute for native evidence, and identical
submits return recorded state without resuming or relaunching work.

If local preparation returns no stage, settlement requires the exact unprepared
scope with no bound result or completion phase. Native scope/operation locks order
that settlement against any binder. Fresh no-effect evidence permits the existing
non-commit release; a changed/missing target remains uncertain and fenced. The
shared claim owner retires the capability only after release. Outer SQL
transactions refuse before capture. Real native submit checks cover all methods,
private forks, concurrent identity/replay, expired/stale/revoked admission, early
cancellation, preparation failure, lost hook replies and partial completion.
Public human merge transport remains a later consumer; no new background kind,
SDK callback, Factory HTTP route or installed UI is enabled by this orchestration.

The existing offline `Recover` dispatcher now recognizes human merges. Its
stopped/inhibited-domain contract remains mandatory. Recovery shares the exact
intent and scope checks with live capture, validates native admission/ref/object
and parent evidence, and preserves an existing canonical primary receipt. An
unadmitted unchanged target establishes non-commit. An admitted exact result can
establish a missing receipt and reconcile PR metadata/branch SQL through the
existing idempotent owners. It never writes another ref or repeats notification,
cross-reference or push-event dispatch. The done push-event marker permits
complete status; missing/partial dispatch becomes `needs_intervention`. Changed
evidence or contradictory metadata stays fenced. Original merger attribution
survives later permission loss. Exact release retires stale private capabilities.
Native checks include returned synchronous writers and bounded stop points,
including an inhibited deferred queue. The existing callback integration root
also runs an owned Linux child against a file-backed database and actual bare
Git repositories. That child uses generated hooks, the private router, native
notifiers and the shipping repository submit/push-event join. At the four saved
submit checkpoints, the parent checks for active children, SIGKILLs and reaps
the worker, then reopens the same database and Git repositories before setting
offline inhibition. Recovery validates exact receipts, merge history and branch
metadata without changing dispatch records; a second reopen and submit replay
must preserve the result and leave ownership idle. These checkpoints are outside
synchronous Git. Active Git/hook checkpoints use the existing Linux process
integration root as an isolated subreaper companion. It stops the owned worker
before discovering its descendants, kills only that captured tree, waits for
the worker and reaps every adopted Git/receive/hook child by exact PID. The
callback fixture reopens persistent state only after that companion exits.
Checkpoints surround actual prepared admission and committed native-hook
decisions; they do not substitute a timeout or synthetic native outcome. A
prepared admission with an unchanged target remains fenced with its saved
operation and capability intact. Native commit before the reply completes can
reconstruct a receipt and reconcile metadata, with dispatch still explicitly
pending intervention. Metadata-only merge writing remains separate work before
public transport activation. The offline marker alone never establishes writer
stop. The companion is test instrumentation,
not a product process supervisor or an automatic recovery mechanism.

Stopped human-merge recovery also retires a leftover files-backend target ref
lock before releasing a known outcome. A killed Git process can leave that lock
behind even when SQL ownership is safely released; a fresh push can then be
admitted and still fail to advance the ref. Cleanup is limited to the saved
target ref's regular `.lock` file containing exactly the captured result OID and
newline, with unchanged file identity and no substituted ref-directory symlinks.
A foreign, malformed, symlinked or unreadable lock retains the fence. An admitted
old target remains uncertain and never enters cleanup. No other lock or Git ref
is removed. The callback integration's reentry mode starts a fresh native worker
after actual process-tree death/recovery: safe non-commit can accept a new human
command, a recovered committed branch can accept a new PR, and uncertain state
still refuses a new writer. Historical operation receipts remain unchanged and
replay correctly after later ref advancement.

The shared native LFS engine now verifies source-repository association and exact
size plus shared-store content presence/size before associating each pointer with
the target repository. It preserves the caller's execution/cancellation context,
rechecks the conditional human guard after storage I/O and repeats source checks
inside the short owned metadata transaction. Storage and SQL errors stop before
Git, and consumer/cat-file failures propagate through the scan rather than becoming
silent success. Ordinary merge and rebase-update callers use this same engine.
The actual 200 KiB binary payload, canonical Git pointer and private-fork target
association are qualified for all five human methods and an ordinary merge.
Missing content/source association, incorrect sizes, real filesystem/SQL failures,
operation cancellation, session loss and request cancellation refuse before native
Git hooks. Same-repository pointers retain their valid existing association.

LFS association transactions are per object; an earlier valid association can
remain when a later association fails. This is separate from the primary ref
receipt. A real second-object SQL failure verifies that partial state, unchanged
Git and no LFS I/O or additional associations on operation replay. Successful
objects are reused from the shared store, rather than physically duplicated.
Desktop candidate-upload transport and installed LFS journeys remain later work.

Manual merge preparation is a separate SDK read operation,
`repository.manual_merge.snapshot`. It selects a full commit OID already reachable
from the current target branch, returning exact PR/issue/actor/repository identity,
target branch/tip and idle native revision. It shares `pull.ReadManualMergeCommit`
with the existing ordinary manual writer. Current native manual eligibility
requires merge permission but deliberately skips new-push draft/status/CI gates;
it does not require a readable source repository or a live PR head. Disabled
manual policy, closed/merged PRs, archive state, missing units and permission loss
remain visible refusals. Two native observations and revision/session checks reject
changed preparation. The typed SDK selects commit OIDs rather than revision aliases
or tag objects; the ordinary writer retains its existing full-tag resolution.

Actual SDK calls over the private pinned Unix socket qualify these reads without
operation/history/notification/feed allocation or ref writes. Git/ref, branch
identity, permission, native revision and session races refuse. The ordinary
manual-write regression also exposed automatic ORM timestamp substitution:
`SetMerged` now preserves the caller's explicit merge time for that update.
Historical commit-author time, manual status, one history row, issue closure and
unchanged target Git are checked through the existing writer; an ordinary push
merge still succeeds through the same metadata owner. This preparation is not a
write grant.

The paired SDK now defines `HumanManualMergeRequest` and its canonical command
and receipt, distinct from ref-push merging. The internal
`SubmitHumanManualMergePrimary` captures exact human/PR/target/commit/revision
evidence without a background token or Git-write scope. Under the exact saved
generation, scope and intent, native manual metadata, issue closure, merge history
and the operation receipt commit in one SQL transaction. It shares
`pull.ApplyManualMergeMetadata` with the ordinary writer. Current session,
permission, policy, target and cancellation checks run before and inside SQL;
withdrawn authority or changed targets roll the metadata back. Unknown SQL errors
or altered capture/generation retain the fence rather than granting a retry.

Fifteen real native SQL/Git cases cover commit, refusal, cancellation, rollback,
tampering and replay. The fixture now explicitly passes its historical Git dates
through its restricted command environment; selected, stored and receipt dates
agree with 2001 rather than wall-clock time. A committed primary retains its owner
and capability with completion pending; lookup/replay does not repeat the write.
`FinishHumanManualMerge` now consumes that committed primary and shares
`pull.FinishManualMerge` with the ordinary manual writer. Exact native metadata,
history, canonical receipt, held generation/scope and unchanged target/selected
commit are checked before a durable one-shot finishing claim. Native merge
notifications and cross-reference actions then run outside SQL. A partial failure
stays claimed with `needs_intervention`; replay cannot resend effects.
`CompleteHumanManualMerge` rechecks the evidence and successful finishing before
atomically completing and releasing ownership, then retires the private proof.
There is no merge push, branch synchronization or push event for manual marking.

Fourteen native completion cases cover real UI/feed notifications, referenced
issue closure, concurrent finishing/release, SQL release rollback, changed
metadata/history/receipt/target, generation/proof/scope refusal and late
cancellation/expiry. The admitted primary remains historical truth, and lookup
or submit replay never reapplies it.

The offline `Recover` dispatcher now recognizes manual marks and shares the live
completion path's exact native receipt/parent evidence reader. It requires a
stopped, inhibited writer domain and matching saved capture, scope and generation.
An atomic unadmitted command with unchanged open native state becomes a known
non-commit. A committed receipt and exact manual metadata/history remain historical
truth: successful recorded finishing yields complete; missing or uncertain delivery
stays `needs_intervention` without replay. Known outcomes and release commit
together before stale proofs are retired. Contradictory or missing evidence stays
fenced. Recovery refuses an outer SQL transaction that could still roll back.

Fifteen native recovery cases write the real primary into a private SQLite file,
close/reopen it, run inhibited recovery and reopen it again. They qualify native
row/history/notification/Git preservation, cancelled pending commands, partial and
uncertain dispatch, missing execution proof, altered evidence, missing history and
atomic release rollback.

`SubmitHumanManualMerge` now composes capture, SQL primary, one-shot native
finishing and release under the original execution owner. It uses the same
primary implementation as the retained producer boundary. Existing commands
return their saved outcomes without resuming. Once the primary commits,
bookkeeping continues independently of request cancellation; the response still
requires the current human session. Known partial finishing returns its committed
`needs_intervention` record. Unknown SQL/dispatch/release failures retain their
fence and receipt for stopped recovery.

Sixteen native submit cases qualify the complete path, duplicate identity and
busy ownership, early/captured/late cancellation, expiry, session withdrawal,
changed targets, actual primary/dispatch/release SQL failures and discarded service
results. Receipt/history/notification state remains unchanged on retry and there
are no Git or push-event effects. Five disclosed native checkpoints cover
before-primary, inside-primary, after-primary, after-finishing and after-release
state. Git readers close before the inside-SQL checkpoint; no Git writer is part
of this operation.

Six owned Linux worker SIGKILL/reap cases now qualify those checkpoints and an
altered committed receipt. The fixture checks every Go thread's `/proc` child
list before killing its own PID, then reopens the same native database and Git.
Death inside the open SQL transaction rolls back the uncommitted manual mark and
history. Committed receipts and exact native rows survive; missing delivery stays
`needs_intervention`, finished work releases complete, and altered evidence stays
fenced. Recovery and submit replay never repeat native metadata or dispatch.
The finishing checkpoints observe actual queued UI delivery before termination.
`Authority.SubmitHumanManualMerge`, `GetHumanManualMerge` and
`CancelHumanManualMerge` now expose this native owner through the admitted private
callback. The new declared capability `native.repository.merges.write` is separate
from human review capability; it never replaces current native merge permission.
Actor/repository-bound operation IDs scope early cancellation and retained reads.
Submit/cancel assert the saved installation while lookup returns the live namespace
even for absence, preventing an old command from being adopted after replacement.
The SDK validates exact result identity/digest and canonical typed receipt;
submission also binds it to the complete captured preparation. Strict request
shapes and typed admission/query/conflict/unavailable errors cover the transport.

Fifteen actual manager/private Unix/native SQL/Git cases qualify these SDK methods,
including a real connection dropped after successful submit, concurrent identity,
post-commit session withdrawal, partial completion, malformed envelopes/results,
capability/method/reader refusal and genuine package replacement. Native replay
does not repeat history or change Git.

The ordinary ref-push merge producer is now exposed by
`Authority.SubmitHumanMerge`, `GetHumanMerge` and `CancelHumanMerge` on the same
declared merge capability and admitted private callback. It calls the existing
repository-service join with its real push-event engine. Submit/cancel assert
the saved installation; lookup remains scoped to the current actor/repository
and live namespace. Typed results bind the immutable digest and canonical receipt,
including method-specific result topology and complete Git object identity;
submit additionally binds the exact captured preparation.

Twenty-one actual SDK/manager/private-API/generated-hook/Git cases qualify all
five methods and a private fork, single native target push, native completion,
prepared-hook session refusal, both sides of commit admission, partial finishing,
reply loss, malformed input/output, concurrent replay and installation replacement.
Retries leave target and hook counters unchanged. Canonical commands and outcome
ordering now reside in Factory/Store. `HumanMerges` implements the neutral writer
port for both ordinary ref-push methods and manual marking. It resolves existing
canonical repository/operator mappings, checks the live human and reads native
preparation only when no operation already exists under that identity. The caller
must persist its returned capture before dispatch.

Strict retained-preparation decoding binds native kind, repository/operator/PR,
method, exact message or selected commit, operation identity and native digest.
Receipt projection checks that original preparation and keeps the native digest
separate from the canonical command binding. Committed completion remains pending
or intervention until native bookkeeping completes; early cancellation retains
an unbound no-effect tombstone. Lookup/cancel allow own-receipt cleanup after
repository permission loss, requiring current principal and installation. A
replacement installation cannot adopt a capture even when its lookup is absent.
`Application.HumanMerges` now consumes this admitted writer over the single
canonical command journal. It commits winning preparation before dispatch,
reconciles original captures without refreshing them and retains pending completion
without replaying primary effects. Scoped lookup/cancel checks live admission and
recovers original method/message or commit from Store. Shared HTTP/Product now
exposes strict canonical merge/manual submit and scoped lookup/cancel through
Fountain's frontend/private wrappers. The manifest declares the separate merge
capability. Escaped 64 KiB messages pass the 512 KiB submit envelope through HTTP
and the native SDK callback; cancellation and native lookup keep their small
envelopes. Responses omit private receipts/preparation and face a
final current-session check. Installed UI and desktop HTTPS access follow.

The corresponding Fountain source changes are in `sdk/native_{issues,comments}.go`,
`sdk/native.go` and `routers/web/extensions/callback{,_issues,_comments}.go` in the sibling
`forgejo-ext` repository. That existing read callback now supports an explicit
full-body selector, stable IDs and participant/label/time metadata, and rechecks
issue-unit access after I/O using fresh repository facts. The existing composer
continues to request bounded previews. The enriched callback must be built with
its consumer; an older host refuses these product reads instead of returning a
fabricated complete view. No provider database access was added to Soda.

| Constructor | Consumer role |
| --- | --- |
| `NewServiceBackground` | Shared authenticated native service callback transport for revision reads, snapshots and persisted operations. |
| `NewServiceObserver` | Permission-checked issue/PR evidence and actor resolution for readiness and dispatch. |
| `observation.NewServiceReadinessSource` | Canonical acceptance/dispatch evidence and idle revision observations over the service observer. |
| `observation.NewCompletionSnapshotSource` | Exact completion inventory and source-qualified local capture with existing per-pass call accounting. |
| `NewIssueGraphWriter` | Neutral graph plans/facts/outcomes over the admitted native graph protocol. |
| `NewPublisher` | Candidate validation, conditional branch publication and PR creation. |
| `NewReviewer` | Conditional native review operations. |
| `NewMerger`, `NewCheckAssessor` | Conditional merge, complete delivery discussion and native check evidence. |

The web facade wires these into `factory/control`. Background integration needs
the configured Unix callback socket, expected native host peer UID and a
restricted credential file for the chosen actor. Review and merge credentials
are separately configured. Constructors defer I/O until use.

The [`observation`](observation/) subpackage owns the Factory observation adapters,
including selected acceptance evidence, lifecycle occurrence validation, dispatch
inputs and completion page/body/ref brackets. They no longer depend on HTTP API
objects or sessions. This provider subpackage imports the controller's ports;
the core Fountain client and Factory engine remain independent of each other.
Canonical actor/repository keys resolve only through the adopted Fountain host
before native credential use. Missing observers yield unavailable/refusal results,
and foreign mappings never fall back to equal numeric IDs. Snapshot revision,
visibility, digest, completeness and exact selected-source rules are unchanged.
Completion reads retain their per-pass native-call budget, prefix continuation
and current-tip qualified local capture; hidden/incomplete evidence cannot seal a
subset. Existing private-socket and PostgreSQL integrations exercise these seams.

Share one `ServiceBackground` between the observer and operation adapters using
`observer.ShareBackground(background)`. Native bootstrap revokes the previous
service admission, so separate transports can invalidate each other's reads
and operations. Snapshot observations are bound to equal idle native revisions.

Bind `ServiceObserver.Identities` to the adopted host before creation recovery.
`ReadCreations` accepts canonical repository IDs, resolves the native repository
before credential use, and maps observed original creators to Factory actors.
Every returned event carries the canonical repository; native issue IDs/indexes,
delivery identities, cursor positions, original digests and retention/provenance
coverage remain provider facts. Missing/foreign mappings refuse without a native
request. This boundary does not convert graph effects or publication consumers.

`GraphCodec` performs no I/O or admission. It translates neutral graph plans to
the existing SDK encoding, restores that encoding for approved packet digests
after JSONB storage, and projects raw receipts for Factory adoption/store guards.
Permission, actor/repository/installation scope and exact conditional mutation
checks remain with their existing adapter/host owners. An opaque receipt stays
provider evidence; decoding it does not grant execution or spending authority.

`IssueGraphWriter` requires the adopted `FountainIdentities` binding. Owner
permission reads, generated-issue reads, operation submission/lookup and feedback
resolve canonical repository/actor IDs before native access. The credential is
still checked against its native owner, and permission reads keep the stable
revision bracket. Factory composition maps the configured native graph actor
into its canonical coordinator key without changing credential custody.

`IssueGraphWriter.DecodeGraphReceipt(ctx, raw)` uses lookup-only registry mappings
for receipt repository/actor facts; the original receipt, native issue IDs,
indexes and content versions remain intact. Generated-issue creators are observed
under the adopted host only after a validated native read. Operation outcomes
retain raw receipts, unknown states and cancellation tombstones. Unknown or
malformed receipts cannot become adoptable through decoding. Feedback reconciles
the native author and source marker after a lost reply. The API completion source
also resolves native page/body/ref scope while keeping local material canonical.
`Publisher` also requires the adopted identity binding. Repository remotes,
credential-owner checks and publication/PR/correction payloads resolve native
scope before access. Returned operation metadata and adopted PR authors use
canonical keys; receipts, candidate data, native PR identities and operation IDs
retain their original meaning. Adoption takes context for identity lookups and
checks raw native receipt tuples against the canonical recorded work. Lookup and
cancellation use existing mappings and owning-installation admission after actor
credential withdrawal.

`Reviewer` and `Resolver` require that same adopted identity binding. Review
repository/reviewer/PR-author fields resolve before native reads or payload
construction, including pending-review checks and nested head repository fields.
Observed PR authors and adopted reviewers are canonical; raw receipt bytes,
review/comment IDs, exact inline key mappings and immutable candidate facts stay
intact. Context-aware adoption compares the native receipt to the canonical work.
Thread-resolution readback checks canonical operation scope before native access
and compares the native resolver, exact head and resolved conversation digest.
Publication, review and resolution share lookup-only operation-scope projection
through `FountainIdentities.LocalOperationOutcome`. Missing identities cannot be
allocated by an opaque outcome. Credential-free lookup/cancel, zero-scope absence
and cancellation tombstones retain their meaning.

`CheckAssessor` and `Merger` also require the adopted identity binding. Check
targets and assessor keys resolve to native snapshot scope without changing
exact-head status reduction. Merge PR authors, reviewers, actors and repositories
resolve before native credential access, evidence reads or packet construction.
Native approval/merger identities are compared in that namespace; returned
observation, receipt-adoption and completion actors are canonical. Raw receipts,
PR/issue/review IDs, commit/ancestry facts and original closing occurrences retain
their meaning. Context-aware receipt adoption verifies the native tuple against
canonical recorded work. Lookup/cancel retain owning-installation admission after
credential loss and use existing mappings without allocation. These adapters do
not grant CI, merge authority or installation/desktop completion by themselves.

`Merger.ReadDeliveryMaterial` resolves canonical merge scope before accessing
credentials, snapshots or authenticated review bodies. It validates the exact
native PR, refs, complete discussion and unchanged revision, then returns the
canonical repository and participant actors. Authenticated discussion may observe
a new participant on the adopted host; an opaque operation outcome still cannot.
Missing optional attribution stays unknown. Native review/comment/thread IDs,
raw bodies, content versions, blocking review IDs and conversation digests stay
intact, including the resolver-independent digest encoding. Both native and
projected material retain the existing size bound.

## Extension manager control

The dashboard wires manager control independently of factory actor credentials.
It uses `FORGEJO_EXTENSION_SERVICE_CALLBACK_SOCKET` and the explicitly pinned
`FORGEJO_EXTENSION_SERVICE_CALLBACK_PEER_UID` from its service configuration.
Missing or invalid callback settings leave manager control unavailable. Root
reaches this transport through the private operator listener; Fountain verifies
the mapped Soda service peer. Manager requests neither bootstrap background
admission nor carry an actor PAT. Factory background configuration does not
select or replace this manager callback.

## Effects and outcomes

Reads do not authorize new work. Publication, review and merge adapters can
submit native mutations under persisted operation IDs and exact observations.
Lost replies are reconciled by lookup; unknown outcomes remain waiting or fenced
instead of being treated as success. Actor credentials stay in restricted files,
and candidate Git processing belongs to [publish](publish/README.md).

See [dashboard wiring](../web/server.go),
[Forgejo reference](../../docs/reference/forgejo.md),
[factory interfaces](../../docs/architecture/factory-interfaces.md), and
[trust boundaries](../../docs/architecture/trust.md).
