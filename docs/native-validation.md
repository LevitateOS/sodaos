# Native validation

**Historical bounded U08 native x86_64 proof is accepted; new UI and final product
acceptance are pending.** The [handoff](implementation-status.md) records exact
`8b823db` build/rollout/regressions and scoped earlier lifecycle/fresh-project proof.
The four retained environments and infra route are not a fresh appliance install
or independent aarch64 result. No old browser harness validates current API-only source.

This is the product validation guide, not another roadmap. Features own their
focused tests; extend existing product entrypoints rather than copying scenarios
into the [outside support tools](native-support.md). Those tools have separate
[remaining checks](native-support.md#remaining-validation), not a second readiness
gate. Optional media/helper completion is not required to use authorized tools.

Name the exact revision/artifacts, builder, target, architecture, developer client,
actions and cleanup limits before execution. Rehearse preserved-state changes before
separately approved cutover. A build permit is not install/restart, disk erasure,
provider registration, routing or fixture permission. Record reused evidence by
exact bytes/target, not a second independent PASS. Missing access is **unverified**.

## Project OS baseline checks

The [Project OS baseline](project-os.md) is a source-backed contract, not another
readiness gate. Extend existing image/tool, account/access, project-state/workload
and terminal tests for the actual candidate. Required commands/terminfo, genuine
login/groups/shared paths and managed-service cgroup ownership need native checks;
package metadata or disabled inner workload cgroups alone prove none of those.

For existing-root additions, declare the exact native maintenance transaction and
fresh backup/interruption scope; verify the original CID, accounts/keys, configuration,
homes/shared tools and workload data survive. Prove the selected tmux session's same
process and owned cleanup independently from its socket. Historical request-owned
terminal probe results do not accept managed reattachment/navigation retention.
Current authored probes and UI ports require new, scoped execution; do not relabel
old results. See the [terminal proof requirements](terminal-integration.md#selected-persistence-mechanism--tmux-source-candidate).
No project/package/service mutation is authorized by these requirements.

For selected profiles, extend these same product-owned journeys rather than creating
a parallel guest/readiness suite. Record distribution, exact image/package inputs,
interface and native architecture. Verify original account/HOME/groups, shared mise
paths and installed bytes from shell, SSH and GUI-launched tools; edits through an
editor and terminal must reach the same checkout. Check per-user display denial,
project-admin trust, actual private service access, app-state persistence and safe
input focus while navigating Forgejo and moving between page/drawer. Explicit desktop
end, logout/access expiry, last-viewer retention and project Stop need their declared
effects checked independently of terminal End and AI Cancel.

For every new creation, compare the persisted profile ID, distribution ID/version,
interface/session type, architecture, exact image ID/digest and recipe revision with
the native project's labels and `/etc/os-release`. An unavailable profile must fail
before reservation/native creation. A legacy environment with missing metadata must
remain legacy/unknown and keep its original CID/root; changing the configured image
default or mutable tag must not change its displayed identity. Exercise the current
single-image build/install callers while introducing the matrix: each exact profile
artifact must be native to its architecture, and dashboard/application artifacts must
not acquire the selected project distribution accidentally.

### KDE desktop profile checks

Use an ordinary newly joined, password-locked member; do not enable a reusable Linux
password, password SSH, automatic display-manager login or an operator repair path.
An explicit Start must produce one exact desktop ID and a login-quality session for
that original account. Observe UID/GID/groups, `HOME`, `/run/user/UID`, logind/PAM or
the declared equivalent, the user service manager, session D-Bus, Wayland socket,
portals/PipeWire, shared mise settings and the owned graphical cgroup. Starting with
`podman exec --user` and a compositor process is not sufficient evidence.

On Fedora, prove the supported Plasma Wayland session; do not install or select a
Plasma X11 session to make an X11-only VNC/RDP server pass. Before accepting the
KRFB/private-RFB/WebSocket/noVNC investigation candidate, inspect the exact packaged
versions and prove a headless software-rendered session, no public listener,
short-lived private transport authentication, authorized browser attachment, resize,
keyboard/pointer, Unicode, explicit clipboard copy in each direction and clean refusal
when the native server asks for interactive approval or a standing password. Compare
KRDP only if its browser gateway, headless behavior and credential boundary are tested
as one complete path. An upstream limitation or failed first probe is a concrete
finding to analyze against the existing container; it is not automatic VM selection.

The finite first-slice matrix uses Alice and Bob in the same project:

1. Start one desktop for each account and confirm separate runtime directories,
   Wayland/display endpoints, buses, app/browser profiles, credential stores and
   process scopes. Alice cannot discover, view or control Bob's desktop through Soda.
   Project root's documented native visibility is not a browser observation feature.
2. In Alice's creating Soda sign-in context, open the same desktop in two views. Both
   may observe, but only one explicit input lease can inject keyboard/pointer events;
   transfer control without overlap or implicit eviction. Another Soda sign-in context,
   including one for Alice, cannot adopt, replace or renew the live desktop.
3. Move Alice's original attachment between Spaces and the drawer, Hide and reconnect
   to the exact desktop ID; no new compositor, desktop, home or app process appears.
   After loss of the last visible authorized viewer, reattach within the 30-minute
   deadline to the same running app and unsaved editor state and confirm that successful
   visible reattachment clears that grace deadline. Output, retry and hidden views do
   not clear or renew grace; duplicate views do not extend the 12-hour hard cap or the
   creating context's authority, and neither cap exceeds that Soda sign-in context's
   configured expiry. Use bounded test values where supported; do not wait out
   production durations merely to create evidence.
4. Exercise Desktop Lock and the selected authorized unlock design from detach and
   reconnect. It must neither trap the password-locked account nor accept a reusable
   Soda/Forgejo credential as a general PAM password. Desktop power actions cannot
   stop/suspend the whole project without the existing explicit project operation.
5. On native safety-lease loss, explicit Soda logout, context rotation/expiry or
   confirmed authority loss, verify immediate transport denial and bounded cleanup of
   the graphical scope. The user service manager, managed tmux, ordinary SSH, project
   services and nested workloads remain. A cleanup timeout stays uncertain/unavailable
   and cannot be replaced under the same ID.
6. End Alice's desktop and independently prove its graphical cgroup, session-owned
   D-Bus/systemd-activated GUI apps and private transport state are gone while Alice's
   terminal/SSH/shared user infrastructure and Bob's desktop continue.
   Then Stop/Start the project and verify durable homes, app/browser settings,
   per-user credential-store files, checkouts, shared files/tools and service data
   remain while old desktop processes/IDs do not.
7. In a GUI editor and terminal, edit the same dirty checkout seen over SSH; use the
   packaged file manager, basic editor and browser without manual dependency repair.
   Exercise provider sign-in only with separately supplied private fixture inputs and
   record no cookie, token, clipboard content or screen containing secrets.

The first desktop acceptance excludes audio, camera/microphone, USB/device access,
GPU acceleration, printing, multiple monitors, cross-user sharing/control and GUI
file transfer. Do not make those unselected peripherals prerequisites. Mark OpenAI
and Claude Computer Use unavailable on Linux according to current vendor docs; a
generic screen-driving test, API computer-use agent or successful browser automation
is not evidence that either Linux desktop app supplies its native Computer Use feature.

Existing terminal/native results remain valid only for their recorded mechanisms
and targets. New Fedora/KDE claims require affected native proof. Test advertised
GUI applications separately from native computer use, marking unavailable provider
capabilities honestly. For AI runs, prove no-browser execution, dedicated credentials,
exact viewer attachment and resolver continuation without modifying personal work.
GNOME and unselected runtime alternatives do not become validation prerequisites.

The [batteries-included contract](project-os.md#batteries-included-by-default) requires
a fresh-profile journey without manual prerequisite repair. Verify ordinary file/
archive/search/transfer operations, native build/link/debug tooling, shared runtime
selection and a native service. KDE adds its default editor/file-manager/browser,
fonts, explicit clipboard transfer and native credential storage. Separate intentional
repository dependency installation or personal sign-in from missing platform packages.
Record repairs as candidate defects, incorporate them into packaging and rerun the
affected journey before claiming readiness. Do not treat a package list as runtime proof.

## Native source and build evidence

Follow [installation](installation.md) on the matching native x86_64 builder. Resolve/review the real Go dependency metadata, build with `scripts/build-native.sh x86_64`, then explicitly run `scripts/check-native.sh x86_64`. The latter runs the authored Go, TypeScript/UI and native staging checks; it does not enroll, install or restart services. Dependency/compiler/test failures belong in their source, not suppressed flags.

Record actual source revision, native OS/architecture/tool versions, commands, output and defects in an ordinary operator log or issue. Do not manufacture PASS lines or an acceptance schema. No CI workflow runs automatically.

## Local workspace checks and installed-journey source ports

The step-5 workspace and current journey controls have local coverage, not current
installed acceptance. `bun run test:pages` (`test:spaces-page` remains an alias)
runs `scripts/test-spaces-page.ts` and `TestNativeConnectionFixture` against the
authorized stock Forgejo at `http://localhost:3300`. Forgejo supplies native HTML
and authentication; a fresh retained Go backend/OAuth fixture serves the canonical
candidate assets. The parent exercises real consent, session reuse, native profile
draft/history and logout. All Spaces, runner and repository consumers run inside
that parent; their operation APIs/socket peers are synthetic, not installed
project/runner/provider operations. Missing prerequisites or a failed consumer
fail this command rather than becoming a skip. `bun run test:layout` exercises
20 width/theme/running-state combinations with real emitted Lit/xterm, a native-form
fixture and synthetic peers. Both are in `bun run test`, which prepares browser assets
once and also enables local Lit runtime/settings-link checks; neither contacts a
retained appliance. The [source-check aggregate](typescript.md#local-source-checks)
shares these suites with `check-native.sh` without requiring its sealed stage for
ordinary iteration. Native artifact/revision and packaging gates remain separate. Pointer/keyboard panes, sidebar/tab overflow, compact visibility, form
selection, beforeunload cancellation and stable owners have focused browser coverage.
Physical keyboards, actual Forgejo menus/forms/diff/comment/clipboard behavior and
native process/CLI continuity still need the applicable installed scope.

### Mandatory page gate on a final native builder

The page gate is part of `bun run test` → `bun run check:source` →
`scripts/check-native.sh ARCH`; Mac browser receipts do not replace it. Before
running the final gate in its **clean frozen candidate worktree**, provide:

- The documented authorized stock Forgejo fixture at **localhost:3300**, with the
  candidate's template overrides loaded and the existing non-admin
  **soda-screenshot** account. Use the supported template/public mounts; no
  fabricated HTML or alternative authentication harness. The existing consumers
  also require an accessible public fixture repository with **ID 1**; inspect first,
  never repurpose an unrelated repository to meet that assumption. Fresh fixture
  initialization must explicitly cover that repository. Repository owners read the
  existing fixture's synthetic creation profile; this is not native helper proof.
- That fixture's own restricted `.local/screenshot-fixture/create-output.txt` in
  the worktree. Both Go and browser consumers use it through their existing
  private-input code. Ignored files do not arrive with Git. A different fixture's
  password or borrowed CLI credential is not a substitute. The saved screenshot
  browser profile is for capture; it is not the page harness's authentication input.
- Root Bun dependencies, Playwright's Chromium and the **Google Chrome channel**
  used by the actual BFCache parent. Its loopback TLS certificate pin is scoped to
  its test browser; no global trust changes are needed.

Inspect/preserve any existing fixture first. If none exists or its service/account/
browser installation needs actions outside the recorded grant, obtain that precise
fixture/setup authorization before proceeding. Do not silently reinitialize data.
`bun run test:pages` is the supported focused prerequisite check and must run **all**
consumers; it creates fresh retained fixture evidence and a fixture OAuth app.
Once setup is ready, build/seal the exact candidate using pinned Go 1.26.7/Bun 1.4.2
on matching-native Linux, then run `scripts/check-native.sh ARCH` in that same
worktree. The full gate repeats its page phase normally. Export only after the
applicable checks pass. Do not transplant an earlier revision's seal/check receipt
or omit the page phase because another computer passed it.

`tests/installed/sodaspaces-controls.ts` follows actual project views, New chooser
and per-terminal menus; it does not authorize requests or substitute API writes.
The runner retains its target/actor/path/body/private-input/replay guards. Its latest
source ports were locally exercised, not rerun on retained projects. Old terminal
permission does not authorize six sessions, shared Stop, fault injection or CLI use.

## Read-only Sodaspaces browser probe

`tests/installed/sodaspaces.ts` is opt-in. The real isolated local journey passed;
exact revisions, failures and scope are in the [handoff](implementation-status.md).
The exported-payload run at `ee8091a` also passed step 3's bounded x86_64 delivery
checks. This is not installed appliance, project-runtime or release acceptance.
It uses stock 15.0.7, the candidate's served CSS/JS, real native forms and OAuth,
then read-only drawer states. It never seeds cookies/sessions, substitutes responses,
creates repositories/environments, joins or installs keys. Protective request
interception on the two exercised pages aborts unapproved origins/writes and makes
the run fail, not pass. Chromium CDP Fetch pauses each redirect hop before sending
it; Playwright route callbacks do not include those hops. The exact stock logout
`POST /-/fetch-redirect` form containing only `redirect=/` is navigation, not an
environment write, and is allowed. No request/response is fulfilled or replaced.

Execution needs explicit target and authentication-transition permission. Supply
an existing approved **public repository with Issues enabled and a plain new-issue
form**, two existing password-login fixture users who can view it, automatic native
theme selection, and the existing Soda OAuth client ID. MFA/captcha/insufficient
consent are not bypassed. The probe changes native browser sessions and Soda login/
logout/grants, which can affect upstream refresh counters; it does not revoke grants
or edit callbacks/secrets. It types, preserves, then discards only its own synthetic
unsaved issue title without submitting it. An existing-environment view requires
approved helper read scope; an absent reservation suffices for initial OAuth proof.

Use a fresh restricted browser home with the selected CA already trusted by
Chromium. Keep `HOME/sodaspaces-run/cdp.sock` within 103 bytes. All input/password/CA files are absolute regular mode-0600 files; the
home is mode 0700. No TLS bypass or sandbox disabling is selected. Reuse prepared
pinned Playwright/Chromium and the root workspace's pinned WebSocket dependency; the
probe does not download browsers. Anonymous raw-path/cache checks require existing
`curl`: they disable curl configuration/proxies, verify the supplied CA and hostname,
retain raw paths, refuse redirects and bound headers/body/time. This avoids Bun's
observed constrained-CA verifier limitation without bypassing TLS or changing trust.
`native-browser.ts` launches stock sandboxed
Chromium and attaches through a private Unix socket/pipe with `noDefaults`, not a
TCP debugger port. This avoids Playwright's always-focused/visible override and
BFCache-disabling launch flag; it does not synthesize visibility or restore events. Private request:

```json
{
  "origin": "https://approved-fixture.example",
  "target": "approved-fixture",
  "revision": "FULL_40_CHARACTER_CANDIDATE_REVISION",
  "repository_path": "/alice/approved-repository",
  "repository_id": "42",
  "oauth_client_id": "EXISTING_PUBLIC_CLIENT_ID",
  "ca_file": "/private/fixture-ca.pem",
  "users": [
    {"id": "1", "login": "alice", "password_file": "/private/alice-password"},
    {"id": "2", "login": "bob", "password_file": "/private/bob-password"}
  ]
}
```

Later authorized invocation (not permission):

```sh
SODA_NATIVE_VALIDATE=approved-fixture bun tests/installed/sodaspaces.ts \
  /private/request.json /private/browser-home --allow-auth-transitions
```

The source checkout must be clean and match the declared revision. A fresh
`sodaspaces-run/` below the home retains the private profile and exclusive sanitized
`result.json`, including failures. Existing run directories are refused untouched.
No screenshots, traces, raw callback URLs, provider bodies or credential dumps are
recorded. The caller must separately bind the installed backend/config/artifacts;
served asset hashes alone do not prove its binary revision or installed migration.

Coverage includes raw proxy alias/encoding denials, native version/asset routes,
conditional asset revalidation, actual Soda cookie attributes, anonymous/native-only
cookies, two OAuth returns, actor/CSRF logout denials, native-only account switching,
Soda-only logout, stale tabs, native unsaved form coexistence, keyboard/focus/backdrop,
narrow/wide automatic themes and actual back-forward restoration. The native launcher leaves BFCache enabled and omits Playwright's focus emulation. If a real
BFCache restoration does not occur, the probe records incomplete scope and exits 2,
not a synthetic pass; failure exits 1. Exit 0 is only this scoped journey. Current
asset revalidation is not an update rehearsal; native provisioning/SSH, final product,
backend artifact binding, cutover and independent aarch64 acceptance remain separate.

## Explicit Sodaspaces access mode

Append `--allow-environment-access` to the existing browser invocation and add a
`public_key_file` (absolute mode-0600 public Ed25519 key file) to each declared user.
This mode requires the read-only journey and real BFCache proof first, then refuses
an already reserved repository. It performs a real nonowner create denial, one owner
create, each user's separate public-key save/join and own connection observation.
CDP admits only one exact actor/path/body-bound write per explicit test action; extra,
wrong-target or repeated requests fail before transmission. No private SSH key goes
to the browser/backend. Native Copy must produce native success feedback and paste
its command through the real clipboard into the run's own unsaved issue title; that
form is never submitted. A failed provisioning attempt is retained, not replayed on
a new profile. This is browser/helper confirmation, not SSH proof by itself.

`tests/installed/developer-access.py /private/client-run` then consumes the passed
browser result, an independently operator-verified public project host key and
client-local private keys. Its private `target.json` has `target`, `revision`,
`project_id`, `subnet`, `browser_result`, `host_key_file` and ordered `users` matching
the browser result. Each user declares `id`, `login`, `key_file`, `administrator`
(first user true, second false). An optional restricted `ssh_config_file` selects
an explicitly approved client transport (for example management-SSH forwarding).
That mode records no direct-route proof; original-key pinning, forced public-key
authentication and separate connections remain mandatory for SSH/SCP/SFTP/denials. `SODA_NATIVE_VALIDATE` must match `target`. All input
files are absolute restricted regular files; no host-key scan or trust bypass occurs.
It records actual client hostname/architecture and script hash, tests project-IP SSH,
PTY, SCP/SFTP, owner/nonowner sudo and cross-user key denial, and retains its own new
home probe directories/results. Historical fixed-name U08 invocation remains in Git;
no old fixture is silently retargeted. Record the actual client namespace and route:
a fixture-local client does not establish builder/laptop or Tailnet reachability.

Neither authored entrypoint is native evidence until its exact run passes. The
handoff now records passed bounded phase-5 native access and separately approved
phase-6 preserved-state cutover; those records do not authorize replay or other targets.

For an existing private repository, use `--private-repository` after
`--allow-auth-transitions`, with the ordinary read-only input (no public-key files).
It cannot combine with access mode. Both declared users must actually have native
repository access. The anonymous stage requires stock Forgejo's 404, no visible Soda
button and no environment reads; the remaining native login/OAuth/stale/BFCache/form
journey is unchanged. Do not make retained repositories public to fit a probe.
When an own connection is displayed, the probe records its usable SSH command and
public fingerprint, bound to the declared actor/repository, for independent operator/
client comparison. These public observations contain no cookie, grant or private key.

Native Forgejo request logging uses the supported empty `LOGGER_ROUTER_MODE` and
console access logger with method, `URL.EscapedPath` and status only. The default
router logger includes raw query strings, including OAuth state. Do not enable that
logger or add queries, referers, cookies, authorization headers or bodies to evidence.
General upstream service/error logging remains native. Native configuration and
query-free OAuth request observations are recorded in the phase-5 handoff; old private
journals/evidence are retained, not cleared or described as never having logged state.

## Minimum management controls — native proof still required

The helper/API/independent drawer source adds explicit Start/Stop and own-key preview/
apply/removal. Local Go, temporary-filesystem Python and DOM doubles are not proof of
those actions on an appliance. Before execution declare the exact fixture/project,
users, helper/API binaries, boot-enablement changes, temporary SSH public-key inputs
and retained run-owned files; obtain any missing lifecycle/access-mutation scope.
No previous terminal-only approval is silently expanded into stopping a shared project.

Use the existing installed access/state journeys and real browser integration, not a
second readiness harness. Observe current unit/container ID, image, account/marker/
host-key identities and later writes first. Prove explicit Stop stops the same unit/
container and disables host-boot start; Start reenables/starts that existing container,
retaining accounts, keys, tools and data. A stopped/unknown state must never trigger
implicit creation, restart, account repair or operation replay. Test current-owner,
nonowner/member and distinct operator authority and shared-impact confirmation.

For key rotation, retain original inputs/evidence, add a declared replacement public
key, preview/confirm the native update and prove actual new-key SSH with its private
key remaining on the client. Explicitly remove/apply the intended old test key and
verify *authentication refusal*, not transport/routing failure. Confirm another user's
access, already authenticated sessions, home/files, membership and privilege state
survive. Test stale preview/saved-set/last-key guards and truthful uncertain outcomes;
do not manufacture ambiguous native drift or delete unrelated keys as a probe fixture.
Destroy remains outside this proof and requires its own selected contract and approval.

The template owner mounts `mountSodaspaces` through the
[component contract](terminal-integration.md). Preserve real native OAuth, TLS, forms,
clipboard and BFCache; source component tests or injected browser state do not replace
that integrated proof. Any retained deployment still needs current paired backups and
its exact affected-component scope, including the changed helper.

## Integrated existing-account browser terminal mode

Append `--allow-existing-terminal` to the authenticated probe and add the exact
private-input declaration `"terminal_actions": ["create", "end"]`. Old read-only
inputs and mixed environment-access/terminal scope are refused. This requires
explicit native shell creation/End permission for both existing members; it does not
expand older approvals. After the ordinary OAuth/BFCache journey, each user follows
the shared project/name New chooser and named per-terminal End confirmation.

Only a bounded transient socket-output buffer recognizes run-owned identity/home/TTY
facts; no transcript or authentication frames are retained. Escape, focus escape,
same-target Refresh and explicit HTTP End are checked. The admitted End is bound to
the exact environment/session ID observed in that user's socket frame. An accepted
`ending: true`, socket closure or absent metadata is **not native cleanup proof**;
returned shell PID/start facts require independent host-side cleanup observations.
No environment creation, Join, lifecycle or key action is permitted by this mode.
The current UI port is authored and locally fixture-tested, not installed proof.

## Integrated six-session workspace matrix (authored, not installed proof)

`--allow-workspace-matrix /private/matrix.json` is a **separate** opt-in on
`tests/installed/sodaspaces.ts`. It cannot combine with access/management mode or
inherit the old single-terminal approval. The original restricted base input,
`terminal_actions: ["create", "end"]`, clean exact revision, target environment
variable, CA/browser trust and real OAuth/BFCache prerequisites still apply.
No installs, Join, Stop/Start, key changes, credential seeding, fault injection or
unrelated cleanup are admitted. Every lifetime POST is consumed once against the
original actor/path/body; UI helpers do not authorize or call mutation APIs.

The extra restricted JSON's closed shape is owned by
[`sodaspaces-matrix-input.ts`](../tests/installed/sodaspaces-matrix-input.ts):

- `target`, `revision` equal the base input; `actors` equal its two IDs in order;
  `sessions_per_actor: 6`; `actions: ["create", "attach", "hide", "return", "end"]`.
  This means **twelve** new managed sessions total, six concurrently per actor,
  followed by explicitly named End. Failed runs preserve outstanding exact IDs;
  no catch/finally block ends remote sessions or repairs native state.
- `projects`: exactly two distinct `{environment, repository_id, repository_path,
  ssh}` records. The first repository equals the base repository. `ssh` contains
  the two existing project-account aliases in actor order, each named
  `soda-matrix-…`; all four aliases must differ. `ssh_config` is an absolute
  restricted regular file with independently pinned project identities and
  client-local personal keys. No key enters browser input. No routing claim is
  inferred from this configuration.
- `cli: []`, `cli_effects: []`, `provider_use: "none"` omits all CLI execution and
  explicitly records **not run**. To select it, provide exactly one case for each
  of `codex`, `claude`, `pi`: `{tool, version, prompt_file, ready_text, expected_text,
  minimum_output_bytes}`. Versions are exact observed `--version` strings;
  prompts are restricted files up to 4096 bytes containing **non-secret fixture
  tasks**, not credentials or requests for tool/workspace mutation. `ready_text`
  is the version-specific observed input-ready indicator (1–80 characters), never
  a login, directory-trust or tool-permission prompt. Both transports must observe
  it before submitting a task; no sleep, automatic confirmation or authentication
  substitutes for readiness. The expected output (1–80 characters) must not occur
  in the prompt; declared streaming volume is 4096–65536 bytes. All tools and
  personal credentials must already be configured under the approved accounts;
  absence/failure is not a pass and never triggers installation or authentication.
- Selecting CLIs additionally requires
  `provider_use: "browser-and-ssh-for-declared-clis"` and exact `cli_effects`:
  `["personal-cli-state", "provider-calls", "browser-and-ssh-pty",
  "interactive-input", "interrupt-and-disconnect"]`. This permits twelve prompt
  submissions (three tools × two actors × browser/SSH), ordinary CLI-owned state
  writes, input, interrupt and closure of the exact run-owned client SSH processes.
  It does not authorize arbitrary commands, project cleanup or provider resources.

The shared scenario drives real New/name, All/Sessions, panes, compact projection,
Hide/Return and named End controls. It verifies exact create correlation and later
attachment IDs, refuses an already observed writer in a second guarded page,
preserves six DOM/xterm owners through layout and checks original PID/start/memory
through reload. Independent **ordinary project SSH** checks require the same account,
exact shell, owned tmux socket, transient unit, cgroup and record. End success requires
all owned resources absent/inactive and siblings still live; missing permissions or
unknown observations fail rather than being treated as absence. HTTP End may be
accepted or transport-unconfirmed: neither is the independent cleanup result.

CLI scenarios record versions, browser/tmux/TERM/terminfo, bounded Unicode/ANSI
indications, declared streaming volume, explicit paste/mouse/resize/selection/
interrupt interactions, exact reload without input replay and ordinary SSH PTY
comparison. No transcript, prompt, auth frame or private diagnostic is retained.
Programmatic paste and wire indications are **not visual/physical-keyboard proof**;
results explicitly require review of redraw, mouse/paste semantics, history/selection,
resize/interrupt, streaming and SSH comparison. No CLI acceptance boolean is fabricated.
Long-lived CLI child cleanup and retained application state still require scoped
native review. Local driver fixtures use emitted components, fixture-only loopback
TLS/browser trust and synthetic peers/native observations: not Forgejo, SSH or tmux
acceptance. The existing native boundary probe below remains required, including its
retained framing failure.

## Integrated existing-project management mode

`--allow-existing-management /private/management.json` additionally runs the
existing-terminal mode, owner Stop/Start with an active browser terminal and a
run-owned persistent home marker, native nonowner lifecycle denial, and explicit
saved-key/native Apply/revocation using two temporary client keys. Original saved
keys are never deleted. New SSH authentication, refusal of the removed temporary
key and survival of already authenticated temporary-key/Bob sessions are separate
checks. Successful completion removes only the two new saved keys through the UI,
explicitly reapplies originals and verifies the original managed files/identities.
Failure does not trigger automatic restoration, mutation replay or cleanup of keys.

Its base browser input also requires `terminal_actions: ["create", "end"]` for
those terminal operations; the separate management scope still authorizes Stop/Start
and the declared key changes. The private management JSON has exactly `target`, `project`, `cid`, `ssh_config`,
`key_a`, `key_a_public`, `key_b`, `key_b_public`, `original_alice`, `original_bob`.
All file references are absolute restricted regular files. The trusted SSH config
must define pinned `soda-e2e-host`, `soda-e2e-alice` and `soda-e2e-bob` aliases for
this declared fixture and existing project. Project aliases must use an independently
verified `HostKeyAlias` pin, not a stale IP-key entry: the probe overrides HostName
with the current CID-bound native address and compares it with the drawer's SSH
command. Podman can change that address on Start; it is not a persistent identity.
Root SSH only observes state; every
lifecycle/key mutation goes through the actual protected browser UI/API, with
single-use actor/path/body/method-bound request admission. Private keys stay on the
client and never enter browser inputs. Management forwarding is not laptop routing
proof. The current bounded scenario deliberately requires the existing Alice/Bob
fixture accounts; it is not an arbitrary project maintenance tool.

## Native terminal boundary probe

`internal/host/terminal_native_test.go::TestInstalledTerminalBoundary` is an opt-in
product test, not browser proof or permission to open a shell. Build the matching-native
`internal/host` test binary into a new ignored artifact path; invoke only that test on
the explicitly approved fixture with `SODA_NATIVE_VALIDATE` equal to its hostname and
`SODA_TERMINAL_NATIVE_INPUT` naming a private absolute JSON file. Input fields are
`target`, `project`, exact `container_id`, `terminal_protocol:"managed-tmux-v1"`,
and exactly two `accounts`, each with `login`, stable `identity`,
expected native `uid`, `gid`, `home`, numeric supplementary `groups`, and `admin`.
Obtain these expected values independently; do not infer them from terminal output.

The root-only probe creates an exclusive marker and temporary 0600 Unix socket beside
that 0600 file in its private directory. It runs the candidate helper in-process,
without replacing/restarting the installed helper or touching project images/accounts/
keys. It opens existing-account shells, checks identity/home/groups/TTY, resize,
Ctrl-C, real/effective/saved credentials, shared-profile settings and current sudo
permissions; it refuses a mismatched marker/actor without repair. It creates managed
project-systemd/tmux sessions, detaches/re-attaches the same shell PID/start identity
and in-memory variable, then explicitly Ends the owner and independently checks
process disappearance. Required packages/managed program must already be delivered
under separate scope; the probe installs nothing. Further cases cover **owner** transport
EOF, a real 60-second silent lease and SIGKILL of only a test-owned child helper;
independent exec observations must confirm the login, foreground job and launcher
are gone. The internal child-mode environment flag is used only by that parent test,
not a standalone invocation or installed-service control. Native shell/sudo bookkeeping may write normal history/
audit state; no transcripts or credentials are captured. Keep inputs, marker and
result; an occupied run refuses replay. `terminal-proof.json` records only this scope.

The [approved isolated x86_64 proof](implementation-history.md#approved-native-terminal-fixture-proof)
passed the **earlier request-owned version** of this probe plus independent continuously
held own-key SSH/process-preservation observations. It does not validate this managed-
tmux revision. That proof closes only its original native boundary, not the later
public browser/OAuth/proxy journey or installed delivery. Further executions still
need their exact target/action scope; compiling a test or setting opt-in variables
is not permission to open shells, kill helpers or change services.

## Read-only installed observations

After separately authorized installation, inspect:

- `rpm-ostree status`, the extensions journal and the active native packages;
- `systemctl status soda-host.socket forgejo.service soda-dashboard.service soda-proxy.service cockpit.socket tailscaled.service`;
- native service journals, listener addresses, subordinate UID/GID mappings, helper socket owner/group/mode, service UID and persistent directory ownership;
- TLS trust and the configured Forgejo/Sodaspaces browser origin, native Forgejo clone URLs, Cockpit's root-only PAM policy;
- actual project inspection/IPs, client routes and existing firewall policy.

Do not paste credentials, full container environment dumps, provisioning password hashes or private keys into evidence. Service/listener state alone does not prove a login or development journey.

## Explicitly permitted product journey

These actions **change real state**. Use explicitly approved users, repositories,
projects and credentials with real client reachability; existing fixture grants
are not reusable permission. Extend/invoke product-owned installed tests. Keep shared
installation identity, precise denial results and failure-safe bounded snapshots;
a failed inspection is not evidence of absent state or forbidden access.

1. Complete the core-owned native Forgejo operator setup and OAuth bootstrap. Sign in through the browser. Both old Soda frontends are removed from current source; bounded native Sodaspaces browser coverage is recorded in the handoff. Use native Forgejo operator setup, not the retired Soda People form. Test Soda's authority boundary: Forgejo administrator status alone never grants native Cockpit/root or extra Soda operator authority. Do not duplicate upstream administrator-API permission tests.
2. Provision the approved Alice/Bob identities through native Forgejo; each completes native password/security requirements and authenticates independently. Exercise native-page/Sodaspaces identity matching and implemented development-key controls within their explicit action scope. Private keys stay on their clients; no database-seeded browser success.
3. Alice creates an ordinary Forgejo repository with native Git credentials and then creates its Soda environment. Only its human owner may create that environment. Both explicitly select **Add me to this project**. No creator auto-enrollment is assumed.
4. From the real developer client, verify the SSH host key through native operator access and connect to the displayed project IP as Alice and Bob. Exercise interactive SSH, a noninteractive command, SCP and SFTP. Do not disable host-key checking to manufacture a pass.
5. Run `tests/installed/project-os.sh` inside the project with `SODA_NATIVE_VALIDATE` set only for that named target. Inspect `sudo -l`: Alice is project-local administrator, Bob is not. Neither acquires a host account or the host engine socket. A second project must have separate writable state and native identities.
   Also run `tests/installed/project-foundation.sh` as an ordinary member in the
   approved project. It requires the same explicit native scope, checks packaged
   tools and creates one private `soda-foundation.*` directory in the project-local home
   (not a potentially noexec/ephemeral `/tmp`). It compiles/links C against OpenSSL/zlib and a C++17 sample via CMake/Ninja,
   runs CTest, GDB and strace, and retains sources/build/logs/ELF evidence. It does
   not install packages, use provider credentials, start services/containers or clean up.
   A missing package or required operator repair fails that image candidate; no
   native execution of this new probe has yet been recorded.
6. Both use personal home checkouts, ordinary Git commits/pushes/merges and `~/shared`. Verify shared writes/readback by both. No Soda-managed branches/selectors or cleanup are expected.
7. Alice installs a shared tool with native mise using the documented root/global path (see [development environment](development-environment.md)). Execute `tests/installed/shared-tools.sh` from the real client with explicit Alice/Bob/project inputs. Verify both resolve and execute the same `/opt/mise/installs` installation without independent downloads.
8. In Alice's ordinary checkout of `tests/fixtures/workload`, provide a disposable database password and explicitly run `tests/installed/workloads.sh`. This builds images and starts services. Confirm the web/database from Bob and the actual client, not only localhost. Inspect a real bind-mounted file and persistent database write/read. Do not delete volumes automatically.
9. With separate permission, stop/start the existing `soda-project@ID.service`; then authorize a host reboot independently. Verify the same project container, accounts/homes/SSH host keys, shared installs/files, service configuration and database data survive. Do not remove/replace the project container to make it start.
10. Once implemented, verify the browser terminal is the user's existing project-local account/home, with explicit session lifetime, origin/CSRF, bounded transport and cross-project denials. Opening it must not create/join/start anything or expose host root.

Test Soda's customization and integration, not upstream Forgejo business logic:
OAuth/session/CSRF handling, actor matching, environment authorization after native
rename/transfer, template/asset delivery and Caddy route boundaries. Cover the
Sodaspaces drawer's stale/denied/error states, keyboard/focus, narrow/wide display
and exact-version hook compatibility, without automatic Linux remapping. Native
account/repository operations above supply fixtures and exercise Soda's project
integration; they are not independent tests of Forgejo's implementation. Keep only
focused native smoke checks where Soda changes could cause a regression. Do not
add general upstream login/MFA, administration, collaboration or Git/LFS/package
conformance suites, or rebuild the retired 179-group JSON register.

The highest-risk profile is nested Podman with private cgroups, user-namespace allocation, fuse and the selected capabilities/seccomp/SELinux arrangement. If a real blocker appears, correct the concrete mechanism. Only then investigate the project-scoped host fallback; do not expose an unrestricted host socket, enable a privileged parent or introduce a VM substitute without revisiting the design.

## Operator Tailnet journey

Read-only: inspect the actual native state/peers and UI consistency. With separate permissions, exercise browser sign-in, exit-node selection/advertisement, LAN preference and provider approval. Preserve the native daemon's state rather than replacing it with a Soda inventory.

Forgejo Git advertisement is refreshed only when its actual private listener accepts the Tailnet IP. The supplied first-install/activation may bind a selected LAN IP instead: choose the intended Tailnet private address for activation or explicitly change/restart the native Forgejo port mapping before requesting refresh. The helper must not advertise an unreachable endpoint. Browser/OAuth origins remain fixed operator configuration, not an invented `host:30000` URL. Tailnet enrollment does not itself establish project subnet routes; inspect and approve those separately.

## Operator Runners journey

The [runner completion plan](runners-port.md#implementation-and-completion-gate)
owns the dashboard migration's source/native/provider/preservation exits. The
existing installed operator probe still reads Cockpit inventory only. The
[runner-native preparation guide](runners-native-validation.md) now documents
callable runner phases, strict per-phase private inputs, fixed native state/proof
observations and official Forgejo workflow dispatch/exact-run reads. Their shared
Soda-pages driver integration and matching native export remain pending; importing
the module is not an installed journey. No existing read-only probe opts into these
effects. Local Go-HTML/Lit and new parser/transport-double/filesystem fixtures are
not this native proof.

Provider registration and jobs are **not read-only checks**. Supply explicitly approved Forgejo resources and tokens. Verify create/list/start/stop/restart, actual native runner account/capacity and a genuine provider-scheduled job on trusted code. Verify configured Forgejo administration links and one local slot per runner. Provider workflows/results stay provider-owned.

Removing a runner destroys its local state. Only exercise removal on an explicitly disposable runner with permission, and inspect provider-side cleanup separately. An unavailable provider/account is unverified, not a local-fake success.

## Compatible follow-up checks

- On the authorized native builder, the source entrypoint also exercises console/fetch process doubles and project-tool build-boundary tests. They do not query a real Tailnet or authenticate provider CLIs.
- On the actual host, inspect the [operator welcome](console-welcome.md) in an interactive root login and confirm noninteractive SSH/transfer output stays quiet. Printed configured origins are not proof of a listening service.
- Follow [branding review](branding-review.md) separately for the optional `branding` renderer tests and actual browser component check. It needs explicit native browser/renderer prerequisites, a disposable target and a fresh evidence directory; it is not an automatic source-gate side effect.
- In Alice/Bob's project, inspect `tea --version`/`gh --version`. With separate provider/account permission, follow [CLI authentication](project-clis.md) and confirm each uses only their own credential state. Mere CLI availability is not API compatibility evidence.
- Capture actual UI only under the [screenshot brief](screenshot-capture.md); no generated or component-sheet images stand in for installed product behavior.

## Independent aarch64 evidence

Repeat on actual matching-native aarch64 access when authorized. No sibling barrier, cross-build substitute or emulator result is native installed proof. Record that architecture's own results and unresolved differences.
