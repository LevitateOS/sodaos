# Runner native validation

This guide owns runner test inputs, per-effect gates, native observations and
installed scenarios. [Runner contracts](runners-port.md) own product behavior;
the [combined plan](native-pages-runners-plan.md) owns coordination/retirement.
Use the [current handoff](implementation-status.md) for installed state, active
permissions and evidence, not historical fixture approvals as defaults.

## Owners and entrypoints

| Owner | Contract |
| --- | --- |
| `tests/installed/runners-input.ts` | Strict phase/target/actor/runner/provider inputs and distinct permission strings; no native effects. |
| `tests/installed/runners.ts` | `loadRunnerInput(file, permission)` validates restricted inputs and clean source revision. `exerciseRunners(operatorPage, deniedPage, input, permission, permit, evidence)` exercises one phase on pages supplied by the shared driver. It does not authenticate, navigate, launch a browser or substitute responses. |
| `tests/installed/runners-native.ts` and `runner-state.py` | Pinned SSH transports the actual product-owned read-only observer on stdin. It verifies native hostname/architecture, CLI inventory, accounts, credential modes/hashes, effective confinement and selected state hashes. No arbitrary remote program, account/path selector, native mutation or repair operation is accepted. |
| `tests/installed/runners-provider.ts` | Existing Forgejo 15.0.7 APIs: version/repository preflight, one workflow dispatch returning its run ID, or a GET of that exact run. The optional command seam is for local unit tests only; installed callers use real curl, never doubles. |
| `tests/fixtures/runner/native-support.yaml` | Manual-only two-step trusted job with a unique observation and an explicit 0–600 second hold. It records native UID/account/PID/start and a retained proof file, and checks a shared workspace in the second step. It neither checks out code nor installs tools. |

The integrated driver is `tests/installed/sodaspaces.ts`, which owns private browser
startup, native authentication/consent and one-shot request admission. After explicit
execution approval, its command shape is:

```sh
SODA_NATIVE_VALIDATE=ACTUAL_HOST bun tests/installed/sodaspaces.ts \
  /private/auth-input.json /private/fresh-browser-home --allow-auth-transitions \
  --runner-phase /private/runner-input.json --allow-runner-list
```

Replace the last flag only with the exact phase's approval. The existing auth input
uses operator first, denied actor second (an administrator in the default cross-role
fixture); its target, origin, revision and CA must match the runner input. Existing repository fields remain required by the
shared auth-input contract, but runner mode does not enter repository/environment/
terminal journeys. It creates an isolated second cookie context, checks fresh Soda
and Forgejo identity/role facts, and uses automatic native Runners entry/consent,
not the drawer-only OAuth helper. No browser security bypass or cookie seeding.

The native asset preflight checks the full emitted module graph and settings CSS,
including epoch-versioned URLs, against source bytes and the installed proxy's
`max-age=0, must-revalidate` contract. This is not the stock-preview six-hour cache
contract or a substitute for installed backend artifact verification. Do **not** add another login
script or run `bun tests/installed/runners.ts` expecting a journey: this module has
no standalone CLI. Preserve these driver contracts:

1. Call `loadRunnerInput` before any authentication/native work; retain its existing
   browser-origin/CA/private-home and clean-revision checks. Verify actual delivered
   bytes with the existing installed-artifact verifier, not the caller's revision
   string alone.
2. Supply the two already connected native Runners pages, matching `operator_id`
   and `denied_id`. Its own actor fixture must establish an operator **without**
   site-admin rights and a site administrator **without** Soda operator authority.
   That remains the default and is mandatory for effectful phases. A retained
   `list`-only observation can instead supply `native_admins` with exact boolean
   `operator`/`denied` expectations. The driver still reads and checks both native
   and Soda identities/roles freshly, and records their actual facts; it never
   treats site administration as Soda authority. Other phases reject this field.
   This is an explicitly labelled retained read case, not a replacement for the
   independent cross-role native proof.
   The module requires the native `/?soda-view=runners` mount and original actor;
   it refuses the old standalone page and cannot manufacture the missing page body.
   The current adapter is for the root-mounted origin used by the existing runner
   caller; a different `AppSubUrl` needs the page owner's explicit caller port.
