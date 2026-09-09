# Spaces: multi-project terminal workspace

**Design proposal, not an implemented page or an execution grant.** The user selected
Spaces beside Explore and explicitly requested multiple concurrent terminals across
multiple projects, with UI/UX design first. The layout, interaction details and limits
below are recommendations for review, not silently accepted implementation decisions.
The [leading plan](sodaspaces-plan.md) owns product scope; the
[handoff](implementation-status.md) distinguishes installed evidence from this proposal.

## 1. Product model

Spaces is the place to **do work across projects**, not a dashboard of environment
cards with a terminal hidden underneath. The repository drawer is its compact view.

- **Project environment:** the shared Linux container, accounts, tools and files.
- **Terminal session:** one personal, managed shell/process boundary in a project.
  A project can have several of the current person's terminals simultaneously.
- **Terminal tab:** a view of an existing terminal, not its owner or a new shell.
- **Pane group:** a rectangular area showing one selected tab. Several groups allow
  simultaneous visible terminals, including terminals from different projects.
- **Working set:** the tabs/layout deliberately opened in this browser window.
  The navigator also lists kept/detached sessions that are not in that working set.

Changing project selection, filtering, reordering or moving a tab must never change
which project/account its shell belongs to. There is no single globally active project
that silently retargets every pane. No command broadcast or moving processes between
projects. One session has one writable attachment and appears in only one pane at a
time; selecting an already visible session focuses that pane instead of duplicating it.

## 2. Page structure

Entry: **Spaces**, directly after **Explore**, linking to `/-/soda/spaces`.
On Spaces, highlight that destination with `aria-current="page"`. No notification-like
badge in the native navbar: merely browsing Forgejo should not start a terminal poller.

```text
Soda    Issues  Pull requests  Milestones  Explore  Spaces
──────────────────────────────────────────────────────────────────────────────
Spaces                  5 terminals · 2 projects     Layout   + New terminal ▾
┌───────────────────────┬───────────────────────────────┬──────────────────────┐
│ Find project/terminal │ api · shell  api · build  +    │ web · dev  web · git +│
│                       ├───────────────────────────────┼──────────────────────┤
│ MY TERMINALS          │ alice @ acme/api              │ alice @ acme/web     │
│ ▾ acme/api    Running │                               │                      │
│   shell      In view  │ $                             │ $                    │
│   build      New output│                              │                      │
│   logs       Kept 24m │                               │                      │
│ ▾ acme/web    Running │                               │                      │
│   dev        In view  │                               │                      │
│   git                │                               │                      │
│                       │                               │                      │
│ OTHER PROJECTS        │                               │                      │
│   acme/docs   Stopped │                               │                      │
└───────────────────────┴───────────────────────────────┴──────────────────────┘
```

The example shows an explicitly selected two-pane layout. **Default to one large
terminal**, remember the chosen layout, and do not split automatically just because
another terminal or project exists.

### Header and visual treatment

- Reuse canonical Soda branding, existing light/dark palette, typography, borders,
  focus styles and native navigation destinations. No new component library.
- One compact workspace toolbar, not a hero, statistics cards or resource gauges.
  No invented CPU/load, command progress, branch or current-directory telemetry.
- Project/account identity stays in application chrome outside terminal-controlled
  text. A subtle active-pane outline and a text label convey keyboard focus.
- Session labels default to `Terminal 1`, `Terminal 2`, etc. Let the user rename them
  to `build`, `server` or `git`. These are labels, not assertions about running jobs.
  Do not trust shell/OSC titles as repository identity or execute labels as commands.
- Show full `owner/repository` in the focused pane header. Short tab labels are useful,
  but duplicate repository basenames must be disambiguated without relying on color.

### Navigator

Start around 260 CSS pixels wide, resizable/collapsible. It scrolls independently;
there is no second scrolling page surrounding the terminal canvas.

- **My terminals:** project-grouped sessions belonging to this Soda sign-in context,
  including kept/detached and uncertain sessions. Keep ordering stable; output must
  not reorder rows under the pointer. Expand groups with sessions initially.
- **Other projects:** authorized associated environments without one of these sessions.
  Show running/stopped/unknown and membership status. Do not enumerate every Forgejo
  repository just to offer workspace creation.
- Search matches authorized project/session labels, not shell output or filesystem
  contents. Filtering changes only the navigator, never the displayed terminals.
- A project disclosure expands its sessions. Clicking a terminal focuses it if already
  open, or opens its existing view in the focused group. This is never creation.
- Each project has an explicit `New terminal` action. The global/pane `+` names the
  selected project or opens a project picker; there is no hidden target inherited
  from whichever repository happened to be visited last. Creation needs current
  membership and a running environment. `Start environment`, `Join` and `New terminal`
  stay separate.
