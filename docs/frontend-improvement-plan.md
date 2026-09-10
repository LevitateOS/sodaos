# Frontend improvement: decision and implementation guide

Research consolidated on 10 September 2026 against `8e812dc`
(`feat(spaces): close local layout integration and journey ports`). This document
is the detailed handoff for agents assessing and implementing the frontend cleanup.
It records a researched recommendation and its acceptance requirements; the
documentation change itself implements no UI, dependency or checker changes.

**Recommendation: retain the current Go/Forgejo boundaries and shared Lit
workspace. Complete mandatory design-token consolidation, enforce template
diagnostics, extract readable typed views, and clarify source ownership.**

The [Lit implementation plan](lit-migration-plan.md) remains the owner of the
feature sequence. Steps 1–5 are locally implemented, including the layout and
journey closure in the reviewed commit. The cleanup described here fits after
that source baseline and before further UI expansion in step 6. It does not
repeat the rendering ports, replace the session model or mark step 6 complete.
If implementation has advanced since this review, preserve that work and apply
the same contracts to the current owners rather than restoring this checkout.

## Implementation progress after the reviewed baseline

The local implementation now covers canonical workspace tokens and computed
page/drawer presentation checks; required actual-source analysis and independent
negative fixtures; typed stateless environment/project/terminal/workspace views;
and [one authored Spaces source directory](../frontend/spaces/README.md), with
compiler/build/import/payload and production/journey callers ported together.
Owners still retain drafts, admission, original targets and live resources.
Observed unread/lifecycle attention (6a) and candidate/installed-driver source
coverage (6b) are locally complete. Required candidate checks include the new
six-session/two-project driver fixtures. **6c now has bounded x86_64 native proof**:
`b8af68c` passed the full terminal boundary and two-actor six-session matrix, with
independent named-End cleanup, and was delivered to both approved preserved targets.
Native browser/BFCache and retained account-access checks passed. Selected CLI/provider,
physical-keyboard and broader native/aarch64 acceptance remain unrun; this is not
whole-product acceptance.
See the [current handoff](implementation-status.md) for exact checks, failed
iterations and remaining proof, and [checker resolution](../tools/lit-check/README.md)
for the analysis-only compiler boundary and known event-parameter gap. Dated research
observations below remain historical, not current gate status.

## 1. Reading order and document ownership

Read this guide for the consolidated decision, evidence and cleanup requirements.
Then inspect the current owners and the relevant product contracts before editing.

| Document | Authority |
| --- | --- |
| [Architecture](architecture.md) and [Sodaspaces plan](sodaspaces-plan.md) | Product scope, upstream boundaries and overall order |
| [Current handoff](implementation-status.md) | What actually ran, on which source/target, and remaining proof |
| [Lit implementation plan](lit-migration-plan.md) | Completed feature contracts and remaining steps 6a–6c |
| This guide | Cleanup rationale, token/checker/composition work and acceptance criteria |
| [Lit authoring](lit.md) and [TypeScript development](typescript.md) | Runtime, compiler, dependency, build and authoring conventions |
| [Spaces design](spaces-design.md) and [drawer design](spaces-drawer-design.md) | Interaction, density, accessibility and page/drawer projections |
| [Terminal contract](terminal-integration.md) and [API guide](dashboard-api.md) | Exact session lifetime, commands, authorization and API semantics |
| [Forgejo integration](forgejo-frontend-integration.md) | Supported native templates/assets and compatibility responsibilities |
| [Project OS](project-os.md), [native support](native-support.md), [deferred work](deferred.md) | Preserved native owners, execution effects and excluded scope |

The detailed cleanup should not be copied into every feature guide. Keep links and
short applicable requirements there. Record actual results in the handoff; do not
turn historical passing counts into claims about a later candidate.

## 2. Decision and priorities

The problems are concrete: large compressed templates, inconsistent styling despite
existing tokens, confusing source names, and HTML bindings that ordinary strict
TypeScript does not check. Neither Lit nor browser rendering requires those problems.
The workspace already has substantial state, concurrency and terminal-lifetime
behavior which a renderer change would have to preserve.

Use these priorities when making implementation choices:

1. **Reliable behavior and authority.** Preserve working native Forgejo workflows,
   Go-side authorization, original-target actions, drafts and exact terminal
   resources. Keep the outstanding native/CLI compatibility question visible.
