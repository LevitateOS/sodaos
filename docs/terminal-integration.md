# Drawer terminal integration contract

The [selected mechanism removals](refactoring-plan.md#selected-mechanism-removals)
replace browser-owned shell lifetime with native ownership, exact native lookup and
a disposable current browser cache. This is a **source candidate**, not an installed
upgrade. The [handoff](implementation-status.md) owns target state/permissions;
[history](implementation-history.md) owns earlier receipts. Evidence for the former
guard/lease/receipt implementation does not establish this candidate's native behavior.

## Boundary with planned desktop and automation views

This guide owns personal managed terminals across supported Project OS profiles.
KDE adds graphical access to the same native account/home/tools; it does not replace
tmux or make a PTY a desktop transport. Graphical viewers and
[AI run-owned processes](services-and-ai-plan.md#live-ai-terminals-in-spaces-and-the-drawer)
have their own authorization and lifetime contracts. Do not impose those scopes on
personal shells, fabricate a personal sign-in for automation, commandeer a shell or
assume personal-terminal inventory includes jobs. Desktop/agent availability cannot
be inferred from successful PTY attachment.

## Multi-project workspace design and implementation

The [Spaces design](spaces-design.md) defines a project/session sidebar, direct
cross-project splits with local tabs, focus versus advisory attention and a measured
compact projection. The [drawer design](spaces-drawer-design.md) defines the
native-left/workspace-right composition and flat view of this window's tabs without
destroying its desired full-page layout. The shared Lit workspace owns independent
exact-ID children, not a second terminal backend. Designs and local emitted-browser
checks are not native concurrency or CLI compatibility proof.

## Open product question — browser-terminal compatibility

Modern interactive CLIs, including Codex CLI, Claude Code and Pi, still require
appropriate browser and native evidence. Basic shell/transport support does not prove
Unicode/cursor behavior, alternate-screen redraw, mouse/paste, resize, history,
streaming output or reconnection usability. Compare actual selected versions with
ordinary SSH in a capable local terminal. This concern does not select another
renderer, remove the browser terminal or authorize provider use.

## Current workspace layout and continuity

Native Forgejo hooks mount one shared workspace on explicit opening. The drawer is
a non-modal aside, not an outside-click-dismissed dialog. Desktop begins at half
width; measured cell capacity and a 480px native-left minimum constrain resizing.
Otherwise document-local Forge/Terminal controls select a full-width surface. Desired
width survives; navigation/BFCache defaults to Forge unless explicitly opened.
Native forms, routing, notifications and beforeunload remain Forgejo-owned.

Flat terminal hosts remain in one owner layer through tabs, pane moves, splits,
maximize, compact projection and project details. Splits create views, never shells,
and require measured 56-column × 12-row children after chrome. The drawer flattens
the desired tree and offers a This page filter without switching the selected project.
Rename is display metadata. Hide and show change presentation only: no retention,
Return or Keep operation exists. End remains a separate named, Cancel-first
confirmation. Project Stop remains a separate shared-impact action.

`soda-spaces:v3:<actor>` stores current exact locators, pane order/selection, finite
ratios and sidebar preference: at most 32 KiB, 64 entries/leaves, 127 nodes and depth
64. It stores no credentials, names, transcript or input. No older format or pending
request namespace is imported. Corrupt/obsolete current data resets to an empty
arrangement with a notice; a later save may overwrite it. Storage failure does not
destroy live components or permanently disable saving. Cache loss/reset never
Creates or Ends work; authorized native inventory discovers surviving shells.

Focus/visibility changes preserve live renderers and sockets without network writes.
Input requires the visible screen to own focus. Escape inside xterm remains shell
input; Ctrl+Shift+Enter focuses its actions menu. Pagehide retires the document's
attachment, not native work. Navigation/BFCache uses fresh components and their
original-actor bootstrap; restoration inspects and attaches only the saved exact ID.
A missing, ended or unavailable target never falls through to Create.

Attention reflects actual output/unread, connection loss, another observed writer,
ending or stale/unavailable observations—not deadlines, agent progress or prose.
Late callbacks cannot update successor owners, send input or claim cleanup.

## Managed terminal implementation and proof limits

### Allocation and creation

The server issues one random terminal ID through a protected reservation POST before
any shell Create can be sent. The reservation captures bounded dimensions/name,
original account/identity and an opaque digest of the creating Soda context. It is
stored beside native runtime state, expires after two minutes and starts **no shell**.
The browser publishes/persists the ID before opening its Create transport. It never
invents a second request ID or automatically retries Create.

Native reservation consumption, Create and End share the project-local parent lock.
Create consumes permission before native dispatch. An expired, consumed, changed or
ended permission cannot start work. This placement matters: a paused helper dial,
lost reply or web restart cannot deliver a late Create after End removed permission.
An unused reservation is not reported as absent while it could still authorize Create.
It is not published as a running shell in the collection. Logout cannot be used to
consume an old context's reservation through a fresh sign-in.

### Native lifetime

The image includes stock tmux, required terminfo and the helper-owned program at
`/usr/libexec/soda/project-terminal`. Project init creates its root-controlled runtime
parent. Missing/changed support refuses; Open never installs packages or replaces roots.

One project-systemd `soda-terminal-ID.service` runs foreground tmux as the original
non-root account. `Type=exec`, control-group killing, `Restart=no`, bounded startup
and stop, null standard IO, a restricted environment and umask remain. A short
privileged `ExecStartPost` checks its own cgroup and systemd's MAINPID, waits for the
native socket, seals its parent and creates the session. Because `tmux -D` disables
`exit-empty`, the same native command queue restores `exit-empty on` **after**
`new-session`. Shell exit can therefore end the server normally.

There is no root lifetime guard, owner WebSocket, watchdog, abandonment timer or
native RuntimeMaxSec. Browser closure, logout, authentication expiry and web/helper
restart revoke/detach access, not native work. Systemd owns the remaining process
lifetime. Explicit End stops that exact unit and confirms its cgroup empty before
removing only admitted runtime files. Project Stop ends the project's processes;
it does not delete persistent homes, checkouts or project data.

### Lookup, capacity and access

Native binding, socket inode/peer credentials and the non-evicting writer lock remain
mandatory. Inventory/inspect use a shared parent lock; mutations use its exclusive
lock. Exact native unit/cgroup observations distinguish absence from unavailable
state. Changed account, unsafe paths, unknown records, unsupported supervision or
failed observation refuse; they do not become complete empty truth. Reads do not
collect files. Allocation/creation may retire only recognized completed runtime state
once native absence/empty cgroup is confirmed; unknown/older layouts are not converted.

Capacity is actual native work: at most 64 active managed terminals **per project**,
including abandoned shells. At most 128 runtime directories bound allocations and
inspection. Expired unused allocations can be reclaimed at admission. No web registry
reserves permanent unconfirmed slots or stores five-minute cleanup receipts. There
is no durable browser history or automatic reconciliation/replay service.

A locator is not authority. Every actual protected operation checks current Soda
session, expected actor, CSRF/Origin as applicable, acting-provider authorization,
repository visibility and original own membership login. Fresh authorized sign-ins
for that same native account can discover/attach its work; they cannot acquire someone
else's work or evict a writer. Operator/site-admin/owner status is no membership bypass.
IO retains bounded attachment expiry/heartbeat and periodic authority checks. Those
bounds close the PTY **client**, not its tmux server. Pending transports/operations
are bounded at 128; provider/native IO occurs outside the web peer lock with
cancellation and post-IO session checks. Stop closes project admission and peers.

The [API guide](dashboard-api.md#terminal-websocket--managed-tmux-contract) owns wire
fields and exact response semantics. An unavailable outcome stays unavailable until
another explicit observation/action; it is not proof of cleanup, permission to Create
again or a permanent browser admission flag.

### Maintenance boundary

For **this new native representation**, web/helper restart does not own shell
termination. It still interrupts access and requires target-scoped maintenance
approval and consistent preservation. This claim has not been proved natively.

Retained projects may still contain the former root-guard/owner-stream implementation.
Restarting their owner service can terminate those old sessions. They are not
transparently adopted or converted, and this source approval grants no retained-state
conversion/cleanup or service restart. Consult current target state and the
[affected-component maintenance procedure](installation.md#retained-sodaspaces-cutover).
A new image does not retrofit an existing writable root.

## Selected persistence mechanism — tmux, source candidate

The [leading decision](sodaspaces-plan.md#resumable-terminal-decision--tmux) selects
stock Rocky-packaged tmux, not interchangeable backends. Project account/profile and
same-root delivery requirements belong to [Project OS](project-os.md).

### Source comparison

| Candidate | Finding and decision |
| --- | --- |
| **tmux — selected** | Exact new/attach, no-server-start flag, foreground server, native screen/history and separate sockets exist in upstream 3.2a. No custom newest-release build is required. |
| **shpool v0.11.4** | No require-existing CLI/protocol operation: a missing/exited session can create a shell. List-then-attach races; `--no-daemonize` prevents daemon startup, not shell creation. Not selected; no sentinel/fork workaround. |
| **Zellij / GNU Screen** | Capable multiplexers, not claimed unable to attach. No demonstrated advantage for this selected path; Zellij's workspace/plugin/web-token/resurrection surface is not needed. |
| **abduco / dtach** | Explicit attach exists, but process survival alone does not restore terminal screen state. Do not add a second renderer/replay system to close that gap. |
| **Custom PTY broker / systemd alone** | Systemd supervises processes, not screen history. Keep native tmux and Soda's actual authorization/transport boundary. |

Reviewed tmux [manual][tmux-man], [client][tmux-client], [attach handler][tmux-attach],
[server][tmux-server] and [configuration queue][tmux-cfg] at upstream 3.2a commit
`3b929f332aafa7f1080eacc31feb11ffbb1d1841`; shpool
[CLI][shpool-cli], [protocol][shpool-protocol] and [selection][shpool-server] at
v0.11.4 commit `3a1020c74d1ac8eb191270922d834aa452cac816`.
Rocky 9 BaseOS indexes listed `tmux-3.2a-5.el9` for [x86_64][rocky-x86] and
[aarch64][rocky-arm]. These are source/package-availability facts, not installed
binary, downstream-patch, signature or compatibility proof. Research remains in
`.artifacts/research/terminal-options/` and the mechanism-removal check directory.

### Bounded native ownership

- Use original marker-bound non-root credentials/groups/home/shell. Preserve trusted
  project/CID/user-namespace checks; no caller-selected command, UID, socket, unit
  properties or host Podman flags. Never install support on Open.
- Create only through the consumed native reservation. Use a private `-S` socket and
  managed `-f` configuration; no personal startup configuration or default server.
  All subsequent commands use `-N`, including `new-session`. Attach is exactly
  `tmux -N -S SOCKET attach-session -E -t =soda`, without `new-session -A`, `-d`/`-x`,
  target-prefix fallback or create-on-missing behavior.
- Supervise the server and its cgroup, not just the attachment client. Normal shell
  exit and explicit End are native lifecycle actions; authentication/IO deadlines
  bound access only. Verify process/cgroup disappearance independently. End does not
  undo completed writes or promise to terminate work deliberately launched outside
  the terminal cgroup. Never kill a UID, personal tmux server or unrelated SSH job.
- Keep tmux status hidden by default, native copy-mode available, and history/buffers
  bounded. Tmux screen/history is not identical to xterm scrollback. Disable automatic
  clipboard writes and consume terminal-driven OSC actions. No transcript/debug/
  pipe-pane logging, resurrection plugins or edits to personal shell/SSH configuration.
- Preserve native terminal raw-readiness gating, bounded input/output/queues, resize
  and sane interrupt behavior. Native readiness does not certify an agent's readiness.

### Remaining native proof and multi-session extension

The current source candidate needs matching native evidence for normal shell exit,
preparation failure/partial startup, exact absence versus unavailable state, capacity,
reservation expiry/consumption/End races, duplicate writers and actor denial. Prove
unchanged shell PID/start and in-memory state through attachment loss and web/helper
restart, then independent exact End cleanup with unrelated work preserved. Verify
navigation/reload, unsaved editor/build state and actual selected CLI rendering/input
where those claims are intended. Do not count synthetic peers or an older guard-era
receipt as this proof. No native execution is granted here.

[tmux-man]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/tmux.1
[tmux-client]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/client.c
[tmux-attach]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/cmd-attach-session.c
[tmux-server]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/server.c
[tmux-cfg]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/cfg.c
[shpool-cli]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/libshpool/src/lib.rs
[shpool-protocol]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/libshpool/src/protocol.rs
[shpool-server]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/libshpool/src/daemon/server.rs
[rocky-x86]: https://dl.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/Packages/t/
[rocky-arm]: https://dl.rockylinux.org/pub/rocky/9/BaseOS/aarch64/os/Packages/t/

## Complete content boundary

Authored modules are TypeScript, with Bun-emitted JavaScript at stable public URLs:

```ts
import {mountSodaspaces} from '/assets/sodaspaces-drawer.js';
const controls = mountSodaspaces(mountNode, {expectedUserId, repositoryId});
controls.refresh(); // mounting alone is inert
```

Canonical native-page IDs are context hints, not authentication. Resolve the original
Soda actor/CSRF once for a fresh workspace; do not silently adopt a later identity.
Actual server operations remain authorized. `invalidate()` retires stale context;
`dispose()` precedes real removal. Neither hiding nor switching views requires either.
Native Forgejo retains forms, navigation and beforeunload. JSON reads remain
MIME-checked, streaming-bounded, same-origin/no-store and redirect-rejecting.

## Terminal-only mounting surface

The workspace owns locators, layout and project/rename/Hide commands. Standalone
callers must explicitly choose `new` or `existing`; pending/request-ID modes and
implicit no-locator mounting are retired. The renderer injection argument is fourth.

```ts
import {mountTerminal} from '/assets/sodaspaces-terminal.js';
const terminal = mountTerminal(mountNode, {
  expectedUserId, csrfToken, repositoryId, environmentId, login,
}, {kind: 'existing', id: terminalId});
```

Resolve original own membership login through the protected API, not a provider
rename or caller-selected privilege. Explicit Open on `new` reserves/publishes an
ID before Create; `restore()` only inspects/attaches its existing ID. Neither path
provisions, joins, starts or repairs a project. `invalidate`, `disconnect`, `dispose`
and `started` remain; no retain/return API exists. `soda-terminal-locator` events
carry an exact ID or confirmed-End `null`, never a request namespace. Components do
not write their own separate storage records.

Load the canonical Spaces/terminal CSS and locked local xterm assets; no CDN. Lit
owns controls, never xterm's screen descendants. Preserve native forms with scoped
styles. Cookies remain HttpOnly; original actor/CSRF accompanies actual protected
requests and the bounded first socket frame, never storage, URLs or logs. Terminal
output and credentials must not enter evidence.

## Packaging and evidence

Existing build/stage/install owners retain exact payloads, renderer locks, attribution
and licenses. Coordinated helper/program/browser delivery still requires the selected
target/action grant. Current local checks, historical native receipts and remaining
installed/CLI work are recorded separately in the [handoff](implementation-status.md)
and [history](implementation-history.md); none authorizes a retained-root conversion.
