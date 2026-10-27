# soda-project-maintain

`soda-project-maintain` provides explicit `muse`, `canlang`, `factory-mcp`, `terminal-support`
and `release` operations.
Muse installs packaged public tools and restores the launch interface.
Terminal support repairs one admitted helper in an idle retained root.
It operates on the project's retained container and requires appliance root.

The host configuration supplies Project networking and tool inputs; Payload v4
selects the Project image. Tailnet state and its image belong to the
[extension lifecycle](../../docs/architecture/networking.md), so the retired
`tailnet_management` and `tailnet_image` machine fields are rejected.

The [Project OS maintenance contract](../../docs/reference/project-os.md#same-root-maintenance)
owns admission, supported roots, effects and interrupted recovery. Terminal repair:

```sh
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project p0123456789abcdef01234567 --check
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project p0123456789abcdef01234567
```

`--development-installed-host` explicitly selects the already installed host helper
for an unqualified development repair. Normal repair consumes the complete installed
release's bundled input and metadata. Neither operation upgrades the host.

`release --release-dir DIRECTORY [--check]` consumes one complete downloaded
GitHub release. Normal admission requires its exact native qualification receipt.
`release --recover` resumes the recorded release; `release --rollback` restores
the recorded host/support pair with idle-state checks. The internal
`release-reconcile` boot action verifies the active pair and retires its recorded
recovery copies before reopening admission. See the
[operator procedure](../../docs/guides/operator-setup.md#explicit-release-maintenance)
for effects and recovery and the [qualification boundary](../../docs/architecture/release.md#host-update-ownership).

Terminal repair includes running and stopped Rocky 10.2 roots with empty managed
terminal/factory state. A stopped Project remains stopped: native namespace access
uses its exact retained root and existing UID/GID mappings. The compatibility probe
uses temporary private runtime mounts, without starting init or user workloads.
Container identity and runtime state are checked around the operation; the exact
root is unmounted afterward. Unknown state, active work or uncertain cleanup blocks
completion. The same container, accounts, keys, files and volumes are preserved.
Verified completion retires only the recorded old/new terminal recovery copies
before reopening admission; interrupted retirement resumes idempotently. Older
unrecorded copies remain untouched. The
[Project OS contract](../../docs/reference/project-os.md#terminal-support)
defines file checks and the coordinated rollback source.

## Normal use

Choose the exact project ID from Soda, then run on the appliance:

```sh
sudo /usr/libexec/soda/soda-project-maintain \
  muse --project p0123456789abcdef01234567
```

The ID has the form `p` followed by 24 lowercase hexadecimal characters.
The Muse operation checks the matching project container's identity and waits briefly
for it to be running. It does not start a stopped project.

| Option | Meaning |
| --- | --- |
| `--project ID` | Required exact project identity. |
| `--config PATH` | Absolute host configuration; defaults to `/etc/soda/host.json`. |
| `--tools PATH` | Absolute public tool directory; defaults to `/usr/share/soda/muse-tools`. |
| `--bind-only` | Restore the launch interface without installing tools or the system bus prerequisite. |
| `--provider-store` | Stage the shared provider store's currently selected native binary instead of the pinned tool directory. The store is only ever opened read-only; Projects receive a staged copy and gain no write access. Fails closed on a stale or missing selection. |

The project startup service stages the store selection automatically at
launch (`muse --provider-store --project ID`, best-effort so boot never
blocks), then restores the interface (`muse --bind-only --project ID`),
preceded by a best-effort `soda-provider-update refresh` due-check that never
blocks the retained Project. Full maintenance is also available as an
explicit operator action:

```sh
sudo /usr/libexec/soda/soda-project-maintain \
  muse --provider-store --project p0123456789abcdef01234567
```

With `--provider-store`, the native binary is resolved from the shared
provider store's current selection and verified against its selected digest
before staging, so retained Projects are served current bytes without
per-Project downloads or image rebuilds. Staging also plants
`/usr/local/libexec/soda/muse-selection.json`, the admitted-identity receipt
carrying the selection's `checked_at` that the in-Project launch path
revalidates at every invocation against the live host selection and its TTL.
A digest drift between the receipt and the live selection refuses the
outdated staged bytes; a stale or unknown live selection refuses even when
the receipt still reads current; a missing receipt refuses whenever any store
knowledge exists and only grandfathers the legacy staged binary on hosts
where the store was never admitted. The staged binary itself is fully
re-hashed against the receipt digest at invocation (never a size-only
check). Image-owned wrappers still come from `--tools`. The default
pinned-directory path is unchanged until the provider store rollout
completes. Where the store view is not mounted into a running Project, the
host launch gate still refuses a guest whose hash no longer matches the
fresh selection; serving v2 bytes into such a Project without maintenance
needs the read-only store-view mount (named in the UP-REVIEW-03 C/RQ03/RQ05
report) or host-side restaging at invocation.

The same startup service runs `factory-mcp --project ID` to restore the fixed
Factory socket directory in retained Projects. It binds
`/var/lib/soda/dashboard/factory-ipc` at `/run/soda/factory-ipc` read-only, with
execution, devices and set-ID behavior disabled on that mount. This reuses the
native mount operation and checks the exact running container before attaching.
It does not recreate or restart the Project. An existing correct mount is left
in place. Worker credentials are staged separately per run; access to this
socket alone grants no Factory tool authority.

## Inputs and effects

Maintenance reads host configuration and `/usr/share/soda/release.json`, validates
the configured Muse release and public tools, and uses native Podman access.
Full maintenance installs `dbus-broker` if missing, starts `dbus.socket`, and
replaces these project files with the admitted public executables:

- `/usr/local/bin/muse`
- `/usr/local/bin/soda-identity-compose`
- `/usr/local/libexec/soda/muse`

It attaches the dedicated public launch directory at
`/run/soda-muse-interface` with restricted mount permissions. Accounts, checkouts,
volumes and unrelated installed tools remain in the same project root.
Credentials are not included in the public tool payload.

## Canlang tools

Use the exact retained Project identity on an appliance that already carries
the selected public Canlang maintenance payload:

```sh
sudo /usr/libexec/soda/soda-project-maintain \
  canlang --project p0123456789abcdef01234567
```

This command accepts only `--project ID`; it has no `--check`, `--tools`,
`--config`, development override or download path. It verifies the root-owned
lock and archive, neither group/world writable, in `/usr/share/soda/canlang-tools` against the
selected source revision and archive SHA256. Their
release input owner is
[`soda-stage-canlang`](../../tools/release-assets/README.md#admit-offline-canlang-tools).

The Project must already be running. Maintenance waits briefly for running
state and verifies its retained container identity/incarnation around staging;
it does not start a stopped Project. This differs from the separate
terminal-support repair's admitted stopped-root path.

Stop `can-dev` previews, compilers and platform commands before maintenance.
Any process whose executable resolves beneath `/opt/canlang` causes refusal,
including the shipped Node/Bun runtime used by tool commands. Maintenance
does not kill those processes or silently stop previews. It also refuses
foreign files at the four `/usr/local/bin` command destinations or a linked
tool root.

Verified bytes are staged privately beneath `/home` in the same retained
Project, then `/opt/canlang` is replaced by a directory rename. Existing tool
bytes remain at `/opt/.canlang-before-ARCHIVE_SHA256` for explicit operator
inspection/cleanup; an occupied previous destination blocks another attempt.
If the new-directory rename fails, the old directory is restored. Other
interruptions are retained for inspection, not automatic replay or a promised
complete rollback.

Only the installed tools and their four command symlinks are changed. Project
accounts, checkouts, credentials, files and volumes remain in the same root.
Canlang preview/example state is outside the tool installation and is not
reset or copied by maintenance. Startup does not run this installation; a
release upgrade or retained Project tooling refresh is an explicit operator
action. Local package/workflow checks do not qualify this maintenance on an
installed appliance or a production application image.

Returns `0` on completion and `1` with a prefixed stderr diagnostic on failure.
When Muse is disabled, `--bind-only` returns successfully without attaching it;
full maintenance reports that the runtime is disabled.

See [Project OS maintenance](../../docs/reference/project-os.md#same-root-maintenance),
[Muse subscription access](../../docs/reference/project-os.md#muse-subscription-access),
and [credentials](../../docs/reference/credentials.md#normal-muse-command).


Release maintenance freezes Fountain intent and source/target component pins in
its format-2 journal. It prefetches both selected sides before mutation and uses
Fountain’s offline package command after stopping the service. Recovery and
rollback retain installation identities and generations; disabled optional
packages are excluded from acquisition.

Exit status `75` from release reconciliation means the core committed while
extension readiness remains failed or unknown. The command retains the completed
journal and never automatically rolls back or rewrites desired enablement. The
reconcile service leaves this result for inspection; explicitly restarting it
performs a fresh bounded readiness check even when no operation marker remains.
