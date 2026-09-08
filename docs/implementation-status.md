# Current handoff

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus delivered **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | Explicit stable-ID create/key/join/own-connection controls passed bounded native x86_64 build/export/browser/Copy/SSH at `bdbce8e`. Copied private-v3 migration/paired rollback and separately approved retained cutover passed, with native browser and own-access observations. Both standalone frontends remain removed |
| Browser terminal | Native helper passed bounded x86_64 PTY/teardown/SSH-preservation proof. Protected browser transport and independent drawer terminal component/locked renderer packaging are source implemented and locally tested. Template mounting, genuine combined browser proof and deployment remain pending; installed helper unchanged |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v5 SQLite with unchanged grant encryption, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Built `bdbce8e` dashboard/strict-config runners CLI, native hooks and namespaced proxy/config on `soda-test`; schema v5 and stock Forgejo 15.0.7. Unchanged helper/default project image/other native components retain prior `8b823db` provenance; old frontends are no longer served |
| Acceptance | Historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`); Sodaspaces steps 5–6 have passed bounded execution and approved cutover. U01 architecture acceptance was withdrawn; no final-product/aarch64 acceptance |

Source removal is **not retained-appliance deployment**. `ee8091a` passed bounded
read-only delivery/browser proof; `bdbce8e` subsequently passed native create/key/join/
Copy/SSH on the separate fresh fixture. Separately approved `soda-test` cutover then
delivered those affected payloads with recorded configuration and schema v5. Root returns to configured Forgejo
home; OAuth can return to a freshly resolved repository under that origin using
single-use stored context and the acting grant, never a caller-supplied URL.
Schema v5 adds internal login cancellation contexts after v4's repository/expected-
user IDs. The historical return-path column remains unused. Both copied and live
v3 → v5 migration preserved original rows/ciphertext before login. Old pending OAuth
must restart; subsequent normal expiry/login/logout changes session/grant rows.
New consent requests read user/repository/organization scopes, not administrator
expansion; actual grants, not requested scope names, govern authority. Redirecting is not native-session
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

**Complete scoped journey passed at probe `dd793a2`**, against unchanged `31e73bf`
backend/UI bytes. `browser-x/sodaspaces-run/result.json` records real BFCache
restoration, both users' real OAuth returns, native-only/Soda-only transitions,
identity mismatch/no premature environment reads, protected logout actor/CSRF
denials, cookie scope, native unsaved-form preservation, keyboard/blur/explicit
reload, Escape/backdrop/focus return and 360/1280-pixel light/dark layout. Raw
proxy path/encoding denials and exact served asset hashes/conditional revalidation
also passed. Only **absent** environment views were native (three observations);
existing/running/stopped/incomplete views remain source/DOM fixtures in this run.
No responses or sessions were faked. This is bounded browser integration evidence,
not milestone/release acceptance or installed CoreOS/aarch64 validation.

`artifact-binding.json` binds the running backend's `/proc/1/exe` hash to its local
Go/VCS build and the four mounted/served UI hashes, with exact cached image IDs.
Read-only inspection of Soda's own fresh schema-v5 database found zero projects,
memberships and development keys. It did not inspect Forgejo's database. Services,
all failed containers, private profiles, inputs and logs remain retained. No real
stage/image build, installed checks, retained-state migration or cutover occurred.

The failed attempts remain under `browser-*`/`logs/`, with their exact revisions.
They exposed probe assumptions rather than requiring product/upstream changes:

- Empty native data regions have no visible box; wait on dialog/ARIA readiness.
- Playwright routes omit redirects; CDP Fetch guards every exercised-page hop.
- Playwright forces pages focused/visible. Stock Chromium now attaches with public
  `connectOverCDP({noDefaults:true})` through a private Unix socket/pipe, retaining
  sandbox/TLS and real focus/visibility/BFCache rather than synthesizing events.
- Native chrome/body focus transitions invalidate Soda; focus return is asynchronous.
- Forgejo's stock logout broadcasts navigate session tabs home and its link action
  posts `/-/fetch-redirect`. Only the exact navigation-only `redirect=/` form is
  allowed. Workers, upstream navigation and native beforeunload stay unmodified.
- BFCache history restoration needs a commit wait, not a fresh load-event wait.

Final local checks passed: full uncached Go suite, 46 Node tests (43 DOM plus three
probe/transport checks), 36 Python build fixtures, Node/bash syntax. Logs are under
`logs/final-*`. Go used cached 1.26.7 with readonly/offline modules. No Cockpit
retest, aggregate native-stage check or additional dependency install was implied.
Transport cleanup/refusal hardening also passed the full native journey at
`2d6a2b3`: `browser-y/sodaspaces-run/result.json` and its exit record. The final
`artifact-binding-final.json` ties that probe to unchanged product bytes; prepared
browser/tool identity and probe source hashes are retained separately. Both full
passes observed real BFCache. Documentation checks covered 55 Markdown files,
240 local links and 33 anchors with no errors; whitespace checks passed.

## Read-only native stage and exported delivery closure

The user gave standing implementation/testing approval for this planned work.
Executed the production `scripts/build-native.sh x86_64` and
`scripts/check-native.sh x86_64` in a fresh clean detached `ee8091a` worktree at
`.artifacts/worktrees/stage-ee8091a/`. Both exited 0. This built all native commands,
Cockpit, Tea, project/dashboard images and the four OCI archives, fetched locked
runner inputs, staged and sealed the actual payload. No placeholder stage was used.

The aggregate verified that exact stage before/after checks: Go suite, 46 Node
tests, Cockpit TypeScript and 60 tests, 36 build fixtures and **11 actual-stage
checks** passed. Separate uncached web/store/Forgejo/config/nativebuild races passed.
The exported bundle at `.artifacts/stage-validation-ee8091a/export/x86_64/` verified
before and after browser use. A mistaken first verifier invocation at the bundle
root failed with exit 127; its log remains. The correct `tools/soda-artifacts`
invocations passed. No source/product correction or dependency-baseline edit was
needed. Actual mutable package resolution is recorded by the build; the new project
image is not granted the older installed runtime's lifecycle/SSH/workload acceptance.

A **new** retained local delivery fixture, `.artifacts/delivery-ee8091a/`, used the
export's readonly templates/assets/branding and Caddy recipe, stock Forgejo/Caddy
images matching exported OCI config IDs, and the **built dashboard image at its
configured UID 2000**, read-only root with no capabilities. Rootless shared network/
user namespaces and only `127.0.0.1:32443` published; no builder appliance install.
Fixture-only configuration, new native users/repository/OAuth client and TLS were
initialized through native CLI/official APIs. No helper is connected. The previous
local fixture and retained `soda-test` installation/projects were untouched.

The exact `ee8091a` product-owned browser journey exited 0 against that delivered
payload: both users' real OAuth/consent/identity/cookie/logout flows, real native
blur/stale-tab and BFCache restoration, native unsaved form preservation, keyboard/
backdrop/focus and responsive light/dark rendering passed. Native `soda-auto` default
and its served stylesheet were separately verified against the export. Three native
environment observations were **absent**, not running/stopped/incomplete proof.
Executed backend `/proc/1/exe` matches the exported binary; runtime image IDs match
exported OCI configs. Read-only Soda database inspection found fresh schema v5 and
zero projects, memberships and development keys; no Forgejo database inspection.

Evidence: `.artifacts/stage-validation-ee8091a/` (build/check/race/export logs and
manifest) and `.artifacts/delivery-ee8091a/` (private native result, artifact/image/UID/
port/theme bindings, inputs and profiles). Both fixtures' services, all worktrees,
outputs and failures remain retained. **Step 3's bounded x86_64 read-only exit is
satisfied; step 4 is next.** This is not first-install/activation, retained-state
migration/cutover, existing-project access, full product/release or aarch64 acceptance.

## Access-action plan revision

Documentation-only review after `afda2d9` reconciles the leading plan and API guide
with the completed read-only build/export/browser evidence and standing testing
approval. Step 4 now explicitly separates action-time identity checks, pending writes,
late/stale results, confirmed versus uncertain outcomes and safe read-only observation
without replay. Existing-member idempotency/degraded access and current-owner/new-join
server authority remain intact. Step 5 extends the existing guarded native journey
with bounded access requests on an isolated helper-backed target; neither retained
browser fixture supplies provisioning or SSH proof. No new roadmap, backend contract,
helper protocol, recovery subsystem or live cutover is implemented by this revision.

Checks for this revision: documentation inspection passed (55 Markdown files,
242 local relative links, 34 Markdown anchors, zero errors); diff-whitespace checks
passed. No product tests, build, native execution, fixture mutation or data cleanup.

## Explicit access actions source implementation

Implemented step 4 after `f940338`, without a new frontend stack, helper protocol,
permission inventory or recovery subsystem:

- Create accepts only canonical decimal-string `repository_id`. The existing
  `visibleRepository` path checks actual user/repository consent, fresh subject and
  stable-ID visibility; the acting human must be the current owner. Reservation
  precedes native provisioning, uniqueness survives concurrent requests and create
  never joins. The unused owner/name Forgejo adapter is removed. The API's existing
  bounded decoder now uses `strictjson` to reject duplicate fields/invalid UTF-8 too.
- The existing four hook/assets add separate Create, development-key summary/public-
  key save, Add me and own SSH connection controls. Public-only input is checked
  before transmission and by the retained Go SSH parser. Save never joins or updates
  existing Linux keys. Existing-member original logins and degraded API access remain.
- Each explicit action rechecks page/session/provider consistency, sends one protected
  POST, disables duplicate dispatch/logout while pending and safely rereads afterwards.
  Close/blur/BFCache cannot cancel or replay native work. Close/backdrop invalidate
  synchronously before the native queued close event; late writes cannot populate a
  reopened/stale drawer. Uncertain create/join stays blocked in that document with
  operator-inspection guidance, not a claim that a missing row proves no native effect.
- Own connection rendering validates association, original login, current running
  address and public host-key/fingerprint fields; stopped/unavailable/stale views
  clear the command and Copy target. Copy delegates through stock 15.0.7's inspected
  `clipboard.js`, not a replacement handler. Address display is not routing proof.

Performed under standing local testing approval: full uncached Go suite; uncached
web/store/Forgejo/config/nativebuild races; **89 Node tests** (86 drawer DOM cases and
three retained probe/transport cases); **36 Python build fixtures**. All passed.
Source checks include owner transfer/admin non-bypass, large/invalid IDs, provider/
consent/actor denials, concurrent reservations and native/persistence failures,
separate actions, private/options/multiple-key refusal, pending/stale/queued-close
results, no replay or false membership and unavailable connection/Copy behavior.
Earlier focused passes and final logs are retained in
`.artifacts/research/access-f940338/`. Documentation/whitespace checks accompany the
change; no new dependencies or baseline versions changed.

This is handler/store/DOM and packaging-fixture evidence, **not** a new real image/
stage/export, native browser/Copy interaction, helper account/key installation, SSH,
project-runtime or aarch64 pass. No Cockpit retest, fixture service restart, helper
connection, retained database/config/OAuth migration, VM/project change or cutover
occurred. Both browser fixtures, `soda-test`, all four retained environments, earlier
worktrees/artifacts and private credentials/evidence remain untouched. Step 4's local
exit is satisfied; step 5 must bind updated delivered UI/backend/helper/project bytes
to real account/key/SSH results before native access is claimed.

## Phase-5 execution started

The user requested phases 5 and 6 after `658f2af`. A fresh native x86_64 KVM fixture
`soda-native-spaces-658f2af` is retained under `.artifacts/access-vm-658f2af/`, with a
new overlay over the preserved CoreOS base, localhost SSH 22230/browser 33443,
private generated host/client keys/password/Ignition inputs and the existing operator
public key. Native extension layering completed; one fixture-only activation reboot
was requested. The first SSH observation incorrectly assumed Python existed before
activating the layered deployment; exit 127/broken-pipe evidence is retained. The
retained `soda-test` hostname, architecture and application/helper service status were
read only; no retained service, configuration, callback, database or project changed.
An initial socket check used the wrong path; the actual `/run/soda/host.sock` is
root:soda 0660 and its socket unit is active.

The existing native browser journey now has an explicit, single-use actor/path/body-
bound access mode; the retained SSH/PTY/SCP/SFTP probe takes declared Sodaspaces
connections instead of historical fixed U08 fixture names. Stock 15.0.7's native
Copy tooltip appends to the document body by default; Soda's Copy button now uses
its supported `data-tooltip-appendto="parent"` attribute so feedback stays inside
the modal top layer. This is source-backed preparation, not a completed native Copy
or access pass. Phase 5 will use a separate fixture-local client network namespace;
no builder routing change or laptop-route proof is implied. Phase-6 preserved-state
rehearsal and live cutover have not begun. Local source checks passed: 89 Node tests,
37 Python build fixtures and focused Go scripts/nativebuild tests, plus documentation
links/anchors and whitespace. They do not establish a native access result.

## Phase-5 bounded native access proof

Candidate **`bdbce8e736b26dfaf81c37b1917386a536ef1fac`** passed production native
x86_64 build/check/export from its clean detached worktree, including full Go,
89 Node tests, Cockpit TypeScript/60 tests, 37 Python fixtures and 11 actual-stage
checks. Separate uncached web/store/Forgejo/config/nativebuild races passed. The
transferred verifier checksum and bundle inventory were checked before first
installation on the new CoreOS fixture; `verify-installed` subsequently passed.

The new fixture completed native layering/activation reboot, first installation,
native Forgejo setup, production `soda-setup` and `soda-activate` with private TLS.
The old initialization recipe wrongly expected a redirect: stock 15.0.7's
`InstallDone` returns HTTP 200 after committing installation. The failure is retained;
native CLI inspection confirmed only Alice existed, then a separately guarded
continuation created Bob/OAuth/repository without replaying installation or replacing
state. Public host-key inspection first omitted the production `soda-` container
prefix; no container was changed. A diagnostic image comparison initially failed on
Podman's omitted `sha256:` prefix; exact digest comparisons then passed. All failed
observations remain, not relabelled as native defects.

The exact candidate's real sandboxed/TLS-trusted browser access journey passed:
real BFCache/account/cookie/logout/stale/native-form checks, nonowner create denial,
one owner-created environment **`p4a530c394bcd53e563d3076d`**, separate Alice/Bob
public-key saves and real helper-backed joins, own connections, native Copy success
feedback and actual clipboard paste for both users. No Soda sessions/grants or
membership rows were seeded and no native response was substituted. Observed states
were absent and running; stopped/incomplete/uncertain-result branches retain their
source/DOM evidence, not invented native observations.

The separately named `soda-phase5-client` used a normal bridge network namespace
inside the fixture, read-only root, no capabilities and no-new-privileges. Both users
passed direct **10.90.0.2** SSH, interactive PTY, bidirectional SCP/SFTP, project-root
UID-map separation and expected owner/nonowner sudo behavior; Alice's key was denied
for Bob's login with a real public-key denial, not transport failure. Host trust came
from independently read public project host-key bytes, compared with browser output;
private client keys were never uploaded to Soda. All probe directories remain.
This proves that fixture-local client path, **not builder/laptop/Tailnet routing**.

`verify-installed`, runtime image IDs, the backend's actual `/proc/1/exe` and the new
project's image/labels match the export. The native schema-v5 Soda DB has one project,
two memberships and two development keys, with integrity checked. Source under
`internal/host/`, `cmd/soda-host/`, `project-os/`, `internal/runners/` and
`cmd/soda-runners/` is unchanged from installed `8b823db`; compiled provenance and
mutable image package resolution remain distinct from source equality.

Evidence: `.artifacts/stage-validation-bdbce8e/`, retained worktree
`.artifacts/worktrees/stage-bdbce8e/`, and `.artifacts/access-vm-658f2af/` (browser-a
result, exact artifact binding and copied client results). The VM/overlay/base,
project/accounts/keys, services, exited client/conversion containers, private inputs,
profiles and failures remain retained. Phase 5's bounded x86_64 access exit passed;
this is not whole-product/operator/workload/lifecycle/aarch64 acceptance.

A follow-up audit found five OAuth-state query lines in the fresh fixture's default
Forgejo router journal; no code/token parameter lines were observed by that bounded
audit. No raw journal or parameter values were emitted. Supported native logging
configuration now disables the query-bearing router logger and retains console
method/escaped-path/status access records. Native effective settings and a repeated
real read-only/OAuth/BFCache journey (`browser-c`) passed, with query-free OAuth access
records and zero audited credential/state query lines afterward. General service/error
logging remains native. Earlier journals and failed recipe diagnostics are preserved.
The default is also authored in `appliance/config/forgejo.env`, with a focused template
check; this is an explicit configuration follow-up to the built `bdbce8e` images,
not a claim that a later source revision was rebuilt or installed wholesale.

## Phase-6 preserved-state rehearsal

Evidence is retained in `.artifacts/phase6-658f2af/`. At rehearsal, `soda-test` was schema v3
with 3 profiles, 2 keys, 4 projects, 7 memberships, 10 sessions and 9 encrypted grants;
all four project roots were running and exact Soda hooks/assets absent. Existing
browser tunnels and trusted Forgejo TLS work. Only its original Soda service was
stopped for a SQLite backup plus matching private config/key/artifact capture and
resumed unchanged. The resume recipe initially checked the wrong mutable `:dev` tag;
inspection established the actual image-pinned Quadlet, which was resumed and verified
against the original image/config and added to the backup. No live schema, callback,
configuration, helper, project, default project image or runner-service change occurred.

The consistent private set is `/var/lib/soda-sodaspaces-bdbce8e/backup` on `soda-test`
and `.artifacts/phase6-658f2af/backup/` on the builder. Copies on the fresh fixture
(`/var/lib/soda-phase6-rehearsal/`) ran the actual prior and `bdbce8e` images with
network=none, no helper mount, UID2000, no capabilities, read-only root and explicit
copy-only writable data. The first negative recipe expected an unsanitized key error;
production correctly emitted its sanitized startup-stage message. That attempt is
retained; separate reviewed copies then passed all eight cases:

- legacy config and missing/wrong key refusal, leaving copied v3 bytes/rows unchanged;
- successful v3 → v5 migration preserving every original column/row and encrypted
  grant/key-check byte, with integrity/FKs and all migrated session contexts checked;
- healthy empty v5, future-version refusal, prior-image refusal of migrated v5;
- healthy paired prior-image/config/key/v3-copy rollback, preserving all original data.

This is actual copied private-state/native-image evidence, not a live rollback or
permission to restore an old backup after later writes. Only exact run-owned rehearsal
containers were stopped; all copies, failed/exited containers and original roots remain.

Official acting-owner API reads confirmed retained OAuth application 4, its unchanged
client and sole prior `https://localhost:24443/oauth/callback`; the planned callback is
`https://localhost:24444/-/soda/oauth/callback`. No application PATCH or Forgejo DB access
occurred. Retained `/u08-alice-8417/shared-alice` is private and Issues-enabled; it must
not be made public to fit a probe. The declared read-only private-repository probe
variant requires native anonymous 404/no Soda reads and then the normal authenticated
journey, never environment/key writes. That variant passed as probe revision
`0992f20ab295c1199ba58ccf34ac377012d67b9c` against the fresh fixture's built `bdbce8e`
images plus tested query-free logging configuration (`browser-d`). A new synthetic
private repository with native Alice ownership/Bob read access was used; no existing
repository visibility changed and no new Soda environment was created. Its three
absent observations and real BFCache are not retained-target running-view evidence.
Official reads confirm retained Alice ownership/Bob write access already exists.

