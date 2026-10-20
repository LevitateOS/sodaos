# Page contracts

Paths below are relative to `docs/public/` and are the exact leaves in the
[target tree](proposed-tree.md). Each page owns a reader task. The source slices
link to [investigation evidence](source-audit.md); internal documentation remains
the authority for the corresponding product contract.

Every published page starts with one descriptive H1 and a standalone introduction
of 140–220 characters, immediately followed by a body organized under meaningful
H2 sections. H3 subsections belong beneath a relevant H2. Move detailed rules out
of the introduction without dropping their meaning; preserve existing anchors
where possible. The source README owns the complete authoring contract and
article templates. Write the finished intended behavior in present tense. Include prerequisites, actor,
observable success and consequential failure/recovery where applicable. Link to
the canonical public procedure instead of repeating it. Do not publish source
paths, task IDs, candidate revisions, fixture instructions or unfinished-feature
chronology as user steps.

## Start here

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| H01 `10-Start-here/10-index.md` | Handbook entrance and routes for the first successful journey | Route readers by install, build request, watch/answer, human development or operation. Link the complete first build path without duplicating its procedures. | S01–S12 |
| H02 `10-Start-here/20-product-model.md` | Product concepts, scope and authority map | Host, native repository, accepted build request, issue/work item, Project, checkout/workspace, attempt, run, session/view and app. Private trusted team, host-kernel sharing, separate CI and explicit app deployment. Detailed permissions and provider custody link to Setup. | S01–S12 |

## Install

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| I01 `20-Install/10-deployment-options.md` | Choose appliance hardware/VM or direct Docker deployment | x86_64, private clients, persistent storage, console and installation-network prerequisites; distinguish host authority and lifecycle. No invented sizing, named-cloud images, ARM or WSL2 path. | S09 |
| I02 `20-Install/20-verify-downloads.md` | Select and verify the released distribution and artifacts | Exact release/tag and authenticating its bytes before use. ISO/rootfs acquisition versus pinned Docker installer/image. No release-build or local candidate workflow. | S09 |
| I03 `20-Install/30-install-on-premises.md` | Common ISO installation onto hardware or a VM's selected disk | Boot, review destination, explicit erasure, root password, Project subnet and installed first boot. Make data destruction unmistakable at its real step. Hand off to U01 for setup/activation. | S09 |
| I04 `20-Install/40-deploy-to-cloud.md` | Prepare a compatible VM/private-cloud environment | Title: “Install in a VM or cloud”. Virtual disk, boot/console, networking, access and persistence; then reuse I03 for ISO disk installation. No speculative provider image/import branch or duplicate installer instructions. | S09 |
| I05 `20-Install/50-docker.md` | Install and operate the released Docker distribution | Dedicated operator-managed Linux, rootful privileged runtime, cgroups/network context, pinned installer, `/etc`/`/var`/`/home` persistence and same-image recreation. Version transition/recovery links to Operate; never equate outer-container privilege with host isolation. | S09, S12 |

## Set up

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| U01 `30-Set-up/10-operator-setup.md` | First native Forgejo administrator, Soda operator binding and activation | Separate appliance guided `soda-install configure` from Docker installer-managed setup or runtime `docker compose exec` with `soda-setup`/`soda-activate`; the appliance command requires installed FCOS. Exact operator identity, activation outcomes and inspection of partial setup. Keep ‘Trust and open the private browser address’ as a handoff anchor to U02. Ongoing Factory funding is F03. | S01, S09 |
| U02 `30-Set-up/20-private-networking-and-tls.md` | Private origins, routes and HTTPS client trust | Browser origin/listener, Git advertised endpoints and Project subnet are different settings. Local/supplied certificates, client trust, private DNS and separate application hostnames. Verify reachability from the intended client; do not require a purchased domain. | S10 |
| U03 `30-Set-up/30-people-and-access.md` | Onboarding, authority changes and offboarding across distinct identities | Native forge user/team/repository access, configured operator, Project membership/admin, host root, provider owner and app-local account. Permission loss can disconnect managed attachments without deleting Linux accounts, ending retained shells or revoking installed SSH. Separate browser sign-out, saved-key removal, installed-key application and native Git access. No deferred synchronized offboarding/remapping promise. | S01, S03, S06, S08 |
| U04 `30-Set-up/40-provider-connections.md` | Supported AI account connection, custody and personal delegation | Enroll, inspect, reconnect, disconnect/revoke and launch an authorized personal managed CLI; distinguish leases, provider quota and subscription authentication. Credentials never enter prompts or shared files. Explain rejected/exposed credentials, unconfirmed return and no silent billing/account fallback. Factory sponsorship belongs to F03. | S03 |
| U05 `30-Set-up/50-first-connection.md` | A teammate's first native sign-in, explicit Project Join and real terminal | Start from an existing authorized repository Project and follow P02/P04's canonical procedures; account-only browser Join works without an SSH key. Inspect retained creation/join outcomes. Optional direct SSH/key setup links to D02; provider connection is optional for a normal shell. | S01, S06, S07 |
| U06 `30-Set-up/60-tailscale.md` | Optional appliance Tailnet and Project connectivity | Fountain enablement, supported sign-in and host versus Project identities/companions; policy, subnet routes and Git advertisement refresh. Link U02 for DNS/TLS and client proof. No assumption of a host-installed `tailscale up` workflow or mandatory Tailnet for LAN use. | S10 |

