# Soda acceptance support

Run bounded native checks and retain evidence tied to a source revision and
target. This package is for operators and release developers collecting
observations from an explicitly selected builder, host or fresh VM.

## Commands

| Binary / action | Use |
| --- | --- |
| `soda-acceptance exec` | Run an existing check locally or through pinned SSH. |
| `soda-acceptance native` | Run an exact-source remote build phase with owner `P02`, `--remote` and `--request`. |
| `soda-acceptance vm` | Boot a fresh KVM fixture with owner `P03` and `--config`; optional `--hold` and `--restart` control its lifecycle. |
| `soda-acceptance probe-ssh` | Observe a pinned Git SSH endpoint with owner `P11` and `--remote`. |
| `soda-acceptance report` | Combine finalized observations into a Markdown handoff. |
| `soda-acceptance-remote` | Target-side dispatcher for `native-phase REQUEST`, `cockpit-account` and `project-state`. |
| `soda-host-probes` | Helpers used by the installed host/operator shell checks. |
| `factory-fountain-host-check` | Fresh local Fountain + installed Soda package + separate Factory, real TLS login/admitted reads/session withdrawal and clean shutdown. |
| `factory-credential-operation-check` | Isolated real Linux Secret Service observation of asynchronous native borrowing, concurrent withdrawal/replacement, cancellation, deadlines and lease expiry. Requires its private keyring driver; never selects the personal login keyring. |

The installed host check reads Payload v4, verifies six acquired image config
IDs and installed runtime files, and compares the required Soda package with its
verified cached archive and Fountain receipt. It does not require optional
Tailnet to be downloaded. `SODA_EXPECT_TAILNET` defaults to `disabled`; select
`enabled` only for a fixture whose administrator enabled it. Tailnet advertisement
checks require a running, enrolled host node and use its container's LocalAPI.
For observer-only checks against a retained development source, `SODA_HOST_PROBES`
can name a separately staged reviewed probe. Record that probe's revision/hash
separately; it does not change or qualify the source's shipping bytes.

From the repository root, inspect the available flags:

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- --help
cargo run -p soda-acceptance --bin soda-acceptance -- exec --help
```

## Collect an observation

### Isolated native credential operations

Build `factory-credential-operation-check` from this acceptance package. Its
single argument is a fresh `/home` profile inside an owned private fixture root.
It refuses to run as root, against an existing profile, or unless
`DBUS_SESSION_BUS_ADDRESS`, `XDG_DATA_HOME` and `GNOME_KEYRING_CONTROL` select that
root's private bus/data/control paths. A driver must start a private D-Bus and
GNOME Keyring daemon, unlock with a generated nonempty password over stdin, run
the observer and stop/await those owned services. It must never select the
personal bus/keyring or put the unlock password in arguments or a file.

The observer uses generated fixture token bytes in the actual encrypted keyring
and barrier-controlled native asynchronous callbacks. Actual OS withdrawal and
independent-owner replacement occur while those callbacks wait; their results
must be refused. It also checks caller cancellation, owner retirement, the real
eight-second operation budget, the unchanged full thirty-second native lease,
fresh recovery and absence of plaintext fixture tokens in the saved keyring.
It cleans up its own items. These are native credential-operation checks; no
provider HTTP, OAuth grant, account binding, Go transport or webview is qualified.

The same binary's `--github-transport PRODUCTION_BACKEND OBSERVED_BACKEND
PG_PAYLOAD NEW_PROFILE` mode qualifies the native read transport instead. It
first runs the production native executable with real owned PostgreSQL and
keyring credentials, checking actual GitHub refusal and unbound Factory state.
It restarts the retained profile with a diagnostic composition of the same
production runtime/handler/SDK. That transport observes headers and pauses
before forwarding real HTTPS, allowing actual OS rotation/withdrawal and caller
cancellation to be observed. It never replaces GitHub replies. The mode proves
that cancellation closes Unix HTTP/cancels Go promptly, invalid scopes never
reach GitHub, stale results refuse and backend/PG retirement completes. Tokens
are generated invalid fixtures: successful OAuth, private reads, canonical
account binding and desktop UI remain unqualified. Use only its isolated keyring
driver; no personal account or plugin credentials are selected.

Use `--github-verify PRODUCTION_BACKEND OBSERVED_BACKEND PG_PAYLOAD NEW_PROFILE`
with the same private keyring driver for the initial-token path alone. It never
creates a guessed account or saves a token. The production executable receives
actual GitHub 401 for a generated invalid token; the diagnostic composition
observes the same real SDK request and proves cancellation/runtime retirement.
OS account enumeration stays empty throughout and Factory stays unbound. This
mode does not qualify successful OAuth identity verification or credential
storage; those require a registered public client and real human consent.

`--github-issues BACKEND PG_PAYLOAD NEW_PROFILE` observes only the new issue
selectors. The production executable uses a generated invalid fixture credential
in its isolated keyring. Actual GitHub rejects list/detail before issue data is
read; malformed selectors refuse locally, Factory stays unbound, and the fixture
item and native processes are cleaned up. This proves authenticated refusal and
native wiring, not successful private issue access or canonical/UI integration.

`--github-discussion BACKEND PG_PAYLOAD NEW_PROFILE` checks the issue/PR ordinary
conversation selectors through the production native executable and isolated
real keyring/PG. Generated invalid credentials receive real GitHub 401 without
comment data; invalid selectors refuse, Factory stays unbound and fixture state
is cleaned up. Successful public provider reads are a separate component check;
neither observation qualifies authenticated private access or the connected UI.

`--github-pulls BACKEND PG_PAYLOAD NEW_PROFILE` checks PR list/detail selectors
through the production native executable with isolated real keyring/PG and
invalid fixture tokens. It reuses the existing record-selector observation
owner: actual GitHub authentication refusal, local invalid-query refusal,
unbound Factory state and fixture cleanup. Public list/detail semantics have a
separate real-provider component observation. Successful authenticated/private
PR reads, canonical binding and the desktop journey remain unqualified.

`--github-diff BACKEND PG_PAYLOAD NEW_PROFILE` observes native diff discovery
authentication refusal with a generated invalid token in isolated real OS
custody/PG. Malformed/half commit pairs and invalid scope selectors refuse
locally; Factory stays unbound and fixture resources are cleaned up. Public
merge-base discovery and requested-pair qualification are separate observations;
the backlog records their actual completed and pending coverage.

Use `--github-diff-pins BACKEND PG_PAYLOAD NEW_PROFILE SAVED_DISCOVERY_JSON`
to check the non-empty native descriptor using a previously observed real
merge-base/head pair. It sends that pair through production native IPC and
requires actual GitHub authentication refusal for an invalid fixture credential.
It skips the previously qualified malformed-descriptor checks. Successful
public requested-pair/mismatch reads remain a separate provider observation.

### Isolated Fountain/Factory host admission

Build `factory-fountain-host-check` with the existing acceptance package. Run
it as an ordinary Linux user inside the existing `soda-native-pg-check --once`
owner, using a freshly created PostgreSQL profile and a separate fresh host
profile, both under `/home`:

```sh
cargo build --locked -p soda-acceptance --bin factory-fountain-host-check
target/debug/soda-native-pg-check --once /home/OPERATOR/SELECTED_PG_PAYLOAD \
  /home/OPERATOR/NEW_PG_PROFILE -- \
  target/debug/factory-fountain-host-check \
  /home/OPERATOR/CURRENT_STAGE /home/OPERATOR/FOUNTAIN_SOURCE \
  /home/OPERATOR/NEW_HOST_PROFILE
