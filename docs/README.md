# Soda OS documentation

**Mandatory:** [Software freshness](development/freshness.md) takes priority over reproducibility. AI provider CLIs must stay current, all managed inputs need frequent updates, and every new baseline must resolve fresh upstream inputs.

Navigate by intent. Each important subject has one canonical owner; other documents
link to it rather than redefining the contract.

For product evaluation, use the [AI software factory OS overview](https://sodaos.dev/ai-software-factory-os),
[technical reference](https://sodaos.dev/technical-reference),
[comparisons](https://sodaos.dev/compare) and
[operating guides](https://sodaos.dev/use-cases). The website repository owns
those editorial pages; `docs/public` below owns the synchronized handbook.

Soda OS coordinates an automatic software factory within authorized boundaries.
Start with the [target product contract](product/overview.md). The
[operator interface](reference/factory.md) documents the existing command surface;
the [Factory handbook](public/20-Factory/10-software-factory.md) describes the
intended product journey. Spaces connects repository environments with human and
factory work.

## Understand Soda

| Document | Owns |
| --- | --- |
| [Product overview](product/overview.md) | Target factory lifecycle, operating rules and major concepts |
| [Projects](product/projects.md) | Shared repository environments, containers, accounts, checkouts, profiles, joining and persistence |
| [Spaces](product/spaces.md) | Human and factory sessions and views beside native Forgejo collaboration |
| [Scope](product/scope.md) | Retain/adapt/retire/defer decisions and excluded product work |
| [Architecture overview](architecture/overview.md) | Topology, components, data flow |
| [Factory interfaces](architecture/factory-interfaces.md) | Target component APIs, records, transitions and process bindings |
| [Trust model](architecture/trust.md) | Authority, identity, privilege boundaries |
| [Networking](architecture/networking.md) | Access, Tailnet, Cockpit boundary |
| [Release architecture](architecture/release.md) | Supported platform scope and install/update candidate model |

## Operate Soda

| Document | Owns |
| --- | --- |
| [Installation](guides/installation.md) | Build, provision, install, activate |
| [Docker deployment](guides/docker.md) | Released-image installation, runtime setup, persistent volumes and host requirements |
| [Installation media](guides/media.md) | CoreOS ISO and first-boot media |
| [Operator setup](guides/operator-setup.md) | Forgejo bootstrap and Soda setup |
| [Develop in a project](guides/develop.md) | SSH, Git, tools, everyday project use |
| [Project services](guides/project-services.md) | Nested workloads inside a project |
| [Project CLIs](guides/project-clis.md) | Packaged forge CLIs in projects |
| [Canlang applications](guides/canlang-applications.md) | Installed development tools, private app lifecycle/access and stopped recovery |
| [Founder demo](guides/founder-demo.md) | Presenter story, exact actions, real versus recorded results, fixtures and rehearsal for the intended departmental-app journey |
| [Local testing](guides/local-testing.md) | Local fixture host access |
| [User handbook](public/10-Start-here/10-index.md) | Release-day operator/developer handbook |

## Reference

| Document | Owns |
| --- | --- |
| [HTTP API](reference/api.md) | Callable Soda routes and semantics |
| [Credentials](reference/credentials.md) | Provider connections, grants and credential custody |
| [Project OS](reference/project-os.md) | Project runtime baseline contracts |
| [Factory](reference/factory.md) | Existing bounded software-work operator interface |
| [Terminal](reference/terminal.md) | Managed terminal and WebSocket contracts |
| [Forgejo customization](reference/forgejo.md) | Supported Forgejo presentation integration |
| [Configuration](reference/configuration.md) | Dashboard config fields and paths |

## Develop Soda

| Document | Owns |
| --- | --- |
| [Development index](development/README.md) | How to work on this repository |
| [Source ownership](development/ownership.md) | Language, package and process boundaries |
| [Remaining factory work](development/factory-remaining-tasks.md) | The current founder-demo checklist |
| [Founder-demo implementation checklist](development/factory-remaining-tasks.md#founder-demo-checklist) | 39 connected tasks for the journey, conversational decisions/JEV, app use at its own address, policy update and demo qualification |
| [Handbook todo list](development/public-handbook/cutover.md#what-can-proceed-now) | Ready work, partially available work and exact dependencies for all 41 public pages, examples, captures, validation and handoff |
| [Workflow contracts](development/factory-workflow-contracts.md) | D01–D18 schemas, algorithms, outcome/content routes, authority/recovery limits and decisive tests |
| [Canlang runtime contract](development/canlang-soda-os-implementation-plan/production-workerd.md) | Proposed provider, storage, process and packaging contracts with technical substeps |
| [Go ownership](development/go.md) | Package placement and house style |
| [Go packages](development/go-packages.md) | Package file/role convention |
| [TypeScript](development/typescript.md) | Bun workspace and frontend TS |
| [Lit](development/lit.md) | Spaces Lit components |
| [Testing](development/testing.md) | Native/source validation and finished-factory acceptance criteria |
| [Banning unit tests](development/no-unit-tests.md) | Removing unit tests and enforcing the ban per language |
| [Why unit tests are forbidden](development/why-no-unit-tests.md) | The six reasons behind the unit-test ban |
| [Native support tools](development/native-support.md) | Build/support tool effects |
| [Front-end VM update](development/frontend-vm-update.md) | Shipping front-end changes to the retained local development VM |
| [Release workflow](development/release.md) | Running the release pipeline |
| [Cockpit](development/cockpit.md) | Stock Cockpit branding port |

## Design and research

| Document | Owns |
| --- | --- |
| [Branding](design/branding.md) | Logo, palette, asset rules |
| [Spaces UX](design/spaces-ux.md) | Spaces page and persistent panel design |
| [Factory demo journey](design/factory-demo-journey.md) | Founder persona, product-wide journey, source-backed UX gaps and target screen/flow behavior |
| [Factory worker MCP redesign](design/factory-worker-mcp.md) | Shared/role-specific Factory tools, generic System One judgments, Library Admin, issue work contracts and proposed research routing |
| [Spaces sheets](design/spaces/README.md) | Annotated visual design sheets |
| [Avatars](design/avatars.md) | Robot avatar rendering |
| [Host strategy](research/host-strategy.md) | CoreOS host capability strategy |
| [Factory capability map](research/factory-capability-map.md) | Source assessment of target requirements, existing capabilities, gaps and Soda/Fountain/upstream ownership |
| [Library boundaries](development/library-boundaries.md) | Existing library callers and retained technical constraints |
| [Licensing](research/licensing.md) | License and attribution notes |
| [Predecessor reuse](research/predecessor-reuse.md) | Selected reuse attribution |

## Authority rules

1. **One subject, one owner.** Change the contract in the canonical document; link elsewhere.
2. **Product docs describe the intended product**, not this week's implementation progress.
3. **Reference docs describe interfaces that exist** in the current tree.
4. **Completed plans are absorbed and deleted.** Do not leave `*-plan.md` chronology as architecture.
5. **Research is non-normative.** Strategy and research inform decisions; architecture owns the result.
6. **New documents need a distinct ownership reason.** Prefer expanding an owner over adding a peer file.
7. **Do not invent status trackers.** Prefer issues, milestones and Git history for transient work.

`docs/public/` is the website handbook source path consumed by the external
`soda-os-website` docs sync. Keep that path stable unless the website contract changes.