2. **Mandatory visual consistency.** Replace hardcoded visual styling with the
   existing canonical tokens across the full Spaces page and native drawer. This
   is a completion requirement, independent of renderer choice.
3. **Readable, checked authoring.** A maintainer should be able to identify the
   state owner, render a coherent section and reuse a wrapper without duplicating
   commands. Relevant template faults must fail the required check command.
4. **Limited rework.** Correct the demonstrated problems while preserving useful
   implementation and tests. Framework uniformity and the amount of HTML rendered
   by Go are subordinate to those outcomes.

### Alternatives considered

| Direction | Benefit | Cost or limitation | Decision |
| --- | --- | --- | --- |
| Retain Lit with typed composition and a dedicated analyzer | Preserves current resource owners; addresses styling and authoring directly | Separate analysis compiler and known event-typing gaps need ownership | Selected recommendation |
| React TSX for the Soda workspace | Compiler-native typed props, callbacks and children composition; React already exists in separate Cockpit | Port reactive state, synchronous guards, render/measurement timing, disposal, build and tests; no measured payback yet | Strongest alternative, not the current implementation task |
| Preact TSX | Similar JSX composition/checking with its own runtime model | Similar migration and another renderer practice; no measured bundle constraint selecting it | Not selected |
| Substantive Go-rendered management/forms with browser terminals | Useful for request/response pages or a real no-JavaScript management requirement | New form/fragment/data/draft contracts while substantial workspace state remains in the browser | Feasible, but not justified by the current problems |

React can mount inside the existing Soda island without taking over Forgejo. It
does not require a client router, JavaScript server or PatternFly. Conversely,
keeping Lit does not mean keeping giant templates or abandoning checks. Go can
render modular named templates, and typed server components such as `templ` are
possible for Soda-owned pages; they do not replace Forgejo's template engine or
solve live browser resource ownership.

The independent reviews disagreed. The architecture reviewer ultimately selected
retained Lit after the real-source analyzer experiment. The challenger ultimately
preferred React, assigning greater weight to long-term compiler integration and
future UI authoring. This guide adopts retained Lit because the current problems
have a demonstrated correction path and a complete renderer migration has no
measured benefit here. This is engineering judgment, not unanimous consensus,
proof that React is wrong, or a forecast of developer hours. No production bundle
size, performance or full migrated editing-cost comparison was performed.

## 3. Verified starting point

### Rendering and state owners

Paths below describe the reviewed source, not an instruction to keep every filename.

| Source | Actual responsibility | Cleanup implication |
| --- | --- | --- |
| [Spaces Go handler](../internal/web/spaces_page.go) and [template](../internal/web/templates/spaces.html) | Authorize and render the document shell, canonical assets, fixed navigation and labelled Soda account; mount the client workspace | Keep meaningful server authority. The 14-line template is not evidence of a broken architecture |
| [Page bootstrap](../frontend/spaces/sodaspaces-page.ts) | Validate the server actor, mount and refresh the shared workspace | Keep thin; no second page controller or copied session state |
| [Shared workspace](../frontend/spaces/sodaspaces-workspace.ts) | Both page and drawer; child handles, original bindings, locators, commands, layout projection, focus and storage effects | Historical drawer-only name obscures shared ownership; preserve one concrete owner |
| [Project controls](../frontend/spaces/sodaspaces-project.ts) | Environment/access views, key drafts, reads, explicit mutations and pending/uncertain state | First presentation extraction target; keep guards and drafts together |
| [Terminal](../frontend/spaces/sodaspaces-terminal.ts) | Controls plus imperative xterm/transport, attach/restore, input bounds, retention and disposal | Extract presentation without moving resource authority |
| [Layout](../frontend/spaces/sodaspaces-layout.ts) | Pure bounded parsing, migration, serialization and pane transformations | Keep ordinary TypeScript with explicit inputs; no DOM/network/storage effects |
| [API contracts](../frontend/spaces/sodaspaces-api.ts) | Bounded JSON reads and runtime validation from `unknown` | Reuse; renderer types cannot replace runtime validation |
| [Native adapter](../frontend/spaces/sodaspaces.ts) | Supported Forgejo hook, non-modal aside, native coexistence, width and document departure | Keep native navigation/forms and lazy loading intact |

