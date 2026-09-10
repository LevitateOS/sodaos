# Drawer terminal integration contract

## Multi-project workspace design and implementation

The user now explicitly wants multiple concurrent terminals across multiple projects,
with design before implementation. The research-backed [Spaces design](spaces-design.md)
now defines a project/session sidebar, direct cross-project splits with local tabs,
focus versus advisory attention, measured compact behavior and the same drawer view.
Static annotated sheets supersede the old grid-first interactive mockup; no semantic
agent status is real without an explicitly implemented signal adapter. These designs
reuse the same managed session mechanism, not separate terminal owners. The
[Lit implementation plan](lit-migration-plan.md) now owns the rendering ports and
subsequent concurrency/page/layout sequence. Management and terminal controls now
use Lit; the shared multi-session page/drawer now has local emitted-browser checks.
Bounded v2 pane layouts, shared chrome and compact behavior now have local coverage;
observed attention is implemented. Bounded x86_64 native concurrency/cleanup and
both-target delivery are recorded; selected CLI and broader native acceptance remain pending.
It preserves current HTTP End and exact restore, not the older socket-close contract. The
[drawer design](spaces-drawer-design.md) specifies the native-left/terminal-right
composition, a flat view of this window's open tabs without destroying full-page
pane layout, and the difference between native navigation, right-side view changes,
compact Forge/Terminal switching and explicit Hide. Drawings are not runtime proof.
The backend now uses ID-keyed ownership and a bounded authorized collection, and the
shared workspace owns multiple exact-ID children. Original bindings, one writer,
finite retention, capacity reservations, Stop and logout/rotation remain mandatory;
local rendering is not native concurrency or CLI compatibility proof.

The [handoff](implementation-status.md) retains `22d8591` evidence/failures and the
later **b8af68c** native boundary, two-actor six-session matrix and independent End
cleanup proof. Its startup correction waits for first native output before transport
ready/input forwarding, preserving bounded startup/input/lease limits. That prevents
tmux's initial flush from losing input; it does not certify an agent's readiness or
browser usability. Older source-slice status below is historical, not current deployment.

## Open product question — browser-terminal compatibility

The user has put the browser-terminal direction into question before treating it as
settled product design. Soda expects modern interactive command-line applications,
including Codex CLI, Claude Code and Pi. A browser xterm renderer and WebSocket
attachment may be a useful convenience, but source-level protocol support or a
successful basic shell session does not establish that these applications are usable
there.

This is a deferral and an explicit compatibility concern, not a decision to remove
the browser terminal, replace tmux, change transport, or select another renderer.
Before relying on the browser terminal as a primary development surface, obtain
target-appropriate evidence using the actual selected CLIs: Unicode and cursor
behavior, alternate-screen redraw, mouse and paste input, resize, scrollback, long
streaming output, reconnect/retention and ordinary SSH coexistence. Native SSH in a
capable local terminal remains the comparison path; it is not evidence that the
browser-terminal experience is equivalent.

## Current workspace layout and continuity

The native `custom/header` and `custom/footer` hooks load one `sodaspaces.js`
shell mounting `mountSodaspaces` once on explicit opening. The shell now uses a
**non-modal aside**, not `showModal`, a backdrop or outside-click dismissal.
Desktop starts at half width; a pointer/keyboard separator adjusts it between
35–65 percent where measured terminal cells and a 480px native-left region fit.
Otherwise document-local **Forge / Terminal** controls select a full-width surface,
without Hide, Return, disposal or a saved Terminal-visible flag. Desired width is
preserved; new navigation/BFCache defaults to Forge unless explicitly opened.
Native body/container widths adapt without cloning/reparenting forms or replacing
navigation, notifications or beforeunload. Real native route/pane-width compatibility
still needs validation; the 20-case synthetic layout fixture is not that proof.

The page and drawer now share one Lit workspace with session tabs and independent
Environment/Access controls bound to each named project. Explicit New creates a
correlated session; selecting an existing session attaches only that ID. Flat terminal
hosts remain in one flat owner layer through pane moves, splits, maximize, compact
projection and details. The page has a resizable/collapsible project/session sidebar;
the drawer flattens the same desired tree, with right-content-only Sessions and an
explicit This page filter. Splits create views, never shells, and require measured
56-column × 12-row children after chrome. Rename, Hide, Continue/Keep and named,
Cancel-first End confirmation retain the original account/project. Actions live in
per-terminal menus; Stop remains in named project management with its shared warning.