## Factory

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| F01 `40-Factory/05-software-factory.md` | The entire outcome-to-verified-version journey | Desired work → reviewed direction → native graph → readiness/preparation → coding/checkpoints → independent review/correction/checks → merge → dependent reassessment → integrated verification. Explain seven production roles plus Library Admin without an internal tool catalog. Link detailed owners below. | S02–S05 |
| F02 `40-Factory/10-plan-new-work.md` | Start, save, resume and revise a build request | Objective before repository selection; private optional department context; alternatives and consolidated experience; attributable choices and material questions; delivery approach; repository association and bounded issue generation. Inspect original uncertain submissions; no duplicate native work or acceptance by silence. | S02 |
| F03 `40-Factory/20-factory-authorization.md` | Enable and fund standing Factory operation | Repository policy/target/native actors/harnesses, nonempty checks and merge conditions; operator capacity/resources; provider sponsorship/roles/allowance. Explain finite planning/attempt/correction/intent/rolling limits and their separate owners. Link U04 for account enrollment and P03 for preparation. | S03 |
| F04 `40-Factory/30-issue-work-and-dependencies.md` | Accepted native issue work, executable plans and prerequisite outcomes | Authorized creation/adoption, exact inputs, work type/expertise/outputs, factual answers versus decisions and native blocked-by graph. Inspect Architect task graph/lanes/ownership/checks and Builder trusted checkpoints/outputs; preserve independent work when prerequisites fail and reject stale continuation. Research create/reuse inherits bounded follow-up funding, releases/settles the blocked parent's slot and returns exact accepted evidence for fresh reassessment; negative and inconclusive results differ. Cover cycles, unreadable evidence and edit/reopen invalidation. Exact metadata/routing/non-code acceptance awaits research contract DEC04. | S02, S04, S05 |
| F05 `40-Factory/40-ci-runners.md` | Supply separately managed native Actions verification capacity | Register/configure external capacity through native Forgejo, required-check names and exact candidate evidence, unavailable infrastructure and trusted-code scope. Coding/review roles are not runners. No usable placeholder for deferred Soda-provisioned local execution. | S05 |
| F06 `40-Factory/50-review-and-merge.md` | Inspect and resolve candidate review, correction and native delivery | Candidate/head/base identity, independent findings, Fixer outcomes, fresh review or permitted verified-correction acceptance, optional advice/disputes, required CI and native protections. Changed inputs invalidate evidence; require confirmed native merge and confirmed original issue closure. General human PR procedures stay in D01. | S05 |
| F07 `40-Factory/60-application-verification.md` | Verify delivered outcomes at the integrated intent version | Approved checks/journeys/disposable fixtures, exact source/artifact receipts and independent Test Pilot coverage. Missing preparation, partial/inconclusive results and bounded repair preserve outcomes/usage. A finished issue or empty queue is insufficient; verified completion is separate from app installation. | S05 |
| F08 `40-Factory/70-factory-feed.md` | Follow, understand and discuss live work | Project/global following, truthful role/controller posts, filters, threads, source excerpts, graphs and artifact previews/downloads. Comments route to discussion, clarification, existing work or bounded new work with inspectable receipts. Reading/following is passive; a reply is not automatically accepted scope or funding. | S04 |
| F09 `40-Factory/80-knowledge-and-decisions.md` | Find, reuse and maintain shared knowledge and decision records | Claims, provenance, applicability, contradictory evidence, governance, validity and revisions; scoped Library Admin edit/remove/restore/consolidate/split/relate/supersede. History and real decision authority survive curation. No manufactured issue completion or visibility expansion. Research execution belongs to F04. | S04 |
| F10 `40-Factory/90-factory-intervention.md` | Resolve waits, blocked work and interrupted execution | Distinguish objective, capacity, preparation, account and permission causes. Pause/resume/cancel, explicit retry, confirmed takeover, grant withdrawal, bounded stall repair and remaining allowances. Unknown effects/termination/credential return reconcile the original operation. Include the current operator status/stop/reconcile interface as support, linking O08 for whole-host recovery. | S03, S05 |