- Empty results distinguish **no matches**, **no sessions** and **status unavailable**.
  An authorization/network failure is not an empty project list.

### Terminal canvas

- Each pane group has its own tab strip. Groups can contain tabs from any project;
  always include the project in mixed-project tab labels.
- Layout menu: **Single**, **Side by side**, **Stacked**, **Four panes**. Recommend a
  maximum of four simultaneously visible panes initially, not a limit of four sessions.
  Extra sessions remain tabs; existing server capacity limits still apply.
- Split creates an **empty view**, with `Choose an existing terminal` and an explicit
  `New terminal in …` action. It does not clone the current shell, its cwd or its process.
- Drag a tab to another group, or use `Move to pane` from its menu/keyboard controls.
  Reuse the same renderer/attachment where possible; never End/create to move a view.
- Reducing the pane count consolidates tabs into remaining groups. It does not close,
  hide or end those sessions. Maximizing one pane is a reversible layout change.
- Tab overflow uses scrolling plus a searchable `All terminals` switcher, not tiny
  unreadable tabs or a second hidden set of sessions.
- An inactive tab may show **new output** when actually observed. No flashing badges,
  fake build-completion statuses or desktop-notification permission request. Detached
  output cannot be advertised as observed when there is no attached output stream.

## 3. Action language and consequences

| Action | Visible result | Native consequence |
| --- | --- | --- |
| Select/open existing terminal | Focus or attach that exact session | No shell creation; fresh authorization required |
| New terminal in `owner/repo` | Add a separately identified session tab | Explicit new shell in that existing project/account |
| Move tab / Split view / Maximize | Rearrange the workspace | No new session and no lifetime renewal |
| Hide tab | Remove it from the working set; keep it discoverable in navigator | Detach/finite retention, not End |
| End terminal… | Show ending, then confirmed ended or cleanup uncertain | Terminate only this managed terminal's supervised processes |
| Keep for two hours | Show actual bounded deadline | Explicit finite retention request; auth/hard limits still win |
| Continue working | Clear eligible detached retention while deliberately attached | Explicit return; no new shell |
| Stop environment… | Shared-impact confirmation and stopped state | Existing project-wide stop, affecting other people/workloads |

A tab's `×` means **Hide**, with accessible name `Hide build` and a tooltip showing
its effective retention (30 minutes by default, respecting an existing explicit
deadline and authentication cap). End is a clearly named action, not the same icon
with different behavior on different pages. Hide's first-use notice explains how
to find kept sessions again; routine tab switches should not generate toasts.

Always confirm explicit End: name the project/session and explain loss of unsaved
in-process state, while files and independently managed services remain. Do not
claim to detect an idle shell or an unsaved editor reliably. Do not offer an Undo
that resurrects a dead process. Natural shell exit is `Ended`, not necessarily success.

Stop belongs in **Project details**, never beside the terminal's everyday new/tab
controls. Its confirmation explains shared impact; it cannot claim to count all
SSH sessions or workloads from Soda terminal metadata. The project owner/operator
boundary remains distinct from ordinary members and arbitrary Forgejo site admins.

Bulk operations are secondary: explicit selection may offer `Keep selected` and
`End selected…`, with named scope and per-session results. No prominent `End all`
button or atomic-success claim. These are optional after individual controls work.

## 4. Lifetime and switching: no surprise process loss

The [terminal contract](terminal-integration.md) remains authoritative. In particular:

- Switching tabs/projects, focusing a different pane or another application does not
  end a terminal or stop its container. Nonselected open tabs remain the working set;
  visual tab switching alone must not start a detached countdown.
- Hiding/removing a tab or closing/leaving its owning document makes it eligible for
  the existing **30-minute** retention. Show the actual server deadline, not a new
  client-side timer on every render. Moving/reconnecting must not renew it implicitly.
- Native navigation/reload restores the saved working-set locators and reattaches
  existing sessions only. It cannot recreate expired ones, replay input, or clear
  abandonment deadlines just because restoration succeeded.
- After automatic restoration, a compact banner can offer **Continue this workspace**,
  explicitly naming/counting affected sessions. Apply the existing Return semantics
  to each eligible attached session and report partial failures; do not silently keep
  every listed project/session alive. A conscious single-terminal selection/Return
  acts only on that terminal. Metadata polling, output and pings never count as return.
- Leaving for longer is explicit: `Keep for two hours` shows the effective deadline,
  capped by original authentication and the existing 12-hour terminal maximum.
  Warn near expiry without stealing focus or emitting a countdown announcement every
  second. Do not promise that reauthentication rescues an expired original session.