3. Adapt its existing exact-request guard to `permit(actor, route, serializedBody)`.
   Allow exactly one same-origin POST to `/-/soda` plus that route, with the declared
   actor and exact body. Consume it before transmission; clear pending permission
   on abort. The body can contain a registration token: never log it, persist it,
   put it in browser storage or expose it to reactive UI state. No blanket POST
   allowlist. Provider phases use their separately gated native API client, not
   borrowed browser cookies.
4. Retain the supplied `RunnerEvidence` object in a fresh restricted attempt,
   including on failure. Catch only the fixed stage/error summary, never browser
   exceptions, request bodies or provider responses. No screenshots/traces/network
   logs around credentials. No automatic retry or cleanup on an unconfirmed result.

`operator.ts`/`operator.sh` and their existing Cockpit/Tailnet checks are unchanged.
They are not silently opted into registration, jobs or destructive checks.

## Exact private inputs and effect gates

A restricted, regular, non-symlink JSON input file (maximum 16 KiB) supplies the
following common fields. This is a **synthetic example**, not an approved target,
valid candidate revision or real credential location:

```json
{
  "phase": "list",
  "target": "runner-fixture",
  "architecture": "x86_64",
  "revision": "0000000000000000000000000000000000000000",
  "origin": "https://fixture.invalid",
  "ca_file": "/private/fixture-ca.pem",
  "ssh_config": "/private/fixture-ssh.conf",
  "ssh_host": "runner-fixture",
  "operator_id": "1",
  "denied_id": "2",
  "runner_id": "probe-one",
  "preserved_ids": ["baseline"]
}
```

Set `SODA_NATIVE_VALIDATE` to the approved **actual** hostname and pass exactly
`--allow-runner-PHASE`. Other permission strings, unknown fields, inline token
values, arbitrary URLs/commands, duplicate IDs and a fixture ID also listed as
preserved are rejected. All other existing local runners must be declared in
`preserved_ids`; undeclared retained capacity stops the phase before effects.
The private SSH config owns the root key and known-host pin; the probe disables
forwarding, agents, connection sharing and local/remote command overrides. It
does not establish new routing, trust or host access. CA and SSH files must also
be restricted regular files; browser trust stays with the shared driver.

| Phase / exact opt-in | Effects and required additional input |
| --- | --- |
| `list` / `--allow-runner-list` | Root/CLI and operator-page reads plus nonoperator GET denial. No mutation, including no expected-denied POST. Opening the page must preserve the observed state. |
| `register` / `--allow-runner-register` | Adds the declared absent local runner and starts/enables it. Requires `registration` below and an **already approved/created system provider record**. The probe never creates/resets that provider record. |
| `start`, `stop`, `restart` / corresponding exact opt-in | One exact-ID browser action on an existing fixture runner. Start enables boot start; Stop disables boot start and may interrupt a job; Restart enables boot start and may interrupt a job. No automatic sequence or recovery. |
| `remove` / `--allow-runner-remove` | Exact-ID browser confirmation destroys the declared disposable local account/state/work/credentials. Provider registration/history remain; their inspection/cleanup is separate. No preserved runner may be this target. |
| `dispatch` / `--allow-runner-dispatch` | One real provider-scheduled trusted job. Requires `provider` below, separate provider credentials/scope and a running fixture listener. Returns the exact provider run ID/number. An unknown response is **not** authority to retry. |
| `job` / `--allow-runner-job` | Reads the exact returned provider run and native proof on the declared runner, including after Stop. Reports actual status and PID/start/cgroup observation; successful runs must have completed both fixture steps. No polling, latest-run selection or mutation. Missing proof remains unverified (including cancellation before assignment). |

For registration, add only:

```json
"registration": {
  "scope": "system",
  "uuid": "33834eef-e758-48c4-a676-1745426747aa",
  "labels": "soda-native:host",
  "token_file": "/private/approved-runner-token"
}
```

Use the actual provider-issued UUID/token, not these example bytes. The password
field is cleared after submission and on failure. The native token file remains
provider-owned persistent state; the probe does not print or migrate it.

For dispatch or exact-run observation, add only:

