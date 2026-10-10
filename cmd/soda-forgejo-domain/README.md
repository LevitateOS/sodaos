# soda-forgejo-domain

Control the appliance's Forgejo writer domain during offline native-mutation
recovery. This is a root-only host operator command for the installed
`forgejo.service` and `soda-forgejo` container.

Read the [offline recovery procedure](../../docs/guides/operator-setup.md#offline-native-mutation-recovery)
and its [recovery authority](../../docs/architecture/trust.md#sequencing-and-database-recovery)
before using the state-changing verbs.

## Inspect current state

```sh
sudo /usr/bin/soda-forgejo-domain status
```

Output reports the unit's activity and mask, container presence, and offline
marker presence. Native reservation ownership and fencing generations are
inspected through Forgejo's own `admin native-operation status` command.

## Recovery controls

The command accepts exactly one of these verbs:

| Verb | Effect |
| --- | --- |
| `stop` | Stop Forgejo and wait up to 60 seconds for the unit to become inactive and its container to disappear. |
| `inhibit` | Require stopped writers, runtime-mask the unit and create the offline marker. |
| `status` | Print bounded host-side state. |
| `lift` | Remove the offline marker and unmask the unit. |
| `start` | Require an absent marker and unmasked unit, start Forgejo and check unit activity. |

Start an authorized recovery by stopping writers, then inhibiting restart:

```sh
sudo /usr/bin/soda-forgejo-domain stop
sudo /usr/bin/soda-forgejo-domain inhibit
sudo /usr/bin/soda-forgejo-domain status
```

Perform the native reconciliation step in the recovery guide with the exact
recorded owner and generation. Once reconciliation permits restarting writers:

```sh
sudo /usr/bin/soda-forgejo-domain lift
sudo /usr/bin/soda-forgejo-domain start
```

The marker is resolved from the fixed appliance Forgejo data-path declaration,
normally `/var/lib/soda/forgejo/gitea/nativeop-offline`. Unsafe or unobservable
paths refuse the relevant operation. A fenced native reservation remains held
for intervention; this tool has no force-unlock verb and never opens native SQL.

`--help` prints usage. Exit status is `0` on success, `2` for invalid arguments
or a non-root caller, and `1` when the requested operation fails.
