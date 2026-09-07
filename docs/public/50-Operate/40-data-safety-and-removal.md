# Data safety and removal

Preserve local work and understand native deletion scope before changing accounts, workloads, repositories or storage.

## Know what is not disposable

Project environments retain account records, SSH host keys, homes, installed
tools, shared files, configuration and service data. Their writable roots are
part of the product's persistent state, not replaceable build cache.

An upstream image, Git remote or previous host deployment does not contain every
later local write. Check for unpushed commits, untracked files, database data,
private credentials and shared services before any destructive native operation.

## Before removing local material

1. Identify the exact project, account, files/volume and affected users.
2. Inspect each checkout with `git status` and review local branches. Push needed
   commits to an authorized remote, and preserve untracked/non-Git data separately.
3. Use native database backup procedures and protect important tool/configuration
   state through [tested backups](30-backups-and-restoration.md).
4. Coordinate with users and stop affected writers or jobs before removal.
5. Review the exact native command/action and its effect; retain its result and
   verify only the intended material changed.

Avoid broad `podman system prune`, `down -v`, project-container deletion and
recursive file removal as fixes for startup, route or permission failures.
A failed destructive command can already have removed data; repeated execution
is not a harmless status check.

## Project and person lifecycle

Soda does not provide a project-delete/archive/rebuild workflow or coordinated
person deprovisioning. Native deletion is not a substitute for such a workflow:
it can leave Soda records, Linux accounts and repository associations inconsistent.
Do not delete a root or row merely to remove a dashboard entry.

For a person's departure, separate data handoff from
[access revocation](10-people-and-access.md#offboarding-and-revocation).
Forgejo disablement, project SSH keys/sessions, external Git credentials and
Tailnet policy have distinct native owners. Removing one does not automatically
remove the others or safely hand over project administration.

## Repository and runner deletion

Forgejo owns repository deletion and its native safeguards. Deleting a Git
repository does not delete or back up an associated Soda environment. Likewise,
removing a local checkout does not revoke the remote account or its keys.
Coordinate linked repository ownership/destructive changes with the operator.

The [Runners page](../30-Use-Soda/50-ci-runners.md#remove-or-replace-a-runner) has
an explicit local removal operation. It destroys the selected runner's local
account/work state; the provider's record and history require separate review.
Preserve any needed job files before confirming it.

## After unexpected loss or a partial failure

Stop affected writes, retain non-secret diagnostics and identify what remains.
Do not reset, reinstall or recreate project state to make inspection look clean.
Use your tested restoration procedure and account for writes newer than the
backup. [Fallback](../30-Use-Soda/60-updates-and-fallback.md#understand-fallback-limits)
changes software/deployment selection; it is not an undo command for deleted data.
