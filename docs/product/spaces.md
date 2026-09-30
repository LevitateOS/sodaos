# Spaces

**Spaces** (Sodaspaces) is Soda's native Forgejo extension for human project work:
a repository environment entry, shared management views and terminals that stay
available while Forgejo pages navigate.

Use it for development, debugging and intervention alongside factory work. Issues,
candidate pull requests, reviews and CI remain in Forgejo; admission, status and
cancellation use the [factory operator interface](../reference/factory.md).
Opening Spaces or joining a project does not authorize an agent run.

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
| Spaces page | Bounded listing and navigation for environments the actor may use |
| Workspace panel | Persistent terminal and project workspace beside native Forgejo pages |
| Repository Spaces settings | Create and inspect the environment for that repository |
| Workspace management | Project controls and managed terminals |
| Operator Runners settings | Local CI capacity (Soda operator only) |
| Operator Tailnet settings | Host Tailnet controls and enrollment policy |

“Move runners to the dashboard” means this bounded native-interface extension, not
a revived standalone Soda UI. Forgejo retains Actions settings, scheduling and
permissions.

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
| Factory admission, execution policy and run controls | Protected `soda-factory` operator configuration and commands |
| Local Sodarunners capacity | Global Soda-operator settings |
| Host Tailnet and enrollment policy | Global Soda-operator settings |

## Integration boundary

Preserve native Forgejo handlers, forms and session security. The maintained fork
supplies generic extension routes, contribution mounts and the persistent browser
host; Soda adds pages and a workspace panel as an administrator-installed extension.
Keep the fork's upstream patch set small and covered by the session, authorization,
native-route and package checks in the
[Forgejo extension implementation plan](../development/forgejo-extensions-plan.md).

Customization rules: [Forgejo customization](../reference/forgejo.md).
