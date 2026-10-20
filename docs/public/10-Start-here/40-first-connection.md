# Make your first Project connection

Join an existing Project in the browser and open your persistent human terminal. Account-only Join works without an SSH key or AI provider enrollment when your Forgejo access permits it.

Browser Join uses your authorized Forgejo account and does not require an SSH key or an AI provider account.

## Join the repository Project

1. Open **Workspace**, select the intended repository Project and open **Project settings**. Confirm the repository and the Forgejo account that will own your Project access.
2. Confirm the Project is running. If it is stopped, ask its current repository owner or the configured Soda operator to Start the existing Project before joining.
3. For browser-only access, leave **Also install my saved public keys for external SSH** unchecked when shown. Review this choice before submitting Join.
4. Choose **Join environment** and confirm the resulting Project login and membership.

Join requires current code-write permission on that repository; if the action
is denied, ask a Forgejo repository administrator to grant the appropriate
permission and retry. A Soda operator role by itself does not grant repository
access.

You can join with an account only. External SSH key enrollment is optional and applies when you need direct SSH, SCP, SFTP or a remote editor. Never provide a private key. If you do add a public key for direct access, explicitly apply it to your Project account; see [Direct SSH and development access](../30-Projects-and-development/40-connect-and-develop.md).

## Open a human terminal

After Join completes, open the repository Project's workspace terminal and run a simple command such as `whoami` or `pwd`. The browser terminal runs as your Project Linux account in that repository's persistent environment. If the Project is stopped, ask its current repository owner or the configured Soda operator to Start it, then retry terminal creation. If Join or terminal creation returns an uncertain result, inspect the existing membership or session before repeating the action; do not create a second Project to recover.

## Understand access boundaries

Project membership and native Forgejo permission are separate. Losing repository permission can block managed access, while a shell already running in the Project may remain. Complete member offboarding uses explicit [Project access revocation](../40-Configure/10-people-and-access.md#review-access-when-roles-change), which also ends that member's native sessions and processes.

Use the [Project guide](../30-Projects-and-development/10-projects-and-workspaces.md) for membership and lifecycle details, and [workspace terminals](../30-Projects-and-development/20-workspace-terminals.md) for reconnecting to a retained session. A provider account is needed only for an AI CLI, not an ordinary human shell.

## Make a first contribution

Use a small documentation change to complete the journey from browser access to native review:

1. Choose a repository where you have code-write permission and an existing README. Use a practice repository if the team does not want onboarding changes in its main source.
2. Confirm the terminal's Project and account before running commands. Join provisions your account; it does not clone the repository.
3. [Create your personal HTTPS token](../30-Projects-and-development/30-collaboration.md#create-a-personal-https-token) for the intended repository and keep it in your personal password manager. Browser sign-in and Project Join do not authenticate Git commands.
4. Follow [Clone a personal checkout](../30-Projects-and-development/30-collaboration.md#clone-a-personal-checkout) to copy the repository into your home and create a branch, entering the token when Git asks for a password.
5. Follow [Make and publish a small change](../30-Projects-and-development/30-collaboration.md#make-and-publish-a-small-change) to set your commit identity, add one onboarding sentence and open a native pull request.
6. Confirm the PR shows your branch, intended target and only that sentence. Leave merge to the repository's review policy. A PR or merge does not install or deploy an application.

Your result is a personal checkout retained in the Project and a native review record linked to your own account. No provider subscription or external SSH key is needed for this browser-terminal route; native Git authentication is still required.

## Continue development

After the terminal is ready, use [Forgejo collaboration](../30-Projects-and-development/30-collaboration.md) for native repository work, [shared tools and files](../30-Projects-and-development/50-shared-tools-and-files.md) for the Project environment, and [Project services](../30-Projects-and-development/60-project-services.md) when the repository needs a persistent development service.
