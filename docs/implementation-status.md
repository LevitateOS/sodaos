# Current handoff

## Frontend architecture review and implementation guide — documentation only

The user requested a consolidated agent handoff and reconciliation of the existing
implementation/design docs. [Frontend improvement](frontend-improvement-plan.md)
records the researched recommendation: retain Go/Forgejo + Lit, make canonical
design-token adoption mandatory across page/drawer, add enforced template diagnostics,
extract readable typed views and clarify source/build ownership. The Lit plan remains
the feature sequence, with cleanup after locally completed step 5 and its exits in
step 6b. Existing/current step-6 work must be preserved rather than reset.

Compared the Lit plan and related docs against `8e812dc`
(`feat(spaces): close local layout integration and journey ports`). Its step-5 closure
remains intact. Corrected stale singleton and single-terminal descriptions in the
design documents; step 6 attention/candidate/native/CLI work remains outstanding.
Lit/TypeScript authoring and both designs now carry the same token/checking contract.

Research evidence is consolidated in the guide; optional raw reports/fixtures remain
under `.artifacts/frontend-architecture-review-20260910/`. Isolated analyzer 2.0.3 with
TypeScript 5.9.3 analyzed the three actual Lit modules and reported seven review items;
the positive fixture passed and the final negative fixture produced ten diagnostics
with unknown-event checking enabled. Product TS7 browser checking passed during
research. Root single-lock analyzer integration is **not implemented**, callable-event
parameter checks remain incomplete, and no renderer migration was tested end to end.
The independent reviewers disagreed; the guide records the rationale for retaining Lit.

This documentation change includes no application, manifest/lock, generated payload,
service, retained fixture/project, native execution or deployment changes. Earlier
step-5 results below are historical evidence, not new runs for this guide.

**Documentation verification:** compared the latest commit and relevant source/plan
contracts, checked whitespace and all **196 local links/anchors** across the nine
changed/new documents, and confirmed no tracked changes outside `docs/`. Results:
`.artifacts/frontend-architecture-review-20260910/documentation-check.json`.
No build, typecheck, application/native test or preview generation was run for this
documentation edit. The terminal contract and its open compatibility question were
not edited. The user subsequently requested committing this documentation; no push
is included.

## Step 5 source closure — local checks passed; no installed or CLI acceptance

Step 5a–5d is now locally source-complete, following `9ad7847`, `54686d0` and
`d2821fa`. The shared workspace retains bounded v2 locators/layout, flat stable
xterm owners, measured panes, sidebar/tab navigation, original-target New/Rename/End
and compact Forge/Terminal visibility. Final review added inert/hidden action and
focus guards, screen-focused managed input, sibling invalidation on definite terminal
actor/authority loss, finite-retention eligibility and named pane ownership. Reopening
an already-open native surface cannot Return. Confirmed cleanup removes only the
matching locator/navigation observation; accepted End and uncertainty do not.

**Journey and integration closure:** existing installed callers now follow scoped
project controls, Environment/Access, the project/name chooser and named End menu.
Their UI-only helpers are exercised against the actual emitted workspace. Terminal
mode requires exact private `terminal_actions: ["create", "end"]`; the End request
is admitted only for the original actor/environment/session observed on the socket.
The chooser's environment must match the original observed project before submission;
a local wrong-target case proves refusal without a socket. Old read-only inputs cannot
silently authorize it. These are source ports and local
fixtures, **not execution on an installed candidate**. Asset revalidation covers all
canonical Sodaspaces modules/styles and the shared Lit runtime.

`test:spaces-page` now consumes actual authorized Go handler HTML/CSP and production
page/assets, with synthetic GET-only API responses. `test:layout` runs 20 combinations
(1440/1024/900/390/320 × light/dark × running/stopped), with actual fitted cells,
≥480px desktop native-left, compact form/selection preservation, beforeunload cancel,
reduced-height menu/terminal fitting and exact Hide/Return distinction. Both are in
`bun run test`. Pointer reorder/edge split, sidebar bounds/collapse, overflow search,
compact pane selection and stable host/renderer/socket behavior have focused checks.
The fixture's native form and synthetic socket are not stock Forgejo or tmux proof.

**Executed locally**, pinned Bun 1.4.2 / Go 1.26.7:
- Strict TypeScript; frontend **168 pass / 3 gated skips** (Go-page and layout cases
  then run separately; the private browser-pipe opt-in remains skipped).
- Go HTML/browser **1 pass**; measured layout **20 cases pass**; explicit Lit
  **105 pass**; Forgejo **26 pass / 18 optional skips**; Cockpit **60 pass**.
- All Go packages; race checks for web/store/host; **7** temporary-filesystem
  packaging and **3** canonical-payload tests. No installed-stage test was substituted.
- Isolated preview generation and byte equality for **136** canonical public files
  plus branding projection. No live mount/service changed. Formatting/whitespace,
  changed-document links and the unchanged compatibility-question text were checked.

Evidence: `.artifacts/spaces-step5-f28f86e/`, especially `*-closure.log`,
`packaging-final.log`, `payload-final.log`, `preview-bytes-closure.log`,
`layout-1788992027945/` and `.artifacts/spaces-page-sqv4Og/`. Earlier failures remain
retained. The user's **Open product question — browser-terminal compatibility**
from `7afde63` is byte-identical.

**Next: step 6**, observed unread/lifecycle attention, exact candidate build/coverage
closure and separately scoped concurrent/native/CLI acceptance. Actual Codex CLI,
Claude Code and Pi, physical keyboards, stock Forgejo form/menu/diff/comment/clipboard
coexistence, native process/cleanup and the historical failed framing probe remain
unverified. No native build/export/delivery, retained target/project mutation,
provider action, root recreation or push occurred in this closure. Prior native
results and Rocky 10.2's undeployed candidate retain their separate provenance.

## Step 5 shared chrome and compact adapter — local, journey closure still pending

The shared workspace now supplies the page sidebar, flat native tabs/right-content
Sessions view, authorized search/This page filtering, explicit project/name New
chooser and per-terminal menus. End has an original-target loss warning and Cancel
focus; Hide and named Environment/Access keep their distinct owners. Keyed pane
chrome adds drag-edge/reorder and named keyboard move destinations. Native compact
Forge/Terminal is measured against terminal cells plus a 480px native-left target;
visibility is document-local, uses no form cloning/reparenting, and never calls
Hide/Return. Normal navigation remains native; BFCache starts compact Forge.

**Local checks:** strict TypeScript at `types-5cd.log`; frontend **163 pass / 2 opt-in
skips** at `frontend-5cd-second.log` under `.artifacts/spaces-step5-f28f86e/`.
The selector-port failures, missing native fixture CSS/inert hit-test failure and
old 1024px fixed-ratio expectations are preserved there, not discarded. Browser
fixtures use emitted Lit/real xterm and synthetic peers, not native tmux or CLI proof.
**Still to close step 5:** installed-journey source/input/fixture ports, real Go HTML
plus emitted page-bootstrap integration, the full measured native/layout/theme/
keyboard matrix and final review/docs. Attention remains step 6. No native target,
retained project, deployment, provider resource or push was changed.

## Step 5 pane/geometry foundation — local source, remaining chrome/compact work

After `9ad7847`, pure split/move/reorder/resize/consolidation and compact projection
now drive stable flat terminal hosts. Split creates an empty view, not a shell;
maximizing/shrinking preserves desired layout/ratios. Page mode supports simultaneous
visible panes; native mode flattens the same tabs without changing group membership.
Keyboard separators and named move destinations are wired. Xterm/input visibility
and visible-only fit have presentation-only facades; actual rendered cell metrics
supply split minimums. Geometry waits for the xterm render frame: measuring the old
grid against its already-updated column count reproduced a layout loop, now corrected.
The keyed pane/tab chrome uses `repeat` through the single shared runtime; both public
asset roots and an unsupported-directive refusal remain tested.

**Local checks:** strict TypeScript; frontend **160 pass / 2 existing opt-in skips**;
explicit Lit **100 pass**; Forgejo **26 pass / 18 optional skips**. The real-xterm
browser case checks host/renderer/socket continuity, actual resize cells and no
native actions across split/move/resize/maximize/compact/consolidation. Logs and early
geometry/directive-fixture failures: `.artifacts/spaces-step5-f28f86e/` (`*-5b*`,
`panes-*`). These are synthetic-peer checks, not native process/CLI evidence.
Final chrome/sidebar/chooser/menus, drag-edge/reorder affordances, full compact native
coexistence and installed-journey ports still need completion in step 5. No deployment,
retained target/lifecycle/provider action or push occurred.

## Step 5a — bounded layout storage and exact restoration, local source

Started step 5 from `f28f86e`. `sodaspaces-layout.ts` now owns the pure typed pane/
locator representation and closed-shape, byte/node/depth-bounded v2 parser, v1
migration and serialization. The shared workspace uses it for one-pane selection,
Hide and locator promotion; stable local keys survive request-to-ID promotion.
Hidden saved work no longer briefly mounts/attaches during restore. Partial reads,
unknown outcomes and failed storage preserve locators; malformed/present-v2 records
cannot fall back to v1 or be overwritten. Acknowledged cleanup removes only its
matching saved reference; live drafts never become replayable reload commands.
The production manifest includes the new module. No DB/API/native change occurred.

**Local checks:** strict TypeScript; frontend **157 pass / 2 existing opt-in skips**;
explicit Lit **99 pass**; Forgejo **26 pass / 18 optional skips**. New pure parser/
identity tests and emitted-browser migration/reload/storage-failure cases exercise
the real workspace and xterm with synthetic peers. Evidence and earlier type/old-
storage-selector failures: `.artifacts/spaces-step5-f28f86e/` (`*-5a*` logs).
This is not native tmux/CLI proof. Steps 5b–5d (pane operations/geometry, final chrome,
compact behavior and installed-journey ports) are next, not complete. No deployment,
fixture mutation, project lifecycle, dependency upgrade or push was performed.

## Steps 5–6 plan reconciliation — documentation only

Revised `docs/lit-migration-plan.md` against the implemented `18aceb2` workspace:
5a locator/layout migration, 5b pane operations/measured terminal geometry, 5c shared
page/drawer projections and 5d compact/native coexistence with installed-journey ports.
Step 6 separates observed attention, candidate/source coverage and scoped native proof
using actual Codex CLI, Claude Code and Pi. The plan assigns concrete owners and exits;
it does not add another frontend, backend registry, OS phase or acceptance framework.

The planned browser v2 format preserves exact/pending/hidden locators and is separate
from implemented SQL v6. Stable local terminal owners survive layout operations;
measured cell/chrome checks replace the provisional pixel floor. Corrected stale
owner/preview descriptions and the two design introductions; the leading plan points
to these slices instead of repeating completed ports. Steps 5–6 remain unimplemented.

**Checked:** source/document review, **111 local Markdown paths/anchors**, whitespace
and byte-identical preservation of `docs/terminal-integration.md` against `7afde63`.
Logs: `.artifacts/spaces-plan-7afde63-ZRN2q2/`. No build, typecheck, application test,
preview publication, native action, deployment or push was performed. Earlier evidence
and failed native probes remain unchanged; source-ready, native-validated and delivered
are distinct exits. The user's compatibility note, now committed separately in
`7afde63`, was not edited. All retained targets/roots/credentials remain untouched.

## Shared Spaces page and drawer — local step 4, no rollout

Implemented the requested initial shared workspace in source after `d451c67`.
`GET /-/soda/spaces` is Soda-owned escaped Go HTML, authorized with the Soda session
and acting grant, not fabricated native Forgejo context. Anonymous/expired access
has explicit Connect; grant/provider failure is unavailable, not an empty workspace.
Only this page permits local modules/xterm styles through its CSP. JSON actor,
CSRF/origin, framing, private caching and credential boundaries remain intact.

Fixed `destination=spaces` uses an append-only schema-v6 OAuth flag, with no mixed
repository intent or caller URL. Populated synthetic v5 preservation, wrong-key
rejection before migration and logout-winning finalization tests pass. Existing
repository/home returns remain. No OAuth client, grant key or retained database was
changed. Matching backend/assets/schema require separately approved delivery.

The global native Spaces link, thin page bootstrap and drawer use the same Lit
workspace. Flat terminal owners retain exact hosts across session tabs and project
details. Management/access moved into `sodaspaces-project.ts`, preserving original-
target drafts, authorization, uncertain writes and key/lifecycle checks. Explicit
New, exact attach, Rename, session/drawer Hide, selected Continue/Keep and confirmed
End share the terminal component. Metadata observes actual deadlines without renewal.
Versioned actor-scoped storage holds bounded locators, not names, transcripts or
credentials. Unknown cleanup is not replaced; legacy pending is never guessed.
Confirmed-ended owners cannot create replacements through their old controls.
Definite actor change detaches siblings and disables the stale workspace.

Canonical staging includes page/project assets. Preview generation projects the
production manifest into both the retained branding layout and a complete public
tree, including locked xterm. Validation used isolated output under
`.artifacts/spaces-step4-d451c67/`; no live preview mount, service, appliance, provider
resource or project was changed.

**Executed locally:** pinned strict TypeScript; all Go packages; web/store/host race
suites; frontend **126 pass / 2 existing opt-in skips**; explicit Lit **93 pass**;
Forgejo **26 pass / 18 optional skips**; Cockpit **60 pass**; seven temporary-filesystem
packaging tests; canonical payload tests; emitted real-xterm **16-case** drawer
layout matrix; isolated preview generation. Ten new workspace browser cases cover
three sessions across two projects in both bootstrap modes, correlated creation,
End uncertainty, actor invalidation, Hide/Show, target-bound Rename, project drafts,
legacy locators and late authorization. HTTP/socket peers are synthetic, not native
tmux/CLI proof. Existing management tests were ported to their extracted owner.

**Evidence:** `.artifacts/spaces-step4-d451c67/`, including early fixture/type and
inventory failures and successful reruns. Initial 320-light and 1440-dark layout
images were reviewed. Multi-session controls occupy explicit rows: the layout test
uses a 12-row-sized screen floor instead of the singleton's 650px floor. This is
not final UX acceptance. Steps **5–6** (layouts/compact behavior and attention),
installed-journey selector ports and native concurrency/CLI acceptance remain pending.
Existing installed probes describe the earlier delivered UI; do not run them
unchanged as proof of this candidate. Historical failed native terminal probes
remain failures. No native build/export, deployment or push occurred. The user's
separate 18-line terminal compatibility note is preserved unstaged.


## ID-keyed terminals and bounded Spaces API — local step 3, no rollout

Implemented Lit-plan step 3 in source after the user's request to continue.
`internal/web/terminal.go` owns the browser transport; `terminal_sessions.go` owns
immutable bindings, lifecycle metadata/actions and native-owner/receipt lifetime;
`spaces.go` owns the authorized bounded collection. There is no schema, native
helper/program, project image, provider credential or deployment change.

The registry now admits independent same-project IDs, with one writer per ID,
64 live/reserved slots and 128 transports. Each entry retains its original context/
token, actor, project/repository, Linux login and hard lifetime. Stop cancels every
ID and pending transport of its project across contexts; logout/rotation retains its
original cancellation boundary. Native/provider IO is outside the registry lock;
reservation/admission and binding rechecks remain serialized. A paused native dial
cannot block logout. Unconfirmed dispatch/cleanup retains its exact slot.

