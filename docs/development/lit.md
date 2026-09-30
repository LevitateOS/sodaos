# Lit components

Soda uses Lit for the Spaces page, persistent Workspace panel and project and
terminal controls. Fountain owns the native page and panel host. Soda supplies
browser entry modules through its standalone extension package; the host provides
the mount context and keeps the Workspace panel mounted while native Forgejo pages
navigate. Xterm and its socket remain resources of the terminal component.
Forgejo owns its own pages, forms, authentication and native scripts. See the
[Forgejo integration contract](../reference/forgejo.md) for that boundary.

## Runtime and builds

The root `package.json` declares Lit and the single `bun.lock` pins its resolved
dependencies. Use the repository's Bun commands:

```sh
bun install --frozen-lockfile
bun run typecheck
bun run build:forgejo
bun run test:lit
```

`appliance/soda-extension/extension.json` declares the Spaces, Runners and Tailnet
page entries and Workspace panel entry. `scripts/build-soda-extension.ts` bundles
their module graphs, builds their CSS and packages the locked xterm assets and
notices. Extension components load those package assets through the mount context's
asset base. Add an extension module through its owning entry graph; do not stage a
second copy in `internal/release/build/forgejo-payload.json`.

Forgejo branding retains the shared `assets/branding/forgejo/lit.ts` runtime at
`public/assets/soda/forgejo/lit.js` with its license. Forgejo templates do not load
it eagerly. `scripts/build-forgejo.ts` maps supported Lit imports in Forgejo-side
modules to that single runtime and appends the presentation cache epoch. The
extension build bundles its own Lit graph and does not depend on Forgejo's public
Lit URL. See [TypeScript build ownership](typescript.md#porting-source).

## Authoring

Use standard `import {LitElement, html} from 'lit'` statements. Strict TypeScript
checks expressions around Lit templates; the [template analyzer](typescript.md#compiler-boundaries)
checks property, boolean and event binding positions in actual source. `bun run
typecheck` runs both checks and independent analyzer fixtures. Callable-event
parameter compatibility remains a known analyzer gap. These source checks do not
prove native installed behavior.

Use readable templates and typed helpers for coherent stateless sections. Add a
reactive element when it owns independent state or a resource lifetime. Static
reactive-property declarations work with the repository's decorator settings.
Declare fields and initialize them in the constructor so native class fields do
not shadow Lit's reactive accessors:

```ts
import {LitElement, html} from 'lit';

class SodaExample extends LitElement {
  static properties = {label: {type: String}};
  declare label: string;

  constructor() {
    super();
    this.label = '';
  }

  protected render() {
    return html`<p>${this.label}</p>`;
  }
}

customElements.define('soda-example', SodaExample);
```

Lit defaults to shadow DOM, where global Soda and xterm styles and document
selectors do not reach. A component using `createRenderRoot() { return this; }`
renders into light DOM and shares the package styles. Neither rendering choice
permits replacing native Forgejo forms or HTMX-owned fragments.

Rendering must not create or join an environment, replay mutations or reopen a
terminal. Keep drafts, action guards, requests and resource lifetimes in their
concrete owners. Preserve terminal DOM and socket identity across reactive updates
and panel view changes; component disposal detaches the terminal without issuing
native End. Clean up external listeners and observers through component lifecycle
callbacks.

A keyed list preserves identity within its render part, not across different
pane parents. Keep the terminal layer stable when panes move or maximize. A
parent's `updateComplete` does not wait for all children or layout: await the
relevant child, recheck disposal and use ResizeObserver for terminal geometry.
The private `WorkspaceMeasurement` controller owns resize, font and viewport
subscriptions; layout persistence and command policy stay in `SodaSpaces`.

## Shared styling

Use the existing palette, semantic colors, typography, spacing and control roles
for both Spaces page and Workspace panel. Keep page layout selectors scoped to
the page and panel layout selectors scoped to the panel. Preserve the native
Forgejo theme and form layout. Check resolved appearance in light and dark themes
and compact and wide views when changing shared tokens, fonts or geometry.

Upstream references: [Lit overview](https://lit.dev/docs/),
[reactive properties](https://lit.dev/docs/components/properties/),
[render roots](https://lit.dev/docs/components/shadow-dom/#implementing-createrenderroot),
and [lifecycle](https://lit.dev/docs/components/lifecycle/).
