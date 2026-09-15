# Soda OS documentation

Navigate by intent. Each important subject has one canonical owner; other documents
link to it rather than redefining the contract.

## Understand Soda

| Document | Owns |
| --- | --- |
| [Product overview](product/overview.md) | What Soda OS is and the major concepts |
| [Projects](product/projects.md) | Project environments, profiles, joining, persistence |
| [Spaces](product/spaces.md) | Sodaspaces workspace inside Forgejo |
| [Scope](product/scope.md) | Deferred and excluded product work |
| [Architecture overview](architecture/overview.md) | Topology, components, data flow |
| [Trust model](architecture/trust.md) | Authority, identity, privilege boundaries |
| [Networking](architecture/networking.md) | Access, Tailnet, Cockpit boundary |
| [Release architecture](architecture/release.md) | Install/update candidate model |

## Operate Soda

| Document | Owns |
| --- | --- |
| [Installation](guides/installation.md) | Build, provision, install, activate |
| [Installation media](guides/media.md) | CoreOS ISO and first-boot media |
| [Operator setup](guides/operator-setup.md) | Forgejo bootstrap and Soda setup |
| [Develop in a project](guides/develop.md) | SSH, Git, tools, everyday project use |
| [Project services](guides/project-services.md) | Nested workloads inside a project |
| [Project CLIs](guides/project-clis.md) | Packaged forge CLIs in projects |
| [Local testing](guides/local-testing.md) | Local fixture host access |
| [User handbook](public/10-Start-here/10-index.md) | Release-day operator/developer handbook |

## Reference

| Document | Owns |
| --- | --- |
| [HTTP API](reference/api.md) | Callable Soda routes and semantics |
| [Credentials](reference/credentials.md) | OAuth, grants, schema, maintenance |
| [Project OS](reference/project-os.md) | Project runtime baseline contracts |
| [Runners](reference/runners.md) | Local CI runner capacity contracts |
| [Terminal](reference/terminal.md) | Managed terminal and WebSocket contracts |
| [Forgejo customization](reference/forgejo.md) | Supported Forgejo presentation integration |
| [Configuration](reference/configuration.md) | Dashboard config fields and paths |

## Develop Soda

| Document | Owns |
| --- | --- |
| [Development index](development/README.md) | How to work on this repository |
| [Go ownership](development/go.md) | Package placement and house style |
| [Go packages](development/go-packages.md) | Package file/role convention |
| [TypeScript](development/typescript.md) | Bun workspace and frontend TS |
| [Lit](development/lit.md) | Spaces Lit components |
| [Python](development/python.md) | Python format/lint/complexity |
| [Testing](development/testing.md) | Native and source validation |
| [Native support tools](development/native-support.md) | Build/support tool effects |
| [Release workflow](development/release.md) | Running the release pipeline |
| [Cockpit](development/cockpit.md) | Stock Cockpit branding port |

## Design and research

| Document | Owns |
| --- | --- |
| [Branding](design/branding.md) | Logo, palette, asset rules |
| [Spaces UX](design/spaces-ux.md) | Spaces page and drawer design |
| [Spaces sheets](design/spaces/README.md) | Annotated visual design sheets |
| [Avatars](design/avatars.md) | Robot avatar rendering |
| [Host strategy](research/host-strategy.md) | CoreOS host capability strategy |
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
