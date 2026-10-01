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
| PUT | `/api/repositories/{repositoryID}/factory/policy` | Owner/admin repository factory policy (command envelope) |
| PUT | `/api/repositories/{repositoryID}/factory/operator-grant` | Operator repository permission (command envelope) |
| PUT | `/api/repositories/{repositoryID}/factory/environment-grant` | Owner environment setup permission (command envelope) |
| PUT | `/api/repositories/{repositoryID}/factory/sponsorships/{connection}` | Connection-owner sponsorship (command envelope) |
| PUT | `/api/factory/capacity` | Operator appliance capacity (command envelope) |
| GET | `/api/spaces` | Spaces inventory for the native actor |
| GET/POST | `/api/environments` | List / create environments |
| GET | `/api/environments/{id}` | Environment detail |
| POST | `/api/environments/{id}/join` | Explicit Join |
| GET | `/api/environments/{id}/members` | Members |
| GET | `/api/environments/{id}/connection` | Connection details |
| GET | `/api/environments/{id}/os` | OS/profile observation |
| GET/POST | `/api/environments/{id}/lifecycle` | Lifecycle read / Start/Stop |
| GET/POST | `/api/environments/{id}/access-keys` | Managed access keys |
| GET/POST | `/api/environments/{id}/preparation` | Preparation read / maintenance hold |
| POST | `/api/environments/{id}/preparation/acceptances` | Maintainer requirement acceptance (decision envelope) |
| POST | `/api/environments/{id}/preparation/actions` | Hold / inspect / administrator approval |
| POST | `/api/environments/{id}/identity/launch` | Start a broker-authorized identity lease |
| GET/POST | `/api/environments/{id}/terminal-sessions` | Inspect/reserve terminal sessions |
| GET/POST | `/api/environments/{id}/terminal-sessions/{terminalID}` | Inspect / End / Rename |
| GET | `/api/environments/{id}/terminal` | Native terminal WebSocket attachment |
| GET/POST | `/api/settings/runners` | Operator runner view / registration request |
| POST | `/api/settings/runners/{runner}/{action}` | Operator runner action |
| GET | `/api/settings/tailnet` | Host Tailnet settings |
| POST | `/api/settings/tailnet/host` | Host Tailnet actions |
| POST | `/api/settings/tailnet/enrollment` | Enrollment policy |
| GET | `/api/environments/{id}/tailnet` | Project Tailnet state |
| POST | `/api/environments/{id}/tailnet` | Project Tailnet actions |

Create binds an immutable profile. Join creates the project-local account and
records membership only after confirmed native success. Terminal details are in
the [managed terminal contract](terminal.md).

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
