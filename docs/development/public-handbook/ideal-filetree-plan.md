# Public handbook filetree

Redesign `docs/public/` around the finished SodaOS product: a private software
factory appliance, native Forgejo collaboration, persistent repository Projects,
human development and independently administered private applications.

The current handbook has **six sections and 41 published pages**, plus the
unpublished source README. The initial refresh replaced the old five-section,
24-page structure with seven sections; the owner's subsequent reordering moves
everyday work ahead of operator setup and installation. All pages have been
written and the predecessor folders removed. The subsequent coverage and
usability follow-through revised every page. The
[handbook remaining tasks](cutover.md) now owns 18 prioritized tasks, complete
41-page allocation, outstanding screenshot captures and the website handoff.
It distinguishes completed source work from missing facts and live walkthroughs.

## Read the plan

| Document | Owns |
| --- | --- |
| [Complete target tree](proposed-tree.md) | Every final handbook filename and section |
| [Page contracts](page-contracts.md) | Each page's reader task, boundaries, source owners and exceptions |
| [Source investigation](source-audit.md) | Workflow traces, stale assumptions and independent boundary challenges |
| [Coverage and usability audit](coverage-audit.md) | Original 41-page review, source-backed findings and predicted task prominence; evidence rather than the current task queue |
| [Handbook remaining tasks](cutover.md) | Current priorities, completion criteria, all 41 page assignments, capture inventory and historical source cutover |

These records retain the source investigation, page boundaries, coverage
evidence and remaining handbook work in one living plan. They do not establish
product requirements or duplicate the Factory implementation backlog. Absorb
completed planning guidance into its owning documentation; retain the coverage
audit while its findings remain actionable.

## Inputs and authority

| Input | Inspected baseline |
| --- | --- |
| Checkout | `/home/vince/Projects/sodaos` |
| Source commit | `6c0e38ef1fa9530e4f96b7c6fca16dd342093178` |
| Initial working changes | None; source and existing public pages were clean |
| Structural inventory | 2,547 tracked repository paths; 25 under `docs/public/`, comprising 24 Markdown pages and the README; no tracked public image assets |
| In-scope allocation | The entire public handbook; relevant source workflows across frontend, Go APIs/coordinator, Rust host/installer/identity/release and operational configuration |
| Website reader | `/home/vince/Projects/soda-os-website`, clean at `4f36a9fb22284334e61f41827c9c8c5e02b6dfdb`; read-only investigation |
| Governing guidance | [Repository guidance](../../../AGENTS.md), [documentation authority](../../README.md), [source ownership](../ownership.md) |
| User direction | Refresh the entire public tree from deep source investigation; describe the final intended product rather than current WIP behavior |

The [product overview](../../product/overview.md),
[scope](../../product/scope.md), [Projects](../../product/projects.md) and
[Spaces](../../product/spaces.md) define the intended product. Architecture and
trust define its boundaries. Adopted design requirements add the complete
Factory/feed/library experience. Design alternatives still marked proposed do
not become settled requirements through inclusion in this plan.

Source establishes real responsibilities, callers, state and failures. It does
not make an obsolete workflow authoritative. In particular, current operator
commands, fixture UI, historical guides and implementation task priorities do
not narrow the final public product to what runs today.

The source refresh changes public documentation, its in-repository link consumers
and directly affected maintenance guidance. It does not run a service, modify
credentials, build a release, change a VM, publish documentation or mutate the
website checkout.

## Structural decisions

1. Organize by reader journey: **Start here → Factory → Projects and development
   → Configure → Operate → Install**. Navigation, first connection and optional
   provider enrollment come first. Factory begins with its overview and Feed;
   repository authorization, Project preparation and CI setup live together
   under Configure. Human contribution keeps the Project, browser terminal,
   Git, optional SSH, shared tools/services and Canlang sequence together.
   Operate starts with troubleshooting and places backup before maintenance
   and updates. Install keeps VM preparation before the common disk procedure
   and operator binding after the installation choices.
2. Preserve the website's source contract: `docs/public/`, numbered section
   folders, direct numbered Markdown children and globally unique page slugs.
   Folder nesting cannot represent deeper topics under the current reader.
3. Give each procedure one public owner. Start here owns personal provider
   connections; Configure owns Factory sponsorship and Project preparation.
   Factory consumes those grants and readiness without duplicating setup.
   Development services and private application runtimes have different
   lifetimes. Backups and host recovery have dedicated owners.
