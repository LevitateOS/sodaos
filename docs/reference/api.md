# Soda HTTP API

Callable Soda routes under the configured Forgejo HTTPS origin at `/-/soda/`.
This is a **reference for interfaces that exist in the current tree**. Product
behavior belongs in [Spaces](../product/spaces.md), [Projects](../product/projects.md)
and [Trust](../architecture/trust.md).

Source owners: `internal/web`, `internal/web/api`, `internal/web/auth`.

## Browser namespace

All routes below are relative to `/-/soda/` unless noted. Protected APIs require the
Soda adapter session, expected-actor header, and CSRF/origin checks on mutations.
Bookmark bridges redirect to fixed native Forgejo views; they do not embed authority.

## Auth and session

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/login` | Start OAuth with fixed destination |
| GET | `/oauth/callback` | OAuth callback |
| GET/POST | `/api/login/cancel` | Cancel pending OAuth / coordinated logout helper |
| GET | `/api/session` | Current Soda session |
| POST | `/api/session/logout` | End Soda session |
| GET/PATCH | `/api/me/preferences` | Actor preferences |
| GET/POST | `/api/me/development-keys` | List/add development-access public keys |
| DELETE | `/api/me/development-keys/{key}` | Remove a development key |
| GET | `/api/me/forgejo-keys` | List Forgejo public keys (`read:user`) |
| GET | `/api/forgejo/me` | Provider identity (`read:user`) |

## Page bridges

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/spaces` | Bookmark bridge to Spaces native view |
| GET | `/repositories/{repositoryID}/settings/spaces` | Repository Spaces settings bridge |
| GET | `/settings/runners` | Operator Runners bridge |
| GET | `/settings/tailnet` | Operator Tailnet bridge |
| GET | `/` | Redirect to configured Forgejo home |
| GET | `/healthz` | Health check |

Repository IDs on bridges must be canonical positive integers. Bridges refuse
arbitrary return URLs.

## Environments and Spaces

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/repositories` | Repository picker for create/inspect (`q`, `page`) |
| GET | `/api/repositories/{repositoryID}/profiles` | Supported creation profiles |
| GET | `/api/spaces` | Spaces inventory for the actor |
| GET/POST | `/api/environments` | List / create environments |
| GET | `/api/environments/{id}` | Environment detail |
| POST | `/api/environments/{id}/join` | Explicit Join |
| GET | `/api/environments/{id}/members` | Members |
| GET | `/api/environments/{id}/connection` | Connection details |
| GET | `/api/environments/{id}/os` | OS/profile observation |
| GET/POST | `/api/environments/{id}/lifecycle` | Lifecycle read / Start/Stop |
| GET/POST | `/api/environments/{id}/access-keys` | Managed access keys |

Create binds an immutable profile. Join creates the project-local account and
records membership only after confirmed native success.

## Terminals

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/environments/{id}/terminal-sessions` | Reserve a terminal ID |
| GET/POST | `/api/environments/{id}/terminal-sessions/{terminalID}` | Inspect / End |
| GET/POST | `/api/environments/{id}/terminal-session` | Legacy session helper surface |
| GET/POST | `/api/environments/{id}/terminal` | Terminal WebSocket attach |

Managed terminals use project-local tmux under Soda lifetime authority. Exact
attach semantics: [Terminal](terminal.md).

## Runners (operator only)

| Method | Path | Purpose |
| --- | --- | --- |
| GET/POST | `/api/settings/runners` | List / register capacity |
| POST | `/api/settings/runners/{runner}/{action}` | Start/Stop/Restart/Remove |

Server-side gate: session user ID must equal configured `operator_id`.
Contracts: [Runners](runners.md).

## Tailnet (operator and project)

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/settings/tailnet` | Host Tailnet settings |
| POST | `/api/settings/tailnet/host` | Host Tailnet actions |
| POST | `/api/settings/tailnet/enrollment` | Enrollment policy |
| GET | `/api/repositories/{repositoryID}/tailnet-options` | Create-time options |
| GET/POST | `/api/environments/{id}/tailnet` | Project Tailnet state/actions |

Projects inherit enrollment policy, never host device identity.
See [Networking](../architecture/networking.md).

## Security invariants

- No fabricated native context, HTML relay or borrowed Forgejo cookie as Soda auth.
- Page loads and redirects never register a runner, create a terminal or change
  lifecycle state.
- Request/response bodies are bounded; unknown fields are rejected on strict endpoints.
- Logout-winning persistence: completed logout denies later admission checks.

Credential and schema maintenance: [Credentials](credentials.md).
