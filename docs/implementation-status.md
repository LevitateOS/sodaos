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

- **Authorized:** routine local source implementation, builds and tests for selected
  work, including existing local fixtures within their approved scope. No repeated
  permission handoff is needed for that work.
- **Not authorized by the completed delivery:** another retained-target cutover,
  service/project/VM lifecycle, new native fixture, real provider registration/job,
  network/trust/capability change, publishing/automatic CI or cleanup. Such effects
  need their applicable target/action grant; old approvals do not renew themselves.
- **Cockpit Runners retirement is not approved.** Keep its package, backing logic and
  tests pending accepted replacement parity and separate removal scope/delivery approval.
- The isolated runner fixture's one reboot and exact `probe-one` removal grants were
  used. Its time-bounded VM hold is not a new lifecycle grant. The two-GET diagnostic
  authorization was also completed, not permission to reopen the listener.
- Retained roots, current v9 data, credentials, fixtures, archives, stopped records,
  failed attempts and later writes remain protected. No blanket pruning or blind
  restoration is authorized; old backups may predate later writes.

## Remaining work

1. **Native pages/Runners:** step 6 remains separately gated. Accept the delivered
   parity/coverage map, then authorize the coordinated Cockpit Runners removal
   candidate and target delivery. Reuse applicable isolated mutation evidence; do
   not create retained capacity merely to fill a matrix. Preserve Tailnet, ordinary
   Cockpit and runner backend/native contracts. The [combined plan](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation)
   owns this work; completed steps are not a checklist to replay.
2. **Broader acceptance:** physical-keyboard/real terminal-editor use, wider
   Access/error/profile-menu and CLI/provider combinations, intended-client routing,
   aarch64 and whole-product/release acceptance remain outside the bounded delivery
   claim. They are not new blockers for completed step 5; follow the
   [terminal](terminal-integration.md), [CLI](project-clis.md) and
   [native validation](native-validation.md) owners when that work is selected.
3. **Other product work:** [Sodaspaces](sodaspaces-plan.md), [Project OS](project-os.md),
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
- **Worktrees:** all 17 linked worktrees were removed at the user's request. The main
  checkout, Git history and surviving exports remain. Deleted ignored worktree
  outputs cannot be cited as retained evidence, even where the archive names them.

| Evidence | Location |
| --- | --- |
| `soda-test` delivery, backups and failed/corrected observers | `.artifacts/step5-cutover-19824ec/` |
| Validation delivery and final completion receipt | `.artifacts/step5-validation-b5af330/`, including `completion.json` |
| Guest-side paired backups/rehearsals on both retained targets | `/var/lib/soda-native-pages-19824ec-cutover/` |
| Stopped exec-record inspection | `.artifacts/step5-exec-api-7e39229/` |
| Final isolated runner caller/removal/page evidence | `.artifacts/r4-overlap-06/`, `.artifacts/r4-departure-03/`, `.artifacts/r4-remove-01/`, `.artifacts/r4-final-pages/` |

Other retained fixtures and private evidence keep their existing custody; moving
documentation does not authorize cleanup or re-execution.

## Latest local change

Documentation only: separated this snapshot from the historical record and retargeted
historical deep links. Checked archive-body preservation, local links/anchors and
`git diff --check`. No builds, application tests, target contact or native actions ran.
