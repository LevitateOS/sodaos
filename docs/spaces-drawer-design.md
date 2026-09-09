# Spaces drawer: browse the forge, work in terminals

**Design specification, not deployed behavior.** The user approved the direction of
the full-page [Spaces design](spaces-design.md) and requested an equally concrete
right-half drawer. This specifies that complementary surface—not another workspace,
terminal implementation or UI stack. The [Lit implementation plan](lit-migration-plan.md)
owns the shared component/backend sequence for both surfaces; no production Lit
component is ported yet. Shared session/action/attention semantics remain in the
parent design and [terminal contract](terminal-integration.md).

The [visual sheets](design/spaces/README.md) show native-forge browsing alongside
terminal work, the drawer's session switcher, and compact Forge/Terminal switching.
Their native page and terminal content are fictional schematic drawings; they do not
claim upstream rendering, working forms, authentication or native process continuity.

## 1. Two useful surfaces, not a modal

```text
┌───────────────────────────────────┬──────────────────────────────────────┐
│ Native Forgejo navigation         │ Spaces  Sessions ▾  !1  Full page Hide│
│ acme/web       [Sodaspaces]        ├──────────────────────────────────────┤
│ Code Issues Pull requests …       │ api / Auth  api / Review !  web / Dev +│
│                                   ├──────────────────────────────────────┤
│ Browse files, reviews, issues,     │ alex @ acme/api ▾                  ⋯ │
│ branches and repository settings. │                                      │
│                                   │ Native terminal / CLI agent          │
│ Write a comment here while        │                                      │
│ the terminal keeps running.       │                                      │
└───────────────────────────────────┴──────────────────────────────────────┘
                                    ↔ resizable divider
```

- Open from the existing **Sodaspaces** repository-action button, not a new repository
  tab. The global **Spaces** navbar destination still opens the full-page workspace.
- Start at **50/50**. The drawer occupies the right side's full available height, with
  its own compact header; Forgejo's existing navbar stays with the native left page.
  Do not put a second Forgejo navbar above the terminal or copy its account menu.
- Keep both sides clickable, scrollable, selectable and keyboard-usable. No backdrop,
  outside-click dismissal, desktop focus trap, inert native page or fake forge iframe.
- Native page content actually reflows into its left width. Hiding half an unchanged
  desktop page underneath the drawer is not acceptable. Preserve native handlers,
  routes, forms, beforeunload, clipboard, notifications and browser history.
- Independent scrolling: the native page scrolls normally; the terminal scrolls its
  own history. The drawer header/tabs never scroll away with a long issue thread.
  Do not add a third outer scrollbar around a fixed-height terminal card.
- Native menus/popovers must remain operable near the divider. A real native modal
  retains its native modal behavior and may cover both sides; Soda must not override
  it to keep terminal input active. Native overlays must not get trapped under the
  drawer's z-index. Processes still survive ordinary focus changes.

### Width and density

Retain the **35–65%** divider range, but clamp to useful actual widths: aim for at
least 480px of native page and 56 terminal columns plus padding. At the current
14px mono size this normally requires roughly 960px for both surfaces. When no
valid split remains, use the compact switcher below; 800px is not a reason to keep
an unusable 810px two-column layout.

Pointer drag and an accessible keyboard separator resize both surfaces. Announce
percentage on deliberate adjustment, not every terminal fit. Keep the user's desired
ratio so a temporary small viewport does not overwrite their normal desktop preference.
Fit the real terminal using measured font metrics; never send zero-sized PTY resizes.
Do not reduce the font automatically. On a typical 1440px display, half-width leaves
room for roughly 80 columns; wider displays provide more.

Native media queries may use viewport width even when Soda narrows the body. The
implementation must review the actual selected Forgejo pages and use supported,
Soda-scoped layout adaptation where needed—not assume `body {width: 50%}` proves
navbar, diff, form and popover compatibility. Keep the merged native repository
header's inline actions above 1000px pane width and its disclosure at/below that
width; the schematic sheets do not override that real behavior. Code/diff horizontal scrolling can
stay inside the native file region; the whole page must not disappear under the drawer.

