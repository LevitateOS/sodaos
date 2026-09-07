# The Soda dashboard

Use one dashboard for repository work, collaboration, your profile and persistent development environments.

## Sign in

Open the exact HTTPS Soda URL supplied by your operator. **Sign in** takes you
to Forgejo for your password, any required password change, MFA and application
consent. Check the destination before entering credentials. Return to Soda after
authorization; you do not create a separate Soda password.

Administrative consent is separate from ordinary development access. If access
is denied, ask the relevant administrator to review your permission rather than
using somebody else's token. Follow [First connection](../20-Deploy/30-first-connection.md)
for private networking and key setup.

## Find your work

| Area | Use it for |
| --- | --- |
| My work, search and notifications | Find repositories, assigned work and requested reviews within your access |
| Repositories | Browse code/history, create or import a repository, and open its environment |
| Issues and pull requests | Discuss work, review changes and merge through Forgejo's rules |
| Releases, wiki and packages | Publish or read repository documentation and deliverables |
| Actions | Inspect provider workflows and runs; provider pages own their results |
| Environments | Create an eligible repository's environment, explicitly join and find connection details |
| Profile | Soda preferences and public development-access keys |
| Git keys and account security | Your Forgejo-owned credentials and security settings |
| Administration | Permitted Forgejo administration and separately authorized Soda operations |

Follow **Open in Forgejo** or the corresponding native link when a task uses
Forgejo's own interface. It uses the same identity and native permissions, not
a second copy of repository state. Host services are administered separately in
[Cockpit](10-cockpit.md), not through a developer dashboard terminal.

## Open a workspace

Select the project you want to work in and open its environment. Join explicitly
if you have no workspace there. Once joined, open its workspace terminal in the
browser or use the displayed SSH connection details with your preferred editor.
The terminal runs as your own existing project-local account; opening it does
not join a project, create another environment or grant a host shell.

## Handle errors without duplicating work

An expired session requires sign-in again. An unavailable provider or environment
is not an empty repository and does not mean your files have disappeared. Preserve
unsaved text before navigating away from a failed form.

If a create, join, commit or other write loses its response, inspect the result
before submitting again. A timeout is not proof that nothing changed. Read the
specific diagnostic; do not delete an environment to make an error disappear.

## Sign out

Sign out before changing accounts on a shared browser. Soda sign-out ends the
Soda session, not every Forgejo session, SSH session, CLI token or running task.
Manage those through their respective tools. See [People and access](../50-Operate/10-people-and-access.md).
