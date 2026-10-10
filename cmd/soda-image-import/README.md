# soda-image-import

`soda-image-import` makes the images bundled with a Soda appliance available in
the host's ordinary Podman storage. It is a root-only boot service helper for
appliance operators; normal project users do not run it.

## Normal use

The installed `soda-image-import.service` runs
`/usr/libexec/soda/soda-image-import` before `soda-host.service`. The command
accepts no arguments and has no configurable input paths.

For an operator diagnosing a failed import, inspect the service first:

```sh
systemctl status soda-image-import.service
journalctl -u soda-image-import.service
```

An explicit manual import on the installed appliance uses root access:

```sh
sudo /usr/libexec/soda/soda-image-import
```

## Inputs and effects

- Requires native Linux x86_64, `/usr/bin/podman`, installed release metadata at
  `/usr/share/soda/release.json`, and its OCI layout at `/usr/share/soda/images`.
- Validates the bundled content and its release bindings before importing images
  for dashboard, Forgejo, extension, proxy, Project OS and Tailnet.
- Imports missing exact image identities locally; images already present are
  retained. It does not start, replace or delete workloads.
- Returns `0` on success and `1` on failure, with a diagnostic on stderr. The
  command has a ten-minute deadline; the service allows eleven minutes.

See [release architecture](../../docs/architecture/release.md),
[installation](../../docs/guides/installation.md), and the
[service definition](../../system/host/image/soda-image-import.service).
