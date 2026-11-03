# Update SodaOS and use fallback safely

Software freshness is a [mandatory requirement](../../development/freshness.md): keep provider CLIs and managed dependencies current; an old baseline cannot justify stale software. The independent automatic CLI updater is still implementation work, so this requirement must not be read as a claim that it is already deployed.

## Update an appliance

SodaOS host updates are explicit operator actions through native rpm-ostree;
there is no separate Soda release updater or published automatic host update
channel. The appliance's vendor configuration disables Zincati auto-updates for
the derived Soda image. Zincati can still report **active** while idle; that
does not mean automatic updates are enabled. See the upstream
[disabled-update behavior](https://coreos.github.io/zincati/usage/auto-updates/#disabling-auto-updates).

To move an appliance to a new release, take a
[backup](50-backups-and-restoration.md), then reinstall from the new release's
media or explicitly select its verified complete Soda host image with
rpm-ostree. A retained local OCI archive or a `:candidate` reference is not a
channel that discovers future releases. Do not substitute the upstream Fedora
base image: it does not contain Soda's host content. Administrator Zincati
overrides belong under `/etc/zincati/config.d/`: the same filename replaces the
vendor file, and later-sorting settings take precedence. Enabling updates alone
does not provide a Soda update graph.

Retained Project tooling is refreshed separately with `soda-project-maintain`;
see [Project OS maintenance](../../reference/project-os.md#same-root-maintenance).
The source [development deployment](../../development/forge-development.md)
updates selected components separately from host release transitions.

After the update, compare the booted release with the selected target:

```sh
sudo rpm-ostree status
```

A successful boot alone does not prove that Projects and extensions are ready; open Soda and check them before resuming work.

## Recreate a Docker runtime

For Docker, use the same release image, Compose file, `.env` and project name to stop and recreate the runtime while retaining its `/etc`, `/var` and `/home` volumes. This is container lifecycle management, not an appliance transition. Do not pull a different tag into an existing stateful deployment as an assumed upgrade. See [Install with Docker](../60-Install/50-docker.md).

## Recover an interruption

If an update is interrupted, keep the current state intact and follow [Recovery](80-recovery.md). Never overwrite newer state with an older database or use a host rollback as a substitute for data restoration. [Backups](50-backups-and-restoration.md) remain a separate protection and restore path.
