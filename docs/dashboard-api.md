# Soda environment/access API

The root React `dashboard/` and its duplicate Forgejo workflow adapters have been
removed. This guide describes the **retained Go source**, not an installed rollout
or a completed Sodaspaces button/drawer. Native Forgejo owns collaboration/account/
administration pages. The old Go/HTMX frontend is also removed. A native read-only
hook/drawer caller is now authored; real browser/proxy proof and mutation controls
remain pending.

## Browser namespace

Source now mounts the API and OAuth routes at **`/-/soda/` on `forgejo_url`**.
Paths below are relative to that prefix: `/api/session` means
`/-/soda/api/session`, and `/login` means `/-/soda/login`. The direct loopback
backend still answers `/healthz` and redirects `/`; unprefixed API/login/callback
paths are not aliases. Caddy forwards only the Soda prefix unchanged and leaves
native Forgejo routes upstream-owned. This foundation is source-tested, not
installed or a completed authenticated drawer. Actor-context guards and
repository-bound OAuth return handling now exist in the backend; native-page
context capture is authored, while the real browser/proxy round trip remains unproven.

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

## Retained operations

| Endpoint | Behavior |
| --- | --- |
| `GET /api/session` | Soda acting identity, CSRF token, explicit Soda operator flag and configured Forgejo browser URL |
| `POST /api/session/logout` | `{}`; cancel this Soda login context, its pending OAuth and session/grant; expire both Soda cookies; not global Forgejo/Linux logout |
| `GET/PATCH /api/me/preferences` | Soda-only display name; PATCH `{display_name}` |
| `GET/POST /api/me/development-keys` | Own development-access public keys; POST `{public_key}`; not native Git key management |
| `GET /api/forgejo/me` | Bounded acting-grant identity inspection; native stable ID must match the Soda session |
| `GET /api/environments?repository_id=ID` | Required single canonical repository ID; fresh acting-user/visibility check, zero or one reservation, current repository context and advisory `can_create`; no catalog |
| `POST /api/environments` | `{owner,repository}`; acting-grant native human-owner check, reservation, actual native create; no implicit join |
| `GET /api/environments/{id}` | Provisioning record, nullable live observation, own login and current-authority hint; incomplete reservations remain inspectable |
| `POST /api/environments/{id}/join` | `{}`; new joins require fresh acting identity, actual user/repository consent and repository visibility by the stored ID before native account provisioning; membership only after confirmed success |
| `GET /api/environments/{id}/members` | Current native human/org owner or explicit Soda operator sees permitted members; otherwise own membership only |
| `GET /api/environments/{id}/connection` | Own membership required; current IP/running state and fixed public Ed25519 host key/fingerprint; `routing_verified:false` |

Collection lookup requires current visibility even for operators/members; provider
failure is not an absent environment. It returns `items` (zero or one), `repository`
(`id`, `owner_id`, `owner`, `name`) and `can_create` (absent and current human owner).
Missing/duplicate/malformed/unknown query fields or queries over 8 KiB return 400.
Creation still takes `{owner,repository}`; the advisory read is never authorization.

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
button/drawer caller is authored and stable-ID creation remains pending. No shared native cookie
or completed authenticated embedding is implied.

## Browser/session/security contracts

Both reviewed fixes—repository authorization and callback/logout cancellation—are
implemented and locally source-tested, not deployed or browser/proxy validated.

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
  Bodies are bounded to 64 KiB; unknown/trailing fields/data fail. Outputs/errors
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
is unchanged. The authored drawer discards stale reads and requires reloading native page
context on resume/BFCache restoration before exposing actions, then compare the
page/session/provider IDs. The [read-only milestone](sodaspaces-plan.md#native-context-and-authenticated-reads)
invalidates on hidden/blurred/pagehide/restored documents and requires an explicit
full native-page reload before further environment reads. It must not auto-reload
away unsaved native form edits; another Soda fetch is not refreshed native context.
An anonymous or mismatched page offers explicit sign-in,
not automatic account switching or mutation replay. Native logout is not global
Soda logout; do not claim atomic cross-system session revocation. Browser behavior
still requires real verification; source DOM doubles do not establish it.

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
and Cockpit tests remain. Sodaspaces browser replacement coverage is pending, not
inferred from historical React results.

See [credential/migration constraints](dashboard-credentials.md),
[current plan](sodaspaces-plan.md), [frontend integration](forgejo-frontend-integration.md)
and [handoff](implementation-status.md) for current scope and performed checks.
