# Implementation handoff

## Current local verification after execution approval

The user's “go until finished”, following the listed execution gates, authorized
local build/test work. On the x86_64 workstation (Go 1.27.0, Node 24.20.0,
pnpm 11.25.0), executed `go test ./...`, dashboard `vp run check`, `vp test --run`
and `vp build`: all passed. The dashboard suite ran 16 tests across 10 files.
The first UI run exposed two unavailable Jest-style assertions; these now inspect
native DOM properties. Restored the already-selected Markdown dependencies missing
from the manifest and resolved the real lockfile with lifecycle scripts disabled.
Logs are in ignored `.artifacts/logs/{all-go-tests,dashboard-typecheck,
dashboard-ui-tests,dashboard-build,dashboard-install}.log`.

This supersedes earlier **local** unbuilt/unexecuted statements below for the
checked source. It is not a full native artifact build, installed verification,
Go 1.26 baseline run, aarch64 proof or U milestone completion. No VM, service,
provider resource, project state or networking was changed. Deployment/migration,
developer routing and installed workload/persistence acceptance remain pending.

## U15 release/asset source and local verification

Connected native release list/detail/create/changed-field edit and bounded asset
upload/download. Creation requires an explicit draft state (the UI defaults to
draft); duplicate tags and upstream permission errors are not retried. The pinned
release PATCH ignores empty title/body replacements, so Soda rejects those rather
than claiming a clear. Asset downloads first resolve repository/release-scoped
metadata and then use only the fixed native attachment UUID path with the acting
grant. External URLs and redirects are never credential-forwarding targets.
Uploads are bounded to 32 KiB; inert downloads to 8 MiB before browser headers.
No Soda artifact store or release records were added.

Full local Go suite, dashboard type check, existing 21 UI tests and production
build passed; `.artifacts/logs/u15-releases-*.log`. Focused Go tests cover duplicate
creation, changed-field edits, ignored clears, scoped download denial, redirects,
lossless attachment IDs and invalid uploads. Release-specific DOM and installed
provider proof remain pending, as do wiki/packages and the rest of U15–U20.
No release, tag or asset was created on a real provider.

## U14 source and local verification; U08 access observation

Connected native Actions runs/detail, repository tasks, recursive workflow-file
selection and explicit dispatch; repository/organization secret and variable
configuration is native, with write-only secret values and no Soda persistence.
Run/event payloads and runner credentials are excluded from DTOs. Run polling
retains one response and stops on navigation, logout, terminal status or failure.
Run/task pagination reads the native public API cap because those endpoints
return body totals without Link headers. Workflow discovery retains the pinned
first-directory rule, native recursive tree pagination and a 30-second/10,000-entry
bound. Provider denial is not retried with another authority.

Pinned web run routes exist for logs/artifacts/cancel/rerun, but OAuth2.Verify
explicitly excludes those web paths. No runner protocol, password, or borrowed
cookie adapter was introduced. Native-login fallback remains an explicit U17
disposition, not completed parity or user acceptance of the gap.

Executed local full Go suite, dashboard type checking, 21 tests in 12 UI files
and production build successfully; logs `.artifacts/logs/u14-*.log`. No real run,
secret, variable, repository or VM mutation was executed.

Read-only U08 builder observation attempts failed before execution: the recorded
`vince@192.168.2.253` address rejected public-key authentication, and
`vince@linux-infra.dimensionlab.net` had no trusted ED25519 host-key entry here.
Strict host-key checking was retained. Logs `u08-builder-observation.log` and
`u08-builder-hostname-observation.log` contain only those failures. Those were
mistaken SSH-to-self attempts, not a builder access blocker: subsequent local
`hostname`/address inspection confirmed this workspace is already on
`linux-infra.dimensionlab.net` at `192.168.2.253`. Local
`scripts/test-vm.sh status` reported the existing guest running, and the pinned
`scripts/test-vm.sh ssh 'hostname; uname -m'` succeeded, reporting `soda-test`
and `x86_64`. No SSH repair is required. These read-only observations did not
change VM state or establish installed product acceptance.

## U13 source and local verification

Connected My work, native permission-filtered repository/issue/PR search,
notification read/unread/pin updates and native user profiles/activity. Personal
work filters use the acting grant, not a caller-selected user ID. Profile-owned
repository search resolves the user through Forgejo and never widens a missing
owner into an unfiltered search. Notification navigation extracts only a trusted
repository subject number, never provider URL hosts/query credentials. Native
notification permission and 205 acknowledgement semantics were inspected, as were
native profile/activity visibility rules. Added verified `write:notification`
consent; no work/activity/notification inventory is stored in Soda.

Executed `go test ./...`, dashboard type checking, 18 DOM/unit tests in 11 files,
and production build: all passed locally. Logs: `.artifacts/logs/u13-*.log`.
Focused cases cover personal-actor binding, large IDs, notification denial/links,
missing-owner scope, superseded search and independently failing overview panels.
Installed native-user verification, advanced activity/coverage decisions and the
remaining U14–U20 work are still pending. No provider or VM mutation was executed.

## Core/support rebase integration — source only

