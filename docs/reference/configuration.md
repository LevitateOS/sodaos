# Soda service configuration

Configuration for the Soda extension's private Go service and native identity
broker. Source of truth:
`internal/config` and the installed JSON/unit files under the appliance.

Forgejo supplies browser identity and extension authority. Bounded agent work has
separate protected [factory operator configuration](factory.md); do not add
factory credentials or run authority to a human project configuration.

## Fields

| Field | Meaning |
| --- | --- |
| `listen` | HTTP listen address for the dashboard process |
| `forgejo_url` | Public Forgejo HTTPS origin |
| `forgejo_internal_url` | Internal Forgejo API URL |
| `database_dsn_file` | Restricted secret file holding the PostgreSQL connection URL |
| `host_socket` | Unix socket path for the privileged host helper |
| `identity_socket` | Private identity broker administration socket |
| `grant_key_file` | Path to the key that encrypts retained identity credentials in the Soda database |
| `operator_id` | Stable Forgejo user ID recorded at setup |

Paths and secret files are operator-protected. Never commit secret contents or pass
them on argv in shared logs.

## Related

- Setup procedure: [Operator setup](../guides/operator-setup.md)
- Credential maintenance: [Credentials](credentials.md)
- Platform installed-path consts: `internal/platform`
