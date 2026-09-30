# Spaces

**Spaces** (Sodaspaces) is Soda's Fountain-hosted workspace for human and factory
work: repository environments, shared management views and live AI CLI and human
terminal sessions that stay available while native Forgejo pages navigate.

It provides visibility and intervention for the
[factory lifecycle](overview.md#software-factory-workflow), alongside manual
development and debugging. Issues, candidate pull requests, reviews and CI remain
native Forgejo records. Opening Spaces or joining a project does not authorize
agent execution. The [factory reference](../reference/factory.md) describes the
existing operator command interface.

Fountain owns the generic persistent browser host and its installed page and panel
contributions. Soda supplies the Spaces page and a preferred Workspace panel; the
page and panel share one Lit workspace owner. Forgejo keeps its native navigation,
forms, sessions and route lifecycle. The panel stays mounted while native pages
change, and account, enrollment/consent, installation and external-navigation flows
remain under the host's normal handling. The host contract lives in the
[Forgejo extension reference](../reference/forgejo.md).

Normal Forgejo links preserve the same terminal element, xterm renderer, WebSocket
and shell attachment in the persistent panel. A page or displayed repository
change never retargets a terminal. A new target requires explicit terminal
selection.

The host rechecks the signed-in account when returning from cached pages before
showing private state or resuming operations. An expired flow cannot continue as a
different actor. There is one workspace host; the former Soda-only shell and
dashboard route are removed as part of the native cutover.

## Product surface

| Surface | Purpose |
| --- | --- |
| Spaces page | Find authorized repository environments and their human and factory work |
| Workspace panel | Persistent human terminals and live factory CLI activity beside native Forgejo pages |
| Repository Spaces settings | Create and inspect the environment for that repository |
| Workspace management | Environment controls, sessions, factory progress and intervention |
| Native Actions settings | Forgejo CI configuration and separately managed capacity; local Soda execution is deferred |
| Operator Tailnet settings | Host Tailnet controls and enrollment policy |

Forgejo retains Actions settings, scheduling and permissions. Existing Soda runner
observation and cleanup controls are described in the
[runner reference](../reference/runners.md); the
[feature disposition](scope.md#feature-disposition) owns future local capacity scope.

## Sessions and views

Spaces uses the [Project environment model](projects.md#environment-relationships).
A session identifies an existing execution in a particular Project, native account
and process incarnation. A pane, split or browser attachment is a view of that
session. Repository navigation never changes that binding.

| Session | Execution owner | Browser interaction |
| --- | --- | --- |
| Human terminal | Joined member under their own Project account | Authorized terminal input and explicit End; manually starting an AI CLI here does not create a factory attempt. |
| Factory coding, correction or review | Factory controller under the assigned nonhuman role account, bound to repository, issue, attempt and run | Live output for currently code-write-authorized viewers; pause, cancel and takeover through factory controls, never direct terminal input or human-terminal End. |

Both kinds use the Project execution foundation. They retain different grants and
control semantics; a shared UI does not turn a factory run into a member's shell.
The backend rechecks session visibility and each control operation independently.
Human session inventory remains member-scoped; repository permission to observe
factory work does not permit attachment to another person's terminal.

Closing the browser, hiding a pane or losing an attachment does not end execution,
return its lease or cancel an attempt. Reattachment targets the same live session;
missing or stale locators do not launch replacement shells, resume conversations
or attach to another run. Splits add views, never executions.

Live CLI visibility does not promise durable terminal recordings or transcript
replay. An ended run exposes its status, candidate, findings and check evidence;
its old session cannot be reattached as though it were still running. Human
terminal transport and live-state behavior remain owned by the
[terminal reference](../reference/terminal.md).

## Factory visibility

People can follow coding, review and correction in their real CLI sessions and
connect that activity to the issue, attempt and candidate PR. Spaces exposes the
current stage, outcome and any blocker or intervention request. Viewing a session
does not grant input or execution authority; available actions follow the
[trust boundary](../architecture/trust.md#factory-authority-boundary).

Factory progress does not depend on keeping a browser tab open. Hiding or closing
a view does not cancel work. The available intervention actions and their effects
are owned by the [operating rules](overview.md#human-intervention); their layout
belongs to subsequent interaction design.

## Workspace behavior

- The Spaces page and persistent panel share one multi-session workspace with flat
  session owners.
- Splits create views, never shells.
- Hide and show change presentation only; human terminal End is a separate
  confirmed action. Factory actions follow the session rules above.
- Hiding a terminal changes its workspace presentation. It does not end the shell.
- Ordinary Forgejo navigation preserves the live terminal view: the same mounted
  component, xterm renderer and WebSocket attachment. Browsing another repository
  must not retarget the terminal's project, account or session.

UX composition details live in [Spaces UX](../design/spaces-ux.md).
Wire contracts live in [Terminal](../reference/terminal.md) and [HTTP API](../reference/api.md).

## Settings ownership

| Setting | Location |
| --- | --- |
| Project profile / Spaces creation | Repository settings |
| Factory policy and intervention | Repository administration and work context under the [factory authority rules](../architecture/trust.md#factory-authority-boundary); detailed control placement remains to be designed |
| Local Sodarunners capacity | Deferred; existing observation/cleanup controls are documented in the [runner reference](../reference/runners.md) |
| Host Tailnet and enrollment policy | Global Soda-operator settings |

## Integration boundary

Preserve native Forgejo handlers, forms and session security. Fountain supplies
generic extension routes, contribution mounts and the persistent browser host;
Soda adds pages and a workspace panel as an administrator-installed extension.
Missing host capabilities follow the
[Fountain consumption boundary](../architecture/trust.md#fountain-consumption-boundary).

Customization rules: [Forgejo customization](../reference/forgejo.md).
