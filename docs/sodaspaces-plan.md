# Sodaspaces implementation plan

Add one **Sodaspaces button beside Forgejo's repository actions**, opening a
**right-side shared-environment drawer**. No new tab or standalone page.
Both old Soda frontends and duplicate forge adapters are removed; the drawer and
its complete authenticated integration are **not implemented**. Step 1 is underway:
routing/configuration/scoped cookies and backend actor/return context are in source.
Native-page wiring and real browser/proxy proof remain pending. Step 2's backend
repository reads/new-join checks are implemented; native-page context and steps 3–6
remain incomplete. The [security review](implementation-status.md#security-review-and-fix-plan)
confirmed two existing gaps: callbacks could outlive Soda logout, and new joins
did not check repository access. Both fixes below are now source-implemented and
locally tested. Neither is deployed; browser/proxy proof remains pending.

**Next milestone:** finish the native-page caller from steps 1–2 and deliver the
[read-only button/drawer in step 3](#3-deliver-the-read-only-button-and-drawer).
Its implementation slices and completion checks are specified there; no new roadmap.

## Selected approach

- **Backend:** existing Go API, SQLite, encrypted Forgejo OAuth grants and restricted
  native helper. Keep `soda-dashboard` and all persistent project/service identities.
- **Frontend:** stock Forgejo, two small custom-template hooks, browser `<dialog>`,
  scoped native styling and vanilla JavaScript using `fetch` with the JSON API.
  **No added HTMX, React, component library or frontend build.** Forgejo's own
  frontend and Cockpit's separate React/PatternFly stack remain intact.
- **Routing candidate:** existing Caddy, with only `/-/soda/` sent to the Go backend
  on Forgejo's existing HTTPS origin. All other native routes stay with Forgejo.
  The routing foundation is authored/source-tested, not native proxy/browser proof.
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
of UI wiring. Preserve them while completing the native-page/read-only milestone;
source tests and hidden buttons are not installed authentication or authorization proof.

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
race/migration tests. Browser/proxy and installed rehearsal remain pending. The
steps below record the selected contract, not permission for native execution.

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
fresh new-join checks; focused denial/concurrency/degraded-access tests. Native UI,
stable-ID creation and installed verification remain pending. The following steps
record the implemented contract, not further execution permission.

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
plus affected documentation checks. Real browser/proxy/access proof still requires
its separately approved scope; neither mocked helper success nor these fixes revoke
existing Linux accounts, keys, SSH sessions or workloads.

### 3. Deliver the read-only button and drawer

**Next milestone, not implemented:** complete the pending native-page identity/read
caller from steps 1–2 and this drawer as one candidate. Commit hook/assets with
source tests first, packaging with fixtures next, then the opt-in native journey.
Run native proof only with its separate approval. Read-only UI source may proceed
before that proof; do not mark the milestone complete or enable step-4 controls
on fixture results alone.

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

- `scripts/stage.py` copies only the four files into
  `/var/lib/soda/forgejo/gitea/{templates/custom/,public/assets/}`. Keep readable
  0644 files/0755 new directories and existing canonical branding. Retain selected
  `STATIC_CACHE_TIME=0` and verify asset revalidation; no new cache/build pipeline.
- `internal/nativebuild/bundle.go` admits only the two template filenames and their
  required ancestors, and requires all four files in the inventory. Do not whitelist
  arbitrary templates or Forgejo data. Update bundle fixtures, staged-path assertions
  in `tests/packaging/test_staging.py`, and applicable notices together.
- Add exact hook/asset destination conflict refusal to `scripts/install-native.sh`
  **before its first host write**; current ancestor checks alone do not prevent
  overwriting an existing file. Refuse occupied destinations and symlinks rather than
  merge/adopt operator hooks. Extend ownership/label handling narrowly for the new
  readable template paths (current Forgejo UID/GID 1000), not recursive changes to
  the mutable Forgejo tree. Author temporary-filesystem preflight tests.
- First-install remains first-install only. Existing-target delivery belongs to the
  reviewed rehearsal/cutover procedure: preserve custom files, require an explicit
  decision for conflicts, back up exact prior bytes and verify effective CustomPath,
  ownership/labels and template reload needs. Do not use install/setup/activation
  scripts as an upgrade shortcut or restart Forgejo merely to try a hook.

#### Tests, native proof and completion

- Add `scripts/sodaspaces_templates_test.go` for the two Soda templates and
  `tests/frontend/sodaspaces.test.mjs` for the actual script's DOM/fetch behavior.
  Use Node's test runner and the already-pinned Cockpit `jsdom` dependency, not a
  root frontend manifest. Wire the JS test into `scripts/check-native.sh` and
  document its standalone source-check command; the aggregate still requires a
  clean revision and actual stage. Fixture context/dialog doubles are not upstream
  rendering, CSS, focus or native browser evidence.
- Cover signed/anonymous/large/malformed IDs; missing rows/context; matching and
  mismatched actors; contextual login and Soda-only logout; malformed/oversized/HTML
  responses; 401/403/404/409/unavailable states; absent versus existing/incomplete
  environments; close/reopen/late reads; stale/BFCache reload gating; exact fragment;
  and no environment/key writes. Packaging fixtures exercise missing required files,
  extra templates, modes and pre-write conflict refusal, not just recipe strings.
- Add opt-in `tests/installed/sodaspaces.mjs` using existing pinned Playwright. Under
  explicit fixture/target permission, exercise actual stock Forgejo → Caddy → Go
  OAuth → safe repository return → drawer, anonymous and two-account transitions,
  native-only and Soda-only account changes, logout, BFCache/focus, keyboard/backdrop/
  Close, responsive themes and native form/navigation coexistence. Verify actual
  Soda cookie host/path/security attributes without logging values, no native-cookie
  borrowing, and protected logout's actor/CSRF behavior. Include actual proxy path/
  encoding checks and only focused native route smoke checks. Do not seed an
  authenticated session or call intercepted responses native proof.
- Before that run, agree exact candidate/images, target and isolated private origin/
  TLS trust, allowed users/repos/OAuth application/callback operations, database/key
  copies, process starts/restarts and retention. Existing VM/fixture approvals are
  exhausted. Use authorized existing tools, not a new support platform. New fixtures,
  credential changes or retained-target deployment are not authorized by this plan.
  The installed separate-origin/schema-v3 pairing cannot host this candidate through
  casual live edits; follow [credential rehearsal](dashboard-credentials.md).
- No environment creation/join/key save is needed for the first real OAuth proof.
  Record which existing-state views were native observations versus local fixtures;
  use existing environments only under explicit read scope. Preserve browser profiles
  privately, accept password/credential files rather than argv values, and retain
  sanitized outcomes without raw callback URLs, cookies, bodies, traces or credentials.
- **Exit:** source/template/DOM and real staged-payload checks pass, and the approved
  native run proves identity matching, safe OAuth return, stale-tab blocking and the
  read-only dialog without native navigation regressions. Record exact revision and
  failures in the handoff. No such tests ran by writing this plan. Only then proceed
  to step-4 mutation controls; step-5 integrated access proof and step-6 cutover remain
  separate, as do the existing-account terminal and independent aarch64 acceptance.

### 4. Wire the explicit access actions

**Files:** the small drawer script/templates and retained Go environment/key handlers.

1. **Create shared environment:** change the existing POST to accept stable
   `repository_id`, resolving the current human owner server-side. No organization-
   owned creation or operator/admin impersonation. Preserve reservation-before-
   provisioning, uniqueness and honest incomplete results. Creation does not join.
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

**Exit:** handler/UI tests cover owner/non-owner, missing/bad keys, duplicate submits,
concurrent reservations, incomplete native results and no false membership. Pending
controls use truthful labels; close/reopen or a failed response never replays a
mutation. Safely reread after completion or uncertainty; no jobs/recovery subsystem.

### 5. Validate the integrated native experience

- Use the existing pinned browser tooling, not a new frontend dependency stack.
  Add a focused native-page Sodaspaces journey and adapt retained connection probes
  to the mounted API. Do not restore retired standalone browser journeys.
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

**Exit:** exact-candidate native-page and access evidence for this slice, with no
native workflow/security regression hidden by mocked responses. Native execution
and fixture/provider mutations require their own applicable authorization.

### 6. Rehearse and cut over separately

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

## Follow-up and limits

The requested **terminal** is a separate follow-up into the user's existing
project-local account/home, with bounded PTY/transport and session lifetime. It is
not required for initial SSH access and must not implicitly create, join, start a
project or expose a host shell.

No lifecycle controls, resource charts, member-management screens, environment
catalog, private-resource branching, generalized recovery or update platform.
[Architecture](architecture.md), [integration](forgejo-frontend-integration.md),
[deferred scope](deferred.md) and [licensing](licensing.md) remain authoritative.
Stock Cockpit and its native integrations remain operator-only; providers own CI.
Production `internal/host/` and `project-os/` own native behavior. Outside
[support tools](native-support.md) invoke owned checks, not a second readiness gate.

Local source checks were authorized; this plan grants no deployment, restart,
new fixture, provider action, routing or cleanup permission. Only historical bounded
**U08** proof is accepted—not Sodaspaces or final-product acceptance. Retained console,
operator/provider, licensing, native-support and independent aarch64 obligations
remain in the [handoff](implementation-status.md), not another expanded UI roadmap.
