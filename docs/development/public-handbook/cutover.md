# Public handbook todo list

This is the current documentation task list for all **41 published pages**.
It supersedes the execution order in the [coverage audit](coverage-audit.md);
that report retains the original findings and source evidence. The completed
filetree cutover is retained below as history, not another open queue.

Public articles describe the **finished intended product in present tense**.
Implementation gaps stay in the existing Factory and Canlang backlogs. Missing
measurements, interface facts and live walkthrough results are inputs to these
documentation tasks; they must not become WIP disclaimers or invented commands.
At the owner's request, a `doc-pending` notice now identifies a specific missing
documentation fact or procedure. It does not change the selected product
behavior into an implementation-status claim.

## Completed baseline

The 20 October 2026 follow-through reviewed and changed every published page.
Keep these completed improvements; do not schedule the original 40 audit
findings as though none had been addressed:

- Seven sections and 41 stable page owners, with task-oriented entry links,
  single H1s, 140–220-character summaries and meaningful heading hierarchies.
- A continuous Factory example, human first contribution, precise authority
  and allowance explanations, preparation and SSH examples, and a compilable
  Canlang example with separate preview and language-server instructions.
- Fuller installation, private networking, app setup, updates, cold backups,
  restoration and diagnostic procedures describing the intended product.
- **17 Mermaid diagrams** and **11 screenshot briefs**. All screenshots still
  await capture; no completed image assets are attached.
- Source checks passed for all 41 articles, 243 local links, summary lengths,
  headings, retained anchors and visual JSON. All 17 diagrams parsed; the three
  changed diagrams were rendered and inspected. All 52 shell fences passed
  syntax checks, and the Canlang sample passed check and compile.

These are source-validation results, not executed installation, provider,
Factory delivery, SSH, update or restoration journeys. The next tasks preserve
that distinction.

## Current handbook order

The owner's subsequent reordering replaces the initial seven-section layout
with **six sections**, retaining all **41 articles**:

1. **Start here:** entrance, product model, Spaces, first connection and optional
   provider connections.
2. **Factory:** overview, Feed, planning, execution/dependencies, review,
   verification, knowledge and intervention.
3. **Projects and development:** Project model, browser terminals, collaboration,
   SSH/editor access, shared tools, services and Canlang development.
4. **Configure:** people/access, Factory authorization, Project preparation,
   CI runners, private networking and optional Tailscale.
5. **Operate:** troubleshooting, administration, Cockpit, private applications,
   backups, Project maintenance, updates, recovery and deliberate removal.
6. **Install:** deployment choice, download verification, VM preparation,
   common disk installation, Docker and initial operator setup.

