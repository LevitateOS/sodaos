# Spaces UX design

Design specification for the Spaces page and persistent workspace panel. Product contract:
[Spaces](../product/spaces.md). Terminal behavior: [Terminal](../reference/terminal.md).
Annotated sheets: [design/spaces](spaces/README.md).

## Direction

Spaces supports persistent human development and intervention alongside the
software factory. The first-use journey below creates a human project; agent
admission and run controls use the [operator interface](../reference/factory.md).

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

The native Forgejo header stays with Forgejo's page. Login, consent, callback and
the remaining auth/recovery flows listed in [Trust](../architecture/trust.md) use
the host's normal top-level handling. Prefer one composition per step over
dashboard clutter.

## Full-page workspace

- Project/session navigation in a sidebar.
- Local tabs within a pane; splits for simultaneous views.
- Focus is input ownership; attention is advisory from real signals.
- Measured pane minima after xterm render; do not collapse on stale geometry.
- Rename is display metadata. Hide is presentation-only. End is confirmed.
- Project Stop is a separate shared-impact action.

## Persistent panel composition

```text
┌─────────────────────────────┬──────────────────────────────┐
│ Native Forgejo browsing     │ Spaces workspace / terminals │
└─────────────────────────────┴──────────────────────────────┘
```

Fountain owns this generic host layout and retains installed extension panels
while native Forgejo pages change. Soda owns only the workspace contribution; it
does not supply a replacement shell, frame or copy of Forgejo navigation.

- Two useful surfaces, not a modal.
- Preserve native forms, routing and focus on the Forgejo side.
- The Spaces page and panel share one workspace owner layer.
- Compact layouts keep controls and terminal input usable; Hide does not End work.

## Profile and desktop extension

Terminal and Desktop identify the same project account and real files. A desktop
tab references an exact graphical session with its own display/input owner; it is
not a PTY ID in the personal-terminal API. Compact screens show one usable selected
view with accessible navigation.

## Sheets and tooling

Hand-authored SVG sheets under `docs/design/spaces/` are design source. Preview and
render TypeScript tools compile with Bun; rendered PNGs belong in ignored
`.artifacts/`. Sheets are not runtime UI or installed screenshots.
