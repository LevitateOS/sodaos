# Spaces design: parallel work, one terminal workspace

**Revised design specification, not full implementation/acceptance or deployment
permission.** This replaces the earlier grid-first proposal and interactive mockup.
The user requested this redesign after researching existing multi-terminal/agent tools.
The [leading plan](sodaspaces-plan.md) owns scope; the [handoff](implementation-status.md)
owns execution evidence. The [Lit implementation plan](lit-migration-plan.md) now owns
the detailed source sequence for this page and its companion drawer. Steps 1–5 are
locally implemented: Lit controls, ID-keyed sessions/collection, authenticated
page/fixed return, bounded v2 layout, measured stable panes and shared page/drawer
chrome with compact projections. Current journey source ports and Go-page integration
have local fixture coverage. Steps 6a/6b now add observed attention and candidate/
extended-driver source coverage; 6c still owns native/selected-CLI acceptance. Static sheets remain design references, not
runtime evidence.

The [frontend improvement guide](frontend-improvement-plan.md) now owns the detailed
post-step-5 token, template-checking and composition cleanup. It retains this shared
Lit workspace and its existing interaction/resource contracts. The dimensions below
remain design intent; implementation must express reusable visual values through
canonical tokens and deliberate density variants, not a separate Spaces palette.

**Direction:** a project/session sidebar for finding work, tabs for switching within
a pane, and direct splits for the few terminals being viewed together. Native CLI
agents remain inside real terminals. No agent-chat frontend, worktree-per-task policy,
new application origin or new preview server.

[Visual design sheets](design/spaces/README.md) show the default desktop, cross-project
split, mobile terminal/switcher and failure states, plus the complementary
[right-half drawer](spaces-drawer-design.md) with native-forge browsing on the left.
They are static drawings with fictional content, not another functioning mockup or
screenshots of installed Soda.

## 1. Design decisions

| Keep | Change from the previous proposal |
| --- | --- |
| Spaces immediately after Explore, at `/-/soda/spaces` | Deliver inside the existing application origin; do not send users to another port |
| Shared project roots and personal native sessions | Name sessions for the work; do not create a new environment/worktree for every agent |
| Searchable project/session sidebar | One stable list with attention filtering, not separate active/kept catalogs or dashboard cards |
| Real terminal tabs and cross-project panes | Direct **Split right / Split below**, drag-resize and maximize; remove the layout-preset dropdown |
| One large terminal by default | No automatic grid as session count grows, and no arbitrary four-pane product limit |
| Hide, End and Stop are different actions | Keep End/Stop out of everyday toolbar controls; make Hide's finite retention explicit |
| Existing access/lifetime boundaries | Agent activity is advisory, separate from attachment, project state and authorization |

No automatic task naming, command execution, agent launch, agent installation, branch
selection, shared-state promotion, command broadcasting or Git-review backend is added.
The user can type any installed CLI in their shell. A session called “Auth refactor”
is a user label, not a claim that Soda understands the task.

## 2. Information model and navigation

- **Project:** the actual shared environment and its repository association.
- **Session:** one personal native terminal, immutably bound to its original project,
  account and Soda sign-in context. Several sessions may exist in one project.
- **Tab:** that session's place in a pane. One session occupies at most one pane in
  this browser window; selecting it elsewhere focuses/moves the existing view.
- **Pane:** a tab group with one selected terminal. Different panes can show different
  projects. There is no global project selector that retargets all shells.
- **Working set:** sessions deliberately open in this window, including inactive tabs.
  Kept/hidden sessions remain discoverable but are not implicitly attached by listing.

The sidebar lists **projects → sessions**; there is no extra task/workspace nesting
between them. Tabs repeat only the current pane's working set, not a second global
session inventory. User labels can change without changing IDs or native targets.

“My sessions” means this Soda sign-in context, including its other browser windows.
It does not include arbitrary SSH/personal tmux, other users, or a separate login on
another device. Another writer is shown as **Attached elsewhere**, not forcibly evicted.

## 3. Desktop composition

```text
soda    Issues  Pull requests  Milestones  Explore  Spaces       Soda: alex
────────────────────────────────────────────────────────────────────────────
Spaces  [Find a session…]                    [Next attention] [+ New terminal]
──────────────────────┬─────────────────────────────────────────────────────
All  |  Attention (1) │ api / Auth refactor   api / Review auth !   …   +
                     ├─────────────────────────────────────────────────────
▾ acme/api       ⋯   │ alex @ acme/api            Split right  Below  ⛶  ⋯
  Auth refactor      │
  Review auth     !  │  Native CLI / shell, almost all available space
  Test watcher    •  │
                     │
▾ acme/web       ⋯   │
  Checkout flow      │
  Dev server         │
  Release notes 24m  │
──────────────────────┴─────────────────────────────────────────────────────
```