Final retained observations match every preflight field, all four container/image/
running-state identities and every original Soda table row against the backup at that
observation. Key fingerprint and original-login shapes fit the drawer contract. The
fresh fixture still passes installed/runtime artifact binding with exactly one project,
two memberships and two development keys. No later backup should be assumed current.

Follow-up local checks passed: uncached full Go tests, **91 Node tests**, **37 Python
build fixtures**, JS syntax, whitespace and documentation links/anchors. Evidence is
`.artifacts/research/phase6-bdbce8e/`. No new compiled backend/helper/UI payload or
whole native bundle was built after `bdbce8e`; the follow-up changes are logging
configuration, probe/tests and documentation.

**The user subsequently approved live cutover.** The
[affected-component procedure](installation.md#retained-sodaspaces-cutover) covers
fresh-at-cutover backups, owner-native callback editing, image-pinned backend,
strict-config `soda-runners` CLI, proxy/namespace/hooks and query-free native logging.
Unchanged helper, project roots/default image and runner services are not upgrade
targets. See the executed cutover below.

## Approved retained cutover

The user explicitly approved the documented affected-component cutover after the
rehearsal. Evidence is under `.artifacts/cutover-c007eb6/`; the fresh private backup
on `soda-test` is `/var/lib/soda-cutover-c007eb6/backup`. The old rehearsal backup was
not reused as current state. The actual image-pinned unit, prior image, consistent
SQLite data, credentials/key/config, proxy, native configuration and file metadata
were preserved. An initial SCP transfer could not preserve two relative bundle
symlinks; that partial tree remains untouched. A separately named tar transfer passed
the production bundle verifier before deployment.

Built `bdbce8e` dashboard image/binary and strict-config runners CLI, exact four native
hooks/assets, namespaced proxy/config and reviewed query-free logging were delivered.
The actual owner changed only app 4's callback through native Applications settings;
client identity/name/confidential setting and credential-file bytes were preserved.
No API PATCH, secret generation or Forgejo DB access occurred. Only affected services
were restarted; no helper/project/default-image/runner-service or routing change.

Live schema v3 → v5 passed integrity/FK checks and preserved every original column/row
and grant/key-check ciphertext **before browser login**, with migrated contexts checked.
The running backend executable/image match the export. Native runner `list` succeeded.
The retained host lacked the selected static-cache setting (native default six hours);
asset validation caught it. `STATIC_CACHE_TIME=0` was then applied, Forgejo restarted
and actual conditional reads returned 304. Earlier input/cache/startup-preflight
failures remain, not overwritten. A restricted CA copy was used without changing the
original CA file or global trust.

The real private-repository browser journey passed at probe `c007eb6` (`browser-d`),
with three running views and genuine BFCache plus normal authentication/identity/
logout/form guards. No environments, keys or memberships were written by the probe.
The follow-up probe `44819462d52de86fe8d40e3b278ec26ce48b942a` also passed on the
delivered target (`browser-e`), recording both users' usable displayed own commands/
fingerprints, three running views and genuine BFCache. Those displayed values match
original membership logins and independent operator public-host-key observations.
Native `ssh-keygen` independently matched fingerprints for all four roots.

All **seven existing memberships across four projects** then passed direct own-key
SSH identity and PTY checks from `linux-infra.dimensionlab.net`, using the unchanged
`tun8417` route via `169.254.84.2`. No keys/accounts were created or updated, no project
files were written by the probe commands, and no project lifecycle/routing action was
performed. This is infra reachability, not laptop/Tailnet proof. Native private Git
HTTP advertisement at the unchanged Forgejo origin passed using the real native Alice
credential, without retaining its body or exposing the credential.

Final checks preserved all original profile/key/project/membership/key-check rows,
four container/image/running-state identities, helper bytes, native Forgejo/proxy units
and credential/TLS bytes. Affected files and running backend match the export. The old
Soda listener is absent; a preserved old SSH forward may still bind locally but no
longer serves Soda. Native query-free request/OAuth logging was observed with zero
audited credential/state query lines. Normal authentication/expiry/logout left two
sessions/grants; the pre-login check had preserved all ten sessions/nine grant rows.
Fresh paired backup is also retained at `.artifacts/cutover-c007eb6/backup/`.

The scoped phase-6 exit passed. **92 Node tests, 38 Python build fixtures**, JS/Bash
syntax, documentation and whitespace checks passed. The VM web-tunnel wrapper now
advertises/forwards only native 24444 on future invocation, with a fake-SSH regression;
no running tunnel was changed. No new Go compilation/native build
was needed or performed in this cutover turn. This is an affected-component deployment
of `bdbce8e` artifacts plus recorded configuration, not a wholesale install of the
later probe/document revision. Full product/provider/aarch64 acceptance, console and
laptop routing remain separate. All old/fresh fixtures and failures are retained.

## Browser terminal plan

The next concrete item is planned in the existing
[Sodaspaces plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal), not
another roadmap. Candidate: local terminal renderer, same-origin authenticated
WebSocket and one fixed Unix-helper operation into an existing project account.
Native PTY/account/owned-process teardown proof comes before UI wiring; Podman client
exit is not assumed to terminate container exec. No private-key collection, automatic
join/start, host shell, project-image replacement or durable terminal sessions.

Inspected current API/session/helper/provisioning owners, upstream Podman v5.8.2
exec source, terminal package metadata/types and Go WebSocket/PTY documentation.
Research is retained in `.artifacts/research/terminal-plan-6206578/`; the initial
Podman manual URL failed, then the correct `.md.in` source was retrieved. No native
command, product test, build, dependency installation or deployed-state change ran.
Only planning documentation and link/whitespace checks changed; the terminal remains
unimplemented. Native helper changes and retained rollout need their applicable scope.

## Native terminal boundary source

This records the initial `10321ce` source checkpoint; approved native execution is
recorded in the following section.

Implemented the first source slice of the [terminal plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal):
fixed embedded project-local Python PTY launcher, bounded private Unix WebSocket
operation/client, immutable-container-ID/namespace checks, marker/account validation,
credential dropping, framing/backpressure, heartbeat expiry and owned-shell teardown.
Streaming does not hold the mutation lock or inherit the buffered RPC timeout. Helper
shutdown now cancels and waits for pending/hijacked streams. No public API, drawer
terminal, native service change or project-image modification was made.

Coder/websocket v1.8.15 was genuinely resolved, without other dependency upgrades;
its ISC license text is included in the already-bundled root NOTICE. Local full Go,
focused host/command races and **49 Python build tests** passed. The new process tests
exercise actual local PTYs, resize/Ctrl-C, EOF/final output, silent-peer expiry and
slow-consumer cleanup with an unprivileged clean test shell; credential dropping is
unit-tested, not native project proof. No host accounts were created. The opt-in
`TestInstalledTerminalBoundary` is authored but skipped without private native input;
it uses a temporary root-private helper, not an installed service replacement, and
records only its bounded account/TTY/explicit-close scope. Lost-helper, unrelated
SSH/workload preservation and full browser evidence remain separate required checks.

Evidence: `.artifacts/research/terminal-native-163ccf9/`. At this initial checkpoint no
VM/installed/helper service, retained project, key, membership, routing or provider
state changed. UI wiring was held until the actual native gate; the separately
approved proof below now closes that bounded gate, not browser delivery. Retained
`soda-test` rollout still needs separate approval and a current backup.

## Approved native terminal fixture proof

The user approved temporary root-private helper/PTY proof on the existing isolated
`soda-native-spaces-658f2af`, without installed service replacement/restart, project
lifecycle/account/key or routing changes. Preflight independently confirmed both
native accounts, marker IDs, groups/homes, the existing container/image and unchanged
installed helper/services/product rows. No terminal had launched at this observation.

Actual Podman is **5.8.4**, not the builder's 5.8.2. Native preflight caught two source
assumptions: Go templates require `.ID` inside `json`, and auto-created namespaces
are reported as `private`. Inspected exact upstream source and observed UID/GID maps
`0:1000000:262144`; the candidate now checks private mode plus actual shifted mappings,
not just a create-time mode name. No runtime configuration/capability was changed to
fit the check. Original failed preflight and reviewed inputs remain under
`.artifacts/terminal-vm-10321ce/` and guest `/var/lib/soda-terminal-10321ce/`.

**Bounded native gate passed.** Final native test binary from `fae1696`, with verified
builder/guest SHA256 equality, passed on existing Rocky project
`p4a530c394bcd53e563d3076d`: both original accounts' real/effective/saved UID/GID,
supplementary groups, HOME/cwd, real PTY, resize/Ctrl-C, current sudo boundary and
shared mise/project-Podman profile settings. A mismatched marker/actor was refused.
Explicit close and independent observations proved actual owned login termination.
EOF, a real silent 60-second lease and SIGKILL of only the test-owned temporary helper
also ended the login, foreground job and project-local launcher. Final observation
times were approximately 0.19s, 60.18s and 0.20s respectively—not merely socket-close
or host-CLI exit observations.

Both users' ordinary own-key SSH processes survived throughout (14 paired observations
in final `run-e`), over the existing management SSH direct-TCP forwarding path to
project `10.90.0.2`; no laptop/direct-builder-route claim. The earlier bridge client
was already stopped and was left untouched. Original preflight and `run-a` parser
failures remain; the latter reached a native PTY but did not recognize Bash CSI/CR
output. The probe parser, not the shell/profile, was corrected. Successful earlier
`run-b`–`run-d` observations and all new private inputs/binaries/results remain.

