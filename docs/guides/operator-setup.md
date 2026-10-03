# Operator bootstrap

Prepare Forgejo identity and private access for software collaboration and bounded
agent work. Factory execution and persistent human Projects have separate authority.

Soda uses Fountain’s native Forgejo extension authority. Setup reads the native
operator identity through `/api/v1/user`.

Media continuation: [Installation media](media.md).
Install/activate context: [Installation](installation.md).

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
   Setup records `operator_id` and creates the grant-encryption key. It refuses
   to overwrite existing configuration. On success it revokes the bootstrap
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

After browser access is configured, prepare the private repository, publishing and
review actors, selected agent profile and protected factory configuration through
the [factory operator reference](../reference/factory.md). A repository, project
join or issue label does not admit work. Follow the
[first-task walkthrough](../public/30-Use-Soda/15-software-factory.md) for explicit
admission, verification and human merge.

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
`soda-forgejo-domain` host operator command (`appliance/bin/soda-forgejo-domain`,
staged to `/usr/bin` alongside `soda-activate`): `stop` stops
`forgejo.service` and verifies
no container or unit survivor remains; `inhibit` runtime-masks the unit and
creates the offline marker, resolving the deployment `AppDataPath` from its
`app.ini` (`FORGEJO__server__APP_DATA_PATH` wins) and refusing to guess;
`status` reports unit, mask, container and marker state only, never native
reservation rows; `lift` removes the marker and unmasks; `start` refuses
while the marker exists. Reservation diagnostics stay behind Forgejo's own
`admin native-operation status`. There is no force-unlock verb.
