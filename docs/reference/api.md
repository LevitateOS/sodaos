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

| Method | Path | Purpose |
| --- | --- | --- |
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
| GET | `/api/repositories/{repositoryID}/factory/lifecycle/{event}` | Optional-effect configuration with capability proof |
| PUT | `/api/repositories/{repositoryID}/factory/lifecycle/{event}` | Save effect configuration with revision CAS (owner/admin) |
| GET | `/api/repositories/{repositoryID}/factory/lifecycle/{event}/actions?issue=N` | Bounded durable action results for one issue/event |
| PUT | `/api/repositories/{repositoryID}/factory/sponsorships/{connection}` | Connection-owner sponsorship (command envelope) |
| GET, POST | `/api/factory/departments` | Current actor’s reusable product direction |
| POST | `/api/factory/intents` | Save immutable product intent revision with predecessor CAS |
| GET | `/api/factory/intents` | Owner-private bounded request listing with stable cursor |
| GET | `/api/factory/intents/{intentID}` | Current actor’s current intent revision |
| GET | `/api/factory/intents/{intentID}/history` | Owner-private immutable revision history, newest first |
| GET | `/api/factory/intents/{intentID}/allowance` | Current planning allowance head with available balance |
| POST | `/api/factory/intent-allowances` | Save allowance revision with head CAS (body-carried intent) |
| POST | `/api/factory/intents/{intentID}/explore` | Bounded approach/assumption/consolidation/consistency judgments for a current draft; revision fields stay flat with one added `exploration` object (`content`, `challenges`, `recommendation`, `consistency`) |
| POST | `/api/factory/intents/{intentID}/generate` | Native issue graph creation from accepted current scope |
| GET | `/api/factory/intents/{intentID}/generation` | Durable generation outcome for the current revision |
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
| POST | `/api/factory/intake` | Authenticated native webhook intake (HMAC; issues and comments queue readiness assessment) |
| GET | `/api/spaces` | Spaces inventory for the native actor |
| GET/POST | `/api/environments` | List / create environments |
| GET | `/api/environments/{id}` | Environment detail |
| POST | `/api/environments/{id}/join` | Explicit Join |
| GET | `/api/environments/{id}/members` | Members |
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

Deleting a saved development key accepts a JSON body with `confirm_last: true`
when the actor explicitly confirms removal. If that key is the final saved key,
unconfirmed deletion returns HTTP 409. The Store checks this within the deletion
transaction, including concurrent removals. This is the saved-key confirmation
requirement in [Project access](../product/projects.md); it does not revoke an
installed Project SSH key.

## Public probes

| Method | Path | Purpose |
| --- | --- | --- |
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

The feed contains at most 60 local facts, ordered newest first: completed durable intent graph adoption creates deduplicated issue facts, published native PR effects create PR facts, and verified native merges create merge facts only after confirmed original issue closure. Every item contains `id`, `kind`, decimal `issue`, optional decimal `pr`, `created_unix`, `summary`, and `url`. Kinds match the settings names. URLs use current local native issue/PR paths; PR events describe creation rather than current review state. Issue timestamps come from observed native creation and are zero when unavailable; PR/merge timestamps are the durable completion observation. Pending or unconfirmed effects are excluded. Disabling a kind filters existing facts without deleting native history. No email, webhook, or external notification is sent.

Product intent routes use the admitted native actor; body ownership cannot select another principal. Department POST accepts the department object and attributes ownership to the current actor; GET returns only that actor’s departments. Intent POST accepts an `IntentRevision` with immutable `id`, `intent`, `predecessor`, objective, reviewed ideas/tasks, repository, finite `max_issues` and `planning_calls`; the server attributes owner and creation time. A changed predecessor returns 412. Accepted scope must include a repository and tasks, and any selected repository requires current native owner/admin authority. GET is owner-scoped.

Exploration POST accepts `{revision}` for the exact current draft; mismatched inputs return 412. It reserves a finite planning call against both the intent’s allowance and operator credit, records typed advice, and returns a successor revision for human review. Advice grants no acceptance or execution permission. Generation POST requires the actor’s current accepted intent, native repository owner/admin and a current lifecycle session; it uses the enrolled conditional native graph actor, immutable generation identity and durable receipts. Recovery reconciles the existing operation rather than creating a replacement. Generation GET returns the durable outcome, or `{state:"pending",revision_id}` before creation begins. Execution funding remains separate from consultation credit.
