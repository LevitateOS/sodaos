# Managed terminal contract

Soda provides managed terminals in the Forgejo native workspace extension for
human project members. Soda retains access and lifetime authority; stock
packaged **tmux** retains live terminal state. This is not a replacement for
ordinary SSH and not a public terminal server.

A browser terminal is manual development access. Opening one does not admit agent
work or grant factory authority; use the [factory operator interface](factory.md).

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

## Boundary with desktop and automation

KDE adds graphical access to the same native account; it does not replace tmux.
Automation or AI-run processes have separate authorization and lifetime contracts.
Do not commandeer a personal shell for automation inventory.

## Source owners

- Native extension UI: `frontend/spaces/`
- Private API/WS: `internal/web/api/extension_terminal.go`
- Privileged attach: `internal/host/terminal`

Project execution admission follows the current repository-write policy in
[Projects](../product/projects.md#explicit-joining).
