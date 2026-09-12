# Current implementation status

This is a replace-in-place snapshot, not an execution log. Installed state below is
**last verified state**, not a fresh health/liveness observation. Detailed receipts,
failures and past approvals are in [implementation history](implementation-history.md);
they are evidence, not current instructions or renewed execution permission.

## Installed state

**Combined-plan step 5 is complete on both retained targets. No further step-5
deployment or unresolved exec-record blocker remains.**

| Target | Installed affected components | Soda schema | Preserved projects / membership access checks |
| --- | --- | --- | --- |
| `soda-test` | Paired `19824ec` | v9 | 4 original roots; 7 memberships' SSH/PTY checks passed |
| `soda-native-spaces-658f2af` | Paired `19824ec` | v9 | 2 original roots; 4 memberships' SSH/PTY checks passed |

- Exact candidate: `19824ec55245baaf5ff5a6ad9557a53aa7f3b0f4`.
  Verified export: `.artifacts/runner-projection-19824ec/export/x86_64/`.
  Later source/test/driver/documentation commits are not a new installed build.
- Delivery paired the dashboard, native helper/runner commands, Forgejo custom files,
  Cockpit Runners payload and one `project-account` program in each original root.
  Accounts, keys, memberships, credentials and later writes were preserved.
- All six original Rocky 9 roots remain; project images/defaults were not changed by
  this cutover. Only future creation uses the retained Rocky 10.2 image. Neither VM
  nor project was restarted/recreated during this cutover.
- Forgejo/Caddy image identities were unchanged. Cockpit Runners remains installed;
  stock Cockpit and Tailnet remain. Both retained runner inventories were empty.
- Validation's two stored Podman exec records were confirmed stopped and preserved.
  The temporary diagnostic listener was stopped; the records are not a prune task.

