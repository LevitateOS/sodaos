# Forgejo: built-in Git

Use Soda's built-in Forgejo for identity, repositories, Git permissions and collaboration, with your own credentials for each client.

## Sign in

Forgejo owns your account, password, MFA and repository permissions. Soda's
sign-in redirects there; the native Forgejo link opens the same service using
its configured HTTPS origin. A project Linux username/password or the host root
password is not a substitute for your Forgejo credentials.

Use account-security settings for password changes, recovery and application
consent. Follow the [Forgejo user guide](https://forgejo.org/docs/latest/user/)
for its native controls. Site administration, Soda operator authority and host
root remain separate.

## Create or import a repository

In the Soda dashboard, create a repository with the intended owner, visibility,
name and initialization settings. For existing code, use the repository import
flow and review the source host's authorization before providing credentials.
An import creates a Forgejo repository; it does not automatically keep two Git
hosts synchronized.

Open the result and verify its files and access settings. Its human owner can
then [create a Soda environment](20-projects-and-workspaces.md). Creating,
importing or forking a repository does not automatically provision an environment,
join anybody or clone their home directory.

For ongoing work hosted elsewhere, use that host's clone URL and permissions in
your project checkout. Do not assume external Git credentials authenticate you
to Soda.

## Register your Git key

Development-access keys in Soda authenticate incoming project SSH. Git keys in
Forgejo authenticate Git operations; these registries serve different purposes.

For Git from inside a project:

1. Connect as your own project-local user.
2. Generate a passphrase-protected key with `ssh-keygen -t ed25519`, choosing an
   unused filename. Keep its private half in your own home, not shared storage.
3. Register only its `.pub` contents through your Git keys page or Forgejo's
   **SSH / GPG Keys** settings, with a label identifying the project/client.
4. Copy the repository's actual SSH clone URL. Verify the Git server's host key
   with your operator before accepting a first connection.
5. Unlock your key in your own SSH agent when needed. A restarted session may
   need another unlock; browser sign-in does not unlock it.

HTTPS Git with your own supported credential is another native option. Never
put a password/token into a clone URL, shared log or tracked repository file.
No private key or browser session is copied automatically when you join.

## Clone, commit and push

```sh
git clone CLONE_URL ~/repo-name
cd ~/repo-name
git config user.name "YOUR_NAME"
git config user.email "YOUR_GIT_EMAIL"
git switch -c my-change
git status
```

Replace the values with your repository's URL and own author identity. Work in
ordinary personal checkouts. Push commits and open a pull request through your
repository's normal workflow; a local commit alone is not a remote backup.

Copy clone URLs rather than deriving them from the Soda browser origin or the
project IP. The bundled Forgejo Git SSH listener uses its own appliance endpoint,
separate from port 22 inside each project.

## Collaboration and administration

Use [issues, reviews, releases and Actions](35-collaboration.md) through their
Soda views and native Forgejo links. Repository owners manage collaborators and
protections through Forgejo's permissions. Site administrators use authorized
administration views; being a site administrator does not grant Cockpit access.

Repository access does not automatically grant or revoke project Linux access.
Coordinate destructive native repository/account changes with the Soda operator
before changing an environment's associated identity. See
[People and access](../50-Operate/10-people-and-access.md).