The three main Lit modules are approximately 468, 282 and 361 lines in the reviewed
checkout. Compressed single-line templates make those counts a poor estimate of
complexity. Requests, identity, uncertain outcomes and resource lifetimes account
for much of the code. Formatting and extraction should expose those responsibilities,
not split them into arbitrary helpers merely to reduce line counts.

All three render in **light DOM**. Shared styles are available through ordinary
CSS scope; shadow isolation is not the cause of the styling inconsistency. A
shadow-DOM slot example is not automatically a safe wrapper replacement here.

### What the last commit completed

`8e812dc` closed local step-5 integration: bounded v2 layout/locators, stable measured
terminal owners, shared navigation/actions, compact Forge/Terminal visibility,
current installed-journey source ports and actual Go HTML/CSP to emitted-page
integration. The handoff records a 20-case measured layout fixture and broader local
checks. Those results are historical evidence, not checks rerun for this document.

Step 6 still owns observed unread/lifecycle attention, candidate coverage closure
and scoped native/selected-CLI proof. Existing synthetic peers, local native-form
fixtures and journey-driver tests do not prove stock Forgejo, real tmux processes,
physical keyboards or Codex CLI/Claude Code/Pi compatibility.

### Forgejo and Cockpit remain separate boundaries

The selected stock Forgejo version is 15.0.7. Exact-tag source confirms custom
template precedence and the `custom/extra_links` and `custom/extra_tabs` hooks.
These mechanisms do not register Soda Go handlers inside Forgejo or automatically
supply native authenticated navbar context to Soda's separate HTML response.
Official customization also does not guarantee compatibility across upgrades.

Keep Forgejo-owned forms, permissions, authentication and administrative workflows
in its supported native templates and handlers. No fork, custom executable, HTML
relay, iframe or borrowed session context is part of this cleanup. Reuse visual
roles and appropriate native partials; do not promise the same executable component
can be shared between Go templates and Lit. Existing split settings wrappers need
branch/structure coverage when changed; a Soda renderer migration would not fix them.

Cockpit retains its separate React/PatternFly pages, backing logic and tests. Its
React dependency is not already shipped in the Soda workspace. Planned Runners
settings do not establish a need to rewrite the current workspace, and must retain
their separate Soda-operator authority.

## 4. Executed checking research and limitations

These are dated observations, not additional dependency baselines. The manifests
and root lockfile remain the authority for installed versions. Research inspected
Bun 1.4.2, TypeScript 7.0.2 and Lit 3.3.3; React probes used Cockpit's React 18.3.1
and types 18.3.13, and Preact probes used 10.29.8.

| Executed probe | Observed result | What it establishes |
| --- | --- | --- |
| Actual browser configuration with repository TypeScript | Exit 0 | Ordinary browser TS checks passed during research; not the complete product suite |
| Deliberately invalid Lit fixture under TS7, also naming `ts-lit-plugin` | Exit 0 | `tsc` checks interpolated TS expressions but not their HTML binding positions; naming an editor plugin does not add CLI diagnostics |
| Typed Lit helper calls under TS7 | Incorrect boolean and callback arguments rejected | Ordinary typed function boundaries work for view composition |
| `lit-analyzer` 2.0.3 with repository TS7 | Crashed before analysis | This native compiler distribution lacks the classic JavaScript compiler API expected by the analyzer |
| Same analyzer with isolated TypeScript 5.9.3 | Positive fixture passed; negative fixture produced 10 diagnostics with unknown-event checking enabled | A separate pinned analysis path works for representative properties, booleans, event names/callability, directives and markup |
| Isolated analyzer on the three actual Lit modules | Seven diagnostics; analysis completed | Real imports/static properties are analyzable, but current source is not yet a green analyzer gate |
| ESLint Lit 2.3.1 with ESLint 9.39.2, scope-limited JS fixture | Four syntax/authoring errors caught; binding type errors missed | Complementary linting, not a template type checker or proof of TS lint integration |
| React/Preact TSX under TS7 | Typed native/custom property and handler mismatches rejected | Stronger contextual checking, with explicit custom-element declarations/wrappers |
| Invalid HTML nesting under React TSX | Accepted | JSX typing is not full HTML semantics, accessibility or browser behavior validation |

The analyzer's negative fixture included a number bound to the native string-valued
input property, a misspelled property, a string boolean binding, a non-callable event,
unknown native/custom events, bad custom properties, misplaced `repeat` and an
unclosed tag. It produced nine errors and one warning. `no-unknown-event` is
**disabled even in the strict preset** and must be explicitly enabled.

