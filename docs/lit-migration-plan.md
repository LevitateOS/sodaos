# Lit implementation plan: Spaces page and right drawer

Implement the selected [Spaces page](spaces-design.md) and
[right-half drawer](spaces-drawer-design.md) with the shared Lit scaffold, not two
frontends or another preview application. The [leading plan](sodaspaces-plan.md)
owns product scope/order; this document is its **single detailed implementation
sequence for the Lit workspace**. [Lit guidance](lit.md) owns framework/build rules,
the [terminal contract](terminal-integration.md) owns native lifetime, and the
[API guide](dashboard-api.md) describes implemented endpoints, not the proposals here.

**Current source: steps 1–2's management and terminal controls are ported, with local
emitted-browser and layout checks. Step 3's ID registry/collection and caller contracts
are source-implemented with local Go/race/browser coverage. Step 4's authenticated
page/fixed return and shared multi-session workspace now have local source/browser
coverage; steps 5–6 remain unimplemented.** Xterm and the
managed transport remain imperative resources; no native lifetime redesign occurred. The canonical public preview projection is implemented; changing live preview mounts
and full installed acceptance remain separately scoped; see the handoff for exact evidence. This revises
`8f03910`'s drawer-first migration plan to incorporate the managed-tmux implementation
and the approved full-page direction/complementary drawer design. Keep the two small
rendering ports first; build the real multi-session product on them. Do not port the
superseded interactive mockup or reintroduce its grid presets/fake state machine.
The user subsequently requested end-to-end implementation and local testing.
Deployment, new appliance fixtures and project recreation remain separately scoped.

## 1. Starting point and decisions

| Source fact | Consequence |
| --- | --- |
| `c675411` supplies Lit 3.3.3, one root Bun lock and one staged core runtime | Reuse it; no React rewrite, router, UI kit, dependency upgrade or per-component bundle. Management and terminal components import Lit; the native adapter loads them on demand. |
| `sodaspaces.ts` owns native hooks, aside, divider and document departure | Retain a small native adapter; do not render native Forgejo with Lit. |
| `sodaspaces-drawer.ts` supplies real management/access actions and a terminal facade | Port these owners, guards and tests rather than replacing them with demo controls. |
| `sodaspaces-terminal.ts` implements create/exact attach, restore, retain, Return and HTTP End | The old socket `{type:'close'}` End and request-owned lifetime are obsolete. |
| Management Refresh retains a terminal with the same environment/login, but `reset()` still clears management drafts | Preserve current explicit reset boundaries in the rendering-only port; do not claim Refresh already preserves every form value. |
| `internal/web/{terminal,terminal_sessions,spaces}.go` now implements ID-keyed sessions and a bounded authorized collection | Independent sessions are backend-tested; the shared multi-session UI and native proof remain separate. |
| Native guard/tmux is already per terminal ID | Keep that mechanism; concurrency needs regression/native proof, not another PTY broker. |
| `22d8591` has isolated same-shell reload and End/cleanup evidence | Preserve it; full native probe, editor/build/history/network/failure acceptance remains incomplete. |
| Incoming repository actions are inline above 1000px **pane** width, a disclosure at/below it | Preserve real native form nodes and this correction; schematic drawings do not override native header implementation. |
| `/spaces`, the collection, names and fixed OAuth return do not exist | Implement them explicitly on `/-/soda/`, with the existing application origin (33443 on the isolated deployment). |

The shell design is now concrete: Soda-owned Go HTML with canonical branding,
configured-origin native links and an explicitly labelled server-derived Soda
account. It does **not** impersonate Forgejo's authenticated navbar, CSRF, account
menu, notifications or theme authority. The remaining work is implementation and
compatibility/security verification, not discovery of a way to run upstream Go
handlers from a template override.

## 2. Concrete owners

Paths are repository-relative; step-4 workspace/page paths below remain proposed.