## 2. Compact, terminal-first drawer chrome

Three rows, then the terminal fills the rest:

1. **48px workspace header:** `Spaces`, **Sessions**, attention count when nonzero,
   **Open in Spaces**, **Hide**. The attention badge opens the switcher's Attention
   filter; it never automatically selects/answers a terminal. On narrower widths,
   Full page uses an accessible expand icon and attention moves into Sessions; do
   not clip Hide or shrink hit areas.
2. **36–40px terminal tab strip:** session tabs with project labels, a labelled
   overflow list and **+ New terminal**. No Environment/Access cards above the prompt.
3. **32px trusted context row:** **original login @ owner/repository**, project menu,
   selected-session menu. Show a warning row only for an actual lifecycle/connection
   problem or supported reported attention, not a permanent setup summary.

A small connection footer is optional within the terminal area. The terminal's own
prompt and agent controls remain native terminal content; there is no Soda composer,
Send button, approval button or universal agent launcher.

The selected tab remains identifiable while typing on the left, but the terminal
must not show a keyboard-focus border/cursor as though it owns input. Blue focus
follows the actual focused control; amber attention never steals it. Clicking into
an issue comment must neither send those keystrokes to the terminal nor detach it.

## 3. Tabs show the same working set

The drawer has no permanent project sidebar: at half width it would consume the
terminal. Instead it presents **all this window's open session tabs**, including
sessions assigned to different panes on the full page. The full page's saved pane
layout remains intact.

Use one representation: derive the flat strip from the existing pane tree in stable
pane/tab order. Selecting a drawer tab updates the selected session in its original
group; it does not move that session between groups or create another attachment.
There is no independently synchronized drawer session list or second backend owner.

- Tabs show a short unambiguous project label and the user session name; full
  `owner/repository` is always present in trusted context and accessible labels.
- Overflow scrolls and has a searchable **Open tabs (N)** list. Keep a reachable `+`;
  do not reduce six tabs into six unreadable labels or hide tabs behind management.
- **+** opens the same project/name chooser as the full page. It visibly defaults
  to the selected terminal's project, not whichever repository is on the left.
  With no selected terminal it may preselect the native page's verified association;
  creation still requires an explicit project/account review and action.
- A newly created tab joins the selected session's original group (or the initial
  group if none exists). It opens no other shell, environment, account or worktree.
- **× = Hide this tab**, with the same effective-retention tooltip and uncertainty
  handling as the full page. End is an explicitly named menu action with confirmation.
- Hiding removes that tab from the working set, including its original pane. If a
  group becomes empty, the normal full-page empty-pane rule applies. The session can
  remain discoverable as kept; removal from a view is not process cleanup.
- Pane splits, drag-reordering across groups and arrangement happen in full Spaces.
  The drawer's first design deliberately has no recursive split controls or second
  layout editor. **Open in Spaces** is the visible way to arrange these same sessions.

## 4. Session switching without blocking the forge

**Sessions** replaces only the right terminal content with a full-height switcher;
the header, existing tab strip and a **Back to terminal** control remain available.
It is not a modal or a second sidebar squeezed alongside the shell. Terminal views
are visually covered/hidden, not disposed or detached. The left page is untouched.

The switcher reuses the full-page authorized session collection and state labels:

- Search project/session names; stable project-grouped rows; full project/account on
  selection; kept, attached-elsewhere, ended and unconfirmed rows remain honest.
- Filters: **All**, **This page: owner/repo**, **Attention**. This-page filtering is
  explicit and affects only the list. It never changes a displayed terminal or starts
  anything. Omit it on non-repository native pages. On each open, default to All so
  a previous repository filter does not silently follow navigation.
- Selecting an open tab returns to/focuses that exact session. Selecting a kept row
  performs exact authorized attach and deliberate Return only when eligible, with
  the actual deadline shown until confirmed. Other-window writers cannot be evicted.
