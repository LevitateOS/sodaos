# Drawer terminal integration contract

## Workspace layout slice — source, not deployed

The native `custom/header` and `custom/footer` hooks load one `sodaspaces.js`
shell mounting `mountSodaspaces` once on explicit opening. The shell now uses a
**non-modal aside**, not `showModal`, a backdrop or outside-click dismissal.
Desktop starts at half width; a pointer/keyboard separator adjusts it between
35–65 percent. Native body/container widths adapt, without replacing navigation,
forms, notifications or beforeunload. Below 800px the workspace fills the viewport;
Hide returns to the native page. Real native route/pane-width compatibility still
needs validation; the synthetic layout fixture is not that proof.

The content has Terminal, Environment and Access view tabs with roving keyboard
focus. Management and SSH forms are no longer above the terminal. The terminal
fills its available panel; its renderer observes resize and tab/hide restoration.
There is still **one terminal**, not multiple session tabs or an implemented `+`.
Those require the next protected session-lifetime change.

**Focus/visibility changes and Hide/reopen retain the same component and socket.**
They do not fetch again, provision, restart or replay anything. Pending operations
remain guarded in that same component; hiding is not cancellation. End terminal is
a separate action. No outside click dismisses the pane and no focus trap blocks the
native page. Escape inside xterm remains shell input; Ctrl+Shift+Enter focuses End
terminal. The complete component retains the existing action-time session/provider
checks and uncertain-mutation guard.

**Important remaining limits:** pagehide/BFCache still retire this page's component.
Reload/navigation, lost transport, authorization loss and the backend's earlier-of-
session-expiry/two-hour limit still end the terminal. Refresh still disposes the
existing renderer and cannot remount a used terminal. Hiding currently retains a
live socket, not a server-detached session; the proposed 30-minute retention deadline
is **not implemented**. Do not deploy this slice as completed session continuity.

## Complete content boundary

```js
import {mountSodaspaces} from '/assets/sodaspaces-drawer.js';
const controls = mountSodaspaces(mountNode, {expectedUserId, repositoryId});
controls.refresh(); // explicit initial read; mounting alone is inert
```

IDs are canonical native-page string hints, not authentication. An anonymous page
can omit `expectedUserId` for explicit contextual OAuth, never to borrow a Soda
identity. The module owns only its mount: authentication, Create/Join, current saved
public-key lifecycle, Start/Stop, own SSH details/native Copy and terminal controls.
The shell owns visibility/width; native Forgejo owns routing and all its workflows.

Call `controls.invalidate()` on confirmed stale context, and `controls.dispose()`
before real removal/context replacement. **Do not dispose merely to hide the drawer
or switch a view.** Disposal aborts reads and clears content, not dispatched native
mutations. A malformed/uncertain mutation result blocks further mutation replay but
permits safe reads and explicit Soda logout. No automatic repair/recreation.

JSON reads are MIME-checked, streaming-bounded to 64 KiB, same-origin/no-store and
redirect-rejecting. Actions freshly check Soda/session/provider identity; the server
remains the authorization boundary. No browser-selected Linux identity or privileges.
Native-only logout in another tab is not atomic Soda/SSH logout; the interface shows
its actual Soda actor rather than treating browser focus as authentication.

## Terminal-only mounting surface

The complete component embeds this itself; never mount a second terminal for the
same current context/project (the backend rejects duplicate streams).

```js
import {mountTerminal} from '/assets/sodaspaces-terminal.js';
const terminal = mountTerminal(mountNode, {
  expectedUserId, repositoryId, environmentId, login,
});
```

Resolve environment and original own membership login through the protected API,
not a provider rename or caller-supplied privilege. Mounting starts nothing. Only
explicit Open loads the local renderer, authorizes and opens the existing account's
shell; it never creates, joins, starts or repairs a project. `invalidate`,
`disconnect`, `dispose` and the `started` getter retain their existing API.

Load `/assets/sodaspaces{,-drawer,-terminal}.css` and
`/assets/soda-terminal/xterm.css`. The renderer is lazy-loaded from local
`/assets/soda-terminal/{xterm,addon-fit}.mjs`; no CDN or frontend framework is added.
Keep xterm-specific styles scoped, so native form rules do not corrupt its textarea.
Browser cookies remain HttpOnly; actor/CSRF travels only in the bounded first socket
frame. No credential or terminal transcript belongs in logs. Terminal-driven OSC
clipboard/title/hyperlink actions are consumed; manual paste remains possible.
Scrollback, input/output, frames, queues and authorization waits stay bounded.

`internal/web/terminal.go` owns protected transport, context/project slots,
authorization, logout/rotation and shutdown. `internal/host/` owns native launch,
process supervision and independent safety lease. Those backend contracts are
unchanged in this layout slice. Closing a socket is not independent proof of process
cleanup, and ending a terminal does not undo completed writes or stop all detached
native workloads. See the [API](dashboard-api.md) and
[remaining workspace work](sodaspaces-plan.md#product-correction--development-workspace-not-a-modal-form).

## Packaging and evidence

Existing native build/stage/verification/install owners retain exact local payloads,
renderer locks and MIT notices; unsafe or occupied installation paths still refuse.
The footer's structural inventory review now records the aside/divider/Hide change
and preservation of notification markup. No new dependency or native target action.

Local DOM tests cover view/focus/hide preservation, keyboard controls, no replay,
action-time denial and old genuine-page-departure retirement. The opt-in
`tests/frontend/drawer-layout.test.mjs` passed 16 desktop/mobile/theme/state cases
with sandboxed Chromium, real locked xterm and **synthetic APIs/socket/native form**.
It tests half-width geometry, native-form interaction, keyboard resizing, full-height
canvas, tab changes and same-socket Hide/reopen. It is not real Forgejo navigation,
OAuth, process-continuity or native account/Git evidence. See the leading
[handoff](implementation-status.md) for actual runs and held work. Earlier installed
`2aa4960` journeys remain evidence only for their old product bytes and behavior.
