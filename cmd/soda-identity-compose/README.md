# soda-identity-compose

Opt one nested Compose service into Muse access for a provisioned Soda account.
This command runs as project root inside a Soda project, where the nested
workload engine and Muse launch interface are installed.

Follow [Project services](../../docs/guides/project-services.md#muse-opt-in) for
the ordinary workflow. Account custody and credential exposure belong to
[Credentials](../../docs/reference/credentials.md#identity-broker).

## Prerequisites

- A compatible Linux service defined in your project's Compose file.
- A provisioned Soda login with its root-owned mode-0600 account marker under
  `/var/lib/soda/accounts/` and an authorized Muse subscription connection.
- The project tools `/usr/local/bin/podman-compose`, `/usr/bin/podman` and Muse.
- The project's launch socket at `/run/soda-muse-interface/launch.sock` and a
  tmpfs runtime directory for nested registration.

## Opt in a service

Run from the directory containing your Compose project:

```sh
sudo soda-identity-compose --login soda-tester \
  --file compose.yml --service development --muse
```

Replace the synthetic login and service with the provisioned account and actual
Compose service. `--file` defaults to `compose.yml`. `--login`, `--service` and
`--muse` are required; there are no positional arguments. The account marker
establishes the authorizing Soda identity.

## What happens

The helper creates a runtime registration directory and an additional Compose
file. It adds read-only mounts for the selected service's credential directory,
Muse launcher, Muse helper and public launch interface. The original Compose
file retains ownership of the service's configured user and other volumes.

It runs `podman-compose up -d` for the selected service, resolves exactly one
matching immutable container ID, and registers that incarnation with the Muse
launch interface. Run `muse` inside the selected service afterward.

Cloning or restarting the child requires fresh validation through this helper;
an old registration cannot authorize another container incarnation.

Compose startup output is passed through. Failures go to standard error and
exit `1`. A registration failure can occur after Compose has already started
the service; inspect that service and the reported error before retrying.
