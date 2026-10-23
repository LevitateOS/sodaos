# Manage people and access

Grant and review access separately across Forgejo, Projects, operator controls, provider accounts and private apps. Removing one permission does not revoke every credential or erase retained work.

SodaOS uses several distinct identities. Give each person only the native Forgejo, Project and operator authority needed for their work, and review each separately when responsibilities change.

## Assign distinct roles

Forgejo owns browser sign-in, repositories, teams, issues, pull requests and native Git permissions. A user's repository permission controls native collaboration. Explicit Project Join separately provisions a Linux account and access to the shared repository environment when native code-write authority permits it.

Joining a Project requires current code-write authority on its repository. The [Project guide](../30-Projects-and-development/10-projects-and-workspaces.md) owns creation, Join and Start.

The Soda operator is the Forgejo user ID recorded during appliance setup. A Forgejo site administrator who was not recorded as operator does not automatically gain Soda operator controls. The host root account administers the machine itself. These authorities should be granted only to the people responsible for those tasks.

Provider connections belong to the person who enrolled them and are held separately by the broker. Enrollment does not delegate use to other members; the owner explicitly authorizes any named-member delegation and its credential exposure. See [Provider connections](../10-Start-here/50-provider-connections.md) for delegation, disconnect and revocation. Private applications keep their own accounts and authorization, independent of Forgejo and Project membership.

## Choose authority for the action

A Project lifecycle administrator is the configured Soda operator or current native repository owner, including organization ownership where applicable. This differs from a native Project root/wheel administrator who installs shared packages and services.

| Action | Eligible actor | Prerequisite | Procedure |
| --- | --- | --- | --- |
| Create a Project | Human repository owner | Repository has no Project; organization-owned creation is unsupported | [Create a Project](../30-Projects-and-development/10-projects-and-workspaces.md#create-or-reuse-a-project) |
| Join and open a personal terminal | Native collaborator | Current repository code-write permission and explicit Join | [First connection](../10-Start-here/40-first-connection.md) |
| Start or Stop a Project | Project lifecycle administrator | Current owner/operator authority; coordinate shared interruption | [Project lifecycle](../30-Projects-and-development/10-projects-and-workspaces.md#start-and-stop-the-retained-root) |
| Revoke or restore a member's Project access | Current repository owner or configured Soda operator | Explicit member-specific action; returning through Join also needs current repository code-write permission | [Offboarding](#review-access-when-roles-change) |
| Accept requirements or intervene in Factory work | Current code-write maintainer | Exact accepted inputs and applicable standing grants | [Factory authorization](20-factory-authorization.md) |
| Change repository Factory policy | Native repository owner or administrator | Other grants and native protections still apply | [Factory authorization](20-factory-authorization.md) |
| Grant appliance capacity | Configured Soda operator | Explicit limits for that repository | [Factory authorization](20-factory-authorization.md) |
| Sponsor or delegate a provider connection | Connection owner | Authorized connection, scope and allowance; delegation consent where required | [Provider connections](../10-Start-here/50-provider-connections.md) |
| Accept setup requirements | Current code-write maintainer | Reviewed exact source and complete inputs | [Project preparation](30-prepare-project.md) |
| Install shared prerequisites | Native Project root/wheel administrator | Reviewed privileged effects and coordinated maintenance | [Shared tools](../30-Projects-and-development/50-shared-tools-and-files.md) |
| Install or replace a private app | Configured Soda operator | Prepared image, storage and explicit app authority | [Private applications](../50-Operate/40-private-applications.md) |
| Inspect, Start or Stop an installed app | Recorded app administrator or operator | Authority for that application | [Private applications](../50-Operate/40-private-applications.md) |

One person can hold several roles. Each action still checks its own authority; a generic administrator title does not grant every action.

## Review access when roles change

Use this sequence when a member leaves or loses responsibility for a Project:

1. As the current repository owner or configured Soda operator, open **Project settings → Access → Project members** and choose **Review access** for that member. Review the stable account identity, installed keys and affected sessions and processes before choosing and confirming **Revoke Project access**. Services running as that member are affected too; independently owned services remain separate.
2. Explicitly revoke that member's Project access. Revocation places a durable re-entry hold, removes their installed Project SSH keys, disables native login and ends their native sessions and processes. It retains their Linux identity, home, checkouts and history.
3. Inspect the operation's native results. Confirm installed keys are removed, new SSH login is refused and the member's managed terminals and native processes have ended. Check that unrelated members and independently owned services remain available. A saved public key in the person's account is separate from the removed installed copy.
4. Remove native Forgejo team/repository rights as appropriate. Review personal Git tokens separately: revoking Project access does not delete them, and they remain usable wherever their native permissions allow. Review operator assignment, provider ownership/delegation/sponsorship and each app-local account independently through their owning controls. Rotate any shared credentials the person could access.
5. Retain or transfer files according to the team's data policy. Revoking access does not erase work or undo information already copied.

Removing native Forgejo permission closes managed access that depends on it, but is not the complete Project revocation operation. Browser sign-out likewise does not revoke installed keys or stop independent SSH sessions.

If a revocation result is incomplete or uncertain, keep the re-entry hold in place and inspect the same member and operation. Retry unresolved effects after readback; do not delete the account, replace the Project or report completed offboarding from a browser response alone. A stopped or unreachable Project retains the hold and applies pending revocation before accepting that login on restart.

For example, Alice and Bob share a Project, and Alice runs a preview under her
own Linux account. Revoking Alice ends her terminal, SSH session and preview,
while Bob's session and independently owned services remain available. Alice's
home and dirty checkout remain, but a still-valid repository write grant does
not let her Join again while the hold is active. These separate results make
the difference between blocked re-entry and completed native revocation visible.

### Restore a member deliberately

Review the reason for revocation before choosing **Restore Project access** in the member view as owner/operator. Restoration clears the access hold; it does not grant Forgejo permission or revive ended processes. The member must Join again with current native code-write permission.

Browser terminal use needs no SSH key; reinstall any desired SSH keys through explicit key selection and application. Confirm the member returns to the retained Linux identity and files, then open a new terminal or SSH connection. In the example above, Alice's retained checkout returns with her account; her ended preview requires a new explicit start.

## Recover a native account

Forgejo owns browser passwords, two-factor authentication and recovery codes. Use its account security settings to set up factors and retain recovery codes privately; use the sign-in page's recovery flow when needed. Use the maintained [Codeberg guide to Forgejo two-factor authentication and recovery tokens](https://docs.codeberg.org/security/2fa/) for that native flow, using your own instance rather than Codeberg’s address. The [Forgejo user guide](https://forgejo.org/docs/latest/user/) covers other native account features.

Recovering a Forgejo login differs from reconnecting a provider subscription or rotating a Project SSH key. The configured Soda operator remains the stable Forgejo user ID recorded at setup; an account rename or a different site administrator does not silently replace it.

## Continue onboarding

For a teammate's first Project session, follow [First connection](../10-Start-here/40-first-connection.md). Before their first clone or push, use [personal HTTPS token setup](../30-Projects-and-development/30-collaboration.md#create-a-personal-https-token) to grant the token the required scope within that person's existing repository permissions. Use [Direct SSH and development access](../30-Projects-and-development/40-connect-and-develop.md) when a person needs an explicitly installed key for an editor or file transfer.
