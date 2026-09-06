# Dashboard JSON API — initial implemented contracts

**Source implementation in progress, not built or installed.** This documents actual handlers in `internal/web/api.go`, not the complete future API. The [core plan](dashboard-implementation-plan.md) remains the roadmap. The existing HTMX routes continue to work.

## Security and responses

- Same-origin `soda_session` cookie, HttpOnly, Secure for configured HTTPS, SameSite=Lax. Sessions remain server-side and hashed in SQLite. No Forgejo credential is returned.
- Every API response is `Cache-Control: no-store`. Errors are JSON `{ "error": { "code": "...", "message": "..." } }`; no login HTML, HTMX redirects or raw native/provider diagnostics.
- Unsafe methods require one exact configured `Origin`, one `X-CSRF-Token` matching the session, and UTF-8 `application/json`. Cross-site Fetch Metadata is rejected when supplied. Missing Origin is rejected for JSON writes, unlike the retained legacy form contract.
- JSON writes accept one object, reject unknown fields/trailing values, and have a 64-KiB body limit. User IDs are resolved from the session, not accepted in request bodies. Decimal-string IDs preserve int64 precision.
- Unknown/malformed API paths return JSON 404; unsupported methods return JSON 405 with `Allow`. API expiration is 401; storage failure is 503, not an authentication failure. The browser never automatically retries writes.
- Status/error cases include `unauthenticated` (401), `invalid_csrf` (403), `not_found` (404), `method_not_allowed` (405), `body_too_large` (413), `unsupported_media_type` (415), `invalid_json`/field validation (400), and `store_unavailable` (503).

## Implemented endpoints

| Route | Input / response | Authority |
| --- | --- | --- |
| `GET /api/session` | `{user:{id,login,soda_display_name},csrf_token,soda_operator,forgejo_url}` | Current Soda session; `soda_operator` is not Forgejo administrator or host-root authority |
| `POST /api/session/logout` | `{}`; 204 and expired Soda cookie after actual session deletion | Acting session only; no Forgejo/SSH logout promise |
| `GET /api/me/preferences` | `{display_name}` | Existing Soda-local name, not upstream account data |
| `PATCH /api/me/preferences` | `{display_name:string}`; trimmed name, at most 200 bytes; returns saved object | Acting user's Soda record only |
| `GET /api/me/development-keys` | `{items:[{id,public_key,fingerprint}]}`; empty list is `[]` | Acting user's development public keys only |
| `POST /api/me/development-keys` | `{public_key:string}`; returns canonical registered list; duplicate fingerprint is a no-op | Shared legacy/API parser; one public key without authorized_keys options; no native account/key propagation |

These local session/preferences/key handlers do not call Forgejo with the operator token. `/api/forgejo/...` is deliberately not registered until U04's retained per-user grants and supported authorization/scopes are implemented. The API does not expose the local Soda users table as Forgejo's People inventory.

## Browser migration and OAuth return path

The initial React preview lives at `/app/`, `/app/profile` and `/app/help`; Projects and operator People explicitly link to the existing dashboard. Public assets are compiled by Vite+ and served by Go; there is no production Node service.

`GET /login` keeps `/projects` as its legacy destination. `GET /login?return_to=%2Fapp%2F` requests the preview. Only `/projects` and `/app/` are accepted, stored with the hashed single-use OAuth state. The callback uses the stored destination, never a callback-supplied URL. The installed Forgejo callback remains `/oauth/callback`, state/S256 behavior remains, and current scopes are still `read:user`. This routing change is not U04 credential retention/refresh completion.

## Schema v2 and deployment boundary

`internal/store/migrations.go` replaces unconditional startup schema creation with append-only, transactional migrations. Clean databases get v1 then v2; known v1 databases gain `oauth.return_path` with `/projects` as the default, preserving pending legacy sign-ins and user/key/project/membership/session data. Unknown/unversioned nonempty databases, invalid version records and newer schemas are refused. Migration failures roll back rather than repair/recreate missing tables.

The v1 SQL fixture is frozen separately in `internal/store/testdata/v1.sql`. Authored tests cover existing ready/incomplete projects, profiles/keys/memberships/sessions, single-use OAuth state, foreign keys, repeat opening and refusal paths. They have not run in this source-only phase.

**An older binary is not assumed compatible with v2:** its positional OAuth inserts have only three values. Before an approved deployment, retain a consistent database/config snapshot and matching previous artifact; do not roll back by starting an old binary against v2 or rerun first-install over a live appliance. The new command validates its frontend bundle before opening/migrating the database. This work does not alter the existing test VM.

Pending: protected session-bound provider credentials/refresh, broader API DTOs/errors/pagination, native account/admin flows, repository/environment APIs and installed proof. This is the first connected slice, not U03/U04/U05 completion.