`soda-spaces:v2:<actor>` stores bounded locators, pane membership/order/selection,
finite split ratios and sidebar preference: at most 32 KiB, 64 retained entries/leaves,
127 nodes and depth 64. It stores no names, credentials, transcripts or input. V1 imports
only when v2 is absent, preserving hidden/exact/pending entries and old bytes. Invalid
present v2 is preserved rather than overwritten or silently replaced from v1. Storage
failure does not destroy live owners. Unknown outcomes survive reload; legacy pending
is never guessed. This has local emitted-browser coverage, not concurrent native proof.

**Focus/visibility changes and Hide/reopen retain the same component and socket.**
Focus and switching already attached views do not fetch, provision, restart or
replay anything. Selecting an unattached exact ID reauthorizes attachment only.
Hide requests finite retention for this document's owned attachments; deliberate
reopen Returns only an eligible, finitely retained selected terminal in terminal view,
not the entire working set. Clicking an already-open workspace does not Return.
Pending operations
remain guarded in that same component; hiding is not cancellation. End terminal is
a separate action. No outside click dismisses the pane and no focus trap blocks the
native page. Escape inside xterm remains shell input; Ctrl+Shift+Enter focuses the
terminal actions menu. Managed input requires its visible screen to own focus. The complete component retains the existing action-time session/provider
checks and uncertain-mutation guard.

Refresh now preserves the renderer for the same environment/login. Pagehide retires
only the document's attachment; BFCache restoration and navigation mount fresh
components with fresh authorization. A per-tab `sessionStorage` workspace/terminal
locator preserves the selected repository instead of following the left page. It
contains no credential, transcript or queued command. Switching repositories is
explicit. Signed non-repository pages have a resume hook, not a Spaces listing.

## Managed terminal implementation and proof limits

The image recipe adds stock tmux and required terminfo/tools, and copies the single
helper-owned Python source to `/usr/libexec/soda/project-terminal`. Project init
creates the root-controlled runtime parent. Open verifies the installed program against
the helper's embedded-source SHA256; it does not install or repair missing support.

Creation opens a backend-owned helper connection, independent of browser transports.
One transient project-systemd `soda-terminal-ID.service` contains a **root guard** and
foreground tmux server/descendants running as the original non-root account. The
guard protects the native lease from ordinary account processes. Notify/watchdog,
control-group killing, finite startup/stop/runtime limits and no restart supervise
it. Private runtime records bind account/identity, lease, socket inode/server peer
credentials and exclusive writer lock. Socket parent sealing and exact `-N` attach
refuse replacement/adoption. No personal tmux configuration or default server is used.

The in-memory backend registry is ID-keyed, retaining the original Soda context/token,
actor, project/repository and Linux login on each entry. Its 64 global slots include
detached and cleanup-unconfirmed sessions. It is not restored after backend restart.
Every 15 seconds, current session/membership and fresh acting-user repository authority
must allow renewal of the native 60-second safety lease. Detach or explicit Hide
sets 30-minute retention unless an explicit deadline exists. Output, input, pings
and attach-only retries do not clear/extend it. Explicit Return is active while
attached, otherwise finite; Keep two hours is finite. The healthy/native hard cap
is 12 hours, never beyond the original Soda session expiry. Logout, rotation,
confirmed denial, expiry, shutdown and Stop cancel browser access/ownership; none
is a Linux-account deletion or shared-project idle stop.

