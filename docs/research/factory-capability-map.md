# Factory capability map

This source assessment maps the agreed Soda factory to existing implementation,
required changes and unproved integration boundaries. Most missing product behavior
belongs in Soda: automatic readiness, scheduling, shared-Project execution and
factory views. Existing factory, Project and broker code supplies useful primitives.
Native forge collaboration remains upstream-owned; Fountain supplies generic
extension access, not factory policy or a Soda-specific identity system.

This is a non-normative snapshot, not an implementation sequence or a progress
tracker. Requirements remain in the [product contract](../product/overview.md),
[environment model](../product/projects.md), [Spaces](../product/spaces.md),
[feature disposition](../product/scope.md) and [trust model](../architecture/trust.md).
Do not infer installed support from a source file or an authored test.

## Evidence scope

Inspected on September 30, 2026:

- SodaOS source at `6955952b31489f9e315ed7c63019bd69731e4bd4`.
- Fountain source at `c22b3543f6a1f88ede70ed3f046576b725430934` in the sibling
  `forgejo-ext` checkout. Fountain source links below use that checkout layout.
- Existing task-scoped development receipts where their narrower scope matters;
  no new runtime tests, builds, provider executions or installations were performed.

[Soda's module](../../go.mod) requires `forgejo.org/extension-sdk v0.0.0` with a
local `../forgejo-ext/sdk` replacement. This assessment therefore uses the SDK in
the inspected Fountain checkout. The [candidate producer](../../tools/soda-build/main.go)
requires a clean canonical Fountain checkout and records its exact selected
revision; the source revision above is not evidence of what an appliance has
installed. Fountain's extension README understates current streaming support;
the implementation cited below is decisive.

In the tables, **retain** means an existing capability serves the target purpose;
**adapt** means existing code implements a narrower or different behavior;
**missing** identifies absent product behavior; **unproved** identifies a native
boundary whose required behavior has not been demonstrated. An item can need both
adaptation and proof. **Deferred** capabilities do not gate this factory.

Owner means responsibility for the change: **Soda** owns product policy,
orchestration, Project integration and the provider broker; **Fountain** owns
generic extension contracts; **upstream** owns native Forgejo, Git, Linux,
Podman/systemd, provider CLI and other mature service behavior. Soda packaging an
upstream component does not make it a Soda implementation.

## Fountain and native forge capabilities

| Requirement | Current implementation and evidence | Necessary change or actual gap | Owner |
| --- | --- | --- | --- |
| Native login and browser authority | Fountain [admission/proxy](../../../forgejo-ext/routers/web/extensions/proxy.go) and [callbacks](../../../forgejo-ext/routers/web/extensions/callback.go) bind the extension process, native session and current actor/repository. [SDK contract](../../../forgejo-ext/sdk/contract.go) offers actor, repository, owned-repository search, organization-ownership and public-key reads. | **Retain / consume:** use current native authority and Soda's own product checks. These SDK reads are not issue/PR/write APIs. No parallel login, borrowed cookie or synthetic session is needed for the browser path. | Fountain generic admission/SDK; upstream Forgejo identity; Soda caller policy. |
| Persistent page and panel host | Fountain [workspace host](../../../forgejo-ext/web_src/js/features/extensions.js) preserves contribution roots alongside native navigation and rechecks session generation before revealing cached private state. | **Retain:** Soda adds factory views and shares its Lit workspace owner. No replacement host or Soda navigation shell is required. | Fountain generic browser lifecycle; Soda contributions. |
| Streams and authority loss | Fountain [WebSocket relay](../../../forgejo-ext/routers/web/extensions/websocket.go) implements origin checks, bounded relay, session/process cancellation and a 15-second native authority heartbeat. Soda's [terminal API](../../internal/web/api/extension_terminal.go) adds member/session checks. | **Retain / adapt:** use this generic transport for correctly authorized factory observation. Host heartbeat does not re-evaluate Soda's entire product/contribution policy; Soda must enforce its run/view/control authority continuously. Existing human streams do not prove factory streams. | Fountain transport/native checks; Soda execution and viewer authorization. |
| Browser-independent forge operations | Fountain service IPC forwards an existing admission; [callback handling](../../../forgejo-ext/routers/web/extensions/callback.go) still requires live request/session authority. Soda [factory configuration](../../internal/factory/control/config.go) already uses separate human/publisher/reviewer native actor tokens. | **Native capability exists; target binding unproved:** ordinary unattended actor credentials are legitimate. A generic extension-owned durable delegation API is absent, but that alone does not establish it must be built. Verify whether native principals and Soda's grant lifecycle satisfy the required mutation authority; if not, resolve that precise generic shortcoming in Fountain. | Soda sponsorship/task policy; upstream native credentials; Fountain only for a demonstrated generic extension-authority gap. |
| Credential scope and revocation | Upstream [token creation](../../../forgejo-ext/routers/api/v1/user/app.go) supports selected repository resources; [authorization reduction](../../../forgejo-ext/services/authz/access_token.go) narrows access; [token lookup/deletion](../../../forgejo-ext/models/auth/access_token.go) consults current stored state. | **Retain / strengthen consumption:** the old factory accepts token files without proving their recommended scope. Repository restriction does not encode an attempt, PR/head or sponsoring-human relation. Target grants and revocation cannot be inferred from possession of a bot token. | Upstream token enforcement; Soda configuration/authority checks; generic mutation-bound delegation if existing native authority is insufficient. |
| Dependencies and clarification records | Native [dependency APIs](../../../forgejo-ext/routers/api/v1/repo/issue_dependency.go) read dependency/block relations and add/remove them; [dependency storage](../../../forgejo-ext/models/issues/dependency.go) rejects immediate two-node cycles. Soda's [work client](../../internal/forgejo/work.go) reads issues and writes comments but has no dependency caller. | **Missing Soda consumption, not missing forge storage:** evaluate required outcomes, accepted answers and full graph cycles. Inaccessible blockers can be returned as hidden/confidential placeholders; missing details are not evidence of resolution. The native representation of maintainer-confirmed outcomes/answers still needs to be selected. | Upstream records/access; Soda readiness and accepted-input semantics. |
| Issue and PR change delivery | Native [notifier](../../../forgejo-ext/services/webhook/notifier.go), [webhook preparation](../../../forgejo-ext/services/webhook/webhook.go) and [delivery](../../../forgejo-ext/services/webhook/deliver.go) persist and dispatch webhook tasks, recover unsent tasks and support replay. | **Missing Soda intake:** consume authenticated native observations with authoritative rereads. SDK event subscription is absent, but native webhooks already exist. Do not promise automatic retries after failed delivery, complete dependency-change events or exactly-once execution; dependency-event coverage needs verification. | Upstream webhook machinery; Soda intake, deduplication and bounded reconciliation. |
| PRs, reviews and Actions | Native APIs already serve Soda's [publication and review caller](../../internal/factory/control/publication.go) and [Actions/review reads](../../internal/forgejo/work.go). | **Retain / adapt Soda:** apply the target accepted inputs, complete required-check set and role policy. SDK parity does not justify duplicating native collaboration or CI. | Upstream Forgejo operations; Soda orchestration/evidence. |
| Conditional merge against verified state | Native [merge form](../../../forgejo-ext/services/forms/repo_form.go) accepts `head_commit_id`; [preparation](../../../forgejo-ext/services/pull/merge_prepare.go) checks the fetched head; [merge](../../../forgejo-ext/services/pull/merge.go) serializes the PR and uses a normal push. [Protected-branch hook](../../../forgejo-ext/routers/private/hook_pre_receive.go) rechecks native permission, reviews and status. | **Confirmed contract gap:** no expected-base input rejects a base changed since Soda's evidence before merge preparation. There is also no expected Soda policy/sponsor revision binding. **Unproved:** current-head race behavior through final ref update and atomic sponsorship withdrawal. Preserve existing native checks; add Soda's absent merge caller only after these required guarantees have a supported native path. | Soda merge eligibility; upstream merge engine; Fountain carries any necessary generic conditional-operation or authority extension, never Soda factory policy. |

The existing publication lock and `live` check in
[Soda authority](../../internal/factory/control/authority.go) recheck the configured
human's permission, open issue and startup policy before a separate bot mutation.
That is useful enforcement, but it does not establish that the native mutation
rejects an intervening sponsor/policy withdrawal. The next fact to establish is
the smallest native authority contract that meets the target, not the design of
a new login, general grant framework or replacement event service.

## Factory lifecycle and operating policy

| Requirement | Current implementation and evidence | Necessary change or actual gap | Owner |
| --- | --- | --- | --- |
| Standing authorization and accepted objectives | [Admission](../../internal/factory/control/authority.go) checks a configured human against one private repository; [broker grants](../../internal/web/api/identity_grants.go) already separate provider sponsorship. | **Adapt / missing:** record and enforce repository enablement, operator capacity and provider sponsorship together; accept objectives only from current authorized humans or explicit adoption. Existing token configuration is not this complete policy. | Soda policy; native Forgejo permissions through supported authority. |
| Issue intake and duplicate handling | [Operator CLI](../../cmd/soda-factory/main.go) offers explicit `admit` and `run`; [factory store](../../internal/store/factory.go) records delivery, attempt and run identities. | **Adapt / missing:** automatic intake, durable input revisions and duplicate-safe reassessment. A manually supplied delivery ID does not implement event consumption or a background coordinator. | Soda; upstream event delivery or native reads. |
| Dependency and requirements readiness | Admission reads an open issue and base revision; the [fixed lifecycle](../../internal/factory/lifecycle.go) has no target readiness model. | **Missing:** dependency outcome evaluation, material-question blockers, accepted answers, cycles/unavailable evidence, blocked versus resource-queued states and reassessment only when relevant inputs change. Native issues remain the records; Soda evaluates eligibility. | Soda; upstream issue relations/comments and generic access where needed. |
| Prompt assembly and inspectable assignments | [Task assembly](../../internal/factory/control/execute.go) includes work, run, candidate, findings and CI; [launcher](../../internal/host/workspace/launch.go) tells the CLI to consume task JSON. Objective admission is title/body plus the recorded base and policy. | **Adapt:** assemble accepted requirements, authorized clarifications/dependency outcomes, approved-base instructions, environment inputs, exact diff/check evidence, role policy and blocker output. Record template/input identities; do not put secrets into prompts. | Soda. |
| Queue, capacity and spending rules | [Attempt types](../../internal/factory/types.go) set a three-hour deadline; [lifecycle](../../internal/factory/lifecycle.go) allows four executions and one correction; [workspace configuration](../../internal/host/workspace/config.go) bounds CPU, memory, PIDs and scratch. | **Adapt / missing:** implement the [selected limits](../product/overview.md#concurrency-and-usage-limits), oldest-ready ordering, cross-repository reservations, one attempt/session per repository, cumulative active time and rolling provider accounting. Existing resource bounds do not implement those scheduling/budget rules. | Soda on native resource controls; upstream providers enforce their own quotas. |
| Coding execution | [Workspace runtime](../../internal/host/workspace/runtime.go) creates a disposable rootless worker, imports a source bundle and launches a pinned CLI with recorded resources. | **Replace execution path:** launch bounded processes in the associated persistent Project; retain run identity and resource accounting. Shared-Project execution is **unproved**, not merely a different container flag. | Soda using Project OS and native process controls. |
| Candidate publication | [Publisher](../../internal/host/publish/publish.go) validates a candidate bundle and pushes its assigned branch; [controller](../../internal/factory/control/publication.go) creates a native PR. Publisher credentials stay outside agent execution. | **Retain / adapt:** preserve exact candidate, branch and repository checks while consuming the retained attempt checkout. Revalidate authority before each native write; avoid a replacement Soda Git-authentication service. | Soda publication policy; upstream Git and Forgejo operations. |
| Review and correction | [Verification](../../internal/factory/control/verification.go) launches a separate reviewer; [publication](../../internal/factory/control/publication.go) submits a commit-bound native review; findings feed one repair run. | **Adapt:** same Project container, separate role account and fresh independent Git metadata; coder owns fixes, followed by fresh review of every changed candidate within the selected correction allowance. | Soda role coordination; upstream native reviews. |
| Required CI evidence | Verification observes one configured Actions workflow for the candidate; [forge client](../../internal/forgejo/work.go) checks returned commit/workflow identity. Multiple evaluations currently require intervention. | **Adapt:** evaluate the complete required-check set, current candidate/base and changed verification policy. Missing, skipped or stale evidence cannot pass. Preserve native CI scheduling and use separately managed execution capacity. | Soda eligibility; upstream Forgejo Actions/checks and runners. |
| Automatic merge and dependent work | [Lifecycle completion](../../internal/factory/lifecycle.go) ends at readiness for human merge; the Soda work client has no merge operation. | **Missing:** conditional native merge, authoritative outcome confirmation and dependency reassessment. Native merge support and its exact stale-state guarantees must be assessed separately from Soda's absent caller. | Soda coordination; native merge enforcement, with Fountain changes only for a demonstrated generic shortcoming. |
| Pause, cancel, takeover, retry and interruption | [Reconciliation](../../internal/factory/control/reconcile.go) withdraws authority, cleans recorded resources and sends interrupted work to human intervention rather than resuming silently. | **Adapt / missing:** pause/block continuation with remaining allowances, explicit retry accounting, member-owned takeover, separate cleanup outcome and preserved attempt checkouts. Stop only the assigned run; never delete the shared Project as recovery. | Soda. |
| Factory observation and background service | Operator `status` and `report` expose the execution ledger; the factory entry point is a command workflow. | **Missing:** unattended coordinator, factory API/UI observation and authorized intervention integrated with Spaces. Operator commands may address the same factory; they must not preserve a second engine. | Soda; Fountain supplies generic hosting/transport. |

## Provider identity and execution credentials

| Requirement | Current implementation and evidence | Necessary change or actual gap | Owner |
| --- | --- | --- | --- |
| Subscription enrollment and custody | [Broker control](../../internal/identity/control/control.go), [Codex adapter](../../internal/identity/codex/) and [Muse adapter](../../internal/identity/muse/) implement native enrollment, protected custody and provider-specific behavior. | **Retain:** subscription-only selection, explicit account ownership, no silent account/provider or paid fallback. Verify the selected supported CLI route for the actual execution; broker support is not proof of every harness path. | Soda broker; upstream provider authentication and CLI. |
| Grants, leases and provider concurrency | [Lease control](../../internal/identity/control/lease.go) and private admin/runtime transports bind delegated execution; Codex uses serialized streams while Muse permits independent leases. | **Adapt:** connect sponsorship to target repository/Project roles and enforce the factory's aggregate usage reservations. Keep provider ownership, native publishing actor and run identity distinct. | Soda. |
| Supported factory harnesses | [Factory credential request](../../internal/factory/control/credential.go) selects Codex; [workspace launcher](../../internal/host/workspace/launch.go) runs it. [Muse workspace support](../../internal/host/workspace/muse.go) is an optional tool/connection path inside that worker. | **Gap:** there is no top-level Muse factory-agent execution path. If Muse is selected for a coding/review role, its launch, result and termination contract needs implementation and native proof; existing Muse enrollment alone does not supply it. | Soda harness integration; upstream CLI behavior. |
| Credentials inside a shared Project | [Factory capture](../../internal/host/workspace/credential.go) pauses and kills the entire worker before returning credential state. [Human terminal identity](../../internal/host/terminal/identity.go) demonstrates narrower managed execution binding. | **Adapt / unproved:** attest the actual role, run, checkout and process generation; capture updated credentials and stop all descendants without interrupting the Project. Keep credentials transient and broker/helper sockets outside the container. Human terminal binding is a starting point, not proof this factory boundary works. | Soda broker and privileged execution integration, using native Linux controls. |
| Human AI CLI access | [Human launch API](../../internal/web/api/identity_launch.go), [host identity bridge](../../internal/host/identity.go) and [account UI](../../frontend/spaces/soda-identity.ts) connect joined members to supported provider accounts. | **Retain:** a person may use an AI CLI in their own terminal; that does not create an attempt, acquire factory authority or permit reuse of their credentials by an agent role. | Soda; native provider CLI. |

## Projects and Spaces

| Requirement | Current implementation and evidence | Necessary change or actual gap | Owner |
| --- | --- | --- | --- |
| One repository and persistent Project | [Creation](../../internal/web/api/environments_create.go) reserves one environment and supports explicit human-owner creation; [profiles](../../internal/project/profile.go) select Rocky headless. | **Adapt:** owner-authorized automatic creation/start or reuse of that reservation. Organization-owned creation remains unsupported by the target; incomplete provisioning needs intervention, not a second worker or destructive replacement. | Soda using native Project OS/container lifecycle. |
| Human membership and agent accounts | [Join](../../internal/web/api/environments_join.go) confirms native account creation through [project-account](../../project-os/rootfs/usr/libexec/soda/project-account); current code-write authority gates human execution. | **Retain human Join; missing agent setup:** create the two nonhuman role accounts and fresh run state without granting membership, SSH or sudo. Native ownership and access to approved tools still need proof. | Soda integration; upstream Linux account/permission mechanisms. |
| Checkouts and worktrees | Human Git work is manual; the old factory [runtime](../../internal/host/workspace/runtime.go) imports fresh worker source. | **Missing allocation model:** preserve an attempt-owned coding checkout/branch, create fresh review Git state, and keep both separate from human clones. Use native Git/worktrees within the permitted ownership boundary. Prove source preparation and safe takeover of unfinished changes. | Soda allocation and handoff; upstream Git. |
| Correct development environment | [Project OS image](../../project-os/Containerfile) and [initialization](../../project-os/rootfs/usr/libexec/soda/project-init) supply accounts, Git, mise, CLI tools, tmux, OpenSSH and nested workloads. | **Adapt / unproved:** resolve approved repository tool/setup/service inputs and prepare assigned writable dependencies/test data before agents run. A base image is not proof a particular repository builds; no new manifest format or root access from issue text is implied. | Soda preparation; upstream mise, packages, native services and repository tooling. |
| Persistent roots and Project Stop/Start | [Lifecycle API](../../internal/web/api/lifecycle.go) starts/stops the existing environment with owner/operator checks; native Project OS owns persistent homes, tools, configuration and service volumes. | **Adapt:** factory admission hold, coordinated run termination/credential return and stale-binding rejection. Start clears only the Project hold. Prove human dirty work and unrelated data survive the combined path. | Soda lifecycle policy; upstream Podman/systemd/storage behavior. |
| Human terminals and external access | [Managed terminal contract](../reference/terminal.md), [terminal authority](../../internal/web/api/extension_terminal.go) and [access keys](../../internal/web/api/access_keys.go) provide member sessions and explicit public-key installation. OpenSSH supplies SSH/PTY/SCP/SFTP. | **Retain:** native tmux state, explicit End, stable member identities and existing access-loss rules. No private-key onboarding or automatic human offboarding expansion. Agent role accounts must not become public login accounts. | Soda authorization/supervision; upstream tmux/OpenSSH. |
| Spaces page, panel, tabs and splits | [Soda manifest](../../appliance/soda-extension/extension.json) declares three pages and one preferred panel in one extension package; [workspace](../../frontend/spaces/sodaspaces-workspace.ts) and [layout](../../frontend/spaces/sodaspaces-layout.ts) handle human session views. | **Retain / adapt:** show factory issue/attempt/run/candidate context beside human work. View hide/split/navigation must preserve the execution binding and must not start or cancel agents. | Soda UI; Fountain generic persistent contribution host. |
| Live factory sessions and controls | [Spaces inventory](../../internal/web/api/spaces.go) lists Projects and human terminal state; the existing terminal route is member-scoped. | **Missing / unproved:** real CLI output bound to the factory role/process; read-only factory viewing with current code-write authorization; pause/cancel/takeover through factory controls. Do not expose factory input through human terminal End/attach or promise durable transcript replay. | Soda session model, backend and UI; Fountain generic stream transport. |

## Supporting appliance capabilities

| Requirement | Current implementation and evidence | Necessary change or actual gap | Owner |
| --- | --- | --- | --- |
| Private LAN and project services | Native Project OS networking, [project service boundary](../guides/project-services.md) and [network architecture](../architecture/networking.md) provide project addresses and service ports. | **Retain / verify:** required callers need a real route and working endpoints; an observed address alone is insufficient. Assign agent service access/test data without giving it shared engine administration or human data ownership. | Soda configuration/integration; upstream Linux networking, Podman and service software. |
| Optional Tailnet | [Tailnet API](../../internal/web/api/tailnet.go), [host bridge](../../internal/host/tailnet.go) and [companion runtime](../../internal/host/tailnet/) implement operator policy and project companions. | **Retain / adapt:** consume native extension authority, keep enrollment opt-in and device identities separate. Factory readiness must work over LAN without requiring Tailnet. | Soda policy; upstream Tailscale. |
| Private HTTPS and native browser origin | [Operator setup](../guides/operator-setup.md) defines Caddy, configured origins and client trust; [installer setup](../../internal/installer/setup_linux.go) handles appliance configuration. | **Retain:** native session origin, local/supplied certificates and no purchased-domain prerequisite. Verify selected deployment routing/trust when qualifying it, not by building another login/proxy path. | Soda packaging/configuration; upstream Caddy and Forgejo. |
| Operator bootstrap and privileged helper | Installer setup records the configured operator; [host daemon](../../internal/host/daemon.go) admits fixed privileged operations. | **Retain / adapt:** add only required bounded Project execution operations after native proof. Operator authority remains distinct from repository and provider grants; no arbitrary host commands or mounted helper socket. | Soda. |
| CI capacity and local runners | [Runner API](../../internal/web/api/runners.go) refuses local creation while retaining observation/cleanup; [runner reference](../reference/runners.md) records the deferred execution boundary. | **Retain native CI consumption; defer local execution:** arrange separately managed Actions capacity. Runner OS and isolated local-job implementation are not prerequisites for the first factory and must not be replaced by shared host-account jobs. | Upstream Forgejo Actions/runner; Soda integration and eventual local capacity policy. |
| Cockpit and host diagnosis | [Cockpit configuration](../../appliance/config/cockpit.conf) and the [Cockpit guide](../development/cockpit.md) retain stock host administration. | **Retain:** native operator diagnosis/recovery. Do not recreate custom Cockpit Project pages or use host administration as an agent grant. | Upstream Cockpit; Soda packaging/branding. |
| Branding, avatars and console guidance | [Forgejo assets/templates](../../appliance/forgejo/), [avatar implementation](../../internal/avatar/) and [console entry](../../appliance/bin/soda-console-welcome) supply existing presentation. | **Retain / adapt:** keep attribution and supported image/extension branding; remove superseded native behavior copies during cutover. Presentation does not require a second Soda shell. | Soda assets; Fountain/upstream supported presentation surfaces. |
| Build, installation, update and recovery | [Candidate producer](../../tools/soda-build/main.go), [release packages](../../internal/release/) and [native candidate checks](../../scripts/check-native.sh) implement the existing delivery model. | **Retain / adapt:** package the selected exact Fountain and Soda sources and changed runtime; qualify affected native architectures and existing-state protection when shipping. No release build or production deployment is needed to establish this source map. | Soda release integration; upstream Fedora CoreOS, rpm-ostree and container tooling. |

## Retired and deferred paths

The [scope guide](../product/scope.md#retire) remains authoritative. The removed
OAuth login and Soda-managed Git mediation must stay removed. Replace the old
manual-admission/human-merge/disposable-stage implementation together with its
callers and fixtures; do not maintain it as a second factory. Normal upstream
credentials held by a separately authorized publisher are not the removed
developer credential-relay system.

Rocky headless is the only selected Project profile. Graphical/Fedora profiles,
Soda-provisioned local CI, extra provider adapters, fleet orchestration, conversation
snapshots, hostile tenancy, production deployment after merge and general archival
or destructive recovery machinery remain deferred or excluded as specified in the
scope guide. They are not hidden prerequisites for the retained features above.

## Evidence that still needs native verification

Source establishes useful starting points, not completion of the automatic loop.
Before an implementation plan depends on the target, resolve these bounded facts:

1. Native unattended authority and conditional merge satisfy the exact accepted
   repository, actor, revision and revocation conditions without a session workaround.
2. Two factory role accounts in a persistent Project can use the approved repository
   environment while preserving human state and separate writable Git administration.
3. Attestation, credential return and termination cover the assigned run and all
   descendants while unrelated Project processes remain running.
4. A live factory CLI can be observed without granting input, exposing credentials
   or tying execution lifetime to a browser attachment.
5. Stop/Start, interruption and human takeover preserve unfinished work, enforce
   the admission hold and reject stale process/lease bindings.

A retained native development receipt at Soda source `21122ccbf` covers Codex
`0.153.4` in the old disposable worker on Linux x86_64, including credential return,
whole-worker termination and delegation withdrawal. It does not prove shared-Project
execution, a primary Muse factory harness, installed appliance behavior or aarch64.
The earlier Fountain C03 receipt covers a bounded native x86_64 identity/preference
bridge at older source revisions, not full current browser, stream or appliance
qualification. Detailed receipts stay in task artifacts/history; neither receipt
justifies rerunning a full release build to answer a narrower integration question.