```json
"provider": {
  "token_file": "/private/approved-provider-pat",
  "repository_id": "7",
  "repository_path": "fixture/trusted",
  "workflow": "native-support.yaml",
  "commit": "0000000000000000000000000000000000000000",
  "observation": "unique-approved-attempt",
  "hold_seconds": 0
}
```

`job` additionally requires `provider.run_id`, a positive safe integer from the
**actual dispatch response**, not the local runner ID or repository run number.
The client verifies the repository ID before dispatch and binds observation reads
to repository, workflow filename, full commit and exact observation/hold inputs.
It does not list runs and choose a newest one. Provider response bodies, titles,
event payloads and arbitrary URLs never enter evidence. The PAT travels only on
curl's stdin header channel, not argv/environment/logs. Curl uses the declared CA,
no redirects, no ambient config and bounded output/time. No borrowed CLI/setup
credentials, provider role inventory or custom scheduling mechanism.

Before publishing the fixture, select a unique provider label for this capacity
and explicitly approve the exact trusted repository/commit. Publishing or changing
that workflow/label is **not** done by these modules. `hold_seconds` is explicit
because an active-job interruption test differs from a short successful job. The
workflow retains only its exact marker/proof resources:

- `.soda-native-step-OBSERVATION` in the job workspace;
- `work/soda-native-proof-OBSERVATION.json` under that runner's account home.

The first creation is exclusive; do not reuse an observation or delete an earlier
proof to retry. The second step updates only that proof. Host execution remains
trusted-repository execution; neither the workflow nor these tests implements OCI
isolation or makes provider credentials safe from arbitrary trusted job code.

## Native observations and preservation limits

The read-only observer can also be transported by the existing approved SSH tools
as `python3 -I - ACTUAL_HOST ID ...` with the source file on stdin and matching
`SODA_NATIVE_VALIDATE`. The TypeScript adapter applies a stock remote 75-second
timeout to the **observer/CLI-read process group**, not any runner service/job,
plus a longer client bound. Closing SSH alone is not remote termination proof.

It uses the existing root-only CLI, holds the shared native management lock while
reading selected trees/confinement, then requires unchanged CLI observations.
Each hash traversal is descriptor-relative, does not follow job-created symlinks,
and refuses changing files, unsupported live entries and size/time bounds.
Private filenames/contents are not exported; aggregate hashes and explicit native
identity/mode facts are. The reader requires the paired CLI protocol/shared lock;
an older target that cannot supply it needs a separately reviewed quiesced baseline
procedure, not fabricated compatibility, a new lock writer or a cleanup shortcut.

These reads are **not a consistent backup while listeners/jobs are writing**.
Management locking does not freeze job files. Preserve partial states and report
unavailable/changed observations; separately approved quiescence and fresh backups
remain required for recoverable maintenance. Lifecycle checks compare fixture
UID/GID/home/shell and registration/credential hashes, allowing legitimate job work
changes. Declared preservation runners additionally require unchanged full selected
tree hashes, process identities and listener policy; other inventory changes are
refused. Use idle/quiescent preservation baselines. Unrelated live job/process/file
changes invalidate the comparison; they are not automatically labelled data loss.

The reader now records the boot UUID and bounded recursive cgroup-v2 PID/start/
effective-UID tuples for each selected unit. It refuses missing/changing membership,
foreign UIDs, links, unsupported layouts and oversized trees. Effective KillMode,
ProtectControlGroups and Delegate policy are checked alongside existing confinement.
Post-operation reads additionally check the fixture's prior tuples against procfs,
even if a process has left the unit; PID reuse with a different start is not survival.
Stop/Remove require an empty current scope, and Stop/Restart/Remove refuse surviving
prior incarnations. Unexpected reboot during a single phase is an error. These are
bounded before/after observations, not a process journal or proof against arbitrary
future forks/hostile jobs. Native execution must still establish the actual process
and provider outcomes; a green listener or the single workflow marker is insufficient.

Real idle/active lifecycle, CLI/Cockpit/web concurrent operations, interrupted/partial
outcomes, retained activation and boot policy need the combined plan's approved
native execution step. No reboot,
provider deletion, fault injection, prior-version fixture creation or blanket
cleanup phase is implemented or implied here; schedule those separately under that
plan's explicit grants. The remaining provider record/history after local Remove
must be checked through approved native provider access, never inferred from the
local descriptor disappearing.

