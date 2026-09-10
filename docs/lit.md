# Lit components

The management drawer and terminal controls now use Lit for Soda-owned interactive
UI, loaded only on workspace opening/restoration. Xterm and its transport remain
imperative resources of the terminal component. Native dashboard views now mount
Spaces, Runners and repository controls directly; the old Go shells and their
auto-boot callers are removed. The native page/drawer journey shares these same
component owners.
Forgejo owns its pages, forms, permissions, authentication and native scripts;
Cockpit keeps React/PatternFly. See the handoff for actual local checks; no new
installed browser/CLI compatibility proof follows from the port.

The [Spaces implementation plan](lit-migration-plan.md) defines the drawer-first
ports, ID-keyed backend and shared page/drawer workspace (now locally implemented),
including locally completed step-5 layouts and step-6a/6b attention/candidate source
coverage. Scoped native/selected-CLI acceptance remains step 6c.
It incorporates the existing managed-tmux contract; it does not
restart the older request-owned terminal migration. Retained native adapters and
required validation remain explicit.

The [frontend improvement guide](frontend-improvement-plan.md) consolidates the
researched cleanup after locally completed step 5: mandatory canonical tokens,
enforced template diagnostics, typed view composition and clearer source ownership.
It preserves this renderer and the existing action/terminal owners. The analyzer is now integrated into required local checks through the root Bun
workspace; see [tool ownership and resolution](../tools/lit-check/README.md).

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

The compiler appends the shared presentation `?v=` epoch to these URLs and all
other relative external Soda imports. See [cache ownership](typescript.md); the
runtime remains one module identity per document, including lazy imports.

This works beneath Forgejo's `AppSubUrl` too, without a CDN, import map or a second
TypeScript resolver. Existing native and terminal imports keep their public URLs.
Core Lit and `lit/directives/repeat.js` imports share this runtime; repeat is used
for stable pane/tab chrome. Other subpath imports fail until exports and build
mapping are added together. Each addition must keep one runtime.

## Authoring

Strict TypeScript checks expressions and typed helper calls inside/around `html`
templates, but does not check their native/custom property, boolean or event binding
positions. A language-service plugin listed in `tsconfig` does not run under `tsc`.
Follow the [checker contract](frontend-improvement-plan.md#required-analyzer-integration):
`bun run typecheck` retains the product compiler and requires actual-source analysis
plus independent checker fixtures. The development-only runner binds bare analyzer
compiler imports to its workspace's classic compiler; a package alias alone is not
sufficient. Unknown-event checking is enabled and warnings also fail the gate.
Callable-event parameter compatibility remains a known analyzer gap. This local gate
is not fully TSX-equivalent checking or installed/native behavior proof.

Use readable multiline templates and typed functions for stateless sections and
wrappers. A wrapper may accept a `TemplateResult` body and specific typed callbacks;
its call site receives normal TypeScript checking while its internal bindings need
the analyzer. Extract coherent Environment/Access/status views before broader chrome.
Keep drafts, synchronous action guards, requests and resource lifetimes in their
existing concrete owners. Add a reactive element only for independent state/lifetime,
not simply to shorten a render method.

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

The workspace's private `WorkspaceMeasurement` controller owns only resize/font/
viewport subscriptions. Its `hostUpdated` hook waits for an actual rendered canvas;
connection alone is too early. Invalidation/disconnection retire subscriptions and
fence delayed callbacks, while ordinary Hide/view/Refresh preserve them. Geometry,
layout persistence and command policy stay in `SodaSpaces`. This is not a controller
base class or permission to split terminal transport/retention into separate stores.

The loopback browser smoke test compiles a test-only component through the real
build and loads the emitted runtime over HTTP. Native browser/access journeys
remain separate from this scaffold proof.

## Shared styling

Canonical token adoption is mandatory across both Spaces surfaces. Follow the
[token scope](frontend-improvement-plan.md#5-mandatory-token-consolidation): reuse
the existing palette, semantic colors, typography, spacing and control roles; define
only necessary shared density/terminal roles. Separate token availability from the
full-page body/navbar/footer selectors. A drawer must not acquire a full-page marker
just to inherit variables. Preserve the native selected theme and native form layout.

Check source for repeated raw visual values, including fallbacks and inline styling,
and verify resolved appearance in both themes and compact/wide states. Computed styles
cannot prove that source used tokens. Keep measured terminal/pane geometry distinct
from visual styling, and carry layout/focus tests when font or chrome dimensions change.

Upstream references: [Lit overview](https://lit.dev/docs/),
[reactive properties](https://lit.dev/docs/components/properties/),
[render roots](https://lit.dev/docs/components/shadow-dom/#implementing-createrenderroot),
and [lifecycle](https://lit.dev/docs/components/lifecycle/).
