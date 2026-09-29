# Spaces

**Spaces** (Sodaspaces) is the persistent human workspace beside native Forgejo:
a repository environment entry, shared management views and terminals that stay
available while forge pages navigate.

Use it for development, debugging and intervention alongside factory work. Issues,
candidate pull requests, reviews and CI remain in Forgejo; admission, status and
cancellation use the [factory operator interface](../reference/factory.md).
Opening Spaces or joining a project does not authorize an agent run.

Forgejo owns one generic persistent browser host for installed extension panels.
When Soda is enabled, ordinary signed-in Forgejo navigation uses that host: native
Forgejo pages occupy the browsing area and Soda's Lit workspace stays mounted
beside them. Login, logout, enrollment/consent callbacks, installation, external
navigation and failed authentication remain top-level. The host never turns
credential query values into a return URL. Exact route and frame contracts belong
to the [Forgejo extension reference](../reference/forgejo.md) once implemented.

Forgejo owns native navigation, forms and routing in the browsing area; Soda adds
its contribution and workspace panel without a second header or nested drawer.
Keep the existing measured split and compact Forge/Terminal switch. Normal links
change the Forgejo page while preserving the same terminal element, xterm renderer,
WebSocket and shell attachment. A page or displayed repository change never
retargets the terminal. A new target requires an explicit terminal selection.

The host rechecks the signed-in account when returning from cached pages before
showing private state or resuming operations. An expired flow cannot continue as a
different actor. There is one workspace host; the former Soda-only shell and
dashboard route are removed as part of the native cutover.

## Product surface

| Surface | Purpose |
| --- | --- |
| Spaces page | Bounded listing and navigation for environments the actor may use |
| Workspace host | Forgejo's generic persistent extension host with the Soda workspace panel |
| Repository Spaces settings | Create and inspect the environment for that repository |
| Environment drawer | Management controls and managed terminals beside native forge content |
| Operator Runners settings | Local CI capacity (Soda operator only) |
| Operator Tailnet settings | Host Tailnet controls and enrollment policy |

“Move runners to the dashboard” means this bounded native-interface extension, not
a revived standalone Soda UI. Forgejo retains Actions settings, scheduling and
permissions.

## Workspace behavior

- The drawer is a non-modal aside, not an outside-click-dismissed dialog.
- Page and drawer share one multi-session workspace with flat terminal owners.
- Splits create views, never shells.
- Hide and show change presentation only; End is a separate confirmed action.
- The shell toggle hides the workspace drawer so Forgejo takes the full width
  and shows it again; the choice persists across reloads.
- Ordinary Forgejo navigation must preserve the live terminal view: same mounted
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
