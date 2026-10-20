# Collaborate in a Forgejo repository

Forgejo owns repositories, Git access, issues, pull requests, reviews, and releases. Use its native pages and permissions for collaboration; repository activity does not create a Soda Project or deploy an application.

## Create or import a repository

To start a repository or bring existing source into Forgejo:

1. Create an empty repository or use Forgejo's import flow.
2. Choose its owner and visibility, then grant collaborators the access they need.
3. Confirm the repository appears in Forgejo before copying its displayed clone URL.

A human repository owner creates the persistent development Project separately from repository settings; importing or cloning code does not create one. See [Projects and workspaces](10-projects-and-workspaces.md).

## Clone and authenticate

Use your own Forgejo account with code-write permission on the repository. The first-contribution steps below use HTTPS and a personal access token, so you can follow them from a browser Project terminal without a Git SSH key. Copy the repository's displayed **HTTPS** clone URL; confirm it points to your Forgejo instance.

For SSH Git instead, add your Git public key in your native Forgejo account settings and use the displayed SSH clone URL. Project SSH access keys log you into a Project; they do not grant native Git access. [Connect and develop](40-connect-and-develop.md) owns Project login-key installation. Provider connections and Factory credentials are separate from your personal Git identity.

### Create a personal HTTPS token

In the browser, sign in to the same Forgejo instance as yourself:

1. Open your account **Settings → Applications → New access token**.
2. Enter a recognizable **Token name**, such as `handbook-first-contribution`.
3. Under **Repository and organization access**, choose **Specific repositories**. Search for the intended repository, add it and confirm it appears in the selected list.
4. Under **Select permissions**, set **repository → Read and write** (`write:repository`). Leave unrelated categories at **No access**.
5. Select **Generate token**. Copy the value while it is displayed and save it in your personal password manager on your own device. Forgejo shows it only once.

