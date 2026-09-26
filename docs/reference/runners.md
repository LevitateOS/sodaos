# Local CI runner contracts

Local CI execution is unavailable until isolated jobs are supported. Registration,
Start and Restart fail without creating accounts, saving credentials or dispatching
execution. The former shared-account host runner and its service are not shipped.

Forgejo owns Actions settings, workflows, scheduling, secrets, variables and results.
Factory candidate verification observes Forgejo Actions against the exact candidate
commit; use separately managed provider capacity. CI jobs remain separate from
implementation and review agents; see the [factory interface](factory.md).

## Product boundary

- The operator-only **Runners** destination mounts through Forgejo administration
  customization and protected `/-/soda/api/` endpoints.
- Only the stable Forgejo user ID recorded as `operator_id` may inspect or clean up
  local runner state. Site, organization and repository administration do not grant
  appliance authority.
- Native Forgejo **Actions → Runners** surfaces remain Forgejo-owned.

## Existing local state

Listing reads local descriptors and systemd state. It does not infer provider
online/offline status, busy state, queue depth or usable execution capacity.
Existing experimental listeners must be stopped or removed through these retained
operator controls before manual existing-state maintenance. No update or cleanup
is implied by changing source files.

| Action | Effect |
| --- | --- |
| List / Refresh | Read local descriptors and systemd state |
| Register / Start / Restart | Unavailable; no execution side effects |
| Stop | Disable and stop the validated unit; may interrupt an active job |
| Remove | Stop/disable, delete its Linux account and local state. Provider registration remains. |

Report partial cleanup failures honestly. No automatic provider cleanup or rollback.
Neither browser nor helper accepts a selected UID, account, state path, unit name,
executable or command. Fixed runner methods use the root:soda Unix socket. Browser
mutations require expected actor, origin and CSRF checks and a current session
immediately before cleanup dispatch.

## Deferred execution boundary

Future local execution requires a separate Rocky headless Runner OS with mise,
fresh disposable job environments and no listener credentials, runtime socket or
persistent developer homes. This artifact and its native isolation qualification
are not currently implemented. Do not replace that boundary with host execution.

## Source owners

| Responsibility | Owner |
| --- | --- |
| Native page entry | Forgejo templates, `frontend/runners/` |
| Web API | `internal/web/api/runners.go` |
| Local observation and cleanup | `internal/runners/`, host daemon runner routes |
| Packaging | Release payload and stage scripts |

API routes: [HTTP API](api.md).