## Prepared installed cases — explicit execution later

These cases use the existing phase driver and native operator tools. They are not
a second runner harness or executable approval. Each phase invocation gets a fresh
private browser/evidence directory and exact input file; do not edit an input after
approval or reuse an occupied attempt. Native commands below run only in the approved
fixture's pinned **root SSH session**. `probe-one` is a synthetic disposable ID, not
permission to operate on an existing runner. Substitute only the reviewed exact ID.
Never enable tracing, record password entry or print client state/command lines.

### Execution proposal to fill before any effects

Record these concrete values in the private attempt's existing approval/evidence
record, not in source or a new readiness database:

| Required selection | What the approval must bind |
| --- | --- |
| Candidate | Full revision, architecture, passing check/export receipt, manifest and actual delivered-byte verifier receipt. |
| Target | Actual hostname, origin, CA, SSH host-key pin/config, private browser home, existing versus newly approved fixture and affected activation components. |
| Actors | Effectful fixtures require a nonadmin Soda operator and nonoperator site administrator; list-only input may declare actual roles through `native_admins`. Root/Cockpit is separate. Record stable IDs and restricted login-input paths, never passwords. |
| Provider | Already approved system runner record: native provider numeric ID/inspection URL and UUID, unique label, exact repository ID/path, workflow commit/file, provider actor/PAT scope and restricted token files. Local ID is not provider ID. |
| Preservation | Every existing local runner ID/account/state and relevant project/root/terminal baseline; required quiescence and fresh backup scope. Unknown/unsupported inventory stops activation. |
| Disposable resources | Exact local runner ID, provider record, unique observation per job, hold duration and exact run IDs returned by dispatch. No unrelated capacity may share the fixture label. |
| Allowed cases | List/registration/job/lifecycle, 20-second lock hold and cancelled waiter, reboot, local Remove and provider cleanup separately. Each grant names its actual effects, not an `all` flag. |
| Failure handling | Retain evidence/partial state; stop automatic writes. Name the person who may approve further observation or corrective actions. No automatic retry, old-snapshot restore or cleanup. |

Past selections and consumed grants are recorded in the [handoff](implementation-status.md)
and linked history. They are not defaults for these case inputs.

### A. Registration and successful trusted job

1. With the provider administrator's separate grant, create/select the exact system
   runner record in native Forgejo. Record its numeric ID/UUID and inspection URL;
   supply its token only through the restricted registration file. Select a unique
   label and publish the manual-only trusted workflow at the approved full commit.
   Neither the Soda page nor the test publishes or resets this record.
2. Run `list`, then the separately approved `register` phase for the absent local ID.
   Require native account/UID/private-state/confinement/slot observations and page
   inventory agreement; retain all preservation hashes. Never erase partial creation
   to retry. Inspect actual installed runner/package versions, not builder versions.
3. Run `dispatch` with `hold_seconds: 0` and a never-used observation. Retain the
   returned run ID/number immediately. If the response is lost, inspect the exact
   approved workflow/observation in native Forgejo; do not dispatch again or select
   a newest run automatically.
4. Use `job` with that exact returned ID. Queued/running status is not success;
   a later separately invoked read may observe completion without another dispatch.
   Require matching repository/commit/inputs, two completed fixture steps and native
   account/proof correlation. Inspect the provider's actual result and native runner
   online state. Keep its record/history; do not add provider health to Soda inventory.

### B. Idle and active-job Stop/Start/Restart

1. With no active/queued work on disposable capacity, invoke `stop`, `start`, then
   `restart` as separately selected phases. Stop must be inactive/disabled with an
   empty cgroup and no surviving prior tuples; Start must be active/enabled without
   re-registration; Restart must be active/enabled with old incarnations gone.
   UID/home/credentials/client version and all baseline runners must be unchanged.
2. For **each** active interruption, use a new approved dispatch/observation with
   `hold_seconds: 600` (or another approved bounded value). Before interruption,
   invoke `job` and require the exact run to be running with a live step-1 native
   proof and recursive process membership. Do not interrupt a merely queued run or
   infer activity from the listener; restart the procedure with a new approved
   observation if the hold elapsed, never by replaying an uncertain dispatch.
