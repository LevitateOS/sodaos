# Soda HTTP API

Soda's product API is private to the Forgejo native extension service. It is not
an independently callable browser API under `/-/soda/`. The public
`/-/soda/avatars/v1/` route serves generated avatars. Forgejo pages and navigation
remain native.

Source owners: `internal/web/api`, `internal/web/auth`, and the extension service
registration in `internal/web/api/extension_terminal.go`.

Factory run status, stop and reconcile use the separate
[`soda-factory` operator command](factory.md). These routes do not create agent
runs or grant factory execution authority.

## Native extension API

The private service socket serves requests from the Forgejo extension runtime.
The extension SDK supplies verified actor, contribution and session-generation
authority to Soda, and application handlers consume that verified context
directly. Mutations also require the native same-origin request checks. These
paths are private service routes, not public URLs to call directly:

Host Tailnet settings require the optional `tailnet` package's admin page and a
current native site administrator. Project Tailnet routes retain Workspace
repository and Project permissions.

| Method | Path | Purpose |
| --------- | ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET | `/api/session` | Current native actor and Soda profile |
| GET/PATCH | `/api/me/preferences` | Actor preferences |
| GET/POST | `/api/me/development-keys` | List/add development-access public keys |
| DELETE | `/api/me/development-keys/{key}` | Remove a development key |
| GET | `/api/me/forgejo-keys` | List the current actor's native Forgejo public keys |
| GET | `/api/repositories` | Native repository picker (`q`, `cursor`) |
| GET | `/api/repositories/{repositoryID}/profiles` | Supported creation profiles |
| GET | `/api/repositories/{repositoryID}/factory` | Factory policy, effective grants and missing authority |
| GET, PUT | `/api/repositories/{repositoryID}/factory/activity` | Local lifecycle facts with observation time/truncation and independent owner/admin activity settings |
| PUT | `/api/repositories/{repositoryID}/factory/policy` | Owner/admin repository factory policy (command envelope) |
| PUT | `/api/repositories/{repositoryID}/factory/operator-grant` | Operator repository permission (command envelope) |
| PUT | `/api/repositories/{repositoryID}/factory/environment-grant` | Owner environment setup permission (command envelope) |
| GET, PUT | `/api/repositories/{repositoryID}/factory/repair` | Explicit owner-funded finite repair grant; current revision, immutable charges and bounded cycle/run facts |
| GET | `/api/repositories/{repositoryID}/factory/lifecycle/{event}` | Optional-effect configuration with capability proof |
| PUT | `/api/repositories/{repositoryID}/factory/lifecycle/{event}` | Save effect configuration with revision CAS (owner/admin) |
| GET | `/api/repositories/{repositoryID}/factory/lifecycle/{event}/actions?issue=N` | Bounded durable action results for one issue/event |
| PUT | `/api/repositories/{repositoryID}/factory/sponsorships/{connection}` | Connection-owner sponsorship (command envelope) |
| GET | `/api/repositories/{repositoryID}/factory/access` | Repository Factory permission, readiness and execution usage |
| GET, POST | `/api/factory/departments` | Current actor’s reusable product direction |
| POST | `/api/factory/intents` | Save immutable product intent revision with predecessor CAS |
| GET | `/api/factory/intents` | Owner-private bounded request listing with stable cursor |
| GET | `/api/factory/intents/{intentID}` | Current actor’s current intent revision |
| GET | `/api/factory/intents/{intentID}/history` | Owner-private immutable revision history, newest first |
| GET | `/api/factory/intents/{intentID}/overview` | Current owner-private request overview; unknown/foreign requests 404 without disclosure |
| GET | `/api/factory/intents/{intentID}/revisions?limit=1..5` | Recent scope revisions with structural deltas and affected work states |
| POST | `/api/factory/intents/{intentID}/explore` | Bounded approach/assumption/consolidation/consistency judgments for a current draft; revision fields stay flat with one added `exploration` object (`content`, `challenges`, `settled`, `recommendation`, `consistency`); supported/reversible choices settle autonomously, only consequential assumptions interrupt |
| POST | `/api/factory/intents/{intentID}/plan` | Plan the exact current draft and operation; 201 names the launched assignment/run/revision. An uncertain start returns 202 with those same admitted identities and `launch_confirmed:false`; inspect the existing run instead of submitting another plan. |
| GET | `/api/factory/intents/{intentID}/plan` | Read the current draft's retained planning status and canonical result; never funds or launches work. |
| POST | `/api/factory/intents/{intentID}/plan/review` | Explicitly compare and check the planning agent's concrete walkthrough against the exact retained current revision; ordinary status reads never invoke JEV. |
| POST | `/api/factory/intents/{intentID}/generate` | Native issue graph creation from accepted current scope |
| GET | `/api/factory/intents/{intentID}/generation` | Durable generation outcome for the current revision |
| POST | `/api/factory/feed/posts` | Submit a post with `repository`, `body`, `idempotency_key` and optional `parent_id`; a reply uses the same post model |
| GET | `/api/factory/feed/posts/{id}` | Post from the communication store or a canonical workflow record, with its parent and optional pipeline or product-change projection; reads never start work |
| POST | `/api/factory/feed/posts/{id}/retry` | Revise the exact authored post or reply using `expected_revision` and `body`, then reassess its intake |
| GET | `/api/factory/feed/posts/{id}/replies?repository=N` | Direct child posts in the authorized thread, with `before` paging; descendants retain their immediate parents |
| POST | `/api/factory/feed/posts/{id}/review` | Current repository administrator and original product owner explicitly opens a post-bound successor draft; strict body `{ "revision": N }` |
| PUT | `/api/factory/capacity` | Operator appliance capacity (command envelope) |
| POST | `/api/repositories/{repositoryID}/factory/actions` | Code-write pause/resume of factory dispatch (command envelope) |
| POST | `/api/factory/runs/{runID}/actions` | Code-write run stop/retry/takeover (command envelope; takeover also requires membership) |
| GET | `/api/factory/runs/{runID}` | Code-write run status: recorded facts, display binding, observed host state |
| GET | `/api/factory/runs/{runID}/output` | Code-write read-only run output WebSocket (status/output/closed frames; input rejected) |
| GET | `/api/factory/commands/{commandID}` | Durable command outcome read-back |
| GET | `/api/repositories/{repositoryID}/factory/issues/{issueID}` | Work detail: acceptance, validity, readiness, attempt, publication/review/correction, checks, merge, closure, links, capabilities, queued requests and observation time |
| GET | `/api/repositories/{repositoryID}/factory/issues/{issueID}/acceptance-preview` | Current revision-bound acceptance facts (optional repeated `comment_id`; metadata only) |
| POST | `/api/repositories/{repositoryID}/factory/issues/{issueID}/acceptances` | Code-write exact-inputs acceptance (command envelope) |
| POST | `/api/repositories/{repositoryID}/factory/issues/{issueID}/withdrawal` | Code-write acceptance withdrawal (command envelope) |
| GET | `/api/repositories/{repositoryID}/factory/issues/{issueID}/assignment` | Current dispatch assignment: exact bound inputs, staged prompt, recorded result, publication operation IDs, effects, completion and cancellation |
| GET | `/api/repositories/{repositoryID}/factory/issues/{issueID}/plan?limit=1..20` | Recorded plan and recent priority comparisons for one native issue (optional before_unix/before_snapshot cursor) |
| POST | `/api/factory/intake` | Authenticated native webhook intake (HMAC; repository creation publishes the Factory opening question; issue events queue readiness assessment) |
| GET | `/api/spaces` | Spaces inventory for the native actor |
| GET/POST | `/api/applications` | Stored application inventory / operator-admitted initial references and administrator grant |
| GET/POST | `/api/repositories/{repositoryID}/application-builds` | Operator stored build inventory / explicit native packaging of one current verified delivery and environment |
| POST | `/api/repositories/{repositoryID}/application-builds/{build}/setup` | Operator creates or recovers the selected build's separate first-owner setup reservation |
| GET/POST | `/api/applications/setups` | Operator provisional owner-setup inventory / admitted initial references without an owner |
| POST | `/api/applications/setups/{setup}/actions` | Operator explicit Begin/Inspect/Complete/Cancel/Renew of the exact setup generation |
| GET | `/api/applications/{id}` | Administrator/operator stored application references and last observation |
| POST | `/api/applications/{id}/actions` | Explicit operator Install/image replacement or administrator/operator Inspect/Start/confirmed Stop |
| GET/POST | `/api/applications/{id}/snapshots` | Operator native snapshot inventory / stopped backup |
| POST | `/api/applications/{id}/restores` | Operator confirmed fresh-data Restore, reconcile or abort |
| GET/POST | `/api/environments` | List / create environments |
| GET | `/api/environments/{id}` | Environment detail |
| POST | `/api/environments/{id}/join` | Explicit Join |
| GET | `/api/environments/{id}/members` | Members |
| GET/POST | `/api/environments/{id}/members/{member}/access` | Owner/operator native member-access preview / revision-bound Revoke or Restore |
| GET | `/api/environments/{id}/connection` | Connection details |
| GET | `/api/environments/{id}/os` | OS/profile observation |
| GET/POST | `/api/environments/{id}/lifecycle` | Lifecycle read / coordinated Start/Stop (dispatch withdrawal, run stop, maintenance hold, no-revive verification) |
| GET/POST | `/api/environments/{id}/access-keys` | Managed access keys |
| GET/POST | `/api/environments/{id}/preparation` | Preparation read / maintenance hold |
| POST | `/api/environments/{id}/preparation/acceptances` | Maintainer requirement acceptance (decision envelope) |
| POST | `/api/environments/{id}/preparation/actions` | Hold / inspect / administrator approval |
| POST | `/api/environments/{id}/identity/launch` | Start a broker-authorized identity lease |
| GET/POST | `/api/environments/{id}/terminal-sessions` | Inspect/reserve terminal sessions |
| GET/POST | `/api/environments/{id}/terminal-sessions/{terminalID}` | Inspect / End / Rename |
| GET | `/api/environments/{id}/terminal` | Native terminal WebSocket attachment |
| GET | `/api/settings/tailnet` | Host Tailnet settings |
| POST | `/api/settings/tailnet/host` | Host Tailnet actions |
| POST | `/api/settings/tailnet/enrollment` | Enrollment policy |
| GET | `/api/environments/{id}/tailnet` | Project Tailnet state |
| POST | `/api/environments/{id}/tailnet` | Project Tailnet actions |

