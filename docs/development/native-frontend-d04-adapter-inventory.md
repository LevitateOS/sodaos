# NFX-D04 host-layout adapter inventory

The public boundary belongs to the [Forgejo integration contract](../reference/forgejo.md#native-extension-presentation).
This inventory covers the remaining D04 source adapters from Soda `72284a49`,
using Fountain's callable C03/A04 providers at `042b3bc3` and the pinned
`fountain-browser-contract.d.ts` mirror.

| Removed adapter | Current owner and input |
| --- | --- |
| `sodaspaces-page.ts`: navbar/footer queries, ResizeObserver subscriptions, viewport-minus-chrome calculation and inline mount height | Fountain supplies the native container; the page forwards public presentation/lifecycle updates to its retained Workspace |
| `sodaspaces-workspace.ts`: host-ancestor `[hidden]`/`[inert]` inspection and document visibility polling gate | The host supplies visible/concealed state through `ContributionPresentation.visible` and suspend/resume; internal callers use that state and the owner's connection |
| `sodaspaces-workspace.css`: native page/container gutter overrides and legacy drawer Hide-control reservation | Fountain owns its page gutters and contribution placement; Soda styles only its mount and content |

Workspace measurement still observes the Soda element and its own canvas,
measures terminal cells, and applies local pane geometry. Local terminal/Project
view hiding, menu bounds, focus management and product guards remain consumer
responsibilities. Public presentation updates and concealment retain the same
owner and browser resources; retirement releases them through the existing
lifecycle. No replacement host-height or host-DOM adapter is added.

Entry/route/transport migration and final native composition are separate joins.
D04 source removal does not by itself qualify every native container or installed
terminal/process preservation; those remain the applicable F06/I04–I08 checks.
