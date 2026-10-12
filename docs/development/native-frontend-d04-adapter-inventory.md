# NFX-D04 host-ancestor adapter inventory (D-lane prep)

Source: Soda `frontend/spaces/*` at wave1/D-prep-migration base, Fountain
`sdk/browser/extension-contract.d.ts` at d240b821a4 (H-N source present in
`web_src/js/features/extension-mount.js`, wired by `extensions.js`).

## Removed in this prep slice (safe against current Fountain source)
- `soda-extension.ts`: duplicate scoped fetch/WebSocket construction is bypassed
  when the host supplies `MountContext.request`/`stream`; unexpected
  `contractVersion` refuses before effects. Soda domain validation
  (extension/page/panel identity, `id()` repository check, disposed guard) and
  all protocol parsing in callers are retained. Local transport remains only as
  fallback for callers not yet receiving host helpers (Tailnet entry, tests).

## Retained Soda-internal (not host adapters; keep)
- `soda-native-paths.ts` `nativeAppSubUrl`: shared native-prefix helper for
  native navigation links only; API authority stays with the mount transport.
- Workspace/Factory/Terminal internal `querySelector`/`closest('[hidden]')`
  on Soda-owned shadow/DOM subtrees, terminal cell sizing
  (`sodaspaces-workspace.ts` canvas measure), focus management.

## Host-ancestor adapters remaining (remove only at H-C/A04 + H-N lifecycle)
| File | Adapter | Replacement |
| --- | --- | --- |
| `soda-workspace-panel-entry.ts` | `root.closest('[data-extension-workspace]')`, `.extension-workspace-panel(s)` ancestor toggling, `extension-workspace-navigation` host event, footer/`getBoundingClientRect` height sizing | Host `update(presentation)` + native container (H-C) |
| `sodaspaces-page.ts` | `document.querySelector('footer')`, `#navbar` ResizeObserver, viewport height sizing | Native container sizing (H-C) |
| `soda-spaces-entry.ts` | `window.frameElement[data-extension-workspace-frame]` and `[data-workspace-direct]` panel detection returning no-op mounts | One shared owner registration (H-OWNER/D02) |
| `sodaspaces-workspace.ts` | `pageshow`/`pagehide`/`visibilitychange` + `window.location` route parsing as lifecycle/authority-adjacent signals | Host lifecycle `update/suspend/resume/retire` (C05/D05) |
| `sodaspaces-workspace-drawer.ts` | `window.location.assign(...)` success navigation | Host programmatic native navigation (H-ROUTE/D03) |
| `sodaspaces-factory.ts`, `sodaspaces-terminal.ts` | `pagehide`/`pageshow` invalidation listeners | Host lifecycle (D05) |

No adapter above is removed in this slice: each still needs its callable
H-C/H-OWNER/H-ROUTE/C05 host replacement, and removing it now would break
current page/panel behaviour.
