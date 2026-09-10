# Lit implementation plan: Spaces page and right drawer

Implement the selected [Spaces page](spaces-design.md) and
[right-half drawer](spaces-drawer-design.md) with the shared Lit scaffold, not two
frontends or another preview application. The [leading plan](sodaspaces-plan.md)
owns product scope/order; this document is its **single feature implementation
sequence for the Lit workspace**. The [frontend improvement guide](frontend-improvement-plan.md)
owns the researched cleanup detail within that sequence. [Lit guidance](lit.md) owns framework/build rules,
the [terminal contract](terminal-integration.md) owns native lifetime, and the
[API guide](dashboard-api.md) describes implemented endpoints, not the proposals here.

**Current source: steps 1–2's management and terminal controls are ported, with local
emitted-browser and layout checks. Step 3's ID registry/collection and caller contracts
are source-implemented with local Go/race/browser coverage. Step 4's authenticated
page/fixed return and shared multi-session workspace now have local source/browser
coverage at `18aceb2`. Step 5a–5d now has local source/browser/geometry coverage:
bounded v2 layout, stable measured panes, shared chrome, compact native coexistence,
current journey source ports and actual Go HTML→emitted-page integration. Canonical
tokens, required template analysis, typed stateless views and source/build ownership
are implemented. Steps 6a/6b now have local observed-attention and candidate/driver
coverage. Step 6c now has bounded x86_64 terminal-boundary, two-actor six-session
matrix/cleanup, both-target preserved delivery and native browser/access evidence;
selected CLI/provider and broader native acceptance remain outstanding.** The sequence below
is reconciled with that source, not another design cycle. Xterm/transport and native
tmux ownership stay imperative; no Preact, second frontend or backend redesign is
selected. Canonical branding and complete-public preview projections exist; changing
live mounts and installed acceptance remain separately scoped. See the handoff for
actual evidence. Steps 1–5 below retain their completed source-slice contracts;
their historical singleton/port instructions are not tasks to repeat. Do not port
the superseded mockup or reintroduce grid presets/fake state machines.
The user subsequently requested end-to-end implementation and local testing.
Deployment, new appliance fixtures and project recreation remain separately scoped.

## Extension after the current terminal implementation

