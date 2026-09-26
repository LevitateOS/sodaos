# Spaces

**Spaces** (Sodaspaces) is the persistent human workspace beside native Forgejo:
a repository environment entry, shared management views and terminals that stay
available while forge pages navigate.

Use it for development, debugging and intervention alongside factory work. Issues,
candidate pull requests, reviews and CI remain in Forgejo; admission, status and
cancellation use the [factory operator interface](../reference/factory.md).
Opening Spaces or joining a project does not authorize an agent run.

The persistent outer document is a dedicated Soda HTML shell. `/workspace` is the
home entry; the address bar carries the shell path itself, `/-/soda/<framed-path>`
with the frame query intact and no locator parameter. The entry normalizes to the
canonical `/-/soda/workspace` on first framed navigation. Native Forgejo pages load
in a same-origin iframe (same origin, not `/-/soda/`, no credential query). Child
navigations update the shell path with `replaceState`; they do not use canonical
Forgejo URLs as the top-level document.

Signed-in browsing uses that workspace host. The navbar Spaces link and Spaces OAuth
return go to `/workspace`. Ordinary signed-in Forgejo documents wrap into it
with an admitted frame path. Login, logout, signup/activate, password recovery,
two-factor/passkey, provider OAuth link, OAuth authorize/grant, callback, install,
failed `soda-connect`, and any `soda-view` host stay top-level. If the framed
document lands on those, the shell replaces itself with that URL. Credential query
never becomes `to`. Unsigned `/-/soda/spaces` stays a bookmark that establishes
the native actor first.

Forgejo owns navbar, profile, notifications, forms and routing inside the iframe.
The Soda shell has no second header. Layout is two surfaces: framed Forgejo on the
left, workspace on the right, with the same measured split and compact
Forge/Terminal switch as the native drawer. The right surface mounts the same
page-kind Spaces workspace as the native dashboard view: the listing is only its
empty/first-use state, and live terminals stay mounted while the iframe navigates.
The native drawer mounts the native-kind binding of that same workspace on Forgejo
documents outside the shell; framed Forgejo never mounts a nested drawer.

Direct visits to `/?soda-view=spaces` still render that native Spaces view, kept
for OAuth-failure display and older links; it is not the browsing host.

There is no separate-origin Soda UI. Lit supplies Soda's management and workspace
views under the configured Forgejo origin at `/-/soda/`.

## Product surface

| Surface | Purpose |
| --- | --- |
| Spaces page | Bounded listing and navigation for environments the actor may use |
| Workspace host | Dedicated Soda HTML document at `/workspace`; the shell path itself (`/-/soda/<framed-path>`) names the framed Forgejo path |
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

Use stock Forgejo handlers, forms, scripts and authentication, with supported
template hooks first and necessary targeted overrides second. Do not deploy an
unchanged copy of the upstream template tree. Do not introduce a Forgejo fork,
source patch set or custom Forgejo executable.

Customization rules: [Forgejo customization](../reference/forgejo.md).
