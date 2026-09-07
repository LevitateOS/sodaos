# Architecture revision plan: first-class headless Forgejo integration

**Status: proposed revision plan, not implementation or deployment approval.**
The user requested an architectural revision to support complete Forgejo frontend
replacement coherently. This plan replaces the assumption that stock REST
coverage alone can fulfil that requirement. It does not select a Forgejo upgrade,
a concrete patch, new authentication protocol or an execution target.

The [dashboard implementation plan](dashboard-implementation-plan.md) remains the
leading U01–U20 sequence. The steps here are design/implementation work assigned
to those owners, not another milestone counter or qualification pipeline.
[Current research](forgejo-frontend-integration.md),
[coverage register](forgejo-api-coverage.md), [architecture](architecture.md),
[deferred scope](deferred.md) and [handoff](implementation-status.md) apply.

**Implementation work breakdown:**
[`forgejo-headless-implementation-plan.md`](forgejo-headless-implementation-plan.md)
assigns concrete source paths, H01–H08 tasks within existing U owners, contract
compatibility, native build integration, upgrade/rebase checks and delivery gates.
It is authored work, not evidence that patches or interfaces already exist.

## 1. Architectural commitment

**Soda is the complete presentation layer. Forgejo is the application authority.
The headless integration is a maintained product component connecting them.**

Keep React/PatternFly/Vite+/Zustand, the Go backend, protected sessions, SQLite
extension records, existing adapters and the native project integration. No HTMX
conversion, codebase reset, repository reseeding or VM/data replacement is needed.

Use stock APIs when sufficient. For demonstrated gaps, the proposed architecture
permits narrowly scoped Forgejo-side API additions sharing existing application
functionality. Their concrete contracts and maintenance responsibility must be
reviewed before implementation. This is not permission to independently redesign
Forgejo business rules or indiscriminately expose its internal functions.

```text
Browser: Soda React UI only
             |
Same-origin Soda Go API
  - browser session / CSRF / bounded responses
  - Soda environment/access authority
             |
Explicit acting-user Forgejo interfaces
  - suitable stock APIs
  - reviewed gap-specific API handlers inside Forgejo
             |
Forgejo application functionality and authorization
  - identity, repository access, collaboration, administration
  - native Git and provider-owned persistence
```

Forgejo's own HTML handlers may continue to exist upstream and in the build;
they are not part of Soda's delivered user experience. Hiding their links is not
sufficient: authentication, navigation and browser ingress must also be closed.
Separate operator Cockpit and ordinary Git/SSH/package transports remain selected.
Required licenses and attribution remain intact.

## 2. Ownership and implementation boundaries

| Owner | Owns | Must not acquire |
| --- | --- | --- |
| Soda React | Forms, navigation, rendering, transient input and usable failure states for every required workflow | Provider credentials in browser persistence, native Forgejo HTML or a second application authority |
| Soda Go | Secure browser session, CSRF/origin checks, explicit adapters, bounded composition and legitimate environment/access operations | Forgejo repository clones/indexes, copied permissions, arbitrary proxying or direct provider storage access |
| Forgejo API handlers | Authenticate the acting identity, enforce scopes/native target permissions, validate requests and return defined data/results | Host helper authority, Soda membership policy or operator-token impersonation |
| Forgejo application functionality | Existing rules, credential verification, Git computation, mutations, transactions and persistence | Soda-specific project orchestration or a copied implementation maintained separately in Soda |
| Core build/config owners | Exact Forgejo source/patch/image inputs, compatibility evidence, licenses and appliance delivery | An independent updater, publishing platform or duplicate P-owned build contract |

Where HTML handling mixes rendering and computation, extract only the concrete
shared functionality needed by both callers **inside Forgejo**. Its existing web
handler and the new data handler must use that same implementation. Do not call
an HTML handler and strip markup; do not copy its business logic into Soda.
Some functions assume web context or implicitly authorized inputs: each extraction
must identify those assumptions and preserve the checks explicitly. This is not a
mandate for a broad service-layer refactor or generic endpoint framework.

## 3. Workflow-first audit — before another screen-by-screen expansion

U01/U17 expand the existing coverage register at action level; do not create a
second synchronized inventory. For each workflow record:

- User outcome, required steps, data and commands, including recovery from normal
  validation/expiry/conflict errors and any reauthentication challenge.
- Exact selected upstream source/configuration; stock endpoint or internal owner;
  actual middleware, user scope and resource authority. Swagger absence alone is
  not proof of a missing capability.
- Classification: sufficient stock interface; existing interface needing a Soda
  adapter; missing upstream data/command interface; or unimplemented Soda view.
