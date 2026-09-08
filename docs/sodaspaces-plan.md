# Sodaspaces implementation plan

Add one **Sodaspaces button beside Forgejo's repository actions**, opening a
**right-side shared-environment drawer**. No new tab or standalone page.
Both old Soda frontends and duplicate forge adapters are removed. The read-only
hooks/context caller and real isolated x86_64 Forgejo/Caddy/browser journey now
pass; see [exact evidence and limits](implementation-status.md#isolated-local-sodaspaces-browser-execution).
Step 2's backend repository reads/new-join checks are implemented. Step 3's bounded
x86_64 exit passed at `ee8091a`: real native build/stage/export and an exported-payload
browser journey. Step 4 is now source-implemented with local handler/DOM coverage;
step 5's bounded native x86_64 access exit passed at `bdbce8e`. Step 6's separately approved retained cutover and native browser/SSH observations also passed. The [security review](implementation-status.md#security-review-and-fix-plan)
confirmed two existing gaps: callbacks could outlive Soda logout, and new joins
did not check repository access. Both fixes below are now source-implemented and
locally tested, then delivered in the approved retained cutover. Deterministic race
coverage remains handler/store evidence; installed observations have their own scope.

**This implementation sequence is complete at its bounded x86_64 scope.** The next
concrete item is the [existing-account browser terminal](#next-item-existing-account-browser-terminal).
Runner settings and independent acceptance obligations remain [separate follow-up](#follow-up-and-limits).
The read-only step-3 gate alone did not establish appliance installation, project
access or retained-state cutover; those later results have distinct evidence.

## Selected approach

- **Backend:** existing Go API, SQLite, encrypted Forgejo OAuth grants and restricted
  native helper. Keep `soda-dashboard` and all persistent project/service identities.
- **Frontend:** stock Forgejo, two small custom-template hooks, browser `<dialog>`,
  scoped native styling and vanilla JavaScript using `fetch` with the JSON API.
  **No added HTMX, React, component library or frontend build.** Forgejo's own
  frontend and Cockpit's separate React/PatternFly stack remain intact.
- **Routing candidate:** existing Caddy, with only `/-/soda/` sent to the Go backend
  on Forgejo's existing HTTPS origin. All other native routes stay with Forgejo.
  The isolated browser journey now exercises this routing; appliance cutover is separate.
- **Authority:** Forgejo owns identity, native sessions, permissions, Git keys,
  collaboration and administration. Soda owns its additional data and real
  environment/access integration—not copied roles or another password authority.
- **Environment model:** persistent and shared, with personal Linux accounts inside
  each project. Opening the drawer never creates, joins, starts or repairs anything.
  Sodaspaces is a UI name, not disposable Codespaces or a browser IDE.

## Delivery sequence

Keep each slice coherent, with its focused tests and affected guide updates in the
same change. The [API guide](dashboard-api.md) describes today's retained endpoints;
planned changes here do not claim those contracts already exist. The two security
fixes landed as separate backend/test commits, moving new-join authorization ahead
of UI wiring. Preserve them while adding explicit actions to the proven read-only
caller; source tests and hidden buttons are not installed authorization proof.

Standing implementation/testing approval covers routine planned local work, including
isolated delivery tests; no repeated approval gate is needed for that scope. Record
exact candidates, targets, private inputs, effects and retention before execution.
Preserve all existing fixtures, projects, credentials and failed evidence. Unrelated
provider/host-network changes and retained-appliance cutover remain outside that
scope; this document itself grants no execution permission.

### 1. Establish the browser-to-backend contract

**Files:** `internal/{config,web,store}/`, `cmd/soda-setup/`,
`appliance/config/proxy.Caddyfile`, `appliance/bin/soda-activate` and their callers.

- Use the configured `forgejo_url` as the single browser origin. Retire the separate
  `public_url`/setup flag and distinct-origin activation rule in a coordinated
  configuration change, including strict-loader consumers such as `soda-runners`.
  Keep origin validation; do not put a path into an origin field. Native Forgejo's
  URL, WebAuthn RP-ID/origin, Git advertisement and private listeners stay unchanged.
- Proxy only the exact Soda namespace, forwarding its prefix unchanged; Go owns
  the mounted routes and generated links. Target paths are `/-/soda/api/...`,
  `/-/soda/login` and `/-/soda/oauth/callback`. Keep `/healthz` available internally.
  Verify path boundaries/normalization and native route collisions; do not capture
  Forgejo's `/api/`, `/user/`, `/assets/`, Git/LFS/package or other `/-/` routes.
- Give Soda uniquely named, host-only, Secure/HttpOnly/SameSite=Lax cookies scoped
  to `/-/soda/`. Define legacy-cookie and pending-login cutover behavior explicitly;
  do not accept ambiguous duplicate cookies or borrow Forgejo's session cookie.
  Cookie paths are not a security boundary between applications on one origin.
- Use the retained session/acting-grant checks. Compare native template context,
  Soda's session and fresh provider subject by stable user ID; disable environment
  actions on mismatch and offer explicit re-authentication. Check expected-actor
  mismatches server-side too, but treat client context as a guard, **not proof of a
  native session or authorization**. Define account-switch/stale-tab behavior;
  do not promise atomic logout across Forgejo, Soda and Linux.
- Keep exact-origin/CSRF/fetch-metadata and bounded-JSON checks, PKCE/single-use state,
  encrypted grants, refresh serialization and logout-winning persistence. Preserve
  Soda-only logout and actual upstream consent handling; no CORS relaxation,
  bootstrap-token fallback, automatic consent revocation or mutation replay.
- Bind return repository/expected-user IDs to the existing short-lived OAuth
  transaction. After validated callback, resolve the repository through the acting
  grant and reconstruct its native URL under configured Forgejo, optionally with
  a drawer-opening fragment. No arbitrary `return_to` or callback-selected URL;
  use configured Forgejo home if context is unavailable. Define only the necessary
  OAuth-state storage change and test existing schema-v3 preservation; do not revive
  the historical destination column as a general redirect mechanism.

**Exit:** contract tests cover routing/callbacks, anonymous and mismatched actors,
CSRF rejection, state replay/expiry, refresh/logout races and old-state handling.
Resolve identity/stale-tab rules before moving on. Verify the smallest real stock-
Forgejo OAuth/proxy browser round trip under approved fixture scope before enabling
mutation controls; mocks are not that proof. If supported mechanisms cannot meet
this contract, stop and explain the precise gap—no fork or substitute frontend.

#### OAuth callback and logout fix

**Source-implemented:** schema v5, conditional finalization/cancellation and focused
race/migration tests. The bounded isolated browser/proxy journey passed; deterministic
callback/logout races remain handler/store evidence, not browser race proof. Retained-
state rehearsal is still pending. The steps below record the selected contract.

**Files:** `internal/web/{auth,api}.go`, `internal/store/{store,login,grants,migrations}.go`
and their focused tests. Preserve the existing PKCE/state/cookie/actor/consent and
safe-return checks; no provider changes or global logout mechanism.

1. Promote the review's paused-callback/logout reproduction into a normal Go
   regression test. The invariant is **no usable session/grant can survive a
   successful Soda logout of the same login context**, even if callback response
   headers arrive later. Checking the old session before network I/O or merely
   clearing the OAuth cookie is insufficient.
2. Add one small persisted, opaque browser-login context, referenced by Soda sessions
   and OAuth transactions, never a new identity authority or public API field. Its
   current OAuth state hash is the pending-attempt marker; no extra generation
   counter or new browser cookie is needed. `/login` reuses the authenticated
   session's context, or a valid pending OAuth cookie's context
   for an anonymous flow, and atomically replaces that marker with the new attempt.
   A genuinely fresh anonymous start creates a context. A cancelled/expired bound
   attempt must never fall back to anonymous completion.
3. Keep OAuth state single-use before provider exchange. Retain the context marker
   while that exchange/identity/consent/repository lookup is in flight. Afterwards,
   one short store transaction must check the live context and matching marker,
   consume the marker, save the profile, rotate the session and insert its encrypted
   grant together. A missing/superseded marker fails closed with an explicit restart
   message and no profile/session/grant write or session-cookie replacement. Do not
   hold a database transaction or introduce a login-context lock across provider
   HTTP calls; keep the existing serialized grant refresh.
4. After the existing actor/CSRF checks, logout atomically invalidates that context,
   its pending attempt and attached session/grant. Carry the context from the
   authenticated request so an already-authorized logout still removes a replacement
   if callback commits first; do not rely solely on deleting the old cookie's row.
   If logout commits first, callback finalization fails. A late cookie for a deleted
   session is unusable. A stale, unauthenticated logout must not falsely report 204.
   Do not cancel another browser context or all sessions belonging to the user.
5. Preserve normal first sign-in and explicit later re-authentication. Superseded
   callbacks must not expire a newer flow's cookies. Keep context/attempt expiry
   bounded to the associated session/login lifetime; restarting the process cannot
   restore a cancelled attempt. Native Forgejo logout and Linux access stay separate.
6. Use an append-only migration for the bounded context records/references. Give
   existing Soda sessions independent contexts without changing token hashes,
   identities, expiry or grant ciphertext/key. Old pending OAuth rows lacking the
   cancellation binding must require a new sign-in, not gain an anonymous fallback.
   Update API/credential guides with that compatibility rule and rehearse before
   deployment; do not rewrite v4, downgrade markers or touch retained private state.

**Tests/exit:** deterministic barriers cover logout while exchange/return lookup is
paused, callback commit before an already-authorized logout, delayed Set-Cookie,
pending and claimed attempts, superseding sign-in, anonymous success, explicit
sign-in after logout, expiry/replay/restart and transaction failure. Verify no
resurrection, no stale profile changes, and unrelated sessions/grants preserved.
Retain refresh/logout tests. Genuine v3/v4 migration fixtures must preserve product
records/encrypted bytes and reject wrong/missing keys before schema changes. These
are Soda handler/store tests, not new upstream authentication conformance tests.

### 2. Make environment reads repository-scoped

**Files:** `internal/web/{environments_api,environment_authority}.go`,
`internal/forgejo/ownership.go` and `internal/store/`.

- Emit the native repository's stable ID from template context, not a parsed URL or
  owner/name guess. Treat it as untrusted input; resolve repository visibility and
  current ownership with the acting grant and existing `RepositoryByID` client.
- Replace the unused catalog read with a required repository-ID query on
  `GET /api/environments?repository_id=...` (under the new public prefix). Query the
  existing unique association directly, returning absent or one retained reservation,
  current repository context and derived action availability. Do not serialize all
  projects and filter in JavaScript, or mistake lookup failure for absence.
- Reuse detail/own-membership/live-observation and connection operations. Apply
  authorization to direct environment-ID callers too, without removing legitimate
  own-member degraded reads or explicit Soda operator visibility. Preserve
  `authority_unavailable`, original Linux logins and nullable native observations.
  No copied permission inventory or automatic Linux/key revocation is introduced.

**Exit:** tests cover absent/existing/incomplete state, inaccessible and invalid IDs,
rename/transfer, human/org-owner versus administrator boundaries, provider failure
and no cross-project disclosure through alternate routes.

#### Repository authorization fix

**Source-implemented:** required repository lookup, direct-ID read boundaries and
fresh new-join checks; focused denial/concurrency/degraded-access tests. The read-only
native read-only UI initially passed with absent views only. Stable-ID creation and
mutation controls subsequently passed local coverage and step 5's bounded native
helper-backed access journey, including a real running view. The following
steps record the implemented contract, not further execution permission.

**Files:** the step-2 owners above plus `internal/web/provider.go` where its existing
acting-grant handling is reused. No helper protocol, Linux account or permission
inventory redesign. Implement the new-join guard here, not when UI buttons arrive.

1. Promote the review's catalog → unauthorized new join reproduction into focused
   handler tests. Require bounded, canonical, single `repository_id` input for the
   collection read and use the unique stored association directly. Resolve native
   visibility first; denial/unavailability is never an empty successful lookup.
2. For a **new** join, load the retained association server-side and use its stable
   `RepositoryID`, never caller-supplied owner/repository/privilege fields. Require
   the actor's grant and actual `read:user` / `read:repository` consent; verify the
   fresh provider subject matches the Soda session and call `RepositoryByID` with
   that same grant. Use the verified current login for new Linux-name validation.
   Neither Soda operator nor generic site/org administrator status bypasses this
   new-join check. Native
   repository visibility suffices; do not invent an owner-only/write-role rule.
3. Complete authorization before disclosing provisioning state or invoking the
   helper. Missing/revoked grants, missing consent, subject mismatch, 403/404,
   malformed/oversized responses and provider failure must produce the existing
   sanitized authentication/denial/unavailable errors with **zero native account
   calls and no membership write**. Do not fall back to setup credentials, cached
   ownership or a prior drawer read. Recheck on each new-join request; permission
   can still change upstream afterwards, so promise no distributed atomicity.
4. Apply the read boundary to direct-ID detail/member callers too: an ordinary
   nonmember needs current repository visibility before metadata/native inspection.
   Keep explicit Soda operator inspection and existing members' legitimate degraded
   own-account/connection reads. Full member lists still require current human/org
   ownership or the configured Soda operator; visibility alone is not elevation.
   Connection remains own-membership-only. Reuse verified request-local context
   rather than duplicating provider calls or persisting copied permissions.
5. Existing-member joins stay non-mutating/idempotent and retain their original
   Linux login; do not reinstall keys or revoke access on provider failure. For
   authorized new members retain readiness/key/Linux-name checks, the fixed account
   helper, and membership only after confirmed success. Preserve honest native/
   persistence failures and no automatic retry. Adapt callers/errors to the removed
   catalog; stable-ID creation and drawer controls remain the separate step-4 work.

**Tests/exit:** cover valid own/collaborator joins; no grant/consent; denial, timeout,
invalid response and subject mismatch; rename/transfer; operator/admin non-bypass;
direct-ID disclosure; and provider access lost between a drawer read and join.
Assert helper-call and membership-write absence on every new-join denial. Preserve
original-login/idempotent/degraded own-member and explicit operator-read tests,
CSRF/actor checks, concurrent joins, native failure and result-persistence failure.
Run the full Go suite and focused web/store/Forgejo/config races for both fixes,
plus affected documentation checks. Read-only browser/proxy proof passed at the
recorded scope; helper-backed native access proof remains step 5. Neither mocked
helper success nor these fixes revoke existing Linux accounts, keys, SSH sessions
or workloads.

### 3. Deliver the read-only button and drawer

**Bounded x86_64 exit passed at `ee8091a`:** source/DOM/conflict tests, production
native build/aggregate checks, actual stage/export verification and the
[opt-in native journey](native-validation.md#read-only-sodaspaces-browser-probe)
against exported hooks/branding and the built dashboard image passed. OAuth, native
tab transitions and BFCache were real; only absent-environment views were native.
This is not appliance installation or existing-project/runtime acceptance. Hook/assets,
source tests, packaging fixtures and the opt-in native journey landed in separate
slices. This completed gate permits step-4 implementation, not a claim that mutation
controls or native provisioning have passed.

**New product source:** `appliance/forgejo/templates/custom/{header,footer}.tmpl`
and `appliance/forgejo/public/assets/sodaspaces.{css,js}`. No new backend endpoint,
schema change, frontend build, component library or upstream executable is planned.

#### Native context and authenticated reads

1. Use the exact stock 15.0.7 hooks: `custom/header` loads our local CSS;
   `custom/footer` emits hidden Soda-owned markup and loads our script after the
   native script tag. `routers/common/auth.go` supplies `.IsSigned`/`.SignedUserID`;
   `services/context/repo.go` supplies `.Repository.ID`. Emit IDs in escaped HTML
   data attributes, not inline executable JSON or parsed owner/name URLs. Guard
   absent/broken/being-created repository context. Keep IDs as canonical decimal
   strings through JavaScript, including values above its safe Number range.
2. Respect native `AppSubUrl` for local asset URLs and verify the effective custom
   asset configuration; never assume a CDN contains Soda's files. The supported
   API deployment remains origin-only with fixed `/-/soda/`, not a new subpath mode.
   Missing/invalid repository context, a malformed signed actor or unsupported
   placement means no active mount, not guessed endpoints. Anonymous context remains
   valid for explicit sign-in. Do not alter `base/head_script` or native `window.config`.
3. On explicit open, bootstrap `GET /-/soda/api/session`. Compare its user ID to the
   signed native-page ID, then check fresh `/-/soda/api/forgejo/me` with
   `X-Soda-Expected-User-ID`. Only matching page/session/provider IDs unlock reads.
   Anonymous/mismatched pages get explicit sign-in/re-authentication, never the
   bootstrap actor silently substituted for the page actor. Login links carry only
   the validated repository ID and, when signed in, expected native user ID.
4. Keep explicit **Sign out of Soda** separate: use the displayed, bootstrapped
   Soda actor and its in-memory CSRF token with the protected JSON logout endpoint.
   This is not environment permission or native Forgejo/Linux logout. Neither OAuth
   navigation nor logout runs merely from opening the drawer. Stale/409 login failures
   require another explicit start rather than an automatic redirect/retry loop.
5. After identity matching, call the required repository-scoped collection read.
   Zero items means absent; one item selects the detail read. Validate IDs against
   the requested repository and selected environment before rendering. Denial or
   provider failure is never absence; do not fall back to a catalog or cached ID.
   Preserve `authority_unavailable`, nullable observations and original own login.
   Member-list, key and connection endpoints are not needed in this slice.
6. Use a small GET/JSON reader with same-origin credentials, no-store, redirect
   rejection, a 64-KiB response cap (including actual streamed bytes) and checks for
   required field types. Build only fixed Soda paths; use text/value assignment,
   never response HTML. Keep CSRF and results in memory, not browser storage/logs.
   Abort/discard reads on Close; a request generation prevents late responses from
   overwriting a reopened drawer. A new open/Refresh starts a fresh check sequence.
7. On pagehide, loss of visibility/window focus, or BFCache restoration, invalidate
   the sequence and clear Soda's displayed data. On resume, keep environment reads
   disabled until a **full native-page reload** obtains fresh template context.
   Offer an explicit Reload action; do not auto-reload away unsaved native form edits,
   scrape a fetched page, or treat another Soda API read as fresh native identity.
   Guard initial pageshow/focus so normal load is not a reload loop. Keep native
   beforeunload behavior intact. This is snapshot consistency, not live native-session
   authentication or atomic cross-system logout.

#### Read-only drawer behavior

- Mount once per document: move only our `type="button"` into
  `.repo-header .repo-buttons`; leave native actions/forms/listeners intact. Missing
  or ambiguous mounts fail closed, with no polling or general DOM-repair framework.
- Use browser `showModal()`/`close()` and the [verified native styling](forgejo-frontend-integration.md#minimal-button-drawer-and-loading-candidate).
  Provide a labelled, right-aligned `<dialog>`, filling scrolling content wrapper,
  keyboard focus entry/return, Escape, backdrop and Close. Scope CSS to Soda-owned
  elements; apply loading classes to content, never the positioned dialog. Keep
  Close usable, status readable and `aria-busy` truthful at narrow/wide widths.
- Render checking, explicit authentication/mismatch, absent, provisioned/running,
  stopped, incomplete, denied and unavailable states. Distinguish the stored
  provisioning result from a live observation; neither an IP nor running status
  proves client reachability. Show actual actor/repository and any verified own
  project login. No Create, key-save, Join, SSH/copy, lifecycle or terminal controls,
  including dormant/hidden versions; those belong to step 4 or later.
- Exact `#sodaspaces` may reopen the drawer after OAuth; it grants no authority and
  still starts the full identity/read sequence. Other native fragments/navigation
  remain untouched. Close/reopen/Refresh must issue no environment or key mutation.
  OAuth/logout and provider-grant refresh can change authentication state: read-only
  here describes environment behavior, not a claim that authentication has no effects.

#### Packaging and conflict refusal

**Implemented and tested:** source/conflict fixtures, actual native stage/export and
exported-payload browser checks passed at `ee8091a`. First-install/activation and
retained-appliance delivery remain unproven by that run.

- `scripts/stage.py` copies only the four files into
  `/var/lib/soda/forgejo/gitea/{templates/custom/,public/assets/}`. Keep readable
  0644 files/0755 new directories and existing canonical branding. Retain selected
  `STATIC_CACHE_TIME=0` and verify asset revalidation; no new cache/build pipeline.
- `internal/nativebuild/bundle.go` admits only the two template filenames and their
  required ancestors, and requires all four files in the inventory. Do not whitelist
  arbitrary templates or Forgejo data. Update bundle fixtures, staged-path assertions
  in `tests/packaging/test_staging.py`, and applicable notices together.
- Preserve `scripts/install-native.sh`'s implemented exact hook/asset destination
  and unsafe-ancestor refusal **before its first host write**, with temporary-
  filesystem preflight regressions. Refuse occupied destinations and symlinks rather
  than merge/adopt operator hooks. Keep ownership/label handling narrow for the new
  readable template paths (current Forgejo UID/GID 1000), not recursive changes to
  the mutable Forgejo tree.
- First-install remains first-install only. Existing-target delivery belongs to the
  reviewed rehearsal/cutover procedure: preserve custom files, require an explicit
  decision for conflicts, back up exact prior bytes and verify effective CustomPath,
  ownership/labels and template reload needs. Do not use install/setup/activation
  scripts as an upgrade shortcut or restart Forgejo merely to try a hook.

#### Tests, native proof and completion

- Retain `scripts/sodaspaces_templates_test.go` for the two Soda templates and
  `tests/frontend/sodaspaces.test.mjs` for the actual script's DOM/fetch behavior.
  Use Node's test runner and the already-pinned Cockpit `jsdom` dependency, not a
  root frontend manifest. `scripts/check-native.sh` runs `tests/frontend/*.test.mjs`;
  preserve that wiring and the standalone source-check command. The aggregate requires
  a clean revision and actual stage. Fixture context/dialog doubles are not upstream
  rendering, CSS, focus or native browser evidence.
- Cover signed/anonymous/large/malformed IDs; missing rows/context; matching and
  mismatched actors; contextual login and Soda-only logout; malformed/oversized/HTML
  responses; 401/403/404/409/unavailable states; absent versus existing/incomplete
  environments; close/reopen/late reads; stale/BFCache reload gating; exact fragment;
  and no environment/key writes. Packaging fixtures exercise missing required files,
  extra templates, modes and pre-write conflict refusal, not just recipe strings.
- Retain opt-in `tests/installed/sodaspaces.mjs` using existing pinned Playwright.
  Within the applicable fixture/target scope, exercise actual stock Forgejo → Caddy → Go
  OAuth → safe repository return → drawer, anonymous and two-account transitions,
  native-only and Soda-only account changes, logout, BFCache/focus, keyboard/backdrop/
  Close, responsive themes and native form/navigation coexistence. Verify actual
  Soda cookie host/path/security attributes without logging values, no native-cookie
  borrowing, and protected logout's actor/CSRF behavior. Include actual proxy path/
  encoding checks and only focused native route smoke checks. Do not seed an
  authenticated session or call intercepted responses native proof.
- Carry forward the real-browser corrections, not the failed probe assumptions:
  use `tests/installed/native-browser.mjs` with sandbox and fixture-only trusted TLS,
  without Playwright's forced focus/visibility or synthetic BFCache events. Wait for
  dialog/ARIA readiness and asynchronous focus return; browser chrome/body may own
  focus while Soda must clear. Use history commit waits for real BFCache restoration.
  Preserve stock logout's SSE navigation and exact native `POST /-/fetch-redirect`
  body `redirect=%2F`; do not intercept its response or disable native workers,
  navigation or beforeunload. CDP Fetch guards every redirect hop before transmission.
- Record exact candidate/images, isolated origin/TLS, users/repos/OAuth operations,
  private inputs, process effects and retention for each run under standing testing
  approval. Use existing owned tools, not a new support platform. Preserve previous
  fixtures and artifacts; do not adopt occupied paths. The installed separate-origin/
  schema-v3 pairing cannot host this candidate through casual live edits; follow
  [credential rehearsal](dashboard-credentials.md) before separate live cutover.
- No environment creation/join/key save is needed for the first real OAuth proof.
  Record which existing-state views were native observations versus local fixtures;
  use existing environments only under explicit read scope. Preserve browser profiles
  privately, accept password/credential files rather than argv values, and retain
  sanitized outcomes without raw callback URLs, cookies, bodies, traces or credentials.
- **Exit:** source/template/DOM and real staged-payload checks pass, and the approved
  native run proves identity matching, safe OAuth return, stale-tab blocking and the
  read-only dialog without native navigation regressions. Record exact revision and
  failures in the handoff. This exit is satisfied at the bounded `ee8091a` scope above;
  proceed to step-4 mutation controls. Step-5 integrated access proof and step-6 cutover
  remain separate, as do the existing-account terminal and independent aarch64 acceptance.

### 4. Wire the explicit access actions

**Status:** source-implemented; full Go, focused races, 89 Node tests and 36 build
fixtures passed. See the [handoff](implementation-status.md#explicit-access-actions-source-implementation).
The [API guide](dashboard-api.md) now specifies the stable-ID create body. The
contracts below are implemented with the existing stack; updated-payload browser,
real account/key/SSH evidence and appliance cutover remain steps 5–6.

**Files:** the existing drawer script/templates/styles, retained Go environment/key
handlers and their focused tests. Keep the restricted helper and store as owners
of native operations and legitimate Soda records.

1. **Create shared environment:** change the existing POST to accept a canonical
   decimal-string `repository_id`, rejecting the old owner/name body. Resolve through
   the acting grant's fresh subject, actual user/repository consent and
   `RepositoryByID`; require the current human owner server-side, independently of
   advisory `can_create` or an earlier read. No organization-owned creation or
   operator/admin impersonation. Preserve reservation-before-provisioning, unique
   association by stable ID and honest incomplete results. Creation does not join.
   The API guide/callers/tests use this contract; the unused owner/name Forgejo
   adapter is removed. JSON writes also reject duplicate fields and invalid UTF-8.
2. **Save public key:** show registered development-key summaries; when needed,
   accept one public key through the retained key API. No private-key upload,
   native Git-key changes, key selector or automatic later propagation.
3. **Add me:** wire the already-protected join from the
   [step-2 authorization fix](#repository-authorization-fix), using the real account/
   key helper and recording membership only after confirmed success. Existing
   members retain their original login. Surface unsupported Linux names, native-
   account failure and result-persistence failure without automatic repair.
4. **Connect:** display the member's current project login/IP, readonly SSH command,
   native Copy control and public host-key fingerprint. Stopped/unavailable state
   must not advertise a usable connection; an IP is not proof of client routing.

#### Submission and result handling

- Require an explicit action, a non-stale native document and matching page/session/
  fresh-provider IDs before submission. Recheck the current session/provider on the
  action path; after any asynchronous read, confirm the same active context before
  dispatch. Send the expected actor and in-memory CSRF token on protected JSON writes.
  The server still owns operation-specific authority; a client precheck is not
  authorization or atomic native-session verification. Keep existing-member joins
  idempotent with their original login and preserve degraded own-access API behavior.
- Reuse fixed same-origin paths, redirect rejection, no-store, bounded streamed JSON
  and field/association validation for the additional reads and write results. Render
  text/values, not response HTML. Keep development keys separate from Forgejo Git keys;
  saving a key neither joins nor changes an existing member's installed keys. Verify
  the native Copy mechanism against the selected upstream source before wiring it.
- Prevent duplicate dispatch while an action is pending, including close/reopen.
  Keep Close usable and pending labels truthful. Read abort/generation handling must
  not be mistaken for mutation cancellation: the server/helper may continue after
  an abort, timeout, logout or page departure. Late results must not repopulate a
  closed, stale, reopened or differently authenticated drawer.
- Distinguish confirmed success, confirmed rejection and an uncertain outcome
  (transport loss, unreadable response or native/result-persistence failure). Do not
  optimistically add membership or advertise SSH. Never chain create → join or key
  save → join; never replay a write on open, Refresh, reload or OAuth return.
- After completion/uncertainty, use bounded read-only observation when context is
  still valid; stale documents require explicit full-page reload first. Refresh keys,
  repository reservation/detail or own connection as applicable, without polling.
  A missing membership after helper/persistence failure is not proof that no Linux
  account was created, nor permission to retry/repair. Keep unresolved native outcomes
  explicit and direct them to operator inspection; no durable jobs, browser operation
  journal, automatic compensation or generalized recovery subsystem.

**Exit:** handler/UI tests cover current owner/non-owner/org/admin boundaries,
rename/transfer and access loss between read and submit, malformed/large stable IDs,
actor/CSRF/consent denial, missing/bad/duplicate keys, duplicate submits/concurrent
reservations, incomplete native results and no false membership. Exercise stale/blur/
BFCache, close/reopen, pending logout, delayed/malformed/lost responses and safe rereads
with zero write replay. Preserve original-login idempotency and own-only connection
access; stopped/unavailable/malformed connection data must not expose a usable Copy
command. These local checks passed. Source/helper doubles establish these branches,
not actual SSH provisioning; step 5 supplies the helper-backed native evidence.

### 5. Validate the integrated native experience

**Bounded x86_64 exit passed at `bdbce8e`:** real native build/export/first delivery,
stock-browser create/key/join/Copy and actual own-key SSH/PTY/SCP/SFTP from the fresh
fixture's separate client bridge namespace. See the [handoff](implementation-status.md#phase-5-bounded-native-access-proof).
This is not builder/laptop routing, a lifecycle/workload retest or aarch64 acceptance.
The requirements below remain the owned regression scope, not instructions to replay
fixture mutations.

- Extend the existing product-owned native-page journey using the pinned tooling
  and proven browser launcher; adapt retained connection probes to the mounted API.
  Keep the read-only mode's environment-write refusal. Add an explicit bounded access
  mode/scenario allowing only the intended key/create/join requests for declared
  fixture actors/repository/retained reservation. Preserve redirect-hop guards and
  native navigation checks; do not broadly allow POSTs or substitute responses.
  Do not restore retired standalone browser journeys or add a frontend stack.
- Use a separately recorded isolated helper-backed target for access proof. Neither
  retained local browser fixture has a helper, projects, memberships or development
  keys; each recorded three absent observations, not existing/running/stopped/incomplete
  native views. Bind the new run to the actual candidate backend/helper/project image
  and delivered UI, private credentials, synthetic actors/repository and declared
  client route. Preserve both earlier fixtures, all run-owned roots and failed results.
  This is planned isolated testing under standing approval, not permission to connect
  a browser fixture to the builder's unrestricted host socket or mutate `soda-test`.
- Exercise real Forgejo → Soda OAuth → repository return → drawer, explicit owner
  create, two users' key/join/own-connection paths and actual SSH access. Cover native
  account switching, expiry/re-consent/logout, denied access and interrupted requests.
  Keep fixture-based failures distinct from actual native provisioning evidence.
- Test Soda's additions, not upstream Forgejo business logic. Check button placement,
  drawer behavior, OAuth integration and that our proxy leaves native routes with
  Forgejo. Use only focused smoke checks for native behavior directly affected by
  our hooks/styles/routing; do not add suites for upstream repository actions,
  administration, MFA/WebAuthn implementation or Git/LFS/package semantics.
  Preserve separate Cockpit Tailnet/Runners logic/tests. Review our selector/hooks
  against the exact stock Forgejo version before accepting an upgrade.
- Run authorized Go/race, JavaScript/browser, Cockpit, build-fixture and staged-
  payload checks through their actual callers. Record revision, scope and failures
  in the [handoff](implementation-status.md); source passes are not installed proof.

**Exit:** exact-candidate native-page and access evidence for this slice: real
reservation/provisioning, each user's explicit key/join, original own login, public
host-key verification and actual own-key SSH from the recorded client. Record which
state/failure views were real versus fixture-only; do not add lifecycle mutations just
to fill a view matrix. No native workflow/security regression may be hidden by mocked
responses. Apply standing approval within planned isolated testing; unrelated provider,
host-network and retained-appliance actions still require separate scope.

### 6. Rehearse and cut over separately

**Bounded exit passed after separate approval.** Fresh/copied-state rehearsal preceded
a new paired backup and affected-component cutover on `soda-test`. Native callback/
namespace/config/schema-v5 delivery, private-page browser/own-connection checks and
all seven existing memberships' SSH/PTY observations passed. Four roots and persistent
product records were preserved; no project lifecycle or routing change occurred.
See the [cutover handoff](implementation-status.md#approved-retained-cutover) and
[affected-component procedure](installation.md#retained-sodaspaces-cutover). The steps
below remain the maintenance contract, not permission to replay it.

- Author the affected-component procedure using [credential preservation](dashboard-credentials.md),
  [installation](installation.md) and [native validation](native-validation.md).
  Rehearse fresh and copied populated state before requesting live cutover; preserve
  the existing database/grant key, OAuth application, four projects and later writes.
- Keep the existing Forgejo origin and OAuth client. The application's actual owner
  can update its callback through native Applications settings without regenerating
  the secret. **Do not blindly PATCH the OAuth application:** inspected 15.0.7's API
  update also regenerates its secret. Plan the callback transition, scoped-cookie
  re-login and strict-config consumers together; do not rerun bootstrap on old state.
- Back up matching DB/config/key/artifacts, proxy configuration and affected custom
  files before an approved rollout. Verify custom path, ownership/labels and actual
  template reload requirements; restart only explicitly approved affected services.
  Preserve operator customizations and failed evidence. Old backups are not a
  lossless rollback after later writes; no replacement/pruning of project roots.

**Exit:** separately approved, rehearsed cutover plus matching native-browser/access
observations. This does not accept the entire appliance or independent aarch64 work.

## Next item: existing-account browser terminal

**Native boundary proved; protected transport/component source implemented.**
The fixed launcher and private helper stream/client passed actual existing-account,
PTY/profile, EOF/lease/owned-helper-loss and independent SSH-preservation checks on
`soda-native-spaces-658f2af`, with no installed helper replacement. Protected browser
transport, a self-contained terminal component and pinned local renderer packaging
are now source implemented, not deployed or genuinely browser-proven. See the [native
proof](implementation-status.md#approved-native-terminal-fixture-proof) and
[component contract](terminal-integration.md).

**Parallel ownership:** another agent owns all Forgejo template overrides and layout.
Do not wire this component deeply into the current design or edit those templates.
The host supplies one mount node and immutable page/environment hints, loads local
styles/module and disposes on close/context change. The component owns its explicit
Open/Disconnect, renderer and stale lifecycle; no native DOM discovery or auto-mount.
Deliver one explicit **Open terminal** action for an existing member of a provisioned,
running environment. Show the original project login and open its native login shell
in its existing home. Keep the terminal inside the existing dialog, widened when
needed, with connection status and Disconnect—not a new repository tab or browser IDE.
Opening the drawer alone must never launch it.

### One bounded candidate

`native drawer → same-origin WebSocket → Go authorization → existing Unix helper →
project-local PTY/login shell`

- Use locally served **xterm.js plus its fit addon** for terminal rendering/resize,
  not a homemade escape-sequence parser or a new component framework. Use
  **coder/websocket** for Go's browser/helper streams. Pin reviewed dependencies and
  ship exact upstream distribution assets/notices through existing build/stage owners;
  no CDN, runtime download, root SPA or new frontend bundler.
- Add one fixed helper operation, not a host command/socket proxy. Resolve the
  container from trusted project state and verify labels/running status; execute by
  the verified container ID. The backend supplies only stored login, stable actor ID
  and bounded terminal controls—never caller-selected command, UID, privilege, cwd,
  environment, host address or Podman flags.
- A small fixed Python launcher owned/embedded by `internal/host/` runs via
  `podman exec --interactive` using existing project Python. Validate the root-owned
  `/var/lib/soda/accounts/<login>` identity marker and passwd entry; allocate a PTY,
  initialize native groups, drop to the non-root account and execute its login shell
  in its home with a clean environment. Input reaches only that PTY. No project-file
  installation, image replacement, private key, password or new account is needed.
- The launcher owns the shell lifetime and a bounded heartbeat lease over its private
  framed stdin/stdout channel. Podman CLI exit/detach is **not** proof that container
  exec ended. Closing a terminal must close its PTY and end its exact owned login
  process, without signalling all processes for that UID or stopping the project.
  Validate this before UI work; if the candidate cannot do it safely, stop and report
  the concrete constraint rather than add a second backend or unrestricted socket.

Source basis: today's helper buffers JSON under a global lock/short timeouts;
`project-account` supplies identity markers and the image supplies Python. Inspected
Podman v5.8.2 exec source permits detach; verify the actual target runtime before proof.
Xterm 6.0.0 and fit 0.11.0 are locked upstream distributions for the local renderer;
archive and extracted file hashes live in `appliance/terminal-assets.lock.json`.
Coder/websocket is now resolved/pinned in `go.mod`/`go.sum`, with its ISC text in
`NOTICE`. These source facts are not native target evidence. Research is linked
from the handoff.

### Authorization and connection lifetime

- Proposed route: `GET /-/soda/api/environments/{id}/terminal`, WebSocket only.
  Before upgrade require a valid Soda session, exact configured HTTPS Origin,
  acceptable fetch metadata and own membership; reject query parameters. Native
  WebSocket cannot send our custom actor/CSRF headers: its first bounded message must
  supply `expected_user_id`, `repository_id`, `csrf_token`, rows and columns instead.
  Verify those against the authenticated session and stored association before any
  native call. Reuse shared checks, without weakening ordinary JSON API protection.
- Before launch, require fresh acting-provider subject, actual user/repository consent
  and current repository visibility through `visibleRepository`. Existing degraded
  connection reads are **not shell-launch authority**. Operator/site-admin/repository-
  owner status cannot substitute for own membership or select another person's login.
  Preserve original membership login after a provider rename; mismatch/missing native
  account/marker, UID 0, missing home, stopped or incomplete state means refusal, not repair.
- This is Soda-authenticated web access, not an SSH-key-possession check. Direct
  SSH/SCP/SFTP and their key policies remain unchanged; neither key changes nor native
  Forgejo-only logout are claimed to atomically revoke the other access mechanisms.
- After authorization accept only bounded input bytes, resize and close; return
  terminal bytes and sanitized status. No bearer tickets in URLs/subprotocols, terminal
  transcripts, keystroke/output logging, session recording or persisted reconnect state.
  Disable terminal-driven clipboard writes, automatic link opening and page mutations;
  manual user paste remains possible. Bound scrollback and clear it on invalidation.
- Bound authentication wait, frames, dimensions, scrollback, queues and concurrent
  connections; use backpressure. Allow one stream per Soda login-context/project,
  rejecting duplicates rather than evicting a terminal. No quota-management subsystem.
- Disconnect on explicit Close/Disconnect, stale page/blur/hidden/pagehide, shell exit,
  transport loss, Soda logout/session rotation/expiry or service shutdown. Reuse the
  existing explicit full-page reload rule; no reconnect/input replay on focus or BFCache.
  Tell users that switching tabs/apps ends this browser connection. Keep native
  beforeunload intact. Escape while terminal-focused goes to the shell; provide
  `Ctrl+Shift+Enter` to focus Disconnect and retain normal dialog Escape elsewhere.
- Bound lifetime to the earlier of Soda session expiry or two hours; use a 60-second
  lost-peer lease, renewed only while local session/membership checks succeed. Soda
  logout actively cancels matching pending/active streams; periodic local checks catch
  missed invalidation within that lease. Test logout versus registration/spawn races.
  No provider polling or global Linux revocation. Closing can interrupt foreground
  work; completed commands and deliberately detached native workloads are not undone.

### Implementation order and exit checks

1. **Native boundary first — `internal/host/`.** Add the fixed launcher/stream path
   outside the mutation lock and buffered timeout, with owned contexts/shutdown and
   no new listener/capability. Focused tests plus an authorized native proof must show
   correct UID/GID/groups, HOME/cwd, shared-tool profile, sudo boundary, TTY/resize/
   Ctrl-C and exit. Missing/mismatched identity launches no shell. EOF/lost heartbeat/
   helper loss must end the owned PTY/login process and preserve unrelated SSH/workloads.
   **Do not proceed to UI while native ownership/teardown is unresolved.**
2. **Protected transport — `internal/web/terminal.go` and existing auth/shutdown.**
   Test actor/Origin/CSRF/consent/membership/provider denials with zero native calls;
   then logout/spawn races, rotation, slow consumers, bad frames, duplicates and
   shutdown. Keep only a small live-stream collection: no SQLite migration, jobs or
   copied permissions. Prove Caddy upgrade/closure through the existing namespace,
   without broader proxy routes or weaker TLS/CSP.
3. **Drawer and packaging — existing hook/assets and one terminal module.** Load
   on explicit use; show truthful connection/refusal states. Test keyboard escape,
   resize, Unicode, paste, TUI Escape, stale/late events and no implicit launch/replay.
   Stage exact local vendor assets/notices through production callers and update
   inventories/conflict checks and API/developer guidance. Preserve native navigation,
   forms and current drawer actions.
4. **Integrated proof, then scoped delivery.** Extend the owned installed journey
   with a separate terminal opt-in, never read-only mode. Use genuine OAuth, trusted
   sandboxed browser, actual proxy/helper and two existing fixture accounts. Prove
   identity/home, a retained run-owned file visible over SSH, interactive editing/
   resize/Ctrl-C, each close path and unaffected ordinary SSH/unrelated sessions.
   Observe actual process exits, not just socket closure. Run Go/race, DOM, packaging
   and native-stage checks against exact bytes; keep aarch64 claims independent.

**Done means:** a genuinely authenticated native-page terminal into the correct
existing account, proven authorization/PTY lifetime and unchanged project identities,
keys, memberships and ordinary access—not just a rendered prompt or mocked helper.

**Execution boundary:** this plan authorizes no native action. Apply existing standing
approval only to covered implementation/local tests; declare the exact fixture,
helper/service changes, process effects and retained probe files and obtain any missing
native grants. A later `soda-test` rollout needs fresh paired backups, affected-byte
review and separate approval (including the newly changed helper). Do not restart or
replace projects to deliver this feature. Browser HTTPS reachability is not laptop
project-subnet/SSH routing proof. Runner settings and other remaining work stay separate.

## Follow-up and limits

**Runner configuration placement:** the user selected moving Soda's local runner
capacity/service configuration from Cockpit into the unified SodaOS/Forgejo native
interface, as operator-only settings—not the repository Sodaspaces drawer or a
revived standalone dashboard. Inspect supported native administrator extension points
before implementation; no Forgejo fork or copied permission authority. Provider-owned
registration authority, workflows, scheduling and results remain upstream-owned.
Reuse the existing backing logic/tests and preserve the Cockpit Runners page until
the replacement works and its removal is coordinated. Tailnet stays in Cockpit.
Repository ownership or arbitrary Forgejo site administration does not confer Soda
operator authority. This is remaining work, not a delivered move or deployment grant.

No lifecycle controls, resource charts, member-management screens, environment
catalog, private-resource branching, generalized recovery or update platform.
[Architecture](architecture.md), [integration](forgejo-frontend-integration.md),
[deferred scope](deferred.md) and [licensing](licensing.md) remain authoritative.
Stock Cockpit and its native integrations remain operator-only; providers own CI.
Production `internal/host/` and `project-os/` own native behavior. Outside
[support tools](native-support.md) invoke owned checks, not a second readiness gate.

Standing implementation/testing approval covers planned local work and isolated
delivery validation, not destructive cleanup, unrelated provider/host-network changes
or silent retained-appliance cutover. Step 3's bounded x86_64 read-only exit passed;
only historical **U08** has user-accepted bounded native project-runtime proof.
Step 5 also passed bounded Sodaspaces access execution; none is final-product or
aarch64 acceptance. Retained console,
operator/provider, licensing, native-support and independent aarch64 obligations
remain in the [handoff](implementation-status.md), not another expanded UI roadmap.
