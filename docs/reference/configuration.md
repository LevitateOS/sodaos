# Dashboard configuration

Configuration for the Soda Go API/OAuth service. Source of truth:
`internal/config` and the installed JSON/unit files under the appliance.

## Fields

| Field | Meaning |
| --- | --- |
| `listen` | HTTP listen address for the dashboard process |
| `forgejo_url` | Browser-facing Forgejo HTTPS origin |
| `forgejo_internal_url` | Internal Forgejo URL for server-side API calls |
| `database` | Path to Soda SQLite database |
| `host_socket` | Unix socket path for the privileged host helper |
| `oauth_client_id` | Forgejo OAuth client ID |
| `oauth_secret_file` | Path to OAuth client secret file |
| `grant_key_file` | Path to grant-encryption key file |
| `operator_id` | Stable Forgejo user ID recorded at setup |
| `admin_token_file` | Optional; not retained as ordinary runtime auth |

Paths and secret files are operator-protected. Never commit secret contents or pass
them on argv in shared logs.

## Related

- Setup procedure: [Operator setup](../guides/operator-setup.md)
- Credential maintenance: [Credentials](credentials.md)
- Platform installed-path consts: `internal/platform`
