# Current handoff

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus planned **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | Four native hook/assets now implement the read-only context/drawer candidate; both standalone frontends remain removed. Native browser proof and mutation controls pending |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v5 SQLite with unchanged grant encryption, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Last recorded `8b823db` dashboard/helper/runner companion/default new-project image; stock Forgejo 15.0.7. Historical React `/app/` preview and HTMX defaults remain installed |
| Acceptance | Only historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`). U01 architecture acceptance was withdrawn; no Sodaspaces/final-product/aarch64 acceptance |

Source removal is **not deployment**. No native image/stage build or installed
retest of the removal commits occurred. The read-only Sodaspaces source caller is
now authored, not installed or native-browser-proven. Root returns to configured Forgejo
home; OAuth can return to a freshly resolved repository under that origin using
single-use stored context and the acting grant, never a caller-supplied URL.
Schema v5 adds internal login cancellation contexts after v4's repository/expected-
user IDs. The historical return-path column remains unused; installed data was not
migrated. Old pending OAuth must restart; existing session/grant bytes are preserved.
New consent requests read user/repository/organization scopes, not administrator
expansion; actual existing grants remain intact. Redirecting is not native-session
transfer or cross-origin authorization.

Keep [API](dashboard-api.md), [credential migration](dashboard-credentials.md),
[architecture](architecture.md) and [current work](sodaspaces-plan.md) authoritative.
Acting grants/current native ownership from `fed66cb` remain in retained callers;
no setup-token, stale-creator or copied-permission fallback was restored.

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

The first real journey passed trusted raw proxy path/version/asset/revalidation
checks and sandboxed browser startup, then stopped on its own readiness assertion:
an anonymous drawer's empty data region has zero height in native CSS. Separate
sanitized native DOM observations confirmed a mounted button, visible dialog and
sign-in, completed ARIA busy state and no page-script errors. The probe now waits
for the visible drawer plus completed ARIA state, not a box on the empty region.
Probe `ebab030` then passed anonymous/native-cookie-only contexts but failed during
the first real OAuth sequence. The probe now records fixed route labels/status codes
and precise OAuth substeps, never URLs/queries/bodies or exception text.
Diagnostics at `5cbcdd8`/`fcedb04` showed real OAuth callback/repository return and
session/provider/repository reads succeeding. The assertion was in the probe:
Playwright route callbacks omit redirect hops, so its authorization counter and
redirect guard did not run there. The probe now uses CDP Fetch on its exercised
pages to inspect each hop before transmission; focused doubles test allowed and
refused hops. Separate native DOM diagnosis observed the matching actor and real
absent-environment state. Account switching/accessibility/BFCache and a complete
journey are still pending; these failures are not a milestone pass.

Probe `72a3c2b` passed the real first OAuth/absent state, cookie scope and protected
logout denial checks, then failed its overly strict keyboard assertion. Native
Chromium permits Tab to browser chrome (not background page controls); this correctly
invalidates/clears Soda. Native close/focus-return events are asynchronous. The probe
now permits that browser behavior while asserting no underlying-page focus, data
clearing and explicit reload, and waits for the actual close-event focus return.
Probes `e1c0c54`/`f3948c3` reached account switching after passing those checks and
360/1280-pixel light/dark layout. Native logout diagnosis identified competing link
and SSE navigations, not a Soda account error. Stock `SignOut` broadcasts logout;
its notification/stopwatch workers navigate session tabs home. The probe now checks
blur clearing before logout, waits for actual native sign-out without a competing
click-navigation waiter, and explicitly revisits the repository if upstream took
the old page away. Native workers, navigation and beforeunload remain unmodified.

The next attempts (`0576e60`/`fb6f32e`) stopped before logout because Playwright
forces both tabs focused/visible. Separate source-backed diagnostics reproduced
this in both prepared browser channels; another CDP session disabling focus
emulation does not undo Playwright's session override. The selected correction is
a stock Chromium process attached with public `connectOverCDP({noDefaults:true})`,
using a private Unix WebSocket/pipe rather than a TCP debugger. Sandbox/TLS stay
normal; no synthetic events or upstream worker suppression. A real two-tab diagnostic
with that launcher observed the background page unfocused/hidden and foreground
page focused/visible. Full-journey re-execution with this launcher remains pending. Full native build/stage, installed checks,
retained-state rehearsal and cutover remain unperformed.

## Remaining work and permission boundary

- Preserve the two implemented security fixes and regression coverage while wiring
  native-page context; no installed acceptance is inferred from local test results.
- Implement/prove the supported native button/drawer/authenticated Soda connection,
  then explicit create/join/key/connection controls and existing-account terminal.
  The verified template hook alone is not this integration. Stop if it needs a fork.
- Rehearse exact candidate/config/grants/populated-state preservation before a
  separately approved cutover; finish fresh/populated native product proof and
  independent native aarch64 validation. No current UI/final product is accepted.
- Complete console delivery/interactive proof, Tailnet and both providers' real
  runner journeys, intended-client routes, native branding and package/tool closure.
- Close [support-tool validation gaps](native-support.md#remaining-validation) and
  [actual-artifact licensing/source obligations](licensing.md). Optional media and
  incomplete outside helper ports are not product gates.

Local source builds/tests were authorized on this development machine. Existing
fixture/reboot grants have been used; there is no new permission for deployment,
restart, fixture creation, provider mutation, routing, destructive cleanup or other
targets. Check actual liveness/addresses only within the applicable scope; recorded
routes and agents are not promises of present availability. No new native actions
or evidence inspection occurred during this documentation cleanup.

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
