# Working inside a project

Adapted from the predecessor's developer handbook for project-local identities and shared resources. The [Project OS baseline](project-os.md) owns supported userspace, native permissions, persistent state and same-root maintenance. Recorded x86_64 clients have bounded SSH/file-transfer, personal Git, shared-tool and lifecycle evidence, not general laptop/client or final-product acceptance; see the [current handoff](implementation-status.md#accepted-native-evidence).

## Join, verify and connect

Sign in through Forgejo and explicitly connect Sodaspaces. Today's Join requires a saved public development-access key: use the drawer's key controls, then select **Add me to this project**. The creator joins explicitly too. No private key is uploaded and no human account is created on the appliance host.

The selected browser-only path will provision the same real account without an external SSH key; it is not implemented yet. The browser terminal itself uses Soda's bridge, not SSH. Optional reviewed Forgejo profile-key selection and automated outbound Git setup are separate unfinished actions, not current behavior. SSH/editor instructions below remain native alternatives, not the intended prerequisite for development in the drawer.

Use the project's displayed IP, not its label as a hostname. The operator must establish the real client route to the project subnet; a bridge address alone is not connectivity. Before accepting a first SSH host-key prompt, obtain the project's fingerprint through trusted operator access. On the authorized host, the operator can inspect the public host key using:

```sh
podman exec soda-PROJECT_ID ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
```

Substitute the actual native project ID. Do not disable host-key checking or accept an unexplained key change. With the verified address, ordinary commands and transfers apply:

```sh
ssh alice@PROJECT_IP
ssh alice@PROJECT_IP 'whoami; printf "%s\n" "$HOME"'
scp ./local-file alice@PROJECT_IP:~/
sftp alice@PROJECT_IP
```

An optional client-side entry makes key selection clear:

```sshconfig
Host soda-example
    HostName PROJECT_IP
    User alice
    IdentityFile ~/.ssh/YOUR_PERSONAL_PRIVATE_KEY
    IdentitiesOnly yes
```

This alias exists only in your client's SSH configuration. Soda does not provide project DNS or an SSH gateway.

## Editors and personal checkouts

VS Code's **Remote - SSH** extension can connect to the same account/IP; open your ordinary checkout under your project home. Follow Microsoft's [Remote SSH guide](https://code.visualstudio.com/docs/remote/ssh) for remote extensions and forwarded ports. Other SSH-capable editors and terminal editors use the same native access. Keep editor servers, extensions and personal credentials in your own project home.

- `~/shared` accesses `/srv/project/shared`, shared through the native `soda-project` group.
- Clone ordinary repositories into your home, such as `~/repo-name`; there is no mandatory repository schema or Soda checkout synchronization.
- The environment's Forgejo repository association determines web management authority. Existing native wheel/SSH rights are not automatically synchronized after transfer; see the [authority boundary](project-os.md#ownership-and-trust). Other checkouts do not change either authority or the Forgejo identity provider.

## Git credentials and collaboration

Soda's registered public key authenticates **inbound project SSH**, not Git operations from inside the project. Use your own native Forgejo/external-host Git credentials. There is no automatic key generation, registration, agent forwarding or copied browser session for Git.

For deliberate manual Git setup, generate an outbound SSH key inside your project home using native `ssh-keygen`, with a passphrase and an unused filename. Register only its public part through the Git host's key settings. Keep its private part in that home with normal restrictive permissions; never store it in `~/shared` or the repository. A Forgejo profile key carries your normal cross-repository permissions; project-local storage does not scope it to one repository or hide it from project sudo/appliance root. Verify the Git host's fingerprint through its operator/published guidance before accepting it. HTTPS credentials are a separate native option. This manual workflow does not settle the [automated credential model](sodaspaces-plan.md#ssh-directions-and-the-proposed-git-setup).

Copy the actual clone URL from Forgejo; do not construct it from the project IP or an old appliance port. Configure Git author identity as appropriate and use the ordinary workflow:

```sh
git clone CLONE_URL ~/repo-name
cd ~/repo-name
git config user.name "Alice Example"
git config user.email "YOUR_GIT_EMAIL"
git remote -v
git switch -c my-change
# Edit, commit, push and review through the authoritative Git host.
```

Repository permissions remain with that host. Joining the Soda environment does not grant repository access. A local commit is not a remote backup, and a Git merge does not install tools, migrate a database, restart services or remove temporary work.

## Native provider CLIs

The project image recipe includes Tea and GitHub CLI, with native personal authentication rather than shared credentials. See [project CLIs](project-clis.md) for source inputs, build requirements and later login guidance. Their versions/availability were checked in the native x86_64 projects; that does not imply authenticated Tea/GitHub CLI sessions. Personal native Git SSH was exercised separately.

## Shared installed tools

The shared mise data/install/shim tree is `/opt/mise`; its global version configuration is `/etc/mise/config.toml`. These native settings come from mise's configuration interfaces. Shared installations remain within one project filesystem, not copied between machines.

As the project administrator:

```sh
sudo -i
umask 022
mise use --global node@24
exit
```

Alice and Bob should then resolve the same installed Node through `/opt/mise/shims`:

```sh
mise where node
node --version
```

The profile and SSH server set the environment for interactive and non-interactive sessions. Ordinary members can consume but not overwrite the root-owned shared installation. Two matching version strings alone are not proof of a shared install—check the actual paths and permissions.

A repository may carry a normal `mise.toml`. Review it before `mise trust`; trust is not an instruction to execute unknown repository code blindly. Missing tool versions require installation through the appropriate native permissions. Do not tell every member to run `mise install` against an unwritable shared tree or silently substitute independent per-user downloads.

Personal tools can coexist under personal paths, with explicit personal mise environment overrides if desired. There is no Soda-owned private-toolchain clone/selector. Authenticate assistants and other tools with your own credentials; do not borrow shared tokens or another person's home.

## Services, ports and persistence

Project-native root/wheel administrators administer the shared nested engine; other members consume its normal service endpoints. See [project services](project-services.md) for that boundary and the ordinary Compose example. Unlike the predecessor, different project environments do not share one host network namespace, so coordinate ports within the project rather than assuming an appliance-wide workspace port pool.

For a service listening only on the project's loopback interface, forward it from your client:

```sh
ssh -N -L 8080:127.0.0.1:8080 alice@PROJECT_IP
```

Open `http://127.0.0.1:8080` on that client. For team access, bind/publish to the intended reachable project interface and use the deployment's permitted private route. Never open public ingress merely to bypass a routing problem.

Open a new shell after changing shell configuration; this is native shell behavior, not live propagation by Soda. Coordinate service changes and authorized stop/start with teammates. Normal startup preserves the existing writable root; a new image does not update it. Follow [bounded same-root maintenance](project-os.md#deliver-required-additions-without-replacing-roots) for required platform additions; its concrete delivery recipe is still source work, not permission to upgrade a project. General recovery/image replacement remains deferred. Recorded default-bridge and stop/start/reboot evidence is bounded to its candidate/target; repeat affected [installed checks](native-validation.md) for a new runtime, rather than treating those results as universal compatibility or terminal-process persistence.
