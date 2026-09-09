# Lit migration plan

Port the Soda-owned drawer and terminal presentation incrementally, preserving
native Forgejo workflows and the current action contracts. The intended benefit
is one readable representation of UI state and markup, with fewer manual DOM
updates. Converting every browser script is not the objective.

**Status:** planning only. The shared Lit scaffold is complete at `c675411`; no
production component has been ported. [Lit authoring/build guidance](lit.md) owns
framework conventions. [Sodaspaces](sodaspaces-plan.md) and the
[terminal contract](terminal-integration.md) continue to own product behavior,
authority and native execution. This plan does not authorize deployment.

## Scope and order

Paths below are relative to the repository root.

| Current source | Decision | Reason |
| --- | --- | --- |
| `appliance/forgejo/public/assets/sodaspaces-drawer.ts` | Port first | Owns Soda markup, view tabs, management/access state and many manual visibility/text updates. |
| `appliance/forgejo/public/assets/sodaspaces-terminal.ts` | Port presentation second | Lit can own controls/status; xterm and the existing transport must retain their imperative lifetimes. |
| `appliance/forgejo/public/assets/sodaspaces.ts` | Keep a small native-page adapter | Owns the repository hook, aside, divider, body sizing, Hide and document lifecycle. Adapt its mount/readiness calls only. |
| `appliance/forgejo/public/assets/sodaspaces-api.ts` | Keep ordinary TypeScript | Response validation and API types are not UI components. |
| `assets/branding/forgejo/notification-preview.ts` | Keep the existing adapter for this migration | Its content is an upstream HTMX fragment; most complexity is request identity, popover/focus behavior and native integration. A Lit shell would not remove that contract. |
| `assets/branding/forgejo/personal-settings.ts` | Keep the existing adapter | Progressively enhances server-owned navigation, disclosures and the native avatar form. |
| `assets/branding/forgejo/repository-actions.ts` | Keep the existing adapter | Adds narrow-pane disclosure behavior around native forms; CSS owns the compact/wide boundary. |
| `assets/branding/forgejo/login-theme.ts` | Keep the small early script | Applies guest appearance before paint; authenticated themes remain Forgejo-owned. |
| Forgejo templates/presentation partials and Cockpit | Keep their current stacks | Server-rendered native pages and Cockpit React/PatternFly are outside the port. |

Drawer-first lets the largest rendering change reuse the existing terminal mount
and test seam, without rewriting the transport while its continuity contract is
pending. The terminal controls follow against the contract implemented at that time.

New Soda-owned interactive features should use this component pattern where they
benefit from it. The planned Spaces page still needs its own authenticated
Go/template page-shell solution; Lit does not resolve that integration gap. Revisit
the notification shell only if a concrete later change removes enough duplicated
UI state to justify a port, while keeping the HTMX target outside Lit rendering.

## Component boundaries

Start with two components in the existing flat browser source directories:
`<soda-spaces>` in `sodaspaces-drawer.ts` and `<soda-terminal>` in
`sodaspaces-terminal.ts`. Preserve the small exported mount facades used by the
native hook and test callers. Each facade creates one element, supplies validated
context, and forwards the drawer's refresh/invalidate/dispose behavior or the
terminal's started/disconnect/invalidate/dispose contract. Keep one
implementation per component; remove replaced DOM-building code in the same change.