### Chrome and dimensions

Use canonical Soda assets and existing light/dark palette, Barlow interface text
and Plex Mono terminal text. Reuse the shared Lit runtime; no additional UI kit,
decoration-heavy cards,
hero section, environment statistics or always-open management inspector.

Token adoption is a hard requirement across the full page and drawer. Reuse canonical
color, type, spacing and control roles; add a missing semantic role once at its shared
owner. Keep terminal-specific colors/font metrics intentional. Source guards and
both-surface theme/focus/geometry checks must demonstrate actual use. Refer to the
[token contract](frontend-improvement-plan.md#5-mandatory-token-consolidation) when the
approximate dimensions below differ from existing form-control tokens; do not blindly
inflate workspace chrome or shrink terminal text to reconcile them.

- Native-like navigation: about 56px; compact workspace bar: 44px.
- Sidebar: initially 248–256px; pointer/keyboard resize from 220–360px and collapse.
- Pane tab strip: 36px desktop; identity/action strip: 32px. These are the only
  permanent rows above the terminal. Lifecycle warnings occupy space only when needed.
- Interface text: 14px, secondary labels 12px; terminal default 14px with approximately
  20px line height. Respect font/zoom preferences, never shrink text to fit more panes.
- Thin separators, restrained selected fill and one blue focus outline. Amber marks
  attention, not focus. Text/icons accompany color. Avoid a forest of rounded cards.
- One canvas fills the remaining height. Sidebar/terminal scroll independently;
  there is no outer scrolling page or fixed-height terminal console inside a card.

The focused pane always shows **original login @ full owner/repository** in trusted
application chrome. Each mixed-project tab also carries an unambiguous project label.
Shell titles, cwd/branch escape sequences and agent text cannot replace that identity.
Do not invent branch, port, resource or working-directory telemetry.

### Sidebar

Project headers disclose sessions and have a project menu. Clicking a disclosure
only expands/collapses; it neither focuses a hidden shell nor creates an anchor shell.
Project details are reached through the menu, not by overloading the disclosure.

Rows have a primary user-chosen label and a small secondary state where useful.
Running project state belongs to the project header, not every terminal. Do not add
an “Open” badge to every healthy row. Show exceptional states and visible-elsewhere
indicators without making a selected row look like the only live process.

- Search matches authorized project/session labels, never transcripts or filesystem
  contents. It filters the sidebar only; current panes do not disappear or retarget.
- **All / Attention** is a view filter over the same list. Keep project and row order
  stable as events arrive. Do not move a row underneath the pointer.
- Attention count is the number of sessions needing attention, not an event total or
  number of connected agents. Aggregate only currently authorized, observed state.
- Hiding a tab leaves its row here with the actual deadline. No second “archive”.
- Projects without sessions remain collapsed with explicit membership/environment
  state. New environment setup stays in the native repository's existing flow.
- Empty states distinguish **No matches**, **No sessions yet**, **Nothing needs your
  attention**, and **Status unavailable**. A failed request is not a successful zero.

### Selecting, creating and naming

Selecting an already open session focuses its existing tab/pane. Selecting a kept
session requests exact authorized attach, never create. Deliberate Return applies
only to the selected eligible session, after attach; display its effective deadline
until confirmed. Automatic restoration does not perform Return.

**New terminal** opens a compact project/name chooser, not a task prompt. It shows the
full project and original account before submission. The pane `+` may preselect its
current project, but never takes an invisible target from the last native repository.
Default label is `Terminal N`; naming is optional. Name is metadata, never shell input.
Use a bounded single-line name (80 Unicode code points, no control characters),
ellipsis plus full accessible text in tight rows. Duplicate names are permitted;
stable IDs and project context still identify the action.

Creation needs current own membership and a running project. Show **Join required**,
**Environment stopped** or **Status unavailable**, with explicit appropriate routes;
do not chain create → join → start → shell. Disable only the pending creation, not the
whole workspace. A lost creation response is uncertain: no automatic retry, “newest
session” guess or replacement. Expose confirmed correlated metadata or explicit
existing-session review before further action.

## 4. Tabs, splits and focus

- **Split right / Split below** divides the focused pane into two resizable siblings.
  The new pane initially offers **Use existing terminal** or **New terminal in …**.
  Splitting alone creates no process, copies no cwd and changes no retention.
- The existing-session chooser lists full project/name and whether the session is
  already open. Choosing an open one **moves its tab**; it never creates another writer.
- Drag a tab to another tab strip to move it, or to a highlighted pane edge to split
  and move. Only valid destinations highlight. Equivalent menu actions name the
  destination pane and its selected session; drag is never the sole control.
- A move preserves ID, renderer/attachment where possible, unread state and lifetime.
  If the source becomes empty, remove that empty pane and give its sibling the space.
- **Maximize** temporarily shows one pane; **Restore panes** returns the exact layout.
  Other sessions remain open. Maximizing is not Hide, detach or process suspension.
- **Consolidate panes** moves all tabs into the focused pane, keeping that selected
  tab first and stable remaining order. It does not End/hide sessions or renew them.
- Overflow scrolls the tab strip and exposes a searchable **Tabs in this pane** menu.
  Do not keep reducing tab widths or require the global sidebar just to find a tab.
- Only the clicked/focused terminal receives keyboard input. Events never focus a
  pane, select a tab, approve a prompt, paste text or broadcast commands automatically.

Target roughly **80 columns** for a coding-agent pane where space permits. A split
may use narrower companion logs/shells, but each child needs at least **56 columns
and 12 terminal rows**, calculated from the actual font/available canvas. Disable a
split that cannot meet this, with an explanation; no fixed four-pane limit is needed.
At a 14px mono font, two comfortable 80-column panes generally need a wide desktop or
collapsed sidebar. The wide design sheet deliberately shows that case.

When resizing the window or opening details makes the saved layout unviable, show
one selected pane plus **Panes (N)** to switch. Preserve the split tree for restoration;
do not end sessions, discard groups or send zero-size PTY resizes. Split count remains
bounded by available geometry and the existing global native session capacity.

## 5. Attention without pretending to understand the agent

There are **four separate facts**: project state, terminal lifetime, connection state,
and optional reported activity. A green connection is not “agent working”; an agent's
completion report is not shell exit, successful tests or confirmed process cleanup.

| Display | Required fact | Clearing / interaction |
| --- | --- | --- |
| New output · dot | This document observed output while that tab was not selected | Clear unread marker on deliberate viewing; no semantic completion claim |
| Needs attention | Explicit terminal notification, or a supported lifecycle signal | Open the exact session; the UI never answers the prompt |
| Waiting for input · reported | A supported agent integration explicitly reported waiting | Viewing clears unread, **not waiting**; only a later valid report clears waiting |
| Working · reported | Supported lifecycle report | No signal means unknown, not idle; no spinner inferred from silence |
| Finished · reported | Supported completion report | Unread completion clears on viewing; shell remains open, no auto-End |
| Activity unavailable / last reported | Integration observation is stale or disconnected | Do not show old information as a live state or fabricate a replacement |
| Connection lost / expiring / unconfirmed | Actual Soda transport/lifetime/cleanup metadata | Opens that session's own warning and safe next action |

The first usable Spaces slice needs real session/lifecycle attention and observed
unread output. Semantic agent statuses require a separately implemented **bounded,
explicitly enabled** signal adapter. The current renderer consumes terminal OSC
controls; this design does not claim hooks or a reporting protocol already exist.
Do not install agent wrappers, change project profiles or scrape CLI prose as a shortcut.
Review exact upstream protocol/security before adding such an integration. Signals
are untrusted advisory text, length/rate bounded, tied to an authorized session—not
instructions, HTML, permission decisions, external URLs or auto-open browser actions.
No private output is copied into a durable notification/transcript store.

**Next attention** explicitly focuses the next relevant session in stable sidebar
order, wrapping once; it is disabled with an explanation when none qualify. The
Attention filter includes waiting, blocking lifecycle problems and unacknowledged
explicit notifications/completion reports. Ordinary **new output alone does not count**
or enter this filter: noisy logs must not dominate it. Reading clears notification
unread state but not waiting or a blocking problem. Do not globally reorder sessions.
Count/dot changes never steal focus or announce every output chunk. No desktop
notification permission prompt or sound by default. No poller/badge is added to the
Forgejo navbar merely because the user is browsing native pages.

## 6. Lifetime, errors and destructive actions

The [terminal contract](terminal-integration.md) remains authoritative: original
sign-in/account binding; one writer; normal 30-minute detached retention; explicit
Keep for two hours; original authentication and 12-hour hard maximum always win.
Output, polling, pings and automatic reattachment do not renew abandonment.

| Action/state | Exact design behavior |
| --- | --- |
| Switch tab/pane/project; maximize | Keep all open sessions alive within their existing authority/lifetime; no detached countdown merely for tab selection |
| Hide tab (`×`) | Tooltip/accessibility name **Hide NAME — kept until TIME**; remove view, request finite retention; retain row. If request is unconfirmed, show that, not a promised deadline |
| Page navigation/reload | Restore bounded UI locators/layout and attach exact surviving IDs. Show retained deadline and **Continue working** per session; no input replay or automatic Return |
| Keep for two hours | Explicit request in session menu; display actual capped deadline only after confirmation |
| Connection lost | Retain bounded display, disable input, show known expiry with observation age, **Reconnect existing**. No queued keys or silently launched replacement |
| Attached elsewhere | No writable terminal/retained transcript from another window; ask to detach there. No force takeover |
| End terminal… | Confirm full project/name/account and process loss. Submit End; show **Ending…** until actual native outcome. No transient message race or socket-close success assumption |
| Ended / expired | Clear live controls; explicit **New terminal** creates a new ID. No “Undo”, “Resume” or restart masquerading as continuity |
| Creation/cleanup unconfirmed | Keep the capacity-reserving row; input/attach/automatic retry disabled. Explain operator inspection. No repair, replacement or freeing a slot based on a closed socket |
| Capacity reached | Refuse new creation without evicting anyone. Let the user review their own sessions and explicitly End one; do not expose other users' metadata or retry in a loop |
| Required native support missing/refused | Report the refusal and operator next step. No install-on-Open, fallback PTY or project replacement |
| Authorization lost / Soda logout | Cancel affected access and clear private renderers/locators. Whole-context loss affects all its sessions; project denial is not another project's stop |
| Stop environment… | Only in Project details, authorized owner/operator, shared-impact confirmation. Start uses the same retained root but cannot restore ended shells |

End confirmation copy: **“End ‘Auth refactor’ in acme/api?”** Follow with: “This ends
this terminal and processes in its managed session. Unsaved in-process work will be
lost. Files and independently managed services remain.” Default focus is **Cancel**.
No guessing whether an editor is dirty and no generic red `×` that sometimes means End.

Session menu: Rename; Move to pane; Keep for two hours; Continue working when eligible;
Hide; divider; End terminal…. Pane menu: Split right/below; Maximize/Restore;
Consolidate panes. Project menu: Open repository; Project details. Separate ownership,
not one huge menu. Bulk Keep/End, floating windows and stacked-pane mode are not part
of this first design; they can be reconsidered after core interactions work.

## 7. Mobile, compact drawer and accessibility

At about 800px and below, show **one full-width terminal**, never a scaled desktop grid.
Use a compact native-like header, a **Sessions** button with current project/session,
and a small overflow menu. Touch targets are at least 44px. Pane selection goes into
**Panes (N)** only when more than one pane exists; no extra always-visible toolbar.

**Sessions** opens a full-height navigation view with the same search, project grouping
and Attention filter. It is a view change, not Hide, detach or a modal over a live prompt.
Selecting returns to the exact terminal. Explicit Back returns without changing the
selection. Project details uses another view with its own Back action. Keyboard focus
returns to the invoking control unless the user deliberately selected a terminal.

The terminal fits the visual viewport when a mobile keyboard opens; no fixed bottom
bar covers the prompt. Keep at least the identity, return control and input region
usable. Validate real device keyboards, orientation and selection; the sheets do not
prove those behaviors. Browser zoom must not silently become session/font resizing.

The **repository drawer** is this workspace's compact surface; its
[dedicated design](spaces-drawer-design.md) specifies the 50/50 native-forge/terminal
composition, viable 35–65% resizing, all-open-session tab strip, temporary session
switcher, management access and compact Forge/Terminal behavior. It derives its flat
tabs from this window's pane tree, keeping the full-page layout intact; it is not a
second session inventory. One terminal is visible, with **Open in Spaces** for full
arrangement. Native browsing never retargets shells; same-document view switches,
real navigation and explicit Hide have deliberately different lifetime effects.

Use actual tablist/tab/panel semantics, roving keyboard focus and accessible pane
names. Arrow/Home/End navigate a focused tab strip, not the shell. Separators have
keyboard resize controls and visible focus. Keep the existing Ctrl+Shift+Enter escape
from xterm to controls; preserve Escape, Ctrl-C/D, browser Cmd/Ctrl-K/W/T and clipboard
behavior. Every palette/switch/move action has a visible button; do not assume another
global shortcut is available. Use xterm accessibility support, not a second renderer.

## 8. Page, data and implementation boundary

The Spaces handler is Soda-owned Go/template HTML at `/-/soda/spaces` behind the existing
proxy—on the isolated deployment's **33443** origin, not a new listener or tunnel.
The official Forgejo 15.0.7 `custom/extra_links` hook places the link after Explore;
preserve its existing appearance controls and native hidden-navbar behavior.

The page shell uses canonical branding, fixed configured-origin native navigation
links and an explicitly labelled **Soda account** derived server-side. It does not
render fake Forgejo notification/admin/account context or load native scripts against
invented CSRF data. Native workflows remain upstream links and pages. Use existing
Soda/system appearance inputs honestly; do not silently promise synchronization with
another Forgejo sign-in's preferences. This is a bounded page, not a second frontend
framework or copied Forgejo password/permission authority.

Preserve the implemented transaction-bound **fixed Spaces OAuth return**, actor/CSRF/Origin,
PKCE, encrypted grants, context rotation/logout and fresh repository authorization.
A native-only sign-out is not atomic Soda/SSH logout; styling never proves identity.

The backend now implements a bounded ID-keyed registry with immutable original
bindings; the historical `{context, project}` singleton is superseded. Preserve
that actual concurrency owner and the current 64 global slots including
uncertain sessions, one private native guard/tmux server per ID, Stop's project gate
and context-wide cancellation. No native multiplexer/backend replacement is selected.

One bounded authorized environment/session collection serves page and drawer. Filter
before disclosing rows/counts; keep degraded observation distinct from launch authority.
Bound provider/helper fan-out and distinguish unavailable enumeration from empty data.
Do not copy provider roles or scan arbitrary project processes. Session labels/signals
are bounded Soda metadata within the original session lifetime, not durable jobs.

Store only bounded per-window session locators, order, pane ratios/selection and
sidebar preference. No credentials, transcripts, queued keys or cross-window layout
synchronization. Authorization loss invalidates private UI data. Metadata refresh
cannot recreate terminal renderers, replace IDs or become a keep-alive mechanism.

## 9. Implementation and review acceptance

Follow the single [Lit sequence](lit-migration-plan.md#4-ordered-implementation-slices).
The rendering ports, ID-bound concurrency/collection, authenticated page/shared
drawer and direct layouts are locally implemented through step 5. Apply the
[frontend cleanup](frontend-improvement-plan.md#8-implementation-order-and-exits)
without repeating them; step 6 retains actual attention and candidate/native closure.
The native helper/tmux owner is preserved. Required template checks, typed composition
and canonical token adoption are part of source acceptance, not optional polish.

Review must show both surfaces working against real authorized sessions, with the
original identity, measured readable panes, stable renderer/attachment lifetime and
no native effects from layout. Real unread/lifecycle states precede any explicitly
selected semantic signal adapter; no fake statuses or additional preview origin.

Native acceptance under applicable scope still requires six sessions/two projects:
unchanged PID/start and shell memory/editor/build state through navigation, reload,
layout, drawer and network loss; End preserves siblings/unrelated SSH/services;
duplicate writer, Stop, logout/rotation, expiry and uncertain cleanup win. Real
editor/paste/history/resize, mobile keyboard and both themes need review. See the
implementation plan's source/browser/native matrix, not a second progress checklist.

Static sheet rendering is design review only. It does not resolve the outstanding
native probe, authorize project recreation for Rocky 10.2, change retained services or
count as full product acceptance.

## 10. Research basis

Official pages/docs and published interface images were inspected; the products were
not installed or tested. Retained research is in `.artifacts/research/multi-terminal-ui/`.
Borrow interaction patterns, not implementation code, branded assets or business rules.

- [cmux](https://www.cmux.dev/), [groups](https://www.cmux.dev/docs/workspace-groups),
  [notifications](https://www.cmux.dev/docs/notifications): sidebar navigation,
  visible splits and attention. Do not copy its group-anchor creation/deletion effects.
- [Superset terminal](https://docs.superset.sh/terminal-integration) and
  [activity](https://docs.superset.sh/agent-status): pane tabs, move/merge and explicit
  lifecycle signals. Worktree-per-task and orchestration are not Soda requirements.
- [Agent Deck](https://github.com/asheshgoplani/agent-deck): grouped/searchable session
  overview and quick attachment rather than showing every terminal at once.
- [Wave](https://www.waveterm.dev/): clear block focus, direct layout and magnify.
  Browser/editor/widget platform is not selected.
- [Zellij](https://zellij.dev/features/): useful pane ergonomics and stacking; no
  replacement of managed tmux, resurrection, floating-window or plugin subsystem.
- [Conductor](https://www.conductor.build/): task organization is useful; central
  agent chat, cloud sandboxes and replacement review UI are deliberately not copied.
