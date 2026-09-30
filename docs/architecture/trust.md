# Trust model

This document owns authority, identity and privilege boundaries for Soda OS.

## Distinct authorities

| Authority | Grants |
| --- | --- |
| Host root | Host OS, Podman, systemd, destructive appliance operations |
| Configured Soda operator | Appliance runner/Tailnet settings; recorded Forgejo user ID from setup |
| Forgejo site administrator | Forgejo administration |
| Organization owner/admin | Organization Forgejo authority only |
| Repository owner | May create that repository's project; not host access |
| Project member | Project-local Linux account after confirmed Join |

These authorities are not interchangeable. Owning a repository does not grant
appliance access. Arbitrary site or organization administrators are not project
root. Setup tokens are not ordinary acting-user credentials.

## Identity split

- **Forgejo** owns passwords, factors, native sessions, permissions, Git credentials
  and collaboration.
- **Soda** owns preferences, development-access public keys, environment membership
  and product grants, plus factory execution identity and run-scoped authority.
- **The installed Soda extension** carries only the live native actor and bounded
  request authority needed by the current browser operation. The Soda service
  checks its product policy and does not receive Forgejo cookies or session IDs.
- Native Git keys and collaboration remain upstream-owned.
- Soda does not maintain a second password, provider-role inventory or CI scheduler.

Stable Forgejo identity binds project membership to its original Linux login.
Native rename or transfer does not silently remap Linux users, ownership or already
installed keys. Web administration and native wheel/SSH rights are not automatically
synchronized on transfer; see [Project OS](../reference/project-os.md).

## Runtime identities

Soda service UID/GID values and native runner accounts are not developer host
onboarding. The web process's privileges and human authorization are separate
boundaries.

## Factory authority boundary

The factory runs only for a trusted team on one operator-managed appliance. Agent
instructions and repository content are untrusted inputs even there. Automation
is disabled until three standing authorizations are recorded:

| Authority | May authorize |
| --- | --- |
| Current native repository owner/administrator | Enable the repository factory; select target branch, supported harness/model per role, a nonempty set of required native CI checks, merge method and repository limits; authorize scoped native publication, review and merge actors within the other grants. |
| Configured Soda operator | Permit the repository to use appliance capacity within explicit resource/concurrency limits; stop appliance execution. This does not grant repository access or merge rights. |
| Provider-account owner | Select and sponsor a supported connection for this repository/project and its assigned execution roles, within an explicit usage allowance; withdraw that sponsorship. |

One person may hold several roles, but each authority is checked separately.
Enabling automation records the policy and granting identities; it neither creates
a human project membership nor bypasses environment creation/ownership rules.
Execution uses a distinct scoped run identity, not the browser session or a
developer's login. The connection owner, execution sponsor and native publication
or merge actor remain attributable rather than impersonating an issue author.

Every new issue is considered. Automatic agent work requires an accepted objective
from a currently code-write-authorized human collaborator, or an explicit adoption
of that objective by such a maintainer. An issue from another source waits for
adoption; reading or merely commenting on it is not adoption. Factory agents cannot
create new spending authority by filing their own issues. Routine accepted issues
proceed under the standing policy without per-issue administrator approval.

Only write-authorized maintainers can adopt changes to requirements/dependencies,
confirm product decisions, or pause, cancel, resume, retry or take over repository
work. Other comments are context or suggestions, never policy or automatically
accepted requirements. A material issue edit suspends work until its accepted
revision is re-established; edits by unauthorized sources cannot silently replace
that revision. Native repository administrators change repository factory policy;
only the operator can enlarge appliance capacity and only the provider owner can
enlarge account sponsorship. Every action remains bounded by the other grants.

Policy is held outside agent-writable checkouts. An agent-authored configuration,
label or instruction cannot enable automation, waive a blocker, grant a role or
raise a limit. Policy changes and grant/permission loss invalidate affected active
authority and require reassessment. Permission to stop or reduce one's own grant
is not permission to grant someone else's authority. Native logout ends browser
control, not a separately authorized unattended grant; explicit revocation and
current native permissions govern that grant.

An agent may work in its assigned environment, but its execution identity does not
confer host, Soda operator or Forgejo administrator authority. Soda mediates native
writes and automatic merge within the authorized repository scope. Agents cannot
grant themselves broader access, receive general Forgejo write authority or bypass
native repository policy. Issue content, comments and labels cannot enlarge the
factory's configured authority.

