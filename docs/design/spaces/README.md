# Spaces design preview

An interactive **design mockup** for [the Spaces proposal](../../spaces-design.md),
not a second product frontend, live terminal, Forgejo page or deployment fixture.
It is deliberately outside the native payload/build entrypoints. Every screen is
labeled; sample output must not be published as installed-product evidence.

## Run locally

From the repository root, with the pinned Bun 1.4.2 and prepared root dependencies:

```sh
bun node_modules/typescript/bin/tsc -p docs/design/spaces/tsconfig.json
bun node_modules/typescript/bin/tsc -p docs/design/spaces/tsconfig.tools.json
bun docs/design/spaces/review.ts --serve
```

Open `http://127.0.0.1:33450/` on that machine. This listener is loopback-only;
remote clients need their own deliberate forward. Ctrl-C stops it. The command
emits browser JavaScript into a fresh ignored `.artifacts/spaces-design/TIMESTAMP/`.
No dependency installation, service activation or appliance contact is performed.
It refuses an occupied port rather than adopting another server.

Use the preview toolbar to choose **Working** or **Away & errors**, and dark/light
themes. `?scene=away&theme=light` selects the latter scene directly. A new document
resets to fictional sample state; it does not restore processes or browser storage.

## What to review

- Six labeled sample sessions in two projects; separate project and session states.
- Mixed-project tab groups, Single/Side-by-side/Stacked/Four-pane layouts, maximize,
  moving via the action menu and keyboard tab selection. Layout changes retain sample
  IDs; they never imply real native process preservation.
- Explicit project selection for a new **sample** tab, rename, Hide with retained
  discovery, finite-retention messages, End confirmation/cancel and sibling isolation.
- Kept, offline, attached-elsewhere, ended and cleanup-unconfirmed sample states.
  The offline reconnect illustration keeps a finite deadline until deliberate Continue.
- Project details are a non-modal inspector. Shared-impact Stop is visibly disabled
  and explicitly not simulated; there is no hidden operational command.
- Compact screens show one pane and a full-height project/session switcher. Other
  groups remain selectable; no tiny four-terminal phone grid.
- Canonical Soda symbol, Barlow/Plex fonts and shared palette are loaded directly from
  existing assets. No branding regeneration, component library or new dependency.

The original proposed default remains **single pane**; this review opens two panes
to demonstrate cross-project use. Sidebar/divider drag resizing, drag-and-drop tabs,
bulk actions, creation/ending transition timing, native header/authentication, xterm
input/history/editor behavior and durable navigation/reattachment are **not** supplied
by this prototype. The production implementation must use actual session ownership
and renderer lifetimes, not this sample DOM rerendering model.

## Design-only browser review

```sh
bun docs/design/spaces/review.ts --check
```

This emits the preview, starts a temporary loopback server, opens a fresh sandboxed
Chromium context, runs sample interaction/overflow checks and captures seven views:
desktop dark/light, retained/offline, cleanup uncertainty, mobile terminal, mobile
switcher and narrow mobile retention. It closes its browser/server afterward.
Generated PNGs and `review.json` stay in a fresh ignored directory. The record contains
actual viewport/theme, browser version, source revision/dirty flag and authored-file
hashes. Review images visually; passing assertions are not product acceptance.

No credentials/profiles from deployed fixtures are read. The server has a fixed
asset map; all other routes/methods return404. Browser connections outside the preview
are refused, and the page's CSP disallows fetch/WebSocket connections. No backend
API, provider login, project mutation, package install or VM action is available.
Production Forgejo/Soda/Cockpit and the pending Rocky rollout are unchanged.