Use light DOM for these first ports, retaining the existing scoped CSS, native
clipboard delegation and xterm stylesheet. Keep the current wrapper classes,
landmarks, labels and contract IDs; set explicit host display/min-size rules so
custom-element defaults do not change grid/flex sizing. Lit renders only inside
Soda's supplied content mount, never over repository actions, native forms or an
HTMX swap target. Shadow DOM can be selected for a future isolated component when
its styles, focus and delegated events have been designed for that boundary.
[Lit documents these render-root tradeoffs](https://lit.dev/docs/components/shadow-dom/).

Use strict TypeScript and static property declarations as in [the guide](lit.md).
Keep immutable page context separate from observed session/environment data and
editable drafts. Credentials and internal state must not reflect into attributes.
Derive visibility/disabled states from the component's state instead of maintaining
parallel DOM flags. Validate external responses through the existing API functions.
No application store, router, base-component framework, event bus or new UI kit is
needed for these two components.

Lit updates asynchronously. Keep action guards synchronous: a second click before
the DOM disables a button must not dispatch another request. Render/updated hooks
may update presentation, focus after a deliberate interaction, or fit an existing
terminal; they must not create, join, start, apply keys, log out or open a socket.
Wait for `updateComplete` where callers need rendered controls, and preserve the
user's current focus when an asynchronous result arrives.
[Lit lifecycle](https://lit.dev/docs/components/lifecycle/) and
[event handling](https://lit.dev/docs/components/events/) define these mechanics.

## 1. Port the management drawer

**Change:** replace its `node()` construction, manual text/hidden/disabled updates
and repeated list rebuilding with Lit templates and typed state. Keep the existing
Terminal, Environment and Access views, status/outcome messages, roving tab focus,
OAuth link, explicit commands and native clipboard target.

- Retain `refresh`, mutation dispatch, timeout/abort, generation checks and uncertain
  outcome handling with this concrete owner. Move methods only as needed to express
  state; do not invent a generic request/controller framework. Keep actor/provider
  rechecks and the exact server-owned operation/response contracts.
- Move `readSodaJSON` from the terminal module into `sodaspaces-api.ts` when updating
  its two callers, so the shared bounded JSON reader has a clear owner and the
  component dependency runs drawer → terminal → API, without a cycle.
- Bind Soda form values deliberately. Ordinary status updates, tab switches and
  Hide/reopen must retain draft text, selection and confirmation state. Clear them
  only at the existing explicit reset boundaries. Preserve key-list action identity
  and predictable focus when a row changes. If a keyed-list directive is needed,
  add its export/resolver mapping through the one shared Lit runtime with coverage.
- Keep the current imperative `mountTerminal` in a stable, always-present terminal
  host for this first port. Hide panels rather than removing them on view changes.
  Never let a routine Lit render clear xterm's host or instantiate another terminal.
- Adapt the native shell's first-render/focus path for asynchronous rendering while
  retaining its single mount, explicit opening and `refresh`/`dispose` contract.
  A late render must not reopen a hidden drawer or move focus out of the native page.
  Context changes invalidate the old component; they cannot retarget a live terminal.
  After awaiting the rendered terminal host, recheck generation, stale and disposed
  state before invoking `terminalFactory` or applying focus.

**Exit:** all existing drawer action and race assertions run against the port.
Cover anonymous/expired/mismatched sessions; absent, incomplete, stopped, running
and unavailable environments; administrator/member differences; key review/change/
last-key removal; explicit Create/Join/Start/Stop; safe Copy; stale responses and
unknown mutation outcomes. Hidden controls and rapid repeated clicks dispatch no
extra commands. Native input nodes and unsaved values survive drawer interactions.
Invalidate/dispose a verified refresh while its Lit update is pending: its terminal
factory must never run afterwards, and the old context must not repaint or supply
a terminal to a subsequent mount.
Run the visual/keyboard matrix below before beginning the terminal port.

## 2. Port terminal controls without changing the transport

**Change:** make the terminal's heading, warning, toolbar, status and screen wrapper
Lit-owned. Keep `mountTerminal` compatibility and the current `started`, `disconnect`,
`invalidate` and `dispose` behavior. The component owns the concrete transport/renderer resources;
there is no second session model mirrored in the drawer.

- Keep one stable screen element in the template. Xterm exclusively owns that
  element's descendants; never interpolate terminal output as HTML or render into
  its subtree. Ordinary component updates must preserve its renderer, selection
  and screen node.
- Create/load xterm and open the socket only through the existing explicit Open
  action after current authorization checks. Preserve renderer loading, original
  login binding, frame validation, input/output bounds, OSC protections and resize
  behavior. Do not change backend APIs or terminal wire messages in this port.
- Preserve the operation generation and pending-request guards. End, invalidation
  and actual disposal release the owned observer, timers, listeners, socket and
  renderer exactly once. Hide, tab selection and application focus changes retain
  them. Remove external listeners on disconnection and prevent late effects after
  retirement; reconnecting a retired element must not relaunch a shell.
  For a ready terminal, explicit End sends the existing `{type: 'close'}` frame
  before local teardown.
  Facade `disconnect`, invalidation, disposal, page departure and custom-element
  disconnection close the attachment without synthesizing that End frame.
- Preserve xterm Escape input, Ctrl+Shift+Enter focus on End, hidden-view Open guards
  and the rule that late readiness cannot steal focus from a native page input.
  Registration/mounting remains inert until explicit user action.

**Exit:** preserve every current terminal contract test, using the existing fake
renderer/transport seams for deterministic races. In real Chrome, open once, change
component state repeatedly, resize, switch views, edit native controls, Hide and
reopen: exactly one renderer/socket must remain until an explicit terminating
boundary. Failed authorization, stale readiness and oversized/invalid IO must keep
their current behavior. These checks prove the browser component, not a native PTY.
Test overlapping teardown paths (dispose followed by disconnection, or a late socket
close) for exactly-once cleanup and no extra End frame.

## Relationship to terminal continuity work

The current drawer's `refresh()` calls `reset()`, which disposes its terminal and
clears drafts. `terminalUsed` prevents reopening a previously started/ended terminal;
it does not preserve a running session. Navigation/BFCache and transport loss also
still retire current browser access. The rendering ports must not claim these gaps
are solved or silently change their meaning.

Server-owned tmux reattachment and separating management Refresh from terminal
lifetime remain selected work under the [terminal plan](sodaspaces-plan.md#resumable-terminal-decision--tmux).
They are a separate behavioral change, with their own API/native proof. The drawer
rendering port can proceed independently. Before step 2, use whichever terminal
contract is then implemented: if continuity landed meanwhile, port that current
owner and retain its new tests; do not resurrect the older request-owned lifetime.
Do not add multiple sessions, keep-alive rules or browser-only joining to hide gaps
in this migration. Native continuity acceptance still needs the same real shell
across navigation/reload/network loss, not a remounted renderer or mocked socket.

## 3. Validate and finish each port

Update existing tests with their source group. The JSDOM fixtures currently create
separate document realms while importing browser modules once; a LitElement class
must be evaluated and registered in the realm where it is instantiated. Run component
render/lifecycle contracts using the existing Playwright loopback fixture and emitted
modules, with its synthetic API/renderer seams. Keep DOM-free parsing/contract checks
in Bun. Do not install a second browser test framework, patch global `HTMLElement`
to hide realm errors, or weaken assertions merely to preserve the old harness.
Extend `bun run test:lit` to execute the converted drawer/terminal contracts in
`tests/frontend/` as each port lands. That required browser command must fail if
its browser cannot run; ordinary passing tests with these cases skipped do not
satisfy the exit.
Await rendered updates instead of adding arbitrary sleeps.

| Check | Required evidence |
| --- | --- |
| Source/build | `bun run typecheck`, `bun run test:frontend`, `bun run test:forgejo`, `bun run test:lit`; converted browser action/race contracts must execute, not skip. |
| Browser lifecycle/layout | Extend the existing `tests/frontend/drawer-layout.test.ts` fixture; run with `SODA_DRAWER_LAYOUT=1` and the established longer browser timeout. Preserve real xterm styling plus synthetic transport/IO checks. |
| Widths and themes | Light/dark at full 1440px; half-desktop windows at 720, 800 and 960px; 1440px split view with actual 35/50/65% pane sizing; 390/320px mobile. Cover running/stopped and error/long-content states with representative cases. |
| Native coexistence | Keep native DOM/form identity, unsaved values, keyboard order, tab/Hide focus, clipboard integration, menus/notifications and native navigation usable. Repository actions remain inline above the existing 1000px pane boundary and compact only at/below it. |
| Payload/preview | Build through `build:forgejo`/`build:preview`, update the canonical payload only for actual new files, run focused Python/Go staging checks, and verify served bytes plus one shared Lit runtime. Project any changed root-level Sodaspaces assets into the existing preview public mount too; the branding-only preview command does not copy them. |
| Native-page visuals | Use `scripts/screenshot.ts` and the authorized fixture profile from [screenshot capture](screenshot-capture.md). Capture the repository page at half/full width; use synthetic fixtures for mutating states unless the exact native action is authorized. |

Keep the runtime scoped to the existing supported module hooks. The first port
will make repository pages import Lit through their Soda component dependency;
other native pages need no global Lit script. Do not add a separate bundle/runtime
per component. Preserve `AppSubUrl`-safe asset resolution; the existing Sodaspaces
native mount still accepts only an empty sub-URL, so asset-prefix tests are not a
claim of new backend/subpath support. Add directives only when used, through the
shared runtime mapping, and keep dependency/license metadata current.

Finish each step in a coherent local commit with its tests and handoff evidence.
Suggested implementation commits are `refactor(ui): render sodaspaces drawer with lit`
and `refactor(ui): render terminal controls with lit`. Remove superseded helpers and
listeners as their callers move; no production feature flag or dual-renderer period.
Source rollback is a coherent Git revert and local asset rebuild, preserving fixture
and project state. Appliance delivery keeps its existing separate target/action
approval and preserved-state requirements.

The migration is complete when both selected components use one declarative render
path, the behavior/visual checks above pass, no duplicate handlers/terminal mounts
remain, and docs record the exact remaining product gaps. The retained native
adapters are deliberate boundaries, not unfinished ports.
