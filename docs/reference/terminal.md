# Managed terminal contract

Soda provides managed terminals in the Forgejo native workspace extension for
human project members. Soda retains access and lifetime authority; stock
packaged **tmux** retains live terminal state. This is not a replacement for
ordinary SSH and not a public terminal server.

A browser terminal is manual development access. Opening one does not admit agent
work or grant factory authority; use the [factory operator interface](factory.md).
The target [Spaces session model](../product/spaces.md#sessions-and-views) adds
factory observations with distinct authority. The member terminal interface below
does not itself provide that integration or authorize attaching as a factory role.

UX composition: [Spaces UX](../design/spaces-ux.md). Routes: [HTTP API](api.md).

## Goals

- Attach to the member's existing project-local account and home.
- Opening a terminal must not create, join, start or repair a project.
- Forgejo owns browser navigation and native page presentation. The extension
  workspace panel reattaches the same terminal session after navigation.
- Browsing another repository must not retarget the terminal's project, account or
  session.

## Workspace rules

- The Spaces page and persistent extension workspace panel share one multi-session
  workspace with flat terminal owners.
- Splits create views, never shells.
- Hide/show change presentation only; End is a separate confirmed action.
- Actor-scoped session storage holds locators and layout only: no credentials,
  names, transcripts or input.
- Cache loss never Creates or Ends work; authorized native inventory discovers
  surviving shells.
- Attention reflects actual output/unread, connection loss, another writer, ending
  or unavailable observations—not agent progress guesses.

## Persistence mechanism

Use stock Rocky-packaged tmux:

- Private `-S` socket and managed `-f` configuration; no personal default server.
- Create only through a consumed native reservation.
- Attach is exact: `tmux -N -S SOCKET attach-session -E -t =soda` with no
  create-on-missing behavior.
- Supervise the server cgroup, not only the attachment client.
- End ends supervised processes for that terminal; it does not delete files or stop
  independent SSH, services or workloads.
- No transcript logging, resurrection plugins or edits to personal shell/SSH config.

## Extension mounting

The Forgejo extension supplies the native workspace mount. It shares the
multi-session workspace owner with the Spaces page contribution; Soda does not
serve a shell document, frame, or standalone terminal page.

Resolve membership login through the private native extension API. Explicit Open on `new` reserves
an ID before Create; restore only inspects/attaches an existing ID.

## Installed support readiness

Project runtime state and terminal support readiness are separate observations.
An already-running Project can need terminal maintenance. Project settings show
the support state returned with environment detail and Spaces inventory:

| State | Meaning |
| --- | --- |
| `ready` | The admitted Project helper bytes match the host helper exactly. |
| `support_mismatch` | An appliance administrator must update this Project's terminal support. |
| `inspection_unavailable` | Installed support could not be confirmed; retry or request operator inspection. Stopped roots currently report this state without being started. |
| `maintenance_in_progress` | Maintenance exclusion or a retained interrupted-operation record blocks managed admission. |

The host observes the actual root-owned executable with bounded native file
inspection; image labels and helper version strings do not establish readiness.
It checks the helper's parent directories, ownership, mode, size and SHA-256.
Expected and observed digests stay in native operator diagnostics. The terminal
request rechecks support before invoking the Project helper, and the helper's
existing exact-match verification remains in place.

Explicit [same-root maintenance](project-os.md#same-root-maintenance) uses shared
cross-process admission exclusion with terminal reservation/creation, Project
creation and lifecycle mutations, preparation, factory launch and identity starts.
Terminal admission holds exclusion until the native operation reports metadata
or readiness. A disconnected browser does not prove that managed work is idle;
maintenance inspects retained native work before replacement. Recovery records
keep admission closed after an interrupted maintenance process.

Support failures never recommend signing in. Only an HTTP 401 requests the normal
Fountain login flow; a changed native session generation requests a page refresh,
and missing membership or repository permission requests access. Terminal request
failures and native support-error frames retain the original locator and screen,
freeze input, and do not replay Create. Unknown mutation outcomes require
inspection of that exact locator. The bounded wire codes are documented in the
[HTTP API](api.md#terminal-readiness-and-errors).

## Boundary with desktop and automation

Graphical access is [deferred](../product/projects.md#selected-profiles). Any future
desktop uses the same native account and does not replace tmux.
Automation or AI-run processes have separate authorization and lifetime contracts.
Do not commandeer a personal shell for automation inventory.

## Source owners

- Native extension UI: `frontend/spaces/`
- Private API/WS: `internal/web/api/extension_terminal.go`
- Privileged attach and support observation: `lib/host/src/terminal/`
- Private host client/wire: `internal/host/terminal*.go`

Project execution admission follows the current repository-write policy in
[Projects](../product/projects.md#explicit-joining).
