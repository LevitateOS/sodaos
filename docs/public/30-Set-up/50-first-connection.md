# Make your first Project connection

Join an existing Project in the browser and open your persistent human terminal. Account-only Join works without an SSH key or AI provider enrollment when your Forgejo access permits it.

Browser Join uses your authorized Forgejo account and does not require an SSH key or an AI provider account.

## Join the repository Project

Start from the repository's native Forgejo page and open its Spaces/Project view. Select Join and review the account and Project being joined. Join requires current code-write permission on that repository; if the action is denied, ask a Forgejo repository administrator to grant the appropriate permission and retry. A Soda operator role by itself does not grant repository access.

You can join with an account only. External SSH key enrollment is optional and applies when you need direct SSH, SCP, SFTP or a remote editor. Never provide a private key. If you do add a public key for direct access, the Project administrator installs it into your Project account; see [Direct SSH and development access](../60-Develop/20-connect-and-develop.md).

## Open a human terminal

After Join completes, open the repository Project's workspace terminal and run a simple command such as `whoami` or `pwd`. The browser terminal runs as your Project Linux account in that repository's persistent environment. If the Project is stopped, Start it through its Project controls, then retry terminal creation. If Join or terminal creation returns an uncertain result, inspect the existing membership or session before repeating the action; do not create a second Project to recover.

## Understand access boundaries

Project membership and native Forgejo permission are separate. Losing repository permission can block managed access, while a shell already running in the Project may remain. Use the [Project guide](../50-Spaces-and-Projects/20-projects-and-workspaces.md) for membership and lifecycle details, and [workspace terminals](../50-Spaces-and-Projects/40-workspace-terminals.md) for reconnecting to a retained session. A provider account is needed only for an AI CLI, not an ordinary human shell.

## Continue development

After the terminal is ready, use [Forgejo collaboration](../60-Develop/10-collaboration.md) for native repository work, [shared tools and files](../60-Develop/30-shared-tools-and-files.md) for the Project environment, and [Project services](../60-Develop/40-project-services.md) when the repository needs a persistent development service.
