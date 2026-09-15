# Credentials and grants

Soda keeps OAuth client secrets, grant-encryption keys and adapter sessions separate
from Forgejo's native credential store. This document owns durable credential
contracts and operator maintenance. Route behavior: [HTTP API](api.md).

## New installations

1. Operator creates a Forgejo site-admin access token with `write:user` (includes
   `read:user`) and supplies it through a mode-0600 file.
2. `soda-setup` calls the native API, records `operator_id`, creates the OAuth client
   and retains the OAuth secret plus grant-encryption key. It does not keep a
   bootstrap-token copy.
3. Setup refuses to overwrite existing configuration.
4. `soda-activate` applies bind address and TLS, then starts the dashboard/proxy.

Developers use native Forgejo account creation. Soda has no account/password
administration frontend.

## Operator identity

`operator_id` is the stable Forgejo user ID recorded at setup. Only that identity
may use protected operator runner and Tailnet settings APIs. Forgejo site-admin
status alone is not sufficient.

## Grant encryption

Session-bound grants are encrypted with the configured grant key file. Protect
grant-key and OAuth-secret files as operator secrets. Never expose them in source,
argv, logs, screenshots or evidence.

## Existing-install maintenance

Rerunning setup/activation is not the maintenance path for existing installs.
Credential rotation, bootstrap-token retirement and schema-compatible grant updates
are explicit operator procedures against the live data volume. Revocation and
deletion are separate explicit actions, never automatic migrations.

When schema or return-destination fields change, preserve existing sessions where
the migration defines compatibility; otherwise require reauthentication. Do not
invent silent grant rewriting as repair.

## Native consent

Soda verifies required OAuth scopes after exchange (including `read:user`,
`read:repository` and `read:organization` where those surfaces need them). Failed
named transactions return with bounded UI markers, not arbitrary return URLs.

## Source owners

- Config load: `internal/config`
- Sessions and grants: `internal/store`, `internal/web/auth`
- Setup/activate scripts and units under `appliance/` and `scripts/`
