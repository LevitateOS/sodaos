# Local test host and retained access

Access paths for the local development fixture host. This is operator/developer
convenience documentation for retained lab state, not product acceptance and not a
permission grant.

Destructive cleanup, provider registration and fixture replacement need explicit
task approval. See [AGENTS.md](../../AGENTS.md#permissions-and-preservation).

## Typical access patterns

- SSH to the fixture host as configured for the lab account.
- Forgejo and Spaces through the configured private HTTPS origin with the lab CA
  trusted in the browser.
- Cockpit through loopback forwarding only.
- Project SSH as `user@project-ip` after Join.

Exact hostnames, addresses and ports change with the fixture; record them in the
task or local notes rather than treating this file as a live inventory.

## Frontend development against the fixture

Spaces frontend work can target the fixture's Forgejo origin with trusted local
TLS. Prefer the repository's documented Bun workspace commands in
[TypeScript](../development/typescript.md).

## Related

- [Installation](installation.md)
- [Testing](../development/testing.md)
- [Screenshot capture](../design/screenshot-capture.md)
