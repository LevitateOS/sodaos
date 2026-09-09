# Lit components

The management drawer and terminal controls now use Lit for Soda-owned interactive
UI, loaded only on workspace opening/restoration. Xterm and its transport remain
imperative resources of the terminal component.
Forgejo owns its pages, forms, permissions, authentication and native scripts;
Cockpit keeps React/PatternFly. See the handoff for actual local checks; no new
installed browser/CLI compatibility proof follows from the port.

The [Spaces implementation plan](lit-migration-plan.md) defines the drawer-first
ports, ID-keyed backend and shared page/drawer workspace (now locally implemented),
followed by layouts and attention. It incorporates the existing managed-tmux contract; it does not
restart the older request-owned terminal migration. Retained native adapters and
required validation remain explicit.

## Runtime and builds

The root `package.json` declares Lit; the single `bun.lock` records its resolved
dependencies. Use the pinned Bun and existing commands:

```sh
bun install --frozen-lockfile
bun run build:forgejo
bun run build:preview
bun run typecheck
bun run test:forgejo
bun run test:lit # emitted runtime, project/workspace and terminal contracts in Chromium
```

`assets/branding/forgejo/lit.ts` exports the upstream core API. The build bundles
it once into `.artifacts/forgejo-js/lit.js`; the production payload and preview
stage it at `public/assets/soda/forgejo/lit.js` alongside its BSD license. No
template loads it eagerly. A component importing Lit loads the runtime on demand.

Write standard `import {LitElement, html} from 'lit'` statements. The production
build uses each module's payload destination to emit a relative runtime URL:

| Component destination | Generated runtime import |
| --- | --- |
| `public/assets/soda/forgejo/example.js` | `./lit.js` |
| `public/assets/example.js` | `./soda/forgejo/lit.js` |

This works beneath Forgejo's `AppSubUrl` too, without a CDN, import map or a second
TypeScript resolver. Existing native and terminal imports keep their public URLs.
Core Lit imports are supported initially. Subpath imports such as
`lit/directives/repeat.js` fail clearly until their exports are added to the shared
runtime and build mapping together; each future addition must keep one runtime.

## Authoring

Use `.ts` and the existing strict browser configuration. Static reactive-property
declarations avoid changing the repository's decorator configuration. Declare
fields and initialize them in the constructor so native class fields do not
shadow Lit's reactive accessors:

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

New production modules still require explicit entries in
`internal/nativebuild/forgejo-payload.json`. Register real components only from
their page's existing supported hook/entrypoint. The native adapter dynamically
imports the drawer on demand and checks departure/Hide before mounting it. The smoke component is test-only.

Choose the rendering boundary per component. Lit defaults to shadow DOM, where
global Soda/Forgejo/xterm styles and document selectors do not reach. Rendering
into light DOM with `createRenderRoot() { return this; }` reuses existing styles
but gives up style isolation. Neither choice permits rendering over native forms
or HTMX-owned fragments.

Keep state changes separate from effectful commands. Rendering must not create or
join an environment, replay mutations or reopen terminals. Preserve terminal DOM
and socket lifetimes across reactive updates, view changes and Hide. Clean up
external listeners and observers through component lifecycle callbacks. Use the
current terminal facade, including exact `restore`, finite `retain`, deliberate
`returnToWork` and HTTP End. Document/element disposal detaches, not native End.

A keyed list preserves identity within its own render part, not across different
pane parents. The workspace plan selects a stable terminal layer; moving/maximizing
views must not disconnect/recreate terminal elements. Add `repeat` through the
shared runtime/export mapping when it is actually used, not an independent bundle.
A parent's `updateComplete` does not wait for all children or layout: await the
relevant child, recheck retirement and use ResizeObserver for terminal geometry.

The loopback browser smoke test compiles a test-only component through the real
build and loads the emitted runtime over HTTP. Native browser/access journeys
remain separate from this scaffold proof.

Upstream references: [Lit overview](https://lit.dev/docs/),
[reactive properties](https://lit.dev/docs/components/properties/),
[render roots](https://lit.dev/docs/components/shadow-dom/#implementing-createrenderroot),
and [lifecycle](https://lit.dev/docs/components/lifecycle/).