The protected per-ID `terminal-sessions/{terminalID}` API reports lifecycle/effective
and hard deadlines, accepts End/Return/Keep/Hide and bounded display-only Rename.
The separate `terminal-attempts/{requestID}` read resolves an exact uncertain create,
never the newest session. The old singleton endpoint returns 410. The socket selects
correlated create or exact attach, emits ID/request/writer-generation locators then
readiness, and carries only input/resize thereafter. Hide defaults only an unset
deadline; Hide/active Return must match the original attachment generation. A stale
window cannot change its successor writer's lifetime. See the
[wire contract](dashboard-api.md#terminal-websocket--managed-tmux-contract).
End acknowledges **ending**, not process disappearance. Slots remain reserved until
native cleanup acknowledgment; a lost/uncertain dispatch or cleanup reports
`unconfirmed` and refuses replacement of that ID. Independent explicitly created IDs
can coexist in the same project within capacity. Only acknowledged cleanup releases
a slot; up to 128 authorized receipts last at most five minutes/original authentication.
Absent/expired receipts are unknown, not cleanup proof. The client reads the exact
result after End and keeps uncertain locators. No automatic reconciliation/retry frees them.
The native guard/safety lease remains independent; an operator must inspect uncertain
outcomes. Cleanup proves the selected cgroup empty before removing only admitted
runtime files; unsafe or changed state is retained, not recursively deleted.

Local HTTP/DOM/temporary-filesystem/PTY doubles are not native proof. The isolated
`22d8591` run added tmux/program bytes to both original roots and verified actual
browser same-PID/memory reload and acknowledged/independently observed cleanup.
The separate native probe still fails raw-output confirmation; its complete two-
account/resize/interrupt/lease/helper-loss journey is not passed. Broader screen/
history/editor/build/network and unrelated-workload proof remains required.

That probe requires `terminal_protocol: managed-tmux-v1` and exact container inputs;
old private approvals cannot accidentally run it. The isolated maintenance recipe
and scoped backup waiver are in the handoff, not permission for another root/target.
A new image still does not upgrade a retained writable root.

## Selected persistence mechanism — tmux, source candidate

The [leading plan](sodaspaces-plan.md#resumable-terminal-decision--tmux) selects
**stock Rocky-packaged tmux**, not interchangeable backends. The original decision
was documentation-only; the source candidate above now implements that mechanism. The [Project OS baseline](project-os.md)
owns the shared account/profile/state and same-root delivery contract; this guide
owns the terminal mechanism. No broader OS redesign is a prerequisite.

### Source comparison

| Candidate | Finding and decision |
| --- | --- |
| **tmux — selected** | Explicit new versus attach, exact targets, no-server-start flag, foreground server, native screen/history and separate sockets. Required mechanisms exist in upstream 3.2a, matching Rocky 9's published base version; a custom newest-release build is unnecessary. |
| **shpool v0.11.4** | Good persistence-only UX and in-memory screen restoration, but neither released CLI nor attach protocol has a require-existing operation. Missing/exited sessions can fall through to shell creation. `--no-daemonize` only prevents daemon startup; list-then-attach races, and a Created reply arrives after the effect. Its creation-relative `--ttl` is not a detached grace period. Not selected; no fork or sentinel-shell workaround to approximate attach-only. |
| **[Zellij][zellij-cli] / GNU Screen** | Capable native multiplexers, not claimed unable to persist/attach. Zellij v0.45.1 also has a larger workspace/plugin/web-token surface; we do not need its web frontend or session resurrection. Neither offers a demonstrated advantage over the existing-OS tmux path for this slice. |
| **[abduco][abduco] / [dtach][dtach]** | Explicit attach-only exists, but process survival is not complete screen restoration. Abduco recommends dvtm for redraw problems; dtach passes raw bytes and relies on application redraw. Avoid adding a second renderer/replay mechanism to close that gap. |
| **Custom PTY broker / systemd alone** | Systemd supplies process supervision, not retained terminal screen state. Keep Soda's small authorization/lifetime adapter, not a new terminal emulator or general orchestration service. |

Reviewed sources: tmux [manual][tmux-man], [client connection][tmux-client],
[attach handler][tmux-attach] and [foreground server][tmux-server] at upstream
3.2a commit `3b929f332aafa7f1080eacc31feb11ffbb1d1841`; shpool
[CLI][shpool-cli], [protocol][shpool-protocol] and
[session selection][shpool-server] at v0.11.4 commit
`3a1020c74d1ac8eb191270922d834aa452cac816`.
Rocky 9 BaseOS indexes list `tmux-3.2a-5.el9` for both
[x86_64][rocky-x86] and [aarch64][rocky-arm]. These are source/package-availability
facts, not installed binary, downstream-patch, signature or compatibility proof.
Research inputs and comparisons are retained in `.artifacts/research/terminal-options/`.

### Bounded native ownership

- **One managed browser terminal → one private tmux server/session and project-local
  systemd service/cgroup.** Run the foreground server as the original marker-bound
  non-root account, with its real groups/home/login shell. Keep the existing trusted
  project/CID/user-namespace checks; no caller-selected login, socket, command, unit
  properties or host Podman flags. Creation refuses unsafe runtime paths or occupied
  destinations instead of deleting/adopting them; attachment resolves only the
  verified existing owned socket. No human host account or new public listener.
- **Soda owns access; tmux is not authentication.** A stable terminal ID is only a
  locator. Bind it to the user, project, original login, Soda login context and native
  lifetime; never reuse an ended ID for a new shell. Fresh same-user/session/consent/
  repository authorization is required before attach; IO stays gated by that live
  authorized attachment and server session/membership lease checks, not a provider
  request for every keystroke. Preserve logout-winning dispatch and rotation
  invalidation. No degraded-read or operator membership bypass.
  Initially retain one managed terminal per context/project and the global 64 bound,
  counting detached terminals too. Reject duplicate writers; no automatic client
  eviction or cross-context adoption. Minimal lifetime metadata is legitimate Soda
  state, not copied provider permissions, a transcript or a replayable job journal.
- **Separate creation from attachment in the helper.** Creation alone starts a
  foreground `tmux -D` with an explicit `-S` private socket and `-f` managed config,
  then explicitly creates its session. Use `-N` for subsequent control commands,
  including new-session, so they cannot replace a failed supervised server.
  Reattachment uses the native form
  `tmux -N -S SOCKET attach-session -E -t =SESSION`: no server startup, no target
  prefix/glob fallback and no update-environment side effects. These are command
  shapes, not a public arbitrary-command API. Do not use `new-session -A`, a default
  socket, user-supplied config, attach `-d`/`-x` or a missing-session create fallback.
  Plain `attach` can start a server and execute startup configuration: **`-N` matters**.
- **Supervise the server, not only its PTY client.** `-D` disables daemonization but
  also turns `exit-empty` off; it does not implement expiry or guarantee clean startup.
  Bound startup, empty-server lifetime and shutdown. Respect the separate
  [project/systemd/nested-workload owners](project-os.md#runtime-and-supervision-boundaries);
  disabled inner workload cgroups are not proof of terminal cgroup supervision.
  A project-local guard and systemd cgroup must cover the server and remaining owned
  descendants, including guard/server/helper failure and jobs ignoring hangup. Keep restart/resurrection
  disabled. End/expiry terminates only that managed service's processes, not a UID,
  default tmux server, other browser terminals or independently managed SSH/services.
  Jobs still inside that cgroup are included even if they ignored PTY hangup; this is
  not a promise to terminate workloads deliberately launched outside it. Verify
  owned process disappearance independently; socket closure is insufficient.
- **Keep transport, retention and safety leases separate.** A lost browser client
  detaches; it must not expire the native session's authority lease merely because
  the WebSocket ended. The server-side lifetime owner may renew that short native
  lease only within the authorized session/retention deadline, independent of browser
  output/heartbeats. Missing owner/guard or actual expiry still produces bounded
  cleanup. Carry the proposed 30-minute detached/closed-drawer window and explicit
  finite longer option (for example two hours away), without output/automatic reconnect
  renewing abandonment. Only a deliberate authenticated return or explicit finite
  extension may change that abandonment deadline, not transport attachment itself.
  Focus loss alone is not detachment, silence is not proof an editor/build is idle,
  and there is no browser-idle stop of the shared project.
  Authentication expiry, explicit Soda logout and confirmed authority loss remain
  hard browser-access boundaries; no unauthenticated reattachment grace is selected.
  The source candidate replaces the healthy-active two-hour cap with the original
  session-bound 12-hour maximum, separately from these safeguards.
- **Default to the drawer UI, not a nested dashboard.** Hide tmux's status bar by
  default; Soda's later session tabs map to separate managed terminals. Native tmux
  copy-mode and splits remain available. Tmux renders its own terminal screen:
  hiding the bar does **not** make its history identical to xterm's native scrollback.
  Make history/selection/paste usable and review it visually; do not promise replay
  of all past output into a fresh browser. Bound tmux history/buffers and renderer
  queues, disable automatic clipboard writes and retain xterm's OSC protections.
  No transcript/debug/pipe-pane logging or resurrection plugins. Preserve ordinary
  user shell history/configuration; do not edit `.bashrc`/SSH to force tmux, import
  personal tmux startup commands into the managed server or adopt independent sessions.
- **Use the project OS, not a new dependency platform.** Add the normal signed Rocky
  package through `project-os/Containerfile` and managed unit/config through existing
  production owners. Record actual resolved RPMs/notices and verify the inner/outer
  TERM descriptions, login profile and native systemd/cgroup behavior on each target.
  No new Rust service, custom tmux build or runtime download. A new image does not
  retrofit retained writable roots; missing support must refuse honestly, not install
  on Open, fall back to the disposable PTY or replace a project. Existing-project
  package/helper delivery requires its exact maintenance scope, preserving later work.

### Remaining native proof and multi-session extension

The single managed terminal is implemented. Preserve its helper/API/renderer
boundaries in the [Lit ports and ID-keyed extension](lit-migration-plan.md), not a
parallel backend or support-tool gate. Refresh and unrelated management updates
must not end a running terminal. Navigation/reload restores the authorized selected
session, not the left page's repository; pagehide disposes only this document's
attachment. Same-document tab/layout changes retain the live renderer/socket.
The singleton limit was removed by the implemented ID-keyed backend slice, not a
UI bypass; the browser layout does not change the server's bounded capacity.

Prove a real unchanged shell PID/start identity, unsaved editor state and running
build across native navigation, reload, temporary network loss and drawer/view
changes; verify alternate-screen restoration, detached output/history, Unicode,
resize, Escape/Ctrl-C, keyboard focus, selection and paste. Then prove missing/ended/
expired targets create nothing, duplicate/racing attaches cannot steal a writer,
logout/rotation/denial/Stop win, and finite detached/lease/guard/server/helper failure
cleanup preserves unrelated SSH/tmux and other accounts/terminals. Source tests plus
real OAuth/trusted-browser/native observations are required; old request-PTY cleanup
passes and synthetic layout tests are not this proof. Restart/reboot resurrection
of running terminals is not selected; retained files and normal project persistence
remain separate. ID-keyed metadata, creation correlation and bounded confirmed-cleanup
receipts are current source contracts with local Go/race/browser coverage. The
current installed-journey ports are authored, not installed execution evidence.
No further native execution is granted by this plan.

[tmux-man]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/tmux.1
[tmux-client]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/client.c
[tmux-attach]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/cmd-attach-session.c
[tmux-server]: https://github.com/tmux/tmux/blob/3b929f332aafa7f1080eacc31feb11ffbb1d1841/server.c
[shpool-cli]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/libshpool/src/lib.rs
[shpool-protocol]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/shpool-protocol/src/lib.rs
[shpool-server]: https://github.com/shell-pool/shpool/blob/3a1020c74d1ac8eb191270922d834aa452cac816/libshpool/src/daemon/server.rs
[rocky-x86]: https://dl.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/Packages/t/
[rocky-arm]: https://dl.rockylinux.org/pub/rocky/9/BaseOS/aarch64/os/Packages/t/
[zellij-cli]: https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/cli.rs
[abduco]: https://github.com/martanne/abduco
[dtach]: https://github.com/crigler/dtach

## Complete content boundary

Authored modules live in `.ts`; these browser imports address Bun-emitted `.js`
assets at their stable public URLs, not unchecked JavaScript sources.

```ts
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

```ts
import {mountTerminal} from '/assets/sodaspaces-terminal.js';
const terminal = mountTerminal(mountNode, {
  expectedUserId, repositoryId, environmentId, login,
});
```

Resolve environment and original own membership login through the protected API,
not a provider rename or caller-supplied privilege. Mounting starts nothing. Explicit
Open may create a terminal for an existing account; `restore()` only attaches a
stored existing locator after fresh authorization. Neither provisions, joins, starts
or repairs a project. `retain()` and `returnToWork()` request the bounded lifetime
operations above. `invalidate`, `disconnect`, `dispose` and `started` remain available;
renderer disposal detaches, while End is the separate protected operation.

Load `/assets/sodaspaces{,-drawer,-terminal}.css` and
`/assets/soda-terminal/xterm.css`. The renderer is lazy-loaded from local
`/assets/soda-terminal/{xterm,addon-fit}.mjs`; no CDN. Lit will own controls through
the already scaffolded shared runtime, never the xterm-owned screen descendants.
Keep xterm-specific styles scoped, so native form rules do not corrupt its textarea.
Browser cookies remain HttpOnly; actor/CSRF travels only in the bounded first socket
frame. No credential or terminal transcript belongs in logs. Terminal-driven OSC
clipboard/title/hyperlink actions are consumed; manual paste remains possible.
Scrollback, input/output, frames, queues and authorization waits stay bounded.

`internal/web/terminal.go` owns protected transport, context/project slots,
authorization, logout/rotation and shutdown. `internal/host/` owns native launch,
process supervision and independent safety lease. The source candidate revises
these contracts as described above. Closing a socket is not independent proof of process
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
`tests/frontend/drawer-layout.test.ts` passed 16 desktop/mobile/theme/state cases
with sandboxed Chromium, real locked xterm and **synthetic APIs/socket/native form**.
It tests half-width geometry, native-form interaction, keyboard resizing, full-height
canvas, tab changes and same-socket Hide/reopen. It is not real Forgejo navigation,
OAuth, process-continuity or native account/Git evidence. See the leading
[handoff](implementation-status.md) for actual runs and held work. Earlier installed
`2aa4960` journeys remain evidence only for their old product bytes and behavior.
