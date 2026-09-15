# Spaces

**Spaces** (Sodaspaces) is the Soda workspace experience beside native Forgejo:
a repository environment entry, shared management views and terminals that stay
available while forge pages navigate.

The persistent outer document is a dedicated Soda HTML shell at `/-/soda/workspace`.
Native Forgejo pages load in a same-origin iframe. The address bar stays on that
workspace URL. Optional `?to=` names the framed Forgejo path (same origin, not
`/-/soda/`, no credential query). Child navigations update `to` with `replaceState`;
they do not use canonical Forgejo URLs as the top-level document.

Signed-in browsing uses that workspace host. The navbar Spaces link and Spaces OAuth
return go to `/-/soda/workspace`. Ordinary signed-in Forgejo documents wrap into it
with admitted `to`. Login, OAuth, install, failed `soda-connect`, and remaining
`soda-view` hosts (repository Spaces, Runners, Tailnet) stay top-level. Unsigned
`/-/soda/spaces` stays a bookmark that establishes the native actor first.

Forgejo owns navbar, profile, notifications, forms and routing inside the iframe.
The Soda shell has no second header. Layout is two surfaces: framed Forgejo on the
left, workspace on the right, with the same measured split and compact
Forge/Terminal switch as the native drawer.

There is no separate-origin Soda UI. Lit supplies Soda's management and workspace
views under the configured Forgejo origin at `/-/soda/`.

## Product surface

| Surface | Purpose |
| --- | --- |
| Spaces page | Bounded listing and navigation for environments the actor may use |
| Workspace host | Dedicated Soda HTML document at `/-/soda/workspace`; `to` names the framed Forgejo path |
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

## Remaining workspace-host decisions

Host, address bar, signed-in entry, chrome, and split layout are selected. Further
host work happens in this order. Do not fold auth into chrome.

1. **Prove that composition** in the browser and native fixtures.
2. **Auth that may leave the shell.** Login, consent, and callback stay outside the
   shell until this step selects otherwise.
3. **Prove those auth flows** in the browser and native fixtures.

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
