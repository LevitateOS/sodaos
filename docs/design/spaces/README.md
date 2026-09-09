# Spaces visual design sheets

The current design is [Parallel work, one terminal workspace](../../spaces-design.md),
with the complementary [right-half drawer design](../../spaces-drawer-design.md).
It replaces the earlier grid-first interactive mockup after research into cmux,
Superset, Agent Deck, Wave, Zellij and Conductor. These are **static annotated drawings**,
not live UI, another app to connect to, or installed-product screenshots.

## Design sheets

| Source | Design case |
| --- | --- |
| [`focus.svg`](focus.svg) | Default desktop: project/session navigation, local tabs, one large terminal and direct split controls |
| [`split.svg`](split.svg) | Wide desktop: two independent projects, resizable sibling panes, input focus versus agent attention |
| [`mobile-states.svg`](mobile-states.svg) | Two 390×844 mobile frames (terminal and full-height switcher), plus disconnected/uncertain/kept examples |
| [`drawer-work.svg`](drawer-work.svg) | 50/50 native-forge browsing/comment editing and a tabbed terminal drawer; independent repositories and keyboard focus |
| [`drawer-sessions.svg`](drawer-sessions.svg) | Native code browsing remains usable while a searchable project/session switcher occupies only the drawer content |
| [`drawer-compact.svg`](drawer-compact.svg) | Compact Forge/Terminal switching, compared with native navigation, full-page Spaces and explicit Hide |

Native pages are schematic context, not newly authored replacements for Forgejo's
UI or captures of its actual handlers. In particular, keep the merged native compact
repository header/action disclosure rather than reproducing schematic native chrome.
The [Lit implementation plan](../../lit-migration-plan.md) owns the real shared
page/drawer sequence; these sheets are design inputs, not a component prototype. All session names, output and state are fictional. “Working” / “Waiting” require a
real explicit signal integration before the product may show them. The sheets do not
supply that integration, real xterm rendering, authentication, multiple native sessions,
mobile keyboard behavior or retention/process proof. Refer to the specification for
behavior, authority and first-slice boundaries, not to placeholder transcript wording.

`sheets.css` references the canonical Soda palette, unmodified symbol and existing
Barlow/Plex fonts. No upstream product code/artwork, new dependency or branding
regeneration is included. SVG sources are hand-authored design assets, not generated
product payloads; rendered PNGs belong in ignored `.artifacts/`.

## Render without a server or tunnel

With the root-pinned Bun and already prepared root dependencies:

```sh
bun node_modules/typescript/bin/tsc -p docs/design/spaces/tsconfig.tools.json
bun docs/design/spaces/render-sheets.ts
```

The renderer opens a fresh sandboxed Chromium context and fulfills every page request
from a fixed local asset map. The reserved `.invalid` origin is only an interception
key, never a contacted website. It opens **no HTTP listener or port**, reads no existing
browser profile/credentials and does not contact Forgejo, Soda APIs or a project.
It closes the browser/context on completion or failure. No dependency installation,
application build, deployment or service change is involved.

Each run writes the sheets as PNGs and `render.json` to a fresh
`.artifacts/spaces-redesign/TIMESTAMP/`. The record contains source hashes, actual
revision/dirty status, browser version, dimensions and measured mono character width.
The renderer checks font loading, SVG parsing, whole-sheet text bounds and unexpected
requests/page errors; visually inspect the result for overlap and design quality.
These are drawing checks, **not UI interaction tests or product acceptance**.

The SVG/CSS asset references are fulfilled by the renderer; opening a naked SVG
in an arbitrary file viewer need not load those assets. Use its PNG outputs for review.
No new browsing endpoint is required. Real Spaces belongs at `/-/soda/spaces` on the
existing application origin (33443 in the isolated deployment), after implementation.

## Superseded mockup

`index.html`, `preview.css`, `preview.ts`, `review.ts` and their older compiler/capture
records belong to the **superseded `8fe3468` mockup**, not this redesign or a product
implementation candidate. Do not use its grid selector, fake state machine or
`--serve`/33450 path as the current design/review workflow. Source and historical
captures are retained so prior observations are not erased or relabelled.
