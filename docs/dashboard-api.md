# Soda environment/access API

The root React `dashboard/` and its duplicate Forgejo workflow adapters have been
removed. This guide describes the **Go environment/access contract**. Native Forgejo
owns collaboration/account/administration pages; the Go/HTMX frontend is also removed.
The integrated drawer passed bounded native create/key/join/Copy/SSH proof, followed
by separately approved preserved-state cutover and existing-account observations.
See the [handoff](implementation-status.md#approved-retained-cutover) for exact payloads,
configuration, client reachability and acceptance limits.

The split-view drawer uses
[managed project-local tmux](terminal-integration.md#selected-persistence-mechanism--tmux-source-candidate)
with separate creation, exact attach, detach and HTTP End. Isolated `22d8591` has
bounded same-shell reload/cleanup evidence, not the full native safety/UX matrix.
Browser-only joining and explicit own-Forgejo public-key selection now have local
source/script/browser coverage. Native delivery/access proof and automated outbound
Git credentials remain pending. The [Lit plan](lit-migration-plan.md) steps 1–4 now have local source
and test coverage; concurrent native/CLI acceptance and delivery remain separate.

## Spaces page and fixed OAuth return

`/-/soda/spaces` is selected as a Soda-owned Go/template HTML page linked from native
Forgejo's global navigation. The [leading plan](sodaspaces-plan.md#spaces-page--selected-not-implemented)
defines the workspace and Soda-owned shell (canonical assets, fixed native links
and labelled Soda identity, not fabricated native context). `GET /spaces` now serves
that escaped HTML shell with a server-authorized actor. Anonymous/expired sessions
receive explicit Connect; unavailable grant/provider authority produces 503, not a
complete empty workspace. Queries are refused. Existing repository-scoped collection
guards remain unchanged.

HTML derives the actor from the protected Soda session and acting grant; JSON still
requires its expected-user header, and mutations still require CSRF/origin checks.
Only this HTML route allows local modules and xterm's inline styles through its CSP;
API/avatar restrictions, framing denial, private no-store and no-referrer remain.
No project data, credentials or executable inline bootstrap are embedded.

`GET /login?destination=spaces` accepts exactly one fixed destination and no
`repository_id`. An append-only schema-v6 boolean binds that intent to the existing
OAuth transaction. Callback returns only to configured-origin `/-/soda/spaces`;
omission retains repository/home behavior. Unknown, empty, duplicate and mixed
intents fail. No caller URL, historical `return_path`, additional consent or OAuth
client change is used. See [credential preservation](dashboard-credentials.md#schema-v6-spaces-return).

The page and drawer mount the same Lit workspace and flat terminal-owner layer:
explicit create/exact attach, Rename, Hide, Continue/Keep and confirmed End.
Versioned, actor-scoped session storage holds at most 64 locators, no names,
transcripts or credentials. Legacy exact IDs require fresh metadata; legacy pending
is never guessed. Unknown cleanup preserves the locator; only acknowledged cleanup
can retire it. Storage failure permits live use without guaranteed restoration.
These are local Go/emitted-browser results with synthetic HTTP/socket peers, not
concurrent native tmux or selected-CLI acceptance.

## Browser-only Join and optional public keys

`POST /api/environments/{id}/join` accepts `{"ssh_keys":"none"}` for account-only
provisioning, or `{"ssh_keys":"saved"}` to install the actor's saved development
keys. New browser controls default to none. Legacy `{}` preserves saved-key behavior,
now also permitting an empty set. Identity, fresh repository visibility, Linux-name
eligibility, provisioning and confirmation-before-membership checks remain. Existing
membership returns its original login without applying/removing keys. The native
account script verifies its exact identity receipt, creates password-locked accounts,
and refuses changed/unassociated key files instead of using Join as revocation.

`GET /api/me/forgejo-keys?page=1` lists only the acting user's profile public keys
through native `GET /user/keys`. Pages 1–8 contain at most ten entries; no global
fingerprint query or arbitrary username is accepted. Fresh subject, owner, user-key
type, bounds and normalized public-key checks apply. Responses contain ID, title,
public key and computed fingerprint, plus page/more; reading/selecting does not
persist a Soda key or invoke the helper. Explicit Save uses the existing development
key API; optional Join/Apply remains separate. No automatic upstream synchronization,
Git registration or private-key handling is introduced.

## Operator runner settings

`GET /settings/runners` serves Soda-owned Go HTML; `GET /login?destination=runners`
binds its fixed return through schema v7. No repository ID or arbitrary URL is
accepted. The protected JSON routes are:

- `GET /api/settings/runners`: bounded local inventory, configured slots/listeners;
  not provider online/busy/available capacity. Forgejo links use the configured
  public origin. Unavailable reads are errors, not empty inventory.
- `POST /api/settings/runners`: existing strict runner registration fields;
  Forgejo's internal URL is server-selected. Requires an explicitly supplied native
  registration token. No provider record is borrowed or silently reset.
- `POST /api/settings/runners/{id}/{start|stop|restart|remove}`: body
  `{"confirm_id":"exact-id"}`. Fixed native lifecycle and partial effects follow
  the [runner guide](runners-port.md).

Every read/mutation checks configured operator ID, fresh acting-user identity and
original Soda context. API actor/CSRF/origin guards remain mandatory. Site admin
status grants no access. Tokens never enter responses; failed mutations report
unconfirmed local/provider effects and are never replayed. The shared native runner
lock serializes Cockpit/CLI/web reads and mutations. Local source/fixture checks
passed, not provider/native acceptance; Cockpit remains installed.

## Browser namespace

Source now mounts the API and OAuth routes at **`/-/soda/` on `forgejo_url`**.
Paths below are relative to that prefix: `/api/session` means
`/-/soda/api/session`, and `/login` means `/-/soda/login`. The direct loopback
backend still answers `/healthz` and redirects `/`; unprefixed API/login/callback
paths are not aliases. Caddy forwards only the Soda prefix unchanged and leaves
native Forgejo routes upstream-owned. Actor-context guards, repository-bound OAuth
returns and native-page context capture passed both isolated and retained native
journeys. The namespace was delivered in the separately approved retained cutover.

## Expected actor

Protected calls require **`X-Soda-Expected-User-ID`**, a canonical positive decimal
Forgejo user ID (signed 64-bit range). Missing, duplicate or malformed values return
`400 invalid_actor_context`; disagreement with the authenticated Soda session
returns `403 identity_mismatch` before the handler runs. This header does **not**
authenticate a native browser session, choose the acting account or grant authority.
Normal session, CSRF, grant and operation-specific authorization checks still apply.

`GET /api/session` alone may omit the header to inspect the current Soda identity
and CSRF token. If supplied there, the header is still checked. The native-page
caller must compare its signed-in stable ID with that session and the fresh
`GET /api/forgejo/me` result, not silently substitute the bootstrap identity when
they differ. Explicit Soda logout also uses the expected **Soda** actor and CSRF;
it does not sign out Forgejo or Linux. The retained connection probe checks its
supplied fixture username against bootstrap before using the actor ID.

The terminal WebSocket is the sole transport-specific exception: browsers cannot
send custom actor/CSRF headers on upgrade. Its first bounded message carries those
values instead; ordinary JSON API guards are unchanged.

## Terminal WebSocket — managed-tmux contract

`GET /api/environments/{id}/terminal` requires WebSocket, with no query string (even
bare `?`), subprotocol bearer value or cross-origin upgrade. Require a valid Soda
session, exact configured HTTPS Origin and compatible fetch metadata; own membership
and provisioned state are checked before upgrade. No operator/site-admin/owner bypass.

The first text frame (within 5 seconds, at most 4096 bytes, strict JSON) is:

```json
{"action":"create","request_id":"0123456789abcdef0123456789abcdef","expected_user_id":"1","repository_id":"7","csrf_token":"session CSRF value","cols":80,"rows":24}
```

Exact existing attachment uses `"action":"attach","id":"32 lowercase hex characters"`
instead; attach forbids `request_id` and a nonempty name, create forbids `id` and
requires a fresh random 32-lowercase-hex `request_id`. Optional create `name` is at
most 80 Unicode code points, without control/format characters; it is metadata,
never shell input or a native tmux target. Duplicate correlations in the same context
are refused while a reservation or receipt exists, even across projects. This is not
idempotent creation or a retry service. Missing/expired/other-context IDs create nothing.
Rows are 2–300, columns 2–500. Verify actor, association and the same Origin/CSRF check
used by JSON mutations, then fresh acting-provider identity, actual user/repository
consent and repository visibility. Degraded reads cannot launch. Only the stored
original membership login reaches the helper. Post-upgrade authorization failure
closes with sanitized 1008 status, not provider errors or credentials.

Then accept only text JSON `input` with base64 `data` (1–16384 decoded bytes), `resize`
with bounded rows/columns, with no extra fields. Browser heartbeats and socket End
controls are forbidden. First return `session` with `id`, original `request_id` and
this writer's random `attachment_id`, then `ready`, `output` (base64, at most 4096 decoded bytes) and sanitized
`closed`/`reason` frames. Total frames are limited to 32768 bytes; queues/write waits
are bounded. No command, environment, UID, host address or Podman flags select launch.

An ID-keyed registry retains each original context/token, actor, project/repository,
Linux login, creation/hard deadline and owner. Multiple same-project sessions are
independent, with one writer per ID; 64 slots globally including detached/cleanup-
unconfirmed slots, no eviction. The project-wide pre-upgrade writer refusal is gone;
exact-ID writer admission happens after the bounded authenticated first frame. Pending transports are
separately bounded at 128. Lifetime ownership is independent of the socket. Every
15 seconds recheck current local session/membership and fresh acting-user repository
authority before renewing the 60-second native safety lease. Original session expiry
and a 12-hour native maximum remain hard limits. Logout/rotation/Stop/shutdown cancel
pending and live attachments/owners, serialized with reservation/admission. Stop
cancels every ID and pending transport of that project, across contexts. Provider and
native IO run outside the registry lock, with cancellable owners and binding rechecks. No transcript,
input replay, implicit join/start, cross-context adoption or global Linux/SSH revocation.

### Exact session metadata, actions and creation outcomes

`GET /api/environments/{id}/terminal-sessions/{terminalID}` requires fresh repository
authority and the original own account/context/token. No query parameters or ID aliases.
It returns `{"terminal":null}` for an unknown/unauthorized ID, or a `terminal` object:
`id`, `request_id`, `environment_id`, `repository_id`, `user_id`, `login`, `name`,
`created_at`, `hard_until`, `retain_until`, `effective_until`, `ready`, `attached`,
`state`. Times are Unix seconds. `retain_until:0` means deliberately active;
`effective_until` is the earlier retention/hard deadline, never zero. States are
`opening`, `ready`, `ending`, `unconfirmed`, or receipt-only `ended`.

`GET /api/environments/{id}/terminal-attempts/{requestID}` uses the same authority
and response. It finds only that exact correlated attempt, never the newest session.
**Null/absent records are unknown**, not proof of cleanup or of no native effect.
IDs/correlations/attachment generations are locators, not credentials.

`POST` on the exact session path uses normal expected-actor/Origin/CSRF and strict JSON:

- `{"action":"end"}` acknowledges `{"ending":true}`; read that ID afterwards.
- `{"action":"rename","name":"…"}` sets/clears bounded runtime-only metadata.
- `{"action":"retain","seconds":1800}` or `7200` explicitly sets finite retention,
  capped by the original hard/authentication deadline.
- `{"action":"hide","attachment_id":"…"}` sets 30 minutes **only if unset**.
- `{"action":"return","attachment_id":"…"}` clears retention only for that current
  attached writer. Detached Return remains finite (30 minutes); it may omit the
  generation, but cannot activate a successor writer. A stale supplied generation
  is refused. Hide never replaces a shorter or longer existing deadline.

A writer generation comes from that socket's `session` frame, not a metadata GET.
Generation mismatch returns `409 attachment_changed`, missing target actions 404,
and ending/expired/unconfirmed targets 409. No mutation creates a replacement.
Disconnect sets 30 minutes only if unset. Input/output, automatic attach and polling
never renew abandonment. Effective deadlines are admission/renewal limits, not a
promise that asynchronous native cleanup finishes at that exact instant.

Native acknowledgement alone releases a live slot and may retain an authorized
`ended` receipt: at most 128, five minutes, capped by original authentication.
A full receipt bound never evicts another receipt; the released ID can therefore be
unknown. Expired/absent receipts or backend restart are not cleanup proof. Unconfirmed
native dispatch/cleanup keeps its slot reserved. There is no durable history,
reconciliation, restart adoption or automatic replay.

The old `/terminal-session` endpoint returns **410**, requiring a fresh page; it
never chooses an ID. Old uncorrelated `pending` browser locators remain uncertain.
New callers persist `pending:<request_id>` before creation and an exact ID afterwards;
End acknowledgement alone does not erase the locator. Coordinated future backend/
asset delivery is required; no live rollout occurred here.

### Bounded Spaces collection

`GET /api/spaces` requires the usual actor guard and rejects all queries. It scans
only Soda associations, authorizes each before exposing names/rows or inspecting the
helper, and returns `{"items":[…],"complete":true|false}`. Each item has `environment`
(the existing association DTO, not a live copy of Forgejo names/permissions), original
`login`, `environment_administrator`, `authority_unavailable`, `native_unavailable`,
nullable `observed`, and `terminals` for this original context/account only. Ended
receipts are not history in this collection. Denied nonmembers have no placeholder
or disclosed count. Existing members retain degraded own observations, but degraded
authority exposes neither elevated controls nor session metadata; it cannot launch.
Configured operator observation is not a terminal-authority bypass.

Bounds: four concurrent collection requests; sequential provider/helper calls per
request; an eight-second inspection budget and two seconds per row (existing failed-
grant-refresh cleanup can add up to five seconds). Scan at most 128 associations with
one bounded overflow detection row; publish at most 32 rows and 64 KiB JSON. Stored
text is bounded before loading (ID/IP 128 bytes, name 1024, repository 2048); oversized
rows fail unavailable, not silently truncated. Limits/native/authority unavailability
produce `complete:false` or 503, never a falsely complete empty catalog. There is no
pagination/cursor or total private count; other associations remain reachable through
their authorized repository drawer. Collection reads never create, attach or renew
sessions; expired owners remain subject to their normal cleanup.

See the [component contract](terminal-integration.md) and handoff for installed
`22d8591` versus this local source evidence. Actual concurrent native/CLI proof,
Spaces HTML/OAuth and the shared multi-session UI now consume this contract in
source; full layouts, attention and native acceptance remain separate work.

## Explicit lifecycle and own SSH key updates — source, not installed proof

All ordinary actor/Origin/CSRF/strict JSON protections apply. These routes reject
query parameters and never accept caller-selected Linux identities, targets or flags.

- Lifecycle GET returns `environment` and `boot_enabled`. POST accepts
  `{"action":"start"}` or `{"action":"stop","confirm_stop":true}`. Current project
  administration is freshly resolved through the acting grant; the explicitly
  configured Soda operator is a distinct permitted authority. Ordinary members or
  arbitrary site administrators cannot stop/start. Require a provisioned project,
  existing isolated container and the selected project unit path. Only an empty
  drop-in list or Fedora's stock global
  `/usr/lib/systemd/system/service.d/10-timeout-abort.conf` is admitted; arbitrary
  or project-specific overrides still refuse. This trusts installed host-root
  configuration, not caller-selected policy or byte attestation of root edits.
  As with creation, the installed root-owned project unit is trusted configuration;
  this is not a byte attestation against arbitrary host-root edits.
  Start enables the unit for host boot and starts it; Stop disables boot start and
  stops it, interrupting everyone's sessions/workloads. No desired-state DB copy,
  recreate/repair or automatic rollback. Confirm the same container ID and actual
  resulting running/boot state. Partial/unconfirmed results are 502 and need inspection.
- Access-keys GET requires own existing membership and fresh user/repository consent/
  visibility, without operator bypass. Return `login`, native file SHA256 `revision`,
  `installed_fingerprints`, `saved_fingerprints` and `applied:false`. The target is the
  original marker-bound non-root account in a running project. Refuse unsafe paths,
  ownership, symlinks/hardlinks, options/comments/noncanonical key files; no adoption.
- Access-keys POST accepts that `revision`, the exact reviewed `saved_fingerprints`
  array (including empty `[]`) and `confirm_empty:true` only when removing the last
  managed keys. A changed saved set returns 409 before helper execution. The server
  supplies actual current own saved public keys, never caller key material or login.
  The helper checks the managed file revision before atomic replacement and verifies
  the result. A changed native revision, failed write or uncertain response is not a
  success/retry grant; refresh and inspect. Return `applied:true` only on confirmed
  file update. This is not proof of new-key possession/client SSH reachability.

The dedicated root-owned `/etc/ssh/authorized_keys/<login>` file is the existing
Soda-managed key set. Preview exposes its complete canonical fingerprint set, including
canonical root edits present before preview; explicit Apply confirms exactly which
entries are removed. Changes after preview refuse by revision. Noncanonical/operator
annotations or unsafe metadata refuse rather than being merged. Other accounts,
home files, projects, privileges and authenticated SSH sessions are untouched. Saved
key deletion affects future joins only; explicitly apply to each chosen existing
project. Neither operation revokes Forgejo Git keys, OAuth or browser terminal access.

## Retained operations

The backend also serves the public, read-only
[`GET/HEAD /-/soda/avatars/v1/{hash}` image endpoint](avatars.md#public-image-contract).
It requires no session and performs no identity/database lookup. This image route
does not change the authentication rules of the JSON operations below.

| Endpoint | Behavior |
| --- | --- |
| `GET /api/session` | Soda acting identity, CSRF token, explicit Soda operator flag and configured Forgejo browser URL |
| `POST /api/session/logout` | `{}`; cancel this Soda login context, its pending OAuth and session/grant; expire both Soda cookies; not global Forgejo/Linux logout |
| `GET/PATCH /api/me/preferences` | Soda-only display name; PATCH `{display_name}` |
| `GET/POST /api/me/development-keys` | Own development-access public keys; POST `{public_key}`; not native Git key management |
| `DELETE /api/me/development-keys/{key}` | `{}`; own saved key only; `existing_project_access_changed:false`. No native key removal or session termination |
| `GET/POST /api/environments/{id}/lifecycle` | Source implemented: observed running/boot-enabled state; explicit authorized Start/Stop of existing unit/container, never recreate |
| `GET/POST /api/environments/{id}/access-keys` | Source implemented: own managed-file preview and explicit compare-and-swap of saved keys into that existing account |
| `GET /api/forgejo/me` | Bounded acting-grant identity inspection; native stable ID must match the Soda session |
| `GET /api/environments?repository_id=ID` | Required single canonical repository ID; fresh acting-user/visibility check, zero or one reservation, current repository context and advisory `can_create`; no catalog |
| `POST /api/environments` | `{"repository_id":"ID"}`; canonical decimal string, fresh acting subject/user+repository consent/ID lookup/current human-owner check, reservation, actual native create; no implicit join |
| `GET /api/environments/{id}` | Provisioning record, nullable live observation, own login and current-authority hint; incomplete reservations remain inspectable |
| `POST /api/environments/{id}/join` | `{ssh_keys:"none"}` (new browser default), `{ssh_keys:"saved"}`, or legacy `{}`; new joins require fresh acting identity, actual user/repository consent and repository visibility by the stored ID before native account provisioning; membership only after confirmed success |
| `GET /api/environments/{id}/members` | Current native human/org owner or explicit Soda operator sees permitted members; otherwise own membership only |
| `GET /api/environments/{id}/connection` | Own membership required; current IP/running state and fixed public Ed25519 host key/fingerprint; `routing_verified:false` |

Collection lookup requires current visibility even for operators/members; provider
failure is not an absent environment. It returns `items` (zero or one), `repository`
(`id`, `owner_id`, `owner`, `name`) and `can_create` (absent and current human owner).
Missing/duplicate/malformed/unknown query fields or queries over 8 KiB return 400.
Creation takes only `{"repository_id":"ID"}`. Numeric/noncanonical IDs and the old
owner/name body are rejected. The advisory read is never authorization: create
rechecks fresh subject, actual user/repository consent and current ownership through
`RepositoryByID` before reserving or calling the helper. Rename/transfer does not
select a different association; organization-owned creation and operator/admin
impersonation remain unsupported.

Direct-ID detail/member reads require current repository visibility for ordinary
nonmembers, before native inspection or metadata disclosure. Existing members retain
own degraded reads and the explicit Soda operator retains inspection authority.
Environment detail/members report `authority_unavailable` when ownership cannot be
verified. Failure withholds elevated visibility without discarding own membership/
connection access. Cached `owner_id` is historical, not authorization. Resolve current
native ownership by stored repository ID and native organization `is_owner`, not
`is_admin`. These reads do not remap Linux identities/permissions or revoke prior
access. Organization-owned **creation** remains unsupported; do not infer otherwise
from the current-owner visibility check on transferred repositories.

New-join requests independently check the acting grant's `read:user` and
`read:repository` consent, fresh subject and `RepositoryByID` visibility; neither
operator nor generic administrator status bypasses that check. Use the fresh login
for a new account. Denial/missing grant/consent/provider failure never calls the
account helper or records membership. Existing-member joins return their original
login without reinstallation or a new provider check. This does not continuously
synchronize access or revoke existing Linux accounts after native permission changes.

Repository-scoped reads and new-join authorization are implemented; the native
read-only button/drawer caller passed its bounded native journey. Stable-ID creation
and the explicit drawer actions are now implemented in
[step 4](sodaspaces-plan.md#4-wire-the-explicit-access-actions), with local handler/DOM
coverage and later bounded native provisioning/access evidence. No shared native
cookie is used.

## Browser/session/security contracts

Both reviewed fixes—repository authorization and callback/logout cancellation—are
implemented and locally source-tested. The isolated read-only browser/proxy journey
passed, followed by explicit native access proof and approved retained deployment.
Deterministic callback/logout races retain their focused handler/store evidence;
no runtime failure matrix or whole-product acceptance is inferred.

- `GET /` redirects to configured Forgejo home. `GET /login` accepts optional
  `repository_id` and `expected_user_id` with the same positive-ID representation;
  omit unavailable context, never send zero. IDs are stored with the single-use
  PKCE transaction, not forwarded as provider authorization parameters. Nonempty
  or repeated `return_to` remains refused. Errors are plain text, not templates.
- Callback checks the fresh provider subject against the stored expected user
  before changing profiles/sessions/grants. Mismatch returns 403, consumes the
  transaction and preserves existing Soda state; sign in again explicitly.
  With actual repository consent, the callback resolves the stored repository ID
  through the acting grant, then builds its current native path plus `#sodaspaces`
  under the configured HTTPS origin. Missing/inaccessible/invalid context falls
  back to Forgejo home. Callback-supplied IDs/URLs, provider URLs and the historical
  `return_path` column never select the destination.
- Schema v5 adds a bounded internal login context shared by pending OAuth and the
  rotating Soda session. Its state-hash marker survives the single-use claim during
  provider I/O. Callback finalization atomically checks the marker, saves the profile
  and rotates the session/grant. Logout invalidates the same context, including a
  replacement committed after logout authenticated. Late cookies cannot revive it;
  superseded callbacks do not write cookies or profiles. Other browser contexts are
  untouched. Contexts expire with the login/session; later login initiation removes
  expired contexts and their credential rows, not projects/profiles/keys. No global
  native logout.
  Existing sessions receive independent contexts without changing their credentials;
  pre-v5 pending OAuth requires restarting sign-in, not anonymous completion.
  A stale cookie at explicit login returns 409 and expires Soda cookies so the user
  can start again; failed callbacks never silently retry. Grant encryption/keys and
  project records are unchanged. Auth queries are limited to 8 KiB; malformed encoding,
  duplicate state/code and oversized code/state fail closed. Login/callback use
  `Referrer-Policy: no-referrer` to keep query data out of subsequent referrers.
- New login requests only `read:user read:repository read:organization` for current
  callers; `administration=1` no longer requests extra consent. Existing grants are
  not silently revoked or rewritten; scopes come from upstream introspection.
  A redirect does not establish/transfer a native Forgejo session.
- Cookies are host-only, Secure/HttpOnly/SameSite=Lax, scoped to `/-/soda/` and
  named `__Secure-sodaspaces-session` / `__Secure-sodaspaces-oauth`. Duplicate,
  empty or oversized Soda cookies fail closed. Native Forgejo and old `soda_session`
  / `soda_oauth` cookies are not authentication inputs. Cutover requires a fresh
  Soda login; old pending browser flows restart rather than gain a callback alias.
  Cookie paths do not isolate mutually untrusted applications on the same origin.
- Keep single-use state, PKCE, callback binding, cookie protections, session rotation,
  the existing provider/session-bound grant encryption, serialized refresh and
  logout-winning refresh and callback finalization. No second password or provider-role authority.
- API IDs are decimal strings. Unsafe methods require the exact configured `ForgejoURL` Origin,
  same-origin fetch metadata when present, a matching `X-CSRF-Token` and UTF-8 JSON.
  Bodies are bounded to 64 KiB; duplicate/unknown fields, invalid UTF-8 and trailing data fail. Outputs/errors
  are bounded and sanitized; no credentials in errors or browser responses.
- No generic Forgejo proxy, browser-selected base URL, operator-token fallback,
  automatic retry of uncertain mutations or HTML result masquerading as JSON.
  Unknown/removed API paths return JSON 404; `/app/` no longer serves a SPA.
- Native creation/account or result-persistence failure retains honest incomplete
  state. Do not recreate, prune, replace or claim a failed join succeeded.
- Selected saved public keys are installed at explicit SSH-enabled join; no automatic later propagation,
  Linux offboarding, Git authorization or client-routing proof is promised.

## Native-page and stale-tab boundary

The backend sees a Soda session and a declared page actor, **not Forgejo's live
browser session**. It catches a changed Soda cookie versus the old page actor;
it cannot detect native-only login/logout in another tab while the Soda session
is unchanged. The source workspace now retains its mount/socket on focus/visibility
and Hide changes; those are not authentication loss. Actual pagehide retires the
document's component/attachment; the shell re-mounts and reauthorizes on BFCache
restoration, and native navigation/reload can restore the exact surviving session. The initial native
page/session/provider IDs are compared, and each mutation rechecks the current Soda
session/provider. Another Soda fetch is not fresh native-session authentication;
show the actual Soda actor and do not promise atomic native-only logout. Never
auto-reload away unsaved native forms. The historical
[read-only milestone](sodaspaces-plan.md#native-context-and-authenticated-reads)
used a stronger blur/hidden teardown rule which the user rejected; its passing
checks are not proof of the revised interaction.
An anonymous or mismatched page offers explicit sign-in,
not automatic account switching or mutation replay. Native logout is not global
Soda logout; do not claim atomic cross-system session revocation. The bounded native
journey exercised real blur/BFCache and stock logout navigation while preserving
native unsaved forms. Mutation-time behavior has focused source/DOM coverage, not
native proof inferred from that read-only run.

The drawer separately rechecks session/provider identity on each explicit action,
then sends one protected POST. Create never joins; key save never joins or propagates
keys to existing memberships. Pending actions disable duplicate submission and Soda
logout; Hide remains usable and retains rendering/state without cancelling native
execution. Actual component disposal invalidates rendering but does not undo writes. A lost,
malformed or failed native/persistence response is not success or proof of no effects.
Safe rereads do not replay writes. Uncertain create/join remains blocked in that
document pending operator inspection; there is no persistent browser operation journal
or recovery mechanism. A full reload does not repair an unrecorded native account.

Only a provisioned, currently running own membership exposes a validated current
address, readonly SSH command, native Copy target and public host-key fingerprint.
Stale/closed/unavailable views clear the command/Copy target. `routing_verified:false`
remains explicit; neither the command nor the address proves client routing.

## Retained callers, tests and packaging

`internal/web/{api,environments_api,environment_authority,provider,auth}.go` and
`internal/{forgejo,store,host}/` retain the actual integration/security code.
Templates, static assets, form handlers, repository/People pickers and their dedicated
clients/tests are removed. `/profile`, `/people`, `/keys`, `/projects`, their form/
join routes and `/logout` return 404; use the protected JSON logout operation.
The Go command/container/service/database keep their names. Go serves no embedded
or standalone frontend; native Forgejo serves the four hook/assets. `/healthz`
remains independent of browser rendering.

Focused Go tests retain CSRF/input/error, provider/grant/race, native ownership,
create/join/reservation, key and persistence coverage. Negative route tests ensure
retired forge adapters do not reach provider/helper operations. React-specific DOM
and browser orchestration tests were removed with their callers; their historical
source and U08 logs remain in Git/ignored evidence. Native SSH/Git/workload/lifecycle
and Cockpit tests remain. Sodaspaces read-only browser coverage passed independently
of historical React results; helper-backed native create/join/SSH coverage remains
step 5.

See [credential/migration constraints](dashboard-credentials.md),
[current plan](sodaspaces-plan.md), [frontend integration](forgejo-frontend-integration.md)
and [handoff](implementation-status.md) for current scope and performed checks.
