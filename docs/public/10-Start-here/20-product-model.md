# Product model

SodaOS is a private software-factory appliance for a trusted team. Forgejo owns native repository collaboration, Soda coordinates bounded Factory work and persistent Projects support both human development and authorized Factory roles.

## The main parts

| Concept | What it means |
| --- | --- |
| Host | The Fedora CoreOS appliance. Its operator administers host services, storage and networking. |
| Native repository | A Forgejo repository and its native issues, pull requests, reviews, checks and merge results. These records remain authoritative for collaboration and code delivery. |
| Build request | A person's saved direction for desired work, alternatives and decisions. It can exist before a repository is selected or any issue is created. |
| Accepted issue | Native repository work whose exact objective, required outcomes and prerequisites have been accepted by a currently authorized maintainer. It can enter Factory readiness assessment under standing policy. |
| Project | One persistent repository environment shared by joined humans and authorized Factory roles. It retains accounts, tools, files and services across ordinary stop/start. |
| Checkout or workspace | A person's or Factory assignment's working copy inside the Project. Sharing an environment does not mean sharing a writable checkout. |
| Attempt | A bounded effort to satisfy an accepted issue under recorded inputs, limits and outcomes. |
| Run | One supervised agent execution within an attempt, with its own identity, permitted work and recorded result. |
| Session or view | A human terminal or Factory CLI execution is a session. Spaces is an authorized view of live sessions and work; it does not own the session's lifetime. |
| Application | A separately administered workload with its own identity and lifecycle. Developing or merging its source does not install or deploy it. |

## How work is governed

Factory automation requires standing authorization from separate owners. A native repository owner controls repository policy, target branch, native actors and required checks. The configured Soda operator controls appliance capacity. The owner of a supported provider account controls sponsorship and its allowance. Current native permissions still apply. One person may hold several responsibilities, but one grant does not imply another.

Once an authorized maintainer accepts an issue's exact inputs, ready work proceeds automatically within those standing grants and limits. A material issue change, missing prerequisite, unavailable resource or changed permission can suspend affected work and trigger reassessment. An agent can report a question or dependency; it cannot supply human authority, waive a blocker or grant itself additional access. See [Factory authorization](../40-Factory/20-factory-authorization.md) for grant ownership and [Issue work and dependencies](../40-Factory/30-issue-work-and-dependencies.md) for readiness and task outcomes.

Forgejo owns issue and pull-request records, reviews, native CI and merge. Soda records accepted inputs, coordinates separate production role identities and reports observed effects. The roles include Product Lead, Architect, Builder, Inspector, Fixer, Strategist and Test Pilot, plus a scoped Library Admin for reusable knowledge and decision records. The Feed presents their reports and confirmed outcomes without replacing the native source records.

## Shared environment and trust boundary

Humans and Factory roles use the same persistent Project with separate accounts and checkouts. The Project is a development environment, not a disposable container created for each agent. Ending or cleaning up a run does not destroy or recreate the shared Project or remove unrelated work.

Projects have host-enforced boundaries but share the host Linux kernel. Soda is designed for a trusted private team, not hostile public multitenancy. Project membership, native repository access and host administration are separate permissions. Docker deployment has its own operator trust boundary; see [Docker](../20-Install/50-docker.md).

## CI and applications

Required verification comes from separately managed native Forgejo Actions capacity. Coding and review agents are not CI runners; a Builder checkpoint does not replace a required native check. See [CI runners](../40-Factory/40-ci-runners.md).

Factory can verify an integrated application version against approved outcomes, but verification does not install or deploy the application. An operator explicitly installs and administers a private application through its separate lifecycle. See [Application verification](../40-Factory/60-application-verification.md) and [Private applications](../70-Operate/40-private-applications.md).

For detailed human and provider identity, onboarding and credential custody, use [People and access](../30-Set-up/30-people-and-access.md) and [Provider connections](../30-Set-up/40-provider-connections.md).