| Owner | Responsibility |
| --- | --- |
| `appliance/forgejo/public/assets/sodaspaces.ts` | Native repository button/resume hook, outer split/compact surface, width, native DOM coexistence and page lifecycle. No session inventory or API mutations of its own. |
| `…/sodaspaces-drawer.ts`, `<soda-spaces>` | One reusable Lit workspace root, mounted in drawer or page mode. Owns observed session metadata, this document's working set, selection/pane tree and concrete child references. Keep `mountSodaspaces` as the thin native/test facade. |
| `…/sodaspaces-terminal.ts`, `<soda-terminal>` | One immutable terminal binding and its xterm/attachment resources, IO guards, status and explicit terminal commands. Keep `mountTerminal` and injection seams during the ports. |
| `…/sodaspaces-project.ts`, `<soda-project-controls>` (step 4) | Extract the already-ported management/access markup and commands when both surfaces need named-project details. Owns drafts, pending/uncertain writes and original target; no terminal or second workspace. |
| `…/sodaspaces-layout.ts` (step 5) | Typed pane-tree transformations and bounded locator parsing/serialization, with direct callers/tests. No DOM, network, credentials or generic application store. |
| `…/sodaspaces-page.ts` (step 4) | Thin entrypoint for the Go page: validate its server-supplied actor context and mount the same workspace in page mode. No client router or second session owner. |
| `…/sodaspaces-api.ts` | Bounded JSON reader, response validation, concrete API types/errors. Move `readSodaJSON` here from the terminal while updating both callers. |
| `internal/web/{server,auth,terminal,terminal_sessions,management,spaces}.go` | Page/collection, original-context ID-keyed sessions, fresh authorization, cancellation and existing native helper calls. |
| `internal/store/{login,migrations}.go` and focused store queries | Fixed OAuth destination and bounded reads of legitimate Soda associations; no durable terminal registry or copied provider roles. |
| Existing `internal/host/` and `project-os/` | Native account/guard/tmux/lease/cleanup authority, unchanged unless a concrete tested defect requires a scoped correction. |

The workspace's map of live child handles is **not** a second backend session model.
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
  stay unique once several terminals exist; preserve old selectors only where they
  still represent a genuinely singular control, such as the active Copy target.
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
  one parent, keyed by session ID in stable owner order. Pane chrome/layout changes
  position and visibility of those existing hosts; they do not reparent/remove them.
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
- Add `internal/web/spaces.go` and one Soda-owned Go template. HTML GET derives the
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

Implement the approved designs directly, not the superseded grid mockup:

- One pane by default; typed split tree with pane IDs, ordered session IDs, selection
  and ratios. Split right/below creates an empty view, not a shell. Move preserves the
  session host; empty groups collapse; maximize/restore and consolidate preserve work.
  Add pointer operations **and** named keyboard/menu equivalents. No arbitrary four-
  pane limit, font shrinking, cwd copying or command broadcasting.
- Keep terminal owners in the stable layer described above. Tabs/metadata can reorder
  without changing those owners. Hide inactive views from input/accessibility while
  retaining their sockets; fit only visible nonzero geometry, preserving last valid
  dimensions. Observe canvas/font/visual-viewport changes, not just outer window resize.
- Measure the design's 56-column/12-row split minimum and ~80-column target. When the
  tree cannot fit, show the selected pane with **Panes (N)**; keep the saved tree and
  desired ratios, not a destroyed/reserialized replacement layout.
- Full page gets project/session sidebar, local pane tabs and search/attention filters.
  Drawer gets all-open-session tabs flattened from the same tree, a right-content-only
  Sessions switcher, named Environment/Access views, and Open in Spaces. No permanent
  drawer sidebar, recursive drawer layout editor or second working-set list. Remove
  the old `Use repository on the left` replacement control; this-page filtering or
  explicit named-project details cannot replace the open working set.
- Retain the 50/50 native split and viable 35–65% divider, respecting native-width
  and measured terminal minimums. Reuse incoming pane/container adaptations and the
  1000px repository-action disclosure. Never clone/reparent native forms. Menus,
  notifications, clipboard and real native modals remain functional above the drawer.
- Compact Forge/Terminal switches visibility in the same document, preserving DOM,
  drafts, scroll and attachments. New native navigation shows Forge in compact mode;
  explicit drawer intent can select Terminal. On widening restore the desired ratio.
- Native navigation/Open in Spaces is normal same-origin navigation, not in-place
  SPA routing. Preserve cancelled beforeunload without detaching; completed pagehide
  detaches without End; next document/BFCache uses fresh authorization and exact IDs.
  Automatic restore does not Return. Whole-drawer Hide retains only this document's
  owned sessions; re-opening never Returns them all. Show each actual capped deadline.

**Exit:** same live DOM/renderer/socket across same-document split/move/maximize/
consolidate/search/details/compact transitions, correct visual/keyboard order and
no zero-size resize, input leaks or focus theft. Native document replacement may
create renderers, **not native shells**. Full acceptance requires the native matrix.

### Step 6 — truthful attention and finish

Implement observed background-output unread dots and actual lifecycle problems.
Keep list order stable and count only qualifying authorized sessions. Output alone
never enters Attention; viewing clears unread, not a waiting state. No focus steal,
input injection, sound or navbar poller when the workspace has never opened.