The [leading plan](sodaspaces-plan.md#extension-order-and-dependency-boundaries)
now selects Project OS profiles, Desktop and AI run views. Preserve completed Lit
steps and outstanding 6c evidence; do not repeat this migration or treat the new
features as already covered. All profiles inherit the existing Project OS contracts.

Extend the shared page/drawer owners only when each new native operation has a
defined authorization, identity and lifetime contract. Profile choices must reach
real provisioning; desktop controls must reach a real project-user graphical
session; AI views must attach to the exact run. Keep transport-specific resources
with their owner and reuse existing presentation/layout. Do not generalize personal
terminal IDs or leases into an unvalidated universal session backend.

Extend local behavior tests for denial, stale targets, shared navigation, correct
input focus and existing-terminal preservation. Native profile/account/desktop
proof belongs to the existing product validation owners; no static design sheet,
synthetic socket or terminal matrix is graphical/runtime acceptance.

## Cleanup after step 5

The architecture research was checked against `8e812dc`
(`feat(spaces): close local layout integration and journey ports`). Preserve its
completed layout, shared controls, current journey ports and Go-page integration.
The [consolidated guide](frontend-improvement-plan.md) recommends retaining Lit and
defines the next cleanup before further UI expansion: consolidate canonical tokens
across both surfaces, integrate the demonstrated separate analyzer, extract typed
project views, then improve remaining presentation and source organization.

This is not a replay of steps 1–5 or a new renderer/backend migration. If step 6a
has already advanced, preserve those changes and apply the same requirements to its
current owners; do not reset work to the reviewed commit. Step 6b's candidate closure
must include token adoption, enforced template diagnostics and the preserved behavior
checks. The earlier isolated analyzer experiment is now a required root-workspace gate;
known callable-event parameter-typing gaps remain explicit. Native/CLI acceptance is still 6c.

## 1. Starting point and decisions

These were the step-4 starting facts, not a claim that the completed step-5 work
is still absent. The handoff records exact revisions and checks.

| Source fact | Consequence |
| --- | --- |
| `c675411` supplies Lit 3.3.3, one root Bun lock and one staged core runtime | Reuse it; no React rewrite, router, UI kit, dependency upgrade or per-component bundle. Management and terminal components import Lit; the native adapter loads them on demand. |
| `sodaspaces.ts` owns native hooks, aside, divider and document departure | Retain a small native adapter; do not render native Forgejo with Lit. |
| `sodaspaces-drawer.ts` owns shared `Slot` handles, one selected view and project-control instances; `sodaspaces-project.ts` owns management writes/drafts | Replace the one-view projection with pane layout, not the existing action owners. The drawer filename is historical: it also serves the page. |
| `sodaspaces-terminal.ts` implements create/exact attach, restore, retain, Return and HTTP End | The old socket `{type:'close'}` End and request-owned lifetime are obsolete. |
| Slots have stable local UUID keys and exact/pending locators; `selectSlot()` can invoke `restore()` | Keep owner keys stable through pending-to-ID promotion. Do not use the effectful selection path for pure split/move/resize operations. |
| `soda-spaces:v1:<actor>` stores up to 64 locator entries and hidden flags, with a 16,384-code-unit read limit, but no pane tree or saved selection | Step 5a migrates this bounded format without dropping unresolved/hidden entries. Browser layout versioning is not a new DB migration. |
| `soda-workspace:<actor>` separately stores native repository resume hint, open state and width | Keep native surface preferences in the native adapter; do not copy session/layout authority into it or persist a compact Terminal-visible flag. |
| Project details preserve drafts on switching; their explicit Refresh still calls `reset()` | Preserve those deliberate reset boundaries; workspace layout/search/metadata refresh must not reset project forms. |
| ID-keyed backend/collection and a three-session/two-project emitted-browser fixture exist | Extend the real source callers/tests. HTTP/socket doubles are not native process or CLI proof. |
| Native guard/tmux is already per terminal ID | Keep that mechanism; concurrency needs regression/native proof, not another PTY broker. |
| `22d8591` has isolated same-shell reload and End/cleanup evidence | Preserve it; full native probe, editor/build/history/network/failure acceptance remains incomplete. |
| Incoming repository actions are inline above 1000px **pane** width, a disclosure at/below it | Preserve real native form nodes and this correction; schematic drawings do not override native header implementation. |
| `spaces_page.go`, the Go template, native link and fixed OAuth return are implemented with schema v6 | Preserve server-derived Soda identity, API guards and original-context sessions; no new route/origin/auth mechanism is needed for layouts. |
| `build:preview` emits branding and complete public trees from the production payload | Extend that same manifest when adding a module; isolated output is not a live preview refresh. |
| The current layout test checks a provisional 216px floor; installed journeys retain old selectors | Step 5 replaces that proxy with measured cells/chrome checks and ports the existing journey callers. Neither gap waits for a native rollout to be authored. |

The shell design is now concrete: Soda-owned Go HTML with canonical branding,
configured-origin native links and an explicitly labelled server-derived Soda
account. It does **not** impersonate Forgejo's authenticated navbar, CSRF, account
menu, notifications or theme authority. The remaining work is implementation and
compatibility/security verification, not discovery of a way to run upstream Go
handlers from a template override.

## 2. Concrete owners

Paths are repository-relative. `sodaspaces-layout.ts` was added in step 5a;
the page/workspace/project owners remain shared. Keep new layout logic there
only when it is pure and shared by concrete callers, not to build a generic store.

| Owner | Responsibility |
| --- | --- |
| `frontend/spaces/sodaspaces.ts` | Native repository button/resume hook, outer split/compact surface, width, native DOM coexistence and page lifecycle. No session inventory or API mutations of its own. |
| `…/sodaspaces-workspace.ts`, `<soda-spaces>` | Shared workspace, live child handles and original bindings; effects, storage IO and authorized restore. Step 5 adds one pane model with page/drawer projections; step 6 adds bounded observed attention. Keep `mountSodaspaces` and its discriminated bootstrap, not a new page controller. |
| `…/sodaspaces-terminal.ts`, `<soda-terminal>` | One immutable terminal binding and its xterm/attachment resources, IO guards, status and explicit terminal commands. Keep `mountTerminal` and injection seams during the ports. |
| `…/sodaspaces-project.ts`, `<soda-project-controls>` | Implemented named-project management/access, drafts and pending/uncertain writes. Layouts reuse these instances; they do not change their targets or duplicate their requests. |
| `…/sodaspaces-layout.ts` (step 5a) | Pure pane-tree transformations, locator/layout parsing, migration and serialization. Takes explicit measurements for geometry decisions; no DOM, storage IO, network, credentials or native lifetime effects. |
| `…/sodaspaces-page.ts` | Implemented thin Go-page bootstrap: validate its server-supplied actor and mount the shared workspace. No client router or second session owner. |
| `…/sodaspaces-api.ts` | Existing bounded JSON reader, response validation and concrete API types/errors. Reuse them; layout state is not server metadata or authorization. |
| `internal/web/{server,auth,terminal,terminal_sessions,management,spaces,spaces_page}.go` | Page/collection, original-context ID-keyed sessions, fresh authorization, cancellation and existing native helper calls. |
| `internal/store/{login,migrations}.go` and focused store queries | Fixed OAuth destination and bounded reads of legitimate Soda associations; no durable terminal registry or copied provider roles. |
| Existing `internal/host/` and `project-os/` | Native account/guard/tmux/lease/cleanup authority, unchanged unless a concrete tested defect requires a scoped correction. |
| `tests/frontend/{workspace,drawer-layout,drawer-controls,sodaspaces,terminal}.test.ts` and focused layout tests | Local behavior/geometry and source-owner identity checks; pure transforms run in Bun, components in emitted Chromium modules. |
| `tests/installed/{sodaspaces,sodaspaces-management}.ts`, their inputs/fixtures, and `internal/host/terminal_native_test.go` | Port existing product journeys with the UI; retain exact target/action/private-input gates. Own later native/CLI proof, not a parallel support-tool suite. |

The workspace's live child handles are **not** a second backend session model.
It knows only what this document attached and rendered. Server metadata remains
observed authority; the pane tree contains IDs/order/selection, not duplicate
session objects, credentials or native targets. Drawer tabs are a projection of
that same tree. A full page and a native document cannot share DOM objects across
navigation; they share bounded locators and exact surviving native session IDs.

Keep notification preview/HTMX, personal settings, repository actions and early
guest-theme scripts as their existing small TypeScript adapters. Keep native
templates/forms/scripts and Cockpit React/PatternFly. No event bus, global store,
base-component hierarchy, interchangeably pluggable backends or lazy feature flags.

## 3. Lit mechanics that constrain the implementation

- Use light DOM for these components: `createRenderRoot()` returns `this`. Keep
  scoped Soda/xterm styles and explicit host display/min-size rules. Lit renders
  only Soda mounts, never a native form or HTMX-owned fragment. Namespaced IDs must
  stay unique across terminals and retained project controls. Keep repository-qualified
  Copy targets; preserve selectors only where their original meaning still holds.
- Follow strict TypeScript/static properties/`declare` plus constructor initialization.
  Bind context/objects as properties with `attribute: false`; internal reactive state
  uses `state: true`. Establish each terminal's binding once; metadata/name updates
  must not replace it. Never reflect CSRF, credentials, grants or transcripts. Validate
  external data from `unknown`; no blanket casts or compiler/decorator changes.
- Rendering is not an action dispatcher. `render` stays pure; `updated` may arrange
  an existing renderer, never create/join/start/stop, log out, apply keys, Return or
  open sockets. Explicit commands and an explicitly invoked authorized restore path
  own those effects. Set busy/generation guards synchronously, before the next render.
- Await `updateComplete` for the relevant component, then recheck context/generation,
  visibility and retirement before mounting/focusing. A parent's completion does not
  await every child or settle geometry; use the child's completion and ResizeObserver.
  Handle render rejection honestly and retire only affected resources, with no replay.
- Xterm exclusively owns descendants of one stable empty screen node. Never bind its
  output to Lit templates, replace that node in a conditional template, or use
  `unsafeHTML`. Labels/reported activity are escaped text, not HTML or commands.
- Add `repeat` only when keyed key/session lists need it: export it through
  `assets/branding/forgejo/lit.ts`, map exactly `lit/directives/repeat.js` to the shared
  runtime in `scripts/build-forgejo.ts`, and test both asset roots/prefixes. Other
  subpath imports remain rejected. No new dependency is needed for that directive.
- Keys preserve DOM identity **within a repeat part**, not across different pane
  parents. Do not nest live terminals inside independently keyed pane templates and
  assume moving a tab preserves them. Start with a stable flat terminal layer under
  one parent, keyed by the immutable local owner key in stable owner order. Pane
  chrome/layout changes position/visibility of those hosts, not their parent/lifetime.
  Measure each pane's screen rectangle and apply bounded layout coordinates. Validate
  labels, tab/tabpanel associations, roving focus, clipping and overlay hit-testing;
  stable DOM alone is not accessibility or xterm focus proof.
- `disconnectedCallback` and explicit disposal must clean up external listeners,
  observers and the document's resources exactly once, with `super` calls. A retired
  element cannot relaunch on reconnect. The stable-layer candidate avoids relying on
  disconnected/reconnected callbacks as a cross-pane move protocol. Page departure
  detaches; native End is a different operation.

These choices follow official [properties](https://lit.dev/docs/components/properties/),
[lifecycle](https://lit.dev/docs/components/lifecycle/),
[render roots](https://lit.dev/docs/components/shadow-dom/) and
[keyed lists](https://lit.dev/docs/templates/lists/). Inspected locked-package source:
`lit-element@4.2.2` connection/update methods, `@lit/reactive-element@2.1.2`
render-root/update activation, and `lit-html@3.3.3` repeat key/part behavior.
Snapshots are at `.artifacts/research/lit-spaces-619c5d9/`; inspection is not a port test.

## 4. Ordered implementation slices

### Step 1 — port the current management drawer

**Source port and local component/layout checks completed.** No deployed-runtime
claim. Tests run emitted modules in
Chromium rather than trying to construct Lit elements in Bun/JSDOM's other realm.

Replace manual node/text/hidden/disabled/list reconstruction in
`sodaspaces-drawer.ts` with `<soda-spaces>` templates and typed state. Retain the
current Terminal/Environment/Access views for this first, rendering-only commit.

- Preserve the complete facade, including `refresh`, `invalidate`, `dispose`,
  `retain` and `returnToWork`, and the injected terminal factory. Adapt native
  first-render/focus readiness without reopening a hidden drawer after a late update.
- Keep the current imperative `mountTerminal` in an always-present host. Same-target
  Refresh preserves it; stale identity/original target changes retire it. Recheck
  generation after render readiness before calling the factory or `restore()`.
- Keep current HTTP actions, CSRF/actor/provider checks, timeouts, native Copy,
  pending-action guards and uncertain-outcome handling. Do not move native operations
  into lifecycle hooks or change the singleton terminal wire in this port.
- Preserve drafts through ordinary reactive updates, view switches and Hide; retain
  deliberately documented explicit reset boundaries. Do not continuously assign
  input values in a way that loses selection/composition. A key review remains bound
  to its reviewed revision; stale data cannot silently authorize Apply.
- Move the JSON reader to its API owner and remove replaced DOM helpers/listeners
  in the same commit. No old/new-renderer production switch.

**Exit:** port every existing drawer action/race assertion, including hidden/rapid
click refusal, anonymous/mismatched/expired context, absent/stopped/incomplete/unknown
state, owner/member differences, Create/Join/Start/Stop, keys/last-key confirmation,
Copy and unknown results. Pending Lit updates after invalidation/disposal must never
call the terminal factory or repaint another context. Native input nodes/unsent values
and one same-target terminal survive ordinary updates/view/Hide. Run the browser
matrix below, not only a markup snapshot.

### Step 2 — port the managed terminal controls, not the native transport

**Source port and local emitted-browser/layout checks completed.** Both production
components now use Lit. The tests preserve every previous terminal/resume assertion
and add stale-input, late loading, duplicate command and malformed-frame checks.
CLI compatibility and installed acceptance remain separate, not inferred passes.

Convert presentation in `sodaspaces-terminal.ts` to `<soda-terminal>`. Preserve the
actual current facade: `started`, `restore`, `retain`, `returnToWork`, `disconnect`,
`invalidate`, `dispose`, and explicit Open/End/Continue/Keep controls.

- Keep current session lookup/create/attach, lazy xterm, original account binding,
  OSC/input/frame/queue/resize limits and bounded attach-only retries. Registration
  and mounting stay inert. Authorized restore can attach a known ID, never create.
- HTTP `terminal-session` End acknowledges `ending`; it is **not** a socket close
  frame or proof that native cleanup is finished. Preserve this distinction in the
  port; step 3/4 adds the designed per-ID outcome display/confirmation.
- Keep stable xterm/screen/socket identity through Lit updates, view switches and
  Hide. Actual detach/disposal releases browser resources but does not send End.
  Pagehide retires the document; BFCache/native navigation reauthorizes before exact
  attach. Render/lifecycle methods must not perform automatic Return.
- Preserve late-event/generation guards and exactly-once cleanup through overlapping
  End/dispose/socket-close paths. Automatic readiness cannot steal native input focus.
  Keep shell Escape/Ctrl-C/D and Ctrl+Shift+Enter out to controls.

**Exit:** all terminal and resume assertions execute against emitted Lit components
in the correct browser realm, including 401/403, invalid/oversized IO, retained/
uncertain/elsewhere/ended state, lost creation reply, no fallback/replay, cleanup and
late readiness. Exactly one renderer/socket survives repeated same-document updates.
These tests are browser/transport-double evidence, not new native tmux proof.

### Step 3 — real concurrent session and collection contracts

**Source implemented and locally checked.** Exact API contracts and bounds now live
in [the API guide](dashboard-api.md#exact-session-metadata-actions-and-creation-outcomes).
The collection scans at most 128 associations, publishes at most 32 rows/64 KiB, and
uses four request slots, sequential IO and bounded inspection time. The old singleton
route returns 410; the browser uses exact IDs/correlations and observes cleanup
receipts. Original native/helper wire is unchanged; no native concurrent/CLI acceptance
or rollout is claimed. The following requirements record this slice's contract.

Revise `internal/web/terminal.go`, `server.go`, Stop/logout/shutdown callers and
focused Go tests before making `+` create concurrent sessions. Update the API guide
and TypeScript validators together when these endpoints actually land.

1. Replace the singleton key with an ID-keyed registry. Each entry retains its
   immutable context/session token, actor, project/repository, original login,
   creation/hard expiry and native owner. With only 64 live/reserved slots, bounded
   scans suffice for project/context cancellation; no synchronized permission indexes.
   Preserve 128 pending transports, one writer per ID, no eviction and reservation of
   uncertain native outcomes. Remove the **project-wide** pre-upgrade writer refusal;
   enforce the per-ID refusal after the bounded authenticated first frame instead.
2. Add protected GET/POST metadata/actions at
   `/api/environments/{id}/terminal-sessions/{terminalID}`, plus GET `/api/spaces`
   for the authorized project/session collection (all under `/-/soda/`). These are
   implemented source routes, not installed proof. Keep the existing WebSocket route and
   separate create/exact-attach semantics. Names are optional bounded
   metadata (80 Unicode code points, no controls), never shell input or tmux targets.
   Rename must independently authorize the original context/project/account.
3. Correlate a create attempt with a random 32-hex `request_id` carried in its
   authenticated create frame and retained on the server's reservation. A
   protected `/terminal-attempts/{requestID}` read can find **that attempt**, not the
   newest terminal in a project.
   Reject duplicate active correlation within the same context; it is neither a
   capability nor an automatic replay/idempotent-create service. If the response or
   record is absent, preserve uncertainty—never infer no native effect and retry.
4. Metadata must expose actual effective/hard deadlines, attachment and lifecycle
   states. Add a default-if-unset Hide action distinct from explicit Keep, so Hide
   does not overwrite a shorter/longer existing deadline. Bind Hide/active Return
   to the current attachment generation; a delayed request from a departed window
   must not change a successor writer's lifetime. This is a bounded attachment
   locator checked with normal context/CSRF authority, not a bearer credential.
   End returns `ending`, then the client observes that ID's result. Retain a
   bounded, short-lived, authorized cleanup receipt only after native acknowledgement
   (at most 128 receipts, five minutes, capped by original authentication); free its
   native slot then. An absent/expired receipt or backend restart is **unknown**,
   not cleanup proof. No durable terminal history, job journal or restart adoption.
5. Add a dedicated bounded Spaces collection; **do not** make missing `repository_id`
   mean catalog on the existing repository API. Enumerate legitimate Soda associations
   with bounded DB scan/results, response size, provider/helper concurrency and time.
   Reuse current authority helpers before exposing names, rows or counts. Existing
   own-member degraded observations are not launch/attach permission; unavailable
   authorization hides elevated/session data. Denied projects are not placeholders
   that disclose private names/counts. A scan/time/response limit reports incomplete
   or unavailable, not a complete empty result. Define and test these concrete bounds
   in the endpoint commit; do not call unbounded `Store.Projects` and fan out blindly.
6. Stop still closes admission and cancels **all IDs of that project**, across
   contexts; another project survives. Logout/rotation cancels that original context
   or token, including pending transports. Snapshot registry data under its lock,
   do bounded external authorization/inspection without new long global critical
   sections, then recheck the same binding before dispatch or publication.
7. Replace the singleton browser metadata/`pending` locator caller in the same
   source slice. Do not keep an old endpoint that arbitrarily chooses one session.
   If an already loaded old client is incompatible, fail explicitly and require a
   fresh page; never silently select another ID. No compatibility shadow registry.

**Exit:** two same-project and one other-project session operate independently in
handler/helper doubles. Test same-context other-window writers; wrong context/actor/
project/ID; aliases/malformed input; simultaneous create/admission/Stop/logout; denied/
unavailable collections and no unauthorized helper inspection; label bounds; correlated
lost results; actual-deadline caps; per-ID End/cleanup uncertainty; capacity reservations
and confirmed release. Native guard remains per-ID and must pass the later real journey.

### Step 4 — one real Lit workspace in both entry points

**Source implemented locally; no rollout.** See the leading handoff for exact checks
and synthetic-peer versus native evidence. The following remains the behavior and
acceptance contract, not a claim that installed concurrency/CLI acceptance passed.

Deliver the first usable vertical slice, initially one visible pane with multiple
tabs. Do not wait for every drag/layout enhancement to exercise real integration.

- Extend `<soda-spaces>` from its current single environment into the shared workspace:
  authorized project/session navigation, exact-ID working-set children, select,
  explicit create/rename, Hide, Continue/Keep and confirmed End. Extract existing
  management/access controls into the named-project component now, without copying
  requests, retaining a second renderer or introducing a generic services package.
  Pending results/drafts stay bound to the selected management target; switching
  details never retargets an in-flight write or invalidates sibling terminals.
- Add `internal/web/spaces_page.go` and one Soda-owned Go template alongside the
  existing `spaces.go` collection. HTML GET derives the
  Soda actor server-side. Anonymous/expired access offers explicit Connect; grant/
  provider failure is not a fake empty workspace or redirect loop. API calls retain
  expected-user/CSRF/Origin guards. Data is authorized before server rendering or JSON
  disclosure. Render escaped context, not executable inline JSON or native cookies.
- Add fixed `destination=spaces` on the existing login route, with exactly one value
  and no simultaneous `repository_id`. Store a boolean Spaces-return flag in the
  existing OAuth transaction through an append-only migration, default false for
  existing rows. The callback maps that flag only to configured-origin
  `/-/soda/spaces`, never a caller URL or browser-storage destination. Do not revive
  historical `return_path`. Omitted destination preserves repository/home returns;
  unknown/duplicate/ambiguous intent fails and old pending rows gain no new intent. Preserve encrypted grant bytes/key,
  PKCE/state, rotation and logout-winning finalization. Rehearse populated-v5
  preservation and wrong-key rejection before any separately approved delivery.
  No OAuth client edit, secret regeneration or additional consent is needed for Spaces.
  Update API/credential and retired-frontend/packaging assertions only for this bounded
  HTML exception, not by removing the old-route and credential-preservation guards.
- Add the native `custom/extra_links` Spaces link after Explore **with** the working
  handler/page, preserving appearance controls/hidden navbar. Mount the same Lit root
  through a thin page entrypoint. Replace the repository-only facade input with a
  discriminated native-page/Spaces-page bootstrap contract; page mode does not invent
  a repository or treat an omitted native actor as a match. Anonymous native context
  still offers only explicit Connect. Root `/`, `/app/` and retired forge APIs do not
  become new aliases. No native Go handler/context import or client-side router.
- Give only the Spaces HTML route a reviewed CSP permitting its fixed local script,
  style/font/image and same-origin API/WebSocket resources. Keep API/avatar protections,
  framing denial, no-store/private responses and no-referrer where appropriate. Do not
  enable a global permissive CSP, unsafe HTML/eval, inline executable bootstrap or CDN.
- Stage the necessary canonical fonts/palette and shared component/page CSS; light DOM
  must work without assuming native scripts or a fabricated Fomantic page context.
  Native clipboard delegation remains on native pages; the Go page needs its own small
  explicit Clipboard API action with the same validated value/failure behavior, not
  Forgejo's entire script runtime. Native/system appearance is not shared auth state.
- Version and bound per-window locator storage. Import only valid legacy exact IDs
  after fresh metadata checks, never the old singleton's `pending` marker as a guessed
  session. Store no names/transcripts/CSRF/queued input; names come from authorized
  metadata. Storage failure still allows live use, with no promised reload restoration.
  Duplicated windows may inherit locators; one-writer refusal wins, not a client lock.

**Exit:** real Go routes plus emitted Lit UI expose two sessions in one project and
one in another through both surfaces; API/helper doubles remain labelled as such
until native execution. Test fixed-return/state/expiry/logout races, anonymous and
mismatched actors, denied/unavailable/no-match views, no implicit create/join/start,
key/management regressions, old storage/client refusal and native form preservation.
End confirmation shows the named account/project and follows metadata, not transient
text or socket-close ordering. The page and drawer have one implementation of actions.

### Step 5 — full layout and native-left/drawer-right behavior

**Locally source-complete:** 5a (`9ad7847`), 5b (`54686d0`), shared chrome/compact
adapter (`d2821fa`) and the subsequent journey/Go-page/geometry closure recorded in
the handoff. `bun run test:layout` now owns the 20-case measured synthetic fixture
and is included in `bun run test`; `test:spaces-page` consumes actual Go handler
HTML/CSP with emitted assets and synthetic read-only API responses. Current installed
journeys use scoped shared controls and exact `terminal_actions: ["create", "end"]`
for their bounded single-terminal path. They were ported and locally exercised,
**not run against an installed candidate**. Native form integration, physical-device
keyboard behavior, real concurrent processes and selected CLIs remain step-6 proof.
The requirements below describe the implemented slice, not instructions to repeat it.

The following commit-sized slices record the implementation from `18aceb2`'s shared
workspace. Each slice includes its own callers, focused tests and handoff;
local implementation/testing does not require another design approval. None grants
native execution or deployment. Do not add a second workspace, framework or API.

#### 5a — layout model and locator migration

**Owners:** new pure `sodaspaces-layout.ts`; storage IO/restore and live handles stay
in `sodaspaces-drawer.ts`. Wire the model into the existing one-pane UI immediately,
not as dormant scaffolding. `sodaspaces.ts` retains native open/width/resume hints.

- Use a typed binary split tree: stable pane IDs, axis/desired ratio and two children;
  leaves contain ordered owner keys and a selected key. Keep focused-pane identity
  and temporary maximize separately from the desired tree. One owner occupies at
  most one leaf. The locator table holds exact ID **or** correlated pending attempt,
  scoped to its environment; live bindings/handles remain solely in the workspace.
- Preserve the existing local owner key while a new attempt acquires its request ID
  and later its terminal ID. Promotion updates one locator, not the host, tab or
  native binding. Serialize only known locators, excluding unsent New drafts from
  saved references without executing them on reload. Confirmed cleanup removes its
  matching saved locator/reference together; an accepted End alone does not.
  Pane membership supplies open-tab order/visibility; do not maintain a second flat
  drawer order or synchronized hidden flag. Locators remain retained even when hidden
  outside pane membership; unavailable metadata never removes a saved pane placement.
- Plan a browser-only **v2** record at `soda-spaces:v2:<actor>` containing locator
  records, pane tree/selection and bounded sidebar preference. Bound encoded input
  and output to 32 KiB, 64 retained locator records, 64 leaves/127 tree nodes and
  depth at most 64. The structural ceiling follows the existing working-set capacity;
  geometry is the ordinary split limit, not an arbitrary four-pane product cap.
  Empty panes consume only UI structure, never native slots. Validate unique IDs,
  canonical locators, references, finite ratios strictly between 0 and 1, node shape
  and selected membership before traversal/publication. Store no names, login or
  permission snapshots, transcripts, credentials, attachment generations, queued
  input or DOM coordinates.
- Prefer a valid v2 record. Only when that key is absent, import valid v1 entries
  in stored order into one pane; preserve hidden entries and exact correlations.
  V1 has no saved selection: choose the first nonhidden entry in that stored order,
  never a newest server session. Validate before writing; leave v1 untouched if the
  v2 write fails. After successful promotion, v2 alone owns future writes; do not
  dual-write or import v1 again over a present v2 record. Retained old-format bytes
  are not a synchronized fallback or permission for old assets to adopt sessions.
- Malformed, oversized or future-version records are retained unchanged, reported,
  and never overwritten by an empty/default workspace. Storage failure allows live
  use with no restoration promise. Missing rows in an incomplete collection and
  null/expired receipts cannot discard locators or prove cleanup. Preserve pending
  and hidden records across partial restore and asynchronous metadata publication.
  Legacy singleton exact-ID import still needs fresh metadata; legacy `pending`
  remains unconfirmed, not a migration shortcut into creation.
- Persist deliberate selection/layout changes, not a compact Terminal-visible flag.
  Reauthorize before attaching any restored ID; hidden/unresolved entries are not
  implicitly attached by parsing/listing. Duplicated windows may inherit locators;
  the server's one-writer refusal wins. No browser lock, BroadcastChannel layout
  synchronization, durable backend registry or new SQLite migration is needed.

**Exit:** round-trip/migration tests cover exact and pending entries, hidden work,
selection, late promotion, duplicate/cross-project aliases, missing references,
malformed trees/ratios, limits, future versions and denied storage. Emitted-browser
reload/partial-read tests preserve exact locators with no create or automatic Return;
same-document migration/metadata refresh never replaces a live renderer.

#### 5b — pane operations and terminal geometry

**Owners:** pure transforms in the layout module; commands/placement in the shared
workspace; measured fit/input/control focus in the terminal owner and scoped CSS.

- Split right/below creates an explicit empty leaf. Move/reorder preserves an existing
  owner; moving the last tab collapses the empty source. Retain one initial empty
  pane when no tabs remain. Maximize/restore preserves the exact desired tree;
  consolidate keeps the focused selection first and stable remaining order.
  Supply labelled keyboard/menu destinations as well as pointer tab/edge dragging
  and keyboard/pointer separators. Deliberate resizing updates desired ratios within
  measured child minimums; compact projection never overwrites them. Do not add bulk
  Hide/End, floating windows or stacked-pane mode.
- Position existing hosts in the stable flat layer using measured pane screen
  rectangles. Never put live terminals inside a recursively rendered pane tree or
  reparent them during a move. Lit keys alone do not prove cross-parent continuity.
  If keyed chrome needs `repeat`, export/map/test it through the one shared runtime.
- Separate layout selection from attachment/creation commands: the current
  `selectSlot()` calls `restore()` and cannot be reused blindly for transforms.
  Selecting an unattached exact locator may deliberately authorize attach; split,
  move, resize, maximize, search and projection changes cannot. Pending operations
  capture their original owner/project and intended pane; moving while pending
  must not dispatch twice, retarget a result or remount on ID promotion. Replace
  workspace-wide mutation blocking with bounded action/target guards where needed
  so another terminal remains usable; do not introduce jobs or a generic store.
- Give the terminal owner only the small presentation-only facade needed for
  visibility/fit/control focus. Readiness, fonts and ResizeObserver/visual viewport
  callbacks recheck generation, connection and visibility. Fit only visible nonzero
  geometry; hidden siblings retain last valid dimensions and sockets. No observer
  loop that recreates a renderer, no zero-size resize and no hidden-input route.
- Use actual xterm/font cell metrics and available screen area **after** tab/context
  chrome, warnings, padding and scrollbars. A proposed split requires each child to
  fit **56 columns × 12 rows**; aim for **80 columns** where viable. Reject an
  unviable split with an explanation. On later shrink, show the selected pane plus
  **Panes (N)** and retain the desired tree/ratios for widening. Do not certify the
  minimum from configured `80×24`, the old 650px assertion or the provisional 216px
  floor. Compact single-terminal views may be narrower than 56 columns; that is not
  permission to split them or shrink text. Record real mobile/keyboard limitations.
- Preserve tab/tabpanel associations, visual and keyboard order, clipping and
  overlay hit testing. Only the deliberately focused visible terminal receives
  input. Native typing, shell Escape/Ctrl-C/D, browser shortcuts, selection and paste
  must stay native. Ctrl+Shift+Enter reaches a stable selected-session control even
  when End moves into a menu; pending/disabled End must not trap keyboard focus.

**Exit:** real-xterm browser tests record host/screen/renderer/socket identities,
constructor/disposal counts, actual fitted cells and sent resize/input frames across
all transforms, font/zoom changes and late callbacks. Split/move/maximize/consolidate
send no create/attach/lifetime actions; no sibling input, focus theft or zero fit.
Pure tree tests alone and screenshots alone do not satisfy this slice.

#### 5c — full-page and drawer projections

**Owners:** shared workspace/templates/CSS and thin page bootstrap; reuse existing
project and terminal command owners. No second page session list or API caller.

- Replace step 4's stacked project/session/lifecycle rows with the selected compact
  chrome. Full Spaces gets its resizable/collapsible project/session sidebar, local
  pane tabs, search and overflow menus. Drawer gets flat open tabs derived in stable
  pane/tab order, a right-content-only **Sessions** view and Back to terminal, not
  a permanent sidebar or recursive layout editor. Selecting a drawer tab updates
  its existing group selection without moving it; switching surfaces preserves the
  full-page tree. All and explicit **This page** filter affect navigation only.
- Search only authorized project/session labels; filtering never hides, closes or
  retargets open panes. Preserve incomplete/denied/unavailable/no-match distinctions.
  Hidden sessions appear in the same authorized navigation, not step 4's separate
  hidden/existing button catalogs. Do not discard opaque unresolved locators merely
  because their metadata is unavailable. Attention filters/counts are **step 6a**;
  do not ship fake zero counts or placeholders as though signals already exist.
- Use the same explicit project/name New chooser in both surfaces, with original
  account/full project visible before dispatch and optional bounded metadata name.
  Default from the selected session, not an invisible native-left repository.
  Keep the chooser's pending target independent of later navigation; no automatic
  Join/Start, agent launch, worktree creation or command copy/broadcast.
- Put Rename, Keep, eligible Continue, Hide and **End terminal…** in the selected
  session controls. End confirmation names session/project/original account, warns
  of in-process loss, defaults focus to Cancel and observes exact cleanup metadata.
  Do not preserve the always-visible step-4 End checkbox merely to keep selectors.
  End acceptance/closed sockets/deadline passage do not prove native cleanup.
- Named Environment/Access views reuse the existing project-control instances and
  drafts; they never take their write target from whichever pane is now focused.
  Preserve explicit Refresh reset behavior and uncertain writes. Stop stays here,
  with shared-impact confirmation, not next to New or Hide. Keep native Copy on
  native pages and the page's explicit Clipboard API action/failure handling.
- The destructive `Use repository on the left` replacement control is already gone.
  The current `Repository on the left` is only a details entry point; fold it into
  explicit named details/This page navigation without replacing the working set.
  Keep original account/full repository visible in trusted chrome, including mixed-
  project tabs; shell titles and untrusted output never supply that identity.
- Stage every new module/style through `forgejo-payload.json` and both canonical
  preview projections. Test the real Go HTML/actor bootstrap and emitted page module
  together, not only direct `mountSodaspaces({kind:'page'})` in a fixture. Preserve
  fixed OAuth return/CSP and escaped context; import no native globals or Preact.

**Exit:** both surfaces expose the same sessions/actions with usable keyboard/touch
controls, stable selection, truthful errors and preserved project drafts/terminal
owners. Update local fixtures and affected installed-journey selectors in this slice;
retain their action/authorization assertions rather than bypassing controls via APIs.

#### 5d — compact surfaces, native coexistence and journey ports

**Owners:** native adapter/outer CSS, shared workspace's compact views, and the
existing browser/installed journeys. This closes the step-4 probe-port debt.

- Keep 50/50 by default and desired 35–65% resizing, clamped to a usable native left
  region (design target at least 480px) plus measured terminal minimums. If no viable
  ratio exists, use **Forge / Terminal**, not an unusable split just above 800px.
  Preserve desired widths/tree/sidebar preferences while temporarily compact.
- Compact switching changes visibility in the same document, preserving native form
  nodes, unsent values, scroll and every live terminal owner. Expose only the visible
  surface to focus/accessibility; no desktop focus trap. Mobile keyboard/visual
  viewport, orientation, zoom and fonts must leave the prompt and controls usable.
- New native navigation/Back/BFCache defaults to Forge when compact; explicit drawer
  intent can reveal Terminal. Restore locators independently of that visibility.
  Open in Spaces remains ordinary navigation: cancelled beforeunload changes nothing;
  completed pagehide detaches, not Ends. New documents reauthorize exact surviving
  IDs and selection, never create replacements or automatically Return them all.
- Sessions/details/maximize/compact transitions are not Hide. Explicit drawer Hide
  retains only this document's owned attachments with writer generations; reopening
  can Return only the deliberately selected eligible session. Show actual capped
  deadlines and uncertain results. No layout callback renews abandonment.
- Verify native code/diffs/PR comments/forms, selection/clipboard, notifications,
  account menus and modals at actual left-pane widths. Retain the repository-action
  1000px **pane** breakpoint and supported scoped adaptations; no native form
  cloning/reparenting, iframe, hidden half-page or overlay under the drawer.
- Finish ports of `tests/installed/sodaspaces.ts`, `sodaspaces-management.ts` and their
  input/fixture tests. Replace singleton IDs/old view and Open/End assumptions with
  named-project/session controls and exact IDs. Keep real OAuth, both actors, Copy,
  management, stale-account and native-form checks. Retain private-file, occupied-run,
  target/protocol/action gates; six sessions or shared Stop are not covered by an
  old single-terminal approval. Author/run local driver fixtures now; installed
  execution waits for section 5's target-specific scope. Add no permanent old/new
  renderer selector or second installed scenario in outside support tools.

**Step 5 source exit:** 5a–5d are integrated in one shared Lit workspace; the measured
width/theme/keyboard matrix below and migrated local journey fixtures pass on the
actual candidate. No unported production/installed caller is silently left for a
final sweep. Same-document layout preserves live DOM/renderer/socket; document
replacement preserves exact native IDs, not DOM. Native proof is still separate.

### Step 6 — truthful attention and finish

#### 6a — observed unread and lifecycle attention

**Implemented locally after `f3f348b`:** typed current-generation observations,
one unread flag per owner, observational lifecycle/deadline reasons, All/Attention
and deliberate Next. The mounted workspace owns a cancellable 30-second UI clock
and at-most-once-per-minute existing GET refresh while visible. No unopened-navbar
poller, output-driven request or lifetime renewal was added. See the leading handoff
for actual checks; native/CLI proof remains 6c.

Apply the [post-step-5 cleanup](#cleanup-after-step-5) to the shared presentation
before expanding it, preserving any already implemented attention work. New attention
views use the same canonical tokens and checked typed composition; they do not
introduce another local palette, giant template or action/state owner.

**Owners:** terminal owner emits bounded typed observations; shared workspace owns
per-owner unread state and stable authorized navigation. Reuse existing metadata
and error contracts, not DOM/status-text scraping or another transport subscriber.

- Emit an output observation only after a valid frame from the current attachment
  passes existing bounds. Carry identity/generation, not a transcript/snippet; keep
  one unread flag per retained owner and coalesce rendering, not one DOM update per
  output chunk. No new replay buffer, stored unread history or registry of agents.
- Output while a tab is not the selected visible terminal in a visible document
  marks unread (including behind Sessions/details or compact Forge). Merely focusing
  a native comment beside a visible terminal does not change session selection.
  Deliberate viewing clears unread after the view is available; metadata refresh,
  automatic restore or output must not clear it or select a pane. A read marker
  never clears an unresolved lifecycle problem.
- Attention uses actual blocking connection/lifetime/cleanup observations, including
  unconfirmed outcomes and imminent expiry derived from the observed effective
  deadline (five-minute UI warning, not a changed lifetime). Keep connection loss,
  attached-elsewhere, ending, unconfirmed and acknowledged-ended distinct. Expose
  bounded typed reasons where the current generic terminal message loses those
  distinctions; never infer them by parsing displayed prose. Stale/unavailable
  metadata is not a successful empty list, a live promise or cleanup proof.
- Add All/Attention and explicit Next attention to the same page/drawer navigation;
  count distinct currently authorized qualifying sessions, not events/unknown agent
  totals. Keep stable order, wrap deliberate Next once and explain when none qualify.
  Ordinary output alone never enters Attention. Clear private observations on lost
  authority; retaining an opaque uncertain locator is not authority to display it
  as another context's live session. Missing receipts still cannot release a slot.
- No focus theft, input injection, sound, desktop permission prompt or poller on an
  unopened native navbar. Any needed mounted-workspace observation refresh stays
  bounded/cancellable and uses the existing caller; output/polling never Returns.
  Semantic Working/Waiting/Finished and terminal notification protocols remain
  unavailable until a separately selected, source-reviewed, explicitly enabled
  session-bound adapter exists. No CLI prose heuristics, wrappers or profile changes.

**Exit:** browser tests cover noisy hidden output, multiple visible panes, deliberate
viewing, metadata staleness, late generations, lost authority, stable filtered order,
correct counts and navigation without mutation. Ordinary logs never become Attention
or fabricated agent completion; no test treats a socket close as native cleanup.

#### 6b — close candidate source and installed-journey coverage

**Locally implemented and checked:** the existing guarded journey now selects a
separate six-session/two-project matrix only with its additional exact private scope.
Both actor journeys run against emitted components and synthetic peers in local
fixtures. Native SSH/process observations and selected CLI/browser-versus-SSH
scenarios are authored, not executed; see the [input/effect contract](native-validation.md#integrated-six-session-workspace-matrix-authored-not-installed-proof)
and leading handoff. Combined local checks include tokens/checker/views/source
mapping, attention, driver fixtures, Go/race and canonical staging/preview bytes.
Historical failed native assertions and 6c's visual/native requirements remain.

Reconcile current API/credential/installation and feature guides with the delivered
source contracts, including schema v6 and the bounded Go HTML exception. Retain
historical execution records. Remove genuinely superseded helpers only as their
callers move; legacy locator preservation/refusal is not dead code to prune blindly.

Close the [frontend improvement requirements](frontend-improvement-plan.md#9-validation-and-completion)
on the actual candidate: canonical tokens consumed on both surfaces, source checks
against new raw visual definitions, a required analyzer with verified compiler
resolution and negative fixtures, and readable typed presentation with preserved
owners. Keep the known analyzer event-parameter gap documented and behavior-tested.
Source moves must include compiler/build/payload/import and caller changes together.

Extend the **ported product-owned journeys** with the six-session/two-project matrix
and exact per-ID correlation, writer, retention and cleanup checks. Add the actual
Codex CLI, Claude Code and Pi browser-versus-SSH scenarios with declared versions,
inputs and effects. Native execution is 6c, not a side effect of authoring these tests.
Review input gates so past fixture/probe approvals cannot authorize additional
sessions, installs, lifecycle/fault injection, provider use or cleanup. Do not drop
failed native assertions, substitute a shell smoke, or add a parallel acceptance
framework. Record source checks and remaining native cases separately.

**Exit:** local build/type/browser/Go/race/payload regressions pass; page bootstrap,
new layouts/attention and installed-driver fixtures all exercise current callers.
No duplicate action implementation, stale-selector bypass or unexplained skipped
essential case. Source readiness is not native/CLI acceptance or release approval.

#### 6c — scoped native and selected-CLI proof

Use section 5's build/delivery gates and existing authorized tools, with new exact
scope where required. `b8af68c` diagnosed/fixed the retained first-input loss caused
by tmux startup flushing, then passed `TestInstalledTerminalBoundary` and the native
two-actor matrix with independent named cleanup. Both approved targets received the
backed-up candidate; native browser/BFCache and retained SSH/PTY checks passed. Keep
the earlier failures and exact revision/target distinctions in the handoff. The
broader cases below, including selected CLIs, are not all completed by that evidence.

On the approved candidate/target, exercise both real surfaces, six sessions across
two projects and both account boundaries. Verify actual PID/start/memory/editor/build
continuity through layout, native navigation/reload and bounded connection loss;
End only the named managed service, independently checking owned cleanup and sibling
SSH/services/files. Test one-writer refusal, Stop/logout/rotation/expiry and approved
lease/helper/guard failure/uncertain outcomes. Nothing authorizes unrelated cleanup.

For **each** selected CLI, record actual version, native shell/tmux/terminfo and browser
context; test Unicode/cursor, alternate-screen redraw, mouse/paste, resize/interrupt,
scrollback/history/selection, long streaming output and reconnect/retention, with
ordinary SSH as comparison. Use restricted personal credentials and explicit provider
scope; no borrowed Soda grant, credentials in output/screenshots, automatic agent
installation or profile rewrite. Compare ordinary SSH and personal tmux without
forcing either into managed Soda sessions or claiming equivalent browser behavior.
Missing tools/credentials/approval mean not run, not passed. A real compatibility blocker returns for a product decision; it does not
silently select a new renderer/backend or remove the browser terminal.

**Exit:** retain exact candidate/architecture/target results, failures and exclusions.
Only observed native scopes are accepted; x86_64 does not imply aarch64 or direct
laptop routing. Retained rollout still requires separate approval. Full product,
operator/provider, keyless onboarding/Git and Rocky-root decisions keep their scope.

## 5. Required verification and delivery boundary

**Testing realm:** production Lit tests already run emitted modules in sandboxed
Chromium; JSDOM remains appropriate for the native adapter with injected content,
not foreign-realm Lit constructors. Keep pure layout/parser tests in Bun. Extend
`test:lit` with the new component/geometry/attention contracts; await actual child
readiness, fonts and layout, not arbitrary sleeps. No patched HTMLElement, second
framework, removed races or essential cases made optional. Each slice must exercise
its changed production callers; later native permission is not a reason to leave
source journey ports or deterministic local failure checks unfinished.

| Layer | Required checks (actual results belong in the handoff, not inferred from this table) |
| --- | --- |
| Local source/build | Frozen pinned Bun inputs with dependency lifecycle scripts disabled; `bun run typecheck`, `test:frontend`, `test:forgejo`, `test:lit`, retained Cockpit tests, Go tests/races for changed web/store/host callers. Required browser cases fail when the browser cannot run. |
| Template diagnostics | Required analyzer alongside the product compiler, under the root single lock; actual-source inventory, positive fixtures and individual negative binding/markup cases. Unknown-event checks explicitly enabled; required warnings/errors fail. Separate analysis compiler and callable-event limitations documented. |
| Design tokens | Canonical visual roles consumed in page and drawer, including fallbacks/inline presentation. Source checks detect new raw values; computed appearance, density, light/dark, focus and measured geometry are verified. Token availability must not activate full-page native-shell styles in the drawer. |
| Storage/layout | V1→v2 migration, pending promotion, hidden/unresolved records, stable selection/order, malformed/bounded/future data and storage failure. Pure transforms send no effects; actual reload/partial metadata tests verify unchanged exact targets and no replay/Return. |
| Component/geometry | Run `bun run test:layout` (`tests/frontend/drawer-layout.test.ts`, `SODA_DRAWER_LAYOUT=1`) and full-page workspace tests with real xterm/synthetic IO. Observe actual fitted columns/rows and transmitted resize frames after fonts/chrome settle, not a 216px proxy. Record host/screen/renderer/socket identity and disposal counts, focus/input isolation, clipping/overlays and screenshots through all pane/surface operations. |
| Widths/themes | Light/dark; page 1920/1440/960/800/720/390/320; drawer 1440 with viable 35/50/65% ratios and widths just above/below the measured compact threshold. Long labels, warnings, menus, font load/change, zoom, orientation and visual keyboard viewport. Assert disabled unviable splits and exact desired-layout restoration, not tiny fonts or forced ratios. Simulated viewport checks are not real-device keyboard proof. |
| Native coexistence | Code/diffs/PR comments, forms/unsent values, native menus/modals/notifications, Copy, cancellation of beforeunload, Back/Forward and signed non-repository resume. Check the 1000px pane breakpoint, not just viewport size. |
| Payload | `build:forgejo`, production payload/inventory and focused Python/Go staging checks. Exactly one runtime/directive instance from both public asset roots; missing/unreadable runtime and new component assets fail. AppSubUrl-safe **assets** do not enable currently unsupported backend subpath deployment. |
| Existing preview | `build:preview` already projects canonical branding and the complete public tree, including root modules/styles and locked xterm. Use isolated `--out`/`--public-out` for local checks; compare emitted and served bytes and both runtime import paths. Updating a live mount/service needs its own scope. No copied asset list, stale-byte test, mockup server or extra tunnel. |
| Installed-driver source | Port `sodaspaces.ts`, `sodaspaces-management.ts`, inputs and local probe fixtures in 5c–5d, then extend native/CLI scenarios in 6b. Preserve opt-in target/action/protocol gates, private inputs, replay refusal and negative assertions. Do not run unported probes or replace UI commands with direct-API shortcuts. |
| Native page visuals | `scripts/screenshot.ts` and [capture rules](screenshot-capture.md), only with the applicable fixture/profile scope. Incoming localhost:3300 screenshot/scaffold evidence is not integrated backend/project proof; 33443 remains the isolated full application origin. |
| Installed terminal acceptance | Under exact target/action approval, six sessions/two retained projects: same PID/start/memory/editor/build through navigation/reload/network loss and both surfaces; real history/selection/paste/resize/interrupt; per-ID End preserves siblings/SSH/services; duplicate writer, Stop/logout/rotation/expiry, lease/helper/guard failure and uncertain cleanup. |

#### Evidence and delivery gates

These are exits within the existing sequence, not additional numbered milestones or
authorization to execute a target. Follow [native validation](native-validation.md),
[installation](installation.md) and the leading handoff for exact permissions/state.

1. **Local source ready:** 5a–5d and 6a–6b have their actual focused and combined
   results, including migrated installed-driver fixtures and real Go-page/emitted-
   bootstrap integration. Retain early failures and label synthetic HTTP/socket
   evidence. No passing count from `18aceb2` or an older branch substitutes for new
   layout/attention coverage. Missing native approval does not block this local work.
2. **Candidate prepared and fixture delivery authorized:** use production exact-
   revision native build/check/export, without a sibling-architecture barrier.
   Before scoped affected-component delivery, review exported module/template bytes,
   cache/mixed-client handling and matching backend/schema-v6/config/grant-key/OAuth-
   client state. Browser v2 layout is separate from SQL v6. Review preserved-state
   rehearsal and fresh matching backups before mutation; verify served bytes after
   approved fixture delivery. Build/export success alone permits no install/restart
   or live preview refresh.
3. **Native scope accepted:** run 6c only on approved targets/actions and retain
   independent process/cgroup cleanup, original roots/accounts/keys/later writes,
   client path and CLI results. Fix the recorded raw-output failure rather than
   relabel it; one successful shell/SSH path does not validate Codex/Claude/Pi in
   the browser. Keep failed, unrun and partially reached cases explicit. Native
   x86_64, aarch64, real mobile keyboards and laptop reachability are distinct facts.
4. **Retained rollout separately approved:** review the affected-component procedure
   and its interruption with the operator/project users, deliver exact tested bytes
   only to the named target, then repeat relevant preservation/access checks. Backend
   replacement/shutdown cancels its runtime session ownership; navigation continuity
   is not daemon-upgrade resurrection. Old backups are not lossless rollback of
   later writes. No first-install-as-updater, install-on-Open, automatic replay or
   cleanup beyond exact authorized run-owned resources.

The undeployed Rocky 10.2/existing-root decision, `soda-test`, operator/provider and
console gaps, keyless Join, agreed Git credentials and other appliance work remain
in the [leading sequence](sodaspaces-plan.md#remaining-work--ordered), not hidden
prerequisites or additions to pane layout. This revision grants no project/package/
capability change, new fixture, credential use or lifecycle/fault-injection action.

**Completion:** distinguish source-ready, scoped-native-validated and delivered.
Both surfaces must share one ID-bound workspace with real layout/attention behavior;
applicable native/CLI checks must have their own results before claiming acceptance.
A drawing, rendered UI, source pass or scoped milestone is not final product/release
acceptance, and unresolved compatibility concerns are not a silent terminal redesign.