Final verification confirmed unchanged container/image/running identity, installed
helper bytes and affected native service PIDs/start times, product rows and native
marker/accounts/groups, SSH host key and DB integrity. No candidate helper/launcher
remained. No installed service replacement/restart, account/key/lifecycle/provider or
routing change occurred; ordinary shell/sudo bookkeeping was permitted, not claimed
absent. `soda-test` was not contacted. Full browser authorization/transport, logout
races, drawer/renderer/packaging and genuine browser proof are **still unimplemented**;
proceed with plan step 2, not a new architecture or installed rollout.

Final local regressions passed: full uncached Go suite, host/command race tests,
49 Python build tests, documentation links and whitespace checks. The default local
suite skips the explicitly opted-in native probe. Only the native package test
binaries were built/transferred; no whole-appliance build/stage/export, renderer
dependency installation or deployment ran during this proof.

## Protected browser terminal and independent drawer component

Implemented `internal/web/terminal.go`: exact-origin/query/subprotocol/fetch guards,
pre-upgrade session/own-membership checks, bounded first-message actor/repository/CSRF,
fresh acting-provider consent/visibility, original membership login, one pending/live
slot per login-context/project and bounded frame/queue/write/lifetime handling. Native
dispatch and lease renewal serialize with local logout/rotation; pending authorization
is cancelled too. Session reads now expose their existing minimum expiry internally
(no migration). Shutdown closes hijacked streams before DB shutdown. No browser
heartbeat, automatic join/start/reconnect or copied provider authority was added.