4. Retain the 22 old page slugs whose subject survives. Retire `dashboard` in
   favor of `spaces`; merge `forgejo` into `collaboration`. Their website
   references and historical fragments need an explicit cutover, not stale
   published compatibility articles.
5. Explain the nine intended Factory profiles, including Researcher and
   Library Admin, in the overview and relevant task pages. Writing is a
   Builder/Inspector specialty. Do not add a page per robot, MCP tool, internal
   schema or component variant. Research belongs with issue work; reusable
   knowledge and decisions get their own lifecycle owner.
6. Keep code contribution, public API internals, release construction, test-VM
   instructions, qualification diaries and backlog status outside the handbook.
   Public product boundaries remain useful: x86_64, Rocky headless, separately
   managed Actions capacity, private trusted teams and explicit app deployment.
7. Include no empty folders, placeholder pages, authored source manifest or
   hypothetical screenshot filenames. Add actual reviewed captures under
   `assets/` during writing when they explain the final interface.
8. Preserve all 41 semantic leaf slugs and article headings during the reorder:
   website page URLs and fragments stay unchanged. Renumbering and moving 39
   files changes sidebar groups and adjacent-page order. Update source links
   and current allocations together; preserve explicitly historical path
   evidence from the original audit and cutover.

## Decisions that affect writing

| ID | Evidence or conflict | Selected treatment and precise limit |
| --- | --- | --- |
| DEC01 | Old public pages describe standalone Soda sign-in, OAuth and a dashboard; scope retires them and the extension mounts native pages | Rewrite around native Forgejo identity, Workspace and Feed. No separate login or dashboard chapter. Backend names such as `soda-dashboard` remain valid service identifiers. |
| DEC02 | Old pages describe separate disposable agent environments; product and current preparation/lifecycle source use the persistent repository Project | One Project, separate human/coder/reviewer accounts and checkouts. Cleanup never becomes environment deletion. |
| DEC03 | The later accepted Factory contracts select nine profiles, including Researcher and Library Admin | Follow the current [product overview](../../product/overview.md). This supersedes the initial filetree audit's seven-plus-curation allocation; it does not add a new native account for each profile. |
| DEC04 | Research, repository writing and standalone content now have selected routes in the [workflow contracts](../factory-workflow-contracts.md) | Explain the intended outcome, evidence and acceptance boundaries. HB02–HB04 in the current task list clarify remaining prose and examples; missing exact UI facts must not produce invented controls. |
| DEC05 | Spaces is the product concept; the extension declares global Workspace, a persistent Workspace panel and repository Feed | Use those distinct surface names, preserve original Project/session context and avoid guessed URLs. Reconcile any later naming change with the extension declaration before capturing click paths. |
| DEC06 | Broader department/library sharing and exploratory Test Pilot execution remain proposed | Document creator-private departments, project-scoped library and approved verification manifests. Broader sharing or new verifier powers require a separate product decision. |
| DEC07 | Hardware minima, detailed firmware/device support and named-cloud images are not selected by current contracts | Explain qualitative deployment prerequisites and the common ISO path. Do not derive hardware requirements or cloud support from test-fixture settings. Any numeric support table needs canonical authority first. |
| DEC08 | Missing published media, selected upgrade receipts or installed integration proof are implementation/release gaps | Keep final deployment/update chapters. Their procedures select supported release metadata; omit WIP disclaimers and do not invent supported release pairs. |
| DEC09 | App RPO/RTO, automatic backup cadence and schema migration support are not established | Describe explicit stopped backups, fresh restore and compatible image replacement. No automatic backup SLA, in-place data rollback or migration promise. |

## Completion and evidence limits

| Dimension | Result for this documentation audit |
| --- | --- |
| Existing handbook coverage | Complete: every one of the 25 existing paths has a disposition in the cutover table |
| Established workflow tracing | The 12 capability slices in the source investigation have concrete entrypoint/state/authority/lifecycle traces. This is a documentation responsibility audit, not a claim to have reviewed every function in the 2,547-path repository. |
| Independent challenge | Completed for Factory/Spaces, setup/sponsorship, Project/application/storage and run/host recovery boundaries by the three specialist audits; coordinator reconciled their recommendations |
| Exact target allocation | Complete: 41 page owners plus README, allocated to writing packets and exclusive writers; unresolved contract details above gate only dependent instructions |

Performed verification is recorded in [cutover](cutover.md#verification).
No application tests, provider runs, browser journeys, installed operations or
release qualification were performed. Source inspection cannot establish those
results, and this plan makes no code-correctness or shipped-product claim.
