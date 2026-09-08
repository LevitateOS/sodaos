# Drawer terminal integration contract

The template/layout agent owns **all Forgejo overrides, navigation, dialog placement,
width and existing drawer actions**. The terminal implementation does not discover or
modify those nodes, auto-mount, intercept native navigation or replace beforeunload.
There is no new tab, standalone UI, CDN or component framework.

## Small mounting surface

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

Source tests use synthetic provider/session/helper/renderer inputs, not native-page
acceptance. The earlier native PTY gate passed independently. Actual template mounting,
whole-candidate native build/stage, real OAuth/proxy/browser proof and deployment remain
separate work; this document does not authorize native execution or retained rollout.
