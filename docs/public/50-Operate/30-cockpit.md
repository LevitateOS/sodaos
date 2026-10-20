# Use Cockpit for host administration

Use Cockpit to inspect and administer the SodaOS host through its stock system pages. Sign in with the host's root/operator credentials and keep access on a private operator network.

Cockpit provides the stock administration interface for the SodaOS host. The host root/operator credential is distinct from Forgejo, Project and application accounts.

## Connect privately

Cockpit listens on TCP port 9090. Reach it only over a private network or through an SSH tunnel from an authorized operator client. Do not expose host administration to the public internet.

From a client with authorized SSH access, replace `OPERATOR` with the existing SSH account and `HOST` with the appliance address. For an appliance, forward to its loopback listener:

```sh
ssh -N -L 127.0.0.1:19090:127.0.0.1:9090 OPERATOR@HOST
```

For Docker, connect to the outer host. The Soda runtime shares its network, so
the default Cockpit listener is also reachable through the outer host's loopback
address. Replace `DOCKER_HOST` with that host's SSH address:

```sh
ssh -N -L 127.0.0.1:19090:127.0.0.1:9090 OPERATOR@DOCKER_HOST
```

The SSH server connects to the forwarding destination. If an operator has
restricted Cockpit to a particular listening address, use that address as the
destination instead. The Docker installer's browser bind address does not
publish or restrict Cockpit's port.

Keep that terminal open and visit `https://127.0.0.1:19090` in the same client's browser. The forward listens only on client loopback. SSH authenticates the tunnel; the Cockpit page separately requires the host root/operator credential, or the [runtime root credential for Docker](#set-the-docker-runtime-password). Close the SSH session when finished.

HTTPS still checks Cockpit's certificate. A certificate for the host name can report a name mismatch at the loopback address, and a private certificate can require client trust. Verify the endpoint and certificate through your trusted operator path before entering credentials; the SSH tunnel does not make an unexpected certificate trustworthy. Confirm the overview identifies the intended host/runtime and that its services and journal load.

If the client cannot reach the port, first check its route and the host firewall, then confirm the Cockpit service is active from a local console or another authorized host session.

## Set the Docker runtime password

If the Docker runtime's root password has not been set, establish it before
signing in to Cockpit. On the outer Docker host, run this command from the
original deployment directory as the operator who ran the installer. Prefix
the command with `sudo` if that account requires it for Docker access:

```sh
docker compose exec soda passwd
```

Enter and confirm the new password at the hidden prompts. After the command
reports success, sign in to Cockpit as `root` using that password. This sets the
runtime's Linux credential; it does not change the outer host's SSH password
or any Forgejo, Project or application account.

## Inspect the host

The stock pages cover system overview, journal logs, storage and mounts, networking and firewall, system services, Podman containers, terminal, files and host accounts. Use them to identify failed services, capacity pressure, network configuration and runtime state.

Review diagnostic output before sharing it; it can contain private configuration. A host Linux account does not grant Forgejo, Project or application identity.

## Appliance and Docker contexts

For an appliance, Cockpit administers Fedora CoreOS and Soda's host services. For Docker, Cockpit inside the Soda runtime can inspect that runtime, while the Linux operator separately administers the outer host and Docker Engine.

The Docker host's SSH credentials and the runtime root password are distinct. Use the outer host's own management path for its firewall, forwarding, disks and daemon.

Cockpit does not replace Soda's procedures for Project maintenance, application lifecycle or qualified release transitions. Follow [Administration](20-administration.md) to choose the component owner, then use [Troubleshooting](10-troubleshooting.md) when a symptom needs routing to a repair procedure.
