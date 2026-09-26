# Collaborate on code

Authorize clear objectives, inspect exact-commit verification and merge results through Forgejo using your human account.

For agent work, follow the [factory walkthrough](15-software-factory.md). Soda
publishes the assigned candidate and a fresh reviewer submits findings. Your team
retains the final merge decision. The same native issues, branches and reviews
also support manual development.

## Issues and notifications

Open the repository's issues to report a problem or propose a task. Supply a
clear title, expected result and relevant non-secret context; use labels,
milestones and assignees within your repository permissions. Comments and
notifications belong to Forgejo, not a second Soda discussion database.

Use My work, search and notifications to find assigned issues and requested
reviews. A result is limited by your repository access. Do not paste tokens,
private diagnostics or unrelated personal data into public issues or attachments.

## Review and merge

1. Create a branch in your own checkout and push it with your own Git credential.
2. Open a pull request against the intended base branch. Review the comparison
   before submitting; a branch with a similar name is not the same revision.
3. Discuss changes, leave review comments, approve or request changes as allowed.
4. Check the latest head revision, required reviews, protections and CI results
   before merging.
5. If the branch changed or a native check rejects the merge, reload and review
   the new state rather than repeatedly submitting an old merge request.

Forgejo's rules decide whether a merge is allowed. Follow the
[Forgejo user guide](https://forgejo.org/docs/latest/user/) for native review and
repository operations. Merging does not update installed tools, apply database
migrations or restart the project's services.

## Repository settings, organizations and teams

Owners manage collaborators, branch/tag protections, deploy keys, hooks and
Actions settings through upstream-authorized controls. Organizations and teams
organize Git permissions; they do not assign project Linux accounts or host
privileges. A collaborator may need both repository access and an explicit
[Soda environment join](20-projects-and-workspaces.md).

Use separate native secrets/variables for automation. Review webhook destinations
and events before enabling them: hooks send real repository data externally.
Coordinate repository rename, transfer, archive or deletion with the operator
when a Soda environment is associated; these are not environment lifecycle tools.

## Actions and local capacity

Use Actions to inspect workflows, runs and checks. Follow the native provider
links for job logs, artifacts or run controls presented there. Forgejo owns its
scheduling and results; GitHub owns those for GitHub-hosted repositories.

Starting a local listener does not mean a job passed. The host operator manages
[local runner capacity](50-ci-runners.md) separately in Cockpit. Keep workflow
labels aligned with the registered provider labels, and only run trusted code
on the team's local runner capacity.

## Releases, wiki and packages

Use releases for tagged deliverables and their notes/assets. Verify the intended
tag/commit and visibility before publishing; a repository release is not a
Soda host update. Use the repository wiki for project documentation and its
history to review changes.

Packages belong to their provider owner/namespace. Use the exact package name,
version, files and native installation guidance shown by Forgejo. Authenticate
package clients with your own authorized credential through the native secret
channel. A package listing does not install anything in your project.

When a task opens Forgejo's own interface, it remains the same repository and
permission system. No Soda action copies upstream roles or bypasses a native denial.

## Review a factory candidate

Check the issue objective and the pull request's final head commit. Its CI result
and fresh review must reference that exact commit. A repair creates a new candidate
and invalidates the earlier evidence; verify it again before human merge.

Resolve unclear requirements or exhausted limits yourself before authorizing a
new attempt. Agent comments, labels and repeated deliveries cannot reset the
attempt's limits. Keep findings in Forgejo and infrastructure status in the linked
run record. See the [factory walkthrough](15-software-factory.md).
