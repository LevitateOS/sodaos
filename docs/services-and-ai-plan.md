# Services marketplace and repository AI automation

The user requested three additions: a simple Podman services marketplace, an AI
issue resolver, and AI PR review with optional fixes and a configurable loop count.
These are requested product work. This document proposes their implementation;
**none of these features is implemented or validated yet**.

Two product choices are pending: appliance-wide versus project-local marketplace
installation, and automatic trusted-contributor runs versus a maintainer trigger.
The candidate below assumes **appliance-wide, operator-managed apps** and
**automatic runs for trusted contributors after repository opt-in**. Do not treat
those proposed defaults as a user answer. Existing projects, runner registrations,
credentials, services and retained validation fixtures remain unchanged.

## Containers, pods and the three lifetimes

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
| AI run | One issue attempt or one PR review/fix attempt; repository policy and Forgejo Actions | Fresh runner job container and checkout, with optional job services |

AI runs must not use a developer's retained project, home, credentials or terminal.
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
| Agent command | Claude Code, Codex, pi, or a repository-maintained noninteractive command |
| Reviewer and fixer | May use the same command or separately configured commands/models |
| Setup and tests | Repository-owned commands executed inside the job sandbox |
| Credentials | Named native Forgejo secret references; no values in versioned configuration |
| Maximum fix rounds | Integer; zero means review only; proposed default two, upper bound five |
| Run timeout | Finite overall deadline, covering setup, agents, tests and publication |
| Output | Native issue-linked PR or review, test result and Forgejo Actions run link |

The agent-command presets must be checked against the selected tools' actual
noninteractive interfaces and credential support before claiming compatibility.
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

### Issue journey

1. Resolve the repository, issue and trusted configuration revision using native
   authority. An issue body is task data, never a shell command or authorization.
2. Create a fresh checkout of the selected default-branch commit in a job container.
3. Run setup, the fixer and required tests within the configured deadline.
4. Publish a new automation branch and issue-linked PR, including actual test
   outcomes and the run link. Do not push directly to the default branch or close
   an issue merely because an agent says it succeeded.

### PR journey and exact loop semantics

1. Capture the PR's exact head and base commits and review the proposed changes.
2. With fixes disabled, publish the review and finish.
3. With fixes enabled and actionable findings, run one fix round, run tests and
   review the resulting tree again. Each fixer invocation consumes one round.
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

Bot publication must not reset the round budget by triggering another run. Use
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

## Implementation sequence and completion criteria

1. Settle marketplace placement and trigger policy. Inspect exact upstream runtime,
   token and UI extension contracts; retain one candidate for each feature.
2. Implement the Services catalog, operator API/native helper, persistent recipes,
   usable ingress and native/Lit UI. Complete all three app journeys; do not count
   a read-only catalog or generated unit files as a working marketplace.
3. Add isolated container execution to runner configuration, provisioning,
   lifecycle, staging and focused tests. Preserve existing host runners and Cockpit
   Runners until its separately selected replacement works. Tailnet stays in Cockpit.
4. Implement repository AI setup, workflow integration and issue-to-PR publication.
   Complete a fake-agent local journey, then a separately scoped real provider run.
5. Add PR review, bounded fixes, stale-head refusal, fork handling and recursion
   prevention. Test zero rounds, early success, exhaustion, malformed output,
   cancellation and concurrent human commits as observable outcomes.
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
