# soda-image-import

`soda-image-import` acquires the images pinned by a Soda release into the
host's ordinary persistent Podman storage. It is a root-only boot service helper for
appliance operators; normal project users do not run it.

## Normal use

The installed `soda-image-import.service` runs
`/usr/libexec/soda/soda-image-import` before `soda-host.service`. The command
takes no arguments for boot acquisition. Trusted Tailnet preparation and
maintenance callers use `ensure --payload ABSOLUTE_PAYLOAD_PATH --set required|tailnet`;
there are no image-reference or registry override flags.

For an operator diagnosing failed acquisition, inspect the service first:

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
  `/usr/share/soda/release.json` (or the trusted ensure invocation's absolute path).
- Validates Payload v4 and verifies each selected configuration identity and
  native Linux/amd64 platform. Boot selects dashboard, Forgejo, proxy, PostgreSQL,
  Project OS and Canlang runtime. Only the explicit `tailnet` set selects Tailscale.
- Pulls missing images by sealed manifest reference; cached matching configurations
  support offline startup. It does not start, replace or delete workloads.
- Returns `0` on success and `1` on failure, with a diagnostic on stderr. The
  command has a ten-minute deadline; the service allows eleven minutes.

See [release architecture](../../docs/architecture/release.md),
[installation](../../docs/guides/installation.md), and the
[service definition](../../system/host/image/soda-image-import.service).