Resolved the build/context/installation-guide overlap in favor of U02's real React payload and missing-lockfile guard. The full native build skips the dashboard in its generic command loop and calls the core `build-dashboard.sh` with `--payload-only`; support then builds/exports the dashboard image exactly once with native identity metadata. The standalone dashboard entrypoint retains its existing image build and does not depend on support tooling. Container context rules admit the generated dashboard assets while retaining dashboard dependency/dist exclusions and denying other artifact trees by default. Staging and the core Containerfile continue to consume that same payload. This is source integration, not executed build or runtime evidence; builds/tests and dependency resolution remain held.

## Native support source implementation — not executed

Implemented the active outside-support source against the coordinated native plan. The [support guide](native-support.md) documents inputs, effects, recipes, shared-file ownership and retention; [notices](native-support-notices.md) identify selected predecessor reuse. Core U08/U20 still own product journeys/readiness. P07/P08 are not new work queues; P09/P10 media remains unselected and unimplemented.

- **P01/P02:** separate `tools/soda-{artifacts,acceptance}`, restricted/exclusive evidence, bounded streaming redaction, literal pinned-SSH/stdin transport, separate native/evidence outcomes, and one exact-source remote prepare/build/check/bundle phase at a time. No release account, publisher, scenario registry or copied product flow.
- **P03:** fresh matching-native KVM ownership, QMP negotiation/deadlines, validated host trust/hostname, isolated overlay/NVRAM, bounded shutdown/restart and retained private work. The existing `soda-test` helper, VM and backing files are untouched.
- **P04:** explicit native Go/local-engine guards, checkout/output locking, ELF/OCI/blob/source/base checks, resolved service-image IDs, public dependency/package/CLI metadata, allowlisted bundles/notices/checksums and verifier exclusion from installed payloads. Existing build/stage paths and core service references remain authoritative; React packaging is not implemented here.
- **P05:** independently pinned CoreOS architecture inputs, explicit trusted signer/keyring verification, immutable fresh cache output, public/bootstrap separation, restricted strict Butane conversion, verified transfer and first-install preflight/partial-state refusal. No app setup/OAuth/migration implementation is duplicated.
- **P06/P11:** read-only immutable installed-byte/service-image binding and authored host/order/PAM-account/TLS and retained Cockpit/Tailnet/Runners/advertisement observations, plus an opt-in trusted provider-job fixture outside CI discovery. Provider mutations/interactive reviews still require their exact grants and actual observations.
- **P12/P13:** ordinary support handoff reporting with source/platform/file-identity checks, separate missing/failure/cleanup states and core-owned result references. Reporting source exists; **no x86_64 or aarch64 native exit has been completed by this work**.

Authored Go and Python fixture/regression checks, preserving existing core/packaging/Cockpit checks. Per the execution boundary, **no build, test suite, dependency installation, VM/SSH/provider operation, installation, network change, commit/push or publication was run for this implementation**. Work performed is source/research/editing and formatting/static review only. Public CoreOS stream metadata was inspected; no CoreOS image/signature was downloaded or booted. Current tool/installer/firmware/runtime compatibility remains unverified.

Shared changes are limited to P04/P05 format/identity/provisioning concerns in build/check/stage/install and the existing Containerfile base argument, plus the P06 host check. Source review is the next step; executing the authored checks/builds requires a named matching-native builder/revision and a separate grant. Fresh installation/VM/operator work additionally needs its own named targets/actions. Do not attribute the historical successes below to these new bytes or delete private/persistent artifacts to obtain a fresh attempt.

## Current execution — 2026-09-06

**Recorded M15 x86_64 build/source checks passed; the local dashboard is activated and its operator browser journey is verified.** The isolated `soda-test` CoreOS KVM host runs Forgejo, the Soda dashboard, Caddy and native operator services. See [local testing](local-testing.md) for URLs, private credential locations, exact scope and logs. Initial native startup fixes and resolved Go metadata are in `88be176`; navigation changes are in `95a194d`, and dashboard-access/repository-picker changes are in `c96530c`. The combined tree with branding, console and project-CLI follow-ups has not been rebuilt or retested. No artifact publication was performed.

The full M16–M17 developer/project/workload journeys remain pending. AArch64 M18 remains unverified. A first install recovered from the discovered copy/label defects is not a fresh-disk proof of the final installer. Nested Podman and direct client routing remain the highest native risks. Additional installations, network changes and provider/lifecycle operations still require named targets and explicit permission.

### Dashboard activation and browser access

Completed Forgejo's native installer on the isolated VM with a new local `operator` administrator, created a scoped native token, and ran the real `soda-setup` OAuth bootstrap and `soda-activate`. Soda is at `https://localhost:24443`, Forgejo at `https://localhost:24444`, through loopback-only SSH tunnels; Cockpit remains at port 29090. The existing localhost test certificate/CA is used, not public production TLS. Added `scripts/test-vm.sh web-tunnel` and documented the separate laptop forwards and private credentials.

Added and executed `tests/installed/dashboard.mjs`: real browser TLS verification, Forgejo password authentication, consent/S256 OAuth callback into Soda, secure HTTP-only session, authenticated Projects/Profile/operator People navigation, and CSRF-protected sign-out. Chromium trusts the CA in an isolated NSS home; no certificate bypass or normal browser/system trust changes were used. The native dashboard process is UID/GID 2000 with zero effective capabilities, services are active, and no failed systemd units were observed. Evidence is in `dashboard-browser-check.log` and `dashboard-services.log`.

