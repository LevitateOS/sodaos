# Drawer terminal integration contract

**Current source contract below, not the selected future UX.** The user rejected its
modal blocking, management-form layout and blur/close teardown. The next change is a
non-modal native-page/workspace split with full-content terminal tabs and bounded
same-process reattachment; see the [product correction](sodaspaces-plan.md#product-correction--development-workspace-not-a-modal-form).
None of those changes is implemented yet. Preserve actual authorization and native
ownership while deliberately replacing the old mounting/lifetime expectations and
tests; do not treat old blur-termination passes as acceptance of the new workflow.

The native `custom/header` and `custom/footer` hooks load one `sodaspaces.js`
dialog shell, which mounts `mountSodaspaces` once on explicit opening. It preserves
native repository actions, forms, clipboard and beforeunload. Content and terminal
modules do not discover or modify native navigation or auto-open a shell.
There is no new tab, standalone UI, CDN or component framework.

## Complete drawer content (minimum controls)

`/assets/sodaspaces-drawer.js` now exports `mountSodaspaces(mountNode, context)` with
`context = {expectedUserId, repositoryId}` (canonical native-page string hints). It
owns only content beneath that mount: Connect/Sign out, Create, Join, saved public
keys/removal, key review/Apply, Start/Stop, SSH details/Copy, Refresh and the existing
terminal component. It does not own a dialog, repository button, native selectors or
Forgejo layout. `sodaspaces.js` is now only the native dialog/context shell; its
historical duplicate API/action implementation is removed. The hooks load the
complete component, terminal styles and pinned xterm CSS. No second caller runs.

Load `/assets/sodaspaces-drawer.css` and the terminal styles below. Mounting itself is
inert; call the returned `.refresh()` on explicit drawer opening/Refresh to inspect
state (reads only). Call `.dispose()` before removal/close/context replacement, or
`.invalidate()` when native page/authentication context becomes stale. No remount to
bypass the full-page reload rule after blur/hidden/BFCache or an uncertain operation.
Closing also retires this page's mount: reopening offers explicit full-page reload,
not another component that could evade terminal/uncertain-operation guards. Refresh
cannot replace an opened/ended terminal. Do not promise cancellation of an already
dispatched mutation when closing the UI. Anonymous page context may offer contextual
OAuth; it never substitutes a session identity for the missing native actor hint.
All mutations recheck the current Soda session before dispatch; the server remains
the authority. JSON responses are MIME-checked and streaming-bounded to 64 KiB.

Start/Stop includes a visible boot-start policy and explicit shared-impact confirmation.
Key removal is saved preferences only; Review shows actual installed versus saved
fingerprints, then Apply makes the separate native change, with last-key confirmation.
A malformed/uncertain mutation response blocks further mutation replay, not safe reads
or explicit Soda logout. Native failures never become automatic repair/recreation.

The complete module embeds `mountTerminal` itself. Only use the standalone terminal
surface below when the template owner already supplies equivalent drawer controls;
do not mount two terminal components for the same displayed environment.

## Small terminal-only mounting surface

Load these local styles through the template owner's supported asset hook:

- `/assets/soda-terminal/xterm.css`
- `/assets/sodaspaces-terminal.css`

Import `/assets/sodaspaces-terminal.js` using a native module script, then mount once
inside the desired drawer content area:

```js
import {mountTerminal} from '/assets/sodaspaces-terminal.js';
const terminal = mountTerminal(mountNode, {
  expectedUserId: nativePageUserId,
  repositoryId: nativePageRepositoryId,
  environmentId: selectedEnvironment.id,
  login: ownEnvironmentDetails.login,
});
```

IDs are canonical decimal **strings**, except the environment's `p` + 24 hex ID.
Page user/repository IDs are consistency hints, not authentication. Obtain environment
ID and original own login from the existing protected Soda environment API; no template
access to Soda sessions, credentials or copied permissions is needed. Do not supply a
fresh provider rename instead of the stored membership login. The module independently
rechecks its Soda session and environment association/login before connecting.

Mounting makes **no request and launches nothing**. The component supplies its own
Open terminal, warning, status, terminal screen and Disconnect. Only the explicit Open
button fetches session/context, loads the pinned renderer and opens a WebSocket. It
never creates, joins, starts or repairs a project. Do not auto-click or invoke it from
page load, drawer opening, focus or restored browser history.

- Call `terminal.dispose()` **before closing/removing the drawer or replacing its
  environment/repository/account context**. It closes transport, clears renderer data,
  removes only its own subtree and unregisters its listeners.
- Call `terminal.invalidate()` when the host detects stale native-page/authentication
  context, or before local logout. Active/pending Soda logout is independently enforced
  by the backend. `terminal.disconnect()` ends this component without removing it.
- An ended/stale component requires full-page reload, not remount/reconnect/replay to
  evade the rule. The module also invalidates on window blur, hidden/pagehide and
  persisted pageshow. Do not reconnect on focus or BFCache.
- While terminal-focused, Escape belongs to the shell. Let its key handler run before
  interpreting Escape as dialog closure (do not swallow it in a document capture
  handler). Ctrl+Shift+Enter focuses Disconnect. Normal native dialog Escape outside
  the screen remains the template owner's responsibility.

The renderer is lazy-loaded from `/assets/soda-terminal/{xterm,addon-fit}.mjs`; browser
cookies remain HttpOnly, and authentication travels only in the bounded first socket
message. No terminal data, CSRF, cookie, query or transcript belongs in logs. OSC
clipboard/title/hyperlink actions are consumed; no link addon is installed. Manual
user paste works through xterm. Scrollback, input/output buffers and frame sizes are
bounded. Disconnect text does not claim that closing a browser socket proves native
process cleanup or undoes completed writes/detached workloads.

## Source and evidence boundary

`internal/web/terminal.go` owns the public protected route, live context/project slots,
logout/rotation/shutdown and periodic local authorization/peer checks. `internal/host/`
owns the fixed native operation and independent project lease. See the
[API contract](dashboard-api.md) and [terminal plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal).

`appliance/terminal-assets.lock.json` pins actual npm archive integrity and each shipped
upstream file hash; `scripts/fetch-terminal.py` writes build artifacts, not source or
runtime files. Native build/stage/bundle/install owners include the exact distributions
and both MIT notices, and refuse occupied component destinations. No template or
layout override is installed by this module itself.

Template mounting is source implemented. Source tests use synthetic provider/session/
helper inputs, not native-page acceptance. The opt-in `tests/frontend/drawer-layout.test.mjs`
uses sandboxed Chromium and the real locked renderer/styles, but synthetic APIs/socket:
its 16 light/dark, running/stopped, desktop/mobile combinations are local layout proof
only. Separately, native candidate `2aa4960` passed the real combined OAuth/proxy/
helper/browser terminal and management journeys on the isolated x86_64 fixture,
with independent shell-disappearance and SSH key-revocation observations. See the
leading [handoff](implementation-status.md) for exact evidence and the public-mode
packaging correction. Retained delivery and broader operator/provider/aarch64
acceptance remain separate; this document grants no execution or rollout permission.
