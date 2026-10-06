# Credentials and grants

Soda keeps provider credential-encryption keys separate from Forgejo's native
credential store. Product identity comes from the verified
Forgejo extension request; Soda does not issue browser login sessions. This
document owns durable credential contracts and operator maintenance. Route
behavior: [HTTP API](api.md).

Factory execution, publication actors and provider account authentication are
distinct from these credentials. Their authority belongs to
[Trust](../architecture/trust.md#factory-authority-boundary); delivery and
CLI execution belongs to the [factory reference](factory.md). The Identity Broker
below owns provider custody and delegation. Supervised factory runs in a shared
Project satisfy the [execution boundary](../architecture/trust.md#project-execution-boundary):
credential return captures only the recorded run boundary after its descendants
retire; whole-container capture is removed.

## New installations

1. Operator creates a Forgejo site-admin access token with `read:user` and supplies it through a mode-0600 file.
2. `soda-setup` calls the native API, records `operator_id`, retains broker
   encryption material in restricted files, and revokes the bootstrap token
   once its configuration is written. It does not keep a bootstrap-token
   copy. A failed setup keeps the token so the operator can retry with it.
3. Setup refuses to overwrite existing configuration.
4. `soda-activate` applies bind address and TLS, then starts the dashboard/proxy.

Developers use native Forgejo account creation. Soda has no account/password
administration frontend.

## Operator identity

`operator_id` is the stable Forgejo user ID recorded at setup. Only that identity
may use protected operator runner and Tailnet settings APIs. Forgejo site-admin
status alone is not sufficient.

## Credential encryption

Broker-held provider credentials are encrypted with the configured key file.
Protect encryption-key files as operator secrets. Never
expose them in source, argv, logs, screenshots or evidence.

## Existing-install maintenance

Rerunning setup/activation is not the maintenance path for existing installs.
Credential rotation, bootstrap-token retirement and broker schema updates are
explicit operator procedures against the live data volume. Revocation and deletion
are separate explicit actions, never automatic migrations.

## Source owners

- Config load: `internal/config`
- Native browser authority: `internal/web/auth`
- Provider credentials and grants: `internal/identity`, `internal/store`
- Setup/activate scripts and units under `system/` and `scripts/`

## Identity Broker

### Authentication and billing choice

The selected provider, account and native authentication method are the connection
contract. Soda-managed AI execution uses subscription access only. Authentication
failure, credential expiry or subscription limits must stop the affected execution
and explain the required user action. Soda must never automatically switch to
usage billing, another account, another provider or another authentication method.
Changing the selection requires an explicit user choice.

Credential format alone does not establish billing: a subscription credential can
contain a key, and OAuth can authorize usage-billed access. Each integration must
verify its native subscription route and prevent environment, configuration and
upstream overage settings from silently selecting usage billing. A route whose
subscription-only behavior cannot be established is not admitted by the broker.

### Current providers

The userspace `soda-identity` service keeps Codex ChatGPT and Muse Code subscription
connections private to their Soda owner. Connections, enrollments and leases carry
an explicit `provider_id` (`codex` or `muse`). Both use subscription device sign-in. Connect in
project controls and complete the selected provider’s native sign-in.
Choose a connection when starting Codex. Device sign-in
must be enabled by the upstream account or organization; unsupported enrollment
requires upstream setup, never silent conversion to API billing. The verified
protocol is Codex CLI `0.153.4`; operator configuration pins actual executable bytes.
Claude.ai subscription credentials are not admitted into broker custody:
[Anthropic's credential rules](https://code.claude.com/docs/en/legal-and-compliance#authentication-and-credential-use)
prohibit third-party collection, storage and intermediation of these session
tokens. An end user may sign into the unmodified Claude Code binary through its
native flow; that is separate from Soda's shared credential custody. An explicit
upstream agreement covering broker custody is required before adding that route.
OpenRouter's documented [usage billing](https://openrouter.ai/support/#how-do-i-get-billed-for-my-usage-on-openrouter)
does not satisfy subscription-only admission. API billing and federation adapters
are not supported.

A project grant names one Soda user and project. Its owner must confirm provider
permission to share and accept credential exposure. Current provisioned membership
and repository execution authority are checked by native extension admission. Requester,
execution, sponsoring owner and grant stay separate. Repository contents and
browser-supplied execution IDs cannot expand authority.

This is controlled credential exposure: authorized code can read and copy the
upstream account credential. Project root/wheel and host administrators are trusted.
Other processes under the same project login can also read that login's staged
credentials; human-session termination does not terminate those unrelated processes.
The provider sees the connected account, not a separately scoped Soda execution.
Soda cannot invalidate copies or recall already accepted upstream requests; provider
logout/revocation is separate. This mode does not restrict where repository data
can be sent. Git read/publication/review permissions remain independent.

Each Codex connection has one active stream across enrollment, human and factory use.
A human start reports unavailable while busy. Factory runs acquire by execution
identity and bind the exact run, container, unit and systemd incarnation. Human
sessions bind to the exact project OCI incarnation, terminal, lease and systemd
invocation. The broker delivers credentials only after native attestation of that
exact binding. Runs receive no broker or host socket. A copied ID alone is not
an admission credential.

Browser detach leaves the managed Codex session and lease running. Normal End or
the maximum twelve-hour human deadline freezes the complete tool cgroup, captures
updated auth and terminates descendants before custody is returned. Removing a
project grant or disconnecting first denies further access, then terminates its
registered execution. A failed termination keeps the connection blocked. Crash,
reboot or interrupted return never restores the original seed; reconciliation
terminates the recorded boundary and uncertain streams require one Soda-level
reconnection. Local project files survive authentication failures.

### Muse subscription custody

Muse uses the pinned upstream CLI `1.4.0-R4161.1` and its native device login.
Meta attaches subscription billing to the CLI credential produced during onboarding;
Soda never creates an additional API key. The native OAuth record includes the CLI
key as well as an access token. A manually supplied key is not accepted as Muse
subscription enrollment.

Muse permits concurrent independent leases. Runtime credentials are immutable,
restricted, and read-only; completing one execution does not replace the encrypted
connection or end another execution. Disconnect denies admission before retiring all
leases; withdrawing a grant retires only its executions. Unconfirmed termination
blocks admission. Restart reconciliation must confirm retirement before reuse.

Concurrent and fresh-container credential reuse are supported by native evidence
for this pinned release. Automatic token refresh is unverified. If Muse rejects
its credential, the connection requires device reconnection; Soda does not provide
an undocumented Meta refresh service or switch to pay-as-you-go billing. The native
file backend uses `TBH_CREDENTIAL_BACKEND=file`; it is verified for the selected
bytes, rather than assumed to be a stable public interface across releases.

### Normal Muse command

After one Muse enrollment, run `muse` or `muse exec` from the provisioned account
in an authorized project. The launcher derives identity from kernel peer evidence
and the current account marker; caller arguments do not select an actor or project.
When several authorized Muse connections exist, set `SODA_MUSE_CONNECTION` to the
chosen connection ID. Settings and working directories belong to each invocation;
subscription execution removes inherited API-key overrides. Existing personal
settings seed the private view and remain unchanged. CLI settings or trust edits
in that view are discarded at retirement; edit the personal settings file for
persistent preferences.

Each execution has a separate systemd unit and broker lease. The only container
interface is the launch socket at `/run/soda-muse-interface/launch.sock`.
Mount its dedicated public directory read-only, so socket replacement after service
restart remains visible. Never place broker administration, credential-delivery
sockets or credentials in that directory. The broker must confirm termination
before deleting runtime credentials. A rejected upstream credential requires
reconnection through Soda; generic command failures are not proof of authentication
failure.

### Service configuration

The broker runs as the dedicated `soda-identity` operator with its private
state. Factory execution and retirement go through the host daemon over its
fixed process contract; the broker never shells out to container runtimes.
The dashboard stays separate. `soda-identity.socket` admits administration at
`/run/soda/identity/admin.sock` for the Soda service group; the execution socket
`/run/soda/identity/runtime.sock` is owner-only (root host also has access).
Systemd preserves the socket identity across service restarts and kills the full
broker service cgroup, including enrollment children. Neither socket enters a
project.

`/etc/soda/identity.json` selects these paths, the host socket, a separate broker
PostgreSQL `database`, private base64 32-byte `key_file`, and `codex` fields `binary`,
`version`, `sha256`, `root`. Store broker state and private key under
`/home/soda-identity`; keep the key mode `0600` and outside dashboard mounts.
Both providers require a private tmpfs enrollment root. `codex.root` is a private tmpfs directory under `/run/soda-identity`. CLI credentials,
logs and enrollment state remain in that tmpfs. Completed credentials are opaque
AES-GCM ciphertext in the broker database, authenticated to the connection and
revision. The host key permits unattended startup: encryption protects a database
copy, not a compromised administrator or disk plus key. A wrong/missing key fails
startup instead of generating a replacement.

Host configuration selects `identity_socket` and a verified `codex_harness` release
package with `codex_harness_sha256`. The host stages the supplied package into a
managed session's private tmpfs. Supervised factory runs stage the proved harness
version selected by their run record with its actual executable digest; the only
proved factory harness is Codex CLI `0.157.1`. Provider credentials must not be
embedded in that package, images, persistent project homes or exported artifacts.

Provision configuration and a fresh key explicitly, then enable the broker sockets
and service. `soda-setup` does not invent a Codex account, choose an external package
or overwrite an existing broker key. Starting a native service or installing a
release remains subject to the normal installation guide.

Source owners: `identity` records, `identity/control` custody, provider-specific
protocol packages, `identity/client` private transport, `store` encrypted rows,
`web/api` native extension authority, and native terminal executors for attestation
and complete execution termination.

For project Muse execution, host configuration supplies `identity_socket`,
`muse_socket`, `muse_version` and `muse_sha256` from the pinned release manifest.
Muse factory runs stay unavailable until their own supervised-run proof passes.
