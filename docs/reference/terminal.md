# Managed terminal contract

Soda provides managed browser terminals for project members. Soda retains access and
lifetime authority; stock packaged **tmux** retains live terminal state. This is not
a replacement for ordinary SSH and not a public terminal server.

UX composition: [Spaces UX](../design/spaces-ux.md). Routes: [HTTP API](api.md).

## Goals

- Attach to the member's existing project-local account and home.
- Opening a terminal must not create, join, start or repair a project.
- In the workspace shell, ordinary Forgejo navigation happens inside the iframe,
  so the mounted workspace component, xterm renderer, and WebSocket attachment
  survive it. On native documents the drawer re-attaches the same session after
  navigation instead.
- Browsing another repository must not retarget the terminal's project, account or
  session.

## Workspace rules

- Page and drawer share one multi-session workspace with flat terminal owners.
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

## Mounting

Canonical Spaces drawer:

```ts
import {mountSodaspaces} from '/assets/sodaspaces-drawer.js';
const controls = mountSodaspaces(mountNode, {expectedUserId, repositoryId});
controls.refresh();
```

Terminal-only:

```ts
import {mountTerminal} from '/assets/sodaspaces-terminal.js';
const terminal = mountTerminal(mountNode, {
  expectedUserId, csrfToken, repositoryId, environmentId, login,
}, {kind: 'existing', id: terminalId});
```

Resolve membership login through the protected API. Explicit Open on `new` reserves
an ID before Create; restore only inspects/attaches an existing ID.

## Boundary with desktop and automation

KDE adds graphical access to the same native account; it does not replace tmux.
Automation or AI-run processes have separate authorization and lifetime contracts.
Do not commandeer a personal shell for automation inventory.

## Source owners

- Browser: `frontend/spaces/`
- API/WS: `internal/web/api/terminal*.go`
- Privileged attach: `internal/host/terminal`