It missed callable handlers with incorrect event parameter kinds, including a
keyboard-event parameter for click and for a declared custom event. Typed helper
callbacks improve the function boundary; their internal template bindings still
need analysis and behavior coverage. Existing imperative listeners do not become
typed merely because surrounding markup changes framework.

The seven actual-source diagnostics were four widened ARIA strings, one widened
`draggable` string and two potentially undefined ordinary attribute values. They
are review items, not seven established runtime bugs. Lit 3's single-value ordinary
attribute commit uses an empty string for nullish values; the analyzer's older
wording is not reliable runtime evidence. Select a precise literal, empty value or
attribute omission according to the actual DOM contract; do not change an ARIA
string into a boolean-presence attribute merely to silence the diagnostic.

React/Preact custom-element checks required explicit JSX intrinsic declarations.
That declaration probe does not establish React 18 custom-element runtime behavior;
appropriate wrappers/event maps would still be necessary. A framework input prop
accepting a number also differs from directly assigning a native DOM string property.
Do not present those as identical checker contracts.

### Required analyzer integration

Keep the repository's product compiler. The demonstrated candidate is a narrowly
owned **development-only** analyzer package using the compatible classic compiler.
Integrate it through the root Bun workspace and **single root lockfile**. This
integration was not implemented or proven by the isolated experiment.

The implementation must establish all of the following:

- The analyzer resolves its own compatible `typescript` package; ordinary product
  checks still resolve the existing compiler. A package alias alone is insufficient
  because the analyzer uses bare `require('typescript')` internally.
- Installation from the locked dependency graph reproduces that resolution without
  manual `node_modules` edits, a product compiler downgrade, a private checker fork
  or dependency lifecycle hooks. Use Bun; do not add a separate Node requirement.
- `bun run typecheck` includes the actual-source analyzer as a required check while
  retaining all current compiler boundaries. A focused script may exist beneath
  that command; an optional editor setting is insufficient.
- Coverage follows every authored Lit source as files move. Empty input, skipped
  new modules, analyzer crash and missing tool are failures, not successful checks.
- Required rule severities fail the process, including faults otherwise reported
  as warnings. Explicitly enable unknown-event checking. Test negative cases
  individually so unrelated errors cannot conceal a silently ineffective rule.
- Retain a positive fixture using the production static-properties/`declare` pattern,
  nested typed templates and allowed directives. Check that normal authoring remains
  accepted instead of making a permanently red negative fixture part of product input.
- Document the event-parameter limitation and cover real pointer/keyboard/command
  behavior. Do not describe this toolchain as full TSX-equivalent checking.
- Keep analysis tooling out of emitted browser payloads and native runtime assets.
  Extend the existing build checks if needed to prove this boundary.

Fix the seven diagnostics individually and review the resulting DOM semantics.
Do not broadly disable rules, weaken strict TS, add blanket casts or change the
production reactive-property pattern to satisfy tool discovery. ESLint or future
Lit Labs tooling should be added only for a specific uncovered rule; the research
did not establish a newer production-ready replacement for this checker.

## 5. Mandatory token consolidation

### Canonical owners and scope

[The palette](../assets/branding/theme/palette.css) defines canonical light/dark
values. [Shared component CSS](../assets/branding/forgejo/components.css) defines
semantic roles, spacing, typography and control dimensions. Theme selection remains
with the existing adapters; shared values must not override a signed native page's
selected theme.

The current semantic declarations are inside a selector for full Soda pages which
also changes body/navbar/footer layout. **Separate token availability from those
full-page effects.** Make the same definitions available to both Soda surfaces
without adding a full-page marker inside the drawer or copying a synchronized set
of declarations. Verify imports, inheritance, computed `color-scheme` and selectors
on both actual entry points. Global value definitions alone do not activate a theme.

Audit all four workspace stylesheets—`sodaspaces-page.css`,
`sodaspaces-drawer.css`, `sodaspaces-terminal.css` and `sodaspaces.css`—plus visual
values in their TypeScript/templates and the shared presentation rules they use.
The page stylesheet currently contains shared workspace rules; filenames do not
define their true scope. Include raw fallbacks inside `var(...)`, inline style
values and terminal theme configuration in the audit. A fallback can hide a missing
token just as effectively as an ordinary literal.

