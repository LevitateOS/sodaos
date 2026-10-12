# Soda service configuration

Configuration for the Soda extension's private Go service and native identity
broker. Source of truth:
`internal/config`, `cmd/soda-identity/src/main.rs` and the installed JSON/unit
files under the appliance.

Forgejo supplies browser identity and extension authority. Bounded agent work has
separate protected [factory operator configuration](factory.md); do not add
factory credentials or run authority to a human project configuration.

## Dashboard fields

| Field | Meaning |
| --- | --- |
| `listen` | HTTP listen address for the dashboard process |
| `forgejo_url` | Public Forgejo HTTPS origin |
| `forgejo_internal_url` | Internal Forgejo API URL |
| `database_dsn_file` | Restricted secret file holding the PostgreSQL connection URL |
| `host_socket` | Unix socket path for the privileged host helper |
| `identity_socket` | Private identity broker administration socket |
| `operator_id` | Stable Forgejo user ID recorded at setup |
| `forgejo_graph_credential_file`, `forgejo_graph_actor_id` | Paired absolute restricted secret-file path and separately enrolled native service actor for bounded issue graph operations. Requires the existing background service admission inputs. Current human intent-owner code-write permission remains independently checked. The native token must grant `read:user`, `write:repository` and `read:issue`: submission and owner proofs need the first two, adoption snapshot reads need `read:issue`, and snapshot reads ride only enrolled token bindings. |
| `typesafe_api_key_file` | Absolute server-only key file, regular and accessible only to its owner; never delivered to Projects or browser inputs. |
| `typesafe_model` | System One classifier model; defaults to `jev-latest`. |
| `factory_judgment_call_budget` | Finite cumulative consultation ceiling, between zero and 100,000 calls. Zero disables spending. Reserved failed or interrupted calls remain spent; changing this ceiling does not erase durable usage. Each intent has its own additional finite allowance. |

Paths and secret files are operator-protected. Never commit secret contents or pass
them on argv in shared logs.

The identity broker's credential-encryption key is provisioned independently
and is not a dashboard configuration field.

## Identity broker fields

`soda-identity --config PATH` reads `/etc/soda/identity.json` by default. The
configuration is one UTF-8 JSON object, at most 1 MiB, with exact field names.
Duplicate keys, unknown fields, trailing JSON and excessive nesting are refused;
the nesting limit is 100 below the root. Case variants are not aliases.

| Field | Meaning |
| --- | --- |
| `database_dsn_file` | Restricted file holding the broker's PostgreSQL connection URL |
| `key_file` | Restricted file holding the broker's base64-encoded 32-byte encryption key |
| `admin_socket` | Broker administration Unix socket path |
| `runtime_socket` | Broker execution Unix socket path |
| `host_socket` | Privileged host helper Unix socket path |
| `codex` | Codex provider settings |
| `muse` | Muse provider settings |

The five service paths must be explicit absolute paths. Administration and
execution sockets must differ. Each provider uses these fields:

| Field | Meaning |
| --- | --- |
| `binary` | Selected provider executable |
| `version` | Selected executable version |
| `sha256` | Selected executable digest |
| `root` | Private provider enrollment root |

An omitted provider or omitted provider string defaults to an empty value; a
provider with an empty `binary` is disabled. Explicit `null` does not substitute
for a string or provider settings. Unknown provider fields are refused. Provider
authority, root custody and key permissions are described in
[Credentials](credentials.md#service-configuration).

## Related

- Setup procedure: [Operator setup](../guides/operator-setup.md)
- Credential maintenance: [Credentials](credentials.md)
- Platform installed-path consts: `internal/platform`
