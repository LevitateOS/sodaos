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
| Localization | Next queued slice; no localization changes yet. |
| Customization reduction | Conditional review, not a blanket rewrite or release gate. |
| Native build / installed validation / delivery | Outside this arm64 macOS source session; the x86 machine handles deployment for the current setup. |

Runner and navigation fixes have their own post-fix evidence, separate from the
audit's baseline checks. This agent continues source fixes and local checks only,
not appliance installation, service operations or deployment.

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

### 3. Align Soda presentation text with native localization

Use the [customization](forgejo-frontend-integration.md#shared-presentation-components)
and [Lit](lit.md) boundaries for the existing Soda page titles, entry/status messages
and controls. Inspect the actual custom locale/template mechanism before passing
presentation strings to Lit. Keep native form handling and accessibility semantics.
No second translation framework, Tailnet UI ownership or unsupported claim that
all Forgejo languages have been translated follows from adding localization keys.

Record the exact strings/locales and affected browser checks with the source slice;
do not silently expand this into restyling every Forgejo page.

### 4. Reduce only demonstrated integration maintenance costs

First make the customization guide's current recipe unambiguous, leaving historical
evidence in history. For code/assets, select concrete affected overrides or stylesheet
loads only when a smaller native hook/partial or measured loading change preserves
behavior. The audit's template/CSS counts alone do not justify deleting overrides,
rebuilding the asset pipeline or claiming a speedup.

This is a conditional improvement after correctness and navigation, not a requirement
to rewrite all customization before shipping a bounded fix.

## Parallel-work boundary

- **This agent:** the work above and this status file. The completed runner slice
  changed only runner handlers/tests and their documentation, not shared OAuth,
  migrations, the host protocol, page entry or payload manifests.
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

**Latest change — native navigation cues:** 7 Go template tests and 10 selected
frontend/Forgejo tests passed, including 25 navigation fixture scenarios. The emitted
asset build and full TypeScript/Lit checks passed. New checks reproduced both missing
active states and the missing stable Spaces target before the fix. Evidence:
[navigation receipt](forgejo-extension-history.md#native-navigation-current-page-cues)
and `.artifacts/forgejo-navigation-YFesQX/`.

The prior [runner correction](forgejo-extension-history.md#runner-post-decode-mutation-admission)
retains its 35 regression cases and focused Go/race evidence at
`.artifacts/runner-admission-fix-ut6yVW/`; it was not changed or retested by the UI slice.
The [original audit](implementation-history.md#forgejo-extension-source-audit) remains
baseline evidence. No native fixture login, installed validation, provider action,
Tailnet implementation or deployment occurred. Localization remains queued.