Native page/authentication/history, Runners authorization, Cockpit read-path and
preservation checks passed at their recorded scope. SSH/PTY used **pinned management
forwarding**, not demonstrated direct/laptop routing. Managed terminals were quiescent;
this was not active-session continuity proof. Empty runner inventories prove read
paths, not retained-runner migration. See the [delivery record](implementation-history.md#step-5--bounded-retained-delivery-completed-on-both-targets).

## Current permissions

Execution policy is owned by [AGENTS.md](../AGENTS.md#permissions-and-preservation).
The active grant record is:

- **Authorized:** routine local source implementation, builds and tests for selected
  work, including existing local fixtures within their approved scope.
- **No outstanding grant from the completed delivery** for another retained-target
  cutover/lifecycle, new native fixture, provider job/registration, network/trust/
  capability change, publishing/automatic CI or cleanup.
- **Step-6 source retirement/build/check is approved** by the user's request to do
  step 6. The [parity review](implementation-history.md#step-6--source-retirement-parity-review)
  covers the delivered baseline. Installed removal still requires explicit per-target
  scope/delivery approval; both retained targets keep their fallback until then.
- **Tailnet:** the user requested the [full implementation plan](tailnet-integration-plan.md)
  for native host controls, automatic ephemeral project enrollment and eventual
  stock Cockpit administration. This change is planning only; no real enrollment,
  networking/capability change, fixture lifecycle or installed Tailnet removal is granted.
- The isolated runner fixture's one reboot and exact `probe-one` removal grants were
  used. Its time-bounded VM hold is not a new lifecycle grant. The two-GET diagnostic
  authorization was also completed, not permission to reopen the listener.
- Retained roots, current v9 data, credentials, fixtures, archives, stopped records,
  failed attempts and later writes remain in custody. No cleanup/restore is approved.

## Remaining work

1. **Native pages/Runners:** step-6 source removal and native x86_64 build/check/
   export/verification passed for **`dc38af9`**. No installed retirement occurred.
   The [combined plan](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation)
   owns remaining actual-occupant inventory, target-specific removal rehearsal and
   explicit per-target removal delivery. Both retained targets still keep Cockpit Runners.
2. **Tailnet:** [the implementation plan](tailnet-integration-plan.md) is written;
   host UI/backend, automatic project enrollment and Tailnet source/installed
   retirement remain unimplemented. Begin with its bounded native/runtime/credential
   decisions; it does not reopen runner step 5 or authorize either installed removal.
3. **Broader acceptance:** physical-keyboard/real terminal-editor use, wider
   Access/error/profile-menu and CLI/provider combinations, intended-client routing,
   aarch64 and whole-product/release acceptance remain outside the bounded delivery
   claim. They are not new blockers for completed step 5; follow the
   [terminal](terminal-integration.md), [CLI](project-clis.md) and
   [native validation](native-validation.md) owners when that work is selected.
4. **Other product work:** [Sodaspaces](sodaspaces-plan.md), [Project OS](project-os.md),
   [Services/AI](services-and-ai-plan.md) and the [installer](coreos-installer.md)
   retain their own selected scope and remaining work. Runner completion does not
   complete or authorize those roadmaps; consult the relevant guide, not the full archive.

## Retained fixtures and evidence

- **Isolated runner VM:** `soda-native-runners-2cdc238`, custody under
  `.artifacts/runners-vm-2cdc238/`. Last installed candidate was `19824ec`; `baseline`
  remained running/enabled, while `probe-one` local account/state was removed only
  after a verified private archive. Both provider records/history and credential
  inputs remain. Check the recorded owner/deadline before any separately authorized
  reuse; no current VM liveness is asserted here. The [isolated proof record](implementation-history.md#step-4-bounded-x86_64-completion--19824ec-installed-overlap-departure-and-exact-remove-passed)
  preserves each case's actual revision, including earlier jobs/reboot.
- **Local browser fixture:** `sodaos-local-forgejo` at `localhost:3300`, volume
  `soda-pages-0f1d2b1-data`. Retain its account/repository, OAuth state and private
  inputs. `.local/screenshot-fixture/create-output.txt` is the preserved regular
  credential file, not a worktree symlink. Use the [screenshot guide](screenshot-capture.md)
  for login; do not recreate fixtures or expose credentials to recover a stale path.
- **Worktrees:** the original 17 linked worktrees were removed at the user's request.
  Step 6 added one fresh detached build checkout at `.artifacts/step6-native-dc38af9/`,
  retained with its native outputs and restricted copy of the local fixture input.
  Deleted earlier worktree outputs cannot be cited as retained evidence, even where
  the archive names them.

| Evidence | Location |
| --- | --- |
| Step-6 source/native checks, export and offline packaging delta | `.artifacts/step6-source/`; sealed export `export/x86_64/` |
| `soda-test` delivery, backups and failed/corrected observers | `.artifacts/step5-cutover-19824ec/` |
| Validation delivery and final completion receipt | `.artifacts/step5-validation-b5af330/`, including `completion.json` |
| Guest-side paired backups/rehearsals on both retained targets | `/var/lib/soda-native-pages-19824ec-cutover/` |
| Stopped exec-record inspection | `.artifacts/step5-exec-api-7e39229/` |
| Final isolated runner caller/removal/page evidence | `.artifacts/r4-overlap-06/`, `.artifacts/r4-departure-03/`, `.artifacts/r4-remove-01/`, `.artifacts/r4-final-pages/` |

Other retained fixtures and private evidence keep their existing custody; moving
documentation does not authorize cleanup or re-execution.

## Latest built source candidate

**Source candidate:** `dc38af94c0b6a0eba7db5c06a8a71bcb6828c414` removes only
Cockpit's runner presentation and updates staging/verifier, installed operator/
CLI-native drivers and current operator journeys. Tailnet/React/PatternFly and
native runner services/CLI/backend remain. Added native browser failure/busy/
confirmation parity coverage; old Cockpit overlap inputs are rejected in favor of
separately gated CLI/native contention. That revised installed case is authored,
not executed on an appliance.

Passed focused local checks, then fresh pinned native x86_64 `build-native.sh`,
`check-native.sh` (including the full source suite and 11 packaging tests), bundle
export and independent verification. Two optional Python source checks skipped.
The verified predecessor-to-candidate comparison removes exactly the old runner
package's 40 inventory entries (38 files), nothing else; Tailnet files are byte-
identical. Synthetic staging also proves stale runner dist output is not copied.
This is an offline packaging comparison, not an installed occupant inventory:
retained targets include older hashed assets requiring explicit review.

Export: `.artifacts/step6-source/export/x86_64/`.
`build-info.json` SHA-256:
`8ca9ef5b170742f1804750f2071d62227982606abaddb111d029597d750e4a40`.
Logs/delta: `.artifacts/step6-source/`; detailed scope in
[history](implementation-history.md#step-6--source-retirement-parity-review).
No retained appliance contact, VM/provider action, installed removal or retained-
state cleanup occurred. Source/native export acceptance is not per-target retirement.

## Latest local change

Documentation only: added `docs/tailnet-integration-plan.md`, covering native host
UI parity, operator-managed OAuth enrollment, per-project authority/native lifetime,
credential isolation, staged implementation, focused/native acceptance and scoped
Cockpit retirement. Linked it from owning guides and replaced permanent Cockpit-
Tailnet placement wording with the selected, not-yet-implemented direction.

Grounded the plan in current page/auth/schema, project/helper/systemd, Tailnet and
packaging callers plus the retained upstream research. Checked affected links,
source paths, ownership/scope consistency and `git diff --check`. No application
code, builds, application tests, target contact or network/provider mutations ran.
