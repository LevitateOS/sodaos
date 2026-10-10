# Dashboard configuration reader

Soda service startup uses this package to load operator JSON and separately
read configured credentials. It validates inputs without writing configuration,
generating secrets or starting a service.

## Use inside the Soda module

Import `github.com/levitateos/sodaos/internal/config` and call
`config.Load(path)` to obtain a validated `Config`. The normal installed file
is `/etc/soda/dashboard.json`, provisioned by
[`soda-setup`](../../cmd/soda-setup/README.md).

The [public entry points](config.go) are:

- `Load`: decode one bounded configuration and validate listener, origins and
  paths; it does not read the credentials referenced by those paths.
- `BaseURL`: admit an HTTP(S) origin without user credentials, application path,
  query or fragment.
- `Secret`: read one nonempty restricted credential file, bounded to 64 KiB.
- `GrantKey`: read a separately provisioned base64 key encoding exactly 32 bytes.
- `Config.BackgroundServiceConfigured`: report whether any native background
  admission input is set; full validation still requires the complete set.

## Inputs and limits

Configuration is bounded to 64 KiB, refuses unknown fields and trailing JSON,
requires a loopback listener behind the private HTTPS proxy, and requires HTTPS
for the browser origin. Empty listener and identity-socket values receive the
defaults in source. Credential and runtime paths are validated separately from
the contents they reference.

Keep database credentials, provider tokens and grant-key bytes in the configured
restricted files. `Secret` trims outer whitespace and rejects embedded newline
or NUL bytes. `GrantKey` requires a restricted regular file; never generate a
replacement merely because startup cannot read it, since persisted grants use
the existing key. Returned secrets must stay out of logs and browser responses.

The [configuration reference](../../docs/reference/configuration.md) owns the
operator fields and paths. Follow [operator setup](../../docs/guides/operator-setup.md)
for provisioning and [credentials](../../docs/reference/credentials.md) for
credential maintenance.
