# Visible experience and development qualification

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r42"></a>
## R42 — Finish the persisted factory read model and API

**Execution tasks:** [R42.1](../factory-remaining-tasks.md#r42-1), [R42.2](../factory-remaining-tasks.md#r42-2), [R42.3](../factory-remaining-tasks.md#r42-3), [R42.4](../factory-remaining-tasks.md#r42-4), [R42.5](../factory-remaining-tasks.md#r42-5), [R42.6](../factory-remaining-tasks.md#r42-6).

**Work:** Implement and integrate. **Owner lane:** C. **Accepted inputs:** [R10](01-boundaries.md#r10).

**Recorded baseline:** Extend canonical read model with workflow producers and persisted role/verb feed. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Extend the one existing read model for actual phases, plans/tasks/lanes/checkpoints, review/acceptance costs, blocked owner, reassessment/stall/repair and exact app verification. Use producer-owned records and native links.

**Ralph absorption addition:** Present R32's communication and the actual
procedure as durable role-authored feed history. Keep author/role/run/assignment,
communication verb, recorded time/revision, linked issue/PR/plan/finding and
reported versus confirmed outcome distinct. Retain post revisions and related
question/reply history with stable identities, bounded ordered paging and honest
missing/truncated/stale state. Reuse canonical evidence owners and existing
visibility/authority checks; history must survive reload and service restart.
Exact storage/post schema follows the
[feed design](../../design/factory-feed-and-ralph-integration.md), not an invented
second task database.

**Required work/feed/library projections:** Implement the
[full design](../factory-remaining-tasks.md#full-worker-design-integration) against
R32/R33/R34/R40 records, not a parallel task or decision database:

- [ ] Supply authorized prepared search/read projections for knowledge and
  decisions: applicability, provenance, exact revision, support/governance/
  validity, authority/application state, conflicts, reuse and supersession.
  Start with indexed/filterable retrieval; reads do not run a model or spend.
- [ ] Supply Library Admin inventory, reference/impact and maintenance-result
  projections, including consolidation/split mappings, removal/restoration and
  historical links. Preserve access for every source, snippet, count and cache;
  collection grants are separate from feed subscriptions and attempt lineage.
- [ ] Project questions, answers, incorporation, proposals, role drafts,
  coverage/dispositions, decisions, artifacts and changed prerequisites into
  the same feed/thread/inbox identities. Decision logging, actual authority and
  work application stay separate. Routine updates coalesce with visible history;
  consequential changes have their own posts, with stable newest-first ordering.
- [ ] Consume durable projection obligations with replayable identity. Expose
  recorded/applied state separately from delayed search/feed indexing. Deletion,
  access loss, supersession and concurrent updates cannot leak cached content,
  resurrect old state or require repeating the original mutation/native effect.
- [ ] Project the native issue work contract for the structured detail/editor,
  list and feed previews: editable metadata, accepted worker inputs, requested
  route, actual assignment, deliverable assessment and directional required-result
  relationships remain distinct. Include validation/freshness/access states;
  native closure alone cannot become accepted output or parent readiness.

Release each prepared projection to R31 reads and R43 views as its producer is
ready. Qualify a complete empty result versus incomplete/unavailable pages,
cross-project denial, out-of-order events, lost projection work and restart.
The [component catalog](../../design/factory-feed-components.md#13-component-data-and-mcp-relationship)
maps each presentation family to required source records and identifies missing
producer schemas; a generic summary or a fixture alone does not supply them.
Its [exception-routing contract](../../design/factory-feed-components.md#failure-component-routing)
defines deterministic compositions from role records, lifecycle/effect state,
source validity and access. Preserve simultaneous exceptions, exact affected
scope and stable identities through late/duplicate updates; a missing projection
cannot become a successful empty result or trigger a model call on read.

**Additional acceptance:** All eight verbs and their role-specific formal
results have real producer-to-store-to-API coverage. Reads spend nothing and do
not mutate the workflow. Cross-account/repository isolation, ordering/revision
consistency and missing evidence are tested; a request, report or accepted
submission is never mislabeled as an observed downstream effect.

**Accept only when:** Read routes are bounded, authorized, secret-free and spend nothing; submitted/pending is not verified success. Each new producer supplies persisted data and integration tests; no fixture-only DTO or parallel dashboard.

**Implementation notes:** First define the combined-query/cursor contract against existing `factory_journey_readback.go` Store/API records; this preparation is ready. Consume R32's post/comment/revision output to implement subscription preferences and the combined feed; do not replace canonical workflow records. Build one authorized combined subscribed-feed query across projects with newest posts first and a deterministic unique tie-break, bounded keyset paging and rechecked visibility. Thread/comment reads retain their own provenance. Then add each workflow producer's reported/applied/verified projection as its receipt becomes available; full qualification waits for the listed consumers, not initial query implementation. Test equal timestamps, concurrent insert/page continuation, cross-project/account denial, unsubscribe/access revocation and restart persistence in PG/API. Subscription never grants access; forbidden entries and counts must not leak. Expose comment-to-plan/issue result/failure links without claiming submitted work succeeded.

**Retained activity consumer:** [Bounded lifecycle actions](../../product/overview.md#bounded-lifecycle-actions) already own three local activity summaries and separately revisioned display preferences. `internal/factory/activity.go`, `internal/store/factory_activity.go`, `internal/web/api/factory_activity.go` and `frontend/spaces/sodaspaces-factory-activity.ts` implement them, with Store/API/browser source tests retained. After R13/R19/R26/R27 produce actual effects, carry these confirmed facts into the new read/UI composition rather than replacing them with agent reports. Acceptance: all three defaults remain enabled; only current owners/admins change the display revision; stale writes refuse; hiding preserves history, grants and execution; pending PR/merge or unconfirmed original issue closure cannot become success. R43 preserves the settings surface. Subscription defaults/filter interaction is a design choice within R42/R43; neither subscriptions nor display settings acquire execution authority or introduce outbound notifications.

**Principal source owners:** [internal/store/factory_journey_readback.go](../../../internal/store/factory_journey_readback.go), [internal/web/api/factory_journey_readback.go](../../../internal/web/api/factory_journey_readback.go), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go).

**Additional runtime/qualification gates:** [R13](02-prompt-to-pr.md#r13), [R17](02-prompt-to-pr.md#r17), [R24](03-review-merge-reassessment.md#r24), [R28](03-review-merge-reassessment.md#r28), [R29](03-review-merge-reassessment.md#r29), [R31](04-controls-repair-verification.md#r31), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R36](04-controls-repair-verification.md#r36), [R37](04-controls-repair-verification.md#r37), [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40). Source-only work cannot release these gates.

<a id="r43"></a>
## R43 — Complete Work/Spaces and current native frontend composition

**Execution tasks:** [R43.1](../factory-remaining-tasks.md#r43-1), [R43.2](../factory-remaining-tasks.md#r43-2), [R43.3](../factory-remaining-tasks.md#r43-3), [R43.4](../factory-remaining-tasks.md#r43-4), [R43.5](../factory-remaining-tasks.md#r43-5), [R43.6](../factory-remaining-tasks.md#r43-6), [R43.7](../factory-remaining-tasks.md#r43-7), [R43.8](../factory-remaining-tasks.md#r43-8), [R43.9](../factory-remaining-tasks.md#r43-9), [R43.10](../factory-remaining-tasks.md#r43-10), [R43.11](../factory-remaining-tasks.md#r43-11).

**Work:** Implement and qualify. **Owner lane:** C. **Accepted inputs:** [R42](05-experience-and-development.md#r42).

**Recorded baseline:** Fresh Lit Work/Spaces feed and changed native joins; reuse suitable independent Ralph code/assets, seven roles and stable reading/drafts. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Render actual producer/read-model states, links, actionable waits and admitted intervention in existing Work/Spaces. Reuse native ownership/lifecycle/navigation/terminal work and requalify changed joins, geometry, accessibility, assets and attribution.

**Full design implementation:** Build the [entire component catalog](../../design/factory-feed-components.md)
through the existing Lit owners: all seven production role families, Library
Admin, shared conversation/decision/knowledge components and actual Factory/Jev
events. Use the documented source-to-component and non-happy-path mappings;
there is no worker-facing widget-ping tool. Implement source contracts as needed
with R32/R33/R34/R40 and R42, preserving one canonical record per fact.

- [ ] Render the combined subscribed feed newest first, with clearly separated
  posts and avatar/name/role/project/time/status in an unboxed header outside
  each card. No default timeline spine or nested multi-role issue wrapper.
  Use medium-detail work posts consistent with the accepted visual direction;
  full applications/logs/evidence open separately, with no literal-tweet cap.
- [ ] Implement real questions/answers, comments/replies, safe artifacts,
  proposal/decision inspection and links to resulting plans/issues/effects.
  Preserve discussion drafts, reading position and explicit return-to-latest
  behavior. Submitted human comments invoke JEV assessment; selected new work
  reaches Product Lead LLM authoring and native issue creation. Show each stage
  and discussion-only/clarification/existing-work outcomes on the comment; a
  confirmed issue gets its own linked post. Reads and agent-generated replies
  do not invoke intake; normal Factory authority governs execution. Replies do
  not silently reorder the feed.
- [ ] Build library and decision search/browse/detail with project/component/
  kind/applicability/status filters, sources, examples, history, conflicts,
  replacements and usage. Authorized humans can propose corrections or apply
  permitted governance/maintenance actions against the same domain objects.
- [ ] Render Library Admin edits, consolidations, splits, removals/restorations,
  supersession, conflicts, affected work and maintenance results. Show proposed,
  refused, applied and projection-pending states accurately; a maintenance bot
  cannot author a new product policy through a cosmetic edit.
- [ ] Preserve every material exception in the default post: missing evidence,
  stale inputs, known blocker, diagnosis needed, incomplete coverage, disputed
  finding, failed regression, partial adoption, invalid fixture, conflicting
  decisions and access loss. Simultaneous problems retain their affected scope.
  Recovery follows the actual owner/receipt; no invented success or retry power.
- [ ] Implement the [structured native issue presentation](../../design/factory-feed-components.md#structured-issue-presentation):
  work summary, objective, deliverables/acceptance, workflow/current role,
  waiting-on/needed-by relationships, context/boundaries, results and existing
  discussion. Adapt detail to work type, including knowledge-only research;
  keep specialties separate from roles and native Open/Closed separate from
  Factory/result state. Native issue and Work detail use the same projection.
- [ ] Provide matching structured creation/editing and secondary raw access to
  the same native Soda block, preserving prose and drafts. Show inherited values,
  field errors, unsupported routes, concurrent edits and effects on accepted
  inputs; never rewrite an active worker's contract through a UI field. Add
  compact list/feed previews, with full issue detail reached through a link.
- [ ] Connect explicit issue adoption and reacceptance to the existing
  revision-bound acceptance preview and mutation in
  `internal/web/api/factory_issue_view.go` and `issue_acceptances.go`. Let a
  current code-write maintainer inspect the exact title/body, selected source
  comments and prerequisite outcomes, then record their authorized acceptance
  and inspect its result. Distinguish a role reply, comment-intake clarification,
  Project setup acceptance and issue-requirement acceptance. The current Work
  detail displays acceptance status but does not provide this complete human
  action. Cover imported work, changed requirements, selected answers and
  non-code prerequisite confirmation, including stale preview, authority loss
  and uncertain-response readback. This completes R43.8/R43.9's existing
  ownership and unblocks the [HB12 reader walkthrough](../public-handbook/cutover.md#hb12-follow-the-daily-author-and-contributor-journeys);
  do not substitute private API calls for the public procedure or add approval
  to qualifying native creation and validated generated work.

Use focused producer-backed browser cases for the changed interactions, including
keyboard/focus, narrow/wide and light/dark layouts, reload, account change,
late updates and inaccessible artifacts. Reuse renderer primitives and existing
valid checks rather than creating a separate implementation/test per catalog row.
For the issue surfaces, cover a normal implementation issue and a research-only
prerequisite, structured/raw round-trip, a changed work type, invalid metadata,
an active-input change, a concurrent edit and an accepted negative result that
does not release its parent. Include narrow layout and keyboard operation.

**Ralph absorption addition:** Absorb Ralph's useful capabilities and interaction
behavior into a fresh Lit/TypeScript interface in this same Work/Spaces experience.
The owner explicitly excluded React integration; Ralph's existing appearance
does not constrain Soda's design. Reuse suitable framework-independent code and
assets individually. First map
the complete intended procedure to role, permitted MCP action, feed component,
evidence condition and concrete reuse source. Use one actual avatar/typed
question/reply/history flow to define the new Lit components, their Soda
appearance and host/resource lifetime behavior. Preserve the seven established personas as
production authors and add Library Admin's distinct attribution; select character/artwork assignments explicitly, with technical
specialties as a separate possible dimension. Include structured components for
direction, dependencies, investigation, task lanes/checkpoints, review/correction,
JEV decisions, waits, reassessment/repair and exact app verification. Reuse
Ralph's question, finding, knowledge, proposal and progress presentations and the
marketing film's artifact patterns where they fit real recorded work. Keep one
persistent Workspace owner and the existing terminal/resource lifetimes.

**Additional acceptance:** Real persisted role/verb entries, questions/replies,
evidence links and result confirmations work through both native entry orders,
navigation/reload/account changes and narrow/wide light/dark views. New updates
preserve reading position, expanded entries and drafts. Review/fix/merge and app
verification remain distinguishable. Required keyboard/focus behavior,
accessibility and any reused asset attribution survive integration. The source
reuse map accounts for retained independent code/assets, behavior implemented
in fresh Lit components and demonstrated new work. No React runtime or React
component integration is introduced.

**Role component design:** use the
[full custom component catalog](../../design/factory-feed-components.md) for the
seven distinct role compositions, shared communication/knowledge components and
Factory/Jev events. Its compact/expanded views, allowed actions, canonical state
sources and incomplete/stale/access-lost behavior are required R43 implementation scope.
Include the shared Recorded Decision component and proposed
[Library Admin components](../../design/factory-feed-components.md#library-admin-components):
item edits, consolidation/split mappings, removal/restoration, supersession,
conflicts and affected work. Bind these to canonical decision/maintenance records
from R32 and authorized R42 projections. Logged choice, actual decision authority,
applied workflow effects and search-index freshness remain distinct.
Apply the [work-post presentation direction](../../design/factory-feed-components.md#work-post-presentation):
give the message, role-specific visual, state and relevant interaction enough
space to explain the work. The owner excludes both literal tweets and entire
embedded applications; no fixed post count or tiny-preview cap is implied.
Full workspaces and long evidence open separately; failure/pending state remains
visible in the post.
Reuse Lit renderer primitives while retaining different outcome, finding, task
and verification semantics; this is not one implementation class or MCP endpoint
per catalog row. Finalize producer schemas with R32/R42 and the relevant role
consumers before claiming live component acceptance.
Use the catalog's [per-role exception matrices](../../design/factory-feed-components.md#18-non-happy-path-compositions-for-each-role)
and [combined failures](../../design/factory-feed-components.md#19-knowledge-transport-and-combined-failures)
for concrete component selection and recovery. Qualify the
[routing acceptance cases](../../design/factory-feed-components.md#20-acceptance-of-the-routing-design)
with producer-backed records: required exceptions survive compact views,
unknown/stale evidence cannot imply success, system events retain system
authorship, and visible controls obey current authority and revision checks.

**Accept only when:** Real persisted API-to-browser journeys pass both native entry orders, page/panel/history/reload/account changes and mandatory terminal/renderer/socket assertions. No injected links, absent-renderer success or reopening all 57 NFX tasks as new implementation.

**Implementation notes:** First select a concrete Ralph avatar/question/comment flow and suitable framework-independent assets with provenance, then design it in the existing Lit Work/Spaces owners (`sodaspaces-factory-work-journey-view.ts`, `sodaspaces-workspace.ts`). Project-only feed composition is superseded: the main view combines subscribed projects/feeds, newest posts on top. After R32/R42 supply actual records, implement subscriptions, persisted comment/reply composition, source discussion links and visible agent plan/issue outcomes. Then cover seven-role workflow cards and native joins. Use focused browser tests for reading position, drafts, keyboard/focus, account switch, late updates and pagination, plus changed native entry/terminal lifetime assertions. R03 gates native acceptance; no React integration, copied mock-only feed or reopening all NFX work.

**Principal source owners:** [frontend/spaces/soda-extension.ts](../../../frontend/spaces/soda-extension.ts), [frontend/spaces/sodaspaces-terminal-attachment.ts](../../../frontend/spaces/sodaspaces-terminal-attachment.ts), [Native frontend acceptance](../testing.md#native-frontend-composition).

**Additional runtime/qualification gates:** [R03](01-boundaries.md#r03). Source-only work cannot release these gates.

<a id="r44"></a>
## R44 — Requalify retained Project, provider, human and operator foundations

**Execution tasks:** [R44.1](../factory-remaining-tasks.md#r44-1), [R44.2](../factory-remaining-tasks.md#r44-2).

**Work:** Qualify and repair demonstrated gaps. **Owner lane:** A. **Accepted inputs:** [R03](01-boundaries.md#r03), [R10](01-boundaries.md#r10).

**Recorded baseline:** Current Project preparation/provider/broker/human/operator qualification. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Qualify current Project preparation, protected service identities/Unix mappings, broker custody, provider grants and human SSH/terminal/operator foundations before live factory dispatch. Inspect and repair demonstrated joins only. R41 later composes factory-specific intervention with these accepted foundations.

**Handbook integration finding — native enrollment recipe:** Complete and
execute the existing explicit enrollment recipe before handing it to public
Factory authorization. The manifest example in
[`soda-setup`](../../../cmd/soda-setup/README.md#explicit-factory-enrollment)
sets `identity.database_dsn_file` to `/etc/soda/postgres/soda.dsn`, but
[`validate_target_and_identity`](../../../cmd/soda-setup/src/factory_enrollment.rs)
requires a broker-owned UID 2001/GID 2000 file without group/other access.
[`provision_postgres_secrets`](../../../cmd/soda-setup/src/secrets.rs)
creates that dashboard DSN as root/GID 2000 mode 0640. The example therefore
fails the normal fresh-install ownership/mode checks. Supply the supported
credential preparation and custody steps without weakening those checks or
changing the shared dashboard DSN's ownership. Correct the command README and
[operator guide](../../guides/operator-setup.md#explicit-factory-enrollment),
then exercise first enrollment, matching maintenance and partial native
binding/hook recovery with safe native accounts. Record actual service and
binding readback; browser activation alone is not enrollment. HB12 owns the
reader walkthrough and F03 owns the resulting public procedure. This is a
repair within R44's operator/broker foundation, not another enrollment system.

<a id="project-access-revocation"></a>
**Selected product addition — explicit member revocation (R44.1, audit Q2):**
Implement **Revoke Project access** in the existing Project member view, Go
authorization/API and Rust Project host owner. Only the configured operator or
current native repository owner (including organization-owner authority) may
invoke it; code-write access or repository-admin status alone is insufficient.
This is targeted manual offboarding; automatic cross-system synchronization
remains deferred.

- Preview the stable member identity, Project, installed keys and affected
  sessions/processes, then confirm. Persist an admission hold before native
  effects so a still-valid write grant cannot immediately Join or attach again.
- Remove that member's installed SSH keys, disable native login and terminate
  their managed terminals, native SSH sessions and Project-local processes.
  Preserve their account identity/files, other members and independently owned
  service identities. Preview any service running under the revoked member's
  own UID as affected; do not promise it survives.
- Keep an inspectable operation and per-effect result across restart or lost
  replies. Partial/unknown termination stays pending or failed, with re-entry
  blocked; a successful database write alone cannot report revocation complete.
  A stopped or unreachable Project must not restart accepting that login before
  pending revocation is applied.
- **Restore Project access** explicitly clears the hold under the same authority;
  the member must then Join with current native code-write permission and
  explicitly selected keys. Retain the stable Linux identity and existing data.
- Explain separately how native Forgejo access is removed. This action does not
  revoke copied credentials, change repository permissions or establish an
  untrusted-code security boundary.

Accept through a real two-member Project: revoke a member with browser and SSH
sessions, prove denied reconnect/Join and stopped target processes, retain the
other member and files, then restore and explicitly rejoin. Include stale
authority, lost reply and stopped-Project recovery. Update the existing
Project/trust/API owners and handbook access guide with the implemented
interface. The implementation progress below records the exercised scope.

**Implemented and installed component progress, 23 October:** The owner/operator
Project Access member controls, revisioned durable hold, native keys/login/process
effects and explicit Restore/Join path are implemented. A real two-member retained
Project passed running and stopped-root revoke/restore (revisions1–4), existing
SSH retirement, new SSH/Join denial while held, stable UID/files after explicit
Join, and preservation of the other member's processes/files and native Forgejo
access. A pending member fence prevented root-systemd startup. Existing real
PostgreSQL/API checks cover restart/lost-response retry, stale authority and
independent membership. The complete mutation lifetime is serialized so an older
completion cannot remove a newer startup fence. The canonical service contains
the fence inline. This development VM uses an additional service drop-in, which
strict normal lifecycle admission deliberately refuses; matching candidate-unit
lifecycle qualification remains open. Production unit validation was preserved.

**Immediate implementation — default Providers dialog (R44.2):** Implement the
[provider UX](../../design/factory-feed-and-ralph-integration.md#factory-providers-dialog)
inside Soda Forge: supported provider rows with a hollow circle or green check
in the same row; clicking an unconnected row programmatically starts login and
shows the link/code. Follow completion automatically, retain the connection after
reload and support reconnect, expiry, retry and resuming a pending dialog. Use
the existing enrollment/connection APIs, `frontend/spaces/soda-identity.ts`,
`internal/web/api/identity.go` and the single broker. Expose Providers
from Factory and its missing-provider state. Real provider authorization remains
the user's action on the provider page; starting it and obtaining the code must
not require an agent or shell. This work can start from the existing endpoints
without waiting for the other R44 qualification cases. Accept through a real
unconnected-to-connected installed-browser journey; the manually enrolled VM
account is not acceptance of this UI.

**Development progress, 19 October:** `frontend/spaces/soda-providers.ts` now
implements the dialog and is mounted in Work and Factory settings. The local VM
serves it; the installed browser confirmed saved Codex status and an actual Muse
login start returning its verification link/code. The browser test covers
completion polling, close/reopen without another attempt, expiry/retry,
cancellation and retained connection readback. A new successful human provider
authorization through this dialog is still required to close R44.2 fully.

**Native provider continuation, 23 October:** A source-schema-compatible broker
was configured on the retained VM, and human-approved native Muse enrollment and
reconnect retained a ready connection. Actual MSP initialize/session/catalog and
graceful retirement passed normal credential custody with zero retained leases.
This exposed and repaired real serve-argument, socket-retirement and duplicate
stop behavior. No credential was extracted or manually copied to set up the
fixture, and no connection state was reset through SQL.
The full dialog-led current candidate witness and Factory child execution remain
separate R44/R48/R49 consumers.

**Shared Filetree/library consumers:** Retain completed identity, SQL, HTTP/WS
and library cuts. The remaining L08/LA-G4 and Q1/Q2/Q3/Q9 witnesses use actual
admitted DSN/auth/TLS, transaction deadlines/cancellation, native Unix peer and
installed service identities through current Soda Identity/setup/host callers.
Start at `cmd/soda-identity/tests/http.rs`, `cmd/soda-setup/src/tests/postgres.rs`,
`lib/host/src/project/executor.rs` and the existing host websocket tests. Select
only unproved shipping/provider joins; package tests do not establish installed
behavior. R48 owns funded provider responses, and R52 binds unchanged native
proof to actual shipped identities where required.

**Accept only when:** Required setup is approved and inspectable; wrong peer/account/repository/grant and rotated credentials refuse. Background authority survives browser logout, native/operator identities remain separate, and existing human services remain usable. The new factory intervention/retirement interaction is accepted under R41 and the complete installed journey under R49; it is not a prerequisite for establishing this foundation. Preserve retained identity/tooling work and its current input bindings.

**Implementation notes:** Consume R03's native background authority and R10's scope binding. Start in existing `internal/project`, `lib/host/src/project/admission.rs` and `cmd/soda-identity` preparation/broker handlers; trace one approved preparation and account lease through launch/return. Qualify wrong peer/repository/grant, rotation/logout and human SSH/operator coexistence using retained source checks plus native cases. Release current preparation and credential custody to R11/R12. V03 normal managed Muse admission requires human device-account approval; TestOnly evidence does not close it. That input holds the normal-account case, not every local Project check, and R41 remains the later factory intervention composition.

**19 October local development progress:** The retained test Project now has accepted Todo source inputs and ready native coding/review preparations; both setup and check exited 0, with four actual Canlang examples passing. Expose the bundled Node executable in the Project recipe. The helper now refreshes missing-tool evidence only for an unstarted waiting preparation, allowing the same identity to resume; complete or launched evidence stays frozen. Three focused Rust record tests pass, and the fixed helper is installed only in the test Project. This does not establish provider-backed LLM execution or close R44.

**Native model execution follow-up:** Root-hosted Factory must select the system manager for launch, observation and retirement; the root user manager ends with login sessions. The retained host is updated and 14 focused Rust tests pass. Real Codex retries completed with delegated credentials returned, but their task reports were refused before issue creation. The staged decompose prompt now names all canonical task fields and decision-ID key syntax, retains its generated result fence, and passes a regression through strict parsing and semantic admission. The owner-approved single further trial was submitted once, but JEV linked the title-change request to the older HELLO issue #1; authoring did not run. Existing-work routing currently receives the latest worker summary (“stopped” for #1), not the actual issue objective. Supply objective evidence separately from execution status before evaluating relevance; preserve the existing recorded judgment. The one-trial approval is consumed. Live acceptance of the corrected prompt remains pending; no parent is closed by successful CLI exit alone.

**Provider-specific custody qualification:** The [credential contract](../../reference/credentials.md#authentication-and-billing-choice) and its Codex/Muse custody sections distinguish one Codex stream across enrollment/human/Factory use from concurrent immutable Muse leases. Defining owners are `cmd/soda-identity/src/providers/{codex,muse}`, the broker lease/connection Store, and `lib/host/src/terminal/factory/{codex,muse}`; Muse’s `commands.rs` already removes the inherited key override. With admitted pin/configuration identities from R07’s preparation (not its full qualification) and R03/R10 native binding, next trace actual connection → launch → return/revoke under the selected account. Acceptance: Codex cannot overlap a human lease or restore a stale seed after uncertain return; ending/withdrawing one Muse execution preserves unrelated leases, while disconnect denies admission and retires all; unknown retirement blocks reuse. Expiry/quota/invalid auth and conflicting environment/configuration cannot fall back to usage billing, another account/provider or auth method. Reuse source custody tests and accepted exact-pin Muse concurrency/fresh-container evidence; repeat only changed or unproved joins. R48 supplies only the missing bounded actual-provider observations, including normal Muse approval. A synthetic credential or successful enrollment alone cannot qualify these consumers.

**Principal source owners:** [internal/project](../../../internal/project), [lib/host/src/project](../../../lib/host/src/project), [cmd/soda-identity](../../../cmd/soda-identity), [docs/guides/develop.md](../../../docs/guides/develop.md), [docs/reference/credentials.md](../../../docs/reference/credentials.md).

<a id="r45"></a>
## R45 — Qualify current release-component composition and maintenance

**Execution tasks:** [R45.1](../factory-remaining-tasks.md#r45-1).

**Work:** Qualify and repair demonstrated gaps. **Owner lane:** C. **Accepted inputs:** [R03](01-boundaries.md#r03), [R06](01-boundaries.md#r06), [R44](05-experience-and-development.md#r44).

**Recorded baseline:** Current component/package/controller/maintenance qualification; scoped RCX development acceptance retained. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind Payload v4, seven separate required image roles, including canlang-runtime, thin host/outer Docker, host-only compression/official-image compatibility, Fountain-owned lifecycle/durable intent and Soda package/controller declarations. Verify protected locks, startup ordering, closed admission, offline preparation and enabled-only maintenance.

**Shared Filetree/library consumers:** Carry C/D/H and L10/L17 native residuals
through the real component/installed owner: shipped curl/Bash/Podman and image/RPM
identity, resolver-inclusive bounded transport/TLS failure, and generated unit/
configuration identity. Reuse accepted source profiles and Butane/OCI/parser
fixtures. The native filesystem confinement cases start at
`lib/soda-release-deliver/src/buildx/filesystem.rs`,
`cmd/soda-forgejo-domain/src/marker.rs` and `lib/host/src/tailnet/files/mod.rs`;
actual supported-kernel openat2 behavior, symlink/escape refusal and preserved
prior data need native observations, not another parser conversion. R46 consumes
network-specific results; R52 supplies final shipping-byte witnesses. These are
aliases of existing gates, not extra prerequisite projects.

**Accept only when:** Current component/package/source identities and authenticated start/stop/snapshot/rollback readback match. Failure preserves prior valid bytes/data and stays closed when retirement is uncertain. Reuse accepted RCX source; no obsolete-plan restoration or replacement component architecture.

**Implementation notes:** Consume R03 package authority, R06 importer retirement and R44 preparation. Start at `lib/release-inputs/src/payload.rs::Role`, release maintenance and existing component drivers. Bind Payload v4's seven required image roles including canlang-runtime; qualify current package/controller enablement, offline preparation and same-input maintenance/rollback. Release exact component/lifecycle observations to R46/R51; direct-workerd replacement adds its own P27/P28 producer outputs before shipping. Existing RCX source/development evidence remains accepted. Missing/invalid bytes must preserve prior data and keep admission closed; test changed native joins rather than rebuilding the component architecture.

**Principal source owners:** [lib/release-inputs/src/payload.rs](../../../lib/release-inputs/src/payload.rs), [cmd/soda-project-maintain/src/release_extensions.rs](../../../cmd/soda-project-maintain/src/release_extensions.rs), [tools/release-assets](../../../tools/release-assets).

**HB14 operator handoff findings (20 October 2026):**

- `preview_operation_content` skips prefetch for `release --check`, then
  requires both selected target and retained source package/image content.
  A downloaded release alone therefore need not admit the check. Complete the
  supported preparation route within existing release maintenance and qualify
  a target whose content is initially absent; the inspected command interface
  exposes no separate preparation operation. Do not advise bypassing the check
  or manually populating protected caches. The operator guide and O05 now
  describe the actual preflight effects.
- Reconciliation without `operation.json` checks the current installed payload
  and administrator intent; it does not replay `last-operation` or independently
  prove a requested historical transition. O05/O08 now distinguish this from
  extension attention after recorded core completion. R45/R52 still owe an
  exact qualified source/target pair and its transition, interrupted-recovery
  and stopped-Project receipts for the handbook walkthrough.
- Read-only inspection of the retained development VM found
  `soda-postgres-backup.timer` disabled/inactive with no recorded trigger and
  no backup-service execution timestamp or retained completion message. Image
  preparation declares the timer enabled, so this observation establishes no
  general release defect or cause. Check timer enablement and a completed
  native backup in the installed maintenance witness; the handbook now
  explains how readers distinguish these states. No timer, service or data was
  changed during the documentation check.

<a id="r46"></a>
## R46 — Qualify optional Tailnet and retained networking behavior

**Execution tasks:** [R46.1](../factory-remaining-tasks.md#r46-1).

**Work:** Qualify; named external cases. **Owner lane:** C. **Accepted inputs:** [R45](05-experience-and-development.md#r45).

**Recorded baseline:** Current optional Tailnet/network qualification; actual external enrollment/routes remain separate. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Verify default absence, administrator-enabled optional content, required Soda controller, full stop when disabled and current LAN/private-origin/HTTPS behavior. Exercise actual enrollment/routes only with the specifically authorized provider input.

**Accept only when:** Native authority/failure/restart/disable cases pass on current bytes; lack of enrollment cannot be relabelled successful external access. Optional Tailnet/domain purchase is not required for core factory use. Missing provider input holds only the named external cases and remains explicit at final scope accounting.

**Access/contract status:** the optional Tailnet extension connection is verified in place — separate `tailnet-package` manifest (admin-gated page, Soda lifecycle controller, no controller authority of its own), package-bound contribution checks, and native site-admin admission with operator/non-admin refusal all pass their existing focused tests; the stale duplicate manifest inside the required soda package was removed and contract tests rebound to the optional package with the pinned official image kept. Native enable/disable/restart and real-client route proof remain held on their concrete inputs.

**Implementation notes:** Consume R45's component identity/lifecycle. First qualify default absence and administrator enable/disable/restart through `lib/host/src/tailnet/{admission,lifecycle}.rs` and existing LAN/private-origin handlers. Then, only with the selected human-supplied client/enrollment/route/trust input, exercise actual external connectivity and revoked/disabled access. Release the precise local and external results separately to R55/R52. Existing native drivers cover authority/failure cases; inability to enroll is an explicit held external case, not success or a requirement to purchase Tailnet for core use. Share the same private-client fixture with Canlang P22 where applicable.

**Principal source owners:** [lib/host/src/tailnet](../../../lib/host/src/tailnet), [docs/guides/operator-setup.md](../../../docs/guides/operator-setup.md), [docs/architecture/networking.md](../../../docs/architecture/networking.md).

<a id="r47"></a>
## R47 — Run current composed PostgreSQL and serving-path race acceptance

**Execution tasks:** [R47.1](../factory-remaining-tasks.md#r47-1), [R47.2](../factory-remaining-tasks.md#r47-2), [R47.3](../factory-remaining-tasks.md#r47-3), [R47.4](../factory-remaining-tasks.md#r47-4).

**Work:** Integration qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R11](02-prompt-to-pr.md#r11), [R13](02-prompt-to-pr.md#r13), [R17](02-prompt-to-pr.md#r17), [R20](03-review-merge-reassessment.md#r20), [R23](03-review-merge-reassessment.md#r23), [R26](03-review-merge-reassessment.md#r26), [R29](03-review-merge-reassessment.md#r29), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R37](04-controls-repair-verification.md#r37), [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40), [R41](04-controls-repair-verification.md#r41), [R42](05-experience-and-development.md#r42).

**Recorded baseline:** Current composed API/Start/MCP/PostgreSQL serving and race acceptance. The catalog owns the remaining prerequisites and release condition.

**Additional scoped CI finding:** [Required CI 39150705999](https://github.com/LevitateOS/sodaos/actions/runs/39150705999) failed `TestFactoryCatalogTransportPerHarness` with a stale admitted-input digest at documentation-only `5ebd1a91`; other required child jobs passed. In `internal/web/api/factory_mcp_test.go`, the sink and `catalogPayload` independently call `fakeCatalogCapability`, whose deadline uses `time.Now().Add(time.Hour).Unix()`. `ScopeDigest` includes that deadline, so crossing a second changes the request digest. The next focused repair should derive requests from the actually admitted fixture capability, preserving intentional stale-input refusal tests; prove the transport across that clock boundary. This is an existing R47 fixture criterion, not a new outcome or evidence that serving admission should be weakened. A later green run does not establish that the timing flaw was repaired.

**Deliver:** Compose real API/Start/MCP producers through shared Store transactions, native/host adapters and actual next-stage consumers. Exercise clean and correction routes, all D13 branches, stall repair, Test Pilot and intervention/crash prefixes.

**Store fixture timing follow-up:** [Required CI 39152199163](https://github.com/LevitateOS/sodaos/actions/runs/39152199163) failed `TestFreshRetryAllowancePreservesConsumedPriorRoot` at `factory_attempt_allowances_test.go:365` while saving the prior allowance: “attempt allowance clock moved backward.” The test captures `now` before `RecordDispatchPacket` admits using its later wall clock, then overwrites the returned `CheckpointUnix` with that earlier second. `checkAllowanceNoRefill` correctly rejects a backward checkpoint. After preserving the actually stored checkpoint, the next bounded fixture repair must derive a coherent retry time and prove both prior-root consumption preservation and real backward-clock refusal through the existing PostgreSQL test. Keep the production no-refill guard; a later passing run is not a repair. This is another existing R47 fixture criterion, not a new parent outcome or authorization to change product code during this planning pass.
**Status (allowance-fixture criterion closed):** `TestFreshRetryAllowancePreservesConsumedPriorRoot` passes against real PostgreSQL 17 through the repository's disposable fixture (`go test -count=1 -v ./internal/store/ -run '^TestFreshRetryAllowancePreservesConsumedPriorRoot$'` → PASS, 0.59s): stored checkpoint preserved, coherent retry time derived, prior-root consumption preserved, and the rewind probe confirms the production no-refill guard still refuses genuinely backward input. The disposable container was stopped and its credentials removed afterward. Remaining R47 scope is unchanged.

**Accept only when:** No pre-injected accepted graph/merge authorization or manual phase invocation substitutes for serving progression. Relevant race/rollback/lost-reply cases pass on actual PostgreSQL with no hidden skips; compare real baseline failures, not summary claims. This adds cross-task composition to checks already required within each task.

**Full design composition:** Include R32 decision/library/maintenance records and
R31/R42 retrieval in the existing API/Store cases. Prove stable identity across
work/feed/library, question incorporation, decision authority versus application,
atomic consolidation/split mappings, stale/refused writes, source revocation,
terminal freeze and lost replies with delayed projections. Shared result preflight
must be read-only and finish must recheck current evidence. Reuse constituent
role/native checks; add only missing cross-owner and race cases required by the
[full design task map](../factory-remaining-tasks.md#full-worker-design-integration).

**Implementation notes:** First triage recorded Deep CI 39139499605 against its actual failed jobs and current selectors: release-build source/CLI assertions, Tailnet navigation, PG topology and fixture/toolchain cases. Start at `.github/workflows/deep-ci.yml`, `tests/build/source_checks_test.go`, `tests/frontend/{tailnet,workspace-navigation}.test.ts`, `scripts/pg_runtime_test.go` and the failed package's existing PG fixture. Prepare only demonstrated repairs and focused reproductions; accepted wait-slot/schema fixes stay closed. This can start independently of the later composed proof. As upstream slices land, exercise API/Start/MCP → real PG → adapter → next consumer in existing tests, ending with clean/correction, D13, stall/Test Pilot and intervention routes. Reuse constituent evidence and run only missing cross-task/race cases with no hidden PG skips. The listed parent inputs gate final composition, not initial failure triage; required CI green does not erase Deep failures.

**Triage status:** PG topology/wiring (four unit lines realigned to the merged image-import composition and `Pull=never`), the CLI surface fixture (`soda-image-import` root-refusal prose) and the oci-archive export literals (skopeo copy form) are repaired test-side with no product change and pass; frontend `tailnet.test.ts` passes. Settled per owner direction: Tailnet stays a separately activatable optional extension with its admin-gated page; stale contract tests are rebound to the optional package (R46 records the verified connection). Held on external inputs: candidate/provisioning/installer/wire/backup cases (Fountain checkout, private keys, daemon, sockets, live PG). The navigation browser suite passes 14/14 on the pinned headless-shell build. Complexity gate is fully green: shim/prose/discovery/precommit test repairs pass, the TS gate test is rewritten to the native mechanism, and two small product restorations landed in `check.ts` — fail-closed on missing explicit input (documented policy since the fail-closed fix, regressed at gate unification) and analyzer-stderr forwarding for download diagnostics; the gate's own bun self-tests match baseline with no regression. Composed R47 qualification remains open.

**Principal source owners:** [internal/store](../../../internal/store), [internal/factory/control](../../../internal/factory/control), [internal/web/api](../../../internal/web/api).

<a id="r48"></a>
## R48 — Qualify actual harness transport and first-response semantic behavior

**Execution tasks:** [R48.1](../factory-remaining-tasks.md#r48-1), [R48.2](../factory-remaining-tasks.md#r48-2), [R48.3](../factory-remaining-tasks.md#r48-3).

**Work:** Native/provider qualification. **Owner lane:** A. **Accepted inputs:** [R12](02-prompt-to-pr.md#r12), [R13](02-prompt-to-pr.md#r13), [R16](02-prompt-to-pr.md#r16), [R22](03-review-merge-reassessment.md#r22), [R24](03-review-merge-reassessment.md#r24), [R29](03-review-merge-reassessment.md#r29), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R40](04-controls-repair-verification.md#r40), [R41](04-controls-repair-verification.md#r41).

**Recorded baseline:** Exact harness/provider transport and first-response semantic trials. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use retained drivers and bounded authorized accounts/fixtures to exercise actual Codex and Muse transport/role/native behavior and first-response semantic cases for decomposition, delivery decisions, reassessment and repair.

**Expanded profile usability:** Consume the complete R30 catalog and R31 context
plus R32/R33/R34/R40 consumers. Qualify all seven production profiles and Library
Admin on supported harnesses: required input discovery, choosing the appropriate
shared/role action, receiving its actual result and recovering an uncertain
receipt. Include decision logging/retrieval, question-answer incorporation,
learning reuse, maintenance edits/consolidation/splitting and authority refusal.
Check actual tool-list/prompt usability before changing exposure; do not impose
an arbitrary tiny catalog or treat advertised stubs as usable capabilities.
These cases use the existing finite funded trials when authorized; planning
integration itself does not dispatch a provider call.

**Generic judgment coverage:** Qualify R30's `systemone.noul`, `systemone.choice`
and `systemone.score` on the actual worker harness with supplied context and
bounded usage. Every role exposes the same primitive contracts. Cover a batch
of independent questions, Noul without a fabricated confidence, Choice option
probabilities, Score level scale, a valid uncertain answer, unavailable funding,
provider failure, malformed/partial output and invocation recovery. A stopped
or revoked caller cannot make new calls. A worker judgment cannot masquerade as
a controller acceptance receipt. Use generic inputs without required Factory
objects to verify that the reusable provider boundary is real.

**Accept only when:** Exact CLI/model/account/reasoning/Contributor/config and prompt/result/usage bindings are retained. Broad roots/joins, tiny work, ambiguity, clean zero-call and correction/uncertainty cases meet contracts. A missing input names the precise case; mock or duplicated SDK-binary runs cannot qualify a real provider/harness.

**Implementation notes:** Consume the role consumers listed above, exact R07–R09 harnesses and R44 account custody. Extend retained transport/semantic drivers: group trials by actual harness/configuration and role, covering first-response decomposition, delivery decision, reassessment and repair, with tiny/broad/join/ambiguity cases. Clean clear remains zero-call. Release prompt/result/usage and native effect bindings to R49, within a finite human-approved provider budget. Normal Muse enrollment is V03's external input; no mock account or repeated SDK binary substitutes. Small-model manageability is desirable, not an acceptance gate; use stronger executor judgment for hard coherent work.

**Principal source owners:** [lib/host/src/terminal/factory](../../../lib/host/src/terminal/factory), [internal/factory/control](../../../internal/factory/control), [docs/development/testing.md](../../../docs/development/testing.md).

<a id="r49"></a>
## R49 — Demonstrate the complete development factory with real Work/Spaces

**Execution tasks:** [R49.1](../factory-remaining-tasks.md#r49-1), [R49.2](../factory-remaining-tasks.md#r49-2), [R49.3](../factory-remaining-tasks.md#r49-3).

**Work:** Installed development qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40), [R43](05-experience-and-development.md#r43), [R44](05-experience-and-development.md#r44), [R45](05-experience-and-development.md#r45), [R47](05-experience-and-development.md#r47), [R48](05-experience-and-development.md#r48).

**Recorded baseline:** Complete real development factory plus persisted role feed question/reply/finding/result journey. The catalog owns the remaining prerequisites and release condition.

**Ralph absorption addition:** Include the role-authored feed in this same
installed journey: follow actual issue/task/PR/review/correction/reassessment and
application-verification records, ask and answer a consequential question through
its authorized route, inspect progress and a finding, and distinguish a role's
done/result submission from Factory confirmation. Observe the agreed component
reuse and real persisted updates; illustrative movie data cannot stand in for
this acceptance. Retain the existing R49 inputs and installed proof boundaries.

**Deliver:** On one owned supported development assembly, run real prompt→semantic graph→selection→investigation/plan→native overlap/checkpoints→PR→review/correction→CI→merge/original closure→dependent→reassessment→Test Pilot, with actual Work/Spaces and human intervention.

**Full design journeys:** Extend this same assembly with the
[MCP journeys](../../design/factory-worker-mcp.md#8-complete-journeys-that-stress-the-design)
and [component failure routing](../../design/factory-feed-components.md#failure-component-routing):

- [ ] A product ambiguity becomes an answer, attributed decision, incorporated
  plan and actual initial Git issues; another worker retrieves the decision.
- [ ] An independent defect becomes a correction/regression result and reusable
  knowledge; a later role finds and uses the exact entry, then contradicts or
  amends it when evidence changes.
- [ ] A known blocker resolves while independent work progresses; an uncertain
  cause takes the bounded diagnosis route. Strategist split/combine/dependency
  changes preserve protected issues, and a failed/partial adoption remains honest.
- [ ] Library Admin edits, removes/restores, consolidates, splits, relates and
  supersedes knowledge/decision items with real receipts. Old work references
  remain inspectable; changed decisions notify affected work without silently
  adopting new scope. Conflicts, stale preview, access revocation, a lost reply
  and delayed indexing exercise the actual recovery paths.
- [ ] All resulting posts obey newest-first separation and external author
  headers, use the correct role/system components and survive reload, paging and
  account change. Library/decision search and human actions share the same facts.
- [ ] A worker uses a generic System One result in a consequential decision or
  learning through the existing Factory outlet. Work detail, feed and library
  retain the supplied-evidence/invocation reference and distinguish the model
  judgment from accepted action. Routine evaluations create no mandatory post;
  unavailable or uncertain judgments cannot manufacture verified completion.

Reuse focused R47/R48 evidence for established boundaries and prove the missing
cross-surface joins here. Report actual remaining failures; this criterion does
not infer success from mock cards, a component count or an agent's Done message.

**Accept only when:** Both clean and correction/Jev routes plus pre-completion stall recovery and restart are observed with exact identities/bytes and bounded usage. Retained human services/work survive. Development success is explicitly distinct from shipping-byte qualification.

**Implementation notes:** Consume R39/R40 completion, R43 combined feed, R44/R45 native foundations and R47/R48 composed/provider proof. Use one owned development assembly and the existing testing.md demonstration: prompt through graph/plan/overlap/checkpoint/PR/review/merge/dependant/reassessment/Test Pilot, then correction, stall and restart branches. Include two authorized subscribed projects in the one newest-first feed, a persisted user comment assessed by JEV and authored by a Product Lead LLM into a linked native issue, question/reply, progress/finding and submitted-versus-verified outcome. Include a comment requiring no issue, an existing-work match, assessment/authoring failure and retry without duplicate issues. Confirm denied cross-project visibility and reading/draft continuity through real Work/Spaces. Reuse earlier primitive proof, observe the actual joins and bounded usage; no fixture-injected accepted phases or shipping qualification claim. The [tomorrow manual-testing slice](../factory-remaining-tasks.md#tomorrow-manual-testing) selects the first simple-app outputs for development use; passing that slice does not accept every R49 case.

**End-user entry and readback join:** Begin the existing demonstration before repository selection using the retained intent/department, exploration and saved-review owners named in R12. After R43’s actual interface and R12/R13 consumers are ready, exercise a nontechnical walkthrough, one consequential correction, reload of the exact stored review, repository association and explicit Project creation/Generate. Observe no provisioning/grant/dispatch merely from viewing or department selection; unresolved consequential input holds only its dependent work. Continue the same request through native graph/plan/effects, R32’s comment-request recovery and R42’s confirmed activity/settings. This closes the previously implicit entry-to-Factory-to-feed handoff without reopening accepted intent/NFX component proofs or adding a second demonstration.

**Principal source owners:** [docs/development/testing.md](../../../docs/development/testing.md), [internal/factory/control](../../../internal/factory/control), [frontend/spaces](../../../frontend/spaces).
