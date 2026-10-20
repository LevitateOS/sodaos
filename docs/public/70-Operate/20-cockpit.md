# Use Cockpit for host administration

Use Cockpit to inspect and administer the SodaOS host through its stock system pages. Sign in with the host's root/operator credentials and keep access on a private operator network.

Cockpit provides the stock administration interface for the SodaOS host. The host root/operator credential is distinct from Forgejo, Project and application accounts.

## Connect privately

Cockpit listens on TCP port 9090. Reach it only over a private network or through an SSH tunnel from an authorized operator client. Do not expose host administration to the public internet.

If the client cannot reach the port, first check its route and the host firewall, then confirm the Cockpit service is active from a local console or another authorized host session.

## Inspect the host

The stock pages cover system overview, journal logs, storage and mounts, networking and firewall, system services, Podman containers, terminal, files and host accounts. Use them to identify failed services, capacity pressure, network configuration and runtime state.

Review diagnostic output before sharing it; it can contain private configuration. A host Linux account does not grant Forgejo, Project or application identity.

## Appliance and Docker contexts

For an appliance, Cockpit administers Fedora CoreOS and Soda's host services. For Docker, Cockpit inside the Soda runtime can inspect that runtime, while the Linux operator separately administers the outer host and Docker Engine.

The Docker host's SSH credentials and the runtime root password are distinct. Use the outer host's own management path for its firewall, forwarding, disks and daemon.

Cockpit does not replace Soda's procedures for Project maintenance, application lifecycle or qualified release transitions. Follow [Administration](10-administration.md) to choose the component owner, then use [Troubleshooting](90-troubleshooting.md) when a symptom needs routing to a repair procedure.
