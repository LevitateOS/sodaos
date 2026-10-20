# Cutover and writing tasks

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
