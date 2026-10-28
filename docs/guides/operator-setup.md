# Operator bootstrap

Prepare Forgejo identity and private access for software collaboration and bounded
agent work. Factory execution and persistent human Projects have separate authority.

Soda uses Fountain’s native Forgejo extension authority. Setup reads the native
operator identity through `/api/v1/user`.

Media continuation: [Installation media](media.md).
Install/activate context: [Installation](installation.md).
Docker operators use [Configure private browser access](docker.md#configure-private-browser-access)
for the host SSH tunnel and runtime `docker compose exec` context.

## Guided browser setup

After the release-media disk installation, use a root SSH terminal or local
console and run `/usr/libexec/soda/soda-install configure`. Root password SSH is
enabled by that installation; an installed SSH key may be used instead, and
[key enrollment](../operator/enroll-key.md) is optional.

Select the assigned private address your client can reach. The console supplies
the SSH tunnel command for Forgejo's loopback installer, asks for the native
administrator's `read:user` token through hidden input and invokes setup plus
local HTTPS activation. It passes the token in a restricted temporary file; the
operator does not create a token file or supply a certificate by hand for this path.

Compare the displayed public CA fingerprint, copy only its public certificate
through verified SSH, and trust it using the client's normal certificate settings.
Then verify browser login and a joined Project's real browser terminal. Configure
a Project subnet route separately when direct Project SSH/services are needed.
Active units, a configured address or a displayed Project IP alone prove neither
client reachability nor completed login/execution.

An activated installation produces read-only access guidance on rerun; uncertain
setup/activation preserves state and must be inspected before retry. The
[public operator setup guide](../public/60-Install/60-operator-setup.md) follows this path.

## Deliberate setup with custom configuration

The lower-level commands below are an alternative for an operator selecting a
custom browser origin/certificate, not additional steps after guided configure.

1. Provision operator host access and start the Forgejo unit. Its web listener
   initially binds host loopback. Do not expose an unfinished installer through the
   public reverse proxy.
2. Use SSH forwarding (`ssh -L 33000:127.0.0.1:3000 root@HOST`) and open
   `http://localhost:33000` to complete Forgejo's native installation and create its
   administrator. Activation later applies the private HTTPS origin and native SSH
   clone port 2222.
3. Create an operator access token with `read:user`. Setup reads `/api/v1/user`.
   The token must belong to a Forgejo site administrator for Soda's bootstrap eligibility check.
   Supply it through a mode-0600 file; never expose it to developers.
4. Run `/usr/bin/soda-setup --forgejo-url https://FORGEJO --token-file /home/operator/forgejo-token`.
   Setup records `operator_id` and writes the dashboard configuration; it does
   not provision the identity broker or its encryption key. It refuses to
   overwrite existing configuration. On success it revokes the bootstrap
   token; a failed setup keeps the token so the operator can retry with it.
5. Run `/usr/bin/soda-activate` with the explicit private bind address and
   either `--local-tls` for that IP or an existing certificate/key. Local TLS requires
   explicit trust of the appliance's public root certificate on intended clients.

Both tools ship in `/usr/bin`, which is on the default root PATH; the absolute
paths above work even under restricted SSH environments.

Developers use native Forgejo account creation. Browser pages and product
operations use Forgejo's native extension service on the configured HTTPS origin.
The public `/-/soda/` route serves the avatar provider. Git authentication uses
native Forgejo SSH keys or HTTPS tokens, independently of Soda development-access keys.

Existing installations use [credential maintenance](../reference/credentials.md),
not rerunning setup. Console guidance: [Operator console welcome](../design/console-welcome.md).

For retained Projects reporting a terminal support mismatch, use the explicit
[same-root terminal maintenance procedure](../reference/project-os.md#terminal-support)
from the installed complete release. Signing in again or importing a Project image
does not repair retained support. Host activation remains subject to the separate
[host update ownership](../architecture/release.md#host-update-ownership).

The [public Factory authorization guide](../public/40-Configure/20-factory-authorization.md)
explains the separate owners of repository policy, appliance capacity and provider
sponsorship. The [Factory journey](../public/20-Factory/10-software-factory.md)
describes how eligible accepted issues proceed under standing authority. The
[operator reference](../reference/factory.md) documents the existing command
interface.

## Host updates

There is no separate release-maintenance command. Upgrade an appliance by
reinstalling from new media after a backup, or through the native Fedora CoreOS
image update. See [host update ownership](../architecture/release.md#host-update-ownership)
and [Updates and fallback](../public/50-Operate/70-updates-and-fallback.md).

## Explicit factory enrollment

### Prepare native accounts and credentials

Sign in to native Forgejo as a site administrator and use **Site Administration
→ User Accounts → Create User** to create dedicated, nonadministrator service
accounts for publication, review, merge and issue-graph work. Use distinct
accounts such as `factory-publisher`, `factory-reviewer`, `factory-merger` and
`factory-graph`. The publisher also creates pull requests. The reviewer also
resolves review threads, using a separate token for each operation.

As each target repository's owner, open **Settings → Collaborators**, add only
the selected service accounts and confirm **Write** access. Their native
repository permissions remain required even after operation binding. Keep the
graph account separate from the publication and review accounts.

Run the following preparation in a root shell inside the Soda runtime. On an
appliance, use host root; for Docker, first use `docker compose exec soda bash`
from the deployment directory. These commands use the installed native Forgejo
container and retain token output in private files. They prepare new inputs;
inspect and reuse known existing files during maintenance instead of generating
replacement tokens.

```sh
set +x
umask 077
enrollment_stage=$(mktemp -d /home/factory-enrollment.XXXXXX)
forgejo() {
  podman exec --user 1000 soda-forgejo /usr/local/bin/gitea \
    -c /data/gitea/conf/app.ini "$@"
}
install -d -m 0700 -o 2000 -g 2000 /etc/soda/factory
install -d -m 0700 -o 0 -g 0 /etc/soda/enrollment
for role in publisher reviewer resolver merger graph; do
  test ! -e "/etc/soda/factory/$role.pat" || exit 1
done
for role in publisher reviewer resolver merger graph; do
  account="factory-$role"
  test "$role" != resolver || account=factory-reviewer
  forgejo admin user generate-access-token --username "$account" \
    --token-name "soda-factory-$role" \
    --scopes read:user,read:issue,write:repository --raw \
    > "$enrollment_stage/$role.pat" || exit 1
  test -s "$enrollment_stage/$role.pat" || exit 1
  if test "$role" = graph; then
    install -m 0640 -o 0 -g 2000 "$enrollment_stage/$role.pat" \
      /etc/soda/factory/graph.pat
  else
    install -m 0600 -o 2000 -g 2000 "$enrollment_stage/$role.pat" \
      "/etc/soda/factory/$role.pat"
  fi
done
test ! -e /etc/soda/factory/intake.secret || exit 1
head -c 48 /dev/urandom | base64 > "$enrollment_stage/intake.secret"
install -m 0600 -o 2000 -g 2000 "$enrollment_stage/intake.secret" \
  /etc/soda/factory/intake.secret
install -d -m 0700 -o 2000 -g 2000 /var/lib/soda/dashboard/publication
```

The native token issuer selects all repository resources to permit `read:user`;
the service account's own repository permissions and exact native bindings
still limit operations. `read:user` permits identity confirmation, `read:issue`
permits issue evidence, and `write:repository` includes repository reads and the
selected repository operations. No administrator scope is needed for these
service accounts. A browser-created token needs the same scopes and **All
repositories** resource selection.

For hook enrollment, use a token belonging to the existing native repository
owner or administrator, with `read:user,write:repository`. Generate it through
the same native command with a distinct token name and redirect `--raw` output
to a private file. Install it as root:root, mode 0600, at
`/etc/soda/enrollment/native-hooks.pat`. This token administers the selected
repositories' hooks and is never mounted into the dashboard.

Record actor and token IDs without reading token hashes or values:

```sh
forgejo admin user list
podman exec soda-postgres psql -U postgres -d forgejo -c '
  SELECT t.id AS token_id, t.uid AS actor_id, u.name AS account,
         u.is_admin, t.name AS token_name, t.scope
  FROM access_token t JOIN "user" u ON u.id = t.uid
  WHERE u.name IN ('\''factory-publisher'\'', '\''factory-reviewer'\'',
                   '\''factory-merger'\'', '\''factory-graph'\'')
  ORDER BY t.id;'
forgejo extensions installation soda
```

Use the returned stable installation UUID and the actual numeric repository,
actor and token IDs in the manifest. The token names distinguish review and
thread-resolution authority on the same review account. Enrollment confirms
the selected repository ID against its owner/name through the native API.

### Select native operations and enroll

Basic browser activation does not enable factory work. After choosing and
reviewing the provider pins, native actors, credentials and recovery inputs,
write a mode-0600 `soda-factory-enrollment/v1` manifest and run
`sudo /usr/bin/soda-setup --factory-enrollment-file /etc/soda/factory-enrollment.json`.
The manifest must bind to the current `/etc/soda/dashboard.json` operator ID and
Forgejo origins. It references restricted secret files; it never contains
credential bytes. If `/etc/soda/identity.json` already exists, preserve its
encryption key, credential database, provider roots and complete admitted
binary pins in the manifest, review the replacement and add `--maintenance`.
Pause dispatch and let active managed runs and credential leases retire before
maintenance: the command restarts the broker and dashboard. It validates inputs before
writing, enables the two broker sockets, starts/restarts the broker, and restarts
the dashboard. Check the resulting service state before enabling factory policy.

Factory workers run only on the Muse harness. A stored repository policy that
still names `codex` for a role is refused at dispatch; open the repository's
Factory settings and save the policy again to select Muse. The `codex_harness`
host configuration serves human project terminals only. Retain installed proof
for the selected Muse harness and model before enabling task writes. Pause
dispatch and retire managed runs and leases before changing the host
configuration or restarting `soda-host.service`.

The explicit `soda-setup --factory-enrollment-file` operation obtains the
stable Soda extension installation UUID through Forgejo's native CLI, applies
each owner-selected token/actor/repository tuple with `extensions bind`, and
checks the resulting IDs-only inventory with `extensions bindings`. The private
manifest contains token IDs, actor IDs, repository IDs and operation kinds;
PAT contents remain in their restricted files. To inspect or reconcile one
tuple manually, the native commands are:

```sh
forgejo extensions installation soda
forgejo extensions bind --installation INSTALLATION_UUID --token-id TOKEN_ID --actor-id ACTOR_ID --repository-id REPOSITORY_ID --kind git.ref.publish
forgejo extensions bind --installation INSTALLATION_UUID --token-id TOKEN_ID --actor-id ACTOR_ID --repository-id REPOSITORY_ID --kind pull_request.create
forgejo extensions bind --installation INSTALLATION_UUID --token-id TOKEN_ID --actor-id ACTOR_ID --repository-id REPOSITORY_ID --kind pull_request.review.submit
forgejo extensions bind --installation INSTALLATION_UUID --token-id RESOLVE_TOKEN_ID --actor-id REVIEW_ACTOR_ID --repository-id REPOSITORY_ID --kind pull_request.review.thread.resolve
forgejo extensions bind --installation INSTALLATION_UUID --token-id TOKEN_ID --actor-id ACTOR_ID --repository-id REPOSITORY_ID --kind pull_request.merge
forgejo extensions bind --installation INSTALLATION_UUID --token-id GRAPH_TOKEN_ID --actor-id GRAPH_ACTOR_ID --repository-id REPOSITORY_ID --kind issue.graph.mutate
forgejo extensions bindings --installation INSTALLATION_UUID
```

Use the owner-selected actor and token IDs for each exact operation; these
commands do not mint accounts or choose billing. The configured publisher,
reviewer, thread resolver and merger PATs each need `read:user`: Soda resolves their native owners
through `/api/v1/user` before background reads or operations. Factory enrollment
checks each owner against the selected operation bindings before changing
configuration or services. Forgejo permits `read:user` only on tokens with all
repository resources; native extension operations still require the selected
repository, token, actor and operation binding. Mixed native snapshot reads need
both `read:issue` and `read:repository`. The issue-graph credential needs
`read:issue` for adoption reads, and each background-read credential needs its
own installation/token/actor/repository binding: snapshot reads ride only
enrolled bindings, without granting mutation. Thread resolution uses a separate
token file and resolve-kind binding on the same review actor; the review-submit
binding does not authorize it. Publisher, creator, reviewer, thread resolver and
merger operation scopes remain separate from those read scopes.

Graph tuples may be selected in the same manifest, but are kept outside the
dashboard's five-role `factory_native_bindings` policy choices. Configure the
separate paired fields `forgejo_graph_credential_file` and
`forgejo_graph_actor_id` in the protected dashboard configuration using the
Graph token file and its native owner ID. Enrollment preserves this pair and
existing TypeSafe and lifecycle credential references. Keep the Graph PAT
root:soda (0:2000), mode 0640; the dashboard reads it through its fixed read-only
factory-directory mount. A new credential added while the dashboard is running
needs the same SELinux context as its existing publisher credential:

```sh
chcon --reference=/etc/soda/factory/publisher.pat /etc/soda/factory/graph.pat
```

The [CLI manifest example](../../cmd/soda-setup/README.md#explicit-factory-enrollment)
lists the fixed paths, complete provider-pin shape and hook selection. Install the
repository-scoped issue-open hook before creating tasks:
Fountain retains original creation facts for seven days, so missing hooks,
expired events and deleted task records cannot support historical recovery.

### Confirm enrollment and recover an interruption

After a successful command, inspect `forgejo extensions bindings` and confirm
the selected installation/token/actor/repository/operation tuples. Confirm
`soda-identity.service`, `soda-dashboard.service`, `forgejo.service` and
`soda-host.service` are active, then open Factory settings and refresh current
authority. Each repository's five enrolled policy choices should be available.
The root-only `/etc/soda/enrollment/hooks.json` receipt records repository names,
IDs and hook IDs; the hook's public fields must match the receipt. Preserve
that receipt and the intake secret. Provider sign-in, sponsorship and the
remaining repository/capacity grants are separate steps.

On failure, inspect the command's result before generating another token or
editing the manifest. The CLI restores previous local configurations and
attempts service recovery; native bindings already applied remain. Reuse the
same selected tuples, credential files and hook receipt with `--maintenance`
after resolving the reported cause. Maintenance must retain all repositories
already owned by that hook receipt. An existing unowned or mismatched hook is
preserved and refused. Do not delete it or change its secret to force adoption;
establish its ownership first. A successful authenticated secret update confirms
hook-secret installation because Forgejo deliberately omits secrets from reads.

## Offline native-mutation recovery

When an interrupted conditional or ordinary native mutation leaves its
reservation owner held, the host operator reconciles it offline under the
[recovery authority](../architecture/trust.md#sequencing-and-database-recovery) contract. There
is no force-unlock flag and no timeout-based release; an unknown effect stays
fenced for intervention.

1. Stop every native writer for the data set through the deployment's
   service controls: HTTP/SSH write ingress, native services, queue
   workers, scheduled and admin commands, and all surviving native
   descendants. Verify the whole domain has stopped; main-process exit
   alone is insufficient.
2. Inhibit restart for the duration of reconciliation, then create the
   offline marker at `$AppDataPath/nativeop-offline` (the Forgejo
   `AppDataPath` of this deployment). New ownership claims refuse while
   the marker exists.
3. Run the offline reconciliation command from the Forgejo server binary,
   naming the exact held owner and fencing generation:
   `admin native-operation recover --owner OWNER --generation GENERATION`.
   The command refuses while the marker is absent.
4. Read the printed verdict: `released` means the known effect was
   established and the exact owner released; `idle` means the reservation
   already holds no owner; `fenced` (exit 3) means the owner stays held —
   wrong owner/generation, unaccounted effects or uncertain attribution —
   and needs intervention, not a retry with different names.
5. Lift inhibition by removing the marker, then restart the native writers.

The durable service/domain controls for steps 1, 2 and 5 are the
`soda-forgejo-domain` host operator command (`cmd/soda-forgejo-domain`,
staged to `/usr/bin` alongside `soda-activate`): `stop` stops
`forgejo.service` and verifies
no container or unit survivor remains; `inhibit` runtime-masks the unit and
creates the offline marker in the fixed appliance data directory. The Forgejo
startup and pre-start extension installation enforce `AppDataPath=/data/gitea`, mapped to
`/var/lib/soda/forgejo/gitea` on the host; recovery uses that same declaration.
Changes to `app.ini` or `/etc/soda/forgejo.env` cannot relocate this directory;
`status` reports unit, mask, container and marker state only, never native
reservation rows; `lift` removes the marker and unmasks; `start` refuses
while the marker exists. Reservation diagnostics stay behind Forgejo's own
`admin native-operation status`. There is no force-unlock verb.

The marker mapping stays inside the selected host data volume: parent traversal,
symlinked path components and nonregular marker entries are refused. Marker
observation, creation and removal use the admitted directory descriptor. A new
marker has mode `0600`; an existing regular marker keeps its contents and mode.
A known-missing parent or marker means absent for `status`, `lift` and `start`,
but `inhibit` never creates parent directories. Other filesystem errors refuse
the operation before a successful marker report, unmask or start.