3. Invoke exactly one approved `stop` or `restart`. The driver compares its own
   preflight process set and post-attempt procfs survivors on the same boot, including
   failure paths. Inspect the exact provider run afterward with `job` and its native
   provider page until the observed terminal outcome is known. No success is expected
   merely because systemctl returned; cancellation/failure/status lag must be recorded.
4. Start after Stop only under its own grant, check boot policy and unchanged
   identity/credentials, then use a fresh short job to prove usable capacity. Repeat
   the active Restart case independently. Existing job writes/proofs remain retained.

On a changing/unavailable process or state read, preserve the failed receipt and
perform only separately permitted fresh **reads**. Do not rerun the mutation to get
clean evidence. The observer does not freeze jobs or guarantee atomic backups.

### C. Installed caller overlap and a bounded cancellation failure

The existing `sodaspaces.ts --runner-phase FILE` caller now authors two explicit
phases for steps 1–2 and 4 below: `overlap` / `--allow-runner-overlap` and
`departure` / `--allow-runner-departure`. Both require an idle running disposable
listener and use one 20s lock hold plus one exact native-page Restart. Overlap also
clicks one exact Cockpit Stop and runs one ordinary CLI list; departure navigates
to native Issues only after the POST is observed, then returns/reloads without a
new mutation permit. They never dispatch jobs, register, remove or reboot.
`overlap` alone additionally requires `cockpit: {origin, password_file}` in the
restricted input, with a distinct canonical HTTPS origin and that fixture's root
password file. Trust its actual certificate in the new private browser home first;
no TLS bypass/global trust or borrowed credentials. It reuses the operator login
helper and opens only Runners, **not Tailnet or its advertisement effect**. The
existing native-page actor/CSRF/one-shot guard remains in force. These new phase
ports have local input/postcondition/type coverage; installed results belong in
the leading handoff, not this procedure. The separate CLI waiter remains step 3.

Use an idle disposable runner, with the native page and Cockpit already connected
through their real operator logins. The full phase driver performs preflight reads,
so do not start it behind an intentionally held lock and mistake preflight timeout
for mutation contention. This case uses the prepared native UIs and the root CLI.

1. In root SSH session A, verify the existing lock is a root-owned mode-0600 regular
   file (no link). After explicit lock-hold approval, run:

   ```sh
   test -f /run/lock/soda/runners.lock && test ! -L /run/lock/soda/runners.lock
   test "$(stat -c '%u:%a' /run/lock/soda/runners.lock)" = 0:600
   flock --exclusive /run/lock/soda/runners.lock sleep 20
   ```

   Use a separate SSH session for contenders, so they cannot inherit the holder's
   lock descriptor. The holder releases normally after 20 seconds; do not kill
   unknown processes or leave an unbounded operator-owned lock.
2. During that interval, click exact-ID Restart once in the native page and Stop
   once in Cockpit, with both actions expressly approved. In session B request an
   ordinary CLI read, retaining its timing/status privately:

   ```sh
   printf '{}\n' | /usr/local/libexec/soda/soda-runners list
   ```

   Neither a read nor mutation may finish through the held lock. After release,
   require bounded completion through the shared native owner, compare both UIs
   and native observations, and record which final enabled/active state resulted.
   Do not assert a predetermined winner. No registration/deletion/job occurs here.
3. Independently repeat a **cancelled CLI waiter**, without competing mutations:
   create a mode-0600 file containing `{"id":"probe-one"}` in the run's restricted
   directory. In session B while session A holds the lock, start:

   ```sh
   timeout --preserve-status --signal=TERM --kill-after=5s 2s \
     /usr/local/libexec/soda/soda-runners restart < /PRIVATE/RUN/restart.json
   ```

   Stock timeout supervises only its own command/process group; do not send signals
   to a name lookup or stale recorded PID. Record status and sanitized CLI error
   without `set -e` hiding them. Require a graceful nonzero/context-cancelled result,
   not a forced KILL/timeout fallback, while the independent holder still owns the
   lock. No restart may occur; original listener PID/start/boot policy and all state
   hashes must remain. This proves an installed CLI cancellation under held admission;
   the local lock tests separately establish the exact internal wait/substep paths. After normal lock release, verify an ordinary list and separately approved
   lifecycle action still work. Do not generalize this to killing a dispatched web
   operation: aborting a browser request does not prove native cancellation.