The required scope is the shared Spaces page/drawer/control presentation and its
canonical definitions. Inspect affected native consumers whenever shared tokens
change. Do not claim every stylesheet or Forgejo override in the repository has
been audited; extend the inventory explicitly if the work expands.

### Map roles, not literal spellings

| Visual role | Existing semantic family or action |
| --- | --- |
| Canvas, cards/menus/dialogs, text, muted text, edges | `--soda-page-canvas`, `--soda-page-surface`, `--soda-page-text`, `--soda-page-muted`, `--soda-page-edge` |
| Links, active fill and action states | `--soda-page-link`, `--soda-page-selected`, existing action/hover/pressed/on-action roles |
| Focus | Existing light/dark focus values through a shared semantic binding |
| Warnings and destructive actions | Canonical warning and danger roles; preserve the distinction from selection/accent |
| Controls, buttons and panels | Existing control/button/panel radius, inset and height roles |
| Gaps, padding and margins | Existing `--soda-space-*` scale, after reviewing intent |
| Headings, body, labels, metadata and buttons | Existing `--soda-font-*` roles |
| Terminal monospace, screen colors and missing elevations | Search existing owners first; add a currently necessary semantic role once at the canonical owner |

Do not rename every raw number into a local variable or build a parallel palette.
Shared CSS itself contains some direct visual values; distinguish legitimate token
definitions from repeated consumer styling. A missing semantic role belongs at the
canonical owner with its actual light/dark behavior, not in each consuming component.

Workspace density can differ intentionally from ordinary forms. Existing controls
include 44px dimensions while workspace chrome uses smaller values. Blindly mapping
every toolbar button to the largest control token can consume the terminal area.
Define an intentional shared density variant when needed, preserving readable labels,
focus and usable hit regions. Pixel dimensions in design sheets express design
intent and geometry, not permission for independent hardcoded visual definitions.

Zero, percentages, measured terminal cells, minimum rows/columns, viewport/pane
coordinates, interaction hit regions, z-order and accessibility clipping are not
automatically token violations. Keep values expressing a layout/protocol contract
explicit at their owner. Changes to fonts, padding or chrome dimensions must carry
the existing measured-geometry tests; never shrink terminal text to conceal a failure.
The terminal's contrast/ANSI and monospace requirements also need deliberate roles,
rather than blindly inheriting page background and body typography.

### Enforcement and visual acceptance

Extend the existing frontend/Forgejo presentation checks with a scoped raw-visual-value
guard. Cover authored CSS and presentation sources, exclude generated/vendor material
and allow canonical definitions by ownership. Any necessary exceptions should be
narrow and explain a concrete geometry or interaction contract. Include a deliberately
bad sample to prove new raw visual colors/typography fail; expand spacing/radius/control
checks where the inventory can distinguish them reliably from measured geometry.

Source checks establish token usage. Computed styles establish resolved appearance;
they cannot tell whether the original declaration used a variable or a literal.
Verify resolved colors/fonts/dimensions against the canonical roles and inspect both
real surfaces in light/dark, compact/wide, long labels, open menus/dialogs, warnings,
disabled/busy/selected states and visible keyboard focus. Test native body/navbar,
forms and overlays outside the Soda mount for unintended changes.

Reuse the current page/layout fixtures and screenshot workflow. A new palette file,
token names wrapping the old literals, one screenshot or a passing text scan is not
completion. Token adoption must be visible in the actual controls and remain enforced
for subsequent changes.

## 6. Component composition and preserved owners

### First extraction: project controls

Start with coherent Environment, Access and status sections, then shared field/help/
error and action-row presentation where there are real repeated callers. Use typed
template functions for stateless composition. Create another reactive custom element
only when it owns independent state or lifetime; shorter markup alone does not require
another element, registration or lifecycle.

A typed helper can receive a `TemplateResult` body or a specific typed callback.
For example, the following illustrates the checked function boundary; it is not an
implemented new component or a promise that its internal HTML is checked by `tsc`:

```ts
import {html} from 'lit';
import type {TemplateResult} from 'lit';

function renderAction(
  label: string,
  disabled: boolean,
  onClick: (event: MouseEvent) => void,
): TemplateResult {
  return html`
    <button type="button" ?disabled=${disabled} @click=${onClick}>
      ${label}
    </button>
  `;
}
```

In this form, TS checks the helper's arguments and the analyzer checks its binding
positions. Passing a typed body to a field/section wrapper works similarly. Keep
types beside their owner and derive presentation from current state instead of
creating a second mutable view model. Shared wrappers need clear label/help/error
semantics and unique IDs across retained project instances.

