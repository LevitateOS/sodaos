# soda-console-welcome

Show host operators the appliance's observed network addresses and configured
browser access guidance. This read-only renderer runs automatically for root
interactive logins and at boot through `soda-console.service`.

Project developers use their project addresses and accounts. Their everyday
workflow is described in [Develop in a project](../../docs/guides/develop.md).

## Inspect the welcome

On an installed host, run as root to display the current guidance:

```sh
sudo /usr/libexec/soda/soda-console-welcome
```

The default configuration is `/etc/soda/dashboard.json`. To inspect another
operator-supplied dashboard configuration, pass its path as the sole argument:

```sh
sudo /usr/libexec/soda/soda-console-welcome /home/operator/dashboard.json
```

The command takes no flags. More than one argument produces usage and exit
status `2`; a non-root invocation silently exits successfully.

## What it displays

The renderer uses the host's `id`, `ip`, `nmcli`, `hostnamectl` and `uname`
commands to observe local uplink IPv4 addresses and the hostname. It prints
Cockpit access on the first observed uplink, loopback SSH tunnel guidance, and
the configured Forgejo/Sodaspaces HTTPS origin. The optional `soda-tailnet`
helper supplies Tailnet status when available.

Only the dashboard listener and browser origin are read from configuration;
other configuration fields and credentials are not displayed. Unreadable or
invalid configuration produces Forgejo installer guidance. Missing network
information produces explicit unavailability guidance.

A displayed origin describes configuration; it does not establish listener
health or client reachability. The renderer changes no network or service
configuration. Its boot service separately writes the banner to
`/etc/issue.d/50-soda.issue` and binds the appliance service startup dependencies.

See [Operator console welcome](../../docs/design/console-welcome.md) for delivery
behavior and [Operator setup](../../docs/guides/operator-setup.md) for completing
the setup shown by the banner.
