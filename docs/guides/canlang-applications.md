# Canlang development and private applications

Canlang compiles applications for workerd. Soda installs the compiler, platform
tools and pinned local runtime in Project OS; a separate application container
owns self-hosted state and lifetime. Canlang's artifact, deployment descriptor,
compatibility checks and portable bundle remain authoritative. The Cloudflare
Workers deployment path remains available through Canlang's existing CLI.

The selected self-hosted profile supplies persistent D1, Canlang accounts,
browser/MCP serving, persistent Files and durable scheduled work. It is for one
trusted team per application on a private appliance. App-local accounts are
distinct from Forgejo and Soda administrators. This does not establish support
for every Cloudflare service or hostile-tenant hosting.

## Develop inside a Project

The Project recipe installs `/opt/canlang` and exposes `can`, `can-platform`,
`can-dev` and `can-soda` on the shared path. Compiler/catalog/platform versions
travel together with Node, Bun, Miniflare, workerd, dependency licenses and the
example testkit. The application needs no Canlang source checkout or dependency
downloads at startup. Browser binaries and Cloudflare credentials are separate
operator inputs.

From an application directory containing `app.can`:

```sh
can --version
can check app.can
can compile --format=json app.can > app.json
can dev start --source app.can
can dev check
can dev preview.open
can dev example.run --expected-revision r1
can dev stop
```

Use the revision returned by the preview when it differs from `r1`. A source
must contain real examples; example execution refuses an empty suite. Preview
credentials are private development fixtures, independent of production owner
bootstrap. Preview state defaults to `~/.local/state/canlang/preview`, keyed by
the canonical application source. `CAN_DEV_STATE_ROOT` selects a separate private
development root. Restart retains data, accounts and storage identities. Model
schema/ownership changes refuse retained state; stopping and deliberately using
a fresh root is a development reset, not a production migration. Example
databases remain disposable and do not mutate preview data.

`can dev` supplies the retained D1/Identity preview and supported local bindings,
including Files backed by local R2. Scheduled handlers and migrations require
their owning runtime. To exercise the Soda target's native Files storage and
durable scheduled work, use the installed `can-soda` target with a separate
development data directory, explicit development owner bootstrap and an HTTPS
origin/proxy for Files. Local preview success does not establish native-provider
behavior. Never point development or example execution at production storage or
credentials.
`can lsp` supplies diagnostics, completion, navigation and formatting over
standard LSP; an editor chooses how to connect to it.

Retained Projects use explicit same-root
[Canlang tool maintenance](../../cmd/soda-project-maintain/README.md). Opening a
terminal or importing an image does not update their tools or replace their
roots. Active tool processes must be stopped before maintenance.

## Assemble and provision the application

Use the existing Canlang deployment descriptor and environment configuration,
with separate `DB` (Identity) and `STATE_DB` resource identities. Configure
`CAN_AUTH_ORIGIN` as the application's canonical HTTPS origin, such as
`https://board.soda.test`; this must differ from Forgejo's hostname. The same
compiled artifact can retain its Cloudflare deployment descriptor. The local
target checks supported bindings and producer versions before constructing a
bundle:

```sh
can-soda inventory
can-soda build --artifact app.json --descriptor deployment.json \
  --environment soda-environment.json --out bundle.json
```

