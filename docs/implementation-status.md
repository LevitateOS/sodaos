# Current implementation status

This replace-in-place handoff tracks **Tailnet** and shared retained-target state.
[Forgejo extension implementation](forgejo-extension-status.md) is a separately owned
workstream with its own status and task order; its progress is not maintained here.

Installed state below describes **development fixtures, not customer installations**;
it is last verified state, not a fresh health/liveness observation.
Detailed receipts and completed work belong in [implementation history](implementation-history.md);
historical approvals are not renewed execution permission.

## Isolated Forgejo visual redesign

The canonical `~/Projects/sodaos` checkout on `main` contains the new visual system across the Forgejo override set. The local presentation mounts also use that canonical directory; the deleted redesign worktree is no longer used. [Its independent status](forgejo-redesign-status.md) owns scope, source checks and browser review evidence. The owner-selected Mac localhost:3300 Forgejo presentation is updated on its existing sodaos-local-forgejo_data volume, with a consistent private backup; the separate builder/VM targets are unchanged. The independent status records parity and delivery evidence. The subsequent fastfetch/ASCII identity update and staging checks are recorded in the independent redesign status.

## Spaces first-use journey

The user selected the focused welcome-to-terminal UX and requested a written plan.
The [Sodaspaces journey plan](sodaspaces-plan.md#first-use-journey-implementation-plan)
owns its independent implementation stages and status; the
[Spaces design](spaces-design.md#first-use-journey--selected-13-september-2026) owns
the agreed states and local mockup references. Planning is recorded; implementation
and journey acceptance remain pending. This documentation work changes no installed
state, current execution grant or other workstream's completion status.

## Active work — Tailnet

The owner has selected the [mechanism removals](refactoring-plan.md#selected-mechanism-removals)
for source/local implementation, including the documented policy tradeoffs. That
workstream owns D1–D13 progress; retained targets and native permissions below are
unchanged. The first source slice removes enrollment attempt journals, uses
activation-owned memory node identity, stores one active credential/policy, removes
release-number vetoes, and applies networking after successful project provisioning.
Old Tailnet configuration conversion remains separately authorized maintenance.
The [D1–D13 workstream](refactoring-plan.md#selected-mechanism-removals) records landed
core source changes and local checks, **not exhaustive removal completion**. The
[59e4da3 reconciliation](refactoring-plan.md#removal-reconciliation-follow-up) found
missed project-network guidance, absent completed-companion resource retirement and
remaining ancillary removals/candidates. Native/installed acceptance is not claimed.
Neither retained old terminal runtimes nor Tailnet state were converted or cleaned.

**Stages 1–2 are complete. Stage 3 UI source and emitted-component checks are in
place; the Stage-4 runtime/UI and stock-only Cockpit source candidate is implemented.
A separately approved fresh x86_64 VM now passes native dashboard/OAuth/Tailnet
read and stock Cockpit access smoke. Full native-page parity, project lifecycle/
security/connectivity acceptance and retained-target delivery/removal remain pending.
The user declined local fixture repair and selected further device-independent
source work. The owner clarified that Soda is not currently compatible with the ARM
machine: it is only an ephemeral Forgejo frontend test bed, not a Soda appliance.** The [Tailnet implementation plan](tailnet-integration-plan.md)
owns the feature: native dashboard host controls, automatic ephemeral project
connections and eventual retirement of Soda's remaining Cockpit presentation.

The [stage-1 design](tailnet-integration-plan.md#stage-1-design-decisions) selects:

- Persistent appliance enrollment, separate from independently identified projects.
- An appliance-owned Tailscale companion sharing the exact project's user/network
  namespaces, not its filesystem or PID namespace.
- Upstream Go OAuth/key creation in the existing host helper, addressing the explicit
  managed Tailnet. Only single-use ephemeral keys reach the companion; reusable
  credentials stay on the host. The original host-CLI/socket shortcut cannot assume
  write authority across the shifted user namespace.
- Activation-owned in-memory ephemeral node state, with bounded logout at every
  companion stop. Restart may change addresses/approval and interrupt connections.
  Namespace, DNS and systemd behavior still require native proof.

**Local evidence:** `eed1c2c` records the design and investigation. Upstream OAuth
and synthetic capability tests, four v2 SDK cases, three temporary-filesystem DNS
tests and a synthetic Unix LocalAPI CLI probe passed. These are not native companion
or real-provider proof. Research, including original/corrected probe attempts, is
retained at `.artifacts/tailnet-stage1-G8Rnza/`; see the
[stage-1 receipt](implementation-history.md#tailnet-stage-1--local-runtime-and-enrollment-investigation).

**Stage-2 source:** fixed protected Go APIs/helper methods, root-only credential and
policy storage, host operations and schema-v10 Tailnet OAuth returns now have local
race-tested coverage. Native management defaults off; managed defaults and project
enable/retry refuse, while Off saves intent without claiming disconnection. That slice
added no UI, project image/unit/lifecycle change or real enrollment. Checks and remaining
limits are in the [stage-2 receipt](implementation-history.md#tailnet-stage-2--backend-state-and-authorization).

**Stage-3 source:** native Tailnet selector/navigation/entry, canonical payload and
cache epoch, Lit appliance/enrollment controls, transient credentials/auth links and
scoped confirmations are implemented. Component, source and retained Cockpit checks
passed; the [receipt](implementation-history.md#tailnet-stage-3--native-ui-source-and-bounded-parity)
separates those results from the failed native-page attempt. A later source-only
follow-up fixed lost enrollment drafts on admission/default writes and added passing
CSRF/origin and failed-readback checks, without contacting either device.

**Stage-4 source candidate:** the earlier
[enrollment/incarnation foundation](implementation-history.md#tailnet-stage-4--enrollment-core-and-incarnation-foundations)
now has native run/stop supervision, restricted run/key state, companion occupancy
and incarnation checks, resolver binding/recovery checks, explicit managed Create,
Network controls, own-account SSH projection and compact Spaces summaries wired.
Fresh-install source selects the companion image and opts into management; old
configuration remains opt-out. The x86_64 candidate is built/exported and installed
on the fresh access fixture below, with the first-boot helper correction `3cb7408`.
No enrollment or provider job was run; older retained targets remain unchanged.

**Stock-only Cockpit candidate:** custom Tailnet presentation and its workspace,
build and unused dependencies are removed. Canonical current-design branding is
staged independently; the operator journey uses stock Overview and read-only native
checks, without advertisement refresh. Retained vendor attribution is preserved.
Both installed fallbacks remain untouched. The
[integration receipt](implementation-history.md#tailnet-stage-4--runtime-ui-and-stock-cockpit-source-candidate)
records source/browser/packaging-fixture checks and limits.

After the user rebased these commits onto newer upstream source, a
[consistency repair](implementation-history.md#tailnet-rebase-consistency-repair)
corrected stale presentation hashes and restored dropped test fixes. Tailnet/runtime
and stock-only Cockpit source remain intact; installed state and permissions are
unchanged. These checks are not native acceptance.

### Next step

Keep Forgejo frontend validation separate from native Soda acceptance. The ARM
machine is an ephemeral Forgejo frontend test bed only; do not request a Soda
installation inventory, deploy Soda, or expect project/Tailnet/Cockpit runtime there.
Any selected frontend checks establish only their actual presentation scope, not
backend authorization, enrollment or installed appliance acceptance.

The [Stage-5 native proof](tailnet-integration-plan.md#stage-5--native-candidate-and-isolated-proof)
now has `soda-native-tailnet-bb3a13c` for the approved fresh installation and access
smoke, not blanket project/provider lifecycle permission. Native runtime proof is
not a prerequisite for independent frontend/source work. [Stage-3 native-page parity](tailnet-integration-plan.md#stage-3--native-host-tailnet-ui-and-parity)
remains pending beyond the native login/OAuth/read checks actually run. The declined
local fixture repair remains declined. Cockpit Tailnet stays installed fallback on older retained targets;
source retirement does not authorize or imply its removal there.

The earlier authorized local start failed because template and preview-asset mounts
point into deleted `.artifacts/worktrees/combined-candidate-0f1d2b1/`. No subsequent
repair/start was performed; the container and data volume remain retained.

The separate [native proof proposal](tailnet-integration-plan.md#remaining-native-proof-proposal)
requires a specifically authorized isolated fixture and exact inputs/actions. It
covers namespace/TUN/LocalAPI isolation, DNS ownership/recovery, systemd stop/restart
ordering and, in a separately approved phase, real enrollment/connectivity. No
credential, provider action or project lifecycle scenario is currently selected.
The fresh-VM access grant below does not authorize the remaining enrollment,
namespace/DNS/lifecycle matrix or changes to older retained targets.

## Forge decision — retain Forgejo

The user [chose to retain Forgejo for now](architecture.md#forge-selection), avoiding
OneDev's Java/JVM server environment. The [OneDev evaluation](onedev-replacement-research.md)
is reference research only; its plugin proof and replacement outline are not active
work. The Go-alternative survey did not establish a full forge with the required
native Go application-plugin mechanism. No fork or separate frontend is selected.

Public evidence remains at `.artifacts/onedev-research-34y1pE/` and
`.artifacts/go-forge-alternatives-ANIOFa/` (including `review-summary.json`). No server
or plugin was built or run during these reviews. The decision changes documentation,
not runtime state, fixture custody, existing execution permissions or the separate
Tailnet work.

## Current permissions

[AGENTS.md](../AGENTS.md#permissions-and-preservation) owns execution policy.

- **Authorized:** routine local source implementation, builds and tests for selected
  work, including existing local fixtures within their approved scope. The owner
  explicitly approved D1–D13 mechanism removals and their documented behavior changes;
  [the refactoring plan](refactoring-plan.md#selected-mechanism-removals)
  owns that source work. This does not grant retained-state conversion/cleanup,
  provider requests, fixture/service lifecycle, native delivery or publishing.
- **Tailnet:** the user explicitly selected Stage 4 source implementation and local
  tests, and then explicitly requested source completion rather than another blocker
  handoff. The source candidate is prepared; further source fixes/tests need no new
  source-work grant.
  Stage-3 native-page/device acceptance remains deferred. The user
  previously approved starting exact `sodaos-local-forgejo`; that attempt failed.
  The user then declined source-mount repair and chose device-independent work.
  The owner clarified that ARM is only an ephemeral Forgejo frontend test bed, not
  a currently compatible Soda target. The earlier request for a Soda inventory on
  ARM was based on a mistaken assumption, not an outstanding prerequisite or grant.
  No further local fixture repair/start, ARM contact or retained-target deployment is authorized.
  Stage 1/2 grants are complete.
  No enrollment, project networking/capability changes, other fixture lifecycle,
  retained-target deployment or installed Tailnet removal is authorized.
- **Fresh Tailnet access VM:** the user approved a separate x86_64 VM and installation
  for dashboard/Cockpit access. `soda-native-tailnet-bb3a13c` was provisioned with its
  own disk/NVRAM/host and operator keys/passwords, extension-activation reboot,
  native Forgejo/Soda bootstrap and loopback-only browser tunnels. The source-fixed
  helper and required dashboard service restoration were applied only there, with
  the old helper retained. The owned 24-hour hold ends approximately
  **2026-09-13 22:37 UTC**; disks/inputs remain afterward. This does not authorize
  real Tailscale enrollment, provider jobs, global client trust changes, cleanup,
  another start after the hold, or maintenance of older retained targets.
- **Metadata repair / source workspace retirement:** the user requested the Count Me
  fix on the fresh Tailnet VM and removal of the obsolete project `cockpit/`
  directory. The exact OS override was backed up, the native link restored and
  Count Me retried once successfully. The source directory was retired by relocating
  provenance and archiving ignored outputs, not deleting their contents. This grant
  is complete; no other target repair, package removal or cleanup is implied.
- **Native Cockpit additions:** the user approved the revised baseline on the fresh
  Tailnet VM and requested continuation. Four native addons and their additive
  dependencies were installed/live-activated, the exact Accounts hiding override
  was preserved outside the active configuration, and native access/PAM checks
  completed. No reboot was needed. This does not select the conditional VM,
  kernel-dump or recording roles, authorize report collection/upload or changes to
  projects/policy/accounts, extend the VM hold, or affect older retained targets.
- **Forgejo administrator navigation:** the user explicitly approved delivery of
  `6c76af9` to the fresh VM and its brief Forgejo restart. The 24-file public delta,
  restricted backups, single Forgejo service restart and native entry/read/logout
  checks are complete. No other service/VM lifecycle, provider operation, cleanup
  or repeat restart is granted by that completed action. The user subsequently
  approved the sidebar correction: the two-template `574b917` delta, fresh backups,
  one further Forgejo stop/start and native sidebar/read/logout checks are complete.
  That follow-up does not grant another restart or extend the existing VM hold.
- **Paired fresh-VM runtime upgrade:** the user explicitly approved applying the
  merged frontend/backend together after the compatibility warning. The `c13781f`
  native build, two minimal image layers, six binaries, matching public graph,
  Tailnet unit/default-image configuration, fresh consistent backups and coordinated
  service window are complete. No projects, runners or saved Tailnet state existed;
  no old-state conversion or project-root change was needed. Native access/contract
  and preservation checks passed. This does not select real project/provider jobs,
  enrollment, another service/VM restart, cleanup or an extended VM hold.
- **Runners:** approved step-6 source retirement/build/check is complete. No installed
  removal or further retained-target cutover/lifecycle grant remains from that work
  or the completed step-5 delivery.
- No outstanding grant for provider jobs/registration, network/trust changes,
  publishing/automatic CI, cleanup or restoration. The isolated runner fixture's
  one reboot/exact `probe-one` removal and the two-GET diagnostic grants were used;
  its old time-bounded VM hold is not a new lifecycle grant.
- Preserve retained roots, v9 data, credentials, fixtures, archives, stopped records,
  failed attempts and later writes. Commands, input files and old receipts are not
  new permission.

## Fresh Tailnet access fixture

**`soda-native-tailnet-bb3a13c`**, custody `.artifacts/tailnet-vm-bb3a13c/`, is the
new x86_64 access fixture. Base installation is the sealed `bb3a13c` export;
`soda-host` was corrected from clean source `3cb7408`. The original bundle
and install marker are unchanged: this is an explicitly recorded helper correction,
not a newly sealed full bundle. Schema v10, zero projects, original base service
image IDs, six active service/socket units and enforcing SELinux were observed.

Native Forgejo operator login, actual OAuth consent/return, Tailnet `NeedsLogin`
with unconfigured enrollment, Spaces mount, root Cockpit/stock Overview, native
socket/CLI/Services/Logs access and stock menu logout passed. No custom Cockpit
package is installed. Full visual/keyboard/theme and project/provider proof remain
pending. Access/password-file/public-CA paths belong in
[local testing](local-testing.md#fresh-tailnet-vm-access); failures, hashes and the
precise limited verification scope are in the
[fresh VM receipt](implementation-history.md#fresh-tailnet-vm-installation-and-access-smoke).

The subsequent [OS metadata repair](implementation-history.md#native-os-metadata-repair-and-cockpit-workspace-retirement)
restored `/etc/os-release` to the vendor link after preserving the exact bad
Soda override. Count Me now reads Fedora 44 metadata and completed both native
requests successfully; its timer and historical logs remain. Provisioning source
no longer overrides OS metadata. The top-level `cockpit/` directory is gone;
tracked provenance is retained under `assets/branding/cockpit/provenance/`, and
ignored outputs/dependency links are archived under
`.artifacts/cockpit-metadata-fix-Ace3yt/retired-workspace/`.

The [native administration additions](implementation-history.md#native-cockpit-administration-additions)
are now installed on this fixture: Podman 130, Files 43, SELinux and Diagnostic
Reports 367; Accounts is visible. rpm-ostree added 42 packages with no RPM
replacements/removals and applied them live. Boot ID, all three appliance container
IDs/images/running states, zero project records and the corrected helper hash were
preserved. Root login, all addon pages and logout passed; native PAM admits root
and denies existing non-root `nobody`, with SELinux still enforcing. Config/PAM/TLS
backup and the old override remain under guest
`/var/lib/soda-candidate-bb3a13c/cockpit-admin-additions/`.
Evidence: `.artifacts/cockpit-admin-additions-lwhgJX/`. Existing browser sessions
need logout/login to refresh cached navigation; no global Cockpit restart was used.
The pending native deployment also carries the additions for next boot. Source
provisioning/preflight now requires the addons and no longer stages Accounts
hiding; the original sealed bundle/old private Ignition remain unchanged.

The [administrator navigation delivery](implementation-history.md#administrator-navigation-delivery-to-fresh-vm)
subsequently applied the exact `6c76af9` public UI delta: five templates and nineteen
changed emitted modules, with epoch `2026-09-13.admin-settings-1`. Only Forgejo was
restarted, retaining its image; dashboard/proxy runtime identities and configuration,
helper, schema, user/repository identities and project records were unchanged.
A fresh native operator login found **Site administration → Soda → Runners/Tailnet**
while Soda still returned 401; both explicit entries then passed authorized reads
and coordinated logout. Neither link appears in global navigation. No new native
bundle/image is implied. Guest backups, including the consistent stopped-writer
Forgejo database snapshot, remain under
`/var/lib/soda-candidate-bb3a13c/forgejo-nav-6c76af9/backup/`.

The [sidebar correction delivery](implementation-history.md#administrator-sidebar-correction-delivery)
then applied `574b917`: one changed layout and one new native navbar override.
Runners/Tailnet are now ordinary entries in the **left administration menu**, and
`#soda-admin-settings` is absent. Native desktop placement, keyboard order,
no-Soda-session visibility, protected entry/reads and logout passed. Only Forgejo
was restarted; its image, other core runtimes, configuration and data identities
were preserved. Fresh restricted backups remain under
`/var/lib/soda-candidate-bb3a13c/forgejo-sidebar-574b917/backup/`.
No browser modules, package versions or module epoch changed in this delivery.

The subsequent [paired runtime upgrade](implementation-history.md#fresh-vm-paired-runtime-upgrade)
installed the merged source built at `c13781f`: 21 browser modules/three templates,
six native binaries and the Tailnet unit, with epoch
`2026-09-13.admin-settings-mechanism-1`. The dashboard runs immutable image
`sha256:1e07589f84a765d05ba3cc1a3aa3448472f7846459553ef319457895af30153c`;
future project creation uses
`sha256:f2c35d274660b5f74fab4b1b5871c2b65c260520569410590261f43b6b53ad55`.
Both reuse the exact prior image layers plus one program file; OS packages, original
images, Forgejo/proxy/Tailscale image selections and the original install marker are
preserved. Current helper SHA-256 is
`52a944ffad682892e5397144a55e6043606c932c596f69d1eebb102ec74197bc`.

No projects, runner records, companions or saved Tailnet policy existed before the
window, and none were created by validation. The proxy closed browser admission;
Forgejo/dashboard/helper were stopped for consistent database backups and paired
publication, then restarted and verified before reopening. All original Soda table
rows matched the fresh backup before browser login. Native sidebar, new runner
inventory/API/CLI, Tailnet NeedsLogin/unconfigured state, Spaces and logout passed.
Schema v10, user/repository/project identities, credentials, private configuration
apart from the selected future image, and enforcing SELinux were preserved.
Backup custody: `/var/lib/soda-candidate-bb3a13c/paired-c13781f/backup/`;
local evidence: `.artifacts/paired-runtime-upgrade-tFj0Do/`. This is a recorded paired
delta, not a new sealed full bundle. Native project-terminal/enrollment/lifecycle
acceptance remains pending.

## Installed state

**Native pages/Runners combined-plan step 5 is complete on both retained targets.**
There is no outstanding step-5 deployment or exec-record blocker. No Tailnet change
or installed Cockpit retirement has been delivered to these older retained targets.

| Target | Installed affected components | Schema | Preserved projects / membership checks |
| --- | --- | --- | --- |
| `soda-test` | Paired `19824ec` | v9 | 4 original roots; 7 memberships' SSH/PTY checks passed |
| `soda-native-spaces-658f2af` | Paired `19824ec` | v9 | 2 original roots; 4 memberships' SSH/PTY checks passed |

- Exact installed candidate: `19824ec55245baaf5ff5a6ad9557a53aa7f3b0f4`.
  Export: `.artifacts/runner-projection-19824ec/export/x86_64/`.
- All six original Rocky 9 roots, accounts, keys, memberships, credentials and later
  writes were preserved. Only future creation uses the retained Rocky 10.2 image.
  Neither VM nor project was restarted/recreated during the cutover; Forgejo/Caddy
  image identities were unchanged.
- Both targets retain **Cockpit Runners and Tailnet**, plus stock administration.
  Both runner inventories were empty at validation; this is not runner-migration proof.
- SSH/PTY checks used pinned management forwarding, not demonstrated direct/laptop
  routing. Managed terminals were quiescent, not active-session continuity proof.
- Two validation Podman exec records were confirmed stopped and preserved. The
  temporary diagnostic listener was stopped; neither is a cleanup/reopen task.

Component delivery, authorization/browser/native checks and preservation details:
[completed step-5 receipt](implementation-history.md#step-5--bounded-retained-delivery-completed-on-both-targets).
Later source/documentation commits are not installed builds.

## Latest built source candidate

**`bb3a13c6a421a7dbaa3ebc39c1ffd6cf168c734b`**, native x86_64, now includes the
Tailnet runtime/UI and stock-only Cockpit candidate. Build/seal, stage verification,
11 actual-staging tests, export and verification with the exported verifier passed.
Five OCI archives and 571 inventory entries are sealed; no custom Cockpit package
is included. Both companion binaries reported the selected Tailscale `1.102.4` in
networkless, read-only version probes—not daemon/enrollment tests.

Export: `.artifacts/tailnet-native-bb3a13c-Y7Qdcn/export/x86_64/`.
`build-info.json` SHA-256:
`70f426557fe142c400d1571ecd00305e2f81281264e659b5c9cf3a6720263492`.
The [native build receipt](implementation-history.md#tailnet-x86_64-native-build-and-export)
records two preserved failed attempts and the packaging fixes. The aggregate
`check-native.sh` was not run: its mandatory native-page fixture remains unavailable
and repair declined. Source checks, artifact verification and staging tests do not
replace that gate or installed runtime/security/connectivity/visual acceptance.

The older Runners-only `dc38af94c0b6a0eba7db5c06a8a71bcb6828c414` export remains at
`.artifacts/step6-source/export/x86_64/`, with recorded `build-info.json` SHA-256
`8ca9ef5b170742f1804750f2071d62227982606abaddb111d029597d750e4a40`.
Its [step-6 receipt](implementation-history.md#step-6--source-retirement-parity-review)
and all older retained targets/fallbacks remain unchanged. The fresh access VM uses
the latest bundle plus the explicitly recorded `3cb7408` helper correction above;
that correction is not included in this unchanged sealed export.

## Remaining work

1. **Active — Tailnet:** validate the integrated source candidate with the separately
   authorized native proof above, then minimal paired delivery and actual-occupant
   removal rehearsal/approval. Namespace/DNS/stop/restart/enrollment/client outcomes
   remain unaccepted. Stock Cockpit native access/logout passed on the fresh VM;
   full visual/theme/keyboard acceptance remains pending. Source retirement is
   complete; older retained-target installed retirement is not done.
2. **Separate — installed Runners retirement:** source retirement is complete, but
   actual-occupant inventory, target-specific removal rehearsal and explicit
   per-target delivery approval remain. The [combined plan](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation)
   owns this work. Keep both installed fallbacks; do not reopen step 5 or interpret
   Tailnet work as removal permission.
3. **Broader acceptance:** physical-keyboard/real editor use, wider Access/error/
   profile-menu and CLI/provider combinations, intended-client routing, native
   aarch64 and whole-product/release acceptance remain outside completed delivery
   claims. These are not new gates for that completed work; consult the
   [terminal](terminal-integration.md), [CLI](project-clis.md) and
   [native validation](native-validation.md) owners when selected.
4. **Other roadmaps:** [Sodaspaces](sodaspaces-plan.md), [Project OS](project-os.md),
   [Services/AI](services-and-ai-plan.md) and the [installer](coreos-installer.md)
   retain their own remaining scope. Moving this handoff's focus does not complete
   or authorize those roadmaps.

## Retained fixtures and evidence

- **Isolated runner VM:** `soda-native-runners-2cdc238`, custody under
  `.artifacts/runners-vm-2cdc238/`. Last installed candidate `19824ec`; `baseline`
  remained running/enabled. Only `probe-one` local account/state was removed after
  a verified private archive. Both provider records/history and credential inputs
  remain. Check the recorded owner/deadline before separately authorized reuse;
  no current liveness is asserted. See the [isolated proof receipt](implementation-history.md#step-4-bounded-x86_64-completion--19824ec-installed-overlap-departure-and-exact-remove-passed).
- **Local browser fixture:** `sodaos-local-forgejo`, `localhost:3300`, volume
  `soda-pages-0f1d2b1-data`. The authorized Stage-3 start attempt failed on missing
  template/preview source bind mounts in the deleted combined-candidate worktree.
  Exact container `0634b216624c` remains exited; no replacement was performed. Preserve accounts/repositories, OAuth state and private
  inputs. `.local/screenshot-fixture/create-output.txt` is the retained regular
  credential file, not a symlink. Follow the [screenshot guide](screenshot-capture.md)
  for login; do not recreate the fixture or expose credentials to repair access.
- **Build worktree:** `.artifacts/step6-native-dc38af9/`, with native outputs and its
  restricted fixture-input copy, remains retained. The earlier 17 linked worktrees
  were removed at the user's request; their deleted outputs are not retained proof.

| Evidence | Location |
| --- | --- |
| Tailnet stage-4 enrollment core and native incarnation/recipe source checks | `.artifacts/tailnet-stage4-source/` |
| Tailnet device-independent draft fix and failure-path checks | `.artifacts/tailnet-source-followup/` |
| Tailnet stage-3 authorized fixture-start failure and narrow mount inventory | `.artifacts/tailnet-stage3-acceptance/` |
| Tailnet stage-3 source/component checks and failed native-page attempt | `.artifacts/tailnet-stage3-XEzD7o/`; `.artifacts/pages-FcP3af/` |
| Tailnet stage-2 local source checks and failed/corrected attempts | `.artifacts/tailnet-stage2-O5WkS5/` |
| Tailnet stage-1 research, synthetic probes and receipt | `.artifacts/tailnet-stage1-G8Rnza/`; preceding research `.artifacts/tailnet-enrollment-TDPVW1/` |
| Step-6 source/native checks, export and offline packaging delta | `.artifacts/step6-source/` |
| `soda-test` delivery, backups and failed/corrected observers | `.artifacts/step5-cutover-19824ec/` |
| Validation delivery and completion receipt | `.artifacts/step5-validation-b5af330/`, including `completion.json` |
| Guest-side paired backups/rehearsals on both retained targets | `/var/lib/soda-native-pages-19824ec-cutover/` |
| Stopped exec-record inspection | `.artifacts/step5-exec-api-7e39229/` |
| Final isolated runner caller/removal/page evidence | `.artifacts/r4-overlap-06/`, `.artifacts/r4-departure-03/`, `.artifacts/r4-remove-01/`, `.artifacts/r4-final-pages/` |

Other retained fixtures/private evidence keep their existing custody. Documentation
reorganization does not authorize cleanup or re-execution.

## Latest local change

Runners/Tailnet now render inside the existing `/admin` shell in source, with
fixed bookmark/OAuth returns and bounded selectors. The reviewed candidate keeps
one content owner, preserves the ordinary drawer on rejected selectors and marks
only the matching Soda sidebar entry instead of also highlighting Dashboard.
The paired presentation epoch is `2026-09-13.admin-host-1`. The
[native page contract](forgejo-soda-pages-plan.md) owns the resulting admin rendering
gate; protected Soda operator authority remains independent.

Affected Go packages, strict TypeScript/Lit, emitted assets, the frontend suite
(252 passed, 7 explicit skips), and focused navigation/connection/cache browser
checks (11 passed) passed locally. The [source review receipt](implementation-history.md#runnerstailnet-admin-shell-host-move-source-only)
records fixes and exact commands. Native admin-page acceptance remains pending:
the former nonadmin fixture is ineligible, and no retained account, service or
fixture was changed to run it. Installed callers and current fixture prerequisites
now reflect that boundary, without granting roles automatically.

This admin-host source candidate is not delivered. The historical
[merge/reapplication check](implementation-history.md#sidebar-upstream-merge-and-reapplication-check)
verified the earlier sidebar bytes against their own source; it does not establish
that the new `/admin` host or its module graph is installed. Older targets,
providers, projects and all existing backups/evidence are unchanged.

The latest recorded fresh-VM delivery is the approved
[paired frontend/backend upgrade](implementation-history.md#fresh-vm-paired-runtime-upgrade)
on `soda-native-tailnet-bb3a13c`, built from the original, pre-rebase `c13781f`
source—not the newer admin-host candidate above. Matching native binaries/images/
public assets were installed with epoch `2026-09-13.admin-settings-mechanism-1`;
the sidebar correction remains intact. Affected Go race/command, Python and
frontend checks passed alongside reused merge TypeScript/Lit/Forgejo evidence.
Native checks passed for the new empty runner inventory, Tailnet/Spaces bootstrap,
operator access/logout, installed identities and preservation. Existing OS packages,
roots, credentials and previous artifacts were not replaced or cleaned up. No
old-state conversion was necessary on this empty fixture. No real project/provider
operation or full new-runtime acceptance matrix is claimed. The documentation
rebase does not deploy the admin-host candidate or grant further target actions.
