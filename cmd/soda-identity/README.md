# soda-identity

Keep provider subscription connections private and delegate credentials to
authorized Soda executions. This package supplies the private host broker
service and its Rust library. People connect their provider accounts through
project controls; host operators provision and supervise the service.

[Credentials](../../docs/reference/credentials.md#identity-broker) owns provider
admission, grants, credential exposure and retirement. The
[HTTP API](../../docs/reference/api.md) describes the user-facing routes.

## Installed service

`soda-identity.service` runs as the dedicated `soda-identity` user. Its installed
invocation is:

```sh
/usr/libexec/soda/soda-identity
```

Systemd supplies the separate administration and execution listeners through
`soda-identity.socket` and `soda-identity-runtime.socket`. Inspect an existing
installation with:

```sh
systemctl status soda-identity.service soda-identity.socket soda-identity-runtime.socket
journalctl -u soda-identity.service
```

The administration socket defaults in the appliance units to
`/run/soda/identity/admin.sock`, accessible to the Soda service group. The
runtime socket is `/run/soda/identity/runtime.sock`, owner-only. Neither socket
belongs inside a project.

## Configuration and prerequisites

The binary reads `/etc/soda/identity.json`, or an explicit file selected with
`--config PATH`. Provision configuration, the broker PostgreSQL database,
restricted database connection file and encryption key before starting it.
The host helper socket must be available for execution attestation and retirement.

The five service paths must be absolute, and the two broker sockets must differ.
The key file holds a base64-encoded 32-byte key in a restricted regular file.
Enabled providers require operator-selected executable bytes, version, digest
and private tmpfs enrollment roots. A provider with an empty `binary` is disabled.
See [Service configuration](../../docs/reference/configuration.md#identity-broker-fields)
for exact JSON fields and admission rules.

## Runtime behavior

The broker stores encrypted connections, grants, leases and execution records in
PostgreSQL, manages native device enrollment, and delegates execution through
the host helper. It reconciles recorded executions at startup and periodically
checks retirement. Unconfirmed termination keeps affected custody unavailable
for further use.

When launched outside socket activation, it binds the configured Unix sockets
and refuses occupied paths. Use the installed service to retain socket identity
across restarts. SIGTERM and SIGINT drain admitted work before normal shutdown;
startup or listener failure reports to standard error and exits `1`.

The `soda_identity` library backs this service; end users do not need to integrate
it directly. Installation and live-state maintenance follow
[Installation](../../docs/guides/installation.md) and
[Credential maintenance](../../docs/reference/credentials.md#existing-install-maintenance).