No developer users, repositories, project containers or provider runners were created. This is operator authentication/navigation evidence, not the full product journey. Corrected operator command examples to use `/usr/local/sbin` explicitly because the CoreOS root SSH PATH omits it. Dashboard bootstrap intentionally restarted only the VM's Forgejo service, not the host or VM itself.

### Forgejo repository picker

Replaced manual `owner/repository` entry on Projects with an accessible native select populated from Forgejo's `GET /users/{username}/repos`, verified against the installed 15.0.7 API schema. Requests target the signed-in user, not `/user/repos` for the privileged operator token. Pagination continues until an empty page; the dashboard filters by stable owner ID and excludes every existing project reservation, including incomplete provisioning, then sorts names. Creation still re-fetches the selected repository and enforces ownership server-side.

Added native Forgejo creation/refresh links and distinct empty/provider-error states that retain the existing Projects table. Go tests cover pagination (including server page-size caps), malformed/failed responses, cancellation, ownership/privacy filtering, existing reservations, empty states and forged selection rejection. All source/staging checks pass. Rebuilt and loaded only the dashboard image, updated its staged/native binary, and restarted only `soda-dashboard.service`; application data and the other services were preserved. The old image archive was retained after Podman's refusal to overwrite it directly.

The real browser check now verifies the picker or its empty state, native create/refresh links, and the existing OAuth/navigation/logout journey. The test operator currently owns zero repositories, so the live run verified the empty state; populated/filtering cases are covered by Go tests, not claimed as an installed populated-repository journey. No repository or project fixture was created. Evidence: `repository-picker-tests.log`, `repository-picker-build.log`, `repository-picker-deploy.log`, and `dashboard-browser-check.log`. Changes are committed in `c96530c`.

### Cockpit/Tailnet native correction

The first interactive Tailnet read exposed a missing SELinux PAM session transition: root authenticated successfully but its bridge remained in `cockpit_session_t`, where Tailscale socket access and stock systemd operations were denied. Restored the native Fedora Cockpit PAM stack while retaining the required UID-0 account gate. A new authenticated Cockpit WebSocket session now runs in the native operator context and successfully reads `tailscale status --json` and LocalAPI preferences. SELinux remains enforcing; no socket permission changes, daemon restart or Tailnet enrollment were performed. Native PAM account checks allow root and deny the existing non-operator `core` account.

Added a staging regression for the root-only gate and ordered SELinux session rules. The old staged config fails it; the corrected stage passes all five packaging checks, along with Go, TypeScript, 60 Cockpit tests and installed host checks. Existing Cockpit users must log out and back in to receive the correction. This correction is included in `88be176`.

### Accounts navigation

At the operator's request, hide only Cockpit's stock Accounts menu entry through the native `/etc/cockpit/users.override.json` merge patch. The source config is staged for future installations and applied to `soda-test`; no packages, host accounts or native account tools were removed, and no services restarted. Native `cockpit-bridge --packages` before/after output confirms that `users` loses only its Accounts label and every other menu entry is unchanged (`cockpit-packages-before.log` / `cockpit-packages-after.log`). Added a packaging regression; all six packaging checks, Go tests, TypeScript checks and 60 Cockpit tests pass. Browser sessions may need logout/login to discard cached manifests. This navigation change is committed in `95a194d`.

## Core implementation continued — protected grants and first workflow

Resumed from clean `13bd49a`. U03/U04 now have AES-256-GCM session-bound grants,
actual-scope introspection, bounded code/refresh exchange, serialized local
refresh and logout-safe update-only persistence. Production validates the
restricted external key before migrations; schema v3 preserves existing product
records and refuses wrong existing encryption keys. Setup/activation source
provides the new key only for first installs; existing-state migration and
rollback are documented separately, not executed.

U05–U07 now have individually registered acting-user account/settings/Git-key,
Forgejo People/create-person, repository list/create/detail/content and Soda
environment list/create/detail/join/member/connection APIs, with connected React
routes/forms. New APIs never read the bootstrap token. Forgejo People uses actual
native admin authority, not Soda operator ID. Creation reserves once, does not
join the creator, and reports retained incomplete native results. Join calls the
real fixed account helper before saving membership. The new core-owned helper
`/connection` reads only a fixed public Ed25519 host key and observed address;
stopped containers stay stopped, and routing is explicitly unverified.

Follow-up source adds actual Link/total-header pagination (without following
provider URLs), safe GFM/relative README rendering with raw HTML and automatic
image loading disabled, bounded 8-MiB authenticated attachment downloads and a
frontend render-error boundary. Markdown pins were selected from public npm
metadata; no dependencies were resolved or installed. Pagination, download
headers/size/traversal and Markdown safety tests are authored, not executed.

U01 inspected matching upstream v15.0.7 OAuth, route middleware and repository
search source. Important limits: confidential-client consent can remain at old
scopes; token responses omit scope, so the new flow introspects actual consent.
Native refresh counters are per user/application: another session may invalidate
older refresh material, which requires reauthentication rather than credential
sharing or changing upstream settings. See [API coverage](forgejo-api-coverage.md).

Authored Go encrypted-storage/binding/legacy-session/key-restart tests, acting-user
admin denial/nonoperator-admin tests, refresh/logout race, JSON environment
create/explicit-join/failure tests and fixed-public-key helper tests. Authored DOM
repository-create/inert-content/ref and failed-join tests. Go formatting and diff
checks only have run. No dependency resolution, build, type check, product test,
provider mutation, native service/network operation or deployment has run. The
VM, builder infrastructure and backing image are unchanged.