Create binds an immutable profile. Join creates the project-local account and
records membership only after confirmed native success. Terminal details are in
the [managed terminal contract](terminal.md).

### Product changes from feed replies

A reply to a canonical accepted design or verified application result retains
its server-resolved product identity in `source.product` (`intent`,
`accepted_revision`). A client cannot supply or infer that identity. When the
recorded post judgment selects new work, the reply waits for explicit
product review instead of creating an independent issue graph. Ordinary issue
replies keep their existing route.

Post detail and thread reads may include `product_change` with `intent`,
`accepted_revision`, `state`, `can_review` and optional `draft_revision`.
States are `review_required`, `draft`, `accepted` and `superseded`. Decision IDs
retain the native `d` plus 24 hexadecimal characters format. `can_review` is
presentation only: the mutation rechecks the current native session, repository
administration, original intent owner, exact post revision and accepted head.

The review POST returns the complete updated post projection. Its transaction
creates one same-intent successor draft and a receipt on the original post.
It retains prior outcomes and adds the exact attributed request; planning and
acceptance remain separate actions. When the accepted revision has a completed
generation, the successor carries its exact recorded submitted task graph;
incomplete or incorrectly bound generation refuses review. This preserves
planning context without copying new issue or execution authority. Bounds-only
draft edits retain those task identities and effective outcomes. Replay returns
that first draft without moving a later head. A stale post or competing head
returns 412. Once reviewed,
the original post cannot be edited or retried into different work; a new reply
records a further proposal.

