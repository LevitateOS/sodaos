# Runner native validation preparation

This guide implements the runner-owned preparation in
[runner step 4](runners-port.md#4-prepare-product-owned-native-journeys-and-a-paired-candidate).
It is not execution permission, an appliance updater or a second source/native gate.
The [lane handoff](forgejo-soda-pages-plan.md#implementation-lanes-and-handoff)
continues to own the shared browser driver, authentication and shell/schema delivery.

**Current status:** runner inputs, callable scenarios, fixed read-only native
observations and provider dispatch/exact-run observations have source implementations.
The historical candidate `9476858` passed the ordinary aggregate source gate and
focused local input/transport-double/filesystem checks; this is not a fresh receipt
for the pulled combined checkout. Native connection/logout and page bodies have
now landed (`6097564`, `a77dea1`), followed by navigation retirement and cache/history
source. The explicit `97a2d5e` handoff is incorporated. The existing installed driver
now calls the runner phases through a distinct branch with isolated actor contexts
and a one-shot page/actor/path/body guard. Its local source-backed tests use doubles;
no installed runner phase has been executed. Follow the revised step 4 sequence for independent local work, cross-lane
dependencies and execution approvals rather than treating all three as one hold. No target/provider inputs have been selected, and no matching
native candidate build/check/export has run for this work. Do not call step 4's
verified-candidate exit complete or these scenarios provider-validated.

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
uses operator first, denied administrator second; its target, origin, revision and
CA must match the runner input. Existing repository fields remain required by the
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
tree hashes and listener policy; other inventory changes are refused.

A provider status and the exact proof PID/start/cgroup are observations, not full
process-tree cleanup or provider-online/busy acceptance. Real idle/active lifecycle,
CLI/Cockpit/web concurrent operations, interrupted/partial outcomes, retained
activation and boot policy still need step 5's exact native execution. No reboot,
provider deletion, fault injection, prior-version fixture creation or blanket
cleanup phase is implemented or implied here; schedule those separately under that
plan's explicit grants. The remaining provider record/history after local Remove
must be checked through approved native provider access, never inferred from the
local descriptor disappearing.

## One paired candidate and maintenance owner

Use the existing producer/verifier rather than a new manifest format or updater:

```sh
# Only after the shared source handoff, with its exact clean revision and pinned
# matching-native Linux toolchain, in a fresh worktree/output location:
scripts/build-native.sh x86_64
scripts/check-native.sh x86_64
.artifacts/native/x86_64/tools/soda-artifacts bundle \
  --source "$PWD/.artifacts/native/x86_64" --arch x86_64 \
  --revision "$(git rev-parse HEAD)" --out /private/new-export-parent/x86_64
```

The export parent must already exist; outputs must be fresh. Use the independent
matching-native aarch64 recipe on that architecture, not emulation or a sibling
barrier. No script here creates a fixture, installs the appliance or publishes
artifacts. The host default remains Go 1.27.0, but the required Go 1.26.7 is now
available at `.artifacts/runners-step4-continued/toolchain/go/bin/go`, downloaded
from go.dev and verified against its published SHA-256. Select its `bin` directory
with command-local PATH and `GOTOOLCHAIN=local`; no shared pin or host installation
changed. The `97a2d5e` page/driver handoff is incorporated. Toolchain availability and editor
handoff are no longer blockers; only actual build/check/export receipts establish
candidate readiness. Broader Soda-pages acceptance is still separately recorded.

Review this affected set against the **real sealed inventory and target state**:

| Artifact | Delivery/compatibility responsibility |
| --- | --- |
| `images/dashboard.oci` | The running backend is in this image. Verify its source/config digest and update the target's actual image pin through the approved recipe. A separately staged `soda-dashboard` binary does not replace this service. |
| `rootfs/usr/local/libexec/soda/soda-host` and `soda-runners` | Paired management lock/protocol owners. Stop new management admission and wait for old CLI/helper operations before replacement. Restarting `soda-host` affects project/browser terminal integration too; declare that interruption. |
| `rootfs/usr/local/libexec/soda/soda-runner-launch` | Matching Forgejo-only launcher. It is not a management-lock owner. Do not restart existing listeners merely to replace this file. |
| Canonical Forgejo custom payload/assets/notices | Consume the Soda-pages handoff and `forgejo-payload.json`/sealed export, not a second handwritten file list. Review native template reload/service interruption with that owner. |
| Existing runner units, sysusers/tmpfiles and package requirements | Inspect actual effective confinement and `rpm` versions. Change only separately reviewed deltas; do not recreate accounts, assign new UIDs, rewrite descriptors or upgrade clients incidentally. |
| Obsolete installed `soda-runner-helper` | Explicit retirement only during approved maintenance after all old writers exit. Its absence in new source/bundles does not remove an installed independently callable writer. |
| Cockpit Runners and Tailnet | Preserve both payloads and backing logic. Retirement of only the runner presentation remains step 7, not this candidate preparation. |

The existing full producer also builds Project OS/service images. Their presence
in a bundle does not select them for delivery or permit replacing project roots.
No standalone runner image, GitHub client, whole-appliance upgrade or first-install
recipe is introduced.

Before calling the candidate ready, record the actual source revision, native arch,
sealed inventory/checksum, package/version evidence, shared browser handoff, actual
target/schema/private inputs, affected bytes, required interruptions and exact
executor. Follow [credential/schema rehearsal](dashboard-credentials.md#controlled-existing-state-rehearsal-before-live-deployment)
on copied state without starting cloned listeners with copied live credentials.
Common build/schema/backup/delivery phases have one executor and one exact-candidate
receipt across both lanes. Later target writes require fresh applicable backups;
old evidence is not lossless rollback.

**Remaining step-4 exit:** finish applicable local driver/guard validation and
remaining scenario preparation, select target/provider inputs for approval, and
use the prepared pinned builder to produce/check/inspect the matching native
export and target-specific compatibility recipe. The local parser/transport-double/
filesystem tests below are preparation evidence, not substitutes for those exits.

## Local checks

The ordinary `bun run check:source` owns TypeScript, emitted-page/Cockpit/Go and
Python checks. Focused cases are `tests/frontend/runner-journey-input.test.ts` and
`tests/build/test_runner_state.py`. They use synthetic private input/command outputs
and owned filesystem fixtures; workflow bodies receive syntax checks, not provider
execution. Do not add a second installed readiness gate or relabel these as real
browser/root/provider acceptance. Exact executed checks live in the
[handoff](implementation-status.md).