U09 follow-up source connects commit/file history, pinned commit detail/bounded
unified diff, branch/tag list/create, comparisons, SHA-preconditioned file
create/edit/upload, personal forks and one-time HTTPS Git imports. These operations
use native permissions, do not execute Git in Go or copy/mutate environments, and
keep import credentials transient. Authored stale-file/protected-branch/import
ownership and editor-draft tests. Added a real-handler OAuth callback/grant test
and conservative token expiry measured before the exchange, not after later
identity/introspection calls. All checks remain authored, unexecuted. Blame and
full native diff/import capability coverage remain U09/U17 audit work.

U10 source now adds issue list/filter/create/detail/edit, comments/edit,
assignments/state/labels/milestones, reactions, own subscription and bounded issue
attachment upload, with connected React views and sanitized lossless IDs. The
selected native router requires `issue` scopes separately from `repository`;
default consent now includes `write:issue`. Native permission checks remain on
every operation, no issue data is stored in Soda, and these operations never call
the environment helper. Authored native-denial, actor-binding, multipart-boundary
and create-issue DOM tests; none executed. Structured native issue templates,
comment-attachment/reaction details and full stale/permission/native journey
coverage remain source work. Attachment downloads explicitly use configured
native Forgejo links, not borrowed cookies or a fabricated custom binary proxy.

U11 source adds connected PR list/create/detail, a shared native issue/PR
conversation component, commit/file/diff/status inspection, reviewer requests,
explicit-head reviews/new-side inline comments and protected native merge. Native
review/merge source was inspected at v15.0.7: review comments use file line numbers,
reviews receive explicit `commit_id`, and merges receive `head_commit_id` with
force/auto-merge/branch deletion fixed false. File/diff reads check displayed
head/base/merge-base before and after retrieval; stale snapshots fail closed.
No merge operation calls the project helper. Authored changed-diff, stale-review,
merge-protection/no-force and retained-draft DOM cases; none executed. Existing
inline-thread rendering, old-side positions, team reviewers and full native
permission/conflict evidence remain pending. Native pending comments can remain
after a failed review submission; no automatic retry or invented recovery occurs.

U12 hook source now exposes native metadata/list/create/edit with URLs and
credentials write-only to the browser. Inspected v15.0.7 hook implementation:
responses contain decrypted authorization headers; PATCH resets omitted
headers/events/filter, does not rotate signing secrets, and cannot change existing
package/action event flags. The adapter redacts outputs, preserves omitted values
within the request and rejects unsupported changes rather than claiming success.
Forgejo owns target delivery/host policy; Soda never calls a hook URL. Its pinned
default resolves an empty allowed-host setting to external hosts and verifies TLS;
no appliance webhook override was found by source inspection. Authored redaction,
preservation, unsafe-input and native-denial tests remain unexecuted. U12 now also has connected repository metadata/feature/merge settings, native
collaborator permission inspection/set/remove, Git deploy-key list/add/remove,
and branch/tag protection list/create/edit. Repository settings submit only
changed fields; native rename/transfer/archive/delete are not exposed. Deploy
keys remain separate from Soda development keys, and provider access changes do
not propagate to Linux. Pinned branch/tag PATCH implementations were inspected;
omitted protection values are retained, otherwise-ignored nested push flags are
rejected, and IDs remain lossless. Added settings-only-patch, privilege-denial,
public-key rejection and protection dependency/opaque-ID cases plus a focused DOM
test. Only gofmt/diff inspection ran. Advanced branch-policy form details,
the full native permission matrix and advanced form details remain pending.

Organization/team source now adds native directory/create/profile, members/team
lists, scoped team detail/description, explicit member and repository assignment
changes and bounded policy APIs. Default consent adds verified `write:organization`.
Pinned router, organization/team mutation and unit definitions were inspected:
organization PATCH clears omitted profile strings (retained in-request), team
policy has parent-permission dependencies and administrator units cannot be
pretended to accept limited overrides. Forgejo resolves actual organizations,
teams, repositories and all permissions; no organization/member/role rows are
stored in Soda. Org-owned environments remain unsupported under the existing
human-owner rule. Authored preservation, opaque-ID, direct-denial and ignored-field
cases are unexecuted. Advanced team policy/protection forms and native matrix
acceptance remain source/execution work.

### Current milestone ledger

