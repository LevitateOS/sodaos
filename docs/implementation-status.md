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

## Active work — Tailnet

**Stages 1–2 are complete. Stage 3 UI source and emitted-component checks are in
place; Stage 4 source implementation is underway. Actual native-page parity,
complete project runtime and installed proof remain pending.
The user declined local fixture repair, deferred configured-device validation to the
intended ARM device and selected further device-independent source work.** The [Tailnet implementation plan](tailnet-integration-plan.md)
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
- Run-scoped native state intended to preserve identity across daemon-only restarts,
  with explicit logout when ending project access. Namespace, DNS and systemd
  behavior still require native proof.

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

**Stage-4 first slice:** bounded official-SDK key issuance, durable no-replay run
admission/attempt observations, and native incarnation/companion-recipe primitives
now have local race-tested coverage. No shipping runtime caller is activated; Create,
supervision, lifecycle/DNS/key-file integration and project UI remain unfinished.
See the [Stage-4 receipt](implementation-history.md#tailnet-stage-4--enrollment-core-and-incarnation-foundations).

### Next step

Continue the approved Stage-4 implementation: wire the trusted native consumer and
supervision outside the project admission gate, resolver/exec/stop checks, then
explicit Create selection and project Network/connection UI. [Stage 3 native-page parity](tailnet-integration-plan.md#stage-3--native-host-tailnet-ui-and-parity)
is still unproven, not replaced by component-shell tests. The user declined repairing
the broken local fixture and identified the ARM device as the intended target for
configured-device validation. Its exact identity, current installed configuration
and approved checks/delivery effects must be established before contact; no ARM
action or deployment was granted. Do not make local fixture repair a prerequisite
for unrelated source work. Cockpit Tailnet stays source/installed fallback.

The earlier authorized local start failed because template and preview-asset mounts
point into deleted `.artifacts/worktrees/combined-candidate-0f1d2b1/`. No subsequent
repair/start was performed; the container and data volume remain retained.

The separate [native proof proposal](tailnet-integration-plan.md#remaining-native-proof-proposal)
requires a specifically authorized isolated fixture and exact inputs/actions. It
covers namespace/TUN/LocalAPI isolation, DNS ownership/recovery, systemd stop/restart
ordering and, in a separately approved phase, real enrollment/connectivity. No
fixture, credential, provider action or maintenance window is currently selected.
Source/UI completion does not authorize that execution, deployment or native
configuration changes.

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
  work, including existing local fixtures within their approved scope.
- **Tailnet:** the user explicitly selected Stage 4 source implementation and local
  tests. That work is in progress; no new source-work handoff approval is required.
  Stage-3 native-page/device acceptance remains deferred. The user
  previously approved starting exact `sodaos-local-forgejo`; that attempt failed.
  The user then declined source-mount repair and chose device-independent work,
  deferring configured-device validation to the intended ARM device. No further local
  repair/start, ARM contact or deployment is authorized. Stage 1/2 grants are complete.
  No enrollment, networking/capability changes,
  other fixture lifecycle, deployment or installed Tailnet removal is authorized.
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

## Installed state

**Native pages/Runners combined-plan step 5 is complete on both retained targets.**
There is no outstanding step-5 deployment or exec-record blocker. No Tailnet change
or installed Cockpit retirement has been delivered.

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

**`dc38af94c0b6a0eba7db5c06a8a71bcb6828c414`** retired only Cockpit's Runners
presentation. Native runner services/CLI/backend and Tailnet's Cockpit payload remain.
Focused checks and a fresh native x86_64 build/check/export/independent verification
passed. Revised installed contention coverage is authored, not executed on an appliance.

Export: `.artifacts/step6-source/export/x86_64/`.
`build-info.json` SHA-256:
`8ca9ef5b170742f1804750f2071d62227982606abaddb111d029597d750e4a40`.
The offline packaging comparison removed 40 old runner entries (38 files), with
Tailnet byte-identical. It is not an actual installed occupant inventory; older
hashed assets remain on targets. See the
[step-6 receipt](implementation-history.md#step-6--source-retirement-parity-review)
for checks, skips and exact scope. No Tailnet native bundle build supersedes it.

## Remaining work

1. **Active — Tailnet:** follow the next source slice and separately gated native
   proof above. The feature is not complete after moving only the host screen;
   project automation and replacement acceptance precede Tailnet retirement.
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

Merged upstream `e7760d4` into the Tailnet work at `13d31aa`, retaining both histories.
Resolved shared navigation/templates/cache epoch and documentation without dropping
the redesign, incoming session checks or Tailnet source work. The [merge receipt](implementation-history.md#tailnet-and-forgejo-source-merge)
records strict TypeScript/Lit, 53 source/browser tests (24 opt-in skips) and race-enabled
Go checks, including nativebuild/template source contracts. Evidence:
`.artifacts/merge-tailnet-forgejo-TSGPpY/`. No fixture repair/start, ARM/device contact,
provider action, deployment, push or cleanup occurred. Stage 4 remains in progress;
this source merge neither activates automation nor supplies native acceptance.
