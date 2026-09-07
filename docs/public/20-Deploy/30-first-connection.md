# Make the first connection

Sign in to Soda, register your public development key and establish trusted access to your project environment.

## Obtain the connection information

Your operator supplies the Soda HTTPS URL, Forgejo identity/onboarding details
and the approved LAN or Tailnet route. Browser origins, the host's operator SSH
address, Forgejo's Git endpoint and each project's IP are different endpoints.
Do not substitute one for another.

For cloud access, connect your client to the permitted Tailnet. The operator
must also establish and approve the project-subnet route; reaching the dashboard
does not by itself prove that you can reach a project. See
[Tailscale](../30-Use-Soda/40-tailscale.md#route-the-project-subnet).

## Trust the browser endpoints and sign in

Open the exact Soda HTTPS URL. Check an unexpected certificate or hostname with
your operator before entering credentials. For a private certificate authority,
install the verified CA through your client's normal trust mechanism. Do not
disable browser or CLI TLS checks to bypass a warning.

Select **Sign in** and complete Forgejo's login, initial password change, MFA and
consent as applicable. You return to Soda as your Forgejo identity. Use a private
channel for an initial password; never paste it into a project or issue.

## Add your public development key

Use an existing personal SSH key or create one on your client:

```sh
ssh-keygen -t ed25519
```

Choose a passphrase and an unused filename; do not overwrite an existing key.
Keep the private file on that client. Open your Soda **Profile**, find the
public development-access keys and register only the `.pub` file's contents.
Check the resulting fingerprint.

These keys are installed into a project's account when you explicitly join.
They are not your Git-host key registry, and subsequent profile edits do not
synchronize existing project accounts. See [People and access](../50-Operate/10-people-and-access.md).

## Join and obtain the project address

Open the environment you intend to use and select **Add me to this project**.
Wait for its real provisioning result, then use its connection details: your
project-local login, current IP and public SSH host key. The repository owner
must join too. See [Projects and workspaces](../30-Use-Soda/20-projects-and-workspaces.md).

`PROJECT_USER` and `PROJECT_IP` below mean those displayed values. An example IP
in documentation is not the address of your installation.

## Verify and test SSH

Compare the first SSH prompt with the project's public host-key fingerprint
obtained through the trusted dashboard or your operator. If the page supplies
the public key rather than a fingerprint, save that **public** key to a file on
your client and inspect it with `ssh-keygen -lf PROJECT_HOST_KEY.pub`.

Connect using the corresponding personal private key:

```sh
ssh -i /path/to/personal_private_key PROJECT_USER@PROJECT_IP
```

Accept only a matching identity. A later unexpected host-key change needs
investigation, not disabled checking or blind deletion of a known-host entry.
The operator can verify a project's public key through native host access.

In the session, `whoami` should show your project-local login and `$HOME` your
own project home. Commands and transfers use the same native account:

```sh
ssh PROJECT_USER@PROJECT_IP 'whoami; printf "%s\n" "$HOME"'
scp ./local-file PROJECT_USER@PROJECT_IP:~/
sftp PROJECT_USER@PROJECT_IP
```

Use `-i` or your client SSH configuration if the desired key is not selected by
default. [Connect your editor](../40-Develop/10-connect-and-develop.md) and clone
with your own Git credentials. Project access does not require a host account
or developer login to Cockpit.

## If access fails

- **Timeout or no route:** check the actual client route, project state and
  private firewall policy with the operator. Do not open public ingress.
- **Permission denied:** confirm the displayed login, selected key and completed
  join. A network error is not an authentication decision.
- **Browser works, Git fails:** use the repository's real clone URL and separate
  Git key/permission, not the project SSH endpoint.
- **Unavailable environment:** preserve the diagnostic and ask the operator to
  inspect it; do not create a replacement to recover files.