Semantic Working/Waiting/Finished remains unavailable until a separately selected,
source-reviewed, explicitly enabled session-bound signal adapter exists. Do not
infer from silence/prose, install agent wrappers or rewrite project profiles. If
selected later, bound text/rate/staleness and keep advisory signals separate from
permission, terminal cleanup and task/test success. No durable notification platform.

Remove superseded helpers/locators and update current guides/tests as each caller
moves. No second new feature plan, dead migration flags or aggregate "PASS" inferred
from a prior branch. Each step ends in a coherent commit with its actual evidence.

## 5. Required verification and delivery boundary

**Testing realm:** current Bun/JSDOM tests create separate window constructors while
importing modules once. LitElement must be evaluated/registered in the browser realm
where it is instantiated. Move rendering/action/lifecycle assertions to emitted
modules in the existing sandboxed Playwright fixture; keep pure parsers/layout in
Bun. Do not monkey-patch global HTMLElement, install a second framework, drop races
or turn essential cases into skips. Extend `test:lit` to run the converted
`tests/frontend/` contracts; await real readiness, not arbitrary sleeps.

| Layer | Required checks (actual results belong in the handoff, not inferred from this table) |
| --- | --- |
| Local source/build | Frozen pinned Bun inputs with dependency lifecycle scripts disabled; `bun run typecheck`, `test:frontend`, `test:forgejo`, `test:lit`, retained Cockpit tests, Go tests/races for changed web/store/host callers. Required browser cases fail when the browser cannot run. |
| Component/geometry | Extend `tests/frontend/drawer-layout.test.ts` (`SODA_DRAWER_LAYOUT=1`, established long timeout) and add focused Spaces layout/interaction cases using real xterm and synthetic API/IO. Record node/renderer/socket identities as well as screenshots. |
| Widths/themes | Light/dark; wide 1920 and 1440; 1440 with viable 35/50/65% splits; standalone 960/800/720; mobile 390/320; long labels, errors, font/zoom and visual keyboard viewport. Actual minimums can require compact mode rather than all three ratios. |
| Native coexistence | Code/diffs/PR comments, forms/unsent values, native menus/modals/notifications, Copy, cancellation of beforeunload, Back/Forward and signed non-repository resume. Check the 1000px pane breakpoint, not just viewport size. |
| Payload | `build:forgejo`, production payload/inventory and focused Python/Go staging checks. Exactly one runtime/directive instance from both public asset roots; missing/unreadable runtime and new component assets fail. AppSubUrl-safe **assets** do not enable currently unsupported backend subpath deployment. |
| Existing preview | `build:preview` currently copies branding destinations only. Wire changed root-level Sodaspaces modules/styles into the existing public mount through the canonical payload owner; never hand-maintain a second list or test stale bytes. Publishing to a live fixture is separate from building. No mockup server/extra tunnel. |
| Native page visuals | `scripts/screenshot.ts` and [capture rules](screenshot-capture.md), only with the applicable fixture/profile scope. Incoming localhost:3300 screenshot/scaffold evidence is not integrated backend/project proof; 33443 remains the isolated full application origin. |
| Installed terminal acceptance | Under exact target/action approval, six sessions/two retained projects: same PID/start/memory/editor/build through navigation/reload/network loss and both surfaces; real history/selection/paste/resize/interrupt; per-ID End preserves siblings/SSH/services; duplicate writer, Stop/logout/rotation/expiry, lease/helper/guard failure and uncertain cleanup. |

Fix the retained `TestInstalledTerminalBoundary` raw-output confirmation failure;
do not replace its missing two-account/native safety proof with the existing browser
same-PID smoke or new screenshots. Verify actual owned cgroup/process cleanup and
unrelated workload preservation. Native x86_64 and aarch64 evidence remain separate.

Before delivery use the production exact-revision build/check/export and reviewed
affected-component procedure, preserving DB/key/OAuth client, both original roots,
all later writes/private inputs and failed evidence. Backend replacement/shutdown
cancels managed sessions: coordinate that process-memory loss explicitly; reload
reattachment is not daemon-upgrade resurrection. No install-on-Open, old-DB rollback,
first-install-as-updater or fixture cleanup shortcut. The undeployed Rocky
10.2 candidate/existing-root decision is **separate**, not permission to erase or
upgrade projects while delivering Lit. `soda-test`, operator/provider work, keyless
Join and agreed Git credential setup keep their own scope.

**Completion:** both real surfaces share ID-bound sessions and actions through one
Lit implementation, the relevant source/browser/native checks have their own recorded
results, and remaining native/credential/operator gaps stay explicit. A rendering
port or static drawing alone is not completion of Spaces or final product acceptance.
