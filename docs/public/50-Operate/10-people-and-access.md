# People and access

Onboard people through Forgejo-backed identity, then manage browser, project, Git and network access as distinct responsibilities.

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