```

The selected stage contains the matching native `fountain`,
`bin/soda-extension`, `bin/soda-factory-server` and `package/assets/` plus
`package/extension.json`. Build the host from the matching sibling source with
the native SQLite tags; build the two Soda commands normally (the appliance
Factory composition, without `factory_native`). The existing
`scripts/build-soda-extension.ts --package=soda --out=NEW_STAGE/package
--terminal-assets=PREPARED_TERMINAL_ASSETS` builds the current assets; copy the
current `system/containers/extension/extension.json` beside them. The checker
stages the backend and its profile-specific launcher, then uses Fountain's real
stopped-host package CLI to install and enable it. Host static files come from
the explicitly selected Fountain source. No container or retained deployment
is selected. `curl`, `openssl` and `id` must be available.

The checker owns a fresh native SQLite Fountain store, generated local TLS
certificate and local human account, and starts the real Factory command against
the managed PostgreSQL ledger. Its native package page and API proxy use the
normal session generation and pinned Unix callback. The absent Forge socket
is deliberate: Factory requests must reach the separate Factory service.
It verifies anonymous denial, real login, current identity, an observed empty
repository catalogue, stale-generation conflict and replay of the retired
cookie after native logout. It then stops both servers before the PostgreSQL
owner stops its child. Failure retains the private profiles and logs.

Add `--issues` immediately after `factory-fountain-host-check` to qualify the
populated repository/issue journey on that same real host. It creates two
private fixture repositories and native issues/comments using the ordinary
local provider REST API with the generated fixture administrator's credentials.
Those credentials stay in restricted files and are never used for Factory
reads. Factory reads use the normal native cookie session, installed package
proxy and private callback. A second ordinary account logs in separately to
prove private denial, read-only collaboration, rename stability and immediate
withdrawal after the real collaboration grant is deleted. The journey also
checks issue filters/paging, same-number repository isolation, complete Unicode
bodies, assignees/labels, comment continuations, capabilities and strict query
refusal. It creates no PAT, enrollment, model call or external repository.
Its summary is recorded in `result.json.issues`; private HTTP captures retain
the exact provider and Factory responses for failures.

Use `--pull-code` instead of `--issues` for the populated PR/code-review
journey. Ordinary native REST writes create a private repository with actual
Git commits, open/draft/closed PRs, a binary file, full reviews and inline
comments/replies, a private pending review and ordinary PR discussion. Factory
reads still use native human cookies through the installed extension and
separate Factory. One-item pages qualify metadata, immutable file pairs,
reviews, threads and replies. A separate read-only collaborator logs in to
check draft privacy, distinct author/reviewer capabilities and actual non-author
approval. Replacing a reviewed line through the native file API must advance
the hidden PR head, reject the old file continuation and retain the original
review anchor while reporting the thread outdated. Deleting the collaboration
grant must withdraw all PR surfaces in that still-live session. The summary is
`result.json.pull_code`. Status/protection/Actions-job qualification remains a
separate journey; a passing code-review mode does not establish those semantics.

Use `--pull-checks` for installed status/protection/Actions qualification. Real
PR creation triggers a local workflow; explicit native dispatch creates a second
run at the same head. No runner is enrolled. Four actual waiting jobs are paged,
then two are cancelled through the ordinary native UI route. Their richer job
facts remain separate from the commit statuses Actions also publishes. Native
status/protection writes qualify required globs, missing/pending/failure/success,
latest-context identity, stale continuations and independent approval/merge
gates. A second human checks private denial, read collaboration, separate Actions
unit withdrawal and access revocation. Actual writer permission plus an official
approval admits a normal local native merge, read back as merged. Controlled
fault injection removes only the owned fixture's exact hidden PR head with Git
compare-and-swap, checks unavailable status/job facts, and restores that ref
before merging. Its public summary is `result.json.pull_checks`.

Use `--read-edges` for the remaining provider qualification cases. Ordinary local
REST/Git writes create two private repositories with the same PR number and a
six-file diff: rename, deletion, modification, binary, a literal pathspec-like
Unicode filename and a 7,000-line Unicode file. The installed API must preserve
repository/PR identity, complete file paging, exact counts, explicit binary
absence and a truncated UTF-8 patch. It rejects another repository's file
cursor, retains the exact diff through a native repository/Git-path rename,
withdraws PR reads when their native unit is disabled and denies archived
mutation capabilities while retaining readable files. The public summary is
`result.json.read_edges`. These four modes together provide the populated local
provider evidence; each mode runs in its own fresh owned host/PG profiles.

The fixture REST writer waits briefly only for Fountain's explicit ordinary
writer reservation-fenced refusal, which its native owner emits before effects.
It retains each attempt privately. Other HTTP failures and uncertain transport
results stop immediately; they are never automatically replayed. REST status
IDs are repository-local indexes; collaboration status identities retain the
native global record ID.

Profiles are mode 0700; generated passwords, HTTP captures/cookies and native CLI
output remain restricted. The public `result.json` fields contain only outcomes,
fixture counts and the local origin. Plain host admission does not qualify
populated collaboration semantics; the selected mode adds its own journey.
These modes do not qualify browser rendering, a distribution release, desktop
Fountain enrollment or a retained appliance upgrade. Select fresh upstream
inputs before claiming a fresh release baseline; cached component checks are
labelled at their actual scope in the implementation backlog.

### Selected local or remote observation

Execution actions require an admitted owner, full commit revision, `x86_64`
architecture, non-secret target name and a fresh private evidence directory.
For example, with an already prepared pinned SSH connection file:

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- exec \
  --owner P06 --revision FULL_40_CHARACTER_COMMIT_SHA --arch x86_64 \
  --target ACTUAL_GUEST_HOSTNAME \
  --remote /home/soda-builder/private/guest.json \
  --evidence /home/soda-builder/private/new-host-observation \
  -- env SODA_NATIVE_VALIDATE=ACTUAL_GUEST_HOSTNAME bash -s < tests/installed/host.sh
```