- Proposed interface work, U owner, tests and actual evidence. A missing API is
  required integration, not a scope waiver or a native-frontend fallback.

Audit complete paths: onboarding → initial password change → MFA/consent → session
renewal; file → history → blame → originating commit; compare → review → merge;
workflow → run → jobs/logs/artifacts/controls; account/security and administrator
operations. Distinguish missing UI implementation from upstream API limitations.

Known starting points: blame, aggregate diff, Actions human interfaces, webhook
secret/event edits, release field clearing, authentication/security. Boards,
graphs and administration need further audit. Existing wiki/package interfaces
must not be declared missing merely because their Soda screens are unfinished.

**Exit:** every required inventory action has a concrete integration route or an
identified design problem with an owner. No claim of complete feasibility until
the authentication/security and administrator paths have been reviewed too.

## 4. Interface contract requirements

For each addition, review an endpoint-specific contract before coding:

1. Exact method/path, request/response types, accepted upstream revision and
   compatibility expectations. Preserve suitable stock interfaces rather than
   inventing a universal replacement API. Choose names/versioning with upstream
   conventions; hypothetical endpoint names are not existing capabilities.
2. Acting-user authentication, least required scopes, repository/admin gates and
   reauthentication requirements. Recheck permissions at execution, not from a
   menu or a copied Soda role. Denied users never fall back to privileged tokens.
3. Stable identities, full-SHA snapshot binding where needed, explicit pagination,
   truncation and missing/binary/invalid-encoding results. Output bounds do not
   alone bound Git computation: define time, input/work and concurrency limits
   appropriate to the concrete operation; propagate cancellation.
4. Native error mapping, secret-safe diagnostics and mutation preconditions.
   Ambiguous writes remain unconfirmed and are not automatically replayed.
5. Protocol behavior for files/streams: authorization, redirects, range/resume if
   required, limits and cancellation; no browser escape into provider HTML.
6. Contract and real integration tests covering successful, denied, malformed,
   expired, canceled and bounded cases. Keep private data out of artifacts/logs.

A compatibility check should verify the required interface revision/capability,
not assume a floating image tag implements it. Select the smallest concrete
check during contract design; no speculative discovery/negotiation framework.
Missing capability fails clearly inside Soda, never redirects to Forgejo.

## 5. Authentication is a separate design gate

U04/U05/U16 must design Soda-only login, first-password change, MFA/passkeys,
recovery, consent, security-sensitive changes and session lifecycle while Forgejo
continues to verify credentials and own identity. Existing OAuth code/PKCE,
encrypted grants and session protections are valuable baseline mechanisms, not
proof that the frontend can remain entirely in Soda.

Specify native challenge ownership, allowed transitions, expiry/replay binding,
rate limiting, CSRF/login-CSRF, account-enumeration resistance, grant issuance,
refresh/revocation and logout semantics. WebAuthn requires an explicit origin/RP-ID
and enrollment-compatibility decision. Inventory supported external identity
providers and redirects; identify any genuine conflict with the Soda-only
experience rather than silently adding an exception or dropping a login method.

No generic password-to-token shortcut, shared browser cookies, MFA bypass,
operator minting of user grants or second Soda password database. Any transient
credential relay must be explicitly threat-modeled with strict secret handling;
it is not authorized by the desired presentation alone.

**Exit:** a reviewed end-to-end authentication/security design with native
capability evidence and security tests. Do not replace current working login,
change live origins or close provider ingress until the replacement is ready.
If upstream support is insufficient, return the exact required changes for a
decision; do not hide the architectural gap behind the two U09 read endpoints.

## 6. First implementation: U09 read contracts

After review, use blame and aggregate comparison as the first bounded vertical
slice of this architecture, not as a shortcut to declaring all parity solved.

- **Blame:** reuse Forgejo's blame engine and ignore-revs behavior. Define pinned
  attribution spans, original paths/line numbers and native commit identities;
  specify resource/size/binary limits and bounded navigation.
- **Comparison:** reuse the native comparison/diff engine. Define direct versus
  merge-base semantics, resolved base/head/merge-base identities, net changed
  files, renames and bounded hunks. Do not concatenate per-commit patches or
  create a temporary PR to obtain an aggregate view.
- Add Forgejo-side tests using native repositories, Soda adapter tests and React
  workflow tests. Prove authorization and Git results, not just DTO shapes.
- Keep private repository denial, read/write separation, moved refs, unusual
  paths, empty/binary/large files, divergent histories and cancellation in scope.

The existing public source investigation found no blame/aggregate-patch API in
15.0.7, 16.0.3 or the inspected development snapshot. No published rejection of a
blame API was found in the issue/design research; performance/abuse discussions
justify real limits, not an assumption that upstream forbids the API.

