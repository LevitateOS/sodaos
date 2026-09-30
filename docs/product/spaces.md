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
| Operator Runners settings | Local CI capacity (Soda operator only) |
| Operator Tailnet settings | Host Tailnet controls and enrollment policy |

“Move runners to the dashboard” means this bounded native-interface extension, not
a revived standalone Soda UI. Forgejo retains Actions settings, scheduling and
permissions.

## Factory visibility

People can follow coding, review and correction in their real CLI sessions and
connect that activity to the issue, attempt and candidate PR. Spaces exposes the
current stage, outcome and any blocker or intervention request. Viewing a session
does not grant input or execution authority; available actions follow the
[trust boundary](../architecture/trust.md#factory-authority-boundary).

Factory progress does not depend on keeping a browser tab open. Hiding or closing
a view does not cancel work. The detailed controls and their layout belong to the
subsequent interaction design, within the product lifecycle above.

## Workspace behavior

- The Spaces page and persistent panel share one multi-session workspace with flat
  terminal owners.
- Splits create views, never shells.
- Hide and show change presentation only; End is a separate confirmed action.
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
| Factory policy and intervention | Repository-scoped factory authority; configuration and control placement remain to be designed |
| Local Sodarunners capacity | Global Soda-operator settings |
| Host Tailnet and enrollment policy | Global Soda-operator settings |

## Integration boundary

Preserve native Forgejo handlers, forms and session security. Fountain supplies
generic extension routes, contribution mounts and the persistent browser host;
Soda adds pages and a workspace panel as an administrator-installed extension.
Missing host capabilities follow the
[Fountain consumption boundary](../architecture/trust.md#fountain-consumption-boundary).

Customization rules: [Forgejo customization](../reference/forgejo.md).