| Milestones | Source | Built / tested / installed | Remaining |
| --- | --- | --- | --- |
| U01 | Partial first-workflow scope/token/visibility audit | No new execution | Full action inventory, dependency closure/licenses, remaining native capability research |
| U02 | Partial preview/build packaging and new routes | None | Real lockfile/resolution, full error boundary/dev arrangement and frontend verification |
| U03/U04 | Connected encrypted-grant/config/API source and focused tests | None | Executed migration/key/refresh/consent/permission suite, more callback/scope race cases, controlled upgrade rehearsal |
| U05 | Native profile/Git SSH keys/People + Soda preferences/development keys | None | Native onboarding proof, remaining account/key coverage and focused UI cases |
| U06 | Repository list/create/detail, ref-aware text/GFM/README, bounded downloads and native pagination metadata | None | Full visibility/ref/content tests, dependency resolution, browser/native-write proof |
| U07 | Connected persistent create/join/inspect/member/connect paths | None | Additional partial-result/authorization/connection cases and native evidence |
| U08 | Pending | None | Explicit build/deployment/fixture/lifecycle permissions, approved direct client route and two-project/workload/persistence proof |
| U09 | Connected history/ref/compare/file-write/fork/basic-import source | None | Expanded focused tests, blame/native diff/import capability audit and real native Git verification |
| U10 | Connected issue/comment/label/milestone/reaction/subscription/issue-upload source | None | Native templates/comment-asset detail, expanded focused cases and two-user native proof |
| U11 | Connected bounded PR/review/merge source and revision-bound mutation tests authored | None | Inline-thread/old-side/team detail and real reviewer/merger/native conflict journeys |
| U12 | Connected repository settings/access/protection/hooks and organization/team source | None | Advanced team/protection forms, sub-action coverage and full native permission matrix |
| U13 | Connected overview/search/notifications/native profiles/activity | Local Go/UI/type/build checks passed; no installed proof | Native two-user visibility/update journeys and coverage closure |
| U14 | Connected runs/tasks/workflows/dispatch and repository/organization Actions configuration | Local Go/UI/type/build checks passed; no provider mutation | Real approved provider/runner journeys and explicit native run-control gap disposition |
| U15 | Connected releases and bounded assets | Local Go/type/build and existing UI suite passed; no installed proof | Wiki/packages, release DOM cases and native permission/transfer journeys |
| U16–U20 | Pending | None | Admin expansion, coverage decisions, gated cutover, polish and final native architectures/fresh-install/upgrade proof |
| E01–E03 | Unselected | None | Explicit selection required; no dormant controls added |

No U milestone is complete. GFM rendering is not parity with all Forgejo Markdown extensions. Native
account-security links and consent recovery are labeled upstream dependencies;
no runtime fallback or conditional environment extension was implemented.
[Credential migration/rollback](dashboard-credentials.md) remains an authored
procedure, not installation evidence. Execution gates do not block continued
independent source implementation.

## Core implementation started (first batch, historical)

The user explicitly requested implementation of the complete leading core plan. Began from clean `b7241b2`; the P plan remains outside support and E01–E03 remain conditional. This is the first connected source batch, **not completion of U01–U20 or new native evidence**.

- **U01 audit in progress:** added [Forgejo API coverage](forgejo-api-coverage.md) from the saved 15.0.7 schema with its hash, endpoint/option families, authority ownership and explicit unresolved scope/refresh/API-gap work. Inspected public npm metadata for React Router 7.18.3 and React-plugin peers; selected only the router addition. No broad admin-token proxy or invented upstream endpoint was added.
- **U02 source in progress:** added `dashboard/` with client React/PatternFly/Vite+/Zustand, browser routing, canonical assets, session state, actual profile/key forms and explicit legacy Projects/People links. Go mounts compiled output at `/app/`, verifies manifest/entry/assets/licenses before opening the database, keeps legacy routes/default intact and returns 404 for missing asset files. Added a dashboard-only native build entrypoint and integrated its assets into existing full build, image, staging and check entrypoints. Infrastructure/P tools were not ported.
- **U03 source in progress:** added JSON session discovery/logout, Soda preference and development-key APIs; method/content/body/unknown-field validation; mandatory JSON origin/header CSRF checks for every unsafe method; non-redirecting JSON 401/403/404/405/413/415/503 failures and no-store responses. API identities are decimal strings, and public-key parsing is shared with the legacy form. No new API uses an operator provider credential.
- **Schema v2:** append-only transactional migrations recognize existing v1, preserve product/session rows and add the constrained OAuth return path. `/login?return_to=/app/` binds the return destination to single-use state without changing the native callback or weakening PKCE. Unknown/newer/invalid schemas are refused. The old binary's positional OAuth inserts are not v2-compatible; preserve a consistent pre-deployment DB/config/artifact set. No live database was opened or migrated.
- **Tests authored:** Go API/method/CSRF/privacy/key/session tests, legacy migration/preservation/refusal tests, OAuth return binding, frontend filesystem/manifest/routing tests, TypeScript fetch/session-race/form tests and staging checks. No tests or type checks have been executed for this batch.

**Remaining immediate work:** U01 exact selected-version OAuth scope/middleware/refresh audit; U02 dependency resolution/real lockfile, full frontend verification, error boundaries and trusted HTTPS development arrangement; U03/U04 protected session-bound grants/configuration/refresh/logout concurrency; full Forgejo account/People/repository/environment APIs and subsequent collaboration/admin milestones. The current OAuth still requests `read:user` and discards the identity access token; the preview exposes only Soda-owned operations and preserves native/legacy links. Do not mistake it for expanded Forgejo authorization or U05 completion.

**Execution:** source/metadata inspection, source edits, formatting and diff/documentation review only. No dependency installation/resolution, compilation/build, product tests, generated frontend assets, provider mutations, VM/network/service actions or deployment. The new dashboard lockfile is deliberately absent rather than fabricated; frozen build scripts fail with an explicit preflight message until authorized resolution/review. Go formatting completed. Frontend formatter attempts failed at the wrapper/workspace/config boundary because the new dashboard dependencies are not installed; no successful frontend format/type/test result is claimed, and no dependency installation was used to bypass the gate. The running test VM remains unchanged. See [dashboard source instructions](../dashboard/README.md) and [API/migration contracts](dashboard-api.md).

### Provider response boundary follow-up

