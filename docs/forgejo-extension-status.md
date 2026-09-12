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
| First implementation slice | **Next: runner post-decode mutation admission.** Reproduced, not fixed. |
| Navigation and presentation | Queued below; no source changes yet. |
| Customization reduction | Conditional review, not a blanket rewrite or release gate. |
| Native build / installed validation / delivery | Not performed for these follow-up changes. No target selected. |

This status separation changes documentation only. It does not turn the passing
baseline checks into evidence that the runner defect is fixed.

## Implementation order

### 1. Correct runner mutation admission — next

**Scope:** `internal/web/runners.go` and focused runner handler tests, following the
[runner authority contract](runners-port.md#required-web-and-native-authority-path).

- Keep the configured-operator, actor, Origin/CSRF and provider checks before body
  decoding. Do not weaken that early gate.
- After decoding/validation, recheck the original session with the existing
  `requireCurrentSession` immediately before create/start/stop/restart/remove
  helper dispatch. No new authentication helper or operation scheduler is needed.
- Turn the audit reproduction into checked-in regressions using the existing
  real-router/store and synthetic-peer fixtures. Cover logout, changed actor,
  context/CSRF, store failure/cancellation and unchanged-session success.

**Completion:** the formerly failing logout cases refuse dispatch, unchanged valid
requests dispatch once, existing authorization/validation cases remain intact,
and affected Go/race checks pass. This is admission safety, not rollback of a
previously admitted operation. No real runner registration or lifecycle action is
needed to establish this source correction.

### 2. Make navigation coherent

**Scope:** existing Spaces/Runners navigation and their consumers, under the
[native page contract](forgejo-soda-pages-plan.md).

Add selected-view styling and `aria-current` from validated page context. Preserve
operator-only visibility, native links, invalid-selector refusal, ordinary dashboard
behavior and repository identity/back-links. Extend existing template and emitted
browser tests; keep asset inventory/cache-version changes paired where affected.
Do not invent a client router or manufacture repository-admin template context.

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

- **This agent:** the work above and this status file. Start with runner-specific
  handlers/tests; the first slice does not need changes to shared OAuth, migrations,
  the host protocol, page entry or payload manifests.
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

**Audit evidence, reused—not rerun by the status separation:**
35 selected top-level web tests, 7 template tests, 54 frontend/Forgejo checks and the
emitted asset build passed. The separate runner probe had five passing unchanged
controls and **five failing logout regression expectations**. Exact scope, versions
and commands remain in the [audit receipt](implementation-history.md#forgejo-extension-source-audit)
and `.artifacts/forgejo-native-audit-XdXOQq/`.

**Latest change:** created this dedicated implementation handoff and removed the
Forgejo audit/repair queue from the Tailnet-focused status. Documentation links and
whitespace checked; no production source, fixture or installed state changed.
Update this file in place after each Forgejo slice; keep the audit as baseline
research and the shared target record separate from this source work.