- No automatic project stop, indefinite keep-alive, durable terminal resurrection,
  transcript storage or interpretation of shell activity as user presence.

**Discovery scope is explicit:** “My terminals” means Soda-managed sessions in the
current Soda sign-in context, including its other browser windows—not other users'
terminals, arbitrary SSH/personal tmux shells, or sessions from a separate login/device.
The same session cannot accept input from two windows. Show `Attached elsewhere`;
require detaching there before reconnecting here. Do not add a force-takeover button
or promise to focus another browser tab when the browser does not permit it. Different
terminals can be used concurrently in different windows.

## 5. State and failure design

Keep **project**, **session** and **connection** state separate. A running environment
does not imply a connected terminal; a dropped socket does not prove process exit.

| Situation | UI and permitted next step |
| --- | --- |
| Creating | Stable pending row/tab; disable duplicate submission for that action |
| Connected | Project/login visible, input enabled in the focused terminal |
| Kept/detached | Actual deadline and `Open existing` / `Keep for two hours` |
| Attached elsewhere | Readable state; no eviction, duplicate writer or new-shell fallback |
| Network lost | Keep bounded current display, disable input, explain retention; exact reattach only, no buffered keystrokes |
| Ended/expired/missing | Mark ended; explicit `New terminal` gets a new ID; never relabel recreation as resume |
| Ending | Keep the row/capacity reservation until native confirmation |
| Creation/cleanup uncertain | Keep an explicit reserved row and safe operator-inspection guidance; no retry/repair shortcut |
| Authorization lost / signed out | Stop access and clear private rendered state/locators; no stale screen usable under a replacement actor |
| Project stopped | Terminals end; explicit authorized Start does not revive shells |
| Status fetch failed | Mark observation unavailable/stale; preserve no false “zero sessions” result |

A project-specific denial or unavailable observation must not retarget or falsely
mark the other projects stopped. Clear/end the affected access as required; a whole
Soda-context logout/expiry clears all its terminals. Errors stay associated with the
project/session that produced them, rather than becoming a page-wide spinner.

Do not use a transient status sentence as End's success contract: installed evidence
already exposed the race between socket closure and the HTTP End response. Render the
session's actual ending/confirmed/uncertain state regardless of arrival order.

A lost creation reply must not be resolved by picking the newest terminal or retrying
creation. The multi-session metadata/creation contract must expose unambiguous pending
entries/locators; when this document cannot correlate one, offer the authorized existing
session list for explicit review instead of guessing. Keep all current buffer, stream,
capacity and uncertain-cleanup bounds. No durable jobs or generic recovery system.

## 6. Project management without crowding the terminals

A project menu opens a **Project details** inspector with the exact project/login in
its title. Reuse legitimate Environment and Access actions, status and callers rather
than make a second provisioning backend. It is not always visible.

On a wide screen this is a dismissible non-modal side panel that reflows the canvas;
on a compact screen it is a separate view with an explicit return to terminals.
Opening/closing it does not unmount/end those sessions. Keep repository links native,
including normal modifier-click/new-tab behavior and unsaved-form protections.

New/absent project onboarding links to the native repository's existing Create/Join
flow. Do not hide a create→join→start sequence inside `New terminal`. Zero-key Join and
outbound Git setup remain separate unfinished work, not fake-completed through this UI.

## 7. Responsive, keyboard and accessibility behavior

- Use available canvas size, not only viewport width. Aim for panes around at least
  480 CSS pixels wide with enough terminal rows; validate actual font/fit behavior.
  Below a usable split size, expose one selected group and a group switcher while
  preserving the saved split layout. Never squeeze four terminals into phone tiles.
- At roughly the existing 800px compact boundary, use one full-height terminal and
  a project/session switcher view. Terminal, project details and native repository
  navigation remain explicit; a mobile keyboard must not cover the prompt/actions.
- Keyboard-accessible tab lists, pane selection and move controls; drag is optional.
  Arrow/Home/End behavior belongs to the focused tab list, not to the shell.
- Preserve shell Escape/Ctrl-C/Ctrl-D, browser tabs/address-bar shortcuts, clipboard
  rules and the existing Ctrl+Shift+Enter escape to terminal controls. Do not bind a
  global command palette over Ctrl/Cmd+K/W/T or assume OS/browser shortcuts are free.
  Provide a visible `Switch terminal` button and review any new shortcut in browsers.
- Visible focus, text/icon states as well as color, meaningful pane/tab/action names,
  sufficient contrast and usable touch hit areas. Announce important state changes,
  not every output chunk. Reuse xterm accessibility support, not a second renderer.