4. Independently prepare an unconfirmed browser request: hold the same lock for
   20 seconds, click one approved exact-ID Restart in the native page, establish
   that its POST was dispatched, then navigate to native Issues before the holder
   releases. Record only the fixed operation path/timing, never headers or bodies.
   The window is established by the held lock, not a guessed sleep after dispatch.
   After release, use fresh native reads to establish whether the action occurred.
   Return/refresh the page and require zero additional mutation POSTs; a departed
   request must not become automatic success or be replayed. Cancellation can prevent
   the operation or leave it unconfirmed; do not assume either outcome in advance.
   This permits only the declared request, client navigation and lock hold—not
   provider/host-network fault injection or another native action to repair it.
5. Retain per-caller dispatch/return times, sanitized results and process/state
   receipts. Local lock tests already cover internal Restart substeps; this installed
   case establishes real caller overlap and cancellation, not every possible kernel,
   account-deletion or filesystem fault. Additional intrusive fault injection needs
   an exact reviewed procedure/grant; it is not silently included.

### D. Pre-existing activation and separately approved reboot

1. Before any affected delivery, retain native state/process/boot and provider
   observations for the pre-existing runner and all other protected roots. If no
   prior-version baseline exists, obtain separate permission to create that fixture;
   do not invent one by downgrading a retained runner or copying live credentials.
2. Use the combined plan's copied-state rehearsal and affected-artifact activation
   recipe: fresh backups under approved quiescence, old-writer drain and paired
   management delivery. Do not run cloned listeners. After activation and first
   opening the new page, compare descriptor/account/UID/work/credential/client and
   enabled/running state. Opening the page must not rewrite or activate anything.
3. Only with a distinct reboot grant, capture the enabled/running policy, all
   identities/registration/work hashes, boot UUID and exact candidate bytes. Request
   the normal native `systemctl reboot` on that same fixture. Do not add a reboot
   flag to a read-only runner phase or redirect to another target while it is offline.
4. After the authorized target returns, collect fresh **unlinked** observations;
   do not pass a pre-reboot process baseline to a single-phase survivor check.
   Require a changed boot UUID, unchanged accounts/credentials/work/client/policy,
   enabled listeners running and disabled listeners inactive. Compare native provider
   state and a separately approved fresh job. Never treat reused numeric PIDs across
   different boots as retained processes, or old backups as lossless rollback.

### E. Exact local Remove and provider aftermath

After separate destructive approval for the disposable ID, run `remove` and require
no local account/state, no remaining unit cgroup and no surviving pre-operation
process incarnations. Other declared runners and retained roots must survive.
Partial outcomes remain unconfirmed; do not run Remove again automatically.

Use the recorded **provider numeric ID/inspection URL**, not the local ID, to verify
that its native Forgejo registration/history still exists. The automated `job` phase
requires the local runner for native correlation and is not the post-removal history
reader. Inspect the exact saved run IDs via native provider UI. If provider cleanup
is expressly approved, remove only that record through native controls, verify its
outcome and retain job/history/evidence as the provider actually supports. No blanket
inactive-runner deletion, work-tree pruning or shared fixture cleanup.

## One paired candidate and maintenance owner
The [runner compatibility contract](runners-port.md#paired-artifact-compatibility)
owns the affected-artifact relationships. Use the [native build/export contract](native-support.md#build-and-artifact-contract)
and [installation maintenance procedure](installation.md#retained-sodaspaces-cutover),
not another build recipe or lane handoff here. Tool versions come from source pins;
current candidates, target state and results come from the handoff.

## Local checks

The ordinary `bun run check:source` owns TypeScript, emitted-page/Cockpit/Go and
Python checks. Focused cases are `tests/frontend/runner-journey-input.test.ts` and
`tests/build/test_runner_state.py`. They use synthetic private input/command outputs
and owned filesystem fixtures; workflow bodies receive syntax checks, not provider
execution. Do not add a second installed readiness gate or relabel these as real
browser/root/provider acceptance. Exact executed checks live in the
[handoff](implementation-status.md).
