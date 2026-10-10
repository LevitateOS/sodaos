# soda-setup

`soda-setup` bootstraps the appliance's native operator binding and dashboard
configuration. It is a root-only operator command installed at
`/usr/bin/soda-setup`. It also generates database credentials during first boot.

## Before starting

Complete Forgejo's native installation and create its site administrator using
the private loopback access described in
[operator setup](../../docs/guides/operator-setup.md). Create a token belonging
to that administrator with `read:user`, and save it in a mode-0600 file.
Setup contacts Forgejo's `/api/v1/user`; it needs a reachable native origin.

Choose the final private HTTPS browser origin, with no credentials, application
path, query or fragment. Run through native root access:

```sh
sudo /usr/bin/soda-setup \
  --forgejo-url https://forgejo.example.test \
  --token-file /home/operator/forgejo-token
```

The example hostname represents your appliance's actual private origin; the
token stays in its restricted input file. The interactive
[soda-install configure](../soda-install/README.md) flow can guide this process.

## Options

| Option | Meaning |
| --- | --- |
| `--forgejo-url ORIGIN` | Required HTTPS browser origin for full setup. |
| `--forgejo-internal-url ORIGIN` | Native Forgejo HTTP(S) origin; defaults to `http://127.0.0.1:3000`. |
| `--token-file PATH` | Required operator token file for full setup. |
| `--out PATH` | New absolute dashboard configuration path; defaults to `/etc/soda/dashboard.json`. |
| `--provision-db-only` | Generate or reuse database credentials without binding an operator. |
| `--help` | Show command usage. |

## Outputs and next steps

Full setup records the Forgejo operator ID, writes the dashboard configuration,
creates `grant-key` beside it, and creates or reuses the PostgreSQL credentials
under `/etc/soda/postgres`. Existing configuration and encryption keys are not
overwritten. The new configuration and key are private files.

After writing configuration, setup revokes the bootstrap token. If revocation
cannot be confirmed, it reports that configuration was written and tells you to
inspect and revoke the token in Forgejo before continuing activation. Earlier
setup failures retain the bootstrap token for retry.

The first-boot `soda-pg-provision.service` uses:

```sh
/usr/bin/soda-setup --provision-db-only
```

This creates `super.passwd`, `forgejo.passwd`, `soda.passwd` and `soda.dsn`, or
reuses an existing complete matching set. It does not initialize roles or bind
an operator. Partial or conflicting existing credentials require inspection.

Successful commands return `0`; execution failures return `1`, and invalid
flags return `2`. After full setup, follow
[operator setup](../../docs/guides/operator-setup.md) for service ownership,
explicit activation and client TLS trust. Existing installations use
[credential maintenance](../../docs/reference/credentials.md) instead of rerunning setup.