## 7. Build, security updates and maintenance ownership

Before adopting a patch set, U01/U02/U03/U17 and the shared build/config owners
must agree one source-backed delivery candidate:

- Exact upstream commit, a small reviewable patch series, provenance and reasons
  per patch. Prefer upstream contributions; acceptance/release dates are unknown.
  Remove downstream patches when equivalent compatible upstream work is adopted.
- Chosen tracked patch/lock paths, authoritative build recipe and actual callers;
  no parallel stock/patched runtime selector or dormant backend implementation.
  Stock input remains installed until an approved replacement is delivered.
- Native x86_64/aarch64 Forgejo builds, correct licenses/source availability,
  image identity and integration with existing staging/install contracts.
  Aarch64 unavailability does not block useful x86_64 work or become false proof.
- Dependency/security-update review, patch rebase and test ownership. Test both
  affected native behavior and new API contracts against each proposed upstream
  update; clean patch application is not compatibility evidence.
- Compatibility across Forgejo/Soda/schema/config/grants, with affected strict
  config consumers accounted for. A downstream read API should not introduce
  schema changes without a concrete need; authentication may have wider effects.

Name maintainers and the review/rebase process before shipping. This repository
can own its patch source and recipes, but no new external repository, release
pipeline or publication is assumed. Separate build permission from upstream
contribution, image publication and appliance installation permission.

## 8. Delivery sequence and existing milestone ownership

| Step | Deliverable | Existing owners / gate |
| --- | --- | --- |
| 1 — Adopt boundaries | Review this revision; replace the blanket stock-only assumption with bounded Forgejo-side integration responsibility | U01/U17; architecture decision, no running-system change |
| 2 — Audit complete workflows | Expand the one coverage register and surface authentication/admin risks before further broad screen expansion | U01/U04/U05/U09–U17; research and source planning |
| 3 — Review contracts and delivery | Specify first U09 contracts, authentication design, patch/build ownership and compatibility tests | U02/U03/U04/U09/U16/U17; explicit contract/patch approval before implementation |
| 4 — Implement first vertical slice | Native shared functionality → Forgejo API → Soda Go → React, with feature-owned tests | U09; existing local test/build authorization, no deployment implied |
| 5 — Close remaining workflows | Implement the audited authentication/security, Actions, collaboration and admin interfaces/views | Owning U04–U16 milestones; no duplicated P product suite |
| 6 — Validate integrated candidate | Exact source/build checks; approved isolated fixture proof; populated-state rehearsal and controlled affected-component rollout | U03/U04/owning feature/U20; separately approved target, mutations and installation |
| 7 — Close exposure and cut over | Verify all required UI coverage, Soda navigation/authentication and ingress separation; then default SPA cutover | U17/U18/U20; no gap waivers, no release claim from bundling alone |

Useful independent source work and approved local tests may continue during the
audit. Do not repeatedly ask for already granted local build/test permission.
Specific new patch scope, provider mutations, fixtures, installation and publication
still have their own authorization boundaries.

## 9. Preservation and acceptance

Preserve the current `soda-test`, installed `8b823db` affected components, all four
persistent environments, accounts/keys/memberships, repositories, dirty work,
workloads, backup sets and failed/successful evidence. U08 remains accepted only
for its recorded revision/scope; changed auth/provider behavior needs new affected
regression evidence, not relabeling of historical results.

Before approved rollout, rehearse against a consistent restricted populated-state
copy. Retain prior artifacts and record version-changing migration effects.
Never assume an older binary or database backup is a lossless rollback after new
writes. No resets, replacement containers, global cleanup or first-install replay.

Final acceptance requires:

- Every required workflow operates inside Soda with real native results, including
  authorization failures and authentication/security challenges.
- Forgejo alone owns provider authority and state; ordinary operations introduce
  no host-root, project-admin or operator impersonation path.
- Tests distinguish app-owned navigation, provider-returned object/content links,
  authentication redirects, file/protocol URLs and direct browser origins. Users
  cannot reach the Forgejo frontend in the delivered deployment; Git/package/API
  protocols and retained Cockpit remain deliberately separated and functional.
- Matching source/patch/bundle evidence, both target architectures as required
  by U20, preserved populated state and affected retained-service regressions.
- Unimplemented required functionality remains incomplete. Neither an unavailable
  message, a native-page link nor a written contract is completed parity.

## 10. Immediate review decisions

Approve or amend the ownership boundary first. Then resolve: the complete workflow
audit, the first patch baseline/contracts, named maintenance/build responsibility,
and the separate authentication/security design. These are engineering decisions,
not another request to allow Forgejo frontend fallbacks. This plan itself makes
no code, service, database, dependency, image or infrastructure changes.
