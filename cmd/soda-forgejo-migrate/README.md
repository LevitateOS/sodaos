# soda-forgejo-migrate

Prepare the appliance's Forgejo configuration for startup by removing legacy
inline database password settings. Host operators normally encounter this as
the one-shot `soda-forgejo-migrate.service`, required before Forgejo starts.

The appliance supplies the database password through `PASSWD_URI=file:...`.
A remaining inline `[database]` `PASSWD` conflicts with that setting and can
block startup. This helper removes that conflict from `app.ini`.

## Normal operation

The installed service invokes `/usr/bin/soda-forgejo-migrate` without arguments
against `/var/lib/soda/forgejo/gitea/conf/app.ini`. It runs only when that path
exists. Inspect its result on an installed host with:

```sh
systemctl status soda-forgejo-migrate.service
journalctl -u soda-forgejo-migrate.service
```

For an explicitly selected configuration file during operator maintenance:

```sh
sudo env SODA_FORGEJO_APP_INI=/home/operator/forgejo/app.ini \
  /usr/bin/soda-forgejo-migrate
```

The selected file must be readable and writable. An unset or empty
`SODA_FORGEJO_APP_INI` selects the installed default. The binary has no CLI flags;
extra arguments are ignored.

## Effects and results

The helper removes case-insensitive `PASSWD` assignments only inside the exact
`[database]` section. Password settings in other sections survive. It rewrites
the file in place, retaining ownership and mode; a final unterminated line gains
a newline even when no password entry was removed.

Removed-line counts go to standard error without password contents. Missing or
nonregular files are skipped with exit status `0`; read or write failure exits
`1`. See [Host services](../../system/host/services/README.md) for startup order
and [Operator setup](../../docs/guides/operator-setup.md) for bootstrap and recovery.