Replace placeholders with the selected revision and actual target. The remote
file references the SSH user, host, port, private key file and trusted
`known_hosts` file. Owned process execution requires Linux. VM execution also
requires native x86_64 Linux, KVM, matching firmware and verified boot inputs.

Add `--secret-file FILE` for known private values and `--artifact-file FILE` for
public artifact hashes. `--timeout` accepts positive durations up to 24 hours
and defaults to 30 minutes. Output includes captures and a finalized
`observation.json` when evidence finalization succeeds; execution, evidence and
cleanup outcomes are recorded separately. Failed work and VM disks are retained.

## Create a handoff

```sh
cargo run -p soda-acceptance --bin soda-acceptance -- report \
  --arch x86_64 --revision FULL_40_CHARACTER_COMMIT_SHA \
  --record /home/soda-builder/private/host-run/observation.json \
  --out /home/soda-builder/private/new-support-evidence.md
```

Repeat `--record` to include more observations. The report checks revision,
architecture and retained file hashes, and preserves failed or missing scopes.

The [native support guide](../../docs/development/native-support.md) owns request
formats, owner labels, trust, VM lifecycle and evidence rules. Use the
[testing guide](../../docs/development/testing.md) for the product acceptance
journeys invoked by these support tools. Builds, remote commands and VM actions
have real effects; select an authorized target and action before running them.
