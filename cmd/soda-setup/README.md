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
| `--factory-enrollment-file PATH` | Apply a private, versioned broker and factory callback manifest. |
| `--maintenance` | Explicitly replace an existing identity configuration during enrollment. |
| `--help` | Show command usage. |

## Outputs and next steps

Full setup records the Forgejo operator ID, writes the dashboard configuration,
and creates or reuses PostgreSQL credentials under `/etc/soda/postgres`. The
identity broker's encryption key and configuration are separately provisioned;
setup never creates, changes or removes that key.

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

## Explicit factory enrollment

Factory enrollment is a separate operator action after browser setup. It reads a
private `soda-factory-enrollment/v1` JSON manifest supplied with
`--factory-enrollment-file`; the manifest contains paths and selected pins, not
secret bytes. It must identify the exact existing dashboard config, operator ID,
and Forgejo origins, and carries strict identity-broker settings plus existing
callback and credential-file paths. Referenced files and provider binary digests
are checked before configuration writes. Existing identity configuration
refuses replacement unless `--maintenance` is explicit.

On success, setup writes `/etc/soda/identity.json`, updates the dashboard's
background/review/merge and intake paths, enables both broker sockets, starts or
restarts the broker, restarts the dashboard, and reconciles the exact native
operation tuples selected in the manifest. It first checks the Soda extension's
stable installation UUID and current binding inventory, invokes the native
`extensions bind` command for each selected tuple (idempotently), then rereads
the inventory and reports which tuples were already present or added. The CLI
validates that each token exists and belongs to the selected actor; operation
scopes and current account/repository permissions are still checked by native
operation admission. PAT contents never enter the manifest or process argv.
Failure restores the prior local configuration and attempts service recovery
based on whether identity configuration existed before enrollment. Native bindings
already added by a partial native command remain and are reported for inspection.

The manifest also selects the root-only `/etc/soda/enrollment/native-hooks.pat`
and exact repository IDs/owner names for the issue intake hook. Setup verifies
each repository ID using the authenticated Forgejo API, inventories its hooks,
and creates only an absent `forgejo` JSON hook at
`http://host.containers.internal:8080/api/factory/intake` with the `issues`
event and selected intake secret. It rereads the nonsecret hook fields after
creation. A root-owned `/etc/soda/enrollment/hooks.json` receipt records only
the selected Forgejo target, repository IDs/names and hook IDs; it contains no
secret. An existing destination is accepted for maintenance only when its ID,
repository and public configuration match that receipt. Setup then sends the
selected secret through Forgejo's authenticated PATCH request and confirms the
public fields again. Forgejo omits webhook secrets from reads, so the successful
PATCH response is the evidence that the selected secret was applied; the GET
readback does not reveal or verify secret bytes. An incomplete multi-repository
enrollment can resume with `--maintenance` only when this receipt matches the
same target and repositories. Unowned or conflicting hooks are preserved and
refused. The hook administration PAT is not mounted into the dashboard.

The background PAT must carry the selected issue and repository read scopes for
mixed native evidence, plus the operation-specific scope required by the chosen
binding. Reviewer and merger credentials remain separate files. Install a
repository-scoped issue-open hook before issue creation to retain recovery facts.
Only native events still retained by Fountain can be recovered; retention is
bounded to seven days.

The manifest is strict: unknown fields are rejected. The dashboard factory
inputs use the installed fixed mount paths shown here.
Create publisher/reviewer/merger and intake files as `soda` (UID/GID 2000), mode
0600, beneath the private `/etc/soda/factory` directory; the publication
directory is also owned by `soda` and mode 0700. The callback is the dashboard's
`/ipc/host.sock` mount, backed by the Forgejo socket owned by UID 1000. Broker
DSN, encryption key and provider files must be private and readable by
`soda-identity` (UID/GID 2001:2000). Use the operator ID and Forgejo origins
from the current dashboard configuration. Replacements preserve existing
configuration ownership and mode; a new broker file is installed as
2001:2000 mode 0600. The selected binding tuples use the actual token IDs and
actor IDs from the existing native Forgejo records; no token values are copied
into this manifest.

```json
{
  "format": "soda-factory-enrollment/v1",
  "bindings": {
    "installation_id": "550e8400-e29b-41d4-a716-446655440000",
    "operations": [
      {"token_id": 101, "actor_id": 42, "repository_id": 7, "kind": "git.ref.publish"},
      {"token_id": 102, "actor_id": 42, "repository_id": 7, "kind": "pull_request.create"},
      {"token_id": 103, "actor_id": 43, "repository_id": 7, "kind": "pull_request.review.submit"},
      {"token_id": 104, "actor_id": 44, "repository_id": 7, "kind": "pull_request.merge"}
    ]
  },
  "hooks": {
    "token_file": "/etc/soda/enrollment/native-hooks.pat",
    "repositories": [
      {"repository_id": 7, "owner": "operator", "name": "demo"}
    ]
  },
  "target": {
    "dashboard_config": "/etc/soda/dashboard.json",
    "identity_config": "/etc/soda/identity.json",
    "operator_id": 42,
    "forgejo_url": "https://forgejo.example.test",
    "forgejo_internal_url": "http://127.0.0.1:3000"
  },
  "dashboard": {
    "forgejo_background_socket": "/ipc/host.sock",
    "forgejo_background_host_uid": 1000,
    "forgejo_background_credential_file": "/etc/soda/factory/publisher.pat",
    "forgejo_review_credential_file": "/etc/soda/factory/reviewer.pat",
    "forgejo_merge_credential_file": "/etc/soda/factory/merger.pat",
    "factory_intake_secret_file": "/etc/soda/factory/intake.secret",
    "factory_publication_root": "/var/lib/soda/dashboard/publication"
  },
  "identity": {
    "database_dsn_file": "/etc/soda/postgres/soda.dsn",
    "key_file": "/etc/soda/identity/key",
    "admin_socket": "/run/soda/identity/admin.sock",
    "runtime_socket": "/run/soda/identity/runtime.sock",
    "host_socket": "/run/soda/host.sock",
    "codex": {"binary": "", "version": "", "sha256": "", "root": ""},
    "muse": {"binary": "", "version": "", "sha256": "", "root": ""}
  }
}
```

Enabled Codex and Muse entries require complete pins for the supported versions
(`0.153.4` and `1.4.0-R4161.1`, respectively), the exact executable digest and
a private provider root. An all-empty provider entry means disabled.