Continue through the existing plan, walkthrough review, acceptance and generate
routes. `accepted` describes scope only. It does not claim implementation,
verification or release. Generation retains the original acceptance and
completion only for unchanged, closed, precisely pinned delivered work. New,
changed or open work requires fresh acceptance and completion before whole-app
verification; preview and company release retain their separate lifecycle.

### Project member access

Project member-access GET previews the stable Linux account, installed public
keys and affected processes. POST accepts `{action: "revoke"|"restore",
revision, confirm: true}`. Revocation retains files and identity while removing
installed keys, disabling login and ending the selected account's processes.
The durable hold blocks Join and managed access before native effects. A202
response reports pending or uncertain effects; inspect and retry the same
revision before restoring access. Restore preserves the disabled account until
the member explicitly Joins again with current repository permission and chosen
keys. Forgejo permissions, provider connections and app-local accounts remain
separate authorities.

### Application administration

Application build GET accepts only an optional single `delivery_id` query and
returns `{builds, administrator:{id,name}}`. It reads stored inventory and starts
no native work. POST accepts exactly `{delivery_id, environment}` with required
`environment: "company"|"preview"`. The server resolves the current accepted
intent, canonical F10 delivery and exact passed native verification tuple; the
caller supplies no source commit, path, digest, image or build command. Native
packaging uses only that environment's retained approved-check output and the
selected runtime, verifies the resulting image bytes, then records the artifact.
An older check without that output cannot create a verified build.