The self-contained `sodaspaces-terminal.js`/CSS component supplies explicit Open and
Disconnect, local lazy xterm/fit, bounded Unicode input/output, suppressed OSC clipboard/
link/title actions and full-page stale/reload behavior. It neither discovers nor edits
Forgejo markup. **The other agent owns all template overrides/layout**; the small
[mount/dispose contract](terminal-integration.md) is the only integration surface.
Existing hook templates, navigation and original drawer implementation were untouched.
Host template mounting remains intentionally unwired, not a second standalone UI.

Actual npm archive integrity and per-file hashes pin xterm 6.0.0/fit 0.11.0. Native
build/stage/bundle/first-install source now carries seven new component/distribution/
MIT-notice files, verifies upstream hashes and refuses occupied exact destinations.
This adds no bundler/CDN/runtime download or arbitrary-template adoption. Local
fetches wrote only ignored `.artifacts/browser-terminal/vendor`.

Local full Go, focused web/host/store/command/nativebuild races, **104 Node tests**
(including standalone DOM/renderer doubles) and **51 Python build tests** passed.
New server cases exercise malformed first auth, actor/association/CSRF/provider and
pre-upgrade denials with zero helper calls, original login, duplicate refusal, pending
and active logout/shutdown, OAuth rotation, logout during fresh authority, and bad
controls/browser-heartbeat refusal. DOM cases cover inert mounting, Unicode, scoped
keyboard handling, stale/late events, disposal and bounded renderer backlog. Packaging
tests exercise the real stage/preflight in synthetic temporary trees; they are not a
new real native-stage result. Evidence: `.artifacts/browser-terminal/`.

