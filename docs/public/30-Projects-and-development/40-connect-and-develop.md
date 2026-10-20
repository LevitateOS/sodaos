# Connect to a Project and develop

Use a browser terminal or ordinary SSH to work in your joined Project account. Browser development needs no SSH key or AI provider; an external editor and file transfer need an SSH route.

## Prepare SSH access

Join the repository's Project and confirm that your native Forgejo account has current code-write permission. In the Project access details, review the Linux login, reachable address, and SSH host-key fingerprint. Install a public key only through the explicit Project key application action. Never upload or share a private key.

Check the displayed fingerprint against the host key your SSH client receives before accepting a first connection. Saved keys and installed access are separate: removing a saved key does not itself revoke a copy already installed in a Project.

## Rotate or remove Project SSH keys

Keep a working browser terminal or existing SSH route open until verification completes. Saving a public key and installing it into a Project are separate actions:

1. In Project access settings, paste one new public key or review and select one of your native Forgejo public keys. Choose **Save public key** after checking its contents. Never paste private key material.
2. Review the target Project's installed key fingerprints alongside the saved set. Confirm both the Project and Linux login. This review does not modify access.
3. Explicitly apply the saved set to that Project. Applying replaces its managed key set with the reviewed saved set; it is not an automatic update to every Project.
4. From a second client connection, verify login with the new key before removing the old key from the saved set. If you save both during rotation, both remain installed until the next explicit apply.
5. Remove the old saved key, review the Project's installed and proposed fingerprints again, and explicitly apply the reduced set. Confirm final-key removal only when you intend to remove every managed SSH key from that Project account.
6. Verify a new connection accepts the new key and refuses the removed old key. Use a client configured to offer only the key being checked; an agent offering another installed key can hide a failed rotation.

Existing SSH sessions are not ended by a key update, and account-only browser access remains separate. Removing the saved key alone leaves installed copies unchanged. If an apply response is uncertain or the preview changes, refresh the same Project's key review and inspect its installed fingerprints before retrying. Complete member offboarding uses [Project access revocation](../40-Configure/10-people-and-access.md#review-access-when-roles-change).

## Connect from a client

From a client with a network route to the Project, replace `USER` with the displayed Linux login and `PROJECT_ADDRESS` with that Project's reachable address:

```sh
ssh USER@PROJECT_ADDRESS
```

A successful connection opens a shell under your Project account. Use the same `USER@PROJECT_ADDRESS` target for compatible SSH-based editors and file-transfer clients. SCP and SFTP use the Project's ordinary SSH service.

Clients need a real route to the Project address; appliance Tailnet enrollment alone does not provide one. For LAN or Tailnet routing and private hostnames, see [Private networking and TLS](../40-Configure/50-private-networking-and-tls.md).

## Diagnose connection failures

If the server refuses the key, check that the public key was explicitly applied to this Project and that you used its displayed Linux login. If Project access was revoked, the owner/operator must deliberately [restore that access](../40-Configure/10-people-and-access.md#restore-a-member-deliberately) before you rejoin and reinstall the intended keys.

If the host fingerprint differs from the Project's displayed value, stop and verify the address or host identity before connecting. If the address times out, confirm the client's route and Project reachability with its administrator. Changing repository Git credentials does not fix SSH access to the Project.

## Run Muse manually

Join the Project and enroll or obtain authorized use of a Muse connection through [Provider connections](../10-Start-here/50-provider-connections.md). In your own browser or SSH terminal, change to your personal checkout and run one of these entry points:

```sh
muse
```

This opens an interactive native Muse session. For the native command execution entry point, use:

```sh
muse exec
```

Native Muse help explains the arguments for the task you want to run. A manual launch uses your provisioned account and current directory; it does not create a Factory attempt or grant Factory sponsorship.

### Select a Muse connection for one invocation

With exactly one ready Muse connection authorized for your Project account, `muse` selects it when `SODA_MUSE_CONNECTION` is unset. When several are eligible, the launcher asks you to choose instead of selecting an arbitrary subscription.

```doc-pending
{
  "id": "muse-connection-id",
  "title": "Find a Muse connection ID",
  "message": "Instructions for finding an authorized connection's exact ID will be added once the supported screen or command is confirmed for both owners and delegated members.",
  "guidance": "The single-ready-connection path above does not require an ID. For explicit selection, an account label, owner name or sign-in code cannot replace the exact connection ID."
}
```

Replace `CONNECTION_ID` below with the exact ID of the intended Muse connection. It must be owned by you or explicitly delegated to you in this Project. A display label, owner name, provider name or device sign-in code is not the connection ID:

```sh
SODA_MUSE_CONNECTION=CONNECTION_ID muse
```

The inline selection applies to this invocation. An unavailable, unauthorized or non-Muse ID is rejected even if another eligible connection exists; Soda does not silently substitute that account. Current Project membership and repository permission remain necessary. See [provider account selection](../10-Start-here/50-provider-connections.md#choose-an-account-for-personal-muse-use) for ownership and delegation.

Personal settings seed a private invocation view; temporary settings or trust edits there are discarded when it ends. Edit your personal settings for lasting preferences. Subscription execution removes inherited API-key overrides.

### Diagnose a Muse launch failure

- **`choose an authorized muse connection with SODA_MUSE_CONNECTION`:** several ready Muse connections are available. Select the intended connection explicitly for the invocation.
- **`identity authority denied`:** check that the terminal belongs to your joined Project account, your repository permission remains current and the selected ID identifies a ready Muse connection authorized for that account. For delegated use, ask the connection owner to check the named-member grant for this Project. This message can also reflect a refused launch identity; it does not establish that the provider rejected a credential.
- **`muse launch service unavailable`:** ask the Project administrator to inspect the Project's launch interface. The host administrator performs any required [supported maintenance](../50-Operate/60-project-maintenance.md).
- **The provider rejects its credential:** the connection owner reconnects through [provider connection controls](../10-Start-here/50-provider-connections.md#enroll-or-reconnect-an-account). An ordinary command failure alone does not establish a sign-in failure.

## Keep checkout and credential ownership clear

Keep personal clones and working directories in your home. Use native Forgejo authentication when cloning or pushing: Project login keys are not Git credentials. Follow [personal HTTPS token setup](30-collaboration.md#create-a-personal-https-token) and its clone/push example to authenticate as yourself.

Shared tools and team files are available to Project members under the shared locations; coordinate edits there with other users. Factory checkouts and their credentials remain separate from personal work. For browser-only access, open a managed [Workspace terminal](20-workspace-terminals.md). For shared tool administration, see [Shared tools and files](50-shared-tools-and-files.md); for persistent local databases and containers, see [Project services](60-project-services.md).