A build contains `{id, repository, delivery, artifact, setup_id?}`. Delivery
records `state`, `environment`, intent/revision, delivery/run, source commit and
bundle-file SHA-256. Artifact additionally carries the native invocation and
manifest digest, actual app name, artifact/bundle/storage/tool digests, runtime
image, application image and fixed origin. Build identity includes environment;
preview and company outputs have different intended origins and independent
setup/data identities. Packaging neither starts an app nor grants company
access. Lost-response recovery reads the same recorded build before retrying
the explicit command; it does not invent a successful result.

Build setup POST accepts an empty object. It uses the server-held artifact and
current operator as administrator, reserves fresh app/data/port identities, and
returns the ordinary provisional setup record below. Retrying returns that same
reservation, even after a newer product direction exists. Creating a new setup
still requires a current verified delivery. Completing its verified native
owner receipt links the stopped application to the original project. Preview
setup is never an alias for company data or an employee release.

Initial POST accepts `{application: Spec, administrator_id: positive Forgejo ID}`.
`Spec` is the canonical mechanical reference in `internal/application`; its
artifact/bundle/tool digests and image IDs identify admitted bytes, not an app
manifest. Only the configured Soda operator can register references. Registration
reserves the hostname, loopback port and private data identity without native
effects. Canlang's stopped bootstrap must provision that data separately; no
passwords, provider secrets or app-local users are accepted here.

GET returns `{application, administrator_id: string, observation: null|Observation,
operator: boolean}`; the string preserves the stable administrator ID exactly.
Owner setup uses a separate provisional admission. `POST /api/applications/setups`
accepts `{application, administrator_id}`; `administrator_id` is a positive decimal
string, and `application` contains the ordinary initial reference fields except
`revision` and `owner_team`. Its `id` and `data` are fresh opaque application IDs.
The server fixes a 30-minute expiry and returns `{id, application,
administrator_id, expires_at, state, observation}`. This record's administrator
is numeric; it is not an ordinary registered application. GET returns
`{setups: [...]}` containing pending records and invokes no native operation.

`POST /api/applications/setups/{setup}/actions` accepts
`{action, expected_container, confirm}`. Actions are `begin`, `inspect`,
`complete`, `abort` and `renew`; completion/cancellation/renewal require
`confirm:true`. Begin and Inspect may use an empty generation only while none
is known. Completion requires the exact setup container; Abort may use empty
only when native absence is confirmed. Responses are `{setup,result}`, where
result includes the checked observation and, for Begin, a transient `setup_url`
with the credential only in its fragment. These responses are
`Cache-Control:no-store`. Soda persists no setup token or owner password.
Expiry blocks Begin, while Inspect, matching receipt completion and Abort
remain usable. An expired pending setup can be renewed: `renew` re-arms the
retained reservation for one fresh 30-minute window (at most three renewals,
within 24 hours past expiry) and then re-begins it on the same address and
data, reusing the retained generation when it still exists. Completion requires
a stopped setup generation, exact Canlang receipt, and fresh stopped
owner-bound normal container before the provisional record and ordinary
admission settle atomically.
Repeated settled actions return `{setup}` without replaying native effects.

Inventory is `{applications: [...], complete: boolean}` with at most32 rows.
Only assigned administrators see their records; the configured operator sees
the bounded appliance inventory. Reads do not inspect/install/start native
resources. Apps require a hostname distinct from Forgejo, since cookies ignore
ports. Sessions and app identities grant separate authority.

Authorized linked-project readers also receive minimal `launchable` records:
application ID/revision/name/origin, project, environment and availability.
An optional `delivery_id` comes only from the native observed bundle's verified
Factory join for that same project and environment. It lets an employee open
the exact F10 result; an unmatched result stays unavailable. These records
expose no container, artifact, administrator grant or lifecycle controls.