The [complete tree](proposed-tree.md), [page contracts](page-contracts.md) and
[page allocation](#complete-page-allocation) follow that order. Page IDs such as
U05 and F03 remain stable references even when their chapters change. The
39 moved files retain their semantic leaf slugs, titles, summaries and headings;
all 41 website URLs and article fragments remain unchanged. Current source
links follow the moved pages. The original file-disposition table and
audit evidence remain explicitly historical.

**Website handoff:** resynchronize the source inventory and provenance, sidebar
sections and adjacent-page navigation. Review the directory component's
section-ID icon mappings for `start-here`, `factory`,
`projects-and-development`, `configure`, `operate` and `install`. The existing
generic folder fallback is available for unmapped groups. The source-specific
legacy `prepare-the-factory` compatibility anchor, if retained by the website,
must follow `20-Factory/10-software-factory.md`. Review stale generated-content
expectations against this inventory; no new page-route redirects are needed
for this reorder. HB18 also retains the `doc-pending` rendering handoff below.
No website or generated output is changed by this source reordering.

**Reorder validation:** the website's read-only source discovery and link
rewriter accept six sections, all 41 pages and all 350 public links. Page
contracts, the tree and this allocation match that exact order. Compared with
the pre-move snapshot, every article retains its title, summary, headings and
content except relative link destinations; all 22 Mermaid diagrams, 12 visual
briefs, nine pending-information notices and command/language examples are
unchanged. All 41 route slugs are preserved. The incoming-public/owner-document
link check passes for 157 links after two task links were normalized to their
explicit anchors. Whitespace checks include the moved files. No example,
diagram render or native product journey needed reexecution for these path-only
article changes. Website sidebar/adjacent-page presentation still needs the
receiving owner's synchronization; this is source validation, not publication.

## Pending information notices

The [source authoring contract](../../public/README.md#information-to-add-later)
defines strict `doc-pending` JSON and the website's rendering behavior.
Nine notices on six articles identify the information to add later, with
already-supported interim guidance. Each remains part of its existing task:

| Notice ID | Owning article and section | Existing task and needed input |
| --- | --- | --- |
| `measured-host-sizing` | [I01 — Select capacity](../../public/60-Install/10-deployment-options.md#select-capacity-for-your-workload) | HB10: R45/R52/R54 measured basic and small-team profiles, with update/backup reserve. |
| `muse-connection-id` | [D02 — Select a Muse connection](../../public/30-Projects-and-development/40-connect-and-develop.md#select-a-muse-connection-for-one-invocation) | HB06: supported ID discovery for owners and delegated members. |
| `native-factory-enrollment` | [F03 — Authorize repository work](../../public/40-Configure/20-factory-authorization.md#authorize-repository-work) | HB12/R44: corrected credential preparation, enrollment and recovery recipe. |
| `issue-requirements-acceptance` | [F03 — Authorize repository work](../../public/40-Configure/20-factory-authorization.md#authorize-repository-work) | HB12/R43: exact adoption/reacceptance controls and confirmed readback. |
| `native-files-client` | [D05 — Check access and retained results](../../public/30-Projects-and-development/70-canlang-development.md#check-access-and-retained-results) | HB08/P29: maintained authenticated upload, finalization, attachment and readback example. |
| `canlang-publisher-inputs` | [D05 — Assemble an application bundle](../../public/30-Projects-and-development/70-canlang-development.md#assemble-an-application-bundle) | HB07/P29: supported descriptor generation and matching environment preparation. |
| `canlang-image-build` | [D05 — Export a prepared application image](../../public/30-Projects-and-development/70-canlang-development.md#export-a-prepared-application-image) | HB07/P27–P29: matched runtime/tools acquisition and application image build recipe. |
| `release-content-preparation` | [O05 — Apply a qualified appliance release](../../public/50-Operate/70-updates-and-fallback.md#apply-a-qualified-appliance-release) | HB14/R45: preparation of locally required release content before preflight. |
| `offline-forgejo-invocation` | [O08 — Reconcile a native Forgejo mutation](../../public/50-Operate/80-recovery.md#reconcile-a-native-forgejo-mutation) | HB14/R04: complete packaged offline invocation with the retained deployment's mappings. |

Keep implementation tasks, absent walkthrough fixtures and unexecuted live
checks in their existing internal owners. Screenshots retain their `doc-visual`
placeholders. A notice is not evidence that its task is resolved, and supplying
an owner name is not a reason to remove it.

**Website handoff:** HB18 must add recognition and validation of this fence
before synchronization. The inspected website importer
`scripts/docs-sync/markdown.mjs` currently falls back to generic code rendering
for this language; it does not implement the component. The receiving owner
should render the fixed **Details to follow** label, topic, message and optional
guidance as a static accessible note, without raw JSON, code controls or the
source-only ID. Keep notes out of headings, summaries and adjacent-page previews.
The shared template should adopt this contract through its website owner.
No website source or generated output was modified here. Valid notices may
render while the underlying inputs remain outstanding; renderer work need not
wait for those inputs.

**Source validation:** all nine blocks pass strict JSON, allowed-field, string,
ID uniqueness, placement and task-mapping checks. Eight public articles changed:
six contain notices and two installation callers now link to capacity planning
without implying measured profiles are already available. All 41 articles
retain their summaries and headings; all 350 local links resolve. Existing
commands, language examples, 22 Mermaid diagrams and 12 visual briefs are
unchanged. Whitespace checks pass. No website component was rendered or
published; that verification belongs to the receiving implementation.

## Priorities and execution order

There are **18 tasks: six complete and 12 open**, each with scope and completion criteria below.
**P1** closes a consequential ambiguity or a gap in a core reader journey.
**P2** improves supporting examples, recognition or consistency. **P3** is the
separate website handoff after the source work.

### What can proceed now

The 12 open tasks are the remaining total. **HB17 has source work ready to do.
HB15 and HB18 have partial preparation or review work available. The other
nine await specific inputs for their remaining work.** HB16 has integrated all
currently available material; HB08's source clarification is included in that pass.
Readiness to make progress differs from readiness to close a task: later facts,
assets and live results still need integration and their affected checks.
Several waiting tasks already have completed source improvements.

Use this table to choose the next task. The dated results below retain prior
work and its limits; an earlier “continue with” note does not override this
current order.

| Open task | Current readiness | Remaining documentation action | Input needed to finish |
| --- | --- | --- | --- |
| [HB06 — Muse connection selection](#hb06-make-choosing-a-muse-connection-self-contained) | Waiting for interface | Document where owners and delegated members find a connection ID; verify two choices and a rejected selection. | Supported ID-discovery screen or command from the identity/Project interface owner. |
| [HB07 — Publish a Canlang app](#hb07-connect-canlang-source-to-a-prepared-private-application) | Waiting for publisher recipe | Complete and execute descriptor generation, matched runtime/tools acquisition, image construction and operator handoff. | Supported author-facing recipe and inputs from Canlang P29 with P27/P28 release outputs. |
| [HB08 — Full-profile development example](#hb08-add-a-full-profile-canlang-development-example) | Source clarification integrated; runnable remainder waits | Once the client route exists, finish authenticated upload through attachment/readback and execute the native example. | HB07 bundle preparation plus a maintained upload-client procedure or browser-form example from Canlang P29. |
| [HB10 — Hardware sizing](#hb10-publish-measured-deployment-sizing) | Waiting for measurements | Publish measured basic/small-team profiles, workload assumptions and update/backup reserve. | Actual R45 sizing observations and R52/R54 release integration; the owner has no measurements to supply. |
| [HB11 — Installation walkthrough](#hb11-follow-installation-through-a-working-private-client) | Waiting for installation targets and accounts | Follow fresh appliance and Docker installation through authenticated browser/Git/Project use; correct observed instructions. | Disposable installation target, Docker-qualified release/host, private client and documentation accounts/repository. |
| [HB12 — Daily reader journeys](#hb12-follow-the-daily-author-and-contributor-journeys) | Waiting for fixture, enrollment recipe and interface | Execute author/contributor, Factory, Feed and terminal journeys; fix observed confusion and record actual outcomes. | Current documentation fixture, safe sample work, R44's corrected native enrollment recipe and completed intended interface; the owner confirmed no fixture is available. |
| [HB13 — App setup and Restore](#hb13-follow-app-setup-and-same-address-restoration) | Waiting for app path and fixture | Execute owner/member setup, stopped Install, Start, backup and same-address Restore with recognizable data. | Prepared app, private TLS, accounts/snapshot and completed P14.4/P25/P26 behavior, including the storage-identity correction. |
| [HB14 — Maintenance and recovery](#hb14-make-maintenance-and-recovery-results-recognizable) | Waiting for recipes and native evidence | Complete executable recovery instructions and observe maintenance, cold restore, access removal and qualified update/fallback. | R04 offline Forgejo invocation; R45 release-content preparation and qualified pair; isolated recovery/two-member fixtures and R44.1/R52 evidence. |
| [HB15 — Screenshots](#hb15-complete-the-existing-screenshot-briefs) | Brief review available; capture each view when ready | Reconcile existing briefs with final prose and controls. Capture each available specified view independently, inspect annotations/accessibility and attach its real asset. | Each view's actual interface, documentation access and public-safe sample state. Current sign-in observations satisfy none of the 12 briefs; captures need not wait for every walkthrough. |
| [HB16 — Integrate all 41 pages](#hb16-integrate-new-material-across-all-page-owners) | Current-source integration done; later inputs wait | Integrate the remaining owner-supplied facts, procedures, walkthrough corrections and captures as they arrive; recheck affected callers. | Reviewed all 41 pages and changed 28 in this pass. Final closure includes the later inputs; do not repeat the completed all-page review without relevant changes. |
| [HB17 — Validate handbook source](#hb17-validate-the-changed-handbook-source) | **Ready now — next task** | Complete source validation using the HB16 results for unchanged inventory, summaries, links, visual JSON and examples; check remaining affected diagrams/examples and record live checks separately. | No new external input for source validation. Validate later-added captures or commands when those changes arrive. |
| [HB18 — Website handoff](#hb18-hand-off-the-completed-source-to-the-website-owner) | **Preparation can proceed** | Prepare the route/anchor and rendering handoff from current source, including explicit unresolved inputs. | Final closure needs integrated/validated source, its exact revision and website-owner verification. This docs task does not modify or publish the website. |

The detailed checkboxes below remain the completion record. Input dependencies
are also todos: their existing product owners produce the missing behavior,
recipe, measurements or fixture; the linked handbook task then incorporates and
verifies that result. Do not mark a task complete just because its blocker has
been recorded, and do not hold independent writing behind a missing capture or
runtime witness.

Expected frequent tasks are following work, answering questions, understanding
waits, opening a terminal and reviewing a result. This is a hypothesis from the
workflow, not measured usage. Recovery and access removal remain easy to find
because their consequences matter even when use is infrequent.

1. Complete HB17's remaining affected source checks, preserving HB16's
   current-source integration and passing checks. Work one task per goal turn;
   preserve the completed HB01–HB05/HB09 results.
2. Prepare HB18's handoff within this repository. Keep its final website
   verification open until the exact source and receiving owner are ready.
3. Resume HB06–HB08/HB10 when their named owners supply missing facts or
   procedures. Reconcile additions with HB16 and rerun only affected HB17 checks.
4. Combine HB11–HB14 walkthroughs with HB15 captures when their real environments
   are available. Reuse exact product-owner observations; a missing fixture
   does not require repeating a completed source review.

Documentation writers own prose, examples and Mermaid. Interface/release owners
supply missing facts and native observations. Captures require the actual
documented interface. Only the affected tasks wait on those inputs. None of
these tasks creates a new runtime feature, product decision gate or permission
to publish.

Page IDs below resolve to exact filenames in the
[complete page allocation](#complete-page-allocation).

## Core journeys and missing examples

<a id="hb01-make-everyday-tasks-easy-to-find"></a>
### HB01 — Make everyday tasks easy to find

- [x] **P1 · Completed 20 October 2026.** Scope: H01/H02, P01, F08–F10 and O09.
  Preserve the daily-task links already added. Check that a request author,
  contributor and operator can enter their own journey without following the
  installation sequence first.
- **Done when:** “What needs me?”, “Why is it waiting?”, “Open my terminal”,
  “Start or revise work”, “Review a result”, “Find a decision” and “Recover an
  interruption” each lead directly to their owning instructions. Explain
  repository/Project/session/app and permission boundaries at the point of
  action. Preserve current routes and chapter order; add no manual contents list.
  Use the reader walkthrough in HB12 to correct confusing link wording.
- **Result:** reviewed all seven scoped pages and changed five. H01 and P01 now
  link all seven everyday tasks directly to their owning instructions; H01 also
  exposes app administration and access removal. Product concepts link to their
  procedures, Feed guidance distinguishes a question for the reader from a
  resource wait, and troubleshooting routes directly to the affected operation.
  Corrected P01's permission wording: authorized Feed reading differs from live
  terminal observation, which needs code-write access. Checks passed for 96 local
  links, both sets of seven task entrances, summaries, hierarchy and retained
  anchors; diagrams and examples are unchanged. Whitespace checks passed.
  HB12 still owns observation of unfamiliar readers following the actual
  interface and any resulting wording corrections.

<a id="hb02-remove-remaining-factory-acceptance-ambiguities"></a>
### HB02 — Remove remaining Factory acceptance ambiguities

- [x] **P1 · Completed 20 October 2026.** Scope: F02–F04.
  In F04, qualify “The authorized owner decides whether the parent can proceed”:
  research that satisfies an accepted prerequisite leads to fresh readiness
  assessment under standing policy. Human input is needed for a remaining
  material choice or waiver, not after every research answer.
- **Done when:** supported positive or negative answers can satisfy an
  investigative question; negative evidence cannot satisfy a required working
  capability; inconclusive evidence remains unresolved. Align F02's “changes
  to accepted revisions” with F03's exact title/body, selected-comment and
  dependency rules. Unrelated discussion or reactions do not invalidate work.
- **Result:** F02 now names the accepted inputs and links to F03's exact
  invalidation rules. F04 distinguishes submitted research, required non-code
  acceptance and automatic reassessment; it preserves maintainer acceptance
  and the dependent approver's confirmation without adding another approval
  after the result is accepted. Positive, negative and inconclusive outcomes
  retain their different consequences. Both changed articles passed summary,
  heading/anchor preservation and 18 local-link checks; existing diagrams and
  examples are unchanged. Whitespace checks passed. No factual input remains
  for HB02; native journey validation stays with HB12.

<a id="hb03-show-the-full-factory-procedure-with-focused-mermaid-views"></a>
### HB03 — Show the full Factory procedure with focused Mermaid views

- [x] **P1 · Completed 20 October 2026.** Scope: F01 and the relevant F02–F10 stage owners.
  Extend the existing outcome-route overview with a readable step-by-step
  execution view: accepted work, readiness and task plan, checkpoints,
  independent review/correction, current required checks, confirmed merge and
  original issue closure, dependent reassessment and applicable integrated
  verification. Keep deployment explicit and separate.
- **Done when:** readers can follow the happy path and find the relevant
  branches for clarification, existing work/already-satisfied outcomes, research,
  standalone content, stale evidence, a wait, correction, pause/resume and
  cancellation. Put detailed branches beside their owning explanation instead
  of one unreadable graph. Selected alternatives alone become executable work;
  waiting never implies delivery and cancellation requests never imply confirmed
  settlement. Preserve nine role responsibilities without inventing a
  Copywriter role or per-stage human approvals. Every diagram has single-line
  `accTitle` and `accDescr`, ordinary supported syntax, and no presentation
  directives. Explain its takeaway in prose; parse and inspect changed renders.
- **Result:** six Factory pages changed, with five added diagrams and five
  revised diagrams. F01 adds the accepted-issue-to-delivery path; F02 separates
  current/existing outcomes from input and approach selection; F06 adds the
  independent-review/correction path; F07 shows verification outcomes; F10
  distinguishes pause/cancel requests from confirmed settlement and eligible
  resume. Existing diagrams now retain checkpoint versus issue-delivery
  boundaries, required non-code acceptance, repository-wide reassessment and
  complete-intent verification readiness. Corrected prose that had allowed
  children to continue after a terminal task failure: that run stops remaining
  children and preserves checkpoints.
  All 11 Factory diagrams parsed; all 10 changed diagrams rendered with
  accessibility fields and were visually inspected at article width. Split the
  wide planning view and simplified verification labels after inspecting their
  first renders. Structure, summaries, retained anchors, 56 local links and
  whitespace checks passed across the affected chapter; existing command
  examples and four screenshot briefs are unchanged. The handbook now contains
  22 Mermaid diagrams. No factual question remains for HB03; these checks do
  not establish native Factory execution, which stays with HB12.

<a id="hb04-work-through-research-and-standalone-writing-outcomes"></a>
### HB04 — Work through research and standalone writing outcomes

- [x] **P2 · Completed 20 October 2026.**
  Scope: F02/F04/F06/F08/F09.
  Add a small complete non-code example alongside the existing catalog
  interface-copy example: a sourced research answer or standalone document,
  with a brief, acceptance criteria, evidence, correction and confirmed outcome.
- **Done when:** readers can distinguish repository documentation/copy changes
  from standalone content and an investigative result. Builder/Inspector
  specialties own writing and checking; a new agent role is unnecessary.
  Show what makes claims supported, copy fit its audience and an artifact
  accepted. Do not require a fictional PR, app verification or publication for
  standalone delivery. Link reused knowledge and its authority to F09.
- **Result:** F04 now contains two complete fictional catalog examples: a
  sourced, revision-bound research answer with correction and prerequisite
  acceptance, and a standalone Markdown tip with an executable brief,
  independent editorial/factual review, exact-artifact acceptance and native
  closure. The final tip has 51 body words within its stated 40–70-word limit.
  F02 links both routes and scopes required CI to repository-change delivery;
  F06 distinguishes source changes from standalone results; F08 connects
  artifact inspection to acceptance evidence; F09 retains the boundary between
  reusable findings and authorized product decisions. The examples use existing
  evidence and Feed procedures without guessing acceptance controls.
  All five pages passed summary/hierarchy, retained-anchor, 41 local-link and
  whitespace checks; the six existing Mermaid diagrams and four screenshot
  briefs are unchanged. Native walkthrough evidence remains with HB12.

<a id="hb05-complete-personal-git-authentication-before-the-first-contribution"></a>
### HB05 — Complete personal Git authentication before the first contribution

- [x] **P1 · Completed 20 October 2026.** Scope: U05, D01–D03 and U03.
  Complete the existing browser-first contribution with one supported personal
  HTTPS authentication route, linked at its actual prerequisite.
- **Done when:** readers can obtain appropriately scoped native credentials,
  enter/store them safely, clone and push under their own account, and recognize
  successful authentication. Distinguish forge Git credentials from Project
  SSH keys, provider credentials and Factory identities. Verify native token
  scopes and the supported storage recommendation rather than inventing them.
  Keep generic Forgejo account administration in its upstream owner.
- **Result:** reviewed and changed all five scoped pages. D01 now gives the
  native Settings → Applications token procedure: select the intended
  repository, set repository Read and write (`write:repository`) and retain
  the once-displayed token in a personal password manager. The native token
  form and Git HTTP scope checks confirm the route; upstream Forgejo and Git
  references are linked in the article. Clone and push use password-prompt
  entry with configured Git credential helpers disabled for those commands,
  without assuming a Project keyring or storing a token in shared files.
  The example includes checkout-local commit identity, observable branch/PR
  results, public anonymous-clone behavior, token replacement/deletion and
  distinct authentication, permission, branch-policy and TLS failures.
  U05 links token setup before clone; U03/D02/D03 preserve separate Project,
  personal Git, provider, CLI and Factory identities and revocation boundaries.
  Added one safe screenshot brief for the token form before generation,
  included in HB15 below. All five articles passed summary, heading, retained
  anchor and 48 local-link checks; eight shell fences passed syntax checks,
  visual JSON is valid and whitespace checks passed. No factual input remains
  for HB05. Actual token creation and native clone/push/PR observations remain
  with HB12; no native authentication journey was executed for this task.

<a id="hb06-make-choosing-a-muse-connection-self-contained"></a>
### HB06 — Make choosing a Muse connection self-contained

- [ ] **P1 · Missing supported discovery surface.** Scope: U04 and D02.
  The command using `SODA_MUSE_CONNECTION` exists, but the handbook does not yet
  tell readers where to obtain the real connection ID.
- **Done when:** someone with two authorized connections can locate their real
  identifiers, distinguish labels/owners/providers, select the intended one
  and understand a rejected selection. Identify the supported screen or command
  with the interface owner; do not expose an internal database workaround or
  invent a UI control. Preserve the simple single-connection path.
- **20 October investigation — open, awaiting interface-owner input:** reviewed
  and improved both scoped pages. U04 now distinguishes connection owner, label,
  provider and ID; D02 explains single-ready-connection selection, explicit
  selection without fallback and separate ambiguity, authority, launch-service
  and provider-credential failures. Removed the unsupported instruction to
  obtain an ID from an unspecified connection record or operator.
  The native selector in `lib/host/src/muse/connection.rs`, its broker caller
  in `lib/host/src/daemon/broker.rs`, and the launcher confirm these semantics.
  The discovery requirement remains unmet:
  - `frontend/spaces/soda-identity.ts` displays owner connection labels with IDs
    only in hidden option values; selected details omit the ID. It fetches the
    Project-authorized inventory but displays only Codex in its launch selector.
  - `frontend/spaces/soda-providers.ts` displays provider/account/status details,
    without a connection ID. The Factory sponsorship selector also keeps the
    ID in an option value. The Project Muse launcher has no connection-list
    command.
  - The owner and Project-authorized inventory APIs return the required metadata,
    but use private prepared native transport. They are not a supported public
    URL or end-user `curl` recipe; no database, browser-inspector or private-API
    workaround was added to the handbook.
  **Required input:** the identity/Project interface owner supplies the intended
  supported screen or command that exposes each authorized Muse connection's
  exact ID, provider, recognizable account label and owner to both the owner
  and a delegated Project member. Then document that navigation, verify two
  eligible choices plus a rejected selection, and complete HB06. This question
  has been sent to the owner. No missing-interface disclaimer appears in public
  articles. Both articles passed summary/hierarchy, retained-anchor and 15 local
  link checks. Four command examples, the Mermaid diagram and the screenshot
  brief are unchanged; visual JSON still parses. Whitespace checks passed.
  No provider launch or native selection walkthrough was executed. Continue
  with HB07 while this task waits.

<a id="hb07-connect-canlang-source-to-a-prepared-private-application"></a>
### HB07 — Connect Canlang source to a prepared private application

- [ ] **P1 · Publisher interface/recipe input needed.** Scope: D05 and O04.
  Extend the working `app.can`/`app.json` example through the supported packaging
  route and trusted operator handoff. The current operator guide starts with an
  already prepared OCI image.
- **Done when:** an app author can locate the supported runtime/recipe inputs,
  use the applicable `can-soda build` and image-construction steps, export the
  exact image and supply the identities the operator needs. Explain every input
  and expected artifact, then link to existing load/setup/Install/Start steps.
  Resolve the publisher-facing entry point and distribution of required inputs
  with their owner. Do not invent a manifest, registry, packaging wizard or
  implicit deployment step.
- **20 October investigation — open, publisher recipe unspecified:** reviewed
  and improved D05/O04. D05 now includes packaging compilation with the installed
  native catalog and scenario receipts, and names the artifact, deployment
  descriptor and environment inputs, separate Identity/state bindings, private origin,
  `can-soda inventory`/`build`, exclusive output creation and the successful
  bundle result. It adds the supported Podman OCI-image export command.
  O04 lists the publisher's archive/provenance, exact image, app/source,
  artifact, module bundle, storage, runtime/tools and origin references, and
  links that handoff to existing setup/Install/Start instructions.
  These are partial steps, not a complete source-to-image tutorial.
  - Canlang's `packages/cloudflare/src/cli/soda.ts` and
    `deploy/soda.ts` confirm that build consumes existing descriptor/environment
    files and assembles installed modules. Its integration driver constructs a
    descriptor, but neither that driver nor hand-authored internal records
    establish an installed author-facing generation command.
  - Soda's `system/containers/canlang/Application.Containerfile` is the existing
    app-layer recipe. The release includes a selected runtime image and tools
    archive identity. A supported publisher workflow for acquiring that exact
    set and producing the descriptor remains unspecified; the owner could not
    supply a recipe.
  **Required resolution:** Canlang publishing and Soda release owners supply
  the complete supported descriptor-generation and matched-input acquisition
  procedure. This remainder is now explicit in the existing
  [Canlang P29 task](../factory-remaining-tasks.md#canlang-p29), consuming P27/P28's
  admitted release outputs. Then finish and execute the application-layer build
  and source-to-operator example before closing HB07.
  Both articles passed summary/hierarchy, retained-anchor and 12 local-link
  checks. The three new shell fences passed syntax checks; the prior Todo source,
  six command fences, Mermaid diagram and two screenshot briefs are unchanged.
  Visual JSON and the new P29-to-HB07 link passed; whitespace checks passed.
  Packaging compilation, bundle assembly, image build/export and native
  installation were not executed.
  No new screenshots or assets were added. Continue with HB08.

<a id="hb08-add-a-full-profile-canlang-development-example"></a>
### HB08 — Add a full-profile Canlang development example

- [ ] **P2 · Canonical example and commands needed.** Scope: D05 and O04.
  Retain the simple Todo preview. Add a separate worked route for an application
  that needs the full Soda profile, such as authorized file access.
- **Done when:** the selected real example explains configuration, isolated
  development data, private owner credentials, HTTPS, the expected access
  result and safe cleanup. It makes clear which preview can exercise which
  capability and keeps production identities/data out of development. Obtain
  the actual example and commands from the Canlang owner; link language detail
  rather than duplicating that handbook.
- **20 October result — open, client procedure and bundle preparation needed:**
  reviewed and changed D05/O04. D05 now includes the maintained
  `scripts/fixtures/soda-runtime.can` source, private development directories,
  owner/member credential requirements and bootstrap commands, the team ID
  distinction, fixed-port HTTPS forwarding, expected owner/member file access,
  retained scheduled work and clean shutdown. O04 links to the development
  example while keeping installed app identities and storage separate.
  The example's page only lists entries; no upload button or fixture-seeding
  endpoint is claimed.
- **Corrected source fact:** ordinary `can dev` supports Files through its local
  bindings; the previous blanket Files refusal was stale. Both D05 and the
  existing maintainer Canlang guide now distinguish that preview from native
  Soda Files storage and durable scheduled work. Canlang's selected source
  already contains `planLocalBindings()` and Files R2 setup in
  `packages/cloudflare/src/dev/{preview-builder,local-bindings}.ts`; its tools
  wrappers use the same preview implementation. Scheduled handlers and
  migrations still require their owning runtime.
- **Remaining input:** HB07's supported descriptor/bundle preparation route is
  still missing. The maintained native Files fixture also needs a public
  authenticated upload-client procedure, or a maintained browser form example
  with verified controls. Its integration driver supplies source evidence for
  upload intent → bytes → finalization → attachment and access results, but is
  not a standalone reader command. No published language-handbook procedure
  covers that client sequence. These omissions are recorded under existing
  [Canlang P29](../factory-remaining-tasks.md#canlang-p29); do not mark this as a
  complete, executed source-to-file tutorial.
- **Checks:** both published pages retain their original 211/196-character
  summaries and anchors; heading hierarchy and all 15 local links passed.
  Four new shell fences passed `bash -n`; the nine existing shell fences are
  unchanged. The new Canlang sample exactly matches its maintained fixture and
  passed check and compilation with the native catalog and scenario receipts.
  The existing Todo example, one Mermaid diagram and two screenshot briefs are
  unchanged; visual JSON parses. No new visuals or assets were needed.
  A separate source review found no factual or credential/storage-safety issue.
  The linked language permissions page was read locally, but its GitHub URL
  could not be fetched by the web tool. Bootstrap, serving, HTTPS, authenticated
  upload and restart were not executed. Whitespace checks passed. Continue
  with HB09.

<a id="hb09-replace-service-placeholders-with-one-usable-worked-example"></a>
### HB09 — Replace service placeholders with one usable worked example

- [x] **P2 · Completed 20 October 2026.** Scope: D03/D04 and P03.
  Keep the general Compose/systemd reference, but connect it to one real,
  maintained development-service example and its actual health check.
- **Done when:** readers can follow administrator setup, member endpoint access,
  persistent storage, assigned Factory test data and restart verification using
  concrete sample values. Explain expected observations and cleanup that
  preserves human data. Link preparation to that service where relevant;
  ending a terminal or running a test must not imply deleting shared state.
- **Result:** reviewed and changed all three pages. D04 contains a worked
  PostgreSQL service derived from `tests/fixtures/workload/compose.yaml` and
  `tests/installed/workloads.sh`: the same exact image, native secret and named
  volume, with a loopback-only sample port, explicit Compose project identity,
  private administrator setup and the actual `pg_isready` probe. Separate
  non-superuser human/coder/reviewer database owners and revoked default public
  access replace use of one shared administrator credential. Member SQL,
  SSH-forwarded access, private role password files, a transactional readiness
  probe, explicit restart/readback and preservation-focused cleanup complete
  the example. The readiness command fails on an incorrect database or
  cross-database grant rather than merely printing the unexpected value.
  D03 links shared-service access to these data boundaries; P03 connects the
  example to accepted source, administrator prerequisites and separate role
  readiness.
- **Checks:** all three summaries remain unchanged at 190/203/212 characters;
  hierarchy, retained anchors and 19 local links passed. Eight new shell fences
  passed `bash -n`; six existing shell fences, one Mermaid diagram and one
  screenshot brief remain unchanged. Visual JSON and whitespace checks passed.
  Compose syntax parsed; Docker Compose warned that its non-Swarm mode ignores
  the Podman secret UID/GID/mode options, so this is not evidence of equivalent
  Docker secret behavior. The example targets the Project's Podman provider.
  No new visual was needed; SQL and the account/data table explain the task.
- **Executed behavior and limits:** an isolated local Podman container using the
  exact fixture image passed the documented role/database SQL, hidden password
  prompts, TCP/password authentication, committed member write, both role probes,
  actual denial of Factory connections to the human database and explicit
  stop/start with the retained row. An intentionally incorrect human-database
  grant failed the updated readiness command and was reverted. The task-owned
  container, test volume, native secret and generated credentials were removed
  afterwards. Local validation used container loopback with external networking
  disabled after the host's rootless network helper could not open its network
  namespace. Installed Project networking, the packaged Podman Compose provider,
  a real SSH forward and Factory credential delivery were not executed; HB12
  retains that composed reader journey. No unresolved product fact remains for
  HB09. Continue with HB10.

<a id="hb10-publish-measured-deployment-sizing"></a>
### HB10 — Publish measured deployment sizing

- [ ] **P1 · Measurements needed.** Scope: I01/I04/I05 and O04.
  Replace qualitative sizing indirection with the selected basic and
  recommended small-team profiles once release owners provide observations.
- **Done when:** CPU, RAM, disk, workload/concurrency and update/backup reserve
  have a stated measured basis readers can use. Distinguish host capacity from
  fixed per-app ceilings. Link the supported release information and explain
  the workload assumptions. Do not derive minima from fixture allocations,
  promise an unmeasured user count or invent cloud support.
- **20 October investigation — open, measured profiles unavailable:** reviewed
  capacity guidance in all four scoped public pages; no article changes were
  warranted without measurements. I01 already distinguishes workload concurrency,
  retained data, update/backup reserve and Docker host overhead. I04/I05 link to
  that guidance, and O04 correctly describes per-app caps rather than host
  reservations. No numeric minimum or recommended host allocation was added.
- **Evidence boundary:** the inspected release/native documentation provides
  build-worker limits, media-build scratch, acceptance-VM allocations and a
  compressed rootfs ceiling, not measured installed workload capacity. The
  existing release qualification records cover platform, identity, execution
  and persistence without supplying the missing sizing observations. Historical
  Canlang component receipts referenced by the runtime plan were unavailable at
  their named local paths; their documented scope does not establish these
  profiles. `lib/host/src/application/runtime.rs` confirms the separate 1 GiB,
  two-CPU and 128-process/thread application limits.
- **Required input and existing owners:** follow
  [supported sizing under R45/R52/R54](../factory-remaining-tasks/06-candidate-release-closure.md#supported-sizing).
  R45 must supply the actual revision/distribution and hardware, allocated
  CPU/RAM/disk, defined basic workload, and the selected small-team workload
  of two active Factory issues across Projects, one human terminal and one
  small private app. Include actual Project/app counts, provider concurrency
  constraints, peak resource use, startup/task responsiveness, installed and
  retained disk use, and update/rollback/backup free-space observations.
  Distinguish Docker host overhead and record observed low-resource failures
  and recovery. R54 incorporates measured guidance before R51; R52 confirms it
  on the frozen candidate for both installation forms. These existing owners
  remain responsible; HB10 does not create another benchmark or release gate.
- **Next action and checks:** the owner confirmed that no sizing measurements
  are available. Once the existing sizing owners supply measured results,
  put the measured profiles and their scope in I01, link the exact
  release information, and reconcile I04/I05/O04. The public pages, examples,
  diagrams and visual blocks are unchanged, so their passing checks were not
  rerun. Task-reference and whitespace checks passed. No deployment, capacity
  benchmark or qualification ran. Continue with HB11 while this input is pending.

## Walkthroughs against the intended interface

<a id="hb11-follow-installation-through-a-working-private-client"></a>
### HB11 — Follow installation through a working private client

- [ ] **P1 · Source corrections and partial private-client checks done;
  fresh installation walkthroughs remain.**
  Scope: I01–I05, U01/U02/U04/U06 and O01/O02.
  Follow the written appliance and Docker paths from verified download through
  setup and activation to browser trust, native Git and Project access from
  the intended private client. Exercise optional provider/Tailnet paths when
  they are part of the documented scenario.
- **Done when:** prompts, command contexts, ownership, certificate transfer,
  network endpoints and success/failure observations agree with the article.
  Fix undocumented steps and misleading examples. Record exactly what was
  exercised and what was not; localhost success does not prove private-client
  access. Take matching HB15 captures during this work.
- **Result:** reviewed all 11 scoped pages and changed U01/O02. U01 now
  explains the durable-configuration/token-revocation failure: inspect the
  administrator's native token settings, revoke the bootstrap token if still
  active, confirm revocation and continue activation without replaying setup.
  O02 corrects Docker's Cockpit tunnel for host networking and adds the
  runtime's separate root-password setup, command context and expected result.
  Sources: [setup](../../../cmd/soda-setup/src/setup.rs),
  [Docker Compose](../../../system/docker/compose.yaml),
  [Cockpit listener](../../../system/host/config/cockpit.socket.conf) and
  [Docker operator guide](../../guides/docker.md).
- **Executed on 2026-10-20:** selected the published appliance release
  `v0.1.2`, downloaded its `sodaos-x86_64.iso` into a fresh directory and ran
  I02's `gh release verify-asset` command. GitHub's attestation verification
  passed for SHA-256
  `d9383da2638dcac642ba1d9451137ba5d3ff62ed9372e3b095716121363f4a6f`.
  The release supplies appliance assets; its notes/assets do not provide a
  Docker installation handoff.
- **Retained-host observations:** pinned SSH reached the retained appliance.
  Installed release metadata identifies `v0.1.2`, and the running Forgejo
  image ID matches that metadata's Forgejo config digest. Forgejo, the host
  socket and proxy are active. The previously saved client CA failed TLS
  verification. Retrieving only the current public CA over pinned SSH and
  comparing its SHA-256 fingerprint on host and client succeeded. With that
  verified CA, private-address HTTPS returned 200 and `/api/healthz` reported
  cache/database success. The advertised native Git SSH endpoint on port 2222
  answered a public-host-key scan; the scan did not install or establish trust
  in the key. No Project container was running.
- **Limits and remaining input:** these are read-only observations of an
  existing installation. They do not establish a fresh ISO boot/disk install,
  guided enrollment, setup failure recovery, browser trust-store sign-in,
  authenticated clone/fetch or Project access. The HTTPS checks used an
  explicit verified CA file, not a browser trust-store change. Docker Cockpit
  password setup and forwarding were checked against source, not executed on
  a Docker deployment. Complete the remaining journey with a disposable
  appliance installation target, a selected Docker-qualified release and
  suitable Docker host, then native accounts/sample repository and a running
  Project reachable from the client. Do not reinstall the retained VM to
  obtain these observations. Optional provider/Tailnet enrollment and HB15
  captures were not exercised; no image assets were added.
- **Checks:** both changed articles retain their 183/182-character summaries
  and all existing headings/anchors. Heading hierarchy and nine local links
  passed. The changed tunnel and new password command passed shell syntax
  checks; nine other shell examples and the Mermaid diagram are unchanged.
  No visual blocks changed. Task links and whitespace checks passed.
  Task scratch and validation output are under
  `/home/vince/Projects/.sodaos-handbook-goal-20261020/HB11/`.
  HB11 remains open; continue with HB12 rather than retrying the unavailable
  fresh-install paths.

<a id="hb12-follow-the-daily-author-and-contributor-journeys"></a>
### HB12 — Follow the daily author and contributor journeys

- [ ] **P1 · Source walkthrough improved; current documentation fixture and
  completed intended interface needed for live qualification.**
  Scope: H01/H02, U03–U05, F01–F10, P01–P04 and D01–D05.
  Use the existing catalog request and human contribution examples. Follow
  request creation, preparation, Feed question/answer, dependency/check evidence,
  independent review, delivery and integrated verification; separately follow
  Join, personal checkout, contribution and terminal reconnect.
- **Done when:** a reader unfamiliar with the implementation can use the public
  instructions and recognize success without private source knowledge. Include
  representative wait, changed-input and permission situations, plus locating
  a Library decision. Check that routine progress needs no invented approval.
  Exercise the exact command examples where applicable and capture only real
  observations. Use concrete confusion to revise HB01/HB02/HB05/HB06 rather
  than require a new usability programme or claim measured readership.
- **Result:** reviewed all 24 scoped articles and changed 11: U04/U05,
  F02/F03/F08/F10, P02/P03/P04 and D04/D05. Added exact request creation,
  save/resume and repository-continuation controls; explained that repository
  continuation requests native work generation. Located Factory authority
  settings and the separate owners' save actions. Named provider enrollment
  navigation, moved the Project running-state/key choice before Join, and
  explained the single role-preparation action with independent coder/reviewer
  results. Terminal instructions now distinguish a new shell from reconnecting
  the original session. Corrected the inline psql quit command and made the
  local Canlang preview's returned URL, actors and team choice explicit.
- **Discussion and intervention:** F08 now explains saving a combined Feed
  subscription, opening a post's Comments, using a suggested answer as an
  unsent draft, and selecting Post. Role replies, recorded answers and
  comment-intake Answer clarification/Submit answer remain distinct from
  authorized issue acceptance. F10 names repository, run and shared-Project
  controls, their scopes and pending readback, preserving the broader intended
  cancellation contract. The existing Feed screenshot brief now uses the
  actual Post label; its placeholder and capture intent are unchanged.
- **Native observation on 2026-10-20:** a fresh private browser profile reached
  the retained VM's real sign-in form. Its normal certificate trust check
  failed. For interface inspection only, the browser used the server's leaf
  public-key pin after a separate HTTPS check validated its chain and hostname
  against the public CA retrieved through pinned SSH in HB11. This does not
  establish normal browser trust-store setup. One sign-in attempt using the
  retained private credential did not establish a session. Subsequent
  read-only Workspace, Feed, Project and token-settings visits returned the
  sign-in page; the old sample repository URL returned 404 while unauthenticated,
  which does not prove the repository was deleted. No password reset or
  repeated password attempt was made. The owner confirmed that no current
  documentation fixture is available.
- **Remaining live scope:** provide a matching documentation instance with
  current test identities, a safe sample repository/Project, provider and
  standing grants, required checks and a supported verification fixture.
  Follow the catalog request through preparation, questions, dependencies,
  independent review, delivery and integrated verification; separately prove
  personal Join/checkout/PR/reconnect, permission and changed-input handling,
  and Library retrieval. These authenticated journeys and their screenshots
  were not executed. Source review cannot establish that they work.
- **Existing product owner:** the
  [R43 frontend task](../factory-remaining-tasks/05-experience-and-development.md#r43)
  now explicitly includes the missing human issue-adoption/reacceptance path
  over the existing acceptance preview/mutation, including selected answers
  and non-code prerequisite confirmation. The present Work detail displays
  acceptance status; Project setup acceptance and a comment reply do not
  supply that issue action. Public articles retain the final intended
  semantics; no private API workaround or fictional control was added.
- **Checks:** all 11 changed articles retain their 140–220-character summaries
  and every existing heading/anchor. Hierarchy, 77 local links and all six
  strict visual JSON blocks passed; one screenshot brief changed. All 26
  shell fences and eight Mermaid diagrams are byte-identical to their
  previously checked versions, so those examples/diagrams were not rerun.
  Task-reference and whitespace checks passed. No image assets were created.
  Private browser state and validation output remain outside the repository
  under `/home/vince/Projects/.sodaos-handbook-goal-20261020/HB12/`.
  HB12 remains open; continue with HB13.

<a id="hb13-follow-app-setup-and-same-address-restoration"></a>
### HB13 — Follow app setup and same-address restoration

- [ ] **P1 · Partially completed 20 October 2026; installed journey unverified.**
  Scope: O04/O06/O07, with D05 and U02 handoffs.
  Use a documentation app with recognizable sample records/files to follow
  first-owner setup, stopped Install, explicit Start, access and fresh Restore.
- **Done when:** the actual labels and results support the written procedure:
  owner/member access, selected snapshot identity, same app/address, retained
  old data and explicit restart. Distinguish Replace from Restore and reconcile
  expiry or uncertain-response instructions with observed behavior. Reuse
  qualification observations when available; documentation validation does not
  itself certify the runtime.
- **Source and instructions:** reviewed all five scoped pages and changed four
  (O04/O06/O07/U02); D05's publisher handoff remains owned by HB07. O04 now
  distinguishes Soda lifecycle administration from Canlang accounts, explains
  capability expiry/lost-response inspection, names actual Start/Stop/Inspect
  controls and adds the source-backed stopped production-member procedure.
  It retains private credential-file requirements, password limits, existing
  account/refusal behavior and successful helper exit before Start. U02 adds
  the separate app-origin DNS/certificate/client checks. O06 separates a
  stopped snapshot and optional unbound copy from same-address Restore and
  gives recognizable record/file/receipt and snapshot-time access checks.
  O07 reconciles native permission removal with explicit Project revocation
  and preserves app-linking and retained-data boundaries.
- **Product-owner finding:** Canlang's storage producer writes v2 while Soda's
  snapshot backup admits only v1. Added this source mismatch to the existing
  P25 work in
  [production application setup and recovery](../canlang-soda-os-implementation-plan/production-workerd.md#guided-application-setup-and-recovery).
  That owner also retains the missing trusted snapshot inventory and durable
  same-address data-binding transition under P25/P26. Guided first-owner
  setup remains in P14.4/P26, and P30 owns the installed recovery witness.
  The public articles describe these selected final behaviors; the internal
  record identifies the missing implementation and evidence. No runtime code
  was changed or storage mismatch reproduced against a deployed app here.
- **Native limitation:** the owner confirmed no current documentation fixture
  is available. No prepared production app, private app-origin trust,
  owner/member accounts or selected snapshot was available for this journey.
  Historical recovery/image/ENOSPC/VM receipt paths named by the product owner
  are absent on this host; its written component and unbound-copy results do
  not establish the complete same-address operation. No first-owner claim,
  production member provision, Install/Start, backup or Restore was executed
  in this pass. Expiry, uncertain responses, preserved app/address, retained
  old data and restarted owner/member access still need the real fixture and
  completed implementation. Existing retained data was not altered.
- **Checks:** four changed pages pass summary length, one H1, heading hierarchy,
  all 28 existing anchors and 22 local links. Both strict visual JSON blocks
  pass; one existing setup screenshot brief was clarified and both await real
  capture. Five new or split shell fences pass `bash -n`; the member JSON
  example parses and its flags, paths and identity rules were checked against
  the production recipe and Canlang source. Three Mermaid diagrams are
  byte-identical to their previously checked versions and were not rerendered.
  Whitespace and task-reference checks pass. No assets were created.
  Validation output is outside the repository under
  `/home/vince/Projects/.sodaos-handbook-goal-20261020/HB13/`.
  HB13 remains open; continue with HB14.

<a id="hb14-make-maintenance-and-recovery-results-recognizable"></a>
### HB14 — Make maintenance and recovery results recognizable

- [ ] **P1 · Partially completed 20 October 2026; native journeys remain.**
  Scope: O01–O03 and O05–O09, with I05/U03 context.
  Reconcile the existing instructions with retained Project maintenance,
  a complete cold recovery set and fresh isolated restore, access removal,
  and an exact supported update/fallback pair.
- **Done when:** readers can identify all retained domains, verify recovered
  identities/data, fence the original before cutover and distinguish unfinished
  release operations from exit-75 extension attention. Add small real redacted
  output excerpts only where they help identify the state or next action.
  Preserve app restore versus whole-system restore and appliance versus Docker
  boundaries. Reuse the existing implementation/release owners' evidence;
  make no new backup SLA, cross-version recovery or rollback promise.
- **Source and instructions:** reviewed all ten scoped pages and changed five
  (O03/O05/O06/O08/U03). O03 now distinguishes maintenance preflight from
  completed repair, preserves stopped/running requirements and explains Muse
  partial effects and Canlang's retained-directory retry refusal. O05/O08
  distinguish an unfinished recorded transition from extension attention after
  core completion and a later readiness check against current intent. O06
  adds a recovery inventory with recognizable post-restore checks, timer versus
  completed-backup observations and sequential database-restore failure
  behavior. U03 uses the selected revocation/restoration control names and a
  two-member example that preserves files and unrelated services. Existing
  summaries, headings and anchors are retained. The operator guide's stale
  claim that release preflight imports images was corrected alongside O05.
- **Existing product owners:** recorded the missing supported release-content
  preparation route under
  [R45](../factory-remaining-tasks/05-experience-and-development.md#r45).
  `--check` skips acquisition but requires selected source/target content to
  exist locally. Recorded the missing packaged offline Forgejo invocation
  under [R04](../factory-remaining-tasks/01-boundaries.md#r04): stopping the
  server removes its container, so the public guide now identifies the exact
  binary/configuration/data context and native subcommands without inventing
  a host executable or restarting the web server for recovery. The full
  supported invocation and installed result still need that owner.
- **Read-only installed observations:** used the retained VM's existing
  dedicated pinned SSH configuration after the default configuration and
  default host-key record proved unsuitable; no trust record was changed.
  Reconciliation is inactive with a successful recorded exit, but that alone
  does not identify a qualified transition. The backup timer is disabled and
  inactive with no trigger, and the backup service has no execution timestamp
  or retained `backup complete:` journal message. No service was changed.
  Image preparation declares timer enablement; R45 retains this scoped
  observation without asserting a general release defect. The installed
  maintenance helper rejects bare `--help` with its operation-selector
  diagnostic; this was not a maintenance invocation or a successful journey.
- **Remaining input and evidence:** no exact qualified release-pair receipt,
  isolated complete cold recovery target or current documentation fixture is
  available. Existing `v0.1.2` download verification is not an upgrade pair.
  R45/R52 still need the bound transition/interrupted-recovery/stopped-Project
  receipts and complete cold-set restoration; R44.1 still needs the real
  two-member revoke/reconnect/restore witness. No maintenance, update,
  rollback, backup, data restore or access revocation was executed here.
  Native recovery subcommands and source-derived output phrases were checked
  against their owners, not presented as observed recovery transcripts.
- **Checks:** five changed articles pass summaries, heading hierarchy, all 32
  retained anchors and 40 local links. The new backup-inspection shell fence
  passes `bash -n`; existing shell fences are unchanged apart from replacing
  the misleading standalone Forgejo command with labeled native subcommands.
  Two Mermaid diagrams are byte-identical to their previously checked
  versions and were not rerendered. No visual JSON or assets changed.
  Owner/task references and whitespace pass. Validation and read-only
  observations are retained outside the repository under
  `/home/vince/Projects/.sodaos-handbook-goal-20261020/HB14/`.
  HB14 remains open. The current readiness table selects HB16 while HB15
  awaits the unavailable capture fixture.

## Visuals, integration and handoff

<a id="hb15-complete-the-existing-screenshot-briefs"></a>
### HB15 — Complete the existing screenshot briefs

- [ ] **P2 · Actual screens and safe sample data needed.** Scope: the
  [12 captures below](#screenshots-awaiting-capture).
  Pair captures with HB11–HB14 where possible. Each source block already owns
  its navigation, access/state, sample data, viewport, theme and editing brief.
- **Done when:** real PNG/JPEG/WebP files under `docs/public/assets/` are attached
  with relative `image` fields; capture provenance, labels, sparse annotations,
  caption and alt text match the actual screen and remain readable at narrow
  widths. Preserve the placeholder and source-only production brief. Omit
  `image` until a real capture exists. No generated UI, raster diagrams, added
  visual panels or image quota.

<a id="hb16-integrate-new-material-across-all-page-owners"></a>
### HB16 — Integrate new material across all page owners

- [ ] **P2 · Current-source integration done; later inputs remain.**
  Scope: all 41 allocated pages.
  Reconcile new examples with the pages they link to: consistent actors,
  vocabulary, example values, prerequisites, expected results and consequences.
  Keep already clear pages unchanged.
- **Done when:** each procedure has one owner; daily instructions are easy to
  scan; dense rules sit beneath appropriate headings; examples identify where
  commands run and what success means. Retain precise permissions, defaults,
  exceptions and failure behavior. Keep summaries useful as standalone previews,
  existing filenames/order and meaningful anchors. Do not convert the handbook
  into API internals, a Factory implementation backlog or a second Forgejo or
  Canlang manual. Update affected callers when a heading must change.
- **20 October integration result:** reviewed all **41 pages** and changed
  **28**; the other 13 remain clear in their existing form. Preserved all
  filenames, chapter order, summaries and headings/anchors. Split dense
  prerequisites, ordered connection/sponsorship and Canlang native-file steps,
  and linked the owning procedure at recovery and permission boundaries.
  Reconciled provider-account ownership, Soda operator authority, host
  administration, repository policy administration and Project access holds.
  Restoration of a member's Project access and current repository write
  permission are separate prerequisites; a Project administrator does not
  automatically have host maintenance authority.
- **Procedure integration:** clarified Docker client/outer-host/runtime command
  contexts and the runtime-root entry point, linked Project maintenance to its
  host owner, and separated browser enrollment from native Factory enrollment.
  Factory prose now distinguishes submitted output from trusted checkpoints,
  exact candidate/base evidence from later reassessment, and observer actions
  from Test Pilot recording. Canlang development owns packaging and hands the
  result to app administration; bootstrap/member CLI sessions finish before
  another process opens the same native storage. App lifecycle and cold backup
  guidance link their canonical owners. Updated five stale page-contract rows
  for access restoration, research acceptance, publisher handoff, app setup and
  same-address Restore.
- **New source finding:** the native Factory enrollment manifest example names
  the root-owned dashboard DSN where enrollment requires a private broker-owned
  DSN. Recorded the exact mismatch and recipe/native qualification work under
  [R44](../factory-remaining-tasks/05-experience-and-development.md#r44).
  U01/F03 explain the separate prerequisite without publishing the known-failing
  manifest as a working recipe. R44 supplies the corrected operator procedure;
  HB12 follows it with the real author/contributor journey.
- **Checks and limits:** all 41 pages pass the page inventory, single-H1,
  140–220-character summary, hierarchy, retained-anchor and **350 local-link**
  checks. All **12 visual blocks** parse and retain the required fields; they
  are unchanged, await real captures and contain no image paths. Both JSON
  examples parse. All **22 Mermaid diagrams** and non-shell examples are
  unchanged. The one changed shell fence uses the installed absolute
  `soda-factory` executable path with unchanged arguments and passes shell
  syntax checking. Source confirms its shipped location; no native Factory
  command was executed. Whitespace and affected internal-document links pass.
  Validation output and the before-snapshot are in
  `/home/vince/Projects/.sodaos-handbook-goal-20261020/HB16/`.
  No runtime journey, screenshot capture, website generation or publication
  ran. HB16 remains open only for integration of the named later inputs;
  continue with HB17 without repeating this completed review.

<a id="hb17-validate-the-changed-handbook-source"></a>
### HB17 — Validate the changed handbook source

- [ ] **P1 · Ready now; reuse HB16's passing integration checks.**
  Scope: all source links and
  structural invariants; changed diagrams, examples and image references.
  Use existing Markdown/website reader tooling read-only and appropriate
  language/command checks. Do not add unit tests or a new checker framework.
- **Done when:** inventory still matches all page owners; H1/summary/heading
  rules, local links/anchors, strict visual JSON and real asset paths pass.
  Pending-information blocks satisfy their exact JSON contract, use unique
  IDs and map to the existing tasks above; their presence does not fail source
  validation merely because the information remains outstanding.
  Mermaid parses with accessibility fields and supported syntax; changed
  diagrams and new captures have been inspected. Applicable examples have
  meaningful checks and outcomes, not just shell syntax. Whitespace checks
  pass. State which native journeys remain unverified; do not rerun unchanged
  passing checks without a relevant change.

<a id="hb18-hand-off-the-completed-source-to-the-website-owner"></a>
### HB18 — Hand off the completed source to the website owner

- [ ] **P3 · Handoff preparation ready; final website verification awaits source integration.**
  Preserve the [website cutover allocation](#shared-integration-writers).
  Confirm its route/fragment work still reflects the website's actual state
  before treating historical entries as remaining work.
- **Current component handoff:** implement the
  [`doc-pending` contract](../../public/README.md#rendering-contract-for-the-website)
  and accept the nine current notices above. Recognize the fence before generic
  code rendering, validate its JSON and keep its content out of navigation,
  headings, summaries and adjacent-page previews. Verify text escaping,
  accessible note labeling, narrow layouts and rejection of malformed blocks.
  Adopt the contract in the shared website-owned authoring template. This
  receiving-owner work can proceed while factual inputs remain outstanding.
- **Done when:** the website owner has the exact source revision and required
  route/anchor changes, then validates generated navigation, previews, diagrams,
  screenshots and source-only production-field exclusion using its existing
  process. This repository's work does not modify generated website HTML.
  Commit, synchronization, push and publication follow explicit authorization;
  this task list grants none.

## Complete page allocation

All **41 pages** are allocated below. HB16/HB17 apply to every row after relevant
changes; the table lists each page's other content or walkthrough tasks.
Maintainer README/FILETREE files are excluded from article work. IDs retain the
existing [page contracts](page-contracts.md).

| ID and source page | Tasks |
| --- | --- |
| H01 [Handbook entrance](../../public/10-Start-here/10-index.md) | HB01, HB12 |
| H02 [Product model](../../public/10-Start-here/20-product-model.md) | HB01, HB12 |
| P01 [Spaces](../../public/10-Start-here/30-spaces.md) | HB01, HB12 |
| U05 [First connection](../../public/10-Start-here/40-first-connection.md) | HB05, HB12 |
| U04 [Provider connections](../../public/10-Start-here/50-provider-connections.md) | HB06, HB11, HB12, HB15 |
| F01 [Software factory](../../public/20-Factory/10-software-factory.md) | HB03, HB12 |
| F08 [Factory Feed](../../public/20-Factory/20-factory-feed.md) | HB01, HB03, HB04, HB12, HB15 |
| F02 [Plan new work](../../public/20-Factory/30-plan-new-work.md) | HB02–HB04, HB12, HB15 |
| F04 [Issue work and dependencies](../../public/20-Factory/40-issue-work-and-dependencies.md) | HB02–HB04, HB12 |
| F06 [Review and merge](../../public/20-Factory/50-review-and-merge.md) | HB03, HB04, HB12, HB15 |
| F07 [Application verification](../../public/20-Factory/60-application-verification.md) | HB03, HB12 |
| F09 [Knowledge and decisions](../../public/20-Factory/70-knowledge-and-decisions.md) | HB01, HB03, HB04, HB12, HB15 |
| F10 [Factory intervention](../../public/20-Factory/80-factory-intervention.md) | HB01, HB03, HB12 |
| P02 [Projects and workspaces](../../public/30-Projects-and-development/10-projects-and-workspaces.md) | HB12, HB15 |
| P04 [Workspace terminals](../../public/30-Projects-and-development/20-workspace-terminals.md) | HB12, HB15 |
| D01 [Collaboration](../../public/30-Projects-and-development/30-collaboration.md) | HB05, HB12, HB15 |
| D02 [Connect and develop](../../public/30-Projects-and-development/40-connect-and-develop.md) | HB05, HB06, HB12 |
| D03 [Shared tools and files](../../public/30-Projects-and-development/50-shared-tools-and-files.md) | HB05, HB09, HB12 |
| D04 [Project services](../../public/30-Projects-and-development/60-project-services.md) | HB09, HB12 |
| D05 [Canlang development](../../public/30-Projects-and-development/70-canlang-development.md) | HB07, HB08, HB12, HB13 |
| U03 [People and access](../../public/40-Configure/10-people-and-access.md) | HB05, HB12, HB14 |
| F03 [Factory authorization](../../public/40-Configure/20-factory-authorization.md) | HB02, HB03, HB12 |
| P03 [Project preparation](../../public/40-Configure/30-prepare-project.md) | HB09, HB12, HB15 |
| F05 [CI runners](../../public/40-Configure/40-ci-runners.md) | HB03, HB12 |
| U02 [Private networking and TLS](../../public/40-Configure/50-private-networking-and-tls.md) | HB11, HB13 |
| U06 [Tailscale](../../public/40-Configure/60-tailscale.md) | HB11 |
| O09 [Troubleshooting](../../public/50-Operate/10-troubleshooting.md) | HB01, HB14 |
| O01 [Administration](../../public/50-Operate/20-administration.md) | HB11, HB14 |
| O02 [Cockpit](../../public/50-Operate/30-cockpit.md) | HB11, HB14 |
| O04 [Private applications](../../public/50-Operate/40-private-applications.md) | HB07, HB08, HB10, HB13, HB15 |
| O06 [Backups and restoration](../../public/50-Operate/50-backups-and-restoration.md) | HB13, HB14 |
| O03 [Project maintenance](../../public/50-Operate/60-project-maintenance.md) | HB14 |
| O05 [Updates and fallback](../../public/50-Operate/70-updates-and-fallback.md) | HB14 |
| O08 [Recovery](../../public/50-Operate/80-recovery.md) | HB14 |
| O07 [Data safety and removal](../../public/50-Operate/90-data-safety-and-removal.md) | HB13, HB14 |
| I01 [Deployment options](../../public/60-Install/10-deployment-options.md) | HB10, HB11 |
| I02 [Verify downloads](../../public/60-Install/20-verify-downloads.md) | HB11 |
| I04 [VM and cloud installation](../../public/60-Install/30-deploy-to-cloud.md) | HB10, HB11 |
| I03 [On-premises installation](../../public/60-Install/40-install-on-premises.md) | HB11, HB15 |
| I05 [Docker](../../public/60-Install/50-docker.md) | HB10, HB11, HB14 |
| U01 [Operator setup](../../public/60-Install/60-operator-setup.md) | HB11 |

## Screenshots awaiting capture

HB15 completes these **12 briefs on 11 pages**, including the token form added
by HB05. The exact placeholder
identifies the block; follow the full production instructions in its source.
Assets ready: **zero**. Mermaid diagrams are authored source, not pending images.

| Page | Existing placeholder sentence |
| --- | --- |
| I03 | Final installer review identifies a disposable target disk before explicit erase confirmation. |
| U04 | Connected provider status shown to its account owner without exposed sign-in material. |
| F02 | Review a consolidated product direction and proposed outcomes before choosing a repository or creating work. |
| F06 | Inspect one native pull request with its exact candidate revision, independent review and required check evidence. |
| F08 | Follow a role-authored Feed post, read its thread, and review an unsent answer draft. |
| F09 | Inspect a scoped Project knowledge item with its evidence references and attributed revision history. |
| P02 | Project Overview shows its running identity and the browser-only Join action for a write-authorized account. |
| P03 | Factory preparation pairs accepted source and administrator review with separate recorded coder and reviewer results. |
| P04 | The persistent Workspace panel keeps your Project terminal attached beside a native Forgejo page. |
| D01 | In Applications, select your repository and its Read and write permission before generating a token. |
| O04 | Review the prepared app and private address before opening its separate first-owner setup. |
| O04 | Stopped app record with independent Project link and explicit Start action. |

## Completed source cutover

The [target tree](proposed-tree.md) and [page contracts](page-contracts.md) define
the implemented source handbook. All five writing packets are complete: three
GPT-6 Luna High writers produced the public copy and updated in-repository
consumers. One GPT-6.1 Sol Medium review covered 13 critical pages; takeover,
Docker setup and maintenance-recovery corrections were integrated. The
coordinator also corrected the omitted backend writer in database restore
instructions and their private guidance. Website source synchronization and
retired-route integration remain outstanding; this task did not publish or
modify the website checkout.

## Existing file disposition

This table records the original five-section to seven-section cutover. Its
source and target paths are historical; use the [current tree](proposed-tree.md)
for the subsequent six-section reordering.

All paths in this table are relative to `docs/public/`. “Rewrite” means retain
the useful subject/slug while replacing stale prose from the canonical contracts
and inspected workflows; it does not authorize copying old instructions as final
behavior. Secondary successors receive the named responsibility, not duplicate
whole articles.

| Existing file | Primary target | Action and secondary allocation |
| --- | --- | --- |
| `README.md` | `README.md` | Rewrite authoring/ingestion rules and section intent after all owners exist; never publish the planning records |
| `10-Start-here/10-index.md` | `10-Start-here/10-index.md` | Rewrite task entrances and complete first-build journey; remove old dashboard/CLI-led guidance |
| `10-Start-here/20-product-model.md` | `10-Start-here/20-product-model.md` | Rewrite unified native identity, shared Project and independent app model; link detailed owners |
| `20-Deploy/05-verify-downloads.md` | `20-Install/20-verify-downloads.md` | Move/rewrite; share deployment choice with new I01 |
| `20-Deploy/10-deploy-to-cloud.md` | `20-Install/40-deploy-to-cloud.md` | Move/rewrite VM preparation; reuse common ISO install and retire speculative cloud-image branches |
| `20-Deploy/20-install-on-premises.md` | `20-Install/30-install-on-premises.md` | Move/rewrite as common media/disk procedure for hardware and VMs |
| `20-Deploy/22-docker.md` | `20-Install/50-docker.md` | Move/rewrite released Docker path and actual persistence/privilege boundary |
| `20-Deploy/25-operator-setup.md` | `30-Set-up/10-operator-setup.md` | Split initial setup from U02 networking/TLS, U04 provider connection and F03 ongoing Factory grants |
| `20-Deploy/30-first-connection.md` | `30-Set-up/50-first-connection.md` | Move/rewrite native sign-in, browser-only Join and first terminal; SSH detail goes to D02 |
| `30-Use-Soda/05-dashboard.md` | `50-Spaces-and-Projects/10-spaces.md` | Retire obsolete page/slug; salvage find-work/navigation and genuine failure handling only; no standalone login or manual admission |
| `30-Use-Soda/10-cockpit.md` | `70-Operate/20-cockpit.md` | Move/rewrite stock host administration |
| `30-Use-Soda/15-software-factory.md` | `40-Factory/05-software-factory.md` | Rewrite complete lifecycle overview; split detailed responsibilities across F02–F10; operator commands belong in F10 |
| `30-Use-Soda/20-projects-and-workspaces.md` | `50-Spaces-and-Projects/20-projects-and-workspaces.md` | Split lifecycle/Join from new P03 preparation and P04 sessions; replace disposable-agent assumptions |
| `30-Use-Soda/30-forgejo.md` | `60-Develop/10-collaboration.md` | Merge native repository/Git tasks into one collaboration owner; retire redundant `forgejo` slug |
| `30-Use-Soda/35-collaboration.md` | `60-Develop/10-collaboration.md` | Merge/rewrite native collaboration; Factory acceptance/review belongs in F04/F06 and scoped Feed discussion in F08 |
| `30-Use-Soda/40-tailscale.md` | `30-Set-up/60-tailscale.md` | Move/rewrite optional Fountain-owned containerized Tailnet; shared DNS/TLS is U02 |
| `30-Use-Soda/50-ci-runners.md` | `40-Factory/40-ci-runners.md` | Move/rewrite separately managed native Actions capacity; required policy in F03, candidate acceptance in F06 |
| `30-Use-Soda/60-updates-and-fallback.md` | `70-Operate/50-updates-and-fallback.md` | Move/rewrite native qualified transitions; retained Project refresh in O03 and interrupted host recovery in O08 |
| `40-Develop/10-connect-and-develop.md` | `60-Develop/20-connect-and-develop.md` | Move/rewrite explicit installed public keys, current native access and personal Git/editor work |
| `40-Develop/20-shared-tools-and-files.md` | `60-Develop/30-shared-tools-and-files.md` | Move/rewrite actual shared installations/state for humans and assigned Factory roles |
| `40-Develop/30-project-services.md` | `60-Develop/40-project-services.md` | Move/rewrite shared development service lifetime; independent apps in O04 |
| `50-Operate/10-people-and-access.md` | `30-Set-up/30-people-and-access.md` | Move/rewrite authority/offboarding; connection mechanics U04, Factory sponsorship F03, installed SSH keys D02 |
| `50-Operate/20-administration.md` | `70-Operate/10-administration.md` | Split health/host operations from F10 run control, O03 Project maintenance, O08 recovery and O09 diagnosis |
| `50-Operate/30-backups-and-restoration.md` | `70-Operate/60-backups-and-restoration.md` | Move/rewrite complete consistency set, separate broker custody and stopped private-app fresh restore |
| `50-Operate/40-data-safety-and-removal.md` | `70-Operate/70-data-safety-and-removal.md` | Move/rewrite real persistence/deletion effects; remove unsupported lifecycle-program promises |

After cutover remove the old `20-Deploy/`, `30-Use-Soda/`, `40-Develop/` and
`50-Operate/` directories. There is no archived public handbook or published
stub copy. Git retains the predecessor content.

## Packet allocation and dependencies

Every exact target writer list is given by the page IDs in
[page contracts](page-contracts.md); each ID maps to one literal filename. The
current input writer lists are the corresponding rows in the disposition table.
Shared integration files below have one coordinator writer.

| Packet | Exclusive target pages | Source correction and writing duties | Required outputs before dependent work |
| --- | --- | --- | --- |
| W1 Install and access | I01–I05, U01–U06 | Distribution/media/Docker; native bootstrap; shared network, identity and provider procedures. Retire OAuth and host-Tailscale assumptions. | Publish stable U02/U04 topic boundaries early for all other writers. DEC07 gates invented support numbers only. |
| W2 Factory | F01–F10 | Entire intended work lifecycle, research/output contracts, accounts/grants, CI, exact review/merge, app verification, Feed/Library and interventions. Move narrow operator commands to support. | Consume U04 connection boundary and P03 readiness boundary. Resolve research contract DEC04 before exact metadata/routing/non-code acceptance steps; other pages can proceed. |
| W3 Projects and development | P01–P04, D01–D05 | Native Workspace/session model; persistent Project lifecycle/preparation; merged Forgejo collaboration; explicit SSH, shared tools/services and installed Canlang development. | Publish P03 readiness boundary early. Coordinate U03 key/access ownership and O03/O04 lifecycle boundaries. |
| W4 Operations and private apps | O01–O09 | Host health/Cockpit; retained-root support; independent app administration; qualified transitions, consistent backups, offline recovery and diagnostic routing. | Consume U02 network contract, D05 development boundary and F10 run-settlement boundary. Can proceed alongside W1–W3 as writer capacity becomes available. |
| W5 Integration and navigation | H01/H02, source README, in-repo/website consumers | Whole journey entrances, route/fragment and cross-link cutover, source ownership/navigation, removal of old paths and final consistency scan. | Start link mapping immediately; finalize only after W1–W4 pages and their actual headings exist. |

Separate these operations within each packet:

- **Move:** apply selected paths and remove predecessor owners, using the table;
  moving text alone does not complete the rewrite.
- **Correct:** rewrite the stale product assumptions and complete missing reader
  tasks from the exact page contract and canonical sources. Do not create mock
  endpoints, internal schema tutorials or WIP disclaimers to fill a page.
- **Verify:** validate real page structure/links/routes and intended-product
  consistency. Real captures or command examples must come from the selected
  interface and owning contract; fixture output is not an installed journey.

Three Luna High writers handled W1, W2 and W3 concurrently. Released capacity
then handled W4 and W5, with explicit file handoffs to the coordinator for final
corrections. One Sol Medium reviewer inspected the 13 selected critical pages;
the coordinator handled predecessor deletion and final source-reader validation.
No additional review agents or application builds were needed.

## Shared integration writers

The coordinator owns these files during the eventual cutover:

- `docs/public/README.md`, H01 and H02.
- `docs/README.md`, `docs/development/README.md` and the links in
  `docs/product/scope.md` that currently point into the old public paths.
- Any additional in-repository consumers found by searching `docs/public`, old
  section names and the retired filenames before deletion.
- The five living plan files, until their completed decisions are absorbed.

The website is a separate checkout. Its required integration is recorded here;
no website change or synchronization has been performed in this phase. During
that cutover give one writer the following files and discovered co-consumers:

- `scripts/docs-sync/profiles.mjs`: old source-keyed `prepare-the-factory`
  compatibility anchor. Replace reliance on it with a meaningful explicit
  `Prepare the factory` heading in the new F01 overview and update source-keyed
  handling with the move. Do not silently lose incoming fragments.
- `scripts/docs-sync.test.mjs`: adapt the existing integration case for the new
  source path/fragment contract; do not add unit tests.
- `src/app/content/platform.ts` and
  `src/app/components/organisms/download/download-follow.tsx`: replace
  `/docs/forgejo` and its `create-or-import-a-repository` fragment with the
  D01 collaboration successor. Preserve the operator-setup trust handoff anchor.
- `src/app/content/handbook-social-images.ts` and
  `src/app/docs/content.test.ts`: update obsolete dashboard/forgejo metadata and
  title assumptions; do not keep the old dashboard narrative to satisfy a check.
- `src/app/pages/docs/routes.tsx`, composed by `src/worker.tsx`: deliberately
  redirect retired `/docs/dashboard` to `/docs/spaces`
  and `/docs/forgejo` to `/docs/collaboration`, including the repository fragment,
  when route compatibility is implemented. Keep redirects out of handbook prose.
- `src/app/docs/generated/`: regenerate through the existing sync tool as one
  set after the public source is committed; never hand-edit generated pages,
  manifest hashes or TypeScript content imports.

Retained slugs keep their routes even when their source folder changes.
Reconcile old heading fragments separately; a matching slug does not prove its
anchors or authored website link text still make sense. Preserve useful headings
where their meaning remains, and update known authored consumers otherwise.

## Verification

The completed source cutover passed the website's actual
`snapshotFromDirectory` reader: seven sections, 41 pages and 41 HTML documents
rendered in memory, with all public Markdown links and fragments valid. The
meaningful `prepare-the-factory`, operator-setup trust and repository-creation
fragments are retained. No website snapshot or published file was written.

The actual file set matches all 42 planned leaves, including the unpublished
README; all 41 pages have the required single H1 and immediate description.
The 22 predecessor paths and their empty section directories are removed.
Tracked changes passed `git diff --check`, and the new files passed separate
whitespace checks. Private source links and an H1-only fragment found by the
actual reader were corrected before the passing run. Historical predecessor
attribution is retained explicitly as history, without a broken live link.

Performed in this filetree phase:

- Read-only source/contract/website discovery investigation and specialist
  boundary challenges documented in the source audit.
- Inventory/allocation reconciliation passed: 25 existing files, 41 unique page
  owners, 22 retained routes, two retired routes and 19 new pages across seven
  sections; every target belongs to exactly one writing packet.
- Plan artifact checker passed: five documents, 45 local links, 42 unique target
  leaves and zero errors. Website `validateSourceModel` and filename/order checks
  passed against the proposed tree; this validates names/model, not unwritten
  page content or a rendered website.
- `git diff --check` passed for the tracked development-index edit; a separate
  whitespace check passed for all five new documents.

For the content cutover, use the smallest useful checks in this order:

1. Verify all 41 page owners exist, old paths are gone, slugs/orders are unique,
   and no page lacks its descriptive opening or promised success/failure path.
2. Use the existing website directory-snapshot reader to validate the draft
   public tree, Markdown links/fragments and allowed assets without changing
   generated website files or requiring a premature commit.
3. Scan for substantive stale models: separate Soda login/OAuth, standalone
   dashboard, manual issue admission/human merge as mandatory, disposable stage
   containers, local Soda CI placeholders, graphical profiles, provider fallback,
   automatic deployment, destructive recovery or backup guarantees. Review
   context; legitimate service identifiers and explicit scope boundaries remain.
4. Reconcile in-repository and authored website routes/anchors. Once source is
   committed through the owner's normal workflow, use the existing website sync
   and source comparison; committing/publishing is not implied by this plan.
5. Inspect rendered navigation, chapter links and real captures. Run affected
   existing integration checks only where the actual website change warrants
   them; do not add unit tests or run a product build for a documentation move.

These checks establish a coherent handbook source and integration. They do not
qualify Factory execution, provider authentication, an installed appliance,
hardware/cloud support, update transitions or deployment. Those retain their
existing product/release owners and evidence requirements.