- Existing terminal tabs can also dismiss the switcher by deliberate selection.
  Back/Escape inside the switcher returns without changing the selected session.
  Search gets focus only when the user opens it; output never opens it automatically.
- Empty/denied/unavailable results follow the full-page distinctions. A lost read is
  not an empty successful list; no retrying creation from a missing row.

Attention is the same bounded, advisory model as full Spaces: ordinary output is
only a quiet unread dot; semantic agent states require real explicit reporting.
Opening or reading a waiting terminal does not resolve its pending question. Session
attention does not become a Forgejo notification or a global background poller when
the drawer has never been opened.

## 5. Browsing another repository is normal

**The left page's repository and the terminal's project are independent.** The main
sheet intentionally shows a pull request in `acme/web` beside a terminal in `acme/api`.
That is useful cross-project work, not an account mismatch or a warning condition.

- Switching native Code/Issues/Pull requests, following a commit, opening a different
  repository or using browser Back never retargets a shell, renames its tab or creates
  a replacement. The right identity remains `alex @ acme/api` until an explicit session
  selection changes the view.
- Native page loads replace the document. Restore the authorized working-set IDs,
  selected session and drawer width, then reattach surviving sessions. The renderer
  can be new; the native shell must be the same. Brief **Reconnecting existing…** is
  honest; a replacement prompt is not continuity.
- Automatic restoration does not send Return, queued input, provisioning or other
  mutations. Existing server deadlines stay in force. Offer per-session **Continue
  working** when required, not an invisible keep-alive for every navigated repository.
- **Repository difference is not actor difference.** Native page user/Soda actor
  mismatch, expiry and actual lost authorization still block/clear affected access
  through existing guards. Never use the old page's actor or left repository to bypass
  fresh authorization for the selected terminal's original project.
- On signed native pages without a repository, the existing global resume hook can
  reopen this same working set. Do not add a second terminal instance or invent an
  association for native administration/settings pages.

## 6. Drawer ↔ full Spaces ↔ native page

| Action | What changes | What stays |
| --- | --- | --- |
| Open/reopen Sodaspaces | Restore this window's drawer and selected surviving session | Original targets; no implicit new terminal or automatic project start |
| Select terminal / open Sessions / view project details | Only the right-hand view or focused tab | Native form/scroll and existing sessions |
| Open in Spaces | Normal same-origin navigation to `/-/soda/spaces`; focus selected session in its saved pane | Exact session IDs, groups, labels and server deadlines; no fresh shells |
| Browser Back from Spaces | Native page history and authorized drawer restoration | Existing surviving session identity; no arbitrary `return_to` URL or promised draft resurrection |
| Hide workspace | Restore full-width native page; retain this document's owned working-set sessions within finite limits | Files/projects and discoverable surviving sessions; no End or Stop |
| Compact Forge/Terminal switch | Change which surface is visible in the same document | Native draft/scroll, open state and sessions; **not** Hide workspace |

**Open in Spaces is navigation, not an in-place maximize.** Keep native unsaved-form
warnings. If the user cancels navigation, leave the drawer and attachments untouched.
Do not copy/submit/comment-store the native form or bypass beforeunload. Modified
click may open another browser tab, but that tab must show Attached elsewhere for
existing writers rather than stealing them. Users can keep working at half width
without ever visiting the full-page destination.

Hide applies finite retention only to this document's owned attachments—not every
session in the sign-in or another window. Existing explicit shorter/longer deadlines
and authentication limits still govern; display actual results per session. An
unconfirmed request is not a promised 30-minute grace. Reopening does not silently
Return all hidden terminals; a deliberate session selection/Continue acts on that
eligible session. Retain exact uncertain outcomes without replay or automatic repair.

## 7. Environment and access remain available, not in the way

The trusted project menu exposes **Environment**, **Access / SSH** and **Open repository**
for the explicitly named project. These reuse the existing management/API owners and
server-side authority, not duplicate implementations or the left page's implicit target.

