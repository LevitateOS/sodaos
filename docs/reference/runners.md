# Local CI runner contracts

Soda owns local runner **capacity** on the appliance. Forgejo owns Actions settings,
workflows, scheduling, secrets, variables and results.

## Product boundary

- Operator-only **Runners** destination mounts through Forgejo administration
  customization and protected `/-/soda/api/` endpoints.
- Only the Forgejo user ID recorded as `operator_id` may read or mutate runner state.
- Site, organization or repository administration does not grant that appliance
  authority.
- Native Forgejo **Actions → Runners** surfaces remain Forgejo-owned.

## Runner OS

Soda selects a dedicated **Rocky headless Runner OS** for isolated jobs. It shares
Rocky-family conventions with Project OS but is a separate job artifact: it does not
reuse a Project OS image or inherit persistent account/home/SSH/tmux contracts.

Mise is required. A repository may use the same `mise.toml` developers use after an
explicit trust decision; CI has its own installation and cache locations and never
borrows developer homes, credentials or `/opt/mise`.

The intended model creates a fresh disposable job environment per job. Provider
results and explicit caches may outlive the environment; arbitrary filesystem state
does not. Ubuntu and Arch are not selected runner images.

## Local capacity model

Each runner has:

- a stable validated local ID
- a dedicated noninteractive `soda-runner-<id>` account
- private state under `/var/lib/soda/runners/`
- one job slot
- a systemd instance

The service has no sudo or Linux capabilities, a read-only host view apart from its
own state, and network access. Jobs execute repository code; only trusted
repositories and contributors belong on this capacity.

Listing reads local descriptors and systemd state. It does **not** project provider
online/offline, busy state, queue depth or disk use as facts Soda does not observe.

## Authority path

1. Authenticate a Soda session through Forgejo OAuth.
2. Require `session.User.ID == Config.OperatorID` before reading or mutating runners.
3. Apply expected-actor, origin and CSRF checks on mutations.
4. Recheck the current session immediately before helper dispatch.
5. Dispatch only fixed runner methods on the root:soda Unix socket.

Neither layer accepts a browser-selected UID, account name, state path, unit name,
executable, provider command or host flag. Only `provider=forgejo` is accepted.

## Operations

| Action | Effect |
| --- | --- |
| List / Refresh | Read local descriptors and systemd state |
| Register | Store provider UUID/token and labels; enable/start listener. Provider record must already exist. |
| Start | `systemctl enable --now` for the validated unit |
| Stop | `systemctl disable --now`; may interrupt an active local job |
| Restart | Enable then restart; leaves boot start enabled |
| Remove | Stop/disable, delete Linux account, remove local state directory. Provider registration remains. |

Report partial failures honestly. No automatic provider cleanup or rollback fabric.

## Source owners

| Responsibility | Owner |
| --- | --- |
| Native page entry | Forgejo templates, `frontend/runners/` |
| Web API | `internal/web/api/runners.go` |
| Native state | `internal/runners/`, host daemon runner routes |
| Packaging | release forgejo payload and stage scripts |

API routes: [HTTP API](api.md).
