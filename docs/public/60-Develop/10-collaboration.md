# Repository collaboration

Forgejo is the native home for repositories, Git access, issues, pull requests, reviews and releases. Use its repository pages and permissions for collaboration; repository activity does not create a Soda Project or deploy an application.

## Create or import a repository

Create an empty repository in Forgejo when starting new work, or use its import flow to bring in an existing repository. Choose its owner and visibility deliberately, then grant collaborators the access they need. A human repository owner can create the repository's persistent development Project separately from repository settings; importing or cloning code alone does not create one. See [Projects and workspaces](../50-Spaces-and-Projects/20-projects-and-workspaces.md).

For command-line work, copy the clone URL shown by Forgejo for that repository. Use your own native Forgejo Git credentials. For HTTPS, register or provide the token through Forgejo's supported personal authentication flow; for SSH Git, add the Git public key in Forgejo's account settings. Project SSH access keys serve a different purpose: they log you into a Project and do not grant native Git access. [Connect and develop](20-connect-and-develop.md) covers that Project access path.

Create branches and commits with your normal Git client, then push them to the native repository. Use Forgejo issues and comments to discuss work, and open a pull request when a change is ready for review. Reviewers can comment, request changes or approve according to repository policy. Repository administrators configure collaborators, protected branches and required checks in Forgejo. If a required check is unavailable, inspect the repository's Actions status and runner configuration; a coding or review session does not provide CI capacity. The [Forgejo user guide](https://forgejo.org/docs/latest/user/) covers native user and collaboration features in more detail.

Forgejo retains the record of pull requests, reviews, Actions runs and native releases. Soda Factory automation can propose work and publish a candidate under its separate authorization, but its evidence and intervention process belong to the [Factory review and merge](../40-Factory/50-review-and-merge.md) and [CI runner](../40-Factory/40-ci-runners.md) procedures. Human repository collaboration remains in Forgejo.

Repositories may also use their native wiki, packages and release features where enabled. Keep credentials in your account's native settings, not in shared Project files or repository content. For the packaged `tea` and `gh` command-line clients, see [Shared tools and files](30-shared-tools-and-files.md). Personal provider credentials are distinct from Forgejo Git authentication and Factory grants.