New protected routes provide per-ID metadata/End/Return/Keep/Hide/Rename, exact
`request_id` lookup and `/api/spaces`; [the API guide](dashboard-api.md#exact-session-metadata-actions-and-creation-outcomes)
owns their fields and concrete bounds. Hide only defaults an unset deadline, and
Hide/active Return uses that socket's attachment generation. Metadata shows actual
capped effective/hard deadlines. Native acknowledgement releases a slot and may
leave an authorized receipt (128 maximum, five minutes/original authentication).
An absent/expired receipt is unknown, not cleanup proof. Names are bounded runtime
metadata, not shell input. No native-session adoption, replay or durable history.

The collection authorizes before exposing/inspecting rows, retains degraded own-
member observations but hides elevated/session data, and distinguishes confirmed
repository denial from unavailable actor authority. It scans at most 128 associations,
publishes at most 32 rows/64 KiB and bounds concurrent requests/inspection time.
Truncation/unavailability is incomplete or 503, not a complete empty catalog.
The old singleton endpoint returns 410. The Lit caller now saves a correlated
pending locator, reads only its exact attempt/ID and observes End's result without
forgetting an unconfirmed ID. Old uncorrelated pending locators never select another
session. The current drawer still shows one terminal; multi-session UI is step 4.

**Executed locally**, retained in `.artifacts/terminal-ids-a666c63/`: all Go packages;
web/store/host race suites; strict root/browser/test/Cockpit TypeScript; frontend
116 pass/2 existing opt-in skips; explicit Lit browser suite 83 pass/no skips;
Forgejo 26 pass/18 optional skips; Cockpit 60 pass; seven temporary-filesystem
packaging tests; and the explicit 16-case integrated xterm/layout matrix. Browser
builds ran through the test commands. Screenshots use the historical fixture prefix
`.artifacts/merge-5c845a7-560265b/layout-1788970568341/`; 320-light and 1440-dark samples
were visually reviewed; the final 16-case rerun is also retained at
`.artifacts/merge-5c845a7-560265b/layout-1788971957875/`. Tests include independent
IDs/projects/contexts, one writer,
Stop/logout, correlation, generation/deadline guards, labels, capacity/receipts,
denied/unavailable/bounded collections, missing outcomes and refusal of attach
readiness without that socket's new writer locator. Early TS narrowing
errors and an aggregate-test timeout are retained; the slow-helper fixture now has
explicit release/teardown, and complete/race reruns passed. These are source/browser/
helper-double results, not real concurrent tmux or selected-CLI proof.

**Next:** step 4 fixed OAuth return/HTML shell/shared multi-session workspace and
canonical preview asset projection; then layout/attention and native acceptance.
No native build/export, installed journey, live-preview update, project/service/
provider mutation, deployment or push occurred. Installed `22d8591`, its historical
native probe gap, retained Rocky roots and separate action/target gates are unchanged.
The user's browser-terminal compatibility note remains preserved and unstaged.

## Lit terminal-controls port — local proof, backend/workspace still pending

After the user's confirmation, continued with xterm/tmux while retaining actual CLI
compatibility as a separate acceptance gate. `sodaspaces-terminal.ts` now owns a
light-DOM `SodaTerminal` component. Its complete facade and existing single-session
wire remain; xterm owns one stable screen node and its imperative IO resources.
Rendering does not create, attach, Return or End. No Go/helper/native change occurred.

The component waits for its own rendered screen before opening xterm and rechecks
retirement/Hide. Late socket-open after Hide cannot send creation. Ready rendering
rechecks generation before focus/observer setup; zero-size geometry is not sent.
Stale renderer input cannot reach a successor attachment. Lifetime commands use a
synchronous busy guard; dispose/detach abort pending browser requests and release
resources exactly once, without sending End. A failed initial send retains creation
uncertainty. These bounded race fixes do not add replay, durable jobs or native
cleanup receipts. HTTP End still acknowledges **ending**, not confirmed disappearance.

Every previous terminal/resume assertion now runs against emitted Lit modules in
sandboxed Chromium, without patching Lit/HTMLElement constructors. The 38 terminal cases also
cover rapid commands, wrong account/401/403, malformed/oversized frames, output/input
bounds, refused writers, hidden loading, unknown End and stale callbacks. Management
fixture restore now follows the asynchronous facade. `test:lit` includes both ports.

Actual pinned local results under `.artifacts/lit-terminal-37c3941/`: strict root/
browser/test/Cockpit typecheck; frontend 95 pass/2 existing opt-in skips; explicit Lit
browser suite 77 pass/no skips; Forgejo 26 pass/18 optional/export/browser skips;
Cockpit 60 pass; Go `TestSodaspaces`; seven temporary-filesystem packaging tests; and
the explicit current 16-case xterm light/dark 1440/900/390/320 layout run passed.
Browser builds ran through the test commands. The first test-fixture typing failures
are retained; they were corrected without casts/suppressions. Layout screenshots:
`.artifacts/merge-5c845a7-560265b/layout-1788966274308/` (historical fixture prefix);
320-light and 1440-dark samples visually reviewed. Additional checks cover synchronous
native-focus retirement before observer setup and keyboard exit during pending
lifetime authorization. These are component/transport-double results, not native
shell or selected-CLI proof.

**Remaining:** step 3 ID-keyed backend/authorized collection/new wire and correlated
creation/cleanup outcomes; step 4 fixed OAuth return/migration and real Spaces page/
shared workspace; step 5 pane/tab/compact behavior; step 6 attention and complete
acceptance. The existing preview's root-asset projection is still pending. Reviewed
the current registry/Stop callers for the next slice, but did not change them. No
native build/export, installed journey, live-preview update, service/project/provider
mutation, deployment or push command occurred. The user's concurrent CLI-compatibility
note remains unstaged/preserved; only the separate Lit-status sentence in that guide
belongs to this commit. Installed `22d8591`, its broader native probe failure, the
Rocky retained-root decision and separate target/action approval gates are unchanged.

## Lit implementation started — management drawer port, not end-to-end completion

The user requested the complete six-step implementation. This source slice ports
`SodaSpaces` management rendering to light-DOM Lit, preserving real API actions,
context/CSRF/provider checks, synchronous busy guards, uncertain outcomes, explicit
Stop/last-key confirmations and the original terminal facade. The bounded JSON
reader now belongs to `sodaspaces-api.ts`; terminal presentation/transport is otherwise
unchanged. There is no second renderer or migration flag.

The native adapter dynamically imports the component/runtime only on opening or
saved restoration. Late loading/readiness checks departure, generation and Hide;
it cannot repaint a retired document or steal focus from an edited native control.
Same-target Refresh/view/Hide preserves the terminal host. Ordinary reactive updates
preserve key drafts/selection; explicit Refresh still resets management drafts.
Completed hidden reads defer terminal mounting until deliberate Return. Incomplete
provisioning now refuses the terminal factory even if a contradictory running
observation is supplied. Disposal detaches browser resources, never sends native End.

Converted every existing management assertion and the four native-shell/real-content
cases to emitted modules in their actual Chromium realm. Pure native-adapter tests
still use injected content; terminal transport tests retain their existing doubles
until step 2. Added rapid-click, draft/node identity, hidden-loading/Return, retirement,
401/403 and incomplete/unavailable checks. `test:lit` includes these real-browser
contracts; missing browser support fails rather than skips them. The runtime smoke
uses pinned Playwright Chromium rather than requiring an unrelated global Chrome.

Actual local checks with pinned Bun, frozen lock installation and lifecycle scripts
**disabled**: strict root/browser/test/Cockpit typecheck; frontend 74 pass/2 existing
opt-in skips; Forgejo 26 pass/18 optional/export/browser skips; explicit Lit browser
suite 39 pass/no skips; Cockpit 60 pass; Go `TestSodaspaces` and seven temporary-
filesystem packaging tests passed. `build:forgejo` ran through those commands.
The explicit current 16-case light/dark 1440/900/390/320 layout run passed with real
xterm and synthetic IO/API, not native processes. Visually reviewed 320-light and
1440-dark samples. Its final screenshots are under
`.artifacts/merge-5c845a7-560265b/layout-1788961798864/` (the fixture's historical output
prefix); logs/exits and prior failures are under `.artifacts/lit-implementation-c754a07/`.

Earlier failures are retained: browser-fixture typing, incomplete-provisioning mount,
JSDOM importing the newly ported component, a malformed AssetUrlPrefix in the layout
fixture, and waiting for an intentionally empty native root to become visible. These
were corrected, not bypassed. No installed journey, live preview refresh, native build/
export, service/project/provider/credential mutation, rollout or push command occurred.

**Still pending:** terminal presentation port; ID-keyed sessions/collection and new
wire; fixed OAuth return/migration; real Go Spaces page and shared workspace; full
pane/tab/compact design and attention; root-asset preview projection; wider/new-feature
and installed acceptance. No Spaces page or concurrent backend is implemented by this
commit. The concurrent user edit in `docs/terminal-integration.md` raising actual
Codex/Claude Code/Pi compatibility is preserved outside this slice's commit. It is
not answered by these component/renderer tests. Installed `22d8591`, its failed broader
native probe, the Rocky retained-root decision and separate deployment gates remain.

## Spaces page/drawer with Lit — merged-source implementation plan

The user requested pulling/resolving origin/main, learning the incoming Lit direction,
and planning both real Spaces surfaces with it. Merge `619c5d9` completed the Git
portion below. Reworked `docs/lit-migration-plan.md` into the single detailed sequence
under the leading Sodaspaces plan: preserve behavior in the management/terminal
rendering ports, then implement ID-keyed sessions/authorized collection, the Go page
and shared Lit workspace, direct layouts/compact drawer and truthful attention.
The full-page and companion-drawer designs remain UX inputs, not fake implementations.

The plan keeps light DOM, one shared runtime, static strict-TS properties, native
adapters/Cockpit and xterm's imperative screen ownership. It corrects the incoming
pre-tmux assumptions: current same-target Refresh preserves the terminal, restore
attaches an exact ID, HTTP End acknowledges ending, and document disposal detaches.
It explicitly handles Lit async readiness/retirement and per-render-part keyed identity;
the selected layout candidate keeps terminal hosts in a stable flat layer instead of
reparenting/remounting them on pane moves. Its real browser/focus proof is future work.

Proposed backend deltas are kept distinct from implemented wire: per-ID metadata/
actions, request-correlated creation, attachment-bound default Hide/active Return,
actual capped deadlines and bounded short-lived native-cleanup receipts. The page
uses Soda-owned Go HTML, configured native links and a labelled Soda actor, not
fabricated Forgejo session/CSRF/notification context. Fixed Spaces OAuth intent,
page-only CSP, shared styles/clipboard, canonical payload/preview projection,
legacy-locator refusal and source/browser/native acceptance are included. No new
agent signal adapter, event bus, router, UI kit, native broker or dependency is added.

Updated the leading plan, Lit guide, design links, architecture, current API/terminal/
Project OS status, deferred boundary and AGENTS guidance to remove stale unimplemented-
tmux/page-shell assumptions. Historical delivery/failure evidence is preserved and
labelled, not converted into new acceptance. The actual API/backend/frontend/native
implementation did not change after the merge resolution.

Read official Lit lifecycle/properties/render-root/list documentation and the locked
LitElement/reactive-element/repeat source versions. Snapshots and notes:
`.artifacts/research/lit-spaces-619c5d9/`. Markdown path/anchor and whitespace checks
passed; the focused Go/template-inventory merge checks are recorded below. No Lit
install/build/typecheck/browser suite, application preview update, native execution,
service/project/provider/credential change or rollout was performed for the plan.
No push command was issued by this session; during planning the remote-tracking
reflog recorded an external/concurrent push updating `origin/main` to `619c5d9`.
The working source stayed on that merge throughout.

**Next source slice:** Lit step 1, with converted action/race tests and the existing
stable terminal host. No production component, multi-session backend or Spaces HTML
handler is ported/implemented yet. Installed isolated `22d8591`, the separate Rocky
10.2 root decision, `soda-test` exclusion and incomplete native safety/UX proof stay
unchanged. This plan does not authorize deployment or imply final product acceptance.

## Merge of origin/main `8f03910` — resolved, focused source checks only

At the user's request, fetched origin and merged its six commits into local
`ca1da85` without rebase/history rewriting. Preserved the incoming Lit scaffold and
migration guide, screenshot correction and responsive native repository/page work,
alongside managed tmux and the full-page/drawer designs. The three conflicts were
`custom/footer.tmpl`, its presentation inventory and this handoff. The footer keeps
the signed non-repository resume hook, the narrow-pane button's accessible name/text
span, repository-only disclosure script and unchanged notification markup. Reviewed
and recomputed only that merged structural hash; both branches' evidence remains.
A focused Go regression also checks the merged script scope and accessible button.

Prepared Bun 1.4.2 and offline/read-only-module Go 1.26.7 checks ran locally:
`go test -mod=readonly ./scripts -run '^TestSodaspaces' -count=1` passed;
`bun test tests/forgejo/presentation/inventory.test.ts` passed its production
inventory/caller check, with the separate unavailable embedded-template export
check skipped. Logs/exits: `.artifacts/merge-ca1da85-8f03910/`. No Lit dependency is
installed in this builder's current workspace; no dependency installation, aggregate
build/typecheck/browser/native suite, preview refresh, service/VM/provider/project
change, rollout or push occurred. Incoming Lit scaffold evidence remains scoped to
its originating run, not a merged-candidate test pass. Spaces/Lit implementation
planning is the next requested work, not application implementation permission.

## Right-half Spaces drawer — complementary design, not implementation

The user called the full-page Spaces design good and explicitly requested the
right-half tabbed terminal design alongside usable native forge browsing. Added
`docs/spaces-drawer-design.md`: default 50/50, viable 35–65% resize, full-height native
left/drawer-right composition, compact three-row terminal chrome, and a flat view of
this window's open tabs derived from the full-page pane tree without destroying it.
A temporary right-only Sessions view supplies project/search/attention/explicit
this-page filtering; named Environment/Access views remain reachable. Browsing a
different repository is intentionally independent of the terminal's original target.

The design distinguishes same-document tab/view changes, native document replacement,
normal navigation to full Spaces, and explicit Hide/finite retention. Compact
Forge/Terminal switching preserves the native document and session owners; new native
navigation shows Forge rather than covering the destination with a restored terminal.
Native beforeunload/modal/account/notification behavior remains authoritative, with
actual actor mismatch distinct from an ordinary repository difference. No second
session owner, framework, backend, native process or orchestration behavior is supplied.

Three new hand-authored SVG sheets show an unsent native comment beside a terminal
in a different project, native code browsing beside the right-only session switcher,
and compact surface/continuity behavior. Native pages and terminal content are
schematic, fictional drawings—not actual Forgejo rendering or runtime evidence.
The existing offline renderer now emits all six sheets with no listener. Design-tool
strict TypeScript, font/SVG/whole-sheet text bounds and unexpected-request/page-error
checks passed, along with Markdown path, SVG XML and Git whitespace checks. The new
sheets were visually reviewed. Final PNGs/source-hash/revision/dirty/font record:
`.artifacts/spaces-redesign/1788955718668/`; the first drawer render remains at
`1788955555323/`. These are drawing checks, not interaction/accessibility/native tests.

Updated the parent Spaces design, leading plan, terminal guide and visual README.
No application code, credentials, dependency installation, listener/tunnel, existing
service, VM, project/provider state, deployment or push changed. True multi-session
ownership and these integrated controls remain unimplemented; native reflow, forms,
same-shell navigation and mobile keyboard/viewport behavior need real validation.
Installed `22d8591`, the separate Rocky root decision and native proof gaps are unchanged.
The work remains based on `f5886aa`; `origin/main` advanced by six commits during this
turn. No merge/rebase or incorporation of that concurrent upstream work was performed.

## Spaces redesign — research-backed specification and static sheets

After reviewing online terminal/agent workspace references, the user requested the
actual redesign. `docs/spaces-design.md` now replaces the grid-first proposal with
project/session navigation, local tabs, direct split/move/resize/maximize, measured
compact behavior and attention distinct from connection/lifetime. Ordinary new output
is a quiet unread dot, not an attention alarm; semantic agent states need real explicit
signals. No agent-chat frontend, mandatory worktrees, native backend replacement,
arbitrary four-pane limit or first-slice bulk actions are selected by the redesign.
Original account/context binding, one writer, finite retention and cleanup reservations
remain intact. The first real multi-session slice and signal adapter are unimplemented.

Three hand-authored SVG sheets in `docs/design/spaces/` show focused desktop,
wide cross-project terminals, and two 390×844 mobile frames plus lifecycle/error cases.
They reference canonical assets through `sheets.css`; they are static drawings with
fictional content, not another interactive app or installed evidence. The old `8fe3468`
mockup/source/captures are explicitly superseded as design material, not erased.
The leading plan, terminal guide and design README point to the revised specification.

The design-tool strict TypeScript check passes. Initial checking found an un-narrowed
HTML canvas element; an explicit `instanceof` check fixed it without a cast/suppression.
The sandboxed Chromium renderer then produced three sheets with canonical font loads,
valid SVG, whole-sheet text bounds and no unexpected page requests/errors; all three
were visually reviewed. Final images and source-hash/revision/dirty/font-metric record:
`.artifacts/spaces-redesign/1788954085183/`; earlier render remains at `1788953854599/`.
Markdown path/SVG XML and Git whitespace checks passed. These are drawing checks,
not responsive interaction, xterm, mobile keyboard or native process validation.

The renderer fulfills a fixed local asset map in a fresh browser, opens no listener,
and closes that browser/context. No new port/tunnel, existing preview service change,
credential access, dependency installation, production build/test, VM contact,
deployment or push occurred. The real Spaces implementation still belongs on the
existing `/-/soda/spaces` application origin (33443 on the isolated deployment).
Last installed candidate remains `22d8591`; the separate Rocky 10.2 existing-root
decision and remaining native terminal proof are unchanged.

## Spaces visual review — interactive mockup, no product changes

**Historical `8fe3468` mockup, superseded by the redesign above.** Its behavior/checks
are not current design acceptance or evidence of an implemented Spaces page.

Continued the user-requested design phase with `docs/design/spaces/`: an explicitly
labeled HTML/CSS/strict-TypeScript mockup using the existing canonical symbol,
Barlow/Plex fonts and shared palette. It shows six fictional sessions across two
projects, mixed-project tabs/panes, layouts/maximize, move via menu, search,
project-scoped sample creation, rename, Hide versus confirmed End, retention/error
states, non-modal project details and compact single-pane/session switching.
There is no live xterm, API, auth, process or persistence implementation. The default
proposal remains a single pane; the initial two-pane scene demonstrates concurrency.

Both design compiler boundaries (extending existing strict configurations) and the
sandboxed Chromium design-only checks passed. Seven captures and the source-hash /
revision-dirty record are retained at `.artifacts/spaces-design/1788951512029/`.
Reviewed final desktop dark, mobile light and retained/offline images, plus the prior
mobile switcher and 320px retention images. The earlier capture exposed clipped
native-select text with Barlow on Chromium. Native selects now use a system font;
corrected captures/checks passed. Earlier captures remain at `1788950997794/` and
`1788951252411/`; none is installed-product evidence.
The checks cover stable sample IDs through keyboard/move/layout/filter/detail changes,
explicit creation, Hide/End cancellation and sibling isolation, mobile selection,
page overflow and absence of external requests/page errors. Touch keyboards,
real terminal rendering/resize/retention and native authority remain unverified.

`review.ts --serve` is an opt-in loopback-only design server; `--check` starts and
closes its own temporary server/browser and writes ignored outputs. These are not
production build/staging paths or a second product readiness gate. At that handoff,
a separate design server was running on **127.0.0.1:33450** on this builder; its PID/log are recorded in
`.artifacts/spaces-design/preview-server.{pid,log}`. Verified the preview title and
404 for the Soda API route. Remote access needs its own explicit client forward;
this is not the deployed Spaces route. No existing Forgejo preview mount, account,
credential, VM, project, package baseline or installed service was changed; no
dependency install or deployment/push occurred. Rocky 10.2 remains pending its
separate project-recreation decision. See the [preview guide](design/spaces/README.md).

## Spaces multi-project UI/UX — design proposal only

The user requested design first for the Spaces top-bar destination and multiple
concurrent terminals across multiple projects. `docs/spaces-design.md` now records
a proposed project/session navigator, cross-project tab/pane workspace, compact
drawer relationship, explicit Hide/End/Stop semantics, lifetime/uncertainty states,
responsive/keyboard behavior and bounded implementation/acceptance sequence.
The concurrency goal is selected; pane limits/layout details and optional bulk
controls remain recommendations for review, not implemented features.

Source inspection confirmed the selected Forgejo 15.0.7 navbar's official
`custom/extra_links` hook is immediately after Explore; the current override can
carry the link without a backend fork. The Go page's authenticated shell/fixed
OAuth return and authorized collection still need implementation review. Current
`terminalKey{context, project}` remains a single-session constraint requiring a
real backend change, not just extra tabs. Preserve original-context/account and
one-writer/native safety boundaries; no all-user terminal discovery or cross-login
adoption is selected.

Only design documentation and cross-references changed. No build/test, application
implementation, VM contact, rollout, project recreation or push occurred for this
request. Rocky 10.2 remains built/checked but undeployed pending the separate
existing-project decision below.

## Rocky 10.2 baseline — built and checked, not deployed

The user requested upgrading the Rocky baseline to 10.2. Candidate
`c0b4b917fa786a55f65ba75b93e0b69306263cd8` selects the official 10.2 image in both
project and Go backend Containerfiles. No unrelated CLI/dependency pins or host
OS baseline changed. Registry metadata advertises amd64 and arm64 variants.
A native x86_64 builder inspection confirmed `/etc/os-release` 10.2 and availability
of the existing systemd/Python/tmux/Podman/fuse-overlayfs/slirp4netns requirements;
no package-list workaround was necessary.

From a fresh clean exact-revision worktree, production `build-native.sh x86_64`,
`check-native.sh x86_64` and `soda-artifacts bundle` all completed successfully using
Go 1.26.7/Bun 1.4.2. Full Go/strict TypeScript/Bun checks, 69 Python build fixtures
(one opt-in skip) and all 11 actual-stage packaging tests passed. Build metadata
records the resolved 10.2 base and real EL10 package inventory. Exported
`build-info.json` SHA256:
`adf0bb57352260ba9c283e4a933d8b1dc805863da7b47ddf17a09cbc95ffe10c`.
These are image/source/staging results, not project-systemd/tmux/workload installed
proof on 10.2 or native aarch64 evidence.

Artifacts, inspection container name and logs remain under `.artifacts/rocky-10.2/`.
No VM was contacted or changed for this upgrade; the isolated application remains
on the previous `22d8591` deployment described below. Changing a default image
cannot upgrade either existing writable root. There is no implemented same-root
EL9→EL10 migration. Recreating the two test project containers needs explicit
confirmation; the earlier backup waiver is not silently treated as permission to
erase projects. Preserve Forgejo accounts/repositories and exclude `soda-test`.

## Available for testing — isolated deployment `22d8591`

**Installed affected-component candidate:**
`22d85916c861b540d959336dafbe46b84ab3d795` on
`soda-native-spaces-658f2af`, browser origin `https://localhost:33443` through the
existing infra forward. This is not automatic laptop reachability. The user
explicitly waived backups and requested deployment again. No backups, root
replacement, project restart, capability/network/provider changes or push occurred;
`soda-test` was not contacted. Both original project CIDs and sshd PIDs (40/72)
remain unchanged. Retained Cockpit was not redeployed.

The sysfs correction below was built from a fresh clean exact-revision worktree
with Go 1.26.7/Bun 1.4.2. Full production build/check/export completed; all 11 actual
stage packaging tests passed. The first check had a local synthetic PTY interrupt
failure; its log remains, and a separately logged complete repeat passed (69 Python
fixtures, one opt-in skip). The verified export's `build-info.json` SHA256 is
`003725ecaa56e1c8cf9db18941c97762856ce623593644be5925c9a413210e21`.
The corrected helper/backend/default project image and exact program in both
retained roots were rolled forward, without changing their capabilities or roots.
Generated Forgejo files match the corrected candidate (unchanged from `3c2a7d4`).
Live helper/backend executable hashes and all four service states were checked.

**Real browser smoke passed:** stock Forgejo Alice login and OAuth, explicit
terminal creation, actual shell input/output, full page reload with the same shell
PID and retained shell variable, explicit End acknowledged by the API, and fresh
metadata reporting no terminal after native cleanup. An independent native read
then confirmed that shell PID absent and zero managed units in both roots.
The first browser attempt reached End but expected a transient status message;
the corrected check observes the actual End response and cleanup metadata instead.
This does not establish broader screen/history/editor/BFCache/expiry/Stop acceptance.

The separate Go native probe now creates/attaches successfully, but its raw-output
facts parser still failed against tmux output. Experimental multiline/base64 framing
attempts and logs are retained; the tracked probe was restored rather than calling
those failures passes. Its full two-account/resize/interrupt/same-start/lease/helper-loss
journey remains unproven. The failed initial native runtime record is retained, not
adopted or deleted. Browser proof is a bounded installed result, not a substitute
for those remaining cases or aarch64/product acceptance.

Exact recipes, failed attempts, source-bound build/check/export, strict-typed
browser smoke and final native observations are under
`.artifacts/deploy-3c2a7d4/` and `.artifacts/deploy-22d8591/`.

## Isolated deployment and native sysfs admission correction

The user explicitly declined backups and renewed the deployment request.
`3c2a7d4` affected components were deployed to `soda-native-spaces-658f2af`:
helper/backend, generated Forgejo customization, proxy configuration and new-project
image default. Both original project CIDs remain running without project restarts.
Each received native-signature-verified `tmux-3.2a-5.el9.x86_64` after RPM transaction
tests, the exact managed program and updated project-init/runtime parent. No package
upgrades, root replacement, capability changes or backups; original account/key
hashes remained unchanged. Soda SQLite integrity and customization byte checks
passed. Services are active. `soda-test` was not contacted.

The first actual managed terminal probe failed at creation. Native observation
identified read-only sysfs `/sys` and `/sys/fs` owned by unmapped host root (65534),
while delegated cgroup directories are correctly project-root owned. The guard
incorrectly applied mutable-directory ownership rules to those kernel ancestors;
cleanup inspection hit the same refusal. Failed run/runtime records are retained,
not adopted or retried. Source now opens no-follow descriptors, checks actual
kernel filesystem types with native `stat` and read-only sysfs flags, retaining
root ownership/non-writability checks for delegated cgroups and all mutable files.
No ownership/capability workaround was applied. Focused Python boundary tests pass;
rebuilt deployment and native continuity proof for this correction follow separately.
Evidence and exact one-target maintenance recipes: `.artifacts/deploy-3c2a7d4/`.

## Merged deployment preparation — native build/check/export, not installed

Candidate `3c2a7d45f8212c83504f30040f1796d5c855470a` now has a clean
matching-native x86_64 build, full production source/staging check and verified
export under `.artifacts/deploy-3c2a7d4/`. Go 1.26.7 and run-local Bun 1.4.2 were
used; the complete TypeScript migration remains intact. The first build stopped
at an upstream Tea archive HTTP 502. Its worktree/logs remain. A fresh second
worktree used the cached source archive after matching the unchanged Tea lock
SHA256; no historical binaries or stage were substituted.

`build-native.sh x86_64`, `check-native.sh x86_64` and `soda-artifacts bundle`
completed successfully for that exact candidate. Checks include Go packages,
all strict TypeScript boundaries, Bun frontend/Forgejo/Cockpit tests, 68 Python
build fixtures (one opt-in skip), and all 11 actual-stage packaging tests.
The expected negative locale fixture diagnostic remains in the log. Exported
`build-info.json` SHA256 is
`d7e0481a7e5c6f007efa8a20e76cab7208551ccfde45996e51102a71ac0711a0`.
This closes the missing build/stage/export evidence, not installed acceptance.

Pinned-SSH read-only inspection reached `soda-native-spaces-658f2af`: its four
application/helper services and both original exact-CID projects are running;
original Alice/Bob identities/groups remain. Both project roots lack tmux and
have the expected systemd `system.slice`/sshd cgroup paths. Path observations do
not prove managed terminal supervision. The Rocky tmux RPM was downloaded and
its dependencies, file list and `/etc/shells` scriptlet effects inspected locally;
its signature is not established by the builder's NOKEY result. No project
package/program was installed, backup/quiescence/rehearsal performed, service
restarted, native terminal probe executed, or deployment/push performed.
`soda-test` was not contacted. Confirm the selected target, bounded service/project
interruption and copied-root rehearsal before retained-root maintenance; preserve
both lasting roots and all later writes. Exact maintenance and native process
continuity/cleanup proof remain outstanding.

## Tmux / Bun-TypeScript merge — local validation, no deployment

Merged `5c845a7` with fetched `origin/main` at `560265b`, preserving the incoming
presentation, CoreOS strategy, Cockpit and Bun/strict-TypeScript work. Conflict
resolution ports the resumable drawer, terminal and navigation behavior plus tests
into the `.ts` owners; no authored JS/MJS/CJS files remain in the tracked tree.
Browser URLs still serve generated JavaScript and locked upstream xterm assets,
not TypeScript source. Root Bun workspace/lock, unknown-response narrowing and all
four strict compiler boundaries remain intact. No unchecked-JS configuration,
blanket casts or type-check suppressions were introduced.

Local x86_64 checks under `.artifacts/merge-5c845a7-560265b/`:

- Downloaded run-local Bun 1.4.2 and verified its official release SHA256; did not
  change the global Bun. Preserved the old Cockpit dependency directory, then ran
  the root frozen install with unchanged manifest/lock and disabled lifecycle scripts.
- All four strict TypeScript configurations pass. Bun root tests pass 64 frontend
  and 15 Forgejo tests (two and fifteen opt-in skips); Cockpit passes all 60 tests.
  Minified Forgejo emission and both Cockpit page builds pass.
- Offline/read-only-module Go tests across `./...` and host/web race checks pass.
  Python build fixtures pass 68 tests with one opt-in skip. The first Python run
  exposed a missing `bunfig.toml` in the metadata fixture; it now copies the actual
  public input rather than bypassing the collector.
- Sixteen sandboxed Chromium layout/theme/state cases pass against emitted assets
  and synthetic API/transport. The initial run hit Bun's five-second default; the
  opt-in test now declares a bounded 60-second timeout. Original failure/captures
  remain; final captures are `layout-1788943452775/`. This is not native tmux,
  Forgejo authentication or installed continuity evidence.

The user requested deployment after merging, then explicitly reaffirmed the TS
migration. This merge performs no VM contact, installation/service/project/provider
mutation or push. Full matching-native build/check/export, exact backed-up same-root
maintenance, rehearsal and native continuity/cleanup proof remain pending. The target
and interruption scope still need confirmation before retained-root changes. Earlier
source/native evidence below remains bound to its own bytes and targets.

## Managed tmux terminal candidate — source checked, not delivered

Implemented the next single-terminal source slice on top of `416180e`:

- Stock Rocky tmux/terminfo/tool recipe and one copied helper-owned program. Each
  terminal has a transient project-systemd cgroup containing a root safety guard
  and foreground tmux server/shell as the original marker-bound account. Private
  verified socket/config/binding/lease/writer records; exact attach-only, no personal
  server adoption, restart, root shell, install-on-Open or disposable fallback.
- Backend lifetime owner independent of browser sockets; original Soda context/
  project membership and fresh acting-user repository checks every 15 seconds.
  Thirty-minute detached/Hide retention, explicit Return/two-hour extension and
  original-session-bound 12-hour maximum. Input/output/reconnect do not extend
  abandonment. Logout/rotation/expiry/Stop remain hard browser boundaries.
- Protected metadata/lifetime API, explicit create versus attach, exclusive writer,
  pending Stop gate and 64 terminal slots including detached/uncertain cleanup.
  End reports ending, not disappearance. Native cleanup acknowledgment is required
  to release a slot; failed dispatch/cleanup remains reserved for inspection.
  Registry lifetime is the backend process; no resurrection/reconciliation API.
- Refresh keeps the same renderer; navigation/BFCache reauthorize and reattach using
  per-tab locators, not stored credentials/transcripts/input. Saved workspace stays
  selected across unrelated native repository/non-repository pages; switching is
  explicit. Hide retains the mounted connection while requesting finite retention.
  Signed non-repository resume hook is **not** the unfinished Spaces page/navbar.
- Extended local HTTP/DOM/filesystem/PTY regressions and revised the existing opt-in
  native probe for managed ownership, same PID/start/memory reattach, ended-target
  refusal and owner EOF/lease/helper-loss teardown. Its new protocol/exact-CID input
  rejects old private input files. Authored native checks have **not run**.

Actual local checks in `.artifacts/tmux-416180e/`: offline/read-only-module `go test
./...`; race tests for `internal/web` and `internal/host`; frontend/presentation
suite (62 passes, two opt-in skips); Python build fixtures (68 tests, one skip);
16 sandboxed synthetic Chromium layout cases with local locked assets. Reviewed
1440px dark and 320px light captures under `layout-1788910813980/`. These are local
source/component checks, not a tmux/systemd or native Forgejo browser journey.
An initial frontend failure omitted the new session frame in its double and was
corrected/rerun; all original logs remain. Native-stage packaging invocation did
**not** run tests: `SODA_STAGE must identify the actual native build rootfs`. No
current full native build/stage was prepared; an old stage was not substituted.
Markdown relative-link/anchor checks and `git diff --check` also passed.

**Still required before delivery/acceptance:** actual signed Rocky RPM/downstream
behavior, project-systemd/cgroup/startup/cleanup and guard-failure proof; real native
browser navigation/reload/network/expiry/logout/Stop, editor/build/screen/history/
selection/paste continuity and unrelated SSH/tmux/services preservation. Update the
remaining installed browser journeys deliberately; old request-owned assertions
are not current acceptance. The exact same-root package/file maintenance recipe,
compatibility checks, fresh backup/quiescence and rehearsal remain unfinished.
Missing support refuses. Do not roll out helper/API bytes against old retained roots
or normalize them through image replacement/restarts. Multi-session tabs, zero-key
onboarding, Git credentials and Spaces remain later work.

No dependency resolution/installation, project image build, native tmux execution,
VM contact, package/service/project/key/provider mutation, deployment or push occurred.
Installed isolated `2aa4960`, retained `soda-test`, both isolated mutable roots,
private inputs and historical evidence remain unchanged. The sections below record
older slices/evidence, not current implementation claims. See the current
[terminal contract](terminal-integration.md) and [API](dashboard-api.md).

## Lit migration plan — documented, ports not started

The user requested a concrete [porting plan](lit-migration-plan.md). It selects the
management drawer first, then terminal presentation against its then-current
transport contract, with light DOM and the existing native mount facades. It
retains small native/HTMX adapters and Cockpit, and preserves the current explicit
Refresh disposal limitation until the separately selected continuity work changes
that behavior. The plan includes asynchronous action guards, form/focus and xterm
DOM ownership, browser-realm test adaptation, half/full-width validation and preview
asset projection. It does not implement or deploy a component. Source, upstream
Lit mechanics and the plan were reviewed; documentation links and whitespace were
checked. No build, runtime test or service action was run for this documentation-only
change; the scaffold's earlier execution evidence remains below.

## Lit scaffold — local build, browser and packaging checks passed

The user selected scaffolding before component migration. Lit 3.3.3 is pinned in
the root Bun workspace and resolved in its existing lockfile. The browser build
bundles one core runtime at `public/assets/soda/forgejo/lit.js`, with the upstream
BSD notice beside it. Future modules can import from `lit`; the build maps that
import relative to each module's production payload destination, including beneath
`AppSubUrl`. Lit submodule imports are explicitly rejected until their shared
exports and mapping are added. Existing non-Lit imports retain their public URLs.

No production component imports the runtime yet, and no template loads it eagerly.
The drawer, terminal and native adapters remain unchanged. Forgejo still owns its
native workflows, controls and authority; Cockpit retains React/PatternFly.
[The Lit guide](lit.md) records the strict TypeScript authoring pattern, build
contract, render-root choices and lifecycle responsibilities for later ports.

Local evidence is retained under `.artifacts/lit-scaffold/`:

- Frozen installation, preview build and all four strict TypeScript checks pass.
  No existing dependency versions changed.
- Forgejo tests pass 27 checks with 17 opt-in skips; frontend tests pass 57 with
  two opt-in skips. `bun run test:lit` passes all five checks, including real Chrome
  rendering/reactive updates, independent element state, native form preservation
  and one shared runtime request from both asset roots beneath a URL prefix.
- Three Python payload/staging checks pass, including the actual staging recipe
  with synthetic build inputs. Focused Go missing/unreadable payload checks pass
  for the runtime/license and existing root-level Sodaspaces assets.
- The local Forgejo preview serves the runtime (29,349 bytes) and license exactly
  as built/staged. Its existing generated branding mount was refreshed without a
  container restart. The build guard and browser fixture stay out of production UI.

Initial local checks caught Bun's build-entrypoint resolver kind, TypeScript
narrowing and the negative test's wrapped build diagnostic; corrected checks pass.
This is scaffold and local browser/package evidence. Component ports and native
appliance delivery remain separate work; no native rollout was performed.

## Half-desktop layouts and compact repository header — local preview updated

The repository header now has two rows: a compact repository identity beside the
actions/Sodaspaces controls, followed by native unit navigation. The redundant
"Soda repository" eyebrow is removed. Watch, Star, Fork, feed and gated
transfer/report controls each retain one native owner inside an HTML disclosure.
The user's correction limits the compact action menu to panes at or below 1000px;
wider panes expose these controls inline. CSS owns that breakpoint, and the small
Soda script follows summary visibility on resize, keeps wide actions open and
limits dismissal/focus handling to compact mode. Native controls are not cloned
or reparented. Default-open markup preserves wide action access without JS.
Forgejo's `details.dropdown`
was rejected because its direct list-item keyboard contract cannot contain these
native HTMX forms intact. The disclosure keeps null blur/replacement focus and
does not intercept native modal events. Container sizing also keeps this header
compact beside Sodaspaces; the workspace icon retains its accessible name when
its text is hidden. DOM, visual and keyboard control order agree.

The width-specific correction is verified in the local preview, with evidence
under `.artifacts/repository-header-breakpoint/`: full-width 1440px and compact
800px native captures, three native browser checks (including both sides of the
1000px cutoff, immediate resize interaction, unchanged form nodes and wide-page
access without JS), seven disclosure DOM checks, strict TypeScript, the Forgejo
suite (23 passed, 16 opt-in skipped) and focused Go template checks. Initial
resize checks caught browser blur before ResizeObserver and dismissal before its
callback; focus tracking and interaction-time CSS mode checks resolve both. The
existing preview assets/templates were refreshed; no native action was submitted.

Repository metadata stacks through 1000 CSS pixels, with a full-width clone row
and no compounded metadata spacing.
Dashboard feed/repository columns stack before the feed becomes cramped. Explore
wraps its complete native search/filter group before the input collapses. The
shared header follows Forgejo's 767px mobile boundary; narrow settings use smaller
body insets, and profile, organization and administrator compositions stack their
columns/actions earlier. Native handlers, permissions, controls and navigation
remain upstream-owned.

The user explicitly approved updating `sodaos-local-forgejo` at localhost:3300.
Its generated branding mount is active, with the same named data volume and other
configuration. Missing static Sodaspaces modules/styles were projected from the
existing build and hash-checked terminal assets into its existing public mount;
previous bytes and Compose configuration are retained. This restores preview asset
loading, not a backend/project/terminal execution proof or appliance delivery.
Future branding changes require `bun run build:preview`; this does not rebuild the
separately projected Sodaspaces assets in the local public mount.

The follow-up header redesign is retained under
`.artifacts/repository-header-redesign/`. The generated branding payload includes
the new disclosure script; Forgejo's template reload activated the template
changes locally without recreating the container. Read-only native browser checks
pass at 1440, 960, 800, 720, 390 and 320 pixels and with 720/480px panes inside a
1440px browser. They cover one identity/actions row, on-screen disclosure bounds,
unchanged native guest guards, visible action labels, Enter/Space, Escape/focus,
outside dismissal, real count-link navigation and no browser exceptions. Five DOM checks also pass, including
HTMX-like replacement focus and native-modal event isolation; these are simulated
events, not native form submissions or multi-fork modal runtime proof. Strict
TypeScript, the Forgejo suite (21 passed, 15 opt-in skipped), the frontend suite
(57 passed, two opt-in skipped), and Go script tests pass. Initial checks caught
the unsupported native-dropdown keyboard behavior and a 320px panel alignment
issue; corrected checks pass. The first Go payload check was denied access to the
host build cache, separate from test correctness.
Verified native screenshots cover 800px dark, 720px light and 390px dark; the
served disclosure script matches the generated production asset byte-for-byte.
With cache access, the Go suite exposed two macOS temporary-directory symlink
failures in packaging tests; these are separate from the browser/source checks.
Both failed packaging tests and the payload readability checks subsequently pass
with a canonical temporary directory. No appliance delivery or native repository
action submission was performed.

Evidence is retained under `.artifacts/responsive-half-desktop/`. Chrome measured
eight real routes at 720, 768, 800, 900, 960, 1440 and 390 pixels: all 56 responses
were HTTP 200 with no document-width overflow. Visual review covers repository,
dashboard, Explore, issue/pull lists and personal settings; authenticated admin/org
pages have source review only because the retained screenshot fixture lacks that
access. Browser component checks cover light/dark control sizes and clone-row
placement, and native repository/Explore checks cover spacing, resizing and menu
stability. Strict TypeScript, Forgejo source tests and Go script tests pass.
The native personal-settings journey also passes navigation, focus/escape, avatar
dialog/fallback and all permitted destinations through 320 pixels. Its initial
320px header overflow was fixed by removing compounded mobile icon margins/gaps.

The screenshot helper also now leaves native styles loaded unless `--local-css`
is explicitly selected. Its previous always-truthy empty Map removed them from
ordinary captures. Failed verification exposed the missing preview static files;
those failures and an initial browser-test timeout at Bun's five-second default
remain recorded, followed by successful checks with the multi-viewport timeout.
No fixture accounts/repositories, retained projects, provider configuration or
appliance deployment changed.

## Bun/TypeScript conversion — source validated; preview subsequently activated

All 36 authored JavaScript entrypoints/modules/tests (including the avatar CJS
helper) are ported to strict TypeScript. Root scripts, browser modules, tests and
Cockpit pass their separate strict configurations; legacy unchecked JS inclusion
is removed. Browser response and private journey inputs are narrowed from unknown
values. Upstream xterm JavaScript remains unchanged under its distribution lock.

Bun 1.4.2 owns builds, tests, command-line scripts and installed-journey tooling.
Cockpit uses Vite+'s programmatic builder and `bun:test`, with Bun file/glob/hash
operations and explicit browser minification. Root browser assets are minified by
Bun into ignored build output and staged through `@build/forgejo-js/`, preserving
public URLs. Native build/check callers and tool metadata no longer require Node.
`bunfig.toml` selects Bun for dependency shebangs; dependency lifecycle scripts are
disabled because the selected packages provide locked platform binaries. Node in
project toolchains and provider runners retains its separate workload role.

The private Chromium helper uses Bun's Unix WebSocket server, subprocess API and
file-descriptor reader/writer. Bun.file borrows descriptors, so the helper closes
its owned descriptors explicitly. Real pipe exchange/shutdown passed locally;
the direct `ws` dependency was removed. No installed authentication, project
lifecycle or appliance deployment was performed by this migration.

Actual local evidence under `.artifacts/bun-typescript-port/`:

- All four strict TypeScript checks pass. Default root tests pass 73 tests, with
  opt-in browser journeys skipped; Cockpit passes 60 tests and both page builds.
- A fresh frozen dependency install and copied Cockpit build/test pass with a
  failing Node executable first on PATH and no invocation of it. The first audit
  identified Parcel watcher's optional Node source-build hook; disabling that hook
  and using the locked prebuilt package was verified in `no-node-install-3` and
  `no-node-fresh-cockpit-2` logs. An initial copied fixture omitted branding assets;
  the corrected copy includes them. No dependency versions were incidentally upgraded.
- The minified drawer passed all sixteen width/theme/state combinations in a real
  isolated Chromium fixture. The Bun pipe smoke and six actual HTTP avatar pixel
  comparisons also passed. Failed pipe experiments and corrected evidence remain.
- Go checks for `internal/nativebuild`, `internal/web` and `scripts` pass. The seven
  Sodaspaces Python staging/preflight tests and payload tests pass. The broader
  Python run passed 52 tests but had seven failures and one error in unchanged
  Linux terminal tests on macOS (including absent `os.setresgid`), plus one skip.
  That is not native Linux validation; full native x86_64/aarch64 build/check and
  installed journeys retain their existing authorization and proof boundaries.

At the migration handoff, the existing local `sodaos-local-forgejo` preview still
bound the former JS source directory, so its login script returned 404.
`bun run build:preview` now projects the production branding payload, including
minified JS, into `.artifacts/forgejo-preview/branding/`. Its bytes and absence of
TypeScript files were checked. The generated Compose candidate changes only that
read-only branding mount, retains the existing named data volume/configuration,
and passes Compose configuration validation. Original/candidate files are retained
under `.artifacts/bun-typescript-port/preview-compose/`. The migration left the
running container unchanged because its earlier restart authorization was single-use.
The user subsequently approved activation during the responsive-layout work above;
that section records the actual preview update.

## Root Bun workspace and TypeScript scaffolding

The root `package.json` now owns Bun 1.4.2, shared script/test dependencies and
one generated `bun.lock`, with Cockpit as its UI workspace. Playwright and the
native browser helper's direct WebSocket dependency belong to the root. Existing
require loaders now resolve from that owner. All prior locked package versions
survive; additions are Bun/JSDOM/WebSocket types and their declaration dependencies.
Native build/check callers use the root commands; public input collection and the
bundle allowlist carry the root manifest/lock plus the Cockpit manifest.

[TypeScript development](typescript.md) and AGENTS.md establish migration rules.
Shared strict settings include checked indexed access and exact optional properties,
with separate scripts, browser, tests and Cockpit configurations. Cockpit passes
these settings after bounded missing-value guards, explicit potentially undefined
props, and stronger fixture assertions. Its stream character access and exit-node
selection retain their current behavior. Root JS/MJS remains included with
`checkJs: false`: those files are not yet ported or claimed type checked. New TS
files inherit strict checks. Forgejo browser transpilation/emission and source
conversion remain the next migration work; current served JS paths stay intact.

Local macOS arm64 validation used Bun 1.4.2, Node 24.20.0 and TypeScript 7.0.2:
fresh workspace frozen install, all four compiler projects, 54 frontend checks,
16 Forgejo checks, 60 Cockpit tests and both Cockpit builds passed. Fifteen opt-in
browser checks remained skipped. Six Go bundle/input tests passed with Go 1.27.1;
two metadata fixtures passed with the optional Caddy check skipped. A negative
compiler probe confirmed strict/indexed/optional checks and rejection of Bun/Node
globals in the browser configuration. Screenshot help and the native-browser helper
import passed without launching a browser. Shell syntax and whitespace checks passed.

Validation logs/probes and previous dependency directories are retained under
`.artifacts/typescript-scaffold/`. Initial compiler failures are preserved there;
they were resolved without disabling strict options. No native appliance build,
installed journey, retained project/provider mutation, deployment or push occurred.
Earlier migration evidence below remains tied to its original package layout.

## Cockpit package management — Bun

The Bun pin was corrected to 1.4.2 after the initial migration. The official
macOS arm64 binary is retained under `.artifacts/bun-1.4.2/`; its frozen install
left the lock unchanged, and type checking, all 60 Cockpit tests and both page
builds passed. Build/check version guards and installation guidance match 1.4.2.
The globally installed Bun was not changed.

Cockpit now pins Bun 1.4.2, with a generated `cockpit/bun.lock` migrated from
pnpm and `bun run build`, `bun run typecheck`, and `bun run test` commands.
All 287 locked package names/versions are preserved. The explicit
`@parcel/watcher` install-script allowance moves to `trustedDependencies`;
the pnpm lock/workspace configuration is removed. Native build/check callers,
input collection, bundle allowlist/tool metadata and installation instructions
now use Bun. Older retained bundles keep their matching original verifiers.

This is a package-manager migration. Vite+/TypeScript and the pinned Node runtime
remain; converting authored JS/MJS sources and test runners is separate work.
A fresh Bun 1.4.0 frozen install, strict TypeScript check, all 60 Cockpit tests and both
page builds passed locally on macOS arm64. The isolated validation copy/build log
is retained under `.artifacts/bun-migration/`. Its initial build lacked the sibling
branding assets; copying those assets fixed resolution. The remaining build warning
is the intentionally external native `../base1/cockpit.js` script. Metadata fixture
checks passed (two tests; optional Caddy check skipped), as did shell syntax and
whitespace checks. Six focused Go bundle/input checks passed with Go 1.27.1
on macOS; the initial default-cache attempt was sandbox-denied and the broader
retry was interrupted without a result. No native appliance build, installed
validation or deployment.

## CoreOS product strategy — documentation only

Recorded the user's custom CoreOS-based distro direction in the
[OS product strategy](os-product-strategy.md), linked from the architecture, leading
plan, deferred guide and README. It ranks host health/first-boot guidance, native
runner/project resource protection, real private connectivity, maintenance windows,
backup/cold restore and future boot/storage ownership. Each proposal has a bounded
first slice, user outcome, acceptance conditions and qualified effort estimates.

The fastest recommendation is the existing-console/CLI health report; resource
enforcement is the strongest near-term host investment. Boot/provisioning/lifecycle
ownership provides the strongest long-term distro case. A sufficiently privileged
service could manage these mechanisms, so the strategy makes no artificial claim
that portability must become impossible. Current delivery still provisions upstream
CoreOS; a Soda boot artifact, verified boot chain and general recovery are proposals.

Source review distinguished native runner descendants from external engine work,
and actual project cgroups from the outer Podman launcher unit. Estimates separate
source candidates from bounded native proof. The strategy leaves immediate tmux/
workspace work, deferred implementation scope, provider authority and the separately
reserved Updates work with their existing owners.

Checks: reviewed production source and official upstream documentation; touched-doc
relative links/anchors and whitespace passed. Evidence is retained under
`.artifacts/research/os-product-strategy-eb758ff/`. No product code, package/dependency
changes, builds/product tests, native execution, project/provider/network mutations,
deployment or push. This records direction and recommendations, not implemented
features or additional execution authority.

## Merged presentation and Sodaspaces payload

Merged upstream `416180e` with local `8d9b2eb`, preserving both histories and their
evidence below. The shared header keeps the local presentation revision/asset
versions plus the incoming drawer, terminal and xterm styles. Its reviewed inventory
hash now binds the combined bytes; stale home-logo and seven settings-role entries
were corrected to match the existing local templates. The deployment allowlist adds
the nine local templates and both Soda Forge wordmarks, and drops deleted `issues.css`.
All 367 payload entries have their selected source/build ownership; no new product
feature or installed state change is implied by this merge.

Local checks passed: Go scripts and host packages; focused nativebuild payload,
allowlist and private-output tests; 62 Node frontend/presentation checks; and nine
Python payload/synthetic-staging checks. The opt-in Chromium layout test was skipped.
Prepared the existing frozen Cockpit dependency lock with install scripts disabled;
no dependency manifest/lock changed. Initial host/nativebuild checks hit macOS
temporary-path length/symlink constraints; the relevant checks passed with real,
short `TMPDIR=/private/tmp`. The installed-stage suite could not run without an
actual `SODA_STAGE`; no native appliance build, installed journey or deployment ran.
Initial failures and corrected local results remain under
`.artifacts/merge-8d9b2eb-416180e/`.

## Project OS baseline consolidated — documentation only

At the user's request, replaced the stale, unlinked [Project OS guide](project-os.md)
with the existing foundation: native accounts/sudo/trust, supported tools and ordinary
extension points, persistent versus runtime state, separate SSH/browser/Git credentials,
service/cgroup ownership and same-container lifecycle. Linked it from the leading
plan, architecture, agent/readme and feature/installation/validation guides. Corrected
old claims that the image/GitHub CLI packaging and default-bridge/lifecycle checks
had never completed; retained evidence remains bounded to its original bytes/targets.

Two source distinctions matter: web management consumes current Forgejo authority,
while native wheel grants use the creation-time owner label and are not revoked by
that account helper; inner workload cgroups being disabled does not prove terminal
systemd/cgroup cleanup. Do not invent synchronization, new capabilities or broad
native compatibility to hide either boundary.

The selected required-addition delivery is bounded **same-root native maintenance**:
reviewed package/dependency/scriptlet/file/service effects, original CID/accounts and
fresh matching preserved-state scope, no image replacement or install on Open. Its
concrete recipe/checks remain source work for tmux delivery, not a fleet updater or
execution permission. Tmux supervision/retention remains the immediate slice; zero-key
real onboarding and the unresolved personal Git credential model are separate later
work. No new universal Project OS planning phase is selected.

Review used production source and the saved public
`.artifacts/e2e-dad2945/export/x86_64/inputs/native-build.json`: that prior x86_64 image
record includes `vim-minimal`, `less`, CA packages and the selected Tea/GitHub CLI
versions, not tmux. It is not an audit of retained projects or evidence for current
HEAD. Notes/document checks are retained under
`.artifacts/research/project-os-baseline-0848079/`.

Checks performed: source/retained-metadata review, Markdown relative-link/anchor
inspection and `git diff --check`. No product code, dependency/package installation,
build/test, terminal execution, VM contact, service/project/key/provider mutation,
deployment or push. Source layout `4cb7f7d`, installed isolated `2aa4960`, retained
`soda-test`, both isolated project roots, credentials and earlier evidence are unchanged.
Tmux/native delivery and new acceptance remain unimplemented/unverified.

## Resumable terminal selection — tmux, documentation only

Following the user's request to find the best native persistence option, the
[leading plan](sodaspaces-plan.md#resumable-terminal-decision--tmux) now selects
**stock Rocky-packaged tmux** for the next single resumable terminal. The
[terminal contract](terminal-integration.md#selected-persistence-mechanism--tmux-source-candidate)
records the comparison, exact upstream sources, native ownership and proof required.

Source review found the decisive shpool v0.11.4 limitation: no require-existing
option in its CLI/attach protocol; missing/exited sessions can create a new shell.
List-before-attach still races. Tmux 3.2a supports exact attach-only with `-N`,
private sockets and a foreground server. Rocky 9 metadata lists `3.2a-5.el9` for
both x86_64/aarch64; this is availability, not installed package/native proof.
Research is retained under `.artifacts/research/terminal-options/`.

The selected candidate keeps xterm/Soda authentication and uses one private,
project-local supervised tmux server per managed browser terminal, under the
original Linux account. Creation, attachment, retention and End are separate.
Systemd/cgroup ownership plus the native safety lease must supervise the server and
remaining owned descendants, not just its attachment client. Ordinary SSH/tmux and
other terminals stay separate. Tmux history/selection/paste need actual UX review;
its hidden status bar does not make scrollback identical to a plain xterm shell.

**Still unimplemented:** package/unit/config, detached lifetime owner, resumable
API/metadata, Refresh/navigation continuity and new native tests. First prove one
session; multi-terminal tabs and Spaces reuse it afterwards. Existing first-layout
source `4cb7f7d`, installed isolated `2aa4960`, retained `soda-test`, both isolated
project roots and all earlier evidence are unchanged. No automatic shell replacement,
restart resurrection, forced SSH configuration or runtime package installation.

Checks performed: public upstream source/release/package-metadata inspection,
documentation relative-link/anchor inspection and `git diff --check`. No product
build/test, dependency installation, tmux/shpool session execution, VM contact,
service/project/key/provider change, deployment or push occurred. This selects the
source candidate, not runtime acceptance or new execution scope.

## Global Spaces page selected — documentation revision only

The user selected a global Spaces link alongside native navigation and a Soda-owned
Go/template page at `/-/soda/spaces`. The [leading plan](sodaspaces-plan.md#spaces-page--selected-not-implemented)
now supersedes the environment-catalog exclusion and records authorized listings,
a fixed OAuth return and reuse of the same drawer/terminal sessions. Architecture,
API, deferred-scope and agent guidance distinguish this planned HTML addition from
today's API-only service. Future Runners navigation remains Soda-operator-only.

**Not implemented:** navbar/page handler, authorized collection, Spaces OAuth return
or authenticated page-shell composition. Shared assets do not import Forgejo's native
session/template/CSRF context. Resolve the supported shell without borrowing cookies,
relaying HTML or recreating upstream authentication/workflow authority. Existing
JSON protections, current repository-filtered API and independent logout stay intact.

Terminal continuity remains the immediate coding task; this revision does not gate
it on a new planning project. Source layout slice `4cb7f7d` and its evidence remain
unchanged. Documentation link/anchor inspection and `git diff --check` passed. No
product code, builds/tests, native execution, credential/provider changes, deployment
or push in this revision; no Spaces runtime acceptance is claimed.

## Workspace step 1 — first source/layout slice, not native delivery

Under the user's instruction to start the four implementation steps, replaced the
modal shell with a non-modal aside and a desktop 50/50 native-page/workspace split.
A pointer/keyboard separator adjusts width between 35–65%; narrow screens use the
full workspace with Hide returning to the native page. There is no backdrop,
outside-click dismissal, inert native page or focus trap. Native routing, forms,
notifications and beforeunload remain untouched. Source scopes native body/container
widths, but actual Forgejo page-specific reflow still requires browser validation.

Terminal, Environment and Access **view tabs** now separate management/SSH forms from
the full-height terminal. These are not multiple terminal sessions yet. All three
frontend layers no longer retire on window blur/hidden visibility. Hide/reopen and
view changes retain the same component/socket without new reads, commands or native
mutations. End terminal is explicitly separate. Keyboard tabs, separator and terminal
focus escape are covered. Footer structural inventory was reviewed; notification
markup is unchanged and its hash was calculated from the actual changed template.

**Remaining work is significant:** the backend still owns a single request-bound
PTY per context/project, with its existing two-hour/session expiry limit. Actual
pagehide/BFCache, transport loss and Refresh still end the terminal; there is no
server-detached 30-minute grace, multiple terminal tabs, navigation restoration or
command replay. Those are step 2, not a claim made by retaining a live browser
mount. Browser-only joining, Forgejo-key reuse and outbound Git setup (step 3) are
unchanged; no new keys, scopes or provider resources were created. Combined native
workflow proof (step 4) has not run. Installed fixtures remain unchanged.

Checks actually run on this first source slice:

- 54 frontend Node tests passed; one opt-in layout test skipped in that invocation.
  Includes late terminal readiness not stealing focus from the native pane and
  refusal of synthetic hidden-view key deletion / terminal-launch clicks.
- Separately, the sandboxed Chromium layout test passed all 16 combinations of
  1440/900/390/320 widths, light/dark and running/stopped synthetic states. Real
  locked xterm/CSS, synthetic HTTP/WebSocket and a native-form stand-in—not actual
  Forgejo/OAuth/helper proof. It checks full-height canvas, half-width desktop,
  editable left form, pointer-click non-dismissal, keyboard resize, view changes
  and same-socket Hide/reopen. Desktop/mobile captures were visually inspected.
- Go tests passed for `./scripts` and `./internal/nativebuild` (the latter cached).
- Presentation inventory passed; embedded upstream-caller test skipped because its
  optional local export was absent. Seven Sodaspaces Python packaging fixtures passed.

Logs/screenshots: `.artifacts/workspace-d57f128/`, with final layout captures in
`layout-1788897855045/`. Visual review of an earlier 320px capture caught the Access
label painting beneath Hide despite the initial layout test passing. The tab strip
now has a separate clipped width; a new non-overlap assertion and all 16 browser
cases passed, and the corrected narrow capture was inspected. Earlier captures are
retained, not relabeled as final. Documentation link/
anchor inspection passed (94 files, 563 links, 70 anchors, zero errors) and
`git diff --check` passed. No dependency installation, native build/export, real fixture
journey, service/container/account/key mutation, deployment or push. This is a
reviewable first implementation slice, not acceptance of all four steps. Preserve
all existing native evidence and both retained isolated projects.

## Product UX rejected — workspace correction is next, not delivery

After using the installed fixture, the user rejected immediate terminal termination
on tab/app changes and the modal management-form drawer. They clarified that **all
development happens in the drawer**: a terminal fills its entire content area,
session/management tabs occupy a compact top bar, and the native Soda/Forgejo page
must remain usable on the left at roughly half-screen width. The current backdrop
turns that page into a large dismiss target; passing the old tests did not establish
an acceptable development experience.

The revised requirements and proposed credential integration are now recorded at
[the start of the leading plan](sodaspaces-plan.md#product-correction--development-workspace-not-a-modal-form).
No runtime fix has been made. Source and installed `2aa4960` still have modal blocking,
three layers of blur/hidden retirement, a two-hour absolute terminal cap, one stream
per context/project, socket-owned PTY lifetime and mandatory SSH keys at join. Native
Forgejo navigation replaces the document: merely removing blur or changing CSS
cannot preserve development while using the left pane. Multiple terminal tabs,
non-modal pane-width layout and bounded same-process reattachment remain source work.

The user also asked to reuse Forgejo profile public keys and automate project Git
credentials rather than demand manual SSH entry. Inspection of exact Forgejo 15.0.7
source confirms acting-user public-key listing/registration through official APIs;
listing uses `read:user`, registration needs `write:user`, which Soda does not currently
request. Browser terminal transport itself uses no SSH key. Browser-only joining
therefore needs an intentional API/helper/project-script change, not a UI-only bypass.
Existing keys, account markers and memberships must be preserved.

The recommended outbound Git candidate generates a distinct keypair inside each
user/project account and registers only its public half, rather than distributing
one private key everywhere. This credential model is **not yet selected or implemented**:
a Forgejo profile key still carries the user's normal cross-repository Git permissions,
and project sudo administrators/root can read project-resident private keys. At-rest/
passphrase handling and explicit user consent must be agreed. Do not blindly install
all profile keys for inbound project SSH, including workspace-generated Git keys;
that could silently enable lateral access. No automatic later synchronization or
new provider mutation authority follows from this discussion.

Checks this turn: tracked tree inspection and product/exact-selected-upstream source
review; documentation edits only. The documentation checker passed (94 Markdown
files, 563 relative links, 71 anchors, zero errors), as did `git diff --check`.
Earlier public terminal UX comparisons are retained in
`.artifacts/research/terminal-ux-5f33120/`. No builds/product tests, browser/native execution,
key generation/registration, credential/scope change, deployment or push occurred.
The technical evidence below remains valid at its recorded scope, **not acceptance
of the rejected UX or proof of these corrections**. Preserve both isolated projects,
all later user work and the installed-versus-exported candidate distinctions.

## Integrated native E2E passed — bounded isolated x86_64 fixture

Installed application/helper candidate `2aa4960` passed the real integrated
existing-project management journey and fresh Create/Join journey with test source
`1c02fb5`. This is bounded first-product integration evidence, not retained delivery,
whole-appliance/operator/provider/aarch64 or release acceptance. `soda-test` was not
contacted. The original fixture project and all earlier inputs/failures remain.

- `browser-g` passed actual TLS/native Forgejo OAuth/two identities, BFCache,
  original-login browser terminals and Escape/Disconnect/no-reconnect, explicit
  Stop with a live terminal, disabled boot start, same-container Start, restored
  boot policy, persistent home marker, account/group/host-key/key-file preservation,
  actual nonowner lifecycle denial, temporary-key replacement and explicit removal.
  New key B authenticated; removed A failed **public-key authentication**, while
  already authenticated A and Bob SSH sessions survived. Both temporary saved keys
  were then explicitly removed/applied through the UI; original key rows and managed
  files are preserved. No original saved key was deleted.
- A single new public run-owned repository `/alice/soda-e2e-1c02fb5` was created
  through Alice's actual native form. `browser-access` passed owner Create,
  nonowner denial, each user's explicit key Save/Join, own connection and native
  Copy/real paste. New project `p9568a9ea83a6c84e66c6b975`, CID
  `cfdaa1def67a97b37c4af5c5b61b851aeb3a328caf6b2b8a8ac21d72ded86748`, is retained.
  The product-owned `developer-access.py` passed independent SSH/PTY/SCP/SFTP,
  UID-map, owner/nonowner sudo and cross-user authentication-denial checks. The
  client used explicit management-SSH forwarding, **not direct/laptop routing proof**;
  the script now accepts a private SSH configuration and records that distinction.
- Final independent observations bind the actual running backend/helper and all
  368 custom-file bytes. Original users/key rows/grant-key check and original managed
  key files match preflight/backup. Schema5 integrity/FKs pass; counts are users2,
  keys2, projects2, memberships4, sessions9/grants9 after legitimate auth transitions.
  Both projects are running/boot-enabled on the unchanged original project-image
  default. Earlier process observations independently confirmed terminal shell
  disappearance; socket closure alone was not the evidence.

Evidence: `.artifacts/e2e-2aa4960/`, including the fresh paired guest/local backup,
all browser profiles/results, new client files/transfers, native observations and
failed runs. Preserve new home markers and both project roots. No reboot, host
routing, provider job, original-key deletion, project-image replacement or retained
rollout occurred. Old backups are not lossless rollback over the new repository,
project, auth transitions or later writes.

Retained failures matter: D/F used a stale client IP after native Start changed it
from `10.90.0.2` to `.3`; the helper/UI correctly exposed the new address. The probe
now compares the current CID-bound address with the drawer and uses a separately
pinned host-key alias, without changing routes. G observed `.3` then `.4` and passed.
E failed before management at real back-forward restoration; it is not a passed run.
B's native PTY parser failure and both pre-write maintenance refusals remain below.

Final mode verification also found a packaging discrepancy: 11 legacy theme/logo
adaptations inherited private-checkout 0600 modes in the bundle, while the reviewed
fixture maintenance installed public files as 0644. All bytes match; the explicit
mode differences are retained in `final-check-reviewed.json`, not hidden as exact
mode equivalence. Source staging now normalizes only its run-owned public adaptation
files/directories, with a private-checkout regression test. The corrected packaging
passed a clean native build, `check-native.sh x86_64` and verified export at
`dad2945`, retained in `.artifacts/e2e-dad2945/`. Every staged Forgejo public asset now has
0644 file/0755 directory modes even from the private worktree. No new application
behavior was introduced: production application/helper source remains the tested
`2aa4960` code. The newer whole bundle is not installed; current fixture runtime
and its 11 explicitly recorded mode differences remain bound to the evidence above.
No additional restart or retained rollout was performed to hide that distinction.

## Integrated E2E execution started — isolated fixture only

The user authorized proceeding with E2E testing after the explicit build/native
journey, Stop/Start and temporary SSH-key rotation proposal. Execution is bounded
to this x86_64 builder and `soda-native-spaces-658f2af`; preserve its original project,
accounts, keys, later writes and evidence. Fresh paired backup precedes fixture
service changes. No `soda-test`, provider, routing, project deletion or host reboot
is included. New work/evidence: `.artifacts/e2e-806d0d9/`.

Clean `806d0d9` passed full native build, `check-native.sh x86_64` and verified bundle
export from its own detached worktree. Read-only pinned-SSH preflight confirmed the
original CID/image, one project/two memberships/two keys, schema5, running services
and query-free native logging. No fixture mutation or browser proof yet.

Preflight reproduced a real lifecycle compatibility blocker: systemd
`259.8-1.fc44` supplies the global `service.d/10-timeout-abort.conf`, setting only
`TimeoutStopFailureMode=abort`. The helper wrongly required no drop-ins whatsoever.
Source now admits exactly that stock vendor path (or no drop-ins), while refusing
all other/combined overrides and retaining fixed unit/CID validation. Host policy
was not changed. Focused host/web tests passed; the corrected candidate still needs
its own clean build/check and integrated native execution. Initial bundle/evidence
remain preserved, not relabeled as the corrected build or E2E success.

Follow-up: clean `2aa4960` passed full native build/check/export. Its exact dashboard,
helper and full customization payload were installed only on the isolated fixture,
after a fresh quiesced paired backup (guest `/var/lib/soda-e2e-2aa4960/backup`, local
`.artifacts/e2e-2aa4960/fixture-backup.tar`). Original project CID/image/running state,
Soda rows/grant ciphertext and credentials were preserved; project defaults/units,
accounts/keys, routing and `soda-test` were untouched. Two pre-write maintenance
refusals are retained: the full payload also contains 11 legacy theme/logo files,
and this fixture used its original verified `:dev` reference rather than the retained
target's image-pinned unit. The fixture now uses the candidate's immutable image pin.

The real guarded public-repository OAuth/browser journey passed against this installed
candidate (`browser-a`): trusted TLS, actual native forms and two identities, actor/
CSRF denials, native theme/focus/form coexistence and real BFCache. This is not
yet terminal/lifecycle/key E2E proof. An explicit existing-member terminal mode is
now authored in the same installed entrypoint; native execution and independent
process-disappearance checks are next. Stop/Start and key rotation remain unexecuted.

The terminal follow-up passed as test revision `5f459a1` against unchanged installed
`2aa4960` product bytes (`browser-c`). Both users exercised real native login/OAuth,
mounted xterm, original login/UID/GID/groups/home/TTY, Escape/focus escape, explicit
Disconnect and no remount after Refresh. Independent pinned host observations
confirmed both original shell PID/start identities disappeared, original managed
keys remained byte-identical and the CID was unchanged. `browser-b` is a retained
failed fact-parser attempt; native PTY CSI/CR framing was corrected in the probe,
not stripped from the product shell or turned into a synthetic response.

An explicit existing-project management extension is now authored in the same
installed entrypoint: Stop/Start, persistent marker, live-terminal interruption,
nonowner denial and two temporary-key rotation/revocation with independent SSH and
held-session checks. It preserves original saved keys and uses only exact approved
fixture aliases; root transport observes, UI/API performs mutations. It has not yet
run. This does not authorize new targets, deletion, routing or a retained rollout.

## Mounted Sodaspaces management and terminal in the established UI

The native repository button/right dialog now mounts the complete control component
through a single `sodaspaces.js` shell. Its historical duplicate API/action caller is
removed. Connect/logout, create, saved-key add/remove, join, own-key review/Apply,
Start/Stop, SSH/native Copy, Refresh and explicit terminal Open/Disconnect remain
separate actions. Existing native forms/navigation/notification hooks are preserved.
The drawer uses shared typography/tonal 44px buttons, responsive width and scoped
terminal styles. It retires on close/stale context; reopening requires an explicit
full-page reload rather than evading terminal or uncertain-operation guards. Refresh
cannot remount a used terminal. Mutations recheck session identity before dispatch;
JSON reads are MIME-checked and streaming-bounded to 64 KiB. No backend authority,
provider rule, schema, lifecycle or key semantics changed in this UI integration.

The merge's partial-payload packaging hold is resolved **in source**: staging and
the compiled verifier share `internal/nativebuild/forgejo-payload.json` (357 exact
entries, including 229 templates, presentation assets/fonts/notices and generated
locale/terminal inputs). The installer refuses unsafe ancestors and occupied
customization destinations, and chowns only admitted entries, not mutable data trees.
Native build now verifies the locked complete upstream 15.0.7 English catalog before
merging the Soda namespace; GPL/font/Soda/renderer notices remain paired with payloads.
This is not a new native candidate build, exported bundle or installed frontend.

Local checks passed: full Go suite; focused web/host/store/nativebuild races; 65 Node
tests (two opt-in/export-dependent checks skipped); 60 Python build tests (one opt-in
Caddy check skipped); shell/installed-probe syntax, documentation and whitespace.
A separate opt-in sandboxed Chromium layout test passed all 16 combinations of
1440/900/390/320 widths, light/dark and running/stopped, with real source styles and
hash-verified xterm, native-form preservation, keyboard escape and reload behavior.
Its APIs/WebSocket are synthetic: **not real Forgejo/OAuth/helper terminal proof**.
Logs and screenshots: `.artifacts/ui-integration-c6df099/`. Initial failures are
retained: presentation hook snapshots required explicit integration review, a closure
test wrongly required stock `custom/extra_tabs`, and the actual-stage packaging suite
refused to run without `SODA_STAGE` (zero tests; not a passing stage check).

The existing installed journey was adapted to reload/retired-context semantics and
current status text but was not executed. Combined native OAuth/proxy/helper/browser
proof, full candidate build/check/stage and native Start/Stop persistence/new-key
success/old-key refusal remain pending. No dependency install, service reload,
VM/project/account/key/provider mutation, retained rollout or push occurred. The
previous terminal-only fixture approval does not cover lifecycle/key mutations;
obtain exact scope first. Destroy and operator runner relocation remain separate.

## Public contributor profile redesign

Local presentation revision `2026-09-08.32` implements the approved compact
horizontal identity header and open repository rows. All five public-profile
callers share the same header, including personal projects, packages and code
search. Native tab payloads/defaults, activity/email/organization privacy gates,
follow/unfollow HTMX targets and permission-gated actions remain authoritative.
The block confirmation is shared outside the morph target on each caller.
Organization branches retain their previous compositions. No settings, backend,
account data or appliance changes were made.

Validation: focused `go test ./scripts -run TestForgejo` passed with local toolchain
and offline module settings. Three settings source-contract tests and the embedded
native-caller inventory check passed. The new read-only profile browser suite
passed 13 tests (12 theme/width combinations across seven public destinations,
320–1440px, plus keyboard opening of the native actions menu). The full inventory
check still fails on the pre-existing `custom/footer.tmpl` hash mismatch; a separate
scan confirmed that is the only hash mismatch. It was not silently rebaselined.

Reviewed native captures are under `.artifacts/public-profile-redesign/review-*`:
desktop light/dark repositories, public activity and empty packages; light mobile
repositories, empty followers and projects; desktop starred repositories. Captures
verify URL/status/landmark/revision/server stylesheet bytes/browser errors. The
first captures under `verified/` and `final-*` are superseded: cached CSS produced
stale rendering. `profiles.css?v=8` resolves it in the reviewed captures.

Remaining evidence gaps: code search redirects in this fixture and is not accepted
as a capture; profile README, populated people/organizations/badges/package versions,
private/admin/self combinations and actual follow/block mutations were not exercised
natively. Activity privacy has source-rendered permission-matrix coverage. These
limits are not claims of complete native functional acceptance. Existing local
preview only, using template reload; no service restart or deployment.

## Shared milestone, issue and pull-request rows

Local revision `2026-09-08.31` gives milestone, issue and PR lists one shared
appearance in `components-list.css`: 24px row padding, 20px linked titles,
13px secondary metadata, aligned status icons, uniform backgrounds, matching
progress tracks and one quiet separator between complete items. Hover/focus
feedback remains visible. The PR-only `issues.css` adapter and repository list
corner adapter are removed; milestone CSS retains only its specialized layout
and Markdown preview. The native shared issue partial is unmodified, so repository,
dashboard, milestone-content and notification-subscription callers share the
same presentation without copying permissions, routes or script hooks. Existing
status/label colors, selection, reviews, assignees and comments remain native.
Page chrome and other collection families keep their existing presentation.

Actual checks: focused Go Forgejo source checks passed; the milestone item-boundary
and responsive tests passed; all 16 component-boundary checks passed. The new
work-item component test passed for issue/PR samples in four caller compositions,
both themes and 1440/1024/700/601/600/390/320px, including long titles/labels/branches,
keyboard checkbox selection, matching title hover, and unrelated-list protection.
Its isolated native-snippet gallery is under
`.artifacts/work-item-consistency/components/`. Source review confirmed native
checkbox, pinning, status-popup and PR metadata contracts remain unchanged.
The 236-entry inventory's native caller check passes; its pre-existing footer
hash mismatch remains the only hash failure and was not silently accepted.

Read-only native layout checks passed for the rows in 56 route/theme/width
combinations: open/closed issue and PR lists, open/closed milestones and milestone
issue contents, at 1440/1024/390/320px. Requested URLs, 200 responses, main landmarks,
revision and production stylesheet bytes were checked. Rows fit at every width.
The pre-existing milestone **detail header/filter** still overflows at 390/320px;
that surrounding layout is outside this list-item pass. All six collection routes
fit. Global populated lists, native owner bulk actions and mutation journeys were
not newly exercised; no credentials/resources/preferences were changed for coverage.

Desktop dark and mobile light full-page diagnostic captures of all three lists
were made through `scripts/screenshot.mjs` and reviewed. They remain separate
from accepted evidence: `--verify` still rejects the existing
`/assets/sodaspaces.js` 404. Logs, native layout measurements and diagnostics are
retained in `.artifacts/work-item-consistency/`. Activation used only the existing
local preview and native template reload; no restart or appliance deployment.

## Milestone list items

Local revision `2026-09-08.30` applies the approved uniform-canvas row design to
milestone items only. Global and repository lists share one small row partial:
compact sans-serif titles, repository identity where applicable, two-line rich
text previews, aligned native deadlines/progress/counts and quiet separators
between complete items. Mobile metadata stacks inside its row. Focused embedded
links expand the preview; full native Markdown remains on the milestone detail.
Tracked time, updated/closed dates, no-deadline state, completeness and gated
repository actions remain native. Page headings, search/filter/navigation,
pagination and empty states are unchanged, protected by before/after boundary
hashes. Native project lists retain their existing reused milestone-card styles.

Go Forgejo source checks and focused item-boundary/layout checks passed. The
component check covers light/dark at 1440/1100/1024/900/768/700/390/320px, including
long descriptions and keyboard reveal. Read-only native layout checks covered
12 populated repository views (two open lists and one closed list at four widths),
with no overflow and uniform backgrounds. The global screenshot fixture has no
milestones, so its populated native view remains unverified. No fixture or
repository mutation was used to manufacture coverage.

Verified capture attempts were rejected because the existing preview returns
404 for `/assets/sodaspaces.js`. Diagnostic desktop/mobile captures made through
`scripts/screenshot.mjs` are under `.artifacts/milestone-rows/`, separately labeled
and not accepted verified screenshots. The missing asset is outside this
item-only change. The previously known footer inventory hash mismatch also
remains; the updated 236-entry inventory preserves that unresolved mismatch.
Activation used local template reload only; no appliance changes or restart.

## Repository settings consistency overhaul

Local revision `2026-09-08.29` replaces the repository-settings sidebar with
compact grouped navigation beside one task title, retaining native repository
identity and unit navigation. Personal and repository settings now explicitly
share `.soda-settings-shell`; `components-settings.css` owns their navigation,
1120px usable canvas, 24/16px gutters, 40px body inset, section placement and
actions. The old competing repository sidebar/grid adapters were removed.
Principal file inputs share width constraints, fixing the native avatar picker's
mobile overflow. Ordinary repository containers keep their zero-padding contract.

The pass covers General settings and native Units partials, branch/tag protection,
collaborators, deploy keys, webhooks and child editors, Actions runners/secrets/
variables and LFS leaves. Open sections, shared empty states, semantic notices,
44px tonal controls, parent links and explicit primary-form roles replace the
older attached task stacks. Native save boundaries, controls, IDs, permission
and feature gates, destructive dialogs, provider dispatch and technical viewers
remain intact. LFS totals remain visible; wide tables scroll within their canvas.
Ordinary landing autofocus was deliberately removed; dedicated tag editing and
panel/editor focus remain. Shared runner/webhook/secret/variable presentation uses
an explicit opt-in; organization/administrator default callers retain native
markup contracts. No locale additions or service restart were required.

Actual checks: Go Forgejo checks passed, including exact native navigation across
256 gate combinations and shared default-caller regressions. Five personal/repo
source-contract tests passed; the repository snapshot covers 26 native templates.
The final component/gallery/ordinary-repository batch passed 42 checks, including
light/dark, 1440/1000/900/899/768/390/320px, keyboard navigation, no-JavaScript
fallback, long labels, gutters, upload width and table containment. All 11 native
personal-settings regression checks passed after the shared-style extraction.
The inventory covers 235 overrides/helpers and exact local/native callers; its
embedded-native-caller check passes. The pre-existing unrelated `custom/footer`
hash mismatch remains the only inventory hash failure and was not silently
accepted by this pass.

Read-only live UI inspection reused the user's already-open Chrome session on
`alice/activity-workbench/settings`. All seven permitted landing destinations
were checked at 1440, 900, 899, 390 and 320px with revision `.29`, the 40px inset
and no page overflow. Branch-rule creation and the Forgejo webhook editor were
checked at 1440/390/320px; actual menu navigation, provider selection and custom
event disclosure worked without a submission. No console errors were observed
in the captured browser log. The viewport was restored and the tab returned to
General settings. These are live DOM/UI checks, not verified screenshot captures
or proof of persisted workflows.

The production-derived gallery and six component captures are under ignored
`.artifacts/forgejo-presentation/repository-settings-{light,dark}.html` and
`.artifacts/repo-settings-overhaul/components/`; logs and a review index are in
`.artifacts/repo-settings-overhaul/`. The existing screenshot fixture has no
repository-admin access; dedicated owner-profile captures through
`scripts/screenshot.mjs` remain a verification prerequisite. Actions, LFS and Git
hooks are not exposed by the inspected repository's native gates. Populated
credentials/protection rules, mirrors, runner setup, provider delivery history
and success/error/mutation journeys remain unverified. Source and component
completion is not full native visual or functional acceptance. No resources,
credentials, permissions or saved preferences were changed; activation used
native template reloads only in `sodaos-local-forgejo`. Appliance rollout is separate.

## Personal settings consistency review

Local revision `2026-09-08.26` consolidates native and nested section headings on
the shared 24px token at every width, retaining the 40px body inset and 44px
controls. Shared form padding now preserves inline search-icon clearance.
Disclosure summaries use button typography. Personal webhook event fieldsets are
open, with the main legend aligned to the section-heading column.

SSH/GPG/principal, WebAuthn, authorized/owned OAuth and token repository-selector
empty inventories now use shared empty-state presentation. Generic explanatory
copy is accompanied by an explicit localized empty-result title; empty authorized
OAuth no longer claims access has been granted. Personal Actions secrets,
variables and runners select the same treatment through explicit context
adapters, preserving other shared callers. Runner setup's last-chance credential
guidance becomes a personal-settings warning. Empty personal cleanup previews
omit their blank heading and empty table. Native controls, populated branches,
form handlers, capability gates and submission boundaries remain intact.

Go Forgejo source checks passed. The final component/gallery/repository-container
run passed 26 checks; the native personal-settings suite passed 11, enumerating
all 11 permitted destinations at 1440/390/320px and exercising navigation across
the 900px transition, avatar focus/fallback, key/password editors, empty notices,
token repository selection and custom webhook events. The three settings source
contract checks and embedded native caller inventory check pass. All 229 inventory
entries were inspected for hash mismatches: only the previously recorded merged
`custom/footer.tmpl` mismatch remains, so the full inventory suite is still not
green. Changed templates have reviewed hashes and caller mappings.

Verified full-page native screenshots for all 11 landing pages and three child
editors are in `.artifacts/settings-consistency-pass/release-desktop/` (dark,
1440×1000) and `release-mobile/` (light, 390×844). Visual review covered the full
pages, with an independent desktop review. Earlier directories contain interim
captures. The production-derived gallery was regenerated separately; its registry
renderer now resolves both native asset URL prefixes. Evidence is local-only.

Actions/Storage, enrolled/mandatory factors, providers, populated credentials,
existing cleanup-rule previews and mutation/error submission journeys remain
access- or mutation-dependent native coverage gaps. No resources, credentials or
preferences were created/changed to manufacture coverage. Activation used native
template reloads in `sodaos-local-forgejo`; no service restart or appliance rollout.

## Settings feedback, empty inventories and action placement

Revision `2026-09-08.23` presents SSH-disabled guidance as info, recovery/key-loss
guidance as warnings, and preserves the account-deletion danger message. Static
notices explicitly remain visible within native forms without exposing inactive
validation messages. Cargo context and consequences are separate info/warning
notices before its action in English; other translations retain their complete
warning text. No locale-cache restart or catalog changes were needed.

Access tokens, personal webhooks, organizations, repositories and cleanup rules
now use shared empty-state presentation only when the native inventory is empty.
Webhooks no longer render a blank heading. Section-level actions use heading rows
that wrap on mobile; page-level actions use a left toolbar; submissions follow
fields/guidance. This includes Appearance, enrollment and Cargo/Chef. Native
absolute header-action positioning was removed within personal settings after
the 320px visual review caught overlap on Cleanup rules.

Go Forgejo checks, 15 component checks and 11 native settings checks passed.
The native suite checks every permitted destination plus message/empty-state
visibility and heading-action flow; component tests protect hidden validation.
All 44 route/width layout audit states passed. Verified screenshots are under
`.artifacts/settings-feedback/`; final Packages captures use `mobile-release/`
and `desktop-release/`. Earlier capture folders include intermediate evidence.
Representative native dark desktop/light mobile views were reviewed. Populated
credentials, mutation journeys, foreign-language Cargo splitting and gated
Actions/Storage remain outside native coverage. Existing shared webhook caller
parity is protected; the unrelated merged-footer inventory hash mismatch remains
recorded. Local template reload only; no submissions or appliance rollout.

## Shared settings inset across every destination

Revision `2026-09-08.22` replaces the incomplete direct-section-body inset with
one 40px inset on the shared personal-settings content canvas. Native section
headings, inventory headings and fieldset legends return to the outer edge.
This covers Profile, Account, Appearance, Blocked users, Security, Keys,
Applications, Webhooks, Organizations, Repositories and Packages. Conditional
Actions/Storage and child editors use the same canvas; their gated states remain
unverified natively. Shared form legends no longer reset inline margins.

A read-only audit passed all 44 route/width combinations (all 11 visible
destinations at 1440/900/390/320px), with the 40px inset, heading alignment,
HTTP/URL checks and no document overflow or page errors. All 14 component checks
and 11 native settings checks passed; the latter now enumerates every permitted
menu destination. Go Forgejo checks passed and the production gallery was
regenerated. Verified representative screenshots are under
`.artifacts/settings-uniformity/`; final Cargo alignment uses `packages-final/`.
The existing merged-footer inventory hash mismatch is still separately recorded.
No data changes, permission changes, service restart or appliance deployment.

## Settings body inset

Local revision `2026-09-08.21` uses the requested 40px inline-start inset to direct settings
section bodies, leaving titles flush. Nested native body wrappers do not compound
the padding; the Security password section now explicitly selects the body role.
Go Forgejo checks and all 14 component checks passed, including the inset and
overflow checks at five widths. Activated by local template reload only.

## Consistent section heading placement

Revision `2026-09-08.19` removes personal settings explanation/title columns.
Account, Appearance and all Security subpartials use one column: heading, then
body, with a 12px gap and shared section spacing. The obsolete explained role
and responsive exception were removed. Inventory heading actions stack below
on mobile. Repository/organization/admin section titles already precede their
content; their navigation and technical data grids remain unchanged. Native
forms, notices, enrollment state and submission boundaries are preserved.

Go Forgejo checks and 14 focused component checks passed, including heading/body
placement at 1440/900/899/390/320px. All 11 native settings browser checks passed,
including native heading placement, navigation and editor behavior. The obsolete
blocked-users row selector now accepts its dedicated empty state. Verified screenshots of Account, Appearance,
Security and Keys are under `.artifacts/section-headings/`; representative dark
desktop and light mobile images were reviewed. Gallery fixtures use the same
section roles. The known merged-footer inventory hash mismatch remains outside
this change. Activation used only the existing local preview template reload;
no account mutations or appliance rollout.

## Empty page and section presentation

Local revision `2026-09-08.18` introduces a centered open empty-page treatment
for Blocked users and a compact left-aligned treatment for deploy-key sections.
Shared empty-state icons use a tonal circular accent; inset notification states
also use the compact composition. Existing native messages, conditions, populated
rows and permitted actions remain unchanged. The gallery includes both variants.

Go Forgejo checks and 13 focused component checks passed, including both themes
at 1440/390/320px. Verified native Blocked users screenshots were visually reviewed
in dark desktop and light mobile under `.artifacts/empty-states/`. Deploy-key and
organization caller states were source-checked, not captured with an owner session.
The inventory native-caller check passes, but the full inventory check exposes a
pre-existing `custom/footer.tmpl` hash mismatch from the merged Sodaspaces work;
that unrelated baseline was not silently approved. All changed template hashes
match their reviewed entries. The merged header also referenced an unstaged
`sodaspaces.css`: its existing source was copied into the ignored local public
asset directory to resolve the capture 404. No service restart, account/resource
mutation or appliance rollout occurred.

## Merge of Forgejo presentation and Sodaspaces work

Merged remote `f838b80` with local `a652450`, preserving both histories, native
presentation/notification/avatar changes and the local environment/terminal/security
work. Shared header/footer hooks retain both features. Avatar requests remain public
and credential-stripped; other `/-/soda/*` traffic retains same-origin API/OAuth
credentials. No separate Soda browser origin or old schema/deployment state was restored.
Both real upstream dependency/checksum sets and packaged license notices are retained.

At this merge, the hooks referenced shared presentation templates/assets beyond the
then-current bounded staging allowlist. Staging refused that incomplete payload.
The source inventory above supersedes that hold; native delivery validation is still needed.
This merge does not deploy the preview, mount the new management module or change any
retained fixture/service/account/key. Imported preview evidence refers to its original
workspace and is not newly executed evidence here.

Checks: full Go suite, 121 local Node tests and Python build fixtures (58 passed,
one opt-in Caddy check skipped) passed. Focused Go races and documentation checks
passed; the resolution diff against `origin/main` is whitespace-clean. Existing remote
whitespace in native-parity templates/notices/assets is retained, not rewritten as
part of conflict resolution. Logs are under `.artifacts/merge-f838b80/`. An initial combined Forgejo
browser-test invocation failed at import because root Playwright is unavailable;
those browser-dependent tests remain unverified, not counted as passing. Original
merge/test failures are preserved. No dependency install, native stage/build,
browser fixture, deployment or provider action was performed.

## Selected tonal buttons — uniform 44px sizing

Local presentation `2026-09-08.17` implements the user's selected C — Tonal,
44px design, superseding revision .16's mixed sizes. Primary actions use tinted
surfaces and blue text; neutral actions are open. Shared buttons use 6px corners,
600-weight 14px labels and 14px horizontal padding. Native mini/tiny/small/compact
classes, icon actions, count labels and adjoining single-value controls all align
at 44px. Competing form, repository, home and administration button declarations
were removed. Native semantic colors, loading/disabled behavior and joined edges
remain. Single-value principal-form fields align with actions; multiline and
multiple-selection controls retain growing areas. The narrow header now fits
44px targets at 320px without shrinking the canonical logo.

Go Forgejo source checks passed. The full browser suite passed 39 checks with two
opt-in native session tests skipped; the dedicated native settings suite passed
11, including menu navigation, avatar dialog/focus and no-JavaScript behavior.
The final focused component suite passed 12 checks, including the additional
native-size and adjoining-input assertion. A read-only native audit recorded
1,298 control measurements across 13 routes, light/dark and 1440/900/390/320px
(104 page states), with no size failures, horizontal overflow or browser errors.

Final verified real-route captures are under `.artifacts/tonal44/release-desktop/`
and `release-mobile/`: Appearance, repository code, issue creation, Keys and
Explore. Representative final desktop/mobile captures were visually reviewed.
The production-derived gallery remains `.artifacts/forgejo-presentation/`;
`.artifacts/button-options/` is the separate design-choice comparison, not native
route evidence. Check logs and dimensional evidence are in `.artifacts/tonal44/`.
Admin/provider/conditional and populated credential states, long translations
and submission journeys remain unverified natively. No account data or preferences
were submitted. Activation used local template reloads only; no container restart
or appliance rollout occurred.

## Pronouns removed from Soda presentation

Personal profile editing, the pronoun privacy checkbox, administrator user editing
and public profile display no longer expose pronouns. The public-profile partial
is an attributed exact-native override with only its pronoun suffix removed.
The gallery and coverage inventory (229 overrides/helpers) reflect this change.
Hidden personal/admin form fields retain existing native values so unrelated
saves do not implicitly clear data. Forgejo's database, API and locale catalogs
remain unchanged; this is presentation removal, not an upstream feature fork.

Local revision `2026-09-08.15` is active via template reload. Go Forgejo checks,
five source/inventory checks and eleven settings browser checks passed. Verified
dark desktop settings/public-profile and light mobile settings captures under
`.artifacts/profile-without-pronouns/` were visually reviewed. Native profile
mutations and administrator routes were not exercised; administrator parity and
hidden-value contracts were checked in source. No saved data was changed.

## Settings menu link activation fix

A native pointer-click reproduction showed focusout closing the settings menu
before its destination link received focus, cancelling navigation. The handler
now checks `relatedTarget` instead of the transient `document.activeElement`.
Local presentation `2026-09-08.14` is active via template reload. Eleven settings
browser checks passed, now including twelve real link navigations across all
three menu groups at desktop/mobile widths with HTTP status, URL and page-heading
assertions. The Go Forgejo suite and four source/inventory checks also passed.
No account data was submitted or changed.

## Clickable avatar and modal dialog

The actual profile portrait now opens the native avatar form in a browser modal
dialog. A translucent dark pencil overlay appears on hover/focus and remains
visible on coarse pointers. The same form is moved, never copied; native upload,
source selection and deletion hooks remain intact. Escape, explicit close and
backdrop clicks restore focus; Tab wraps within the dialog. Without JavaScript,
unsupported dialogs or with server errors, the expanded inline editor remains
available alongside authoritative alerts. The local preview is active at
`2026-09-08.13` via template reload only.

The Go Forgejo suite, four source/inventory checks and eleven settings browser
checks passed, covering seven widths, keyboard/overlay/modal behavior and the
actual native no-JavaScript fallback. Verified dark 1440px and light 320px captures
in `.artifacts/avatar-modal/release-{desktop,mobile}/` were visually reviewed.
Earlier captures in that directory tree expose a corrected native dialog-style
conflict and are not final evidence. No upload/delete POST was submitted; native
server-error and lookup-enabled submission journeys remain unverified.

## Avatar source simplification

The avatar editor now starts directly with file selection when Gravatar is disabled;
it submits the native `source=local` field without showing a lone radio. When
lookup is available, both source radios and the saved selection remain native.
Active local presentation is `2026-09-08.12`, via template reload only. The Go
Forgejo suite passed, including rendered checks for both capability states and
both saved source selections; four source/inventory checks passed. Verified dark
1440px and light 320px captures under `.artifacts/avatar-source/` were visually
reviewed. No account settings, upload or deletion were submitted; lookup-enabled
runtime coverage remains unavailable on this fixture.

## Avatar editor refinement

The profile avatar disclosure now aligns its source and upload fields without
native indentation, uses a single native file-selector boundary, compact help
text and a borderless explicit delete action. Upload constraints, source choices,
form handlers and delete hooks are unchanged. This is active only in the existing
local preview at presentation `2026-09-08.11`; templates were reloaded without a
container restart. The Go Forgejo checks, four source/inventory checks and ten
settings browser checks passed. Verified dark desktop and light 320px captures
were visually reviewed under `.artifacts/avatar-refinement/final-{desktop,mobile}/`;
an earlier 390px capture was also reviewed. No upload or deletion was submitted.
The earlier comprehensive review package below predates this focused refinement.

## Personal settings structural overhaul — local review candidate

The approved personal-settings overhaul is implemented through Forgejo 15.0.7
configuration, template overrides, shared styles and a small presentation-only
script. The existing local preview is active at presentation `2026-09-08.10`.
There is no personal-settings sidebar or artwork hero: one actual identity row
and grouped destination menus compose around distinct profile, preference,
security, inventory and focused-editor layouts. Mobile uses one disclosure below
900px. Native routes, permissions, handlers and save boundaries remain upstream.

Profile retains one identity/address/privacy save and a separate avatar form.
Account places email management before its disclosed password editor and final
deletion warning. Appearance retains four saves. Security retains native factor
state and links to Account for passwords. Keys, Applications, resources and
conditional operational/child pages retain their native structures and gates.
Shared OAuth, runner, webhook and cleanup adapters use explicit personal-caller
inputs and preserve native root context; the organization Applications flag is
not used as a personal-only presentation gate. The test-only inventory covers
228 overrides/helpers, including 37 personal-settings files, their native callers,
compositions and required states.

The complete English locale was generated from the exact embedded native 15.0.7
INI plus Soda additions. Duplicate keys/namespaces are rejected; native INI bytes
and JSON catalogs retain upstream ownership. The complete file was copied into
the existing preview volume and activated with the single user-authorized restart
of `sodaos-local-forgejo`, retaining its image, configuration and data. Subsequent
template changes used native reloads. No appliance deployment occurred.

Actual checks and review evidence:

- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off SODA_FORGEJO_GALLERY=1 go test
  -mod=readonly ./scripts -run TestForgejo -count=1` passed. Tests retain native
  control/capability contracts, mandatory enrollment gates, caller-specific title
  handling, selected cleanup values and native OAuth/runner root context.
- The complete existing Forgejo browser suite passed 37 checks, with its separately
  invoked personal-session journey skipped in that batch. The dedicated settings
  run passed all 10 checks, including navigation at 1440/1000/900/899/768/390/320px,
  Escape/outside dismissal, focus return, password and avatar disclosures, key
  panel focus, token-select dimensions, no-JavaScript fallback and fixture errors.
- Read-only native inspection covered 14 routes at six widths: 84 combinations
  without document overflow or page errors. Forgejo 15.0.7 uses Go's native
  `http.NewCrossOriginProtection` in `routers/web/web.go`; an initial audit's
  hidden-CSRF-input assumption was corrected against exact source. That middleware
  was not changed. No POST security/submission journey is claimed.
- The review package contains 94 verified native captures in light/dark at
  1440×1000 and 390×844, including full-page content and open editors. Requested
  URLs, status, landmarks, presentation revision, stylesheet hashes and browser
  errors were checked by `scripts/screenshot.mjs`. The helper now supports
  `--full-page` and forces a fresh document for fragment-only capture requests.
  Failed intermediate capture attempts are excluded from the review manifest.
- `.artifacts/personal-settings/review.html` groups native evidence by family and
  links the actual check logs, route observations and verification sidecars.
  `.artifacts/forgejo-presentation/gallery-{light,dark}.html` is the separate
  production-derived component gallery. Earlier generated concepts remain in
  `.artifacts/settings-design-concepts/`; they are not native evidence.

Native landing pages and accessible child editors were visually reviewed, including
below-the-fold controls. This is not full visual or functional acceptance.
Remaining prerequisites include enrolled/mandatory TOTP and WebAuthn, recovery and
key verification challenges, populated credentials/OAuth grants/applications,
providers, organization memberships/adoption permissions, configured webhooks and
delivery history, populated cleanup previews, Actions/runners/secrets/variables,
and enabled Storage/Quota. Successful/error POST journeys and native fallback for
untranslated Soda additions were not manufactured. Nonpersonal shared callers
have source and regression checks, not newly authorized owner/admin sessions.
No credentials, saved preferences, resources or permissions were created or changed.

The earlier [UX investigation](forgejo-settings-ux-proposal.md) and its evidence in
`.artifacts/settings-ux-investigation/` remain the baseline for comparison, rather
than the current implementation contract. See `appliance/forgejo/README.md` and its
locale guide for the maintained presentation and local activation contracts.

## Ordinary repository container spacing

Local revision `2026-09-08.5` removes the extra 24px top padding from project
lists and wiki revisions. The ordinary repository body container now explicitly
owns zero padding; the shared repository header supplies the navigation gap.
Native fluid/explicitly padded canvases keep their separate gutter contract.
The earlier redesign had consolidated width without removing these family-local
insets; its coverage did not establish equivalent container spacing.

A read-only native regression compares Code, Projects, Issues and Releases at
1440px and 390px, asserting equal padding, width, left alignment and gap below
repository navigation. It passed, as did focused Forgejo Go checks. Wiki revision
padding removal is source-reviewed; no wiki fixture was created for verification.
Templates/assets refreshed only in the existing local preview.


## Repository file toolbar correction

Local revision `2026-09-08.4` aligns the branch picker, compare/find/add controls,
clone protocol buttons, URL field, copy and menu controls to the same 44px row.
The shared toolbar stylesheet owns their geometry and joined edges; the previous
40px clone-input minimum is removed. Dropdown contents remain native. A bounded
flex override lets the URL shrink instead of overflowing narrow viewports.

Focused Forgejo Go checks, both inventory checks and all 10 component-boundary
checks passed. The new toolbar case includes the owner-only Add file button,
light/dark and 1440/390/320px layouts. Native non-owner desktop/mobile captures of
`/alice/activity-workbench` passed route/revision/style verification and were
visually inspected: `.artifacts/screenshots/capture-vUErxz/` and `capture-MkUotY/`.
Owner-only Add file is fixture-tested, not claimed as authenticated owner evidence.
The earlier complete review package remains a record of revision `.3`; this is a
scoped local correction, with no form submissions or appliance deployment.


## Presentation redesign local candidate

Candidate `2026-09-08.3` is active only in the existing local Forgejo preview.
The inventory accounts for all 206 overrides/helpers and their local/embedded
15.0.7 callers across eight compositions. 125 templates select new explicit
presentation roles; the remaining helpers/native structures use existing shared
components or specialized family composition. Shared controls, typography,
spacing, open sections and editor containers replace duplicated page rules.
Native routes, inputs, permissions, CSRF and script hooks remain upstream-owned.

Actual checks: focused `go test ./scripts -run TestForgejo` passed using the local
toolchain and offline dependency settings; all 30 opt-in Node checks passed,
including component states, native Explore overflow/navigation, notification
lifecycle, milestone layout, gallery responsiveness and both sides of inventory
coverage. The gallery uses production intro/empty partials and the real registry,
with minimal native markup fixtures. It passed light/dark at 1440, 900, 390 and
320px. Native template source was read through the running container's embedded
resource export; no Forgejo source fork or rebuild was introduced.

The local review package is `.artifacts/forgejo-presentation/review.html`, with
light/dark galleries, a manifest and 36 verified native viewport captures across
nine accessible routes at 1440×1000 and 390×844. Each final PNG has a sidecar with
URL/status, landmark, revision, registry and stylesheet hashes, viewport/theme
and browser errors. The signed-out account-settings redirect was rejected with
no accepted image. Visual inspection caught and fixed primary-anchor text losing
contrast; a focused regression check now covers it. Earlier `review-*` and
`login-*` capture folders are superseded by the `final-*` candidate images.

This is **not full visual acceptance**. Owner-only repository editors/settings,
administrator pages, organization fixtures, setup/provider/authentication states,
populated packages and specialized canvas/permission/interaction states remain
unverified on native pages. No resources or permissions were created to fill
those gaps, no forms were submitted, no saved preferences were changed, and no
appliance rollout occurred. See `docs/forgejo-presentation-review.md` for the
review entry points and precise evidence limits.

## Shared repository form presentation

Issue/PR composers and milestone, project, release and wiki forms now opt into
the existing shared form controls. Repository creation/editor pages share their
container width, open heading treatment, explanatory copy and divider spacing
in `components-forms.css`. Removed competing wiki/release/project layout rules
and file-editor header/commit-choice cards rather than layering another theme.
Native templates, form actions, field names, editor internals and gates are unchanged.

Focused Forgejo checks passed after updating the stylesheet-ownership assertion
for release forms; the initial stale assertion failure was resolved by moving
ownership, not changing the native-body checks. New-issue desktop capture was
inspected (`.artifacts/screenshots/capture-cfXgkj/`); browser measurements verified
1440px/390px widths without horizontal overflow and one issue form. The separate
manual screenshot profile was signed out: milestone/release captures showed login
and wiki/project showed 404 (`capture-xXavbV`), not successful form evidence.
Remaining permission-restricted forms and dark rendering need native visual review.
Local templates reloaded for CSS versions only; no submissions or deployment.

## Reduce decorative cards and dividers

Shared toolbar, list, empty-state and form-section frames are removed. Native
settings navigation and attached form sections use open backgrounds; profile
privacy controls no longer have an extra enclosing box. Milestone cards and
descriptions, repository-sidebar rules and the landing README frame are removed.
Blank settings/milestone dividers retain a 24px margin; other removed section
borders retain their existing padding and gaps. Control, table-row, alert and
dialog boundaries remain. Changes are scoped to existing presentation owners.

Focused Forgejo checks and whitespace checks passed. Local templates were reloaded
for stylesheet versions. Desktop captures of repository/profile/explore and mobile
profile/explore/milestone detail were inspected (`capture-wlMSyp`, `capture-BGWs4M`
under `.artifacts/screenshots/`); inspection prompted removal of a remaining native
README segment frame and profile legend rule. Restricted admin/organization forms
and dark variants were not newly visually checked. No backend or appliance changes.

## Repository metadata sidebar

The repository code landing override now places existing description, website,
topics and their native editor, counts/size and conditional language statistics
in a right sidebar. Metadata is rendered once with its existing permission gates;
file, directory and blame views keep their previous layout. Below 1000px the
metadata stacks above the code. The main toolbar no longer has an enclosing card.
No release/contributor data fetch or backend change was added.

All focused Forgejo source checks passed, including upstream body recovery after
removing the explicit layout changes. Reloaded templates only in the existing
local preview. Inspected desktop/mobile captures (`capture-br6UQ5`,
`capture-mLOcnf` under `.artifacts/screenshots/`); browser measurements confirmed
1440px and 390px document widths, one sidebar/topics/summary instance, and no
sidebar on the README file view. The screenshot fixture lacks topic-admin rights;
interactive topic editing and populated language statistics were not exercised.
No appliance deployment or native acceptance.

## Soda robot avatars — source implementation

Original `soda-robot-v1` artwork now has 44 modular SVG variants and an eight-by-four
background/accent palette in one embedded DiceBear JSON definition. The pinned Go
renderer serves public GET/HEAD `/-/soda/avatars/v1/{hash}` with bounded inputs,
deterministic ETags and no session, identity/database lookup or outbound fetch.
The development-only catalog uses the same renderer and is not appliance payload.

Caddy source routes only `/-/soda/avatars/*` on the Forgejo origin to the existing
backend, dropping Cookie/Authorization for those requests. First activation derives
the supported provider URL from `forgejo_url`; native database-backed avatar
settings and explicit offline mode remain operator-owned. Uploaded photos/files
are preserved. This does not implement the broader Sodaspaces origin/session work.
New bundles include and require the exact avatar dependency notices. See
[behavior, configuration and restoration](avatars.md).

Local checks on 2026-09-08: full `go test -mod=readonly ./...` passed with Go 1.26.7
on macOS arm64; avatar/web race suites passed. The backend binary built locally
with the same toolchain and `go mod verify` passed. All 34 Python build fixtures passed,
including real Caddy 2.10.2 routing against test-owned loopback upstreams, mocked
first activation and actual notice collection. Caddy was downloaded into ignored
tooling and verified against its published SHA-512 checksum, not installed.
Some unchanged Go results were cached. Initial broader runs failed on macOS's
symlinked temp path and BSD `cp`; using a real workspace TMPDIR and the already
installed GNU coreutils resolved them without changing runner/provisioning code.
The first Caddy checksum comparison mistakenly used SHA-256; the correct SHA-512
comparison passed before the binary was executed. Earlier failed records remain.

Inspected all parts, all 32 palette pairs and the generated 100-robot grids.
Chrome checked all 100 images at 24/32/64/128px across light/dark and square/circular
modes, with no missing images or external resource requests. Six production HTTP
images also matched reference pixels under the restrictive response CSP, with no
browser errors or external requests. The first pixel comparison used different
screen positions; the corrected check uses the same position to avoid SVG
antialiasing differences. Final preview and
captures: `.artifacts/avatars/preview-606454540/`; check logs:
`.artifacts/avatars/checks/`. These are artwork/local-test evidence, not Forgejo
screenshots or native installed acceptance.

No live Forgejo settings, avatar uploads/deletions, retained fixture data, services,
appliance routing or installed artifacts were changed by this avatar work. Native
profile/list/discussion and upload/delete smoke checks, local proxy/backend rehearsal
and appliance rollout remain pending their target-specific authorization. The
direct-port local Forgejo preview cannot exercise this route through a template
reload alone.

## Tighter template spacing

Reduced larger margins, padding and layout gaps across 42 Forgejo presentation
stylesheets. At that stage, shared panels used an 18px desktop inset (16px narrow),
list rows used 16px vertical padding, and page intros used a 192px minimum with
224px artwork. Work-item rows now use the shared contract recorded above.
Typography, control minimum heights and native workflow markup remain unchanged.
Changed stylesheet URLs are versioned in the header hook.

Focused Forgejo source checks passed with local Go and dependency resolution
disabled (`go test -mod=readonly ./scripts -run TestForgejo -count=1`); whitespace
checks passed. Inspected local candidate-CSS captures of repository exploration,
repository milestones and profile settings at 1440×1000 and 390×844. Captures:
`.artifacts/screenshots/capture-rkXrOj/` and `capture-LQW7zh/`; desktop baseline:
`capture-PA9rin/`. Settings mobile autofocus scrolls to the form. Repository
milestone cards already had flush content in the baseline; that native styling
is unchanged. Other page variants and dark mode were not newly visually checked.
No service reload, deployment or native acceptance was performed.

## Page illustration goal

The accumulated Forgejo-focused scripts suite passes (`go test ./scripts -run TestForgejo -count=1`) after correcting three stale parity normalizers for intentional artwork suppression. All 34 literal illustration references resolve to local assets. Native restricted-page rendering remains outstanding; neither check establishes that visual evidence.

Repository Actions no-workflows state now selects a distinct transparent workflow-tile illustration; native permission-specific guidance is preserved. Populated/filtered run lists, dispatch and log viewer remain undecorated. Focused body-parity/shared-presentation tests passed. The attempted native Actions capture returned 404; artwork rendering remains unverified.

Public registration now has a distinct transparent welcome-folder illustration, gated to enabled standalone registration. Exact prompt and original output are retained. Focused authentication/shared-presentation tests passed. Native guest desktop/mobile screenshots confirm disabled registration excludes the image; enabled registration rendering remains unverified because the local instance disables registration.

All current administrator layout callers are source-assessed. Admin artwork is now opt-in: account creation selects its distinct illustration; operational/configuration pages receive none. Redundant suppression flags were removed. Native admin verification remains pending.

Administrator account creation now selects a distinct transparent identity-card illustration through an explicit presentation input. The generated asset and exact prompt are retained; focused administrator parity/presentation tests passed. Native admin capture remains pending.

Wiki welcome now uses a distinct transparent reference-book illustration, preserving native text and the writer/mirror action gate. Repository content source-parity/gate tests passed; native read-only welcome was captured at desktop/mobile widths. Repository project wrappers and shared callers are assessed in the checklist.

Personal/organization project creation now uses a distinct planning-board illustration; editing excludes it. Personal creation was visually checked at desktop and mobile widths, and focused context/presentation tests passed. Organization and edit-state captures remain pending; checklist records list/board no-image decisions and the remaining repository callers.

Team creation now has a distinct transparent member-card illustration, gated to the creation state. Editing permissions and invitation acceptance retain focused native content. Native markup parity/parse and shared presentation tests passed; organization screenshots remain pending an accessible existing organization. The checklist advances to organization projects.

The [per-page checklist](page-illustration-checklist.md) inventories 206 current
overrides/helpers and tracks shared-template page variants separately. Migration,
fork, 404, discussion subscriptions and watched repositories now have distinct
illustrations. Native forms, state, permissions and meaningful status text remain
upstream-owned. Migration progress and 413 retain concise native diagnostics
without additional decorative art. Public contributor profile tabs were also
assessed without extra art: native identity, authored content and activity visuals
take precedence, supported by desktop/mobile captures. The personal package
registry now has a compact wrapping illustration, with focused package tests and
native desktop/mobile empty-state captures checked. Organization package registry
has a distinct shared-shelf illustration and passing focused package tests;
native verification is pending because the local public organization inventory
is empty. Package version lists and the common detail shell were assessed without
added art to prioritize release selection, installation content and metadata;
these are source decisions, not populated-page runtime evidence. Package settings and cleanup forms were assessed; six upstream settings callers
were added explicitly to the inventory, with personal landing/add-rule desktop
captures inspected. Personal registry settings now has its own maintenance illustration, scoped by an
explicit landing-template artwork input; desktop/mobile captures and add-cleanup
isolation were checked, and focused package/shared-boundary tests passed.
Organization registry settings was assessed without a second decorative header;
its native identity and direct cleanup/Cargo controls take precedence. Other
organization settings routes are now explicitly queued. Personal webhooks now have a connection illustration on the list page only;
desktop/mobile and new-form isolation captures were inspected, with focused
webhook/shared-boundary tests passing. Personal organization memberships now has a distinct card scene, with native
desktop/mobile empty-state captures checked and stock membership content verified
unchanged after the layout call. Personal repository settings was assessed without artwork to prioritize its
repository/directory inventory and permission-dependent confirmations; native
empty-state desktop capture was inspected. Personal and organization blocked-user pages were assessed without decorative
art; personal empty-state capture was inspected. Conditional personal Actions
and storage routes are explicitly queued. Existing profile artwork was retained and verified on desktop/mobile. The
screenshot helper now has an optional `--scroll-top` flag, exercised to inspect
headers after native form autofocus; default capture behavior is unchanged.
Account artwork was retained and verified on desktop/mobile, with native forms
left untouched. Appearance swatch artwork was also retained and verified on desktop/mobile,
without changing saved preferences. Security landing artwork was verified on desktop/mobile. Enrollment/re-enrollment
now explicitly suppress decorative art to prioritize the QR/passcode flow; native
enrollment capture remains unperformed. Focused shared-presentation tests passed.
Keys artwork was retained and verified on desktop/mobile; native SSH, GPG and
principal subpanels were source-reviewed without added decoration. Applications landing artwork was retained and verified on desktop/mobile;
OAuth editing and token creation now suppress inherited decoration; token
creation final desktop/mobile rendering was checked after restoring its existing presentation classes and focused shared-presentation tests
passed. Native OAuth edit verification remains pending. Shared OAuth list/create/grant
sections were traced to personal, organization and admin callers; they need no
independent artwork, while remaining page owners stay explicitly queued; individual decisions and unexercised variants remain in the checklist.

Each new PNG was visually inspected and verified as transparent RGBA. Focused
onboarding, status, presentation-boundary and notification-preview tests passed
for their respective changes. Local templates were reloaded and actual fixture
light-theme pages were inspected at 1440/390px; 404 covered general/repository
contexts, and subscriptions/watching covered shared-template isolation. A status
stylesheet version bump corrected observed cached sizing. Exact prompts, rejected
outputs, capture paths and per-page limits are in the checklist and linked records.
No fixture/account state or provider resources were changed. Dark appearance,
POST journeys and production deployment are not implied by these captures.

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus delivered **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | Explicit stable-ID create/key/join/own-connection controls passed bounded native x86_64 build/export/browser/Copy/SSH at `bdbce8e`. Copied private-v3 migration/paired rollback and separately approved retained cutover passed, with native browser and own-access observations. Both standalone frontends remain removed |
| Minimum management controls | Start/Stop, own saved-key removal and reviewed native key apply/revoke, plus independent complete drawer content, are source implemented. Native lifecycle/SSH rotation and integrated delivery still pending |
| Browser terminal | Native helper passed bounded x86_64 PTY/teardown/SSH-preservation proof. Protected browser transport and independent drawer terminal component/locked renderer packaging are source implemented and locally tested. Template mounting, genuine combined browser proof and deployment remain pending; installed helper unchanged |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v5 SQLite with unchanged grant encryption, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Built `bdbce8e` dashboard/strict-config runners CLI, native hooks and namespaced proxy/config on `soda-test`; schema v5 and stock Forgejo 15.0.7. Unchanged helper/default project image/other native components retain prior `8b823db` provenance; old frontends are no longer served |
| Acceptance | Historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`); Sodaspaces steps 5–6 have passed bounded execution and approved cutover. U01 architecture acceptance was withdrawn; no final-product/aarch64 acceptance |

Source removal is **not retained-appliance deployment**. `ee8091a` passed bounded
read-only delivery/browser proof; `bdbce8e` subsequently passed native create/key/join/
Copy/SSH on the separate fresh fixture. Separately approved `soda-test` cutover then
delivered those affected payloads with recorded configuration and schema v5. Root returns to configured Forgejo
home; OAuth can return to a freshly resolved repository under that origin using
single-use stored context and the acting grant, never a caller-supplied URL.
Schema v5 adds internal login cancellation contexts after v4's repository/expected-
user IDs. The historical return-path column remains unused. Both copied and live
v3 → v5 migration preserved original rows/ciphertext before login. Old pending OAuth
must restart; subsequent normal expiry/login/logout changes session/grant rows.
New consent requests read user/repository/organization scopes, not administrator
expansion; actual grants, not requested scope names, govern authority. Redirecting is not native-session
transfer or cross-origin authorization.

Keep [API](dashboard-api.md), [credential migration](dashboard-credentials.md),
[architecture](architecture.md) and [current work](sodaspaces-plan.md) authoritative.
Acting grants/current native ownership from `fed66cb` remain in retained callers;
no setup-token, stale-creator or copied-permission fallback was restored.

## Expanded component audit merged into canonical main

Merged exact branch candidate `919bb97cdc4d3c4db84bda6cfd57e02c870dfd3b`
(`codex/forgejo-expanded-component-audit`) with canonical `85f29a5` using a
non-fast-forward merge. The sole textual conflict was this handoff: both audit and
newer notification implementation/activation evidence were retained. The header
merged automatically and retains the notification stylesheet alongside the new
feature owners. Notification source/tests, Explore overflow and milestone grid
fixes remain byte-for-byte unchanged from pre-merge main. Both parent histories
are preserved; no rebase, cherry-pick or history rewrite.

Merged-tree checks on this development machine:
- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -json -count=1 -mod=readonly ./scripts -run TestForgejo`:
  **97 top-level tests passed**, no failures/skips, plus their subtests. There are
  98 matching source declarations; `TestForgejoBrandingMatchesSVGMaster` requires
  the separate `branding` build tag/native renderer and was not selected. The
  audit-only tree has 96 declarations, so its earlier reported count is not used
  as the merged runtime count. The two newer notification tests are included.
- `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/*.test.mjs`:
  **19 reported tests passed**, no failures/skips, including the eight boundary
  subtests, guest theme, Explore, milestone and newer notification fixture suite.
- `node --check scripts/screenshot.mjs`, CSS registry correspondence (46 files,
  each registered exactly once), and staged/working-tree whitespace checks passed.
  Logs retained under `.artifacts/component-merge-checks/`.

No template reload, authenticated journey, new capture, fixture/provider mutation,
dependency installation, service restart or deployment was performed for this merge.
The preview's cached templates still predate the audit: the former `admin-org.css`
and `workflow-details.css` references may outlive their removed source files until
an authorized reload. Shared source-mounted CSS changes are not a complete native
activation. Boundary/milestone fixtures load the merged registry; native-page tests
still use the existing server templates, and notification tests simulate signed-in
markup/responses. The audit's new native form/profile markers and other server
changes remain source-tested, not newly rendered. Earlier notification activation
is evidence for its pre-merge candidate only. Administrator/owner-only workflows,
populated boards, package cleanup, provider/POST journeys and production staging
retain the audit's documented limits.

## Notification bell quick-view local activation

User authorized activation and testing after `20ad60f`. Reloaded templates only in
`sodaos-local-forgejo` (`forgejo manager reload-templates` returned `Reloaded`).
Real signed-in requests now exercise the native `ctx.Context.FormBool` rendering
branch successfully; no product-code correction was needed for activation.

- Populated native inbox: three real unread entries rendered at 1440, 768, 390 and
  320px. Both bell anchors, panel containment and Escape/focus return passed. The
  native unread count remained **3 before / 3 after**. No entries were opened.
- On `/notifications`, the popup coexisted with exactly one native notification
  div/table. An ordinary `div-only` refresh still returned the full native fragment;
  “View all notifications” navigated to the normal page. Recorded notification
  requests were GET-only and no page errors occurred.
- Existing non-admin screenshot fixture profile exercised the real empty inbox in
  dark mode; zero rows, truthful empty copy, dismissal and **0 before / 0 after**
  unread count passed. Populated light/mobile and empty dark screenshots were read.
- The 320px signed-in navbar already extends 3px beyond the viewport before the
  popup opens (`navbar-left/right` and appearance link); the popup fits and does not
  increase document width. This unrelated navbar issue remains, not a popup pass
  disguised as a whole-page no-overflow claim.
- Focused Go Forgejo tests and all three browser suites (notification fixture,
  Explore overflow, milestone layout) passed again after reload. `git diff --check`
  passed. Evidence/scripts/screenshots are retained privately under
  `.artifacts/local-forgejo/notification-activation-20260908/`.

An isolated browser used the retained local fixture credential privately for normal
login; the existing screenshot profile was reused for empty-state checks. Exploratory
checks initially used an explicit submit-type selector absent from the native login
button and assumed five entries where the real account has three; test assumptions
were corrected, not fixture data. No new fixture, notification-status mutation,
account preference change, service restart or appliance deployment occurred. Actual
pinned rows, live badge changes from new events, account switching/session expiry
and native read-on-navigation remain unexercised; those existing synthetic/source
checks are not relabeled native evidence. Production staging/delivery remains pending.

## Notification bell quick-view source implementation (pre-activation evidence)

The requested [plan](notification-preview-plan.md) is implemented in source using
the existing native notification fragment and bundled HTMX. A signed-in-only footer
hook enhances both stock bell anchors; a presentation query flag selects a compact
list of the five native unread-plus-pinned entries. Dedicated markup avoids full-page
notification IDs/hooks. Native destinations, auth, queries, badge updates and read
behavior remain Forgejo-owned. There is no Soda backend, upstream patch, new library,
status POST or second poller. Missing JavaScript/HTMX/popover support retains native
bell navigation. The panel handles loading/errors/retry, cancellation, response
identity, focus/Escape/outside dismissal and native HTMX redirects. A 10-second
request timeout bounds the loading state. Retry/View all/Pinned/loading/empty/error
copy is custom English pending localization; existing notification/close keys are reused.

Performed locally:
- Offline readonly focused Go `./scripts -run TestForgejo`: passed, including compact
  and ordinary fragment branches, faithful embedded template-context method lookup,
  zero/one/five rows, pinned/content escaping, native subpath links and signed-in hooks.
- `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/notification-preview.test.mjs`:
  passed against the real native HTMX bundle with browser-only signed-in markup and
  notification-response fixtures. Both bells, long five-row content, 320/390/768/1440px,
  dark/light scheme requests, loading/empty/errors/retry, stale responses, Enter/Escape,
  modified clicks, focus/outside dismissal, GET-only preview traffic, HX-Redirect and
  no-JS/no-HTMX/no-popover native fallback were checked. No authenticated fixture data
  or existing private credentials were accessed. Initial tests exposed a null detail
  on custom abort events; events now supply the native expected element detail.
- Existing Explore overflow, milestone layout and seven guest-theme tests passed.
  `git diff --check` passed. No dependency installation or full native build occurred.

Native authenticated rendering, actual unread/pinned data, badge/full-page coexistence,
real account switching and user visual review remain unverified. No template reload,
service restart, notification mutation or appliance deployment occurred. Existing local
mounts expose asset source, but changed/new templates require an explicitly authorized
reload to activate. Production template/asset staging remains separately pending.

## Notification bell quick-view investigation (historical, before implementation)

The requested [implementation plan](notification-preview-plan.md) is now authored:
prove compact native rendering, enhance both bells using bundled HTMX, validate
interaction/native coexistence, then deliver under separate target/action scope.
Unread plus pinned is the proposed native-matching default; no UI code or runtime
work was performed while writing the plan. Documentation whitespace checks passed.

The [integration guide](forgejo-frontend-integration.md#notification-bell-quick-view-investigation)
records source findings for stock 15.0.7 (`d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`).
JSON notification APIs do not accept the ordinary web session, but the existing
`/notifications?div-only=true` HTML-fragment route does; Forgejo already ships HTMX.
The recommended candidate is a supported compact template branch loaded with native
HTMX, not a Soda Go/API proxy. Native unread results include pinned entries, so
strictly-unread-only semantics need a decision rather than silently filtering a page.
Anonymous local GET checks confirmed API 401, native fragment login redirect and
HTMX-aware 204/HX-Redirect. No existing credentials, authenticated requests, state
mutations, UI implementation, builds/tests, service reloads or deployment occurred.
`git diff --check` passed. Authenticated rendering and candidate interaction checks
remain unperformed; investigation is not implementation acceptance.

## Expanded component audit

Audited the `82379b4` expansion across all 201 template overrides/helpers and
38 linked CSS files. The [audit](forgejo-components-audit.md) and
[composition contract](../appliance/forgejo/README.md#presentation-component-contract)
record the resulting boundaries. Shared controls now preserve native focus/error
states; button-local theme variables give native primary actions one color owner.
Settings table padding is separate from card padding. Ordinary repository/org
width rules exclude fluid canvases; the bounded pull-files canvas is centered.
Repository-context status pages constrain their grid instead of expanding native
navigation beyond mobile width.

Repository, administrator, organization, projects, packages/code search, shared
runner/configuration/quota/webhook/moderation adapters now have explicit owners;
competing old rules and the mixed admin/workflow aggregators are removed. Native
profile-card callers opt in through a component marker. Two nested principal
settings forms opt in explicitly, preserving compact row/dialog/search forms.
The migrating wrapper derives guest state from native `.IsSigned`. The unused
`finalize_openid` override is removed. There are still 201 template files (one
removed, one native OAuth-list override added), four Soda partials and 45 CSS
files; all CSS files are registered exactly once. No Lit dependency was added.

Executed locally for this audit:

- Offline readonly `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1
  -mod=readonly ./scripts -run TestForgejo` passed, covering 96 top-level tests
  plus their native composition/hash/security cases. Earlier runs exposed stale
  CSS-owner/cache-version assertions; those were corrected to the new contracts.
- The combined guest-theme, Explore overflow, milestone layout and component
  boundary JavaScript run passed all 18 reported tests. The new boundary suite
  covers eight cases, including light/dark focus/errors and native button states,
  table padding, fluid widths, narrow error pages, compact-form isolation, profile
  reuse and horizontally reachable package table columns. The error-border,
  table-padding, fluid-width and status-grid tests reproduced pre-fix failures.
  The boundary suite passed again after centering the bounded pull-files canvas.
- Read-only stock 15.0.7 embedded/source inspection retained native hooks and
  confirmed the unreachable OpenID template and anonymous migration route. Exact
  upstream commit: `d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`.
- Captured and inspected 40 candidate CSS screenshots using
  `scripts/screenshot.mjs --local-css`, at 390px and 1654px, plus ten initial
  mobile baseline captures. Candidate captures cover account settings, public
  repository/list/release/wiki/project/activity/profile/package views, migration
  selection, pull files, branches/commits, error pages and guest sign-in/disabled
  registration/recovery notices. Evidence is retained under ignored
  `.artifacts/screenshots/audit-candidate-*`, `audit-final-*` and
  `audit-pull-files-centered/`. The latter recapture verifies the final wide canvas.
- Screenshot helper syntax, CSS registry/file correspondence and
  `git diff --check` passed. Existing development dependencies were reused.

The preview binds `/Users/vince/Projects/sodaos`, not this audit worktree. The
capture option substitutes only candidate Soda CSS in the isolated browser;
native server templates/scripts remain unchanged. New form/profile class markers,
the migration guest flag and removed unused override have source/caller-test
evidence only and await applying/reloading the templates. Missing quota/Actions
routes produced native errors, and global code search redirected to Explore;
those captures are not evidence of those workflows. The non-admin fixture cannot
exercise administrator/owner-only settings. Populated project boards, package
cleanup, provider authentication, native POST/error responses and appliance
staging/deployment were not newly exercised. No fixtures, account preferences,
credentials, service lifecycle or provider resources were changed.

## Expanded native Forgejo branding

Broad supported stock 15.0.7 header/layout overrides now cover repository pages
and settings, account settings, administrator and organization views, and nine
secondary authentication wrappers. Three coordinated task teams added 183 template
override files from the `a81bfae` baseline, including 141 during the final authoring
sprint; the source now contains 201 overrides/helpers. The added detailed families
include code/edit/diff/history, issues/pulls/milestones, releases/wiki/projects,
Actions/runners/webhooks, storage and access lists, profiles/packages/imports,
administrator monitoring, account/security, organization/team, federated auth,
and setup pages. These are template-file counts, including shared partials, not
independently exercised workflows. The shared native leaf forms/lists/scripts,
permissions and handlers remain upstream-owned. The page-marker adapter reaches
whole native pages from their shared header/helper; it is not a widget root.
Common settings sidebars/cards have one CSS owner. Principal native forms use a
positive structural adapter; nested dialog/table/row-action and settings search
forms keep native sizing. Guest theme routes share one presentation gate.
Six new settings illustrations from the separate art task are mapped by native
page flags, preserving its asset/provenance commits and earlier artwork.

Local source tests passed with the existing offline Go toolchain and readonly
modules (`go test -count=1 -mod=readonly ./scripts -run TestForgejo`). These check
native template composition, permission seams, exact-stock recovery hashes,
escaping, theme placement, artwork selection and CSS boundaries. Stock preview
`reload-templates` succeeded. Initial browser checks covered all 11 account
sidebar pages at desktop/390px, all 16 administrator sidebar pages at 390px,
and 12 repository sections at desktop/390px. One narrow native stacktrace overflow
and clipped-popup risks were found and corrected. After combining all three teams,
the full focused Go suite and all seven guest-theme JavaScript tests passed.
Final browser checks exercised 27 distinct pages at 390px and 1654px, plus all ten
native migration-provider forms at 390px, with no page-level horizontal overflow
after correcting the direct system-notices table. The native file editor mounted
CodeMirror and retained its commit form; a desktop release page was visually
checked. Browser viewport overrides were reset. The owner code-search URL
redirected to the profile under the existing configuration, so that route remains
source-tested only. Registration/recovery remain truthfully disabled. Setup,
MFA, activation, consent, POSTs, populated packages/organization teams/project
boards, Actions dispatch and fullscreen logs were not executed or fabricated.
Guest light/dark navigation was checked earlier and returned to light; no native
account preference, fixture, provider, native stage or deployed VM was changed.
All task commits were collected onto `main`; production staging and the Sodaspaces
drawer remain separate unfinished integration work.

`scripts/screenshot.mjs` is a small local capture helper: manual `--login` in a
dedicated private profile, then viewport PNGs for supplied URLs. Fixture changes
stay manual. It uses installed Chrome and existing Playwright; this development
checkout links the preinstalled desktop package through ignored `node_modules/`.
Verified two real guest preview captures at 390×844, interactive login-window
open/close, and persistent test-cookie reuse across browser runs with a local
temporary HTTP server (1440×1000 output). No real Forgejo login was submitted and
no Forgejo fixtures, native installation or deployment were changed. Usage is in
[screenshot capture](screenshot-capture.md#quick-local-page-screenshots).

## Explore tab overflow correction

The Explore tab wrapper now has a bounded 420px width (capped by the existing
100% maximum), with the native overflow-menu filling it rather than sizing to
visible children. Tabs align to the start so Forgejo's trailing overflow-button
reservation is not consumed by centering. This removes the ResizeObserver feedback
loop that repeatedly moved Organizations into/out of the popup. Native navigation,
visibility, overflow logic and keyboard handling remain unchanged; no custom JS.
The shared toolbar stylesheet cache version is bumped.

Read-only local Chrome checks on all three real anonymous Explore routes passed
at 1440, 768, 700, 390 and 320px, then back at 1440px. The new opt-in
`tests/forgejo/explore-overflow.test.mjs` waits for native initialization/fonts and
checks no child reparenting over 700ms after settling, no page overflow, all three
wide-screen links, narrow-screen popup visibility/destination and Escape dismissal.
With pre-fix CSS intercepted in the isolated browser, the same observation found
14 child mutations in 700ms; fixed pages had zero. An initial test attempted Escape
before the popup's deferred focus; focusing the menu item first corrected that test
race. Run with `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/explore-overflow.test.mjs`.
The milestone layout regression and offline readonly focused Go Forgejo suite also
passed; `git diff --check` passed. No login, data writes, service/template reload,
new dependency or deployment occurred. Local live-mounted CSS can be hard-refreshed;
the header cache-version change awaits the next template reload. Authenticated-only
extra tabs and translated labels were not newly exercised.

## Shared Forgejo presentation components

The local stock 15.0.7 preview uses small Go template partials for page intros,
empty content and the guest theme button. The component audit separates page
shell/tokens, intro, toolbar/native controls, forms, list/pagination, empty feedback
and guest theme/shell ownership. Page files retain only their specific metadata
and layout. Explore now delegates navigation, visibility and overflow to native
`explore/navbar`; context-switcher CSS has an explicit wrapper. All list callers
use the same wrapper contract. Search selectors cannot reach nested dialog buttons;
filled actions use the selected theme's action colors. Login no longer decorates
Forgejo's loader pseudo-element. Guest theme listeners load only on anonymous
routes with a guest toggle.

[Composition contract](../appliance/forgejo/README.md#presentation-component-contract)
and [full audit](forgejo-components-audit.md). `.soda-page` is a full-page shell,
not a root for the future drawer. Home/login keep distinct content layouts; the
native dashboard Vue widget, milestone cards and notification row actions retain
bounded page adapters. The original extraction is recorded in `ce4129c`.

Native forms, permission gates, translations, asset prefixes and notification
replacement hooks remain upstream-owned. Exact embedded 15.0.7 source comparison
found no unexplained behavior divergence in issues, milestones, notifications,
subscriptions or organization creation. The documented historical research mirror
is absent on this machine; the audit used `forgejo embedded view` from the existing
stock preview. No Lit dependency or frontend build was added. The authenticated
drawer and production staging remain pending.

Executed for this audit on the development machine:

- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -mod=readonly ./scripts -run 'TestForgejo'`
  passed with Go 1.27.1 darwin/arm64. Tests render real Soda partials/callers with
  native seams stubbed, including creation permission branches, guest route gates,
  singleton placement, escaping, subpaths and Explore delegation. This is not a
  locked native toolchain or a full Go/native build check.
- `node --test tests/forgejo/login-theme.test.mjs`: all seven tests passed, now
  including actual head timing before the toggle exists and a no-toggle case.
- Reloaded templates only in existing `sodaos-local-forgejo`. All twelve signed-in
  route variants rendered at 1654px and 390px without horizontal overflow; shared
  intro artwork loaded and search inputs measured 44px. Native Explore overflow
  opened its Organizations menu item and navigated successfully. The syntax dialog
  opened/closed normally; its nested Cancel button matches zero toolbar rules even
  before Forgejo reparents the dialog. The personal context menu stayed within 390px.
- Notification bulk action uses dark action blue with white text; list wrappers and
  twenty visible native rows were verified without submitting actions. Expanded
  native repository initialization/advanced controls fit at 390px; organization
  fieldsets and required input remained intact.
- Guest home/login and all three directories rendered at measured 480px in both
  light/dark modes without overflow. Each page had one toggle (40px; login 44px),
  persisted the choice across navigation and kept native `data-theme`. Login's
  idle submit has no authored `::after`. Unrelated password recovery loaded no guest
  script or theme attribute. Both browser viewport overrides were reset afterward.
- All sixteen linked component/page CSS responses matched source bytes. Changed
  guide file links and `git diff --check` passed.

No fixture records, native account preferences, providers, dependencies, appliance
stage or deployed VM were changed. The guest local choice was restored. Native
notification POST/live replacement, creation POST/server-error and transient
loading/disabled journeys were not newly exercised; source contracts were reviewed.
Signed-in light appearance and organization/team context menus were not exercised
(the local account has no organization context). Template overrides still require
exact-version review and browser checks on upgrades. This is bounded local preview
validation, not production or final-product acceptance. Custom introductory copy
remains English pending the existing i18n work.

## Local New Organization preview

Official `org/create.tmpl` now uses the Soda shell and existing organization
workshop artwork. Native fields, defaults, visibility values, permission checkbox,
error flags and POST action are retained. Desktop cards match the repository form
with 24px padding/gaps and aligned labels. Local reload and Chrome checks covered
desktop, 390px layout without overflow, visibility selection and required name/
40-character constraint. The form was reset afterward; no organization was created.
Submission, server errors and light appearance were not newly exercised. Custom
intro remains English; no appliance deployment.

## Local New Repository preview

Official `repo/create.tmpl` now has the Soda shell, existing repository-folder
artwork and scoped form cards/styles. All native create-helper/basic/template/
initialization/advanced partials, permission gates and form action remain intact.
Local template reload and Chrome inspection verified expanded initialization and
advanced controls, required name/length constraints, asset loading and a 390px
layout without horizontal overflow. No repository was created; submission,
server-side error paths, template selection and light appearance were not newly
exercised. Custom intro is English. No appliance deployment.

## Local Notifications preview

Official stock 15.0.7 notification partial and subscriptions wrapper now use the
Soda shell, dedicated generated inbox artwork, separate consistent toolbar and
rounded list/empty state. Native notification IDs, sequence hooks, forms, data
attributes, status conditions and pagination remain unchanged. Existing fixtures
supply 40 unread notifications; no new data was seeded. Local Chrome verified
populated/read/empty views, mark-read then unread restoration, subscriptions shell,
and a 390px layout without horizontal overflow. Two fixture status checks were
restored to their original states. PNG alpha and template reload were verified;
`git diff --check` passed. Light appearance, pin/bulk actions and watching filters
were not newly exercised. Custom intro remains English; no appliance deployment.

## Local Milestones layout correction

Fixed the dashboard sidebar consuming the row and squeezing milestone cards off
screen. Stock 15.0.7's `.flex-container { display: flex !important }` defeated the
page's grid; the scoped milestone grid now explicitly overrides it. The stylesheet
cache version is bumped. Native filters, milestone data and templates are unchanged.

The new opt-in `tests/forgejo/milestones-layout.test.mjs` reproduced horizontal
overflow before the fix and passed afterward at 1440, 1024, 768, 700 and 390px.
It uses existing local preview public stock CSS, all authored custom styles in load
order and representative milestone markup in isolated headless Chrome—not an
authenticated native-page journey. Run with
`SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/milestones-layout.test.mjs`.
Offline readonly `go test -count=1 -mod=readonly ./scripts -run TestForgejo` and
`git diff --check` also passed. No service reload/restart, fixture mutation or
appliance deployment was performed. CSS is live-mounted in the local preview;
cached pages may need a hard refresh, and the header version takes effect on the
next separately performed template reload.

## Local Milestones preview

The official dashboard milestones override now uses Soda's separate toolbar,
repository filter panel and progress cards, reusing the checklist illustration.
Native milestone data, filtering, rendered content, dates and pagination remain
upstream-owned. Local reload and browser checks covered the populated 8% fixture,
closed empty state, keyword no-match and 390px layout without horizontal overflow.
Deadline/overdue, tracked time, org context, light appearance and pagination were
not newly exercised. No new fixtures or deployment; custom intro is English.

## Local global Issues preview

The official 15.0.7 dashboard Issues template now has Soda styling and new
checklist artwork. Native query/filter/count/context and shared issue-list logic
remain; Pull requests now shares the same layout with its own heading and icons. Local reload succeeded.
Chrome exercised six populated issues, type switching, closed/no-match empty
states, oldest sorting and 390px layout without horizontal overflow. Pull requests
retains its native list partial. Light appearance, org context and pagination were
not newly exercised. No appliance deployment; custom intro copy remains English.

The Pull requests list now shares the Soda list layout and dedicated collaboration
artwork. Local Chrome checks covered reviewed-by filtering, open/closed/merged
fixture rows and review summaries, no-match search and 390px layout without
horizontal overflow. Native review filters, query state and permissions are
unchanged. No new fixtures or deployment; light mode/pagination not newly checked.

## Local signed-in dashboard preview

The personal home feed now has an official Soda dashboard template override,
new generated workbench artwork, responsive feed/sidebar layout and branded
native empty guide. Native account/org navigation, alerts, heatmap, activity
partial/pagination and Vue repository/organization controls remain composed
upstream partials. Shared shell styles match the explorer; Forgejo account theme
state remains authoritative. Custom dashboard copy is English for now.

The local stock 15.0.7 preview reloaded successfully. Chrome verified empty feed,
repository/organization tab switching, light and auto/dark themes, appearance
navigation and 390px layout without horizontal overflow. Account theme was
restored to `forgejo-auto`. Existing Alice authenticated HTTP rendering returned
200 with populated activity and native repo-list markup; its populated layout was
not visually checked. Organization/team contexts, heatmap and feed pagination
were not newly exercised. No new fixture data or appliance deployment.

The dashboard context switcher now uses a compact Soda trigger and rounded
menu with palette colors, monospace caption, active state and keyboard focus.
Its stock markup and context links are unchanged. Local Chrome inspection
verified the open menu, ArrowDown expansion and personal-context navigation;
organization switching was not newly exercised. CSS-only change and local
template reload; no deployment.

## Forgejo illustration consistency

Reviewed all nine page illustrations together and regenerated the repository
explorer, Users and Issues assets using Home/Dashboard as style references.
The replacements align robot proportions, paper materials and scene balance;
canonical logos remain unchanged. PNG alpha and the cream contact sheet were
inspected, then all three pages were visually checked in local Chrome dark mode
after template reload. Asset query versions refresh cached images. No appliance
deployment or backend changes. Prompts and decisions are recorded in
`assets/branding/forgejo/art-consistency-review.md`.

## Local repository explorer preview

The stock Forgejo 15.0.7 local Docker preview now uses a Soda repository-explorer
wrapper and scoped CSS with the approved folder artwork. Native search, list,
filters, sorting, pagination and main navigation remain upstream partials. Guests
share the login/home theme preference; signed-in pages use the same Soda palette
with light/dark selection inherited from the native account color-scheme.
Local browser checks exercised matching and empty searches, alphabetical sorting,
not-archived filtering, light/dark switching and a 480 CSS-pixel narrow layout
without horizontal overflow. The preview now contains 21 public repositories with sample descriptions/topics;
browser checks confirmed 20 rows on page one and one on page two. Language
variants and a fresh signed-in journey were not exercised. This is local
preview evidence only; appliance staging/deployment remains unchanged.

The Users and Organizations directories now share the Soda shell with distinct
headings and separate newly generated papercraft artwork for individual contributors
and a shared organization workshop. Browser inspection confirmed the page-specific
asset references after local template reload. Native user-list privacy/email
conditions, search, sorting and pagination remain upstream-owned. Local browser
checks covered people search, organization sorting/empty state, light/dark
appearance and 480 CSS-pixel layouts without horizontal overflow. No sample
organizations were added, so populated organization rows remain unexercised.

Signed-in explorer parity now includes the Soda logo/palette, themed native account
menus and a shortcut to native appearance settings. Local Chrome checks using the
existing Vince session covered light and auto/dark themes, profile/admin links,
390 CSS-pixel mobile navigation without horizontal overflow, search and page two.
Vince's original `forgejo-auto` preference was restored after verification. Native
navbar permission conditions are unchanged; private-repository authorization and
additional theme families were not newly tested. No appliance deployment occurred.

Branded empty states now cover the three explorer directories through a shared
presentation partial, used only when native result collections are empty. Native
populated lists and visibility/creation authorization are unchanged. Local browser
checks verified repository reset, people search clearing, organization empty
states for guests and the signed-in administrator, dark desktop and light narrow
layout. Empty state copy remains English; no translations or deployments occurred.

## Accepted native evidence

U08 covers the named infra client → isolated `soda-test` journey, **not** a fresh
appliance install, whole-host upgrade, new UI, release or aarch64 proof:

- `f233a4a`: actual existing-container stop/start and guest reboot preservation.
- `c96c108`: exact-image fresh project and different-UID/default-user/PTY/nested
  exec, SQL and Bob's engine denial. The create-time SYS_PTRACE correction was
  exercised; this does not authorize a privileged parent or unrestricted host socket.
- `8b823db`: full native build/seal/aggregate check; backed-up copied populated-v3
  startup rehearsal and affected-component rollout. Go, Cockpit 60 tests, then-
  dashboard 21 tests, 30 build-fixture and nine staging tests passed at that scope.
  Subsequent corrected operator-probe coverage brought build fixtures to 31.
- After rollout: independent operator/Alice/Bob OAuth/navigation/logout, current
  connection authorization/public keys, root Cockpit/PAM and existing-nobody denial,
  direct own-key SSH/PTY/SCP/SFTP, cross-project/sudo boundaries, retained HTTP/SQL,
  separate personal Git agents/remote refs and shared executable identity checks.
  Four environments and declared state survived; probe files were explicit additions.

Earlier lifecycle evidence was reused **only for unchanged mechanisms**, not claimed
as a new `8b823db` reboot. An image-layer-ID equality probe failed; retained content/
mode/owner/link/capability comparisons found only Tea binary content changed while
runtime configuration/dependency content was unchanged. Native Tea version checked.
An existing-output browser observation failed before a new exclusive observation
path was used. Keep these failures, not just successful retries.

The corrected operator script **fails** on missing
`/etc/profile.d/soda-console-welcome.sh`; the earlier apparent success is invalid.
Tailscale was `NeedsLogin`; zero runners/listeners/capacity were observed. These are
not enrollment or provider-job proof. Service/package observations do not certify
interactive console, visual branding or complete operator journeys.

### Evidence locations

These are retained references, not commands to rerun or evidence revalidated by
this documentation cleanup. Private directories/files remain restricted.

| Location | Retained purpose |
| --- | --- |
| `.artifacts/logs/u08-closure-*` | Build/check exit records, `rollout-8b823db`, host/byte binding, corrected operator failure, browser/Cockpit/PAM, developer/exec/workload/client reads and four-project/image comparisons |
| `.artifacts/logs/u08-completion-*`, `.artifacts/logs/u08-ptrace-*` | Earlier lifecycle, runtime diagnostics, fresh-project exec/Git/workload evidence and failures |
| `.artifacts/test-vm/u08-8417a90/` | Original user/project inputs, bindings and observations |
| `.artifacts/test-vm/u08-completion-952f3b3/`, `.artifacts/test-vm/u08-completion-c96c108/` | Completion inputs, before/after snapshots, connection/Git/client evidence |
| `.artifacts/test-vm/u08-closure-8b823db/` | Merged-candidate browser observations |
| Guest `/var/lib/soda/u08-completion-8b823db/` | Private consistent DB/config/key/helper/runner/unit/prior-image backups and rehearsal |
| `.artifacts/retained-native-u08-c96c108/` | Earlier artifacts moved intact before the merged build |
| `.artifacts/retained-worktree-builds/` | Retained builds; `retention.json` maps old worktree/log paths |
| `.artifacts/retired-dashboard-88fc21f/` | Ignored React outputs/dependency cache moved out of source, not erased |

Preserve project IDs `p4a1ba7562c740b1419169fb5`, `pa7cfcfd898ce306d6b23836b`,
`ped30b9d6932974b14feb2278`, `p7b41edaf83f10a6fd7e579bf`, all identities, keys,
roots, dirty checkouts, workloads/volumes, private inputs and backups. Old backups
predate later writes and are not lossless rollback. The live VM overlay depends on
its retained base. See [local access and retention](local-testing.md).

## Latest source checks

| Revision | Actually executed; not installed acceptance |
| --- | --- |
| `752079e` React removal | Full Go suite; web/Forgejo/nativebuild races; 31 Python build fixtures; Cockpit types and 60 tests; shell/document/whitespace checks. Logs `.artifacts/research/react-removal-88fc21f/` |
| `9f3baa7` HTMX removal | Full Go suite; web/store/Forgejo races; 31 Python build fixtures; shell/document/whitespace/caller checks. Initial OAuth schema/scope test failures and final passes retained in `.artifacts/research/htmx-removal-752079e/`. Cockpit not retested in that slice |

Go checks used cached 1.26.7, readonly modules and disabled resolution; some results
were cached. No dependencies, images, native stage or VM state were changed in
these removals. HTTP join/failure coverage was adapted to the retained JSON API,
not discarded with the HTML forms. Retired browser scripts remain in Git at their
matching revision and must not be run against the new API-only source.

## Minimal UI source inspection

After `c65aa37`, inspected retained Forgejo 15.0.7 header/footer hooks, native
`<dialog>` styles/browser usage, loading CSS, clipboard delegation and selected
Fomantic components. Recorded the [bounded candidate and control states](forgejo-frontend-integration.md#minimal-button-drawer-and-loading-candidate):
footer-hook markup plus moving our own button into the existing repository action
row, scoped right-aligned dialog CSS, content-only native spinner and native copy
controls. No full header override, new tab or library is needed for this candidate.
The row selector/initialization, browser layout/accessibility and authenticated
connection remained untested at that revision. Its Soda Origin/CSRF checks and
two-origin proxy prevented simply wiring native-page fetches; the namespace was
then only a candidate. See the routing foundation below for subsequent source work.

Only source inspection/documentation/link/whitespace checks ran in this slice.
No production source, payload, dependency, browser/native test or installed state
changed. This is not a working drawer or a claim that the whole integration is
four static files. The existing-account terminal remains a separate follow-up.

## Implementation planning

After `64aad1f`, expanded the existing [Sodaspaces plan](sodaspaces-plan.md), not a
parallel roadmap: same-origin API/OAuth contract, repository-scoped reads, small
read-only hook/drawer delivery, explicit access actions, native-browser validation
and separately approved rehearsal/cutover. The candidate uses Go/JSON plus vanilla
JavaScript and native `<dialog>`, with no added HTMX or frontend build. Fixed
`/-/soda/` routing, single-origin configuration, scoped cookies, repository return
context and stable-ID API changes were **planning only** at that revision. The
routing foundation below records the first implemented subset.

Inspected actual Go/config/setup/staging callers and retained Forgejo 15.0.7 routing
and OAuth application handlers. Native callback editing need not rotate the secret;
the upstream API PATCH does. Recorded that distinction and the template allowlist
gap in the integration guide. Source hashes and documentation checks are retained
in `.artifacts/research/sodaspaces-plan-64aad1f/`. Only source inspection and
Markdown/link/whitespace checks ran; no product tests, build, browser, dependency,
provider, native or private-state actions occurred. The terminal stays separate.

## Test ownership clarification

Following `81bacca`, removed broad upstream-regression requirements from the
Sodaspaces plan and native-validation guide. Tests for removed standalone frontends
and duplicate forge adapters were already deleted in `752079e`/`9f3baa7`; no further
upstream-only test files were identified in the current tracked inventory. Retained
Forgejo client/credential/branding tests exercise Soda-owned code, as do native
project Git/access and Cockpit checks. Keep those and narrowly targeted integration
smoke checks; do not recreate upstream business-logic/conformance suites.

This change is documentation-only: source/test inventory inspection and Markdown
link/whitespace checks, no product test execution or installed-state changes.

## Sodaspaces routing foundation

First implementation slice after `e59f99e`: Go mounts API/login/callback routes
under `/-/soda/`; Caddy's source recipe forwards only that prefix unchanged and
leaves other paths with Forgejo. `forgejo_url` is the sole browser origin;
`public_url` / `--public-url` / `SODA_ORIGIN` are retired. Setup, activation, console
output, strict-loader tests, connection/operator probes and staging assertions
follow that contract. This configuration is incompatible with the installed old
loader/config pairing; no retained configuration or OAuth application was changed.

New host-only Secure/HttpOnly/SameSite=Lax cookies use unique names and the Soda
path. Legacy/native cookies are ignored; duplicate/empty/oversized Soda cookies
fail closed. Callback/session rotation and mutations retain PKCE/state, encrypted
grants, exact-origin/CSRF and refresh/logout checks. Go rejects unprefixed API/auth
aliases and encoded/unclean mounted paths without redirects; creation Location
headers include the prefix. Direct backend root/health remain. In that slice,
schema v3 and keys were unchanged and OAuth still returned only to Forgejo home.

Passed full Go suite, web/config/store/Forgejo races, 31 Python build fixtures,
Python/JavaScript/shell syntax and documentation/whitespace checks. Go used cached 1.26.7,
readonly modules and disabled dependency resolution; some results were cached.
Logs: `.artifacts/research/sodaspaces-routing-e59f99e/`. No native image/stage build,
staged-payload execution, Caddy/browser execution, Cockpit retest, deployment,
restart, provider action or retained-state change. The new staging assertion is
authored, not an executed staged-payload result.

No UI/mutation controls were added in `6deaf9a`. Actor/return handling followed in
the next slice below; the drawer and repository-scoped reads remain pending.

## Actor guards and OAuth context

After `6deaf9a`, protected APIs require `X-Soda-Expected-User-ID`, except optional
bootstrap on `GET /api/session`. Malformed/missing context is 400; mismatch with the
Soda session is 403 before handlers. Session/CSRF/provider/operation authorization
remains separate. The header cannot authenticate a native browser session or select
another actor. The retained connection probe now declares its checked fixture actor.

Login accepts bounded optional repository/expected-user IDs. Schema v4 appends two
default-zero fields to the existing OAuth table; atomic consume returns them with
the verifier. Callback checks the fresh provider subject before changing profiles/
sessions/grants, then uses actual consent and acting-grant repository-by-ID lookup
to reconstruct the native return path plus `#sodaspaces`, or home on unavailable
context. Caller callback IDs/URLs and provider URLs are ignored. Query limits,
duplicate/encoding rejection and no-referrer headers protect the authentication
boundary. Existing encrypted grants and project records are not rewritten.

Passed focused/full Go suites, web/store/Forgejo/config races, 31 Python build
fixtures, JavaScript syntax and documentation/whitespace checks. Go used cached
1.26.7, readonly modules and disabled resolution; some results were cached. The
migration test uses a genuine v3-schema local fixture: missing/wrong keys do not
migrate it; a correct key preserves product rows and encrypted bytes. This is not
copied-private-state rehearsal. Logs/source hashes:
`.artifacts/research/sodaspaces-context-6deaf9a/`. No Caddy/browser/native stage or
image execution, Cockpit retest, deployment, provider mutation or retained-data change.

**Milestone 1 still needs real native-page/proxy/browser proof.** The pending caller
must capture native page identity, compare page/session/fresh-provider IDs and reload
stale native context on resume/BFCache restoration before exposing actions. Native-
only login/logout while Soda's session is unchanged is not detectable by this header;
no atomic cross-system logout is claimed. See the [API caller boundary](dashboard-api.md#native-page-and-stale-tab-boundary).
At that revision the drawer, repository-scoped reads and mutation controls were
not implemented. Subsequent backend repairs are recorded below.

## Security review and fix plan

Additional review at `a9fef51` confirmed two pre-existing gaps, **not fixes or new
regressions in that commit**: an in-flight OAuth callback can create a live Soda
session/grant after Soda logout succeeds; and the trusted-team catalog/new-join
handlers do not enforce repository visibility. The latter is the already-planned
repository-scoping boundary, now explicitly required before UI work enables joins.

Review checks actually run: uncached web/store/config/Forgejo Go race suites passed;
two review-only negative assertions failed, reproducing the gaps through real Soda
handlers/temporary SQLite with fake provider/helper responses. Retained source,
overlay, logs and exit records: `.artifacts/research/sodaspaces-security-a9fef51/`.
No tracked source, installed state, provider or native operation changed in review.
The existing logout-winning persistence evidence covers grant refresh, not callbacks.

The user then requested a fix plan. The existing [step-1 callback/logout plan](sodaspaces-plan.md#oauth-callback-and-logout-fix)
selects a bounded persisted login context and atomic cancellation/finalization;
the [step-2 repository plan](sodaspaces-plan.md#repository-authorization-fix) moves new-join
authorization ahead of drawer wiring while preserving legitimate existing-member
access. At that planning revision both were **unimplemented**. No schema change,
session invalidation or Linux revocation occurred then. That change edited documentation only;
relative-link/anchor and whitespace checks ran, not additional product tests or builds.
Native browser/proxy proof, migration rehearsal and rollout remain separately scoped.

### Callback/logout repair in source

Implemented the first fix: one persisted login context and pending-state hash bind
OAuth claims and rotating Soda sessions. Callback finalization atomically checks
cancellation/expiry/supersession before profile/session/grant writes. Logout carries
its authenticated context across concurrent rotation; either commit ordering leaves
no usable session/grant after successful logout. Superseded callbacks write no
cookies; a delayed successful cookie for a deleted session remains unusable. No new
browser cookie, native identity authority or global revocation was introduced.

Schema v5 backfills independent contexts for existing sessions without rewriting
their token/identity/expiry/grant bytes. Pre-v5 pending OAuth must restart. Local
real-v3/v4 fixtures cover migration preservation and wrong/missing-key refusal.
Uncached web/store/config/Forgejo race suites passed, including deterministic HTTP
logout/callback orderings and store cancellation/supersession/rollback/restart tests.
Logs: `.artifacts/research/security-fixes-1b3e355/auth-race.log`; initial focused pass
also retained. Repository authorization remains next. No deployment, native/browser,
provider or retained-data changes; only local source checks ran.

### Repository authorization repair in source

Replaced the catalog with required `repository_id` lookup through the acting grant:
fresh subject, actual read user/repository consent, current visibility and the unique
Soda association. Response includes current repository context and advisory owner
creation availability, never all projects. Direct-ID detail/member reads authorize
ordinary nonmembers before metadata/native inspection; existing members' own degraded
reads and explicit Soda operator inspection remain. Full member-list elevation still
requires current human/org ownership or the configured operator, not visibility alone.

New joins independently repeat current identity/repository checks against the stored
repository ID before readiness disclosure or fixed account calls. No operator/admin
bypass or setup-token fallback. New accounts use the fresh provider login; existing
joins retain the original login with no reinstallation or provider dependency.
Membership remains contingent on helper success. Stable-ID creation is still pending;
this changes neither existing Linux access nor native Git permissions.

Full uncached Go suite passed. Focused race coverage includes denial/no-grant/consent,
subject mismatch, malformed/oversized/timeout responses, direct-ID disclosure, rename/
transfer, access lost between read/join, native failure, concurrent joins, unsaved
results and own connection during provider failure. The initial repository suite
failure exposed the obsolete catalog assertion; the corrected test now requires a
400 JSON response without repository context. Failure and final logs are retained at
`.artifacts/research/security-fixes-1b3e355/`. A further pool-replacement regression
reproduced lost SQLite FK cascades after connection recycling. Foreign-key and
busy-timeout pragmas now apply to every connection through an escaped file URI;
logout removes session/grant rows even after replacement. The failing reproduction
and subsequent full Go/race passes are retained. All 31 Python build fixtures also
passed. No native/browser/proxy execution or retained-project/provider/deployment changes. Both fixes are source-implemented;
real browser proof and v5 preserved-state rehearsal remain required before rollout.

## Next read-only milestone plan

After `ddb2d4f`, expanded [step 3 of the existing plan](sodaspaces-plan.md#3-deliver-the-read-only-button-and-drawer)
for native template IDs, explicit Soda authentication, repository-scoped reads and
one read-only dialog. The two security fixes stay implemented; this does not redo
them or add a roadmap. Selected stale-tab handling clears data and requires an
explicit native-page reload, preserving unsaved native form edits. Hook/assets and
source tests come first, bounded packaging/conflict fixtures next, then an opt-in
native browser journey. No new endpoint/schema/frontend build is planned.

Verified stock 15.0.7 identity fields/footer ordering and existing custom-asset
cache configuration in source. Recorded the installer target-file conflict gap and
narrow template allowlist/mode/ownership work; neither fix is implemented here.
The real OAuth/Caddy return, cookies, stale tabs and accessible native rendering
remain completion checks before mutation controls, with separately approved fixtures
and no implied retained-target migration, restart or delivery.

Changed documentation only. Source inspection and Markdown link/anchor/whitespace
checks ran; no product tests, builds, dependency resolution, browser/proxy execution,
provider or retained-state actions. Planning source hashes and check logs:
`.artifacts/research/read-only-plan-ddb2d4f/`.
The drawer remains absent; only historical bounded U08 is accepted.

## Read-only caller source

After `05217f7`, added two original custom hooks and scoped CSS/vanilla JavaScript
under `appliance/forgejo/`. Only our button moves into the native action row; browser
`<dialog>` owns the overlay. String IDs bind session/provider/repository reads;
explicit contextual OAuth and Soda-only logout remain distinct. Reads are bounded,
time-limited and generation-guarded. Hidden/blurred/restored pages clear data and
require explicit reload, without discarding native form edits automatically. No
create/key/join/connection/lifecycle/terminal controls or backend/schema changes.

Soda template tests and Node/jsdom tests exercise actual markup/script with fake
provider responses/dialog methods, not native rendering or account provisioning.
Focused Go script tests and DOM checks passed locally using cached tools; logs are
in `.artifacts/research/read-only-05217f7/`. The aggregate source-check entrypoint
now invokes the DOM test; its actual-stage requirement is unchanged. Packaging,
opt-in native journey and real browser/proxy proof still follow. No dependencies,
images, stage, services, provider credentials or retained state changed.

## Read-only packaging source

The production stage now copies the four hook/assets with readable modes despite
a private builder umask. The bundle requires them and admits only the two custom
templates/ancestors, not an arbitrary template tree; source LICENSE/NOTICE are
included alongside retained notices. First-install uses an actual readonly
preflight function to refuse occupied hook/asset targets before writes and adjusts
ownership only for the exact new template paths. It remains a first installer,
not a retained-target upgrade or customization merger.

Full uncached Go tests and all 34 Python build fixtures passed locally, including
real stage logic with synthetic build inputs and readonly installer logic against
temporary filesystems. Logs: `.artifacts/research/read-only-05217f7/`. These fixtures
are not an actual native stage/build/install. The added actual-stage assertions
remain unexecuted; no service, native target, credentials or project data changed.

## Native probe source and final local checks

The opt-in [read-only browser journey](native-validation.md#read-only-sodaspaces-browser-probe)
is now authored, not executed against a provider/browser/proxy. It uses actual
native password/consent forms, two existing users and a public repository; only
explicit authentication writes are permitted. It checks served asset bytes and
conditional revalidation, proxy aliases/encoding, scoped cookies, actor/CSRF denial,
OAuth returns, native-only switching, Soda-only logout, stale tabs, native form
coexistence, keyboard/layout/themes and actual BFCache restoration. An unobserved
BFCache restoration returns incomplete scope, not a synthetic pass. Source inspection
confirmed stock version compatibility suffixes and Playwright's default BFCache
exclusion. Sandbox/TLS protections remain enabled, and failed profiles are retained.

CLI preflight tests cover missing permission, sanitized malformed private input and
refusal to finalize into an occupied run. These use synthetic files and git/transport
doubles, never a browser, provider or real credential. Final delivery review also
extended the installed-byte verifier to the exact new template files, with changed
hook bytes/mode regressions. The read-only caller displays a fresh provider rename
while preserving the original own project login.

Full uncached Go suite and web/store/Forgejo/config/nativebuild race suites passed;
43 DOM tests and all 36 Python build fixtures passed. Node syntax, shell syntax and
documentation/whitespace checks passed. Logs and source hashes are retained in
`.artifacts/research/read-only-05217f7/`. Go used cached 1.26.7, readonly modules and
disabled resolution; Node used the existing pinned runtime/Cockpit jsdom dependency.
No dependency installation, real native stage/image build, staged-payload suite,
Cockpit retest, browser launch, Caddy/provider execution, service/VM action, private
state migration or project change occurred. The new source test invocations of the
installed probe stop at preflight. Real OAuth/proxy/browser and staged/installed
verification remain held for an explicitly approved fixture/target and exact artifacts.
Step-4 mutation controls remain absent; no new native/product acceptance is claimed.

## Isolated local Sodaspaces browser execution

The user explicitly approved a new isolated local Forgejo/Caddy fixture on the
existing development machine, with synthetic users/repository and private TLS.
This authorizes this fixture's initialization and authentication/browser checks,
not installation on the builder or changes to `soda-test`, retained environments,
host trust/network policy, provider resources outside the fixture or cutover.

Evidence/state: `.artifacts/local-sodaspaces-31e73bf/`, retained privately. Built
`31e73bf`'s Go backend with the cached pinned toolchain and readonly/offline modules.
Stock cached Forgejo 15.0.7 and Caddy 2.10.2 run with that backend in a new rootless
shared network/user namespace; only `127.0.0.1:31443` is published. This is an
integration fixture, not installed appliance topology or a full native stage.
NSS tools were downloaded/extracted locally, not installed; only a fresh private
browser home's trust database received the new fixture CA. Native CLI/official
APIs created two synthetic users, one public repository and one confidential OAuth
client. No Soda sessions, grants, environments or memberships were seeded; no host
helper is connected. Native passwords/token output went directly from captured
process memory into restricted secret files, not logs. Runtime service logs are
discarded to avoid recording OAuth URLs or other credentials.

Initial container failures are retained: missing fixture SELinux volume labels,
and stock Caddy's file capability refusing execution with an empty capability
bounding set. Only new fixture paths were relabelled. Caddy retains just
`NET_BIND_SERVICE` in the rootless namespace, with no-new-privileges; no host
capability/security policy was changed. Failed containers were not removed.

**Complete scoped journey passed at probe `dd793a2`**, against unchanged `31e73bf`
backend/UI bytes. `browser-x/sodaspaces-run/result.json` records real BFCache
restoration, both users' real OAuth returns, native-only/Soda-only transitions,
identity mismatch/no premature environment reads, protected logout actor/CSRF
denials, cookie scope, native unsaved-form preservation, keyboard/blur/explicit
reload, Escape/backdrop/focus return and 360/1280-pixel light/dark layout. Raw
proxy path/encoding denials and exact served asset hashes/conditional revalidation
also passed. Only **absent** environment views were native (three observations);
existing/running/stopped/incomplete views remain source/DOM fixtures in this run.
No responses or sessions were faked. This is bounded browser integration evidence,
not milestone/release acceptance or installed CoreOS/aarch64 validation.

`artifact-binding.json` binds the running backend's `/proc/1/exe` hash to its local
Go/VCS build and the four mounted/served UI hashes, with exact cached image IDs.
Read-only inspection of Soda's own fresh schema-v5 database found zero projects,
memberships and development keys. It did not inspect Forgejo's database. Services,
all failed containers, private profiles, inputs and logs remain retained. No real
stage/image build, installed checks, retained-state migration or cutover occurred.

The failed attempts remain under `browser-*`/`logs/`, with their exact revisions.
They exposed probe assumptions rather than requiring product/upstream changes:

- Empty native data regions have no visible box; wait on dialog/ARIA readiness.
- Playwright routes omit redirects; CDP Fetch guards every exercised-page hop.
- Playwright forces pages focused/visible. Stock Chromium now attaches with public
  `connectOverCDP({noDefaults:true})` through a private Unix socket/pipe, retaining
  sandbox/TLS and real focus/visibility/BFCache rather than synthesizing events.
- Native chrome/body focus transitions invalidate Soda; focus return is asynchronous.
- Forgejo's stock logout broadcasts navigate session tabs home and its link action
  posts `/-/fetch-redirect`. Only the exact navigation-only `redirect=/` form is
  allowed. Workers, upstream navigation and native beforeunload stay unmodified.
- BFCache history restoration needs a commit wait, not a fresh load-event wait.

Final local checks passed: full uncached Go suite, 46 Node tests (43 DOM plus three
probe/transport checks), 36 Python build fixtures, Node/bash syntax. Logs are under
`logs/final-*`. Go used cached 1.26.7 with readonly/offline modules. No Cockpit
retest, aggregate native-stage check or additional dependency install was implied.
Transport cleanup/refusal hardening also passed the full native journey at
`2d6a2b3`: `browser-y/sodaspaces-run/result.json` and its exit record. The final
`artifact-binding-final.json` ties that probe to unchanged product bytes; prepared
browser/tool identity and probe source hashes are retained separately. Both full
passes observed real BFCache. Documentation checks covered 55 Markdown files,
240 local links and 33 anchors with no errors; whitespace checks passed.

## Read-only native stage and exported delivery closure

The user gave standing implementation/testing approval for this planned work.
Executed the production `scripts/build-native.sh x86_64` and
`scripts/check-native.sh x86_64` in a fresh clean detached `ee8091a` worktree at
`.artifacts/worktrees/stage-ee8091a/`. Both exited 0. This built all native commands,
Cockpit, Tea, project/dashboard images and the four OCI archives, fetched locked
runner inputs, staged and sealed the actual payload. No placeholder stage was used.

The aggregate verified that exact stage before/after checks: Go suite, 46 Node
tests, Cockpit TypeScript and 60 tests, 36 build fixtures and **11 actual-stage
checks** passed. Separate uncached web/store/Forgejo/config/nativebuild races passed.
The exported bundle at `.artifacts/stage-validation-ee8091a/export/x86_64/` verified
before and after browser use. A mistaken first verifier invocation at the bundle
root failed with exit 127; its log remains. The correct `tools/soda-artifacts`
invocations passed. No source/product correction or dependency-baseline edit was
needed. Actual mutable package resolution is recorded by the build; the new project
image is not granted the older installed runtime's lifecycle/SSH/workload acceptance.

A **new** retained local delivery fixture, `.artifacts/delivery-ee8091a/`, used the
export's readonly templates/assets/branding and Caddy recipe, stock Forgejo/Caddy
images matching exported OCI config IDs, and the **built dashboard image at its
configured UID 2000**, read-only root with no capabilities. Rootless shared network/
user namespaces and only `127.0.0.1:32443` published; no builder appliance install.
Fixture-only configuration, new native users/repository/OAuth client and TLS were
initialized through native CLI/official APIs. No helper is connected. The previous
local fixture and retained `soda-test` installation/projects were untouched.

The exact `ee8091a` product-owned browser journey exited 0 against that delivered
payload: both users' real OAuth/consent/identity/cookie/logout flows, real native
blur/stale-tab and BFCache restoration, native unsaved form preservation, keyboard/
backdrop/focus and responsive light/dark rendering passed. Native `soda-auto` default
and its served stylesheet were separately verified against the export. Three native
environment observations were **absent**, not running/stopped/incomplete proof.
Executed backend `/proc/1/exe` matches the exported binary; runtime image IDs match
exported OCI configs. Read-only Soda database inspection found fresh schema v5 and
zero projects, memberships and development keys; no Forgejo database inspection.

Evidence: `.artifacts/stage-validation-ee8091a/` (build/check/race/export logs and
manifest) and `.artifacts/delivery-ee8091a/` (private native result, artifact/image/UID/
port/theme bindings, inputs and profiles). Both fixtures' services, all worktrees,
outputs and failures remain retained. **Step 3's bounded x86_64 read-only exit is
satisfied; step 4 is next.** This is not first-install/activation, retained-state
migration/cutover, existing-project access, full product/release or aarch64 acceptance.

## Access-action plan revision

Documentation-only review after `afda2d9` reconciles the leading plan and API guide
with the completed read-only build/export/browser evidence and standing testing
approval. Step 4 now explicitly separates action-time identity checks, pending writes,
late/stale results, confirmed versus uncertain outcomes and safe read-only observation
without replay. Existing-member idempotency/degraded access and current-owner/new-join
server authority remain intact. Step 5 extends the existing guarded native journey
with bounded access requests on an isolated helper-backed target; neither retained
browser fixture supplies provisioning or SSH proof. No new roadmap, backend contract,
helper protocol, recovery subsystem or live cutover is implemented by this revision.

Checks for this revision: documentation inspection passed (55 Markdown files,
242 local relative links, 34 Markdown anchors, zero errors); diff-whitespace checks
passed. No product tests, build, native execution, fixture mutation or data cleanup.

## Explicit access actions source implementation

Implemented step 4 after `f940338`, without a new frontend stack, helper protocol,
permission inventory or recovery subsystem:

- Create accepts only canonical decimal-string `repository_id`. The existing
  `visibleRepository` path checks actual user/repository consent, fresh subject and
  stable-ID visibility; the acting human must be the current owner. Reservation
  precedes native provisioning, uniqueness survives concurrent requests and create
  never joins. The unused owner/name Forgejo adapter is removed. The API's existing
  bounded decoder now uses `strictjson` to reject duplicate fields/invalid UTF-8 too.
- The existing four hook/assets add separate Create, development-key summary/public-
  key save, Add me and own SSH connection controls. Public-only input is checked
  before transmission and by the retained Go SSH parser. Save never joins or updates
  existing Linux keys. Existing-member original logins and degraded API access remain.
- Each explicit action rechecks page/session/provider consistency, sends one protected
  POST, disables duplicate dispatch/logout while pending and safely rereads afterwards.
  Close/blur/BFCache cannot cancel or replay native work. Close/backdrop invalidate
  synchronously before the native queued close event; late writes cannot populate a
  reopened/stale drawer. Uncertain create/join stays blocked in that document with
  operator-inspection guidance, not a claim that a missing row proves no native effect.
- Own connection rendering validates association, original login, current running
  address and public host-key/fingerprint fields; stopped/unavailable/stale views
  clear the command and Copy target. Copy delegates through stock 15.0.7's inspected
  `clipboard.js`, not a replacement handler. Address display is not routing proof.

Performed under standing local testing approval: full uncached Go suite; uncached
web/store/Forgejo/config/nativebuild races; **89 Node tests** (86 drawer DOM cases and
three retained probe/transport cases); **36 Python build fixtures**. All passed.
Source checks include owner transfer/admin non-bypass, large/invalid IDs, provider/
consent/actor denials, concurrent reservations and native/persistence failures,
separate actions, private/options/multiple-key refusal, pending/stale/queued-close
results, no replay or false membership and unavailable connection/Copy behavior.
Earlier focused passes and final logs are retained in
`.artifacts/research/access-f940338/`. Documentation/whitespace checks accompany the
change; no new dependencies or baseline versions changed.

This is handler/store/DOM and packaging-fixture evidence, **not** a new real image/
stage/export, native browser/Copy interaction, helper account/key installation, SSH,
project-runtime or aarch64 pass. No Cockpit retest, fixture service restart, helper
connection, retained database/config/OAuth migration, VM/project change or cutover
occurred. Both browser fixtures, `soda-test`, all four retained environments, earlier
worktrees/artifacts and private credentials/evidence remain untouched. Step 4's local
exit is satisfied; step 5 must bind updated delivered UI/backend/helper/project bytes
to real account/key/SSH results before native access is claimed.

## Phase-5 execution started

The user requested phases 5 and 6 after `658f2af`. A fresh native x86_64 KVM fixture
`soda-native-spaces-658f2af` is retained under `.artifacts/access-vm-658f2af/`, with a
new overlay over the preserved CoreOS base, localhost SSH 22230/browser 33443,
private generated host/client keys/password/Ignition inputs and the existing operator
public key. Native extension layering completed; one fixture-only activation reboot
was requested. The first SSH observation incorrectly assumed Python existed before
activating the layered deployment; exit 127/broken-pipe evidence is retained. The
retained `soda-test` hostname, architecture and application/helper service status were
read only; no retained service, configuration, callback, database or project changed.
An initial socket check used the wrong path; the actual `/run/soda/host.sock` is
root:soda 0660 and its socket unit is active.

The existing native browser journey now has an explicit, single-use actor/path/body-
bound access mode; the retained SSH/PTY/SCP/SFTP probe takes declared Sodaspaces
connections instead of historical fixed U08 fixture names. Stock 15.0.7's native
Copy tooltip appends to the document body by default; Soda's Copy button now uses
its supported `data-tooltip-appendto="parent"` attribute so feedback stays inside
the modal top layer. This is source-backed preparation, not a completed native Copy
or access pass. Phase 5 will use a separate fixture-local client network namespace;
no builder routing change or laptop-route proof is implied. Phase-6 preserved-state
rehearsal and live cutover have not begun. Local source checks passed: 89 Node tests,
37 Python build fixtures and focused Go scripts/nativebuild tests, plus documentation
links/anchors and whitespace. They do not establish a native access result.

## Phase-5 bounded native access proof

Candidate **`bdbce8e736b26dfaf81c37b1917386a536ef1fac`** passed production native
x86_64 build/check/export from its clean detached worktree, including full Go,
89 Node tests, Cockpit TypeScript/60 tests, 37 Python fixtures and 11 actual-stage
checks. Separate uncached web/store/Forgejo/config/nativebuild races passed. The
transferred verifier checksum and bundle inventory were checked before first
installation on the new CoreOS fixture; `verify-installed` subsequently passed.

The new fixture completed native layering/activation reboot, first installation,
native Forgejo setup, production `soda-setup` and `soda-activate` with private TLS.
The old initialization recipe wrongly expected a redirect: stock 15.0.7's
`InstallDone` returns HTTP 200 after committing installation. The failure is retained;
native CLI inspection confirmed only Alice existed, then a separately guarded
continuation created Bob/OAuth/repository without replaying installation or replacing
state. Public host-key inspection first omitted the production `soda-` container
prefix; no container was changed. A diagnostic image comparison initially failed on
Podman's omitted `sha256:` prefix; exact digest comparisons then passed. All failed
observations remain, not relabelled as native defects.

The exact candidate's real sandboxed/TLS-trusted browser access journey passed:
real BFCache/account/cookie/logout/stale/native-form checks, nonowner create denial,
one owner-created environment **`p4a530c394bcd53e563d3076d`**, separate Alice/Bob
public-key saves and real helper-backed joins, own connections, native Copy success
feedback and actual clipboard paste for both users. No Soda sessions/grants or
membership rows were seeded and no native response was substituted. Observed states
were absent and running; stopped/incomplete/uncertain-result branches retain their
source/DOM evidence, not invented native observations.

The separately named `soda-phase5-client` used a normal bridge network namespace
inside the fixture, read-only root, no capabilities and no-new-privileges. Both users
passed direct **10.90.0.2** SSH, interactive PTY, bidirectional SCP/SFTP, project-root
UID-map separation and expected owner/nonowner sudo behavior; Alice's key was denied
for Bob's login with a real public-key denial, not transport failure. Host trust came
from independently read public project host-key bytes, compared with browser output;
private client keys were never uploaded to Soda. All probe directories remain.
This proves that fixture-local client path, **not builder/laptop/Tailnet routing**.

`verify-installed`, runtime image IDs, the backend's actual `/proc/1/exe` and the new
project's image/labels match the export. The native schema-v5 Soda DB has one project,
two memberships and two development keys, with integrity checked. Source under
`internal/host/`, `cmd/soda-host/`, `project-os/`, `internal/runners/` and
`cmd/soda-runners/` is unchanged from installed `8b823db`; compiled provenance and
mutable image package resolution remain distinct from source equality.

Evidence: `.artifacts/stage-validation-bdbce8e/`, retained worktree
`.artifacts/worktrees/stage-bdbce8e/`, and `.artifacts/access-vm-658f2af/` (browser-a
result, exact artifact binding and copied client results). The VM/overlay/base,
project/accounts/keys, services, exited client/conversion containers, private inputs,
profiles and failures remain retained. Phase 5's bounded x86_64 access exit passed;
this is not whole-product/operator/workload/lifecycle/aarch64 acceptance.

A follow-up audit found five OAuth-state query lines in the fresh fixture's default
Forgejo router journal; no code/token parameter lines were observed by that bounded
audit. No raw journal or parameter values were emitted. Supported native logging
configuration now disables the query-bearing router logger and retains console
method/escaped-path/status access records. Native effective settings and a repeated
real read-only/OAuth/BFCache journey (`browser-c`) passed, with query-free OAuth access
records and zero audited credential/state query lines afterward. General service/error
logging remains native. Earlier journals and failed recipe diagnostics are preserved.
The default is also authored in `appliance/config/forgejo.env`, with a focused template
check; this is an explicit configuration follow-up to the built `bdbce8e` images,
not a claim that a later source revision was rebuilt or installed wholesale.

## Phase-6 preserved-state rehearsal

Evidence is retained in `.artifacts/phase6-658f2af/`. At rehearsal, `soda-test` was schema v3
with 3 profiles, 2 keys, 4 projects, 7 memberships, 10 sessions and 9 encrypted grants;
all four project roots were running and exact Soda hooks/assets absent. Existing
browser tunnels and trusted Forgejo TLS work. Only its original Soda service was
stopped for a SQLite backup plus matching private config/key/artifact capture and
resumed unchanged. The resume recipe initially checked the wrong mutable `:dev` tag;
inspection established the actual image-pinned Quadlet, which was resumed and verified
against the original image/config and added to the backup. No live schema, callback,
configuration, helper, project, default project image or runner-service change occurred.

The consistent private set is `/var/lib/soda-sodaspaces-bdbce8e/backup` on `soda-test`
and `.artifacts/phase6-658f2af/backup/` on the builder. Copies on the fresh fixture
(`/var/lib/soda-phase6-rehearsal/`) ran the actual prior and `bdbce8e` images with
network=none, no helper mount, UID2000, no capabilities, read-only root and explicit
copy-only writable data. The first negative recipe expected an unsanitized key error;
production correctly emitted its sanitized startup-stage message. That attempt is
retained; separate reviewed copies then passed all eight cases:

- legacy config and missing/wrong key refusal, leaving copied v3 bytes/rows unchanged;
- successful v3 → v5 migration preserving every original column/row and encrypted
  grant/key-check byte, with integrity/FKs and all migrated session contexts checked;
- healthy empty v5, future-version refusal, prior-image refusal of migrated v5;
- healthy paired prior-image/config/key/v3-copy rollback, preserving all original data.

This is actual copied private-state/native-image evidence, not a live rollback or
permission to restore an old backup after later writes. Only exact run-owned rehearsal
containers were stopped; all copies, failed/exited containers and original roots remain.

Official acting-owner API reads confirmed retained OAuth application 4, its unchanged
client and sole prior `https://localhost:24443/oauth/callback`; the planned callback is
`https://localhost:24444/-/soda/oauth/callback`. No application PATCH or Forgejo DB access
occurred. Retained `/u08-alice-8417/shared-alice` is private and Issues-enabled; it must
not be made public to fit a probe. The declared read-only private-repository probe
variant requires native anonymous 404/no Soda reads and then the normal authenticated
journey, never environment/key writes. That variant passed as probe revision
`0992f20ab295c1199ba58ccf34ac377012d67b9c` against the fresh fixture's built `bdbce8e`
images plus tested query-free logging configuration (`browser-d`). A new synthetic
private repository with native Alice ownership/Bob read access was used; no existing
repository visibility changed and no new Soda environment was created. Its three
absent observations and real BFCache are not retained-target running-view evidence.
Official reads confirm retained Alice ownership/Bob write access already exists.

Final retained observations match every preflight field, all four container/image/
running-state identities and every original Soda table row against the backup at that
observation. Key fingerprint and original-login shapes fit the drawer contract. The
fresh fixture still passes installed/runtime artifact binding with exactly one project,
two memberships and two development keys. No later backup should be assumed current.

Follow-up local checks passed: uncached full Go tests, **91 Node tests**, **37 Python
build fixtures**, JS syntax, whitespace and documentation links/anchors. Evidence is
`.artifacts/research/phase6-bdbce8e/`. No new compiled backend/helper/UI payload or
whole native bundle was built after `bdbce8e`; the follow-up changes are logging
configuration, probe/tests and documentation.

**The user subsequently approved live cutover.** The
[affected-component procedure](installation.md#retained-sodaspaces-cutover) covers
fresh-at-cutover backups, owner-native callback editing, image-pinned backend,
strict-config `soda-runners` CLI, proxy/namespace/hooks and query-free native logging.
Unchanged helper, project roots/default image and runner services are not upgrade
targets. See the executed cutover below.

## Approved retained cutover

The user explicitly approved the documented affected-component cutover after the
rehearsal. Evidence is under `.artifacts/cutover-c007eb6/`; the fresh private backup
on `soda-test` is `/var/lib/soda-cutover-c007eb6/backup`. The old rehearsal backup was
not reused as current state. The actual image-pinned unit, prior image, consistent
SQLite data, credentials/key/config, proxy, native configuration and file metadata
were preserved. An initial SCP transfer could not preserve two relative bundle
symlinks; that partial tree remains untouched. A separately named tar transfer passed
the production bundle verifier before deployment.

Built `bdbce8e` dashboard image/binary and strict-config runners CLI, exact four native
hooks/assets, namespaced proxy/config and reviewed query-free logging were delivered.
The actual owner changed only app 4's callback through native Applications settings;
client identity/name/confidential setting and credential-file bytes were preserved.
No API PATCH, secret generation or Forgejo DB access occurred. Only affected services
were restarted; no helper/project/default-image/runner-service or routing change.

Live schema v3 → v5 passed integrity/FK checks and preserved every original column/row
and grant/key-check ciphertext **before browser login**, with migrated contexts checked.
The running backend executable/image match the export. Native runner `list` succeeded.
The retained host lacked the selected static-cache setting (native default six hours);
asset validation caught it. `STATIC_CACHE_TIME=0` was then applied, Forgejo restarted
and actual conditional reads returned 304. Earlier input/cache/startup-preflight
failures remain, not overwritten. A restricted CA copy was used without changing the
original CA file or global trust.

The real private-repository browser journey passed at probe `c007eb6` (`browser-d`),
with three running views and genuine BFCache plus normal authentication/identity/
logout/form guards. No environments, keys or memberships were written by the probe.
The follow-up probe `44819462d52de86fe8d40e3b278ec26ce48b942a` also passed on the
delivered target (`browser-e`), recording both users' usable displayed own commands/
fingerprints, three running views and genuine BFCache. Those displayed values match
original membership logins and independent operator public-host-key observations.
Native `ssh-keygen` independently matched fingerprints for all four roots.

All **seven existing memberships across four projects** then passed direct own-key
SSH identity and PTY checks from `linux-infra.dimensionlab.net`, using the unchanged
`tun8417` route via `169.254.84.2`. No keys/accounts were created or updated, no project
files were written by the probe commands, and no project lifecycle/routing action was
performed. This is infra reachability, not laptop/Tailnet proof. Native private Git
HTTP advertisement at the unchanged Forgejo origin passed using the real native Alice
credential, without retaining its body or exposing the credential.

Final checks preserved all original profile/key/project/membership/key-check rows,
four container/image/running-state identities, helper bytes, native Forgejo/proxy units
and credential/TLS bytes. Affected files and running backend match the export. The old
Soda listener is absent; a preserved old SSH forward may still bind locally but no
longer serves Soda. Native query-free request/OAuth logging was observed with zero
audited credential/state query lines. Normal authentication/expiry/logout left two
sessions/grants; the pre-login check had preserved all ten sessions/nine grant rows.
Fresh paired backup is also retained at `.artifacts/cutover-c007eb6/backup/`.

The scoped phase-6 exit passed. **92 Node tests, 38 Python build fixtures**, JS/Bash
syntax, documentation and whitespace checks passed. The VM web-tunnel wrapper now
advertises/forwards only native 24444 on future invocation, with a fake-SSH regression;
no running tunnel was changed. No new Go compilation/native build
was needed or performed in this cutover turn. This is an affected-component deployment
of `bdbce8e` artifacts plus recorded configuration, not a wholesale install of the
later probe/document revision. Full product/provider/aarch64 acceptance, console and
laptop routing remain separate. All old/fresh fixtures and failures are retained.

## Browser terminal plan

The next concrete item is planned in the existing
[Sodaspaces plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal), not
another roadmap. Candidate: local terminal renderer, same-origin authenticated
WebSocket and one fixed Unix-helper operation into an existing project account.
Native PTY/account/owned-process teardown proof comes before UI wiring; Podman client
exit is not assumed to terminate container exec. No private-key collection, automatic
join/start, host shell, project-image replacement or durable terminal sessions.

Inspected current API/session/helper/provisioning owners, upstream Podman v5.8.2
exec source, terminal package metadata/types and Go WebSocket/PTY documentation.
Research is retained in `.artifacts/research/terminal-plan-6206578/`; the initial
Podman manual URL failed, then the correct `.md.in` source was retrieved. No native
command, product test, build, dependency installation or deployed-state change ran.
Only planning documentation and link/whitespace checks changed; the terminal remains
unimplemented. Native helper changes and retained rollout need their applicable scope.

## Native terminal boundary source

This records the initial `10321ce` source checkpoint; approved native execution is
recorded in the following section.

Implemented the first source slice of the [terminal plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal):
fixed embedded project-local Python PTY launcher, bounded private Unix WebSocket
operation/client, immutable-container-ID/namespace checks, marker/account validation,
credential dropping, framing/backpressure, heartbeat expiry and owned-shell teardown.
Streaming does not hold the mutation lock or inherit the buffered RPC timeout. Helper
shutdown now cancels and waits for pending/hijacked streams. No public API, drawer
terminal, native service change or project-image modification was made.

Coder/websocket v1.8.15 was genuinely resolved, without other dependency upgrades;
its ISC license text is included in the already-bundled root NOTICE. Local full Go,
focused host/command races and **49 Python build tests** passed. The new process tests
exercise actual local PTYs, resize/Ctrl-C, EOF/final output, silent-peer expiry and
slow-consumer cleanup with an unprivileged clean test shell; credential dropping is
unit-tested, not native project proof. No host accounts were created. The opt-in
`TestInstalledTerminalBoundary` is authored but skipped without private native input;
it uses a temporary root-private helper, not an installed service replacement, and
records only its bounded account/TTY/explicit-close scope. Lost-helper, unrelated
SSH/workload preservation and full browser evidence remain separate required checks.

Evidence: `.artifacts/research/terminal-native-163ccf9/`. At this initial checkpoint no
VM/installed/helper service, retained project, key, membership, routing or provider
state changed. UI wiring was held until the actual native gate; the separately
approved proof below now closes that bounded gate, not browser delivery. Retained
`soda-test` rollout still needs separate approval and a current backup.

## Approved native terminal fixture proof

The user approved temporary root-private helper/PTY proof on the existing isolated
`soda-native-spaces-658f2af`, without installed service replacement/restart, project
lifecycle/account/key or routing changes. Preflight independently confirmed both
native accounts, marker IDs, groups/homes, the existing container/image and unchanged
installed helper/services/product rows. No terminal had launched at this observation.

Actual Podman is **5.8.4**, not the builder's 5.8.2. Native preflight caught two source
assumptions: Go templates require `.ID` inside `json`, and auto-created namespaces
are reported as `private`. Inspected exact upstream source and observed UID/GID maps
`0:1000000:262144`; the candidate now checks private mode plus actual shifted mappings,
not just a create-time mode name. No runtime configuration/capability was changed to
fit the check. Original failed preflight and reviewed inputs remain under
`.artifacts/terminal-vm-10321ce/` and guest `/var/lib/soda-terminal-10321ce/`.

**Bounded native gate passed.** Final native test binary from `fae1696`, with verified
builder/guest SHA256 equality, passed on existing Rocky project
`p4a530c394bcd53e563d3076d`: both original accounts' real/effective/saved UID/GID,
supplementary groups, HOME/cwd, real PTY, resize/Ctrl-C, current sudo boundary and
shared mise/project-Podman profile settings. A mismatched marker/actor was refused.
Explicit close and independent observations proved actual owned login termination.
EOF, a real silent 60-second lease and SIGKILL of only the test-owned temporary helper
also ended the login, foreground job and project-local launcher. Final observation
times were approximately 0.19s, 60.18s and 0.20s respectively—not merely socket-close
or host-CLI exit observations.

Both users' ordinary own-key SSH processes survived throughout (14 paired observations
in final `run-e`), over the existing management SSH direct-TCP forwarding path to
project `10.90.0.2`; no laptop/direct-builder-route claim. The earlier bridge client
was already stopped and was left untouched. Original preflight and `run-a` parser
failures remain; the latter reached a native PTY but did not recognize Bash CSI/CR
output. The probe parser, not the shell/profile, was corrected. Successful earlier
`run-b`–`run-d` observations and all new private inputs/binaries/results remain.

Final verification confirmed unchanged container/image/running identity, installed
helper bytes and affected native service PIDs/start times, product rows and native
marker/accounts/groups, SSH host key and DB integrity. No candidate helper/launcher
remained. No installed service replacement/restart, account/key/lifecycle/provider or
routing change occurred; ordinary shell/sudo bookkeeping was permitted, not claimed
absent. `soda-test` was not contacted. Full browser authorization/transport, logout
races, drawer/renderer/packaging and genuine browser proof are **still unimplemented**;
proceed with plan step 2, not a new architecture or installed rollout.

Final local regressions passed: full uncached Go suite, host/command race tests,
49 Python build tests, documentation links and whitespace checks. The default local
suite skips the explicitly opted-in native probe. Only the native package test
binaries were built/transferred; no whole-appliance build/stage/export, renderer
dependency installation or deployment ran during this proof.

## Protected browser terminal and independent drawer component

Implemented `internal/web/terminal.go`: exact-origin/query/subprotocol/fetch guards,
pre-upgrade session/own-membership checks, bounded first-message actor/repository/CSRF,
fresh acting-provider consent/visibility, original membership login, one pending/live
slot per login-context/project and bounded frame/queue/write/lifetime handling. Native
dispatch and lease renewal serialize with local logout/rotation; pending authorization
is cancelled too. Session reads now expose their existing minimum expiry internally
(no migration). Shutdown closes hijacked streams before DB shutdown. No browser
heartbeat, automatic join/start/reconnect or copied provider authority was added.

The self-contained `sodaspaces-terminal.js`/CSS component supplies explicit Open and
Disconnect, local lazy xterm/fit, bounded Unicode input/output, suppressed OSC clipboard/
link/title actions and full-page stale/reload behavior. It neither discovers nor edits
Forgejo markup. **The other agent owns all template overrides/layout**; the small
[mount/dispose contract](terminal-integration.md) is the only integration surface.
Existing hook templates, navigation and original drawer implementation were untouched.
Host template mounting remains intentionally unwired, not a second standalone UI.

Actual npm archive integrity and per-file hashes pin xterm 6.0.0/fit 0.11.0. Native
build/stage/bundle/first-install source now carries seven new component/distribution/
MIT-notice files, verifies upstream hashes and refuses occupied exact destinations.
This adds no bundler/CDN/runtime download or arbitrary-template adoption. Local
fetches wrote only ignored `.artifacts/browser-terminal/vendor`.

Local full Go, focused web/host/store/command/nativebuild races, **104 Node tests**
(including standalone DOM/renderer doubles) and **51 Python build tests** passed.
New server cases exercise malformed first auth, actor/association/CSRF/provider and
pre-upgrade denials with zero helper calls, original login, duplicate refusal, pending
and active logout/shutdown, OAuth rotation, logout during fresh authority, and bad
controls/browser-heartbeat refusal. DOM cases cover inert mounting, Unicode, scoped
keyboard handling, stale/late events, disposal and bounded renderer backlog. Packaging
tests exercise the real stage/preflight in synthetic temporary trees; they are not a
new real native-stage result. Evidence: `.artifacts/browser-terminal/`.

No VM/installed service, retained project, account/key/provider or routing action ran
in this source turn. No template override was changed, whole-appliance bundle built,
Chromium/native OAuth journey executed or deployment performed. Real combined
browser/proxy/helper lifecycle proof and native candidate delivery remain required;
prior native-only proof is not public endpoint acceptance. Continue the documented
mounting coordination and integrated proof, not a retained rollout or template fork.

## Minimum management controls — source implementation

Added restricted `/lifecycle` and `/access-keys` helper operations, protected public
routes and own saved-key deletion. Lifecycle validates the existing isolated container
and selected project unit before `systemctl enable/disable --now`, then verifies the
same container and actual running/boot-enabled state. **Start enables host-boot start;
Stop disables it and interrupts all project sessions/workloads.** No recreate, direct
Podman stop competing with systemd, desired-state DB copy or automatic repair. Current
project administration requires fresh authority; the configured Soda operator is a
separate permitted authority. Stop invalidates that project's browser terminals.

Own-key preview/apply always requires existing membership and fresh user/repository
consent, with no operator bypass or caller-selected login. Saved removal is own-user
SQL only and explicitly does not change existing access. Applying the reviewed set
checks both saved fingerprints and the exact native file revision; last-key removal
requires explicit confirmation. The fixed embedded Python reuses the existing marker/
account validator, locks the root-owned dedicated key directory, refuses unsafe paths/
files/noncanonical data and atomically replaces only that account's managed file.
It changes no groups, accounts, homes, unrelated files or authenticated sessions. No
project image/file installation or key propagation to other projects was added.
Canonical root edits in the dedicated managed file are visible in the complete preview;
explicit Apply confirms their inclusion/removal, not hidden drift repair. Noncanonical
annotations/options refuse, and edits after preview fail revision checking. Native
failure/partial results remain uncertain, not claimed as rollback or successful revoke.

`mountSodaspaces` in the new independent `sodaspaces-drawer.js`/CSS provides the complete
minimum content: OAuth connect/local logout, explicit create/join, own public keys and
removal/review/apply, confirmed Start/Stop, actual SSH details/Copy/Refresh and the
existing terminal component. It is inert until `.refresh()` and owns only its supplied
mount. **All Forgejo templates/layout remain with the other agent**; none were edited.
Use the [mount contract](terminal-integration.md), not both old/new callers in one drawer.
The historical caller stays unchanged for existing integration/evidence. Packaging
source includes the two new assets and occupied-destination refusal.

No schema migration or new dependency. Local tests exercise Go authorization/operation
boundaries, real atomic file writes in owned temporary directories with mocked root
metadata, and DOM/API/renderer doubles. These are not native lifecycle/key-possession
proof. Native Stop/Start, new/removed-key SSH, merged template/browser/proxy/helper,
whole-candidate native build/stage and installed delivery remain outstanding. No VM,
installed service, retained project/account/key/provider or route action was executed
in this source pass. Evidence and exact local check results are under
`.artifacts/management-72ce126/`: full Go tests and focused web/host/store/nativebuild
races passed; 114 Node tests and 56 Python build tests passed; document links,
installation-shell syntax and whitespace checks passed. The assembled embedded Python
was also exercised locally as an unprivileged refusal (no host account/file changes).
These checks did not run installed/native tests, SSH key possession or a full appliance
build/stage. Destruction remains an explicit unimplemented decision.

## Login design font assets

Downloaded the website's exact Fontsource 5.3.0 Latin WOFF2 selection into
`assets/branding/fonts/`: Fraunces variable 100–900, Barlow 400/600 and IBM Plex
Mono 400/500, all normal style. Added relative-URL font-face CSS, original family
OFL licenses and package/file provenance. Existing `assets/branding/theme/palette.css`
remains the shared color source, unchanged. These are source assets only; no
Forgejo template, running preview, native staging or deployment was changed.

Verified published archive SHA-512 integrity, font signatures, local CSS paths and
file SHA-256 values. No build, font-rendering/browser test or native validation ran.

## Local branded login preview

With user authorization, added `appliance/forgejo/templates/user/auth/signin.tmpl`
and the custom header CSS hook, plus `assets/branding/forgejo/login.css` and the
approved original papercraft PNG. The login shell uses the website's local fonts,
canonical logo and unchanged shared palette. Native `signin_inner`, head/footer
and scripts remain upstream-owned. This is the light login design; responsive CSS
hides the illustration below 900px. The Sodaspaces drawer remains unimplemented.

Recreated only `sodaos-local-forgejo` on Docker Desktop to bind source directories
read-only, retaining `sodaos-local-forgejo_data` and port 3300. An initial mount failed
because nested mountpoint directories were absent beneath a read-only parent;
created those empty local mountpoints and startup succeeded. The existing other
preview and appliance VM were untouched. No appliance stage/install changes.

Checks: login HTML and all sampled CSS/font/palette/logo/image URLs returned 200;
Alice's native form sign-in succeeded; wrong-password submission rendered the
native error inside the new shell. Image alpha data was verified. An exploratory
foreign-Origin rejection assertion failed (HTTP 200, also with cross-site fetch
metadata), so these probes do not establish CSRF protection; no native middleware
was changed. Browser automation was blocked by the user's password-manager panel;
the user inspected the preview and reported it looked good. Automated mobile,
keyboard, provider/passkey, account-link and CAPTCHA browser checks remain unrun.
Source whitespace checks passed. No full build, test suite or native acceptance.

## Login viewport correction

Removed Forgejo's inherited 80px wrapper bottom padding and first-section margin
on the login page. The flex layout now reserves the footer's actual height instead
of assuming a fixed footer size; artwork height and compact spacing adapt to shorter
viewports. Content may still scroll when genuinely taller than the available space.
Bumped the login CSS URL and reloaded templates only in the local preview.

Browser measurements confirmed document height and footer bottom equal viewport
height at 1654×970, 1366×768 and 390×844; mobile width was also exactly 390px.
Restored the browser viewport afterward. Initial measurements used cached CSS;
the versioned stylesheet loaded the correction. Whitespace checks passed.

## Login theme toggle

Added a single borderless sun/moon button at the top right, shared-palette dark
colors and the canonical dark logo. The guest preference follows system appearance
until explicitly selected, persists in origin/subpath-scoped localStorage, syncs
across tabs and tolerates blocked storage. A head script initializes appearance;
Forgejo's native theme attribute and authenticated account preference are unchanged.
No authentication/provider or appliance deployment changes.

Six Node state tests passed (system changes, explicit choice, toggle/persistence,
blocked storage, storage events and invalid/subpath values). Browser checks confirmed
system dark initial appearance, switching to light, correct next-action labels,
persistence after reload and no desktop vertical overflow in dark mode. The user's
existing “Soda dashboard” wording edit was preserved separately from this commit.

## Public homepage and texture removal

Removed the experimental paper texture asset and CSS references, restoring the
smooth login button. Added the native `home.tmpl` override and scoped `home.css`
for the public homepage: Soda welcome copy, approved papercraft artwork, sign-in
and repository exploration links, shared guest theme toggle and native footer.
The authenticated dashboard is unchanged. No account/authentication handlers,
provider configuration, appliance staging or deployed VM were changed.

Browser checks covered light/dark desktop appearance, shared theme on navigation
to login, native repository-explore and login destinations, and 390px mobile layout
with no horizontal overflow. Desktop homepage height matched the 970px viewport.
Verified login's computed background contains only its gradient, no texture.
Native public HTML/assets served successfully; whitespace checks passed. No full
build or native validation ran. Source is live-mounted only in the local preview.

## Remaining work and permission boundary

The remaining-work plan now has an explicit [minimum user-controls contract](sodaspaces-plan.md#minimum-end-to-end-user-controls),
not just an engineering task list. At the planning checkpoint saved-key handling was
add/list plus join-time installation; the source implementation above now adds removal
and later explicit project apply/revoke, with native validation still outstanding. The user-requested revision selects those bounded own-account actions
alongside Start/Stop, with real SSH verification; automatic synchronization, global
session revocation/offboarding and destructive execution remain outside that scope.
The deferral guide was narrowed accordingly. Template/layout ownership is unchanged.
That earlier planning pass ran documentation/link/whitespace checks only; it changed
no source behavior, dependencies, native target state or execution permissions. The
subsequent source pass and its local tests are recorded above.

- Preserve the implemented security, native-page context and explicit-action
  regressions plus steps 5–6's bounded native delivery/access evidence. Keep read-only
  guard mode separate from explicitly bounded writes; future maintenance needs its
  own exact scope/current backup, not replay of the recorded cutover.
- Finish native template mounting and genuine browser/proxy/helper proof for the
  already implemented terminal source; do not restart its completed native-boundary
  work. The template/layout agent owns the overrides and uses the component contract.
- Follow the single [ordered remaining-work list](sodaspaces-plan.md#remaining-work--ordered):
  terminal integration, minimum access/lifecycle controls (including explicit own-key
  apply/revoke and Start/Stop), the Destroy scope decision, runner settings, operator/
  client gaps, whole-candidate validation and approved delivery. Start/Stop and explicit
  own-key updates now have helper/API/independent content implementations; native
  lifecycle/key-possession validation and template mounting remain, and deletion is
  still deferred. Planning these is not authorization for lifecycle or
  destructive execution.
- Move Soda's local runner capacity/service configuration into operator-only settings
  in the unified native SodaOS/Forgejo interface, as subsequently selected by the
  user. Inspect official administrator extension points and reuse backing logic/tests;
  retain the Cockpit Runners page until a working replacement and coordinated removal.
  Tailnet stays in Cockpit; provider authority and the Soda operator boundary remain
  unchanged. This decision is documentation-only so far, not implementation/deployment.
- Finish full fresh/populated product and independent native aarch64 acceptance;
  scoped x86_64 delivery/browser/SSH results are not final-product acceptance.
- Complete console delivery/interactive proof, Tailnet and both providers' real
  runner journeys, intended-client routes, native branding and package/tool closure.
- Close [support-tool validation gaps](native-support.md#remaining-validation) and
  [actual-artifact licensing/source obligations](licensing.md). Optional media and
  incomplete outside helper ports are not product gates.

Standing implementation/testing approval now covers this planned work; local native
build/stage/export and isolated delivery testing proceeded under it. Preserve all
retained roots, credentials and evidence. It is not an instruction to erase data,
change unrelated provider/host-network resources or silently cut over `soda-test`.
Recorded routes and agents are not promises of liveness.

## Documentation history

The old M/U/P roadmaps, dashboard inventory, 179-group forge audit and detailed
native audit are removed from active documentation, not from Git. Their complete
text and the chronological 2,076-line handoff remain at `9f3baa7`, for example:
`git show 9f3baa7:docs/implementation-status.md`. The native audit's remaining checks
are condensed into the support guide, not declared resolved. Original source/
license findings and all private evidence survive; old milestone labels in tool
arguments/evidence remain valid identifiers, not active roadmap assignments.

This cleanup changes Markdown/links only. Performed documentation link/anchor,
retired-reference and diff-whitespace checks; no build, product test, dependency
resolution, generated provisioning, service/provider/network action or data cleanup.
Checks covered 55 Markdown files, 218 local relative links and 17 Markdown anchors
with no errors; six retired documents have no active references. Logs:
`.artifacts/research/docs-cleanup-9f3baa7/`.

Local PR fixture follow-up (2026-09-08): added seven user-requested PRs through
native APIs within alice/activity-workbench. Browser confirmed 8 open/2 closed,
review summaries and conflict indicator; API confirmed a native draft and
non-mergeable conflicting PR. Added a new generated collaboration image selected
only for Pull requests. No deployment or non-fixture repository changes.

Milestone artwork/fixture follow-up (2026-09-08): dedicated generated steps/flag
illustration now replaces the reused checklist image. Prompt and provenance:
`assets/branding/forgejo/milestones-art-prompt.md`. User-authorized native API writes
added ten milestones and thirty linked issues within the existing three local
fixture repositories. Browser confirmed 9 open / 2 closed milestones, 0/25/33/50/75/100%
progress examples, overdue/upcoming/no-deadline states and empty milestone content.
The ignored one-shot execution record is `.artifacts/local-forgejo/seed-milestone-fixtures.py`;
do not blindly rerun it. No non-fixture repository writes or deployment.

<!-- Illustration queue: personal Actions lists, four runner subpages and both owner storage overviews source-assessed without art; organization general and deletion pages source-assessed without extra art; labels and hooks also source-assessed without extra art; organization applications and Actions source-assessed without art; organization home and members source-assessed without art; team list/member/repository pages source-assessed without art; creation and invitations are next. See the per-page checklist. -->