Rechecked clean HEAD `70fe1da` and read the complete leading plan, handoff,
architecture, inventory, deferred scope and native installation/validation guides.
U03/U04 provider transport now returns sanitized typed HTTP status errors,
preserves request cancellation, and rejects oversized, null, malformed or
trailing JSON responses, including OAuth token responses. It does not retain raw
provider errors or retry with another credential. Existing callers use this
boundary; OAuth scopes and grant retention are unchanged.

Authored focused response-size/format, status/denial/no-retry, cancellation and
OAuth-response tests. Only Go formatting and `git diff --check` ran; no tests,
builds, dependency resolution, provider requests or native changes ran. Requested
specific build/test, backed-up dashboard deployment, fixture and lifecycle
permissions; routing/client authority still needs an explicit selection.

Milestone ledger remains: U01/U02/U03 and Soda-local U05 partial source;
U04 has only this transport preparation, not per-user grant integration;
U06–U20 pending. No new milestone is source-complete, built, source-tested or
installed-verified. No new native fallback was implemented. E01–E03 remain
unselected. Execution permissions do not block independent source work, which
also remains unfinished; this follow-up does not complete the assignment.

## Core/native plan coordination merge

Pulled `origin/main` (`9c8d672`) into local `55ce5cb` with `git pull --no-rebase --no-commit origin main`, preserving both documentation histories. The only textual conflict was the introduction to `docs/implementation-plan.md`; resolved it by retaining the historical M01–M18 context, the leading U plan and subordinate native-support reference. No application-source conflict or application change was involved.

At the user's direction, [dashboard-implementation-plan.md](dashboard-implementation-plan.md) leads the core—including Go/API/auth/data, production `internal/host`/`project-os`, shared build/config contracts and U08/U20 product acceptance. Reworked [native-porting-plan.md](native-porting-plan.md) around outside VM/QMP/SSH/evidence/artifact tools, provisioning transport and retained host-operator integrations. Former P07/P08 redirect to U08/U20; their useful direct-IP, shared-installation, workload and bounded-persistence test details are retained in the core plan. P06 owns host/service observations, P11 outside integrations, and P12/P13 scoped support evidence—not parallel browser/product suites or readiness verdicts. P09/P10 media remains conditional and is not a core gate.

Added explicit shared-file/input/output ownership, evidence reuse and non-circular ordering to both plans; aligned AGENTS, README, architecture, inventory, deferred scope, historical plan, installation, native validation and reuse guidance. Existing authorized tools may support U08 without waiting for the P port. No U/E/P implementation milestone is completed by this coordination, and existing native limitations/evidence are unchanged.

This merge/coordination performs documentation/diff, conflict-marker, local link/anchor, milestone-reference and whitespace review only. No builds, product tests, dependency installation/resolution, artifact generation, VM/service/network/provider operations, push or publication ran. The merged application remains unrebuilt/unretested.

## Unified React dashboard planning

Recorded the user's next frontend direction and a proposed full page/direct-dependency inventory in [dashboard planning](dashboard-plan.md): client-rendered TypeScript/React, PatternFly, Vite+ and Zustand, without SSR, Tailwind or TanStack. Go remains the application API and Forgejo the identity/Git/collaboration authority. The plan separates the first working repository-to-environment flow, later functional coverage and native screens/API gaps; it does not add a host-administration frontend or a second password authority.

Inspected current source/manifests and the previously saved installed Forgejo 15.0.7 API schema. The current OAuth flow requests only `read:user` and does not retain access/refresh tokens for subsequent user-scoped operations; the plan explicitly requires extending that lifecycle rather than using the operator credential as a universal proxy. Updated guidance to distinguish the selected next direction from the still-existing HTMX implementation. Only documentation was changed and whitespace/diff review performed. No dependencies were installed or changed; no builds, product tests, VM operations or live provider requests ran for this planning work. Page scope and new dependency pins remain proposals, not implemented capabilities.

Clarified the upstream boundary at the user's request: Soda extends Forgejo with development environments, project-local access and public-key provisioning; frontend ownership does not transfer Forgejo's backend/data/permission responsibilities. Expanded the plan to explicitly include upstream-authorized Forgejo site-administrator views instead of permanently relegating them to the native UI. Native screens remain fallbacks for unverified interfaces; Cockpit remains host administration. Removed the illustrative backend ratio from the plan. Reviewed the saved `/admin` API schema and updated guidance only; no application, dependency or runtime changes were made.

### Multi-milestone frontend/backend plan

Added [the U01–U20 implementation plan](dashboard-implementation-plan.md) at the user's request. It sequences source-backed capability research, React/static packaging, JSON/schema migration, per-session Forgejo credentials, accounts/admin onboarding, repositories, native environments and early installed proof before broader collaboration/admin coverage, cutover, polish and final verification. It includes source ownership, API/state/security contracts, data/config migration and rollback constraints, page-to-milestone coverage, dependencies, acceptance cases and evidence levels. Forgejo remains upstream; the Go layer does not acquire its business or permission ownership.

Rocky/Fedora profiles, existing-environment lifecycle controls and basic resource caps are conditional E01–E03 tracks, not silently approved scope. Other previously deferred lifecycle/ownership/recovery features remain explicit decisions. Linked the current plan from guidance, architecture, inventory and README and marked the original M01–M18 plan historical. This is documentation only; no implementation milestone, dependency resolution, compilation, product test, provider call, VM action or deployment ran. Documentation paths/anchors, milestone coverage references and whitespace were reviewed; the planning work was committed as `55ce5cb`.