## Spaces and Projects

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| P01 `50-Spaces-and-Projects/10-spaces.md` | Find the correct request, Project, Feed, application or retained human session | Global Workspace and persistent panel, repository Feed, Applications discovery, saved requests, context and navigation. Browsed repository, request repository and retained terminal target remain distinct. Opening/viewing does not create/start/join/fund anything. Detailed Feed, application and terminal tasks stay in F08/O04/P04. | S01, S07, S11 |
| P02 `50-Spaces-and-Projects/20-projects-and-workspaces.md` | Create/reuse, Join, Start and Stop one persistent repository Project | Human repository owner creates the immutable Rocky headless Project; organization-owned creation is unsupported. Current native code-write is required for Join, managed terminals and installed access keys. Preserve original Linux login through native rename/transfer. Separate accounts/checkouts, account-only/key Join and lasting state. Create does not join/clone; inspect uncertain outcomes. Stop affects all users/services and coordinates Factory withdrawal; Start retains the root without reviving prior executions. | S06 |
| P03 `50-Spaces-and-Projects/30-prepare-project.md` | Prepare the shared environment for authorized Factory work | Exact-source setup acceptance, administrator inspection/privileged prerequisites, assigned services, separate unprivileged coder/reviewer readiness and maintenance holds. Changed inputs invalidate affected readiness. Inspect waits/failures and retry the retained bounded preparation; no root agent or environment replacement. | S06 |
| P04 `50-Spaces-and-Projects/40-workspace-terminals.md` | Human terminal sessions, attachments and layout | Member-scoped human inventory; code-write-authorized live Factory observation. Create/name/select/split/hide/show, retained navigation, exact-session reconnect and confirmed End. Detach/hide differs from termination; original account/Project binding survives navigation. Ended Factory runs expose evidence, not terminal reattachment/transcript replay. Watching differs from input/takeover; handle existing writers, sign-in changes, denied access and maintenance. | S07 |

## Develop

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| D01 `60-Develop/10-collaboration.md` | Human native Forgejo repository and collaboration tasks | Create/import/fork, visibility/collaborators, actual clone URLs and native Git credential registration, branches, issues/comments, PRs/reviews/protections, Actions records and native releases/wiki/packages. Keep `Create or import a repository` as an explicit section. Incoming Project SSH/key application belongs to D02. Link upstream for general detail and Factory owners for automation; repository actions do not create Projects or deploy apps. | S01, S05, S08 |
| D02 `60-Develop/20-connect-and-develop.md` | Direct SSH, remote editors and personal development checkouts | Explicit public-key application, current username/address/fingerprint, client route, SSH/SCP/SFTP/forwards and cloning into personal home using one's own Git credentials. Distinguish route/key/host-key failure; deleting a saved key does not itself remove installed access. Managed AI account details link to U04. | S07, S08, S10 |
| D03 `60-Develop/30-shared-tools-and-files.md` | Shared installed tools, native packages and shared files | Project administrator installs shared mise/package versions; members use them, with repository trust and personal overrides explained. Native ownership and real shared paths; coordinate changes across human/Factory use. Tool refresh and backups link to O03/O06. | S08 |
| D04 `60-Develop/40-project-services.md` | Persistent development databases/services in the Project | Native Compose/Podman/systemd administration, ports/listeners/private routing, SSH forwards, secrets, data volumes, readiness/logs and explicit restart. Factory uses assigned resources/test data. Ending a terminal does not stop services; no implied migrations, production deployment or disposal of human data. | S08 |
| D05 `60-Develop/50-canlang-development.md` | Develop Canlang applications with installed Project tools | Check/compile, retained private preview, nonempty examples and expected revision, LSP and separate full-profile development state where needed. Development fixtures never use production identity/data. Source/schema refusal is not a supported production migration. Bundle/runtime administration links to O04. | S08, S11 |

## Operate

