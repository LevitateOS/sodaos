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
specifies merge first. Its authenticated transport, enforcement placement and
coverage of other mutation kinds still need design and native proof. The current merge input's
`head_commit_id` check during preparation does not satisfy expected-base or
application-authority requirements. Automatic merge remains unavailable until
the [mutation acceptance cases](../development/testing.md#factory-acceptance) pass;
no weaker temporary path is part of the target architecture.

#### Operation identity and authorization

The following names describe a **target logical API**, not existing SDK methods
or HTTP routes. Start with `pull_request.merge`; publishing a ref, creating a PR
and submitting a review are separate native effects whose payloads and completion
rules must be specified before adding those operation kinds. A merge authorization
never grants those other actions.

The host derives the installed extension identity from authenticated service
transport and resolves the native actor through native authentication and an
explicitly permitted extension/principal binding. Browser admission is not that
background binding. A JSON `actor_id`, username, repository URL or operation ID
cannot authenticate the caller or select an arbitrary impersonated actor. The
binding's concrete transport is a remaining Fountain design requirement; this
contract does not invent a header or synthetic session to stand in for it.

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
| `repository_id` | Stable native ID of the base repository. Resolve by ID; do not authorize by a mutable owner/name or supplied clone URL. |
| `kind` | Exactly `pull_request.merge` for this first contract; unknown kinds and arbitrary native commands are rejected. |
| `authorization_revision` | Nonempty opaque application revision identifying the policy/accepted-input/grant decision behind this operation. Bound to the entire immutable intent. Fountain records validity and cancellation without interpreting that revision as Soda policy. It is not a bearer secret or a reusable grant. |
| `not_after` | Finite authorization expiry, checked using host time and constrained by the host's supported maximum lifetime. Expiry forbids a later commit; it does not undo a prior write or prove an in-flight write never happened. No in-place renewal. |
| `merge` | The exact operation-specific payload below; all semantic fields participate in intent identity. |

Native IDs use the SDK's decimal-string representation. Ref names are full native
branch refs; object IDs are full IDs in the repository's object format, never
abbreviations. Malformed, unknown or conflicting fields fail validation rather
than silently broadening or defaulting the request. Host-derived fields are
returned in the record, not trusted merely because a request repeats them. Intent
equality compares validated semantic fields, not JSON whitespace or key order.

#### Merge intent and effect

| `merge` field | Required meaning |
| --- | --- |
| `pull_request_number` | Native PR number within `repository_id`; host resolves the native PR identity and verifies its source and target. |
| `head_repository_id`, `head_ref` | Exact native source repository and branch of that PR. Retargeting either invalidates the request even if an object ID happens to match. |
| `base_ref` | Exact target branch in `repository_id`; it must still be this PR's target. |
| `expected_head_oid` | Candidate that received review and required checks; the current PR head must still equal it at the effective write. |
| `expected_base_oid` | Target tip against which the candidate was verified; the target must still equal it at the effective write. |
| `method` | Explicit supported native merge method, permitted by current native repository policy. No force, bypass, delayed auto-merge scheduling or manually-merged mode. |
| `message` | Final requested native merge message, fixed with the intent rather than regenerated from subsequently edited PR text. Native signing and author/committer rules still apply. |

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

#### Submission, lookup and cancellation

| Logical action | Contract |
| --- | --- |
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
known, `cancelled` only when no write occurred and none can occur, or `too_late`
when the write already committed. If the past effect cannot be established, return
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
| `not_committed` | Authoritative evidence that this operation made no target ref update and cannot later do so. Include a bounded reason such as cancelled, expired, stale head/base/target, denied native authority/protection, merge conflict or native failure. A native error alone cannot establish this state. |
| `committed` | The target ref update is attributable to this operation. Return exact old/new target IDs, reviewed head, actor and native PR/merge identity. Cancellation cannot undo it. |
| `indeterminate` | A write may have occurred or may still be in flight. Retain the operation and reconcile native evidence; forbid automatic replacement or duplicate execution. |

For a committed effect, report native PR/issue completion separately as `pending`,
`complete` or `needs_intervention`, with the actual native record references. Git
success followed by a failed post-receive/database step remains a committed write
with incomplete bookkeeping. It must not become `not_committed` because an HTTP
response failed. Soda marks the work complete only after authoritative native merge
and required issue outcomes are confirmed; dependent work cannot proceed on the
operation's Git receipt alone.

Transport errors, server errors, expiry during execution and controller/host restart
do not prove non-commit. Preserve intent identity, cancellation and enough native
evidence to resolve the write across restart. A lookup response or native PR row
alone cannot substitute for attribution when the ref and database disagree. If
evidence cannot resolve the result, leave it indeterminate for intervention;
never rerun the merge merely to obtain a cleaner receipt.

After a lost response, look up the same ID first. If submission is not observed,
only the identical intent under that ID may be retransmitted, after Soda rechecks
authority; host deduplication still covers a delayed original request. If authority
has been withdrawn, cancel that ID instead. Terminal IDs stay terminal, and a
new operation requires a known prior outcome plus freshly accepted authority and
evidence. Neither process restart nor receipt cleanup may restore an old operation's
execution authority. Completion/cleanup errors are recorded separately from the
irreversible write.

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
