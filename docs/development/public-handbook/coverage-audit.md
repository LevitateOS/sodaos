# Public handbook coverage and usability audit

This report preserves the source investigation and findings from the handbook
coverage audit. A subsequent documentation pass revised all 41 public pages,
adding worked journeys, precise permissions/defaults and daily-task entrances.
Use the [current handbook task list](cutover.md) for remaining work; the findings,
page matrix, example priorities and counts below describe the audit baseline,
not 40 still-open tasks.

Inline source paths and line numbers retain that audit baseline. Active links
point to the current articles. The later owner-requested
[six-section order](proposed-tree.md) supersedes this report's initial advice
to retain seven sections; the findings and page IDs remain useful evidence.

This audit records documentation gaps and proposed remedies. Findings describe
the final product contract even where implementation is incomplete. Following
the owner's request, the seven product questions now have selected resolutions
in the existing remaining-work list; this report does not claim those features
are implemented.

The result is **40 findings: 28 P1 and 12 P2**, with **12 worked-example
priorities** and **seven product resolutions** with implementation or
qualification work tracked in the existing backlog.

## Scope and evidence

Audit date: **20 October 2026**. Baseline:
`e03280ad51351cab12d95cf49c065e8a9c7bb4ee`.

- Inventoried all **2,569 tracked repository paths**, including source,
  configuration, tests, assets and documentation.
- Reviewed **all 41 published handbook pages**. The maintainer README is excluded
  from that article count.
- Traced user-facing responsibilities through product contracts, owning
  references, UI controls, API authorization, host commands and lifecycle paths.
  Three specialist investigations covered Factory/native collaboration,
  Projects/identity/development, and installation/operations.
- Distinguished absent information, incomplete procedures, ambiguous wording,
  and unresolved product decisions. An existing paragraph or diagram is not
  evidence that a reader can complete the task.
- Preserved unrelated in-progress Factory source changes. Their presence means
  this is an audit of the recorded checkout and intended contracts, not a
  reproducible assertion about every behavior of the baseline commit.

This is a repository-wide **documentation coverage audit**, not a line-by-line
correctness review of every source function. Whole-tree inventory, targeted
source inspection and complete handbook review are different coverage claims.
No installed appliance, provider enrollment, Factory delivery, release upgrade
or restoration was exercised.

