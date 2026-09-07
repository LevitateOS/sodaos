# Projects and workspaces

Create a persistent environment for a repository, then explicitly join it to receive your own Linux account and connection details.

## Choose the repository

Start in the Soda dashboard with a repository in the bundled Forgejo. Create a
new one, import an existing repository through Forgejo's supported import flow,
or select one already there. Repository creation and environment creation are
separate actions. Git access remains controlled by the repository's owner.

The repository's **human owner** creates its environment and administers that
project. One environment is associated with each repository. An existing
environment is something to open, not a reason to provision another one.
Organization/team repository permissions do not assign a Linux project administrator.

You can also clone other repositories, including external Git hosts, inside an
existing environment. Those checkouts do not change the environment's owner or
Soda's Forgejo-backed identity. See [Built-in Git](30-forgejo.md).

## Create the environment

1. As the repository owner, open the eligible repository in the dashboard.
2. Under its development-environment section, select **Create persistent Rocky
   environment** and review the creation result.
3. Wait for the result and open the environment's detail page.
4. Review its repository association and observed native state.

Soda creates a persistent Rocky Linux environment with SSH, mise and the
project-local workload tools. It does not clone a personal checkout or silently
join the creator. Do not treat a saved reservation or unavailable native state
as a usable environment; read the reported provisioning result.

## Add me to this project

1. Register a valid public development-access SSH key in your
   [profile](../20-Deploy/30-first-connection.md#add-your-public-development-key).
2. Open the intended environment and select **Add me to this project**.
3. Wait for account provisioning to complete. Confirm your membership and the
   login shown by the environment's connection details.
4. Check the project IP and SSH host-key identity before connecting.

Every person joins explicitly, including the creator. A successful join creates
your project-local Linux account/home, installs your registered public keys and
establishes access to shared files. The repository owner receives project-local
administration; ordinary members do not receive sudo or engine administration.

Joining does not grant Git access or register an outbound Git key. Later changes
to profile keys do not automatically update keys already installed in projects.
Existing membership is not a key-resynchronization action.

## Connect and work

Use the **displayed login and project IP**, not the appliance browser hostname
or a guessed project DNS name. Your operator must provide a route from your
client to that project's subnet. Verify its host key as described in
[First connection](../20-Deploy/30-first-connection.md#verify-and-test-ssh).

Open your existing workspace terminal in the dashboard, or connect using ordinary
SSH. Your home is personal; `~/shared` points to the project's shared files.
Clone into your home using your own Git credentials, then follow
[Connect and develop](../40-Develop/10-connect-and-develop.md).

## Incomplete or unavailable state

Missing keys must be corrected before joining. Native account/setup failures
are not successful membership. If a response is lost, refresh and inspect before
retrying; resources may already exist. Give the operator the non-secret diagnostic
and project identity if the native state cannot be inspected.

A stopped or unreachable environment needs operator inspection, not replacement.
Opening its page or terminal is not a request to recreate it. Keep its existing
root, accounts and data intact; [normal maintenance](60-updates-and-fallback.md)
starts the existing project again.