Keep the current project owner responsible for context, drafts, requests, synchronous
busy admission, stale generations, uncertain outcomes and explicit mutations. Move
multi-step command bodies out of dense inline template expressions where useful,
but do not split authority among synchronized stores or invent a controller framework.
Distinguish metadata refresh from deliberate form reset; switching views must preserve
the existing unsent key draft and original target.

Format templates as readable nested markup. Do not hide the same giant template
behind an untyped function or replace it with raw `innerHTML`/`unsafeHTML`. HTML text
escaping, runtime data validation and authorization remain separate requirements.

### Extend to workspace and terminal presentation

After the project slice passes, apply the same approach to navigation, pane/tab chrome,
menus, chooser/confirmation and terminal controls. Keep direct owners and callers
visible. Small reusable presentational pieces are useful; a general component base
class, event bus, parameter bag or interchangeable-renderer layer is not needed.

Preserve these contracts throughout:

- One stable flat terminal-host layer, keyed by immutable local owner identity.
  Moving a terminal between independently keyed parent templates can disconnect it;
  `repeat` only preserves identity within its existing render part.
- Xterm exclusively owns descendants of its stable screen node. View/metadata updates
  do not replace that node, renderer or socket. Same-document continuity differs from
  document navigation, which preserves exact native IDs rather than DOM objects.
- Rendering is pure. Layout, selection, focus and metadata observation do not create,
  join, start, stop, apply keys, End or renew retention. Explicit commands and authorized
  restore keep their existing effect boundaries.
- Busy/epoch guards apply synchronously before the next render. Late imports, fetches,
  socket events, focus and geometry callbacks recheck their original binding and
  retirement; a disposed owner cannot launch or publish into a replacement target.
- Await the relevant child and actual geometry readiness. A parent's `updateComplete`
  neither awaits every child nor proves fonts/layout have settled.
- Disposal detaches document resources; it is not HTTP End. Preserve finite retention,
  explicit Return, one writer per ID, original-target End and exact cleanup receipts.
  Unknown outcome, accepted End and missing metadata are not confirmed cleanup.
- Presentation changes preserve pending/hidden/uncertain locators, failed storage and
  bounded v2 migration. No newest-session fallback, input replay or automatic repair.
- Native form nodes, unsent values, focus, beforeunload behavior and upstream overlays
  remain under native ownership. Scoped styles and the drawer must coexist with them.

Do not reopen the native terminal compatibility architecture as part of component
extraction. The product question and failed probes stay in their existing owners;
local rendering checks do not resolve them.

## 7. Source, build and template organization

Clarify the source tree only after establishing the first checked component slice.
Use a clear Soda workspace source location and names describing shared workspace,
project presentation, terminal resources, layout and native/page adapters. Avoid
creating empty directories or speculative abstractions for hypothetical consumers.
Whether shared branding stays in its existing source root should follow its actual
consumers; no second copy belongs in a new workspace directory.

The [browser build](../scripts/build-forgejo.ts) currently discovers `.ts` files
under the branding and appliance asset roots. It maps unique flat basenames to the
[production payload](../internal/nativebuild/forgejo-payload.json) and requires exact
inventory equality. Moving source therefore requires coordinated changes to:

1. Source discovery, compiler includes and test/fixture imports.
2. Payload source mappings and external module import paths.
3. Both public asset roots, existing public URLs and lazy entry-point behavior.
4. Production/preview staging and its inventory/byte checks.

Keep generated JavaScript in ignored build output. Preserve one locally staged Lit
runtime through [the existing export](../assets/branding/forgejo/lit.ts). Core Lit
and `lit/directives/repeat.js` are the currently supported mappings; introducing
another directive requires its export, mapping and tests together, not a separate
bundle or CDN. Source reorganization does not require public URL changes or a
framework migration. Asset subpath support is not proof of backend subpath deployment.

Keep native Go-template changes bounded. Reuse named partials with explicit inputs
where they actually help, and preserve native form actions/fields, CSRF, branch
conditions and installed-template compatibility. Review matched opening/closing
wrappers together. Existing presentation inventory/native-contract tests own these
checks; do not construct a replacement native component system inside Soda.

## 8. Implementation order and exits

