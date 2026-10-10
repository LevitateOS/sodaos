# soda-host

The privileged Soda host service provisions and operates project containers,
accounts, development access, Tailnet companions and managed terminal/factory
execution. This Cargo package supplies the `soda-host` and
`soda-forgejo-tailnet` binaries, plus the `soda_host` Rust library.

## Installed use

Install the service through the [installation guide](../../docs/guides/installation.md).
On an installed appliance, systemd activates `soda-host` through the restricted
`/run/soda/host.sock` socket. It is owned by `root:soda` with mode `0660`;
the daemon runs as root. Product clients call its fixed private operation surface.

Read the daemon's available flags or inspect service health:

```sh
/usr/libexec/soda/soda-host --help
systemctl status soda-host.socket soda-host.service
journalctl -u soda-host.service
```

`--config PATH` defaults to `/etc/soda/host.json`. `--release PATH` defaults
to `/usr/share/soda/release.json` and overlays installed release metadata.
`--listen-path PATH` binds a socket directly for fixtures and still requires
root. An empty `--release` disables the overlay for fixtures.

The `--tailnet-action run|stop --project ID` form operates a project's companion
lifecycle and is used by the installed service wiring. Preparation failures exit
`78` and require observation before an explicit retry; other runtime failures
exit `1`, and invalid flags exit `2`.

## Forgejo SSH advertisement

`soda-forgejo-tailnet` is a root-only installed helper with no flag parser. It
reads the native Tailnet endpoint and Forgejo listener, updates the SSH domain
in `/etc/soda/forgejo.env`, and may restart Forgejo when a refresh is needed.
Invoke it only for an intended advertisement refresh on the selected host; it
is not a read-only status command:

```sh
sudo /usr/libexec/soda/soda-forgejo-tailnet
```

Procedures and networking authority live in
[operator setup](../../docs/guides/operator-setup.md) and
[networking](../../docs/architecture/networking.md).

## Library consumers

The [`soda_host` modules](src/lib.rs) expose project operations, accounts,
daemon routing, preparation, terminal/factory lifecycles and Tailnet adapters.
These are internal appliance integration APIs. Native backends execute privileged
host operations; socket access and caller authorization remain required.

Use [Projects](../../docs/product/projects.md) and
[Project OS](../../docs/reference/project-os.md) for project semantics,
[terminal reference](../../docs/reference/terminal.md) for terminal behavior,
and the [trust model](../../docs/architecture/trust.md) for privilege boundaries.
