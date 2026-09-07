# Soda environment/access API

The root React `dashboard/` and its duplicate Forgejo workflow adapters have been
removed. This guide describes the **retained Go source**, not an installed rollout
or a completed Sodaspaces tab/drawer. Native Forgejo owns collaboration/account/
administration pages. The existing embedded Go/HTMX pages remain usable while the
supported Sodaspaces integration is implemented.

## Retained operations

| Endpoint | Behavior |
| --- | --- |
| `GET /api/session` | Soda acting identity, CSRF token, explicit Soda operator flag and configured Forgejo browser URL |
| `POST /api/session/logout` | `{}`; delete Soda session/grant and expire its cookie; not global Forgejo/Linux logout |
| `GET/PATCH /api/me/preferences` | Soda-only display name; PATCH `{display_name}` |
| `GET/POST /api/me/development-keys` | Own development-access public keys; POST `{public_key}`; not native Git key management |
| `GET /api/forgejo/me` | Bounded acting-grant identity inspection; native stable ID must match the Soda session |
| `GET /api/environments` | Existing trusted-team Soda reservations; not private Git authorization |
| `POST /api/environments` | `{owner,repository}`; acting-grant native human-owner check, reservation, actual native create; no implicit join |
| `GET /api/environments/{id}` | Provisioning record, nullable live observation, own login and current-authority hint; incomplete reservations remain inspectable |
| `POST /api/environments/{id}/join` | `{}`; own identity/public keys, real fixed native account operation, membership only after confirmed success |
| `GET /api/environments/{id}/members` | Current native human/org owner or explicit Soda operator sees permitted members; otherwise own membership only |
| `GET /api/environments/{id}/connection` | Own membership required; current IP/running state and fixed public Ed25519 host key/fingerprint; `routing_verified:false` |

Environment detail/members report `authority_unavailable` when ownership cannot be
verified. Failure withholds elevated visibility without discarding own membership/
connection access. Cached `owner_id` is historical, not authorization. Resolve current
native ownership by stored repository ID and native organization `is_owner`, not
`is_admin`. These reads do not remap Linux identities/permissions or revoke prior
access. Organization-owned **creation** remains unsupported; do not infer otherwise
from the current-owner visibility check on transferred repositories.

The future repository-context tab/drawer endpoint has **not** been implemented.
Do not invent an endpoint, authenticated embedding mechanism or shared cookie from
this retained API. The leading plan owns that next source slice.

## Browser/session/security contracts

- `GET /login` supports the current `/projects` return target. Fresh requests for
  `/app/` are refused; a previously pending OAuth return to that removed preview
  finishes at `/projects`. Callback query parameters never select the destination.
- New ordinary login requests `read:user read:repository read:organization` for
  remaining callers. Explicit administration consent adds `write:admin` for the
  retained native-authorized People form. Existing grants are not silently revoked
  or rewritten; persisted scopes come from actual upstream introspection.
- Keep single-use state, PKCE, callback binding, cookie protections, session rotation,
  encrypted schema-v3 provider/session-bound grants, serialized refresh and
  logout-winning persistence. No second password or provider-role authority.
- API IDs are decimal strings. Unsafe methods require one exact configured Origin,
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

## Retained callers, tests and packaging

`internal/web/{api,environments_api,environment_authority,provider,auth}.go` and
`internal/{forgejo,store,host}/` retain the actual integration/security code.
Embedded HTML pages in `internal/web/templates/` provide projects, profile/public
keys and native-authorized People, with real native provisioning. The Go command
and its container/service/database keep their existing names; they are not the
removed React directory. No external compiled frontend directory is required.

Focused Go tests retain CSRF/input/error, provider/grant/race, native ownership,
create/join/reservation, key and persistence coverage. Negative route tests ensure
retired forge adapters do not reach provider/helper operations. React-specific DOM
and browser orchestration tests were removed with their callers; their historical
source and U08 logs remain in Git/ignored evidence. Native SSH/Git/workload/lifecycle
and Cockpit tests remain. Sodaspaces browser replacement coverage is pending, not
inferred from historical React results.

See [credential/migration constraints](dashboard-credentials.md),
[leading plan](dashboard-implementation-plan.md), [frontend integration](forgejo-frontend-integration.md)
and [handoff](implementation-status.md) for current scope and performed checks.
