# Current handoff

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus planned **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | None: React/duplicate forge adapters removed in `752079e`; original Go/HTMX pages/forms/assets and exclusive clients removed in `9f3baa7` |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v3 SQLite/encrypted grants, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Last recorded `8b823db` dashboard/helper/runner companion/default new-project image; stock Forgejo 15.0.7. Historical React `/app/` preview and HTMX defaults remain installed |
| Acceptance | Only historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`). U01 architecture acceptance was withdrawn; no Sodaspaces/final-product/aarch64 acceptance |

Source removal is **not deployment**. No native image/stage build or installed
retest of the removal commits occurred. The current source has no usable Soda
browser controls until Sodaspaces is connected. Root and completed OAuth redirect
only to configured Forgejo, not to caller-selected or historical stored paths.
The old OAuth return-path column/default remains unused without a schema migration.
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
connection remain untested. Current Soda Origin/CSRF checks and two-origin proxy
prevent simply wiring native-page fetches; a same-origin namespace remains a
candidate requiring actual callback/cookie/config/route review, not a selected API.

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
context and stable-ID API changes are **planned only**; current source is unchanged.

Inspected actual Go/config/setup/staging callers and retained Forgejo 15.0.7 routing
and OAuth application handlers. Native callback editing need not rotate the secret;
the upstream API PATCH does. Recorded that distinction and the template allowlist
gap in the integration guide. Source hashes and documentation checks are retained
in `.artifacts/research/sodaspaces-plan-64aad1f/`. Only source inspection and
Markdown/link/whitespace checks ran; no product tests, build, browser, dependency,
provider, native or private-state actions occurred. The terminal stays separate.

## Remaining work and permission boundary

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
