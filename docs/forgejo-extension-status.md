# Forgejo extensions — implementation status

**Active, separately owned workstream: native-feeling Forgejo extensions.**
The user assigned this work to the Forgejo extension agent; another agent owns
Tailnet. This is the single current status and implementation order for this
workstream. Do not maintain a second Forgejo task list in the Tailnet handoff.

The [source audit](forgejo-extension-audit.md) records findings at its baseline;
this file records what is being implemented and what remains. Requirements stay
in their existing feature guides, not in a new extension framework.

## Goal and scope

Improve the existing stock Forgejo / Go / Lit integration: correct admission of
Soda operations, coherent native navigation and presentation, and a maintainable
customization boundary. Preserve native Forgejo handlers/forms, Soda's separate
authorization, the shared workspace and persistent Project OS state.

**Not this workstream:** Tailnet implementation or enrollment; Cockpit retirement;
Runner OS, CI scheduling/cache work; installer or desktop roadmaps; forge replacement,
JVM dependencies, a maintained fork, a standalone frontend or a plugin SDK.
Atomic native/Soda logout and independent native Soda handlers are documented
architectural limits, not promised fixes in this queue.

## Current status

| Item | State |
| --- | --- |
| Source audit | Complete at `e415330`, inspecting Soda `5c2f92a` and stock Forgejo 15.0.7. |
| First implementation slice | **Complete locally: runner post-decode mutation admission.** Source fixed; focused Go/race checks passed. |
| Navigation | **Complete locally:** matching Spaces/Runners active styling and current-page accessibility cues; local template/browser/build checks passed. |
| Environment read publication | **Complete locally:** detail/members/connection recheck the original session before returning data; focused Go/race checks passed. |
| Localization | Deferred until the incoming redesign's component structure and wording settle. |
| Customization reduction | Conditional review after coordination with the redesign, not a blanket rewrite or release gate. |
| Native build / installed validation / delivery | Outside this arm64 macOS source session; the x86 machine handles deployment for the current setup. |

Runner, navigation and environment-read fixes have their own post-fix evidence,
separate from the audit's baseline checks. This agent continues source fixes and
local checks only, not appliance installation, service operations or deployment.

## Implementation order

### 1. Correct runner mutation admission — complete locally

`internal/web/runners.go` now checks the original session immediately before create
and lifecycle dispatch; early authorization is unchanged. The checked-in
`internal/web/runner_admission_test.go` covers all five mutations with unchanged,
logout, user/context/CSRF drift, storage failure and cancellation cases.

The [runner authority contract](runners-port.md#required-web-and-native-authority-path)
owns the admission—not rollback—limit. The
[execution receipt](forgejo-extension-history.md#runner-post-decode-mutation-admission)
records the reproduced failures and passing focused Go/race checks. No real runner,
shared authentication mechanism, host protocol, migration or Tailnet code changed.

### 2. Make navigation coherent — complete locally

The existing settings-link module now reads the validated native content host and
matching actor, applying native `active` styling and `aria-current="page"` to Spaces
or Runners. Repository settings retains its own context/back-links; neither global
link falsely claims it is that page. No second selector parser, client router or
permission source was introduced. Operator visibility and retirement are unchanged.

The [native page contract](forgejo-soda-pages-plan.md) owns the behavior. The paired
entry/import epoch is `2026-09-12.native-pages-8`; the canonical payload inventory
and shared Lit runtime remain unchanged. The [navigation receipt](forgejo-extension-history.md#native-navigation-current-page-cues)
records the failing-before/passing-after checks and local-only evidence limits.

### 3. Protect environment read publication — complete locally

The three direct-ID reads in `internal/web/environments_api.go` now use the existing
`requireCurrentSession` immediately before successful publication. Fresh-session
checks match original user/context/CSRF after I/O, while ordinary owner/operator,
degraded-member and own-connection rules remain intact. No shared authentication
helper, store/host protocol, browser component or redesign file changed.

The [API contract](dashboard-api.md#retained-operations) owns the boundary and its
non-atomic limits. The [read-publication receipt](forgejo-extension-history.md#environment-read-session-publication)
records the reproduced late responses and 56 passing regression cases. The user
selected this backend-only slice so visual work could proceed independently.

### 4. Align Soda presentation text with native localization — deferred

Coordinate with the ongoing `codex/forgejo-redesign` work before changing page
structure, shared components or text. Do not create a competing design pass or
localize a presentation being replaced. The existing navigation/accessibility
behavior remains a contract to preserve through the redesign.

Use the [customization](forgejo-frontend-integration.md#shared-presentation-components)
and [Lit](lit.md) boundaries for the existing Soda page titles, entry/status messages
and controls. Inspect the actual custom locale/template mechanism before passing
presentation strings to Lit. Keep native form handling and accessibility semantics.
No second translation framework, Tailnet UI ownership or unsupported claim that
all Forgejo languages have been translated follows from adding localization keys.

Record the exact strings/locales and affected browser checks with the source slice;
do not silently expand this into restyling every Forgejo page.

### 5. Reduce only demonstrated integration maintenance costs

First make the customization guide's current recipe unambiguous, leaving historical
evidence in history. For code/assets, select concrete affected overrides or stylesheet
loads only when a smaller native hook/partial or measured loading change preserves
behavior. The audit's template/CSS counts alone do not justify deleting overrides,
rebuilding the asset pipeline or claiming a speedup.

This is a conditional improvement after correctness and navigation, not a requirement
to rewrite all customization before shipping a bounded fix.

## Parallel-work boundary

- **This agent:** extension correctness and this status file; the current read
  correction is backend/tests/documentation only. Presentation integration follows
  coordination, not competing edits to the redesign.
- **Redesign agent:** current visual work in `codex/forgejo-redesign`, including
  templates, styles, fonts and payload inventory. Those files were left untouched
  by the read-publication correction.
- **Tailnet agent:** its feature, source state/migrations, enrollment, host/project
  controls, native proof and its own progress. Do not implement, reschedule or mark
  that work complete here.
- Shared files such as the page entry, dashboard template, settings-link module,
  locale assets and payload manifest are not exclusively reserved by this document.
  Check current changes and coordinate overlapping edits before touching them;
  preserve the other agent's additions rather than restoring an older file.

[The shared handoff](implementation-status.md) retains target custody, installed
state and permissions. Refer there before any retained-target action; do not copy
its fixture inventories or another workstream's grants into this queue. The current
assignment does not authorize deployment, service/VM lifecycle, real provider work,
network/trust changes or cleanup.

## Evidence and latest change

**Latest change — environment read publication:** all 56 regression cases passed,
within 14 selected top-level Go tests / 102 subcases, both ordinarily and with
`-race` on pinned Go 1.26.7. Before the fix, 46 refusal cases returned 200/data;
8 unchanged controls and 2 existing store/cancellation error refusals passed.
Evidence: [read-publication receipt](forgejo-extension-history.md#environment-read-session-publication)
and `.artifacts/environment-read-publication-HCb4VL/`.

The earlier [navigation](forgejo-extension-history.md#native-navigation-current-page-cues)
and [runner](forgejo-extension-history.md#runner-post-decode-mutation-admission)
receipts retain their separate evidence; the
[original audit](implementation-history.md#forgejo-extension-source-audit) remains
baseline research. No frontend build/test, native fixture login, installed validation,
provider action, redesign/Tailnet implementation or deployment occurred for this
backend-only correction. Presentation work remains deferred for coordination.
