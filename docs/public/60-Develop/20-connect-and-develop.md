# Connect to a Project and develop

Use a browser terminal or ordinary SSH to work in your joined Project account. Browser development needs no SSH key or AI provider; an external editor and file transfer need an SSH route.

## Prepare SSH access

Join the repository's Project and confirm that your native Forgejo account has current code-write permission. In the Project access details, review the Linux login, reachable address, and SSH key fingerprint. Install a public key only through the explicit Project key application action. Never upload or share a private key.

Check the displayed fingerprint against the host key your SSH client receives before accepting a first connection. Saved keys and installed access are separate: removing a saved key does not itself revoke a copy already installed in a Project.

## Connect from a client

From a client with a network route to the Project, replace `USER` with the displayed Linux login and `PROJECT_ADDRESS` with that Project's reachable address:

```sh
ssh USER@PROJECT_ADDRESS
```

A successful connection opens a shell under your Project account. Use the same `USER@PROJECT_ADDRESS` target for compatible SSH-based editors and file-transfer clients. SCP and SFTP use the Project's ordinary SSH service.

Clients need a real route to the Project address; appliance Tailnet enrollment alone does not provide one. For LAN or Tailnet routing and private hostnames, see [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md).

## Diagnose connection failures

If the server refuses the key, check that the public key was explicitly applied to this Project and that you used its displayed Linux login. If the host fingerprint differs from the Project's displayed value, stop and verify the address or host identity before connecting. If the address times out, confirm the client's route and Project reachability with its administrator. Changing repository Git credentials does not fix SSH access to the Project.

## Keep checkout and credential ownership clear

Keep personal clones and working directories in your home. Use native Forgejo authentication when cloning or pushing: Project login keys are not Git credentials. Register outbound Git authentication with Forgejo as described in [Repository collaboration](10-collaboration.md).

Shared tools and team files are available to Project members under the shared locations; coordinate edits there with other users. Factory checkouts and their credentials remain separate from personal work. For browser-only access, open a managed [Workspace terminal](../50-Spaces-and-Projects/40-workspace-terminals.md). For shared tool administration, see [Shared tools and files](30-shared-tools-and-files.md); for persistent local databases and containers, see [Project services](40-project-services.md).
