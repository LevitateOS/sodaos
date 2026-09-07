# Sodaspaces implementation plan

Add one **Sodaspaces button beside Forgejo's repository actions**, opening a
**right-side shared-environment drawer**. No new tab or standalone page.
Both old Soda frontends and duplicate forge adapters are removed; the drawer and
its authenticated integration are **not implemented**. All steps below are pending.

## Selected approach

- **Backend:** existing Go API, SQLite, encrypted Forgejo OAuth grants and restricted
  native helper. Keep `soda-dashboard` and all persistent project/service identities.
- **Frontend:** stock Forgejo, two small custom-template hooks, browser `<dialog>`,
  scoped native styling and vanilla JavaScript using `fetch` with the JSON API.
  Reuse Soda's [template partials and shared CSS](../appliance/forgejo/README.md#presentation-component-contract).
  Lit is an option for a new self-contained interactive feature, including the
  drawer, when that feature justifies it. No Lit dependency or frontend build is
  added by the presentation extraction; vanilla JavaScript remains the current
  drawer candidate. Keep native forms/lists/scripts and Cockpit's separate stack.
- **Routing candidate:** existing Caddy, with only `/-/soda/` sent to the Go backend
  on Forgejo's existing HTTPS origin. All other native routes stay with Forgejo.
  This is a candidate to implement and verify, not an existing proxy/API contract.
- **Authority:** Forgejo owns identity, native sessions, permissions, Git keys,
  collaboration and administration. Soda owns its additional data and real
  environment/access integration—not copied roles or another password authority.
- **Environment model:** persistent and shared, with personal Linux accounts inside
  each project. Opening the drawer never creates, joins, starts or repairs anything.
  Sodaspaces is a UI name, not disposable Codespaces or a browser IDE.

## Presentation foundation

Standardize existing custom pages before adding new ones. The current local
Forgejo preview composes shared page intros, empty content and guest theme controls,
with explicit toolbar, form and native-list CSS adapters. Page styles own only
page-specific layout. This is a presentation system inside Forgejo's customization
surface; the authenticated drawer and appliance delivery below remain pending.
Do not convert native forms/lists to web components. Review each override and its
script-sensitive markup against the exact selected Forgejo version on upgrades.

## Delivery sequence

Keep each slice coherent, with its focused tests and affected guide updates in the
same change. The [API guide](dashboard-api.md) describes today's retained endpoints;
planned changes here do not claim those contracts already exist.

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

### 3. Deliver the read-only button and drawer

**New source:** `appliance/forgejo/templates/custom/{header,footer}.tmpl` and
`appliance/forgejo/public/assets/sodaspaces.{css,js}`—customization text only.

- Use the [verified native primitives and state table](forgejo-frontend-integration.md#minimal-button-drawer-and-loading-candidate).
  Render hidden Soda-owned markup from the footer hook; move only our button into
  `.repo-header .repo-buttons`. Missing row/context means no mount. Do not copy
  the repository header, add a tab or alter native buttons/listeners.
- Open a right-aligned `<dialog>` with a scrolling content wrapper. Show actual
  actor/repository context, checking/absent/existing/incomplete/unavailable states,
  sign-in, Close and safe refresh. Use native theme/form/loading/copy primitives,
  readable status, real disabled controls and `aria-busy`; keep Close usable.
- Keep JavaScript local to these elements: text/value assignment, checked JSON
  responses, bounded reads and stale-response rejection. Closing may abort reads,
  not cancel or undo a native operation. Reopening reads actual state.
- Stage the four files through `scripts/stage.py`; extend
  `internal/nativebuild/bundle.go` only for the required template paths/ancestors.
  Update build/packaging fixtures, inventory and notices together. Preserve existing
  branding and reject retired SPA payloads. Refuse conflicting operator hooks/assets
  before delivery rather than silently replacing them; no generic merge framework.

**Exit:** focused DOM/browser coverage for missing rows, initialization/reopen,
Escape/backdrop/Close/focus return, keyboard use, narrow/wide layouts, themes and
native navigation. Staging/verifier checks exercise real file paths. No mutation
controls are enabled yet; native rendering is not inferred from fixture markup.

### 4. Wire the explicit access actions

**Files:** the small drawer script/templates and retained Go environment/key handlers.

1. **Create shared environment:** change the existing POST to accept stable
   `repository_id`, resolving the current human owner server-side. No organization-
   owned creation or operator/admin impersonation. Preserve reservation-before-
   provisioning, uniqueness and honest incomplete results. Creation does not join.
2. **Save public key:** show registered development-key summaries; when needed,
   accept one public key through the retained key API. No private-key upload,
   native Git-key changes, key selector or automatic later propagation.
3. **Add me:** check the current actor's repository access for a new join, then use
   the real account/key helper and record membership only after confirmed success.
   Existing members retain their original login. Surface unsupported Linux names,
   native-account failure and result-persistence failure without automatic repair.
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