Review is assigned to another agent session with its own execution identity in
the same repository container, as required by the
[review rules](../product/overview.md#review-and-correction) and the
[Project execution boundary](#project-execution-boundary). The reviewer cannot
publish candidate edits or merge. Distinct assignments do not guarantee independent
reasoning.

Reusable subscription credentials are a narrow exception for private trusted work.
Keep them restricted to the assigned execution boundary and out of source, logs
and retained artifacts. A provider credential stream may be reused serially across
runs, with credential state maintained by its supported CLI. Deleting a workspace
does not revoke a provider credential; provider revocation is a separate operator
action. Code in the trusted runtime may extract its injected provider credential.
Containers share a kernel, and an allowlist does not prevent exfiltration through
an allowed service.

Keep four identities distinct: the authorizing human, the principal of one Soda
run, the native actor publishing a candidate, review or merge, and the provider
account supplying model access. A role is a policy template, not authority an
agent can grant itself or its subagents. An issue may trigger work under an
existing policy; it is not an execution grant on its own.

Publication and merge require active authority. Revocation, cancellation or closed
work withdraws further execution, publication and merge authority. Review and CI
must cover the exact candidate; new commits invalidate earlier evidence. Native
repository state and policy must still permit the merge when it occurs. Cleanup
operates only on recorded run-owned resources; a run does not acquire ownership of
a persistent environment merely by using it.

The [factory reference](../reference/factory.md) describes how the existing
operator command enforces its narrower admission and publication model. It does
not impose manual admission, human merge or disposable containers on the target
product.

Factory status follows native repository visibility. Live CLI views and human
input/takeover require current repository code-write authority, as human terminal
access does; viewing does not itself grant control. Provider credentials and
private account custody are not exposed through work status. Human changes to the
candidate after takeover require the same fresh review and checks before automatic
merge. Human intervention cannot mark missing or failed evidence as passing or
bypass native protection; a separate manual merge remains the human's native
forge action, with its actual author and outcome.

## Project execution boundary

The [environment model](../product/projects.md#environment-relationships) places
humans and factory roles inside one persistent Project. Two dedicated, persistent
nonhuman Linux accounts separate coding and reviewing state. These accounts have
no human membership, external SSH access, sudo/wheel rights, shared engine socket
or general native forge credentials. Fresh per-run homes/configuration and private
transient credential storage must not inherit human or earlier agent settings.
The native runner may execute under a role UID without enabling interactive login.

Project Start and Stop require the configured Soda operator or the repository's
current native owner, including native organization ownership where applicable.
Recheck that authority on either action; membership, code-write permission or
repository administrator permission alone does not grant shared-container lifecycle
control. Start may clear the Project lifecycle hold, but cannot clear independent
repository/operator factory pauses or restore withdrawn provider sponsorship.

Each run receives only its assigned writable checkout, scratch and approved test
resources. Human homes, the other role's state, shared Git administration, project
configuration and durable service data are not agent-writable. The native publisher
holds collaboration authority separately; agents receive neither publication nor
merge credentials. Explicit provider sponsorship remains necessary even when the
Project and role account already exist.

A role UID identifies an account, not a live run. Broker attestation and execution
control must bind the repository/Project, actual container, role account, run,
assigned checkout and native process/service generation. Secrets enter only that
execution's private transient storage. Stop and credential capture cover all its
descendants and cannot pause or kill the Project container or unrelated sessions.
Failure to confirm termination/return leaves the affected execution and connection
unavailable under the [interruption rules](../product/overview.md#human-intervention).

Project root/wheel remains trusted and can inspect accounts, processes and secrets.
This is a trusted-team runtime, not hostile-tenant isolation; different accounts
do not remove that limitation. Native permission, process and credential boundaries
must be proved before factory admission to this runtime. Soda's privileged helper
and broker sockets remain outside the Project, and agent control cannot become an
arbitrary helper command. Human takeover uses a separate member-owned execution,
as defined by [Projects](../product/projects.md#accounts-and-checkout-ownership).

## Fountain consumption boundary

Fountain is a generic Forgejo extension platform. It owns native identity,
sessions, repository permissions and supported ways for extensions to act with
bounded authority. Soda owns its factory policy, environments and Identity Broker;
those product concepts do not belong in Fountain.

If Soda appears to need an authentication or native-authorization workaround,
first check the selected Fountain version, its supported contract and the actual
Soda caller. If correct consumption cannot satisfy the requirement, the missing generic
capability is a Fountain shortcoming to resolve there. Soda must not compensate
with borrowed cookies, synthetic native sessions, login relays, impersonation or
a parallel OAuth login. This applies to background factory operations as well as
browser contributions; background authority cannot depend on keeping a person's
browser session alive.

Provider enrollment and account custody remain legitimate Identity Broker duties,
separate from native forge sign-in. The concrete grant protocol and unattended
publication/merge integration must satisfy the following contract before factory
mutations are enabled; this document does not assume it is already available.

### Conditional native mutations

Use separately authorized, repository-scoped native service principals for
unattended collaboration. An ordinary native actor credential is not an
authentication hack, but its repository scope alone does not encode a current
Soda attempt or sponsorship. Keep publisher/reviewer/merge authority outside agent
processes. Browser session callbacks remain for browser operations; no live human
session is required to sustain a valid background grant.

Fountain/native Forgejo must enforce the required operation constraints at the
native mutation boundary. Soda supplies its policy decision through a bounded
operation authorization; Fountain treats application authority identities and
revisions as opaque. The contract must:

- Bind the installed extension, native actor, repository, permitted operation and
  exact target. Bind relevant object/ref expectations and a revocable application
  authority revision; neither an arbitrary command nor a reusable broad bearer
  grant to an agent is an acceptable substitute.
- For merge, require the reviewed head, verified target branch/base and current
  operation authority together. Reject stale expectations at the effective native
  write, including changes after merge preparation. Preserve native permission,
  branch protection, review, status and merge-method enforcement; never substitute
  an unchecked direct push or a separate Soda merge engine.
- Order invalidation and an authorized write at a defined native commit boundary.
  After invalidation is acknowledged, an operation carrying the old authority
  revision cannot commit. An operation that committed first remains an attributable
  completed operation, not something revocation can undo. Soda immediately stops
  new dispatch on withdrawal, and reports pending invalidation honestly until the
  native boundary confirms it; a preflight followed by an unfenced write is not
  this guarantee.
- Deny the operation when current required authority or state cannot be established.
  Unknown outcomes require authoritative lookup and reconciliation, not blind
  retry, assumed success or continued use of an old authorization.

Soda owns what invalidates its authorization: accepted-input or policy changes,
blockers, pause/cancel/takeover, limits, repository sponsorship, operator capacity
and provider sponsorship. Fountain owns generic native enforcement and native
permission checks, without learning issue-readiness rules, budgets or provider
secrets. A change in either authority invalidates affected work; native checks do
not replace Soda's product checks.

This selects a minimal conditional-operation boundary, not a replacement identity
system or an unrestricted durable delegation framework. The logical contract below
covers merge, candidate ref publication, PR creation and review submission. The
coordination mechanism below is selected for the first single-host profile.
The integration design below selects authentication, writer
boundaries, recovery authority and initial methods; their implementation still
requires native qualification. Additional mutation kinds need their own contracts.
The current merge input's `head_commit_id` check during preparation does not
satisfy expected-base or application-authority requirements. Automatic merge remains unavailable until
the [mutation acceptance cases](../development/testing.md#factory-acceptance) pass;
no weaker temporary path is part of the target architecture.

#### Operation identity and authorization

The following names describe a **target logical API**, not existing SDK methods
or HTTP routes. `git.ref.publish`, `pull_request.create`,
`pull_request.review.submit` and `pull_request.merge` are separate native effects,
each with its own immutable authorization and receipt. Authorization for one kind
never grants another action or the whole factory sequence.

The host derives the installed extension identity from authenticated service
transport and resolves the native actor through native authentication and an
explicitly permitted extension/principal binding. Browser admission is not that
background binding. A JSON `actor_id`, username, repository URL or operation ID
cannot authenticate the caller or select an arbitrary impersonated actor. The
binding uses the [background authentication design](#background-authentication)
below. It is a target Fountain capability, not a synthetic native session.

Each authorization permits **one immutable operation**. Its identity is the pair
`(extension_installation_id, operation_id)`; cancellation revokes that operation,
not the actor's other native credentials or another extension's work. Soda records
the operation and its current application authorization revision before sending
it. A repository/attempt-wide withdrawal closes Soda dispatch first and cancels
every recorded outstanding operation. Registration and withdrawal must be ordered
in Soda so that no concurrently created operation escapes that cancellation set.
Fountain does not need a reusable hierarchy of factory grants.

| Field | Meaning and validation |
| --- | --- |
| `extension_installation_id` | Host-derived installation identity, stable across process restart; a different installation cannot adopt old operations. Not caller-selected authority. |
| `operation_id` | Caller-generated opaque unique ID, durably recorded before sending and never reused for another intent. Scoped to the authenticated installation. |
| `actor_id` | Expected stable native user ID; must match the authenticated, permitted service principal. Native account state, token scope and repository permissions remain effective. |
| `repository_id` | Stable native ID of the repository receiving the primary effect; the base repository for PR operations. Resolve by ID; do not authorize by a mutable owner/name or supplied clone URL. |
| `kind` | One of the four operation kinds defined here; unknown kinds and arbitrary native commands are rejected. |
| `authorization_revision` | Nonempty opaque application revision identifying the policy/accepted-input/grant decision behind this operation. Bound to the entire immutable intent. Fountain records validity and cancellation without interpreting that revision as Soda policy. It is not a bearer secret or a reusable grant. |
| `expected_native_revision` | Opaque host-issued revision obtained with the stable native-read protocol below. Reservation acquisition requires equality with the current idle revision. It binds observed native inputs without telling Fountain what Soda considers eligible. |
| `not_after` | Finite deadline for host admission into the guarded commit phase, checked at Git `prepared` or the database admission boundary defined for that kind, using host time and constrained by the host's maximum lifetime. An admitted write may finish afterward. Expiry prevents new admission; it does not release a reservation, undo a write or establish non-commit. No in-place renewal. |
| `payload` | Exactly the payload for the selected kind below; all semantic fields participate in intent identity. |

Native IDs use the SDK's decimal-string representation. Ref names are full native
branch refs; object IDs are full IDs in the repository's object format, never
abbreviations. Malformed, unknown or conflicting fields fail validation rather
than silently broadening or defaulting the request. Host-derived fields are
returned in the record, not trusted merely because a request repeats them. Intent
equality compares validated semantic fields, not JSON whitespace or key order.

#### Merge intent and effect

| Merge payload field | Required meaning |
| --- | --- |
| `pull_request_number` | Native PR number within `repository_id`; host resolves the native PR identity and verifies its source and target. |
| `head_repository_id`, `head_ref` | Exact native source repository and branch of that PR. Retargeting either invalidates the request even if an object ID happens to match. |
| `base_ref` | Exact target branch in `repository_id`; it must still be this PR's target. |
| `expected_head_oid` | Candidate that received review and required checks; the current PR head must still equal it at the effective write. |
| `expected_base_oid` | Target tip against which the candidate was verified; the target must still equal it at the effective write. |
| `method` | Exactly `fast-forward-only` for the first conditional interface, permitted by current native repository policy. No implicit fallback, force, bypass, delayed auto-merge scheduling or manually-merged mode. |

The effect is one native merge updating the identified base ref through the native
merge engine and its normal hooks. Branch deletion is outside this operation.
Native permission, protection, review/status and mergeability failures refuse it.
Soda remains responsible for the full [merge eligibility](../product/overview.md#automatic-merge-conditions)
decision and withdrawing its authorization when that decision ceases to hold.
An opaque revision alone is not proof of current eligibility: its binding to
accepted native inputs and Soda policy must be established by the consumer and
the enforcement design before enabling merge.

Expectations, active authorization and the effective ref update must be ordered
together against cancellation and relevant native changes. Holding a lock during
preparation, or approving in a pre-receive hook and dropping the guard before the
ref update, is insufficient. This defines the required invariant without claiming
a Git/database locking mechanism already satisfies it.

#### Candidate ref publication

`git.ref.publish` permits one branch creation or fast-forward update in one native
repository. It does not create a PR, change another ref, delete a branch or permit
non-fast-forward replacement. Its payload binds:

| Field | Required meaning |
| --- | --- |
| `ref` | One full `refs/heads/...` target, distinct from `comparison_ref`. |
| `expected_old` | Explicit `absent` for creation or one full current OID for update; never an omitted or wildcard lease. |
| `new_oid` | Exact new commit, different from the old tip; an update must be a native fast-forward from that tip. |
| `comparison_ref`, `expected_comparison_oid` | Exact repository branch and tip used to assess the candidate. A changed base rejects publication. |
| `pull_request` | Explicitly absent for initial publication; for correction, the existing PR number and expected author ID. It must remain open/unmerged, in this repository, with this source branch, comparison branch and expected old head. |

Soda validates the candidate bundle in its clean publisher checkout, including
assigned-input ancestry, protected-path and credential checks. It chooses the
exact branch and comparison base, and retains agent/run provenance. Fountain
checks the bound native refs and authority without interpreting those factory
rules. Soda grants publication/create authority only to its enrolled publisher;
its reviewer and agent credentials cannot exercise those operation kinds.

Preserve native smart-HTTP Git transport. `SubmitOperation` durably registers this
intent in `pending/awaiting_receive`; registration does not launch Git or occupy
the native reservation. The publisher then supplies the operation ID and current
installation runtime/service admission alongside its native PAT on Fountain's
supported receive route. This is a new generic Fountain protocol, not a push option
or a header accepted by unmodified Forgejo. The SDK configures restricted secret
files for the configured verified native origin; credentials cannot follow redirects
or appear in command arguments, logs or agent-accessible configuration.

Before starting `receive-pack`, Fountain verifies the installation, kind, recorded
actor/credential generation and route's resolved repository ID. It atomically
claims the reservation, checks its expected revision, requires that cancellation
has not won and the deadline has not elapsed, and changes the durable execution
from waiting to started exactly once. Bind the receiver to the host-only
execution capability; remove caller admission from child environment/diagnostics.
Native push permission and protection checks remain effective. At Git `prepared`,
require exactly the authorized old/new tuple, fresh native authority, unchanged
comparison ref and correction PR conditions, and order cancellation/deadline
against commit admission as for merge. Pre-receive must reject an unexpected
command set before any ref can commit; the prepared hook also enforces the exact
tuple. Git discovery reads do not consume execution; only the matched receive does.

A retry cannot launch another receiver once execution started. A caller restart
may use fresh installation admission to attach to a still-waiting intent, but
cannot renew it or replace an uncertain writer. An ordinary PAT-only push remains
an ordinary native operation; it cannot consume this registration or establish
its success. Matching bytes or an already matching branch are not attribution.
No-op writes refuse; a replay of this operation instead returns its earlier receipt.

The primary effect is the exact branch update; return its old/new OIDs and native
actor, with native push/PR synchronization and event processing as separate
completion. Cancellation before admission prevents publication. After admission,
unknown results retain the reservation until writer quiescence and attribution;
a committed update is never undone by cancellation. Unpublished received objects
are not a branch effect and remain subject to native Git cleanup. Lost push replies
use `GetOperation`, never a replacement push to discover whether the first worked.

#### PR creation

`pull_request.create` creates one same-repository native PR. The payload fixes
`head_repository_id` equal to `repository_id`, full `head_ref` and `base_ref`,
`expected_head_oid`, `expected_base_oid`, final `title` and `body`, and
`allow_maintainer_edit=false`. Reject identical refs, missing branches, native
no-change/duplicate refusals and invalid or over-limit content rather than silently
truncating intent. The first interface has no labels, assignees, milestone,
attachments, fork source, update or retarget operation. Soda puts the accepted
issue/attempt linkage into the fixed body using native references; Fountain does
not accept a Soda-specific issue or attempt grant.

The publisher's approved native operation binding must include PR creation.
Preserve native repository-unit access, head/base permissions, account/block and
quota checks from the API layer, refreshed under ownership. Acquire before the
native duplicate check, comparison preparation and index allocation. An existing
PR is a conflict, even if its author, branch or text matches; only this operation's
receipt proves creation. Corrections validate the already recorded PR instead of
invoking create again.

The primary effect is the native issue/PR database commit. Verify exact refs and
native authority under the held reservation, outside SQL. Split the existing
native service at its model transaction: atomically check cancellation and expiry,
verify owner/intent, then commit issue/PR records and the operation's
`committed` receipt together. Record native PR ID, issue ID, PR number, author and
head/base snapshot. Cancellation must contend on the same operation row/conditional
update so it cannot acknowledge prevention while that transaction commits. A
winning cancellation rolls back/prevents the primary effect; a committed
transaction makes cancellation `too_late`. Admission expiry is not a physical
latest-commit guarantee. No SQL transaction may span Git or wrap the entire service.

After that commit, use the allocated native index to establish the exact internal
PR ref and perform native push-history comments, code-owner requests, mentions
and notifications under bounded completion ownership. Preserve these native
behaviors; do not treat an API error after commit as failed creation. A PR can be
visible while its derived state is incomplete. Report `committed` with completion
`pending` or `needs_intervention`; downstream factory work waits for its required
native state and event handoff. Do not delete/recreate the PR, adopt a similar PR
or blindly rerun uncertain completion to repair a lost response.

Before derived Git writes, persist their exact internal-ref tuple and allowed
completion phase under this owner. Native hooks require that bound execution proof
and the committed primary receipt; this is bounded completion, not permission for
another primary write. An unexpected or uncertain internal ref retains the fence.

#### Review submission

`pull_request.review.submit` submits one final, body-only native review. Bind the
PR number, expected PR author ID, head repository/ref, base ref, exact
`expected_head_oid` and `expected_base_oid`, explicit `commit_id` equal to that
head, final `event` (`APPROVED` or `REQUEST_CHANGES`) and `body`. Its first scope is
same-repository PRs. Inline comments, attachments, pending/draft creation,
submission of an existing draft, editing and dismissal are outside this operation.
A pending review for this actor/PR refuses; do not adopt its comments or delete it.

Use the distinct enrolled reviewer and its approved review-only operation binding.
Preserve native access, account/block, self-review and official-review eligibility
rules; do not call the model as a permission bypass. Soda owns fresh-session
independence, findings and evidence, and rejects approval of the reviewer's own
changes. An agent supplies findings, not native credentials or an approval grant.

Acquire before loading the PR and its review state. At database admission require
an open, unmerged PR with exactly the bound author, source, target and head/base;
no omitted-commit default and no diff-equivalence substitution for a changed head.
Apply the native model's review/comment, official-review and review-request changes
with the operation receipt and ordered cancellation/deadline check in one SQL
transaction. Return native review/comment IDs, reviewer, PR identity, commit,
event and head/base snapshot. The service's mentions and notifications follow the
real commit under bounded completion ownership, not inside an outer transaction.

Cancellation first prevents the submitted review and its accompanying database
effects. Database commit first makes the review historical native evidence with
`too_late` cancellation; it does not withdraw/delete the review. A lost response
reconciles that operation receipt, never another POST or a search for similar text.
Later head/base changes invalidate its use by Soda for merge even if native review
history still marks it approved. A fresh assessment uses a new operation and
current expectations; it never rewrites the old receipt.

#### Sequencing and database recovery

Soda records each operation before dispatch and advances its local ledger only
from attributable native receipts. Publication, PR creation, review and merge are
separate authorizations, not one distributed transaction. A later refusal leaves
previously committed branches, PRs and reviews intact. Cleanup or withdrawal of
native records requires a separately authorized operation; cancellation grants no
such authority. Never fall back to an ordinary push or unguarded POST when the
conditional path refuses. Before each next stage, recheck Soda authority and
bracket fresh native reads with the current idle revision: the previous write
advanced it.

Before correction, Soda invalidates old candidate evidence and cancels outstanding
old-head review/merge intents. A review that committed first remains historical;
uncertain earlier operations prevent advancing. Publish a correction against the
recorded prior candidate and same PR, then require fresh CI and review. Cancellation
of an attempt closes dispatch and cancels every recorded stage operation using the
existing registration/withdrawal ordering; it cannot erase earlier successes.

PR/review notifications, including synchronous Actions effects, use the same
bounded post-primary-effect completion rule as merge. Deferred jobs claim fresh
ownership and must survive a busy gate. Completion needed by a downstream stage
must be observed before that stage advances; queued CI results are still separate
required evidence. A `too_late` cancellation describes the primary effect, not proof
that all admitted completion writers have stopped.

For database-primary operations, native rows and their operation receipt must
commit or roll back together. After lost responses, lookup can therefore identify
the exact result without ref-tip or text heuristics. A transaction error or absent
receipt while a writer may still run does not prove non-commit. After interrupted
execution, the offline recovery authority must establish writer quiescence, inspect
the authoritative transaction result and reconcile derived ref/completion effects
before releasing the exact owner. Inconsistent rows/receipt or uncertain Git
completion stay fenced; restart cannot rerun creation or submission.

#### Submission, lookup and cancellation

| Logical action | Contract |
| --- | --- |
| `ReadNativeRevision()` | Authenticated atomic observation of the host's native mutation revision and idle/busy state from one authoritative database snapshot. Bracket authoritative native input reads with two equal idle observations before binding that revision to an intent. This neither grants access to those inputs nor adopts them for Soda. |
| `SubmitOperation(intent)` | Authenticate the installation/principal, validate the complete intent and durably claim its operation ID before native execution. The same ID and identical intent return or reconcile the existing record, never start a second execution. Different content under the same ID is `intent_conflict`, including a changed authorization revision or expiry. |
| `GetOperation(operation_id)` | Authenticated lookup within the owning installation. Return the recorded intent, effect and native completion evidence. Absence is `not_observed`, not proof that an earlier request cannot still arrive or that a write never occurred. Lookup performs no new mutation. |
| `CancelOperation(operation_id)` | The owning installation can invalidate its operation even if the native actor has since lost permission. Persist cancellation even when submission has not arrived; a delayed submission or duplicate cannot recreate authority. Order cancellation with any in-flight write and return its actual outcome. |

Cancellation and lookup require authenticated operation ownership, not a still-valid
repository write token or possession of the operation ID alone. After extension
disablement, native operator recovery must be able to inspect and invalidate its
operations without re-enabling execution. Disablement blocks new submissions;
it is not evidence that an already dispatched native process has stopped.

Submitting does not promise immediate completion. Closing a connection, timing out,
losing a browser or cancelling a client request does not constitute
`CancelOperation`. A cancellation response is `pending` until its ordering is
known, `cancelled` only when no primary effect occurred and none can occur, or
`too_late` when the primary effect already committed. If the past effect cannot
be established, return
`indeterminate` and keep replacement work blocked; do not claim that cancellation
prevented it. Repeated cancellation preserves the result and cannot relabel a
committed operation as cancelled. This `cancellation_status` is reported alongside
the operation's `effect_state`; it does not overwrite the native write outcome.

#### Outcomes, failures and retries

Every result identifies the owning installation and operation. Once submission
has been observed, include its actor, repository and immutable intent; cancellation
before submission and `not_observed` lookup must report those fields as unavailable
rather than inventing them. Native effect and follow-up completion are separate facts:

| Effect state | Meaning and caller action |
| --- | --- |
| `pending` | Accepted or preparing/executing; no final commit claim. Query or cancel this operation rather than create another. |
| `not_committed` | Authoritative evidence that this operation's primary effect did not occur and cannot later occur. Include a bounded reason such as cancelled, expired, stale head/base/target, conflicting existing output, denied native authority/protection, merge conflict or native failure. A native error alone cannot establish this state. |
| `committed` | The kind's primary effect is attributable to this operation. Return its exact ref tuple or native PR/review record IDs and bound actor/input snapshot as specified above. Cancellation cannot undo it. |
| `indeterminate` | A write may have occurred or may still be in flight. Retain the operation and reconcile native evidence; forbid automatic replacement or duplicate execution. |

For a committed effect, report its required native completion separately as `pending`,
`complete` or `needs_intervention`, with the actual native record references. Git
success followed by a failed post-receive/database step remains a committed write
with incomplete bookkeeping. It must not become `not_committed` because an HTTP
response failed. Soda marks the work complete only after authoritative native merge
and required issue outcomes are confirmed; dependent work cannot proceed on the
operation's Git receipt alone.

Transport errors, server errors, expiry during execution and controller/host restart
do not prove non-commit. Preserve intent identity, cancellation and enough native
evidence to resolve the write across restart. For Git-primary operations, a
transport response or native PR row alone cannot establish attribution when ref
and database state disagree. Database-primary operations use their atomic receipt;
uncertain derived completion cannot erase a known primary commit. An unresolved
primary effect stays indeterminate for intervention; never repeat it merely to
obtain a cleaner receipt.

After a lost response, look up the same ID first. If submission is not observed,
only the identical intent under that ID may be retransmitted, after Soda rechecks
authority; host deduplication still covers a delayed original request. If authority
has been withdrawn, cancel that ID instead. Terminal IDs stay terminal, and a
new operation requires a known prior outcome plus freshly accepted authority and
evidence. Neither process restart nor receipt cleanup may restore an old operation's
execution authority. Completion/cleanup errors are recorded separately from the
irreversible write.

#### Native enforcement placement

The [source trace](../research/factory-capability-map.md#native-merge-enforcement-trace)
locates the final veto checkpoint at Git's `reference-transaction prepared` phase
and the irreversible effect in the subsequent ref-backend finish. Fountain does
not currently generate that hook. The target needs both this checkpoint and a
host-owned ordering guard that remains effective after the hook exits, until the
native writer can no longer commit. This is an enforcement span, not an assertion
that one callback or database transaction makes Git and SQL atomic.

The guard must cover cancellation, the real source and target refs, relevant PR
metadata, native authority/verification inputs and their mutation paths. Preserve
native merge and hook behavior. Its ownership, callback reentrancy, cross-process
lifetime, crash recovery and writer coverage require native proof before use.
Controller death cannot release the guard on the assumption that Git also died.

#### Selected coordination mechanism

Use **one durable exclusive mutation reservation and one native-state revision**
in Fountain's existing database for the first single-host profile. This is a
target capability, not an existing lock. Short atomic transactions claim and
release the reservation; no SQL transaction spans Git execution. The
[mechanism comparison](../research/factory-capability-map.md#conditional-operation-mechanism-comparison)
records why resource-scoped reservations and an inherited OS lock are deferred.

All participating native mutations serialize through this reservation, including
ordinary human/API operations. Participation covers source/base refs, native
actor/token/repository/org/team authority, protections, PR/review/check state and
native issue/comment/dependency inputs. Global scope avoids a resource dependency
graph; unrelated participating changes can still make an intent stale. Independent
telemetry may remain outside only when it cannot affect these inputs. Route
middleware, receive hooks or a database transaction helper alone are not coverage.

The protocol is:

1. Soda brackets its authoritative native reads with two equal idle
   `ReadNativeRevision()` results. Each result reads revision and occupancy together
   from one authoritative database snapshot; separate reads could tear across a
   completed mutation. Input reads must use current database/ref state, not
   stale replicas or caches. Soda decides whether those inputs are accepted and
   eligible, then binds the native and application revisions and exact refs to its intent.
   A stale revision requires fresh reads and Soda's acceptance rules, never
   automatic adoption of changed objectives.
2. A native mutation atomically claims an idle reservation and advances the
   persisted native revision **before its effects**. Every conditional operation also
   compares `expected_native_revision` with the pre-claim value in that transaction.
   Record its execution identity durably before launch; its own revision advance
   does not invalidate its later commit admission. Failed attempts may advance
   the revision; revisions are never reused. This closes the gap before webhook-driven
   withdrawal without putting factory predicates in Fountain.
3. Acquire before native PR/ref locks and SQL mutation transactions. For the first
   merge integration, acquire outside `Merge` and retain ownership through its
   existing preparation, push and synchronous completion. This avoids splitting
   the native engine but also includes preparation/LFS work in the reservation's
   duration. Competing writers reject as busy or defer outside native locks;
   they cannot check the marker and then write without owning it.
4. Native callbacks reenter only with host-authenticated execution/generation
   binding and permission for that specific callback. They do not reacquire the
   gate. Permit required validation and native completion bookkeeping, not arbitrary
   edits to frozen authority/input state. Deferred jobs acquire their own reservation.
   A push option, claimed operation ID or unauthenticated environment value cannot
   grant this privilege; agents and extensions never receive the bypass authority.
5. For merge, in `reference-transaction prepared`, verify the reservation and bound
   native execution, actual source/target identity, the current source OID against
   `expected_head_oid`, and the exact old/new target tuple against the expected
   base and persisted native merge result. Fresh native authority and required
   checks remain effective under the gate. Atomically order cancellation against
   recording the single commit admission, checking `not_after` at that admission.
   Refuse unexpected ref effects or any missing binding. Keep the reservation
   after the hook returns, through publication and native writer quiescence.
   Other kinds use their ref or database admission boundary specified above.
6. Cancellation/lookup control requests remain available while the gate is busy.
   A cancellation recorded before commit admission prevents admission; after
   admission it remains pending until the actual result can be reported. Release
   only after authoritative writer quiescence and effect reconciliation. Completion
   failure may coexist with a known committed effect; unknown effects retain the
   fence and `indeterminate` state.

There is no expiring lease, automatic takeover or replay. Controller death leaves
the durable owner in place, including after restart. Recovery requires intervention
to establish that every admitted writer has stopped and reconcile its actual
effect before explicitly clearing the reservation. A leader PID, hook exit,
command timeout, elapsed deadline or reflog alone cannot authorize clearing it.
Unknown recovery can therefore block participating mutations across the host;
that availability cost is deliberate in this first design. Writer coverage and
callback/process behavior must pass native proof before enabling operations.

**Expiry correction:** `not_after` now limits guarded commit admission, not physical
ref publication. This changes the earlier design's strict physical-write cutoff:
an admitted operation may finish later while cancellation remains pending. A clock
check in a hook, or even moved into Git's backend, can still precede scheduling/I/O
delay. Preserve stock Git and make this narrower guarantee explicit; do not claim a
hard latest-publication time or use expiry to release an unresolved reservation.

#### Background authentication

Use Fountain's existing private Unix callback transport with a new, explicitly
declared background conditional-operations capability. Browser admissions remain
browser-bound. The following additions authenticate background callers without
extending a person's session or introducing another credential store:

1. Fountain assigns a random installation UUID in host-owned installation metadata.
   Replacement, enable/disable and process restart preserve it; removal followed
   by installation creates a new UUID. Old operation records keep their original
   owner. A manifest name, retained extension data directory or runtime `InstanceID`
   cannot substitute for installation identity.
2. After registration, the host delivers a random runtime admission over the
   existing host-to-extension control RPC. The SDK holds it in memory and uses it
   on the existing per-instance callback socket. The host registry binds it to
   installation, current `InstanceID` and declared capabilities; caller JSON
   cannot select another installation. Runtime stop revokes that admission.
3. An external service bootstraps on the existing shared service callback socket.
   Both peers verify configured native Unix peer identities; the host maps the
   observed service peer to exactly one permitted installation. Require explicit
   operator configuration and `native.service.bridge`; socket group membership or
   manifest declaration alone is insufficient. Reject ambiguous mappings. Verify
   the actual kernel UID mapping in the deployed user namespaces before enabling
   this path. Bootstrap issues a separate admission for the current runtime, kept
   only in service memory. It requires no browser and must not block host startup.
4. Service restart bootstraps again and atomically replaces its delegated admission;
   runtime restart invalidates previous runtime/service admissions. A disconnected
   service reconnects and looks up the same operation IDs. Transport revocation
   neither erases operations nor establishes their cancellation or non-commit.

Submission additionally presents a real native repository-restricted personal
access token (PAT) over that authenticated private transport. Use native PAT
verification and permission reduction, without the general OAuth/Actions-token
fallback. A native administrator explicitly approves the binding of installation
UUID, token ID, actor ID, repository ID and operation kind. Effective authority is
the intersection of that binding, manifest capability, native token scope/resource
restrictions and current native account/repository permissions. `actor_id` only
checks equality with the verified token owner.

The caller keeps its PAT in its existing protected service inputs, outside agent
execution. Fountain discards the presented secret after verification and records
token ID, actor provenance and a private fingerprint of the native credential's
hash/salt with the operation. This detects regeneration of the same token row;
the fingerprint is neither a bearer credential nor public receipt data. Native
verification must compare the presented PAT against the current authoritative
hash/salt before capturing that fingerprint, including after a cached-ID lookup.
Under the acquired reservation, compare that fingerprint and reload current token,
account, binding and repository authority, then run the native mergeability checks;
`Merge` alone expects its caller to have done those checks. At prepared admission,
revalidate the bound credential generation and native authority from authoritative
state. Earlier route authentication and cached permissions are insufficient.
Credential rotation authorizes future operations; replay cannot replace an
existing operation's credential, including by regenerating its token ID, or renew
its intent.

Owned lookup/cancellation require the installation admission, not a still-valid
PAT or repository write permission. They grant access to that installation's
operation receipt and cancellation only. Disablement blocks submission and revokes
runtime/service admissions; native administrator intervention remains available.
Installation/binding retirement must order invalidation of its outstanding
operations through the same cancellation control path, reporting pending effects
honestly before claiming withdrawal is complete.

These are attribution and lifecycle boundaries among administrator-trusted
components. They do not sandbox extensions sharing the native OS identity.

#### Native execution binding

The host records an execution ID and generation when it claims the reservation.
Create an unpredictable execution capability, store its verifier with the durable
owner and place the secret in a host-private mode `0600` file. Pass only the file's
path in the sanitized environment of the final native Git child. The extension,
external service and agent receive neither that capability nor its file path.
Credentials must not appear in argv, operation intent, logs or receipts.

Before the push, bind the native prepared result, old/new target tuple, source
identity, actor/token provenance and permitted hook phases to this execution.
Generated native hooks read the restricted file and present execution proof on
the existing private internal channel. Its shared `INTERNAL_TOKEN` authenticates
the channel but does not identify an execution. Require both proofs and validate
the exact owner/generation, route, phase, repository and ref effects. An operation
ID, pusher ID, push option or caller-supplied environment flag is never reentry
authority. Required fencing must not inherit existing internal-hook skip paths.

Bound callbacks may validate the write and perform its normal synchronous native
completion: branch metadata, native PR merged state and authorized issue closure.
They cannot alter unrelated policy, accept visibility/template push options or
borrow authority for another mutation. They neither claim nor release the gate.
Repeated callback delivery cannot grant another prepared admission or duplicate
completion mutations; return recorded completion or uncertainty instead.

After the primary effect is confirmed, native synchronous publication, PR creation,
review and merge notifications also run under the same host-owned completion
context. Permit the Actions runs, statuses
and configured concurrency-group cancellations attributable to that native event,
including notifications from authorized issue closure. Record their completion
separately from the primary effect. This exception does not permit changing frozen
inputs before primary admission or invoking unrelated Actions APIs. Do not
reacquire inline, discard a busy notification or replay uncertain notifications
to repair completion. Deferred jobs and actual runner updates receive no reusable
execution capability and claim their own owner; the parent does not wait for them.

For ordinary native receives, the host binds the permitted proposed ref tuples
after normal pre-receive authorization under that receive's own reservation.
They do not borrow a conditional merge owner or fabricate application authority.
Direct native ref commands carry their outer operation's host execution context.
Private-channel error handling must redact the execution secret and the existing
internal credential as well as the submitted PAT.

Execution generation differs from extension runtime generation. While its owner
is unresolved, the native execution verifier survives controller/server restart
for its already permitted callback phases; restart cannot launch it again.
Retire the verifier and secret file only after writer quiescence and reconciliation.
A retired generation cannot reenter, even if a delayed callback has its old secret.

#### Participating native writers

Claim at the outer logical native service, command or worker **before** its
authorization-dependent reads, SQL transactions and PR/ref locks. Authentication
may perform an initial refusal before claiming, but credentials and permissions
must be refreshed under ownership before effects. Nested participating model/ref
writers require an existing host-owned context and refuse before effects if it is
missing; they must not acquire late while holding upstream locks. This applies to
ordinary human/native operations as well as conditional requests.

| Writer family | Required ownership boundary |
| --- | --- |
| HTTP and SSH receive-pack | Before receiver launch, including the raw SSH command path; retain through writer completion and bound hooks. |
| Native merge, retarget, branch/ref and file edits | Before PR locks, preparation, SQL or Git effects. Temporary preparation refs belong to that execution; published effects must match its permitted native operation. |
| Repository creation/deletion/transfer/rename, units and protections | Enclose the logical operation and derived changes, including direct model callers. |
| Users, native credentials/keys/auth sources, collaborators, organizations and teams | Include permission recalculation, account blocking/deletion and external-directory synchronization. Changes to token restrictions are authority changes. |
| Issues, comments, dependencies, PR metadata, reviews and conversations | Cover creation, editing, deletion and resolution of native accepted-input or eligibility records. Generated changes use the enclosing ordinary owner or the explicitly permitted merge-completion context. |
| Commit status and Actions task/job/run results | Own the complete logical update before task state changes, including resulting statuses and dependent native state. A status-insert guard alone is insufficient. |
| Deferred push processing, PR tests, Actions jobs, mirrors and maintenance | Claim fresh ownership before their effects; preserve busy items using the existing queue. No inherited completion privilege and no waiting for these workers while holding the parent's reservation. |
| Native administration, hook/key regeneration and offline data operations | Online commands participate before effects; restore/schema/raw data work requires the offline operator boundary below. Regeneration must preserve enforcement hooks. |

Only explicitly identified, non-authorizing telemetry writes may run outside this
reservation—for example, token last-used timestamps or log byte counters that do
not change eligibility. Updating a whole model cannot borrow that exception to
change protected columns. Cancellation/lookup and recovery control keep their
separate authenticated control path. Busy means refusal or retained deferred work,
never an acknowledged permission change or a discarded job.

The [source seam map](../research/factory-capability-map.md#native-integration-source-map)
identifies the selected entry points and lock hazards. Implementation must close
every participating caller, including direct model writes; this family inventory
is not a claim that those call sites are already guarded. Unmanaged direct writes
to the native database or repositories are outside the supported running profile.

#### Recovery authority

The owning installation may inspect and cancel its operations. A current native
administrator may inspect/cancel across installations, including disabled ones;
this does not grant online force-release authority. Normal successful execution
releases through the host's recorded completion path only after quiescence and
attribution. An interrupted or indeterminate execution requires the **host
operator**, using a native offline reconciliation command.

The first supported deployment must place every native writer for its data set in
an operator-controlled, fully stoppable execution domain. Recovery stops HTTP/SSH
write ingress, native services, workers, scheduled/admin commands and all surviving
native descendants, and inhibits restart throughout reconciliation. Verify the
whole domain has stopped using the deployment's native service/container controls;
main-process exit, PID disappearance or a process-group signal alone is insufficient.
External SSH writers must be included in that domain or disabled for this profile.
No new per-operation supervisor or automatic takeover service is selected.

With exclusive offline access, the command matches the persisted operation owner
and generation and inspects actual target refs, saved native result and native
completion records. An attributable distinct expected target under the intact
reservation can establish `committed`; a proven pre-admission refusal with no
remaining writer can establish `not_committed`. An admitted operation with an old
target, reflog alone, missing records or inconsistent evidence stays `indeterminate`.
Record native bookkeeping separately; do not repeat the merge to repair it.

Only a known effect plus established quiescence permits a conditional database
update clearing that exact owner and retiring its capability. Preserve the terminal
operation and cancellation outcome; never reset the native revision or erase the
record to unlock the host. Unknown outcomes remain fenced for intervention. There
is no force-unlock flag, timeout-based release or credential requirement on the
original extension/actor for this host-operator recovery.

Ordinary native owners must persist their writer family and affected resource
identities before effects. They need not fabricate a conditional merge intent.
After the same whole-domain stop, offline reconciliation establishes that family's
actual database/ref or maintenance effects and native consistency before release;
a merge-tip comparison cannot reconcile credential, team or Actions changes.
Quiescence is not rollback evidence. Unclassified or uncertain ordinary effects
keep the reservation occupied until accounted for. This requires family-specific
reconciliation, not a generic undo journal or automatic replay.

#### Initial merge methods

The first conditional interface supports native **`fast-forward-only`**. Use
`doMergeAndPush`'s existing prepared base and result before ordinary push; persist
that result under the reservation. It must equal the reviewed head, differ from
the expected base and satisfy native fast-forward and repository-policy checks.
This method creates no new commit, so there is no merge-message input or rewriting
of the candidate's authorship/signature. Native protection remains effective.

Ordinary merge commits, squash and both rebase styles are deferred from this
conditional interface; ordinary native use still participates in the reservation.
Their shared native preparation seam makes later support possible without another
merge engine. This initial scope matches the demonstrated native path and the
current publisher's requirement that candidates descend from admitted input; it
does not claim an upstream limitation or embed that consumer rule in Fountain.

Unsupported methods, diverged candidates and no-op target updates refuse without
changing repository policy or choosing another method. A repository that does not
permit fast-forward-only cannot use this first conditional merge interface. Force,
manually-merged marking, branch update/deletion and delayed auto-merge are outside
the operation. The prototype supports feasibility only; the authenticated path
still requires its acceptance cases before enabling it.

## Frontend and session boundary

Forgejo's small maintained extension layer owns native contribution routes,
session admission, scoped callbacks and one generic persistent browser host.
Installed extensions are administrator-trusted code: backend processes share the
Forgejo operating-system identity and browser assets run on its origin. Process
separation manages lifetime and failure; it does not sandbox extensions or make
their JavaScript safe to install.

Forgejo validates its native session, account state, request origin and core
permission before creating scoped extension authority. Cookies and raw session
IDs remain in Forgejo. A signed-in page, navigation context, repository shown in
the frame or opaque display binding does not grant an extension broader access.
Host callbacks name bounded operations and recheck current authority. Soda also
checks its own membership, configured operator identity, broker consent and
project grants. A Forgejo site administrator is not automatically the Soda
operator.

Multi-request flows use a host-verified binding to one native session generation.
It conveys continuity, not authentication. Logout, session regeneration or an
account switch invalidates the old flow; the next step cannot silently adopt the
new actor. Terminal streams retain authorization for their lifetime and close on
invalid session or lost repository/membership permission. Native state checks are
decisive; observations may accelerate cancellation but do not grant or preserve
access.

Bookmark handlers lead to fixed native views. Page loading, redirects and opening
the persistent workspace do not register a runner, create a terminal or change
project lifecycle state. Native Forgejo authorization protects each view; Soda's
operation rules still govern every private read and mutation.

The Identity Broker holds Codex and Muse subscription credentials and factory
grants. Forgejo account sign-in, WebAuthn and Git authentication remain native;
Soda does not hold Forgejo OAuth grants or mediate project/worker Git requests.
Factory publication uses separately configured actor credentials. See
[Credentials](../reference/credentials.md) and the [factory interface](../reference/factory.md).

## Host helper

The root:soda Unix-socket helper exposes fixed operations, not arbitrary commands,
host Podman flags or a generic forwarding surface. Resolve actor and native target
from trusted state. Existing Linux accounts, homes and permissions remain native
facts; report incomplete provisioning honestly rather than claiming success or
destructively recreating a reservation.

## Maintaining the Forgejo extension layer

The native extension layer adds a maintained Forgejo fork. Keep its source change
small, attributable to the pinned upstream LTS, and reproducible in the appliance
image. Track security and maintenance updates. Verify session admission, operation
permissions, callback bounds, package lifecycle and native routes against the
selected upstream version before updating its pin. Keep the runtime and Soda
service boundaries independent of this maintenance obligation.

Project provisioning, privileged host operations and terminal supervision have
their own Soda and host ownership. The Forgejo process and its trusted extensions
do not gain the Soda database, broker secrets or privileged helper sockets.
