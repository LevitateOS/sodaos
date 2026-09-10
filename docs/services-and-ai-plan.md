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
drawer, launched with predefined prompts and commands. This replaces the earlier
proposal's implication that Actions logs alone would be the user interface.

The user also requested graphical workspaces for the actual Codex/Claude desktop
apps and their computer-use features. The [desktop proposal](#4-desktop-workspaces)
below extends the product beyond terminal transport. It is an architecture proposal,
not an implemented VM backend or a change to existing Rocky projects.

For marketplace and AI automation, two product choices remain pending:
appliance-wide versus project-local installation, and automatic trusted-contributor
runs versus a maintainer trigger.
The candidate below assumes **appliance-wide, operator-managed apps** and
**automatic runs for trusted contributors after repository opt-in**. Do not treat
those proposed defaults as a user answer. Existing projects, runner registrations,
credentials, services and retained validation fixtures remain unchanged.

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
| Desktop workspace | Persistent desktop, apps and user data; separately authorized user or dedicated automation identity | Proposed optional VM guest with its own graphical session; not currently implemented |

Automatic AI runs must not commandeer a developer's existing shell, dirty checkout,
home or personal credentials. Sharing the Spaces UI does not require sharing those
resources or tying execution to one viewer's browser session.
Project service catalogs would instead need project authorization and the existing
nested engine; they are not interchangeable with an operator's host catalog.

## 1. Services marketplace

Start with a small shipped catalog containing **Adminer, Vaultwarden and Homepage**.
Each entry owns its description, upstream links/license, reviewed image reference,
supported architectures, typed configuration, persistent paths, ports and native
health observation. Resolve and record real image digests during packaging; no
fabricated pins, automatic dependency upgrades or unverified architecture claims.

Use ordinary [Quadlet files](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
for container, volume and optional pod definitions. Catalog presentation metadata
does not become a new repository workload language. App data must live outside
the replaceable container layer in explicit persistent storage. This is a separate
contract from retained project roots, which normal Start must never recreate.

The first complete user journey is:

1. Open Services and select an app.
2. Configure the instance name, app-specific settings and actual access endpoint.
3. Review the image, persistent storage and listening address, then Install.
4. See real download/start/failure state; Open appears only with a usable endpoint.
5. Inspect status and bounded logs, Stop and Start while retaining configuration
   and data. A restart or page refresh must not silently install a newer image.

Do not turn arbitrary image names, host paths, shell scripts or Podman flags into
browser-controlled host commands. The configured Soda operator authorizes actions
server-side. A native fixed-operation helper resolves catalog entries and installed
instance targets. Keep host state authoritative rather than running a second
service scheduler or storing a competing running/stopped truth in SQLite.

For the first catalog:

- **Adminer:** use the upstream container and a configured reachable database
  endpoint. Database authentication remains with the database; do not ingest saved
  database passwords into catalog metadata or attach unrelated database volumes.
- **Vaultwarden:** retain its data directory and require a working HTTPS origin
  for the complete browser journey. Native Vaultwarden owns vault accounts and
  encrypted data. Configuration/secret input and backup requirements must be
  reviewed before enabling a real instance. See its
  [deployment examples](https://github.com/dani-garcia/vaultwarden/wiki/Deployment-examples).
- **Homepage:** retain configuration, set the actual allowed host and use the
  selected version's authentication options. Static service links do not require
  giving Homepage the host engine socket. See its
  [installation requirements](https://gethomepage.dev/installation/).

Ingress is part of a usable installation, not a guessed link. Select private
reachable endpoints and TLS configuration explicitly; localhost on the appliance
is not localhost on the user's laptop. Keep third-party app cookies/content off
the authenticated Forgejo origin. Do not default to a public bind or wildcard host
validation. Updates, uninstall/data deletion and automatic discovery are separate
scope; Stop is sufficient to retain an installed service safely in this first version.

## 2. Issue resolver and 3. PR review/fix

Use one repository AI configuration experience with independent controls for the
two event types. Proposed fields:

| Setting | Behavior |
| --- | --- |
| Issue automation | Off, or resolve newly opened issues under the selected trigger policy |
| PR automation | Off, review, or review and fix; include new commits deliberately |
| Agent command and prompt | Claude Code, Codex, pi, or a repository-maintained command; launched once with predefined task input in a real terminal |
| Reviewer and fixer | Reuse the originating resolver for fixes; reviewer may use a separately configured command/model |
| Setup and tests | Repository-owned commands executed inside the job sandbox |
| Credentials | Named native Forgejo secret references; no values in versioned configuration |
| Maximum fix rounds | Integer; zero means review only; proposed default two, upper bound five |
| Run timeout | Finite overall deadline, covering setup, agents, tests and publication |
| Output | Live Spaces/drawer terminal, native issue-linked PR/review, test result and Forgejo Actions run link |

The agent-command presets must be checked against the selected tools' actual
prompt input, unattended execution, PTY and native session-resume interfaces and
credential support before claiming compatibility.
Allow ordinary repository scripts rather than preinstalling every AI tool into
every development project. Optional provider spending limits belong to the actual
provider/CLI integration; a loop count alone is not an exact monetary budget.

Keep the effective automation in ordinary `.forgejo/workflows/` files and native
Actions secrets/variables. The UI should prepare a reviewable change through
supported native Git/API workflows, handle concurrent edits and show unsupported
hand-edited workflows honestly. It must not silently overwrite an existing
workflow or maintain a second authoritative copy in Soda's database. Simply
downloading YAML or adding mock configuration does not complete this feature.

Forgejo owns events, dispatch, queueing, cancellation, job history and results.
Its [v15 workflow reference](https://forgejo.org/docs/v15.0/user/actions/reference/)
documents issue, PR and manual-dispatch events. Use those native interfaces before
considering a webhook receiver. Soda supplies local execution capacity and the
bounded agent integration, not a parallel queue or Git scheduler.

### Live AI terminals in Spaces and the drawer

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

The originating resolver keeps its checkout/branch and agent conversation through
the review/fix loop. Feed structured review findings to that same resolver. When
the CLI requires another process invocation, resume its verified native session
and working directory; do not equate "same executable" with "same conversation".
If continuation is unavailable, report that honestly instead of silently claiming
context preservation. The reviewer can have a separate terminal and clean review
context. A PR opened directly begins the same review/fix flow without an issue step.

Repository configuration still enables events, agent commands, prompts and limits.
Merely opening Spaces or an ordinary New terminal never starts automation.

### Issue journey

1. Resolve the repository, issue and trusted configuration revision using native
   authority. An issue body is task data, never a shell command or authorization.
2. Create a fresh checkout of the selected default-branch commit in a job container.
3. Run setup, the fixer and required tests within the configured deadline.
4. Publish a new automation branch and issue-linked PR, including actual test
   outcomes and the run link, then enter the configured PR review flow. Preserve
   the originating resolver context for subsequent findings. Do not push directly
   to the default branch or close an issue merely because an agent says it succeeded.

### PR journey and exact loop semantics

1. Capture the PR's exact head and base commits and review the proposed changes.
2. With fixes disabled, publish the review and finish.
3. With fixes enabled and actionable findings, send them to the originating
   resolver (or start the configured resolver for a directly submitted PR), run
   tests and review the resulting tree again. Each fix attempt consumes one round.
4. Stop on a clean review with passing required checks, the configured limit,
   timeout/cancellation, an agent/test infrastructure error, or no further change.
   Exhaustion or inconclusive output is not an approval.
5. Publish only against the still-current PR head. Recheck before writing. If it
   changed, mark the attempt superseded instead of overwriting someone else's work.

For example, a limit of two means **review → fix → tests → review → fix → tests →
final review**, with earlier exit when appropriate. A zero limit runs one review.
Structured review output needs validation; command exit zero alone is not a clean
review. Preserve the final diff, findings and actual check outcomes.

Proposed publication policy: append normal commits to a writable same-repository
PR branch only when repository configuration permits it. For a fork or otherwise
unwritable branch, offer a patch or separate follow-up PR; never force-push or infer
permission to write another repository. No automatic merge is proposed.

The initial resolver-created PR must start its first configured review. Subsequent
bot publications must not reset the round budget or spawn duplicate loops. Use
provider concurrency plus explicit attempt/head identity and verified automation
authorship; a bot-like username or issue text is not sufficient. New human commits
can supersede the old attempt. Retry must not duplicate a previously created PR.

### Trust, credentials and the concrete runtime gap

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

A Linux GUI can run in a container with a display server; a GUI does not inherently
require a VM. A VM supplies its own kernel and guest OS, which matters for native
Windows applications and OS-dependent features. The current CoreOS appliance can
remain headless. The proposed desktop runs inside its guest, not on the host's
display. Existing validation appliances being VMs does not mean Soda already
manages desktop guests.

### Verified vendor constraints — 2026-09-10

| Application | Linux desktop availability | Built-in computer use |
| --- | --- | --- |
| OpenAI desktop app with Codex | Preview on supported Ubuntu, Debian and Fedora desktop releases, x64/ARM64 | Documented for macOS and Windows; absent from the Linux preview |
| Claude Desktop with Claude Code | Beta on Ubuntu 22.04+ and Debian 12+, x64/ARM64 | Documented for macOS and Windows; absent from the Linux beta |

Sources: [OpenAI Linux desktop](https://learn.chatgpt.com/docs/linux/linux-app),
[OpenAI computer use](https://learn.chatgpt.com/docs/computer-use),
[Claude Linux desktop](https://code.claude.com/docs/en/desktop-linux) and
[Claude computer use](https://support.claude.com/en/articles/14128542-let-claude-use-your-computer-in-cowork).
OpenAI's former Codex app documentation currently redirects to its desktop app
guide with Codex mode. Recheck these platform-specific pages before packaging;
older Claude general desktop documentation still says Linux is unsupported.

Consequently, installing either GUI in a Linux VM would not currently deliver its
built-in computer use. **A Windows desktop VM is the proposed first compatibility
probe for the complete requested experience.** This is an inference from vendor
support, not Soda runtime evidence or a selected replacement for Linux projects.
OpenAI explicitly describes using a Windows VM to contain foreground computer use.
The app's target must stay on the active, unlocked guest desktop; browser viewer
disconnection must not lock or replace that desktop session. Claude's account/plan
requirements must also be checked with the intended account before a real probe.

Linux remains a useful candidate when the requirement is the GUI app, editors and
browser tools. Claude Cowork on Linux additionally hosts its own QEMU/KVM VM; inside
a desktop VM that would require nested virtualization. Do not silently pass host
devices into current project containers to make it work. An API-based computer-use
integration would be a separate agent integration, not proof that either vendor's
Linux desktop app gained its native feature.

Running a desktop app on the user's laptop against a remote checkout is another
workflow, but remote shell access alone does not move screen capture/input into
Soda. The UI must identify the actual computer and desktop under control. Desktop
apps must run in the intended guest for the proposed streamed-guest experience.

### One bounded candidate and its ownership

Investigate a QEMU/KVM guest with a private virtual-display endpoint and a browser
viewer integrated into the existing Lit workspace. QEMU's
[VNC display](https://www.qemu.org/docs/master/system/invocation.html) and the
[noVNC client](https://novnc.com/info.html) provide a candidate display/input path.
This is a compatibility investigation, not an instruction to install either or
build interchangeable runtime/transport backends. Review exact selected versions,
CoreOS packaging, guest installation inputs and native hardware support first.

Soda's operator provisions guest capacity; authorized users attach to their own
desktop or an explicitly shared automation desktop. Project membership must not
automatically expose another member's desktop, browser cookies or AI account.
Resolve exact guest/session targets server-side. Keep raw console endpoints private
and authorize each browser connection with the existing Soda authority; a generic
browser-controlled host/port proxy or reusable console password is not acceptable.
The existing fixed-operation project helper is not a VM command passthrough.

Preserve the guest disk, home, app state and checkout independently of the viewer.
Hide, navigation and disconnect detach the view. Explicit Stop interrupts guest
work while preserving storage; it is distinct from ending a terminal or cancelling
an AI attempt. Existing personal-terminal logout/retention rules still apply to
existing terminals. A desktop's access and lifetime need their own explicit contract.

Terminal and Desktop must identify the actual workspace and checkout. If an AI run
executes in a guest, its terminal and GUI need access to that same run's files.
Separate containers and guests do not automatically share a filesystem or resolver
conversation. Do not silently copy dirty work or mount retained project roots into
a guest to create that appearance.

Observation should not type into an agent-controlled desktop. Human takeover must
pause/release the agent's control before granting input; merely suppressing browser
input cannot stop a native GUI agent. If an app offers no verified pause/control
interface, use its real stop control and report the limitation. Guest credentials
and provider approvals stay with their intended account and desktop, separate from
repository automation credentials.

The existing issue/PR proposal still uses supported unattended command interfaces.
Launching a GUI app is not evidence of a supported event-to-prompt API, resumable
conversation or machine-readable review result. Inspect those contracts separately
before promising fully automatic GUI-based resolver/reviewer runs. Desktop access
can ship independently; neither an AI job nor a personal desktop should depend on
keeping the Forgejo page open.

### Desktop completion criteria

Before calling this supported, prove app installation/sign-in and a real GUI task
in the selected guest; exact screen/input targeting; browser detach/reconnect and
page/drawer navigation without desktop replacement; authorized observation/control
transfer; isolation between users; and explicit Stop/Start persistence. Include
guest lock/sleep, disconnect during computer use and provider approval handling.
Synthetic viewer tests alone cannot prove these behaviors. Guest provisioning,
private sign-in and provider use require a separately scoped target and inputs;
no retained VM or project is repurposed by this proposal. Validate each claimed
architecture natively before advertising support.

## Implementation sequence and completion criteria

1. Settle marketplace placement and trigger policy. Inspect exact upstream runtime,
   token and UI extension contracts; retain one candidate for each feature.
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
5. Add PR review, bounded fixes, stale-head refusal, fork handling and recursion
   prevention with continuation of the original resolver. Test zero rounds, early
   success, exhaustion, malformed output, view closure versus job cancellation,
   context preservation and concurrent human commits as observable outcomes.
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
