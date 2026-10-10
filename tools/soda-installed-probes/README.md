# soda-installed-probes

Run scoped native checks from a selected client against an installed appliance
or project fixture. Release developers and operators use this support tool to
observe trusted HTTPS, developer access, personal Git and nested workloads.
The implementation is in [internal/acceptance](../../internal/acceptance/README.md).

## Choose a probe

| Command | Inputs and effects |
| --- | --- |
| `service-https ORIGIN CA_FILE` | Make a trusted HTTPS request and report status without capturing the body or following redirects. |
| `developer-access ROOT_DIR` | Read a prepared access request, exercise SSH/PTY/SCP/SFTP and authority boundaries, and write retained access results. |
| `personal-git prepare FIXTURE_DIR` | Prepare private keys and retained personal Git fixture state. |
| `personal-git exercise FIXTURE_DIR` | Exercise the fixture's native Git workflow and record results. |
| `personal-git unlock FIXTURE_DIR` | Perform the fixture's explicit key-unlock phase. |
| `workload-access FIXTURE_DIR` | Exercise configured workload network/database access and write results. |
| `workload-exec FIXTURE_DIR` | Exercise owner workload execution and member denial through SSH. |

The fixture probes require their existing private request/configuration files,
trusted SSH inputs and reachable target. Personal Git and workload probes also
require `SODA_NATIVE_VALIDATE=soda-test` and their expected fixture identities.
Read the selected probe's source and prepared fixture before invoking it;
these commands can write local state and execute remote Git or workload actions.

## Observe configured HTTPS

Run from the repository root with the pinned Go toolchain, an actual HTTPS
origin and an absolute trusted PEM CA file:

```sh
go run ./tools/soda-installed-probes service-https \
  https://soda.example /home/soda-tester/private/soda-ca.pem
```

The CA must be a regular file, not a symlink, and cannot be writable by group
or others. The probe uses only that CA, bypasses proxies and has a 15-second
HTTP timeout. It accepts HTTP status 200 through 399 after successful TLS
validation and prints the observed status. This result does not establish
browser login or authentication.

The binary exits `0` for a successful probe, `1` for failure or incomplete
work, and `2` for a usage rejection represented by `UsageError`. `go run`
reports a child failure as `exit status N`. There is no `--help` action;
invalid dispatcher arguments print the accepted subcommand names.

See [native support](../../docs/development/native-support.md#installed-substrate-and-retained-integrations)
for installed observations and [testing](../../docs/development/testing.md)
for acceptance scope and evidence.