| ID and exact file | Owns | Boundary and required detail | Evidence |
| --- | --- | --- | --- |
| O01 `70-Operate/10-administration.md` | Inspect appliance services, storage and operator health | Identify native host versus Docker contexts, components/data locations, service/log/capacity inspection and controlled maintenance entrypoints. Delegate run intervention, Project refresh, release transitions and recovery to their owners. | S09, S12 |
| O02 `70-Operate/20-cockpit.md` | Stock host or Soda runtime administration through Cockpit | Operator/root authentication, private 9090 access/tunnel, native logs/storage/network/services. Docker Cockpit administers the Soda runtime; the operator separately maintains the outer host. Runtime root password and host SSH credentials differ. No former custom Projects page or separate Soda Updates product. | S10, S12 |
| O03 `70-Operate/30-project-maintenance.md` | Explicit same-root support/tool refresh for retained Projects | Operation-specific prerequisites: terminal-support admits idle retained running/stopped roots; Muse/Canlang refresh requires an already running Project. Stop affected processes explicitly; Canlang refuses active `/opt/canlang` execution. Inspect holds, refresh and verify state without automatic Project Start or killing previews. Default-image updates do not refresh retained roots; no rebuild/recreate repair. | S06, S08, S12 |
| O04 `70-Operate/40-private-applications.md` | Package, install and administer independently managed private apps | Compatible bundle/image, stopped owner/member bootstrap, separate app accounts, operator registration/Install/Replace and administrator Inspect/Start/confirmed Stop. Optionally Link to a project, Open application and sign in app-locally; association changes navigation, not storage/lifetime. Same-data compatible replacement, unknown-outcome inspection, no automatic post-merge deploy. DNS/TLS is U02; stopped backup/fresh restore is O06. | S11 |
| O05 `70-Operate/50-updates-and-fallback.md` | Supported appliance release transitions and distinct Docker lifecycle | Native FCOS/Zincati/rpm-ostree ownership, release-pair admission, closed managed work, stopped roots, qualified support/extension transition, normal outcomes and explicit rollback limits. Interrupted journal recovery belongs exclusively to O08. Docker same-image recreation differs from version updates. Software rollback does not restore data; core committed/extension-uncertain outcomes are not automatic rollback. | S12 |
| O06 `70-Operate/60-backups-and-restoration.md` | Protect and restore the complete consistency set | Forgejo/Soda databases, retained Project roots/homes/volumes, configuration, broker custody and independent app state. Per-database dumps do not coordinate the appliance. Quiesce real writers, keep credentials separately and verify isolated restore. Stopped app restore needs compatible bundle/owner credentials/origin and a fresh data identity, never in-place rollback or retargeting an existing record. No one-click or SLA promise. | S11, S12 |
| O07 `70-Operate/70-data-safety-and-removal.md` | Identify what persists and the consequences of deliberate removal | Attempt cleanup versus checkout/Project data, source versus database/volume state, access removal versus data erasure and native deletion boundaries. Link backup first and identify actual actor. No unsupported archive/delete/rebuild programme or implicit destructive repair. | S06, S08, S11, S12 |
| O08 `70-Operate/80-recovery.md` | Recover interrupted host/release/native-writer operations | Exact operation and last confirmed state, maintenance journals, stopped/inhibited writers and offline native-domain reconciliation. Verify the whole writer domain, including scheduled/admin commands and external SSH writers; the helper controls only its fixed unit/container/marker. Preserve newer state and inspect uncertain effects. Run settlement stays in F10; data restore stays in O06. No automatic destructive recovery. | S12 |
| O09 `70-Operate/90-troubleshooting.md` | Route symptoms to a diagnosis and its repair owner | Separate login/permission, DNS/TLS/route, shared prerequisite, provider wait, CI, full storage, service, terminal and app failures. Collect bounded redacted diagnostics and show confirmed facts. Link the owning procedure instead of copying repair commands or presenting speculative retry loops. | S01–S12 |

## Required cross-page journeys

| Journey | Owning sequence |
| --- | --- |
| First appliance installation | I01 → I02 → (I03 or I04 → I03) → U01 → U02 |
| Docker installation | I01 → I02 → I05 → U01's Docker setup context → U02; never the FCOS-only configure command |
| First authorized build | F01 → F02 → F03, using U04/P02/P03/F05 → F04 → F06 → F07 |
| Observe and answer | P01 → F08 → F04 or F02 when the reply changes accepted direction; F09 for reusable knowledge |
| First human development | U03 → U05 using P02/P04 → D01/D03/D04; optional D02 for direct SSH/editors |
| Intervene and hand work back | F10 → P04/D02 under the human identity → F04/F06 with fresh evidence |
| Build and run a private app | D05 → O04, using U02; O06 for stopped backup/restore |
| Retain a Project across maintenance | F10/P02 → O03/O05 → P02/P03/P04; retained root is never a disposable retry mechanism |
| Recover an interruption | O09 → F10 for a run, O08 for host/native/release state, O06 for data restore |
