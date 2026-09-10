# Shared Spaces browser source

This is authored TypeScript/CSS, not a public deployment directory. Go owns the
bounded HTML/OAuth/API response; Forgejo keeps its own forms, auth and navigation.

- `sodaspaces-workspace.ts`: the **one page/drawer owner**, locators/layout,
  original-target commands, stable flat terminal hosts and navigation observations.
- `sodaspaces-project.ts`: project requests, synchronous admission and key drafts.
- `sodaspaces-terminal.ts`: original account binding, xterm/socket/retention lifetime.
- `*-view.ts`: stateless typed presentation. Readonly render projections are created
  from the owners, never synchronized stores or permission/resource authority.
- `sodaspaces-layout.ts` / `sodaspaces-api.ts`: pure layout and bounded JSON contracts.
- `sodaspaces-attention.ts`: bounded typed observations and pure lifecycle reasons;
  workspace slots own one unread bit, not transcripts, counts or agent semantics.
  Its mounted-only clock/visible GET refresh never creates or renews terminals.
- `sodaspaces.ts` / `sodaspaces-page.ts`: native/page adapters, not another controller.
- `sodaspaces-workspace.css`: shared workspace chrome plus explicitly scoped page
  shell rules. Project, terminal and native-adapter CSS have separate owners.

`tsconfig.browser.json` and the required analyzer discover this directory. The
browser build enforces exact source/payload inventory and rewrites source-relative
module imports to canonical public destinations (also for emitted test fixtures).
`internal/nativebuild/forgejo-payload.json` remains the staging/preview authority.
Published URLs intentionally stay compatible: the workspace owner is still served
as `sodaspaces-drawer.js`, workspace CSS as `sodaspaces-page.css`, project CSS as
`sodaspaces-drawer.css`. These are URLs, not duplicate source owners. The sole Lit
runtime remains in canonical branding; unsupported directives/tool imports fail.
Generated JavaScript stays under ignored `.artifacts/forgejo-js/`. Root
`build:forgejo` first prepares the locked renderer/CSS/licenses under
`.artifacts/browser-terminal/vendor/`; all browser checks call that build and must
work without a previous development checkout's cache. Native staging and isolated
preview still prepare their own destinations using the same locked fetcher.

Rendering never creates a terminal, Returns, changes retention, provisions access
or owns native forms. A control's callback still rechecks the concrete owner's
original target and lifetime. The xterm screen is unconditional; flat terminal
hosts are never moved between keyed template parents. View updates must retain
unsent drafts and renderer/socket identity.
