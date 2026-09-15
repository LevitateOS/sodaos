# Spaces UX design

Design specification for the Spaces page and right-half drawer. Product contract:
[Spaces](../product/spaces.md). Terminal behavior: [Terminal](../reference/terminal.md).
Annotated sheets: [design/spaces](spaces/README.md).

## Direction

A focused first-use journey, then a project/session sidebar for finding work, tabs
for switching within a pane, and contextual splits for the few terminals viewed
together. Native CLI agents remain inside real terminals. No agent-chat frontend,
worktree-per-task policy or separate application origin.

Visual language matches Soda brand tokens: Barlow interface type, Plex Mono for
controls/terminal, square geometry, thin rules, restrained red primary actions and
clear whitespace. Every repository lives on this Forgejo; there is no external
provider picker in onboarding.

## First-use journey

1. Welcome composition with brand-forward Spaces entry.
2. Repository selection from repositories the actor may own/create for.
3. Configuration including immutable Project OS profile.
4. Creation, explicit Join and first terminal.
5. Working workspace with management controls and sessions.

Keep native Forgejo header inside the framed Forgejo document. Login, consent, and
callback leave that host and become the top-level document. Prefer one composition
per step over dashboard clutter.

## Full-page workspace

- Project/session navigation in a sidebar.
- Local tabs within a pane; splits for simultaneous views.
- Focus is input ownership; attention is advisory from real signals.
- Measured pane minima after xterm render; do not collapse on stale geometry.
- Rename is display metadata. Hide is presentation-only. End is confirmed.
- Project Stop is a separate shared-impact action.

## Drawer composition

```text
┌─────────────────────────────┬──────────────────────────────┐
│ Native Forgejo navigation   │ Spaces workspace / terminals │
└─────────────────────────────┴──────────────────────────────┘
```

On native Forgejo documents this remains a right-half drawer. The persistent
Soda HTML host uses the same two surfaces with Forgejo in a same-origin iframe
on the left and the workspace on the right.

- Two useful surfaces, not a modal.
- Preserve native forms, routing and focus on the Forgejo side.
- Drawer flattening and This-page filters must not destroy the full-page layout
  model; both surfaces share one session owner layer.
- Compact Forge/Terminal switching is deliberate; Hide does not End work.

## Profile and desktop extension

Terminal and Desktop identify the same project account and real files. A desktop
tab references an exact graphical session with its own display/input owner; it is
not a PTY ID in the personal-terminal API. Compact screens show one usable selected
view with accessible navigation.

## Sheets and tooling

Hand-authored SVG sheets under `docs/design/spaces/` are design source. Preview and
render TypeScript tools compile with Bun; rendered PNGs belong in ignored
`.artifacts/`. Sheets are not runtime UI or installed screenshots.