This scope permits authenticated Git reads and pushes for the selected repository, within your existing native permissions. Repository-specific tokens also allow read-only access to public repositories; they do not grant writes to unselected repositories. Token scope cannot override branch protections. See [Forgejo token scopes](https://forgejo.org/docs/latest/user/authentication/token-scope/) for other native permission combinations.

```doc-visual
{
  "placeholder": "In Applications, select your repository and its Read and write permission before generating a token.",
  "caption": "A sample personal token form: 1 marks the selected repository; 2 marks its Read and write permission. The token has not been generated.",
  "alt": "Forgejo token form with Specific repositories selected, handbook-practice added and repository set to Read and write",
  "production": {
    "method": "screenshot",
    "instructions": [
      "Use the actual Soda Forgejo interface on an approved documentation instance. Sign in as a sample ordinary contributor with code-write permission on a sample repository named handbook-practice.",
      "Open account Settings, Applications, New access token. Enter handbook-first-contribution as the token name, choose Specific repositories, add handbook-practice and set repository to Read and write. Leave unrelated permissions at No access.",
      "Capture the form before Generate token is pressed, with no generated token, existing token list, personal account information or private repository names visible. Do not create a token for this capture.",
      "Record the product revision and capture date in asset notes. Use the light theme at 1280 by 1000, keeping the access choice, selected repository and repository permission readable."
    ],
    "edits": [
      "Crop to the token form, preserving the Applications or New access token context and the real control labels.",
      "Add a rectangle numbered 1 around the selected repository and a rectangle numbered 2 around repository Read and write. Keep labels unobscured; the caption explains both markers without relying on color.",
      "Verify the caption and alt text against the real capture. Retain readable labels at article width and exclude all credential values."
    ]
  }
}
```

### Enter the token at Git's password prompt

When a clone or push asks for credentials, enter your native Forgejo username and paste the token at the **Password** prompt. The terminal does not echo the password. Use the token even when your browser account has two-factor authentication; do not enter a one-time code as the Git password. [Forgejo's HTTPS authentication guide](https://forgejo.org/docs/latest/user/getting-started/first-repository/#note-on-the-different-types-of-http-authentication) explains this native flow.

The commands below use `-c credential.helper=` to disable configured Git credential helpers for that operation. Keep the token in your personal password manager and enter it again when needed. Do not put it in the clone URL, shell commands, repository files or shared directories. Git's `store` helper writes credentials unencrypted, so do not use it for this tutorial. See [Git credential handling](https://git-scm.com/docs/gitcredentials) and [credential storage](https://git-scm.com/docs/git-credential-store).

Project root/wheel administrators are trusted with member credentials, including credentials used in their processes. Personal storage separates ordinary peers; it does not protect against a Project administrator. Review [Project administrative trust](10-projects-and-workspaces.md#review-administrative-trust-and-ownership-transfer) before using credentials there.

## Clone a personal checkout

Continue from [First connection](../10-Start-here/40-first-connection.md) in your own Project terminal. Use a repository with an existing README and native write access. Complete [personal HTTPS token setup](#create-a-personal-https-token) first.

Replace `HTTPS_CLONE_URL` below with the repository's displayed HTTPS URL, without credentials. The `handbook-first-change` directory is new in your home and must not already contain work. Run each command after the previous one succeeds; answer any Git credential prompts as described above:

```sh
cd "$HOME"
git -c credential.helper= clone HTTPS_CLONE_URL handbook-first-change
cd handbook-first-change
git status --short
git switch -c docs/first-contribution
```

The clone contains the repository's default branch, git status --short is empty, and the final command selects a new personal branch. If the directory or branch already exists, inspect it and reuse your original work or choose another name; do not overwrite it. [Git's branch reference](https://git-scm.com/docs/git-switch) explains branch selection.

A public repository may clone without asking for credentials. That confirms public read access, not your token or ability to push. The push below checks write access.

## Make and publish a small change

From this checkout, set the author identity for your contribution. Replace `Your Name` with your name and `YOUR_COMMIT_EMAIL` with the email you use for commits. Forgejo's account profile settings show a private commit address beside **Hide email address** if you prefer to use that address. These values become part of the commit; they are not login credentials:

```sh
git config --local user.name "Your Name"
git config --local user.email "YOUR_COMMIT_EMAIL"
```

The settings apply to this checkout. Git authenticates pushes with your token independently of the name and email recorded in a commit.

For this tutorial, open the existing README with your preferred editor and add one harmless sentence agreed with the team, such as “Contributors review changes through pull requests.” Use the actual README filename in place of `README.md` if it differs. Review the diff before staging:

```sh
git diff -- README.md
git add README.md
git diff --cached -- README.md
git commit -m "docs: clarify contributor review workflow"
git -c credential.helper= push -u origin docs/first-contribution
```

The staged diff contains only your intended sentence. The commit belongs to your branch. At the push's password prompt, enter your personal token again. A successful first push reports the new remote branch and sets your local branch to track it. Confirm `docs/first-contribution` and its commit appear in the same Forgejo repository.

Open Forgejo for the same repository, create a pull request from `docs/first-contribution` to the intended target branch, and explain the sentence and its purpose. Inspect the native changed-files view, author and candidate commit. Your contribution is ready for review when those match the checkout and no unrelated files appear. Required review and CI still apply; pushing does not merge or deploy the change.

## Resolve Git access failures

- **Authentication is rejected:** confirm you used this instance's personal token at the password prompt, that the target repository is selected and that **repository → Read and write** is enabled. A token you deleted no longer works. Verify your account still has native repository access.
- **The repository cannot be found:** compare the URL with the repository's displayed HTTPS URL and check that your account can view the repository in Forgejo. Private repository access also requires a token whose selected repositories include this one.
- **Cloning works but pushing is refused:** a public clone may have been anonymous, or a token may allow only reads. Check token scope and native code-write permission, then inspect the rejection for branch protections or required policy. Keep the contribution on its own branch; do not bypass protections with a force push.
- **HTTPS cannot connect or the certificate is rejected:** check the address, network route and certificate trust with the operator using [Private networking and TLS](../40-Configure/50-private-networking-and-tls.md). Changing a Git token does not repair the network or certificate.

If you lose a token, generate a replacement through the same personal flow and delete the lost token in **Applications**. Delete a tutorial-only token when you finish using it. Deleting it prevents future authentication with that token; it does not remove your commits, branch or pull request. Never borrow another person's token or Factory credentials to recover access.

## Work with issues and pull requests

Create branches and commits with your Git client, then push them to the native repository. Use Forgejo issues and comments to discuss work. Open a pull request when a change is ready for review; reviewers can comment, request changes, or approve according to repository policy.

Repository administrators manage collaborators, protected branches, and required checks in Forgejo. If a required check is unavailable, inspect the repository's Actions status and runner configuration. A coding or review session does not provide CI capacity. The [Forgejo user guide](https://forgejo.org/docs/latest/user/) covers native user and collaboration features in more detail.

## Review native delivery records

Forgejo keeps the record of pull requests, reviews, Actions runs, native releases, wiki pages, and packages where enabled. Soda Factory automation may propose work and publish a candidate under separate authorization; its evidence and intervention belong to [Factory review and merge](../20-Factory/50-review-and-merge.md) and [CI runners](../40-Configure/40-ci-runners.md). Human repository collaboration remains in Forgejo.

For packaged `tea` and `gh` command-line clients, see [Shared tools and files](50-shared-tools-and-files.md). Personal provider credentials are distinct from Forgejo Git authentication and Factory grants.