Actions accept `{action, revision, expected_container, confirm_stop,
replacement?: Spec, replacement_build?: buildID}` with no query parameters or unknown fields. Install is
operator-only, revision1, empty expected container, unobserved initial record.
Inspect is explicit and accepts an empty generation only while state is unknown.
Start/Stop require the exact current observed container. Stop requires
`confirm_stop: true`; other actions require false except image replacement.
Replacement requires the operator, `confirm_stop: true`, next revision and
unchanged identity/admin/data/team/origin/port. Changed artifact/bundle bytes
require identical nonempty storage fingerprints on the old generation and new
image; changed storage schemas are refused before stopping the writer. It
creates a stopped new container; Start remains separate.

For an inventory-backed replacement, `replacement_build` names the exact
server-held build and excludes a manual `replacement` Spec. The server verifies
its current source/delivery and selected environment, and preserves the live
app's ID, name, origin, data, owner and port. A preview build cannot replace a
company app; an unrelated registered app is not a replacement candidate.
`verify_build` explicitly reads the admitted immutable image's actual bundle
without running its code. Its optional observed `bundle_file_sha256` projects a
verified delivery only when it matches the native check's exact source and
environment. The projection names that check's immutable accepted revision;
an older running image retains its historical proof when the current intent
head advances. This grants no new build or release authority. Image labels or
an operator's project link do not supply proof.

Before an admitted native call, the last observation becomes durably unknown.
Only exact matching native references/generation can restore it through CAS.
Failed/interrupted calls return `native_outcome_unconfirmed`; refresh reads
stored uncertainty and never retries. Explicit Inspect can recover current
native evidence. Previous immutable Specs remain available to operator recovery;
failed replacement never fabricates a rollback or readiness. Local HTTP and
browser tests do not certify an installed appliance.

Operator snapshot GET returns `{snapshots: [{id, created_at, application}],
complete}` with Unix-second timestamps and at most64 native snapshots. POST
accepts `{id, revision, expected_container}` for a stopped backup. Snapshot IDs
are opaque `a` plus24 lowercase hexadecimal digits; filesystem paths are never
accepted. Retrying the same backup ID confirms its original immutable snapshot.

Restore POST accepts `{action: "restore"|"abort", id, snapshot, revision,
expected_container, confirm: true}`. It reserves an immutable pending operation
and a fresh data identity before native effects. An uncertain result preserves
the current binding and both data trees, exposes the pending `restore` in the
application view, and blocks ordinary lifecycle writes. Reconcile with the same
payload and operation ID. Only matching stopped native readback advances the
application revision and data binding; its identity, administrator, address,
owner and bundle remain unchanged. Abort is admitted only before replacement
of the original stopped native generation. Previous data and revisions remain
retained. Replaying a completed operation creates no further native effects.

### Terminal readiness and errors