Authority comes from [documentation ownership](../../README.md#authority-rules),
the [product overview](../../product/overview.md), [scope](../../product/scope.md),
[Projects](../../product/projects.md), [Spaces](../../product/spaces.md) and
[trust](../../architecture/trust.md). Existing interfaces supply exact controls
and commands. Proposed designs and fixture defaults do not become public
requirements merely because code or an old guide mentions them.

### Whole-tree responsibility coverage

Counts below partition the tracked inventory, rather than estimating completion
percentages.

| Source family | Tracked paths | What was checked for handbook relevance |
| --- | ---: | --- |
| `cmd/` | 286 | Install/setup/activation, native identity and provider launch, Project terminals and maintenance, Factory support, host console, migrations and database maintenance. Public operator actions matter; internal service startup is not a tutorial requirement. |
| `internal/` | 688 | Factory coordination, native Forgejo adapters, Project authority, identity, retained store state, web/API controls, host/app/Tailnet boundaries. Acceptance probes, locks, JSON decoding and avatar rendering are supporting infrastructure. |
| `lib/` | 400 | Privileged host lifecycle, app runtime/snapshots, provider/terminal boundaries and release intake/build/image/delivery. Consumer release guarantees matter; release construction is maintainer documentation. |
| `frontend/` | 377 | Workspace/Project/terminal controls, provider delegation, Feed/library/activity/app views, Tailnet and native Forgejo forms. Native features should be oriented and linked, not copied into a second Forgejo manual. |
| `system/` | 84 | Actual Docker prerequisites/ports/volumes, host services, network configuration, Project userspace and app runtime integration. |
| `tools/` | 188 | Release/installed qualification and fixture tools, generators and repository checks. Their existence does not make fixture commands supported end-user recovery tools. |
| `scripts/` | 48 | Packaging, source checks, branding/capture and database integration support; user-relevant service/backup boundaries traced to their owners. |
| `tests/` | 150 | Evidence inventory and scenario coverage context; tests are not a source of new supported product capabilities or hardware minima. |
| `assets/` | 190 | Branding, portraits, fonts and provenance. No missing user task follows from an unused decorative asset. |
| `docs/` | 129 | Intended product, adopted boundaries, existing interface references/guides and every public article. |
| `output/` and repository-root configuration | 29 | Build/configuration outputs and repository tooling; no additional public workflow identified. |

## What users most need to understand

These concepts deserve short explanations at the point of action, with detail
in their existing owner pages.

| Essential understanding | Why it matters | Best public owner |
| --- | --- | --- |
| Repository, Project, checkout, terminal view and installed application have different lifetimes | Closing a view, stopping a Project and stopping an app have different effects; a merge does not deploy anything. | Product model; Projects; terminals; private applications |
| Native repository permission, Project membership, lifecycle administration, provider ownership and app-local identity are separate | Seeing work is not permission to execute it; Join does not grant host administration; a provider connection does not authorize Factory work. | People and access, with contextual task links |
| Normal qualifying issue creation can establish initial acceptance; later accepted inputs are exact | Users should not expect a mandatory manual start for every issue or assume editing accepted text cannot suspend work. | Factory authorization |
| Blocked requirements, a resource queue, active work and an uncertain effect need different responses | The right response may be an answer, capacity, patience, reconciliation or an authorized intervention. | Feed and Factory intervention |
| Plans, checkpoints, review reports, native checks, merge, issue closure and integrated verification prove different things | A plausible report or old green check is not delivery; delivered issues are not proof of a verified application. | Issue work; review and merge; application verification |
| Shared Projects are for trusted teams | Separate accounts/checkouts do not prevent Project root/wheel from reading member files and credentials. | Projects and shared tools |
| Attempt time, provider usage, intent allowance and upstream subscription quota are different limits | CI waits can consume active-attempt time, Codex can be busy across uses, and retries do not erase rolling or aggregate consumption. | Factory authorization and provider connections |
| Uncertain operations retain their identity and persistent state | Repeating Create, reinstalling, deleting a volume or borrowing Factory credentials can make recovery worse. | Relevant task's failure section; recovery |

## Priority and classification

**P1** means a common/core task is incomplete, a material authority/default is
missing, or a consequential operation lacks a complete supported procedure.
**P2** means a useful example, reference detail or discovery improvement.
Priority is not a claim of an exploitable code defect.

**Missing** means the subject is absent. **Thin** means the boundary is mentioned
but the reader cannot yet follow or interpret it reliably. **Ambiguous** means
existing wording can produce the wrong expectation. **Decision** means a writer
needs an authoritative product/interface choice before supplying exact steps.

Source line numbers below refer to the inspected baseline/working tree and may
move. Public owner filenames are resolved in the complete page matrix.

## Cross-cutting gaps

| ID | Priority / type | Missing or inadequate coverage | Concrete remedy and evidence |
| --- | --- | --- | --- |
| G01 | P1 · Thin | No cohesive worked first Factory result. The index gives a good six-stage map, but a reader still has to assemble setup, an actual request, acceptance, progress, review and the final result. | Expand **Plan new work** with one small worked request and link each transition to its owner. Show an explicit objective, accepted outcomes, a resolved assumption, native issue readback, a candidate/check and confirmed delivery; finish with the separate verification/deployment boundary. Evidence: `docs/public/10-Start-here/10-index.md:22`, `docs/public/40-Factory/10-plan-new-work.md:5`, `docs/product/overview.md:32`. Add a separate tutorial only if this makes the existing page unwieldy. |
| G02 | P1 · Thin | The browser first-connection task ends at a working shell. There is no continuous first human contribution from repository/Project selection through a personal checkout, a harmless change and native PR. | Connect **First connection → Collaboration → Connect and develop** with one tested sample repository journey and expected results. Preserve that Join neither clones source nor requires an AI provider or SSH key. Evidence: `docs/public/30-Set-up/50-first-connection.md:7`, `docs/public/60-Develop/10-collaboration.md:5`, `docs/product/projects.md`, `docs/guides/develop.md`. |
| G03 | P1 · Thin | The index has task links, but returning-user questions such as “What needs me?”, “Why is this waiting?” and “Where is my terminal?” are less prominent than the installation/build sequence. | Put concise contributor/request-author/operator entry choices and direct daily-task links near the top of **Start here**. Retain existing routes and chapter order initially. Evidence: `docs/public/10-Start-here/10-index.md:5`, `system/containers/extension/extension.json:7`, `docs/product/spaces.md`. This is a discoverability improvement, not a claim that navigation is absent. |
| G04 | P1 · Thin | Authority boundaries are scattered. A person must infer who can create/Join, Start/Stop, accept requirements, sponsor a provider, change policy or install an app. | Add a compact **action → eligible actor → prerequisite → owning procedure** table in **People and access**, with short definitions in Product model. Source: `docs/architecture/trust.md:13`, `docs/architecture/trust.md:232`, `docs/reference/credentials.md:95`, `internal/web/api/applications.go:172`. Use actual authority; do not turn “admin” into a universal role. |

## Factory, Feed and knowledge gaps

| ID | Priority / type | Missing or inadequate coverage | Concrete remedy and evidence |
| --- | --- | --- | --- |
| F01 | P1 · Ambiguous | “Ensure acceptance” can sound like a separate approval for every issue. “Material change” is too vague for exact accepted-input invalidation. | In **Factory authorization**, distinguish qualifying human creation, validated task adoption from an already accepted intent, and explicit adoption of other imported/outsider/arbitrary Factory-created or unverified issues. Generated-task adoption binds exact native outputs to existing intent authority; it is not a new human approval per task. Show which changed title/body, selected comment or dependency revision invalidates acceptance, including the bounded generated-graph exception; unselected discussion is different. Initial title/body acceptance does not approve all dependency outcomes. Source: `docs/product/overview.md:190`, `:225`, `:262`; `internal/factory/control/acceptance_initial.go:56`; `internal/factory/control/intent_generation.go:181`; `docs/design/factory-worker-mcp.md:681`. |
| F02 | P1 · Missing | Useful numeric limits and the distinction between their clocks are absent. “Bounded” is not enough to configure or diagnose work. | Publish the initial defaults table and accounting explanation below in **Factory authorization**. Link verification and intervention limits to their task owners. Source: `docs/product/overview.md:325`, `:358`, `:369`, `:405`, `:413`. Do not copy the mistaken simplification that all time is charged only while an agent process runs. |
| F03 | P2 · Thin / Decision | Users cannot readily distinguish a provisional plan, an executable plan, planning allowance exhaustion and priority/fairness. The global-ceiling contradiction found during the audit is now corrected. | Explain that unfinished prerequisite outputs may permit a provisional outline, but coding requires a current nonprovisional plan. Explain scoped planning allowances and the recorded preference for candidates waiting at least a day without promising a start deadline. Q1 selects no global lifetime ceiling; qualify its remaining consumers in R11.2/R30.3/R48.2. Source: `docs/product/overview.md:626`, `:684`, `:694`, `:700`; `docs/reference/configuration.md:27`. |
| F04 | P1 · Thin | Review acceptance and required/optional finding dispositions are compressed into generic prose. | In **Review and merge**, show complete clear independent review versus corrected work requiring fresh independent review or permitted Jev acceptance of the verified correction set. Account for every required finding; optional advice may be declined with rationale. New candidates need fresh checks. Native human-approval requirements remain additional gates. Source: `docs/product/overview.md:442`, `:477`; `frontend/forgejo/templates/repo/settings/protected_branch.tmpl:88`. |
| F05 | P1 · Thin | Tasks, lanes, output contracts and checkpoints are abstract; readers cannot easily see why one task waits while independent work continues. | Add a three-task example to **Issue work and dependencies**: two independent roots and one consumer of a required output, with expected evidence and a failed-prerequisite branch. Distinguish task completion from issue delivery. Source: `docs/product/overview.md:68`, `:300`; `docs/design/factory-worker-mcp.md:429`. |
| F06 | P2 · Thin | Feed says filters, evidence and artifacts exist without teaching the actual inspection path. | Show global versus repository Feed, scope/kind filtering, opening the native issue/candidate and previewing/downloading a supporting artifact. Explain unavailable access and incomplete previews without promising a bypass. Source: `system/containers/extension/extension.json:26`, `internal/web/api/factory_feed_subscription.go:43`, `frontend/spaces/sodaspaces-factory-feed-artifact.ts:26`. |
| F07 | P1 · Thin | A saved comment, assessed route, draft and confirmed native effect are easy to conflate. | In **Factory Feed**, use one discussion/clarification/existing-work/new-work example with the next action for each receipt. Show “Answer clarification” and “Revise or retry request” when applicable; an unclear native write must be inspected under its original operation. Typing/reloading is not submission or authorization. Source: `frontend/spaces/sodaspaces-factory-feed-view.ts:262`, `:382`, `:394`, `:458`, `:547`; `docs/product/overview.md:151`. |
| F08 | P2 · Missing / Thin | Library retrieval/curation is described, but how worker learning and actual decisions enter it is unclear. Search/revision controls lack a worked example. | In **Knowledge and decisions**, show one evidence-backed, narrowly scoped lesson and one attributable decision. Explain source/role provenance, revision history, supersession and why advice cannot become policy. Include searching by text/type and examining evidence/revisions. Source: `docs/design/factory-worker-mcp.md:338`, `:827`; `internal/factory/library.go:11`; `frontend/spaces/sodaspaces-factory-library.ts:94`. |
| F09 | P1 · Thin | Integrated verification names manifests, fixtures and receipts but gives no complete accepted-outcome → approved check → observed evidence example. | In **Application verification**, work through two outcomes at one integrated revision. Missing fixture/receipt means incomplete; a failed observation means failed; only complete evidence supports verified. Include the two-run/60-active-minute default per stable intent across retries/version changes. Do not invent a manifest schema or exploratory verifier powers. Source: `docs/product/overview.md:413`; `docs/design/factory-worker-mcp.md:570`. |
| F10 | P2 · Missing | Activity visibility settings and their defaults are absent. Following Factory activity may be confused with native notifications. | Explain the three initially enabled summaries—issue created, PR created, verified merge with original issue closure—and who may change display settings. Hiding an event does not delete history, stop work or send notifications. Distinguish native Forgejo inbox/watch settings. Source: `docs/product/overview.md:722`; `frontend/spaces/sodaspaces-factory-activity.ts:204`; `frontend/forgejo/templates/user/notification/notification_div.tmpl:4`. |
| F11 | P1 · Thin | Intervention is comparatively detailed, but “why waiting?” still requires reading several pages. Resume and retry accounting need explicit comparison. | Add **visible symptom → owner → safe next step** rows for missing requirements, prerequisite outcome, Project readiness, provider busy/quota, CI, exhausted allowance and uncertain effects. Resume uses remaining attempt allowance; an authorized retry creates a new attempt with current-policy attempt allowances, preserving prior history, rolling provider and applicable intent/repair accounting. Source: `docs/product/overview.md:110`, `:385`, `:520`, `:539`; public intervention already owns commands and must retain them. |
| F12 | P1 · Thin | CI setup says “arrange runner capacity” and “name required checks” but does not bridge to a usable native runner/check setup. | Link the official native runner setup procedure from **CI runners**, explain intended runner scope and sensitive one-time registration material, then show a verified workflow/check-name/candidate-result example. Do not introduce Soda-provisioned local runners or treat a local test as CI. Source: `frontend/forgejo/templates/shared/actions/runner_setup.tmpl:13`; `docs/public/40-Factory/40-ci-runners.md:5`; `docs/product/scope.md`. |

### Defaults that should become public reference

These are intended product defaults, not hardware recommendations or currency
estimates. The appropriate owner controls changes; the tightest applicable
constraint wins.

| Limit | Initial value | Public owner |
| --- | --- | --- |
| Active issues across the appliance | 2 | Factory authorization |
| Active attempts / executing Factory sessions per repository | 1 / 1 | Factory authorization |
| Cumulative active time per attempt | 120 minutes | Factory authorization |
| Correction/reverification cycles after the initial candidate | At most 3 | Factory authorization |
| Leased Factory session time per provider connection across repositories | 6 hours per rolling 24 hours | Factory authorization; provider connections |
| Automatic usage-billed spending/overages | Zero | Provider connections |
| Stall repair per progress episode | 2 cycles | Factory intervention |
| Associated stall investigation/proposal/time bounds | 4 investigations, 4 proposal judgments, 120 active minutes in total | Factory intervention |
| Originating intent aggregate active time | 480 minutes | Plan new work; Factory authorization |
| Integrated verification per stable intent | 2 runs / 60 active minutes | Application verification |

The attempt clock includes assessment, coding, correction, review and **CI waits
after activation**. Queued, blocked or paused time is excluded only after agents
are confirmed stopped. Provider accounting instead measures leased Factory
session time until assigned processes stop; overlapping leases add. Upstream
subscription quotas remain separate, including consumption by personal CLI use.
Retrying or splitting work cannot silently refill applicable aggregate budgets.
Source: `docs/product/overview.md:325–423,539–544`.

## Projects, identity and human development gaps

| ID | Priority / type | Missing or inadequate coverage | Concrete remedy and evidence |
| --- | --- | --- | --- |
| P01 | P1 · Missing / Ambiguous | Provider connections appears personal-only and omits named-member delegation, credential-exposure consent and provider-specific active-use locks. | Document **Delegate this subscription in this project**, member selection, provider-terms and credential-exposure confirmation, inspecting active use, revoking a delegate and ending active use. Code running in the authorized account can read/copy its execution credential. Codex has one active stream across enrollment/human/Factory use; Muse permits independent concurrent leases. Explain busy versus rejected credentials and sponsored allowance versus upstream quota. Source: `docs/reference/credentials.md:95–148`; `frontend/spaces/soda-identity.ts:142–269`. |
| P02 | P1 · Thin / Ambiguous | Saved keys versus installed Project keys are distinguished, but rotation/removal has no complete procedure; “SSH key fingerprint” also confuses host identity with a user's public key. | In **Connect and develop**, save/select the public key, review target Project installed fingerprints, explicitly apply the saved set and confirm final-key removal when intended. Verify new-key acceptance and old-key refusal before closing a working route. Existing SSH sessions are separate. Name the **SSH host-key fingerprint** correctly. Source: `frontend/spaces/sodaspaces-project-access.ts:65–193`; `internal/web/api/access_keys.go:25–132`; `frontend/spaces/sodaspaces-project-settings-view.ts:355`. |
| P03 | P1 · Decision | People and access directs an administrator to remove departed-member account access without identifying a supported procedure. Native permission removal does not remove installed keys, accounts or existing processes. | Q2 selects explicit owner/operator revocation and restoration under R44.1. Once implemented, give its offboarding sequence, native-effect checks, blocked re-entry and partial-failure recovery. Do not infer an existing control from membership inventory; automatic cross-system synchronization remains deferred. Source: `docs/public/30-Set-up/30-people-and-access.md:17`; `docs/product/scope.md:104`; `internal/web/api/environments_api.go:165`; `docs/reference/project-os.md:87`. |
| P04 | P1 · Thin | Start/Stop authority and the full confirmation effects are not explicit enough. | Define Project lifecycle administrator as the configured Soda operator or current native repository/organization owner, according to the owning authorization rules. Membership, code-write and repository-admin permission alone do not grant it. Explain that Stop interrupts everyone's SSH/terminals/workloads and disables next-boot start; Start restores boot start, not interrupted sessions. Source: `docs/architecture/trust.md:232`; `frontend/spaces/sodaspaces-project-settings-view.ts:38–84`; `docs/product/projects.md:298`. |
| P05 | P1 · Thin | Trusted-team wording does not explicitly explain root/wheel access to all member files/credentials. Native ownership transfer can leave creation-time wheel grants in place. | Add the actual administrative trust boundary and a transfer checklist to **Projects and workspaces** / **Shared tools and files**. Review wheel access, native permissions and installed keys independently; do not remap stable Linux identities silently. Source: `docs/reference/project-os.md:56–72`; `docs/architecture/trust.md:254`; current public Project transfer text at `:63`. |
| P06 | P2 · Thin | Terminal rename, End's loss of unsaved in-process work and actionable support-state distinctions are missing or generic. | Add rename guidance and a brief stopped/support-mismatch/inspection-unavailable/maintenance table to **Workspace terminals**. Put the unsaved-work warning before End, while retaining what survives: files and independent SSH/services. Inspect the original entry after an uncertain action. Source: `frontend/spaces/sodaspaces-terminal-view.ts:149–209`; `docs/reference/terminal.md:38–94`. |
| P07 | P1 · Thin | Project preparation explains responsibilities but gives no sample selected source, accepted requirement, shared prerequisite or coder/reviewer readiness outcome. | Add one verified small repository example: exact source inputs, required tool/service, writable path, safe smoke check and separate role results. Show a changed input invalidating readiness and the safe maintenance hold. Use a real supported setup interface; do not invent fields or imply an approval ID proves every prerequisite. Source: `docs/public/50-Spaces-and-Projects/30-prepare-project.md:5`; `docs/product/projects.md`; `frontend/spaces/sodaspaces-factory-settings.ts`. |
| P08 | P2 · Thin | Shared tool paths and CLI names are documented without a useful first-use recipe. | Add a reviewed repository-pinned mise example, expected version/check result, trust decision and how to request an administrator-owned prerequisite. Add a short native personal `tea`/`gh` authentication/read-task pointer; tokens stay out of shared paths. Source: `docs/reference/project-os.md:32–85`; `docs/guides/develop.md:19`; `docs/guides/project-clis.md:10`. Verify any new external-tool command before publishing. |
| P09 | P2 · Missing | Manual Muse launch and multiple authorized connection selection are not explained after enrollment. | In **Connect and develop**, show the supported `muse` / `muse exec` path and `SODA_MUSE_CONNECTION=CONNECTION_ID muse` when needed, including where the ID is obtained. Separate launch-service failure from sign-in rejection and Factory sponsorship. Source: `cmd/soda-muse/README.md:7–43`; `docs/reference/credentials.md:150–172`. |
| P10 | P1 · Ambiguous | The Canlang command block places foreground `can lsp` before `can dev stop`, as though the whole block were a sequential one-shot recipe. It also assumes an existing `app.can` and example suite. | Split check/compile, preview/example, editor LSP and cleanup into separate stages. Use the actual returned revision and a valid sample app with examples; show an expected result and the empty-suite refusal. Source: `docs/public/60-Develop/50-canlang-development.md:7–25`; `docs/guides/canlang-applications.md`. Do not invent Canlang syntax to fill the sample. |
| P11 | P2 · Missing | Native account security/recovery is not discoverable from People and access. | Add a brief link to the maintained native Forgejo security/recovery instructions, making clear that browser identity recovery differs from provider reconnect, Project SSH and the configured Soda operator. Source: `frontend/forgejo/templates/user/settings/security/twofa.tmpl:8`; `frontend/forgejo/templates/user/auth/twofa_scratch.tmpl:5`; `frontend/forgejo/templates/user/auth/forgot_passwd.tmpl:5`. Do not duplicate the upstream account manual. |

## Installation, operations and private application gaps

| ID | Priority / type | Missing or inadequate coverage | Concrete remedy and evidence |
| --- | --- | --- | --- |
| O01 | P1 · Missing | On-premises installation asks for password/subnet review without known minimum/default/reserved values. | State the 12-character root-password minimum, default Project subnet `10.89.0.0/24`, and conflicts with reserved `10.88.0.0/16` and `10.90.0.0/24`, alongside LAN/VPN collision checks. Source: `cmd/soda-install/src/wizard/steps.rs:159–247`. Do not prescribe one universal alternative subnet. |
| O02 | P1 · Thin / Decision | Download verification says to verify “verification material” without a complete consumer operation. | Q3 selects GitHub immutable-release asset attestation for the exact repository/tag and downloaded ISO or installer. R52/R53 must exercise it before the handbook gives concrete assets and match/refusal results. Distinguish pre-boot verification from the installer's matching-rootfs authentication. Source: `docs/public/20-Install/20-verify-downloads.md:13`; `docs/architecture/release.md:50–80`. |
| O03 | P1 · Missing / Thin | Docker prerequisites and acquisition/execution are incomplete. | Include the installer-enforced Compose 2.17 floor, required available TCP ports 443/2222/3000/9090 and their distinct purposes, default/reserved subnet checks, and relevant `--bind-ip`, `--project-subnet`, `--install-dir`, `--yes` semantics. Complete qualified-release acquisition → verification → run → inspect, with placeholders explained. `--yes` does not waive validation. Source: `system/docker/install-docker.sh:11–19,173–178,264–268`; `system/docker/compose.yaml:26`; `docs/guides/docker.md:43–50,100–131`. |
| O04 | P1 · Thin | Partial operator setup says “inspect” without an observable distinction between existing binding, incomplete activation and uncertain reservation. | Identify an existing supported read-only command or journal observation that distinguishes these states, then document safe continuation conditions. If no such interface exists, use bounded Administration checks and direct the reader to operator support before another mutation. Do not invent a setup-status command or overwrite a reservation blindly. Source: `cmd/soda-install/src/setup/configure.rs:23–41`; `docs/guides/operator-setup.md:14–36`. |
| O05 | P1 · Thin | Browser origin, Git advertisement and Project routing are correctly separated but lack a concrete values/check example. | Add a purpose/address/port/client-check table and one fictional private network example. Include HTTPS certificate hostname/trust, Git SSH advertisement and Project route separately. Cross-link optional Tailnet without treating enrollment as reachability proof. Source: `docs/architecture/networking.md`; `cmd/soda-activate/README.md`; `system/docker/install-docker.sh:173`. |
| O06 | P2 · Thin | Cockpit recommends an SSH tunnel without showing how to open it or which browser address to use. | Add a private local-forward example using the already-authorized operator account and the actual listener, then its HTTPS login and verification. Explain placeholders and certificate behavior. Source: `docs/public/70-Operate/20-cockpit.md:7`; `docs/guides/operator-setup.md`; `docs/product/scope.md:65`. |
| O07 | P1 · Thin / Decision | Private-app onboarding jumps from “build/provision” to “register/install” without a complete supported path or account-creation walkthrough. | Q4 selects a prepared-app wizard and protected app-owned first-owner browser setup, retaining stopped CLI recovery. P14/P26 implement the setup state/receipt and UI; P29/P30 supply the actual walkthrough, operator/app-administrator permissions, private TLS, Install leaving stopped and explicit Start. Source: `docs/guides/canlang-applications.md:82–186`; `internal/web/api/applications.go:172–196`; `frontend/spaces/sodaspaces-applications.ts:97–166,287–301`. Existing controls do not establish this new flow. |
| O08 | P1 · Thin / Decision | Snapshot restore creates a fresh data identity, but the handbook stops before making that restored app usable. | Q4 selects a separate explicit Restore operation that rebinds the same logical app/address to verified fresh data. P25/P26 must implement it before P29/P30 document and exercise stopped inspection and explicit Start. Keep old data and normal Replace's data fence; unknown outcomes require inspection. Source: `docs/public/70-Operate/60-backups-and-restoration.md:50–67`; `docs/guides/canlang-applications.md:148–186`. |
| O09 | P1 · Thin / Decision | Database dumps and app snapshots have commands, but whole-machine consistent capture/restore remains an instruction to “coordinate” writers. | Q5 selects a complete cold set under R45/R52/R54: stopped appliance disks or all Soda Docker volumes/settings, including external retained mounts, databases, Project roots/homes, broker key/custody, configuration and app data. Give an ordered capture and isolated restoration procedure after qualification. Source: `docs/public/70-Operate/60-backups-and-restoration.md:5–25`; `system/host/services/README.md`; `docs/architecture/overview.md`. The seven-run database timer is not a complete system backup. |
| O10 | P1 · Thin | Update acquisition and recovery/rollback invocations are abbreviated, despite complete check/apply examples. | In **Updates and fallback**, show protected release acquisition, exact qualified source/target check, apply and expected verification; in recovery, give full installed-helper forms and retained-operation preconditions. Preserve that rollback does not restore mutable data. Source: `docs/guides/operator-setup.md:99–136`; `docs/public/70-Operate/50-updates-and-fallback.md:7–45`; `lib/host/README.md`. |
| O11 | P2 · Missing / Decision | Private-app capacity planning omits the native runtime envelope. | The inspected launch uses a read-only container root, 1 GiB memory, 2 CPUs and 128 processes per app. Confirm these are intended public limits before documenting them as stable or configurable; inspect readiness/service evidence when an app fails. Source: `lib/host/src/application/runtime.rs:147–168`. Do not promise a specific restart policy or tuning control from these flags alone. |
| O12 | P2 · Decision | CPU/RAM/disk minima and recommended reserve are not yet qualified. | Q6 selects measured basic and small-team workload profiles under R45/R52/R54. Publish supported values, concurrency and update/backup reserve after measuring the candidate; fixture RAM and payload size are not guarantees. Source: `docs/architecture/release.md:189`; `docs/public/20-Install/10-deployment-options.md:9`; `docs/guides/docker.md:14`. |
| O13 | P2 · Thin | Troubleshooting routes symptoms correctly but offers little concrete evidence collection or successful-versus-failed interpretation. | Add a compact diagnostic worksheet: affected account/action, exact Project/run/app identity, last confirmed state, relevant service/log excerpt and target network path. Link existing inspection commands and show one redacted example per common failure class. Avoid invented error strings, blanket restarts or exposing credential material. Source: `docs/public/70-Operate/90-troubleshooting.md:5`; `docs/public/70-Operate/10-administration.md`; `docs/reference/terminal.md:60`. |

## Complete page review matrix

All 41 articles were reviewed. A row with no new material omission does not
claim the feature was exercised on an installed system. Existing good examples
and boundaries should survive subsequent edits.

| Public source page | Review conclusion / next improvement |
| --- | --- |
| [Start here](../../public/10-Start-here/10-index.md) | Useful task map already exists. G01–G03: prioritize daily entrances and link complete first outcomes. |
| [Product model](../../public/10-Start-here/20-product-model.md) | Strong lifetime separation. G04, F01, P05: exact authority/acceptance/trusted-admin language. |
| [Deployment options](../../public/60-Install/10-deployment-options.md) | Honest supported-mode boundaries. O12 awaits qualified sizing; do not invent a support matrix. |
| [Verify downloads](../../public/60-Install/20-verify-downloads.md) | O02: replace abstract verification instruction with the supported consumer procedure. |
| [On-premises install](../../public/60-Install/40-install-on-premises.md) | O01: exact password/subnet constraints and expected installer result. |
| [VM/cloud install](../../public/60-Install/30-deploy-to-cloud.md) | No additional material omission beyond shared O02/O12. Keep common ISO path and VM storage/console/network responsibilities. |
| [Docker](../../public/60-Install/50-docker.md) | O03: complete prerequisites, options, acquisition and verification sequence. |
| [Operator setup](../../public/60-Install/60-operator-setup.md) | O04: partial-state inspection and safe continuation; preserve existing setup commands. |
| [Private networking/TLS](../../public/40-Configure/50-private-networking-and-tls.md) | O05: concrete browser/Git/Project values and independent checks. |
| [People and access](../../public/40-Configure/10-people-and-access.md) | G04, P03, P11: action/authority map, supported offboarding and native account-recovery pointer. |
| [Provider connections](../../public/10-Start-here/50-provider-connections.md) | P01, P09, F02: delegation/custody/concurrency and link to actual personal CLI use. |
| [First connection](../../public/10-Start-here/40-first-connection.md) | Browser-only Join and `whoami`/`pwd` are useful. G02: continue into first real contribution. |
| [Tailscale](../../public/40-Configure/60-tailscale.md) | No new material omission. Preserve optional Fountain-owned enrollment and separate route/TLS checks. |
| [Software factory](../../public/20-Factory/10-software-factory.md) | Strong overview; link G01 worked task and F11 common waiting reasons. No need for another architecture diagram. |
| [Plan new work](../../public/20-Factory/30-plan-new-work.md) | G01, F03: concrete objective/assumption/delivery example and exact acceptance semantics. |
| [Factory authorization](../../public/40-Configure/20-factory-authorization.md) | F01–F03: automatic initial acceptance, changed inputs, useful defaults and separate limits. |
| [Issue work/dependencies](../../public/20-Factory/40-issue-work-and-dependencies.md) | F05, F11: concrete output/checkpoint/dependency example; retain code/non-code/research distinctions. |
| [CI runners](../../public/40-Configure/40-ci-runners.md) | F12: native setup pointer and verified required-check example. |
| [Review and merge](../../public/20-Factory/50-review-and-merge.md) | F04: complete clear review/correction acceptance and additional native approval gates. |
| [Application verification](../../public/20-Factory/60-application-verification.md) | F09: two-outcome worked verification, missing-fixture case and exact default allowance. |
| [Factory Feed](../../public/20-Factory/20-factory-feed.md) | F06, F07, F10: actual filtering/evidence/comment route and activity-versus-notification behavior. |
| [Knowledge and decisions](../../public/20-Factory/70-knowledge-and-decisions.md) | F08: source-linked contribution and search/revision example. |
| [Factory intervention](../../public/20-Factory/80-factory-intervention.md) | Good action and support-command detail. F02/F11: symptom/owner routing and distinct clocks/resume/retry. |
| [Spaces](../../public/10-Start-here/30-spaces.md) | Correct context retention. G03: make global Workspace, repository/global Feed and saved request target easier to recognize; link terminal tasks. |
| [Projects/workspaces](../../public/30-Projects-and-development/10-projects-and-workspaces.md) | P04/P05: lifecycle authority, confirmation effects and transfer/admin trust. |
| [Prepare Project](../../public/40-Configure/30-prepare-project.md) | P07: real sample inputs, prerequisites, two-role smoke checks and invalidation. |
| [Workspace terminals](../../public/30-Projects-and-development/20-workspace-terminals.md) | P06: rename, unsaved-process warning and support-state recovery table. |
| [Collaboration](../../public/30-Projects-and-development/30-collaboration.md) | Native ownership/import/Git boundaries are covered. G02/F10/P11: worked contribution and focused native-guide links. |
| [Connect/develop](../../public/30-Projects-and-development/40-connect-and-develop.md) | P02/P09: installed-key apply/rotation, host fingerprint and manual Muse launch. |
| [Shared tools/files](../../public/30-Projects-and-development/50-shared-tools-and-files.md) | P05/P08: trusted administrator boundary and pinned-tool/personal-CLI quick starts. |
| [Project services](../../public/30-Projects-and-development/60-project-services.md) | Existing Compose/systemd/forwarding examples are useful. Link them from the common sample rather than add an unrelated service chapter. |
| [Canlang development](../../public/30-Projects-and-development/70-canlang-development.md) | P10: fix sequential LSP presentation and supply a valid sample/revision/result. |
| [Administration](../../public/50-Operate/20-administration.md) | Useful bounded health checks. O04/O13: reuse these in setup and diagnostic procedures. |
| [Cockpit](../../public/50-Operate/30-cockpit.md) | O06: tunnel and expected local HTTPS login example. |
| [Project maintenance](../../public/50-Operate/60-project-maintenance.md) | No new material omission. Preserve running/stopped preconditions and different retry-versus-inspect recovery paths. |
| [Private applications](../../public/50-Operate/40-private-applications.md) | O07/O08/O11: complete bootstrap/registration/install/start path, restored identity and capacity envelope. |
| [Updates/fallback](../../public/50-Operate/70-updates-and-fallback.md) | O10: complete acquisition/recovery forms and retained-operation preconditions. |
| [Backups/restoration](../../public/50-Operate/50-backups-and-restoration.md) | O08/O09: usable restored app and complete consistent recovery set. |
| [Data safety/removal](../../public/50-Operate/90-data-safety-and-removal.md) | Good persistence/removal boundaries. Link clarified P03/O09 rather than invent new destructive helpers. |
| [Recovery](../../public/50-Operate/80-recovery.md) | Good owner-based routing. O04/O10: exact partial-setup/release continuation paths. |
| [Troubleshooting](../../public/50-Operate/10-troubleshooting.md) | O13/F11/P06: recognizable symptoms, bounded evidence and next owner/action. |

## Worked examples to add first

Examples should be coherent, small and tested against the supported interface.
Each needs prerequisites, exact actor/context, input, action, expected result,
one relevant failure case and safe cleanup where applicable. Labels for a
fictional scenario must not masquerade as actual interface fields.

| Order | Example | What the reader learns | Owners |
| --- | --- | --- | --- |
| 1 | One small Factory request from objective to confirmed delivery | State outcomes, correct one assumption, inspect derived native work, follow a dependency, inspect review/check evidence and recognize the separate verification boundary. | Plan new work, linking existing stage owners |
| 2 | Why one issue waits while another runs | Two appliance slots, one repository attempt, a busy Codex connection and separate attempt/provider/intent clocks; show the right owner and no automatic billing fallback. | Authorization; intervention |
| 3 | Prepare a small repository for both roles | Accepted exact setup source, an administrator-provided prerequisite, separate coder/reviewer smoke checks and a changed-input invalidation. | Prepare Project |
| 4 | Join, edit and submit a first human contribution | Provider-free browser Join, own identity/checkout, native Git credentials, ordinary check and PR. | First connection; collaboration; connect/develop |
| 5 | Rotate a Project SSH key | Saved versus installed set, host versus user fingerprint, explicit apply, new-key success and old-key refusal without assuming existing sessions end. | Connect/develop |
| 6 | Review and repair a candidate | Required defect versus optional advice, changed head, fresh check evidence and native branch-protection approval if configured. | Review and merge |
| 7 | Answer a Feed question and inspect its receipt | Saved comment versus accepted answer versus proposed work versus confirmed native issue. | Factory Feed |
| 8 | Verify two integrated outcomes | Exact integrated revision, approved fixtures, expected/observed result, incomplete versus failed, finite correction. | Application verification |
| 9 | Use a repository-pinned tool and a development service | Shared tool/admin boundary, personal trust decision, assigned service endpoint, durable versus disposable data and expected result. | Shared tools; Project services |
| 10 | Canlang preview, editor and cleanup | Valid sample app/examples, actual revision, separate foreground LSP and explicit preview stop. | Canlang development |
| 11 | Install and restore one private application | App-local first-owner setup, exact image/data identity, explicit install/start and same-address fresh-data restore. Await Q4's P14/P25/P26 implementation. | Private applications; backups |
| 12 | Capture and verify a complete recovery point | Writer coordination, every retained domain and an isolated restored-system check. Await Q5's R45/R52 procedure qualification. | Backups/restoration |

Release verification and Docker acquisition examples are also prerequisites to
operator onboarding; Q3's release tasks must verify their actual inputs first. Do not add
an example merely to reach a count. The current handbook already has **25 shell
fences, 14 Mermaid diagrams and 10 screenshot briefs**; adding more visuals alone
will not fill the task gaps above.

## Predicted usage and recommended prominence

There is no user telemetry: SodaOS is unreleased. These are qualitative forecasts
from workflow structure, with different expectations for request authors,
developers and operators. “Frequent” describes repeated use after onboarding,
not the percentage of users or the number of support visits.

| Expected steady-state use | Likely reader | Confidence | Documentation entrance |
| --- | --- | --- | --- |
| Inspect current work, blockers and resulting PR/check evidence | Request authors and maintainers | High: every active delivery produces these questions | First group of index task links; Feed overview; contextual intervention/review links |
| Open/reconnect a terminal and continue a personal checkout | Hands-on developers | High within that persona; lower for request-only users | Direct index/Spaces link; short first connection route |
| Describe/revise work and answer clarifications | Request authors | High that it recurs; exact rate unknown | Plan new work tutorial and Feed response example |
| Understand a failed check/review or readiness wait | Maintainers and developers | Medium-high; frequency depends on project quality | “Why is work waiting?” entry with symptom/owner table |
| Search earlier decisions and evidence | Maintainers and returning contributors | Medium; value grows with project history | Feed/library cross-links and search example |
| Connect/delegate a provider or adjust sponsored use | Account owners | Medium; setup is occasional, busy/quota inspection recurs | Provider page linked from waits and authorization |
| Install/setup the appliance or onboard a new teammate | Operators / new contributors | High first-use importance, low steady-state frequency | Distinct operator and invited-contributor entry paths |
| Maintain Projects, update, back up and inspect host health | Operators | Periodic; exact cadence unselected | Operate entry and task-specific links |
| Recover data, rotate keys or offboard someone | Operators / account owners | Lower frequency, high consequence | Easy to find through task words and relevant failure sections |
| Install/replace private apps | App owners and operators | Uncertain; depends on product adoption | Visible app task link, preserving explicit deployment boundary |

### Recommended navigation treatment

1. Keep the seven-section tree and existing filenames for now. The coverage
   problem does not justify another wholesale route rewrite.
2. Improve the existing index with three short entry choices: **Use an existing
   installation**, **Build something with Factory**, **Install or operate
   SodaOS**. Invited users should not have to read installation first.
3. Place **Follow work**, **Why is it waiting?**, **Open my terminal**,
   **Start or revise a request**, and **Inspect a candidate** high in the task
   links. These are recommendations for content links, not proposed product
   buttons or a manual table of contents.
4. Keep first-time setup visible but stop repeating its full prerequisite chain
   in every daily task. Link the relevant owner with a brief precondition.
5. Expand current owners before adding pages. A separate first-build tutorial
   may be warranted; one page per robot, internal component or optional native
   Forgejo feature is not.
6. Keep the recovery/offboarding routes prominent despite their lower predicted
   frequency. Consequence matters as well as repetition.

Three independent, differently framed TypeSafe consultations agreed on
task-oriented entry links and worked tasks ahead of extra visual volume or
component-reference breadth. This was a secondary prioritization check;
agreement is not evidence of actual usage or product correctness.

<a id="owner-decisions-required-for-exact-instructions"></a>
## Selected resolutions and remaining evidence

The owner requested product resolutions after this audit. The
[existing remaining-work list](../factory-remaining-tasks.md#selected-product-resolutions-for-handbook-blockers)
now owns all seven directions and their acceptance criteria. The questions
are no longer unassigned owner decisions; the remaining blockers are concrete
implementation, measurements and release observations.

| ID | Selected direction | Still required before publishing exact instructions |
| --- | --- | --- |
| Q1 | Scoped allowances; no global cumulative ceiling. | Overview/API contradictions corrected. R11.2/R30.3/R48.2 qualify remaining accounting consumers. Existing usage tracking is retained. |
| Q2 | Explicit owner/operator Project access revocation and restoration. | R44.1 implements the control, native keys/login/session effects, durable re-entry hold and partial-failure inspection. |
| Q3 | GitHub immutable-release asset attestation for the exact repository/tag. | R51 supplies staged assets; R52/R53 prove the actual downloaded-byte verification procedure and its refusal cases. |
| Q4 | Guided prepared-app and app-owned first-owner setup; same-app explicit Restore into fresh data. | P14/P25/P26 implement the setup state/receipt, UI/API/native operations and immutable restore revision; P29/P30 document and exercise them. |
| Q5 | Complete cold appliance/Docker recovery set and isolated fresh restore. | R45 develops the procedure; R52 proves every retained domain on the candidate; R54 integrates the guide. |
| Q6 | Measured supported basic and recommended small-team sizing. | R45/R52 measure declared workloads, CPU/RAM/disk and update/backup reserve. Numeric requirements remain unknown until measured. |
| Q7 | Fixed 1 GiB RAM, two CPUs' worth of time and 128 processes/threads per normal app. | P26/P30/R52 qualify actual limit/failure behavior; P29 documents it without implying aggregate host capacity or tunable controls. |

Three independent counterbalanced JEV consultations selected the same
directions. Guided app setup had the least consistent confidence, so its
existing owner-team precondition was checked separately: a provisional setup
state is required, and CLI recovery remains available. This advice supports
the product choice; it does not qualify any implementation.

The Project lifecycle authority is **not** an unresolved product conflict.
Trust and API authorization agree; P04 is a terminology/procedure fix. Broader
department sharing, new exploratory verifier powers, local Soda CI runners,
public app hosting and automatic deployment remain outside selected scope.

## Suggested implementation order and completion criteria

This was the original follow-through order. The
[current task list](cutover.md#priorities-and-execution-order) supersedes it and
records completed improvements separately from remaining documentation work.

1. **Correct consequential ambiguity:** F01/F02, P01/P02/P04/P05/P10 and O01.
   These mostly have existing authoritative answers.
2. **Complete core daily tasks:** G01–G04, F04–F09/F11/F12 and P07; add the
   highest-value worked examples and direct entry links.
3. **Implement and qualify the selected Q1–Q7 resolutions in their existing
   task owners**, then finish dependent setup, offboarding, application and
   recovery procedures. Independent handbook improvements need not wait.
4. **Add supporting examples and discovery:** native security/runner links,
   terminal support states, library search, shared tools, personal CLI and
   Cockpit; preserve useful existing service and maintenance examples.

For a finished task guide, a reader with the stated role and prerequisites
should reach its claimed result without searching private source/reference
files or guessing a control/placeholder. Verify new commands and samples in the
supported context, including expected failure and cleanup behavior where
relevant. Keep the 140–220-character summaries, meaningful heading hierarchy,
stable useful anchors, contextual links and current visual-source contract.

The resolution follow-up updates the existing backlog/packets, the intended
application contract and contradictory ceiling prose. It does not implement
those features or rewrite public pages. Validation covers inventory, complete
page allocation, local links/anchors, source-reference existence and whitespace.
It does not establish installed-product correctness or successful execution of
the proposed operator examples.

Validation passed: all 41 articles appear exactly once in the review matrix;
all 40 finding IDs are unique; local links and source citations across the
report/index resolve to existing targets, with cited starting lines in range.
The changed documentation passed whitespace checks and `git diff --check`.
No product implementation or public article examples changed. The proposed
operator procedures still require their assigned native/installed checks;
this documentation validation did not run a build, restore, release-asset
verification or public visual regeneration.
