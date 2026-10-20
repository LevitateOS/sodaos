# Collaborate in a Forgejo repository

Forgejo owns repositories, Git access, issues, pull requests, reviews, and releases. Use its native pages and permissions for collaboration; repository activity does not create a Soda Project or deploy an application.

## Create or import a repository

To start a repository or bring existing source into Forgejo:

1. Create an empty repository or use Forgejo's import flow.
2. Choose its owner and visibility, then grant collaborators the access they need.
3. Confirm the repository appears in Forgejo before copying its displayed clone URL.

A human repository owner creates the persistent development Project separately from repository settings; importing or cloning code does not create one. See [Projects and workspaces](../50-Spaces-and-Projects/20-projects-and-workspaces.md).

## Clone and authenticate

Copy the clone URL Forgejo shows for the repository and authenticate with your own native Forgejo Git credentials. For HTTPS, register or provide a token through Forgejo's supported personal authentication flow. For SSH Git, add the Git public key in your Forgejo account settings. A successful clone gives you a local checkout of that native repository.

Project SSH access keys log you into a Project; they do not grant native Git access. To install a key for Project login, follow [Connect and develop](20-connect-and-develop.md). Keep Forgejo and provider credentials in their respective account settings, never in shared Project files or repository content.

## Work with issues and pull requests

Create branches and commits with your Git client, then push them to the native repository. Use Forgejo issues and comments to discuss work. Open a pull request when a change is ready for review; reviewers can comment, request changes, or approve according to repository policy.

Repository administrators manage collaborators, protected branches, and required checks in Forgejo. If a required check is unavailable, inspect the repository's Actions status and runner configuration. A coding or review session does not provide CI capacity. The [Forgejo user guide](https://forgejo.org/docs/latest/user/) covers native user and collaboration features in more detail.

## Review native delivery records

Forgejo keeps the record of pull requests, reviews, Actions runs, native releases, wiki pages, and packages where enabled. Soda Factory automation may propose work and publish a candidate under separate authorization; its evidence and intervention belong to [Factory review and merge](../40-Factory/50-review-and-merge.md) and [CI runners](../40-Factory/40-ci-runners.md). Human repository collaboration remains in Forgejo.

For packaged `tea` and `gh` command-line clients, see [Shared tools and files](30-shared-tools-and-files.md). Personal provider credentials are distinct from Forgejo Git authentication and Factory grants.
