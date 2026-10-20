# Manage people and access

Grant and review access separately across Forgejo, Projects, operator controls, provider accounts and private apps. Removing one permission does not revoke every credential or erase retained work.

SodaOS uses several distinct identities. Give each person only the native Forgejo, Project and operator authority needed for their work, and review each separately when responsibilities change.

## Assign distinct roles

Forgejo owns browser sign-in, repositories, teams, issues, pull requests and native Git permissions. A user's repository permission controls native collaboration. Project membership is separately granted by the Project administrator and provides a Linux account and access to the shared repository environment.

Joining a Project requires current code-write authority on its repository. The [Project guide](../50-Spaces-and-Projects/20-projects-and-workspaces.md) owns creation, Join and Start.

The Soda operator is the Forgejo user ID recorded during appliance setup. A Forgejo site administrator who was not recorded as operator does not automatically gain Soda operator controls. The host root account administers the machine itself. These authorities should be granted only to the people responsible for those tasks.

Provider connections belong to the person who enrolled them and are held separately by the broker. They do not become shared team credentials. See [Provider connections](40-provider-connections.md) for disconnect and revocation. Private applications keep their own accounts and authorization, independent of Forgejo and Project membership.

## Review access when roles change

When someone leaves or changes roles, remove their Forgejo team/repository access and review their Project membership, operator assignment, provider grants, app-local account and any installed development SSH key separately. Signing out of the browser ends that browser session; it does not remove saved client keys. Removing a saved key from Soda's account record does not itself remove a key already installed in a Project account. Native Git access changes with Forgejo permissions, while an installed Project SSH key and an existing Linux account are separate state.

Revoking native permission can close managed browser operations and disconnect managed terminal attachments that depend on that permission. It does not delete the Linux account, automatically end a shell already running in the Project, or revoke an SSH key already installed there. An authorized Project administrator must remove installed keys and account access as a separate action. Review retained sessions and data according to your team's offboarding practices; do not assume access removal erased a person's files or history.

## Continue onboarding

For a teammate's first Project session, follow [First connection](50-first-connection.md). Use [Direct SSH and development access](../60-Develop/20-connect-and-develop.md) when a person needs an explicitly installed key for an editor or file transfer.