## Source merge — 2026-09-06

Merged the local predecessor follow-ups (`0f25570`, `4e751ac`, `f2523ac`) with remote history through `95a194d`. Resolved conflicts in agent guidance, README navigation/status, operator setup and the project image. The image retains native `curl-minimal` and mise checksum fixes alongside Tea/GitHub CLI inputs; combined staging tests retain both the PAM regression and the branding/console checks. The real Go metadata and native installation fixes are preserved.

The earlier native evidence does **not** validate this merged tree or its additional checks. This merge performs source/diff, conflict-marker, whitespace and local documentation-link review only. No dependency resolution, builds, product tests, VM/service operations or provider actions were run. The current request authorizes Git merge/commit/push, not additional native execution.

### Dashboard follow-up merge

Merged `origin/main` through `f4fe066` into `c96530c`, preserving both histories. Resolved five documentation conflicts by retaining the newer dashboard activation/browser and repository-picker evidence alongside the predecessor follow-ups and their unvalidated status. Reviewed the automatic staging/configuration merge; Accounts navigation, PAM checks and branding/console checks are retained. Updated stale uncommitted-change references.

Only source/diff, conflict-marker and whitespace checks were performed for this merge. No builds, tests, dependency resolution, VM/service operations or provider actions were run; the combined tree remains unvalidated.

## Original native artifact and acceptance proposal (scope superseded above)

Upstream commit `9c8d672` added the original [porting proposal](native-porting-plan.md), based on current source `6f7b51e` and predecessor `bc1d3e0`. It maps selected VM/QMP, process/cleanup, SSH/evidence, artifact-inspection and scenario source/tests to proposed destinations, with P01–P13 dependencies, source/native exits and exact-target execution gates. CoreOS remains the proposed host path; bootc/Anaconda, the old account model, Updates and release publication/qualification machinery remain excluded.

The plan explicitly distinguishes application OCI from a bootable host image, public media from private provisioning, a QCOW2 deployment kit from a preinstalled image, and forwarded access from a real client route. It prioritizes fresh x86_64 installation/developer/persistence evidence and preserves independent aarch64 follow-up. Existing `soda-test` state and its backing disk must remain untouched.

Planning/documentation only. Reviewed source/documentation, diffs, local documentation link targets, plan anchors and whitespace. No port implementation, generated artifacts, builds, product tests, dependency resolution, VM/service/network/provider operations or publication ran. At that proposal's creation P01–P13 were not started. The coordination merge above transfers P07/P08 into core U08/U20 and narrows the remaining P scope; active P milestones were still unimplemented at that coordination point. The subsequent source-only support implementation is recorded above; native exits remain unverified. Prior native results and the merged tree's unvalidated status are unchanged.

## Original source handoff (historical)

The M01–M14 entries below describe the original source-only handoff, when builds, tests, type checks, dependency resolution, installation and publication had not run. Their original “not run” statements are historical; current evidence is recorded above and in [local testing](local-testing.md).

## Baseline

