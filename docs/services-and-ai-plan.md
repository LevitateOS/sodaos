# Services marketplace, repository AI and desktop workspaces

The user requested three additions: a simple Podman services marketplace, an AI
issue resolver, and AI PR review with optional fixes and a configurable loop count.
These are requested product work. This document proposes their implementation;
**the marketplace and AI automation are not implemented or validated yet**. The
separate user-requested terminal navigation follow-up adds Open in drawer to the
existing Spaces workspace; its local evidence belongs in the handoff.

The user clarified the desired AI experience: **issue → resolver → PR → review →
the same resolver fixes findings → review again**, with the configured bound. AI
processes should appear as real, named terminals in Spaces and its repository
drawer, launched with user-defined commands and task variables. This replaces the earlier
proposal's implication that Actions logs alone would be the user interface.

The user also requested graphical workspaces for the actual Codex/Claude desktop
apps and their computer-use features. The [desktop proposal](#4-desktop-workspaces)
below extends the product beyond terminal transport. It is an architecture proposal,
not an implemented desktop runtime or a change to existing Rocky projects.
The subsequent user decision selects **Linux desktops**, with Rocky/Fedora headless
and KDE [creation profiles](project-os.md#selected-environment-profiles); GNOME is
deferred. This supersedes the earlier Windows-first compatibility recommendation.

The reviewed first implementation candidate is **appliance-wide, operator-managed
services with online digest-pinned downloads** and **automatic runs for current
trusted contributors after repository opt-in**. These are explicit planning defaults,
not evidence that the user answered the earlier placement/trust questions. Keep one
candidate for each; a different placement or trigger policy changes its scope before
implementation. Do not build parallel global/project catalogs or an alternative AI
scheduler. Existing projects, registrations, credentials and fixtures stay intact.

## Containers, pods and runtime lifetimes

A container runs an image with its own filesystem view and processes. A Podman
pod groups containers and can share namespaces, especially networking. Containers
in the same pod can communicate over localhost; they still need explicit volumes
to share durable files. A pod is not a VM or a Linux account database. See the
[Podman pod contract](https://docs.podman.io/en/latest/markdown/podman-pod-create.1.html).

Use **Services** in the product navigation. Users choose an app, not a container
topology. A one-container app needs no pod. A recipe may use a pod when an app and
its companions actually benefit from sharing networking.

| Product object | Lifetime and owner | Runtime candidate |
| --- | --- | --- |
| Marketplace service | Persistent app and data; appliance operator | Host Podman, native Quadlet/systemd |
| Development project | Existing shared accounts, files and tools; project administrators | Preserve the current Rocky project and nested Podman |
| AI run | Linked issue/PR attempt with bounded review/fix rounds; repository policy and Forgejo Actions | Isolated checkout and agent process, with a live terminal projected into Spaces/drawer |
| Desktop session | Graphical session of a project-local account; persistent app data in that account’s home | KDE within the Project OS foundation; desktop transport is not implemented |

Automatic AI runs must not commandeer a developer's existing shell, dirty checkout,
home or personal credentials. Sharing the Spaces UI does not require sharing those
resources or tying execution to one viewer's browser session.
Project service catalogs would instead need project authorization and the existing
nested engine; they are not interchangeable with an operator's host catalog.

## 1. Services marketplace

### Scope and native ownership

The first candidate is a global **Services** catalog and operator management page
under `/-/soda/services`, linked from Global SodaOS settings. Start with **Adminer,
Vaultwarden and Homepage**, as requested. Adminer is an operator-provisioned appliance
utility in this scope, not a database service installed into a repository. Project
services retain their existing native nested-engine workflow; no project marketplace
controls or second engine backend are implied.

Every catalog view, configuration/log read and mutation requires the configured
Soda operator with fresh server-side authority. Intended users access app URLs under
each app's own authentication; sharing a URL is not a Soda access grant. A general
member-facing service directory is separate from this first management surface.
Homepage is the requested operator-maintained shared start page, independent
of Soda's operator installation inventory; do not duplicate automatic discovery.

Use versioned shipped catalog metadata and ordinary
[Quadlet definitions](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html).
Each entry specifies reviewed license/source, immutable per-architecture image
digests, fixed internal ports, typed bounded settings, native health checks and
explicit durable paths (possibly none). Installation pulls the selected digest
online and verifies its platform/image identity. No `latest`, arbitrary image input
or invented digest; unsupported architectures are unavailable before Install. The
existing four-image appliance bundle does not include these apps. Offline catalog
installation would require an explicitly extended bundle/notices/installer contract,
not an assumption that the current export includes them.

Allocate an opaque instance ID for native paths, units and containers; the bounded
display name never selects a native target. Retain the installed recipe/digest and
non-secret configuration snapshot under `/etc/soda/services/<id>/`, restricted secret
files alongside it, and declared data under `/var/lib/soda/services/<id>/`. Add these
owned paths to staging/tmpfiles and documented backup inputs with correct numeric
ownership and SELinux labels. Native unit/Podman state remains running truth; there
is no competing SQLite running/desired-state model. An installed snapshot remains
manageable if a newer catalog withdraws its entry.

### Install, retry and persistent lifecycle

1. Select the catalog entry, display name, app settings and preprovisioned private
   HTTPS hostname/certificate. Validate exact catalog version, architecture, input
   bounds, data-path type/ownership and hostname/listener collisions before effects.
2. Review the selected image, endpoint, durable storage and application account setup,
   then explicitly Install. A settings edit or page visit does not start a pull.
3. Short fixed Services methods on the existing root:soda Unix-socket helper
   authorize the dedicated Soda service caller and start an exact per-instance
   systemd installation unit. The web handler separately checks the human operator.
   That unit owns the long pull/start work across browser/backend disconnects; the
   UI polls its native phase/status and bounded journal. The current project helper's
   global mutex and three-minute request cannot own registry pulls. Use short control
   calls, serialization per instance and a separate shared proxy-update lock; do not
   block project Join/key/lifecycle operations for an app download.
4. The unit creates protected configuration/data paths, pulls the exact digest,
   atomically writes its owned Quadlet/proxy inputs, reloads systemd, validates and
   applies proxy configuration, then enables/starts and checks the app. Existing
   unexpected paths/units/containers are conflicts, never overwrite or cleanup targets.
5. Show the last confirmed phase on failure. Preserve data, installed configuration,
   pinned image and bounded diagnostics. Explicit Retry examines only that same
   instance, skips completed effects and resumes its original immutable install input;
   double-click, timeout or lost HTTP response must not create another instance. A
   changed form is not a retry and cannot overwrite later app-native settings. If the
   native state cannot safely be identified, report a conflict for operator inspection.

This is bounded correctness for an install operation, not a recovery service or
reconciler. Do not accept host paths, command strings, Podman flags or arbitrary
unit names from the browser. The fixed root operation resolves only owned instances
and catalog inputs. Log reads must be bounded/redacted; never return full inspection,
environments, private app configuration or tokens.

**Start** enables host-boot start and starts the installed pinned instance.
**Stop** disables boot start and stops that instance, interrupting its users while
retaining settings/data. Implement boot intent through the owned Quadlet `[Install]`
drop-in and generator reload, then start/stop the exact generated service; ordinary
`systemctl enable/disable` cannot manage generated Quadlet units. Observe the effective
boot target wiring separately from active state. See the native
[Quadlet enabling contract](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#enabling-unit-files).
Start, reboot and refresh never repull a moving tag or
upgrade the image; use installed digest identity with `Pull=never`. App durable
storage must survive any native container recreation needed by Quadlet. This is
separate from retained Project OS roots, whose normal Start must start the original
container. Updates, uninstall/data deletion and discovery remain separate scope.

### Private ingress and truthful readiness

Use an exact **distinct hostname for each app**, separate from the Forgejo hostname,
on the configured private listener with an operator-preprovisioned matching TLS
certificate and client DNS/route. A different port on the Forgejo hostname does not
isolate cookies. No app under Forgejo paths or `/-/soda/`, wildcard host acceptance,
public bind, public ACME automation or forwarding of Soda cookies/OAuth grants.

Publish backend ports only on unique loopback listeners and generate exact-host
Caddy fragments pointing to them. Extend existing proxy mounts/configuration for
app fragments and restricted certificate paths. Validate the whole candidate before
reload; on failure leave the previous working proxy configuration active and report
this instance unavailable. Stop can leave its route returning unavailable without
deleting configuration. A browser never supplies arbitrary certificate paths or a
proxy destination. Current activation config serves only the Forgejo origin, so this
is required new source work, not existing ingress support.

Report image installation, local unit running/boot-enabled state, app-native health,
proxy configuration and client reachability separately. Open can be offered for a
validated HTTPS URL when the app/proxy are ready, but a loopback health check cannot
claim the user's DNS, route or certificate trust works. Complete the install journey
from the intended client. No endpoint probing API may become arbitrary host forwarding.

### App-specific first-version contracts

- **Adminer:** use the reviewed
  [Docker Official Image](https://hub.docker.com/_/adminer/) (community maintained).
  Configure a fixed operator-selected database target/list through the shipped
  `login-servers` integration; `ADMINER_DEFAULT_SERVER` alone only pre-fills a host.
  Validate driver/host/port and prove the installed login controls enforce the list;
  this is not a claim of network isolation. Keep HTTPS for credential entry. Database
  authentication/authorization remains native, and Soda neither saves database
  passwords nor mounts unrelated database data. Adminer has no mandatory data volume;
  retain only its generated non-secret plugin configuration.
- **Vaultwarden:** retain `/data`, set `DOMAIN` to the exact HTTPS origin and use its
  shipped health check. First-version onboarding uses closed public signup, a protected
  operator admin secret and native invitations delivered through configured SMTP.
  Include the SMTP connection and secret input plus a real invite → account → vault
  journey before calling Install complete; no open-signup bootstrap window. Keep the
  admin token in the supported Argon2-hashed form and private file input supported by
  the pinned app; do not emit it in Quadlet arguments, logs or Soda's database. Vaultwarden
  owns vault identities and its admin configuration; `/data/config.json` can override
  environment inputs, so later Start/Retry must not overwrite app-native changes or
  label the original form as effective settings. Record backup inputs and consistent
  native backup procedure without creating a new backup platform. See the upstream
  [configuration](https://github.com/dani-garcia/vaultwarden/blob/main/.env.template)
  and [deployment guide](https://github.com/dani-garcia/vaultwarden/wiki/Deployment-examples).
- **Homepage:** retain `/app/config`, use its native health endpoint, exact allowed
  Host and external HTTPS URL, and the pinned version's native authentication.
  Current [installation documentation](https://gethomepage.dev/installation/)
  describes v2 authentication; pin/verify that capability instead of assuming older
  versions have it. The first candidate uses its native password mode with private
  auth secret/password input and the declared trusted private audience. Its password
  mode has no app-level rate limiting; do not portray it as public-service protection.
  Auth settings need a reviewed restricted environment-file channel; config-only
  `HOMEPAGE_FILE_*` substitution does not cover every auth setting. Full Podman
  inspection may expose environment secrets and must never reach the UI/evidence.
  Start with static links, no host engine socket, auto-discovery or secret-bearing
  widgets. The install form accepts a bounded initial static-link list and renders native
  configuration. Later edits use the operator's ordinary access to that exact native
  config; ordinary members get no host files, accounts or implied per-user page editor.

Verify pinned app-specific secret input, signup/auth defaults, startup requirements,
health endpoint and x86_64/aarch64 artifacts before packaging. Declare necessary
outbound destinations (database, SMTP, registry and any enabled app features);
network reachability is not permission to broaden the host listener. Include disk
full, pull interruption, unsupported platform, proxy failure, healthy-unit/unhealthy-app,
retry-after-disconnect and normal Stop/Start/reboot persistence in completion checks.

## 2. Issue resolver and 3. PR review/fix

Configuration lives in **Repository settings → AI automation**, alongside the
separate **Sodaspaces** settings page; the
[settings contract](sodaspaces-plan.md#settings-pages-and-os-selection) owns placement,
OS selection, server-side authority and native workflow-backed saving. **Sodarunners**
is global/operator-only. Preserve native repository Actions runners/secrets/variables.
Project OS selection is independent of the isolated job's eligible execution image;
changing AI settings never retargets a personal terminal or project root.

### User-defined commands and variables

The integration is **user-defined CLI commands with documented variables**, not
an AI-provider API or a Soda conversation framework. Configure a **Resolver command**
(used for issue resolution and fixes) and a **Reviewer command**. Either can invoke
Codex, Claude Code, pi or a repository script; the same command can inspect the phase
variable to serve both roles. Models, prompts, CLI flags and provider-specific resume
options belong in that command. Soda must not require provider presets, parse a
vendor's terminal output or pretend that repeating an executable resumes a conversation.

| Setting | Behavior |
| --- | --- |
| Issue automation | Off, or resolve newly opened issues under the configured native trigger/trust policy |
| PR automation | Off, review, or review and fix; explicitly include new commits if enabled |
| Resolver / Reviewer command | Trusted workflow-authored command, run inside the isolated job in its checkout; required only for enabled phases |
| Setup and tests | Optional setup and declared required-check commands, with failures reported accurately |
| Execution image | Eligible shipped headless job image/tool set and compatible native runner labels; independent of a repository's persistent Project OS |
| Credentials | Named native Forgejo secret references; no values in versioned configuration or command fields |
| Maximum fix rounds | Integer 0–5; proposed default two in review-and-fix mode; zero means one review and no fix invocation |
| Run timeout | Positive finite overall deadline within the native runner/operator ceiling, covering setup, commands, tests and publication |
| Output | Real live terminal, native issue-linked PR/review, actual check outcomes and Forgejo Actions run link |

Use the native workflow's explicit Bash shell and ordinary environment variables.
Do not invent another string-template language or splice issue/PR text into shell
source. The job wrapper supplies the following contract; it is planned source work,
not an existing environment-variable API:

| Variable | Value |
| --- | --- |
| `SODA_PHASE` | `resolve`, `review` or `fix` |
| `SODA_EXECUTION_ID` | Native Actions run ID plus rerun number; exact identity for this execution's terminals/artifacts |
| `SODA_PUBLICATION_KEY` | Stable logical event identity used for native branch/PR deduplication, as defined below |
| `SODA_REPOSITORY_ID` | Stable native repository ID; names/URLs are context data |
| `SODA_ISSUE_NUMBER`, `SODA_PR_NUMBER` | Decimal identifier when applicable, empty otherwise |
| `SODA_BASE_SHA`, `SODA_HEAD_SHA` | Exact captured base and current candidate commit; the context file separately records the expected remote PR head |
| `SODA_ROUND` | Zero for the initial resolution/review; 1–N for each fix and its following review |
| `SODA_WORKSPACE` | Absolute attempt-owned checkout path, also the command's working directory |
| `SODA_COMMAND_DIR` | Read-only snapshot of automation scripts from the trusted configuration revision |
| `SODA_CONTEXT_FILE` | Readable UTF-8 JSON file with a versioned schema, captured event/issue/PR text, revisions and links; untrusted task data |
| `SODA_STATE_DIR` | Private attempt-owned directory retained between phases for the user's CLI state/session identifiers |
| `SODA_CREDENTIALS_DIR` | Private runtime directory of configured agent credential files; no Forgejo write/publication credential |
| `SODA_REVIEW_FILE` | Bounded UTF-8 findings from the latest completed review; output path during review, input during fix |
| `SODA_REVIEW_STATUS_FILE` | Review output path for exactly `clean` or `changes_requested`, optionally followed by a newline |

For example, a repository can set `"$SODA_COMMAND_DIR/resolve" "$SODA_CONTEXT_FILE"` and
`"$SODA_COMMAND_DIR/review" "$SODA_CONTEXT_FILE"`. The user-authored scripts translate that
input into their CLI's prompt/stdin/flags. Ordinary quoted variable expansion passes
data; never `eval` event text. Issue and PR command source/scripts come from the
repository's captured default-branch workflow commit, never the PR head or candidate tree; record that commit and the
script tree/blob identities. The PR's selected target/base can be a different branch
and remains task input. Stage scripts read-only in `SODA_COMMAND_DIR`. A relative
script in `SODA_WORKSPACE` is candidate code, not the trusted automation snapshot;
the settings generator must preserve that distinction. Repository code,
including setup/tests, is still executable untrusted input; this rule is not a sandbox
against a malicious command or dependency. Named agent secret references map to
bounded file names in `SODA_CREDENTIALS_DIR`; the user command can use the CLI's
supported credential-file, stdin or environment interface without echoing values. The wrapper must prove
how the selected native runner materializes these private files and removes its
original secret inputs; no automatic `_FILE` support or existing secret mount is
assumed. Never substitute secrets into logged command source or argv.

Agent/provider credentials are deliberately available to the command sandbox.
Setup, tests and agent-invoked repository code in that same security domain may read
or exfiltrate them; private file modes do not prevent that. Their provider scope and
spending authority define the exposure. The Forgejo publication credential belongs
outside that sandbox. Repository writers able to author runnable workflows are also
inside native repository-secret trust unless a stronger supported mechanism is proved.

A nonzero command exit is failure, not review feedback. A successful reviewer writes
both its findings and explicit status; exit zero alone does not mean clean. Missing,
malformed, oversized or contradictory output is inconclusive and stops automatic
fixing. The wrapper clears stale outputs before each review, accepts only regular
non-symlink files at its fixed attempt-owned paths, limits findings to 64 KiB and
validates status after successful exit. Findings remain untrusted data, never shell
source or permission to publish. This small result contract enables a finite loop;
it does not require an AI SDK. Resolver success is determined from actual Git changes
and declared checks, not a vendor's prose. Persist every round's bounded findings and
check outcomes before reusing the paths.

Keep all command phases in one isolated native job so the same attempt preserves
the resolver's checkout, branch and state directory; distinct Actions jobs do not
share a filesystem implicitly.
The user's command chooses whether/how its CLI resumes native conversation state.
A reviewer may use separate CLI state within that directory. A manually requested
native rerun gets a new execution ID and fresh checkout/state;
cross-execution provider conversation restoration is not promised. Its publication
key and budget rule follow the explicit retry contract below. Do not borrow personal
CLI credentials or workspaces. Supported optional
CLIs must include their dependencies and a working launch path; arbitrary commands
remain possible when their repository-managed dependencies are available. Provider
spending limits remain provider/CLI-owned; a round count is not a monetary budget.

Keep effective automation in ordinary `.forgejo/workflows/` files and native Actions
secrets/variables. The settings page prepares a reviewable native Git/API change,
checks the expected configuration revision and preserves unrelated/manual workflow
content. Offer a diff or report unsupported edits; never silently overwrite them.
A proposed change is pending until its required commit/merge takes effect. No second
activation toggle or authoritative workflow copy lives in Soda's database. Merely
exporting YAML or presenting mock configuration does not complete the save journey.

Forgejo owns events, dispatch, queueing, cancellation, job history and results.
Its [v15 workflow reference](https://forgejo.org/docs/v15.0/user/actions/reference/)
documents issue, PR, manual-dispatch and concurrency controls. Use these before
considering a webhook receiver. Soda supplies capacity and the bounded command/terminal
adapter, not a parallel queue or Git scheduler. Set the issues event's `types` to
`[opened]` and the PR event's `types` to `[opened, synchronize]` when enabled; do not
inherit broader upstream edited/reopened defaults. Snapshot trusted configuration at attempt start.
Before checkout, setup, secret materialization or terminal creation, validate current
native event/actor/repository/head eligibility. A workflow from PR content is not a
trusted policy merely because its actor was approved before. Native event/token
selection must uphold that gate; `pull_request_target` is not blanket authority to
execute PR code with its secrets. Disabling automation stops future eligible dispatch;
active native runs require explicit Cancel. Renames resolve by repository ID; transfer, permission
loss, Actions disablement or missing secrets must prevent new work/publication and
report a native failure, not fall back to cached authority.

### Live AI terminals in Spaces and the drawer

The job wrapper launches each CLI once under a run-owned PTY/supervisor inside the
single command job; normal Actions logs are not attachable terminals. Native Actions
Cancel and deadline termination must reach that supervisor and all its owned command
descendants. Viewer loss never restarts or ends a command. The ordinary terminal
rendering is shared, but this process/attachment owner is new source work.

Every running resolver/reviewer has an exact execution identity, repository and
issue/PR link, useful name (for example `Issue #42 · resolver`) and a real terminal
connected to that process. Use the existing shared workspace/terminal rendering;
do not substitute a static log pane, separate agent-chat frontend or second drawer.
Selecting, moving or reopening the view attaches to the existing process and never
replays its initial prompt. A terminal's output alone cannot report semantic agent
states such as "review passed" or "waiting for approval"; use explicit validated
process signals where available.

**Current gap:** personal browser terminals are bound to an existing project
account and the creating Soda sign-in context; logout ends them. The current API
only enumerates those personal terminals and cannot launch an unattended agent.
An automatic event can happen with no browser open, so its process owner must be
the authorized automation run. A permitted browser viewer attaches independently.
The implementation needs a bounded run/terminal integration and current native
repository authorization for discovery/view/control; do not fabricate a human
login session or remove existing personal-terminal logout/lease protections.

Keep view controls separate from job controls. Hide/navigation disconnects the
view without cancelling the job. Cancel ends the explicitly selected automation
attempt through its execution owner and records the result. Default observation
must not let two browser writers and automation concurrently type into one PTY;
interactive takeover, if offered, requires explicit control transfer. Job deadlines
and cancellation still apply even if nobody is viewing. Reopening a completed run
shows its retained result and does not create a replacement process.

The run-owned terminal identity includes attempt and phase; it is not a project
membership or a personal tmux session. One workflow may have sequential resolver and
reviewer processes with separately named phase terminals. Reopen shows the exact live
phase or retained result. Removing access revokes discovery, viewing and control on
fresh authorization and bounded live-connection revalidation; an open socket is not
permanent permission. Log/terminal retention follows native run retention and must
not preserve secrets or invent an unlimited Soda archive.

Repository configuration enables events, commands and limits. Merely opening Spaces
or an ordinary New terminal never starts automation. The same resolver command and
attempt state continue the loop under the command contract above; Soda does not own
an AI conversation or use a GUI app's private protocol.

### Issue journey

1. Resolve repository, issue and trusted configuration revision using native authority.
   Snapshot the default-branch commit. Issue text is task data, never authorization.
2. Start an isolated, attempt-owned checkout and real terminal; run setup, the resolver
   in `resolve` phase and declared required checks within the overall deadline.
3. Before publication re-resolve native authority, require the issue still open and
   eligible, and compare the captured default-branch base. A moved base or closed issue
   supersedes the result. On successful changes, create the deterministic branch and
   issue-linked PR for this publication key with actual check outcomes and the native run link. On no change, report no change;
   on failure, retain the bounded diff/result without claiming a resolved issue.
   Never push to the default branch or close the issue based on command output.
4. If PR automation is enabled, this **same native attempt** owns the initial
   review/fix loop and preserves its checkout/state directory. If PR automation is
   off, finish after PR creation. A human-created PR starts a new attempt directly
   in review phase. No browser must remain open for either journey.

### PR journey and exact loop semantics

1. After the eligibility gate, capture exact remote head/base and trusted configuration,
   then run setup once in the isolated checkout before any CLI phase. The inline
   issue continuation reuses its completed setup and configuration snapshot. Setup failure
   stops both issue and direct-PR attempts. Review that candidate commit with
   `SODA_PHASE=review` and round zero. Required checks run after each state-changing
   resolve/fix; review-only checks run only if explicitly configured.
2. In review-only mode (or with a zero fix limit), record the review and finish.
   Publish a commit-pinned native COMMENT/check/result, not APPROVE or REQUEST_CHANGES.
   A clean result is an automation finding, not a human approval or merge instruction.
3. With fixes enabled and `changes_requested`, invoke the configured resolver with
   `SODA_PHASE=fix`, increment the round, run declared checks and create a candidate
   commit from actual changes. Review that exact candidate again. Each fix invocation
   consumes one round, including a failed or no-change attempt.
4. Stop on clean review, limit, timeout/cancellation, command/check failure,
   inconclusive output or no further change. Required checks must pass before fixes
   are eligible for publication. Checks not run are reported as not run; exhaustion
   and inconclusive output are never approvals. Preserve failed/unpublished diffs
   as bounded native run artifacts rather than pushing known failed fixes.
5. Before any write, resolve current native permission and compare the expected PR
   head **and base**. If either moved or the PR closed/merged, finish as superseded.
   Push only normal commits descending from the captured writable branch head;
   native non-force push rejects divergent updates. A preflight read is not an atomic
   lock: handle rejected pushes and ambiguous responses, re-read published identity,
   and pin reviews/results to the actual commit. Never retry a stale diff on a new head.

A limit of two means **review → fix → checks → review → fix → checks → final review**,
with earlier exits as above. The initial issue resolution does not consume a fix
round. Keep intermediate findings as native run artifacts. Stage fixes locally through
the loop, then publish the tested candidate and one matching final summary. An older
review must not be attached as a verdict on a newer commit. A review failure does not erase earlier evidence or count as a clean
review. Native required-check/branch-protection rules continue to govern the PR.

Append normal commits to a writable same-repository PR branch only when configured.
For a fork or protected/unwritable branch, finish with findings and a patch artifact;
a separate follow-up PR requires its explicit configured native publication policy.
Do not modify another repository, force-push or merge automatically.

### Execution identity, duplicate events and reruns

`SODA_EXECUTION_ID` identifies one native execution and its phase terminals.
`SODA_PUBLICATION_KEY` identifies the logical publication independently of deliveries
and reruns. For an automatic issue-opened event it derives from repository ID and
issue number; for an eligible PR event from repository ID, PR number and event head
SHA. Opened/synchronize deliveries for the same PR head share a key. Configuration
and base snapshots belong to the claimed execution, not a caller-edited key. A native
manual dispatch explicitly requesting a new logical generation uses its native run
identity; show existing linked work before allowing a deliberately additional PR.

Use a deterministic branch in a reserved automation namespace derived from the issue
publication key. Before creating it or its PR, query the exact native branch/PR and
verify repository, immutable creator identity and expected commit. Never trust a
body/title/label or Git author text as proof. An existing unrelated branch is a
conflict, not an adoption opportunity. A lost push/create response requires native
lookup by this key before retry; duplicate delivery cannot create a second branch/PR.

The issue execution performs its first review/fix loop inline. Generic PR-triggered
workflows skip its initial and final bot publications only after verifying the native
PR creator, reserved branch and commit-bound Actions status/run association written
by the trusted publisher. This association lives in native Git/Actions objects, not a
Soda job database. It must remain queryable after the originating execution finishes.
Later human commits to that PR have a new head and follow the configured PR policy;
they are not skipped just because its branch is automation-owned. Native creation
and status-write ordering, duplicate delivery before status appears and conflicting
claims need proof before enabling the workflow; an ambiguous association must stop
for native inspection, never launch another potentially recursive loop.

Use explicit native concurrency per repository and target branch/PR to reduce overlap.
Forgejo documents it as best-effort, not an atomic lock or exactly-once guarantee.
Native branch/PR conflict handling and immutable expected-head publication must
independently protect writes. New eligible human commits supersede old work.

A deliberately requested native **rerun** has a new execution ID and fresh bounded
round budget, while retaining the same publication key and looking up already
published work before any write. It starts with fresh CLI state; if the issue PR
already exists and PR automation
is off, return that result without creating more work. With review enabled it may
resume only from the verified published head through a new review, never replay an
old diff. This is a visible human-requested retry, not automatic loop continuation. Generated
workflows do not automatically rerun failed/exhausted attempts. Duplicate event
delivery is suppressed and gets no new budget. A fresh manual logical generation is
explicitly different from retry and may create additional linked work only as selected.

Prove the exact native event/claim/publication mechanism before enabling workflows.
Fake-command tests cover PR-created delivery while the issue job still runs, final
bot synchronize events, duplicate delivery, ambiguous creation response, explicit
rerun budget, concurrent human pushes/base changes, cancellation during publication,
PR attempts to replace trusted command scripts and mutation of a candidate after
checks. These are bounded integration requirements, not a generic reconciliation
service.

### Trust, credentials and the concrete runtime gap

**Source investigation, not runtime proof:** the current Fedora 44 package source
selects runner **12.13.2** (this is not a read of any retained VM's installed RPM).
Its `internal/pkg/config/config.example.yaml` distinguishes the client engine
endpoint from `container.docker_host`: a URL in the latter **mounts the engine
socket into the job**; `-`/empty suppress that mount. A future rootless candidate
must keep the socket outside command jobs, not expose it to make OCI execution work.
The runner's `act/jobparser/model.go` explicitly says workflow/job `permissions:`
is unsupported. Forgejo **15.0.7** `services/actions/secret.go` supplies the task
credential as `GITHUB_TOKEN`, `GITEA_TOKEN` and `FORGEJO_TOKEN`; its API
`routers/api/v1/permissions/repo_access.go` assigns same-repository non-fork tasks
write access (fork-PR tasks read access). Consequently, rootless OCI execution alone,
secret-name selection, sequential steps or a GitHub-style permissions stanza cannot
establish the selected read-only-command/trusted-publisher boundary. This is not a
reason to fork Forgejo or weaken the boundary. The planned trusted launcher/publisher
outside the command sandbox still needs implementation and native proof, together
with run-owned terminals. No OCI label or automatic workflow was enabled by this
investigation. Sources/evidence: `.artifacts/runner-isolation-a741c65/`,
[Fedora spec](https://src.fedoraproject.org/rpms/forgejo-runner/raw/f44/f/forgejo-runner.spec),
[runner 12.13.2 source](https://code.forgejo.org/forgejo/runner/src/tag/v12.13.2),
[Forgejo token injection](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/services/actions/secret.go),
[Forgejo task repository authority](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/permissions/repo_access.go).

The current Soda Forgejo runner accepts only `name:host` labels in
[`internal/runners/model.go`](../internal/runners/model.go). Its
[configuration](../internal/runners/native_create.go) and
[unit](../appliance/services/soda-runner@.service) do not provision a per-runner
Podman engine. Existing listener or project-runtime evidence does not prove AI job
isolation. Upstream's [runner configuration](https://forgejo.org/docs/v15.0/admin/actions/configuration/)
supports OCI jobs via the `docker` label type, including Podman; that spelling
does not require selecting Docker as the appliance engine.

Investigate one dedicated rootless Podman runner candidate: per-runner storage,
private engine endpoint, required subordinate IDs/runtime directories, native
service lifecycle and compatible systemd confinement. No host root socket in jobs,
no access to marketplace/project state and no weakening existing runner units by
default. Inspect the exact packaged runner and Podman source before implementing
their configuration. Native UID/cgroup/network/cancellation proof is required;
do not assume changing a label makes the hardened existing unit compatible.

Trusted-contributor automatic execution is the proposed first scope. Establish
trust through Forgejo's current permissions and applicable event policy, not
repository-name matching or a caller-provided actor. Public issue text remains
untrusted even when an event can be scheduled. Untrusted/fork execution requires
an explicit supported approval path; never silently give it privileged credentials.

Read/review and write/publication credentials need an explicit verified boundary.
The publisher must execute outside the command sandbox, accept only the bound
repository/attempt/revision and validated result, and never execute repository code
with its write credential. Handoff a bounded immutable, content-addressed Git
commit/tree or patch plus expected parent and observed check receipt, never a path
to the live command checkout. Quiesce command descendants for the snapshot, validate
paths/object bounds, and publish from a clean trusted Git context with repository
hooks/configured credential helpers disabled. Retain the original command job's
checkout/state directory for subsequent fixes, but never mount it into the publisher.
Initial issue publication uses the same immutable handoff as final fixes. Merely
removing a secret from a child process environment
is insufficient when the child can read its parent or later job steps. Maintaining
the one-job command state while publishing the initial PR is a concrete integration
gate: prove a bounded trusted publication path using native interfaces, rather than
claiming that sequential steps or a separate YAML job automatically provide it.
The [v15 PR security guidance](https://forgejo.org/docs/v15.0/admin/actions/security/)
explains why checking out PR code in a privileged `pull_request_target` workflow is
unsafe. Separate YAML jobs alone do not prove a read-only agent: inspect the
automatic token and secrets actually supplied by the selected server/runner.
Do not assume GitHub's `permissions:` behavior works on Forgejo. Resolve that
contract before enabling publication; no downstream Forgejo patch is an option.

Use dedicated repository automation credentials, never Soda OAuth grants, operator
setup credentials or a developer's existing CLI login. Keep event data out of shell
interpolation, private material out of argv/logs/artifacts, and agent outputs as
untrusted data. Retain bounded run results in native Actions. Cleanup may remove
only exact attempt-owned temporary resources, never a retained project or service.

## 4. Desktop workspaces

**Product proposal:** Spaces presents Terminal and Desktop views of an authorized
workspace. Desktop opens the actual remote graphical session, with display,
keyboard and pointer transport. The same desktop can move between the full Spaces
page and repository drawer, maximize, and reconnect while the user browses Forgejo.
A thumbnail, terminal emulator or app launcher alone does not provide this feature.

The user selected Linux desktops: Rocky KDE and Fedora KDE, with both GNOME
variants deferred. Headless Rocky and Fedora Server are the terminal-only choices.
The [Project OS profile contract](project-os.md#selected-environment-profiles) owns
this matrix and creation behavior. Start desktop implementation with Fedora KDE,
where the Codex Linux preview has an explicitly supported distribution.

Desktop is an additional access surface for the same Project OS. Its native account,
HOME, checkouts, shared mise/tools, package state, services and permissions follow
[the existing foundation](project-os.md#one-foundation-for-every-profile). The
CoreOS host remains headless. The first investigation targets the existing project
runtime; a concrete display/session or kernel requirement must justify any proposed
runtime change. Linux/KDE selection does not select QEMU/KVM, a second machine per
project, or a new guest provisioning product.

### Verified vendor constraints — 2026-09-10

| Application | Linux desktop availability | Built-in computer use |
| --- | --- | --- |
| OpenAI desktop app with Codex | Preview on supported Ubuntu, Debian and Fedora desktop releases, x64/ARM64 | Documented for macOS and Windows; absent from the Linux preview |
| Claude Desktop with Claude Code | Beta on Ubuntu 22.04+ and Debian 12+, x64/ARM64 | Documented for macOS and Windows; absent from the Linux beta |

Sources: [OpenAI Linux desktop](https://learn.chatgpt.com/docs/linux/linux-app),
[OpenAI computer use](https://learn.chatgpt.com/docs/computer-use),
[Claude Linux desktop](https://code.claude.com/docs/en/desktop-linux) and
[Claude computer use](https://support.claude.com/en/articles/14128542-let-claude-use-your-computer-in-cowork).
The user-linked [preview announcement](https://community.openai.com/t/codex-in-chatgpt-desktop-app-for-linux-is-now-in-preview/1390027)
prompted this Linux selection. The official platform guide is the compatibility
authority and lists Fedora 43/44. Rocky is not on that supported list;
Claude's Linux desktop currently supports neither selected distribution. Keep
those app-specific limits visible and verify actual packaging/runtime compatibility.
OpenAI's former Codex app documentation currently redirects to its desktop app
guide with Codex mode. Recheck platform-specific pages before packaging.

Linux is the selected direction despite the current computer-use gap. Ship actual
GUI access and supported app/browser workflows, and add native computer use when
the selected app and Linux environment support it. Do not advertise missing
capabilities or make them a prerequisite to the Linux desktop feature. Windows
is outside the selected desktop direction. Reconnect preserves the same desktop
within its finite lifetime; the canonical session contract below owns Lock/unlock
and power-action behavior.

Claude Cowork on Linux is an optional app feature with its own QEMU/KVM and device
requirements, on distributions outside the selected matrix. It neither establishes
a Project OS runtime blocker nor selects a Soda VM. Do not pass host devices into
current projects to make it work. An API-based computer-use
integration would be a separate agent integration, not proof that either vendor's
Linux desktop app gained its native feature.

Running a desktop app on the user's laptop against a remote checkout is another
workflow, but remote shell access alone does not move screen capture/input into
Soda. The UI must identify the actual computer and desktop under control. Desktop
apps must run as the intended project account for the streamed project desktop.

### Native integration, transport and lifetime

The [Project OS desktop contract](project-os.md#desktop-session-and-access-boundary)
is canonical for the login-quality session, exact identity, locked Linux accounts,
finite viewer/desktop lifetime, credential store and scoped cleanup. The selected
first surface is one private desktop per project account, one virtual display and
one keyboard/pointer controller at a time. Duplicate views from the creating Soda
sign-in context may observe that same display; another account or sign-in cannot
adopt it. Do not add cross-user screen-sharing roles or a separate desktop identity
system.

Prove the current-runtime candidate in this order: per-account Plasma Wayland
startup, existing-session KRFB capture/input, private RFB → authenticated WebSocket →
noVNC in the shared Lit view. This is one investigation candidate, not existing
transport support. Fedora's supported Plasma session is Wayland; an X11-only desktop
server cannot silently change that selection. Exact package compatibility, private
transport credentials and display integration must be established before declaring
the candidate workable. The [profile guide](project-os.md#selected-environment-profiles)
records upstream constraints and why remote access does not itself supply a headless
session. A failed probe needs an actual cause; it does not automatically select a VM.

The first desktop includes software rendering, resize, keyboard/pointer and explicit
directional clipboard. Existing SSH/SCP/SFTP handles files. Audio, cameras/microphones,
GPU/USB passthrough, printing, multiple monitors and browser file transfer are outside
this first surface. Keep viewer endpoints private, authenticate each attachment and
resolve all native targets server-side; a repository ID is not permission to forward
to an arbitrary host/port. Browser Origin checks and current authority apply to the
WebSocket too. Transport credentials remain ephemeral runtime-local inputs, never browser
URL/query/JavaScript payloads, logs or evidence. The intended account and project/host
root may inspect native credentials under the documented trust boundary; private
listener confinement and revocation must work independently of hiding that password.

One Lit Desktop view serves both Spaces and the repository drawer. An explicit
**Start desktop** creates the account-owned session; **Open** attaches to an existing
ID. Hide/navigation detaches; last-viewer retention, the hard cap and logout/expiry
follow the separate desktop contract. **End desktop** terminates only that graphical
scope, preserving the user manager, independent tmux/SSH and project services. **Stop project** interrupts all project work and retains durable state;
Restart cannot restore process memory. Terminal End and AI Cancel keep their distinct
owners. Missing native integration refuses with an actionable unavailable state;
opening the viewer never installs packages, joins or starts the project as a repair.

GUI tools, terminal and SSH edit the same real checkout and use the same original
HOME/groups/shared mise/tools. The [batteries-included requirement](project-os.md#batteries-included-by-default)
includes a usable browser/editor/file manager, fonts, clipboard and credential-store
integration. Native desktop Lock/unlock and password-locked-account keyring behavior
must work under the canonical account contract; don't hide a trapping lock screen or
pretend Forgejo OAuth is a Linux password. Project sudo/root retains administrative
access to project state.

### Personal GUI apps and future computer use

Personal desktop apps use that user's project-local configuration and credentials.
Issue/PR commands use a dedicated job checkout and credentials; they cannot borrow an
existing personal desktop. Launching a GUI app does not provide an event-to-prompt
API, resumable conversation or validated review result. The current issue/PR feature
is the command contract above and ships independently of desktop availability.

If a vendor later supports computer use on the selected Linux profile, recheck and
prove the exact app version/account/session before exposing that capability. Identify
the actual remote project desktop under control. A laptop app connected to remote
files still controls its own supported computer-use target; SSH does not relocate
screen capture and input. An API-driven screen agent is a different integration,
not a substitute claim about the vendor desktop app.

Future agent-controlled desktops need explicit agent stop/pause before human input.
A read-only browser view cannot stop an agent's native input. If the app has no
verified control-transfer interface, use its real stop control and state the limit.
Cross-user sharing and graphical issue/PR automation remain outside the first
personal desktop surface; they must not become hidden prerequisites for KDE.

### Desktop completion criteria

Extend the existing [native validation](native-validation.md#kde-desktop-profile-checks)
with an actual GUI task as the original account, two-account display isolation,
correct shared files/tools, one-controller duplicate views, page/drawer navigation,
detach/reattach, Lock/unlock, scoped End/logout/expiry and project Stop/Start
persistence. Prove baseline desktop tasks without package repair and each advertised
app's real installation/sign-in with separately authorized private fixture inputs.
Advertise the OpenAI GUI only on an exact supported Fedora KDE/version/architecture
combination after proof; Claude Desktop remains unavailable on the currently selected
Rocky/Fedora profiles. CLI support is a separate capability.

Do not count synthetic viewer tests as native session/cleanup proof. Test each claimed
profile and architecture's affected behavior; unavailable Linux computer use stays
unavailable, and desktop acceptance does not imply it. No retained VM/project is
repurposed by this proposal. Only the currently supported profiles appear executable
in Create; feasibility research and a populated dropdown are not profile acceptance.

## Marketplace and AI implementation sequence

The [leading extension order](sodaspaces-plan.md#extension-order-and-dependency-boundaries)
owns integration with Project OS and Spaces. These are feature-specific completion
steps, not a replacement foundation or a dependency on GUI availability. Marketplace
and AI can proceed independently; the numbering groups their own completion steps. Run images
reuse applicable project-owned tooling/packaging; an unattended job still needs its
own verified isolation, credentials and execution owner. No duplicate account, tool
installer or project-lifecycle framework is implied.

1. Carry the explicit first candidates above through exact upstream runtime, token
   and UI extension inspection; any changed placement/trust decision revises that
   candidate before implementation, rather than adding a dormant alternative.
2. Implement the Services catalog, operator API/native helper, persistent recipes,
   usable ingress and native/Lit UI. Complete all three app journeys; do not count
   a read-only catalog or generated unit files as a working marketplace.
3. Add isolated container execution to runner configuration, provisioning,
   lifecycle, staging and focused tests. Preserve existing host runners and Cockpit
   Runners until its separately selected replacement works. Tailnet stays in Cockpit.
4. Implement repository AI setup, workflow integration, live run terminals in the
   shared Spaces/drawer UI and issue-to-PR publication. Prove background execution
   with no browser, authorized attachment and view-only navigation. Complete a
   fake-agent local journey, then a separately scoped real provider run.
5. Add PR review, bounded fixes, exact-revision publication, fork handling and
   duplicate-event prevention with the same resolver command/attempt state. Test
   zero rounds, independent event toggles, early success, exhaustion, malformed
   output, failed checks, view closure versus cancellation, retry identity and
   concurrent human commits as observable outcomes.
6. Validate native persistence for services and native isolation/cleanup for AI
   jobs on an explicitly authorized fixture. Cover operator/member denial, secret
   isolation, image/pull/start failures, usable endpoints and actual provider results.
   Update the handoff with exact candidate/evidence before separately approved delivery.

Source/build tests, native fixture provisioning, provider registration, paid agent
runs and retained-appliance deployment are distinct effects. This request selects
feature work; it does not supply a deployment target or private provider inputs.
The [leading plan](sodaspaces-plan.md) keeps existing work and authority boundaries.
No builds, tests, provider jobs, service installs or deployment occurred during this
proposal's research.