These are cleanup slices within the current Lit sequence, not new product milestone
numbers or a replay of steps 1–5. Keep changes reviewable, and preserve unrelated
work if another agent has already continued the feature plan.

| Slice | Required work | Exit before expanding the slice |
| --- | --- | --- |
| Shared styling | Inventory visual literals, expose canonical tokens without full-page effects, replace consumers and define only necessary shared density/terminal roles | Source guard and both-surface theme/geometry checks prove actual token adoption |
| Checker and first typed view | Integrate the isolated analyzer under the root lock, enable required severities, resolve current diagnostics and extract one project section | Positive/negative checker contracts work; product compiler unchanged; existing project behavior preserved |
| Remaining presentation | Extract other project sections, workspace chrome and terminal controls with explicit typed inputs | No duplicate action/state owners; draft, race, focus and host/renderer/socket assertions preserved |
| Source clarity | Rename/move coherent source groups and update build, imports, compiler, payload and callers together | Actual emitted assets load from both entry points with one runtime and unchanged intended public URLs |
| Feature continuation | Resume remaining 6a attention and 6b candidate coverage on the improved shared owners | Cleanup requirements included in candidate checks; native/CLI acceptance remains 6c under its own scope |

Token work is mandatory throughout; it is not an optional final polish phase.
Establish the checker and one representative typed view before broad extraction.
Separating styling from lifecycle refactoring makes regressions attributable.
Do not block all product progress on speculative source reorganization or repeat
checks without a new change, failure or unresolved concern.

### Concrete conditions for revisiting the renderer

Stop the failing approach and report its demonstrated constraint if clean analyzer
integration requires a product compiler downgrade, manual dependency patching or
a maintained private fork; if required diagnostics cannot be resolved without broad
suppression; or if a concrete required composition contract forces duplicate state
or unsafe casts. A later explicit requirement for compiler-native JSX checking would
also materially change the choice.

Use that evidence to assess bounded React TSX as the strongest alternative. Do not
silently begin a partial migration, keep a permanent old/new renderer selector or
restart the architecture debate on every styling issue. Known callable-event gaps
are already part of this decision; document and test them rather than rediscovering
them as a surprise. No forecast of cheaper future maintenance substitutes for an
actual failed contract or new requirement.

## 9. Validation and completion

Use the current product-owned checks and author focused missing coverage. A failure
should explain which behavior or checker contract is missing. Tests that merely
assert the new helper names or mirror the extraction are not useful acceptance.

| Concern | Existing starting owner | Required observation |
| --- | --- | --- |
| Compiler/analyzer | Root `typecheck`, browser/tooling/test compiler boundaries and new focused checker fixtures | Correct compiler resolution, covered source inventory, positive pass and each required negative failure |
| Management/actions | [drawer controls](../tests/frontend/drawer-controls.test.ts), [workspace tests](../tests/frontend/workspace.test.ts) | Drafts, original actor/project, duplicate-click admission, denied/unavailable reads, stale/uncertain writes and explicit confirmations preserved |
| Resource lifetime | [terminal tests](../tests/frontend/terminal.test.ts), workspace tests | Same host/screen/renderer/socket through reactive/layout changes; late work cannot relaunch; detach and End remain distinct |
| Layout and appearance | [drawer layout](../tests/frontend/drawer-layout.test.ts), [pure layout](../tests/frontend/spaces-layout.test.ts), [Spaces page](../tests/frontend/spaces-page.test.ts) | Actual fitted cells, retained desired layout, both themes/surfaces, focus/overlays and token values; no new visual literals |
| Native presentation | [presentation inventory](../tests/forgejo/presentation/inventory.test.ts) and focused native contract/browser tests | Existing template callers/branches, native forms, unsent values and shell styling preserved |
| Emission/staging | [Lit build tests](../tests/forgejo/lit-build.test.ts), [runtime tests](../tests/forgejo/lit-runtime.test.ts), [payload tests](../tests/build/test_forgejo_payload.py) | Both asset roots, exact payload, one runtime, imports/licenses and analyzer exclusion |
| Installed-driver source | Existing [Sodaspaces](../tests/installed/sodaspaces.ts) and [management](../tests/installed/sodaspaces-management.ts) journeys and local driver fixtures | Changed selectors/controls ported without bypassing UI actions or weakening private target/action gates |

