# soda-release-tools

Operator tools for producing Soda candidates and installation media, inspecting
archives and retrieving verified upstream CoreOS inputs. This package supplies
`soda-build`, `soda-candidate`, `soda-artifacts`, `soda-candidate-check` and `soda-release`.

## Choose a command

| Command | Purpose |
| --- | --- |
| `soda-candidate` | Interactive or scripted wrapper around the admitted build controller. |
| `soda-build` | Dispatch a candidate or media build to an isolated worker. |
| `soda-candidate-check` | Verify candidate archives against architecture and exact source revisions. |
| `soda-artifacts` | Inspect OCI archives, fetch verified CoreOS inputs, convert private Butane, or check the Fountain pin against published releases. |
| `soda-release` | Stage immutable GitHub release assets and promote them after the native install smoke. |

Use `--help` on any binary to see its flags. Builds require native Linux
`x86_64`, clean committed Soda and Fountain checkouts, installed toolchains and an
operator-admitted controller with restricted worker configuration. Start in the
Soda checkout root. Follow the
[native support guide](../../docs/development/native-support.md#local-host-content-image-candidate)
for setup and input admission; a direct `cargo run` does not admit a controller.
For the source-to-download path, follow the [release workflow](../../docs/development/release.md).
The default build produces unqualified release media; staging and promotion are
explicit subsequent commands. Staging derives the media path from the candidate
and creates its exact tag when missing. Qualification consumes a protected native
collector result bound to the candidate, source, and downloaded ISO. An operator
pass flag or caller-authored observation does not establish release authority.

`soda-release collect --config /home/RELEASE-AUTHORITY/collector.json --candidate /BUILD/artifacts
--out /home/RELEASE-AUTHORITY/observations-attempt` runs the admitted collector as `soda-qualifier`
with an explicit public candidate snapshot read-only and a separate writable
attempt. The root-only configuration binds both executable hashes, completed
build/candidate/payload hashes, exact public-input hash/size inventory, purpose,
target, output parent and timeout. Every
configuration/executable/evidence ancestor must be root-controlled; a root-owned
file below a caller-writable directory is refused. The collector receives only
`--candidate`, `--out` and `--target` and emits exact-bound `collector.json` plus
the distribution's native observations. Only exit zero and confirmed native-unit
retirement produce `executed.json` and a sealed `release-evidence.json` inventory.
Production authority also requires `runtime` with `directory`, `manifest` and
`manifest_sha256`, plus the two fixed `credentials` entries `guest-ssh-key` and
`workspace-owner-password`, each with a protected source `path` and private
`sha256` pin. Credential sources are root-owned 0600 single-link files in the
authority file's sibling `credentials/` directory. Keep credential values and
their pins out of public observations. Systemd supplies private read-only copies
through `LoadCredential`; build workers receive no credentials.
On SELinux hosts, apply the installed policy's standard `/etc/credstore` mapping
to these admitted credential copies and their directory. A preserved home-file
label can prevent systemd's credential helper from reading them even when their
root ownership and 0600 permissions are correct.

The root-owned runtime manifest has `format: 1` and a `files` map of relative
names to exact SHA-256/byte identities. It stays outside the runtime tree. Every
runtime file must match that complete inventory, with no extra or symlinked
files, shared hard links or writable ancestors. Only that tree is mounted at
`/run/soda-qualification-runtime`; developer source and dependency directories
remain inaccessible. The collector keeps browser profiles, cookies and temporary
files in the attempt's fixed `.work/` tree. The controller records its original
inode before dispatch. After confirmed unit retirement it moves that exact tree
to private `OUT/consumed-work`, outside sealed observations; failed attempts keep
their scratch. Unknown retirement leaves it with the attempt. A successful
finalized journey may remove only that scratch when its frozen scope explicitly
sets `permitted_effects.retire_consumed_scratch: true`. Otherwise it stays retained
without an extra storage/process scan. Development fixtures can
exercise this access path or use the standalone collector boundary.

Maintained journey admission adds `resolved_input: {"path": "/ABSOLUTE/input.json",
"sha256": "HEX"}` to that same collector authority. The shared installed-driver
type `ResolvedQualificationInput` contains the requested scenario, returned
observation and references to prior canonical `request.json`, `executed.json` and
`observations/release-evidence.json` receipts (null when never produced). Store
credential paths only. The Rust controller treats business fields as opaque;
the journey validator owns their meaning.

Before dispatch, `collect` checks the protected source and exact hash, durably
creates `OUT/resolved-input.json` without overwriting, and records its path/hash
in the existing `request.json`. The snapshot is root-owned 0444 beneath the
private output directory and mounted read-only at
`/run/soda-qualification-journey.json`. Only this small document is copied; its
referenced runtime, credentials and old evidence are not duplicated. Recovery
checks the frozen snapshot against the retained request and authority pin,
without substituting a newer source document. A changed input needs a new output
attempt; existing attempts remain intact. Standalone collectors without a journey
document retain their existing request and observation formats. The maintained
`journey run` path below supplies its root-verified fresh resolved source directly;
the request/snapshot remain the same and the existing authority need not be
rewritten with a manually transcribed source hash. A fixed authority source,
when present, must still match.

`soda-release finalize-evidence --out /home/RELEASE-AUTHORITY/observations-attempt` recovers that
exact executed receipt without rerunning the collector or build. Failed,
cancelled, incomplete and mismatched attempts stay unqualified. Development
evidence can finalize but cannot qualify a production release. Docker, appliance
and upgrade qualification all require the protected production receipt; appliance
uses its sealed `appliance/native.json` rather than a smoke-pass flag. Upgrade
stamping retains a private collector locator; promotion rechecks that protected
source and incorporates the typed receipt into its authenticated release summary.
Preparation and staging carry the prospective source and immutable target assets
without claiming upgrade support. Neither the local receipt nor its locator is
an immutable release attachment. The locator is never published.

## Read-only qualification readiness

Build the operator and the maintained installed driver from this checkout:

```sh
cargo build --release -p soda-release-tools --bin soda-release
bun scripts/build-qualification.ts --out /home/OPERATOR/journey-build
```

Use the shared `QualificationScenario` in
`tests/installed/qualification/scenario.ts`. Pin the public executable bytes in
`driver.executable`; record the source revision independently from the shipping
candidate. Keep any working-change provenance alongside local development
checks; a driver base commit alone does not identify uncommitted source bytes.
The driver has no imports from private per-attempt implementations.

Supply `trust.known_hosts` and `trust.appliance_ca` as public path/SHA-256
references. The single-host SSH configuration must name the selected target,
explicit host, root user, protected operator key and pinned known-hosts file,
with `IdentitiesOnly yes`, `StrictHostKeyChecking yes` and `ForwardAgent no`.
No proxy, local command, include or wildcard configuration is executed. The
reader builds its own restricted SSH arguments and performs only fixed reads.
The CA is checked by a normal HTTPS login-page GET without authentication.
Credential values and their hashes do not belong in this scenario.

Keep `factory[].subject.required_checks` (native CI status names) separate from
`check_script_path` (the approved repository file) and its SHA-256. For example,
the retained fixture's status is `public-factory/check.sh`, while the file is
`check.sh`. Missing trust or script-path inputs cannot establish readiness.

```sh
target/release/soda-release journey check \
  --scenario /home/OPERATOR/scenario.json --out /home/OPERATOR/new-check
target/release/soda-release journey status --run /home/OPERATOR/new-check
```

The output must be fresh. Rust creates a private 0700 directory and durable,
read-only `scenario.json` and `request.json` bindings. The driver writes selected
`observation.json` and `readiness.json`; status verifies the retained scenario
and executable and reads the saved result without contacting the target. Use a
new check directory to refresh it. Keep the referenced executable unchanged.
The operator request is local invocation metadata; protected collector attempts
still use their existing separate request/executed/sealed receipt path.

A completed check exits zero even when `ready` is false; each check reports
`pass`, `blocked` or `unknown`. Invalid inputs/invocations exit nonzero. Selected
source/trust, inventory, custody, current connection generation, accepted inputs,
standing preparations, actor bindings and permitted-effect subsets are checked
without arming any effect request. Persistence/cold checks include a `closure`
diagnostic in both check and status. It lists selected versus retained Projects,
each resource's configured management/lifecycle scope and what must happen before
a snapshot. `scope_ready` means the known route and retirement scope cover the
resources; `snapshot_ready` also requires confirmed stopped state. Project
confirmation requires an inactive unit, zero main/control PIDs and a stopped
container. Missing observations stay unknown. A terminal record without a
confirmed actor is a reason to observe, never an ownership guess or an end permit.
Completed run counts and charged minutes are retained history, separate from
leases, unsettled accounting and native work. These requirements describe a final
state; they grant no authority or instruction to stop resources now. The current
Factory/browser route consumes the same shared inventory for actual retirement;
the diagnostic itself grants no lifecycle permission.
Model outcomes, unimplemented supplemental routes and unvalidated paid authority
stay unknown. A user hold remains a blocker. No check result admits execution or
qualifies a release.

## Maintained journey execution admission

`soda-release journey report --run RUN` prints a deterministic projection of saved
readiness, admission, execution/seal, selected phase, supplemental and retirement
receipts. It never contacts the target, finalizes evidence or dispatches work.
Missing results remain unknown. It checks receipt/file hashes and projects only
bounded metadata; credentials, private scratch and provider bodies are not read.
For a historical run whose retained driver predates reporting, supply
`--build /home/OPERATOR/current-journey-build/build.json`. This selects the exact
current report executable from the existing build receipt without changing the
old run or manually copying hashes. Root invocation still requires a trusted
admitted executable and protected run. The report does not qualify a release.

`journey retire-scratch --out COLLECTION` inspects fixed preserved scratch.
`--execute` additionally requires finalized matching successful evidence, a
terminal native worker, the original scratch identity and the frozen retirement
permission. Mounts, live process references, shared files and unknown resources
refuse removal. Source inputs, observations, credentials, caches and VM data are
outside its scope. Durable intent without outcome reports uncertainty; it does
not authorize another removal. Existing completed/failed attempts without a new
scratch-origin record gain no inferred cleanup authority.

The cold owner separately supports `journey cold retire --restored DIR
--result-sha256 SHA` (plan), with `--execute` for the exact verified stopped clone
and its three fresh copies only. Its source capture, source volumes/container,
settings, private environment and provenance stay retained. Normal collection
calls this only after completed final recovery and native stop, with the result
file's exact hash and `permitted_effects.retire_temporary_clone: true`; the common
false/omitted case records retention locally without rehashing the cold set.
Other container references and active mount dependencies block execution.
Allocation is an estimate; receipts separately record observed filesystem free
space. No image/volume/cache pruning or failed-attempt retry is implied.

`soda-release journey run --run RUN` admits the saved exact scope and calls the
existing protected collector. The [release workflow](../../docs/development/release.md#implementation-plan)
documents `ScopeReceipt`, scenario `collection` paths, generated build identities
and the fixed runtime input layout. Root dispatch requires a root-controlled
driver/run, private approval/build receipts and the existing admitted collector
authority, candidate, runtime and fixed credentials. Local user invocations can
record holds/missing authority without reaching that boundary.

`run-admission.json` records `held` separately from `approval_state`; rebuilding
the driver or holding execution does not revoke matching paid scope. Released
matching authority refreshes read-only readiness. Holds, revoked/missing authority
and other blockers write no resolved execution input and dispatch nothing.
Status includes this saved admission without target access. Eligible dispatch
derives the collector hash from `build.json` and phase flags from the shared
scenario, freezes resolved bytes through existing `collect`, and retains normal
worker custody/evidence sealing. It does not change budgets or enroll accounts.
The local boundary does not claim live qualification. Journey dispatch now
retains `collector-intent.json` and exact request/unit `collector-binding.json`.
All resolved collector invocations share a protected lock and immutable claims
under `/home/.soda-qualification-targets`, keyed by distribution/actual UUID;
uncertain prior claims fence new outputs until canonical native retirement is
confirmed. The existing collector request still precedes worker dispatch and
existing outputs only finalize. No timeout or vanished caller clears custody.

The maintained phases retain requests and returned bindings before continuing.
Uncertain Factory replies use the exact normal command lookup, CI/recreation
use their existing remote journals, and lifecycle actions read the same native
resource incarnation. Ready setup/Identity state is adopted through normal APIs;
preparation continuation keeps its exact IDs. See the release workflow for the
provider-free reproduction entrypoints and scope. Partial-setup admission,
supported recovery/retry and live API interruption checks remain J15–J16.

Check and status use the same derived report: `blockers` names missing or
unconfirmed requirements and resource IDs; `local_repairs` remains visible while
`next_action` reports a wait for a user hold or missing/unvalidated paid authority.
Those waits never request enrollment, funding, retry or a model launch. The
`automatic_actions` fields remain false. `last_confirmed_phase` names only the
confirmed read-only observation; `qualification_phase` and future model, head,
PR, publication, CI/review/delivery and selected measurements remain unknown
until their actual canonical receipts are available.

`accounting` separates requested limits from observed sponsorship and rolling
limits, repository charges, held reservations and settled rolling usage. Missing
observations remain null. Rolling microseconds stay decimal strings to preserve
int64 precision. With no held reservations, total rolling charge equals the
settled observation; otherwise it remains unknown because canonical Store
admission owns the conservative held charge. Charged minutes are model-session
accounting, not provider token billing; neither an observed balance nor a saved
approval path admits a new trial. Status reuses the saved observation time and
does not silently refresh balances or alter allowances.

The J08 readiness/permit milestone is recorded in the release task list. Its
retained-request regression used a fresh offline native Chromium context, real
browser Request objects and local fulfillment of every response: the exact
saved action passed once, while changed identities and replay were refused.
No historical action was sent to an appliance. This client-filter result does
not establish current backend context eligibility, retire any resource or
resume a trial. Phase/permit source migration remains the next implementation
step; historical binaries and request evidence stay intact.

## Produce a development candidate

Replace the paths below with the admitted executable, restricted configuration
and a fresh output below `.artifacts/releases/`:

```sh
sudo /ADMITTED/soda-build --worker-config /RESTRICTED/worker.json \
  --arch x86_64 --out /OUTPUT_PARENT/candidate-attempt --development --target candidate \
  --forgejo-source /CANONICAL/FORGEJO-CHECKOUT
```

For media, use the same `soda-build` invocation with `--target media` and
`--rootfs-base-url http://FIXTURE_ADDRESS:PORT`; optional
`--media-compression fast` applies only to development media.

The `soda-candidate` wrapper offers prompts and `candidate`/`media` modes,
including the required `--forgejo-source` input: it suggests the sibling
`forgejo-ext` checkout when one is present, asks otherwise, and forwards the
answer to the controller. Flags only pre-seed answers. See the
[candidate operator guide](../../docs/guides/soda-candidate.md) and native support
guide for rootfs pickup and fixture authority. Builds write artifacts and logs
and execute native tools. Development output is never a qualified release.
Output leaf names use lowercase letters, digits or dashes and must be fresh.

## Check or inspect artifacts

```sh
soda-candidate-check --candidate /CANDIDATE/artifacts --arch x86_64 \
  --soda-revision FULL_SODA_COMMIT --forgejo-revision FULL_FOUNTAIN_COMMIT
soda-artifacts inspect-oci --source /CANDIDATE/artifacts/image.oci.tar \
  --arch x86_64 --revision FULL_SODA_COMMIT
soda-artifacts check-fountain-pin
```

These commands read and validate existing artifacts. Use the actual archive
path and full source revisions from that candidate. They do not qualify a host.
`check-fountain-pin` answers whether a published Fountain release serves the
pinned `fountain-source.json` revision before any build input freezes; run it
from the checkout first and pass the reported URL as `SODA_FOUNTAIN_RELEASE_URL`.

## Fetch and convert inputs

```sh
soda-artifacts fetch-coreos --arch x86_64 \
  --keyring /TRUSTED/fedora.gpg --signer TRUSTED_FULL_FINGERPRINT \
  --out /CACHE/NEW_ATTEMPT
soda-artifacts convert-butane --arch x86_64 \
  --source /PRIVATE/instance.bu --out /PRIVATE/instance.ign
```

`fetch-coreos-iso` accepts the same fetch flags for the upstream ISO. Fetches
download, verify hashes/signatures and retain fresh outputs. Select the keyring
and signer independently. Conversion requires installed native Butane and a real
private output parent; it writes a fresh private Ignition file. See
[verified CoreOS and provisioning](../../docs/development/native-support.md#verified-coreos-and-private-provisioning)
for prerequisites. These tools do not install, activate or publish Soda.
