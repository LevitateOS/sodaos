# Data safety and removal

Separate disposable run cleanup from removal of persistent projects, retained results, credentials and repository data.

Use the [factory operator interface](../30-Use-Soda/15-software-factory.md) to
cancel and reconcile recorded run resources. Its authority does not extend to
persistent human projects or unrelated containers. Identify exact ownership
before any separate native deletion.

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

## Repository deletion

Forgejo owns repository deletion and its native safeguards. Deleting a Git
repository does not delete or back up an associated Soda environment. Likewise,
removing a local checkout does not revoke the remote account or its keys.
Coordinate linked repository ownership/destructive changes with the operator.

## After unexpected loss or a partial failure

Stop affected writes, retain non-secret diagnostics and identify what remains.
Do not reset, reinstall or recreate project state to make inspection look clean.
Use your tested restoration procedure and account for writes newer than the
backup. [Fallback](../30-Use-Soda/60-updates-and-fallback.md#understand-fallback-limits)
changes software/deployment selection; it is not an undo command for deleted data.

## Disposable factory resources

A run owns its checkout, scratch space, recorded containers and network. Use
`cancel` and `recover` through the [factory operator interface](../30-Use-Soda/15-software-factory.md)
to terminate and reconcile these resources. Confirm cleanup independently of the
run outcome. Never select resources by a broad name prefix or use global pruning.

Retained results, ledger records and protected credential enrollment have distinct
retention needs; do not treat them as anonymous scratch. Persistent human Projects
are outside run cleanup, even when they use the same repository. Deleting a
repository or changing an issue does not authorize deletion of human project data.
