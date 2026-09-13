# Projects and workspaces

Choose a repository, create its shared project, then open a terminal and work.
Creating a project, starting it, joining it and creating a terminal are separate
operations. Browser terminals do **not** require SSH keys or Tailnet.

These instructions describe the new Spaces journey. Older installations may still
show the earlier environment/session controls until a compatible upgrade.

## Choose the repository

1. Open **Spaces** in the bundled Forgejo and connect to Soda when prompted.
2. With no projects, choose **Create project**. With existing projects, use the
   same action in **Projects**.
3. Search the repositories you own on **this Forgejo**, select one and continue.
   Use **Create a new repository** for Forgejo's native creation page, then return
   to Spaces and search again. Repository creation does not create a project.

The repository's **human owner** creates and administers its project. Collaborator,
organization/team or site-administrator access alone does not make a repository
eligible for creation. Existing shared projects appear in Projects. An existing
project is something to open or inspect, not a reason to provision another root.

You can clone other repositories, including external Git hosts, inside an existing
project. Those checkouts do not change its owner or Soda identity. See
[Built-in Git](30-forgejo.md).

## Create the project

1. Review the selected repository and available **Project OS**.
2. Leave optional Network settings Off unless your operator has configured and
   authorized their use. No network enrollment is needed for a browser terminal.
3. Choose **Create project** once and wait for its result.
4. Inspect any incomplete or uncertain result before another action.

**Change repository**, Back and **Cancel setup** do not provision anything. They
preserve active-flow choices; leaving or reloading never replays a mutation.
Cancelling the view does not undo work already submitted.

Creation provisions a persistent Rocky Linux development system. It does not clone
personal checkouts, join the creator or create a terminal. A recorded reservation
is not necessarily a usable project. Successful provisioning may already leave the
project running; do not issue another Start unnecessarily.

## Join this project

If stopped, its project administrator can explicitly **Start project**. Other users
must ask the administrator. Unavailable native state needs inspection, not replacement.

Choose **Join project** when the project is ready. This browser-only join creates
your own project-local Linux account/home without SSH keys. Confirm the displayed
account before creating a terminal. Everyone joins explicitly, including the creator.
The repository owner receives project-local administration; ordinary members do not.
Joining does not grant Git access or configure outbound Git credentials.

## Open a terminal and work

Choose **New terminal**. Spaces uses a default name such as **Terminal 1** in the
selected project; no layout or SSH configuration is required. Wait for connection,
then type in the terminal. Your home is personal and `~/shared` points to shared files.
Clone into your home using your own Git credentials, then follow
[Connect and develop](../40-Develop/10-connect-and-develop.md).

Use Projects to switch context. Existing terminals keep their original project and
account. Selecting a project does not move a shell into it. Re-entering Spaces can
attach the same authorized terminal; it does not create a replacement.

- **Hide terminal** hides/detaches its view and preserves the process.
- **End terminal** requires confirmation and stops that named terminal.
- Splitting a pane changes layout, not shell count. Layout controls live in pane
  chrome and do not need to be used for a single terminal.
- **Project settings** separates Overview, Access and Network from terminal work.
  Stopping the whole project is a separate, explicit administrator operation.

## Optional SSH or editor access

Use Access settings to manage public development SSH keys. Never supply a private
key. Review/apply keys explicitly; changes to saved keys are not silently installed
in existing accounts, and Git keys are not automatically development-access keys.

Use the **displayed login and project IP**, not a guessed DNS name or the appliance
browser hostname. Your operator must provide the route. Verify the ordinary SSH
host key using [First connection](../20-Deploy/30-first-connection.md#verify-and-test-ssh).
Tailnet reachability does not replace account or application authorization.

## Incomplete or unavailable state

A failed or incomplete inventory is not proof that you have no projects. Retry the
read, or reload when the page's account context has changed. Do not treat failed
account provisioning as successful membership.

If a creation reply is lost, Spaces reads the original repository's state rather
than automatically creating again. Use **Refresh status** to inspect remaining
uncertainty. Preserve the project identity and non-secret diagnostic for your
operator. Opening, refreshing, hiding or changing layout never requests a new root.
Keep existing roots, accounts and data intact; see
[normal maintenance](60-updates-and-fallback.md).