The output is an immutable transport of the existing contracts and assembled
workerd modules, not another application manifest. It records the compatibility
date and flags. See [Canlang's target guide](../../../canlang/docs/soda-runtime.md)
for configuration and private credential-file formats. `build` exclusively
creates its output; runtime launch verifies the portable bundle identity again.

Create a private, existing, writable directory under `/home` and bootstrap its
owner while the application is stopped:

```sh
can-soda bootstrap --bundle bundle.json --data /home/soda-builder/apps/board-dev \
  --credentials /home/soda-builder/private/board-owner.json
can-soda member --bundle bundle.json --data /home/soda-builder/apps/board-dev \
  --owner TEAM_UUID --credentials /home/soda-builder/private/board-member.json
can-soda serve --bundle bundle.json --data /home/soda-builder/apps/board-dev \
  --owner TEAM_UUID --host 127.0.0.1 --port 24001
```

Credential files must be current-user-owned regular files with mode `0600`;
never put passwords in command arguments or application artifacts. Bootstrap
conditionally claims an empty Identity store and emits the owner team reference.
It does not adopt an existing account or reset a password. Adding a member does
not elevate or reactivate a removed account. Verification/invitation mail,
password-reset delivery and external identity federation are not configured.
Storage locks exclude concurrent
bootstrap, runtime and backup writers. Files requires a fixed backend port and
an actual HTTPS proxy at the configured public origin; direct plaintext URLs
are rejected. SIGTERM drains the Node gateway and direct workerd with a bounded
shutdown deadline. The native exec guard preserves the checked writer lock and
retires workerd if its gateway dies.

## Package and administer

The [runtime recipe](../../system/containers/canlang/Containerfile) consumes the
admitted offline tools archive's production-only `runtime/` subtree. Miniflare,
Wrangler, compiler, Bun and development fixtures stay in Project tools. The [application recipe](../../system/containers/canlang/Application.Containerfile)
adds the exact read-only bundle and labels its existing artifact, bundle, tools,
runtime and origin references. Build-time inputs belong under `/home`; startup
does not compile or download packages. The runtime uses UID/GID `10000`, a
read-only root, 1 GiB memory, up to two CPUs' worth of CPU time, 128 processes
or threads, and app-only writable storage. These fixed limits also appear in
Applications. They are limits rather than reserved CPUs. It
does not inherit Project homes, SSH, sudo, an engine socket or Forgejo credentials.

For production, build the application layer over the exact selected runtime
image using the application recipe's `APP_BUNDLE`, `APP_NAME`, `ARTIFACT_DIGEST`,
`BUNDLE_DIGEST`, `STORAGE_DIGEST`, `TOOLS_DIGEST`, `RUNTIME_DIGEST` and `APP_ORIGIN` inputs. Take `STORAGE_DIGEST` from the built Soda bundle's `storageFingerprint`. Preserve
those values from the admitted bundle and release; labels cannot substitute for
checking bytes. Transfer the resulting OCI archive through the operator's
trusted path and load it into the appliance's Podman store before registration.
There is no browser image-upload endpoint or build during Install.

The operator can instead open **Set up a new application** in Applications,
provide the admitted image references and choose the Soda administrator. Soda
reserves a separate provisional record and fresh data identity. **Begin owner
setup** starts only Canlang's first-owner form at the exact application origin;
normal application and MCP routes remain unavailable. This explicit admission
installs its private route and restarts the shared proxy. This can close the
calling browser connection: Refresh owner setups, Inspect the same setup and
choose Open owner setup again. An exact already ready route/generation reuses
its private credential without restarting the proxy. An unactivated partial
route still requires explicit activation.

Open **Create application owner** and choose the application's email and
password there. Soda keeps the opaque setup credential in a private native file;
the browser receives a temporary fragment link which the form removes from its
history. Passwords never enter Soda's records. After the form succeeds, choose
**Confirm owner and finish**. The native host stops and verifies that exact
setup container, checks the receipt against the admitted artifact, bundle and
storage identity, and installs an ordinary stopped container for its confirmed
owner. Soda atomically registers its administrator and owner. Refresh
Applications and explicitly Start it.

A lost reply remains provisional: Inspect and finish the same setup. The link
expires after 30 minutes; an already written matching receipt can still finish.
An expired link cannot create a new owner, extend itself or replace a missing
credential on existing data. If no owner was created, **Stop and cancel setup**
retains its data and address reservation for operator recovery. Cancelling an
uncreated setup also checks native absence before settling it. No ordinary
application admission accepts an absent owner team.

For explicit stopped terminal provisioning instead, provision production storage as an operator, with a new `a` plus 24 lowercase
hex digit data ID. Set `APP_IMAGE` to the exact loaded image ID, `APP_DATA` to
`/home/soda-applications/data/DATA_ID`, and `OWNER_CREDENTIALS` to a separate
private production credential file. Write that file privately with the format
above; make it regular, owned by `10000:10000` and mode `0600`. Bootstrap using
the production image's own account, rather than a Project user's UID:

```sh
install -d -m 0700 -o 10000 -g 10000 "$APP_DATA"
podman run --rm --read-only --network=none --cap-drop=all \
  --security-opt=no-new-privileges --pids-limit=128 --memory=768m --cpus=2 \
  -v "$APP_DATA:/home/app-state:Z" \
  -v "$OWNER_CREDENTIALS:/run/canlang-owner.json:ro,Z" \
  --entrypoint /opt/canlang/bin/can-soda "$APP_IMAGE" bootstrap \
  --bundle /opt/application/bundle.json --data /home/app-state \
  --credentials /run/canlang-owner.json
```

Record the returned `teamId` as the administration Spec's `owner_team`. Keep
credential files outside the application data tree and backup; retain them in
separate operator custody. Provision members through the same stopped image
with the `member` command and a separate credential file. Never provision while
the application writer is running.

Soda's Go application records retain exact image/artifact/runtime/tool/data
references, revisions, an explicit administrator grant and last observed native
container generation. The [API contract](../reference/api.md#application-administration)
owns registration and action payloads. Operators explicitly register, Install
and replace images; the recorded administrator or operator may Inspect, Start
and confirm Stop. Existing Spaces controls provide passive inventory and these
explicit actions. The Applications page loads inventory and inspects status automatically; opening it does not install, start or deploy anything. The app link remains available when status cannot be read.
Factory merge authority grants no deployment permission.

Native installation requires the exact local images and labels, a private
existing data directory `/home/soda-applications/data/DATA_ID` owned by
`10000:10000`, its private writer lock, and prior Canlang owner provisioning. Install and Replace require the runtime image and tools checksum selected by the
loaded Soda release. Inspect, Start, Stop and recovery of an already admitted
generation remain available after an OS release change; upgrading the app is an
explicit operation. It
binds only `127.0.0.1:20000..29999` and installs an independently supervised
`soda-canlang@APP_ID.service`. The Project lifecycle does not own this unit.
Installation leaves the unit stopped and disabled. Start enables it for boot
and waits for actual backend readiness. Stop disables boot startup and retains
all data. Image
replacement preflights exact references before stopping the old generation,
retains the same data and leaves the replacement stopped until explicit Start. Changed artifact/bundle bytes require matching nonempty `io.canlang.storage` fingerprints on the old container and new image. This permits ordinary UI/code updates with unchanged model declarations. An older container can first receive an unchanged-artifact runtime update to seed the storage fingerprint. Canlang independently checks app/resource identity and the retained schema. Schema migrations remain unsupported; source-bound opaque file/SEND operations may still refuse source changes.

## Configure private access

Operators configure a separate application hostname, private DNS/routing and
client TLS trust. Existing Caddy policy supports internal TLS or supplied
certificates. Supplied certificates must cover each application hostname.
App routes are fixed, root-owned files under `/home/soda-applications/routes`,
mounted read-only in the existing proxy. Installation verifies route ownership,
content and Caddy configuration, then explicitly restarts shared `soda-proxy`:
this briefly interrupts shared ingress. Start, Stop and image replacement never
restart the proxy. App cookies stay on the app hostname; Forgejo cookies and
Identity Broker credentials are not forwarded.

A ready backend does not establish that DNS, LAN/Tailnet routing or client trust
works. Check those from the intended client, then log in through Canlang's normal
browser/MCP flow. Host Tailnet enrollment alone proves none of these app steps.
The runtime blocks development/control routes from public serving.

## Stopped backup and fresh restore

The operator can open **Backups and recovery** in Applications, stop and inspect
the app, then choose **Back up**. **Refresh backups** reads the native inventory,
including each snapshot's timestamp and bundle identity. Opening this section
does not create a snapshot.

To recover, select a compatible backup, confirm that writes after that backup
will be absent, and choose **Restore**. The native host copies into fresh data
and installs a stopped generation. Soda advances the same application's
revision while retaining its ID, address, administrator, owner and previous
data. Explicitly Start after checking the result. Ordinary image replacement
continues to preserve the current data identity.

An interrupted Restore remains pending with writes disabled. Refresh
Applications and choose **Reconcile Restore** to resume the same recorded
operation. **Cancel Restore** succeeds only while the original stopped native
generation remains; a changed generation must be reconciled. Neither path
overwrites retained data. The inventory shows the newest 64 matching snapshots.

Stop the application and confirm the writer is stopped before copying storage.
The native storage helper independently takes the existing writer lock. From an
operator terminal with the installed helper:

```sh
/usr/libexec/soda/soda-canlang-storage backup \
  --source /home/soda-applications/data/DATA_ID \
  --out /home/soda-applications/backups/board-stopped-001
/usr/libexec/soda/soda-canlang-storage restore \
  --source /home/soda-applications/backups/board-stopped-001 \
  --out /home/soda-applications/data/NEW_DATA_ID
```

Output directories must be new. The helper copies the complete authoritative
storage tree, including D1/WAL, Files/DO storage, identity, work and storage
metadata, with an exact checksum inventory. It excludes disposable temporary
files, refuses links/special files and cannot overwrite an occupied restore
target. Restore verifies the snapshot before writing and assigns runtime
ownership when run as root. Reopen the restored app with the matching bundle,
owner and public origin; verify records, authorized file bytes and work receipts
before allowing ordinary writes. This low-level helper creates a separate data
tree; it does not update Soda's application record. Use the explicit Applications
Restore operation when recovering the same application at its existing address.

Backups are operator-triggered. Automatic frequency, tolerated data loss and
recovery targets remain unset. Restoring an older backup omits acknowledged
writes made after that backup. Image rollback cannot reverse database changes.

## Verification boundary and unsupported requirements

Native Rocky checks passed for actual compiled HTTP/D1 writes across restart and
container replacement, installed preview/examples/LSP, authenticated multi-user
CRUD/MCP, concurrent revision conflicts, Files authorization and retained bytes,
missed durable work with duplicate wakeups, unprivileged offline production
images, graceful/abrupt shutdown, local Caddy HTTPS and stopped fresh restore.
Bounded real D1 exhaustion rejected writes visibly, refused startup and recovered
prior acknowledged data after space was freed.
The [runtime evidence and contract](../development/canlang-soda-os-implementation-plan/production-workerd.md#source-baseline-scope-and-evidence)
retains scoped runtime receipts and technical contracts. The
[canonical Canlang recipes](../development/factory-remaining-tasks.md#canlang-execution-recipes)
own remaining steps; [testing](../development/testing.md#canlang-qualification-boundaries)
owns G1–G4 qualification.

Required R2, arbitrary DO namespaces, KV, service bindings, named queues,
Analytics Engine, Workflows, AI, Vectorize, remote provider secrets and migrations
are rejected by this target. Authored periodic schedules have not been qualified.
Its private
SQLite DO supplies Files and internal wakeups; this does not qualify arbitrary
DO programs or R2. No in-memory queue or no-op analytics counts as production
support. Full Files retention/cleanup, wider Files/provider failure modes,
provider-side external delivery and semantic upgrades need separate
qualification before support.

An installed matching-source development CoreOS candidate passed Project Create,
Join, native lifecycle guards, a real browser terminal and retained home after
cold boot. Its application journey passed first-owner setup, owner MCP and Files
access, durable work, stopped backup/fresh restore and stopped reboot retention
through trusted private-host HTTPS. A later candidate's tools-aware Application
Replace retained the same owner, data, file bytes and work receipt. Its appliance
upgrade needed a separately measured development helper to reconcile
the same operation; subsequent Factory and Project checks also used declared
development components. These results do not qualify those fixes as shipped bytes.
The complete Project image separately passed fresh offline ordinary-account
compilation, check/lint/format, persistent authenticated preview, isolated examples
and LSP. The final matching candidate's complete Factory-to-application journey,
two-user installed use, physical LAN/Tailnet clients and complete cold recovery
remain qualification work. Runtime release integration is checked separately in
its existing owners. No candidate has been publicly published.
Eventually requested Cloudflare service breadth remains an owner decision,
independent of this first-app scope.
