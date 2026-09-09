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
Browser-only joining, optional Forgejo-key selection and Git credentials remain
unimplemented. The [Lit plan](lit-migration-plan.md) proposes new concurrency/page
contracts; they are **not** implemented by merging the scaffold or writing the plan.

## Planned Spaces page — not an implemented endpoint

`/-/soda/spaces` is selected as a Soda-owned Go/template HTML page linked from native
Forgejo's global navigation. The [leading plan](sodaspaces-plan.md#spaces-page--selected-not-implemented)
defines the authorized workspace and selected Soda-owned shell (canonical assets,
fixed native links and labelled Soda identity, not fabricated native context). No
page handler, collection contract or Spaces OAuth return has been implemented. The current required
`repository_id` collection API below remains unchanged; do not remove its guard to
resurrect the old unrestricted catalog.

HTML navigation must resolve the actor from the protected Soda session and acting
grant server-side; it cannot send the JSON API's custom actor header. That is not
permission to weaken existing API checks. OAuth needs a fixed, transaction-bound
Spaces destination, never an arbitrary `return_to`. Authorize rows/counts/metadata
before rendering, retain truthful unavailable states and recheck every action's
permissions. Do not borrow native cookies/CSRF or assume CSS imports Forgejo's
session/template context. The page opens the same authorized drawer/sessions, not a
new terminal implementation. The [Lit sequence](lit-migration-plan.md) proposes
`GET /api/spaces`, per-ID `/api/environments/{id}/terminal-sessions/{terminalID}`
metadata/actions, create-request correlation and fixed `destination=spaces` OAuth
intent. None exists yet; the singleton endpoints below remain current. Update this
wire guide and validators with the actual backend commits, not ahead of them.

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
{"action":"create","expected_user_id":"1","repository_id":"7","csrf_token":"session CSRF value","cols":80,"rows":24}
```

Exact existing attachment uses `"action":"attach","id":"32 lowercase hex characters"`
instead; create forbids `id`. Missing/expired/other-context IDs create nothing.
Rows are 2–300, columns 2–500. Verify actor, association and the same Origin/CSRF check
used by JSON mutations, then fresh acting-provider identity, actual user/repository
consent and repository visibility. Degraded reads cannot launch. Only the stored
original membership login reaches the helper. Post-upgrade authorization failure
closes with sanitized 1008 status, not provider errors or credentials.

Then accept only text JSON `input` with base64 `data` (1–16384 decoded bytes), `resize`
with bounded rows/columns, with no extra fields. Browser heartbeats and socket End
controls are forbidden. First return `session` with `id`, then `ready`, `output` (base64, at most 4096 decoded bytes) and sanitized
`closed`/`reason` frames. Total frames are limited to 32768 bytes; queues/write waits
are bounded. No command, environment, UID, host address or Podman flags select launch.

One managed terminal per login-context/project and one writer; 64 slots globally
including detached/cleanup-unconfirmed slots, no eviction. Pending transports are
separately bounded at 128. Lifetime ownership is independent of the socket. Every
15 seconds recheck current local session/membership and fresh acting-user repository
authority before renewing the 60-second native safety lease. Original session expiry
and a 12-hour native maximum remain hard limits. Logout/rotation/Stop/shutdown cancel
pending and live attachments/owners, serialized against dispatch. No transcript,
input replay, implicit join/start, cross-context adoption or global Linux/SSH revocation.

`GET /api/environments/{id}/terminal-session` is a protected metadata-only read,
with fresh repository authority and own original membership. No query parameters.
It returns `terminal:null` or `id`, `login`, `repository_id`, `ready`, `attached` and
Unix `retain_until` (zero means deliberately active). Ending/uncertain slots instead
include `state:"ending"` or `state:"unconfirmed"`, with false readiness/attachment.
IDs are locators, never credentials. Unknown creation outcomes are looked up, not retried.

`POST` at that path uses normal expected-actor/Origin/CSRF protections and strict JSON:

- `{"action":"end","id":"…"}` acknowledges `{"ending":true}`, not native cleanup.
- `{"action":"return","id":"…"}` clears retention only while attached; otherwise 30 minutes.
- `{"action":"retain","id":"…","seconds":1800}` or `7200` sets a finite deadline.

Disconnect sets 30 minutes only if no deadline exists. Input/output, automatic
reattachment and browser noise never renew abandonment. Missing/expired actions
return 404 and create nothing. Cleanup-unconfirmed slots stay reserved and require
operator inspection; there is no automatic replacement/reconciliation. Registry
state is in memory, not resurrected after backend restart. Native failure cleanup
uses the separate guard/lease, not browser availability.

See the [component contract](terminal-integration.md) for navigation/Refresh restore
and the exact limits of installed `22d8591` versus remaining native/delivery proof.
The designed cleanup receipts, names and multiple-session routes are not implemented;
current disappearance after End is not by itself independent native cleanup proof.

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
| `POST /api/environments/{id}/join` | `{}`; new joins require fresh acting identity, actual user/repository consent and repository visibility by the stored ID before native account provisioning; membership only after confirmed success |
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
- New public keys are installed at explicit join; no automatic later propagation,
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