Environment detail and each Spaces inventory row include
`terminal_support: {state}`. States are `ready`, `support_mismatch`,
`inspection_unavailable` and `maintenance_in_progress`; their meaning and native
observation boundary belong to the
[terminal contract](terminal.md#installed-support-readiness). They do not replace
the observed running state, provisioning result or current execution authority.
Members receive no expected/observed helper digests.

Terminal reservation and exact-session operations preserve bounded native support
failures in the JSON error envelope. The same support codes can appear as the
`reason` of a native terminal `closed` frame after WebSocket upgrade:

| HTTP status / code | Required interpretation |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- |
| 503 `terminal_support_mismatch` | Administrator maintenance is required. No reservation or shell is admitted through the mismatch. |
| 503 `terminal_support_unavailable` | Compatibility could not be inspected; retry or request operator inspection. |
| 503 `project_maintenance` | Managed admission is temporarily closed for maintenance or recovery. |
| 503 `terminal_unavailable` | Transport or the exact native outcome is unavailable; inspect the original locator without replaying creation. |
| 403 `session_changed`, 409 `extension_session_changed` | Refresh the current native page; this is a session-generation change. |
| 403 `membership_required`, `repository_write_required` | Existing Project membership and current repository write authority are required. |
| 401 | Native authentication is unavailable; use Fountain's normal login flow. |

Browser messages use status and bounded code, never native stderr or arbitrary
response text. Support inspection and terminal actions do not install helpers,
start stopped Projects or replay Project creation.

Deleting a saved development key accepts a JSON body with `confirm_last: true`
when the actor explicitly confirms removal. If that key is the final saved key,
unconfirmed deletion returns HTTP 409. The Store checks this within the deletion
transaction, including concurrent removals. This is the saved-key confirmation
requirement in [Project access](../product/projects.md); it does not revoke an
installed Project SSH key.

## Public probes

| Method | Path | Purpose |
| ------ | ---------- | ----------------------------------- |
| GET | `/` | Redirect to configured Forgejo home |
| GET | `/healthz` | Health check |

## Security invariants

- Product authority comes from the verified native extension request and its
  current session generation.
- No fabricated native context, HTML relay or borrowed Forgejo cookie is Soda
  authentication.
- Page loads and redirects never register a runner, create a terminal or change
  lifecycle state.
- Request/response bodies are bounded; unknown fields are rejected on strict
  endpoints.

Credential and schema maintenance: [Credentials](credentials.md).

Repository activity returns `{repository, repository_path, revision, settings, items}`. The repository ID is a decimal string and `repository_path` is the current visible native `/owner/repository` path. Settings are the three booleans `issue_created`, `pr_created`, and `verified_merge`; defaults are all true at revision zero. PUT requires `{expected_revision, settings}` with all three booleans, current native owner/admin authority and a current lifecycle session. A stale revision returns 412. These settings are stored separately from execution policy and confer no execution or external publication authority.

Optional-effect configuration names the saving owner/admin as authorizer and the enrolled native UID as submitter; the config view reports `executor_ready` and `submitter_id` (empty while unknown). Enabling an event without a configured executor returns 409 `executor_unavailable`; disabling always saves. Action reads take the original work-issue index as `?issue=N`; rows journal the human `config_owner` alongside the native `target_id` (`pr_number` on PR actions), which may differ from the work issue.

The feed contains at most 60 local facts, ordered newest first: completed durable intent graph adoption creates deduplicated issue facts, published native PR effects create PR facts, and verified native merges create merge facts only after confirmed original issue closure. Every item contains `id`, `kind`, decimal `issue`, optional decimal `pr`, `created_unix`, `summary`, and `url`. Kinds match the settings names. URLs use current local native issue/PR paths; PR events describe creation rather than current review state. Issue timestamps come from observed native creation and are zero when unavailable; PR/merge timestamps are the durable completion observation. Pending or unconfirmed effects are excluded. Disabling a kind filters existing facts without deleting native history. No email, webhook, or external notification is sent.

Product intent routes use the admitted native actor; body ownership cannot select another principal. Department POST accepts the department object and attributes ownership to the current actor; GET returns only that actor’s departments. Intent POST accepts an `IntentRevision` with immutable `id`, `intent`, `predecessor`, objective, reviewed ideas/tasks, repository, finite `max_issues` and `planning_calls`; the server attributes owner and creation time. A changed predecessor returns 412. Accepted scope must include a repository and tasks, and any selected repository requires current native owner/admin authority. GET is owner-scoped.

Exploration POST accepts `{revision}` for the exact current draft; mismatched inputs return 412. It reserves one planning call against the intent’s independent allowance head (409 when unfunded), records typed advice, and returns a saved successor proposal revision: revision fields stay flat with one added `exploration` object, supported/reversible choices settle autonomously, and only consequential assumptions interrupt. Cumulative usage is recorded without an operator-wide admission ceiling. Failed consultations keep the draft and retain the consumed allowance. Advice grants no acceptance or execution permission. Generation POST requires the actor’s current accepted intent, native repository owner/admin and a current lifecycle session; it uses the enrolled conditional native graph actor, immutable generation identity and durable receipts. Recovery reconciles the existing operation rather than creating a replacement. Generation GET returns the durable outcome, or `{state:"pending",revision_id}` before creation begins. Execution funding remains separate from consultation credit.

Repository repair funding uses `PUT /api/repositories/{repositoryID}/factory/repair`
with `command_id`, `expected_revision`, `investigator_runs`, `proposal_judgments`,
`active_minutes`, `active` and `effect`. The current native owner/admin session
supplies the sponsor identity. Revision0 creates the first grant; subsequent
changes use the revision returned by GET or the last command receipt. Limits
are at most4 runs,4 proposal judgments and120 active minutes, with at most2
cycles and30 minutes per cycle. An explicit increase states its effect and
retains every prior charge. An exact command replay returns its original receipt;
changed replay and stale revision refuse. This grant supplies no Project,
subscription membership or native graph permission by itself. GET exposes safe
cycle state, reason and exact worker IDs without private collector/source bytes.
The repository's Factory settings include these repair limits, an explicit
spending checkbox, retained charge usage and episode reasons. An unconfirmed
save retains its exact command for retry; changing or refreshing the view never
silently issues another grant or replenishes consumption.