- Theme/font resizing and navigator/pane resizing must fit the existing PTY correctly.
  Hidden zero-sized panes never resize sessions to bogus dimensions. Selection/paste,
  Unicode, scrollback and alternate-screen editors need real visual/input review.

## 8. Spaces and the repository drawer are one workspace

On native Forgejo pages, keep the non-modal resizable drawer and usable left page.
It presents a compact terminal tab/switcher view of the same working set, with visible
project identity and **Open in Spaces**. Multi-pane layouts belong primarily to the
full-width Spaces page; do not cram a four-pane grid into half a repository page.

Navigating to another repository never retargets an already open terminal. Returning
to Spaces restores the layout and exact surviving IDs, not new sessions. Store only
per-window UI locators/order/layout in bounded session storage. Session names are
bounded Soda-owned metadata in the existing session lifetime, not durable processes
or copied provider permissions. Never store transcripts, queued input or credentials.
Clear identity-bound UI state on logout/context replacement. A different browser
window may choose its own layout without fighting synchronized copies of UI state.

## 9. Source-backed implementation boundaries

- **Navbar placement is supported now:** selected Forgejo 15.0.7
  `templates/base/head_navbar.tmpl` calls `custom/extra_links` immediately after Explore.
  Extend the existing [override](../appliance/forgejo/templates/custom/extra_links.tmpl),
  preserving its appearance controls and upstream signed/hidden-navbar behavior.
  No new backend executable or copied navbar permission logic is required for the link.
- **The page shell is still a real integration task.** `/-/soda/spaces` remains the
  selected bounded Go/template exception. Recommend a visually consistent Soda-owned
  header with native navigation links and an explicitly identified Soda account.
  Do not clone Forgejo's notification/admin/password authority, seed native cookies,
  fabricate counts or load native JS against invented template/security context.
  Native account/settings/notifications operations remain upstream links. Exact shared
  styling/theme and header inputs need review before implementing the page; missing
  template context is not a reason to fork or scrape Forgejo.
- **Authentication:** authenticated HTML derives its actor from Soda's server session,
  not a client-supplied user ID. Provide fixed transaction-bound Spaces OAuth return;
  preserve normal CSRF, Origin, context rotation and fresh repository checks. Native
  Forgejo and Soda sign-ins can differ: show the actual Soda actor and offer explicit
  reconnection, never pretend matching styling makes their sessions identical.
- **Session model:** current `internal/web/terminal.go` keys one terminal by
  `{context, project}`. Multiple sessions require an ID-keyed bounded registry with
  immutable context/project/original-account binding, not just more frontend tabs.
  Retain one native guard/tmux service per terminal, unique IDs, one writer, global
  capacity (currently 64 including unconfirmed sessions), Stop's project gate and
  context/logout-wide cancellation. No unrelated limit increase is implied.
- **Collection:** one bounded authenticated session/environment read model supports
  both page and drawer. Filter by current authority before disclosing rows/counts;
  fresh action authorization remains separate. Reuse legitimate Soda associations
  and current Forgejo APIs; no copied role inventory, provider-data backend or host
  process scanner. Bound provider/helper fan-out; polling does not renew lifetimes.
- **Frontend:** server-rendered Go HTML plus the existing strict TypeScript/Bun and
  xterm components. Separate session ownership from view mounting through concrete
  APIs, not a second SPA, new component library or speculative plugin framework.

## 10. Review and delivery sequence

1. **Review the design with three specific screens:** two projects/six terminals in
   a two-pane workspace; the same working set on mobile; detached/ending/uncertain
   sessions with honest deadlines/actions. Use real terminal-size constraints,
   original branding and empty/permission/error states, not only a pretty happy path.
2. **Build one vertical slice:** supported Spaces link and authenticated fixed-return
   shell, authorized navigator, two terminals in one project plus one in another,
   shared drawer/session identity, explicit new/attach/hide/end. No fake tabs while
   the backend still enforces one session per project.
3. **Add and validate pane groups:** cross-project side-by-side, move/maximize/reduce
   layout, hidden-tab continuity, keyboard/touch, overflow, details inspector and
   narrow-screen behavior. Optional bulk controls come afterward.
4. **Native/browser acceptance:** six sessions across at least two projects; actual
   shell PID/start and in-memory/editor/build state preserved through tab/project/pane
   changes, drawer↔Spaces navigation, reload and temporary network loss. Confirm End
   only kills the selected service, another window cannot steal a writer, and Stop,
   logout/rotation, expiry and native safety/cleanup still win. Check another user's
   isolation and unrelated SSH/services independently. No screenshot-only acceptance.

This design work does not authorize project recreation for the separate Rocky 10.2
rollout, new fixtures/provider mutations or deployment. Those remain separately scoped.