- New backend and privileged integration: Go; no Rust subsystem without a concrete need. Retained Cockpit frontend: TypeScript/React.
- Go 1.26.7 (predecessor source baseline); HTMX 2.0.10 vendored from its npm distribution with license.
- Host candidate: Fedora CoreOS stable 44.20260817.3.2, reported for x86_64/aarch64 by the upstream stable stream metadata. Native compatibility remains unverified.
- Predecessor source: local `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. No changes are made in that repository; Updates remain excluded.

## M01 — source implemented

Go server/configuration, embedded canonical branding, HTMX source, initial templates, narrow process runner reused from predecessor, and authored configuration/server/process tests. Application features follow in subsequent milestones; this foundation is not a completed product.

Build: not run. Validation: not run.

## M02 — source implemented

SQLite via modernc.org/sqlite v1.58.0 (upstream module metadata), schema and concrete store, hashed session tokens and single-use OAuth state storage. Startup opens the persistent database. Tests authored, not run; go.sum/dependency resolution remains a later build input step.

## M03 — source implemented

Forgejo 15.0.7 API client, confidential OAuth setup CLI, persistent Forgejo container definition and operator-native installation instructions. Native API/schema source inspected; no live accounts or OAuth applications created.

## M04 — source implemented

Forgejo OAuth authorization code + S256, session rotation/logout, CSRF/origin checks, profile/public-key forms, operator user creation, and real provider/database wiring. Tests authored; not run. Browser and native OAuth behavior remain unverified.

## M05 — source implemented

Rocky 9.6 + mise project image recipe, systemd/OpenSSH initialization, native project-account setup, root-owned SSH keys and retained writable-rootfs strategy. Native image/account/SSH behavior remains unbuilt and unvalidated.

## M06 — source implemented

Fixed-operation root helper over a root:soda systemd Unix socket; dashboard client wired. Native Podman create-once/start/inspect/account commands, user-namespace mapping, retained project units and routed bridge IP readback. Configure a real route to the chosen project subnet before client SSH; bridge allocation alone is not remote reachability proof. Native helper/network/persistence tests authored, not executed.

## M07 — source implemented

Project discovery/create/detail and explicit join routes now reach the provider, native helper and database. Repository ownership is checked server-side; membership records follow native account success. Real IP inspection and non-ready errors are displayed. Authored a provider/native-double create-flow test, not run.

## M08 — source implemented

Shared mise install/shim path and native global config are wired into login profiles and SSH non-interactive environments. Project-owner installation instructions and authored two-user checks accompany ordinary Git/shared-file guidance. No private resource selection machinery. Build/validation not run.

## M09 — source implemented, high native risk

Nested project-local Podman service, owner-only shared-engine socket, Compose 1.6.0 provider and build/mount/database example. Parent user namespace, capabilities, fuse, SELinux and cgroup combination requires native proof; no privileged host shortcut or dormant fallback added. Build/validation not run.

## M10 — source implemented

Selected native CoreOS package layering for Cockpit, root-only PAM, loopback-first operator access and native /etc/cockpit/branding delivery. Shared frontend dependency closure/build source and licenses ported without old Projects/Updates entrypoints. Feature entrypoints follow in M11/M12. No builds or tests run.

## M11 — source implemented

Tailnet page/native state/store/components and focused tests ported. Native host refresh now updates Forgejo Git SSH advertisement while preserving configured browser/OAuth origins, inspects live environment and restarts the actual container unit when needed. No Tailnet enrollment or native execution performed. M14 additionally verifies the actual Git SSH listener before advertising a Tailnet address, avoiding unreachable clone guidance.

## M12 — source implemented

Runners page, native provider/lifecycle/helper/launcher code, service wiring and focused tests ported. Root-only authorization replaces the incompatible predecessor human-host-admin rule. Configured Forgejo origins now feed native registration and browser links. Provider-client source locks retained; no binary download, registration, build or validation performed.

## M13 — source implemented

Native build/staging, first-install and private Butane provisioning recipes authored; dashboard/proxy Quadlets, TLS activation, service identity/ownership, native extension package requests, provider-client checksum fetch and branding staging included. Routes require explicit deployment configuration; no project DNS/gateway added. Native staging tests authored. No build, package install, provisioning render, provider download, activation or validation executed.

## M14 — source implemented

Integrated authenticated navigation, visible HTMX error responses, pending native-operation states, basic accessibility and direct SSH/SCP/SFTP guidance that is withheld for stopped/unavailable endpoints. Added server-side compatible username validation, HTTPS/loopback configuration guards, provider/native journey and privilege test source, and the missing retained Cockpit process-test support. Removed predecessor-guessed Tailnet service URLs; actual listener inspection now guards Forgejo advertisement. Corrected native Forgejo SVG/favicon destinations and relocated theme imports from inspected upstream template paths without modifying canonical branding assets.

Authored explicit matching-native source/staging check entrypoint and the full later Alice/Bob, persistence, Tailnet and provider-runner journey. Architecture documentation now reflects implemented choices rather than describing them as unresolved. x/sys v0.47.0 source metadata was inspected; dependency resolution was not run.

**Source-complete, unbuilt, unvalidated.** No builds, compilation/type checks, tests, dependency installations, provisioning renders, native activations, enrollments, registrations or CI jobs were executed. M15–M18 remain held.

## Repository guidance update

Customized `AGENTS.md` from the predecessor's engineering guidance: requirement-versus-choice classification, human-maintainable design, coherent refactoring/reuse, source ownership, actual script side effects, scoped commit authorization and separate evidence reporting. Retained SodaOS's execution hold, project-local authority, private networking and persistent-container boundaries; excluded obsolete predecessor test/release commands and UI assumptions.

Documentation-only change. Reviewed the predecessor/current guidance, owning documentation, script source and full diff; local documentation links and whitespace checked. No product behavior changed and no builds, tests, dependency resolution or deployment operations ran.

## Compatible predecessor follow-up ports

Source reference remains `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`; that repository is unchanged.

- **Branding/configuration:** ported Forgejo browser checks, PNG renderer/pixel comparator and focused test source. Added fresh-evidence and explicit native-review boundaries; actual renderer verification is opt-in with the `branding` test tag. Adapted app metadata, native/accessibility theme choices and cache revalidation into a staged Forgejo environment source file, using the selected upstream environment-to-INI encoding. The disposable component sheet is not an appliance payload. No artwork regeneration, browser checks, tests, builds or native configuration changes ran.
- **Console/handbook:** adapted the native operator welcome, main-table and connected-uplink discovery, interactive-only hook and CLI delivery. It reads configured public origins without printing secrets or inventing reachable ports. Added command-double/staging test source. Reworked developer instructions for project-IP SSH, editors, separate Git credentials, real shared mise installations and project-local service ports; adapted the screenshot brief without adding fabricated captures. No native console, tests, account operations or screenshots were executed.
- **Project CLIs:** retained Tea 0.15.1 source lock/license/fetch behavior and adapted its native Makefile build into project-image inputs, with fetch/build-boundary test source. Added the predecessor's GitHub CLI 2.97.0 baseline through GitHub's signed RPM repository inside Rocky rather than copying a Fedora host package. Developer auth remains native and personal. Source/release metadata was inspected; no archives, binaries, RPMs, dependencies, builds or logins were fetched/executed. The reuse inventory and later validation guide separate these ports from the excluded workspace/release/Updates systems. Source review also tightened console origin parsing and removed stale predecessor RPM/page/renderer instructions and a broken link from the Cockpit asset guide. Formatting, diff/whitespace inspection and local documentation path checks are the only checks executed.
