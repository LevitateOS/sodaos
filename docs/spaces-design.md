# Spaces design: parallel work in one Project OS workspace

## Scope of the selected extension

The terminal layout below remains the implemented baseline. Extend this same
workspace for [Project OS profiles](project-os.md#selected-environment-profiles),
KDE desktops and [AI run views](services-and-ai-plan.md); do not build a second UI
or separate development environment. Product order belongs to the
[leading plan](sodaspaces-plan.md#extension-order-and-dependency-boundaries).

- Profile selection belongs to explicit project creation. It never retargets an
  existing terminal, upgrades a root or installs a desktop on Open.
- Terminal and Desktop identify the same project/account and use its real files,
  shared tools and permissions. KDE keeps terminal access available.
- A desktop tab references an exact graphical session, with its own authorized
  display/input owner. It is not a PTY or an ID in the existing personal-terminal
  API. Reuse page/drawer layout and navigation without pretending native protocols
  or lifetimes are interchangeable.
- AI run views identify the issue/PR attempt and execution owner. Personal "My
  sessions" semantics below remain unchanged; authorized run views need explicit
  discovery/attachment and must not appear through a fabricated personal login.
- Move between Spaces and the drawer without creating another native session.
  Observe/control, desktop End, terminal End, project Stop and job Cancel remain
  explicit distinct actions. Output and mere connection do not prove AI status.
- Fit desktop content with its actual display geometry; terminal cell minima and
  PTY resize rules below apply only to terminals. On compact screens, provide one
  usable selected view, accessible navigation and deliberate input focus.

Desktop and AI views remain unimplemented. Existing terminal diagrams, tests and
native evidence do not establish graphical/session or automation behavior.

## Existing terminal design and evidence

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

**Direction:** a focused first-use journey, then a project/terminal sidebar for
finding work, tabs for switching within a pane, and contextual splits for the few
terminals being viewed together. Native CLI
agents remain inside real terminals. No agent-chat frontend, worktree-per-task policy,
new application origin or new preview server.

[Visual design sheets](design/spaces/README.md) show the default desktop, cross-project
split, mobile terminal/switcher and failure states, plus the complementary
[right-half drawer](spaces-drawer-design.md) with native-forge browsing on the left.
They are static drawings with fictional content, not another functioning mockup or
screenshots of installed Soda.

## First-use journey — selected 13 September 2026

**Design selected; current presentation requires visual redesign.** The user selected the focused welcome
concept and walked through repository selection, configuration, creation, joining,
first terminal and working workspace. The [journey implementation plan](sodaspaces-plan.md#first-use-journey-implementation-plan)
owns execution order and status. This section owns the resulting presentation.
The pasted UX proposal supplied context; the agreed journey below is the selected
scope, not automatic adoption of every suggestion in that proposal.

### Visual direction and scope

Keep the native Forgejo header and current Soda style: canonical dark/light tokens,
Barlow interface type, Plex Mono controls/terminal type, square geometry, thin rules,
restrained red primary actions, quiet secondary links and clear whitespace. Reuse
canonical repository and terminal icons; every repository lives on **this Forgejo**.
There is no GitHub branding, external provider picker or clone-URL onboarding field.
Generated images illustrate hierarchy, not exact CSS, supported data or new tokens.

Before the first project, show the page title **Spaces** and subtitle **Your projects
and terminals, together.** One unbordered content panel contains the welcome and setup
steps. Show no empty sidebar, Sessions toolbar, terminal controls, pane dropdown,
Attention filters, drawer controls or SSH forms. Forms may scroll on short screens;
the working terminal later fills the available visual viewport.

The first three setup states replace content inside that panel. They are views in
the existing native Spaces page, not new application origins or modal overlays.
Back/Change preserve nonsecret selections during the active flow and never mutate
resources. Ordinary reload re-reads authorized state; it never replays submission.

### Page compositions

The layout foundation uses the following dimensions, expressed through existing
Soda font, color, spacing and control roles. Spaces-specific width/inset roles live
with the shared definitions in `assets/branding/forgejo/components.css`.

| Role | Centered setup | Working workspace |
| --- | --- | --- |
| Width | Outer frame up to 1240px, centered; fluid 16–48px page inset | Full native page width; bypass the ordinary 1120px content container |
| Reading measure | Welcome up to 560px; repository/configuration forms up to 720px | Project identity in the main header; terminal uses the remaining width |
| Vertical composition | Title/subtitle above one unbordered frame; centered welcome, aligned setup forms and quiet orientation footer; scroll on short screens | Sidebar starts beside the project header and extends to the workspace bottom; canvas fills remaining height |
| Heading scale | Soda title role, fluid 28–40px; 16px body and 14px supporting text | 28px project title role; 22px sidebar heading; existing dense terminal chrome |
| Spacing and rules | 16px title/frame separation, fluid 16–32px frame inset, no outer rule | 80px desktop header with 12px/16px padding; canvas has no outer margin or enclosing border |
| Action hierarchy | Red primary with 44px minimum target; outlined secondary actions and quiet navigation/help | Red New terminal, outlined Project settings; contextual controls keep their dense sizing |

The frame, project-controls hosts and terminal owners remain mounted when setup is
entered or cancelled. Page layout rules do not change the native drawer density.
The outer native content container must not add left/right margins, even when native
styles load after Spaces. This removes the outer gutters, not the Projects sidebar.
The existing 256px default sidebar and 220–360px user resize range remain intact;
compact presentation still follows measured terminal viability. On compact widths,
project identity leads the header and controls wrap without horizontal overflow.
The welcome, setup, post-creation and working-terminal compositions are specified
below. Local implementation and user visual acceptance remain separate evidence.

### Welcome composition

The confirmed no-project state follows the selected focused-welcome reference:
a centered group inside the existing wide frame, with a restrained 64–96px wireframe
cube, a small Soda-red facet, **Create your first project**, and the two-sentence
explanation below. Each sentence starts on its own line at desktop reading widths
and wraps naturally on smaller screens. The illustration is decorative vector
geometry, hidden from assistive technology and colored through Soda's existing roles.

The only button is **Create project**: solid Soda red, at least 224px wide and 52px
high, with a decorative plus. It opens Forgejo repository selection without creating
any resource. **How Spaces works** is a muted underlined help link with an announced
new-tab destination. The heading, primary action and help link have deliberate
spacing; no extra card, shadow or animation competes with them.

An inset rule separates the numbered, noninteractive orientation list from the
welcome. It reads **01 Choose a repository → 02 Create a project → 03 Open a terminal**,
with only the first number accented. The list stacks at narrow widths; it is not a
progress indicator or additional navigation. Loading and failed inventory retain
their recovery states and do not show a guessed welcome. All workspace controls
remain hidden and absent from the welcome's keyboard and accessibility navigation.

### State sequence

| State | Content and primary action | Confirmed transition |
| --- | --- | --- |
| 1. No projects | **Create your first project**; “A shared development system, connected to your repository. Open terminals and work together, right in your browser.” Primary **Create project**; quiet **How Spaces works** help link | Opens repository selection; no project is created |
| 2. Choose a repository | **Choose a repository**; search and a bounded list of eligible Forgejo repositories showing full owner/name; explicit radio selection; **Continue** becomes available after selection | Opens configuration for that exact repository ID |
| 3. Configure project | Selected repository with **Change**, **Project OS** selector, optional unchecked **Enable project Tailnet** when available, primary **Create project** | Submits explicit creation once |
| 4. Creating project | Retain repository identity and selected options; **Creating project…** with an indeterminate busy indicator and announced status; prevent duplicate submission | Read confirmed result and current project state; an uncertain response is not success |
| 5. Project created, not joined | Replace setup panel with project workspace; selected project in sidebar, project name/status and secondary **Project settings**; center **Project created**, explanatory account copy and primary **Join project** | Explicitly creates/attaches this user's membership; no terminal creation |
| 6. Joined, no terminals | Same workspace; terminal outline icon, **Open your first terminal**, “Your account is ready in owner/repository. Open a browser terminal to start working.” Primary **New terminal**; quiet **Project account: login** | Explicitly creates one terminal in the displayed project/account |
| 7. Opening terminal | The workspace shows **Opening terminal…** for the exact pending entry; prevent duplicate clicks | A confirmed terminal attaches and receives deliberate keyboard focus |
| 8. Working | Main canvas is a usable terminal; its tab and sidebar row appear; **New terminal** moves to the project header beside secondary **Project settings** | First-use happy path complete when the user can type and work |

The mockup's “Connected as” helper becomes **Project account: login** in implementation:
membership readiness must not imply an established terminal connection. Use actual
repository/account names, installed profile labels and observed state throughout.

The quiet setup footer reads **Choose a repository → Create a project → Open a
terminal**. It is a high-level orientation aid, not a claim there are only three
backend operations. It disappears with the setup panel after creation. Join remains
an explicit workspace action. No percentage, countdown or completion stage is
invented when the backend supplies only a pending outcome.

### Repository selection and configuration

Both steps share the same 1240px maximum outer frame, a 720px form measure and
identical header/action tracks. The setup frame has a stable 640px height; tall
viewports must not stretch the gap between fields and actions. The Back row and **New project** heading block retain their
position between steps. Longer repository results or configuration fields scroll
within the form, with space reserved for the primary action and orientation footer.
On short screens the surrounding setup can also scroll; no action is clipped under
the footer.

Selection uses contiguous 64px minimum rows, a native radio indicator, neutral fill
and a slim red leading edge. The search input and Search action share one row.
Pagination appears only when there is another page to visit; bounded-search limit
notices remain visible independently. The selected repository becomes a matching
identity row in configuration, with its Forgejo owner/name and a quiet **Change**
action. **Back**, **Change** and **Cancel setup** remain low-emphasis text buttons;
**Continue** and **Create project** share the right action edge and a 224×52px
minimum desktop target. Journey primary actions use the canonical prominent Plex
Mono role (16px/24px); secondary controls retain their quieter density. Back aligns
with the outer panel inset. On compact screens, reduce footer and field spacing so
the ordinary configuration helper and two repository rows remain visible; longer
results and optional fields retain their bounded scrolling.

Use the existing orientation footer as the sole step indicator: repository selection
is current first, then confirmed selection receives a check and project configuration
becomes current. This is form progress, not provisioning progress. The ready form
omits routine status/retry controls. Loading, denied and uncertain outcomes retain
announced feedback and recovery; pending creation says **Creating project…**, retains
the selection and disables navigation without a percentage or extra dashboard.

- Search only the current Forgejo's authorized repositories. Visibility and creation
  eligibility are different facts. The [environment API](dashboard-api.md) owns
  permissions; the current create caller requires the human repository owner and
  refuses organization-owned creation. The illustrative `acme/api` image does not
  expand that authority. Use an eligible personal repository in acceptance fixtures.
- Use stable repository IDs across search, Back, Change and submission. Ignore stale
  search results. Selection has a radio indicator and shape/background treatment,
  with color as additional emphasis. No repository is silently preselected on entry.
- **Create a new repository** opens Forgejo's existing native creation flow. Returning
  to Spaces refreshes the list and allows deliberate selection; no invented callback
  route, automatic project creation or write-capable token is part of this flow.
- Where a visible repository already has an authorized project, offer **Open project**.
  An incomplete reservation offers inspection/recovery, never another Create.
  Do not expose inaccessible project existence or count through picker metadata.
- Project OS defaults to the available Rocky terminal profile and lists only installed,
  compatible choices. Its labels and availability come from the [Project OS owner](project-os.md#selected-environment-profiles).
  No free-form image, architecture/version guess, unsupported desktop choice or extra
  CPU/RAM configuration is introduced by these mockups.
- Tailnet remains optional and unchecked. Availability, admission, selection binding
  and outcomes use the [existing Tailnet contract](tailnet-integration-plan.md).
  Hide unavailable options; if availability changes after selection, explain and
  require the user to review the choice rather than silently changing submitted intent.
- **How Spaces works** points to existing user help, refined as needed. Browser terminal
  onboarding does not request SSH keys. External SSH remains optional under Access.

### Project workspace and contextual actions

After confirmed creation, show the sidebar with the new project selected and a quiet
**Create project** link. Defer project search for the initial one-project/no-terminal
state; show search when there is a collection worth filtering. Returning users with
existing projects enter their authorized workspace directly, bypassing welcome.
Starting another project reuses the setup flow without destroying existing terminal
owners or the saved pane tree; cancel returns to the previous workspace.

The sidebar starts beside the header and extends to the workspace bottom. Its
Projects heading includes the authorized visible-project count. The selected row
uses a repository icon, normal-case owner/name, a status label beneath it, neutral
fill and a slim red leading edge. Compact screens expose this navigation through
**Projects** in the header after creation.

The header gives owner/name the Soda title role, with an inline status dot and text,
then the validated installed profile label beneath it. Running uses the existing
success color; stopped uses the danger role. Missing observation or unavailable
authority/native state reads **Status unavailable**, never an inferred stopped state.
Long identities truncate visually with the full name retained in the title and
accessible text. **Project settings** stays outlined at the right on wide screens.

**Project created / Join** and **Open your first terminal** share one centered
composition in the remaining workspace height: a 560px maximum reading measure,
a 128×104px decorative illustration area (104×85px on compact screens), 16px to
the heading and explanation, 24px before the primary action and 16px before the
helper. The primary action is at least 224×52px. A cube with a success check marks
confirmed creation; a terminal outline with a red cursor marks account readiness.
Both use Soda's existing type and color roles. Short screens scroll this content
without clipping the action; the project header stays in place.

The Join explanation names the actual Forgejo repository and personal account setup.
The first-terminal helper says **Project account: login**. Routine success prose,
Refresh, Back to workspace, Workspace options and pane controls are absent from the
healthy prompt. Pending Join and uncertain/rejected outcomes retain announced
feedback and explicit recovery. The header and prompt geometry remain aligned
across a successful Join; membership does not create a terminal automatically.

Project selection changes navigation context only: it never retargets existing shells. Mixed-project
tabs/panes keep their own trusted original identities. The primary action is
appropriate to the selected project's readiness; pending/denied state never leaves
a mysterious enabled `+`. **New terminal** appears once in the empty workspace and
moves to the header when there is a terminal. Naming is optional through Rename
after creation; no extra naming form interrupts the selected-project happy path.

**Project settings** is a secondary view with Back and **Overview / Access / Network**.
Overview contains authorized lifecycle actions; Access contains account information
and **External SSH — Optional**; Network reuses the current Tailnet view. Keep Stop
out of the work header and retain its shared-impact confirmation. Tab and pane menus
hold Rename, split/move, Hide and End with the distinctions in sections 4 and 6.
Drawer/Attention controls remain contextual and are absent from first-use setup.

The working canvas fills the remaining width and height without an enclosing
margin or border. Keep the functional sidebar and pane resize separators. Its screen has 16px desktop padding and 8px compact padding, with the
account label and tab icon sharing the text grid’s left edge. Xterm fitting and
minimum-size measurement include these insets. The short-name sidebar project card
is approximately 67px high; long names can grow without clipping status. **New terminal** is the red header action; **Project settings** remains
outlined. Tabs fill their strip, using a terminal icon, normal-case display name,
neutral selected fill and a red top edge. Sidebar terminal rows mirror the icon and
name, mark the currently selected entry and retain genuine attention/status text.
A single terminal does not need a tab picker or a Move affordance.

**Pane** contains Split right/below; Maximize/Restore and Consolidate appear once
multiple panes exist. Split admission still follows the measured terminal minima.
**Move** appears when another pane or tab-order position exists and names the
selected terminal. **Tabs** offers the existing searchable picker for multiple tabs.
These contextual controls preserve the original entry and pane targets.

The terminal's own **⋯** menu stays attached to its account/project identity strip,
inside its existing stable owner. It names the terminal, contains Rename and Hide,
keeps the original project's settings available for mixed-project panes, and separates
**End terminal…** with a rule and danger text. End retains its named, Cancel-first
confirmation. Opening menus never recreates a renderer, connects a replacement or
writes to the backend. Native drawer density and xterm's screen/ANSI styling remain
owned by their existing mechanisms.

### Required branches and responsive behavior

| Condition | Presentation and recovery |
| --- | --- |
| Inventory loading / unavailable | Loading or **Could not load projects** with read-only retry; never flash the welcome as a guessed empty result |
| No repository matches / none eligible | Distinct helpful messages, clear search reset or native Create repository link; state permission limits without leaking hidden repositories |
| Ownership or session changes | Refresh authorization; remove stale choices and protect drafts/private context under the existing auth contract; no automatic mutation after reconnect |
| Creation rejected before reservation | Explain the affected field/admission error and allow correction plus an explicit resubmission |
| Creation outcome uncertain / reservation incomplete | Preserve known project identity, inspect current state and show operator guidance where necessary; no automatic retry or root recreation |
| Project stopped | **Start project** only for an authorized actor, otherwise administrator guidance; confirmed Start is followed by Join or New terminal according to membership |
| Project running, user not joined | **Join project** with account-setup explanation; pending/error stays scoped here and never falls through to terminal Create |
| Already joined, no visible terminal | First-terminal copy only when the inventory confirms none; discover existing/hidden terminals before offering unnecessary new creation |
| Terminal Create pending / uncertain / refused | Retain exact pending locator and scoped outcome; recover using the terminal contract, not a new ID or repeated request |
| Disconnected / attached elsewhere | Recovery belongs to the exact terminal; **Reconnect terminal** or other-window guidance, not another New terminal CTA |
| Optional Tailnet setup incomplete | Show network status under Project settings; successful project creation remains successful and is never repeated to repair network observation |

At desktop width, setup stays a centered readable single-column form; workspace uses
the project sidebar. Around 800px, retain a narrower sidebar only while the selected
surface remains usable. Around 640px and at 390px, use a labeled **Projects** navigation
view and one full-width selected terminal. Measured geometry decides split projection,
not a fixed pane count. Preserve layout when returning to a wider screen.

Use keyboard-operable repository radio rows, real labels, visible token-based focus,
announced pending/error states and 44px touch targets. Back restores sensible focus;
successful user-requested terminal creation focuses the shell, while background
refresh never steals focus. Check both themes, long repository names, zoom, short
viewports and mobile keyboards. These requirements complement section 7.

### Interaction and recovery finish

Primary hover/press uses Soda's primary-hover role; neutral actions, selected rows
and open disclosures use the shared neutral surface. Keyboard focus has a visible
outline on buttons, links, fields and disclosures, and repository radio focus also
outlines its whole row. Disabled controls remain visibly subdued and inert. Back
restores its original visible invoker or a sensible control in the replacement view;
background refresh does not request focus.

Only one contextual menu stays open at a time. Escape returns focus to its disclosure;
Tab departure and clicks elsewhere dismiss it. The opened menu's scroll height is
bounded by the viewport and its owning terminal/canvas. Long terminal names remain
in accessible text and tab/title tooltips; menu headings use at most two visible
lines so actions remain reachable. Short viewports reduce header/canvas insets while
keeping terminal geometry and compact projection under their existing owners.

Inventory loading and failure, stopped projects and incomplete/unavailable project
state reuse the centered illustration, title, explanation and explicit action
composition. Loading uses a static clock and explanatory copy rather than simulated
progress; inventory loading exposes no mutation controls or guessed setup footer.
Only authorized stopped projects offer Start. Missing authority/observation offers
read-only recovery, not a guessed lifecycle state.

Pending operations use neutral feedback panels; failures and uncertain outcomes use
canonical warning background, border and text roles. Recovery keeps the affected
project or exact terminal in context. Configuration and workspace notices scroll
within bounded areas on short screens. Terminal reconnect feedback follows the
existing detach behavior: it preserves identity and offers reconnect without claiming
that the renderer or output buffer survives a lost connection. No motion is required
to understand loading, selection or recovery.

Join errors replace the success illustration and heading with **Couldn’t join
project**. Explain the affected account setup, saved project access or unsupported
username in plain language using the API's known error code. Unknown responses
say that joining could not be confirmed; never guess a specific underlying cause.
Do not show project-reservation or write-replay terminology after Join.
**Check join status** performs only reads. Confirmed membership advances to the
first-terminal screen; confirmed absence makes **Try joining again** available as
a separate quiet action. Account-setup failures direct the user to their Soda
administrator before retrying. A refresh is never presented as repairing an account.

### Selected visual references

These generated mockups and their sibling prompt files are local ignored design
artifacts, not installed screenshots or tracked product assets. This written design
is sufficient when those local files are unavailable in another checkout.

| State | Local reference |
| --- | --- |
| Focused welcome, selected | [01-focused-welcome.png](../.artifacts/design/spaces-empty-state/01-focused-welcome.png) |
| Choose repository, corrected neutral icons | [03-choose-repository.png](../.artifacts/design/spaces-empty-state/03-choose-repository.png) |
| Configure project | [04-configure-project.png](../.artifacts/design/spaces-empty-state/04-configure-project.png) |
| Created / Join | [05-project-created-join.png](../.artifacts/design/spaces-empty-state/05-project-created-join.png) |
| Joined / first terminal | [06-open-first-terminal.png](../.artifacts/design/spaces-empty-state/06-open-first-terminal.png) |
| Working terminal, new reference | [07-working-terminal.png](../.artifacts/design/spaces-empty-state/07-working-terminal.png) |
| Terminal menu open, new detail | [08-terminal-menu.png](../.artifacts/design/spaces-empty-state/08-terminal-menu.png) |
| Pane menu open, new detail | [09-pane-menu.png](../.artifacts/design/spaces-empty-state/09-pane-menu.png) |

The alternate empty-sidebar concept (`02-projects-sidebar.png`) was not selected.
The [local journey gallery](../.artifacts/design/spaces-empty-state/journey-gallery.html)
shows the five selected references plus the new working view and two menu details.
The working view preserves the previous screen's project/sidebar geometry, moves
New terminal to the header and fills the main area with a single terminal. A tab
ellipsis exposes Rename, Move to pane when applicable, Hide and separated End;
the Pane menu exposes applicable split/layout actions. Only one menu is open in
each detail. These new images are review references, not an implementation or
acceptance receipt. Their [prompt set](../.artifacts/design/spaces-empty-state/07-09-working-terminal-prompts.md)
records the built-in imagegen inputs.

Creating/Opening remain specified transient states without separate image references.
Older layout sheets remain supporting references for multi-terminal geometry,
subject to this selected control hierarchy. Browser reviews must compare the actual
composition, scale, spacing and emphasis with these references, not only check for
the presence of their labels and buttons.

## 1. Design decisions

| Keep | Change from the previous proposal |
| --- | --- |
| Spaces in native navigation, at `/?soda-view=spaces` | Deliver inside the existing application origin; do not send users to another port |
| Shared project roots and personal native sessions | Name sessions for the work; do not create a new environment/worktree for every agent |
| Searchable project/terminal sidebar after setup | One stable list with contextual attention filtering, not a competing Sessions destination |
| Real terminal tabs and cross-project panes | Direct **Split right / Split below**, drag-resize and maximize; remove the layout-preset dropdown |
| One large terminal by default | No automatic grid as session count grows, and no arbitrary four-pane product limit |
| Hide, End and Stop are different actions | Keep End/Stop out of everyday toolbar controls; make Hide's finite retention explicit |
| Existing access/lifetime boundaries | Agent activity is advisory, separate from attachment, project state and authorization |

This existing workspace implementation adds no automatic task naming, command
execution, agent launch, agent installation, branch selection, shared-state
promotion, command broadcasting or Git-review backend. The subsequently requested
[repository AI automation](services-and-ai-plan.md#live-ai-terminals-in-spaces-and-the-drawer)
selects predefined agent launches and live terminals as separate, unimplemented
work, with run ownership independent of a viewer's sign-in context.
The user can type any installed CLI in their shell. A session called “Auth refactor”
is a user label, not a claim that Soda understands the task.

## 2. Information model and navigation

- **Project:** the actual shared environment and its repository association.
- **Terminal** (internal/API **session**): one personal native terminal, immutably bound to its original project,
  account and Soda sign-in context. Several sessions may exist in one project.
- **Tab:** that session's place in a pane. One session occupies at most one pane in
  this browser window; selecting it elsewhere focuses/moves the existing view.
- **Pane:** a tab group with one selected terminal. Different panes can show different
  projects. There is no global project selector that retargets all shells.
- **Working set:** sessions deliberately open in this window, including inactive tabs.
  Kept/hidden sessions remain discoverable but are not implicitly attached by listing.

The sidebar lists **projects → terminals**; there is no extra task/workspace nesting
between them. Tabs repeat only the current pane's working set, not a second global
session inventory. User labels can change without changing IDs or native targets.

“My sessions” means this Soda sign-in context, including its other browser windows.
It does not include arbitrary SSH/personal tmux, other users, or a separate login on
another device. Another writer is shown as **Attached elsewhere**, not forcibly evicted.

## 3. Desktop composition

```text
soda    Issues  Pull requests  Milestones  Explore  Spaces       Soda: alex
────────────────────────────────────────────────────────────────────────────
Projects              │ acme/api · Running    [New terminal] [Project settings]
[Search projects…]    ├─────────────────────────────────────────────────────
                      │ api / Auth refactor   api / Review auth !       …
                      ├─────────────────────────────────────────────────────
▾ acme/api           │ alex @ acme/api                                  ⋯
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
- Thin separators, restrained selected fill and the canonical visible focus outline. Amber marks
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
Project settings are reached through the header or menu, not by overloading the disclosure.

Rows have a primary user-chosen label and a small secondary state where useful.
Running project state belongs to the project header, not every terminal. Do not add
an “Open” badge to every healthy row. Show exceptional states and visible-elsewhere
indicators without making a selected row look like the only live process.

- Search matches authorized project/session labels, never transcripts or filesystem
  contents. It filters the sidebar only; current panes do not disappear or retarget.
- **All / Attention** is a contextual view filter over the same list, exposed when
  relevant terminals exist rather than in the empty setup. Keep project and row order
  stable as events arrive. Do not move a row underneath the pointer.
- Attention count is the number of sessions needing attention, not an event total or
  number of connected agents. Aggregate only currently authorized, observed state.
- Hiding a tab leaves its row here with the actual deadline. No second “archive”.
- Projects without terminals remain selectable with explicit membership/project
  state. Global project setup follows the first-use journey above; native repository
  entry reuses the same creation owner with that repository already identified.
- Empty states distinguish **No matches**, **No terminals yet**, **Nothing needs your
  attention**, and **Status unavailable**. A failed request is not a successful zero.

### Selecting, creating and naming

Selecting an already open session focuses its existing tab/pane. Selecting a kept
session requests exact authorized attach, never create. Deliberate Return applies
only to the selected eligible session, after attach; display its effective deadline
until confirmed. Automatic restoration does not perform Return.

**New terminal** in a selected project explicitly creates in that displayed project
and original account, with default label `Terminal N`; it opens without a second
project/name form. Naming remains optional through Rename. A contextual action with
no unambiguous project/account target must first present a compact project chooser;
it never takes an invisible target from the last native repository or another pane.
Name is metadata, never shell input.
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

The [terminal contract](terminal-integration.md) owns native lifetime, bounded
attachment authority, one-writer exclusion, allocation, lookup and capacity. Design
sheets do not preserve the superseded browser retention/lease/receipt requirements.

| Action/state | Exact design behavior |
| --- | --- |
| Switch tab/pane/project; maximize | Preserve live owners; presentation is not native lifecycle |
| Hide tab (`×`) | **Hide NAME** removes the view, not the native work; no lifetime request |
| Page navigation/reload | Restore current UI locators/layout and inspect/attach exact surviving IDs; no input replay or new shells |
| Connection lost | Retain bounded display, disable input, show observation age and **Reconnect terminal**; no queued keys or silent replacement |
| Attached elsewhere | No writable terminal/retained transcript from another window; ask to detach there. No force takeover |
| End terminal… | Confirm full project/name/account and process loss. Submit End; show **Ending…** until actual native outcome. No transient message race or socket-close success assumption |
| Native ended / absent | Clear live controls; explicit **New terminal** creates a new ID. No “Undo”, “Resume” or restart masquerading as continuity |
| Creation/cleanup unconfirmed | Keep the exact locator and scoped notice; explicit native inspection may recover. No automatic Create/End replay or permanent browser admission lock |
| Capacity reached | Refuse new creation without evicting anyone. Let the user review their own sessions and explicitly End one; do not expose other users' metadata or retry in a loop |
| Required native support missing/refused | Report the refusal and operator next step. No install-on-Open, fallback PTY or project replacement |
| Authorization lost / Soda logout | Cancel affected access and clear private renderers/locators. Whole-context loss affects all its sessions; project denial is not another project's stop |
| Stop project… | Only in Project settings → Overview, authorized owner/operator, shared-impact confirmation. Start uses the same retained root but cannot restore ended shells |

End confirmation copy: **“End ‘Auth refactor’ in acme/api?”** Follow with: “This ends
this terminal and processes in its managed session. Unsaved in-process work will be
lost. Files and independently managed services remain.” Default focus is **Cancel**.
No guessing whether an editor is dirty and no generic red `×` that sometimes means End.

Session menu: Rename; Move to pane; Hide; divider; End terminal…. Pane menu: Split right/below; Maximize/Restore;
Consolidate panes. Project menu: Open repository; Project settings. Separate ownership,
not one huge menu. Bulk End, floating windows and stacked-pane mode are not part
of this first design; they can be reconsidered after core interactions work.

## 7. Mobile, compact drawer and accessibility

When measured space cannot support the desktop layout, show **one usable terminal**,
never a scaled desktop grid. Use a compact native-like header, a **Projects** button with current project/terminal,
and a small overflow menu. Touch targets are at least 44px. Pane selection goes into
**Panes (N)** only when more than one pane exists; no extra always-visible toolbar.

**Projects** opens a full-height navigation view with the same search, project grouping
and Attention filter. It is a view change, not Hide, detach or a modal over a live prompt.
Selecting returns to the exact terminal. Explicit Back returns without changing the
selection. Project settings uses another view with its own Back action. Keyboard focus
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

Forgejo renders the native Spaces document at `/?soda-view=spaces`, including its
real navigation/profile menus and theme. The existing `/-/soda/spaces` URL remains
a fixed native-login bookmark bridge. The separate Go shell is removed. Native
entry connects once through normal OAuth when needed, then mounts the existing Lit
workspace using the original native actor. First consent and coordinated logout's
partial outcomes remain explicit; this is not atomic SSO.

Preserve the implemented transaction-bound **fixed Spaces OAuth return**, actor/CSRF/Origin,
PKCE, encrypted grants, context rotation/logout and fresh repository authorization.
A native-only sign-out is not atomic Soda/SSH logout; styling never proves identity.

Use the [native terminal inventory and ownership contract](terminal-integration.md#managed-terminal-implementation-and-proof-limits),
not a second workspace registry. Preserve original account binding, one writer,
Stop admission and cancellation of stale access; layout does not own native work.

One bounded authorized environment/session collection serves page and drawer. Filter
before disclosing rows/counts; keep degraded observation distinct from launch authority.
Bound provider/helper fan-out and distinguish unavailable enumeration from empty data.
Do not copy provider roles or scan arbitrary project processes. Session labels/signals
are bounded native-runtime metadata, not durable jobs.

Store only bounded per-window session locators, order, pane ratios/selection and
sidebar preference. No credentials, transcripts, queued keys or cross-window layout
synchronization. Authorization loss invalidates private UI data. Metadata refresh
cannot recreate terminal renderers, replace IDs or become a keep-alive mechanism.

## 9. Implementation and review acceptance

The [first-use journey plan](sodaspaces-plan.md#first-use-journey-implementation-plan)
owns the selected presentation changes above. Reuse the existing
[Lit sequence](lit-migration-plan.md#4-ordered-implementation-slices) and its evidence.
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
