# People and access

Manage human access, execution authority, publishing actors and provider authentication as distinct responsibilities.

Human onboarding and project access do not automatically authorize agent work.
The [factory walkthrough](../30-Use-Soda/15-software-factory.md) explains explicit
admission; the factory identity section below explains credential boundaries.

## Add a person

As the configured Soda operator, use the dashboard's **People** flow:

1. Create the person's Forgejo-backed account with their intended stable login
   and required profile information.
2. Deliver initial credentials through a separate private channel. Never put a
   password in project metadata, issues, screenshots or shared logs.
3. Give them the approved Soda URL, network access instructions and trusted
   browser/SSH identity information.
4. Have them sign in through Forgejo and complete the required first-password
   change and any account-security setup.
5. Have them register their own public development-access key and explicitly
   join the relevant environments.

The password belongs to Forgejo; Soda is not a second password authority. Do not
create a developer in Cockpit or add a human host account to make onboarding work.
Forgejo site administrators use their upstream-authorized administration views;
that status is separate from the configured Soda operator and host root.

## Grant the right project and repository access

The repository's human owner administers its environment. Every person selects
**Add me to this project** to create their own project-local account; ownership
is not an implicit join. Members receive shared-resource access, not engine
administration or host sudo.

Repository collaborators and organizations/teams remain Forgejo-owned. Grant
Git permissions there separately; an environment join is not Git authorization.
A Git permission change likewise does not edit a project's Linux account or
terminate an established SSH session.

## Change credentials deliberately

| Change | Owner and effect |
| --- | --- |
| Password, MFA, recovery and browser applications | Forgejo account-security controls |
| Soda development-access public keys | Profile registry used at join time |
| Keys already installed in an existing project | Project-local authorized keys, administered separately |
| Git SSH/GPG keys and tokens | Forgejo or the external Git host |
| Personal Tea/GitHub CLI/assistant sessions | The native client and provider |
| Host root credentials | Native operator administration |
| Tailnet access and routes | Tailscale administration and client routing |

Profile-key changes do not propagate to existing memberships. The project
administrator/operator must review existing account keys separately and test
replacement access before removing an old key. Project SSH uses root-owned
`/etc/ssh/authorized_keys/LOGIN`; changing a user's ordinary home key file is not
a substitute for that configured registry. Do not share private keys to repair
access.

Keep provider logins and ownership associations stable. A rename, transfer or
reused username is not an automatic Linux UID/home migration. Coordinate with
the operator before changing an identity associated with environments.

## Offboarding and revocation

Inventory all relevant access before acting: Forgejo sessions/applications,
Git keys/tokens, project authorized keys and active sessions, external providers,
CLI credentials and Tailnet policy. Revoking a credential may prevent new
connections without terminating already authenticated sessions.

Use each native owner's revocation controls and confirm the intended result.
Preserve the person's work and hand off shared services first. Soda does not
provide coordinated one-click offboarding, account remapping or project deletion.
Disabling a Forgejo login is not proof that project SSH access ended.

See [Data safety and removal](40-data-safety-and-removal.md) before any native
destructive action. Never delete an environment or database as an access-control
shortcut.

## Factory identities and provider access

Keep four identities distinct: the human authorizing an objective, the execution
principal for one run, the Forgejo actor publishing a candidate or review, and
the provider account supplying model access. Separate implementation and review
actors do not need a new Forgejo account for every process.

The agent receives only its run's permitted operations. Forgejo write tokens stay
with the publisher and review boundary. Provider account authentication may be
injected into a trusted runtime under the selected profile's supported terms;
code in that runtime may extract it. Containers and network allowlists do not
make this a universal secret-confidentiality guarantee.

For Codex account authentication, use a dedicated enrollment. Soda serializes its
use, delivers authentication into bounded tmpfs and returns CLI-maintained state
before another execution. An interrupted return requires cleanup and explicit
reauthentication. Do not restore stale tokens or share the credential stream with
an independent active session. Enrollment changes must wait until the active run
ends. Never put provider state in images, task input, retained output or normal logs.

Renewal, revocation and subscription exhaustion require their actual provider
controls. Workspace destruction removes Soda's copy but cannot invalidate a token
copied elsewhere. A provider login does not grant repository or host access. See
[factory setup](../30-Use-Soda/15-software-factory.md#prepare-the-factory).
