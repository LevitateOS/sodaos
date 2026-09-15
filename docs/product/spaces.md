# Spaces

**Spaces** (Sodaspaces) is the Soda workspace experience inside Forgejo's native
frontend: a repository environment entry, shared management views and a drawer that
keeps forge browsing and project work available together.

There is no standalone Soda frontend. Forgejo supplies header, profile menu and
authentication. Lit supplies Soda's management and workspace views under the
configured Forgejo origin at `/-/soda/`.

## Product surface

| Surface | Purpose |
| --- | --- |
| Spaces page | Bounded listing and navigation for environments the actor may use |
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
- Ordinary Forgejo navigation must preserve the live terminal view: same mounted
  component, xterm renderer and WebSocket attachment. Browsing another repository
  must not retarget the terminal's project, account or session.

UX composition details live in [Spaces UX](../design/spaces-ux.md).
Wire contracts live in [Terminal](../reference/terminal.md) and [HTTP API](../reference/api.md).

## Settings ownership

| Setting | Location |
| --- | --- |
| Project profile / Spaces creation | Repository settings |
| CLI-based AI automation (when offered) | Repository settings |
| Local Sodarunners capacity | Global Soda-operator settings |
| Host Tailnet and enrollment policy | Global Soda-operator settings |

## Integration boundary

Use stock Forgejo handlers, forms, scripts and authentication, with supported
template hooks first and necessary targeted overrides second. Do not deploy an
unchanged copy of the upstream template tree. Do not introduce a Forgejo fork,
source patch set or custom Forgejo executable.

Customization rules: [Forgejo customization](../reference/forgejo.md).