No VM/installed service, retained project, account/key/provider or routing action ran
in this source turn. No template override was changed, whole-appliance bundle built,
Chromium/native OAuth journey executed or deployment performed. Real combined
browser/proxy/helper lifecycle proof and native candidate delivery remain required;
prior native-only proof is not public endpoint acceptance. Continue the documented
mounting coordination and integrated proof, not a retained rollout or template fork.

## Remaining work and permission boundary

The subsequent remaining-work planning pass inspected production helper routes and
`soda-project@.service`, then reconciled the short list and immediate terminal exit
checks. Documentation/link/whitespace checks only; no source behavior, dependency,
build/test execution, target state or lifecycle permission changed in that pass.

- Preserve the implemented security, native-page context and explicit-action
  regressions plus steps 5–6's bounded native delivery/access evidence. Keep read-only
  guard mode separate from explicitly bounded writes; future maintenance needs its
  own exact scope/current backup, not replay of the recorded cutover.
- Finish native template mounting and genuine browser/proxy/helper proof for the
  already implemented terminal source; do not restart its completed native-boundary
  work. The template/layout agent owns the overrides and uses the component contract.
- Follow the single [ordered remaining-work list](sodaspaces-plan.md#remaining-work--ordered):
  terminal integration, basic Start/Stop, the explicit Destroy scope decision, runner
  settings, operator/client gaps, whole-candidate validation and approved delivery.
  Start/Stop have native systemd/Podman mechanisms but no current helper/API/UI controls;
  deletion is still deferred. Planning these is not authorization for lifecycle or
  destructive execution.
- Move Soda's local runner capacity/service configuration into operator-only settings
  in the unified native SodaOS/Forgejo interface, as subsequently selected by the
  user. Inspect official administrator extension points and reuse backing logic/tests;
  retain the Cockpit Runners page until a working replacement and coordinated removal.
  Tailnet stays in Cockpit; provider authority and the Soda operator boundary remain
  unchanged. This decision is documentation-only so far, not implementation/deployment.
- Finish full fresh/populated product and independent native aarch64 acceptance;
  scoped x86_64 delivery/browser/SSH results are not final-product acceptance.
- Complete console delivery/interactive proof, Tailnet and both providers' real
  runner journeys, intended-client routes, native branding and package/tool closure.
- Close [support-tool validation gaps](native-support.md#remaining-validation) and
  [actual-artifact licensing/source obligations](licensing.md). Optional media and
  incomplete outside helper ports are not product gates.

Standing implementation/testing approval now covers this planned work; local native
build/stage/export and isolated delivery testing proceeded under it. Preserve all
retained roots, credentials and evidence. It is not an instruction to erase data,
change unrelated provider/host-network resources or silently cut over `soda-test`.
Recorded routes and agents are not promises of liveness.

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
