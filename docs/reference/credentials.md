# Credentials and grants

Soda keeps OAuth client secrets, grant-encryption keys and adapter sessions separate
from Forgejo's native credential store. This document owns durable credential
contracts and operator maintenance. Route behavior: [HTTP API](api.md).

Factory execution, publication actors and provider account authentication are
distinct from these browser grants. Their authority belongs to
[Trust](../architecture/trust.md#factory-authority-boundary); delivery and
CLI execution belongs to the [factory reference](factory.md). The Identity Broker
below owns provider custody and delegation.

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

## Identity Broker

The userspace `soda-identity` service keeps Codex ChatGPT subscription connections
private to their Soda owner. Connect in project controls, complete OpenAI device
sign-in, and explicitly choose a connection when starting Codex. Device sign-in
must be enabled by the upstream account or organization; unsupported enrollment
requires upstream setup, never silent conversion to API billing. The verified
protocol is Codex CLI `0.153.4`; operator configuration pins actual executable bytes.
Claude subscription tokens, API billing and federation adapters are outside v1.

A project grant names one Soda user and project. Its owner must confirm provider
permission to share and accept credential exposure. Current provisioned membership
and repository execution authority are checked by browser admission. Requester,
execution, sponsoring owner and grant stay separate. Repository contents and
browser-supplied execution IDs cannot expand authority.

This is controlled credential exposure: authorized code can read and copy the
upstream account credential. Project root/wheel and host administrators are trusted.
Other processes under the same project login can also read that login's staged
credentials; human-session termination does not terminate those unrelated processes.
The provider sees the connected account, not a separately scoped Soda worker.
Soda cannot invalidate copies or recall already accepted upstream requests; provider
logout/revocation is separate. This mode does not restrict where repository data
can be sent. Git read/publication/review permissions remain independent.

Each connection has one active stream across enrollment, human and factory use.
Factory waits within its existing deadline; a human start reports unavailable
while busy. Native OCI identity and labels attest factory runs. Human sessions bind
to the exact project OCI incarnation, terminal, lease and systemd invocation. The
broker delivers credentials only after native attestation. Workspaces receive no
broker or host socket. A copied ID alone is not an admission credential.

Browser detach leaves the managed Codex session and lease running. Normal End or
the maximum twelve-hour human deadline freezes the complete tool cgroup, captures
updated auth and terminates descendants before custody is returned. Removing a
project grant or disconnecting first denies further access, then terminates its
registered execution. A failed termination keeps the connection blocked. Crash,
reboot or interrupted return never restores the original seed; reconciliation
terminates the recorded boundary and uncertain streams require one Soda-level
reconnection. Local project files survive authentication failures.

### Service configuration

Broker and factory share the dedicated `soda-identity` operator, its rootless Podman
storage and runtime context. Configure subordinate UID/GID ranges and rootless
Podman prerequisites before use, as for the existing factory. The dashboard stays
separate. `soda-identity.socket` admits administration at
`/run/soda/identity/admin.sock` for the Soda service group; the execution socket
`/run/soda/identity/runtime.sock` is owner-only (root host also has access).
Systemd preserves the socket identity across service restarts and kills the full
broker service cgroup, including enrollment children. Neither socket enters a
project or factory workspace.

`/etc/soda/identity.json` selects these paths, the host socket, a separate broker
SQLite `database`, private base64 32-byte `key_file`, and `codex` fields `binary`,
`version`, `sha256`, `root`. Store broker state and private key under
`/home/soda-identity`; keep the key mode `0600` and outside dashboard mounts.
`codex.root` is a private tmpfs directory under `/run/soda-identity`. CLI credentials,
logs and enrollment state remain in that tmpfs. Completed credentials are opaque
AES-GCM ciphertext in the broker database, authenticated to the connection and
revision. The host key permits unattended startup: encryption protects a database
copy, not a compromised administrator or disk plus key. A wrong/missing key fails
startup instead of generating a replacement.

Host configuration selects `identity_socket` and a verified `codex_harness` release
package with `codex_harness_sha256`. The host stages the supplied package into a
managed session's private tmpfs. Factory selects the same tested CLI version and
its actual architecture-specific digest. Provider credentials must not be embedded
in that package, images, persistent project homes or exported artifacts.

Provision configuration and a fresh key explicitly, then enable the broker sockets
and service. `soda-setup` does not invent a Codex account, choose an external package
or overwrite an existing broker key. Starting a native service or installing a
release remains subject to the normal installation guide.

Source owners: `identity` records, `identity/control` custody, `identity/codex`
provider protocol, `identity/client` private transport, `store` encrypted rows,
`web/api` browser authority, and native workspace/terminal executors for attestation
and complete execution termination.
