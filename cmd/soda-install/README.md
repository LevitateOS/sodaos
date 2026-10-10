# soda-install

`soda-install` is the operator's console for installing Soda onto disk and
continuing setup after first boot. It ships on installation media and the
installed appliance. Use it from native root access on a supported CoreOS host.

## Choose an action

The command accepts exactly one action and no additional flags:

| Action | When to use it |
| --- | --- |
| `disk` | Install from the live CoreOS media onto a selected disk. |
| `configure` | Set up private browser access on the installed appliance. |
| `enroll-key` | Open an optional, temporary laptop public-key import window. |
| `enrollment-serve` | Internal enrollment service entrypoint. |
| `enrollment-receive` | Internal SSH public-key receiver entrypoint. |

`disk`, `configure` and `enroll-key` require an interactive controlling terminal.
The two enrollment entrypoints are service-managed parts of `enroll-key`; use
the console action to begin enrollment.

## Install and continue

Boot the Soda installation media and follow its console. For an explicit disk
installation from the live root terminal:

```sh
soda-install disk
```

The console reviews the destination, operator password and authenticated media
before asking for explicit disk-write confirmation. Disk installation overwrites
the selected destination. The disk action refuses an installed, non-live host.
After the outcome, the console offers reboot or poweroff.

After first boot, use the installed root console or root SSH terminal:

```sh
soda-install enroll-key
soda-install configure
```

Key enrollment requires the native root password and a private IPv4 address
reachable from the laptop. It accepts one public key through a temporary
password-authenticated listener on port `22222`, for at most five minutes.

Configuration guides you through native Forgejo administrator setup, takes its
bootstrap token through hidden input, and invokes Soda setup and activation for
the selected private address. Local HTTPS requires explicit client trust of the
displayed appliance certificate authority. Existing configured access produces
access guidance rather than a second bootstrap.

Invalid command syntax exits `2`; operational failures exit `1`. Only one
interactive installer can run at a time.

Follow [installation media](../../docs/guides/media.md),
[optional key enrollment](../../docs/operator/enroll-key.md), and
[operator setup](../../docs/guides/operator-setup.md) for the complete procedure.
