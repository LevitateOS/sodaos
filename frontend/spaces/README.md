# Shared Spaces browser source

This is authored TypeScript and CSS for the Soda Forgejo extension package, not a
Forgejo public-assets directory. Fountain owns browser identity, navigation and
the lifetime of the native page and persistent workspace-panel contributions.

- `sodaspaces-workspace.ts` owns the shared page and panel workspace, locators,
  layout, original-target commands, stable terminal hosts and navigation checks.
- `sodaspaces-project.ts` owns project requests, synchronous admission and key
  drafts.
- `sodaspaces-terminal.ts` owns the original account binding, issued locator,
  xterm renderer and attachment lifetime.
- `*-view.ts` modules provide stateless typed presentation. They do not own
  permission, request or resource authority.
- `sodaspaces-layout.ts` and `sodaspaces-api.ts` define pure layout and bounded
  JSON contracts.
- `sodaspaces-attention.ts` derives bounded lifecycle observations; mounted
  refreshes never create or renew terminals.
- `sodaspaces-page.ts` and `soda-workspace-panel-entry.ts` adapt the shared owner
  to the native page and persistent panel.

`appliance/soda-extension/extension.json` declares the browser entries.
`scripts/build-soda-extension.ts` checks that declaration against the entries and
their styles, bundles the extension modules, and adds the locked terminal assets
and required notices to the independently installed package. The Forgejo image
payload in `internal/release/build/forgejo-payload.json` owns host branding and
template assets; it does not publish a duplicate copy of the extension UI.

The preview builds and serves the same extension asset package. Generated browser
and terminal assets stay under ignored `.artifacts/`. The Forgejo branding modules
keep their own public Lit bundle, separate from the extension package runtime.

Rendering never creates a terminal, changes native lifetime, provisions access or
owns native forms. Controls retain the original target and bootstrap binding;
server operations authorize access. The [terminal guide](../../docs/reference/terminal.md)
owns native lifetime and exact lookup. Flat terminal hosts are not moved between
keyed template parents, and view updates retain unsent drafts and renderer/socket
identity.
