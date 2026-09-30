# Soda service configuration

Configuration for the Soda extension's private Go service and native identity
broker. Source of truth:
`internal/config` and the installed JSON/unit files under the appliance.

Forgejo supplies browser identity and extension authority. The OAuth application
configured here is used by Soda's account-linking broker, whose callback returns
through Forgejo; it does not create a Soda browser session. Bounded agent work has
separate protected [factory operator configuration](factory.md); do not add
factory credentials or run authority to a human project configuration.

## Fields

| Field | Meaning |
| --- | --- |
| `listen` | HTTP listen address for the dashboard process |
| `forgejo_url` | Public Forgejo HTTPS origin used for native callback validation and identity enrollment |
| `forgejo_internal_url` | Internal Forgejo URL used by the identity broker |
| `database` | Path to Soda SQLite database |
| `host_socket` | Unix socket path for the privileged host helper |
| `oauth_client_id` | Forgejo OAuth application ID used for identity enrollment |
| `oauth_secret_file` | Path to the identity broker's OAuth client secret |
| `grant_key_file` | Path to the key that encrypts retained identity credentials in the Soda database |
| `operator_id` | Stable Forgejo user ID recorded at setup |
| `admin_token_file` | Optional; not retained as ordinary runtime auth |

Paths and secret files are operator-protected. Never commit secret contents or pass
them on argv in shared logs.

## Related

- Setup procedure: [Operator setup](../guides/operator-setup.md)
- Credential maintenance: [Credentials](credentials.md)
- Platform installed-path consts: `internal/platform`
