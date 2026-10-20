# Connect and develop

Develop in a Project through its browser terminal or, when you need an external editor and file transfer, ordinary SSH. Both use your joined Project-local Linux account; normal browser-based development requires no SSH key and no AI provider connection.

Join the repository's Project first and confirm that your native Forgejo account has current code-write permission. From the Project's access details, review the Linux login, reachable Project address and SSH key fingerprint. Install a public key only through the explicit Project key application action. Check the displayed fingerprint against the host key your SSH client receives before accepting a first connection. Never upload or share a private key. Saved keys and installed access are separate: removing a saved key does not by itself revoke a copy already installed in a Project.

From a client with a network route to the Project, connect using the displayed login and address, for example:

```sh
ssh USER@PROJECT_ADDRESS
```

Use the same `USER@PROJECT_ADDRESS` target for compatible SSH-based editors and file-transfer clients. SCP and SFTP use the Project's ordinary SSH service. A client must be able to route to the Project's address; enrolling the appliance in a Tailnet alone does not provide that route. For LAN or Tailnet routing and private hostnames, see [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md).

If the server refuses the key, check that the public key was explicitly applied to this Project and that you are using its displayed Linux login. If the host fingerprint differs from the Project's displayed value, stop and verify the address or host identity before connecting. If the address times out, confirm the client's route and Project reachability with its administrator. These failures have different causes; changing repository Git credentials will not fix SSH access to the Project.

Keep personal clones and working directories in your home. Use native Forgejo authentication when cloning or pushing: Project login keys are not Git credentials. Register outbound Git authentication with Forgejo as described in [Repository collaboration](10-collaboration.md). Shared tools and team files are available to Project members under the shared locations; coordinate edits there with other users. Factory checkouts and their credentials remain separate from personal work.

For browser-only work, open a managed [Workspace terminal](../50-Spaces-and-Projects/40-workspace-terminals.md). For shared tool administration, see [Shared tools and files](30-shared-tools-and-files.md); for persistent local databases and containers, see [Project services](40-project-services.md).