Environment/Access replace the right content, with their exact project/account in the
title and Back to terminal. The tab strip remains reachable. Merely switching these
views does not unmount terminal owners, set retention, or start/stop anything.
Keep real Create/Join, saved-key review/apply/revoke and authorized Start/Stop available.
Required keyless Join/Git credential work remains unfinished, not hidden behind a fake
success. Stop stays in the named Environment view with shared-impact confirmation;
it is never next to `+`, tab close or Hide. Errors stay associated with their project.

The drawer session menu retains Rename, Keep, eligible Continue, Hide and End. End
confirmation names project/session/original account, distinguishes process loss from
persistent files, and waits for actual confirmed/uncertain native cleanup metadata.
There is no destructive all-sessions close, task cleanup or native-process scanner.

## 8. Compact and keyboard behavior

When two usable columns do not fit, show a clear **Forge / Terminal** surface switch
while the workspace remains open. Keep Forgejo's native header available; the compact
workspace switch sits below it. Terminal occupies the remaining visual viewport.
Do not render the forge and terminal as two tiny side-by-side mobile columns.
A new native page navigation/Back shows **Forge** in compact mode so the destination
is not covered by the restored terminal view. Only explicit drawer-opening intent
or selecting Terminal reveals it. Restore session locators independently of surface
visibility; do not persist a Terminal-visible flag that hides every newly visited page.

This is an in-document view change: retain native DOM, unsent form values, scroll and
live terminal owners. Expose only the currently visible surface in the tab order and
accessibility tree; this is not a desktop focus trap. Preserve real native modal
behavior. **Hide workspace** is separate and removes the switch, returning to normal
native browsing with finite retention. On widening, restore the chosen split ratio.

The drawer's Sessions view and terminal tabs still work here; use at least 44px touch
hit areas, overflow menus and no permanently visible pane-layout controls. Respect
mobile visual viewport/keyboard changes so the prompt and surface switch stay usable.
The sheets show layout intent, not mobile browser/keyboard proof.

Keyboard focus stays where the user puts it. Keep shell Escape/Ctrl-C/D and browser
shortcuts; Ctrl+Shift+Enter leaves xterm for the selected tab's controls. Tab-list
arrows/Home/End act only when the tab list has focus. Escape can close an explicitly
opened switcher/menu, never globally Hide the workspace or terminate a shell. Return
focus to its invoker on cancellation; explicit selection may focus the target terminal.
Divider resize, overflow, creation, project details and Hide are keyboard-reachable.

## 9. What implementation must prove

The existing native hook already supplies a fixed non-modal aside, a 35–65% separator
and native body reflow CSS. Its component still has one terminal and management view
tabs. Multiple session tabs/collection/ownership, this chrome and compact projection
are real work, not a template label change. Preserve stock Forgejo; no fork, scraping,
borrowed cookies, replacement native navigation or host/network changes.

Validate with real native pages and the same managed-terminal boundary:

1. Six sessions across two projects; select/hide/create/end independently from the
   drawer, with exact IDs shared with full Spaces and no duplicate writer.
2. While sessions run: browse code/diffs/commits/issues/reviews, write an unsent native
   comment, use native Copy/notification/account menus, scroll both sides and resize
   35/50/65 within viable widths. Verify no clipped native controls or blocked overlay.
3. Navigate to another repository, reload and use Back/Forward. Confirm actual unchanged
   shell PID/start/memory, correct account/project and no input/mutation replay. Test
   cancelled native beforeunload separately from a completed navigation.
4. Move between drawer/full page with a saved multi-pane layout; keep selected IDs,
   handle attached-elsewhere and preserve finite deadlines. Switching right views or
   compact surfaces must not be confused with Hide, End or container Stop.
5. Regress actor mismatch, denied/unavailable collection, expiry/logout, pending native
   outcomes and focused sibling preservation. Native Forgejo rights and Soda operator
   authority remain distinct. Nothing in the drawing proves these checks have passed.

No new origin or preview server is needed. Real implementation belongs on the existing
Forgejo/Soda origin (33443 in the isolated deployment); this design changes no service,
project root, private input, provider resource or separate Rocky rollout decision.
