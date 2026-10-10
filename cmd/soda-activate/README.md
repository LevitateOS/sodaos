# soda-activate

Activate private browser and Git access on a newly configured Soda OS appliance.
This is a host operator command, run as root after Forgejo's native installation
and `soda-setup` have completed.

Follow [Operator setup](../../docs/guides/operator-setup.md) for the complete
sequence and [Installation](../../docs/guides/installation.md) for client
reachability and certificate trust.

## Before activation

- Use an installed appliance with its native `soda` service account and units.
- Have `/etc/soda/dashboard.json` and `/etc/soda/grant-key` from setup,
  plus `/etc/soda/forgejo.env` supplied by the appliance image.
- Select an explicit private appliance IP and the configured Forgejo HTTPS
  browser origin. The bundled internal listeners and private paths must match
  [Service configuration](../../docs/reference/configuration.md).
- Choose Caddy local TLS or supply both a PEM certificate covering the browser
  origin and its PEM private key.

## Run once

For local TLS, setup must already have selected the same private IP as the
browser origin, for example `https://192.168.50.10` on HTTPS port 443:

```sh
sudo /usr/bin/soda-activate --bind-ip 192.168.50.10 --local-tls
```

Alternatively, use an existing certificate and key stored in restricted files:

```sh
sudo /usr/bin/soda-activate --bind-ip 192.168.50.10 \
  --certificate /home/operator/tls/cert.pem \
  --private-key /home/operator/tls/key.pem
```

`--local-tls` and supplied certificate files are separate choices. `--help`
prints usage. Public, loopback, unspecified and multicast bind addresses are
refused.

## Result and follow-up

Activation sets private file ownership and permissions, records the operator ID
for the Forgejo extension, writes proxy and Forgejo environment configuration,
and configures Forgejo SSH on the selected IP at port 2222. Supplied TLS files
are copied into `/etc/soda/tls`. It records `/etc/soda/activated`, reloads systemd,
restarts Forgejo and starts the dashboard and proxy.

Success means those services report active. Complete the later browser login
and SSH validation in the setup guide. With local TLS, explicitly trust the
appliance's public root certificate on each intended client.

If activation fails after writing configuration, inspect the affected service
journals; the activation marker may already exist. An existing marker refuses
reactivation. Existing installations use explicit configuration and
[credential maintenance](../../docs/reference/credentials.md#existing-install-maintenance).

Exit status is `0` on success, `2` for usage or validation failures and `1` for
runtime failures.