Current entry points include `bun run typecheck`, `bun run test:frontend`,
`bun run test:lit`, `bun run test:spaces-page`, `bun run test:layout` and
`bun run test:forgejo`; `bun run test` also retains Cockpit coverage. Use focused
commands during each slice and the required combined candidate checks at closure.
Go/race and packaging checks follow affected callers and the existing plan. Do not
run dependency resolution, builds or native journeys merely because a command is
listed in this document; apply the existing session's local testing authorization
and the target/action boundaries in the handoff without inventing new approval gates.

For local native-page screenshots, follow [screenshot capture](screenshot-capture.md)
and use the authorized fixture/profile. For isolated generated previews use the
existing builder's separate output options; do not overwrite a live mount as a
side effect of documentation or inspection. Preserve retained evidence/private inputs.

Native stock-Forgejo coexistence, six-session/two-project continuity, exact native
cleanup and selected CLI/browser comparison remain the existing step-6 acceptance
obligations. Their target/action scope, execution and evidence are separate from
the local checks above. A new framework, token scan, renderer identity test or
static design sheet proves none of those by itself.

The cleanup is source-complete only when the shared page and drawer consume the
canonical tokens, required template faults fail the standard workflow, components
have readable typed composition with preserved owners, and changed callers/build
paths have the appropriate passing results. Record residual analyzer limitations,
actual commands/results, source revision, retained failures and unrun native cases
in the handoff. Keep source-ready, scoped-native-validated and delivered distinct.

## 10. Research provenance and references

Three independent assignments supplied an architecture review, a decision challenge
and executed tooling evidence. The local research material is retained under
`.artifacts/frontend-architecture-review-20260910/`, including positive/negative
fixtures and isolated tool dependencies. It is ignored and may be absent from a
fresh checkout. This document therefore includes the relevant findings, limitations
and implementation contracts without requiring those files to understand the plan.
Promote only useful reviewed fixture sources/configuration into normal product-owned
tests during implementation; do not copy temporary dependencies or extra lockfiles.

The research ran isolated compiler/analyzer/linter experiments and the actual browser
TypeScript check. It did not implement the gate, port a complete renderer, measure
bundles, rerun full application/native acceptance or alter production dependencies.
Documentation verification for this consolidation is recorded separately in the
handoff. Existing step-5 results retain their original provenance.

Primary sources used in the review:

- [Lit expressions and nested templates](https://lit.dev/docs/templates/expressions/),
  [composition](https://lit.dev/docs/composition/component-composition/),
  [reactive properties](https://lit.dev/docs/components/properties/) and
  [lifecycle](https://lit.dev/docs/components/lifecycle/) describe the current
  composition/property/resource mechanics.
- [TypeScript language-service plugins](https://github.com/microsoft/TypeScript/wiki/Writing-a-Language-Service-Plugin)
  distinguish editor plugins from command-line compiler checks.
  The [native compiler API roadmap](https://github.com/microsoft/typescript-go/issues/4830)
  is time-sensitive context; the local compatibility experiments establish the
  reviewed package behavior.
- [lit-analyzer](https://github.com/runem/lit-analyzer) and
  [ESLint Lit](https://github.com/43081j/eslint-plugin-lit) document different rule
  scopes. The [Lit Labs analyzer](https://github.com/lit/lit/tree/main/packages/labs/analyzer)
  and [CLI](https://github.com/lit/lit/tree/main/packages/labs/cli) were not established
  as drop-in production template diagnostics by this review.
- [React TypeScript](https://react.dev/learn/typescript),
  [incremental integration](https://react.dev/learn/add-react-to-an-existing-project),
  [state identity](https://react.dev/learn/preserving-and-resetting-state) and
  [Preact TypeScript](https://preactjs.com/guide/v10/typescript/) inform the alternative
  analysis. Current documentation is not proof of the repository's older runtime.
- [Forgejo customization](https://forgejo.org/docs/latest/admin/advanced/customization/)
  describes the official extension mechanism. Exact selected-version
  [template lookup](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/templates/base.go),
  [navbar](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/base/head_navbar.tmpl),
  [repository header](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/repo/header.tmpl)
  and [web routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/web.go)
  were inspected rather than inferring upstream limitations from Soda's own shell.
- [Go html/template](https://pkg.go.dev/html/template) and
  [templ composition](https://templ.guide/syntax-and-usage/template-composition/)
  support the feasible server-rendering alternative; neither changes Forgejo's
  native renderer or removes browser-terminal lifetime requirements.
